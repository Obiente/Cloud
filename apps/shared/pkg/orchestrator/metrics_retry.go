package orchestrator

import (
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	"gorm.io/gorm"
)

// FailedMetricBatch represents a batch of metrics that failed to write to the database
type FailedMetricBatch struct {
	Metrics    []database.DeploymentMetrics `json:"metrics"`
	FailedAt   time.Time                    `json:"failed_at"`
	RetryCount int                          `json:"retry_count"`
	Error      string                       `json:"error"`
}

// MetricsRetryQueue handles retry logic for failed database writes
type MetricsRetryQueue struct {
	failedBatches []FailedMetricBatch
	mutex         sync.RWMutex
	maxRetries    int
	retryInterval time.Duration
	maxQueueSize  int
}

// NewMetricsRetryQueue creates a new retry queue with default config
func NewMetricsRetryQueue() *MetricsRetryQueue {
	return &MetricsRetryQueue{
		failedBatches: make([]FailedMetricBatch, 0),
		maxRetries:    5,
		retryInterval: 1 * time.Minute,
		maxQueueSize:  10000, // Limit queue size to prevent memory issues
	}
}

// NewMetricsRetryQueueWithConfig creates a new retry queue from config
func NewMetricsRetryQueueWithConfig(config *MetricsConfig) *MetricsRetryQueue {
	return &MetricsRetryQueue{
		failedBatches: make([]FailedMetricBatch, 0),
		maxRetries:    config.RetryMaxRetries,
		retryInterval: config.RetryInterval,
		maxQueueSize:  config.RetryMaxQueueSize,
	}
}

// AddFailedBatch adds a failed batch to the retry queue
func (mq *MetricsRetryQueue) AddFailedBatch(metrics []database.DeploymentMetrics, err error) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	// Prevent queue from growing too large
	if len(mq.failedBatches) >= mq.maxQueueSize {
		// Remove oldest batch if queue is full
		if len(mq.failedBatches) > 0 {
			mq.failedBatches = mq.failedBatches[1:]
		}
	}

	batch := FailedMetricBatch{
		Metrics:    metrics,
		FailedAt:   time.Now(),
		RetryCount: 0,
		Error:      err.Error(),
	}

	mq.failedBatches = append(mq.failedBatches, batch)
}

// ProcessRetries attempts to retry failed batches
func (mq *MetricsRetryQueue) ProcessRetries(db *gorm.DB) {
	mq.mutex.Lock()
	batches := make([]FailedMetricBatch, len(mq.failedBatches))
	copy(batches, mq.failedBatches)
	mq.mutex.Unlock()

	var remainingBatches []FailedMetricBatch
	var successfulBatches []FailedMetricBatch

	for _, batch := range batches {
		if batch.RetryCount >= mq.maxRetries {
			// Max retries exceeded, give up
			continue
		}

		// Check if enough time has passed since last failure
		if time.Since(batch.FailedAt) < mq.retryInterval {
			remainingBatches = append(remainingBatches, batch)
			continue
		}

		// Attempt to write the batch
		batch.RetryCount++
		batch.FailedAt = time.Now()

		// Require MetricsDB (TimescaleDB) - do not fallback to main DB
		if database.MetricsDB == nil {
			batch.Error = "metrics database (TimescaleDB) not initialized"
			remainingBatches = append(remainingBatches, batch)
			continue
		}
		targetDB := database.MetricsDB

		if err := targetDB.CreateInBatches(batch.Metrics, len(batch.Metrics)).Error; err != nil {
			batch.Error = err.Error()
			remainingBatches = append(remainingBatches, batch)
		} else {
			successfulBatches = append(successfulBatches, batch)
		}
	}

	// Update queue with remaining batches
	mq.mutex.Lock()
	mq.failedBatches = remainingBatches
	mq.mutex.Unlock()

	// Log success if any batches were retried successfully
	if len(successfulBatches) > 0 {
		// log.Printf("[MetricsRetryQueue] Successfully retried %d failed batches", len(successfulBatches))
	}
}

// GetQueueSize returns the current queue size
func (mq *MetricsRetryQueue) GetQueueSize() int {
	mq.mutex.RLock()
	defer mq.mutex.RUnlock()
	return len(mq.failedBatches)
}

// ClearOldBatches removes batches that have exceeded max retries
func (mq *MetricsRetryQueue) ClearOldBatches() {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	var validBatches []FailedMetricBatch
	for _, batch := range mq.failedBatches {
		if batch.RetryCount < mq.maxRetries {
			validBatches = append(validBatches, batch)
		}
	}

	mq.failedBatches = validBatches
}
