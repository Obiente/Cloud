package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"api/internal/database"
	"api/internal/registry"
)

// LiveMetric represents a live metric in memory
type LiveMetric struct {
	DeploymentID   string
	ContainerID    string
	NodeID         string
	CPUUsage       float64
	MemoryUsage    int64
	NetworkRxBytes int64
	NetworkTxBytes int64
	DiskReadBytes  int64
	DiskWriteBytes int64
	RequestCount   int64
	ErrorCount     int64
	Timestamp      time.Time
}

// MetricsStreamer handles live metrics streaming and periodic storage
type MetricsStreamer struct {
	serviceRegistry *registry.ServiceRegistry
	previousStats   map[string]*containerStats
	statsMutex      sync.RWMutex

	// Live metrics cache: deploymentID -> []LiveMetric (last N minutes)
	liveMetrics      map[string][]LiveMetric
	liveMetricsMutex sync.RWMutex

	// Subscribers: deploymentID -> []chan LiveMetric
	subscribers      map[string][]chan LiveMetric
	subscribersMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	// Configuration
	collectionInterval time.Duration // How often to collect (default 5s)
	storageInterval    time.Duration // How often to save to DB (default 60s)
	liveRetention      time.Duration // How long to keep live metrics (default 5min)
	maxWorkers         int           // Max parallel workers for stats collection
	batchSize          int           // Batch size for DB writes
}

// NewMetricsStreamer creates a new metrics streamer
func NewMetricsStreamer(serviceRegistry *registry.ServiceRegistry) *MetricsStreamer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsStreamer{
		serviceRegistry:    serviceRegistry,
		previousStats:      make(map[string]*containerStats),
		liveMetrics:        make(map[string][]LiveMetric),
		subscribers:        make(map[string][]chan LiveMetric),
		ctx:                ctx,
		cancel:             cancel,
		collectionInterval: 5 * time.Second,
		storageInterval:    60 * time.Second, // Store every minute instead of every 5s
		liveRetention:      5 * time.Minute,
		maxWorkers:         50,  // Parallel workers for stats collection
		batchSize:          100, // Batch DB writes
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
}

// Stop stops the metrics streamer
func (ms *MetricsStreamer) Stop() {
	ms.cancel()

	// Close all subscriber channels
	ms.subscribersMutex.Lock()
	for _, chans := range ms.subscribers {
		for _, ch := range chans {
			close(ch)
		}
	}
	ms.subscribers = make(map[string][]chan LiveMetric)
	ms.subscribersMutex.Unlock()
}

// Subscribe adds a subscriber for a deployment's metrics
func (ms *MetricsStreamer) Subscribe(deploymentID string) <-chan LiveMetric {
	ch := make(chan LiveMetric, 10) // Buffered to avoid blocking

	ms.subscribersMutex.Lock()
	ms.subscribers[deploymentID] = append(ms.subscribers[deploymentID], ch)
	ms.subscribersMutex.Unlock()

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

	for i, subCh := range chans {
		if subCh == ch {
			// Remove from slice
			ms.subscribers[deploymentID] = append(chans[:i], chans[i+1:]...)
			close(subCh)
			return
		}
	}
}

// GetLatestMetrics returns the latest metrics for a deployment
func (ms *MetricsStreamer) GetLatestMetrics(deploymentID string) []LiveMetric {
	ms.liveMetricsMutex.RLock()
	defer ms.liveMetricsMutex.RUnlock()

	metrics, exists := ms.liveMetrics[deploymentID]
	if !exists {
		return []LiveMetric{}
	}

	return metrics
}

// collectLiveMetrics collects metrics in parallel and streams to subscribers
func (ms *MetricsStreamer) collectLiveMetrics() {
	ticker := time.NewTicker(ms.collectionInterval)
	defer ticker.Stop()

	lastLogTime := time.Now()

	for {
		select {
		case <-ticker.C:
			shouldLog := time.Since(lastLogTime) >= 30*time.Second
			if shouldLog {
				// log.Printf("[MetricsStreamer] Collecting live metrics...")
				lastLogTime = time.Now()
			}

			nodeID := ms.serviceRegistry.NodeID()
			locations, err := ms.serviceRegistry.GetNodeDeployments(nodeID)
			if err != nil {
				if shouldLog {
					// log.Printf("[MetricsStreamer] Failed to get node deployments: %v", err)
				}
				continue
			}

			if len(locations) == 0 {
				continue
			}

			// Collect stats in parallel using worker pool
			metrics := ms.collectStatsParallel(locations, shouldLog)

			// Store in live cache and stream to subscribers
			now := time.Now()
			for _, metric := range metrics {
				// Add to live cache
				ms.liveMetricsMutex.Lock()
				if ms.liveMetrics[metric.DeploymentID] == nil {
					ms.liveMetrics[metric.DeploymentID] = make([]LiveMetric, 0)
				}
				ms.liveMetrics[metric.DeploymentID] = append(ms.liveMetrics[metric.DeploymentID], metric)

				// Trim old metrics (keep only last N minutes)
				retentionCutoff := now.Add(-ms.liveRetention)
				trimmed := ms.liveMetrics[metric.DeploymentID][:0]
				for _, m := range ms.liveMetrics[metric.DeploymentID] {
					if m.Timestamp.After(retentionCutoff) {
						trimmed = append(trimmed, m)
					}
				}
				ms.liveMetrics[metric.DeploymentID] = trimmed
				ms.liveMetricsMutex.Unlock()

				// Stream to subscribers (non-blocking)
				ms.subscribersMutex.RLock()
				subs := ms.subscribers[metric.DeploymentID]
				ms.subscribersMutex.RUnlock()

				for _, ch := range subs {
					select {
					case ch <- metric:
					default:
						// Channel full, skip to avoid blocking
					}
				}
			}

			if shouldLog && len(metrics) > 0 {
				// log.Printf("[MetricsStreamer] Collected %d live metrics", len(metrics))
			}

		case <-ms.ctx.Done():
			return
		}
	}
}

// collectStatsParallel collects container stats in parallel using worker pool
func (ms *MetricsStreamer) collectStatsParallel(locations []database.DeploymentLocation, shouldLog bool) []LiveMetric {
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
	for i := 0; i < ms.maxWorkers && i < len(locations); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if job.location.Status != "running" {
					results <- statsResult{err: nil, index: job.index} // Skip non-running
					continue
				}

				currentStats, err := ms.getContainerStats(job.location.ContainerID)
				if err != nil {
					results <- statsResult{err: err, index: job.index}
					continue
				}

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
					DeploymentID:   job.location.DeploymentID,
					ContainerID:    job.location.ContainerID,
					NodeID:         job.location.NodeID,
					CPUUsage:       currentStats.CPUUsage,
					MemoryUsage:    currentStats.MemoryUsage,
					NetworkRxBytes: networkRxDelta,
					NetworkTxBytes: networkTxDelta,
					DiskReadBytes:  diskReadDelta,
					DiskWriteBytes: diskWriteDelta,
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
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}
		if result.metric.DeploymentID != "" {
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

	if shouldLog && len(errors) > 0 {
		// log.Printf("[MetricsStreamer] %d errors during stats collection", len(errors))
	}

	return metrics
}

// getContainerStats retrieves stats from Docker
func (ms *MetricsStreamer) getContainerStats(containerID string) (*containerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	return &containerStats{
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
	ticker := time.NewTicker(ms.storageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.liveMetricsMutex.RLock()
			deploymentIDs := make([]string, 0, len(ms.liveMetrics))
			for depID := range ms.liveMetrics {
				deploymentIDs = append(deploymentIDs, depID)
			}
			ms.liveMetricsMutex.RUnlock()

			if len(deploymentIDs) == 0 {
				continue
			}

			// Aggregate metrics per deployment (average over the storage interval)
			var metricsToStore []database.DeploymentMetrics

			for _, deploymentID := range deploymentIDs {
				ms.liveMetricsMutex.RLock()
				liveMetrics := ms.liveMetrics[deploymentID]
				ms.liveMetricsMutex.RUnlock()

				if len(liveMetrics) == 0 {
					continue
				}

				// Group by container and aggregate
				containerMetrics := make(map[string][]LiveMetric)
				for _, m := range liveMetrics {
					containerMetrics[m.ContainerID] = append(containerMetrics[m.ContainerID], m)
				}

				// Create one aggregated metric per container for this interval
				for containerID, metrics := range containerMetrics {
					if len(metrics) == 0 {
						continue
					}

					// Aggregate: average CPU/memory, sum network/disk
					var sumCPU float64
					var sumMemory int64
					var sumNetworkRx int64
					var sumNetworkTx int64
					var sumDiskRead int64
					var sumDiskWrite int64

					for _, m := range metrics {
						sumCPU += m.CPUUsage
						sumMemory += m.MemoryUsage
						sumNetworkRx += m.NetworkRxBytes
						sumNetworkTx += m.NetworkTxBytes
						sumDiskRead += m.DiskReadBytes
						sumDiskWrite += m.DiskWriteBytes
					}

					avgCPU := sumCPU / float64(len(metrics))
					avgMemory := sumMemory / int64(len(metrics))

					// Use the latest timestamp
					latestTimestamp := metrics[len(metrics)-1].Timestamp

					metricsToStore = append(metricsToStore, database.DeploymentMetrics{
						DeploymentID:   deploymentID,
						ContainerID:    containerID,
						NodeID:         metrics[0].NodeID,
						CPUUsage:       avgCPU,
						MemoryUsage:    avgMemory,
						NetworkRxBytes: sumNetworkRx,
						NetworkTxBytes: sumNetworkTx,
						DiskReadBytes:  sumDiskRead,
						DiskWriteBytes: sumDiskWrite,
						Timestamp:      latestTimestamp,
					})

					if len(metricsToStore) >= ms.batchSize {
						// Batch insert
						if err := database.DB.CreateInBatches(metricsToStore, ms.batchSize).Error; err != nil {
							// log.Printf("[MetricsStreamer] Failed to store metrics batch: %v", err)
						}
						metricsToStore = metricsToStore[:0]
					}
				}
			}

			// Store remaining metrics
			if len(metricsToStore) > 0 {
				if err := database.DB.CreateInBatches(metricsToStore, ms.batchSize).Error; err != nil {
					// log.Printf("[MetricsStreamer] Failed to store final metrics batch: %v", err)
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
			cutoff := time.Now().Add(-ms.liveRetention)

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
