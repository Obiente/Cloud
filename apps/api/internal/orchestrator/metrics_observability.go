package orchestrator

import (
	"sync"
	"time"
)

// MetricsStats tracks observability metrics for the metrics streamer
type MetricsStats struct {
	// Collection stats
	CollectionCount      int64
	CollectionErrors     int64
	CollectionsPerSecond float64
	LastCollectionTime   time.Time

	// Container stats
	ContainersProcessed          int64
	ContainersFailed             int64
	ContainersProcessedPerSecond float64

	// Storage stats
	StorageBatchesWritten int64
	StorageBatchesFailed  int64
	StorageMetricsWritten int64
	StorageMetricsFailed  int64
	LastStorageTime       time.Time

	// Retry queue stats
	RetryQueueSize        int64
	RetryBatchesProcessed int64
	RetryBatchesSuccess   int64
	RetryBatchesFailed    int64

	// Subscriber stats
	ActiveSubscribers   int64
	SlowSubscribers     int64
	SubscriberOverflows int64

	// Cache stats
	LiveMetricsCacheSize   int64
	PreviousStatsCacheSize int64

	// Circuit breaker stats
	CircuitBreakerState    int // 0=closed, 1=open, 2=half-open
	CircuitBreakerFailures int64

	// Health
	IsHealthy           bool
	ConsecutiveFailures int64
	LastHealthCheckTime time.Time

	mu sync.RWMutex
}

// NewMetricsStats creates a new metrics stats tracker
func NewMetricsStats() *MetricsStats {
	return &MetricsStats{
		IsHealthy: true,
	}
}

// RecordCollection records a collection cycle
func (ms *MetricsStats) RecordCollection(success bool, containersProcessed, containersFailed int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.CollectionCount++
	ms.LastCollectionTime = time.Now()
	if !success {
		ms.CollectionErrors++
		ms.ConsecutiveFailures++
	} else {
		ms.ConsecutiveFailures = 0
	}

	ms.ContainersProcessed += int64(containersProcessed)
	ms.ContainersFailed += int64(containersFailed)

	// Calculate rate (simple moving average)
	ms.updateRates()
}

// RecordStorage records a storage operation
func (ms *MetricsStats) RecordStorage(success bool, metricsCount int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.StorageBatchesWritten++
	ms.LastStorageTime = time.Now()
	if success {
		ms.StorageMetricsWritten += int64(metricsCount)
	} else {
		ms.StorageBatchesFailed++
		ms.StorageMetricsFailed += int64(metricsCount)
	}
}

// RecordRetry records a retry operation
func (ms *MetricsStats) RecordRetry(queueSize int, success bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.RetryQueueSize = int64(queueSize)
	ms.RetryBatchesProcessed++
	if success {
		ms.RetryBatchesSuccess++
	} else {
		ms.RetryBatchesFailed++
	}
}

// UpdateSubscriberStats updates subscriber statistics
func (ms *MetricsStats) UpdateSubscriberStats(active, slow int, overflows int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.ActiveSubscribers = int64(active)
	ms.SlowSubscribers = int64(slow)
	ms.SubscriberOverflows = overflows
}

// UpdateCacheStats updates cache size statistics
func (ms *MetricsStats) UpdateCacheStats(liveMetricsSize, previousStatsSize int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.LiveMetricsCacheSize = int64(liveMetricsSize)
	ms.PreviousStatsCacheSize = int64(previousStatsSize)
}

// UpdateCircuitBreakerState updates circuit breaker state
func (ms *MetricsStats) UpdateCircuitBreakerState(state CircuitState, failures int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.CircuitBreakerState = int(state)
	ms.CircuitBreakerFailures = failures
}

// UpdateHealth updates health status
func (ms *MetricsStats) UpdateHealth(healthy bool, consecutiveFailures int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.IsHealthy = healthy
	ms.ConsecutiveFailures = consecutiveFailures
	ms.LastHealthCheckTime = time.Now()
}

// GetSnapshot returns a snapshot of all stats
func (ms *MetricsStats) GetSnapshot() MetricsStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return MetricsStats{
		CollectionCount:              ms.CollectionCount,
		CollectionErrors:             ms.CollectionErrors,
		CollectionsPerSecond:         ms.CollectionsPerSecond,
		LastCollectionTime:           ms.LastCollectionTime,
		ContainersProcessed:          ms.ContainersProcessed,
		ContainersFailed:             ms.ContainersFailed,
		ContainersProcessedPerSecond: ms.ContainersProcessedPerSecond,
		StorageBatchesWritten:        ms.StorageBatchesWritten,
		StorageBatchesFailed:         ms.StorageBatchesFailed,
		StorageMetricsWritten:        ms.StorageMetricsWritten,
		StorageMetricsFailed:         ms.StorageMetricsFailed,
		LastStorageTime:              ms.LastStorageTime,
		RetryQueueSize:               ms.RetryQueueSize,
		RetryBatchesProcessed:        ms.RetryBatchesProcessed,
		RetryBatchesSuccess:          ms.RetryBatchesSuccess,
		RetryBatchesFailed:           ms.RetryBatchesFailed,
		ActiveSubscribers:            ms.ActiveSubscribers,
		SlowSubscribers:              ms.SlowSubscribers,
		SubscriberOverflows:          ms.SubscriberOverflows,
		LiveMetricsCacheSize:         ms.LiveMetricsCacheSize,
		PreviousStatsCacheSize:       ms.PreviousStatsCacheSize,
		CircuitBreakerState:          ms.CircuitBreakerState,
		CircuitBreakerFailures:       ms.CircuitBreakerFailures,
		IsHealthy:                    ms.IsHealthy,
		ConsecutiveFailures:          ms.ConsecutiveFailures,
		LastHealthCheckTime:          ms.LastHealthCheckTime,
	}
}

// updateRates calculates rates (simplified - in production, use proper time windows)
func (ms *MetricsStats) updateRates() {
	// Simple calculation: use last 10 seconds if available
	// In production, use a proper sliding window
	now := time.Now()
	if !ms.LastCollectionTime.IsZero() {
		elapsed := now.Sub(ms.LastCollectionTime).Seconds()
		if elapsed > 0 && elapsed < 10 {
			ms.CollectionsPerSecond = 1.0 / elapsed
		}
	}
}
