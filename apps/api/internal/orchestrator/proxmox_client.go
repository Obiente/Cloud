package orchestrator

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"api/internal/logger"
)

// ProxmoxClient handles communication with Proxmox API
type ProxmoxClient struct {
	config     *ProxmoxConfig
	httpClient *http.Client
	ticket     *ProxmoxTicket
	useToken   bool // If true, use API token authentication (no ticket needed)
}

// ProxmoxTicket represents a Proxmox authentication ticket
type ProxmoxTicket struct {
	Ticket string
	CSRF   string
	Expiry time.Time
}

// NewProxmoxClient creates a new Proxmox API client
func NewProxmoxClient(config *ProxmoxConfig) (*ProxmoxClient, error) {
	// Validate that either password or token is provided
	if config.Password == "" && (config.TokenID == "" || config.Secret == "") {
		return nil, fmt.Errorf("either password or token (token_id + secret) must be provided")
	}

	// Create HTTP client with insecure TLS (Proxmox often uses self-signed certs)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // TODO: Make this configurable
		},
	}

	useToken := config.TokenID != "" && config.Secret != ""

	client := &ProxmoxClient{
		config: config,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		useToken: useToken,
	}

	// Authenticate (only needed for password-based auth; tokens are used directly in requests)
	if !useToken {
		if err := client.authenticate(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to authenticate with Proxmox: %w", err)
		}
	} else {
		logger.Info("[ProxmoxClient] Using API token authentication (no ticket needed)")
	}

	return client, nil
}

// authenticate authenticates with Proxmox API and obtains a ticket (password-based only)
func (pc *ProxmoxClient) authenticate(ctx context.Context) error {
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	authURL := fmt.Sprintf("%s/api2/json/access/ticket", apiURL)

	// Password-based authentication only (tokens don't use tickets)
	authData := url.Values{}
	authData.Set("username", pc.config.Username)
	authData.Set("password", pc.config.Password)

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(authData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var authResp struct {
		Data struct {
			Ticket string `json:"ticket"`
			CSRF   string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	pc.ticket = &ProxmoxTicket{
		Ticket: authResp.Data.Ticket,
		CSRF:   authResp.Data.CSRF,
		Expiry: time.Now().Add(2 * time.Hour), // Proxmox tickets typically last 2 hours
	}

	logger.Info("[ProxmoxClient] Successfully authenticated with Proxmox API (password-based)")
	return nil
}

// ensureAuthenticated ensures we have a valid ticket (only for password-based auth)
func (pc *ProxmoxClient) ensureAuthenticated(ctx context.Context) error {
	if pc.useToken {
		// API tokens don't need tickets - they're used directly in requests
		return nil
	}
	if pc.ticket == nil || time.Now().After(pc.ticket.Expiry.Add(-5*time.Minute)) {
		// Ticket expired or about to expire, re-authenticate
		return pc.authenticate(ctx)
	}
	return nil
}

// apiRequest makes an authenticated request to Proxmox API
func (pc *ProxmoxClient) apiRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if pc.useToken {
		// API token authentication: Use Authorization header
		// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
		req.Header.Set("Authorization", authHeader)
		// API tokens don't need CSRF tokens
	} else {
		// Password-based authentication: Use ticket cookie
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})

		// Set CSRF token for write operations
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	return pc.httpClient.Do(req)
}

// APIRequestRaw makes an authenticated request to Proxmox API with raw JSON body
func (pc *ProxmoxClient) APIRequestRaw(ctx context.Context, method, endpoint string, bodyJSON []byte) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if pc.useToken {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret))
	} else {
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// apiRequestForm makes an authenticated request to Proxmox API with form-encoded data
func (pc *ProxmoxClient) apiRequestForm(ctx context.Context, method, endpoint string, formData url.Values) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)

	var body io.Reader
	if len(formData) > 0 {
		body = strings.NewReader(formData.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if pc.useToken {
		// API token authentication: Use Authorization header
		// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
		req.Header.Set("Authorization", authHeader)
		// API tokens don't need CSRF tokens
	} else {
		// Password-based authentication: Use ticket cookie
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: pc.ticket.Ticket,
		})

		// Set CSRF token for write operations
		if method != "GET" {
			req.Header.Set("CSRFPreventionToken", pc.ticket.CSRF)
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return pc.httpClient.Do(req)
}

// CreateVM creates a new VM in Proxmox with cloud-init support
// allowInterVM: if true, allows VMs in the same organization to communicate with each other
func (pc *ProxmoxClient) CreateVM(ctx context.Context, config *VPSConfig, allowInterVM bool) (string, error) {
	// Get next available VM ID
	vmID, err := pc.getNextVMID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get next VM ID: %w", err)
	}

	// Select node (use first available for now)
	nodes, err := pc.ListNodes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list Proxmox nodes: %w", err)
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("no Proxmox nodes available")
	}
	nodeName := nodes[0]

	// Get storage pool (default to local-lvm)
	storage := "local-lvm"
	if storagePool := os.Getenv("PROXMOX_STORAGE_POOL"); storagePool != "" {
		storage = storagePool
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
			return "", fmt.Errorf("storage pool '%s' does not exist on node '%s'. Available storage pools: %v. Please set PROXMOX_STORAGE_POOL to one of the available pools or create the storage pool in Proxmox", storage, nodeName, availableStorages)
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
		"description": fmt.Sprintf("Managed by Obiente Cloud - VPS ID: %s, Display Name: %s", config.VPSID, config.Name),
	}

	// Configure network interface with optional VLAN support
	// SECURITY: Use VLAN tags for network isolation when configured
	netConfig := "virtio,bridge=vmbr0,firewall=1"
	if vlanID := os.Getenv("PROXMOX_VLAN_ID"); vlanID != "" {
		// Add VLAN tag for network isolation
		netConfig = fmt.Sprintf("virtio,bridge=vmbr0,tag=%s,firewall=1", vlanID)
		logger.Info("[ProxmoxClient] Configuring VM network with VLAN tag: %s", vlanID)
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
			cloneFormData.Set("full", "0") // Linked clone (faster)
			// Note: storage parameter is only allowed for full clones, not linked clones
			// Linked clones use the same storage as the template
			logger.Info("[ProxmoxClient] Cloning template %s (VMID %d) to VM %d (linked clone)", imageTemplate, templateVMID, vmID)

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

					// Get template config to see what disk type it uses
					templateConfig, err := pc.GetVMConfig(ctx, nodeName, templateVMID)
					var templateDiskKey string
					var templateDiskValue string
					if err == nil {
						// Find the disk key used by the template
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
							// Check storage for template disk volumes
							storageContentEndpoint := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage)
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
								return "", fmt.Errorf("template %s (VMID %d) does not have a disk configured - cannot clone VM without disk. Please configure a disk for the template first", imageTemplate, templateVMID)
							}
						}
					} else {
						logger.Error("[ProxmoxClient] Failed to get template config: %v", err)
						return "", fmt.Errorf("failed to get template config: %w", err)
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

					// Resize disk after cloning (only if disk already existed, not if we just created it)
					// If we created a new disk, it's already the correct size, so skip resize
					if actualDiskKey != "" {
						// Check if we just created the disk (if so, it's already the right size)
						vmConfigAfter, err := pc.GetVMConfig(ctx, nodeName, vmID)
						if err == nil {
							if disk, ok := vmConfigAfter[actualDiskKey].(string); ok && disk != "" {
								// If disk value contains "size=", we just created it, so skip resize
								if strings.Contains(disk, "size=") {
									logger.Info("[ProxmoxClient] Disk %s was just created with correct size, skipping resize", actualDiskKey)
								} else {
									// Disk exists but may need resizing
									diskSizeGB := config.DiskBytes / (1024 * 1024 * 1024)
									if err := pc.resizeDisk(ctx, nodeName, vmID, actualDiskKey, diskSizeGB); err != nil {
										logger.Warn("[ProxmoxClient] Failed to resize disk %s for VM %d: %v", actualDiskKey, vmID, err)
										// Continue anyway - VM is created, disk can be resized manually
									} else {
										logger.Info("[ProxmoxClient] Resized disk %s for VM %d to %dGB", actualDiskKey, vmID, diskSizeGB)
									}
								}
							}
						}
					}
				} else {
					body, _ := io.ReadAll(resp.Body)
					logger.Warn("[ProxmoxClient] Failed to clone template (status %d): %s, falling back to ISO", resp.StatusCode, string(body))
					useCloudInit = false
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
		vmConfig["cipassword"] = generateRandomPassword(16) // Generate random root password

		// Add SSH key if provided
		if config.SSHKeyID != nil {
			// TODO: Fetch SSH key from database and add to cloud-init
			// For now, we'll add a placeholder
			vmConfig["sshkeys"] = "# SSH key will be added via cloud-init user data"
		}

		// Note: cicustom requires a volume reference (e.g., "user=local:snippets/user-data")
		// not raw base64 data. For now, we use standard Proxmox cloud-init parameters.
		// Package updates and installations can be handled post-provision if needed.
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
			formData.Set(key, fmt.Sprintf("%v", value))
		}

		// Ensure boot order includes the actual disk - this is critical for the VM to boot
		// Use the disk key we detected from the cloned VM or template
		formData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
		formData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
		logger.Info("[ProxmoxClient] Setting boot order to %s and bootdisk to %s for VM %d", actualDiskKey, actualDiskKey, vmID)

		resp, err := pc.apiRequestForm(ctx, "PUT", updateEndpoint, formData)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			logger.Info("[ProxmoxClient] Updated VM %d configuration", vmID)

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
			body, _ := io.ReadAll(resp.Body)
			logger.Warn("[ProxmoxClient] Initial config update failed for VM %d: %v. Response: %s. Retrying with cloud-init config only...", vmID, err, string(body))

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
			retryFormData.Set("cipassword", vmConfig["cipassword"].(string))
			if sshKeys, ok := vmConfig["sshkeys"]; ok && sshKeys != "" {
				retryFormData.Set("sshkeys", fmt.Sprintf("%v", sshKeys))
			}
			retryFormData.Set("boot", fmt.Sprintf("order=%s", actualDiskKey))
			retryFormData.Set("bootdisk", actualDiskKey) // Set bootdisk parameter (required for Proxmox)
			logger.Info("[ProxmoxClient] Retrying with boot order %s and bootdisk %s for VM %d", actualDiskKey, actualDiskKey, vmID)

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
			return "", fmt.Errorf("failed to create VM: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorMsg := string(body)
			if resp.StatusCode == 403 {
				return "", fmt.Errorf("failed to create VM: permission denied (status: %d). The API token needs VM.Allocate, VM.Config.Disk, and Datastore.Allocate permissions. Error: %s", resp.StatusCode, errorMsg)
			}
			if resp.StatusCode == 500 && strings.Contains(errorMsg, "storage") {
				// Try to get available storages for better error message
				availableStorages, listErr := pc.listStorages(ctx, nodeName)
				if listErr == nil && len(availableStorages) > 0 {
					return "", fmt.Errorf("failed to create VM: storage error (status: %d). Error: %s. Available storage pools on node '%s': %v", resp.StatusCode, errorMsg, nodeName, availableStorages)
				}
			}
			return "", fmt.Errorf("failed to create VM: %s (status: %d)", errorMsg, resp.StatusCode)
		}
	}

	logger.Info("[ProxmoxClient] Created VM %d on node %s", vmID, nodeName)

	// Configure firewall rules for inter-VM communication
	if err := pc.configureVMFirewall(ctx, nodeName, vmID, config.OrganizationID, allowInterVM); err != nil {
		logger.Warn("[ProxmoxClient] Failed to configure firewall for VM %d: %v", vmID, err)
		// Continue anyway - VM is created, firewall can be configured manually
	}

	// Start the VM
	if err := pc.startVM(ctx, nodeName, vmID); err != nil {
		logger.Warn("[ProxmoxClient] Failed to start VM %d: %v", vmID, err)
		// Continue anyway - VM is created
	}

	return fmt.Sprintf("%d", vmID), nil
}

// findTemplate finds a template VM by name pattern
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

// generateCloudInitUserData generates cloud-init user data
func generateCloudInitUserData(config *VPSConfig) string {
	userData := "#cloud-config\n"
	userData += "users:\n"
	userData += "  - name: root\n"
	userData += "    ssh_authorized_keys:\n"
	// TODO: Add SSH keys from config.SSHKeyID
	userData += "    sudo: ALL=(ALL) NOPASSWD:ALL\n"
	userData += "package_update: true\n"
	userData += "package_upgrade: true\n"
	userData += "packages:\n"
	userData += "  - curl\n"
	userData += "  - wget\n"
	userData += "  - htop\n"
	return userData
}

// generateRandomPassword generates a random password
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, _ := rand.Int(rand.Reader, charsetLen)
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// getNextVMID gets the next available VM ID from Proxmox
// If PROXMOX_VM_ID_START is set, uses that as the starting range and finds the next available ID
// Otherwise, uses Proxmox's auto-generated next ID
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

// getAllVMIDs gets all existing VM IDs from all nodes
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

// listNodes lists available Proxmox nodes
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

// getStorageInfo gets information about a specific storage pool
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

// listStorages lists available storage pools on a specific node
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

// startVM starts a VM
func (pc *ProxmoxClient) startVM(ctx context.Context, nodeName string, vmID int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", nodeName, vmID)
	// Proxmox API expects form-encoded data for POST requests, even if empty
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start VM: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// StopVM stops a VM (force stop - immediate shutdown, not graceful)
// This uses the /status/stop endpoint which forces an immediate shutdown
// For graceful shutdown, use /status/shutdown instead
func (pc *ProxmoxClient) StopVM(ctx context.Context, nodeName string, vmID int) error {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", nodeName, vmID)
	// Proxmox API expects form-encoded data for POST requests, even if empty
	// /status/stop forces an immediate shutdown (not graceful)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop VM: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// getMapKeys returns all keys from a map as a slice of strings (for logging)
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// waitForVMStatus waits for a VM to reach a specific status with a timeout
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

// DeleteVM deletes a VM
// SECURITY: Verifies VM was created by our API by checking if VM name matches VPS ID
func (pc *ProxmoxClient) DeleteVM(ctx context.Context, nodeName string, vmID int, vpsID string) error {
	// SECURITY: Verify VM was created by our API before deletion
	// Get VM config to check VM name matches VPS ID
	configEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", configEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to get VM config for validation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errorMsg := string(body)

		// If VM config doesn't exist, the VM is already deleted
		if resp.StatusCode == 500 && strings.Contains(errorMsg, "does not exist") {
			logger.Info("[ProxmoxClient] VM %d config does not exist - VM is already deleted", vmID)
			return nil // VM already deleted, nothing to do
		}

		return fmt.Errorf("failed to get VM config: %s (status: %d)", errorMsg, resp.StatusCode)
	}

	var configResp struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return fmt.Errorf("failed to decode VM config: %w", err)
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

// GetVMStatus retrieves the current status of a VM
func (pc *ProxmoxClient) GetVMStatus(ctx context.Context, nodeName string, vmID int) (string, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get VM status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get VM status: %s (status: %d)", string(body), resp.StatusCode)
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

// GetVMMetrics retrieves current VM metrics (CPU, memory, disk) from Proxmox
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

// GetVMDiskSize retrieves the disk size from VM config or storage
// Returns the disk size in bytes, or 0 if not found
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

// resizeDisk resizes a VM disk to the specified size in GB
// disk: disk identifier (e.g., "scsi0", "virtio0")
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

// GetVMConfig retrieves the VM configuration from Proxmox
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

// GetVMIPAddresses retrieves IP addresses of a VM via QEMU guest agent
func (pc *ProxmoxClient) GetVMIPAddresses(ctx context.Context, nodeName string, vmID int) ([]string, []string, error) {
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

// RebootVM reboots a VM
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
		return fmt.Errorf("failed to reboot VM: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// GetVMConsole retrieves console output from a VM
func (pc *ProxmoxClient) GetVMConsole(ctx context.Context, nodeName string, vmID int, limit int32) ([]string, error) {
	// Proxmox doesn't have a direct console log API, but we can get VNC console info
	// For actual console output, we'd need to use VNC or serial console
	// This is a placeholder that returns VNC connection info
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	resp, err := pc.apiRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get console: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get console: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string `json:"ticket"`
			Port   int    `json:"port"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return nil, fmt.Errorf("failed to decode console response: %w", err)
	}

	// Return VNC connection info as console output
	lines := []string{
		fmt.Sprintf("VNC Console for VM %d:", vmID),
		fmt.Sprintf("Port: %d", vncResp.Data.Port),
		fmt.Sprintf("Ticket: %s", vncResp.Data.Ticket),
		"Use a VNC client to connect to the console.",
	}

	return lines, nil
}

// GetVNCWebSocketURL returns the VNC WebSocket URL for terminal access
// This provides full terminal access including boot output, similar to Proxmox web UI
func (pc *ProxmoxClient) GetVNCWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, string, error) {
	// First, get VNC proxy ticket
	// Proxmox API expects form-encoded data for POST requests, even if empty
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get VNC proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get VNC proxy: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			UPID   string      `json:"upid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return "", "", fmt.Errorf("failed to decode VNC proxy response: %w", err)
	}

	// Convert port to int (handle both string and int from API)
	var port int
	switch v := vncResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return "", "", fmt.Errorf("failed to parse port as integer: %v", vncResp.Data.Port)
		}
	default:
		return "", "", fmt.Errorf("unexpected port type: %T", vncResp.Data.Port)
	}

	// Construct WebSocket URL
	// Proxmox VNC WebSocket endpoint: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?port=%d&vncticket=%s", apiURL, nodeName, vmID, port, url.QueryEscape(vncResp.Data.Ticket))

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	return wsURL, vncResp.Data.Ticket, nil
}

// GetSerialConsoleWebSocketURL returns the serial console WebSocket URL for terminal access
// Serial console provides text-based terminal access including boot output
func (pc *ProxmoxClient) GetSerialConsoleWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, error) {
	// Proxmox serial console uses the same VNC WebSocket endpoint but with serial=1 parameter
	// We still need to get a VNC ticket for authentication
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return "", fmt.Errorf("failed to get VNC proxy for serial console: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get VNC proxy for serial console: %s (status: %d)", string(body), resp.StatusCode)
	}

	var vncResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			UPID   string      `json:"upid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vncResp); err != nil {
		return "", fmt.Errorf("failed to decode VNC proxy response: %w", err)
	}

	// Construct Serial Console WebSocket URL
	// Format: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket?port={port}&vncticket={ticket}&serial=1
	// Note: serial=1 must be last parameter, and we need the port even for serial console
	var port int
	switch v := vncResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return "", fmt.Errorf("failed to parse port as integer: %v", vncResp.Data.Port)
		}
	default:
		return "", fmt.Errorf("unexpected port type: %T", vncResp.Data.Port)
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	// Try different parameter orders - Proxmox might be sensitive to parameter order
	// Format 1: port, vncticket, serial
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?port=%d&vncticket=%s&serial=1", apiURL, nodeName, vmID, port, url.QueryEscape(vncResp.Data.Ticket))

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	return wsURL, nil
}

// configureVMFirewall configures firewall rules for a VM to control inter-VM communication
// If allowInterVM is false, blocks all inter-VM communication by default
// If allowInterVM is true, adds VM to a security group that allows communication within the organization
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
		// Strategy: Add a firewall rule that blocks traffic from other VMs on the same bridge
		// We'll block traffic from the bridge interface that originates from other VMs
		// Note: This is a simplified approach. In production, you might want to:
		// 1. Use Proxmox firewall aliases to track VM IPs
		// 2. Create rules that specifically block traffic from other VMs
		// 3. Allow established/related connections to maintain existing sessions

		// Get bridge name (default to vmbr0)
		bridgeName := "vmbr0"

		// Add firewall rule to block inter-VM traffic
		// Rule: Block incoming traffic from other VMs on the same bridge
		// We'll use a rule that blocks traffic from the bridge interface
		// This blocks traffic from other VMs while allowing gateway/internet traffic
		ruleEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", nodeName, vmID)
		ruleData := url.Values{}
		ruleData.Set("enable", "1")
		ruleData.Set("action", "REJECT")
		ruleData.Set("type", "in")
		ruleData.Set("iface", bridgeName)
		ruleData.Set("comment", "Block inter-VM communication (default security)")
		// Note: This is a basic rule. For production, you would:
		// - Use firewall aliases to track VM IPs
		// - Block specific source IPs or subnets
		// - Allow established/related connections

		ruleResp, err := pc.apiRequestForm(ctx, "POST", ruleEndpoint, ruleData)
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

// ensureSecurityGroup ensures a security group exists in Proxmox
// Security groups in Proxmox are managed via firewall aliases and groups
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

// Firewall management methods

// ListFirewallRules lists all firewall rules for a VM
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

// GetFirewallRule gets a specific firewall rule by position
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

// CreateFirewallRule creates a new firewall rule
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

// UpdateFirewallRule updates an existing firewall rule
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

// DeleteFirewallRule deletes a firewall rule
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

// GetFirewallOptions gets firewall options for a VM
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

// UpdateFirewallOptions updates firewall options for a VM
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
