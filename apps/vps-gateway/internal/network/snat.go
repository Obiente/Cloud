package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"vps-gateway/internal/logger"
)

// SNATManager manages iptables SNAT rules for outbound traffic
type SNATManager struct {
	outboundIP    string
	vpsSubnet     string // CIDR notation (e.g., "10.15.3.0/24")
	outboundIface string
	ruleComment   string // Unique comment to identify our rules
}

// NewSNATManager creates a new SNAT manager
// outboundIP: The IP address to use for SNAT (optional, if empty, SNAT won't be configured)
// gatewayIP: The gateway IP address (used to calculate subnet)
// subnetMask: The subnet mask (used to calculate subnet)
// outboundIface: The network interface for outbound traffic (optional, will be auto-detected if empty)
func NewSNATManager(outboundIP, gatewayIP, subnetMask, outboundIface string) (*SNATManager, error) {
	if outboundIP == "" {
		// No outbound IP configured, return nil manager (no-op)
		return nil, nil
	}

	// Validate outbound IP
	ip := net.ParseIP(outboundIP)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid outbound IP address: %s", outboundIP)
	}

	// Validate gateway IP
	gateway := net.ParseIP(gatewayIP)
	if gateway == nil || gateway.To4() == nil {
		return nil, fmt.Errorf("invalid gateway IP address: %s", gatewayIP)
	}

	// Calculate subnet CIDR from gateway and subnet mask
	subnetCIDR, err := calculateSubnetCIDR(gateway, subnetMask)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate subnet CIDR: %w", err)
	}

	// Auto-detect outbound interface if not provided
	if outboundIface == "" {
		detected, err := detectOutboundInterface()
		if err != nil {
			return nil, fmt.Errorf("failed to detect outbound interface: %w", err)
		}
		outboundIface = detected
		logger.Info("Auto-detected outbound interface: %s", outboundIface)
	}

	// Create unique rule comment to identify our rules
	ruleComment := fmt.Sprintf("vps-gateway-snat-%s", outboundIP)

	return &SNATManager{
		outboundIP:    outboundIP,
		vpsSubnet:     subnetCIDR,
		outboundIface: outboundIface,
		ruleComment:   ruleComment,
	}, nil
}

// ConfigureSNAT sets up iptables SNAT rules for VPS outbound traffic
func (s *SNATManager) ConfigureSNAT() error {
	if s == nil {
		// No manager (outbound IP not configured)
		return nil
	}

	logger.Info("Configuring iptables SNAT: %s -> %s on interface %s", s.vpsSubnet, s.outboundIP, s.outboundIface)

	// Check if rule already exists
	exists, err := s.ruleExists()
	if err != nil {
		return fmt.Errorf("failed to check if SNAT rule exists: %w", err)
	}

	if exists {
		logger.Info("SNAT rule already exists, skipping configuration")
		return nil
	}

	// Add SNAT rule: iptables -t nat -A POSTROUTING -s <subnet> -o <interface> -j SNAT --to-source <outbound-ip>
	cmd := exec.Command("iptables",
		"-t", "nat",
		"-A", "POSTROUTING",
		"-s", s.vpsSubnet,
		"-o", s.outboundIface,
		"-j", "SNAT",
		"--to-source", s.outboundIP,
		"-m", "comment",
		"--comment", s.ruleComment,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add SNAT rule: %w (output: %s)", err, string(output))
	}

	logger.Info("Successfully configured SNAT rule: %s -> %s on %s", s.vpsSubnet, s.outboundIP, s.outboundIface)
	return nil
}

// RemoveSNAT removes iptables SNAT rules
func (s *SNATManager) RemoveSNAT() error {
	if s == nil {
		// No manager (outbound IP not configured)
		return nil
	}

	logger.Info("Removing iptables SNAT rule for %s", s.outboundIP)

	// Remove SNAT rule by comment
	cmd := exec.Command("iptables",
		"-t", "nat",
		"-D", "POSTROUTING",
		"-s", s.vpsSubnet,
		"-o", s.outboundIface,
		"-j", "SNAT",
		"--to-source", s.outboundIP,
		"-m", "comment",
		"--comment", s.ruleComment,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Rule might not exist, which is fine
		if strings.Contains(string(output), "No chain/target/match") ||
			strings.Contains(string(output), "Bad rule") {
			logger.Info("SNAT rule not found (may have been removed already)")
			return nil
		}
		return fmt.Errorf("failed to remove SNAT rule: %w (output: %s)", err, string(output))
	}

	logger.Info("Successfully removed SNAT rule for %s", s.outboundIP)
	return nil
}

// ruleExists checks if the SNAT rule already exists
func (s *SNATManager) ruleExists() (bool, error) {
	// List existing rules and check for our comment
	cmd := exec.Command("iptables",
		"-t", "nat",
		"-L", "POSTROUTING",
		"-n", "-v",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to list iptables rules: %w", err)
	}

	// Check if our comment appears in the output
	return strings.Contains(string(output), s.ruleComment), nil
}

// calculateSubnetCIDR calculates the subnet CIDR from gateway IP and subnet mask
func calculateSubnetCIDR(gateway net.IP, subnetMask string) (string, error) {
	gateway = gateway.To4()
	if gateway == nil {
		return "", fmt.Errorf("gateway IP is not IPv4")
	}

	// Parse subnet mask
	var mask net.IPMask
	if strings.Contains(subnetMask, ".") {
		// Dotted decimal notation (e.g., "255.255.255.0")
		maskIP := net.ParseIP(subnetMask)
		if maskIP == nil {
			return "", fmt.Errorf("invalid subnet mask: %s", subnetMask)
		}
		mask = net.IPMask(maskIP.To4())
	} else {
		// CIDR notation (e.g., "24")
		var cidr int
		if _, err := fmt.Sscanf(subnetMask, "%d", &cidr); err != nil || cidr < 0 || cidr > 32 {
			return "", fmt.Errorf("invalid subnet mask: %s", subnetMask)
		}
		mask = net.CIDRMask(cidr, 32)
	}

	// Calculate network address
	network := gateway.Mask(mask)

	// Get CIDR prefix length
	ones, _ := mask.Size()

	return fmt.Sprintf("%s/%d", network.String(), ones), nil
}

// detectOutboundInterface detects the primary outbound network interface
// by checking the default route
func detectOutboundInterface() (string, error) {
	// Get default route interface using 'ip route'
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
	// This is a more robust approach
	cmd = exec.Command("ip", "route", "get", "8.8.8.8")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect outbound interface: %w", err)
	}

	// Parse: "8.8.8.8 via <gateway> dev <interface> ..."
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("could not detect outbound interface")
}
