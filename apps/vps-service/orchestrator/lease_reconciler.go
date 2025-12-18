package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
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
		if vps.InstanceID == nil || *vps.InstanceID == "" {
			logger.Debug("[LeaseReconciler] VPS %s has no instance_id, skipping", vps.ID)
			skippedCount++
			continue
		}

		vmID := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmID)
		if vmID == 0 {
			logger.Warn("[LeaseReconciler] VPS %s has invalid instance_id: %s", vps.ID, *vps.InstanceID)
			errorCount++
			continue
		}

		// Determine node - discover if missing
		nodeName := ""
		if vps.NodeID != nil && *vps.NodeID != "" {
			nodeName = *vps.NodeID
		} else {
			// NodeID is missing - try to discover it
			allNodes, err := GetAllProxmoxNodeNames()
			if err != nil {
				logger.Warn("[LeaseReconciler] VPS %s has no NodeID and cannot discover: %v", vps.ID, err)
				errorCount++
				continue
			}

			// Try each node to find where the VM is running
			var discoveryErr error
			for _, discoveryNode := range allNodes {
				discoveryClient, err := vm.GetProxmoxClientForNode(discoveryNode)
				if err != nil {
					discoveryErr = err
					continue
				}
				foundNode, err := discoveryClient.FindVMNode(ctx, vmID)
				if err == nil {
					nodeName = foundNode
					// Update VPS record with discovered NodeID
					vps.NodeID = &nodeName
					if err := database.DB.Model(&vps).Update("node_id", nodeName).Error; err != nil {
						logger.Warn("[LeaseReconciler] Failed to update NodeID for VPS %s: %v", vps.ID, err)
					} else {
						logger.Info("[LeaseReconciler] Discovered and updated NodeID for VPS %s: %s", vps.ID, nodeName)
					}
					break
				}
				discoveryErr = err
			}

			if nodeName == "" {
				logger.Warn("[LeaseReconciler] Failed to discover node for VPS %s (VM %d): %v", vps.ID, vmID, discoveryErr)
				errorCount++
				continue
			}
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

		// Register the lease via gateway bidirectional stream
		bidiClient := vm.GetBidiGatewayClient()
		if bidiClient == nil {
			logger.Error("[LeaseReconciler] Bidirectional gateway client not available for VPS %s", vps.ID)
			errorCount++
			continue
		}
		
		// Use bidirectional stream
		type gatewayClient interface {
			AllocateIP(ctx context.Context, nodeName, vpsID, organizationID, macAddress string) (*vpsgatewayv1.AllocateIPResponse, error)
		}
		gc := bidiClient.(gatewayClient)
		
		allocCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		allocResp, err := gc.AllocateIP(allocCtx, nodeName, vps.ID, vps.OrganizationID, mac)
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

// GetAllocationsForGateway queries the database and returns all DHCP leases for a specific gateway node
// This is used for syncing allocations to gateways on startup and periodically
func (vm *VPSManager) GetAllocationsForGateway(ctx context.Context, nodeName string) ([]*vpsgatewayv1.DesiredAllocation, error) {
	logger.Debug("[LeaseReconciler] Querying allocations for gateway node: %s", nodeName)
	
	var leases []database.DHCPLease
	if err := database.DB.WithContext(ctx).
		Where("gateway_node = ?", nodeName).
		Find(&leases).Error; err != nil {
		return nil, fmt.Errorf("failed to query DHCP leases for node %s: %w", nodeName, err)
	}

	allocations := make([]*vpsgatewayv1.DesiredAllocation, 0, len(leases))
	for _, lease := range leases {
		// Get organization_id from the VPS record
		var vps database.VPSInstance
		if err := database.DB.WithContext(ctx).
			Select("organization_id").
			Where("id = ?", lease.VPSID).
			First(&vps).Error; err != nil {
			logger.Warn("[LeaseReconciler] Failed to get organization for VPS %s: %v", lease.VPSID, err)
			continue
		}

		allocations = append(allocations, &vpsgatewayv1.DesiredAllocation{
			VpsId:          lease.VPSID,
			OrganizationId: vps.OrganizationID,
			IpAddress:      lease.IPAddress,
			MacAddress:     lease.MACAddress,
			IsPublic:       lease.IsPublic,
		})
	}

	logger.Info("[LeaseReconciler] Found %d allocations for gateway node %s", len(allocations), nodeName)
	return allocations, nil
}
