package orchestrator

import "sync"

// Global metrics streamer registry for accessing from other services
var (
	globalMetricsStreamer *MetricsStreamer
	metricsStreamerMutex  sync.RWMutex
)

// SetGlobalMetricsStreamer sets the global metrics streamer instance
func SetGlobalMetricsStreamer(streamer *MetricsStreamer) {
	metricsStreamerMutex.Lock()
	defer metricsStreamerMutex.Unlock()
	globalMetricsStreamer = streamer
}

// GetGlobalMetricsStreamer returns the global metrics streamer instance
func GetGlobalMetricsStreamer() *MetricsStreamer {
	metricsStreamerMutex.RLock()
	defer metricsStreamerMutex.RUnlock()
	return globalMetricsStreamer
}
