package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// StartLeaseReconciler starts a background goroutine that ensures all VPSes have their DHCP leases registered
// This handles cases where:
// - Gateway was down during VPS creation
// - VPS was created before lease tracking was implemented
// - Database was wiped/migrated but VMs still exist
func (vm *VPSManager) StartLeaseReconciler(ctx context.Context) {
	logger.Info("[LeaseReconciler] Starting background lease reconciliation")
	
	ticker := time.NewTicker(5 * time.Minute) // Run every 5 minutes
	defer ticker.Stop()

	// Run immediately on startup
	if err := vm.reconcileAllLeases(ctx); err != nil {
		logger.Warn("[LeaseReconciler] Initial reconciliation failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("[LeaseReconciler] Stopping lease reconciliation")
			return
		case <-ticker.C:
			if err := vm.reconcileAllLeases(ctx); err != nil {
				logger.Warn("[LeaseReconciler] Reconciliation failed: %v", err)
			}
		}
	}
}

func (vm *VPSManager) reconcileAllLeases(ctx context.Context) error {
	logger.Debug("[LeaseReconciler] Starting lease reconciliation cycle")
	
	// Get all non-deleted VPSes with a node assigned
	var vpsList []database.VPSInstance
	if err := database.DB.WithContext(ctx).
		Where("deleted_at IS NULL AND node_id IS NOT NULL AND instance_id IS NOT NULL").
		Find(&vpsList).Error; err != nil {
		return fmt.Errorf("failed to query VPS instances: %w", err)
	}

	logger.Debug("[LeaseReconciler] Found %d VPS instances to check", len(vpsList))

	reconciledCount := 0
	skippedCount := 0
	errorCount := 0

	for _, vps := range vpsList {
		// Check if this VPS already has a non-public lease in the database
		var existingLease database.DHCPLease
		err := database.DB.WithContext(ctx).
			Where("vps_id = ? AND is_public = ?", vps.ID, false).
			First(&existingLease).Error

		if err == nil {
			// Lease exists, skip
			skippedCount++
			continue
		}

		// No lease found - try to get MAC from Proxmox and register
		if vps.NodeID == nil || *vps.NodeID == "" {
			logger.Debug("[LeaseReconciler] VPS %s has no node_id, skipping", vps.ID)
			skippedCount++
			continue
		}

		if vps.InstanceID == nil || *vps.InstanceID == "" {
			logger.Debug("[LeaseReconciler] VPS %s has no instance_id, skipping", vps.ID)
			skippedCount++
			continue
		}

		nodeName := *vps.NodeID
		vmID := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmID)
		if vmID == 0 {
			logger.Warn("[LeaseReconciler] VPS %s has invalid instance_id: %s", vps.ID, *vps.InstanceID)
			errorCount++
			continue
		}

		// Get Proxmox client for this node
		proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
		if err != nil {
			logger.Warn("[LeaseReconciler] Failed to get Proxmox client for node %s: %v", nodeName, err)
			errorCount++
			continue
		}

		// Get VM network config from Proxmox
		mac, _, err := vm.getVMNetworkInfo(ctx, proxmoxClient, nodeName, vmID)
		if err != nil {
			logger.Warn("[LeaseReconciler] Failed to get network info for VPS %s (VM %d on %s): %v", vps.ID, vmID, nodeName, err)
			errorCount++
			continue
		}

		if mac == "" {
			logger.Debug("[LeaseReconciler] VPS %s (VM %d) has no MAC address yet, skipping", vps.ID, vmID)
			skippedCount++
			continue
		}

		// Register the lease via gateway
		gatewayClient, err := vm.GetGatewayClientForNode(nodeName)
		if err != nil {
			logger.Warn("[LeaseReconciler] Failed to get gateway client for node %s: %v", nodeName, err)
			errorCount++
			continue
		}

		// Allocate IP (gateway will register the lease in database)
		allocCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		allocResp, err := gatewayClient.AllocateIP(allocCtx, vps.ID, vps.OrganizationID, mac)
		cancel()

		if err != nil {
			logger.Warn("[LeaseReconciler] Failed to allocate IP for VPS %s (MAC: %s): %v", vps.ID, mac, err)
			errorCount++
			continue
		}

		logger.Info("[LeaseReconciler] Successfully reconciled lease for VPS %s (MAC: %s, IP: %s)", vps.ID, mac, allocResp.IpAddress)
		reconciledCount++
	}

	logger.Info("[LeaseReconciler] Reconciliation complete: %d reconciled, %d skipped, %d errors out of %d total VPSes",
		reconciledCount, skippedCount, errorCount, len(vpsList))

	return nil
}

// getVMNetworkInfo retrieves the MAC address and current IP from a Proxmox VM
func (vm *VPSManager) getVMNetworkInfo(ctx context.Context, client *ProxmoxClient, nodeName string, vmID int) (mac string, ip string, err error) {
	// Get VM config to find network interface MAC
	config, err := client.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get VM config: %w", err)
	}

	// Look for net0 interface (primary network)
	if net0, ok := config["net0"].(string); ok && net0 != "" {
		// Format: virtio=XX:XX:XX:XX:XX:XX,bridge=vmbr0
		parts := strings.Split(net0, ",")
		for _, part := range parts {
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 && (kv[0] == "virtio" || kv[0] == "e1000" || kv[0] == "rtl8139") {
					mac = strings.ToLower(strings.TrimSpace(kv[1]))
					break
				}
			}
		}
	}

	if mac == "" {
		return "", "", fmt.Errorf("no network interface found in VM config")
	}

	// Try to get current IP from QEMU guest agent (if available)
	// This is optional - we can reconcile with just MAC
	guestNetworks, guestErr := client.GetVMNetworkInterfaces(ctx, nodeName, vmID)
	if guestErr == nil {
		for _, iface := range guestNetworks {
			// Look for interface with matching MAC
			if strings.EqualFold(iface.MACAddress, mac) {
				// Prefer IPv4 addresses
				for _, addr := range iface.IPAddresses {
					if !strings.Contains(addr, ":") { // Simple IPv4 check
						ip = addr
						break
					}
				}
				break
			}
		}
	}

	return mac, ip, nil
}
