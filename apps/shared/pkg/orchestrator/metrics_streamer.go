package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/metrics"
	"github.com/obiente/cloud/apps/shared/pkg/registry"
)

// LiveMetric represents a live metric in memory
type LiveMetric struct {
	ResourceType   string    // "deployment" or "gameserver"
	ResourceID     string    // DeploymentID or GameServerID
	ContainerID    string    `json:"container_id"`
	NodeID         string    `json:"node_id"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    int64     `json:"memory_usage"`
	NetworkRxBytes int64     `json:"network_rx_bytes"`
	NetworkTxBytes int64     `json:"network_tx_bytes"`
	DiskReadBytes  int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64     `json:"disk_write_bytes"`
	RequestCount   int64     `json:"request_count"` // Only for deployments
	ErrorCount     int64     `json:"error_count"`   // Only for deployments
	Timestamp      time.Time `json:"timestamp"`
}

// DeploymentID returns the deployment ID (for backward compatibility)
func (m *LiveMetric) DeploymentID() string {
	if m.ResourceType == "deployment" {
		return m.ResourceID
	}
	return ""
}

// GameServerID returns the game server ID
func (m *LiveMetric) GameServerID() string {
	if m.ResourceType == "gameserver" {
		return m.ResourceID
	}
	return ""
}

// MetricsStreamer handles live metrics streaming and periodic storage
type MetricsStreamer struct {
	serviceRegistry *registry.ServiceRegistry
	previousStats   map[string]*ContainerStats
	statsMutex      sync.RWMutex

	// Live metrics cache: deploymentID -> []LiveMetric (last N minutes)
	liveMetrics      map[string][]LiveMetric
	liveMetricsMutex sync.RWMutex

	// Subscribers: deploymentID -> []chan LiveMetric with metadata
	subscribers      map[string][]*subscriberChannel
	subscribersMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	// Configuration
	config *MetricsConfig
	
	// Retry queue for failed database writes
	retryQueue *MetricsRetryQueue
	
	// Circuit breaker for Docker API
	circuitBreaker *CircuitBreaker
	
	// Observability stats
	stats *MetricsStats
	
	// Graceful degradation
	collectionRateMultiplier float64 // Reduces collection rate under load
	degradationMutex         sync.RWMutex
}

// subscriberChannel wraps a channel with metadata for backpressure detection
type subscriberChannel struct {
	ch           chan LiveMetric
	lastSendTime time.Time
	overflowCount int64
}

// NewMetricsStreamer creates a new metrics streamer
func NewMetricsStreamer(serviceRegistry *registry.ServiceRegistry) *MetricsStreamer {
	ctx, cancel := context.WithCancel(context.Background())
	config := LoadMetricsConfig()
	
	return &MetricsStreamer{
		serviceRegistry:          serviceRegistry,
		previousStats:            make(map[string]*ContainerStats),
		liveMetrics:              make(map[string][]LiveMetric),
		subscribers:              make(map[string][]*subscriberChannel),
		ctx:                      ctx,
		cancel:                   cancel,
		config:                   config,
		retryQueue:               NewMetricsRetryQueueWithConfig(config),
		circuitBreaker:           NewCircuitBreaker(config.CircuitBreakerFailureThreshold, config.CircuitBreakerCooldownPeriod, config.CircuitBreakerHalfOpenMaxCalls),
		stats:                    NewMetricsStats(),
		collectionRateMultiplier: 1.0, // Start at full speed
	}
}

// Start begins metrics collection and streaming
func (ms *MetricsStreamer) Start() {
	// Start live collection (streams to subscribers, stores in memory)
	go ms.collectLiveMetrics()

	// Start periodic storage (batches writes to DB every minute)
	go ms.storeMetricsBatch()

	// Start cleanup of old live metrics
	go ms.cleanupLiveMetrics()
	
	// Start retry processor for failed writes
	go ms.processRetries()
	
	// Start cleanup of stale previous stats
	go ms.cleanupStaleStats()
	
	// Start health checker
	go ms.healthCheck()
	
	// Start backpressure monitor
	go ms.monitorBackpressure()
}

// Stop stops the metrics streamer
func (ms *MetricsStreamer) Stop() {
	ms.cancel()

	// Close all subscriber channels
	ms.subscribersMutex.Lock()
	for _, subs := range ms.subscribers {
		for _, sub := range subs {
			close(sub.ch)
		}
	}
	ms.subscribers = make(map[string][]*subscriberChannel)
	ms.subscribersMutex.Unlock()
}

// Subscribe adds a subscriber for a deployment's metrics
func (ms *MetricsStreamer) Subscribe(deploymentID string) <-chan LiveMetric {
	ch := make(chan LiveMetric, ms.config.SubscriberChannelBufferSize)
	sub := &subscriberChannel{
		ch:           ch,
		lastSendTime: time.Now(),
	}

	ms.subscribersMutex.Lock()
	ms.subscribers[deploymentID] = append(ms.subscribers[deploymentID], sub)
	totalSubs := 0
	for _, s := range ms.subscribers {
		totalSubs += len(s)
	}
	ms.subscribersMutex.Unlock()
	
	ms.stats.UpdateSubscriberStats(totalSubs, 0, 0)

	return ch
}

// Unsubscribe removes a subscriber
func (ms *MetricsStreamer) Unsubscribe(deploymentID string, ch <-chan LiveMetric) {
	ms.subscribersMutex.Lock()
	defer ms.subscribersMutex.Unlock()

	chans, exists := ms.subscribers[deploymentID]
	if !exists {
		return
	}

	for i, sub := range chans {
		if sub.ch == ch {
			// Remove from slice
			ms.subscribers[deploymentID] = append(chans[:i], chans[i+1:]...)
			close(sub.ch)
			break
		}
	}
	
	if len(ms.subscribers[deploymentID]) == 0 {
		delete(ms.subscribers, deploymentID)
	}
	
	// Update stats
	totalSubs := 0
	for _, s := range ms.subscribers {
		totalSubs += len(s)
	}
	ms.stats.UpdateSubscriberStats(totalSubs, 0, 0)
}

// GetLatestMetrics returns the latest metrics for a deployment or game server
func (ms *MetricsStreamer) GetLatestMetrics(resourceID string) []LiveMetric {
	ms.liveMetricsMutex.RLock()
	defer ms.liveMetricsMutex.RUnlock()

	metrics, exists := ms.liveMetrics[resourceID]
	if !exists {
		return []LiveMetric{}
	}

	return metrics
}

// collectLiveMetrics collects metrics in parallel and streams to subscribers
func (ms *MetricsStreamer) collectLiveMetrics() {
	// Use configurable interval with graceful degradation multiplier
	ms.degradationMutex.RLock()
	interval := time.Duration(float64(ms.config.CollectionInterval) * ms.collectionRateMultiplier)
	ms.degradationMutex.RUnlock()
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nodeID := ms.serviceRegistry.NodeID()

			// Collect deployment metrics
			deploymentLocations, err := ms.serviceRegistry.GetNodeDeployments(nodeID)
			if err != nil {
				// Silently continue - will retry next cycle
			}

			// Collect game server metrics
			gameServerLocations, err := ms.serviceRegistry.GetNodeGameServers(nodeID)
			if err != nil {
				// Silently continue - will retry next cycle
			}

			var allMetrics []LiveMetric
			containersFailed := 0

			if len(deploymentLocations) > 0 {
				// Collect stats in parallel using worker pool
				deploymentMetrics, failed := ms.collectDeploymentStatsParallel(deploymentLocations, false)
				allMetrics = append(allMetrics, deploymentMetrics...)
				containersFailed += failed
			}

			if len(gameServerLocations) > 0 {
				// Collect stats using the same Docker container logic as deployments
				gameServerMetrics, failed := ms.collectGameServerStatsParallel(gameServerLocations, false)
				allMetrics = append(allMetrics, gameServerMetrics...)
				containersFailed += failed
			}

			if len(allMetrics) == 0 {
				continue
			}

			// Store in live cache and stream to subscribers
			now := time.Now()
			for _, metric := range allMetrics {
				// Add to live cache (keyed by resource ID)
				ms.liveMetricsMutex.Lock()
				if ms.liveMetrics[metric.ResourceID] == nil {
					ms.liveMetrics[metric.ResourceID] = make([]LiveMetric, 0)
				}
				ms.liveMetrics[metric.ResourceID] = append(ms.liveMetrics[metric.ResourceID], metric)

				// Trim old metrics (keep only last N minutes or max size)
				retentionCutoff := now.Add(-ms.config.LiveRetention)
				trimmed := ms.liveMetrics[metric.ResourceID][:0]
				for _, m := range ms.liveMetrics[metric.ResourceID] {
					if m.Timestamp.After(retentionCutoff) {
						trimmed = append(trimmed, m)
					}
				}
				// Enforce max size per resource
				if len(trimmed) > ms.config.MaxLiveMetricsPerDeployment {
					// Keep only the most recent metrics
					startIdx := len(trimmed) - ms.config.MaxLiveMetricsPerDeployment
					trimmed = trimmed[startIdx:]
				}
				ms.liveMetrics[metric.ResourceID] = trimmed
				ms.liveMetricsMutex.Unlock()

				// Stream to subscribers (non-blocking with backpressure detection)
				ms.subscribersMutex.RLock()
				subs := ms.subscribers[metric.ResourceID]
				ms.subscribersMutex.RUnlock()

				var overflows int64
				for _, sub := range subs {
					select {
					case sub.ch <- metric:
						sub.lastSendTime = now
					default:
						// Channel full, skip to avoid blocking
						sub.overflowCount++
						overflows++
					}
				}
				
				if overflows > 0 {
					ms.stats.mu.Lock()
					ms.stats.SubscriberOverflows += overflows
					ms.stats.mu.Unlock()
				}
			}

			// Update stats
			success := len(allMetrics) > 0
			containersProcessed := len(allMetrics)
			ms.stats.RecordCollection(success, containersProcessed, containersFailed)
			
			// Update cache stats
			ms.liveMetricsMutex.RLock()
			totalLiveMetrics := 0
			for _, m := range ms.liveMetrics {
				totalLiveMetrics += len(m)
			}
			ms.liveMetricsMutex.RUnlock()
			ms.statsMutex.RLock()
			previousStatsSize := len(ms.previousStats)
			ms.statsMutex.RUnlock()
			ms.stats.UpdateCacheStats(totalLiveMetrics, previousStatsSize)

		case <-ms.ctx.Done():
			return
		}
	}
}

// collectStatsParallel collects container stats in parallel using worker pool
// Returns metrics and count of failed containers
func (ms *MetricsStreamer) collectStatsParallel(locations []database.DeploymentLocation, shouldLog bool) ([]LiveMetric, int) {
	return ms.collectDeploymentStatsParallel(locations, shouldLog)
}

// collectDeploymentStatsParallel collects deployment container stats in parallel using worker pool
func (ms *MetricsStreamer) collectDeploymentStatsParallel(locations []database.DeploymentLocation, shouldLog bool) ([]LiveMetric, int) {
	_ = shouldLog // Unused but kept for API compatibility
	type statsJob struct {
		location database.DeploymentLocation
		index    int
	}

	type statsResult struct {
		metric LiveMetric
		err    error
		index  int
	}

	jobs := make(chan statsJob, len(locations))
	results := make(chan statsResult, len(locations))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < ms.config.MaxWorkers && i < len(locations); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if job.location.Status != "running" {
					results <- statsResult{err: nil, index: job.index} // Skip non-running
					continue
				}
				
				// Verify container actually exists before trying to get stats
				_, inspectErr := ms.serviceRegistry.DockerClient().ContainerInspect(context.Background(), job.location.ContainerID)
				if inspectErr != nil {
					// Container doesn't exist - update status in database and skip
					if strings.Contains(job.location.DeploymentID, "gs-") {
						// Game server - update GameServerLocation
						database.DB.Model(&database.GameServerLocation{}).
							Where("container_id = ?", job.location.ContainerID).
							Update("status", "stopped")
					} else {
						// Deployment - update DeploymentLocation
						database.DB.Model(&database.DeploymentLocation{}).
							Where("container_id = ?", job.location.ContainerID).
							Update("status", "stopped")
					}
					results <- statsResult{err: nil, index: job.index} // Skip non-existent container
					continue
				}

				// Use circuit breaker and retry for Docker API calls
				var currentStats *ContainerStats
				var err error
				
				err = ms.circuitBreaker.Call(func() error {
					stats, e := ms.getContainerStatsWithRetry(job.location.ContainerID)
					if e != nil {
						return e
					}
					currentStats = stats
					return nil
				})
				
				if err != nil || currentStats == nil {
					// Silently skip failed containers - they'll be retried next cycle
					results <- statsResult{err: err, index: job.index}
					continue
				}
				
				// Update circuit breaker stats
				ms.stats.UpdateCircuitBreakerState(ms.circuitBreaker.GetState(), int64(ms.circuitBreaker.GetFailureCount()))

				// Get previous stats for delta calculation
				ms.statsMutex.RLock()
				prevStats, hasPrev := ms.previousStats[job.location.ContainerID]
				ms.statsMutex.RUnlock()

				// Calculate deltas
				var networkRxDelta, networkTxDelta, diskReadDelta, diskWriteDelta int64
				if hasPrev {
					networkRxDelta = currentStats.NetworkRxBytes - prevStats.NetworkRxBytes
					networkTxDelta = currentStats.NetworkTxBytes - prevStats.NetworkTxBytes
					diskReadDelta = currentStats.DiskReadBytes - prevStats.DiskReadBytes
					diskWriteDelta = currentStats.DiskWriteBytes - prevStats.DiskWriteBytes

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
				}

				metric := LiveMetric{
					ResourceType:   "deployment",
					ResourceID:     job.location.DeploymentID,
					ContainerID:    job.location.ContainerID,
					NodeID:         job.location.NodeID,
					CPUUsage:       currentStats.CPUUsage,
					MemoryUsage:    currentStats.MemoryUsage,
					NetworkRxBytes: networkRxDelta,
					NetworkTxBytes: networkTxDelta,
					DiskReadBytes:  diskReadDelta,
					DiskWriteBytes: diskWriteDelta,
					RequestCount:   0, // Not tracked for deployments via Docker stats
					ErrorCount:     0, // Not tracked for deployments via Docker stats
					Timestamp:      time.Now(),
				}

				// Store stats for next delta calculation
				ms.statsMutex.Lock()
				ms.previousStats[job.location.ContainerID] = currentStats
				ms.statsMutex.Unlock()

				// Update deployment location
				_ = ms.serviceRegistry.UpdateDeploymentMetrics(job.location.ContainerID, currentStats.CPUUsage, currentStats.MemoryUsage)

				results <- statsResult{metric: metric, err: nil, index: job.index}
			}
		}()
	}

	// Send jobs
	for i, location := range locations {
		jobs <- statsJob{location: location, index: i}
	}
	close(jobs)

	// Wait for workers
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order
	resultMap := make(map[int]LiveMetric)
	var errorList []error

	for result := range results {
		if result.err != nil {
			errorList = append(errorList, result.err)
			continue
		}
		if result.metric.ResourceID != "" {
			resultMap[result.index] = result.metric
		}
	}

	// Convert to ordered slice
	metrics := make([]LiveMetric, 0, len(resultMap))
	for i := 0; i < len(locations); i++ {
		if metric, exists := resultMap[i]; exists {
			metrics = append(metrics, metric)
		}
	}

	if shouldLog && len(errorList) > 0 {
	}

	return metrics, len(errorList)
}

// collectGameServerStatsParallel collects game server container stats in parallel using worker pool
// This reuses the same Docker container stats collection logic as deployments
func (ms *MetricsStreamer) collectGameServerStatsParallel(locations []database.GameServerLocation, shouldLog bool) ([]LiveMetric, int) {
	_ = shouldLog // Unused but kept for API compatibility
	// Convert game server locations to deployment-like format to reuse collection logic
	deploymentLikeLocations := make([]database.DeploymentLocation, len(locations))
	for i, gsLoc := range locations {
		deploymentLikeLocations[i] = database.DeploymentLocation{
			ID:           gsLoc.ID,
			DeploymentID: gsLoc.GameServerID, // Reuse DeploymentID field temporarily
			NodeID:       gsLoc.NodeID,
			NodeHostname: gsLoc.NodeHostname,
			ContainerID:  gsLoc.ContainerID,
			Status:       gsLoc.Status,
			Port:         0,
			CreatedAt:    gsLoc.CreatedAt,
			UpdatedAt:    gsLoc.UpdatedAt,
		}
	}
	
	// Use deployment collection function - it's the same Docker container logic
	deploymentMetrics, failed := ms.collectDeploymentStatsParallel(deploymentLikeLocations, false)
	
	// Convert metrics back to game server format
	gameServerMetrics := make([]LiveMetric, 0, len(deploymentMetrics))
	for _, metric := range deploymentMetrics {
		// Change ResourceType from "deployment" to "gameserver"
		metric.ResourceType = "gameserver"
		// ResourceID is already correct (GameServerID was stored in DeploymentID field)
		
		// Update game server location in database
		_ = database.DB.Model(&database.GameServerLocation{}).
			Where("container_id = ?", metric.ContainerID).
			Updates(map[string]interface{}{
				"cpu_usage":    metric.CPUUsage,
				"memory_usage": metric.MemoryUsage,
				"updated_at":   time.Now(),
			})
		
		gameServerMetrics = append(gameServerMetrics, metric)
	}
	
	return gameServerMetrics, failed
}

// getContainerStatsWithRetry retrieves container stats with exponential backoff retry
func (ms *MetricsStreamer) getContainerStatsWithRetry(containerID string) (*ContainerStats, error) {
	var lastErr error
	backoff := ms.config.DockerAPIRetryInitialBackoff
	
	for attempt := 0; attempt < ms.config.DockerAPIRetryMaxAttempts; attempt++ {
		stats, err := ms.getContainerStats(containerID)
		if err == nil {
			return stats, nil
		}
		
		lastErr = err
		
		// Exponential backoff before retry
		if attempt < ms.config.DockerAPIRetryMaxAttempts-1 {
			if backoff > ms.config.DockerAPIRetryMaxBackoff {
				backoff = ms.config.DockerAPIRetryMaxBackoff
			}
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	
	return nil, fmt.Errorf("failed after %d attempts: %w", ms.config.DockerAPIRetryMaxAttempts, lastErr)
}

// getContainerStats retrieves stats from Docker
func (ms *MetricsStreamer) getContainerStats(containerID string) (*ContainerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ms.config.DockerAPITimeout)
	defer cancel()

	statsResp, err := ms.serviceRegistry.DockerClient().ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	// Decode stats JSON response
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

	return &ContainerStats{
		CPUUsage:       cpuUsage,
		MemoryUsage:    int64(statsJSON.MemoryStats.Usage),
		NetworkRxBytes: networkRx,
		NetworkTxBytes: networkTx,
		DiskReadBytes:  diskRead,
		DiskWriteBytes: diskWrite,
	}, nil
}

// storeMetricsBatch periodically saves aggregated metrics to database in batches
func (ms *MetricsStreamer) storeMetricsBatch() {
	ticker := time.NewTicker(ms.config.StorageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.liveMetricsMutex.RLock()
			resourceIDs := make([]string, 0, len(ms.liveMetrics))
			for resourceID := range ms.liveMetrics {
				resourceIDs = append(resourceIDs, resourceID)
			}
			ms.liveMetricsMutex.RUnlock()

			if len(resourceIDs) == 0 {
				continue
			}

			// Aggregate metrics per resource (deployment or game server)
			var deploymentMetricsToStore []database.DeploymentMetrics
			var gameServerMetricsToStore []database.GameServerMetrics

			for _, resourceID := range resourceIDs {
				ms.liveMetricsMutex.RLock()
				liveMetrics := ms.liveMetrics[resourceID]
				ms.liveMetricsMutex.RUnlock()

				if len(liveMetrics) == 0 {
					continue
				}

				// Determine resource type from first metric
				resourceType := liveMetrics[0].ResourceType

				// Group by container and aggregate
				containerMetrics := make(map[string][]LiveMetric)
				for _, m := range liveMetrics {
					containerMetrics[m.ContainerID] = append(containerMetrics[m.ContainerID], m)
				}

				// Create one aggregated metric per container for this interval
				for containerID, containerMetricsSlice := range containerMetrics {
					if len(containerMetricsSlice) == 0 {
						continue
					}

					// Aggregate: average CPU/memory, sum network/disk
					var sumCPU float64
					var sumMemory int64
					var sumNetworkRx int64
					var sumNetworkTx int64
					var sumDiskRead int64
					var sumDiskWrite int64

					for _, m := range containerMetricsSlice {
						sumCPU += m.CPUUsage
						sumMemory += m.MemoryUsage
						sumNetworkRx += m.NetworkRxBytes
						sumNetworkTx += m.NetworkTxBytes
						sumDiskRead += m.DiskReadBytes
						sumDiskWrite += m.DiskWriteBytes
					}

					avgCPU := sumCPU / float64(len(containerMetricsSlice))
					avgMemory := sumMemory / int64(len(containerMetricsSlice))

					// Use the latest timestamp
					latestTimestamp := containerMetricsSlice[len(containerMetricsSlice)-1].Timestamp

					if resourceType == "deployment" {
						// Aggregate request and error counts
						var sumRequestCount int64
						var sumErrorCount int64
						for _, m := range containerMetricsSlice {
							sumRequestCount += m.RequestCount
							sumErrorCount += m.ErrorCount
						}

						deploymentMetricsToStore = append(deploymentMetricsToStore, database.DeploymentMetrics{
							DeploymentID:   resourceID,
							ContainerID:    containerID,
							NodeID:         containerMetricsSlice[0].NodeID,
							CPUUsage:       avgCPU,
							MemoryUsage:    avgMemory,
							NetworkRxBytes: sumNetworkRx,
							NetworkTxBytes: sumNetworkTx,
							DiskReadBytes:  sumDiskRead,
							DiskWriteBytes: sumDiskWrite,
							RequestCount:   sumRequestCount,
							ErrorCount:     sumErrorCount,
							Timestamp:      latestTimestamp,
						})

						// Record Prometheus metrics for this deployment
						metrics.RecordDeploymentMetrics(
							resourceID,
							avgCPU,
							avgMemory,
							sumNetworkRx,
							sumNetworkTx,
							sumDiskRead,
							sumDiskWrite,
							sumRequestCount,
							sumErrorCount,
						)

						if len(deploymentMetricsToStore) >= ms.config.BatchSize {
							targetDB := database.MetricsDB
							if targetDB == nil {
								targetDB = database.DB
							}
							
							if err := targetDB.CreateInBatches(deploymentMetricsToStore, ms.config.BatchSize).Error; err != nil {
								log.Printf("[MetricsStreamer] Failed to store deployment metrics batch (%d metrics): %v", len(deploymentMetricsToStore), err)
								ms.stats.RecordStorage(false, len(deploymentMetricsToStore))
							} else {
								ms.stats.RecordStorage(true, len(deploymentMetricsToStore))
							}
							deploymentMetricsToStore = deploymentMetricsToStore[:0]
						}
					} else if resourceType == "gameserver" {
						gameServerMetricsToStore = append(gameServerMetricsToStore, database.GameServerMetrics{
							GameServerID:   resourceID,
							ContainerID:    containerID,
							NodeID:         containerMetricsSlice[0].NodeID,
							CPUUsage:       avgCPU,
							MemoryUsage:    avgMemory,
							NetworkRxBytes: sumNetworkRx,
							NetworkTxBytes: sumNetworkTx,
							DiskReadBytes:  sumDiskRead,
							DiskWriteBytes: sumDiskWrite,
							Timestamp:      latestTimestamp,
						})

						// Record Prometheus metrics for this game server
						metrics.RecordGameServerMetrics(
							resourceID,
							avgCPU,
							avgMemory,
							sumNetworkRx,
							sumNetworkTx,
							sumDiskRead,
							sumDiskWrite,
						)

						if len(gameServerMetricsToStore) >= ms.config.BatchSize {
							targetDB := database.MetricsDB
							if targetDB == nil {
								targetDB = database.DB
							}
							
							if err := targetDB.CreateInBatches(gameServerMetricsToStore, ms.config.BatchSize).Error; err != nil {
								log.Printf("[MetricsStreamer] Failed to store game server metrics batch (%d metrics): %v", len(gameServerMetricsToStore), err)
								ms.stats.RecordStorage(false, len(gameServerMetricsToStore))
							} else {
								ms.stats.RecordStorage(true, len(gameServerMetricsToStore))
							}
							gameServerMetricsToStore = gameServerMetricsToStore[:0]
						}
					}
				}
			}

			// Store remaining metrics
			targetDB := database.MetricsDB
			if targetDB == nil {
				targetDB = database.DB
			}

			if len(deploymentMetricsToStore) > 0 {
				// Record Prometheus metrics for remaining deployment metrics
				for _, depMetric := range deploymentMetricsToStore {
					metrics.RecordDeploymentMetrics(
						depMetric.DeploymentID,
						depMetric.CPUUsage,
						depMetric.MemoryUsage,
						depMetric.NetworkRxBytes,
						depMetric.NetworkTxBytes,
						depMetric.DiskReadBytes,
						depMetric.DiskWriteBytes,
						depMetric.RequestCount,
						depMetric.ErrorCount,
					)
				}

				if err := targetDB.CreateInBatches(deploymentMetricsToStore, ms.config.BatchSize).Error; err != nil {
					log.Printf("[MetricsStreamer] Failed to store final deployment metrics batch (%d metrics): %v", len(deploymentMetricsToStore), err)
					ms.stats.RecordStorage(false, len(deploymentMetricsToStore))
				} else {
					ms.stats.RecordStorage(true, len(deploymentMetricsToStore))
				}
			}

			if len(gameServerMetricsToStore) > 0 {
				// Record Prometheus metrics for remaining game server metrics
				for _, gsMetric := range gameServerMetricsToStore {
					metrics.RecordGameServerMetrics(
						gsMetric.GameServerID,
						gsMetric.CPUUsage,
						gsMetric.MemoryUsage,
						gsMetric.NetworkRxBytes,
						gsMetric.NetworkTxBytes,
						gsMetric.DiskReadBytes,
						gsMetric.DiskWriteBytes,
					)
				}

				if err := targetDB.CreateInBatches(gameServerMetricsToStore, ms.config.BatchSize).Error; err != nil {
					log.Printf("[MetricsStreamer] Failed to store final game server metrics batch (%d metrics): %v", len(gameServerMetricsToStore), err)
					ms.stats.RecordStorage(false, len(gameServerMetricsToStore))
				} else {
					ms.stats.RecordStorage(true, len(gameServerMetricsToStore))
				}
			}

		case <-ms.ctx.Done():
			return
		}
	}
}

// cleanupLiveMetrics removes old metrics from memory
func (ms *MetricsStreamer) cleanupLiveMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-ms.config.LiveRetention)

			ms.liveMetricsMutex.Lock()
			for deploymentID, metrics := range ms.liveMetrics {
				trimmed := metrics[:0]
				for _, m := range metrics {
					if m.Timestamp.After(cutoff) {
						trimmed = append(trimmed, m)
					}
				}
				ms.liveMetrics[deploymentID] = trimmed

				// Remove empty entries
				if len(trimmed) == 0 {
					delete(ms.liveMetrics, deploymentID)
				}
			}
			ms.liveMetricsMutex.Unlock()

		case <-ms.ctx.Done():
			return
		}
	}
}

// processRetries periodically attempts to retry failed database writes
func (ms *MetricsStreamer) processRetries() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			targetDB := database.MetricsDB
			if targetDB == nil {
				targetDB = database.DB
			}
			
			queueSize := ms.retryQueue.GetQueueSize()
			if queueSize > 0 {
				beforeSize := queueSize
				ms.retryQueue.ProcessRetries(targetDB)
				afterSize := ms.retryQueue.GetQueueSize()
				success := afterSize < beforeSize
				ms.retryQueue.ClearOldBatches()
				ms.stats.RecordRetry(int(queueSize), success)
			}

		case <-ms.ctx.Done():
			return
		}
	}
}

// cleanupStaleStats removes previousStats for containers that no longer exist
func (ms *MetricsStreamer) cleanupStaleStats() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get list of currently running containers
			nodeID := ms.serviceRegistry.NodeID()
			locations, err := ms.serviceRegistry.GetNodeDeployments(nodeID)
			if err != nil {
				continue
			}

			activeContainerIDs := make(map[string]bool)
			for _, loc := range locations {
				if loc.Status == "running" {
					activeContainerIDs[loc.ContainerID] = true
				}
			}

			// Clean up stats for containers that no longer exist
			ms.statsMutex.Lock()
			var removedCount int
			for containerID := range ms.previousStats {
				if !activeContainerIDs[containerID] {
					delete(ms.previousStats, containerID)
					removedCount++
				}
			}
			
			// Enforce max size limit
			if len(ms.previousStats) > ms.config.MaxPreviousStats {
				// Remove oldest entries (simple approach: remove randomly until under limit)
				// In a production system, you might want to track last used time
				for len(ms.previousStats) > ms.config.MaxPreviousStats {
					for containerID := range ms.previousStats {
						delete(ms.previousStats, containerID)
						removedCount++
						break // Remove one at a time
					}
				}
			}
			ms.statsMutex.Unlock()

			if removedCount > 0 {
				log.Printf("[MetricsStreamer] Cleaned up %d stale previousStats entries", removedCount)
			}

		case <-ms.ctx.Done():
			return
		}
	}
}


// healthCheck monitors the health of metrics collection
func (ms *MetricsStreamer) healthCheck() {
	ticker := time.NewTicker(ms.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := ms.stats.GetSnapshot()
			
			// Check if collection is lagging (no collections in last 2 intervals)
			collectionLagging := false
			if !stats.LastCollectionTime.IsZero() {
				elapsed := time.Since(stats.LastCollectionTime)
				if elapsed > ms.config.CollectionInterval*2 {
					collectionLagging = true
				}
			}
			
			// Check consecutive failures
			unhealthy := stats.ConsecutiveFailures >= int64(ms.config.HealthCheckFailureThreshold) || collectionLagging
			
			// Update health status
			ms.stats.UpdateHealth(!unhealthy, stats.ConsecutiveFailures)
			
			// Graceful degradation: slow down collection if unhealthy
			ms.degradationMutex.Lock()
			if unhealthy {
				// Reduce collection rate by 50%
				if ms.collectionRateMultiplier > 0.5 {
					ms.collectionRateMultiplier = 0.5
					log.Printf("[MetricsStreamer] Health check failed: entering graceful degradation mode")
				}
			} else {
				// Gradually return to normal speed
				if ms.collectionRateMultiplier < 1.0 {
					ms.collectionRateMultiplier = math.Min(1.0, ms.collectionRateMultiplier+0.1)
					if ms.collectionRateMultiplier >= 1.0 {
						log.Printf("[MetricsStreamer] Health check passed: returning to normal speed")
					}
				}
			}
			ms.degradationMutex.Unlock()
			
			// Log health status
			if unhealthy {
				log.Printf("[MetricsStreamer] Health check: UNHEALTHY (failures: %d, lagging: %v)", 
					stats.ConsecutiveFailures, collectionLagging)
			}

		case <-ms.ctx.Done():
			return
		}
	}
}

// monitorBackpressure detects and handles slow/dead subscriber channels
func (ms *MetricsStreamer) monitorBackpressure() {
	ticker := time.NewTicker(ms.config.SubscriberCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var slowSubs []*subscriberChannel
			var deadSubs []*subscriberChannel
			
			ms.subscribersMutex.Lock()
			for deploymentID, subs := range ms.subscribers {
				var validSubs []*subscriberChannel
				for _, sub := range subs {
					// Check if channel is slow (last send was too long ago)
					if !sub.lastSendTime.IsZero() {
						elapsed := now.Sub(sub.lastSendTime)
						if elapsed > ms.config.SubscriberSlowThreshold {
							// Check if channel is full (likely dead)
							select {
							case <-sub.ch:
								// Channel is readable, not dead
								if elapsed > ms.config.SubscriberSlowThreshold*2 {
									slowSubs = append(slowSubs, sub)
								}
								validSubs = append(validSubs, sub)
							default:
								// Channel is full, likely dead subscriber
								if elapsed > ms.config.SubscriberSlowThreshold*3 {
									deadSubs = append(deadSubs, sub)
									continue // Don't add to validSubs
								}
								validSubs = append(validSubs, sub)
							}
						} else {
							validSubs = append(validSubs, sub)
						}
					} else {
						validSubs = append(validSubs, sub)
					}
				}
				
				if len(validSubs) == 0 {
					delete(ms.subscribers, deploymentID)
				} else {
					ms.subscribers[deploymentID] = validSubs
				}
			}
			ms.subscribersMutex.Unlock()
			
			// Clean up dead subscribers
			for _, sub := range deadSubs {
				close(sub.ch)
			}
			
			// Update stats
			totalSubs := 0
			slowCount := len(slowSubs)
			for _, s := range ms.subscribers {
				totalSubs += len(s)
			}
			ms.stats.UpdateSubscriberStats(totalSubs, slowCount, 0)
			
			if len(deadSubs) > 0 {
				log.Printf("[MetricsStreamer] Cleaned up %d dead subscriber channels", len(deadSubs))
			}

		case <-ms.ctx.Done():
			return
		}
	}
}

// GetStats returns observability stats snapshot
func (ms *MetricsStreamer) GetStats() MetricsStats {
	return ms.stats.GetSnapshot()
}

// GetHealth returns the current health status
func (ms *MetricsStreamer) GetHealth() (bool, int64) {
	stats := ms.stats.GetSnapshot()
	return stats.IsHealthy, stats.ConsecutiveFailures
}
