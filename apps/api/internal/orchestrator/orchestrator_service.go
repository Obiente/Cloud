package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"api/internal/database"
	"api/internal/registry"

	"gorm.io/gorm"
)

// OrchestratorService is the main orchestration service that runs continuously
type OrchestratorService struct {
	deploymentManager *DeploymentManager
	serviceRegistry   *registry.ServiceRegistry
	healthChecker     *registry.HealthChecker
	syncInterval      time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
	// Track previous stats per container to calculate deltas
	previousStats map[string]*containerStats
	statsMutex     sync.RWMutex
}

// NewOrchestratorService creates a new orchestrator service
func NewOrchestratorService(strategy string, maxDeploymentsPerNode int, syncInterval time.Duration) (*OrchestratorService, error) {
	deploymentManager, err := NewDeploymentManager(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, err
	}

	serviceRegistry, err := registry.NewServiceRegistry()
	if err != nil {
		return nil, err
	}

	healthChecker := registry.NewHealthChecker(serviceRegistry, 1*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())

	return &OrchestratorService{
		deploymentManager: deploymentManager,
		serviceRegistry:   serviceRegistry,
		healthChecker:     healthChecker,
		syncInterval:      syncInterval,
		ctx:               ctx,
		cancel:            cancel,
		previousStats:     make(map[string]*containerStats),
	}, nil
}

// Start begins all background orchestration tasks
func (os *OrchestratorService) Start() {
	log.Println("[Orchestrator] Starting orchestration service...")

	// Start periodic sync with Docker
	os.serviceRegistry.StartPeriodicSync(os.ctx, os.syncInterval)
	log.Printf("[Orchestrator] Started periodic sync (interval: %v)", os.syncInterval)

	// Start health checking
	os.healthChecker.Start(os.ctx)
	log.Println("[Orchestrator] Started health checker")

	// Start metrics collection
	go os.collectMetrics()
	log.Println("[Orchestrator] Started metrics collection")

	// Start cleanup tasks
	go os.cleanupTasks()
	log.Println("[Orchestrator] Started cleanup tasks")

	// Start usage aggregation (hourly)
	go os.aggregateUsage()
	log.Println("[Orchestrator] Started usage aggregation")

	log.Println("[Orchestrator] Orchestration service started successfully")
}

// Stop gracefully stops the orchestrator service
func (os *OrchestratorService) Stop() {
	log.Println("[Orchestrator] Stopping orchestration service...")
	os.cancel()

	if err := os.deploymentManager.Close(); err != nil {
		log.Printf("[Orchestrator] Error closing deployment manager: %v", err)
	}
	if err := os.serviceRegistry.Close(); err != nil {
		log.Printf("[Orchestrator] Error closing service registry: %v", err)
	}

	log.Println("[Orchestrator] Orchestration service stopped")
}

// collectMetrics periodically collects metrics from all deployments
// Collects every 5 seconds for real-time monitoring
func (os *OrchestratorService) collectMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Log less frequently to avoid spam (every 30 seconds)
	lastLogTime := time.Now()

	for {
		select {
		case <-ticker.C:
			shouldLog := time.Since(lastLogTime) >= 30*time.Second
			if shouldLog {
				log.Printf("[Orchestrator] Collecting metrics (interval: 5s)...")
				lastLogTime = time.Now()
			}
			
			// Get all running deployment locations on this node
			nodeID := os.serviceRegistry.NodeID()
			locations, err := os.serviceRegistry.GetNodeDeployments(nodeID)
			if err != nil {
				log.Printf("[Orchestrator] Failed to get node deployments: %v", err)
				continue
			}

			if len(locations) == 0 {
				// No deployments running on this node - this is fine, just continue
				if shouldLog {
					log.Printf("[Orchestrator] No deployments running on node %s", nodeID)
				}
				continue
			}

			for _, location := range locations {
				// GetNodeDeployments already filters for status="running", but double-check for safety
				if location.Status != "running" {
					continue
				}

				// Get container stats from Docker (these are cumulative since container start)
				currentStats, err := os.getContainerStats(location.ContainerID)
				if err != nil {
					log.Printf("[Orchestrator] Failed to get stats for container %s: %v", location.ContainerID, err)
					continue
				}

				// Get previous stats to calculate deltas
				os.statsMutex.RLock()
				prevStats, hasPrev := os.previousStats[location.ContainerID]
				os.statsMutex.RUnlock()

				// Calculate deltas (incremental values since last measurement)
				// Docker stats are cumulative for: Network Rx/Tx bytes, Disk Read/Write bytes
				// Docker stats are instantaneous for: CPU %, Memory usage bytes
				var networkRxDelta, networkTxDelta, diskReadDelta, diskWriteDelta int64
				if hasPrev {
					// Network and disk I/O are cumulative (total since container start)
					// Calculate deltas to get incremental I/O per interval
					networkRxDelta = currentStats.NetworkRxBytes - prevStats.NetworkRxBytes
					networkTxDelta = currentStats.NetworkTxBytes - prevStats.NetworkTxBytes
					diskReadDelta = currentStats.DiskReadBytes - prevStats.DiskReadBytes
					diskWriteDelta = currentStats.DiskWriteBytes - prevStats.DiskWriteBytes
					
					// Ensure deltas are non-negative (handle container restarts)
					if networkRxDelta < 0 {
						networkRxDelta = currentStats.NetworkRxBytes
					}
					if networkTxDelta < 0 {
						networkTxDelta = currentStats.NetworkTxBytes
					}
					if diskReadDelta < 0 {
						diskReadDelta = currentStats.DiskReadBytes
					}
					if diskWriteDelta < 0 {
						diskWriteDelta = currentStats.DiskWriteBytes
					}
				} else {
					// First measurement - use current values as baseline (will be small or zero)
					// Next measurement will show the actual delta
					networkRxDelta = 0
					networkTxDelta = 0
					diskReadDelta = 0
					diskWriteDelta = 0
				}

				// Record metrics with incremental values
				metric := &database.DeploymentMetrics{
					DeploymentID:    location.DeploymentID,
					ContainerID:     location.ContainerID,
					NodeID:          location.NodeID,
					CPUUsage:        currentStats.CPUUsage,
					MemoryUsage:     currentStats.MemoryUsage,
					NetworkRxBytes:  networkRxDelta,
					NetworkTxBytes:  networkTxDelta,
					DiskReadBytes:   diskReadDelta,
					DiskWriteBytes:  diskWriteDelta,
					Timestamp:       time.Now(),
				}

				if err := database.RecordMetrics(metric); err != nil {
					log.Printf("[Orchestrator] Failed to record metrics for deployment %s: %v", location.DeploymentID, err)
				} else {
					// Store current stats for next delta calculation
					os.statsMutex.Lock()
					os.previousStats[location.ContainerID] = currentStats
					os.statsMutex.Unlock()
				}

				// Update deployment location with current CPU/memory
				_ = os.serviceRegistry.UpdateDeploymentMetrics(location.ContainerID, currentStats.CPUUsage, currentStats.MemoryUsage)
			}
			
			if shouldLog {
				log.Printf("[Orchestrator] Collected metrics for %d deployments", len(locations))
			}
		case <-os.ctx.Done():
			return
		}
	}
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
		log.Printf("[getContainerStats] No BlkioStats found for container %s - disk I/O may not be available", containerID[:12])
	} else if diskRead == 0 && diskWrite == 0 {
		// Log operation names we're seeing (for debugging)
		ops := make([]string, 0, len(statsJSON.BlkioStats.IoServiceBytesRecursive))
		for _, ioStat := range statsJSON.BlkioStats.IoServiceBytesRecursive {
			ops = append(ops, ioStat.Op)
		}
		log.Printf("[getContainerStats] BlkioStats available for container %s but no read/write found. Operations seen: %v", containerID[:12], ops)
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
			log.Println("[Orchestrator] Running cleanup tasks...")
			
			// Keep last 24 hours of raw metrics for real-time viewing
			// Aggregate older metrics into hourly summaries
			aggregateCutoff := time.Now().Add(-24 * time.Hour).Truncate(time.Hour)
			
			// Get all deployments that have metrics older than cutoff
			var deploymentIDs []string
			database.DB.Table("deployment_metrics").
				Select("DISTINCT deployment_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("deployment_id", &deploymentIDs)
			
			if len(deploymentIDs) == 0 {
				log.Println("[Orchestrator] No old metrics to aggregate")
				log.Println("[Orchestrator] Cleanup tasks completed")
				continue
			}
			
			log.Printf("[Orchestrator] Aggregating metrics for %d deployments", len(deploymentIDs))
			
			totalAggregated := 0
			totalDeleted := int64(0)
			
			for _, deploymentID := range deploymentIDs {
				// Get org ID once
				var orgID string
				database.DB.Table("deployments").
					Select("organization_id").
					Where("id = ?", deploymentID).
					Pluck("organization_id", &orgID)
				
				// Find the oldest metric for this deployment
				var oldestTime time.Time
				database.DB.Table("deployment_metrics").
					Select("MIN(timestamp)").
					Where("deployment_id = ? AND timestamp < ?", deploymentID, aggregateCutoff).
					Scan(&oldestTime)
				
				if oldestTime.IsZero() {
					continue
				}
				
				// Aggregate hour by hour
				currentHour := oldestTime.Truncate(time.Hour)
				deletedInDeployment := int64(0)
				
				for currentHour.Before(aggregateCutoff) {
					nextHour := currentHour.Add(1 * time.Hour)
					
					// Check if hourly aggregate already exists
					var existingHourly database.DeploymentUsageHourly
					err := database.DB.Where("deployment_id = ? AND hour = ?", deploymentID, currentHour).
						First(&existingHourly).Error
					
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// Aggregate metrics for this hour
						// For multi-container deployments: we need to aggregate per timestamp first,
						// then average across timestamps within the hour
						// CPU: average across timestamps (each timestamp is the sum/average of all containers at that time)
						// Memory: average across timestamps (each timestamp is the sum of all containers)
						// Network/Disk: sum all incremental values (already handled correctly)
						type hourlyAgg struct {
							AvgCPUUsage      float64
							SumMemoryUsage   float64 // Sum first, then we'll average across samples
							AvgMemoryUsage   float64
							SumNetworkRx     int64
							SumNetworkTx     int64
							SumDiskRead      int64
							SumDiskWrite     int64
							SumRequestCount  int64
							SumErrorCount    int64
							Count            int64
							TimestampCount   int64 // Distinct timestamps to calculate proper averages
						}
						var agg hourlyAgg
						
						// First, aggregate per timestamp (sum memory, avg CPU per timestamp)
						// Then average those values across the hour
						err := database.DB.Table("deployment_metrics").
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
							// For memory: average the total memory usage across timestamps
							// (If we have 3 containers per timestamp, we sum them, then average across timestamps)
							if agg.TimestampCount > 0 {
								agg.AvgMemoryUsage = agg.SumMemoryUsage / float64(agg.TimestampCount)
							} else {
								agg.AvgMemoryUsage = agg.SumMemoryUsage / float64(agg.Count)
							}
							
							// Create hourly aggregate
							hourlyUsage := database.DeploymentUsageHourly{
								DeploymentID:     deploymentID,
								OrganizationID:    orgID,
								Hour:              currentHour,
								AvgCPUUsage:       agg.AvgCPUUsage,
								AvgMemoryUsage:    int64(agg.AvgMemoryUsage),
								BandwidthRxBytes:  agg.SumNetworkRx,
								BandwidthTxBytes:  agg.SumNetworkTx,
								DiskReadBytes:     agg.SumDiskRead,
								DiskWriteBytes:    agg.SumDiskWrite,
								RequestCount:      agg.SumRequestCount,
								ErrorCount:        agg.SumErrorCount,
								SampleCount:       agg.Count,
							}
							
							if err := database.DB.Create(&hourlyUsage).Error; err != nil {
								log.Printf("[Orchestrator] Failed to create hourly aggregate for %s at %s: %v", deploymentID, currentHour, err)
							} else {
								totalAggregated++
								
								// Delete the raw metrics for this hour
								result := database.DB.Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
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
				
				if deletedInDeployment > 0 {
					totalDeleted += deletedInDeployment
				}
			}
			
			log.Printf("[Orchestrator] Aggregated %d hours, deleted %d raw metrics, cleanup tasks completed", totalAggregated, totalDeleted)
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
			log.Println("[Orchestrator] Aggregating usage...")
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
				database.DB.Table("deployment_metrics dm").
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
					database.DB.Table("deployment_usage_hourly duh").
						Select("COALESCE(SUM(duh.avg_memory_usage * 3600), 0) as memory_byte_seconds").
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
				database.DB.Table("deployment_metrics dm").
					Select("COALESCE(SUM(dm.network_rx_bytes), 0) as rx_bytes, COALESCE(SUM(dm.network_tx_bytes), 0) as tx_bytes").
					Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawCutoffBandwidth).
						Scan(&rawBandwidth)
					
					// Sum from hourly aggregates (older than 24 hours, within month)
					var hourlyBandwidth bandwidthRow
				if rawCutoffBandwidth.After(monthStart) {
					database.DB.Table("deployment_usage_hourly duh").
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
