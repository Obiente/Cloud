package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// DHCP metrics
	dhcpAllocationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vps_gateway_dhcp_allocations_total",
			Help: "Total number of DHCP IP allocations",
		},
		[]string{"organization_id"},
	)

	dhcpReleasesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vps_gateway_dhcp_releases_total",
			Help: "Total number of DHCP IP releases",
		},
		[]string{"organization_id"},
	)

	dhcpAllocationsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vps_gateway_dhcp_allocations_active",
			Help: "Number of active DHCP IP allocations",
		},
		[]string{"organization_id"},
	)

	dhcpPoolSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vps_gateway_dhcp_pool_size",
			Help: "Total size of DHCP IP pool",
		},
	)

	dhcpPoolAvailable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vps_gateway_dhcp_pool_available",
			Help: "Number of available IPs in DHCP pool",
		},
	)

	// SSH Proxy metrics
	sshProxyConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vps_gateway_ssh_proxy_connections_total",
			Help: "Total number of SSH proxy connections",
		},
		[]string{"organization_id", "vps_id"},
	)

	sshProxyConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vps_gateway_ssh_proxy_connections_active",
			Help: "Number of active SSH proxy connections",
		},
	)

	sshProxyConnectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vps_gateway_ssh_proxy_connection_duration_seconds",
			Help:    "Duration of SSH proxy connections in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17m
		},
		[]string{"organization_id", "vps_id"},
	)

	sshProxyBytesTransmitted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vps_gateway_ssh_proxy_bytes_transmitted_total",
			Help: "Total bytes transmitted through SSH proxy",
		},
		[]string{"organization_id", "vps_id", "direction"}, // direction: "in" or "out"
	)

	// Gateway health metrics
	gatewayUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vps_gateway_uptime_seconds",
			Help: "Gateway uptime in seconds",
		},
	)

	dhcpServerStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vps_gateway_dhcp_server_status",
			Help: "DHCP server status (1=running, 0=stopped)",
		},
	)
)

// Init initializes Prometheus metrics
func Init() {
	prometheus.MustRegister(
		dhcpAllocationsTotal,
		dhcpReleasesTotal,
		dhcpAllocationsActive,
		dhcpPoolSize,
		dhcpPoolAvailable,
		sshProxyConnectionsTotal,
		sshProxyConnectionsActive,
		sshProxyConnectionDuration,
		sshProxyBytesTransmitted,
		gatewayUptime,
		dhcpServerStatus,
	)
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordDHCPAllocation records a DHCP IP allocation
func RecordDHCPAllocation(orgID string) {
	dhcpAllocationsTotal.WithLabelValues(orgID).Inc()
}

// RecordDHCPRelease records a DHCP IP release
func RecordDHCPRelease(orgID string) {
	dhcpReleasesTotal.WithLabelValues(orgID).Inc()
}

// SetDHCPAllocationsActive sets the number of active DHCP allocations
func SetDHCPAllocationsActive(orgID string, count float64) {
	dhcpAllocationsActive.WithLabelValues(orgID).Set(count)
}

// SetDHCPPoolSize sets the total DHCP pool size
func SetDHCPPoolSize(size float64) {
	dhcpPoolSize.Set(size)
}

// SetDHCPPoolAvailable sets the number of available IPs
func SetDHCPPoolAvailable(available float64) {
	dhcpPoolAvailable.Set(available)
}

// RecordSSHProxyConnection records a new SSH proxy connection
func RecordSSHProxyConnection(orgID, vpsID string) {
	sshProxyConnectionsTotal.WithLabelValues(orgID, vpsID).Inc()
}

// SetSSHProxyConnectionsActive sets the number of active SSH proxy connections
func SetSSHProxyConnectionsActive(count float64) {
	sshProxyConnectionsActive.Set(count)
}

// RecordSSHProxyConnectionDuration records the duration of an SSH proxy connection
func RecordSSHProxyConnectionDuration(orgID, vpsID string, duration float64) {
	sshProxyConnectionDuration.WithLabelValues(orgID, vpsID).Observe(duration)
}

// RecordSSHProxyBytes records bytes transmitted through SSH proxy
func RecordSSHProxyBytes(orgID, vpsID, direction string, bytes int64) {
	sshProxyBytesTransmitted.WithLabelValues(orgID, vpsID, direction).Add(float64(bytes))
}

// SetGatewayUptime sets the gateway uptime
func SetGatewayUptime(seconds float64) {
	gatewayUptime.Set(seconds)
}

// SetDHCPServerStatus sets the DHCP server status
func SetDHCPServerStatus(running bool) {
	status := float64(0)
	if running {
		status = 1
	}
	dhcpServerStatus.Set(status)
}

