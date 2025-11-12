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
	"strings"
	"sync"
	"time"

	"vps-gateway/internal/logger"
)

// Manager manages DHCP leases using dnsmasq
type Manager struct {
	poolStart      net.IP
	poolEnd        net.IP
	subnetMask     net.IPMask
	gateway        net.IP
	dnsServers     []net.IP
	interfaceName  string
	leasesFile     string
	hostsFile      string
	allocations    map[string]*Allocation // vps_id -> allocation
	mu             sync.RWMutex
	dhcpRunning    bool
}

// Allocation represents an IP allocation for a VPS
type Allocation struct {
	VPSID         string
	OrganizationID string
	IPAddress     net.IP
	MACAddress    string
	AllocatedAt   time.Time
	LeaseExpires  time.Time
}

// Config holds DHCP configuration
type Config struct {
	PoolStart     string
	PoolEnd       string
	SubnetMask    string
	Gateway       string
	DNSServers    string // Comma-separated
	Interface     string
	LeasesDir     string
}

// NewManager creates a new DHCP manager
func NewManager() (*Manager, error) {
	config := &Config{
		PoolStart:  os.Getenv("GATEWAY_DHCP_POOL_START"),
		PoolEnd:    os.Getenv("GATEWAY_DHCP_POOL_END"),
		SubnetMask: os.Getenv("GATEWAY_DHCP_SUBNET_MASK"),
		Gateway:    os.Getenv("GATEWAY_DHCP_GATEWAY"),
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
		dnsServers:    dnsServers,
		interfaceName: config.Interface,
		hostsFile:     hostsFile,
		leasesFile:    leasesFile,
		allocations:   make(map[string]*Allocation),
	}

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

	return manager, nil
}

// AllocateIP allocates an IP address for a VPS
func (m *Manager) AllocateIP(ctx context.Context, vpsID, orgID, macAddress, preferredIP string) (*Allocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already allocated
	if alloc, exists := m.allocations[vpsID]; exists {
		return alloc, nil
	}

	// Determine IP to allocate
	var ip net.IP
	if preferredIP != "" {
		ip = net.ParseIP(preferredIP)
		if ip == nil {
			return nil, fmt.Errorf("invalid preferred IP address: %s", preferredIP)
		}
		// Check if IP is in pool
		if !m.isIPInPool(ip) {
			return nil, fmt.Errorf("preferred IP %s is not in DHCP pool", preferredIP)
		}
		// Check if IP is already allocated
		for _, alloc := range m.allocations {
			if alloc.IPAddress.Equal(ip) {
				return nil, fmt.Errorf("IP %s is already allocated", preferredIP)
			}
		}
	} else {
		// Find next available IP
		var err error
		ip, err = m.findNextAvailableIP()
		if err != nil {
			return nil, fmt.Errorf("failed to find available IP: %w", err)
		}
	}

	// Create allocation
	alloc := &Allocation{
		VPSID:         vpsID,
		OrganizationID: orgID,
		IPAddress:     ip,
		MACAddress:    macAddress,
		AllocatedAt:   time.Now(),
		LeaseExpires:  time.Now().Add(24 * time.Hour), // 24 hour lease
	}

	m.allocations[vpsID] = alloc

	// Update dnsmasq hosts file
	if err := m.updateHostsFile(); err != nil {
		delete(m.allocations, vpsID)
		return nil, fmt.Errorf("failed to update hosts file: %w", err)
	}

	// Reload dnsmasq
	if err := m.reloadDNSMasq(); err != nil {
		logger.Error("Failed to reload dnsmasq after allocation: %v", err)
		// Continue anyway - allocation is saved
	}

	// Persist allocation
	if err := m.saveAllocations(); err != nil {
		logger.Error("Failed to save allocations: %v", err)
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

	// Persist allocation
	if err := m.saveAllocations(); err != nil {
		logger.Error("Failed to save allocations: %v", err)
	}

	logger.Info("Released IP %s for VPS %s", alloc.IPAddress.String(), vpsID)
	return nil
}

// ListIPs lists all allocated IPs
// This function syncs with actual DHCP leases to return the real IP addresses
func (m *Manager) ListIPs(ctx context.Context, orgID, vpsID string) ([]*Allocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sync allocations with actual DHCP leases
	if err := m.syncWithLeases(); err != nil {
		logger.Warn("Failed to sync with DHCP leases: %v", err)
		// Continue with existing allocations if sync fails
	}

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

	return m.poolStart.String(), m.poolEnd.String(), m.subnetMask.String(), m.gateway.String(), dnsStrs
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

func (m *Manager) isIPInPool(ip net.IP) bool {
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
	writer.WriteString("# dnsmasq hosts file - managed by vps-gateway\n")
	writer.WriteString("# Format: <ip> <hostname> [mac]\n\n")

	// Write allocations
	for _, alloc := range m.allocations {
		line := fmt.Sprintf("%s %s", alloc.IPAddress.String(), alloc.VPSID)
		if alloc.MACAddress != "" {
			line += fmt.Sprintf(" %s", alloc.MACAddress)
		}
		line += "\n"
		writer.WriteString(line)
	}

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
	
	// Network interface and listen addresses
	// Use listen-address instead of bind-interfaces to have more control
	// Listen on the gateway IP (for DHCP) and 127.0.0.1 (for local DNS queries)
	writer.WriteString(fmt.Sprintf("interface=%s\n", m.interfaceName))
	writer.WriteString(fmt.Sprintf("listen-address=%s\n", m.gateway.String()))
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
	writer.WriteString("dhcp-authoritative\n")
	writer.WriteString("log-dhcp\n")
	writer.WriteString("log-queries\n")

	return nil
}

func (m *Manager) reloadDNSMasq() error {
	// Send SIGHUP to dnsmasq to reload configuration
	cmd := exec.Command("pkill", "-HUP", "dnsmasq")
	if err := cmd.Run(); err != nil {
		// If pkill fails, dnsmasq might not be running - try to start it
		logger.Warn("Failed to reload dnsmasq, attempting to start: %v", err)
		return m.startDNSMasq()
	}
	logger.Debug("Reloaded dnsmasq configuration")
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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		ip := net.ParseIP(parts[0])
		if ip == nil {
			continue
		}

		vpsID := parts[1]
		macAddress := ""
		if len(parts) >= 3 {
			macAddress = parts[2]
		}

		// Create allocation (we don't have org ID or timestamps from file)
		m.allocations[vpsID] = &Allocation{
			VPSID:         vpsID,
			IPAddress:     ip,
			MACAddress:    macAddress,
			AllocatedAt:   time.Now(),
			LeaseExpires:  time.Now().Add(24 * time.Hour),
		}
	}

	return scanner.Err()
}

// syncWithLeases reads the actual dnsmasq leases file and updates allocations
// to match the real IP addresses assigned by DHCP
func (m *Manager) syncWithLeases() error {
	file, err := os.Open(m.leasesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Leases file doesn't exist yet, that's okay
		}
		return fmt.Errorf("failed to open leases file: %w", err)
	}
	defer file.Close()

	// Map of VPS ID -> actual lease info
	leaseMap := make(map[string]struct {
		ip       net.IP
		mac      string
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

		mac := parts[1]
		ip := net.ParseIP(parts[2])
		if ip == nil {
			continue
		}

		hostname := parts[3]
		// Skip if hostname is "*" (unknown hostname)
		if hostname == "*" {
			continue
		}

		// Store lease info by hostname (VPS ID)
		if existing, exists := leaseMap[hostname]; !exists || existing.expires < leaseExpiry {
			leaseMap[hostname] = struct {
				ip       net.IP
				mac      string
				expires  int64
			}{
				ip:      ip,
				mac:     mac,
				expires: leaseExpiry,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read leases file: %w", err)
	}

	// Update allocations with actual lease information
	updated := false
	for vpsID, lease := range leaseMap {
		alloc, exists := m.allocations[vpsID]
		if !exists {
			// New lease for a VPS we don't have in allocations
			// This can happen if a VM was created outside our system
			logger.Debug("Found lease for unknown VPS %s with IP %s", vpsID, lease.ip.String())
			continue
		}

		// Update IP if it differs
		if !alloc.IPAddress.Equal(lease.ip) {
			logger.Info("Syncing allocation for VPS %s: updating IP from %s to %s (from DHCP lease)", vpsID, alloc.IPAddress.String(), lease.ip.String())
			alloc.IPAddress = lease.ip
			updated = true
		}

		// Update MAC if it differs
		if alloc.MACAddress != lease.mac {
			logger.Info("Syncing allocation for VPS %s: updating MAC from %s to %s (from DHCP lease)", vpsID, alloc.MACAddress, lease.mac)
			alloc.MACAddress = lease.mac
			updated = true
		}

		// Update lease expiry
		alloc.LeaseExpires = time.Unix(lease.expires, 0)
	}

	// Update hosts file if allocations changed
	if updated {
		if err := m.updateHostsFile(); err != nil {
			return fmt.Errorf("failed to update hosts file after sync: %w", err)
		}
		if err := m.reloadDNSMasq(); err != nil {
			logger.Warn("Failed to reload dnsmasq after sync: %v", err)
		}
	}

	return nil
}

