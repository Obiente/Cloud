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

	// Game server metrics
	gameServerCPUUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obiente_game_server_cpu_usage",
			Help: "Game server CPU usage (0-1, where 1 = 100%)",
		},
		[]string{"game_server_id"},
	)

	gameServerMemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obiente_game_server_memory_usage_bytes",
			Help: "Game server memory usage in bytes",
		},
		[]string{"game_server_id"},
	)

	gameServerNetworkRxBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_game_server_network_rx_bytes_total",
			Help: "Total network received bytes for game server",
		},
		[]string{"game_server_id"},
	)

	gameServerNetworkTxBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_game_server_network_tx_bytes_total",
			Help: "Total network transmitted bytes for game server",
		},
		[]string{"game_server_id"},
	)

	gameServerDiskReadBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_game_server_disk_read_bytes_total",
			Help: "Total disk read bytes for game server",
		},
		[]string{"game_server_id"},
	)

	gameServerDiskWriteBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_game_server_disk_write_bytes_total",
			Help: "Total disk write bytes for game server",
		},
		[]string{"game_server_id"},
	)

	// Deployment metrics
	deploymentCPUUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obiente_deployment_cpu_usage",
			Help: "Deployment CPU usage (0-1, where 1 = 100%)",
		},
		[]string{"deployment_id"},
	)

	deploymentMemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obiente_deployment_memory_usage_bytes",
			Help: "Deployment memory usage in bytes",
		},
		[]string{"deployment_id"},
	)

	deploymentNetworkRxBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_network_rx_bytes_total",
			Help: "Total network received bytes for deployment",
		},
		[]string{"deployment_id"},
	)

	deploymentNetworkTxBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_network_tx_bytes_total",
			Help: "Total network transmitted bytes for deployment",
		},
		[]string{"deployment_id"},
	)

	deploymentDiskReadBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_disk_read_bytes_total",
			Help: "Total disk read bytes for deployment",
		},
		[]string{"deployment_id"},
	)

	deploymentDiskWriteBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_disk_write_bytes_total",
			Help: "Total disk write bytes for deployment",
		},
		[]string{"deployment_id"},
	)

	deploymentRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_requests_total",
			Help: "Total request count for deployment",
		},
		[]string{"deployment_id"},
	)

	deploymentErrorCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_deployment_errors_total",
			Help: "Total error count for deployment",
		},
		[]string{"deployment_id"},
	)

	// DNS Delegation metrics
	dnsDelegationRecordsPushed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_dns_delegation_records_pushed_total",
			Help: "Total number of DNS records pushed via delegation",
		},
		[]string{"organization_id", "api_key_id", "record_type"},
	)

	dnsDelegationRecordsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obiente_dns_delegation_records_active",
			Help: "Current number of active delegated DNS records",
		},
		[]string{"organization_id", "api_key_id", "record_type"},
	)

	dnsDelegationRecordsExpired = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_dns_delegation_records_expired_total",
			Help: "Total number of delegated DNS records that expired",
		},
		[]string{"organization_id", "api_key_id", "record_type"},
	)

	dnsDelegationPushErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "obiente_dns_delegation_push_errors_total",
			Help: "Total number of errors when pushing DNS records",
		},
		[]string{"organization_id", "api_key_id", "error_type"},
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

// RecordGameServerMetrics records game server metrics in Prometheus
func RecordGameServerMetrics(gameServerID string, cpuUsage float64, memoryUsage int64, networkRxBytes int64, networkTxBytes int64, diskReadBytes int64, diskWriteBytes int64) {
	gameServerCPUUsage.WithLabelValues(gameServerID).Set(cpuUsage)
	gameServerMemoryUsage.WithLabelValues(gameServerID).Set(float64(memoryUsage))
	gameServerNetworkRxBytes.WithLabelValues(gameServerID).Add(float64(networkRxBytes))
	gameServerNetworkTxBytes.WithLabelValues(gameServerID).Add(float64(networkTxBytes))
	gameServerDiskReadBytes.WithLabelValues(gameServerID).Add(float64(diskReadBytes))
	gameServerDiskWriteBytes.WithLabelValues(gameServerID).Add(float64(diskWriteBytes))
}

// RecordDeploymentMetrics records deployment metrics in Prometheus
func RecordDeploymentMetrics(deploymentID string, cpuUsage float64, memoryUsage int64, networkRxBytes int64, networkTxBytes int64, diskReadBytes int64, diskWriteBytes int64, requestCount int64, errorCount int64) {
	deploymentCPUUsage.WithLabelValues(deploymentID).Set(cpuUsage)
	deploymentMemoryUsage.WithLabelValues(deploymentID).Set(float64(memoryUsage))
	deploymentNetworkRxBytes.WithLabelValues(deploymentID).Add(float64(networkRxBytes))
	deploymentNetworkTxBytes.WithLabelValues(deploymentID).Add(float64(networkTxBytes))
	deploymentDiskReadBytes.WithLabelValues(deploymentID).Add(float64(diskReadBytes))
	deploymentDiskWriteBytes.WithLabelValues(deploymentID).Add(float64(diskWriteBytes))
	deploymentRequestCount.WithLabelValues(deploymentID).Add(float64(requestCount))
	deploymentErrorCount.WithLabelValues(deploymentID).Add(float64(errorCount))
}

// RecordDNSDelegationPush records a DNS delegation record push
func RecordDNSDelegationPush(organizationID, apiKeyID, recordType string) {
	orgID := organizationID
	if orgID == "" {
		orgID = "unknown"
	}
	keyID := apiKeyID
	if keyID == "" {
		keyID = "unknown"
	}
	rt := recordType
	if rt == "" {
		rt = "unknown"
	}
	dnsDelegationRecordsPushed.WithLabelValues(orgID, keyID, rt).Inc()
}

// UpdateDNSDelegationActiveRecords updates the count of active delegated DNS records
func UpdateDNSDelegationActiveRecords(organizationID, apiKeyID, recordType string, count int) {
	orgID := organizationID
	if orgID == "" {
		orgID = "unknown"
	}
	keyID := apiKeyID
	if keyID == "" {
		keyID = "unknown"
	}
	rt := recordType
	if rt == "" {
		rt = "unknown"
	}
	dnsDelegationRecordsActive.WithLabelValues(orgID, keyID, rt).Set(float64(count))
}

// RecordDNSDelegationExpired records an expired delegated DNS record
func RecordDNSDelegationExpired(organizationID, apiKeyID, recordType string) {
	orgID := organizationID
	if orgID == "" {
		orgID = "unknown"
	}
	keyID := apiKeyID
	if keyID == "" {
		keyID = "unknown"
	}
	rt := recordType
	if rt == "" {
		rt = "unknown"
	}
	dnsDelegationRecordsExpired.WithLabelValues(orgID, keyID, rt).Inc()
}

// RecordDNSDelegationPushError records an error when pushing a DNS delegation record
func RecordDNSDelegationPushError(organizationID, apiKeyID, errorType string) {
	orgID := organizationID
	if orgID == "" {
		orgID = "unknown"
	}
	keyID := apiKeyID
	if keyID == "" {
		keyID = "unknown"
	}
	et := errorType
	if et == "" {
		et = "unknown"
	}
	dnsDelegationPushErrors.WithLabelValues(orgID, keyID, et).Inc()
}

