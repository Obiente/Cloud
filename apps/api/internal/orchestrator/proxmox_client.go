package orchestrator

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/logger"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
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

// GetAuthCookie returns the authentication cookie value for WebSocket connections
// For password-based auth, returns the ticket cookie.
// For API tokens, we may need to get a ticket first for WebSocket connections.
// Returns empty string if no cookie is available.
func (pc *ProxmoxClient) GetAuthCookie() string {
	if pc.ticket != nil {
		return pc.ticket.Ticket
	}
	// For API tokens, WebSocket connections may require a ticket cookie
	// Try to get a ticket using the API token
	if pc.useToken {
		// API tokens can be used to get a ticket for WebSocket connections
		// This is a workaround for WebSocket which requires PVEAuthCookie
		return ""
	}
	return ""
}

// GetOrCreateTicketForWebSocket gets or creates a ticket for WebSocket connections
// WebSocket connections require PVEAuthCookie even when using API tokens
func (pc *ProxmoxClient) GetOrCreateTicketForWebSocket(ctx context.Context) (string, error) {
	// If we already have a ticket, return it
	if pc.ticket != nil && time.Now().Before(pc.ticket.Expiry.Add(-5*time.Minute)) {
		return pc.ticket.Ticket, nil
	}
	
	// For API tokens, we cannot get a ticket via /access/ticket endpoint
	// The endpoint requires username/password in POST body, not API token in header
	// According to Proxmox docs, API tokens don't use tickets for regular API calls
	// However, WebSocket may require PVEAuthCookie - this is a limitation
	// We'll return empty and try with just Authorization header + vncticket
	if pc.useToken {
		// API tokens cannot obtain tickets - return empty
		// WebSocket connection will use Authorization header instead
		return "", nil
	}
	
	// For password-based auth, ensure we're authenticated
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return "", err
	}
	
	if pc.ticket == nil {
		return "", fmt.Errorf("no ticket available")
	}
	
	return pc.ticket.Ticket, nil
}

// GetHTTPClient returns the HTTP client used by ProxmoxClient
// This allows reusing the same client (with transport, cookies, etc.) for WebSocket connections
func (pc *ProxmoxClient) GetHTTPClient() *http.Client {
	return pc.httpClient
}

// GetAuthHeader returns the Authorization header value for API token authentication
// Returns empty string if using password-based auth (which uses cookies instead)
func (pc *ProxmoxClient) GetAuthHeader() string {
	if !pc.useToken {
		return ""
	}
	return fmt.Sprintf("PVEAPIToken=%s!%s=%s", pc.config.Username, pc.config.TokenID, pc.config.Secret)
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
// Special handling for sshkeys parameter to avoid double encoding
func (pc *ProxmoxClient) apiRequestForm(ctx context.Context, method, endpoint string, formData url.Values) (*http.Response, error) {
	if err := pc.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	reqURL := fmt.Sprintf("%s/api2/json%s", apiURL, endpoint)

	var body io.Reader
	if len(formData) > 0 {
		// Check if sshkeys is pre-encoded (we manually encoded it with %20)
		// If so, manually construct form data to avoid double encoding
		if sshKeysVal, ok := formData["sshkeys"]; ok && len(sshKeysVal) > 0 {
			// sshkeys is already URL-encoded with %20, manually construct form data
			var formParts []string
			for key, values := range formData {
				if key == "sshkeys" {
					// sshkeys is already encoded with %20, use as-is
					formParts = append(formParts, fmt.Sprintf("%s=%s", url.QueryEscape(key), sshKeysVal[0]))
				} else {
					// Other parameters: use normal form encoding
					for _, value := range values {
						tempForm := url.Values{}
						tempForm.Set(key, value)
						encoded := tempForm.Encode()
						formParts = append(formParts, encoded)
					}
				}
			}
			bodyStr := strings.Join(formParts, "&")
			logger.Debug("[ProxmoxClient] Form data body: %s", bodyStr)
			body = strings.NewReader(bodyStr)
		} else {
			// No sshkeys parameter, use standard form encoding
			encodedBody := formData.Encode()
			logger.Debug("[ProxmoxClient] Form data body: %s", encodedBody)
			body = strings.NewReader(encodedBody)
		}
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

	// Configure network interface
	// If VPS_GATEWAY_URL is set, use the gateway bridge (typically vmbr1 or custom bridge)
	// Otherwise, use the default bridge (vmbr0)
	bridge := "vmbr0"
	if os.Getenv("VPS_GATEWAY_URL") != "" {
		// Gateway manages DHCP on a separate bridge
		gatewayBridge := os.Getenv("VPS_GATEWAY_BRIDGE")
		if gatewayBridge == "" {
			gatewayBridge = "vmbr1" // Default gateway bridge
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

		// Add SSH keys from organization
		// Proxmox's sshkeys parameter expects raw SSH public keys separated by newlines
		// Proxmox will handle the encoding internally when processing the form data
		if config.OrganizationID != "" {
			sshKeys, err := database.GetSSHKeysForOrganization(config.OrganizationID)
			if err == nil && len(sshKeys) > 0 {
				var sshKeysStr strings.Builder
				keyCount := 0
				for _, key := range sshKeys {
					// Trim leading/trailing whitespace (SSH keys should be single-line)
					trimmedKey := strings.TrimSpace(key.PublicKey)
					// Remove any newlines/carriage returns from the key itself (keys should be single-line)
					trimmedKey = strings.ReplaceAll(trimmedKey, "\n", "")
					trimmedKey = strings.ReplaceAll(trimmedKey, "\r", "")
					if trimmedKey == "" {
						continue // Skip empty keys
					}
					if keyCount > 0 {
						sshKeysStr.WriteString("\n")
					}
					// Use raw SSH public key - Proxmox expects newline-separated raw keys
					sshKeysStr.WriteString(trimmedKey)
					keyCount++
				}
				// Get the final string and ensure no trailing newline
				sshKeysValue := sshKeysStr.String()
				// Remove any trailing newlines (there shouldn't be any, but be safe)
				sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r\n")
				sshKeysValue = strings.TrimSuffix(sshKeysValue, "\n")
				sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r")
				vmConfig["sshkeys"] = sshKeysValue
				// Debug: check the last character
				lastChar := ""
				if len(sshKeysValue) > 0 {
					lastChar = string(sshKeysValue[len(sshKeysValue)-1])
				}
				logger.Debug("[ProxmoxClient] SSH keys value length: %d, ends with newline: %v, last char: %q", len(sshKeysValue), strings.HasSuffix(sshKeysValue, "\n"), lastChar)
				logger.Info("[ProxmoxClient] Adding %d SSH key(s) to cloud-init for VM %d (org: %s)", len(sshKeys), vmID, config.OrganizationID)
				sshKeysPreview := sshKeysValue
				if len(sshKeysPreview) > 100 {
					sshKeysPreview = sshKeysPreview[:100] + "..."
				}
				logger.Debug("[ProxmoxClient] SSH keys content (preview): %s", sshKeysPreview)
			} else if err != nil {
				logger.Warn("[ProxmoxClient] Failed to fetch SSH keys for organization %s: %v", config.OrganizationID, err)
			} else {
				logger.Info("[ProxmoxClient] No SSH keys found for organization %s", config.OrganizationID)
			}
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
			// Special handling for sshkeys - ensure no trailing newlines
			if key == "sshkeys" {
				if strValue, ok := value.(string); ok && strValue != "" {
					// Final check: remove any trailing newlines (shouldn't be any)
					cleanValue := strings.TrimSuffix(strings.TrimSuffix(strValue, "\r\n"), "\n")
					cleanValue = strings.TrimSuffix(cleanValue, "\r")
					cleanValue = strings.TrimRight(cleanValue, " \t\n\r")
					// Set raw value - let formData.Encode() handle encoding
					formData.Set(key, cleanValue)
					logger.Debug("[ProxmoxClient] Setting sshkeys parameter (length: %d, ends with newline: %v)", len(cleanValue), strings.HasSuffix(cleanValue, "\n"))
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
	
	// Add SSH keys from organization
	if config.OrganizationID != "" {
		sshKeys, err := database.GetSSHKeysForOrganization(config.OrganizationID)
		if err == nil {
			for _, key := range sshKeys {
				// Add each SSH key (cloud-init expects one key per line with proper indentation)
				userData += fmt.Sprintf("      - %s\n", strings.TrimSpace(key.PublicKey))
			}
		}
	}
	
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

// FindVMNode finds which node a VM is running on by checking all nodes
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
			logger.Warn("[ProxmoxClient] Failed to decode VMs on node %s: %v", nodeName, err)
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

// GetVMSSHKeys retrieves existing SSH keys from Proxmox VM config
// Returns the raw sshkeys value from Proxmox (may be URL-encoded or base64-encoded)
func (pc *ProxmoxClient) GetVMSSHKeys(ctx context.Context, nodeName string, vmID int) (string, error) {
	vmConfig, err := pc.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return "", fmt.Errorf("failed to get VM config: %w", err)
	}
	
	if sshKeysRaw, ok := vmConfig["sshkeys"].(string); ok && sshKeysRaw != "" {
		return sshKeysRaw, nil
	}
	
	return "", nil // No SSH keys configured
}

// SeedSSHKeysFromProxmox parses SSH keys from Proxmox config and syncs them with the database.
// Proxmox is the source of truth: keys in Proxmox are seeded to DB, keys not in Proxmox are deleted from DB.
func (pc *ProxmoxClient) SeedSSHKeysFromProxmox(ctx context.Context, sshKeysRaw string, organizationID string, vpsID string) error {
	// Build a map of fingerprints that exist in Proxmox
	proxmoxFingerprints := make(map[string]bool)
	seededCount := 0
	deletedCount := 0
	
	// If Proxmox has keys, parse them
	if sshKeysRaw != "" {
		// URL-decode the value (Proxmox stores it URL-encoded)
		decoded, err := url.QueryUnescape(sshKeysRaw)
		if err != nil {
			// If decoding fails, try using it as-is (might already be decoded)
			decoded = sshKeysRaw
			logger.Debug("[ProxmoxClient] Failed to URL-decode sshkeys, using as-is: %v", err)
		}
		
		// Split by newlines to get individual keys
		keyLines := strings.Split(decoded, "\n")
		
		for _, keyLine := range keyLines {
			// Clean the key line
			keyLine = strings.TrimSpace(keyLine)
			keyLine = strings.ReplaceAll(keyLine, "\r", "")
			if keyLine == "" {
				continue
			}
			
			// Parse the SSH key to validate it and get fingerprint
			parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyLine))
			if err != nil {
				logger.Debug("[ProxmoxClient] Failed to parse SSH key from Proxmox (skipping): %v", err)
				continue
			}
			
			// Calculate fingerprint
			fingerprint := ssh.FingerprintSHA256(parsedKey)
			
			// Track this fingerprint as existing in Proxmox
			proxmoxFingerprints[fingerprint] = true
			
			// Extract comment from key if available (for name matching)
			_, comment, _, _, _ := ssh.ParseAuthorizedKey([]byte(keyLine))
			
			// Check if key already exists in database - check both VPS-specific and org-wide
			// We need to find the key that matches the scope we're seeding for
			var existingKey database.SSHKey
			var foundKey bool
			
			if vpsID != "" {
				// Seeding for a specific VPS - first check for VPS-specific key
				err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id = ?", organizationID, fingerprint, vpsID).First(&existingKey).Error
				if err == nil {
					foundKey = true
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					// VPS-specific key doesn't exist - check for org-wide key
					err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id IS NULL", organizationID, fingerprint).First(&existingKey).Error
					if err == nil {
						foundKey = true
						// Found org-wide key - don't update its name from VPS seeding
						// The org-wide key should keep its own name
						logger.Debug("[ProxmoxClient] Key with fingerprint %s exists as org-wide key %s - skipping name update (VPS-specific seeding)", fingerprint, existingKey.ID)
					}
				}
			} else {
				// Seeding for org-wide - only check for org-wide key
				err = database.DB.Where("organization_id = ? AND fingerprint = ? AND vps_id IS NULL", organizationID, fingerprint).First(&existingKey).Error
				if err == nil {
					foundKey = true
				}
			}
			
			if foundKey {
				// Key exists - update name only if it matches the scope
				// Don't update org-wide key name when seeding from VPS-specific context
				shouldUpdateName := true
				if vpsID != "" && existingKey.VPSID == nil {
					// We're seeding for a VPS, but found an org-wide key
					// Don't update the org-wide key's name - it should keep its own name
					shouldUpdateName = false
				}
				
				if shouldUpdateName && comment != "" {
					// Proxmox has a comment - use it as the name (remove "Imported: " prefix if present)
					oldName := existingKey.Name
					needsUpdate := false
					
					if strings.HasPrefix(existingKey.Name, "Imported: ") {
						// If current name starts with "Imported: ", compare without that prefix
						currentNameWithoutPrefix := strings.TrimPrefix(existingKey.Name, "Imported: ")
						if currentNameWithoutPrefix != comment {
							needsUpdate = true
						}
					} else if existingKey.Name != comment {
						needsUpdate = true
					}
					
					if needsUpdate {
						// Name in Proxmox differs from DB - update DB to match Proxmox
						existingKey.Name = comment
						if err := database.DB.Save(&existingKey).Error; err != nil {
							logger.Warn("[ProxmoxClient] Failed to update SSH key name from Proxmox comment: %v", err)
						} else {
							logger.Info("[ProxmoxClient] Updated SSH key %s name from '%s' to '%s' (from Proxmox comment)", existingKey.ID, oldName, comment)
						}
					}
				}
				// Key exists, skip seeding
				continue
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				// Unexpected error
				logger.Warn("[ProxmoxClient] Error checking for existing SSH key: %v", err)
				continue
			}
			
			// Key doesn't exist in database - seed it
			// Comment was already extracted above
			
			// Generate a name for the key (use comment if available, otherwise use fingerprint)
			seedName := "Imported from Proxmox"
			if comment != "" {
				seedName = fmt.Sprintf("Imported: %s", comment)
			}
			
			keyID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())
			var vpsIDPtr *string
			if vpsID != "" {
				vpsIDPtr = &vpsID
			}
			
			sshKey := database.SSHKey{
				ID:             keyID,
				OrganizationID: organizationID,
				VPSID:          vpsIDPtr,
				Name:           seedName,
				PublicKey:      keyLine,
				Fingerprint:    fingerprint,
			}
			
			if err := database.DB.Create(&sshKey).Error; err != nil {
				logger.Warn("[ProxmoxClient] Failed to seed SSH key to database: %v", err)
				continue
			}
			
			// Create audit log entry for seeded key (system action)
			go createSeededKeyAuditLog(organizationID, vpsID, keyID, fingerprint)
			
			seededCount++
			logger.Info("[ProxmoxClient] Seeded SSH key %s from Proxmox to database (fingerprint: %s)", keyID, fingerprint)
		}
		
		if seededCount > 0 {
			logger.Info("[ProxmoxClient] Seeded %d SSH key(s) from Proxmox to database", seededCount)
		}
	}
	
	// Delete keys from database that are NOT in Proxmox (Proxmox is the source of truth)
	// Get all keys for this organization/VPS from database
	var dbKeys []database.SSHKey
	query := database.DB.Where("organization_id = ?", organizationID)
	if vpsID != "" {
		query = query.Where("vps_id = ? OR vps_id IS NULL", vpsID)
	} else {
		query = query.Where("vps_id IS NULL")
	}
	if err := query.Find(&dbKeys).Error; err != nil {
		logger.Warn("[ProxmoxClient] Failed to fetch keys from database for cleanup: %v", err)
	} else {
		// Check each DB key - if it's not in Proxmox, delete it
		for _, dbKey := range dbKeys {
			if !proxmoxFingerprints[dbKey.Fingerprint] {
				// Key exists in DB but not in Proxmox - delete it
				if err := database.DB.Delete(&dbKey).Error; err != nil {
					logger.Warn("[ProxmoxClient] Failed to delete key %s from database (not in Proxmox): %v", dbKey.ID, err)
				} else {
					deletedCount++
					logger.Info("[ProxmoxClient] Deleted SSH key %s from database (fingerprint: %s) - it no longer exists in Proxmox", dbKey.ID, dbKey.Fingerprint)
				}
			}
		}
	}
	
	if deletedCount > 0 {
		logger.Info("[ProxmoxClient] Deleted %d SSH key(s) from database that no longer exist in Proxmox", deletedCount)
	}
	
	return nil
}

// UpdateVMSSHKeys updates the SSH keys in cloud-init configuration for an existing VM
// This allows updating SSH keys even after the VM has been created
// vpsID can be empty string for org-wide updates, or a specific VPS ID for VPS-specific updates
// excludeKeyID is an optional key ID to exclude from the update (e.g., when deleting a key)
func (pc *ProxmoxClient) UpdateVMSSHKeys(ctx context.Context, nodeName string, vmID int, organizationID string, vpsID string, excludeKeyID ...string) error {
	// NOTE: We don't seed keys here because:
	// 1. If we seed before updating, deleted keys will be re-imported
	// 2. If we seed after updating, we'd be seeding the keys we just set
	// Seeding should be done separately, e.g., on VPS creation or explicit sync
	
	// Fetch SSH keys (VPS-specific + org-wide if vpsID provided, or just org-wide if empty)
	var sshKeys []database.SSHKey
	var err error
	if vpsID != "" {
		sshKeys, err = database.GetSSHKeysForVPS(organizationID, vpsID)
	} else {
		sshKeys, err = database.GetSSHKeysForOrganization(organizationID)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch SSH keys: %w", err)
	}
	
	// Exclude the specified key ID if provided (e.g., when deleting a key)
	originalKeyCount := len(sshKeys)
	if len(excludeKeyID) > 0 && excludeKeyID[0] != "" {
		filteredKeys := make([]database.SSHKey, 0, len(sshKeys))
		excludedCount := 0
		for _, key := range sshKeys {
			if key.ID != excludeKeyID[0] {
				filteredKeys = append(filteredKeys, key)
			} else {
				excludedCount++
				logger.Info("[ProxmoxClient] Excluding key %s (fingerprint: %s) from Proxmox update (key being deleted)", key.ID, key.Fingerprint)
			}
		}
		sshKeys = filteredKeys
		if excludedCount == 0 {
			logger.Warn("[ProxmoxClient] Key %s was not found in the key list to exclude - it may have already been deleted", excludeKeyID[0])
		}
		logger.Info("[ProxmoxClient] Excluding key %s: %d keys before, %d keys after exclusion", excludeKeyID[0], originalKeyCount, len(sshKeys))
	}

	// Build SSH keys string (raw keys separated by newlines)
	// Proxmox's sshkeys parameter expects raw SSH public keys separated by newlines
	// BUT: For single key, it should be just the key with NO newlines
	// IMPORTANT: If we have duplicate keys (same fingerprint) as both org-wide and VPS-specific,
	// we should deduplicate them before sending to Proxmox (Proxmox can only store one instance)
	// Prefer VPS-specific keys over org-wide keys when both exist
	seenFingerprints := make(map[string]bool)
	deduplicatedKeys := make([]database.SSHKey, 0)
	for _, key := range sshKeys {
		if !seenFingerprints[key.Fingerprint] {
			seenFingerprints[key.Fingerprint] = true
			deduplicatedKeys = append(deduplicatedKeys, key)
		} else {
			// Duplicate fingerprint - prefer VPS-specific over org-wide
			for i, existingKey := range deduplicatedKeys {
				if existingKey.Fingerprint == key.Fingerprint {
					// If the existing key is org-wide and the new one is VPS-specific, replace it
					if existingKey.VPSID == nil && key.VPSID != nil {
						deduplicatedKeys[i] = key
						logger.Debug("[ProxmoxClient] Preferring VPS-specific key %s over org-wide key %s (fingerprint: %s) for Proxmox", key.ID, existingKey.ID, key.Fingerprint)
					}
					break
				}
			}
		}
	}
	sshKeys = deduplicatedKeys
	
	var sshKeysStr strings.Builder
	keyCount := 0
	if len(sshKeys) > 0 {
		for _, key := range sshKeys {
			// Aggressively clean the key: remove ALL whitespace, newlines, carriage returns
			trimmedKey := strings.TrimSpace(key.PublicKey)
			// Remove ALL newlines and carriage returns (keys must be single-line)
			trimmedKey = strings.ReplaceAll(trimmedKey, "\n", "")
			trimmedKey = strings.ReplaceAll(trimmedKey, "\r", "")
			trimmedKey = strings.ReplaceAll(trimmedKey, "\t", "")
			// Remove any other control characters
			trimmedKey = strings.TrimSpace(trimmedKey)
			if trimmedKey == "" {
				continue // Skip empty keys
			}
			
			// Check if key already has a comment (SSH keys can have format: "key-type key-data comment")
			// If it doesn't have a comment, add the key name as a comment
			keyParts := strings.Fields(trimmedKey)
			if len(keyParts) >= 2 {
				// Key has at least type and data, check if it has a comment
				if len(keyParts) == 2 {
					// No comment, add the key name as comment
					// Clean the key name to remove any characters that might cause issues
					cleanName := strings.TrimSpace(key.Name)
					// Remove spaces and special characters that might break the key format
					cleanName = strings.ReplaceAll(cleanName, " ", "-")
					cleanName = strings.ReplaceAll(cleanName, "\n", "")
					cleanName = strings.ReplaceAll(cleanName, "\r", "")
					if cleanName != "" {
						trimmedKey = fmt.Sprintf("%s %s", trimmedKey, cleanName)
					}
				}
				// If key already has a comment (len > 2), keep it as-is
			}
			
			if keyCount > 0 {
				// Only add newline BETWEEN keys, not after the last one
				sshKeysStr.WriteString("\n")
			}
			// Use raw SSH public key with name as comment
			sshKeysStr.WriteString(trimmedKey)
			keyCount++
		}
	}
	// Get the final string and AGGRESSIVELY ensure no trailing newline
	sshKeysValue := sshKeysStr.String()
	// Multiple passes to ensure absolutely no trailing newlines
	for strings.HasSuffix(sshKeysValue, "\r\n") || strings.HasSuffix(sshKeysValue, "\n") || strings.HasSuffix(sshKeysValue, "\r") {
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r\n")
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\n")
		sshKeysValue = strings.TrimSuffix(sshKeysValue, "\r")
	}
	// Final trim of any trailing whitespace
	sshKeysValue = strings.TrimRight(sshKeysValue, " \t\n\r")
	
	// Update VM config with SSH keys
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	formData := url.Values{}
	
	if len(sshKeysValue) > 0 {
		// Clean the value: split by newlines (for multiple keys), clean each, rejoin
		// This preserves newlines BETWEEN keys but removes trailing ones
		keyLines := strings.Split(sshKeysValue, "\n")
		var cleanedLines []string
		for _, line := range keyLines {
			// Clean each line: remove carriage returns and trim
			line = strings.ReplaceAll(line, "\r", "")
			line = strings.TrimSpace(line)
			if line != "" {
				cleanedLines = append(cleanedLines, line)
			}
		}
		// Rejoin with newlines (only between keys, NOT at the end)
		cleanValue := strings.Join(cleanedLines, "\n")
		// Remove trailing newline if present
		cleanValue = strings.TrimRight(cleanValue, " \t\n\r")
		
		// Proxmox v8.4 requires sshkeys to be DOUBLE URL-encoded
		// First encode: spaces become %20, + becomes %2B, / becomes %2F
		firstEncoded := url.QueryEscape(cleanValue)
		firstEncoded = strings.ReplaceAll(firstEncoded, "+", "%20")
		// Second encode: %20 becomes %2520, %2B becomes %252B, %2F becomes %252F
		encodedValue := url.QueryEscape(firstEncoded)
		// Replace + with %20 in the double-encoded value
		encodedValue = strings.ReplaceAll(encodedValue, "+", "%20")
		
		// Verify decoded value has no newlines
		if decoded, err := url.QueryUnescape(encodedValue); err == nil {
			if strings.Contains(decoded, "\n") || strings.Contains(decoded, "\r") {
				logger.Error("[ProxmoxClient] ERROR: Decoded value contains newlines! Raw: %q, Decoded: %q", cleanValue, decoded)
			}
			// Log byte representation of decoded value
			decodedBytes := []byte(decoded)
			startIdx := len(decodedBytes) - 5
			if startIdx < 0 {
				startIdx = 0
			}
			logger.Debug("[ProxmoxClient] Decoded value byte length: %d, last 5 bytes: %v", len(decodedBytes), decodedBytes[startIdx:])
		}
		
		formData.Set("sshkeys", encodedValue)
		logger.Info("[ProxmoxClient] Updating SSH keys for VM %d (org: %s) - %d key(s)", vmID, organizationID, len(sshKeys))
		logger.Debug("[ProxmoxClient] SSH keys raw length: %d chars, encoded length: %d chars", len(cleanValue), len(encodedValue))
		logger.Debug("[ProxmoxClient] SSH keys ends with newline: %v, contains newline: %v, contains carriage return: %v", strings.HasSuffix(cleanValue, "\n"), strings.Contains(cleanValue, "\n"), strings.Contains(cleanValue, "\r"))
		// Log a preview of the actual value (first 100 chars)
		preview := cleanValue
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		logger.Debug("[ProxmoxClient] SSH keys preview (raw): %q", preview)
		logger.Debug("[ProxmoxClient] SSH keys encoded: %s", encodedValue)
	} else {
		// If no SSH keys remain after exclusion, we need to clear the sshkeys parameter
		// Don't include sshkeys in the PUT request - Proxmox should keep existing values if parameter is omitted
		// But we want to clear it, so we need to explicitly delete it
		logger.Info("[ProxmoxClient] Clearing SSH keys for VM %d (org: %s) - no keys remain after exclusion", vmID, organizationID)
		
		// Use PUT with delete=sshkeys query parameter (this is how Proxmox web UI does it)
		// PUT /nodes/{node}/qemu/{vmid}/config?delete=sshkeys
		deleteEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/config?delete=sshkeys", nodeName, vmID)
		logger.Debug("[ProxmoxClient] Attempting PUT with delete=sshkeys query parameter: %s", deleteEndpoint)
		// Use empty form data for the PUT request
		emptyFormData := url.Values{}
		deleteResp, deleteErr := pc.apiRequestForm(ctx, "PUT", deleteEndpoint, emptyFormData)
		if deleteErr == nil && deleteResp != nil {
			defer deleteResp.Body.Close()
			if deleteResp.StatusCode == http.StatusOK {
				logger.Info("[ProxmoxClient] Successfully cleared SSH keys for VM %d using PUT with delete=sshkeys", vmID)
				// Verify the deletion after a short delay
				time.Sleep(500 * time.Millisecond)
				verifyKeys, err := pc.GetVMSSHKeys(ctx, nodeName, vmID)
				if err == nil {
					if verifyKeys == "" {
						logger.Info("[ProxmoxClient] Verified: Proxmox now has no SSH keys configured for VM %d", vmID)
						return nil
					} else {
						logger.Warn("[ProxmoxClient] Verified: Proxmox still has SSH keys after PUT with delete=sshkeys, will try fallback")
					}
				}
			} else {
				body, _ := io.ReadAll(deleteResp.Body)
				logger.Debug("[ProxmoxClient] PUT with delete=sshkeys returned status %d: %s", deleteResp.StatusCode, string(body))
			}
		} else {
			if deleteErr != nil {
				logger.Debug("[ProxmoxClient] PUT with delete=sshkeys failed: %v", deleteErr)
			}
		}
		
		// Fallback: send PUT with empty sshkeys value (in case delete=sshkeys doesn't work)
		// Proxmox requires at least one parameter, so we must explicitly set sshkeys to empty string
		formData = url.Values{}
		// Set sshkeys to empty string - Proxmox should clear it when it receives an empty value
		// Double-encode as required by Proxmox API
		encodedEmpty := url.QueryEscape("")
		encodedEmpty = url.QueryEscape(encodedEmpty)
		formData.Set("sshkeys", encodedEmpty)
		logger.Info("[ProxmoxClient] PUT with delete=sshkeys didn't work, will try PUT with empty sshkeys value for VM %d", vmID)
	}

	// Use PUT as per Proxmox web UI behavior (they use PUT for config updates)
	logger.Info("[ProxmoxClient] Sending PUT request to %s to update SSH keys (excluded key: %v, sending %d keys)", endpoint, len(excludeKeyID) > 0 && excludeKeyID[0] != "", len(sshKeys))
	resp, err := pc.apiRequestForm(ctx, "PUT", endpoint, formData)
	if err != nil {
		logger.Error("[ProxmoxClient] Failed to send request to Proxmox: %v", err)
		return fmt.Errorf("failed to update SSH keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errorBody := string(body)
		logger.Error("[ProxmoxClient] Proxmox returned non-OK status %d: %s", resp.StatusCode, errorBody)
		
		// Check if this is the known Proxmox v8.4 sshkeys parsing bug
		// Even though we send valid data, Proxmox may report a false newline error
		if strings.Contains(errorBody, "invalid urlencoded string") && strings.Contains(errorBody, "sshkeys") {
			logger.Warn("[ProxmoxClient] Proxmox v8.4 sshkeys parsing error (possible bug). Error: %s", errorBody)
			logger.Warn("[ProxmoxClient] Attempting to work around by using cloudinit/regen to apply keys")
			
			// Try to work around by calling cloudinit/regen which might apply the keys anyway
			// Even though the config update failed, the keys might be in the form data and regen might pick them up
			regenEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit/regen", nodeName, vmID)
			regenResp, regenErr := pc.apiRequest(ctx, "POST", regenEndpoint, nil)
			if regenErr == nil && regenResp != nil {
				defer regenResp.Body.Close()
				if regenResp.StatusCode == http.StatusOK {
					logger.Info("[ProxmoxClient] Cloud-init regen succeeded - keys may have been applied despite config error")
					// Don't return error - regen might have worked
					return nil
				}
			}
			
			// If regen didn't work, return the original error
			return fmt.Errorf("failed to update SSH keys (Proxmox v8.4 sshkeys parsing issue): %s (status: %d)", errorBody, resp.StatusCode)
		}
		
		return fmt.Errorf("failed to update SSH keys: %s (status: %d)", errorBody, resp.StatusCode)
	}

	// After setting sshkeys, regenerate cloud-init configuration as per Proxmox docs
	// This is especially important when clearing keys (sending empty value) as Proxmox v8.4
	// might not apply the change until cloud-init is regenerated
	regenEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit/regen", nodeName, vmID)
	logger.Debug("[ProxmoxClient] Regenerating cloud-init config for VM %d to apply SSH key changes", vmID)
	regenResp, regenErr := pc.apiRequest(ctx, "POST", regenEndpoint, nil)
	if regenErr != nil {
		logger.Warn("[ProxmoxClient] Failed to regenerate cloud-init config for VM %d: %v", vmID, regenErr)
		// Don't fail the whole operation if regen fails - sshkeys might still work
	} else {
		defer regenResp.Body.Close()
		if regenResp.StatusCode != http.StatusOK {
			logger.Warn("[ProxmoxClient] Cloud-init regen returned non-OK status for VM %d: %d", vmID, regenResp.StatusCode)
		} else {
			logger.Info("[ProxmoxClient] Successfully regenerated cloud-init config for VM %d", vmID)
			// If we cleared keys (sent 0 keys), wait longer and verify again
			// Proxmox v8.4 may need more time to process the cloud-init regen
			if len(sshKeys) == 0 {
				time.Sleep(1 * time.Second) // Wait longer for Proxmox to process
				verifyKeys, err := pc.GetVMSSHKeys(ctx, nodeName, vmID)
				if err == nil {
					if verifyKeys == "" {
						logger.Info("[ProxmoxClient] Verified after regen: Proxmox now has no SSH keys configured for VM %d", vmID)
					} else {
						decodedVerify, _ := url.QueryUnescape(verifyKeys)
						verifyKeyLines := strings.Split(decodedVerify, "\n")
						verifyKeyCount := 0
						for _, line := range verifyKeyLines {
							line = strings.TrimSpace(line)
							if line != "" {
								verifyKeyCount++
							}
						}
						logger.Warn("[ProxmoxClient] Verified after regen: Proxmox still has %d SSH key(s) configured for VM %d (we sent 0) - Proxmox v8.4 may have a bug where empty sshkeys doesn't clear the parameter", verifyKeyCount, vmID)
						// Try one more time: force another cloud-init regen after a delay
						time.Sleep(1 * time.Second)
						regenResp2, regenErr2 := pc.apiRequest(ctx, "POST", regenEndpoint, nil)
						if regenErr2 == nil && regenResp2 != nil {
							regenResp2.Body.Close()
							if regenResp2.StatusCode == http.StatusOK {
								logger.Info("[ProxmoxClient] Forced second cloud-init regen for VM %d to clear SSH keys", vmID)
							}
						}
					}
				}
			}
		}
	}

	logger.Info("[ProxmoxClient] Successfully updated SSH keys for VM %d (org: %s) - %d key(s) sent to Proxmox", vmID, organizationID, len(sshKeys))
	
	// Verify the update by fetching the keys back from Proxmox
	// If Proxmox has keys that aren't in our database, we need to clear them
	// This ensures Proxmox matches our database (the source of truth)
	verifyKeys, err := pc.GetVMSSHKeys(ctx, nodeName, vmID)
	if err == nil {
		if verifyKeys == "" {
			logger.Info("[ProxmoxClient] Verified: Proxmox now has no SSH keys configured for VM %d", vmID)
		} else {
			// Parse keys from Proxmox and check if they all exist in our database
			decodedVerify, _ := url.QueryUnescape(verifyKeys)
			verifyKeyLines := strings.Split(decodedVerify, "\n")
			proxmoxKeyFingerprints := make(map[string]bool)
			for _, line := range verifyKeyLines {
				line = strings.TrimSpace(line)
				line = strings.ReplaceAll(line, "\r", "")
				if line == "" {
					continue
				}
				// Parse the key to get fingerprint
				parsedKey, _, _, _, parseErr := ssh.ParseAuthorizedKey([]byte(line))
				if parseErr == nil {
					fingerprint := ssh.FingerprintSHA256(parsedKey)
					proxmoxKeyFingerprints[fingerprint] = true
				}
			}
			
			// Check which Proxmox keys exist in our database
			expectedFingerprints := make(map[string]bool)
			for _, key := range sshKeys {
				expectedFingerprints[key.Fingerprint] = true
			}
			
			// Find keys in Proxmox that aren't in our database
			extraKeys := make([]string, 0)
			for fp := range proxmoxKeyFingerprints {
				if !expectedFingerprints[fp] {
					extraKeys = append(extraKeys, fp)
				}
			}
			
			if len(extraKeys) > 0 {
				// Proxmox still has keys that shouldn't be there - deletion failed
				// Return an error so the caller knows the deletion didn't work
				return fmt.Errorf("failed to clear SSH keys from Proxmox: Proxmox still has %d key(s) that should have been deleted (fingerprints: %v). This may be due to a Proxmox v8.4 bug where empty sshkeys parameter doesn't clear the keys", len(extraKeys), extraKeys)
			}
			
			logger.Info("[ProxmoxClient] Verified: Proxmox SSH keys match our database (%d keys)", len(sshKeys))
		}
	} else {
		logger.Warn("[ProxmoxClient] Failed to verify SSH keys in Proxmox after update: %v", err)
	}
	
	return nil
}

// createSeededKeyAuditLog creates an audit log entry for a seeded SSH key
// This is called when keys are imported from Proxmox into the database
func createSeededKeyAuditLog(organizationID string, vpsID string, keyID string, fingerprint string) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[ProxmoxClient] Panic creating audit log for seeded key: %v", r)
		}
	}()
	
	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Use MetricsDB (TimescaleDB) for audit logs
	if database.MetricsDB == nil {
		logger.Warn("[ProxmoxClient] Metrics database not available, skipping audit log for seeded key")
		return
	}
	
	// Determine resource type and ID
	var resourceType *string
	var resourceID *string
	if vpsID != "" {
		rt := "vps"
		resourceType = &rt
		resourceID = &vpsID
	} else {
		rt := "organization"
		resourceType = &rt
		resourceID = &organizationID
	}
	
	// Create audit log entry
	auditLog := database.AuditLog{
		ID:             fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:         "system", // System user for seeded keys
		OrganizationID: &organizationID,
		Action:         "SeedSSHKey",
		Service:        "VPSService",
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      "system",
		UserAgent:      "system",
		RequestData:    fmt.Sprintf(`{"key_id":"%s","fingerprint":"%s","source":"proxmox"}`, keyID, fingerprint),
		ResponseStatus: 200,
		ErrorMessage:   nil,
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}
	
	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		logger.Warn("[ProxmoxClient] Failed to create audit log for seeded key %s: %v", keyID, err)
	} else {
		logger.Debug("[ProxmoxClient] Created audit log for seeded SSH key %s", keyID)
	}
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

// TermProxyInfo contains information needed to connect to termproxy WebSocket
type TermProxyInfo struct {
	WebSocketURL string
	Ticket       string
	User         string
}

// GetTermProxyWebSocketURL returns the terminal proxy WebSocket URL and authentication info
// termproxy is the recommended endpoint for terminal access (better than vncwebsocket with serial=1)
// Reference: https://pve.proxmox.com/pve-docs-8/api-viewer/index.html#/nodes/{node}/termproxy
func (pc *ProxmoxClient) GetTermProxyWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, error) {
	info, err := pc.GetTermProxyInfo(ctx, nodeName, vmID)
	if err != nil {
		return "", err
	}
	return info.WebSocketURL, nil
}

// GetTermProxyInfo returns complete termproxy information including ticket and user
func (pc *ProxmoxClient) GetTermProxyInfo(ctx context.Context, nodeName string, vmID int) (*TermProxyInfo, error) {
	// Get terminal proxy ticket from termproxy endpoint
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/termproxy", nodeName, vmID)
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("failed to get term proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get term proxy: %s (status: %d)", string(body), resp.StatusCode)
	}

	var termResp struct {
		Data struct {
			Ticket string      `json:"ticket"`
			Port   interface{} `json:"port"` // Can be string or int
			User   string      `json:"user,omitempty"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&termResp); err != nil {
		return nil, fmt.Errorf("failed to decode term proxy response: %w", err)
	}

	// Convert port to int
	var port int
	switch v := termResp.Data.Port.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case string:
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			return nil, fmt.Errorf("failed to parse port as integer: %v", termResp.Data.Port)
		}
	default:
		return nil, fmt.Errorf("unexpected port type: %T", termResp.Data.Port)
	}

	// Construct WebSocket URL for termproxy
	// According to Proxmox API documentation, termproxy uses vncwebsocket endpoint
	// but with the termproxy ticket (not vncticket parameter name)
	// Format: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket?port={port}&vncticket={ticket}
	// Note: termproxy ticket is used as vncticket parameter
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("vncticket", termResp.Data.Ticket)
	// Note: termproxy doesn't use serial=1, it's already a terminal proxy
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?%s", apiURL, nodeName, vmID, params.Encode())

	// Convert https to wss, http to ws
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	// Get username from config if not provided in response
	user := termResp.Data.User
	if user == "" {
		// Try to get username from config
		if pc.config.Username != "" {
			user = pc.config.Username
			// For API tokens, keep the token ID in the username (format: username@realm!tokenid)
			// For password auth, use just username@realm
			if pc.config.TokenID == "" {
				// Password auth - remove token ID if present
				if idx := strings.Index(user, "!"); idx != -1 {
					user = user[:idx]
				}
			}
			// For API tokens, keep the full format including token ID
		} else {
			// Default to root@pam if no user specified
			user = "root@pam"
		}
	} else {
		// User from termproxy response - check if we need to add token ID for API tokens
		if pc.config.TokenID != "" && pc.config.Username != "" {
			// API token auth - ensure username includes token ID
			if !strings.Contains(user, "!") && strings.Contains(pc.config.Username, "!") {
				// Extract token ID from config username and add to termproxy user
				if idx := strings.Index(pc.config.Username, "!"); idx != -1 {
					tokenID := pc.config.Username[idx:]
					user = user + tokenID
				}
			}
		} else {
			// Password auth - remove token ID if present
			if idx := strings.Index(user, "!"); idx != -1 {
				user = user[:idx]
			}
		}
	}

	return &TermProxyInfo{
		WebSocketURL: wsURL,
		Ticket:       termResp.Data.Ticket,
		User:         user,
	}, nil
}

// GetSerialConsoleWebSocketURL returns the serial console WebSocket URL for terminal access
// Serial console provides text-based terminal access including boot output
// According to Proxmox API documentation:
// - Use vncproxy with websocket=1 parameter (required for serial terminal)
// - Then connect to vncwebsocket with port, vncticket, and serial=1 parameters
// - termproxy does NOT work with API tokens (documented limitation)
// Reference: https://pve.proxmox.com/pve-docs-8/api-viewer/index.html#/nodes/{node}/qemu/{vmid}/vncproxy
// Reference: https://pve.proxmox.com/pve-docs-8/api-viewer/index.html#/nodes/{node}/qemu/{vmid}/vncwebsocket
func (pc *ProxmoxClient) GetSerialConsoleWebSocketURL(ctx context.Context, nodeName string, vmID int) (string, error) {
	// Get VNC proxy with websocket=1 parameter (required for serial terminal)
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)
	formData := url.Values{}
	formData.Set("websocket", "1") // Required for serial terminal per API docs
	resp, err := pc.apiRequestForm(ctx, "POST", endpoint, formData)
	if err != nil {
		return "", fmt.Errorf("failed to get VNC proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get VNC proxy: %s (status: %d)", string(body), resp.StatusCode)
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

	// Convert port to int
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

	// Construct Serial Console WebSocket URL
	// Required parameters per API docs:
	// - node: string (in path)
	// - port: integer 5900-5999 (query parameter)
	// - vmid: integer 100-999999999 (in path)
	// - vncticket: string (query parameter)
	// Format: /api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket?port={port}&vncticket={ticket}
	// Note: When websocket=1 is used in vncproxy, the connection can be used for serial console
	// The RFB protocol handshake may appear initially but should be followed by serial console data
	apiURL := strings.TrimSuffix(pc.config.APIURL, "/")
	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("vncticket", vncResp.Data.Ticket)
	wsURL := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/vncwebsocket?%s", apiURL, nodeName, vmID, params.Encode())
	
	// Validate required parameters are present
	if nodeName == "" {
		return "", fmt.Errorf("node parameter is required")
	}
	if port < 5900 || port > 5999 {
		return "", fmt.Errorf("port must be between 5900 and 5999, got %d", port)
	}
	if vmID < 100 || vmID > 999999999 {
		return "", fmt.Errorf("vmid must be between 100 and 999999999, got %d", vmID)
	}
	if vncResp.Data.Ticket == "" {
		return "", fmt.Errorf("vncticket is required")
	}

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
