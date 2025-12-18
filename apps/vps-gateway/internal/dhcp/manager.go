package dhcp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"vps-gateway/internal/logger"

	"connectrpc.com/connect"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Manager manages DHCP leases using dnsmasq
type Manager struct {
	poolStart          net.IP
	poolEnd            net.IP
	subnetMask         net.IPMask
	gateway            net.IP
	listenIP           net.IP // IP address to listen on (for multi-node support)
	dnsServers         []net.IP
	interfaceName      string
	leasesFile         string
	hostsFile          string
	nodeName           string                     // Gateway node name (provided by VPS service)
	allocations        map[string]*Allocation     // vps_id -> allocation
	mu                 sync.RWMutex
	dhcpRunning        bool
	dnsmasqPID         int
	allocationTTL      time.Duration
	vpsServiceClient   vpsv1connect.VPSServiceClient // gRPC client to VPS Service for lease operations (deprecated)
	vpsServiceClientMu sync.RWMutex                  // Protects client access
	apiClient          APIClient                     // API client for bidirectional stream communication
	apiClientMu        sync.RWMutex                  // Protects API client access
}

// APIClient interface defines the methods needed from the API client
type APIClient interface {
	FindVPSByLease(ctx context.Context, ip string, mac string) (*vpsv1.FindVPSByLeaseResponse, error)
}

// Allocation represents an IP allocation for a VPS
type Allocation struct {
	VPSID          string
	OrganizationID string
	IPAddress      net.IP
	MACAddress     string
	AllocatedAt    time.Time
	LeaseExpires   time.Time
}

// LeaseInfo represents an active DHCP lease from dnsmasq
type LeaseInfo struct {
	MAC       string
	IP        net.IP
	Hostname  string
	ExpiresAt time.Time
}

// Config holds DHCP configuration
type Config struct {
	PoolStart  string
	PoolEnd    string
	SubnetMask string
	Gateway    string
	ListenIP   string // IP to listen on (optional, defaults to gateway IP)
	DNSServers string // Comma-separated
	Interface  string
	LeasesDir  string
}

// NewManager creates a new DHCP manager
func NewManager() (*Manager, error) {
	config := &Config{
		PoolStart:  os.Getenv("GATEWAY_DHCP_POOL_START"),
		PoolEnd:    os.Getenv("GATEWAY_DHCP_POOL_END"),
		SubnetMask: os.Getenv("GATEWAY_DHCP_SUBNET_MASK"),
		Gateway:    os.Getenv("GATEWAY_DHCP_GATEWAY"),
		ListenIP:   os.Getenv("GATEWAY_DHCP_LISTEN_IP"), // Optional: IP to listen on (for multi-node)
		DNSServers: os.Getenv("GATEWAY_DHCP_DNS"),
		Interface:  os.Getenv("GATEWAY_DHCP_INTERFACE"),
		LeasesDir:  os.Getenv("GATEWAY_DHCP_LEASES_DIR"),
	}

	if config.LeasesDir == "" {
		config.LeasesDir = "/var/lib/obiente/vps-gateway"
	}

	// Validate required config
	if config.PoolStart == "" || config.PoolEnd == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_POOL_START and GATEWAY_DHCP_POOL_END are required")
	}
	if config.SubnetMask == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_SUBNET_MASK is required")
	}
	if config.Gateway == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_GATEWAY is required")
	}
	if config.Interface == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_INTERFACE is required")
	}

	poolStart := net.ParseIP(config.PoolStart)
	poolEnd := net.ParseIP(config.PoolEnd)
	gateway := net.ParseIP(config.Gateway)

	// Parse listen IP (optional, defaults to gateway IP for backward compatibility)
	var listenIP net.IP
	if config.ListenIP != "" {
		listenIP = net.ParseIP(config.ListenIP)
		if listenIP == nil || listenIP.To4() == nil {
			return nil, fmt.Errorf("invalid listen IP address: %s", config.ListenIP)
		}
		logger.Info("Using custom listen IP: %s (gateway IP: %s)", listenIP.String(), gateway.String())
	} else {
		// Default to gateway IP for backward compatibility
		listenIP = gateway
		logger.Info("Using gateway IP as listen address: %s", listenIP.String())
	}

	// Parse subnet mask - can be in CIDR notation (e.g., "24") or dotted decimal (e.g., "255.255.255.0")
	var subnetMask net.IPMask
	if strings.Contains(config.SubnetMask, ".") {
		// Dotted decimal notation (e.g., "255.255.255.0")
		maskIP := net.ParseIP(config.SubnetMask)
		if maskIP == nil {
			return nil, fmt.Errorf("invalid subnet mask format: %s (expected dotted decimal like 255.255.255.0)", config.SubnetMask)
		}
		subnetMask = net.IPMask(maskIP.To4())
		if subnetMask == nil {
			return nil, fmt.Errorf("invalid subnet mask: %s (not a valid IPv4 mask)", config.SubnetMask)
		}
	} else {
		// CIDR notation (e.g., "24")
		var cidr int
		if _, err := fmt.Sscanf(config.SubnetMask, "%d", &cidr); err != nil || cidr < 0 || cidr > 32 {
			return nil, fmt.Errorf("invalid subnet mask format: %s (expected CIDR like 24 or dotted decimal like 255.255.255.0)", config.SubnetMask)
		}
		subnetMask = net.CIDRMask(cidr, 32)
	}

	if poolStart == nil || poolEnd == nil {
		return nil, fmt.Errorf("invalid IP addresses in pool configuration")
	}
	if gateway == nil {
		return nil, fmt.Errorf("invalid gateway IP address")
	}
	if subnetMask == nil {
		return nil, fmt.Errorf("invalid subnet mask")
	}

	// Validate subnet mask
	ones, bits := subnetMask.Size()
	if ones == 0 || bits == 0 {
		return nil, fmt.Errorf("invalid subnet mask: %s (parsed as %d/%d)", config.SubnetMask, ones, bits)
	}
	logger.Info("Parsed subnet mask: %s -> /%d", config.SubnetMask, ones)

	// Parse DNS servers
	var dnsServers []net.IP
	if config.DNSServers != "" {
		for _, dns := range strings.Split(config.DNSServers, ",") {
			dns = strings.TrimSpace(dns)
			if ip := net.ParseIP(dns); ip != nil {
				dnsServers = append(dnsServers, ip)
			}
		}
	}
	if len(dnsServers) == 0 {
		// Default to gateway as DNS
		dnsServers = []net.IP{gateway}
	}

	// Ensure leases directory exists
	if err := os.MkdirAll(config.LeasesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create leases directory: %w", err)
	}

	hostsFile := filepath.Join(config.LeasesDir, "dnsmasq.hosts")
	leasesFile := filepath.Join(config.LeasesDir, "dnsmasq.leases")

	manager := &Manager{
		poolStart:     poolStart,
		poolEnd:       poolEnd,
		subnetMask:    subnetMask,
		gateway:       gateway,
		listenIP:      listenIP,
		dnsServers:    dnsServers,
		interfaceName: config.Interface,
		hostsFile:     hostsFile,
		leasesFile:    leasesFile,
		allocations:   make(map[string]*Allocation),
		allocationTTL: 1 * time.Hour,
	}

	// The gateway does not initiate connections to VPS service endpoints.
	// VPS services should connect to the gateway and call SetVPSServiceClient
	// to provide a persistent client. Do not create outgoing clients here.

	// Note: Redis is NOT used for cross-gateway coordination.
	// Redis instances (if configured) are local per gateway, not shared.
	// The database (accessed via VPS service bidirectional stream) is the source of truth.

	// Load existing allocations from file
	if err := manager.loadAllocations(); err != nil {
		logger.Warn("Failed to load existing allocations: %v", err)
	} else {
		logger.Info("Loaded %d existing IP allocations from hosts file", len(manager.allocations))
		for vpsID, alloc := range manager.allocations {
			logger.Debug("Restored allocation: VPS %s -> IP %s", vpsID, alloc.IPAddress.String())
		}
	}

	// Start dnsmasq
	if err := manager.startDNSMasq(); err != nil {
		return nil, fmt.Errorf("failed to start dnsmasq: %w", err)
	}

	// Ensure dnsmasq has the latest hosts file (in case it was updated before dnsmasq started)
	// This is a safety measure - dnsmasq should have read it during start, but reload to be sure
	if len(manager.allocations) > 0 {
		if err := manager.reloadDNSMasq(); err != nil {
			logger.Warn("Failed to reload dnsmasq after loading allocations: %v", err)
		} else {
			logger.Debug("Reloaded dnsmasq to ensure hosts file is loaded")
		}
	}

	// Allow TTL override via env var (seconds)
	if ttlStr := os.Getenv("GATEWAY_ALLOCATION_TTL_SECONDS"); ttlStr != "" {
		if ttlSec, err := strconv.Atoi(ttlStr); err == nil && ttlSec > 0 {
			manager.allocationTTL = time.Duration(ttlSec) * time.Second
			logger.Info("Allocation TTL set to %s via env", manager.allocationTTL.String())
		} else {
			logger.Warn("Invalid GATEWAY_ALLOCATION_TTL_SECONDS=%s, using default %s", ttlStr, manager.allocationTTL.String())
		}
	}

	// Start background reconciler to pick up leases and remove allocations for deleted VPS
	go manager.backgroundReconciler()

	return manager, nil
}

// SetVPSServiceClient sets the VPS Service gRPC client for lease registration/release
// This should be called from main() after the client is created
func (m *Manager) SetVPSServiceClient(client vpsv1connect.VPSServiceClient) {
	m.vpsServiceClientMu.Lock()
	defer m.vpsServiceClientMu.Unlock()
	m.vpsServiceClient = client
	logger.Info("VPS Service client configured for lease operations")
	
	// Trigger initial sync to discover and resolve existing leases to VPS IDs
	// Run in background to avoid blocking the caller
	go func() {
		time.Sleep(2 * time.Second) // Brief delay to let the client stabilize
		logger.Info("Triggering initial hosts file sync after VPS Service client configured")
		if err := m.updateHostsFile(); err != nil {
			logger.Warn("Initial hosts file sync failed: %v", err)
		} else {
			if err := m.reloadDNSMasq(); err != nil {
				logger.Warn("Failed to reload dnsmasq after initial sync: %v", err)
			} else {
				logger.Info("Initial hosts file sync completed successfully")
			}
		}
	}()
}

// SetAPIClient sets the API client for bidirectional stream communication
// This should be called from main() after the API client is created
func (m *Manager) SetAPIClient(client APIClient) {
	m.apiClientMu.Lock()
	defer m.apiClientMu.Unlock()
	m.apiClient = client
	logger.Info("API client configured for lease resolution")
	
	// Trigger initial sync to discover and resolve existing leases to VPS IDs
	// Run in background to avoid blocking the caller
	go func() {
		time.Sleep(2 * time.Second) // Brief delay to let the client stabilize
		logger.Info("Triggering initial hosts file sync after API client configured")
		if err := m.updateHostsFile(); err != nil {
			logger.Warn("Initial hosts file sync failed: %v", err)
		} else {
			if err := m.reloadDNSMasq(); err != nil {
				logger.Warn("Failed to reload dnsmasq after initial sync: %v", err)
			} else {
				logger.Info("Initial hosts file sync completed successfully")
			}
		}
	}()
}

// SyncHostsFromLeases updates the hosts file from current DHCP leases
// This is called when VPS services connect to resolve existing leases to VPS IDs
func (m *Manager) SyncHostsFromLeases() error {
	if err := m.updateHostsFile(); err != nil {
		return fmt.Errorf("failed to update hosts file: %w", err)
	}
	
	if err := m.reloadDNSMasq(); err != nil {
		return fmt.Errorf("failed to reload dnsmasq: %w", err)
	}
	
	return nil
}

// AllocateIP allocates an IP address for a VPS
// If preferredIP is provided and allowPublicIP is true, it can allocate IPs outside the DHCP pool (for public IPs)
func (m *Manager) AllocateIP(ctx context.Context, vpsID, orgID, macAddress, preferredIP string, allowPublicIP bool) (*Allocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already allocated
	if alloc, exists := m.allocations[vpsID]; exists {
		return alloc, nil
	}

	// Determine IP to allocate
	// The database (via VPS service) is the source of truth for preventing duplicate allocations
	// This gateway only manages its local allocations map
	var ip net.IP
	if preferredIP != "" {
		ip = net.ParseIP(preferredIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid preferred IP address: %s", preferredIP)
		}
		// Check if IP is in pool (unless allowPublicIP is true)
		if !allowPublicIP && !m.IsIPInPool(ip) {
			return nil, fmt.Errorf("preferred IP %s is not in DHCP pool", preferredIP)
		}
		// Check if IP is already allocated locally
		for _, alloc := range m.allocations {
			if alloc.IPAddress.Equal(ip) {
				return nil, fmt.Errorf("IP %s is already allocated", preferredIP)
			}
		}
	} else {
		// Find next available IP from local pool
		var err error
		ip, err = m.findNextAvailableIP()
		if err != nil {
			return nil, fmt.Errorf("failed to find available IP: %w", err)
		}
	}

	// Create allocation
	alloc := &Allocation{
		VPSID:          vpsID,
		OrganizationID: orgID,
		IPAddress:      ip,
		MACAddress:     strings.ToLower(strings.TrimSpace(macAddress)),
		AllocatedAt:    time.Now(),
		LeaseExpires:   time.Now().Add(24 * time.Hour), // 24 hour lease
	}

	m.allocations[vpsID] = alloc

	// For public IPs (outside DHCP pool), skip dnsmasq configuration
	// Public IPs are statically configured on the VPS, not via DHCP
	if allowPublicIP && !m.IsIPInPool(ip) {
		logger.Info("Allocated public IP %s (outside DHCP pool) for VPS %s - skipping dnsmasq configuration", ip.String(), vpsID)
	} else {
		// Update dnsmasq hosts file (for DHCP pool IPs)
		if err := m.updateHostsFile(); err != nil {
			delete(m.allocations, vpsID)
			return nil, fmt.Errorf("failed to update hosts file: %w", err)
		}

		// Reload dnsmasq
		if err := m.reloadDNSMasq(); err != nil {
			logger.Error("Failed to reload dnsmasq after allocation: %v", err)
			// Continue anyway - allocation is saved
		}
	}

	// Persist allocation to local file
	if err := m.saveAllocations(); err != nil {
		logger.Error("Failed to save allocations: %v", err)
	}

	// Register lease with VPS Service API (database as source of truth)
	// This prevents duplicate IP allocations across gateways
	isPublic := allowPublicIP && !m.IsIPInPool(ip)
	if err := m.registerLeaseWithAPI(ctx, vpsID, orgID, ip, isPublic, macAddress); err != nil {
		logger.Error("Failed to register lease with VPS Service: %v", err)
		// Remove local allocation if API registration fails
		delete(m.allocations, vpsID)
		m.saveAllocations()
		return nil, fmt.Errorf("failed to register lease with API: %w", err)
	}

	logger.Info("Allocated IP %s for VPS %s (org: %s)", ip.String(), vpsID, orgID)
	return alloc, nil
}

// ReleaseIP releases an IP address for a VPS
func (m *Manager) ReleaseIP(ctx context.Context, vpsID, ipAddress string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alloc, exists := m.allocations[vpsID]
	if !exists {
		return fmt.Errorf("no allocation found for VPS %s", vpsID)
	}

	if ipAddress != "" && !alloc.IPAddress.Equal(net.ParseIP(ipAddress)) {
		return fmt.Errorf("IP %s does not match allocated IP %s for VPS %s", ipAddress, alloc.IPAddress.String(), vpsID)
	}

	delete(m.allocations, vpsID)

	// Update dnsmasq hosts file
	if err := m.updateHostsFile(); err != nil {
		logger.Error("Failed to update hosts file: %v", err)
	}

	// Reload dnsmasq
	if err := m.reloadDNSMasq(); err != nil {
		logger.Error("Failed to reload dnsmasq after release: %v", err)
	}

	// Persist allocation to local file
	if err := m.saveAllocations(); err != nil {
		logger.Error("Failed to save allocations: %v", err)
	}

	// Release lease from VPS Service API (database as source of truth)
	if err := m.releaseLeaseWithAPI(ctx, vpsID, alloc.MACAddress); err != nil {
		logger.Error("Failed to release lease with VPS Service: %v", err)
		// Non-fatal - local release is complete, will clean up eventually
	}

	logger.Info("Released IP %s for VPS %s", alloc.IPAddress.String(), vpsID)
	return nil
}

// ListIPs lists all allocated IPs
// This function syncs with actual DHCP leases to return the real IP addresses
func (m *Manager) ListIPs(ctx context.Context, orgID, vpsID string) ([]*Allocation, error) {
	// Ensure allocations are synced with actual DHCP leases. `syncWithLeases`
	// takes its own lock so callers should not hold `m.mu` here.
	if err := m.syncWithLeases(); err != nil {
		logger.Warn("Failed to sync with DHCP leases: %v", err)
		// Continue with existing allocations if sync fails
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Allocation
	for _, alloc := range m.allocations {
		if orgID != "" && alloc.OrganizationID != orgID {
			continue
		}
		if vpsID != "" && alloc.VPSID != vpsID {
			continue
		}
		result = append(result, alloc)
	}

	return result, nil
}

// GetConfig returns the DHCP configuration
func (m *Manager) GetConfig() (poolStart, poolEnd, subnetMask, gateway string, dnsServers []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dnsStrs := make([]string, len(m.dnsServers))
	for i, dns := range m.dnsServers {
		dnsStrs[i] = dns.String()
	}

	// Convert subnet mask to dotted decimal format
	maskIP := net.IP(m.subnetMask)
	subnetMaskStr := maskIP.String()

	return m.poolStart.String(), m.poolEnd.String(), subnetMaskStr, m.gateway.String(), dnsStrs
}

// SetNodeName sets the gateway node name (told to us by VPS service on registration)
// This is critical for proper lease registration with the correct gateway_node
func (m *Manager) SetNodeName(nodeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeName = nodeName
	logger.Info("[DHCP] Gateway node name set to: %s", nodeName)
}

// GetStats returns DHCP statistics
func (m *Manager) GetStats() (totalIPs, allocatedIPs int, dhcpStatus string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalIPs = m.countIPsInPool()
	allocatedIPs = len(m.allocations)

	if m.dhcpRunning {
		dhcpStatus = "running"
	} else {
		dhcpStatus = "stopped"
	}

	return totalIPs, allocatedIPs, dhcpStatus
}

// Close cleans up the DHCP manager
func (m *Manager) Close() error {
	// Save allocations before closing
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveAllocations()
}

// Helper methods

func (m *Manager) IsIPInPool(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	start4 := m.poolStart.To4()
	end4 := m.poolEnd.To4()
	if start4 == nil || end4 == nil {
		return false
	}

	// Compare IPs as 32-bit integers
	ipInt := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
	startInt := uint32(start4[0])<<24 | uint32(start4[1])<<16 | uint32(start4[2])<<8 | uint32(start4[3])
	endInt := uint32(end4[0])<<24 | uint32(end4[1])<<16 | uint32(end4[2])<<8 | uint32(end4[3])

	return ipInt >= startInt && ipInt <= endInt
}

func (m *Manager) findNextAvailableIP() (net.IP, error) {
	start4 := m.poolStart.To4()
	end4 := m.poolEnd.To4()
	if start4 == nil || end4 == nil {
		return nil, fmt.Errorf("invalid pool configuration")
	}

	startInt := uint32(start4[0])<<24 | uint32(start4[1])<<16 | uint32(start4[2])<<8 | uint32(start4[3])
	endInt := uint32(end4[0])<<24 | uint32(end4[1])<<16 | uint32(end4[2])<<8 | uint32(end4[3])

	// Create set of allocated IPs
	allocatedSet := make(map[uint32]bool)
	for _, alloc := range m.allocations {
		ip4 := alloc.IPAddress.To4()
		if ip4 != nil {
			ipInt := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
			allocatedSet[ipInt] = true
		}
	}

	// Find first available IP
	for ipInt := startInt; ipInt <= endInt; ipInt++ {
		if !allocatedSet[ipInt] {
			return net.IP{
				byte(ipInt >> 24),
				byte(ipInt >> 16),
				byte(ipInt >> 8),
				byte(ipInt),
			}, nil
		}
	}

	return nil, fmt.Errorf("no available IPs in pool")
}

// Note: Redis-based distributed reservation has been removed.
// The database (via VPS service) is the authoritative source for preventing duplicate allocations.
// Gateway only manages local allocations and syncs from database via bidirectional stream.

func (m *Manager) countIPsInPool() int {
	start4 := m.poolStart.To4()
	end4 := m.poolEnd.To4()
	if start4 == nil || end4 == nil {
		return 0
	}

	startInt := uint32(start4[0])<<24 | uint32(start4[1])<<16 | uint32(start4[2])<<8 | uint32(start4[3])
	endInt := uint32(end4[0])<<24 | uint32(end4[1])<<16 | uint32(end4[2])<<8 | uint32(end4[3])

	count := int(endInt - startInt + 1)
	if count < 0 {
		return 0
	}
	return count
}

func (m *Manager) updateHostsFile() error {
	file, err := os.Create(m.hostsFile)
	if err != nil {
		return fmt.Errorf("failed to create hosts file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.WriteString("# dnsmasq hosts/dhcp file - managed by vps-gateway\n")
	writer.WriteString("# DNS entries: <ip> <hostname>\n")
	writer.WriteString("# DHCP static entries: dhcp-host=<mac>,<ip>,<hostname>\n\n")

	// Attempt to discover MACs from active leases so we can emit dhcp-host entries
	// even when allocations were created without a MAC (timing race with DHCP).
	ipToMac := make(map[string]string)
	activeLeases, err := m.GetActiveLeases()
	if err == nil {
		for _, l := range activeLeases {
			if l.MAC != "" && l.IP != nil {
				ipToMac[l.IP.String()] = strings.ToLower(l.MAC)
			}
		}

		// Create allocations from active leases when the lease hostname is a VPS ID
		// (e.g., dnsmasq hostname set to "vps-<id>"). This helps include those
		// active leases in the hosts file even if they weren't previously allocated.
		now := time.Now()
		
		// First pass: handle vps-* hostnames (no API calls needed)
		m.mu.Lock()
		for _, l := range activeLeases {
			if l.Hostname == "" || l.IP == nil {
				continue
			}
			// Treat hostnames that look like our VPS IDs (prefix "vps-") as allocations
			if strings.HasPrefix(l.Hostname, "vps-") {
				vpsID := l.Hostname
				// If allocation exists, merge MAC/IP
				if alloc, ok := m.allocations[vpsID]; ok {
					if alloc.IPAddress == nil || !alloc.IPAddress.Equal(l.IP) {
						alloc.IPAddress = l.IP
					}
					if alloc.MACAddress == "" && l.MAC != "" {
						alloc.MACAddress = strings.ToLower(l.MAC)
					}
				} else {
					m.allocations[vpsID] = &Allocation{
						VPSID:        vpsID,
						IPAddress:    l.IP,
						MACAddress:   strings.ToLower(l.MAC),
						AllocatedAt:  now,
						LeaseExpires: l.ExpiresAt,
					}
				}
			}
		}
		m.mu.Unlock()
		
		// Second pass: resolve non-vps-* hostnames via API (no lock held during network calls)
		// Only attempt if we have connected VPS service instances to avoid startup spam
		m.apiClientMu.RLock()
		client := m.apiClient
		m.apiClientMu.RUnlock()
		
		// Check if we have any connected streams before attempting API calls
		hasConnectedAPI := false
		if svc, ok := client.(interface{ HasConnectedStreams() bool }); ok {
			hasConnectedAPI = svc.HasConnectedStreams()
		}
		
		if client != nil && hasConnectedAPI {
			for _, l := range activeLeases {
				if l.Hostname == "" || l.IP == nil {
					continue
				}
				// Skip vps-* hostnames (already handled above)
				if strings.HasPrefix(l.Hostname, "vps-") {
					continue
				}
				
				// Resolve via API without holding mutex
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				resp, err := client.FindVPSByLease(ctx, l.IP.String(), strings.ToLower(l.MAC))
				cancel()
				
				if err == nil && resp != nil && resp.GetVpsId() != "" {
					vpsID := resp.GetVpsId()
					orgID := resp.GetOrganizationId()
					
					// Now acquire lock to update allocations
					m.mu.Lock()
					if alloc, ok := m.allocations[vpsID]; ok {
						if alloc.IPAddress == nil || !alloc.IPAddress.Equal(l.IP) {
							alloc.IPAddress = l.IP
						}
						if alloc.MACAddress == "" && l.MAC != "" {
							alloc.MACAddress = strings.ToLower(l.MAC)
						}
						if alloc.OrganizationID == "" && orgID != "" {
							alloc.OrganizationID = orgID
						}
					} else {
						m.allocations[vpsID] = &Allocation{
							VPSID:          vpsID,
							OrganizationID: orgID,
							IPAddress:      l.IP,
							MACAddress:     strings.ToLower(l.MAC),
							AllocatedAt:    now,
							LeaseExpires:   l.ExpiresAt,
						}
					}
					m.mu.Unlock()
					
					// Register the discovered lease with the database
					// This ensures future lookups will work even if the VPS service restarts
					// Pass MAC directly since we already have it from the lease
					isPublic := !m.IsIPInPool(l.IP)
					regCtx, regCancel := context.WithTimeout(context.Background(), 5*time.Second)
					if regErr := m.registerLeaseWithAPI(regCtx, vpsID, orgID, l.IP, isPublic, l.MAC); regErr != nil {
						logger.Warn("[DHCP] Failed to register discovered lease for VPS %s: %v", vpsID, regErr)
					} else {
						logger.Info("[DHCP] Registered discovered lease for VPS %s (IP: %s, MAC: %s)", vpsID, l.IP.String(), l.MAC)
					}
					regCancel()
				}
			}
		}
	} else {
		logger.Debug("Could not read active leases while updating hosts file: %v", err)
	}

	// Persist updated allocations with discovered MACs (outside of lock)
	if err := m.saveAllocations(); err != nil {
		logger.Warn("Failed to persist allocations after MAC discovery: %v", err)
	}

	// Write allocations (need to hold lock while reading map)
	m.mu.RLock()
	for _, alloc := range m.allocations {
		// DNS host entry
		writer.WriteString(fmt.Sprintf("%s %s\n", alloc.IPAddress.String(), alloc.VPSID))

		// Prefer allocation MAC if present, otherwise fall back to active lease MAC
		mac := strings.ToLower(strings.TrimSpace(alloc.MACAddress))
		if mac == "" {
			if m, ok := ipToMac[alloc.IPAddress.String()]; ok && m != "" {
				mac = m
				// Update in-memory allocation so subsequent operations see the MAC
				alloc.MACAddress = mac
			}
		}

		if mac != "" {
			writer.WriteString(fmt.Sprintf("dhcp-host=%s,%s,%s\n", mac, alloc.IPAddress.String(), alloc.VPSID))
		} else {
			// Log warning when MAC is missing - DHCP won't be enforced but DNS will work
			logger.Warn("Missing MAC for VPS %s (IP %s) - dhcp-host entry skipped, DNS entry only", alloc.VPSID, alloc.IPAddress.String())
		}
	}
	m.mu.RUnlock()

	return nil
}

func (m *Manager) startDNSMasq() error {
	// Generate dnsmasq config file path
	configFile := filepath.Join(filepath.Dir(m.hostsFile), "dnsmasq.conf")

	// Check if dnsmasq is already running (check for our specific config file)
	// This ensures we only detect our own dnsmasq instance, not system dnsmasq
	cmd := exec.Command("pgrep", "-f", fmt.Sprintf("dnsmasq.*%s", configFile))
	if err := cmd.Run(); err == nil {
		logger.Info("dnsmasq is already running with our config")
		m.dhcpRunning = true
		return nil
	}

	// Generate dnsmasq config
	if err := m.generateDNSMasqConfig(configFile); err != nil {
		return fmt.Errorf("failed to generate dnsmasq config: %w", err)
	}

	// Start dnsmasq
	// All configuration is in the config file, we only need to specify the config file location
	cmd = exec.Command("dnsmasq",
		"--no-daemon",
		"--conf-file="+configFile,
		"--log-facility=-", // Log to stderr
	)

	// Capture stderr to see startup errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Start in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dnsmasq: %w", err)
	}

	// Record PID so we can signal the exact process on reload
	if cmd.Process != nil {
		m.dnsmasqPID = cmd.Process.Pid
	}

	// Wait a moment for dnsmasq to start
	time.Sleep(500 * time.Millisecond)

	// Check if process is still running by checking if it exists
	// Use pgrep to verify the process is running
	checkCmd := exec.Command("pgrep", "-f", fmt.Sprintf("dnsmasq.*%s", configFile))
	if err := checkCmd.Run(); err != nil {
		// Process not found - it exited
		stderrOutput := stderr.String()
		return fmt.Errorf("dnsmasq exited immediately: %s", stderrOutput)
	}

	// Verify dnsmasq is actually listening on port 53
	// Try connecting to 127.0.0.1:53
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		conn, err := net.DialTimeout("udp", "127.0.0.1:53", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			logger.Info("Started dnsmasq and verified it's listening on 127.0.0.1:53")
			m.dhcpRunning = true
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Check if process is still running
	checkCmd = exec.Command("pgrep", "-f", fmt.Sprintf("dnsmasq.*%s", configFile))
	if err := checkCmd.Run(); err != nil {
		stderrOutput := stderr.String()
		return fmt.Errorf("dnsmasq failed to start or exited: %s", stderrOutput)
	}

	// Process is running but not listening - log warning but continue
	logger.Warn("dnsmasq process started but not listening on 127.0.0.1:53. Stderr: %s", stderr.String())
	m.dhcpRunning = true
	return nil
}

func (m *Manager) generateDNSMasqConfig(configFile string) error {
	file, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("failed to create dnsmasq config: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write dnsmasq configuration
	writer.WriteString("# dnsmasq configuration - managed by vps-gateway\n")
	writer.WriteString("# Do not edit manually - this file is auto-generated\n\n")

	// Run as root (container is already privileged, no need to drop privileges)
	// This prevents "unknown user or group: dnsmasq" errors in containers
	writer.WriteString("user=root\n")
	writer.WriteString("\n")

	// Network interface and listen addresses
	// Use listen-address instead of bind-interfaces to have more control
	// Listen on the listen IP (for DHCP) and 127.0.0.1 (for local DNS queries)
	// For multi-node deployments, each gateway should have its own listen IP
	writer.WriteString(fmt.Sprintf("interface=%s\n", m.interfaceName))
	writer.WriteString(fmt.Sprintf("listen-address=%s\n", m.listenIP.String()))
	writer.WriteString("listen-address=127.0.0.1\n")
	writer.WriteString("\n")

	// DNS server configuration
	// Enable DNS server on port 53
	writer.WriteString("port=53\n")
	// Set domain for VPS network (optional, can be configured via env var)
	domain := os.Getenv("GATEWAY_DHCP_DOMAIN")
	if domain == "" {
		domain = "vps.local" // Default domain
	}
	writer.WriteString(fmt.Sprintf("domain=%s\n", domain))
	// Enable hostname expansion (allows hostname.domain resolution)
	writer.WriteString("expand-hosts\n")
	// Make dnsmasq authoritative for the local domain
	writer.WriteString(fmt.Sprintf("local=/%s/\n", domain))
	// Enable reading hostnames from hosts file
	writer.WriteString(fmt.Sprintf("addn-hosts=%s\n", m.hostsFile))
	writer.WriteString("\n")

	// DHCP configuration
	// Enable authoritative mode to prevent unauthorized DHCP servers
	// This ensures dnsmasq only responds to DHCP requests for known MAC addresses
	writer.WriteString("dhcp-authoritative\n")
	// Ignore client-supplied hostnames and use our static dhcp-host entries
	// This ensures VPS IDs are always used as hostnames, not guest OS hostnames
	writer.WriteString("dhcp-ignore-names\n")
	// dnsmasq dhcp-range format: start,end,netmask,lease-time
	// The netmask should be in dotted decimal format (e.g., 255.255.255.0)
	// Convert IPMask back to dotted decimal format
	maskIP := net.IP(m.subnetMask)
	netmaskStr := maskIP.String()
	writer.WriteString(fmt.Sprintf("dhcp-range=%s,%s,%s,12h\n", m.poolStart.String(), m.poolEnd.String(), netmaskStr))
	writer.WriteString(fmt.Sprintf("dhcp-option=option:router,%s\n", m.gateway.String()))

	// DNS servers (for upstream DNS resolution)
	// Combine all DNS servers into a single dhcp-option line (option 6 = DNS)
	if len(m.dnsServers) > 0 {
		dnsList := make([]string, len(m.dnsServers))
		for i, dns := range m.dnsServers {
			dnsList[i] = dns.String()
			// Also add as upstream DNS server for dnsmasq itself
			writer.WriteString(fmt.Sprintf("server=%s\n", dns.String()))
		}
		// Format: dhcp-option=6,1.1.1.1,1.0.0.1 (option number, comma-separated DNS servers)
		writer.WriteString(fmt.Sprintf("dhcp-option=6,%s\n", strings.Join(dnsList, ",")))
	}

	// File paths
	writer.WriteString(fmt.Sprintf("dhcp-hostsfile=%s\n", m.hostsFile))
	writer.WriteString(fmt.Sprintf("dhcp-leasefile=%s\n", m.leasesFile))

	// DHCP options
	writer.WriteString("log-dhcp\n")
	writer.WriteString("log-queries\n")

	return nil
}

func (m *Manager) reloadDNSMasq() error {
	// Prefer signaling the specific dnsmasq PID we started to avoid affecting
	// other dnsmasq instances on the host. Fallback to pkill if PID is unknown.
	if m.dnsmasqPID != 0 {
		cmd := exec.Command("kill", "-HUP", fmt.Sprintf("%d", m.dnsmasqPID))
		if err := cmd.Run(); err != nil {
			logger.Warn("Failed to signal dnsmasq PID %d: %v", m.dnsmasqPID, err)
			// Fallback to generic reload which may affect other processes
			fallback := exec.Command("pkill", "-HUP", "dnsmasq")
			if err := fallback.Run(); err != nil {
				logger.Warn("Fallback pkill failed, attempting to start dnsmasq: %v", err)
				return m.startDNSMasq()
			}
		}
		logger.Debug("Signaled dnsmasq PID %d for reload", m.dnsmasqPID)
		return nil
	}

	// No PID recorded - fallback to pkill
	cmd := exec.Command("pkill", "-HUP", "dnsmasq")
	if err := cmd.Run(); err != nil {
		logger.Warn("Failed to reload dnsmasq, attempting to start: %v", err)
		return m.startDNSMasq()
	}
	logger.Debug("Reloaded dnsmasq configuration (pkill)")
	return nil
}

func (m *Manager) saveAllocations() error {
	// Save to JSON file for persistence
	// For now, we'll just ensure the hosts file is up to date
	return m.updateHostsFile()
}

func (m *Manager) loadAllocations() error {
	// Load allocations from hosts file
	file, err := os.Open(m.hostsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's okay
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// We support two formats in the hosts file:
	// 1) "<ip> <vpsID>" (DNS host entry)
	// 2) "dhcp-host=<mac>,<ip>,<vpsID>" (dnsmasq dhcp-host entry)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// dhcp-host entries contain mac/ip/vps as CSV after the '='
		if strings.HasPrefix(line, "dhcp-host=") {
			rhs := strings.TrimPrefix(line, "dhcp-host=")
			// Format: mac,ip,vpsid
			parts := strings.Split(rhs, ",")
			if len(parts) >= 3 {
				mac := strings.ToLower(strings.TrimSpace(parts[0]))
				ip := net.ParseIP(strings.TrimSpace(parts[1]))
				vpsID := strings.TrimSpace(parts[2])
				if ip == nil || vpsID == "" {
					continue
				}

				// Ensure allocation exists and merge data
				alloc, ok := m.allocations[vpsID]
				if !ok {
					alloc = &Allocation{
						VPSID:        vpsID,
						IPAddress:    ip,
						MACAddress:   mac,
						AllocatedAt:  time.Now(),
						LeaseExpires: time.Now().Add(24 * time.Hour),
					}
					m.allocations[vpsID] = alloc
				} else {
					alloc.IPAddress = ip
					alloc.MACAddress = mac
				}
			}
			continue
		}

		// Fallback: plain "<ip> <vpsID>" DNS host entry
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		ip := net.ParseIP(parts[0])
		if ip == nil {
			continue
		}

		vpsID := parts[1]

		// Ensure allocation exists; mac may be filled by a dhcp-host line later
		alloc, ok := m.allocations[vpsID]
		if !ok {
			m.allocations[vpsID] = &Allocation{
				VPSID:        vpsID,
				IPAddress:    ip,
				MACAddress:   "",
				AllocatedAt:  time.Now(),
				LeaseExpires: time.Now().Add(24 * time.Hour),
			}
		} else {
			alloc.IPAddress = ip
		}
	}

	return scanner.Err()
}

// syncWithLeases reads the actual dnsmasq leases file and updates allocations
// to match the real IP addresses assigned by DHCP
func (m *Manager) syncWithLeases() error {
	// Acquire exclusive lock while we inspect and update allocations
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.leasesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Leases file doesn't exist yet, that's okay
		}
		return fmt.Errorf("failed to open leases file: %w", err)
	}
	defer file.Close()

	// Map of MAC address -> actual lease info
	// We use MAC address as the key because the hostname in dnsmasq leases
	// is set by the VM's OS (e.g., "ubuntu") not the VPS ID
	leaseMap := make(map[string]struct {
		ip       net.IP
		hostname string
		expires  int64
	})

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// dnsmasq leases format: <lease_expiry> <mac> <ip> <hostname> [client_id]
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		// Parse lease expiry (Unix timestamp)
		var leaseExpiry int64
		if _, err := fmt.Sscanf(parts[0], "%d", &leaseExpiry); err != nil {
			continue
		}

		// Check if lease is expired
		if leaseExpiry < time.Now().Unix() {
			continue
		}

		mac := strings.ToLower(parts[1]) // Normalize MAC to lowercase
		ip := net.ParseIP(parts[2])
		if ip == nil {
			continue
		}

		hostname := parts[3]

		// Store lease info by MAC address (most reliable identifier)
		if existing, exists := leaseMap[mac]; !exists || existing.expires < leaseExpiry {
			leaseMap[mac] = struct {
				ip       net.IP
				hostname string
				expires  int64
			}{
				ip:       ip,
				hostname: hostname,
				expires:  leaseExpiry,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read leases file: %w", err)
	}

	// Update allocations with actual lease information
	// Build an ip->lease map to support matching by IP when MAC is not present
	ipMap := make(map[string]struct {
		ip       net.IP
		hostname string
		expires  int64
	})
	for mac, l := range leaseMap {
		ipMap[l.ip.String()] = l
		_ = mac
	}

	updated := false
	for vpsID, alloc := range m.allocations {
		// First try to match by MAC if available
		if alloc.MACAddress != "" {
			allocMAC := strings.ToLower(alloc.MACAddress)
			if lease, exists := leaseMap[allocMAC]; exists {
				// Update IP if it differs
				if !alloc.IPAddress.Equal(lease.ip) {
					logger.Info("Syncing allocation for VPS %s: updating IP from %s to %s (from DHCP lease, MAC=%s)", vpsID, alloc.IPAddress.String(), lease.ip.String(), allocMAC)
					alloc.IPAddress = lease.ip
					updated = true
				}
				// Update lease expiry
				alloc.LeaseExpires = time.Unix(lease.expires, 0)
				continue
			}
			logger.Debug("No active lease found for VPS %s (MAC %s)", vpsID, allocMAC)
		}

		// Fallback: try matching by IP (useful when hosts file contains IP->vps mapping but no MAC)
		if alloc.IPAddress != nil {
			if lease, exists := ipMap[alloc.IPAddress.String()]; exists {
				// Fill MAC (if available from leaseMap via reverse lookup)
				// Find the MAC by searching leaseMap entries for this IP
				foundMac := ""
				for mac, l := range leaseMap {
					if l.ip.Equal(lease.ip) {
						foundMac = mac
						break
					}
				}
				if foundMac != "" {
					alloc.MACAddress = foundMac
					logger.Debug("Filled MAC for VPS %s from lease IP %s -> %s", vpsID, alloc.IPAddress.String(), foundMac)
				}

				// Update lease expiry
				alloc.LeaseExpires = time.Unix(lease.expires, 0)
				updated = true
			} else {
				logger.Debug("No active lease found for VPS %s (IP %s)", vpsID, alloc.IPAddress.String())
			}
		}
	}

	// Remove stale allocations that have no active lease entry (DHCP file is source of truth)
	for vpsID, alloc := range m.allocations {
		if alloc.MACAddress == "" {
			continue
		}

		allocMAC := strings.ToLower(alloc.MACAddress)
		if _, exists := leaseMap[allocMAC]; exists {
			continue
		}

		// Only clean up pool-backed leases automatically; public/static IPs are handled separately
		if !m.IsIPInPool(alloc.IPAddress) {
			continue
		}

		logger.Info("Removing stale DHCP allocation with no active lease",
			"vps_id", vpsID,
			"ip", alloc.IPAddress.String(),
			"mac", allocMAC,
		)

		delete(m.allocations, vpsID)
		updated = true

		// Inform API that the lease is gone; ignore errors to avoid blocking cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := m.releaseLeaseWithAPI(ctx, vpsID, allocMAC); err != nil {
			logger.Debug("Non-fatal: failed to release lease with API during stale cleanup: %v", err)
		}
		cancel()
	}

	// Update hosts file if allocations changed
	if updated {
		if err := m.updateHostsFile(); err != nil {
			return fmt.Errorf("failed to update hosts file after sync: %w", err)
		}
		if err := m.reloadDNSMasq(); err != nil {
			logger.Warn("Failed to reload dnsmasq after sync: %v", err)
		}
		if err := m.saveAllocations(); err != nil {
			logger.Warn("Failed to persist allocations after sync: %v", err)
		}
	}

	return nil
}

// GetActiveLeases returns current active leases parsed from dnsmasq lease file
func (m *Manager) GetActiveLeases() ([]LeaseInfo, error) {
	file, err := os.Open(m.leasesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []LeaseInfo{}, nil
		}
		return nil, fmt.Errorf("failed to open leases file: %w", err)
	}
	defer file.Close()

	var leases []LeaseInfo
	scanner := bufio.NewScanner(file)
	now := time.Now().Unix()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		// expiry mac ip hostname [client_id]
		var expiry int64
		if _, err := fmt.Sscanf(parts[0], "%d", &expiry); err != nil {
			continue
		}
		if expiry < now {
			continue
		}
		mac := strings.ToLower(parts[1])
		ip := net.ParseIP(parts[2])
		if ip == nil {
			continue
		}
		hostname := parts[3]

		leases = append(leases, LeaseInfo{
			MAC:       mac,
			IP:        ip,
			Hostname:  hostname,
			ExpiresAt: time.Unix(expiry, 0),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read leases file: %w", err)
	}

	return leases, nil
}

// backgroundReconciler periodically syncs with dnsmasq leases and prunes
// allocations for VPS instances that no longer exist in the VPS Service.
func (m *Manager) backgroundReconciler() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update hosts file first (discovers leases, resolves to VPS IDs, writes entries)
		// This ensures the hosts file is always up-to-date with active leases
		if err := m.updateHostsFile(); err != nil {
			logger.Warn("backgroundReconciler: failed to update hosts file: %v", err)
		} else {
			// Reload dnsmasq to pick up the updated hosts file
			if err := m.reloadDNSMasq(); err != nil {
				logger.Warn("backgroundReconciler: failed to reload dnsmasq: %v", err)
			}
		}

		// Sync leases (updates MACs and expiries in allocations)
		if err := m.syncWithLeases(); err != nil {
			logger.Debug("backgroundReconciler: syncWithLeases failed: %v", err)
			continue
		}

		// Check allocations against VPS Service and remove ones that are deleted
		m.mu.Lock()
		var toRemove []string
		now := time.Now()
		for vpsID, alloc := range m.allocations {
			// Skip if this allocation still has an active lease (keep until gone)
			if alloc.MACAddress != "" {
				continue
			}

			// If allocation is younger than TTL, skip removing (allow time for DHCP)
			if alloc.AllocatedAt.Add(m.allocationTTL).After(now) {
				continue
			}

			// If we have a VPS Service client, check whether the VPS exists; only remove
			// if VPS is NotFound. If client is not configured, fall back to TTL-based removal.
			// Check configured VPS Service client (if any) to verify VPS existence.
			m.vpsServiceClientMu.RLock()
			client := m.vpsServiceClient
			m.vpsServiceClientMu.RUnlock()

			removedByNotFound := false
			if client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := client.GetVPS(ctx, connect.NewRequest(&vpsv1.GetVPSRequest{VpsId: vpsID}))
				cancel()
				if err != nil {
					if cerr, ok := err.(*connect.Error); ok && cerr.Code() == connect.CodeNotFound {
						removedByNotFound = true
					}
				}
			}

			// If VPS service not configured, fall back to TTL-based removal (keep previous behavior)
			if removedByNotFound || client == nil {
				toRemove = append(toRemove, vpsID)
			}
		}

		for _, vpsID := range toRemove {
			alloc := m.allocations[vpsID]
			delete(m.allocations, vpsID)
			logger.Info("Pruned allocation for deleted VPS: %s (ip=%s)", vpsID, alloc.IPAddress.String())

			// Best-effort inform API that lease has been released
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = m.releaseLeaseWithAPI(ctx, vpsID, alloc.MACAddress)
			cancel()
		}

		if len(toRemove) > 0 {
			// Persist changes
			if err := m.updateHostsFile(); err != nil {
				logger.Warn("backgroundReconciler: failed to update hosts file: %v", err)
			}
			if err := m.reloadDNSMasq(); err != nil {
				logger.Warn("backgroundReconciler: failed to reload dnsmasq: %v", err)
			}
			if err := m.saveAllocations(); err != nil {
				logger.Warn("backgroundReconciler: failed to persist allocations: %v", err)
			}
		}
		m.mu.Unlock()
	}
}

// registerLeaseWithAPI registers a lease with the VPS Service API through persistent gRPC connection
// This ensures the lease is recorded in the central database and prevents duplicate allocations
// If macAddress is provided, it will be used directly; otherwise it will be discovered
func (m *Manager) registerLeaseWithAPI(ctx context.Context, vpsID, organizationID string, ipAddress net.IP, isPublic bool, macAddress string) error {
	m.vpsServiceClientMu.RLock()
	client := m.vpsServiceClient
	m.vpsServiceClientMu.RUnlock()

	// Check if client is configured
	if client == nil {
		logger.Warn("VPS Service client not yet configured - lease registration deferred")
		// Non-fatal for now; lease is already allocated locally
		// In production, this would indicate a configuration problem
		return nil
	}

	// Calculate lease expiry (e.g., 24 hours from now)
	expiresAt := time.Now().Add(24 * time.Hour)
	expiresPb := timestamppb.New(expiresAt)

	// Use provided MAC address if available, otherwise attempt discovery
	macAddr := strings.ToLower(strings.TrimSpace(macAddress))
	if macAddr == "" {
		m.mu.RLock()
		if alloc, ok := m.allocations[vpsID]; ok && alloc != nil && alloc.MACAddress != "" {
			macAddr = alloc.MACAddress
		} else {
			// Fallback: try to find allocation by IP if VPSID lookup failed
			for _, a := range m.allocations {
				if a != nil && a.IPAddress != nil && a.IPAddress.Equal(ipAddress) && a.MACAddress != "" {
					macAddr = a.MACAddress
					break
				}
			}
		}
		m.mu.RUnlock()
	}

	// If MAC still unknown, try scanning active leases file for IP -> MAC mapping
	if macAddr == "" {
		if activeLeases, err := m.GetActiveLeases(); err == nil {
			for _, l := range activeLeases {
				if l.IP.Equal(ipAddress) && l.MAC != "" {
					macAddr = l.MAC
					break
				}
			}
		}
	}

	// Enhanced wait-and-retry: sometimes dnsmasq writes the lease after allocation
	// due to DHCP timing. Poll for a few short intervals (longer than before)
	// to try to capture the MAC before sending the registration to the API.
	if macAddr == "" {
		for i := 0; i < 12; i++ { // ~6s total
			time.Sleep(500 * time.Millisecond)

			if activeLeases, err := m.GetActiveLeases(); err == nil {
				for _, l := range activeLeases {
					if l.IP.Equal(ipAddress) && l.MAC != "" {
						macAddr = l.MAC
						break
					}
				}
			}
			if macAddr != "" {
				break
			}
		}
	}

	// Log resolved MAC for debugging
	if macAddr != "" {
		logger.Debug("Resolved MAC for VPS %s IP %s -> %s", vpsID, ipAddress.String(), macAddr)
	} else {
		logger.Debug("No MAC resolved for VPS %s IP %s when registering lease with API", vpsID, ipAddress.String())
	}

	// Get our gateway node name
	m.mu.RLock()
	gatewayNode := m.nodeName
	m.mu.RUnlock()

	// Create the gRPC request
	req := &vpsv1.RegisterLeaseRequest{
		VpsId:          vpsID,
		OrganizationId: organizationID,
		MacAddress:     macAddr,
		IpAddress:      ipAddress.String(),
		ExpiresAt:      expiresPb,
		IsPublic:       isPublic,
		GatewayNode:    gatewayNode, // Tell VPS service which gateway node we are
	}

	// Call the VPS Service RPC with a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.RegisterLease(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("Failed to register lease with VPS Service: %v", err)
		return fmt.Errorf("register lease API call failed: %w", err)
	}

	if !resp.Msg.Success {
		logger.Error("VPS Service rejected lease registration: %s", resp.Msg.Message)
		return fmt.Errorf("lease registration rejected by VPS Service: %s", resp.Msg.Message)
	}

	logger.Debug("Successfully registered lease with VPS Service for VPS %s (IP %s)", vpsID, ipAddress.String())
	return nil
}

// releaseLeaseWithAPI releases a lease from the VPS Service API
// This removes the lease from the central database
func (m *Manager) releaseLeaseWithAPI(ctx context.Context, vpsID, macAddress string) error {
	m.vpsServiceClientMu.RLock()
	client := m.vpsServiceClient
	m.vpsServiceClientMu.RUnlock()

	// Check if client is configured
	if client == nil {
		logger.Debug("VPS Service client not yet configured - lease release deferred")
		// Non-fatal; lease is already released locally
		return nil
	}

	// Create the gRPC request
	req := &vpsv1.ReleaseLeaseRequest{
		VpsId:      vpsID,
		MacAddress: macAddress,
	}

	// Call the VPS Service RPC with a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.ReleaseLease(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("Failed to release lease with VPS Service: %v", err)
		return fmt.Errorf("release lease API call failed: %w", err)
	}

	if !resp.Msg.Success {
		logger.Debug("VPS Service rejected lease release: %s", resp.Msg.Message)
		// Non-fatal - lease is already released locally
		return nil
	}

	logger.Debug("Successfully released lease with VPS Service for VPS %s", vpsID)
	return nil
}

// AddStaticDHCPLease adds a static DHCP lease for a MAC/IP pair (including public IPs)
func (m *Manager) AddStaticDHCPLease(ctx context.Context, macAddress, ipAddress, vpsID, orgID string, isPublic bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}
	alloc := &Allocation{
		VPSID:          vpsID,
		OrganizationID: orgID,
		IPAddress:      ip,
		MACAddress:     strings.ToLower(strings.TrimSpace(macAddress)),
		AllocatedAt:    time.Now(),
		LeaseExpires:   time.Now().Add(24 * time.Hour),
	}
	m.allocations[vpsID] = alloc
	if err := m.updateHostsFile(); err != nil {
		delete(m.allocations, vpsID)
		return fmt.Errorf("failed to update hosts file: %w", err)
	}
	if err := m.reloadDNSMasq(); err != nil {
		logger.Error("Failed to reload dnsmasq after static lease add: %v", err)
	}
	if err := m.saveAllocations(); err != nil {
		logger.Warn("Failed to persist allocations after static lease add: %v", err)
	}
	logger.Info("Added static DHCP lease: MAC=%s IP=%s VPSID=%s is_public=%v", macAddress, ipAddress, vpsID, isPublic)
	return nil
}

// RemoveStaticDHCPLease removes a static DHCP lease for a MAC/IP pair (including public IPs)
func (m *Manager) RemoveStaticDHCPLease(ctx context.Context, macAddress, ipAddress, vpsID, orgID string, isPublic bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	alloc, ok := m.allocations[vpsID]
	if !ok {
		return fmt.Errorf("no allocation found for VPSID %s", vpsID)
	}
	// Compare MACs case-insensitively and normalize inputs
	if strings.ToLower(strings.TrimSpace(alloc.MACAddress)) != strings.ToLower(strings.TrimSpace(macAddress)) || alloc.IPAddress.String() != ipAddress {
		return fmt.Errorf("allocation mismatch for VPSID %s: expected MAC %s IP %s, got MAC %s IP %s", vpsID, macAddress, ipAddress, alloc.MACAddress, alloc.IPAddress.String())
	}
	delete(m.allocations, vpsID)
	if err := m.updateHostsFile(); err != nil {
		return fmt.Errorf("failed to update hosts file: %w", err)
	}
	if err := m.reloadDNSMasq(); err != nil {
		logger.Error("Failed to reload dnsmasq after static lease removal: %v", err)
	}
	if err := m.saveAllocations(); err != nil {
		logger.Warn("Failed to persist allocations after static lease removal: %v", err)
	}
	logger.Info("Removed static DHCP lease: MAC=%s IP=%s VPSID=%s is_public=%v", macAddress, ipAddress, vpsID, isPublic)
	return nil
}

// RegisterLeaseDirectly registers an existing DHCP lease with the VPS Service API
// without modifying local allocations. This is used during self-healing to register
// leases that already exist in dnsmasq but aren't in the database.
func (m *Manager) RegisterLeaseDirectly(ctx context.Context, vpsID, orgID string, ipAddress net.IP, isPublic bool, macAddress string) error {
	return m.registerLeaseWithAPI(ctx, vpsID, orgID, ipAddress, isPublic, macAddress)
}

