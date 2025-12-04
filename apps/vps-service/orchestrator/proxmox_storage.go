package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Storage and node management operations

func (pc *ProxmoxClient) getNextVMID(ctx context.Context) (int, error) {
	// Check if VM ID start range is configured
	vmIDStartEnv := os.Getenv("PROXMOX_VM_ID_START")
	if vmIDStartEnv != "" {
		var vmIDStart int
		if _, err := fmt.Sscanf(vmIDStartEnv, "%d", &vmIDStart); err != nil {
			return 0, fmt.Errorf("invalid PROXMOX_VM_ID_START value: %s (must be a number)", vmIDStartEnv)
		}

		// Get all existing VM IDs to find the next available one starting from vmIDStart
		vmIDs, err := pc.getAllVMIDs(ctx)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to get existing VM IDs, falling back to Proxmox auto-generated ID: %v", err)
			// Fall through to Proxmox auto-generated ID
		} else {
			// Find next available ID starting from vmIDStart
			nextID := vmIDStart
			for {
				// Check if this ID is already in use
				idInUse := false
				for _, existingID := range vmIDs {
					if existingID == nextID {
						idInUse = true
						break
					}
				}
				if !idInUse {
					logger.Info("[ProxmoxClient] Using VM ID %d (starting from configured range %d)", nextID, vmIDStart)
					return nextID, nil
				}
				nextID++
				// Safety limit: don't go beyond 999999 (Proxmox max is typically 999999)
				if nextID > 999999 {
					return 0, fmt.Errorf("no available VM ID found in range starting from %d (reached limit 999999)", vmIDStart)
				}
			}
		}
	}

	// Fall back to Proxmox's auto-generated next ID
	resp, err := pc.apiRequest(ctx, "GET", "/cluster/nextid", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get next VM ID: %w", err)
	}
	defer resp.Body.Close()

	var nextIDResp struct {
		Data string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&nextIDResp); err != nil {
		return 0, fmt.Errorf("failed to decode next ID response: %w", err)
	}

	var vmID int
	if _, err := fmt.Sscanf(nextIDResp.Data, "%d", &vmID); err != nil {
		return 0, fmt.Errorf("failed to parse VM ID: %w", err)
	}

	return vmID, nil
}

func (pc *ProxmoxClient) getAllVMIDs(ctx context.Context) ([]int, error) {
	nodes, err := pc.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var allVMIDs []int
	vmIDMap := make(map[int]bool) // Use map to avoid duplicates

	for _, nodeName := range nodes {
		resp, err := pc.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu", nodeName), nil)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to list VMs on node %s: %v", nodeName, err)
			continue
		}
		defer resp.Body.Close()

		var vmsResp struct {
			Data []struct {
				Vmid int `json:"vmid"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&vmsResp); err != nil {
			logger.Warn("[ProxmoxClient] Failed to decode VMs on node %s: %v", nodeName, err)
			continue
		}

		for _, vm := range vmsResp.Data {
			if !vmIDMap[vm.Vmid] {
				allVMIDs = append(allVMIDs, vm.Vmid)
				vmIDMap[vm.Vmid] = true
			}
		}
	}

	return allVMIDs, nil
}

// ProxmoxVMInfo represents a VM from Proxmox with its description
type ProxmoxVMInfo struct {
	VMID        int
	NodeName    string
	Description string
	Name        string
	Status      string
}

// ListAllVMsWithDescriptions lists all VMs from all Proxmox nodes with their descriptions
func (pc *ProxmoxClient) ListAllVMsWithDescriptions(ctx context.Context) ([]ProxmoxVMInfo, error) {
	nodes, err := pc.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var allVMs []ProxmoxVMInfo
	vmMap := make(map[int]bool) // Use map to avoid duplicates

	for _, nodeName := range nodes {
		resp, err := pc.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu", nodeName), nil)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to list VMs on node %s: %v", nodeName, err)
			continue
		}
		defer resp.Body.Close()

		var vmsResp struct {
			Data []struct {
				Vmid       int    `json:"vmid"`
				Name       string `json:"name"`
				Status     string `json:"status"`
				Description string `json:"description,omitempty"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&vmsResp); err != nil {
			logger.Warn("[ProxmoxClient] Failed to decode VMs on node %s: %v", nodeName, err)
			continue
		}

		for _, vm := range vmsResp.Data {
			if !vmMap[vm.Vmid] {
				// If description is not in the list response, fetch it from config
				description := vm.Description
				if description == "" {
					vmConfig, err := pc.GetVMConfig(ctx, nodeName, vm.Vmid)
					if err == nil {
						if desc, ok := vmConfig["description"].(string); ok {
							description = desc
						}
					}
				}

				allVMs = append(allVMs, ProxmoxVMInfo{
					VMID:        vm.Vmid,
					NodeName:    nodeName,
					Description: description,
					Name:        vm.Name,
					Status:      vm.Status,
				})
				vmMap[vm.Vmid] = true
			}
		}
	}

	return allVMs, nil
}

func (pc *ProxmoxClient) FindVMNode(ctx context.Context, vmID int) (string, error) {
	nodes, err := pc.ListNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	// Check each node to find where the VM is located
	for _, nodeName := range nodes {
		resp, err := pc.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu", nodeName), nil)
		if err != nil {
			logger.Warn("[ProxmoxClient] Failed to list VMs on node %s: %v", nodeName, err)
			continue
		}

		var vmsResp struct {
			Data []struct {
				Vmid int `json:"vmid"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&vmsResp); err != nil {
			resp.Body.Close()
			// EOF errors often indicate the node is unavailable or not responding properly
			// Log as debug for EOF (common with unavailable nodes) and warn for other errors
			if err == io.EOF || strings.Contains(err.Error(), "EOF") {
				logger.Debug("[ProxmoxClient] Node %s returned empty response (node may be unavailable): %v", nodeName, err)
			} else {
				logger.Warn("[ProxmoxClient] Failed to decode VMs on node %s: %v", nodeName, err)
			}
			continue
		}
		resp.Body.Close()

		// Check if this node has the VM
		for _, vm := range vmsResp.Data {
			if vm.Vmid == vmID {
				return nodeName, nil
			}
		}
	}

	return "", fmt.Errorf("VM %d not found on any node", vmID)
}

func (pc *ProxmoxClient) ListNodes(ctx context.Context) ([]string, error) {
	resp, err := pc.apiRequest(ctx, "GET", "/nodes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer resp.Body.Close()

	var nodesResp struct {
		Data []struct {
			Node string `json:"node"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&nodesResp); err != nil {
		return nil, fmt.Errorf("failed to decode nodes response: %w", err)
	}

	nodes := make([]string, len(nodesResp.Data))
	for i, n := range nodesResp.Data {
		nodes[i] = n.Node
	}

	return nodes, nil
}

func (pc *ProxmoxClient) getStorageInfo(ctx context.Context, nodeName string, storageName string) (map[string]interface{}, error) {
	// Use listStorages and find the matching storage, as the individual storage endpoint
	// may return different formats depending on Proxmox version
	endpoint := fmt.Sprintf("/nodes/%s/storage", nodeName)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get storage info: %s (status: %d)", string(body), resp.StatusCode)
	}

	var storagesResp struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&storagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage response: %w", err)
	}

	// Find the matching storage
	for _, storage := range storagesResp.Data {
		if storageNameVal, ok := storage["storage"].(string); ok && storageNameVal == storageName {
			return storage, nil
		}
	}

	return nil, fmt.Errorf("storage '%s' not found on node '%s'", storageName, nodeName)
}

func (pc *ProxmoxClient) listStorages(ctx context.Context, nodeName string) ([]string, error) {
	endpoint := fmt.Sprintf("/nodes/%s/storage", nodeName)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage pools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list storage pools: %s (status: %d)", string(body), resp.StatusCode)
	}

	var storagesResp struct {
		Data []struct {
			Storage string `json:"storage"`
			Type    string `json:"type"`
			Content string `json:"content"` // e.g., "images,iso,vztmpl"
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&storagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode storage response: %w", err)
	}

	// Filter storages that support VM disk images (content includes "images")
	storages := make([]string, 0)
	for _, s := range storagesResp.Data {
		if strings.Contains(s.Content, "images") {
			storages = append(storages, s.Storage)
		}
	}

	return storages, nil
}

