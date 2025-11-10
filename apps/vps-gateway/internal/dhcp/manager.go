package dhcp

import (
	"bufio"
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
		SubnetMask: os.Getenv("GATEWAY_DHCP_SUBNET"),
		Gateway:    os.Getenv("GATEWAY_DHCP_GATEWAY"),
		DNSServers: os.Getenv("GATEWAY_DHCP_DNS"),
		Interface:  os.Getenv("GATEWAY_DHCP_INTERFACE"),
		LeasesDir:  os.Getenv("GATEWAY_DHCP_LEASES_DIR"),
	}

	if config.LeasesDir == "" {
		config.LeasesDir = "/var/lib/vps-gateway"
	}

	// Validate required config
	if config.PoolStart == "" || config.PoolEnd == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_POOL_START and GATEWAY_DHCP_POOL_END are required")
	}
	if config.SubnetMask == "" {
		return nil, fmt.Errorf("GATEWAY_DHCP_SUBNET is required")
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
	subnetMask := net.IPMask(net.ParseIP(config.SubnetMask).To4())

	if poolStart == nil || poolEnd == nil {
		return nil, fmt.Errorf("invalid IP addresses in pool configuration")
	}
	if gateway == nil {
		return nil, fmt.Errorf("invalid gateway IP address")
	}
	if subnetMask == nil {
		return nil, fmt.Errorf("invalid subnet mask")
	}

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
	}

	// Start dnsmasq
	if err := manager.startDNSMasq(); err != nil {
		return nil, fmt.Errorf("failed to start dnsmasq: %w", err)
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
func (m *Manager) ListIPs(ctx context.Context, orgID, vpsID string) ([]*Allocation, error) {
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
	)

	// Start in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dnsmasq: %w", err)
	}

	m.dhcpRunning = true
	logger.Info("Started dnsmasq")
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
	
	// Network interface
	writer.WriteString(fmt.Sprintf("interface=%s\n", m.interfaceName))
	writer.WriteString("bind-interfaces\n\n")
	
	// DHCP configuration
	// Convert subnet mask to CIDR notation (e.g., 255.255.255.0 -> 24)
	ones, _ := m.subnetMask.Size()
	writer.WriteString(fmt.Sprintf("dhcp-range=%s,%s,%d,12h\n", m.poolStart.String(), m.poolEnd.String(), ones))
	writer.WriteString(fmt.Sprintf("dhcp-option=option:router,%s\n", m.gateway.String()))
	
	// DNS servers
	for _, dns := range m.dnsServers {
		optionNum := 6 // DNS option
		writer.WriteString(fmt.Sprintf("dhcp-option=option:%d,%s\n", optionNum, dns.String()))
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

