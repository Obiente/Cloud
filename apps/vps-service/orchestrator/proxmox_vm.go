package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// VM operations

// CreateVMResult holds the result of VM creation
type CreateVMResult struct {
	VMID     string
	Password string // Root password for the VM
	NodeName string // Node where the VM was created
}

// buildVPSDescription builds a comprehensive description for VPS notes in Proxmox
func buildVPSDescription(config *VPSConfig) string {
	var parts []string

	// Base information
	parts = append(parts, fmt.Sprintf("Managed by Obiente Cloud - VPS ID: %s, Display Name: %s", config.VPSID, config.Name))

	// Organization information
	if config.OrganizationID != "" {
		orgInfo := fmt.Sprintf("Org ID: %s", config.OrganizationID)
		if config.OrganizationName != nil && *config.OrganizationName != "" {
			orgInfo += fmt.Sprintf(", Org Name: %s", *config.OrganizationName)
		}
		parts = append(parts, orgInfo)
	}

	// Creator information
	if config.CreatedBy != "" {
		creatorInfo := fmt.Sprintf("Creator ID: %s", config.CreatedBy)
		if config.CreatorName != nil && *config.CreatorName != "" {
			creatorInfo += fmt.Sprintf(", Creator Name: %s", *config.CreatorName)
		}
		parts = append(parts, creatorInfo)
	}

	// Owner information (always show if available, even if same as creator)
	if config.OwnerID != nil && *config.OwnerID != "" {
		ownerInfo := fmt.Sprintf("Owner ID: %s", *config.OwnerID)
		if config.OwnerName != nil && *config.OwnerName != "" {
			ownerInfo += fmt.Sprintf(", Owner Name: %s", *config.OwnerName)
		}
		parts = append(parts, ownerInfo)
	}

	return strings.Join(parts, " | ")
}

// getMapKeys extracts keys from a map
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// parseRegionNodeMapping parses the PROXMOX_REGION_NODES environment variable
// Format: "region1:node1;region2:node2" or "region1:node1,node2" (multiple nodes per region)
// Returns a map of region -> preferred node name
func parseRegionNodeMapping() map[string]string {
	mapping := make(map[string]string)
	envValue := os.Getenv("PROXMOX_REGION_NODES")
	if envValue == "" {
		return mapping
	}

	// Parse semicolon-separated region mappings
	regionStrings := strings.Split(envValue, ";")
	for _, regionStr := range regionStrings {
		regionStr = strings.TrimSpace(regionStr)
		if regionStr == "" {
			continue
		}

		// Parse "regionID:nodeName" format
		if strings.Contains(regionStr, ":") {
			parts := strings.SplitN(regionStr, ":", 2)
			if len(parts) == 2 {
				regionID := strings.TrimSpace(parts[0])
				nodeName := strings.TrimSpace(parts[1])
				// If multiple nodes are specified (comma-separated), use the first one
				if strings.Contains(nodeName, ",") {
					nodeName = strings.Split(nodeName, ",")[0]
					nodeName = strings.TrimSpace(nodeName)
				}
				if regionID != "" && nodeName != "" {
					mapping[regionID] = nodeName
				}
			}
		}
	}

	return mapping
}

func (pc *ProxmoxClient) CreateVM(ctx context.Context, config *VPSConfig, allowInterVM bool) (*CreateVMResult, error) {
	// Declare rootPassword at function scope so it can be returned
	var rootPassword string

	// Get next available VM ID
	vmID, err := pc.getNextVMID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next VM ID: %w", err)
	}

	// List all available nodes
	nodes, err := pc.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Proxmox nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no Proxmox nodes available")
	}

	// Select node based on region mapping if configured, otherwise use first available
	nodeName := nodes[0]
	if config.Region != "" {
		regionNodeMap := parseRegionNodeMapping()
		if mappedNode, ok := regionNodeMap[config.Region]; ok {
			// Verify the mapped node exists in the cluster
			nodeExists := false
			for _, node := range nodes {
				if node == mappedNode {
					nodeExists = true
					nodeName = mappedNode
					logger.Info("[ProxmoxClient] Using mapped node %s for region %s", nodeName, config.Region)
					break
				}
			}
			if !nodeExists {
				logger.Warn("[ProxmoxClient] Mapped node %s for region %s not found in cluster, using first available node %s", mappedNode, config.Region, nodeName)
			}
		}
	}

	// Get storage pool for VM disks (default to local-lvm)
	storage := "local-lvm"
	if storagePool := os.Getenv("PROXMOX_STORAGE_POOL"); storagePool != "" {
		storage = storagePool
	}

	// Get storage pool for cloud-init snippets (defaults to VM disk storage, but can be separate)
	// Snippets require directory-type storage (dir, nfs, cifs), not block storage (lvm, zfs)
	snippetStorage := storage
	if snippetStoragePool := os.Getenv("PROXMOX_SNIPPET_STORAGE"); snippetStoragePool != "" {
		snippetStorage = snippetStoragePool
		logger.Info("[ProxmoxClient] Using storage '%s' for VM disks and '%s' for cloud-init snippets", storage, snippetStorage)
	}

	// Validate storage pool exists and get storage type
	availableStorages, err := pc.listStorages(ctx, nodeName)
	storageType := "unknown"
	if err != nil {
		logger.Warn("[ProxmoxClient] Failed to list storage pools, continuing anyway: %v", err)
	} else {
		storageExists := false
		for _, s := range availableStorages {
			if s == storage {
				storageExists = true
				break
			}
		}
		if !storageExists {
			return nil, fmt.Errorf("storage pool '%s' does not exist on node '%s'. Available storage pools: %v. Please set PROXMOX_STORAGE_POOL to one of the available pools or create the storage pool in Proxmox", storage, nodeName, availableStorages)
		}

		// Get storage type to determine disk format
		storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
		if err == nil && storageInfo != nil {
			if st, ok := storageInfo["type"].(string); ok {
				storageType = st
				logger.Info("[ProxmoxClient] Storage pool '%s' type: %s", storage, storageType)
			} else {
				logger.Warn("[ProxmoxClient] Could not determine storage type for '%s', defaulting to LVM format", storage)
			}
		} else {
			logger.Warn("[ProxmoxClient] Failed to get storage info for '%s': %v, defaulting to LVM format", storage, err)
		}
	}

	// Determine disk format based on storage type
	// For directory storage, we need to use a different format (no volume name)
	// For LVM/ZFS storage, we can specify the volume name
	diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
	var scsi0Config string
	if storageType == "dir" || storageType == "directory" {
		// Directory storage: format is storage:size=XXG (Proxmox auto-generates the filename)
		// Cannot specify volume name for directory storage - it will be auto-generated
		scsi0Config = fmt.Sprintf("%s:size=%dG", storage, diskSizeGB)
	} else {
		// LVM/ZFS storage: format is storage:vm-XXX-disk-0,size=XXG
		scsi0Config = fmt.Sprintf("%s:vm-%d-disk-0,size=%dG", storage, vmID, diskSizeGB)
	}

	// Create VM configuration
	// SECURITY: Add annotation to mark VM as managed by Obiente Cloud
	// Use VPS ID as VM name to ensure uniqueness (Proxmox doesn't allow duplicate VM names)
	vmConfig := map[string]interface{}{
		"vmid":   vmID,
		"name":   config.VPSID, // Use VPS ID (deployment ID) as VM name for uniqueness
		"cores":  config.CPUCores,
		"memory": config.MemoryBytes / (1024 * 1024), // Convert bytes to MB
		"ostype": "l26",                              // Linux 2.6+ kernel
		"onboot": 1,
		"agent":  1, // Enable QEMU guest agent
		"scsi0":  scsi0Config,
		// Enable serial console for boot output and terminal access
		"serial0": "socket",
		// SECURITY: Mark VM as managed by Obiente Cloud
		"description": buildVPSDescription(config),
	}

	// Configure network interface
	// If gateway is configured, use the SDN bridge (OCvpsnet by default)
	// Otherwise, use the default bridge (vmbr0)
	bridge := "vmbr0"
	if os.Getenv("VPS_GATEWAY_URL") != "" || os.Getenv("VPS_GATEWAY_API_SECRET") != "" {
		// Gateway manages DHCP on SDN bridge
		gatewayBridge := os.Getenv("VPS_GATEWAY_BRIDGE")
		if gatewayBridge == "" {
			gatewayBridge = "OCvpsnet" // Default SDN bridge name
		}
		bridge = gatewayBridge
		logger.Info("[ProxmoxClient] Using gateway bridge %s for VM network (gateway manages DHCP)", bridge)
	}

	// Configure network interface with optional VLAN support
	// SECURITY: Use VLAN tags for network isolation when configured
	netConfig := fmt.Sprintf("virtio,bridge=%s,firewall=1", bridge)
	if vlanID := os.Getenv("PROXMOX_VLAN_ID"); vlanID != "" {
		// Add VLAN tag for network isolation
		netConfig = fmt.Sprintf("virtio,bridge=%s,tag=%s,firewall=1", bridge, vlanID)
		logger.Info("[ProxmoxClient] Configuring VM network with VLAN tag: %s on bridge: %s", vlanID, bridge)
	}
	vmConfig["net0"] = netConfig

	// Use cloud-init for modern Linux distributions (Ubuntu 22.04+, Debian 12+)
	// For older images, fall back to ISO installation
	useCloudInit := false
	imageTemplate := ""
	var templateVMID int // Store template VM ID for later use
	// Track if we created a disk during the clone process (needed for config update)
	var diskCreated bool
	var createdDiskValue string

	switch config.Image {
	case 1: // UBUNTU_22_04
		imageTemplate = "ubuntu-22.04-standard"
		useCloudInit = true
	case 2: // UBUNTU_24_04
		imageTemplate = "ubuntu-24.04-standard"
		useCloudInit = true
	case 3: // DEBIAN_12
		imageTemplate = "debian-12-standard"
		useCloudInit = true
	case 4: // DEBIAN_13
		imageTemplate = "debian-13-standard"
		useCloudInit = true
	case 5: // ROCKY_LINUX_9
		imageTemplate = "rockylinux-9-standard"
		useCloudInit = true
	case 6: // ALMA_LINUX_9
		imageTemplate = "almalinux-9-standard"
		useCloudInit = true
	case 99: // CUSTOM
		if config.ImageID != nil {
			imageTemplate = *config.ImageID
			useCloudInit = true // Assume custom images support cloud-init
		}
	}

	if useCloudInit && imageTemplate != "" {
		// Find template
		var err error
		templateVMID, err = pc.findTemplate(ctx, nodeName, imageTemplate)
		if err != nil {
			logger.Warn("[ProxmoxClient] Template %s not found, falling back to ISO installation: %v", imageTemplate, err)
			useCloudInit = false
		} else {
			// Get template config first to determine template storage
			templateConfig, err := pc.GetVMConfig(ctx, nodeName, templateVMID)
			if err != nil {
				logger.Warn("[ProxmoxClient] Failed to get template config, falling back to ISO: %v", err)
				useCloudInit = false
			} else {
				// Find template disk to determine template storage
				var templateDiskValue string
				var templateStorage string
				var templateStorageType string
				for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
					if disk, ok := templateConfig[diskKey].(string); ok && disk != "" {
						templateDiskValue = disk
						// Extract storage from disk value (format: storage:path or storage:vmID/path)
						parts := strings.Split(disk, ":")
						if len(parts) >= 1 {
							templateStorage = parts[0]
							// Get template storage type
							if storageInfo, err := pc.getStorageInfo(ctx, nodeName, templateStorage); err == nil && storageInfo != nil {
								if st, ok := storageInfo["type"].(string); ok {
									templateStorageType = st
								}
							}
						}
						break
					}
				}

				// Determine if we need a full clone or can use a linked clone
				// Linked clones only work safely when:
				// 1. Template storage matches desired storage (linked clones inherit template storage)
				// 2. Template storage type supports linked clones (directory storage only, not lvmthin/lvm/zfs)
				// For cross-storage clones (e.g. template on 'local', VM on 'local-lvmthin'),
				// let Proxmox handle a full clone directly to the target storage instead of
				// doing a linked clone + manual disk move. This ensures the disk contents and
				// partition table are copied correctly and avoids empty/invalid disks.
				useFullClone := false
				if templateStorage != "" && templateStorage != storage {
					// Template storage doesn't match desired storage - need full clone
					useFullClone = true
					logger.Info("[ProxmoxClient] Template storage '%s' differs from desired storage '%s', using full clone", templateStorage, storage)
				} else if templateStorageType == "lvmthin" || templateStorageType == "lvm" || templateStorageType == "zfs" {
					// Template storage type doesn't support linked clones - need full clone
					useFullClone = true
					logger.Info("[ProxmoxClient] Template storage type '%s' does not support linked clones, using full clone", templateStorageType)
				}

				// Clone from template
				// Proxmox API expects form-encoded data for clone operations
				// Note: For linked clones (full=0), the storage parameter is not allowed
				// The disk will be cloned to the same storage as the template
				// For full clones (full=1), you can specify storage
				cloneEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/clone", nodeName, templateVMID)
				cloneFormData := url.Values{}
				cloneFormData.Set("newid", fmt.Sprintf("%d", vmID))
				cloneFormData.Set("name", config.VPSID) // Use VPS ID as VM name for uniqueness
				cloneFormData.Set("target", nodeName)
				if useFullClone {
					cloneFormData.Set("full", "1") // Full clone (allows storage specification)
					cloneFormData.Set("storage", storage) // Specify target storage for full clone
					logger.Info("[ProxmoxClient] Cloning template %s (VMID %d) to VM %d (full clone to storage %s)", imageTemplate, templateVMID, vmID, storage)
				} else {
					cloneFormData.Set("full", "0") // Linked clone (faster, inherits template storage)
					logger.Info("[ProxmoxClient] Cloning template %s (VMID %d) to VM %d (linked clone on storage %s)", imageTemplate, templateVMID, vmID, templateStorage)
				}

				resp, err := pc.apiRequestForm(ctx, "POST", cloneEndpoint, cloneFormData)
				if err != nil {
					logger.Warn("[ProxmoxClient] Failed to clone template, falling back to ISO: %v", err)
					useCloudInit = false
				} else {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						logger.Info("[ProxmoxClient] Cloned template %s to VM %d", imageTemplate, vmID)

						// Wait a moment for the clone to complete and disk to be available
						time.Sleep(2 * time.Second)

						// Template boot configuration is handled by the setup-proxmox-templates.sh script
						// which fixes fstab and GRUB to use device names instead of UUID/PARTUUID

						// Reuse template config we already retrieved (no need to fetch again)
						// templateDiskValue and templateStorage were already determined before cloning
						var templateDiskKey string
						// Find the disk key used by the template (we already have templateConfig from before cloning)
						for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
							if disk, ok := templateConfig[diskKey].(string); ok && disk != "" {
								templateDiskKey = diskKey
								templateDiskValue = disk
								logger.Info("[ProxmoxClient] Template %s (VMID %d) uses disk %s: %s", imageTemplate, templateVMID, diskKey, disk)
								break
							}
						}
						if templateDiskKey == "" {
							// Template has no disk in config - check if there are any disk volumes in storage
							logger.Warn("[ProxmoxClient] Template %s (VMID %d) does not have disk in config. Checking storage for disk volumes...", imageTemplate, templateVMID)
							// Check storage for template disk volumes - use template storage if we found it, otherwise use desired storage
							storageToSearch := storage
							if templateStorage != "" {
								storageToSearch = templateStorage
							}
							storageContentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageToSearch)
							contentResp, contentErr := pc.apiRequest(ctx, "GET", storageContentEndpoint, nil)
							if contentErr == nil && contentResp != nil && contentResp.StatusCode == http.StatusOK {
								defer contentResp.Body.Close()
								var contentData struct {
									Data []struct {
										VolID   string `json:"volid"`
										VMID    *int   `json:"vmid"`
										Content string `json:"content"`
									} `json:"data"`
								}
								if err := json.NewDecoder(contentResp.Body).Decode(&contentData); err == nil {
									for _, vol := range contentData.Data {
										if vol.VMID != nil && *vol.VMID == templateVMID && vol.Content == "images" {
											// Found a disk volume for the template (could be unused disk)
											volParts := strings.Split(vol.VolID, ":")
											if len(volParts) >= 2 {
												volName := volParts[1]
												// Determine disk key from volume name
												if strings.Contains(volName, "scsi") {
													templateDiskKey = "scsi0"
												} else if strings.Contains(volName, "virtio") {
													templateDiskKey = "virtio0"
												} else if strings.Contains(volName, "sata") {
													templateDiskKey = "sata0"
												} else if strings.Contains(volName, "ide") {
													templateDiskKey = "ide0"
												} else {
													templateDiskKey = "scsi0" // Default to scsi0 for unused disks
												}
												templateDiskValue = vol.VolID
												logger.Info("[ProxmoxClient] Found template disk volume %s in storage (may be unused disk), using disk key %s", vol.VolID, templateDiskKey)
												break
											}
										}
									}
								}
							}

							if templateDiskKey == "" {
								logger.Error("[ProxmoxClient] CRITICAL: Template %s (VMID %d) does not have any disk configured! Template config keys: %v", imageTemplate, templateVMID, getMapKeys(templateConfig))
								return nil, fmt.Errorf("template %s (VMID %d) does not have a disk configured - cannot clone VM without disk. Please configure a disk for the template first", imageTemplate, templateVMID)
							}
						}

						// Wait a bit longer for clone to fully complete
						time.Sleep(3 * time.Second)

						// Verify disk exists in cloned VM and determine which disk key to use
						vmConfigCheck, err := pc.GetVMConfig(ctx, nodeName, vmID)
						var actualDiskKey string
						if err != nil {
							logger.Warn("[ProxmoxClient] Failed to get VM config after clone: %v", err)
						} else {
						// Check for disk - prefer the same type as template, but check all types
						diskKeysToCheck := []string{}
						if templateDiskKey != "" {
							diskKeysToCheck = append(diskKeysToCheck, templateDiskKey)
						}
						// Add other disk types if template disk key wasn't found or is different
						for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
							found := false
							for _, checkKey := range diskKeysToCheck {
								if checkKey == diskKey {
									found = true
									break
								}
							}
							if !found {
								diskKeysToCheck = append(diskKeysToCheck, diskKey)
							}
						}

						for _, diskKey := range diskKeysToCheck {
							if disk, ok := vmConfigCheck[diskKey].(string); ok && disk != "" {
								actualDiskKey = diskKey
								logger.Info("[ProxmoxClient] VM %d has disk %s: %s", vmID, diskKey, disk)
								break
							}
						}

						if actualDiskKey == "" {
							logger.Warn("[ProxmoxClient] Cloned VM %d does not have a boot disk configured", vmID)
							logger.Info("[ProxmoxClient] Template %s (VMID %d) disk config: %s=%s", imageTemplate, templateVMID, templateDiskKey, templateDiskValue)

							// Check if template disk is a cloud-init disk (not a boot disk)
							isCloudInitDisk := false
							if templateDiskValue != "" {
								isCloudInitDisk = strings.Contains(templateDiskValue, "cloudinit")
							}

							// If template only has cloud-init disk or no boot disk, create a new boot disk
							// WARNING: Creating an empty disk - it will not be bootable without importing a cloud image
							// The template should be recreated with a proper boot disk (see vps-provisioning.md)
							if isCloudInitDisk || templateDiskKey == "" {
								logger.Warn("[ProxmoxClient] Template has no boot disk (only cloud-init), creating new boot disk for VM %d. NOTE: This disk will be empty and may not boot. The template should be recreated with a proper boot disk.", vmID)

								// Create a new boot disk using Proxmox API
								// Format for directory storage: storage:size=XXG,format=qcow2 (Proxmox auto-generates filename)
								// Format for LVM/ZFS: storage:vm-XXX-disk-0,size=XXG
								diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
								diskSizeStr := fmt.Sprintf("%dG", diskSizeGB)

								// Determine storage type and format
								storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
								var diskValue string
								if err == nil && storageInfo != nil {
									storageType, ok := storageInfo["type"].(string)
									if ok {
										if storageType == "dir" || storageType == "directory" {
											// Directory storage: must include vmID subdirectory in path
											// Format: storage:vmID/vm-XXX-disk-0.qcow2,size=XXG,format=qcow2
											// Example: local:300/vm-300-disk-0.qcow2,size=10G,format=qcow2
											diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
										} else {
											// LVM/ZFS: storage:vm-XXX-disk-0,size=XXG
											diskValue = fmt.Sprintf("%s:vm-%d-disk-0,size=%s", storage, vmID, diskSizeStr)
										}
									} else {
										// Default to directory format with vmID subdirectory
										diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
									}
								} else {
									// Default to directory format with vmID subdirectory
									diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
								}

								// For directory storage, we need to create the disk volume first, then attach it
								// For other storage types, we can specify it directly in the config
								actualDiskKey = "scsi0"
								var diskResp *http.Response
								var diskErr error

								// If storage type detection failed, assume directory storage for "local" storage pool
								// (common default for directory storage)
								useDirectoryStorage := storageType == "dir" || storageType == "directory"
								if !useDirectoryStorage && (storage == "local" || err != nil) {
									// Default to directory storage if detection failed or storage is "local"
									useDirectoryStorage = true
									logger.Info("[ProxmoxClient] Assuming directory storage for '%s' (detection failed or default)", storage)
								}

								if useDirectoryStorage {
									// Create disk volume first using storage content API
									// Format: POST /nodes/{node}/storage/{storage}/content
									// Parameters: vmid, filename, size, format
									contentFormData := url.Values{}
									contentFormData.Set("vmid", fmt.Sprintf("%d", vmID))
									contentFormData.Set("filename", fmt.Sprintf("vm-%d-disk-0.qcow2", vmID))
									contentFormData.Set("size", diskSizeStr)
									contentFormData.Set("format", "qcow2")

									contentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage)
									logger.Info("[ProxmoxClient] Creating disk volume for VM %d via storage API: %s", vmID, contentEndpoint)
									contentResp, contentErr := pc.apiRequestForm(ctx, "POST", contentEndpoint, contentFormData)
									if contentErr == nil && contentResp != nil {
										if contentResp.StatusCode == http.StatusOK {
											contentResp.Body.Close()
											logger.Info("[ProxmoxClient] Successfully created disk volume for VM %d", vmID)
											// Now attach the disk to the VM config
											diskFormData := url.Values{}
											diskFormData.Set(actualDiskKey, diskValue)
											diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
										} else {
											body, _ := io.ReadAll(contentResp.Body)
											contentResp.Body.Close()
											logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: status %d, response: %s", vmID, contentResp.StatusCode, string(body))
											diskErr = fmt.Errorf("failed to create disk volume: status %d", contentResp.StatusCode)
										}
									} else {
										logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: %v", vmID, contentErr)
										diskErr = contentErr
									}
								} else {
									// For LVM/ZFS, we can set it directly
									diskFormData := url.Values{}
									diskFormData.Set(actualDiskKey, diskValue)
									logger.Info("[ProxmoxClient] Creating boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
									diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
								}
								if diskErr == nil && diskResp != nil && diskResp.StatusCode == http.StatusOK {
									diskResp.Body.Close()
									logger.Info("[ProxmoxClient] Successfully created boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
									diskCreated = true
									createdDiskValue = diskValue
								} else {
									var body []byte
									if diskResp != nil {
										body, _ = io.ReadAll(diskResp.Body)
									}
									logger.Error("[ProxmoxClient] Failed to create boot disk for VM %d: %v. Response: %s", vmID, diskErr, string(body))
									// Continue anyway - we'll try to create it again later
								}
							} else {
								// Template has a boot disk, but it wasn't cloned - try to find and attach it
								logger.Info("[ProxmoxClient] Template has boot disk but it wasn't cloned, searching for disk volume...")

								storageToSearch := storage
								if templateDiskValue != "" {
									parts := strings.Split(templateDiskValue, ":")
									if len(parts) >= 1 {
										storageToSearch = parts[0]
									}
								}

								// List all volumes in storage to find the one for this VM
								storageContentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageToSearch)
								contentResp, contentErr := pc.apiRequest(ctx, "GET", storageContentEndpoint, nil)
								var foundVolume string
								var foundDiskKey string
								if contentErr == nil && contentResp != nil && contentResp.StatusCode == http.StatusOK {
									defer contentResp.Body.Close()
									var contentData struct {
										Data []struct {
											VolID   string `json:"volid"`
											VMID    *int   `json:"vmid"`
											Format  string `json:"format"`
											Content string `json:"content"`
										} `json:"data"`
									}
									if err := json.NewDecoder(contentResp.Body).Decode(&contentData); err == nil {
										// Look for volumes that match this VM ID (cloned disk)
										for _, vol := range contentData.Data {
											if vol.Content == "images" && vol.VMID != nil && *vol.VMID == vmID {
												// Skip cloud-init disks
												if strings.Contains(vol.VolID, "cloudinit") {
													continue
												}
												// Found a disk volume for the cloned VM
												volParts := strings.Split(vol.VolID, ":")
												if len(volParts) >= 2 {
													volName := volParts[1]
													foundVolume = volName
													// Try to determine disk key from volume name
													if strings.Contains(volName, "scsi") {
														foundDiskKey = "scsi0"
													} else if strings.Contains(volName, "virtio") {
														foundDiskKey = "virtio0"
													} else if strings.Contains(volName, "sata") {
														foundDiskKey = "sata0"
													} else if strings.Contains(volName, "ide") {
														foundDiskKey = "ide0"
													} else {
														foundDiskKey = templateDiskKey
														if foundDiskKey == "" {
															foundDiskKey = "scsi0"
														}
													}
													logger.Info("[ProxmoxClient] Found cloned disk volume %s for VM %d, using disk key %s", foundVolume, vmID, foundDiskKey)
													break
												}
											}
										}
									}
								}

								// If we found a volume, add it to the VM config
								if foundVolume != "" && foundDiskKey != "" {
									newDiskValue := fmt.Sprintf("%s:%s", storageToSearch, foundVolume)
									diskFormData := url.Values{}
									diskFormData.Set(foundDiskKey, newDiskValue)
									diskResp, diskErr := pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
									if diskErr == nil && diskResp != nil && diskResp.StatusCode == http.StatusOK {
										diskResp.Body.Close()
										actualDiskKey = foundDiskKey
										logger.Info("[ProxmoxClient] Successfully attached disk %s to VM %d: %s", foundDiskKey, vmID, newDiskValue)
									} else {
										var body []byte
										if diskResp != nil {
											body, _ = io.ReadAll(diskResp.Body)
										}
										logger.Error("[ProxmoxClient] Failed to attach disk to VM %d: %v. Response: %s", vmID, diskErr, string(body))
									}
								} else {
									// No disk found, create a new one
									logger.Info("[ProxmoxClient] No cloned disk found, creating new boot disk for VM %d", vmID)
									diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
									diskSizeStr := fmt.Sprintf("%dG", diskSizeGB)

									// Determine storage type and format
									storageInfo, err := pc.getStorageInfo(ctx, nodeName, storageToSearch)
									var diskValue string
									var storageTypeForDisk string
									if err == nil && storageInfo != nil {
										if st, ok := storageInfo["type"].(string); ok {
											storageTypeForDisk = st
											if st == "dir" || st == "directory" {
												// Directory storage: must include vmID subdirectory in path
												// Format: storage:vmID/vm-XXX-disk-0.qcow2,size=XXG,format=qcow2
												diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storageToSearch, vmID, vmID, diskSizeStr)
											} else {
												// LVM/ZFS: storage:vm-XXX-disk-0,size=XXG
												diskValue = fmt.Sprintf("%s:vm-%d-disk-0,size=%s", storageToSearch, vmID, diskSizeStr)
											}
										} else {
											// Default to directory format with vmID subdirectory
											storageTypeForDisk = "dir"
											diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storageToSearch, vmID, vmID, diskSizeStr)
										}
									} else {
										// Default to directory format with vmID subdirectory
										storageTypeForDisk = "dir"
										diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storageToSearch, vmID, vmID, diskSizeStr)
									}

									actualDiskKey = "scsi0"
									var diskResp *http.Response
									var diskErr error

									// If storage type detection failed, assume directory storage for "local" storage pool
									useDirectoryStorage := storageTypeForDisk == "dir" || storageTypeForDisk == "directory"
									if !useDirectoryStorage && (storageToSearch == "local" || err != nil) {
										// Default to directory storage if detection failed or storage is "local"
										useDirectoryStorage = true
										logger.Info("[ProxmoxClient] Assuming directory storage for '%s' (detection failed or default)", storageToSearch)
									}

									if useDirectoryStorage {
										// Create disk volume first using storage content API
										contentFormData := url.Values{}
										contentFormData.Set("vmid", fmt.Sprintf("%d", vmID))
										contentFormData.Set("filename", fmt.Sprintf("vm-%d-disk-0.qcow2", vmID))
										contentFormData.Set("size", diskSizeStr)
										contentFormData.Set("format", "qcow2")

										contentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageToSearch)
										logger.Info("[ProxmoxClient] Creating disk volume for VM %d via storage API: %s", vmID, contentEndpoint)
										contentResp, contentErr := pc.apiRequestForm(ctx, "POST", contentEndpoint, contentFormData)
										if contentErr == nil && contentResp != nil {
											if contentResp.StatusCode == http.StatusOK {
												contentResp.Body.Close()
												logger.Info("[ProxmoxClient] Successfully created disk volume for VM %d", vmID)
												// Now attach the disk to the VM config
												diskFormData := url.Values{}
												diskFormData.Set(actualDiskKey, diskValue)
												diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
											} else {
												body, _ := io.ReadAll(contentResp.Body)
												contentResp.Body.Close()
												logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: status %d, response: %s", vmID, contentResp.StatusCode, string(body))
												diskErr = fmt.Errorf("failed to create disk volume: status %d", contentResp.StatusCode)
											}
										} else {
											logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: %v", vmID, contentErr)
											diskErr = contentErr
										}
									} else {
										// For LVM/ZFS, we can set it directly
										diskFormData := url.Values{}
										diskFormData.Set(actualDiskKey, diskValue)
										logger.Info("[ProxmoxClient] Creating boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
										diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
									}
									if diskErr == nil && diskResp != nil && diskResp.StatusCode == http.StatusOK {
										diskResp.Body.Close()
										logger.Info("[ProxmoxClient] Successfully created boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
										diskCreated = true
										createdDiskValue = diskValue
									} else {
										var body []byte
										if diskResp != nil {
											body, _ = io.ReadAll(diskResp.Body)
										}
										logger.Error("[ProxmoxClient] Failed to create boot disk for VM %d: %v. Response: %s", vmID, diskErr, string(body))
										// Try LVM/ZFS format as fallback (without format parameter)
										if strings.Contains(string(body), "format") || strings.Contains(string(body), "qcow2") {
											logger.Info("[ProxmoxClient] Retrying with LVM/ZFS format (no format parameter) for VM %d", vmID)
											diskValue = fmt.Sprintf("%s:vm-%d-disk-0,size=%s", storageToSearch, vmID, diskSizeStr)
											diskFormData := url.Values{}
											diskFormData.Set(actualDiskKey, diskValue)
											diskResp2, diskErr2 := pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
											if diskErr2 == nil && diskResp2 != nil && diskResp2.StatusCode == http.StatusOK {
												diskResp2.Body.Close()
												logger.Info("[ProxmoxClient] Successfully created boot disk %s for VM %d with LVM/ZFS format: %s", actualDiskKey, vmID, diskValue)
												diskCreated = true
												createdDiskValue = diskValue
											} else {
												var body2 []byte
												if diskResp2 != nil {
													body2, _ = io.ReadAll(diskResp2.Body)
												}
												logger.Error("[ProxmoxClient] Failed to create boot disk with LVM/ZFS format for VM %d: %v. Response: %s", vmID, diskErr2, string(body2))
											}
										}
									}
								}
							}
						}
					}

						// Resize disk after cloning to match the plan's disk size
						// For linked clones, the disk inherits the template size, so we need to resize it
						// If we just created a new disk, it should already be the correct size, but verify anyway
						if actualDiskKey != "" {
						vmConfigAfter, err := pc.GetVMConfig(ctx, nodeName, vmID)
						if err == nil {
							if disk, ok := vmConfigAfter[actualDiskKey].(string); ok && disk != "" {
								diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)

								// Check if we just created the disk (if so, it should already be the right size)
								// But we still verify and resize if needed, as the size might not match exactly
								justCreated := strings.Contains(disk, "size=")

								if justCreated {
									// Extract size from disk config to verify it matches
									// Format: "storage:vmID/vm-XXX-disk-0.qcow2,size=XXG,format=qcow2"
									if strings.Contains(disk, "size=") {
										sizePart := strings.Split(disk, "size=")
										if len(sizePart) > 1 {
											sizeStr := strings.Split(sizePart[1], ",")[0]
											sizeStr = strings.TrimSuffix(sizeStr, "G")
											if existingSize, parseErr := strconv.ParseInt(sizeStr, 10, 64); parseErr == nil {
												if existingSize == diskSizeGB {
													logger.Info("[ProxmoxClient] Disk %s was just created with correct size (%dGB), skipping resize", actualDiskKey, diskSizeGB)
													// Skip to next iteration - disk is already correct size
													goto skipResize
												} else {
													logger.Info("[ProxmoxClient] Disk %s was created with size %dGB but needs %dGB, resizing...", actualDiskKey, existingSize, diskSizeGB)
												}
											}
										}
									}
								}

								// Resize disk to match plan size
								// For linked clones, this will create a new disk with the correct size
								logger.Info("[ProxmoxClient] Resizing disk %s for VM %d to %dGB (plan size)", actualDiskKey, vmID, diskSizeGB)
								if err := pc.resizeDisk(ctx, nodeName, vmID, actualDiskKey, diskSizeGB); err != nil {
									logger.Error("[ProxmoxClient] Failed to resize disk %s for VM %d to %dGB: %v", actualDiskKey, vmID, diskSizeGB, err)
									// This is a critical error - the VM will have the wrong disk size
									// Continue anyway but log as error so it's visible
								} else {
									logger.Info("[ProxmoxClient] Successfully resized disk %s for VM %d to %dGB", actualDiskKey, vmID, diskSizeGB)
								}
							skipResize:
							} else {
								logger.Warn("[ProxmoxClient] Could not find disk %s in VM %d config after clone", actualDiskKey, vmID)
							}
						} else {
							logger.Warn("[ProxmoxClient] Failed to get VM config after clone for resize check: %v", err)
						}
						} else {
							logger.Warn("[ProxmoxClient] No disk key found for VM %d after clone - cannot resize", vmID)
						}
					} else {
						body, _ := io.ReadAll(resp.Body)
						logger.Warn("[ProxmoxClient] Failed to clone template (status %d): %s, falling back to ISO", resp.StatusCode, string(body))
						useCloudInit = false
					}
				}
			}
		}
	}

	if !useCloudInit {
		// Fallback to ISO installation
		// Note: ISO files must exist in Proxmox ISO storage for this to work
		switch config.Image {
		case 1: // UBUNTU_22_04
			vmConfig["ide2"] = "local:iso/ubuntu-22.04-server-amd64.iso,media=cdrom"
		case 2: // UBUNTU_24_04
			vmConfig["ide2"] = "local:iso/ubuntu-24.04-server-amd64.iso,media=cdrom"
		case 3: // DEBIAN_12
			vmConfig["ide2"] = "local:iso/debian-12-netinst-amd64.iso,media=cdrom"
		case 4: // DEBIAN_13
			vmConfig["ide2"] = "local:iso/debian-13-netinst-amd64.iso,media=cdrom"
		case 5: // ROCKY_LINUX_9
			vmConfig["ide2"] = "local:iso/Rocky-9-x86_64-minimal.iso,media=cdrom"
		case 6: // ALMA_LINUX_9
			vmConfig["ide2"] = "local:iso/AlmaLinux-9-x86_64-minimal.iso,media=cdrom"
		case 99: // CUSTOM
			if config.ImageID != nil {
				vmConfig["ide2"] = fmt.Sprintf("local:iso/%s,media=cdrom", *config.ImageID)
			}
		}
		// Set boot order to boot from CD-ROM first (for ISO installation)
		vmConfig["boot"] = "order=ide2;net0"
	}

	// Configure cloud-init if using template
	if useCloudInit {
		// Cloud-init configuration
		// Use ip=dhcp without specifying interface - cloud-init will auto-detect
		// Specifying interface name can cause issues if the interface name doesn't match
		vmConfig["ipconfig0"] = "ip=dhcp"
		vmConfig["ciuser"] = "root"
		// CRITICAL: Set up cloud-init drive on directory storage (snippetStorage)
		// When cloning templates to LVM/ZFS storage, the cloud-init drive must be on directory storage
		// Without this, cloud-init won't have configuration and the VM will hang waiting for metadata
		vmConfig["ide2"] = fmt.Sprintf("%s:cloudinit", snippetStorage)

		// Root password: use custom if provided, otherwise auto-generate
		if config.RootPassword != nil && *config.RootPassword != "" {
			rootPassword = *config.RootPassword
			logger.Info("[ProxmoxClient] Using custom root password for VM %d (length: %d)", vmID, len(rootPassword))
		} else {
			// Auto-generate root password
			rootPassword = GenerateRandomPassword(32)
			logger.Info("[ProxmoxClient] Auto-generated root password for VM %d (length: %d)", vmID, len(rootPassword))
			// Set it in config so it's included in cloud-init userData
			config.RootPassword = &rootPassword
		}
		// Note: Root password is set in cloud-init userData snippet, not via cipassword
		// The snippet contains the full cloud-init configuration including root password, SSH keys, guest agent, etc.

		// Always generate cloud-init userData and create a snippet file
		// This ensures guest agent, SSH server, and other essential services are properly configured
		// The userData includes: SSH server installation, guest agent installation, root password, SSH keys, etc.
		userData := GenerateCloudInitUserData(config)
		if userData != "" {
			// Create snippet file in Proxmox storage (use snippetStorage, not VM disk storage)
			snippetPath, err := pc.CreateCloudInitSnippet(ctx, nodeName, snippetStorage, vmID, userData)
			if err != nil {
				return nil, fmt.Errorf("failed to create cloud-init snippet for VM %d: %w. Snippets are required to ensure guest agent and SSH are properly configured. Ensure SSH is configured (PROXMOX_SSH_USER, PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA) and the storage supports snippets. See https://docs.obiente.cloud/guides/proxmox-ssh-user-setup for setup instructions", vmID, err)
			}
			// Use cicustom to reference the snippet
			vmConfig["cicustom"] = snippetPath
			logger.Info("[ProxmoxClient] Using cloud-init userData snippet for VM %d: %s", vmID, snippetPath)
			// Don't set cipassword or sshkeys in vmConfig when using snippets (they're in the userData)
		} else {
			return nil, fmt.Errorf("failed to generate cloud-init userData for VM %d", vmID)
		}
	}

	// Create or update VM
	endpoint := fmt.Sprintf("/nodes/%s/qemu", nodeName)
	if useCloudInit {
		// Update cloned VM configuration
		// Proxmox API expects form-encoded data for config updates
		// Note: Don't include disk config in update when cloning - disk already exists and was resized separately
		updateEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)

		// Get the actual disk key from the cloned VM to use in boot order
		vmConfigCheck, err := pc.GetVMConfig(ctx, nodeName, vmID)
		var actualDiskKey string
		if err == nil {
			// Find which disk key exists in the cloned VM
			for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
				if disk, ok := vmConfigCheck[diskKey].(string); ok && disk != "" {
					actualDiskKey = diskKey
					break
				}
			}
		}

		// If we couldn't find a disk, try to get it from template
		if actualDiskKey == "" {
			templateConfig, err := pc.GetVMConfig(ctx, nodeName, templateVMID)
			if err == nil {
				for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
					if disk, ok := templateConfig[diskKey].(string); ok && disk != "" {
						actualDiskKey = diskKey
						break
					}
				}
			}
		}

		// Default to scsi0 if we still don't know
		if actualDiskKey == "" {
			actualDiskKey = "scsi0"
			logger.Warn("[ProxmoxClient] Could not determine disk type for VM %d, defaulting to scsi0 for boot order", vmID)
		}

		formData := url.Values{}
		for key, value := range vmConfig {
			// Skip disk configs when cloning - disk already exists from template and was resized separately
			// UNLESS we just created a disk, in which case we need to include it
			if key == "scsi0" || key == "virtio0" || key == "sata0" || key == "ide0" {
				// Only skip if we didn't create this disk
				// If we created a disk, we need to include it in the config update
				if !diskCreated || key != actualDiskKey {
					continue
				}
				// We created this disk, so include it with the value we used
				if diskCreated && createdDiskValue != "" {
					formData.Set(key, createdDiskValue)
					logger.Info("[ProxmoxClient] Including newly created disk %s in VM config update: %s", key, createdDiskValue)
				}
				continue
			}
			// Skip args parameter - preserve any template configuration
			if key == "args" {
				continue
			}
			// Special handling for sshkeys - Proxmox v8.4 requires double-encoding
			if key == "sshkeys" {
				if strValue, ok := value.(string); ok && strValue != "" {
					// Use the reusable encoding function for Proxmox v8.4 double-encoding
					encodedValue := encodeSSHKeysForProxmox(strValue)
					formData.Set(key, encodedValue)
					logger.Debug("[ProxmoxClient] Setting sshkeys parameter (raw length: %d, encoded length: %d)", len(strValue), len(encodedValue))
				}
			} else {
				formData.Set(key, fmt.Sprintf("%v", value))
			}
		}

		// Ensure boot order includes the actual disk - this is critical for the VM to boot
		// Use the disk key we detected from the cloned VM or template
		formData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
		formData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
		logger.Info("[ProxmoxClient] Setting boot order to %s and bootdisk to %s for VM %d", actualDiskKey, actualDiskKey, vmID)

		// Log form data for debugging (excluding sensitive data like passwords)
		if sshKeysVal, ok := formData["sshkeys"]; ok && len(sshKeysVal) > 0 {
			logger.Info("[ProxmoxClient] Form data includes sshkeys parameter (length: %d chars)", len(sshKeysVal[0]))
			logger.Debug("[ProxmoxClient] SSH keys in form data (raw): %q", sshKeysVal[0])
			logger.Debug("[ProxmoxClient] SSH keys ends with newline: %v", strings.HasSuffix(sshKeysVal[0], "\n"))
			// Log what the encoded form data will look like
			testFormData := url.Values{}
			testFormData.Set("sshkeys", sshKeysVal[0])
			encoded := testFormData.Encode()
			logger.Debug("[ProxmoxClient] SSH keys encoded form data: %s", encoded)
			// Decode it back to verify
			if decoded, err := url.QueryUnescape(strings.TrimPrefix(encoded, "sshkeys=")); err == nil {
				logger.Debug("[ProxmoxClient] SSH keys decoded back: %q", decoded)
				logger.Debug("[ProxmoxClient] Decoded ends with newline: %v", strings.HasSuffix(decoded, "\n"))
			}
		} else {
			logger.Warn("[ProxmoxClient] Form data does NOT include sshkeys parameter")
		}


		// Now update the config with all other parameters
		resp, err := pc.apiRequestForm(ctx, "PUT", updateEndpoint, formData)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			logger.Info("[ProxmoxClient] Updated VM %d configuration", vmID)
			logger.Info("[ProxmoxClient] VM %d configured with cloud-init snippet (guest agent and SSH will be installed on first boot)", vmID)

			// Verify that the disk (scsi0) exists in the VM config
			// This is a safety check to ensure the cloned VM has a disk
			vmConfigResp, err := pc.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), nil)
			if err == nil && vmConfigResp != nil && vmConfigResp.StatusCode == http.StatusOK {
				defer vmConfigResp.Body.Close()
				var configData struct {
					Data map[string]interface{} `json:"data"`
				}
				if err := json.NewDecoder(vmConfigResp.Body).Decode(&configData); err == nil {
					// Check for any disk configuration (scsi0, virtio0, sata0, ide0)
					hasDisk := false
					var diskKey string
					var diskValue interface{}
					for _, key := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
						if disk, ok := configData.Data[key]; ok && disk != nil && disk != "" {
							hasDisk = true
							diskKey = key
							diskValue = disk
							break
						}
					}

					if !hasDisk {
						logger.Error("[ProxmoxClient] WARNING: Cloned VM %d does not have any disk configured! This will cause boot failures.", vmID)
						logger.Error("[ProxmoxClient] Creating boot disk now to fix this issue...")

						// Create a boot disk immediately
						diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
						diskSizeStr := fmt.Sprintf("%dG", diskSizeGB)

						// Determine storage type and format
						storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
						var diskValue string
						if err == nil && storageInfo != nil {
							if st, ok := storageInfo["type"].(string); ok {
								if st == "dir" || st == "directory" {
									// Directory storage: must include vmID subdirectory in path
									// Format: storage:vmID/vm-XXX-disk-0.qcow2,size=XXG,format=qcow2
									diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
								} else {
									diskValue = fmt.Sprintf("%s:vm-%d-disk-0,size=%s", storage, vmID, diskSizeStr)
								}
							} else {
								// Default to directory format with vmID subdirectory
								diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
							}
						} else {
							// Default to directory format with vmID subdirectory
							diskValue = fmt.Sprintf("%s:%d/vm-%d-disk-0.qcow2,size=%s,format=qcow2", storage, vmID, vmID, diskSizeStr)
						}

						// Use scsi0 as the default boot disk
						actualDiskKey = "scsi0"
						var diskResp *http.Response
						var diskErr error

						// Determine storage type for disk creation method
						storageInfoForDisk, errForDisk := pc.getStorageInfo(ctx, nodeName, storage)
						storageTypeForDisk := "unknown"
						if errForDisk == nil && storageInfoForDisk != nil {
							if st, ok := storageInfoForDisk["type"].(string); ok {
								storageTypeForDisk = st
							}
						}

						// If storage type detection failed, assume directory storage for "local" storage pool
						useDirectoryStorage := storageTypeForDisk == "dir" || storageTypeForDisk == "directory"
						if !useDirectoryStorage && (storage == "local" || errForDisk != nil) {
							// Default to directory storage if detection failed or storage is "local"
							useDirectoryStorage = true
							logger.Info("[ProxmoxClient] Assuming directory storage for '%s' (detection failed or default)", storage)
						}

						if useDirectoryStorage {
							// Create disk volume first using storage content API
							contentFormData := url.Values{}
							contentFormData.Set("vmid", fmt.Sprintf("%d", vmID))
							contentFormData.Set("filename", fmt.Sprintf("vm-%d-disk-0.qcow2", vmID))
							contentFormData.Set("size", diskSizeStr)
							contentFormData.Set("format", "qcow2")

							contentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage)
							logger.Info("[ProxmoxClient] Creating disk volume for VM %d via storage API: %s", vmID, contentEndpoint)
							contentResp, contentErr := pc.apiRequestForm(ctx, "POST", contentEndpoint, contentFormData)
							if contentErr == nil && contentResp != nil {
								if contentResp.StatusCode == http.StatusOK {
									contentResp.Body.Close()
									logger.Info("[ProxmoxClient] Successfully created disk volume for VM %d", vmID)
									// Now attach the disk to the VM config
									diskFormData := url.Values{}
									diskFormData.Set(actualDiskKey, diskValue)
									logger.Info("[ProxmoxClient] Creating missing boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
									diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
								} else {
									body, _ := io.ReadAll(contentResp.Body)
									contentResp.Body.Close()
									logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: status %d, response: %s", vmID, contentResp.StatusCode, string(body))
									diskErr = fmt.Errorf("failed to create disk volume: status %d", contentResp.StatusCode)
								}
							} else {
								logger.Error("[ProxmoxClient] Failed to create disk volume for VM %d: %v", vmID, contentErr)
								diskErr = contentErr
							}
						} else {
							// For LVM/ZFS, we can set it directly
							diskFormData := url.Values{}
							diskFormData.Set(actualDiskKey, diskValue)
							logger.Info("[ProxmoxClient] Creating missing boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)
							diskResp, diskErr = pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
						}
						if diskErr == nil && diskResp != nil && diskResp.StatusCode == http.StatusOK {
							diskResp.Body.Close()
							logger.Info("[ProxmoxClient] Successfully created missing boot disk %s for VM %d: %s", actualDiskKey, vmID, diskValue)

							// Update boot order and bootdisk to use this disk
							bootFormData := url.Values{}
							bootFormData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
							bootFormData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
							bootResp, bootErr := pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), bootFormData)
							if bootErr == nil && bootResp != nil && bootResp.StatusCode == http.StatusOK {
								bootResp.Body.Close()
								logger.Info("[ProxmoxClient] Updated boot order to use disk %s for VM %d", actualDiskKey, vmID)
							} else {
								logger.Warn("[ProxmoxClient] Failed to update boot order for VM %d: %v", vmID, bootErr)
							}
						} else {
							var body []byte
							if diskResp != nil {
								body, _ = io.ReadAll(diskResp.Body)
							}
							logger.Error("[ProxmoxClient] CRITICAL: Failed to create missing boot disk for VM %d: %v. Response: %s", vmID, diskErr, string(body))
							// Try LVM/ZFS format as fallback (without format parameter)
							if strings.Contains(string(body), "format") || strings.Contains(string(body), "qcow2") {
								logger.Info("[ProxmoxClient] Retrying with LVM/ZFS format (no format parameter) for VM %d", vmID)
								diskValue = fmt.Sprintf("%s:vm-%d-disk-0,size=%s", storage, vmID, diskSizeStr)
								diskFormData := url.Values{}
								diskFormData.Set(actualDiskKey, diskValue)
								diskResp2, diskErr2 := pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), diskFormData)
								if diskErr2 == nil && diskResp2 != nil && diskResp2.StatusCode == http.StatusOK {
									diskResp2.Body.Close()
									logger.Info("[ProxmoxClient] Successfully created missing boot disk %s for VM %d with LVM/ZFS format: %s", actualDiskKey, vmID, diskValue)

									// Update boot order and bootdisk
									bootFormData := url.Values{}
									bootFormData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
									bootFormData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
									bootResp, bootErr := pc.apiRequestForm(ctx, "PUT", fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID), bootFormData)
									if bootErr == nil && bootResp != nil && bootResp.StatusCode == http.StatusOK {
										bootResp.Body.Close()
										logger.Info("[ProxmoxClient] Updated boot order to use disk %s for VM %d", actualDiskKey, vmID)
									}
								} else {
									var body2 []byte
									if diskResp2 != nil {
										body2, _ = io.ReadAll(diskResp2.Body)
									}
									logger.Error("[ProxmoxClient] CRITICAL: Failed to create missing boot disk with LVM/ZFS format for VM %d: %v. Response: %s", vmID, diskErr2, string(body2))
								}
							}
						}
					} else {
						logger.Info("[ProxmoxClient] Verified VM %d has %s disk: %v", vmID, diskKey, diskValue)
					}
				}
			}
		} else {
			// Config update failed, but VM was already cloned successfully
			// Retry with just cloud-init config (smaller update, more likely to succeed)
			var body []byte
			if resp != nil && resp.Body != nil {
				body, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
			bodyStr := ""
			if len(body) > 0 {
				bodyStr = string(body)
			}
			logger.Warn("[ProxmoxClient] Initial config update failed for VM %d: %v. Response: %s. Retrying with cloud-init config only...", vmID, err, bodyStr)

			// Get the actual disk key from the cloned VM to use in boot order
			vmConfigCheck, err := pc.GetVMConfig(ctx, nodeName, vmID)
			var actualDiskKey string
			if err == nil {
				// Find which disk key exists in the cloned VM
				for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
					if disk, ok := vmConfigCheck[diskKey].(string); ok && disk != "" {
						actualDiskKey = diskKey
						break
					}
				}
			}

			// If we couldn't find a disk, try to get it from template
			if actualDiskKey == "" {
				templateConfig, err := pc.GetVMConfig(ctx, nodeName, templateVMID)
				if err == nil {
					for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
						if disk, ok := templateConfig[diskKey].(string); ok && disk != "" {
							actualDiskKey = diskKey
							break
						}
					}
				}
			}

			// Default to scsi0 if we still don't know
			if actualDiskKey == "" {
				actualDiskKey = "scsi0"
				logger.Warn("[ProxmoxClient] Could not determine disk type for VM %d in retry, defaulting to scsi0 for boot order", vmID)
			}

			// Retry with minimal cloud-init config
			retryFormData := url.Values{}
			retryFormData.Set("ipconfig0", "ip=dhcp")
			retryFormData.Set("ciuser", "root")
			// Safely get cipassword - it might not exist if using snippets
			if cipasswordVal, ok := vmConfig["cipassword"]; ok && cipasswordVal != nil {
				if cipasswordStr, ok := cipasswordVal.(string); ok && cipasswordStr != "" {
					retryFormData.Set("cipassword", cipasswordStr)
				}
			}
			// Include SSH keys in retry if they exist
			if sshKeysVal, ok := vmConfig["sshkeys"]; ok {
				if sshKeysStr, ok := sshKeysVal.(string); ok && sshKeysStr != "" {
					retryFormData.Set("sshkeys", sshKeysStr)
					logger.Debug("[ProxmoxClient] Including SSH keys in retry config update")
				}
			}
			retryFormData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
			retryFormData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
			logger.Info("[ProxmoxClient] Retrying with boot order %s and bootdisk %s for VM %d", actualDiskKey, actualDiskKey, vmID)

			// Now update the config with retry parameters
			retryResp, retryErr := pc.apiRequestForm(ctx, "PUT", updateEndpoint, retryFormData)
			if retryErr == nil && retryResp != nil && retryResp.StatusCode == http.StatusOK {
				retryResp.Body.Close()
				logger.Info("[ProxmoxClient] Successfully applied cloud-init config to VM %d on retry", vmID)
			} else {
				var retryBody []byte
				if retryResp != nil {
					retryBody, _ = io.ReadAll(retryResp.Body)
					if retryResp.StatusCode == 403 {
						logger.Error("[ProxmoxClient] Permission denied updating cloud-init config (token may need VM.Config.* permissions). VM %d was cloned but cloud-init may not work. Error: %s", vmID, string(retryBody))
					} else {
						logger.Error("[ProxmoxClient] Failed to apply cloud-init config to VM %d: %v. Response: %s", vmID, retryErr, string(retryBody))
					}
				} else {
					logger.Error("[ProxmoxClient] Failed to apply cloud-init config to VM %d: %v", vmID, retryErr)
				}
				// Continue anyway - VM exists, cloud-init may not work but VM can still be used
			}
		}
	}

	// Only create new VM if we didn't use cloud-init (no template found or clone failed)
	if !useCloudInit && imageTemplate == "" {
		// Create new VM from scratch
		// Proxmox API expects form-encoded data, not JSON
		// Re-determine disk format in case storage type wasn't detected earlier
		diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
		if storageType == "unknown" || storageType == "" {
			// Try to get storage type again if we don't have it
			storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
			if err == nil && storageInfo != nil {
				if st, ok := storageInfo["type"].(string); ok {
					storageType = st
					logger.Info("[ProxmoxClient] Detected storage type '%s' for fallback VM creation", storageType)
				}
			}
		}

		// Update scsi0 config based on storage type
		// Directory storage types: "dir", "directory", "nfs", "cifs", "glusterfs"
		// Block storage types: "lvm", "lvm-thin", "zfs", "zfspool"
		if storageType == "dir" || storageType == "directory" || storageType == "nfs" || storageType == "cifs" || storageType == "glusterfs" {
			vmConfig["scsi0"] = fmt.Sprintf("%s:size=%dG", storage, diskSizeGB)
			logger.Info("[ProxmoxClient] Using directory storage format for scsi0: %s", vmConfig["scsi0"])
		} else {
			vmConfig["scsi0"] = fmt.Sprintf("%s:vm-%d-disk-0,size=%dG", storage, vmID, diskSizeGB)
			logger.Info("[ProxmoxClient] Using block storage format for scsi0: %s", vmConfig["scsi0"])
		}

		formData := url.Values{}
		for key, value := range vmConfig {
			formData.Set(key, fmt.Sprintf("%v", value))
		}

		resp, err := pc.apiRequestForm(ctx, "POST", endpoint, formData)
		if err != nil {
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorMsg := string(body)
			if resp.StatusCode == 403 {
				return nil, fmt.Errorf("failed to create VM: permission denied (status: %d). The API token needs VM.Allocate, VM.Config.Disk, and Datastore.Allocate permissions. Error: %s", resp.StatusCode, errorMsg)
			}
			if resp.StatusCode == 500 && strings.Contains(errorMsg, "storage") {
				// Try to get available storages for better error message
				availableStorages, listErr := pc.listStorages(ctx, nodeName)
				if listErr == nil && len(availableStorages) > 0 {
					return nil, fmt.Errorf("failed to create VM: storage error (status: %d). Error: %s. Available storage pools on node '%s': %v", resp.StatusCode, errorMsg, nodeName, availableStorages)
				}
			}
			return nil, fmt.Errorf("failed to create VM: %s (status: %d)", errorMsg, resp.StatusCode)
		}
	}

	logger.Info("[ProxmoxClient] Created VM %d on node %s", vmID, nodeName)

	// Configure firewall rules for inter-VM communication
	// Use separate context to avoid parent context cancellation
	firewallCtx, firewallCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer firewallCancel()
	if err := pc.configureVMFirewall(firewallCtx, nodeName, vmID, config.OrganizationID, allowInterVM); err != nil {
		logger.Warn("[ProxmoxClient] Failed to configure firewall for VM %d: %v", vmID, err)
		// Continue anyway - VM is created, firewall can be configured manually
	}

	// Start the VM
	// Use separate context to avoid parent context cancellation
	startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startCancel()
	if err := pc.startVM(startCtx, nodeName, vmID); err != nil {
		logger.Warn("[ProxmoxClient] Failed to start VM %d: %v", vmID, err)
		// Continue anyway - VM is created
	}

	// rootPassword was already captured when it was generated (if cloud-init was used)
	// No need to retrieve it again - it's already in the function-scoped variable
	if useCloudInit {
		if rootPassword == "" {
			logger.Warn("[ProxmoxClient] WARNING: rootPassword is empty for VM %d (cloud-init was used but password not captured)", vmID)
		} else {
			logger.Info("[ProxmoxClient] Returning root password for VM %d (length: %d)", vmID, len(rootPassword))
		}
	} else {
		logger.Debug("[ProxmoxClient] No root password for VM %d (cloud-init not used)", vmID)
	}

	return &CreateVMResult{
		VMID:     fmt.Sprintf("%d", vmID),
		Password: rootPassword,
		NodeName: nodeName,
	}, nil
}

func (pc *ProxmoxClient) StartVM(ctx context.Context, nodeName string, vmID int) error {
	return pc.startVM(ctx, nodeName, vmID)
}

// isVMDeletedError checks if an error indicates the VM was deleted from Proxmox
func (pc *ProxmoxClient) isVMDeletedError(err error, statusCode int, bodyStr string) bool {
	if err == nil && statusCode == 0 && bodyStr == "" {
		return false
	}
	
	// Check error message
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unable to find configuration file") ||
			strings.Contains(errStr, "does not exist") ||
			strings.Contains(errStr, "not found on any node") {
			return true
		}
	}
	
	// Check response body and status code
	if statusCode == 500 || statusCode == 404 {
		if strings.Contains(bodyStr, "unable to find configuration file") ||
			strings.Contains(bodyStr, "does not exist") ||
			strings.Contains(bodyStr, "not found") {
			return true
		}
	}
	
	return false
}

func (pc *ProxmoxClient) startVM(ctx context.Context, nodeName string, vmID int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", nodeName, vmID)
	// Proxmox API expects form-encoded data for POST requests, even if empty
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		// Check if error indicates VM was deleted
		if pc.isVMDeletedError(err, 0, "") {
			return fmt.Errorf("VM %d has been deleted from Proxmox", vmID)
		}
		return fmt.Errorf("failed to start VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if VM was deleted (configuration file not found)
		if pc.isVMDeletedError(nil, resp.StatusCode, bodyStr) {
			return fmt.Errorf("VM %d has been deleted from Proxmox", vmID)
		}
		
		return fmt.Errorf("failed to start VM: %s (status: %d)", bodyStr, resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) StopVM(ctx context.Context, nodeName string, vmID int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", nodeName, vmID)
	// Proxmox API expects form-encoded data for POST requests, even if empty
	// /status/stop forces an immediate shutdown (not graceful)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		// Check if error indicates VM was deleted
		if pc.isVMDeletedError(err, 0, "") {
			return fmt.Errorf("VM %d has been deleted from Proxmox", vmID)
		}
		return fmt.Errorf("failed to stop VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if VM was deleted (configuration file not found)
		if pc.isVMDeletedError(nil, resp.StatusCode, bodyStr) {
			return fmt.Errorf("VM %d has been deleted from Proxmox", vmID)
		}
		
		return fmt.Errorf("failed to stop VM: %s (status: %d)", bodyStr, resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) DeleteVM(ctx context.Context, nodeName string, vmID int, vpsID string) error {
	// SECURITY: Verify VM was created by our API before deletion
	// Get VM config to check VM name matches VPS ID
	configEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", configEndpoint, nil)
	if err != nil {
		// Network/connection error - try to proceed with deletion anyway if we can find the VM
		logger.Warn("[ProxmoxClient] Failed to get VM config for validation (network error): %v. Will attempt deletion anyway.", err)
		// Try to find VM on other nodes or proceed with deletion
	} else {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorMsg := string(body)

			// If VM config doesn't exist, the VM is already deleted
			if resp.StatusCode == 500 && strings.Contains(errorMsg, "does not exist") {
				logger.Info("[ProxmoxClient] VM %d config does not exist - VM is already deleted", vmID)
				return nil // VM already deleted, nothing to do
			}

			// Handle unusual status codes (like 596) - might be Proxmox-specific errors
			// If it's a 4xx or 5xx that's not a standard error, try to proceed with deletion
			if resp.StatusCode >= 400 && resp.StatusCode < 600 {
				// Check if VM exists by trying to find it on other nodes
				logger.Warn("[ProxmoxClient] Got unusual status %d when getting VM config: %s. Will attempt to find VM on other nodes or proceed with deletion.", resp.StatusCode, errorMsg)
				
				// Try to find VM on other nodes
				allNodes, listErr := pc.ListNodes(ctx)
				if listErr == nil {
					for _, otherNode := range allNodes {
						if otherNode == nodeName {
							continue // Skip the node we already tried
						}
						otherConfigEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", otherNode, vmID)
						otherResp, otherErr := pc.apiRequest(ctx, "GET", otherConfigEndpoint, nil)
						if otherErr == nil && otherResp.StatusCode == http.StatusOK {
							otherResp.Body.Close()
							logger.Info("[ProxmoxClient] Found VM %d on node %s instead of %s", vmID, otherNode, nodeName)
							nodeName = otherNode // Update node name for deletion
							goto skipValidation // Skip validation since we found it on another node
						}
						if otherResp != nil {
							otherResp.Body.Close()
						}
					}
				}
				
				// If we can't validate, log warning but proceed with deletion attempt
				// This handles cases where Proxmox API is having issues but VM still exists
				logger.Warn("[ProxmoxClient] Cannot validate VM %d ownership due to API error (status %d), but will attempt deletion. This may be unsafe if VM name doesn't match VPS ID.", vmID, resp.StatusCode)
				goto skipValidation
			}

			return fmt.Errorf("failed to get VM config: %s (status: %d)", errorMsg, resp.StatusCode)
		}
	}

	// Validate VM ownership by checking name matches VPS ID (only if we got a valid response)
	if resp != nil {
		var configResp struct {
			Data map[string]interface{} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
			logger.Warn("[ProxmoxClient] Failed to decode VM config: %v. Will attempt deletion anyway.", err)
			goto skipValidation
		}

		// Check if VM name matches VPS ID (this is how we identify our VMs)
		vmName, ok := configResp.Data["name"].(string)
		if !ok || vmName == "" {
			return fmt.Errorf("refusing to delete VM %d: VM name is missing or empty", vmID)
		}

		if vmName != vpsID {
			return fmt.Errorf("refusing to delete VM %d: VM name '%s' does not match VPS ID '%s'", vmID, vmName, vpsID)
		}

		logger.Info("[ProxmoxClient] VM %d verified as Obiente Cloud managed (name matches VPS ID: %s)", vmID, vpsID)
	}

skipValidation:
	// Proceed with VM deletion (validation may have been skipped due to API errors)
	// Check VM status and force stop it if running (Proxmox requires VM to be stopped before deletion)
	// We use force stop (/status/stop) not graceful shutdown to ensure immediate stop
	status, err := pc.GetVMStatus(ctx, nodeName, vmID)
	if err != nil {
		logger.Warn("[ProxmoxClient] Failed to get VM status before deletion, attempting to force stop VM anyway: %v", err)
		// Try to force stop anyway - better to try and fail than to fail deletion
		logger.Info("[ProxmoxClient] Attempting to force stop VM %d before deletion", vmID)
		if err := pc.StopVM(ctx, nodeName, vmID); err != nil {
			logger.Warn("[ProxmoxClient] Failed to force stop VM %d (may already be stopped): %v", vmID, err)
		} else {
			// Wait for VM to stop
			if err := pc.waitForVMStatus(ctx, nodeName, vmID, "stopped", 30*time.Second); err != nil {
				logger.Warn("[ProxmoxClient] VM %d may not have stopped in time: %v", vmID, err)
			}
		}
	} else if status != "stopped" {
		logger.Info("[ProxmoxClient] VM %d is in status '%s', force stopping before deletion", vmID, status)
		if err := pc.StopVM(ctx, nodeName, vmID); err != nil {
			return fmt.Errorf("failed to force stop VM before deletion: %w", err)
		}
		// Wait for VM to actually stop (with timeout)
		if err := pc.waitForVMStatus(ctx, nodeName, vmID, "stopped", 30*time.Second); err != nil {
			return fmt.Errorf("VM %d did not stop within timeout: %w", vmID, err)
		}
		logger.Info("[ProxmoxClient] VM %d force stopped successfully", vmID)
	}

	// VM is verified as ours and stopped, proceed with deletion
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d", nodeName, vmID)
	deleteResp, err := pc.apiRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(deleteResp.Body)
		errorMsg := string(body)

		// Check if VM was already deleted (404 or 500 with "does not exist" message)
		if deleteResp.StatusCode == 404 ||
			(deleteResp.StatusCode == 500 && strings.Contains(errorMsg, "does not exist")) {
			logger.Info("[ProxmoxClient] VM %d does not exist - already deleted", vmID)
			return nil // VM already deleted, nothing to do
		}

		return fmt.Errorf("failed to delete VM: %s (status: %d)", errorMsg, deleteResp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully deleted VM %d (verified as Obiente Cloud managed)", vmID)
	return nil
}

func (pc *ProxmoxClient) GetVMStatus(ctx context.Context, nodeName string, vmID int) (string, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get VM status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		// Check if VM was deleted (404 or 500 with "does not exist" message)
		if resp.StatusCode == 404 || (resp.StatusCode == 500 && strings.Contains(bodyStr, "does not exist")) {
			return "", fmt.Errorf("VM does not exist (status: %d)", resp.StatusCode)
		}
		return "", fmt.Errorf("failed to get VM status: %s (status: %d)", bodyStr, resp.StatusCode)
	}

	var statusResp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return "", fmt.Errorf("failed to decode status response: %w", err)
	}

	return statusResp.Data.Status, nil
}

func (pc *ProxmoxClient) GetVMMetrics(ctx context.Context, nodeName string, vmID int) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get VM metrics: %s (status: %d)", string(body), resp.StatusCode)
	}

	var metricsResp struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metricsResp); err != nil {
		return nil, fmt.Errorf("failed to decode metrics response: %w", err)
	}

	return metricsResp.Data, nil
}

func (pc *ProxmoxClient) GetVMDiskSize(ctx context.Context, nodeName string, vmID int) (int64, error) {
	vmConfig, err := pc.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return 0, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Look for scsi0, virtio0, sata0, or ide0 disk configuration
	// Format is typically: "storage:vm-XXX-disk-0,size=XXG" or "storage:vm-XXX-disk-0"
	diskKeys := []string{"scsi0", "virtio0", "sata0", "ide0"}
	var diskVolume string
	var storageName string

	for _, key := range diskKeys {
		if diskConfig, ok := vmConfig[key].(string); ok && diskConfig != "" {
			// Parse size from disk config if present (e.g., "local-lvm:vm-301-disk-0,size=20G")
			if strings.Contains(diskConfig, "size=") {
				sizePart := strings.Split(diskConfig, "size=")
				if len(sizePart) > 1 {
					sizeStr := strings.TrimSpace(sizePart[1])
					// Remove any trailing commas or other parameters
					if idx := strings.Index(sizeStr, ","); idx != -1 {
						sizeStr = sizeStr[:idx]
					}

					// Parse size (format: "20G", "100M", etc.)
					var size int64
					var unit string
					if _, err := fmt.Sscanf(sizeStr, "%d%s", &size, &unit); err == nil {
						// Convert to bytes
						switch strings.ToUpper(unit) {
						case "G", "GB":
							return size * 1024 * 1024 * 1024, nil
						case "M", "MB":
							return size * 1024 * 1024, nil
						case "K", "KB":
							return size * 1024, nil
						case "T", "TB":
							return size * 1024 * 1024 * 1024 * 1024, nil
						default:
							// Assume bytes if no unit
							return size, nil
						}
					}
				}
			}

			// Extract storage and volume name for fallback query
			// Format: "storage:volume" or "storage:volume,size=XXG"
			parts := strings.Split(diskConfig, ":")
			if len(parts) >= 2 {
				storageName = parts[0]
				volumePart := parts[1]
				// Remove size parameter and other options
				if idx := strings.Index(volumePart, ","); idx != -1 {
					volumePart = volumePart[:idx]
				}
				diskVolume = volumePart
				break
			}
		}
	}

	// If size not found in config, try to get it from storage API
	if diskVolume != "" && storageName != "" {
		// Query storage volume size
		// Format: /nodes/{node}/storage/{storage}/content/{volume}
		endpoint := fmt.Sprintf("/nodes/%s/storage/%s/content/%s", nodeName, storageName, diskVolume)
		resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			var volumeResp struct {
				Data struct {
					Size int64 `json:"size"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&volumeResp); err == nil {
				if volumeResp.Data.Size > 0 {
					return volumeResp.Data.Size, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("disk size not found in VM config or storage")
}

func (pc *ProxmoxClient) ResizeDisk(ctx context.Context, nodeName string, vmID int, disk string, sizeGB int64) error {
	return pc.resizeDisk(ctx, nodeName, vmID, disk, sizeGB)
}

func (pc *ProxmoxClient) resizeDisk(ctx context.Context, nodeName string, vmID int, disk string, sizeGB int64) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/resize", nodeName, vmID)
	formData := url.Values{}
	formData.Set("disk", disk)
	formData.Set("size", fmt.Sprintf("%dG", sizeGB))

	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to resize disk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to resize disk: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// MoveDisk moves a disk from one storage to another
// disk should be the disk identifier (e.g., "scsi0", "virtio0")
// targetStorage is the destination storage pool
// deleteSource: if true, deletes the source disk after move (default: true)
// Reference: https://pve.proxmox.com/pve-docs/api-viewer/index.html#/nodes/{node}/qemu/{vmid}/move_disk
func (pc *ProxmoxClient) MoveDisk(ctx context.Context, nodeName string, vmID int, disk string, targetStorage string, deleteSource bool) error {
	return pc.moveDisk(ctx, nodeName, vmID, disk, targetStorage, deleteSource)
}

func (pc *ProxmoxClient) moveDisk(ctx context.Context, nodeName string, vmID int, disk string, targetStorage string, deleteSource bool) error {
	// Proxmox API endpoint: POST /nodes/{node}/qemu/{vmid}/move_disk
	// Reference: https://pve.proxmox.com/pve-docs/api-viewer/index.html#/nodes/{node}/qemu/{vmid}/move_disk
	// Parameters:
	//   - disk: The disk identifier (e.g., "scsi0", "virtio0")
	//   - storage: The target storage pool
	//   - delete: Boolean, delete source disk after move (optional, default: false)
	//   - format: Target disk format (optional)
	//   - bwlimit: Bandwidth limit in KiB/s (optional)
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/move_disk", nodeName, vmID)
	formData := url.Values{}
	formData.Set("disk", disk)
	formData.Set("storage", targetStorage)
	if deleteSource {
		formData.Set("delete", "1")
	} else {
		formData.Set("delete", "0")
	}

	logger.Info("[ProxmoxClient] Moving disk %s for VM %d from current storage to %s (delete source: %v)", disk, vmID, targetStorage, deleteSource)
	
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to move disk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to move disk: %s (status: %d)", string(body), resp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully moved disk %s for VM %d to storage %s", disk, vmID, targetStorage)
	return nil
}

func (pc *ProxmoxClient) UpdateVMConfig(ctx context.Context, nodeName string, vmID int, config map[string]interface{}) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}

	for k, v := range config {
		formData.Set(k, fmt.Sprintf("%v", v))
	}

	resp, err := pc.APIRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to update VM config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update VM config: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) GetVMConfig(ctx context.Context, nodeName string, vmID int) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get VM config: %s (status: %d)", string(body), resp.StatusCode)
	}

	var configResp struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return configResp.Data, nil
}

func (pc *ProxmoxClient) RebootVM(ctx context.Context, nodeName string, vmID int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/reboot", nodeName, vmID)
	// Proxmox API expects form-encoded data for POST requests, even if empty
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return fmt.Errorf("failed to reboot VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if VM was deleted (configuration file not found)
		if pc.isVMDeletedError(nil, resp.StatusCode, bodyStr) {
			return fmt.Errorf("VM %d has been deleted from Proxmox", vmID)
		}
		
		return fmt.Errorf("failed to reboot VM: %s (status: %d)", bodyStr, resp.StatusCode)
	}

	return nil
}

func (pc *ProxmoxClient) waitForVMStatus(ctx context.Context, nodeName string, vmID int, targetStatus string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		status, err := pc.GetVMStatus(ctx, nodeName, vmID)
		if err != nil {
			// If we can't get status, continue waiting
			logger.Debug("[ProxmoxClient] Failed to get VM %d status while waiting: %v", vmID, err)
		} else if status == targetStatus {
			return nil
		}

		// Wait before next check
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Timeout reached
	currentStatus, _ := pc.GetVMStatus(ctx, nodeName, vmID)
	return fmt.Errorf("timeout waiting for VM %d to reach status '%s' (current: '%s')", vmID, targetStatus, currentStatus)
}

func (pc *ProxmoxClient) findTemplate(ctx context.Context, nodeName, templatePattern string) (int, error) {
	resp, err := pc.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu", nodeName), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var vmsResp struct {
		Data []struct {
			Vmid     int    `json:"vmid"`
			Name     string `json:"name"`
			Template int    `json:"template"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vmsResp); err != nil {
		return 0, err
	}

	// Find template matching pattern
	for _, vm := range vmsResp.Data {
		if vm.Template == 1 && strings.Contains(vm.Name, templatePattern) {
			return vm.Vmid, nil
		}
	}

	return 0, fmt.Errorf("template %s not found", templatePattern)
}

func (pc *ProxmoxClient) EnableVMGuestAgent(ctx context.Context, nodeName string, vmID int) error {
	// Check current config to see if agent is already enabled
	vmConfig, err := pc.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return fmt.Errorf("failed to get VM config: %w", err)
	}

	// Check if agent is already enabled
	if agentVal, ok := vmConfig["agent"]; ok {
		// Convert to string and check if it's "1" or "enabled"
		agentStr := fmt.Sprintf("%v", agentVal)
		if agentStr == "1" || agentStr == "enabled" {
			logger.Info("[ProxmoxClient] Guest agent already enabled for VM %d", vmID)
			return nil
		}
	}

	// Enable guest agent in VM config
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}
	formData.Set("agent", "1")

	logger.Info("[ProxmoxClient] Enabling guest agent for VM %d", vmID)
	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to enable guest agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to enable guest agent: %s (status: %d)", string(body), resp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully enabled guest agent for VM %d", vmID)
	return nil
}

func (pc *ProxmoxClient) RecoverVMGuestAgent(ctx context.Context, nodeName string, vmID int, organizationID string, vpsID string) error {
	// First, ensure guest agent is enabled in VM config
	if err := pc.EnableVMGuestAgent(ctx, nodeName, vmID); err != nil {
		logger.Warn("[ProxmoxClient] Failed to enable guest agent in VM config: %v", err)
		// Continue anyway - cloud-init update might still work
	}

	// Get current cloud-init config
	ciEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit/dump", nodeName, vmID)
	ciResp, err := pc.apiRequest(ctx, "GET", ciEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to get cloud-init config: %w", err)
	}
	defer ciResp.Body.Close()

	if ciResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(ciResp.Body)
		return fmt.Errorf("failed to get cloud-init config: %s (status: %d)", string(body), ciResp.StatusCode)
	}

	var ciData struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(ciResp.Body).Decode(&ciData); err != nil {
		return fmt.Errorf("failed to decode cloud-init config: %w", err)
	}

	// Get current userData
	// Proxmox VE 8.4 uses 'userdata' field name
	currentUserData, _ := ciData.Data["userdata"].(string)
	// Fallback to 'user' for older Proxmox versions
	if currentUserData == "" {
		currentUserData, _ = ciData.Data["user"].(string)
	}

	// Check if guest agent is already configured
	if strings.Contains(currentUserData, "qemu-guest-agent") &&
		strings.Contains(currentUserData, "systemctl") {
		logger.Info("[ProxmoxClient] Guest agent already configured in cloud-init for VM %d", vmID)
		// Still regenerate to ensure it's applied
	} else {
		logger.Info("[ProxmoxClient] Adding guest agent configuration to cloud-init for VM %d", vmID)

		// Build new userData with guest agent setup
		// Start with cloud-config header if not present
		newUserData := currentUserData
		if !strings.Contains(newUserData, "#cloud-config") {
			newUserData = "#cloud-config\n" + newUserData
		}

		// Ensure packages section exists
		if !strings.Contains(newUserData, "packages:") {
			// Add packages section before runcmd if it exists, otherwise at the end
			if strings.Contains(newUserData, "runcmd:") {
				newUserData = strings.Replace(newUserData, "runcmd:", "packages:\n  - qemu-guest-agent\nruncmd:", 1)
			} else {
				newUserData += "\npackages:\n  - qemu-guest-agent\n"
			}
		} else if !strings.Contains(newUserData, "qemu-guest-agent") {
			// Add qemu-guest-agent to existing packages list
			newUserData = strings.Replace(newUserData, "packages:", "packages:\n  - qemu-guest-agent", 1)
		}

		// Ensure runcmd section exists with guest agent commands
		guestAgentCmds := "  - systemctl enable --now qemu-guest-agent"
		if !strings.Contains(newUserData, "runcmd:") {
			newUserData += "\nruncmd:\n" + guestAgentCmds + "\n"
		} else if !strings.Contains(newUserData, "qemu-guest-agent") {
			// Add guest agent commands to existing runcmd
			if strings.HasSuffix(strings.TrimSpace(newUserData), "runcmd:") {
				newUserData += "\n" + guestAgentCmds + "\n"
			} else {
				// Insert before the last line or append
				lines := strings.Split(newUserData, "\n")
				runcmdIdx := -1
				for i, line := range lines {
					if strings.TrimSpace(line) == "runcmd:" {
						runcmdIdx = i
						break
					}
				}
				if runcmdIdx >= 0 && runcmdIdx < len(lines)-1 {
					// Insert after runcmd:
					newLines := append(lines[:runcmdIdx+1], guestAgentCmds)
					newLines = append(newLines, lines[runcmdIdx+1:]...)
					newUserData = strings.Join(newLines, "\n")
				} else {
					newUserData += "\n" + guestAgentCmds + "\n"
				}
			}
		}

		// Update cloud-init userData
		updateEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit", nodeName, vmID)
		formData := url.Values{}
		// Proxmox VE 8.4 uses 'userdata' field name for cloud-init user data
		formData.Set("userdata", newUserData)

		updateResp, err := pc.apiRequestForm(ctx, "PUT", updateEndpoint, formData)
		if err != nil {
			return fmt.Errorf("failed to update cloud-init: %w", err)
		}
		defer updateResp.Body.Close()

		if updateResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(updateResp.Body)
			return fmt.Errorf("failed to update cloud-init: %s (status: %d)", string(body), updateResp.StatusCode)
		}
	}

	logger.Info("[ProxmoxClient] Successfully recovered guest agent configuration for VM %d. VM should be rebooted for changes to take effect.", vmID)
	return nil
}

func (pc *ProxmoxClient) UpdateVMCloudInitPassword(ctx context.Context, nodeName string, vmID int, newPassword string) error {
	// Update cipassword in VM config (cloud-init password)
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}
	formData.Set("cipassword", newPassword)

	logger.Info("[ProxmoxClient] Updating root password for VM %d", vmID)
	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update password: %s (status: %d)", string(body), resp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully updated root password for VM %d. Password will take effect after VM reboot or cloud-init re-run.", vmID)
	return nil
}

func (pc *ProxmoxClient) UpdateVMCicustom(ctx context.Context, nodeName string, vmID int, cicustom string) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}
	formData.Set("cicustom", cicustom)

	logger.Info("[ProxmoxClient] Updating cicustom for VM %d: %s", vmID, cicustom)
	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		return fmt.Errorf("failed to update cicustom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update cicustom: %s (status: %d)", string(body), resp.StatusCode)
	}

	logger.Info("[ProxmoxClient] Successfully updated cicustom for VM %d", vmID)
	return nil
}

func (pc *ProxmoxClient) GetVMIPAddresses(ctx context.Context, nodeName string, vmID int) ([]string, []string, error) {
	// First check if guest agent is available
	agentAvailable, err := pc.CheckGuestAgentStatus(ctx, nodeName, vmID)
	if err != nil {
		logger.Warn("[ProxmoxClient] Failed to check guest agent status: %v", err)
		// Continue anyway - might be a transient error
	} else if !agentAvailable {
		return nil, nil, fmt.Errorf("guest agent is not available (not installed or not running). VM may need to be rebooted or guest agent needs to be installed")
	}

	// Execute guest agent command to get network info
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		// Guest agent might not be available yet
		return nil, nil, fmt.Errorf("failed to get VM IP addresses (guest agent may not be ready): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Guest agent not ready or VM not running
		body, _ := io.ReadAll(resp.Body)
		logger.Warn("[ProxmoxClient] Guest agent returned non-OK status %d: %s", resp.StatusCode, string(body))
		return nil, nil, nil
	}

	var networkResp struct {
		Data struct {
			Result []struct {
				IPAddresses []struct {
					IPAddress     string `json:"ip-address"`
					IPAddressType string `json:"ip-address-type"`
				} `json:"ip-addresses"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&networkResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode network response: %w", err)
	}

	var ipv4 []string
	var ipv6 []string

	for _, iface := range networkResp.Data.Result {
		for _, ip := range iface.IPAddresses {
			if ip.IPAddressType == "ipv4" && ip.IPAddress != "" && !strings.HasPrefix(ip.IPAddress, "127.") {
				ipv4 = append(ipv4, ip.IPAddress)
			} else if ip.IPAddressType == "ipv6" && ip.IPAddress != "" && !strings.HasPrefix(ip.IPAddress, "::1") {
				ipv6 = append(ipv6, ip.IPAddress)
			}
		}
	}

	return ipv4, ipv6, nil
}

// CheckGuestAgentStatus checks if the QEMU guest agent is running and responsive
// Returns true if the agent is available, false otherwise
func (pc *ProxmoxClient) CheckGuestAgentStatus(ctx context.Context, nodeName string, vmID int) (bool, error) {
	// Try to ping the guest agent - this is the simplest way to check if it's running
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/agent/ping", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to ping guest agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// If ping fails, try a simple info command as fallback
	infoEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/agent/info", nodeName, vmID)
	infoResp, infoErr := pc.apiRequest(ctx, "GET", infoEndpoint, nil)
	if infoErr == nil && infoResp != nil {
		defer infoResp.Body.Close()
		if infoResp.StatusCode == http.StatusOK {
			return true, nil
		}
	}

	return false, nil
}
