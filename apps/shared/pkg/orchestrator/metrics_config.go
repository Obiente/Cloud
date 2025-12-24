package orchestrator

import (
	"os"
	"strconv"
	"time"
)

// MetricsConfig holds all configurable metrics settings
type MetricsConfig struct {
	CollectionInterval          time.Duration
	StorageInterval             time.Duration
	LiveRetention               time.Duration
	MaxWorkers                  int
	BatchSize                   int
	MaxLiveMetricsPerDeployment int
	MaxPreviousStats            int

	// Retry queue config
	RetryMaxRetries   int
	RetryInterval     time.Duration
	RetryMaxQueueSize int

	// Docker API config
	DockerAPITimeout             time.Duration
	DockerAPIRetryMaxAttempts    int
	DockerAPIRetryInitialBackoff time.Duration
	DockerAPIRetryMaxBackoff     time.Duration

	// Circuit breaker config
	CircuitBreakerFailureThreshold int
	CircuitBreakerCooldownPeriod   time.Duration
	CircuitBreakerHalfOpenMaxCalls int

	// Health check config
	HealthCheckInterval         time.Duration
	HealthCheckFailureThreshold int

	// Backpressure config
	SubscriberChannelBufferSize int
	SubscriberSlowThreshold     time.Duration
	SubscriberCleanupInterval   time.Duration
	// Minimum system delta used when computing CPU percentage (nanoseconds)
	MinSystemDelta              time.Duration
}

// LoadMetricsConfig loads metrics configuration from environment variables
func LoadMetricsConfig() *MetricsConfig {
	cfg := &MetricsConfig{
		// Default values
		CollectionInterval:             5 * time.Second,
		StorageInterval:                60 * time.Second,
		LiveRetention:                  5 * time.Minute,
		MaxWorkers:                     50,
		BatchSize:                      100,
		MaxLiveMetricsPerDeployment:    1000,
		MaxPreviousStats:               10000,
		RetryMaxRetries:                5,
		RetryInterval:                  1 * time.Minute,
		RetryMaxQueueSize:              10000,
		DockerAPITimeout:               10 * time.Second,
		DockerAPIRetryMaxAttempts:      3,
		DockerAPIRetryInitialBackoff:   1 * time.Second,
		DockerAPIRetryMaxBackoff:       30 * time.Second,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerCooldownPeriod:   1 * time.Minute,
		CircuitBreakerHalfOpenMaxCalls: 3,
		HealthCheckInterval:            30 * time.Second,
		HealthCheckFailureThreshold:    3,
		SubscriberChannelBufferSize:    100,
		SubscriberSlowThreshold:        5 * time.Second,
		SubscriberCleanupInterval:      1 * time.Minute,
		MinSystemDelta:                  1 * time.Millisecond,
	}

	// Parse environment variables with fallback to defaults
	if val := os.Getenv("METRICS_COLLECTION_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.CollectionInterval = d
		}
	}

	if val := os.Getenv("METRICS_STORAGE_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.StorageInterval = d
		}
	}

	if val := os.Getenv("METRICS_LIVE_RETENTION"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.LiveRetention = d
		}
	}

	if val := os.Getenv("METRICS_MAX_WORKERS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.MaxWorkers = i
		}
	}

	if val := os.Getenv("METRICS_BATCH_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.BatchSize = i
		}
	}

	if val := os.Getenv("METRICS_MAX_LIVE_PER_DEPLOYMENT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.MaxLiveMetricsPerDeployment = i
		}
	}

	if val := os.Getenv("METRICS_MAX_PREVIOUS_STATS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.MaxPreviousStats = i
		}
	}

	if val := os.Getenv("METRICS_RETRY_MAX_RETRIES"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.RetryMaxRetries = i
		}
	}

	if val := os.Getenv("METRICS_RETRY_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.RetryInterval = d
		}
	}

	if val := os.Getenv("METRICS_RETRY_MAX_QUEUE_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.RetryMaxQueueSize = i
		}
	}

	if val := os.Getenv("METRICS_DOCKER_API_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.DockerAPITimeout = d
		}
	}

	if val := os.Getenv("METRICS_DOCKER_API_RETRY_MAX"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.DockerAPIRetryMaxAttempts = i
		}
	}

	if val := os.Getenv("METRICS_DOCKER_API_RETRY_BACKOFF_INITIAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.DockerAPIRetryInitialBackoff = d
		}
	}

	if val := os.Getenv("METRICS_DOCKER_API_RETRY_BACKOFF_MAX"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.DockerAPIRetryMaxBackoff = d
		}
	}

	if val := os.Getenv("METRICS_CIRCUIT_BREAKER_FAILURE_THRESHOLD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.CircuitBreakerFailureThreshold = i
		}
	}

	if val := os.Getenv("METRICS_CIRCUIT_BREAKER_COOLDOWN"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.CircuitBreakerCooldownPeriod = d
		}
	}

	if val := os.Getenv("METRICS_CIRCUIT_BREAKER_HALFOPEN_MAX"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.CircuitBreakerHalfOpenMaxCalls = i
		}
	}

	if val := os.Getenv("METRICS_HEALTH_CHECK_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.HealthCheckInterval = d
		}
	}

	if val := os.Getenv("METRICS_HEALTH_CHECK_FAILURE_THRESHOLD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.HealthCheckFailureThreshold = i
		}
	}

	if val := os.Getenv("METRICS_SUBSCRIBER_BUFFER_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			cfg.SubscriberChannelBufferSize = i
		}
	}

	if val := os.Getenv("METRICS_SUBSCRIBER_SLOW_THRESHOLD"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.SubscriberSlowThreshold = d
		}
	}

	if val := os.Getenv("METRICS_SUBSCRIBER_CLEANUP_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.SubscriberCleanupInterval = d
		}
	}

	if val := os.Getenv("METRICS_MIN_SYSTEM_DELTA"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.MinSystemDelta = d
		}
	}

	return cfg
}
