package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Firewall operations

func (pc *ProxmoxClient) configureVMFirewall(ctx context.Context, nodeName string, vmID int, organizationID string, allowInterVM bool) error {
	// Enable firewall on the VM
	enableEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", nodeName, vmID)
	enableData := url.Values{}
	enableData.Set("enable", "1")

	resp, err := pc.apiRequestForm(ctx, "PUT", enableEndpoint, enableData)
	if err != nil {
		return fmt.Errorf("failed to enable firewall: %w", err)
	}
	if resp != nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			// Firewall might already be enabled or permission issue - log and continue
			logger.Debug("[ProxmoxClient] Firewall enable returned status %d (may already be enabled)", resp.StatusCode)
		}
	}

	if !allowInterVM {
		// Block inter-VM communication by default
		// Strategy: Add firewall rules that allow gateway SSH access, then block inter-VM traffic
		// Note: This is a simplified approach. In production, you might want to:
		// 1. Use Proxmox firewall aliases to track VM IPs
		// 2. Create rules that specifically block traffic from other VMs
		// 3. Allow established/related connections to maintain existing sessions

		// Get bridge name (default to vmbr0)
		bridgeName := "vmbr0"

		// Get gateway IP from environment or use default subnet gateway
		// Gateway IP is 10.15.3.10 for the 10.15.3.0/24 subnet (as per docs)
		gatewayIP := os.Getenv("VPS_GATEWAY_IP")
		if gatewayIP == "" {
			// Default gateway IP for VPS subnet (10.15.3.0/24)
			gatewayIP = "10.15.3.10"
		}

		// First, add a rule to ALLOW SSH from the gateway (before the blocking rule)
		// This ensures the gateway can connect to VPS instances for SSH proxying
		// Rules are processed in order, so we add this at position 0 to ensure it's processed first
		ruleEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules?pos=0", nodeName, vmID)
		allowRuleData := url.Values{}
		allowRuleData.Set("enable", "1")
		allowRuleData.Set("action", "ACCEPT")
		allowRuleData.Set("type", "in")
		allowRuleData.Set("source", gatewayIP)
		allowRuleData.Set("dport", "22")
		allowRuleData.Set("proto", "tcp")
		allowRuleData.Set("comment", "Allow SSH from gateway")

		allowRuleResp, err := pc.apiRequestForm(ctx, "POST", ruleEndpoint, allowRuleData)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to add firewall rule to allow SSH from gateway: %v", err)
		} else if allowRuleResp != nil {
			allowRuleResp.Body.Close()
			if allowRuleResp.StatusCode == http.StatusOK {
				logger.Info("[ProxmoxClient] Added firewall rule to allow SSH from gateway (%s) for VM %d", gatewayIP, vmID)
			} else {
				logger.Debug("[ProxmoxClient] Gateway SSH allow rule creation returned status %d", allowRuleResp.StatusCode)
			}
		}

		// Add firewall rule to block inter-VM traffic
		// Rule: Block incoming traffic from other VMs on the same bridge
		// We'll use a rule that blocks traffic from the bridge interface
		// This blocks traffic from other VMs while allowing gateway/internet traffic
		blockRuleData := url.Values{}
		blockRuleData.Set("enable", "1")
		blockRuleData.Set("action", "REJECT")
		blockRuleData.Set("type", "in")
		blockRuleData.Set("iface", bridgeName)
		blockRuleData.Set("comment", "Block inter-VM communication (default security)")
		// Note: This is a basic rule. For production, you would:
		// - Use firewall aliases to track VM IPs
		// - Block specific source IPs or subnets
		// - Allow established/related connections

		ruleResp, err := pc.apiRequestForm(ctx, "POST", ruleEndpoint, blockRuleData)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to add firewall rule to block inter-VM communication: %v", err)
			logger.Info("[ProxmoxClient] Note: Firewall rules can be configured manually in Proxmox to block inter-VM communication")
			// Continue - firewall rules can be configured manually
		} else if ruleResp != nil {
			ruleResp.Body.Close()
			if ruleResp.StatusCode == http.StatusOK {
				logger.Info("[ProxmoxClient] Added firewall rule to block inter-VM communication for VM %d", vmID)
			} else {
				logger.Debug("[ProxmoxClient] Firewall rule creation returned status %d (may need manual configuration)", ruleResp.StatusCode)
				logger.Info("[ProxmoxClient] Note: Configure firewall rules manually in Proxmox to block inter-VM communication")
			}
		}
	} else {
		// Allow inter-VM communication within organization
		// Create or use a security group for this organization
		securityGroupName := fmt.Sprintf("org-%s", organizationID)

		// Ensure security group exists
		if err := pc.ensureSecurityGroup(ctx, securityGroupName); err != nil {
			logger.Warn("[ProxmoxClient] Failed to ensure security group %s: %v", securityGroupName, err)
			// Continue - security groups can be configured manually
		}

		// Add VM to security group
		// In Proxmox, security groups are configured at the VM level via firewall aliases/groups
		// We'll add the VM to a firewall group that allows inter-VM communication
		groupEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", nodeName, vmID)
		groupData := url.Values{}
		groupData.Set("enable", "1")
		// Note: Proxmox security groups work differently - we'll use firewall rules instead
		// For now, we'll just enable firewall and let the security group handle rules

		groupResp, err := pc.apiRequestForm(ctx, "PUT", groupEndpoint, groupData)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to configure VM for security group: %v", err)
		} else if groupResp != nil {
			groupResp.Body.Close()
			logger.Info("[ProxmoxClient] Configured VM %d for inter-VM communication (security group: %s)", vmID, securityGroupName)
		}
	}

	return nil
}

func (pc *ProxmoxClient) ensureSecurityGroup(ctx context.Context, groupName string) error {
	// Check if security group exists
	// Proxmox uses firewall aliases for grouping
	// We'll create an alias for the organization if it doesn't exist
	// Note: This is a simplified implementation - Proxmox security groups are more complex

	// For now, we'll just log that the security group should be created manually
	// In a full implementation, we would:
	// 1. Create a firewall alias for the organization
	// 2. Add VMs to that alias
	// 3. Create firewall rules that allow traffic between VMs in the same alias

	logger.Debug("[ProxmoxClient] Security group %s should be configured manually in Proxmox", groupName)
	return nil
}

func (pc *ProxmoxClient) ListFirewallRules(ctx context.Context, nodeName string, vmID int) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list firewall rules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list firewall rules: %s (status: %d)", string(body), resp.StatusCode)
	}

	var rulesResp struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rulesResp); err != nil {
		return nil, fmt.Errorf("failed to decode firewall rules response: %w", err)
	}

	return rulesResp.Data, nil
}

func (pc *ProxmoxClient) GetFirewallRule(ctx context.Context, nodeName string, vmID int, pos int) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", nodeName, vmID, pos)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall rule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get firewall rule: %s (status: %d)", string(body), resp.StatusCode)
	}

	var ruleResp struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ruleResp); err != nil {
		return nil, fmt.Errorf("failed to decode firewall rule response: %w", err)
	}

	return ruleResp.Data, nil
}

func (pc *ProxmoxClient) CreateFirewallRule(ctx context.Context, nodeName string, vmID int, ruleData url.Values, pos *int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", nodeName, vmID)
	if pos != nil {
		endpoint = fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules?pos=%d", nodeName, vmID, *pos)
	}

	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, ruleData)
	if err != nil {
		return fmt.Errorf("failed to create firewall rule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create firewall rule: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) UpdateFirewallRule(ctx context.Context, nodeName string, vmID int, pos int, ruleData url.Values) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", nodeName, vmID, pos)

	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, ruleData)
	if err != nil {
		return fmt.Errorf("failed to update firewall rule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update firewall rule: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) DeleteFirewallRule(ctx context.Context, nodeName string, vmID int, pos int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", nodeName, vmID, pos)

	resp, err := pc.apiRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete firewall rule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete firewall rule: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) GetFirewallOptions(ctx context.Context, nodeName string, vmID int) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall options: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get firewall options: %s (status: %d)", string(body), resp.StatusCode)
	}

	var optionsResp struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&optionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode firewall options response: %w", err)
	}

	return optionsResp.Data, nil
}

func (pc *ProxmoxClient) UpdateFirewallOptions(ctx context.Context, nodeName string, vmID int, optionsData url.Values) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", nodeName, vmID)

	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, optionsData)
	if err != nil {
		return fmt.Errorf("failed to update firewall options: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update firewall options: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}
