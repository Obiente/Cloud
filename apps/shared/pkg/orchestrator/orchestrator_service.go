package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/registry"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/client"
)

// OrchestratorService is the main orchestration service that runs continuously
type OrchestratorService struct {
	deploymentManager *DeploymentManager
	gameServerManager *GameServerManager
	serviceRegistry   *registry.ServiceRegistry
	healthChecker     *registry.HealthChecker
	metricsStreamer   *MetricsStreamer
	rollbackMonitor   *RollbackMonitor
	syncInterval      time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewOrchestratorService creates a new orchestrator service
func NewOrchestratorService(strategy string, maxDeploymentsPerNode int, syncInterval time.Duration) (*OrchestratorService, error) {
	deploymentManager, err := NewDeploymentManager(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, err
	}

	// Create game server manager (using same strategy and max deployments for now)
	// TODO: Consider separate configuration for game servers
	gameServerManager, err := NewGameServerManager(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, err
	}

	serviceRegistry, err := registry.NewServiceRegistry()
	if err != nil {
		return nil, err
	}

	healthChecker := registry.NewHealthChecker(serviceRegistry, 1*time.Minute)
	metricsStreamer := NewMetricsStreamer(serviceRegistry)

	// Create rollback monitor (may fail if Docker is not available, but that's OK)
	rollbackMonitor, err := NewRollbackMonitor()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to create rollback monitor: %v (rollback notifications will be disabled)", err)
		rollbackMonitor = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Register global metrics streamer for access from other services
	SetGlobalMetricsStreamer(metricsStreamer)
	
	service := &OrchestratorService{
		deploymentManager: deploymentManager,
		gameServerManager: gameServerManager,
		serviceRegistry:   serviceRegistry,
		healthChecker:     healthChecker,
		metricsStreamer:   metricsStreamer,
		rollbackMonitor:   rollbackMonitor,
		syncInterval:      syncInterval,
		ctx:               ctx,
		cancel:            cancel,
	}
	
	// Register global orchestrator service for access from other services
	SetGlobalOrchestratorService(service)
	
	return service, nil
}

// Start begins all background orchestration tasks
func (os *OrchestratorService) Start() {
	logger.Info("[Orchestrator] Starting orchestration service...")

	// Start periodic sync with Docker
	os.serviceRegistry.StartPeriodicSync(os.ctx, os.syncInterval)
	logger.Debug("[Orchestrator] Started periodic sync (interval: %v)", os.syncInterval)

	// Start health checking
	os.healthChecker.Start(os.ctx)
	logger.Debug("[Orchestrator] Started health checker")

	// Start metrics streaming (handles live collection and periodic storage)
	os.metricsStreamer.Start()
	logger.Debug("[Orchestrator] Started metrics streaming")

	// Backfill missing hourly aggregates on startup
	go os.backfillMissingHourlyAggregates()
	logger.Debug("[Orchestrator] Started backfill of missing hourly aggregates")

	// Start cleanup tasks
	go os.cleanupTasks()
	logger.Debug("[Orchestrator] Started cleanup tasks")

	// Start usage aggregation (hourly)
	go os.aggregateUsage()
	logger.Debug("[Orchestrator] Started usage aggregation")

	// Start storage updates (every 5 minutes)
	go os.updateStoragePeriodically()
	logger.Debug("[Orchestrator] Started periodic storage updates")

	// Start build history cleanup (daily)
	go os.cleanupBuildHistory()
	logger.Debug("[Orchestrator] Started build history cleanup")

	// Start stray container cleanup (every 6 hours)
	go os.cleanupStrayContainers()
	logger.Debug("[Orchestrator] Started stray container cleanup")

	// Start rollback monitor (if available)
	if os.rollbackMonitor != nil {
		os.rollbackMonitor.Start()
		logger.Debug("[Orchestrator] Started rollback monitor")
	}

	logger.Info("[Orchestrator] Orchestration service started successfully")
}

// Stop gracefully stops the orchestrator service
func (os *OrchestratorService) Stop() {
	logger.Info("[Orchestrator] Stopping orchestration service...")
	os.cancel()

	if err := os.deploymentManager.Close(); err != nil {
		logger.Warn("[Orchestrator] Error closing deployment manager: %v", err)
	}
	if err := os.serviceRegistry.Close(); err != nil {
		logger.Warn("[Orchestrator] Error closing service registry: %v", err)
	}
	
	os.metricsStreamer.Stop()
	logger.Debug("[Orchestrator] Stopped metrics streamer")

	if os.rollbackMonitor != nil {
		os.rollbackMonitor.Stop()
		logger.Debug("[Orchestrator] Stopped rollback monitor")
	}

	logger.Info("[Orchestrator] Orchestration service stopped")
}

// GetMetricsStreamer returns the metrics streamer instance
func (os *OrchestratorService) GetMetricsStreamer() *MetricsStreamer {
	return os.metricsStreamer
}

// GetGameServerManager returns the game server manager instance
func (os *OrchestratorService) GetGameServerManager() *GameServerManager {
	return os.gameServerManager
}

// containerStats holds container resource usage statistics
type containerStats struct {
	CPUUsage       float64
	MemoryUsage    int64
	NetworkRxBytes int64
	NetworkTxBytes int64
	DiskReadBytes  int64
	DiskWriteBytes int64
}

// getContainerStats retrieves current stats from a Docker container
func (os *OrchestratorService) getContainerStats(containerID string) (*containerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get container stats (stream=false means get one snapshot)
	statsResp, err := os.serviceRegistry.DockerClient().ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	// Decode stats JSON response - ContainerStats returns JSON that matches StatsJSON structure
	var statsJSON struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  uint   `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		BlkioStats struct {
			IoServiceBytesRecursive []struct {
				Major uint64 `json:"major"`
				Minor uint64 `json:"minor"`
				Op    string `json:"op"`
				Value uint64 `json:"value"`
			} `json:"io_service_bytes_recursive"`
		} `json:"blkio_stats"`
	}
	if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Calculate CPU usage percentage
	cpuUsage := 0.0
	if statsJSON.PreCPUStats.SystemUsage > 0 && statsJSON.CPUStats.SystemUsage > statsJSON.PreCPUStats.SystemUsage {
		cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
		if systemDelta > 0 && statsJSON.CPUStats.OnlineCPUs > 0 {
			cpuUsage = (cpuDelta / systemDelta) * float64(statsJSON.CPUStats.OnlineCPUs) * 100.0
		}
	}

	// Calculate network bytes (sum across all interfaces)
	var networkRx, networkTx int64
	for _, netStats := range statsJSON.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
	}

	// Calculate disk I/O (read and write)
	// Docker's BlkioStats typically uses lowercase operation names: "read", "write"
	// Use case-insensitive matching to handle variations
	var diskRead, diskWrite int64
	for _, ioStat := range statsJSON.BlkioStats.IoServiceBytesRecursive {
		op := strings.ToLower(strings.TrimSpace(ioStat.Op))
		switch op {
		case "read":
			diskRead += int64(ioStat.Value)
		case "write":
			diskWrite += int64(ioStat.Value)
		}
	}
	
	// Log if we're not getting any disk I/O data (for debugging)
	if len(statsJSON.BlkioStats.IoServiceBytesRecursive) == 0 {
		logger.Debug("[getContainerStats] No BlkioStats found for container %s - disk I/O may not be available", containerID[:12])
	} else if diskRead == 0 && diskWrite == 0 {
		// Log operation names we're seeing (for debugging)
		ops := make([]string, 0, len(statsJSON.BlkioStats.IoServiceBytesRecursive))
		for _, ioStat := range statsJSON.BlkioStats.IoServiceBytesRecursive {
			ops = append(ops, ioStat.Op)
		}
		logger.Debug("[getContainerStats] BlkioStats available for container %s but no read/write found. Operations seen: %v", containerID[:12], ops)
	}

	return &containerStats{
		CPUUsage:       cpuUsage,
		MemoryUsage:    int64(statsJSON.MemoryStats.Usage),
		NetworkRxBytes: networkRx,
		NetworkTxBytes: networkTx,
		DiskReadBytes:  diskRead,
		DiskWriteBytes: diskWrite,
	}, nil
}

// cleanupTasks runs periodic cleanup operations
// Aggregates raw metrics into hourly summaries and removes old raw data
func (os *OrchestratorService) cleanupTasks() {
	// Run every 1 hour to aggregate and clean old metrics frequently
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Running cleanup tasks...")
			
			// Keep last 24 hours of raw metrics for real-time viewing
			// Aggregate older metrics into hourly summaries
			aggregateCutoff := time.Now().Add(-24 * time.Hour).Truncate(time.Hour)
			
			// Get all deployments and game servers that have metrics older than cutoff
			var deploymentIDs []string
			metricsDB := database.GetMetricsDB()
			metricsDB.Table("deployment_metrics").
				Select("DISTINCT deployment_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("deployment_id", &deploymentIDs)
			
			var gameServerIDs []string
			metricsDB.Table("game_server_metrics").
				Select("DISTINCT game_server_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("game_server_id", &gameServerIDs)
			
			if len(deploymentIDs) == 0 && len(gameServerIDs) == 0 {
				logger.Debug("[Orchestrator] No old metrics to aggregate")
				logger.Debug("[Orchestrator] Cleanup tasks completed")
				continue
			}
			
			logger.Debug("[Orchestrator] Aggregating metrics for %d deployments, %d game servers", len(deploymentIDs), len(gameServerIDs))
			
			// Process deployments and game servers in parallel batches
			const batchSize = 10 // Process 10 resources concurrently
			totalAggregated := 0
			totalDeleted := int64(0)
			var aggMutex sync.Mutex
			
			// Process deployments in batches
			for i := 0; i < len(deploymentIDs); i += batchSize {
				end := i + batchSize
				if end > len(deploymentIDs) {
					end = len(deploymentIDs)
				}
				batch := deploymentIDs[i:end]
				
				var wg sync.WaitGroup
				for _, deploymentID := range batch {
					wg.Add(1)
					go func(depID string) {
						defer wg.Done()
						aggregated, deleted := os.aggregateDeploymentMetrics(depID, aggregateCutoff)
						if aggregated > 0 || deleted > 0 {
							aggMutex.Lock()
							totalAggregated += aggregated
							totalDeleted += deleted
							aggMutex.Unlock()
						}
					}(deploymentID)
				}
				wg.Wait()
			}
			
			// Process game servers in batches
			for i := 0; i < len(gameServerIDs); i += batchSize {
				end := i + batchSize
				if end > len(gameServerIDs) {
					end = len(gameServerIDs)
				}
				batch := gameServerIDs[i:end]
				
				var wg sync.WaitGroup
				for _, gameServerID := range batch {
					wg.Add(1)
					go func(gsID string) {
						defer wg.Done()
						aggregated, deleted := os.aggregateGameServerMetrics(gsID, aggregateCutoff)
						if aggregated > 0 || deleted > 0 {
							aggMutex.Lock()
							totalAggregated += aggregated
							totalDeleted += deleted
							aggMutex.Unlock()
						}
					}(gameServerID)
				}
				wg.Wait()
			}
			
			logger.Debug("[Orchestrator] Aggregated %d hours, deleted %d raw metrics, cleanup tasks completed", totalAggregated, totalDeleted)
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) aggregateUsage() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Aggregating usage...")
			now := time.Now()
			month := now.Format("2006-01")
			
			// Calculate the start of the current month for accurate aggregation
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			
			// Get all organizations with deployments (active or historical for the month)
			type orgRow struct {
				OrganizationID string
				Count          int
			}
			var orgRows []orgRow
			// Get organizations with running deployments now, plus those with usage records this month
			database.DB.Table("deployment_locations dl").
				Select("d.organization_id, COUNT(*) as count").
				Joins("JOIN deployments d ON d.id = dl.deployment_id").
				Where("dl.status = ?", "running").
				Group("d.organization_id").Scan(&orgRows)
			
			// Also include organizations that have usage records this month (for historical tracking)
			var existingOrgs []string
			database.DB.Table("usage_monthly").
				Where("month = ?", month).
				Distinct("organization_id").
				Pluck("organization_id", &existingOrgs)
			
			orgMap := make(map[string]int)
			for _, r := range orgRows {
				orgMap[r.OrganizationID] = r.Count
			}
			for _, orgID := range existingOrgs {
				if _, exists := orgMap[orgID]; !exists {
					orgMap[orgID] = 0
				}
			}
			
			for orgID := range orgMap {
				// Get all deployments for this organization (running or that ran this month)
				var deploymentIDs []string
				database.DB.Table("deployments d").
					Select("d.id").
					Where("d.organization_id = ?", orgID).
					Pluck("d.id", &deploymentIDs)
				
				if len(deploymentIDs) == 0 {
					continue
				}
				
				// Get actual CPU/memory allocations from deployments table
				type deploymentAlloc struct {
					ID          string
					CPUShares   *int64
					MemoryBytes *int64
				}
				var allocs []deploymentAlloc
				database.DB.Table("deployments").
					Select("id, cpu_shares, memory_bytes").
					Where("id IN ?", deploymentIDs).
					Scan(&allocs)
				
				allocMap := make(map[string]struct {
					cpuShares   int64
					memoryBytes int64
				})
				for _, a := range allocs {
					cpu := int64(1) // Default 1 core if not specified
					if a.CPUShares != nil && *a.CPUShares > 0 {
						// CPU shares: typically 1024 = 1 core, but we'll use as-is
						cpu = *a.CPUShares / 1024
						if cpu < 1 {
							cpu = 1
						}
					}
					mem := int64(512 * 1024 * 1024) // Default 512MB
					if a.MemoryBytes != nil && *a.MemoryBytes > 0 {
						mem = *a.MemoryBytes
					}
					allocMap[a.ID] = struct {
						cpuShares   int64
						memoryBytes int64
					}{cpu, mem}
				}
				
				// Calculate CPU core-seconds and memory byte-seconds from actual runtime
				// We track from deployment_locations: sum all runtime periods this month
				type locationRuntime struct {
					DeploymentID string
					CreatedAt    time.Time
					UpdatedAt    time.Time
					Status       string
				}
				var locations []locationRuntime
				database.DB.Table("deployment_locations dl").
					Select("dl.deployment_id, dl.created_at, dl.updated_at, dl.status").
					Joins("JOIN deployments d ON d.id = dl.deployment_id").
					Where("d.organization_id = ? AND (dl.created_at >= ? OR dl.updated_at >= ?)", orgID, monthStart, monthStart).
					Order("dl.deployment_id, dl.created_at").
					Scan(&locations)
				
				var totalCPUSeconds int64
				
				// Group by deployment and calculate total runtime
				// Sum all runtime periods for each deployment this month
				runtimeByDeployment := make(map[string]int64)
				for _, loc := range locations {
					// Calculate the time this location was active during the month
					locationStart := loc.CreatedAt
					if locationStart.Before(monthStart) {
						locationStart = monthStart // Only count from month start
					}
					
					locationEnd := now
					if loc.Status != "running" {
						// If stopped, use the update time (when it stopped)
						if loc.UpdatedAt.After(locationStart) {
							locationEnd = loc.UpdatedAt
						} else {
							// Updated before month start or before location start - skip
							continue
						}
					}
					
					// Ensure we don't count future times
					if locationEnd.After(now) {
						locationEnd = now
					}
					
					if locationEnd.After(locationStart) {
						runtimeSeconds := int64(locationEnd.Sub(locationStart).Seconds())
						runtimeByDeployment[loc.DeploymentID] += runtimeSeconds
					}
				}
				
				// Calculate CPU totals using allocations (CPU is based on allocation)
				for deploymentID, runtimeSeconds := range runtimeByDeployment {
					alloc, exists := allocMap[deploymentID]
					if !exists {
						// Default allocation if not found
						alloc = struct {
							cpuShares   int64
							memoryBytes int64
						}{1, 512 * 1024 * 1024} // 1 core, 512MB default
					}
					totalCPUSeconds += alloc.cpuShares * runtimeSeconds
				}
				
				// Calculate memory byte-seconds from actual metrics (not allocated memory)
				// Aggregate from raw metrics (recent) + hourly aggregates (older) for the month
				var orgMemoryByteSeconds int64
				
				// Sum from raw metrics (last 24 hours)
				// Calculate memory byte-seconds by summing memory per distinct timestamp * interval
				rawCutoff := time.Now().Add(-24 * time.Hour)
				if rawCutoff.Before(monthStart) {
					rawCutoff = monthStart
				}
				
				type orgMemoryPerTimestamp struct {
					MemorySum   int64
					Timestamp   time.Time
				}
				var orgMemoryTimestamps []orgMemoryPerTimestamp
				metricsDB := database.GetMetricsDB()
				metricsDB.Table("deployment_metrics dm").
					Select(`
						SUM(dm.memory_usage) as memory_sum,
						dm.timestamp as timestamp
					`).
					Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawCutoff).
					Group("dm.timestamp").
					Order("dm.timestamp ASC").
					Scan(&orgMemoryTimestamps)
				
				// Calculate byte-seconds: for each timestamp, use memory * 5 seconds (average interval)
				intervalSeconds := int64(5) // Average interval between metric samples
				for _, m := range orgMemoryTimestamps {
					orgMemoryByteSeconds += m.MemorySum * intervalSeconds
				}
				
				// Sum from hourly aggregates (older than 24 hours, within month)
				if rawCutoff.After(monthStart) {
					var hourlyMemorySum int64
					metricsDB.Table("deployment_usage_hourly duh").
						Select("COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds").
						Where("duh.deployment_id IN ? AND duh.hour >= ? AND duh.hour < ?", deploymentIDs, monthStart, rawCutoff).
						Scan(&hourlyMemorySum)
					orgMemoryByteSeconds += hourlyMemorySum
				}
				
				// Aggregate bandwidth from raw metrics (recent) + hourly aggregates (older) for the month
				type bandwidthRow struct {
					RxBytes int64
					TxBytes int64
				}
				
				// Sum from raw metrics (last 24 hours)
				var rawBandwidth bandwidthRow
				rawCutoffBandwidth := time.Now().Add(-24 * time.Hour)
				if rawCutoffBandwidth.Before(monthStart) {
					rawCutoffBandwidth = monthStart
				}
				metricsDB.Table("deployment_metrics dm").
					Select("COALESCE(SUM(dm.network_rx_bytes), 0) as rx_bytes, COALESCE(SUM(dm.network_tx_bytes), 0) as tx_bytes").
					Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawCutoffBandwidth).
						Scan(&rawBandwidth)
					
					// Sum from hourly aggregates (older than 24 hours, within month)
					var hourlyBandwidth bandwidthRow
				if rawCutoffBandwidth.After(monthStart) {
					metricsDB.Table("deployment_usage_hourly duh").
						Select("COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as rx_bytes, COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as tx_bytes").
						Where("duh.deployment_id IN ? AND duh.hour >= ? AND duh.hour < ?", deploymentIDs, monthStart, rawCutoffBandwidth).
						Scan(&hourlyBandwidth)
				}
				
			}
		case <-os.ctx.Done():
			return
		}
	}
}


// GetDeploymentManager returns the deployment manager instance
func (os *OrchestratorService) GetDeploymentManager() *DeploymentManager {
	return os.deploymentManager
}

// GetServiceRegistry returns the service registry instance
func (os *OrchestratorService) GetServiceRegistry() *registry.ServiceRegistry {
	return os.serviceRegistry
}

// GetHealthChecker returns the health checker instance
func (os *OrchestratorService) GetHealthChecker() *registry.HealthChecker {
	return os.healthChecker
}

// aggregateGameServerMetrics aggregates metrics for a single game server (called in parallel)
func (os *OrchestratorService) aggregateGameServerMetrics(gameServerID string, aggregateCutoff time.Time) (aggregated int, deleted int64) {
	// Get org ID once
	var orgID string
	database.DB.Table("game_servers").
		Select("organization_id").
		Where("id = ?", gameServerID).
		Pluck("organization_id", &orgID)
	
	if orgID == "" {
		return 0, 0
	}
	
	// Find the oldest metric for this game server
	var oldestTime time.Time
	metricsDB := database.GetMetricsDB()
	metricsDB.Table("game_server_metrics").
		Select("MIN(timestamp)").
		Where("game_server_id = ? AND timestamp < ?", gameServerID, aggregateCutoff).
		Scan(&oldestTime)
	
	if oldestTime.IsZero() {
		return 0, 0
	}
	
	// Aggregate hour by hour
	currentHour := oldestTime.Truncate(time.Hour)
	deletedInGameServer := int64(0)
	aggregatedCount := 0
	
	for currentHour.Before(aggregateCutoff) {
		nextHour := currentHour.Add(1 * time.Hour)
		
		// Check if hourly aggregate already exists
		var existingHourly database.GameServerUsageHourly
		err := metricsDB.Where("game_server_id = ? AND hour = ?", gameServerID, currentHour).
			First(&existingHourly).Error
		
		// Check if raw metrics still exist for this hour (if not, we can't recalculate)
		var rawMetricsCount int64
		metricsDB.Table("game_server_metrics").
			Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?", gameServerID, currentHour, nextHour).
			Count(&rawMetricsCount)
		
		// Only create/recalculate if raw metrics exist
		if rawMetricsCount > 0 {
			// Delete existing aggregate if present (to allow recalculation)
			if err == nil {
				metricsDB.Where("game_server_id = ? AND hour = ?", gameServerID, currentHour).
					Delete(&database.GameServerUsageHourly{})
			}
			
			// Aggregate metrics for this hour
			type hourlyAgg struct {
				AvgCPUUsage      float64
				SumMemoryUsage   float64
				AvgMemoryUsage   float64
				SumNetworkRx     int64
				SumNetworkTx     int64
				SumDiskRead      int64
				SumDiskWrite     int64
				Count            int64
				TimestampCount   int64
			}
			var agg hourlyAgg
			
			err := metricsDB.Table("game_server_metrics").
				Select(`
					AVG(cpu_usage) as avg_cpu_usage,
					AVG(memory_usage) as avg_memory_usage,
					SUM(memory_usage) as sum_memory_usage,
					COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
					COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
					COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
					COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
					COUNT(*) as count,
					COUNT(DISTINCT timestamp) as timestamp_count
				`).
				Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
				Scan(&agg).Error
			
			if err != nil {
				logger.Warn("[Orchestrator] Failed to query game server metrics for %s at %s: %v", gameServerID, currentHour, err)
				currentHour = nextHour
				continue
			}
			
			if agg.Count == 0 {
				currentHour = nextHour
				continue
			}
			
			// Calculate CPU core-seconds and memory byte-seconds using actual timestamp intervals
			type metricTimestamp struct {
				CPUUsage    float64
				MemorySum   int64
				Timestamp   time.Time
			}
			var timestamps []metricTimestamp
			
			err = metricsDB.Table("game_server_metrics").
				Select("cpu_usage, memory_usage, timestamp").
				Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
				Order("timestamp ASC").
				Scan(&timestamps).Error
			
			var cpuCoreSeconds float64
			var memoryByteSeconds int64
			
			if err == nil && len(timestamps) > 1 {
				// Calculate intervals between timestamps
				for i := 0; i < len(timestamps)-1; i++ {
					interval := int64(timestamps[i+1].Timestamp.Sub(timestamps[i].Timestamp).Seconds())
					if interval <= 0 {
						interval = 5 // Default 5-second interval
					}
					
					cpuCoreSeconds += (timestamps[i].CPUUsage / 100.0) * float64(interval)
					memoryByteSeconds += timestamps[i].MemorySum * interval
				}
				
				// Handle last timestamp: if it's not at the end of the hour, use remaining time
				if len(timestamps) > 0 {
					lastTimestamp := timestamps[len(timestamps)-1].Timestamp
					timeRemaining := int64(nextHour.Sub(lastTimestamp).Seconds())
					if timeRemaining > 0 && timeRemaining <= 3600 {
						lastCPU := timestamps[len(timestamps)-1].CPUUsage
						lastMemory := timestamps[len(timestamps)-1].MemorySum
						cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
						memoryByteSeconds += lastMemory * timeRemaining
					}
				}
				
				// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
				if cpuCoreSeconds > 0 {
					agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
				}
				
				// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
				if memoryByteSeconds > 0 {
					agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
				} else if agg.TimestampCount > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					estimatedTimestamps := float64(agg.Count) / 1.0
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else {
					agg.AvgMemoryUsage = 0
				}
			} else {
				// Fallback calculation
				if agg.TimestampCount > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					estimatedTimestamps := float64(agg.Count) / 1.0
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				}
			}
			
			hourlyUsage := database.GameServerUsageHourly{
				GameServerID:     gameServerID,
				OrganizationID:    orgID,
				Hour:              currentHour,
				AvgCPUUsage:       agg.AvgCPUUsage,
				AvgMemoryUsage:    agg.AvgMemoryUsage,
				BandwidthRxBytes:  agg.SumNetworkRx,
				BandwidthTxBytes:  agg.SumNetworkTx,
				DiskReadBytes:     agg.SumDiskRead,
				DiskWriteBytes:    agg.SumDiskWrite,
				SampleCount:       agg.Count,
			}
			
			if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
				logger.Warn("[Orchestrator] Failed to create hourly aggregate for game server %s at %s: %v", gameServerID, currentHour, err)
			} else {
				aggregatedCount++
				
				// Delete the raw metrics for this hour in batch
				result := metricsDB.Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
					Delete(&database.GameServerMetrics{})
				
				if result.Error == nil {
					deletedInGameServer += result.RowsAffected
				}
			}
		}
		
		currentHour = nextHour
	}
	
	return aggregatedCount, deletedInGameServer
}

// aggregateDeploymentMetrics aggregates metrics for a single deployment (called in parallel)
func (os *OrchestratorService) aggregateDeploymentMetrics(deploymentID string, aggregateCutoff time.Time) (aggregated int, deleted int64) {
	// Get org ID once
	var orgID string
	database.DB.Table("deployments").
		Select("organization_id").
		Where("id = ?", deploymentID).
		Pluck("organization_id", &orgID)
	
	// Find the oldest metric for this deployment
	var oldestTime time.Time
	metricsDB := database.GetMetricsDB()
	metricsDB.Table("deployment_metrics").
		Select("MIN(timestamp)").
		Where("deployment_id = ? AND timestamp < ?", deploymentID, aggregateCutoff).
		Scan(&oldestTime)
	
	if oldestTime.IsZero() {
		return 0, 0
	}
	
	// Aggregate hour by hour
	currentHour := oldestTime.Truncate(time.Hour)
	deletedInDeployment := int64(0)
	aggregatedCount := 0
	
	for currentHour.Before(aggregateCutoff) {
		nextHour := currentHour.Add(1 * time.Hour)
		
		// Check if hourly aggregate already exists
		var existingHourly database.DeploymentUsageHourly
		err := metricsDB.Where("deployment_id = ? AND hour = ?", deploymentID, currentHour).
			First(&existingHourly).Error
		
		// Check if raw metrics still exist for this hour (if not, we can't recalculate)
		var rawMetricsCount int64
		metricsDB.Table("deployment_metrics").
			Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", deploymentID, currentHour, nextHour).
			Count(&rawMetricsCount)
		
		// Only create/recalculate if raw metrics exist (allows recalculation of existing aggregates)
		if rawMetricsCount > 0 {
			// Delete existing aggregate if present (to allow recalculation with corrected logic)
			if err == nil {
				metricsDB.Where("deployment_id = ? AND hour = ?", deploymentID, currentHour).
					Delete(&database.DeploymentUsageHourly{})
			}
			// Aggregate metrics for this hour
			// First, get basic aggregates
			type hourlyAgg struct {
				AvgCPUUsage      float64
				SumMemoryUsage   float64
				AvgMemoryUsage   float64
				SumNetworkRx     int64
				SumNetworkTx     int64
				SumDiskRead      int64
				SumDiskWrite     int64
				SumRequestCount  int64
				SumErrorCount    int64
				Count            int64
				TimestampCount   int64
			}
			var agg hourlyAgg
			
			err := metricsDB.Table("deployment_metrics").
				Select(`
					AVG(cpu_usage) as avg_cpu_usage,
					AVG(memory_usage) as avg_memory_usage,
					SUM(memory_usage) as sum_memory_usage,
					COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
					COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
					COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
					COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
					COALESCE(SUM(request_count), 0) as sum_request_count,
					COALESCE(SUM(error_count), 0) as sum_error_count,
					COUNT(*) as count,
					COUNT(DISTINCT timestamp) as timestamp_count
				`).
				Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
					deploymentID, currentHour, nextHour).
				Scan(&agg).Error
			
			if err == nil && agg.Count > 0 {
				// Calculate CPU core-seconds and memory byte-seconds properly using actual timestamp intervals
				// Fetch timestamps ordered to calculate intervals
				type metricTimestamp struct {
					CPUUsage    float64
					MemorySum   int64
					Timestamp   time.Time
				}
				var timestamps []metricTimestamp
				metricsDB.Table("deployment_metrics").
					Select(`
						AVG(cpu_usage) as cpu_usage,
						SUM(memory_usage) as memory_sum,
						timestamp
					`).
					Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
						deploymentID, currentHour, nextHour).
					Group("timestamp").
					Order("timestamp ASC").
					Scan(&timestamps)
				
				// Calculate core-seconds and byte-seconds from actual intervals
				var cpuCoreSeconds float64
				var memoryByteSeconds int64
				metricInterval := int64(5) // Default 5 seconds
				if len(timestamps) > 0 {
					for i, ts := range timestamps {
						interval := metricInterval
						if i > 0 {
							intervalSeconds := int64(ts.Timestamp.Sub(timestamps[i-1].Timestamp).Seconds())
							if intervalSeconds > 0 && intervalSeconds <= 3600 { // Sanity check: max 1 hour
								interval = intervalSeconds
							}
						} else if i == 0 && len(timestamps) == 1 {
							// Single timestamp: use default interval
							interval = metricInterval
						}
						cpuCoreSeconds += (ts.CPUUsage / 100.0) * float64(interval)
						memoryByteSeconds += ts.MemorySum * interval
					}
					// Handle last timestamp: if it's not at the end of the hour, use remaining time
					if len(timestamps) > 0 {
						lastTimestamp := timestamps[len(timestamps)-1].Timestamp
						timeRemaining := int64(nextHour.Sub(lastTimestamp).Seconds())
						if timeRemaining > 0 && timeRemaining <= 3600 {
							lastCPU := timestamps[len(timestamps)-1].CPUUsage
							lastMemory := timestamps[len(timestamps)-1].MemorySum
							cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
							memoryByteSeconds += lastMemory * timeRemaining
						}
					}
				}
				
				// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
				// This way, when we query with SUM((avg_cpu_usage / 100.0) * 3600), we get correct core-seconds
				if cpuCoreSeconds > 0 {
					agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
				}
				
				// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
				// This way, when we query with SUM(avg_memory_usage * 3600), we get correct byte-seconds
				if memoryByteSeconds > 0 {
					agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
				} else if agg.TimestampCount > 0 {
					// Fallback: calculate byte-seconds from average memory and default interval
					// If we have timestamps but couldn't calculate intervals, use 5-second default
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					// Assume 5-second intervals, so byte-seconds = avgMemory * interval * timestamps
					// But we want byte-seconds/hour, so: (avgMemory * 5 * timestamps) / 3600
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					// Last resort: use average memory and assume 5-second intervals
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					// Estimate timestamps: assume metrics are evenly distributed over the hour
					estimatedTimestamps := float64(agg.Count) / 1.0 // 1 metric per timestamp
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else {
					// No metrics at all - set to 0
					agg.AvgMemoryUsage = 0
				}
				
				hourlyUsage := database.DeploymentUsageHourly{
					DeploymentID:     deploymentID,
					OrganizationID:    orgID,
					Hour:              currentHour,
					AvgCPUUsage:       agg.AvgCPUUsage,
					AvgMemoryUsage:    agg.AvgMemoryUsage,
					BandwidthRxBytes:  agg.SumNetworkRx,
					BandwidthTxBytes:  agg.SumNetworkTx,
					DiskReadBytes:     agg.SumDiskRead,
					DiskWriteBytes:    agg.SumDiskWrite,
					RequestCount:      agg.SumRequestCount,
					ErrorCount:        agg.SumErrorCount,
					SampleCount:       agg.Count,
				}
				
				if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
					logger.Warn("[Orchestrator] Failed to create hourly aggregate for %s at %s: %v", deploymentID, currentHour, err)
				} else {
					aggregatedCount++
					
					// Delete the raw metrics for this hour in batch
					result := metricsDB.Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
						deploymentID, currentHour, nextHour).
						Delete(&database.DeploymentMetrics{})
					
					if result.Error == nil {
						deletedInDeployment += result.RowsAffected
					}
				}
			}
		}
		
		currentHour = nextHour
	}
	
	return aggregatedCount, deletedInDeployment
}

// backfillMissingHourlyAggregates calculates missing hourly aggregates for all deployments
// This runs once on startup to ensure data consistency
func (os *OrchestratorService) backfillMissingHourlyAggregates() {
	logger.Info("[Orchestrator] Starting backfill of missing hourly aggregates...")
	
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		logger.Warn("[Orchestrator] Metrics database not available, skipping backfill")
		return
	}
	
	// Verify tables exist
	var metricsTableExists bool
	var hourlyTableExists bool
	metricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'deployment_metrics'
		)
	`).Scan(&metricsTableExists)
	metricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'deployment_usage_hourly'
		)
	`).Scan(&hourlyTableExists)
	
	logger.Debug("[Orchestrator] Tables exist: deployment_metrics=%v, deployment_usage_hourly=%v", metricsTableExists, hourlyTableExists)
	
	if !metricsTableExists {
		logger.Warn("[Orchestrator] deployment_metrics table does not exist, skipping backfill")
		return
	}
	
	if !hourlyTableExists {
		logger.Warn("[Orchestrator] deployment_usage_hourly table does not exist, cannot create aggregates")
		return
	}
	
	// Get current time and calculate backfill range (current month start to now)
	// Include ALL hours in the current month, not just older than 24 hours
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	backfillCutoff := now.Truncate(time.Hour) // Current hour (not 24 hours ago)
	
	logger.Debug("[Orchestrator] Backfill range: %s to %s", monthStart, backfillCutoff)
	
	// First check if there are any metrics at all
	var totalMetricsCount int64
	metricsDB.Table("deployment_metrics").
		Where("timestamp >= ? AND timestamp < ?", monthStart, backfillCutoff.Add(1*time.Hour)).
		Count(&totalMetricsCount)
	logger.Debug("[Orchestrator] Found %d total raw metrics in backfill range", totalMetricsCount)
	
	if totalMetricsCount == 0 {
		logger.Warn("[Orchestrator] No raw metrics found in backfill range - deployment_usage_hourly will remain empty")
		return
	}
	
	// Get all deployments that have metrics in the backfill range (entire current month)
	var deploymentIDs []string
	metricsDB.Table("deployment_metrics").
		Select("DISTINCT deployment_id").
		Where("timestamp >= ? AND timestamp < ?", monthStart, backfillCutoff.Add(1*time.Hour)).
		Pluck("deployment_id", &deploymentIDs)
	
	if len(deploymentIDs) == 0 {
		logger.Warn("[Orchestrator] No deployments with metrics to backfill (found %d total metrics but no distinct deployments)", totalMetricsCount)
		return
	}
	
	logger.Info("[Orchestrator] Found %d deployments with metrics for backfill", len(deploymentIDs))
	
	logger.Info("[Orchestrator] Backfilling aggregates for %d deployments", len(deploymentIDs))
	
	totalAggregated := 0
	const batchSize = 10 // Process 10 deployments concurrently
	var aggMutex sync.Mutex
	
	// Process in batches
	for i := 0; i < len(deploymentIDs); i += batchSize {
		end := i + batchSize
		if end > len(deploymentIDs) {
			end = len(deploymentIDs)
		}
		batch := deploymentIDs[i:end]
		
		var wg sync.WaitGroup
		for _, deploymentID := range batch {
			wg.Add(1)
			go func(depID string) {
				defer wg.Done()
				
				// Get org ID for this deployment
				var orgID string
				database.DB.Table("deployments").
					Select("organization_id").
					Where("id = ?", depID).
					Pluck("organization_id", &orgID)
				
				if orgID == "" {
					logger.Warn("[Orchestrator] Deployment %s has no organization_id, skipping", depID)
					return
				}
				
				// Find hours with raw metrics but missing aggregates
				// Iterate through each hour in the backfill range (entire current month)
				aggregatedForDeployment := 0
				currentHour := monthStart.Truncate(time.Hour)
				
				for currentHour.Before(backfillCutoff) || currentHour.Equal(backfillCutoff) {
					nextHour := currentHour.Add(1 * time.Hour)
					
					// Check if aggregate already exists
					var existingHourly database.DeploymentUsageHourly
					err := metricsDB.Where("deployment_id = ? AND hour = ?", depID, currentHour).
						First(&existingHourly).Error
					
					// If aggregate exists, skip (don't recalculate on startup)
					if err == nil {
						currentHour = nextHour
						continue
					}
					
					// Check if raw metrics exist for this hour
					var rawMetricsCount int64
					metricsDB.Table("deployment_metrics").
						Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", depID, currentHour, nextHour).
						Count(&rawMetricsCount)
					
					if rawMetricsCount > 0 {
						// Create aggregate for this hour
						aggregated, err := os.aggregateSingleHour(depID, orgID, currentHour, nextHour)
						if aggregated {
							aggregatedForDeployment++
						} else if err != nil {
							logger.Warn("[Orchestrator] Failed to aggregate hour %s for deployment %s: %v", currentHour, depID, err)
						}
					}
					
					currentHour = nextHour
				}
				
				if aggregatedForDeployment > 0 {
					aggMutex.Lock()
					totalAggregated += aggregatedForDeployment
					aggMutex.Unlock()
					logger.Info("[Orchestrator] Backfilled %d hours for deployment %s", aggregatedForDeployment, depID)
				} else {
					// Log why no aggregates were created
					var totalHoursChecked int
					var hoursWithMetrics int
					var hoursWithAggregates int
					checkHour := monthStart.Truncate(time.Hour)
					for checkHour.Before(backfillCutoff) || checkHour.Equal(backfillCutoff) {
						totalHoursChecked++
						var rawCount int64
						metricsDB.Table("deployment_metrics").
							Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", depID, checkHour, checkHour.Add(1*time.Hour)).
							Count(&rawCount)
						if rawCount > 0 {
							hoursWithMetrics++
						}
						var aggExists database.DeploymentUsageHourly
						if err := metricsDB.Where("deployment_id = ? AND hour = ?", depID, checkHour).First(&aggExists).Error; err == nil {
							hoursWithAggregates++
						}
						checkHour = checkHour.Add(1 * time.Hour)
					}
					logger.Debug("[Orchestrator] Deployment %s: checked %d hours, %d with metrics, %d with aggregates", depID, totalHoursChecked, hoursWithMetrics, hoursWithAggregates)
				}
			}(deploymentID)
		}
		wg.Wait()
	}
	
	if totalAggregated > 0 {
		logger.Info("[Orchestrator] ✓ Backfill completed successfully: created %d hourly aggregates", totalAggregated)
	} else {
		logger.Warn("[Orchestrator] ⚠️ Backfill completed but no aggregates were created (found %d deployments with %d raw metrics)", 
			len(deploymentIDs), totalMetricsCount)
		logger.Warn("[Orchestrator] This may indicate that aggregates already exist or there's an issue with aggregation")
	}
}

// aggregateSingleHour creates a single hourly aggregate for a deployment
// Returns true if aggregate was created, false otherwise
func (os *OrchestratorService) aggregateSingleHour(deploymentID, orgID string, hourStart, hourEnd time.Time) (bool, error) {
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		return false, fmt.Errorf("metrics database not available")
	}
	
	logger.Debug("[Orchestrator] Aggregating hour %s for deployment %s", hourStart, deploymentID)
	
	// Get basic aggregates
	type hourlyAgg struct {
		AvgCPUUsage      float64
		SumMemoryUsage   float64
		AvgMemoryUsage   float64
		SumNetworkRx     int64
		SumNetworkTx     int64
		SumDiskRead      int64
		SumDiskWrite     int64
		SumRequestCount  int64
		SumErrorCount    int64
		Count            int64
		TimestampCount   int64
	}
	var agg hourlyAgg
	
	err := metricsDB.Table("deployment_metrics").
		Select(`
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_usage) as avg_memory_usage,
			SUM(memory_usage) as sum_memory_usage,
			COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
			COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
			COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
			COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
			COALESCE(SUM(request_count), 0) as sum_request_count,
			COALESCE(SUM(error_count), 0) as sum_error_count,
			COUNT(*) as count,
			COUNT(DISTINCT timestamp) as timestamp_count
		`).
		Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
			deploymentID, hourStart, hourEnd).
		Scan(&agg).Error
	
	if err != nil {
		logger.Warn("[Orchestrator] Failed to query metrics for %s at %s: %v", deploymentID, hourStart, err)
		return false, err
	}
	
	if agg.Count == 0 {
		logger.Debug("[Orchestrator] No metrics found for %s at %s", deploymentID, hourStart)
		return false, nil
	}
	
	logger.Debug("[Orchestrator] Found %d metrics for %s at %s", agg.Count, deploymentID, hourStart)
	
	// Calculate CPU core-seconds and memory byte-seconds using actual timestamp intervals
	type metricTimestamp struct {
		CPUUsage    float64
		MemorySum   int64
		Timestamp   time.Time
	}
	var timestamps []metricTimestamp
	metricsDB.Table("deployment_metrics").
		Select(`
			AVG(cpu_usage) as cpu_usage,
			SUM(memory_usage) as memory_sum,
			timestamp
		`).
		Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
			deploymentID, hourStart, hourEnd).
		Group("timestamp").
		Order("timestamp ASC").
		Scan(&timestamps)
	
	// Calculate core-seconds and byte-seconds from actual intervals
	var cpuCoreSeconds float64
	var memoryByteSeconds int64
	metricInterval := int64(5) // Default 5 seconds
	if len(timestamps) > 0 {
		// First timestamp: use time from hour start to first timestamp, or default interval
		firstTimestamp := timestamps[0].Timestamp
		firstInterval := int64(firstTimestamp.Sub(hourStart).Seconds())
		if firstInterval <= 0 {
			firstInterval = metricInterval
		} else if firstInterval > 3600 {
			firstInterval = metricInterval // Sanity check
		}
		cpuCoreSeconds += (timestamps[0].CPUUsage / 100.0) * float64(firstInterval)
		memoryByteSeconds += timestamps[0].MemorySum * firstInterval
		
		// Subsequent timestamps: use actual interval between timestamps
		// For each interval from timestamps[i-1] to timestamps[i], use memory[i-1] (the value at the start of the interval)
		for i := 1; i < len(timestamps); i++ {
			interval := metricInterval
			intervalSeconds := int64(timestamps[i].Timestamp.Sub(timestamps[i-1].Timestamp).Seconds())
			if intervalSeconds > 0 && intervalSeconds <= 3600 { // Sanity check: max 1 hour
				interval = intervalSeconds
			}
			// Use memory from the PREVIOUS timestamp for this interval (memory[i-1] represents memory from timestamps[i-1] to timestamps[i])
			cpuCoreSeconds += (timestamps[i-1].CPUUsage / 100.0) * float64(interval)
			memoryByteSeconds += timestamps[i-1].MemorySum * interval
		}
		// Handle last timestamp: if it's not at the end of the hour, use remaining time
		// Note: We've already counted the interval from timestamps[last-1] to timestamps[last],
		// so we only need to add the remaining time from timestamps[last] to hourEnd
		if len(timestamps) > 0 {
			lastTimestamp := timestamps[len(timestamps)-1].Timestamp
			timeRemaining := int64(hourEnd.Sub(lastTimestamp).Seconds())
			if timeRemaining > 0 && timeRemaining <= 3600 {
				lastCPU := timestamps[len(timestamps)-1].CPUUsage
				lastMemory := timestamps[len(timestamps)-1].MemorySum
				cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
				memoryByteSeconds += lastMemory * timeRemaining
			}
		}
	}
	
	// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
	if cpuCoreSeconds > 0 {
		agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
	}
	
	// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
	// This way, when we query with SUM(avg_memory_usage * 3600), we get correct byte-seconds
	if memoryByteSeconds > 0 {
		agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
	} else if agg.TimestampCount > 0 {
		// Fallback: calculate byte-seconds from average memory and default interval
		// If we have timestamps but couldn't calculate intervals, use 5-second default
		avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
		// Assume 5-second intervals, so byte-seconds = avgMemory * interval * timestamps
		// But we want byte-seconds/hour, so: (avgMemory * 5 * timestamps) / 3600
		metricInterval := int64(5)
		fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
		agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
	} else if agg.Count > 0 {
		// Last resort: use average memory and assume 5-second intervals
		avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
		metricInterval := int64(5)
		// Estimate timestamps: assume metrics are evenly distributed over the hour
		estimatedTimestamps := float64(agg.Count) / 1.0 // 1 metric per timestamp
		fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
		agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
	} else {
		// No metrics at all - set to 0
		agg.AvgMemoryUsage = 0
	}
	
	hourlyUsage := database.DeploymentUsageHourly{
		DeploymentID:     deploymentID,
		OrganizationID:    orgID,
		Hour:              hourStart,
		AvgCPUUsage:       agg.AvgCPUUsage,
		AvgMemoryUsage:    agg.AvgMemoryUsage,
		BandwidthRxBytes:  agg.SumNetworkRx,
		BandwidthTxBytes:  agg.SumNetworkTx,
		DiskReadBytes:     agg.SumDiskRead,
		DiskWriteBytes:    agg.SumDiskWrite,
		RequestCount:      agg.SumRequestCount,
		ErrorCount:        agg.SumErrorCount,
		SampleCount:       agg.Count,
	}
	
	if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
		logger.Warn("[Orchestrator] Failed to create hourly aggregate for %s at %s: %v", deploymentID, hourStart, err)
		return false, err
	}
	
	logger.Debug("[Orchestrator] ✓ Created hourly aggregate for %s at %s (CPU: %.2f%%, Memory: %.2f bytes/sec, SampleCount: %d)", 
		deploymentID, hourStart, agg.AvgCPUUsage, agg.AvgMemoryUsage, agg.Count)
	return true, nil
}

// storageInfo contains storage information for a deployment
type storageInfo struct {
	ImageSize     int64 // Docker image size in bytes
	VolumeSize    int64 // Total volume size in bytes
	ContainerDisk int64 // Container root filesystem usage in bytes
	TotalStorage  int64 // Total storage (image + volumes + container disk)
}

// calculateStorage calculates total storage for a deployment
func (os *OrchestratorService) calculateStorage(ctx context.Context, imageName string, containerIDs []string) (*storageInfo, error) {
	info := &storageInfo{}

	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dcli.Close()

	// 1. Get image size
	if imageName != "" {
		imageSize, err := getImageSize(ctx, imageName)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get image size for %s: %v", imageName, err)
		} else {
			info.ImageSize = imageSize
		}
	}

	// 2. Get volume sizes and container disk usage for all containers
	totalVolumeSize := int64(0)
	totalContainerDisk := int64(0)

	for _, containerID := range containerIDs {
		// Get volume sizes
		volumeSize, err := os.getContainerVolumeSize(ctx, dcli, containerID)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get volume size for container %s: %v", containerID, err)
		} else {
			totalVolumeSize += volumeSize
		}

		// Get container root filesystem disk usage
		containerDisk, err := os.getContainerDiskUsage(ctx, dcli, containerID)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get container disk usage for %s: %v", containerID, err)
		} else {
			totalContainerDisk += containerDisk
		}
	}

	info.VolumeSize = totalVolumeSize
	info.ContainerDisk = totalContainerDisk
	info.TotalStorage = info.ImageSize + info.VolumeSize + info.ContainerDisk

	return info, nil
}

// getImageSize gets the size of a Docker image in bytes
func getImageSize(ctx context.Context, imageName string) (int64, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get image size: %w", err)
	}

	sizeStr := strings.TrimSpace(string(output))
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse image size: %w", err)
	}

	return size, nil
}

// getContainerVolumeSize calculates total size of all volumes attached to a container
func (os *OrchestratorService) getContainerVolumeSize(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect container: %w", err)
	}

	totalSize := int64(0)

	volumes, err := dcli.GetContainerVolumes(ctx, containerID)
	if err != nil {
		logger.Warn("[getContainerVolumeSize] Failed to get volumes for container %s: %v", containerID, err)
		for _, mount := range containerInfo.Mounts {
			if mount.Type == "volume" || (mount.Type == "bind" && strings.HasPrefix(mount.Source, "/var/lib/obiente/volumes")) {
				size, err := getDirectorySize(ctx, mount.Source)
				if err != nil {
					logger.Warn("[getContainerVolumeSize] Failed to get size for volume %s: %v", mount.Source, err)
					continue
				}
				totalSize += size
			}
		}
	} else {
		for _, volume := range volumes {
			size, err := getDirectorySize(ctx, volume.Source)
			if err != nil {
				logger.Warn("[getContainerVolumeSize] Failed to get size for volume %s: %v", volume.Source, err)
				continue
			}
			totalSize += size
		}
	}

	return totalSize, nil
}

// getContainerDiskUsage gets the root filesystem disk usage of a container
func (os *OrchestratorService) getContainerDiskUsage(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	cmd := []string{"sh", "-c", "du -sb / 2>/dev/null | cut -f1"}
	output, err := dcli.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		// Fallback: try df command
		cmd = []string{"df", "-B1", "/"}
		output, err = dcli.ContainerExecRun(ctx, containerID, cmd)
		if err != nil {
			return 0, fmt.Errorf("failed to get disk usage: %w", err)
		}

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) < 2 {
			return 0, fmt.Errorf("unexpected df output format")
		}

		fields := strings.Fields(lines[1])
		if len(fields) < 3 {
			return 0, fmt.Errorf("unexpected df output format")
		}

		var used int64
		if _, err := fmt.Sscanf(fields[2], "%d", &used); err != nil {
			return 0, fmt.Errorf("failed to parse used size: %w", err)
		}

		return used, nil
	}

	sizeStr := strings.TrimSpace(output)
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse disk usage: %w", err)
	}

	return size, nil
}

// getDirectorySize calculates the total size of a directory
func getDirectorySize(ctx context.Context, path string) (int64, error) {
	cmd := exec.CommandContext(ctx, "du", "-sb", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected du output format")
	}

	var size int64
	if _, err := fmt.Sscanf(parts[0], "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return size, nil
}

// updateStoragePeriodically updates storage usage for all running deployments
func (os *OrchestratorService) updateStoragePeriodically() {
	// Run every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Updating storage for all running deployments...")
			
			// Get all running deployments
			var locations []database.DeploymentLocation
			if err := database.DB.Where("status = ?", "running").Find(&locations).Error; err != nil {
				logger.Warn("[Orchestrator] Failed to get running deployments: %v", err)
				continue
			}

			// Group by deployment ID
			deploymentMap := make(map[string][]database.DeploymentLocation)
			for _, loc := range locations {
				deploymentMap[loc.DeploymentID] = append(deploymentMap[loc.DeploymentID], loc)
			}

			logger.Debug("[Orchestrator] Updating storage for %d deployments", len(deploymentMap))

			// Process deployments in parallel batches
			const batchSize = 5 // Process 5 deployments concurrently
			var wg sync.WaitGroup
			var mu sync.Mutex
			updatedCount := 0
			errorCount := 0

			deploymentIDs := make([]string, 0, len(deploymentMap))
			for depID := range deploymentMap {
				deploymentIDs = append(deploymentIDs, depID)
			}

			for i := 0; i < len(deploymentIDs); i += batchSize {
				end := i + batchSize
				if end > len(deploymentIDs) {
					end = len(deploymentIDs)
				}
				batch := deploymentIDs[i:end]

				for _, deploymentID := range batch {
					wg.Add(1)
					go func(depID string) {
						defer wg.Done()
						
						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
						defer cancel()

						// Get container IDs for this deployment
						containerIDs := make([]string, 0)
						if locs, ok := deploymentMap[depID]; ok {
							for _, loc := range locs {
								if loc.ContainerID != "" {
									containerIDs = append(containerIDs, loc.ContainerID)
								}
							}
						}

						if len(containerIDs) == 0 {
							return
						}

						// Get deployment to find image name
						var deployment database.Deployment
						if err := database.DB.Where("id = ?", depID).First(&deployment).Error; err != nil {
							logger.Warn("[Orchestrator] Failed to get deployment %s: %v", depID, err)
							return
						}

						imageName := ""
						if deployment.Image != nil {
							imageName = *deployment.Image
						}

						// Calculate storage
						storageInfo, err := os.calculateStorage(ctx, imageName, containerIDs)
						if err != nil {
							logger.Warn("[Orchestrator] Failed to calculate storage for deployment %s: %v", depID, err)
							mu.Lock()
							errorCount++
							mu.Unlock()
							return
						}

						// Update storage in database
						if err := database.DB.Model(&database.Deployment{}).
							Where("id = ?", depID).
							Update("storage_bytes", storageInfo.TotalStorage).Error; err != nil {
							logger.Warn("[Orchestrator] Failed to update storage for deployment %s: %v", depID, err)
							mu.Lock()
							errorCount++
							mu.Unlock()
							return
						}

						mu.Lock()
						updatedCount++
						mu.Unlock()
					}(deploymentID)
				}

				wg.Wait()
			}

			logger.Debug("[Orchestrator] Storage update completed: %d updated, %d errors", updatedCount, errorCount)
		case <-os.ctx.Done():
			return
		}
	}
}

// cleanupBuildHistory deletes build history older than 30 days
func (os *OrchestratorService) cleanupBuildHistory() {
	// Run daily at midnight
	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	initialDelay := nextMidnight.Sub(now)
	
	// Wait for first run
	select {
	case <-time.After(initialDelay):
		// Run immediately after initial delay
	case <-os.ctx.Done():
		return
	}
	
	// Run daily after initial delay
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			logger.Info("[Orchestrator] Running build history cleanup...")
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			
			// Delete builds older than 30 days
			buildHistoryRepo := database.NewBuildHistoryRepository(database.DB)
			buildIDs, deletedCount, err := buildHistoryRepo.DeleteBuildsOlderThan(ctx, 30*24*time.Hour)
			if err != nil {
				logger.Warn("[Orchestrator] Failed to cleanup build history: %v", err)
			} else {
				// Delete logs from TimescaleDB for the deleted builds
				if len(buildIDs) > 0 {
					buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
					for _, buildID := range buildIDs {
						if err := buildLogsRepo.DeleteBuildLogs(ctx, buildID); err != nil {
							logger.Warn("[Orchestrator] Failed to delete logs for build %s: %v", buildID, err)
							// Continue with other builds
						}
					}
					logger.Info("[Orchestrator] Deleted logs for %d build(s) from TimescaleDB", len(buildIDs))
				}
				logger.Info("[Orchestrator] Deleted %d build(s) older than 30 days", deletedCount)
			}
		case <-os.ctx.Done():
			return
		}
	}
}

// cleanupStrayContainers finds and stops containers that are running but don't exist in the database
// After 7 days, it also deletes associated volumes
func (os *OrchestratorService) cleanupStrayContainers() {
	// Run every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup
	os.runStrayContainerCleanup()

	for {
		select {
		case <-ticker.C:
			os.runStrayContainerCleanup()
		case <-os.ctx.Done():
			return
		}
	}
}

// runStrayContainerCleanup performs the actual cleanup work
func (os *OrchestratorService) runStrayContainerCleanup() {
	logger.Info("[Orchestrator] Running stray container cleanup...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to create Docker client for stray cleanup: %v", err)
		return
	}
	defer dcli.Close()

	// Get node ID
	nodeID := os.deploymentManager.GetNodeID()

	// Get all containers managed by Obiente
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "cloud.obiente.managed=true")

	containers, err := os.deploymentManager.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		logger.Warn("[Orchestrator] Failed to list containers for stray cleanup: %v", err)
		return
	}

	// Get all container IDs from deployment_locations table
	var dbContainerIDs []string
	if err := database.DB.Table("deployment_locations").
		Select("container_id").
		Where("node_id = ?", nodeID).
		Pluck("container_id", &dbContainerIDs).Error; err != nil {
		logger.Warn("[Orchestrator] Failed to query deployment locations: %v", err)
		return
	}

	// Build a map for fast lookup
	dbContainerMap := make(map[string]bool)
	for _, id := range dbContainerIDs {
		dbContainerMap[id] = true
	}

	// Find stray containers (running but not in DB)
	// ContainerList returns []container.Summary
	var strayContainers []container.Summary
	for _, container := range containers {
		// Verify container has cloud.obiente.managed=true (should already be filtered, but double-check)
		if container.Labels["cloud.obiente.managed"] != "true" {
			continue
		}
		
		// Check if container is running
		containerInfo, err := os.deploymentManager.dockerClient.ContainerInspect(ctx, container.ID)
		if err != nil {
			logger.Debug("[Orchestrator] Failed to inspect container %s: %v", container.ID[:12], err)
			continue
		}

		// Only process running containers
		if !containerInfo.State.Running {
			continue
		}

		// Check if container exists in database
		if !dbContainerMap[container.ID] {
			strayContainers = append(strayContainers, container)
		}
	}

	if len(strayContainers) == 0 {
		logger.Debug("[Orchestrator] No stray containers found")
	} else {
		logger.Info("[Orchestrator] Found %d stray container(s)", len(strayContainers))
	}

	// Stop stray containers and record them
	now := time.Now()
	for _, container := range strayContainers {
		// Check if we've already recorded this container
		var existingStray database.StrayContainer
		if err := database.DB.Where("container_id = ?", container.ID).First(&existingStray).Error; err == nil {
			// Already recorded, skip
			continue
		}

		// Stop the container
		logger.Info("[Orchestrator] Stopping stray container %s", container.ID[:12])
		timeout := 30 * time.Second
		if err := dcli.StopContainer(ctx, container.ID, timeout); err != nil {
			logger.Warn("[Orchestrator] Failed to stop stray container %s: %v", container.ID[:12], err)
			continue
		}

		// Record in database
		stray := database.StrayContainer{
			ContainerID: container.ID,
			NodeID:      nodeID,
			StoppedAt:   now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := database.DB.Create(&stray).Error; err != nil {
			logger.Warn("[Orchestrator] Failed to record stray container %s: %v", container.ID[:12], err)
		} else {
			logger.Info("[Orchestrator] Recorded stray container %s (stopped at %s)", container.ID[:12], now.Format(time.RFC3339))
		}
	}

	// Delete volumes for containers stopped more than 7 days ago
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	var oldStrayContainers []database.StrayContainer
	if err := database.DB.Where("stopped_at < ? AND volumes_deleted_at IS NULL", sevenDaysAgo).Find(&oldStrayContainers).Error; err != nil {
		// Check if error is due to missing table (table might not exist yet)
		if database.DB.Migrator().HasTable(&database.StrayContainer{}) {
			logger.Warn("[Orchestrator] Failed to query old stray containers: %v", err)
		} else {
			logger.Debug("[Orchestrator] stray_containers table does not exist yet, skipping volume cleanup")
		}
		return
	}

	if len(oldStrayContainers) == 0 {
		logger.Debug("[Orchestrator] No old stray containers to clean up")
		return
	}

	logger.Info("[Orchestrator] Cleaning up volumes for %d old stray container(s)", len(oldStrayContainers))

	for _, stray := range oldStrayContainers {
		// Verify container has cloud.obiente tags before deleting volumes
		// Check container labels to ensure it's an Obiente-managed container
		// Even though container might be gone, we can check if volumes are Obiente volumes
		containerInfo, err := os.deploymentManager.dockerClient.ContainerInspect(ctx, stray.ContainerID)
		hasObienteTags := false
		containerExists := err == nil
		if containerExists && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			// Container still exists - verify it has cloud.obiente tags
			if containerInfo.Config.Labels["cloud.obiente.managed"] == "true" {
				hasObienteTags = true
			}
		} else {
			// Container is gone - we can only verify by checking volume paths
			// We'll only delete volumes if they're Obiente volumes (path-based check)
			// For Docker volumes, we'll skip if container is gone (too risky)
			hasObienteTags = true // Assume it had tags since it's in stray_containers table
		}

		// Only proceed if container has Obiente tags (or container is gone and we can verify via paths)
		if !hasObienteTags {
			logger.Warn("[Orchestrator] Skipping volume deletion for container %s - missing cloud.obiente tags", stray.ContainerID[:12])
			continue
		}

		// Get volumes for this container
		volumes, err := dcli.GetContainerVolumes(ctx, stray.ContainerID)
		if err != nil {
			logger.Warn("[Orchestrator] Failed to get volumes for container %s: %v", stray.ContainerID[:12], err)
			// Mark as deleted even if we couldn't get volumes (container might be gone)
			database.DB.Model(&stray).Update("volumes_deleted_at", now)
			continue
		}

		// Delete each volume - only delete Obiente-managed volumes
		for _, volume := range volumes {
			// For Obiente Cloud volumes (bind mounts), delete the directory
			// These are stored in /var/lib/obiente/volumes/{deploymentID}/{volumeName}
			if strings.HasPrefix(volume.Source, "/var/lib/obiente/volumes") {
				logger.Info("[Orchestrator] Deleting Obiente volume at %s", volume.Source)
				if err := exec.Command("rm", "-rf", volume.Source).Run(); err != nil {
					logger.Warn("[Orchestrator] Failed to delete Obiente volume %s: %v", volume.Source, err)
				} else {
					logger.Info("[Orchestrator] Deleted Obiente volume %s", volume.Source)
				}
			} else if volume.IsNamed {
				// For Docker named volumes, only delete if container still exists and has cloud.obiente tags
				// This is safer - we don't want to delete volumes from containers that might not be Obiente-managed
				if containerExists && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
					if containerInfo.Config.Labels["cloud.obiente.managed"] == "true" {
						logger.Info("[Orchestrator] Removing Docker volume %s (container has cloud.obiente tags)", volume.Name)
						// Extract volume name from mount name if needed
						volumeName := volume.Name
						if volumeName == "" {
							// Try to extract from path
							if strings.Contains(volume.Source, "/volumes/") {
								parts := strings.Split(volume.Source, "/volumes/")
								if len(parts) > 1 {
									volumeName = strings.Split(parts[1], "/")[0]
								}
							}
						}
						if volumeName != "" {
							if err := os.deploymentManager.dockerClient.VolumeRemove(ctx, volumeName, false); err != nil {
								logger.Warn("[Orchestrator] Failed to remove Docker volume %s: %v", volumeName, err)
							} else {
								logger.Info("[Orchestrator] Removed Docker volume %s", volumeName)
							}
						}
					} else {
						logger.Debug("[Orchestrator] Skipping Docker volume %s - container missing cloud.obiente tags", volume.Name)
					}
				} else {
					logger.Debug("[Orchestrator] Skipping Docker volume %s - container no longer exists (cannot verify tags)", volume.Name)
				}
			}
			// Anonymous volumes are typically cleaned up when the container is removed
		}

		// Mark volumes as deleted
		database.DB.Model(&stray).Update("volumes_deleted_at", now)
		logger.Info("[Orchestrator] Marked volumes as deleted for container %s", stray.ContainerID[:12])
	}
}
