package security

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"vps-gateway/internal/logger"
)

var (
	// Strict validation patterns to prevent command injection
	ipv4Pattern = regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}$`)
	macPattern  = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
)

// validateIPv4 validates an IPv4 address string to prevent injection
func validateIPv4(ip string) error {
	if !ipv4Pattern.MatchString(ip) {
		return fmt.Errorf("invalid IPv4 address format: %s", ip)
	}
	// Additional validation: ensure octets are in valid range
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return fmt.Errorf("invalid IPv4 address: %s", ip)
	}
	return nil
}

// validateMAC validates a MAC address string to prevent injection
func validateMAC(mac string) error {
	normalized := strings.ToLower(strings.TrimSpace(mac))
	if !macPattern.MatchString(normalized) {
		return fmt.Errorf("invalid MAC address format: %s", mac)
	}
	return nil
}

// Manager manages security measures for IP allocations (firewall rules, ARP entries)
type Manager struct {
	uplinkInterface string // Uplink interface (e.g., vmbr0) for routing
}

// NewManager creates a new security manager
func NewManager() (*Manager, error) {
	// Auto-detect uplink interface (default route interface)
	uplinkInterface := os.Getenv("GATEWAY_UPLINK_INTERFACE")
	if uplinkInterface == "" {
		detected, err := detectUplinkInterface()
		if err != nil {
			return nil, fmt.Errorf("failed to detect uplink interface: %w", err)
		}
		uplinkInterface = detected
		logger.Info("Auto-detected uplink interface: %s", uplinkInterface)
	}

	return &Manager{
		uplinkInterface: uplinkInterface,
	}, nil
}

// SecurePublicIP applies security measures for a public IP allocation
// This includes:
// - Firewall rules to prevent IP hijacking (only allow traffic from correct MAC)
// - Static ARP entry to prevent ARP spoofing
// - Routing configuration (if needed)
func (s *Manager) SecurePublicIP(ctx context.Context, publicIP, macAddress, vpsID string) error {
	logger.Info("Securing public IP %s for VPS %s (MAC: %s)", publicIP, vpsID, macAddress)

	// SECURITY: Validate inputs to prevent command injection
	if err := validateIPv4(publicIP); err != nil {
		return fmt.Errorf("invalid public IP: %w", err)
	}
	if err := validateMAC(macAddress); err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	// Validate inputs (secondary check using net.ParseIP)
	ip := net.ParseIP(publicIP)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("invalid IPv4 address: %s", publicIP)
	}

	mac := strings.ToLower(strings.TrimSpace(macAddress))
	if mac == "" {
		return fmt.Errorf("MAC address is required for public IP security")
	}

	// 1. Add firewall rule: Only allow traffic from this MAC using this IP
	if err := s.addFirewallRule(publicIP, mac); err != nil {
		return fmt.Errorf("failed to add firewall rule: %w", err)
	}

	// 2. Add static ARP entry to prevent ARP spoofing
	if err := s.addStaticARP(publicIP, mac); err != nil {
		// Log warning but don't fail - ARP entry might already exist
		logger.Warn("Failed to add static ARP entry (may already exist): %v", err)
	}

	// 3. Ensure IP is routable (add route if needed)
	// For public IPs on vmbr0, routing is typically handled by the network
	// But we can verify the route exists
	if err := s.ensureRoute(publicIP); err != nil {
		logger.Warn("Failed to ensure route for public IP: %v", err)
		// Don't fail - routing might be handled by network configuration
	}

	logger.Info("Successfully secured public IP %s for VPS %s", publicIP, vpsID)
	return nil
}

// RemovePublicIPSecurity removes security measures for a public IP
func (s *Manager) RemovePublicIPSecurity(ctx context.Context, publicIP, macAddress string) error {
	logger.Info("Removing security for public IP %s (MAC: %s)", publicIP, macAddress)

	// SECURITY: Validate inputs to prevent command injection
	if err := validateIPv4(publicIP); err != nil {
		logger.Warn("Invalid public IP during removal: %v", err)
		return err
	}
	if macAddress != "" {
		if err := validateMAC(macAddress); err != nil {
			logger.Warn("Invalid MAC address during removal: %v", err)
			// Continue - we still want to try removing firewall rules
		}
	}

	// Remove firewall rule
	if err := s.removeFirewallRule(publicIP, macAddress); err != nil {
		logger.Warn("Failed to remove firewall rule: %v", err)
		// Continue - rule might not exist
	}

	// Remove static ARP entry
	if err := s.removeStaticARP(publicIP); err != nil {
		logger.Warn("Failed to remove static ARP entry: %v", err)
		// Continue - entry might not exist
	}

	logger.Info("Successfully removed security for public IP %s", publicIP)
	return nil
}

// addFirewallRule adds an iptables rule to only allow traffic from the specified MAC using the IP
func (s *Manager) addFirewallRule(ip, mac string) error {
	// Check if rule already exists
	exists, err := s.firewallRuleExists(ip, mac)
	if err != nil {
		return fmt.Errorf("failed to check firewall rule: %w", err)
	}
	if exists {
		logger.Info("Firewall rule already exists for %s (MAC: %s)", ip, mac)
		return nil
	}

	// Add rule: Allow traffic from this IP only if source MAC matches
	// Rule: -A FORWARD -s <ip> -m mac --mac-source <mac> -j ACCEPT
	// Then add default DROP for this IP from other MACs (handled by default policy)

	// First, add explicit ACCEPT rule for correct MAC
	cmd := exec.Command("iptables",
		"-A", "FORWARD",
		"-s", ip,
		"-m", "mac", "--mac-source", mac,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", fmt.Sprintf("vps-gateway-public-ip-%s", ip),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add firewall rule: %w (output: %s)", err, string(output))
	}

	// Add DROP rule for this IP from other MACs (more restrictive)
	// This prevents IP hijacking by other VPSs
	cmd = exec.Command("iptables",
		"-A", "FORWARD",
		"-s", ip,
		"!", "-m", "mac", "--mac-source", mac,
		"-j", "DROP",
		"-m", "comment",
		"--comment", fmt.Sprintf("vps-gateway-block-hijack-%s", ip),
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		// Log warning but don't fail - the ACCEPT rule is the critical one
		logger.Warn("Failed to add DROP rule for IP hijacking prevention: %v (output: %s)", err, string(output))
	}

	logger.Info("Added firewall rules for public IP %s (MAC: %s)", ip, mac)
	return nil
}

// removeFirewallRule removes firewall rules for an IP
func (s *Manager) removeFirewallRule(ip, mac string) error {
	// Remove ACCEPT rule
	cmd := exec.Command("iptables",
		"-D", "FORWARD",
		"-s", ip,
		"-m", "mac", "--mac-source", mac,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", fmt.Sprintf("vps-gateway-public-ip-%s", ip),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Rule might not exist, which is fine
		if !strings.Contains(string(output), "No chain/target/match") {
			logger.Warn("Failed to remove ACCEPT firewall rule: %v (output: %s)", err, string(output))
		}
	}

	// Remove DROP rule
	cmd = exec.Command("iptables",
		"-D", "FORWARD",
		"-s", ip,
		"!", "-m", "mac", "--mac-source", mac,
		"-j", "DROP",
		"-m", "comment",
		"--comment", fmt.Sprintf("vps-gateway-block-hijack-%s", ip),
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		// Rule might not exist, which is fine
		if !strings.Contains(string(output), "No chain/target/match") {
			logger.Warn("Failed to remove DROP firewall rule: %v (output: %s)", err, string(output))
		}
	}

	return nil
}

// firewallRuleExists checks if a firewall rule exists
func (s *Manager) firewallRuleExists(ip, mac string) (bool, error) {
	cmd := exec.Command("iptables",
		"-C", "FORWARD",
		"-s", ip,
		"-m", "mac", "--mac-source", mac,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", fmt.Sprintf("vps-gateway-public-ip-%s", ip),
	)

	err := cmd.Run()
	return err == nil, nil // If command succeeds, rule exists
}

// addStaticARP adds a static ARP entry to prevent ARP spoofing
func (s *Manager) addStaticARP(ip, mac string) error {
	// Check if static ARP entry already exists
	cmd := exec.Command("arp", "-n", ip)
	output, err := cmd.Output()
	if err == nil {
		// Entry exists, check if MAC matches
		if strings.Contains(string(output), mac) {
			logger.Info("Static ARP entry already exists for %s (MAC: %s)", ip, mac)
			return nil
		}
		// MAC doesn't match, remove old entry first
		s.removeStaticARP(ip)
	}

	// Add static ARP entry
	cmd = exec.Command("arp", "-s", ip, mac)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add static ARP entry: %w (output: %s)", err, string(output))
	}

	logger.Info("Added static ARP entry for %s -> %s", ip, mac)
	return nil
}

// removeStaticARP removes a static ARP entry
func (s *Manager) removeStaticARP(ip string) error {
	cmd := exec.Command("arp", "-d", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Entry might not exist, which is fine
		if !strings.Contains(string(output), "No entry") {
			logger.Warn("Failed to remove static ARP entry: %v (output: %s)", err, string(output))
		}
		return nil
	}

	logger.Info("Removed static ARP entry for %s", ip)
	return nil
}

// ensureRoute ensures a route exists for the public IP
// For public IPs on vmbr0, routing is typically automatic, but we verify
func (s *Manager) ensureRoute(ip string) error {
	// Check if route exists
	cmd := exec.Command("ip", "route", "get", ip)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Route exists
		return nil
	}

	// Route doesn't exist, but for public IPs on vmbr0, this is usually fine
	// The network stack will handle routing automatically
	logger.Debug("No explicit route found for %s (this is normal for public IPs on %s)", ip, s.uplinkInterface)
	return nil
}

// detectUplinkInterface detects the primary uplink interface (default route)
func detectUplinkInterface() (string, error) {
	// Get default route interface
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	// Parse output: "default via <gateway> dev <interface> ..."
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "dev" && i+1 < len(fields) {
				return fields[i+1], nil
			}
		}
	}

	// Fallback: try to find interface with default gateway
	cmd = exec.Command("ip", "route", "get", "8.8.8.8")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect uplink interface: %w", err)
	}

	// Parse: "8.8.8.8 via <gateway> dev <interface> ..."
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("could not detect uplink interface")
}
