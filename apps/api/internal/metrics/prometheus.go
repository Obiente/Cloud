package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_api_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "obiente_api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"method", "endpoint"},
	)

	httpRequestInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "obiente_api_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// Metrics streamer metrics
	metricsCollectionCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_collection_count_total",
			Help: "Total number of metrics collection cycles",
		},
	)

	metricsCollectionErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_collection_errors_total",
			Help: "Total number of metrics collection errors",
		},
	)

	metricsContainersProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_containers_processed_total",
			Help: "Total number of containers processed",
		},
	)

	metricsContainersFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_containers_failed_total",
			Help: "Total number of containers that failed to process",
		},
	)

	metricsStorageBatchesWritten = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_storage_batches_written_total",
			Help: "Total number of storage batches written",
		},
	)

	metricsStorageBatchesFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "obiente_api_metrics_storage_batches_failed_total",
			Help: "Total number of storage batches that failed",
		},
	)

	metricsRetryQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "obiente_api_metrics_retry_queue_size",
			Help: "Current size of the metrics retry queue",
		},
	)

	metricsActiveSubscribers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "obiente_api_metrics_active_subscribers",
			Help: "Current number of active metrics subscribers",
		},
	)

	metricsCircuitBreakerState = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "obiente_api_metrics_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
	)

	metricsHealthy = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "obiente_api_metrics_healthy",
			Help: "Whether metrics collection is healthy (1=healthy, 0=unhealthy)",
		},
	)
)

// HTTPMetricsMiddleware wraps an HTTP handler to record Prometheus metrics
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		httpRequestInFlight.Inc()
		defer httpRequestInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := &statusCodeWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		method := r.Method
		endpoint := sanitizeEndpoint(r.URL.Path)
		statusCode := statusCodeToString(wrapped.statusCode)

		// Record metrics
		httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	})
}

// statusCodeWriter wraps http.ResponseWriter to capture status code
// It also implements http.Flusher if the underlying ResponseWriter does
type statusCodeWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusCodeWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCodeWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher if the underlying ResponseWriter does
func (w *statusCodeWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// sanitizeEndpoint removes IDs and sensitive data from endpoint paths
func sanitizeEndpoint(path string) string {
	// Remove common ID patterns
	// This is a simple implementation - can be enhanced
	if path == "/" {
		return "/"
	}
	// For now, return the path as-is
	// In production, you might want to normalize paths like:
	// /deployments/abc123 -> /deployments/{id}
	return path
}

// statusCodeToString converts status code to string for labels
func statusCodeToString(code int) string {
	return fmt.Sprintf("%d", code)
}

// UpdateMetricsStreamerStats updates Prometheus metrics from metrics streamer stats
func UpdateMetricsStreamerStats(stats interface{}) {
	// This function will be called from the metrics streamer
	// For now, it's a placeholder that accepts the stats interface
	// The actual implementation will extract values from the stats struct
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// UpdateMetricsFromStats updates Prometheus metrics from MetricsStats
func UpdateMetricsFromStats(
	collectionCount int64,
	collectionErrors int64,
	containersProcessed int64,
	containersFailed int64,
	storageBatchesWritten int64,
	storageBatchesFailed int64,
	retryQueueSize int64,
	activeSubscribers int64,
	circuitBreakerState int,
	healthy bool,
) {
	metricsCollectionCount.Add(float64(collectionCount))
	metricsCollectionErrors.Add(float64(collectionErrors))
	metricsContainersProcessed.Add(float64(containersProcessed))
	metricsContainersFailed.Add(float64(containersFailed))
	metricsStorageBatchesWritten.Add(float64(storageBatchesWritten))
	metricsStorageBatchesFailed.Add(float64(storageBatchesFailed))
	metricsRetryQueueSize.Set(float64(retryQueueSize))
	metricsActiveSubscribers.Set(float64(activeSubscribers))
	metricsCircuitBreakerState.Set(float64(circuitBreakerState))
	if healthy {
		metricsHealthy.Set(1)
	} else {
		metricsHealthy.Set(0)
	}
}

