package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"github.com/moby/moby/client"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// LogWriter is an interface for writing provisioning logs
type LogWriter interface {
	WriteLine(line string, stderr bool)
}

// VPSManager manages the lifecycle of VPS instances via Proxmox
type VPSManager struct {
	dockerClient   client.APIClient
	gatewayClient  *VPSGatewayClient // Deprecated - use GetGatewayClientForNode instead
	gatewayClients sync.Map          // Cache of gateway clients per node (key: nodeName string, value: *VPSGatewayClient)
	proxmoxClients sync.Map          // Cache of Proxmox clients per node (key: nodeName string, value: *ProxmoxClient)
}

// parseNodeEndpointsMapping parses the PROXMOX_NODE_ENDPOINTS environment variable (default mapping)
// Format: "node1:host1,node2:host2" or "node1:host1:8006,node2:host2:8006"
// Returns a map of node name -> endpoint (hostname/IP, optionally with port)
func parseNodeEndpointsMapping() map[string]string {
	mapping := make(map[string]string)
	envValue := os.Getenv("PROXMOX_NODE_ENDPOINTS")
	if envValue == "" {
		return mapping
	}

	// Parse comma-separated node mappings
	nodeStrings := strings.Split(envValue, ",")
	for _, nodeStr := range nodeStrings {
		nodeStr = strings.TrimSpace(nodeStr)
		if nodeStr == "" {
			continue
		}

		// Parse "nodeName:endpoint" format (endpoint can be hostname/IP, optionally with port)
		if strings.Contains(nodeStr, ":") {
			// Split on first colon only (endpoint might contain port with colon)
			parts := strings.SplitN(nodeStr, ":", 2)
			if len(parts) == 2 {
				nodeName := strings.TrimSpace(parts[0])
				endpoint := strings.TrimSpace(parts[1])
				if nodeName != "" && endpoint != "" {
					mapping[nodeName] = endpoint
				}
			}
		}
	}

	return mapping
}

// parseNodeProxmoxMapping parses the PROXMOX_NODE_API_ENDPOINTS environment variable (API override)
// Format: "node1:https://proxmox1:8006,node2:https://proxmox2:8006"
// Returns a map of node name -> API URL
// Falls back to PROXMOX_NODE_ENDPOINTS if API override not configured
func parseNodeProxmoxMapping() (map[string]string, error) {
	mapping := make(map[string]string)

	// First, check for API-specific override
	envValue := os.Getenv("PROXMOX_NODE_API_ENDPOINTS")
	if envValue != "" {
		// Parse comma-separated node mappings
		nodeStrings := strings.Split(envValue, ",")
		for _, nodeStr := range nodeStrings {
			nodeStr = strings.TrimSpace(nodeStr)
			if nodeStr == "" {
				continue
			}

			// Parse "nodeName:apiURL" format
			// API URL may contain colons (https://), so we need to split on first colon only
			parts := strings.SplitN(nodeStr, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid Proxmox API mapping format: %s (expected 'nodeName:apiURL')", nodeStr)
			}

			nodeName := strings.TrimSpace(parts[0])
			apiURL := strings.TrimSpace(parts[1])
			if nodeName == "" || apiURL == "" {
				return nil, fmt.Errorf("invalid Proxmox API mapping: node name and URL cannot be empty in '%s'", nodeStr)
			}

			// Validate URL format (should start with http:// or https://)
			if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
				return nil, fmt.Errorf("invalid Proxmox API URL format in '%s': must start with http:// or https://", nodeStr)
			}

			mapping[nodeName] = apiURL
		}
		return mapping, nil
	}

	// Fall back to default PROXMOX_NODE_ENDPOINTS and construct API URLs
	defaultMapping := parseNodeEndpointsMapping()
	if len(defaultMapping) > 0 {
		// Construct API URLs from default endpoints (assume https:// and port 8006)
		for nodeName, endpoint := range defaultMapping {
			// Extract hostname/IP (remove port if present)
			host := endpoint
			if strings.Contains(endpoint, ":") {
				// Endpoint has port, extract just the hostname/IP
				host = strings.Split(endpoint, ":")[0]
			}
			// Construct API URL with https:// and port 8006
			apiURL := fmt.Sprintf("https://%s:8006", host)
			mapping[nodeName] = apiURL
		}
	}

	return mapping, nil
}

// resolveProxmoxURLForNode resolves the Proxmox API URL for a given node name
// Uses PROXMOX_NODE_API_ENDPOINTS or PROXMOX_NODE_ENDPOINTS mapping (required)
func resolveProxmoxURLForNode(nodeName string) (string, error) {
	// Try per-node mapping
	mapping, err := parseNodeProxmoxMapping()
	if err != nil {
		return "", fmt.Errorf("failed to parse Proxmox API mapping: %w", err)
	}

	// Mapping is required - no fallback
	if len(mapping) == 0 {
		return "", fmt.Errorf("PROXMOX_NODE_ENDPOINTS environment variable is required (configure node endpoints mapping, e.g., 'node1:proxmox1.example.com')")
	}

	// If node is specified, use it
	if nodeName != "" {
		apiURL, ok := mapping[nodeName]
		if ok {
			return apiURL, nil
		}
		// Node not found in mapping - return error
		return "", fmt.Errorf("no Proxmox API endpoint configured for node '%s'. Available nodes: %v", nodeName, getProxmoxNodeNames(mapping))
	}

	// If no node specified but mapping exists, return error (node name is required)
	return "", fmt.Errorf("node name is required when using PROXMOX_NODE_ENDPOINTS. Configure PROXMOX_NODE_ENDPOINTS with node mappings (e.g., 'node1:proxmox1.example.com')")
}

// getProxmoxNodeNames extracts node names from mapping for error messages
func getProxmoxNodeNames(mapping map[string]string) []string {
	nodes := make([]string, 0, len(mapping))
	for node := range mapping {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetFirstProxmoxNodeName returns the first node name from the mapping for discovery purposes
// Returns empty string if no nodes are configured
func GetFirstProxmoxNodeName() (string, error) {
	mapping, err := parseNodeProxmoxMapping()
	if err != nil {
		return "", fmt.Errorf("failed to parse Proxmox API mapping: %w", err)
	}
	if len(mapping) == 0 {
		return "", fmt.Errorf("PROXMOX_NODE_ENDPOINTS environment variable is required (configure node endpoints mapping, e.g., 'node1:proxmox1.example.com')")
	}
	// Return first node (order is not guaranteed, but any node works for discovery)
	for nodeName := range mapping {
		return nodeName, nil
	}
	return "", fmt.Errorf("no nodes found in PROXMOX_NODE_ENDPOINTS mapping")
}

// GetAllProxmoxNodeNames returns all node names from the mapping
// Returns empty slice if no nodes are configured
func GetAllProxmoxNodeNames() ([]string, error) {
	mapping, err := parseNodeProxmoxMapping()
	if err != nil {
		return nil, fmt.Errorf("failed to parse Proxmox API mapping: %w", err)
	}
	if len(mapping) == 0 {
		return nil, fmt.Errorf("PROXMOX_NODE_ENDPOINTS environment variable is required (configure node endpoints mapping, e.g., 'node1:proxmox1.example.com')")
	}
	nodes := make([]string, 0, len(mapping))
	for nodeName := range mapping {
		nodes = append(nodes, nodeName)
	}
	return nodes, nil
}

// GetProxmoxConfig gets Proxmox configuration from environment variables
// nodeName is required when using PROXMOX_NODE_ENDPOINTS
func GetProxmoxConfig(nodeName ...string) (*ProxmoxConfig, error) {
	config := &ProxmoxConfig{}

	// Resolve API URL (node-specific, nodeName is required)
	var apiURL string
	var err error
	if len(nodeName) > 0 && nodeName[0] != "" {
		apiURL, err = resolveProxmoxURLForNode(nodeName[0])
		if err != nil {
			return nil, fmt.Errorf("failed to resolve Proxmox API URL for node %s: %w", nodeName[0], err)
		}
		logger.Debug("[GetProxmoxConfig] Using node-specific API URL for node %s: %s", nodeName[0], apiURL)
	} else {
		// Try to resolve without node name (will fail if PROXMOX_NODE_ENDPOINTS is configured)
		apiURL, err = resolveProxmoxURLForNode("")
		if err != nil {
			return nil, err
		}
	}
	config.APIURL = apiURL

	// Get username (default: root@pam)
	config.Username = os.Getenv("PROXMOX_USERNAME")
	if config.Username == "" {
		config.Username = "root@pam"
	}

	// Get password (optional if using token)
	config.Password = os.Getenv("PROXMOX_PASSWORD")

	// Get token (alternative to password)
	config.TokenID = os.Getenv("PROXMOX_TOKEN_ID")
	config.Secret = os.Getenv("PROXMOX_TOKEN_SECRET")

	// Validate that either password or token is provided
	if config.Password == "" && (config.TokenID == "" || config.Secret == "") {
		return nil, fmt.Errorf("either PROXMOX_PASSWORD or both PROXMOX_TOKEN_ID and PROXMOX_TOKEN_SECRET must be provided")
	}

	// If using token, clear password
	if config.TokenID != "" && config.Secret != "" {
		config.Password = "" // Clear password if using token
	}

	// SSH configuration for snippet writing (optional - only needed if using SSH method)
	// SSH host is resolved via resolveSSHEndpoint using node mapping, not from environment
	config.SSHHost = "" // No longer used - resolved via node mapping
	config.SSHUser = os.Getenv("PROXMOX_SSH_USER")
	if config.SSHUser == "" {
		config.SSHUser = "obiente-cloud" // Default SSH user
	}
	config.SSHKeyPath = os.Getenv("PROXMOX_SSH_KEY_PATH")
	config.SSHKeyData = os.Getenv("PROXMOX_SSH_KEY_DATA")

	return config, nil
}

// ProxmoxConfig holds Proxmox API configuration
type ProxmoxConfig struct {
	APIURL   string
	Username string
	Realm    string
	Password string
	TokenID  string // Alternative: use API token instead of password
	Secret   string // Token secret

	// SSH configuration for writing snippet files directly to Proxmox storage
	// SSHHost is no longer used - SSH endpoints are resolved via PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS
	SSHHost    string // Deprecated - not used (SSH endpoints resolved via node mapping)
	SSHUser    string // SSH user for snippet writing (e.g., "obiente-cloud")
	SSHKeyPath string // Path to SSH private key file (e.g., "/path/to/id_rsa")
	SSHKeyData string // SSH private key content (alternative to SSHKeyPath)
}

// NewVPSManager creates a new VPS manager
func NewVPSManager() (*VPSManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Gateway clients are created on-demand per node
	// No global gateway client - each node has its own gateway
	return &VPSManager{
		dockerClient:   cli,
		gatewayClient:  nil,        // Deprecated - use GetGatewayClientForNode instead
		gatewayClients: sync.Map{}, // Cache of gateway clients per node
		proxmoxClients: sync.Map{}, // Cache of Proxmox clients per node
	}, nil
}

// GetGatewayClientForNode gets or creates a gateway client for a specific node
// Clients are cached to avoid recreating for each request
// Returns error if node mapping not found in VPS_NODE_GATEWAY_ENDPOINTS
func (vm *VPSManager) GetGatewayClientForNode(nodeName string) (*VPSGatewayClient, error) {
	if nodeName == "" {
		return nil, fmt.Errorf("node name is required for gateway client")
	}

	// Check cache first
	if cached, ok := vm.gatewayClients.Load(nodeName); ok {
		if client, ok := cached.(*VPSGatewayClient); ok {
			logger.Debug("[VPSManager] Using cached gateway client for node %s", nodeName)
			return client, nil
		}
	}

	// Create new client for this node
	logger.Info("[VPSManager] Creating gateway client for node %s", nodeName)
	client, err := NewVPSGatewayClientForNode(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway client for node %s: %w", nodeName, err)
	}

	// Cache the client
	vm.gatewayClients.Store(nodeName, client)
	return client, nil
}

// GetProxmoxClientForNode gets or creates a Proxmox client for a specific node
// Clients are cached to avoid recreating for each request
// If nodeName is empty, uses default/fallback configuration
func (vm *VPSManager) GetProxmoxClientForNode(nodeName string) (*ProxmoxClient, error) {
	// Use empty string as cache key for default/fallback
	cacheKey := nodeName
	if nodeName == "" {
		cacheKey = "__default__"
	}

	// Check cache first
	if cached, ok := vm.proxmoxClients.Load(cacheKey); ok {
		if client, ok := cached.(*ProxmoxClient); ok {
			logger.Debug("[VPSManager] Using cached Proxmox client for node %s", nodeName)
			return client, nil
		}
	}

	// Create new client for this node
	logger.Info("[VPSManager] Creating Proxmox client for node %s", nodeName)
	proxmoxConfig, err := GetProxmoxConfig(nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Proxmox config for node %s: %w", nodeName, err)
	}

	client, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client for node %s: %w", nodeName, err)
	}

	// Cache the client
	vm.proxmoxClients.Store(cacheKey, client)
	return client, nil
}

// CreateVPS provisions a new VPS instance via Proxmox
// CreateVPS creates a new VPS instance
// Returns: VPS instance, root password (one-time only, not stored), error
func (vm *VPSManager) CreateVPS(ctx context.Context, config *VPSConfig, logWriter LogWriter) (*database.VPSInstance, string, error) {
	logger.Info("[VPSManager] Creating VPS instance %s", config.VPSID)

	// Helper to write log lines
	writeLog := func(line string, stderr bool) {
		if logWriter != nil {
			logWriter.WriteLine(line, stderr)
		}
	}

	writeLog("Starting server setup...", false)

	// Get organization settings to check if inter-VM communication is allowed
	var org database.Organization
	if err := database.DB.Where("id = ?", config.OrganizationID).First(&org).Error; err != nil {
		return nil, "", fmt.Errorf("failed to get organization: %w", err)
	}

	// Fetch organization name if not provided
	if config.OrganizationName == nil {
		orgName := org.Name
		config.OrganizationName = &orgName
	}

	// Fetch organization owner if not provided
	if config.OwnerID == nil || config.OwnerName == nil {
		var ownerMember database.OrganizationMember
		if err := database.DB.Where("organization_id = ? AND role = ? AND status = ?", config.OrganizationID, "owner", "active").First(&ownerMember).Error; err == nil {
			config.OwnerID = &ownerMember.UserID
			// Owner name will be resolved later if needed via user profile resolver
		}
	}

	writeLog("Setting up secure access...", false)

	// Generate bastion SSH key pair for this VPS (required for SSH access)
	bastionKey, err := database.CreateVPSBastionKey(config.VPSID, config.OrganizationID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create bastion SSH key: %w", err)
	}
	logger.Info("[VPSManager] Generated bastion SSH key for VPS %s (fingerprint: %s)", config.VPSID, bastionKey.Fingerprint)

	// Generate web terminal SSH key pair for this VPS (optional, can be removed to disable web terminal)
	terminalKey, err := database.CreateVPSTerminalKey(config.VPSID, config.OrganizationID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create terminal SSH key: %w", err)
	}
	logger.Info("[VPSManager] Generated web terminal SSH key for VPS %s (fingerprint: %s)", config.VPSID, terminalKey.Fingerprint)
	writeLog("Secure access configured", false)

	// Track if we need to clean up keys on failure
	cleanupBastionKey := true
	cleanupTerminalKey := true
	defer func() {
		if cleanupBastionKey {
			if delErr := database.DeleteVPSBastionKey(config.VPSID); delErr != nil {
				logger.Warn("[VPSManager] Failed to delete bastion key after VPS creation failure: %v", delErr)
			}
		}
		if cleanupTerminalKey {
			if delErr := database.DeleteVPSTerminalKey(config.VPSID); delErr != nil {
				logger.Warn("[VPSManager] Failed to delete terminal key after VPS creation failure: %v", delErr)
			}
		}
	}()

	// Create VPS instance record in database with CREATING status first
	// This allows the frontend to show progress immediately
	earlyVPSInstance := &database.VPSInstance{
		ID:             config.VPSID,
		Name:           config.Name,
		Description:    config.Description,
		Status:         1, // CREATING
		Region:         config.Region,
		Image:          int32(config.Image),
		ImageID:        config.ImageID,
		Size:           config.Size,
		CPUCores:       config.CPUCores,
		MemoryBytes:    config.MemoryBytes,
		DiskBytes:      config.DiskBytes,
		OrganizationID: config.OrganizationID,
		CreatedBy:      config.CreatedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Metadata:       "{}",
		IPv4Addresses:  "[]",
		IPv6Addresses:  "[]",
	}

	// Store metadata as JSON if provided
	if len(config.Metadata) > 0 {
		metadataJSON, err := json.Marshal(config.Metadata)
		if err == nil {
			earlyVPSInstance.Metadata = string(metadataJSON)
		}
	}

	// Create the VPS record in database with CREATING status
	// This allows the frontend to immediately show the VPS card with progress
	if err := database.DB.Create(earlyVPSInstance).Error; err != nil {
		// If it already exists (e.g., from a previous failed attempt), log and continue
		logger.Warn("[VPSManager] VPS record %s already exists, will update it: %v", config.VPSID, err)
	}

	// Determine target node from region mapping (same logic as CreateVM)
	// This allows us to use the correct gateway for IP allocation
	targetNodeName := ""
	if config.Region != "" {
		regionNodeMap := parseRegionNodeMapping()
		if mappedNode, ok := regionNodeMap[config.Region]; ok {
			targetNodeName = mappedNode
			logger.Info("[VPSManager] Using mapped node %s for region %s (for gateway selection)", targetNodeName, config.Region)
		}
	}

	// Get Proxmox client for target node (if known) or try all nodes from mapping
	// If region is specified, MUST use that region's node (no fallback to other nodes)
	// If no region is specified, try all nodes sequentially
	var proxmoxClient *ProxmoxClient
	if targetNodeName != "" {
		// Region is specified - MUST use the region's node, fail if unavailable
		client, err := vm.GetProxmoxClientForNode(targetNodeName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get Proxmox client for region %s node %s: %w. The region's node must be available to create VPS in this region", config.Region, targetNodeName, err)
		}
		proxmoxClient = client
		logger.Info("[VPSManager] Using region %s node %s for VPS creation", config.Region, targetNodeName)
	} else {
		// No region specified - try all nodes from mapping sequentially
		allNodes, err := GetAllProxmoxNodeNames()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get Proxmox nodes: %w", err)
		}
		var lastErr error
		for _, node := range allNodes {
			client, err := vm.GetProxmoxClientForNode(node)
			if err == nil {
				proxmoxClient = client
				logger.Info("[VPSManager] Using node %s for VPS creation (no region specified)", node)
				break
			}
			logger.Debug("[VPSManager] Failed to get Proxmox client for node %s: %v (trying next node)", node, err)
			lastErr = err
		}
		if proxmoxClient == nil {
			return nil, "", fmt.Errorf("failed to get Proxmox client after trying all %d nodes: %w", len(allNodes), lastErr)
		}
	}

	// Allocate IP address from gateway if available
	// Use node-specific gateway if we know the target node
	var allocatedIP string
	var macAddress string
	var gatewayClientForNode *VPSGatewayClient

	// Try to get gateway client for the target node
	if targetNodeName != "" {
		client, err := vm.GetGatewayClientForNode(targetNodeName)
		if err != nil {
			logger.Warn("[VPSManager] Failed to get gateway client for node %s: %v (will try after VM creation)", targetNodeName, err)
		} else {
			gatewayClientForNode = client
		}
	}

	if gatewayClientForNode != nil {
		writeLog("Assigning network address...", false)
		// Generate MAC address for the VM (Proxmox will assign one, but we need it for DHCP)
		// Format: 00:16:3e:XX:XX:XX (QEMU/KVM standard prefix)
		macAddress = generateMACAddress()

		// Request IP allocation from gateway with timeout
		// Use independent context to avoid HTTP request timeout issues
		allocCtx, allocCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer allocCancel()

		logger.Info("[VPSManager] Requesting IP allocation from gateway for VPS %s (MAC: %s, node: %s)", config.VPSID, macAddress, targetNodeName)
		allocResp, err := gatewayClientForNode.AllocateIP(allocCtx, config.VPSID, config.OrganizationID, macAddress)
		if err != nil {
			logger.Warn("[VPSManager] Failed to allocate IP from gateway for VPS %s: %v (continuing without gateway IP)", config.VPSID, err)
			writeLog("Network address will be assigned automatically", false)
			// Continue without gateway IP - VM will use DHCP or static IP from Proxmox
		} else {
			allocatedIP = allocResp.IpAddress
			logger.Info("[VPSManager] Allocated IP %s for VPS %s from gateway on node %s", allocatedIP, config.VPSID, targetNodeName)
			writeLog(fmt.Sprintf("Network address assigned: %s", allocatedIP), false)
		}
	} else {
		logger.Debug("[VPSManager] Gateway client not available for node %s, skipping IP allocation", targetNodeName)
	}

	// Provision VM via Proxmox API
	// Use independent context with generous timeout to avoid HTTP request context cancellation
	// The HTTP request context may have a shorter timeout, but VM creation can take 1-2 minutes
	writeLog("Creating server...", false)
	logger.Info("[VPSManager] Starting VM provisioning via Proxmox for VPS %s", config.VPSID)
	var createResult *CreateVMResult
	vmCtx, vmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer vmCancel()
	createResult, err = proxmoxClient.CreateVM(vmCtx, config, org.AllowInterVMCommunication, logWriter)
	if err != nil {
		// If VM creation fails, update VPS status to FAILED and release the allocated IP
		database.DB.Model(&database.VPSInstance{}).Where("id = ?", config.VPSID).Update("status", 7) // FAILED
		if gatewayClientForNode != nil && allocatedIP != "" {
			// Use independent context for IP release
			releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer releaseCancel()
			if releaseErr := gatewayClientForNode.ReleaseIP(releaseCtx, config.VPSID); releaseErr != nil {
				logger.Warn("[VPSManager] Failed to release IP %s after VM creation failure: %v", allocatedIP, releaseErr)
			}
		}
		return nil, "", fmt.Errorf("failed to provision VM via Proxmox: %w", err)
	}

	vmID := createResult.VMID
	rootPassword := createResult.Password
	nodeName := createResult.NodeName

	logger.Info("[VPSManager] Received CreateVMResult: VMID=%s, Password length=%d, NodeName=%s", vmID, len(rootPassword), nodeName)

	// Get actual VM status from Proxmox and map to our status enum
	vmIDInt := 0
	fmt.Sscanf(vmID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, "", fmt.Errorf("invalid VM ID: %s", vmID)
	}

	// Use the node name returned from CreateVM (more reliable than searching)
	if nodeName == "" {
		// Fallback: find the node if not returned (shouldn't happen, but be safe)
		var err error
		nodeName, err = proxmoxClient.FindVMNode(vmCtx, vmIDInt)
		if err != nil {
			return nil, "", fmt.Errorf("failed to find Proxmox node for VM %d: %w", vmIDInt, err)
		}
	}

	// Verify VM actually exists before creating VPS record
	// Proxmox may need a moment to create the VM configuration file, so we retry with backoff
	var proxmoxStatus string
	maxRetries := 10
	retryDelay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		status, statusErr := proxmoxClient.GetVMStatus(vmCtx, nodeName, vmIDInt)
		if statusErr == nil {
			// Successfully got VM status
			proxmoxStatus = status
			break
		}

		errorMsg := statusErr.Error()
		// If the error indicates the VM config doesn't exist, retry (Proxmox might still be creating it)
		if strings.Contains(errorMsg, "does not exist") || strings.Contains(errorMsg, "Configuration file") {
			if attempt < maxRetries-1 {
				// Not the last attempt, wait and retry
				logger.Debug("[VPSManager] VM %d config not yet available (attempt %d/%d), retrying in %v...", vmIDInt, attempt+1, maxRetries, retryDelay)
				time.Sleep(retryDelay)
				retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff, max ~19 seconds total
				continue
			}
			// Last attempt failed - VM creation likely failed
			return nil, "", fmt.Errorf("VM creation failed: VM %d does not exist in Proxmox after %d attempts. The VM may not have been created properly: %w", vmIDInt, maxRetries, statusErr)
		}

		// For other errors, fail immediately (not a timing issue)
		return nil, "", fmt.Errorf("failed to verify VM exists after creation: %w", statusErr)
	}

	if proxmoxStatus == "" {
		return nil, "", fmt.Errorf("failed to get VM status after %d attempts", maxRetries)
	}

	// Map Proxmox status to our VPSStatus enum
	vpsStatus := mapProxmoxStatusToVPSStatus(proxmoxStatus)

	writeLog("Server setup complete!", false)

	// Create VPS instance record in database
	vpsInstance := &database.VPSInstance{
		ID:             config.VPSID,
		Name:           config.Name,
		Description:    config.Description,
		Status:         vpsStatus,
		Region:         config.Region,
		Image:          int32(config.Image),
		ImageID:        config.ImageID,
		Size:           config.Size,
		CPUCores:       config.CPUCores,
		MemoryBytes:    config.MemoryBytes,
		DiskBytes:      config.DiskBytes,
		InstanceID:     &vmID,
		NodeID:         &nodeName, // Store Proxmox node name for gateway routing
		SSHKeyID:       config.SSHKeyID,
		OrganizationID: config.OrganizationID,
		CreatedBy:      config.CreatedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// NOTE: Root password is NOT stored in database for security
	// Password is only returned once in CreateVPS response, then discarded

	// Store metadata as JSON (must be valid JSON or NULL for JSONB columns)
	if len(config.Metadata) > 0 {
		metadataJSON, err := json.Marshal(config.Metadata)
		if err == nil {
			vpsInstance.Metadata = string(metadataJSON)
		} else {
			vpsInstance.Metadata = "{}" // Default to empty object if marshaling fails
		}
	} else {
		vpsInstance.Metadata = "{}" // Empty object for JSONB, not empty string
	}

	// Store IP addresses as JSON arrays (must be valid JSON or NULL for JSONB columns)
	// Only store IPs from config (e.g., static IPs) - do not store gateway-allocated IP
	// Gateway-allocated IPs will be verified via guest agent when available
	// This ensures we only show IPs that we can verify from the actual VPS
	if len(config.IPv4Addresses) > 0 {
		ipv4JSON, err := json.Marshal(config.IPv4Addresses)
		if err == nil {
			vpsInstance.IPv4Addresses = string(ipv4JSON)
		} else {
			vpsInstance.IPv4Addresses = "[]" // Default to empty array if marshaling fails
		}
	} else {
		vpsInstance.IPv4Addresses = "[]" // Empty array for JSONB, not empty string
	}
	if len(config.IPv6Addresses) > 0 {
		ipv6JSON, err := json.Marshal(config.IPv6Addresses)
		if err == nil {
			vpsInstance.IPv6Addresses = string(ipv6JSON)
		} else {
			vpsInstance.IPv6Addresses = "[]" // Default to empty array if marshaling fails
		}
	} else {
		vpsInstance.IPv6Addresses = "[]" // Empty array for JSONB, not empty string
	}

	// Update existing VPS record (created early with CREATING status)
	if err := database.DB.Where("id = ?", config.VPSID).Save(vpsInstance).Error; err != nil {
		// If save fails, try create (in case record was deleted)
		if err := database.DB.Create(vpsInstance).Error; err != nil {
			return nil, "", fmt.Errorf("failed to save VPS instance record: %w", err)
		}
	}

	// VPS created successfully, don't clean up keys
	cleanupBastionKey = false
	cleanupTerminalKey = false

	logger.Info("[VPSManager] Created VPS instance %s (VM ID: %s)",
		config.VPSID, vmID)

	// Return VPS instance and root password (password is NOT stored in database)
	// Password is only returned once in CreateVPS response, then discarded
	return vpsInstance, rootPassword, nil
}

// VPSConfig holds configuration for creating a VPS instance
type VPSConfig struct {
	VPSID          string
	Name           string
	Description    *string
	Region         string
	Image          int // VPSImage enum
	ImageID        *string
	Size           string
	CPUCores       int32
	MemoryBytes    int64
	DiskBytes      int64
	SSHKeyID       *string
	Metadata       map[string]string
	IPv4Addresses  []string
	IPv6Addresses  []string
	OrganizationID string
	CreatedBy      string

	// Optional metadata for VPS notes (organization and user names)
	OrganizationName *string // Organization name (optional, fetched if not provided)
	CreatorName      *string // Creator user name (optional, fetched if not provided)
	OwnerID          *string // Organization owner ID (optional, fetched if not provided)
	OwnerName        *string // Organization owner name (optional, fetched if not provided)

	// Cloud-init configuration
	CloudInit    *CloudInitConfig
	RootPassword *string // Custom root password (optional, auto-generated if not provided)
}

// CloudInitConfig contains cloud-init configuration options
type CloudInitConfig struct {
	Users            []CloudInitUser
	Hostname         *string
	Timezone         *string
	Locale           *string
	Packages         []string
	PackageUpdate    *bool
	PackageUpgrade   *bool
	Runcmd           []string
	WriteFiles       []CloudInitWriteFile
	SSHInstallServer *bool
	SSHAllowPW       *bool
}

// CloudInitUser represents a user to be created via cloud-init
type CloudInitUser struct {
	Name              string
	Password          *string
	SSHAuthorizedKeys []string
	Sudo              *bool
	SudoNopasswd      *bool
	Groups            []string
	Shell             *string
	LockPasswd        *bool
	Gecos             *string
}

// CloudInitWriteFile represents a file to be written via cloud-init
type CloudInitWriteFile struct {
	Path        string
	Content     string
	Owner       *string
	Permissions *string
	Append      *bool
	Defer       *bool
}

// StartVPS starts a VPS instance
func (vm *VPSManager) StartVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox client for the node where VPS is running
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	// Parse VM ID
	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node if not stored in VPS
	if nodeName == "" {
		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("failed to find Proxmox node: %w", err)
		}
		nodeName = nodes[0]
	}

	if err := proxmoxClient.startVM(ctx, nodeName, vmIDInt); err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "has been deleted from Proxmox") {
			logger.Info("[VPSManager] VM %d has been deleted from Proxmox - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Get actual status from Proxmox and update
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "does not exist") {
			logger.Info("[VPSManager] VM %d does not exist in Proxmox after start - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		logger.Warn("[VPSManager] Failed to get VM status after start, defaulting to RUNNING: %v", err)
		vps.Status = 3 // RUNNING
	} else {
		vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// StopVPS stops a VPS instance
func (vm *VPSManager) StopVPS(ctx context.Context, vpsID string, force bool) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox client for the node where VPS is running
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node if not stored in VPS
	if nodeName == "" {
		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("failed to find Proxmox node: %w", err)
		}
		nodeName = nodes[0]
	}

	if err := proxmoxClient.StopVM(ctx, nodeName, vmIDInt); err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "has been deleted from Proxmox") {
			logger.Info("[VPSManager] VM %d has been deleted from Proxmox - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	// Get actual status from Proxmox and update
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "does not exist") {
			logger.Info("[VPSManager] VM %d does not exist in Proxmox after stop - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		logger.Warn("[VPSManager] Failed to get VM status after stop, defaulting to STOPPED: %v", err)
		vps.Status = 5 // STOPPED
	} else {
		vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// RebootVPS reboots a VPS instance
func (vm *VPSManager) RebootVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox client for the node where VPS is running
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node if not stored in VPS
	if nodeName == "" {
		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("failed to find Proxmox node: %w", err)
		}
		nodeName = nodes[0]
	}

	if err := proxmoxClient.RebootVM(ctx, nodeName, vmIDInt); err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "has been deleted from Proxmox") {
			logger.Info("[VPSManager] VM %d has been deleted from Proxmox - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		return fmt.Errorf("failed to reboot VM: %w", err)
	}

	// Get actual status from Proxmox and update
	// Note: Reboot is async, so status might be "running" or transitioning
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		// Check if VM was deleted from Proxmox
		if strings.Contains(err.Error(), "does not exist") {
			logger.Info("[VPSManager] VM %d does not exist in Proxmox after reboot - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				logger.Warn("[VPSManager] Failed to update VPS status to DELETED: %v", err)
			}
			return fmt.Errorf("VM has been deleted from Proxmox")
		}
		logger.Warn("[VPSManager] Failed to get VM status after reboot, defaulting to REBOOTING: %v", err)
		vps.Status = 6 // REBOOTING
	} else {
		// If VM is still running, it's rebooting; if stopped, it might be starting
		if proxmoxStatus == "running" {
			vps.Status = 6 // REBOOTING
		} else {
			vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
		}
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// GetVPSStatus retrieves the current status of a VPS from Proxmox
func (vm *VPSManager) GetVPSStatus(ctx context.Context, vpsID string) (string, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return "", fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return "", fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox client for the node where VPS is running
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return "", fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return "", fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node if not stored in VPS
	if nodeName == "" {
		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return "", fmt.Errorf("failed to find Proxmox node: %w", err)
		}
		nodeName = nodes[0]
	}

	status, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		return "", fmt.Errorf("failed to get VM status: %w", err)
	}

	return status, nil
}

// GetVPSIPAddresses retrieves IP addresses of a VPS
// Priority: 1. Gateway (authoritative for DHCP leases), 2. Guest agent, 3. Database cache
// Updates database cache when IP is discovered from gateway or guest agent
func (vm *VPSManager) GetVPSIPAddresses(ctx context.Context, vpsID string) ([]string, []string, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, nil, fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return nil, nil, fmt.Errorf("VPS has no instance ID")
	}

	var ipv4, ipv6 []string
	var gatewayErr, guestAgentErr error

	// Priority 1: Try gateway first (authoritative source for DHCP-assigned IPs)
	// The gateway manages DHCP leases so it has the most up-to-date IP information
	if vps.NodeID != nil && *vps.NodeID != "" {
		gatewayClient, err := vm.GetGatewayClientForNode(*vps.NodeID)
		if err == nil {
			allocations, listErr := gatewayClient.ListIPs(ctx, vps.OrganizationID, vpsID)
			if listErr == nil && len(allocations) > 0 {
				gatewayIP := allocations[0].IpAddress
				logger.Info("[VPSManager] Got IP %s from gateway for VPS %s", gatewayIP, vpsID)
				ipv4 = []string{gatewayIP}

				// Update database cache if IP changed
				vm.updateIPCacheIfChanged(&vps, ipv4, ipv6)
				return ipv4, ipv6, nil
			} else if listErr != nil {
				gatewayErr = listErr
				logger.Debug("[VPSManager] Failed to get IP from gateway for VPS %s: %v", vpsID, listErr)
			} else {
				logger.Debug("[VPSManager] Gateway returned no IP allocations for VPS %s", vpsID)
			}
		} else {
			gatewayErr = err
			logger.Debug("[VPSManager] Failed to get gateway client for node %s: %v", *vps.NodeID, err)
		}
	}

	// Priority 2: Try Proxmox guest agent
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err == nil {
		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt > 0 {
			// Find node if not stored in VPS
			if nodeName == "" {
				nodes, err := proxmoxClient.ListNodes(ctx)
				if err == nil && len(nodes) > 0 {
					nodeName = nodes[0]
				}
			}

			if nodeName != "" {
				ipv4, ipv6, guestAgentErr = proxmoxClient.GetVMIPAddresses(ctx, nodeName, vmIDInt)
				if guestAgentErr == nil && (len(ipv4) > 0 || len(ipv6) > 0) {
					logger.Info("[VPSManager] Got IPs from guest agent for VPS %s: IPv4=%v, IPv6=%v", vpsID, ipv4, ipv6)

					// Update database cache if IP changed
					vm.updateIPCacheIfChanged(&vps, ipv4, ipv6)
					return ipv4, ipv6, nil
				} else if guestAgentErr != nil {
					logger.Debug("[VPSManager] Guest agent unavailable for VPS %s: %v", vpsID, guestAgentErr)
				}
			}
		}
	}

	// Priority 3: Fall back to database cache (may be stale but better than nothing)
	var cachedIPv4, cachedIPv6 []string
	if vps.IPv4Addresses != "" && vps.IPv4Addresses != "[]" {
		if err := json.Unmarshal([]byte(vps.IPv4Addresses), &cachedIPv4); err == nil && len(cachedIPv4) > 0 {
			logger.Info("[VPSManager] Using cached IPs from database for VPS %s (gateway/guest agent unavailable): IPv4=%v", vpsID, cachedIPv4)
			return cachedIPv4, cachedIPv6, nil
		}
	}
	if vps.IPv6Addresses != "" && vps.IPv6Addresses != "[]" {
		if err := json.Unmarshal([]byte(vps.IPv6Addresses), &cachedIPv6); err == nil && len(cachedIPv6) > 0 {
			logger.Info("[VPSManager] Using cached IPv6 from database for VPS %s: IPv6=%v", vpsID, cachedIPv6)
			return cachedIPv4, cachedIPv6, nil
		}
	}

	// All sources failed
	if gatewayErr != nil {
		return nil, nil, fmt.Errorf("failed to get VM IP addresses: gateway error: %w", gatewayErr)
	}
	if guestAgentErr != nil {
		return nil, nil, fmt.Errorf("failed to get VM IP addresses: guest agent error: %w", guestAgentErr)
	}
	return []string{}, []string{}, nil
}

// updateIPCacheIfChanged updates the database cache if IPs have changed
func (vm *VPSManager) updateIPCacheIfChanged(vps *database.VPSInstance, ipv4, ipv6 []string) {
	updates := map[string]interface{}{}

	// Check IPv4
	if len(ipv4) > 0 {
		newIPv4JSON, err := json.Marshal(ipv4)
		if err == nil {
			currentIPv4 := vps.IPv4Addresses
			if currentIPv4 != string(newIPv4JSON) {
				updates["ipv4_addresses"] = string(newIPv4JSON)
				logger.Info("[VPSManager] Updating IPv4 cache for VPS %s: %s -> %s", vps.ID, currentIPv4, string(newIPv4JSON))
			}
		}
	}

	// Check IPv6
	if len(ipv6) > 0 {
		newIPv6JSON, err := json.Marshal(ipv6)
		if err == nil {
			currentIPv6 := vps.IPv6Addresses
			if currentIPv6 != string(newIPv6JSON) {
				updates["ipv6_addresses"] = string(newIPv6JSON)
				logger.Info("[VPSManager] Updating IPv6 cache for VPS %s: %s -> %s", vps.ID, currentIPv6, string(newIPv6JSON))
			}
		}
	}

	// Apply updates if any
	if len(updates) > 0 {
		if err := database.DB.Model(vps).Updates(updates).Error; err != nil {
			logger.Warn("[VPSManager] Failed to update IP cache for VPS %s: %v", vps.ID, err)
		}
	}
}

// DeleteVPS deletes a VPS instance from Proxmox
// SECURITY: Only deletes VMs that were created by our API
func (vm *VPSManager) DeleteVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox client for the node where VPS is running
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	}

	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node if not stored in VPS
	if nodeName == "" {
		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("failed to find Proxmox node: %w", err)
		}
		nodeName = nodes[0]
	}

	// Release IP address from gateway if available
	// Get gateway client for the node where VPS is running
	if vps.NodeID != nil && *vps.NodeID != "" {
		gatewayClient, err := vm.GetGatewayClientForNode(*vps.NodeID)
		if err == nil {
			if err := gatewayClient.ReleaseIP(ctx, vpsID); err != nil {
				logger.Warn("[VPSManager] Failed to release IP from gateway for VPS %s: %v (continuing with VM deletion)", vpsID, err)
				// Continue with VM deletion even if IP release fails
			} else {
				logger.Info("[VPSManager] Released IP from gateway for VPS %s", vpsID)
			}
		} else {
			logger.Debug("[VPSManager] Failed to get gateway client for node %s: %v", *vps.NodeID, err)
		}
	}

	// Delete snippet file if it exists
	// Use PROXMOX_SNIPPET_STORAGE if set, otherwise fallback to PROXMOX_STORAGE_POOL, then default to "local"
	snippetStorage := os.Getenv("PROXMOX_SNIPPET_STORAGE")
	if snippetStorage == "" {
		snippetStorage = os.Getenv("PROXMOX_STORAGE_POOL")
		if snippetStorage == "" {
			snippetStorage = "local"
		}
	}
	snippetFilename := fmt.Sprintf("vm-%d-user-data", vmIDInt)
	if err := proxmoxClient.deleteSnippetViaSSH(ctx, nodeName, snippetStorage, snippetFilename); err != nil {
		logger.Warn("[VPSManager] Failed to delete snippet file for VPS %s: %v (continuing with VM deletion)", vpsID, err)
		// Continue with VM deletion even if snippet deletion fails
	} else {
		logger.Info("[VPSManager] Deleted snippet file for VPS %s", vpsID)
	}

	// Delete web terminal SSH key
	if err := database.DeleteVPSTerminalKey(vpsID); err != nil {
		logger.Warn("[VPSManager] Failed to delete terminal key for VPS %s: %v (continuing with VM deletion)", vpsID, err)
		// Continue with VM deletion even if terminal key deletion fails
	} else {
		logger.Info("[VPSManager] Deleted terminal key for VPS %s", vpsID)
	}

	// DeleteVM will validate that the VM was created by our API by checking VM name matches VPS ID
	// If nodeName is not set, try to find the VM on any node
	if nodeName == "" {
		allNodes, err := proxmoxClient.ListNodes(ctx)
		if err == nil {
			// Try to find which node the VM is actually on
			for _, node := range allNodes {
				status, statusErr := proxmoxClient.GetVMStatus(ctx, node, vmIDInt)
				if statusErr == nil && status != "" {
					nodeName = node
					logger.Info("[VPSManager] Found VM %d on node %s", vmIDInt, nodeName)
					break
				}
			}
		}
	}

	if err := proxmoxClient.DeleteVM(ctx, nodeName, vmIDInt, vpsID); err != nil {
		// If deletion fails and we have a specific node, try other nodes as fallback
		if nodeName != "" {
			allNodes, err := proxmoxClient.ListNodes(ctx)
			if err == nil && len(allNodes) > 1 {
				logger.Warn("[VPSManager] Failed to delete VM %d from node %s: %v. Trying other nodes...", vmIDInt, nodeName, err)
				for _, otherNode := range allNodes {
					if otherNode == nodeName {
						continue
					}
					if delErr := proxmoxClient.DeleteVM(ctx, otherNode, vmIDInt, vpsID); delErr == nil {
						logger.Info("[VPSManager] Successfully deleted VM %d from node %s", vmIDInt, otherNode)
						goto deletionSuccess
					}
				}
			}
		}
		return fmt.Errorf("failed to delete VM: %w", err)
	}

deletionSuccess:

	logger.Info("[VPSManager] Successfully deleted VPS %s (VM ID: %d)", vpsID, vmIDInt)
	return nil
}

// ReinitializeVPS reinitializes a VPS instance by deleting the VM and recreating it
// This will delete all data on the VPS and reinstall the operating system
// The VPS will be reconfigured with the same cloud-init settings
// Returns: VPS instance, root password (one-time only, not stored), error
func (vm *VPSManager) ReinitializeVPS(ctx context.Context, vpsID string) (*database.VPSInstance, string, error) {
	logger.Info("[VPSManager] Reinitializing VPS instance %s", vpsID)

	// Get VPS instance from database
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, "", fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return nil, "", fmt.Errorf("VPS has no instance ID (not provisioned yet)")
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, "", fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, "", fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use")
	}

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	// Get current cloud-init configuration before deleting the VM
	// We'll use a default config - the actual cloud-init will be reapplied after VM creation
	// via the ConfigService which will load it from the database
	packageUpdate := true
	packageUpgrade := false
	sshInstallServer := true
	sshAllowPW := true
	cloudInitConfig := &CloudInitConfig{
		Users:            []CloudInitUser{},
		PackageUpdate:    &packageUpdate,
		PackageUpgrade:   &packageUpgrade,
		SSHInstallServer: &sshInstallServer,
		SSHAllowPW:       &sshAllowPW,
	}

	// Stop VM if running (required before deletion)
	status, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err == nil && status != "stopped" {
		logger.Info("[VPSManager] Stopping VM %d before reinitialization", vmIDInt)
		if err := proxmoxClient.StopVM(ctx, nodeName, vmIDInt); err != nil {
			return nil, "", fmt.Errorf("failed to stop VM before reinitialization: %w", err)
		}
		// Wait for VM to stop
		if err := proxmoxClient.waitForVMStatus(ctx, nodeName, vmIDInt, "stopped", 30*time.Second); err != nil {
			return nil, "", fmt.Errorf("VM %d did not stop within timeout: %w", vmIDInt, err)
		}
	}

	// Delete the VM (but keep VPS record in database)
	// DeleteVM will validate that the VM was created by our API
	if err := proxmoxClient.DeleteVM(ctx, nodeName, vmIDInt, vpsID); err != nil {
		return nil, "", fmt.Errorf("failed to delete VM: %w", err)
	}

	// Clear instance ID in database (VM is deleted, will be recreated)
	oldInstanceID := vps.InstanceID
	vps.InstanceID = nil
	vps.Status = 1 // CREATING
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, "", fmt.Errorf("failed to update VPS status: %w", err)
	}

	// Get organization settings
	var org database.Organization
	if err := database.DB.Where("id = ?", vps.OrganizationID).First(&org).Error; err != nil {
		return nil, "", fmt.Errorf("failed to get organization: %w", err)
	}

	// Recreate VPS configuration from existing VPS instance
	// Map VPSImage enum to int
	imageInt := int(vps.Image)
	if imageInt == 0 {
		imageInt = 1 // Default to UBUNTU_22_04 if not set
	}

	recreateConfig := &VPSConfig{
		VPSID:          vps.ID,
		Name:           vps.Name,
		Description:    vps.Description,
		Region:         vps.Region,
		Image:          imageInt,
		ImageID:        vps.ImageID,
		Size:           vps.Size,
		CPUCores:       vps.CPUCores,
		MemoryBytes:    vps.MemoryBytes,
		DiskBytes:      vps.DiskBytes,
		SSHKeyID:       vps.SSHKeyID,
		OrganizationID: vps.OrganizationID,
		CreatedBy:      vps.CreatedBy,
		Metadata:       make(map[string]string),
		CloudInit:      cloudInitConfig,
	}

	// Preserve existing IP allocation if available
	// The gateway client should handle reallocation automatically
	if vm.gatewayClient != nil {
		// Release old IP (if allocated)
		if err := vm.gatewayClient.ReleaseIP(ctx, vpsID); err != nil {
			logger.Warn("[VPSManager] Failed to release old IP during reinitialization: %v (continuing)", err)
		}
	}

	// Recreate VM with same configuration
	// Note: We don't need to create new SSH keys - existing bastion and terminal keys will be reused
	// But we need to ensure they exist
	_, err = database.GetVPSBastionKey(vpsID)
	if err != nil {
		// Create bastion key if it doesn't exist
		_, err = database.CreateVPSBastionKey(vpsID, vps.OrganizationID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to ensure bastion SSH key exists: %w", err)
		}
		logger.Info("[VPSManager] Created missing bastion SSH key for VPS %s during reinitialization", vpsID)
	}

	_, err = database.GetVPSTerminalKey(vpsID)
	if err != nil {
		// Create terminal key if it doesn't exist
		_, err = database.CreateVPSTerminalKey(vpsID, vps.OrganizationID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to ensure terminal SSH key exists: %w", err)
		}
		logger.Info("[VPSManager] Created missing terminal SSH key for VPS %s during reinitialization", vpsID)
	}

	// Allocate IP address from gateway if available
	// Determine target node from region mapping or use existing node
	targetNodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		targetNodeName = *vps.NodeID
	} else if vps.Region != "" {
		regionNodeMap := parseRegionNodeMapping()
		if mappedNode, ok := regionNodeMap[vps.Region]; ok {
			targetNodeName = mappedNode
		}
	}

	var allocatedIP string
	var macAddress string
	var gatewayClientForRecreate *VPSGatewayClient

	if targetNodeName != "" {
		client, err := vm.GetGatewayClientForNode(targetNodeName)
		if err == nil {
			gatewayClientForRecreate = client
			macAddress = generateMACAddress()
			allocResp, err := gatewayClientForRecreate.AllocateIP(ctx, vpsID, vps.OrganizationID, macAddress)
			if err != nil {
				logger.Warn("[VPSManager] Failed to allocate IP from gateway during reinitialization: %v (continuing without gateway IP)", err)
			} else {
				allocatedIP = allocResp.IpAddress
				logger.Info("[VPSManager] Allocated IP %s for VPS %s from gateway on node %s during reinitialization", allocatedIP, vpsID, targetNodeName)
			}
		} else {
			logger.Debug("[VPSManager] Failed to get gateway client for node %s during reinitialization: %v", targetNodeName, err)
		}
	}

	// Create new VM via Proxmox
	createResult, err := proxmoxClient.CreateVM(ctx, recreateConfig, org.AllowInterVMCommunication, nil)
	if err != nil {
		// If VM creation fails, release the allocated IP
		if gatewayClientForRecreate != nil && allocatedIP != "" {
			if releaseErr := gatewayClientForRecreate.ReleaseIP(ctx, vpsID); releaseErr != nil {
				logger.Warn("[VPSManager] Failed to release IP %s after VM recreation failure: %v", allocatedIP, releaseErr)
			}
		}
		// Restore old instance ID on failure
		vps.InstanceID = oldInstanceID
		vps.Status = 7 // FAILED
		database.DB.Save(&vps)
		return nil, "", fmt.Errorf("failed to recreate VM via Proxmox: %w", err)
	}

	newVMID := createResult.VMID
	rootPassword := createResult.Password
	newNodeName := createResult.NodeName

	// Update VPS instance with new VM ID
	vmIDIntNew := 0
	fmt.Sscanf(newVMID, "%d", &vmIDIntNew)
	if vmIDIntNew == 0 {
		return nil, "", fmt.Errorf("invalid new VM ID: %s", newVMID)
	}

	// Use the node name from CreateVM result, or fallback to the one we found earlier
	if newNodeName == "" {
		newNodeName = nodeName
	}

	vmIDStr := newVMID
	vps.InstanceID = &vmIDStr
	vps.NodeID = &newNodeName
	vps.Status = 1 // CREATING
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, "", fmt.Errorf("failed to update VPS with new instance ID: %w", err)
	}

	logger.Info("[VPSManager] Successfully reinitialized VPS %s (old VM: %d, new VM: %d)", vpsID, vmIDInt, vmIDIntNew)

	// Refresh VPS instance from database
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, "", fmt.Errorf("failed to refresh VPS after reinitialization: %w", err)
	}

	return &vps, rootPassword, nil
}

// GetGatewayClient returns the gateway client (if available)
func (vm *VPSManager) GetGatewayClient() *VPSGatewayClient {
	return vm.gatewayClient
}

// Close closes the Docker client
func (vm *VPSManager) Close() error {
	return vm.dockerClient.Close()
}

// SyncVPSStatusFromProxmox updates the VPS status in the database based on the actual Proxmox VM status
func (vm *VPSManager) SyncVPSStatusFromProxmox(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// NodeID is required - if missing, discover it and update the VPS record
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		// NodeID is missing - need to discover it
		// Try all nodes sequentially until one succeeds
		allNodes, err := GetAllProxmoxNodeNames()
		if err != nil {
			return fmt.Errorf("VPS %s has no NodeID and cannot discover node: %w. Please set NodeID manually or re-import the VPS", vpsID, err)
		}
		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt == 0 {
			return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
		}
		// Try each node until we find one that can discover the VM
		var lastErr error
		for _, discoveryNode := range allNodes {
			// Get Proxmox client for this discovery node
			discoveryClient, err := vm.GetProxmoxClientForNode(discoveryNode)
			if err != nil {
				logger.Debug("[VPSManager] Failed to get Proxmox client for discovery node %s: %v (trying next node)", discoveryNode, err)
				lastErr = err
				continue
			}
			// Try to discover the node where the VM is running
			nodeName, findErr := discoveryClient.FindVMNode(ctx, vmIDInt)
			if findErr == nil {
				// Success! Update VPS record with discovered NodeID
				vps.NodeID = &nodeName
				if err := database.DB.Model(&vps).Update("node_id", nodeName).Error; err != nil {
					logger.Warn("[VPSManager] Failed to update NodeID for VPS %s: %v (continuing with sync)", vpsID, err)
				} else {
					logger.Info("[VPSManager] Discovered and updated NodeID for VPS %s: %s (via discovery node %s)", vpsID, nodeName, discoveryNode)
				}
				break
			}
			// Check if VM was deleted from Proxmox directly
			// FindVMNode returns "VM X not found on any node" when VM doesn't exist
			if strings.Contains(findErr.Error(), "not found on any node") {
				logger.Info("[VPSManager] VM %d not found in Proxmox - marking VPS %s as DELETED", vmIDInt, vpsID)
				vps.Status = 9 // DELETED
				vps.UpdatedAt = time.Now()
				if err := database.DB.Save(&vps).Error; err != nil {
					return fmt.Errorf("failed to update VPS status to DELETED: %w", err)
				}
				return nil
			}
			// Discovery failed on this node, try next one
			logger.Debug("[VPSManager] Failed to discover VM %d via node %s: %v (trying next node)", vmIDInt, discoveryNode, findErr)
			lastErr = findErr
		}
		// If we tried all nodes and none worked, return error
		if nodeName == "" {
			return fmt.Errorf("failed to discover VM node after trying all %d nodes: %w", len(allNodes), lastErr)
		}
	}

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get actual status from Proxmox
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		// Check if VM was deleted from Proxmox directly
		// GetVMStatus returns "VM does not exist" when VM was deleted
		errStr := err.Error()
		if strings.Contains(errStr, "does not exist") {
			logger.Info("[VPSManager] VM %d does not exist in Proxmox - marking VPS %s as DELETED", vmIDInt, vpsID)
			vps.Status = 9 // DELETED
			vps.UpdatedAt = time.Now()
			if err := database.DB.Save(&vps).Error; err != nil {
				return fmt.Errorf("failed to update VPS status to DELETED: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get VM status: %w", err)
	}

	// Map and update status
	vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		return fmt.Errorf("failed to update VPS status: %w", err)
	}

	logger.Info("[VPSManager] Synced VPS %s status from Proxmox: %s -> %d", vpsID, proxmoxStatus, vps.Status)
	return nil
}

// UpdateOrganizationVPSSSHKeys updates SSH keys in cloud-init for all VPS instances in an organization
// This is called when SSH keys are added or removed
func (vm *VPSManager) UpdateOrganizationVPSSSHKeys(ctx context.Context, organizationID string) error {
	return vm.UpdateOrganizationVPSSSHKeysExcluding(ctx, organizationID, "")
}

// UpdateOrganizationVPSSSHKeysExcluding updates SSH keys for all VPS instances in an organization,
// excluding a specific key ID (e.g., when deleting an org-wide key)
func (vm *VPSManager) UpdateOrganizationVPSSSHKeysExcluding(ctx context.Context, organizationID string, excludeKeyID string) error {
	// Get all VPS instances for this organization
	var vpsInstances []database.VPSInstance
	if err := database.DB.Where("organization_id = ? AND deleted_at IS NULL AND instance_id IS NOT NULL", organizationID).Find(&vpsInstances).Error; err != nil {
		return fmt.Errorf("failed to get VPS instances: %w", err)
	}

	if len(vpsInstances) == 0 {
		logger.Info("[VPSManager] No VPS instances found for organization %s, skipping SSH key update", organizationID)
		return nil
	}

	// Update SSH keys for each VPS instance
	successCount := 0
	for _, vps := range vpsInstances {
		if vps.InstanceID == nil {
			continue
		}

		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt == 0 {
			logger.Warn("[VPSManager] Invalid VM ID for VPS %s: %s", vps.ID, *vps.InstanceID)
			continue
		}

		// Get node name from VPS (required)
		nodeName := ""
		if vps.NodeID != nil && *vps.NodeID != "" {
			nodeName = *vps.NodeID
		} else {
			logger.Warn("[VPSManager] VPS %s has no node ID - skipping SSH key update", vps.ID)
			continue
		}

		// Get Proxmox client for the node where VPS is running
		proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
		if err != nil {
			logger.Warn("[VPSManager] Failed to get Proxmox client for node %s (VPS %s): %v", nodeName, vps.ID, err)
			continue
		}

		// Update SSH keys (includes VPS-specific + org-wide), excluding the specified key if provided
		if err := proxmoxClient.UpdateVMSSHKeys(ctx, nodeName, vmIDInt, organizationID, vps.ID, excludeKeyID); err != nil {
			logger.Warn("[VPSManager] Failed to update SSH keys for VM %d (VPS %s): %v", vmIDInt, vps.ID, err)
			continue
		}

		successCount++
	}

	logger.Info("[VPSManager] Updated SSH keys for %d/%d VPS instances in organization %s", successCount, len(vpsInstances), organizationID)
	return nil
}

// UpdateVPSSSHKeys updates SSH keys in cloud-init for a specific VPS instance
// This includes both VPS-specific keys and organization-wide keys
func (vm *VPSManager) UpdateVPSSSHKeys(ctx context.Context, vpsID string) error {
	return vm.UpdateVPSSSHKeysExcluding(ctx, vpsID, "")
}

// UpdateVPSSSHKeysExcluding updates SSH keys in cloud-init for a specific VPS instance,
// excluding a specific key ID (e.g., when deleting a key)
func (vm *VPSManager) UpdateVPSSSHKeysExcluding(ctx context.Context, vpsID string, excludeKeyID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use")
	}

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	// Update SSH keys (includes VPS-specific and org-wide), excluding the specified key if provided
	if err := proxmoxClient.UpdateVMSSHKeys(ctx, nodeName, vmIDInt, vps.OrganizationID, vpsID, excludeKeyID); err != nil {
		return fmt.Errorf("failed to update SSH keys: %w", err)
	}

	logger.Info("[VPSManager] Updated SSH keys for VPS %s (VM %d)", vpsID, vmIDInt)
	return nil
}

// EnableVPSGuestAgent enables QEMU guest agent for a specific VPS instance
func (vm *VPSManager) EnableVPSGuestAgent(ctx context.Context, vpsID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use")
	}

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	// Enable guest agent in VM config
	if err := proxmoxClient.EnableVMGuestAgent(ctx, nodeName, vmIDInt); err != nil {
		return fmt.Errorf("failed to enable guest agent: %w", err)
	}

	logger.Info("[VPSManager] Enabled guest agent for VPS %s (VM %d)", vpsID, vmIDInt)
	return nil
}

// RecoverVPSGuestAgent recovers QEMU guest agent for a specific VPS instance
// This updates both the VM config and cloud-init to ensure guest agent is properly configured
func (vm *VPSManager) RecoverVPSGuestAgent(ctx context.Context, vpsID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use")
	}

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vm.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err)
	}

	// Recover guest agent (updates both VM config and cloud-init)
	if err := proxmoxClient.RecoverVMGuestAgent(ctx, nodeName, vmIDInt, vps.OrganizationID, vpsID); err != nil {
		return fmt.Errorf("failed to recover guest agent: %w", err)
	}

	logger.Info("[VPSManager] Recovered guest agent for VPS %s (VM %d). VM should be rebooted for changes to take effect.", vpsID, vmIDInt)
	return nil
}

// generateMACAddress generates a random MAC address for a VM
// Format: 00:16:3e:XX:XX:XX (QEMU/KVM standard prefix)
func generateMACAddress() string {
	// Generate random bytes for the last 3 octets
	randBytes := make([]byte, 3)
	rand.Read(randBytes)
	return fmt.Sprintf("00:16:3e:%02x:%02x:%02x", randBytes[0], randBytes[1], randBytes[2])
}

// mapProxmoxStatusToVPSStatus maps Proxmox VM status strings to VPSStatus enum values
// Proxmox status values: "running", "stopped", "paused", "suspended", "unknown"
// VPSStatus enum: CREATING=1, STARTING=2, RUNNING=3, STOPPING=4, STOPPED=5, REBOOTING=6, FAILED=7, DELETING=8, DELETED=9
func mapProxmoxStatusToVPSStatus(proxmoxStatus string) int32 {
	switch strings.ToLower(proxmoxStatus) {
	case "running":
		return 3 // RUNNING
	case "stopped":
		return 5 // STOPPED
	case "paused", "suspended":
		return 5 // STOPPED (treat paused/suspended as stopped)
	default:
		// For unknown or other statuses, default to CREATING
		// This handles cases where VM is still initializing
		return 1 // CREATING
	}
}

// ImportVPSResult represents the result of importing a single VPS
type ImportVPSResult struct {
	VPS     *database.VPSInstance
	Error   error
	Skipped bool // True if VPS was skipped (already exists or doesn't belong to org)
}

// ImportVPS imports missing VPS instances from Proxmox that belong to the specified organization
// This function:
// 1. Lists all VMs from Proxmox
// 2. Filters by description containing "Managed by Obiente Cloud"
// 3. Parses VPS ID and Org ID from description
// 4. Verifies org ID matches the requesting organization (SECURITY: prevents importing VPS from other orgs)
// 5. Checks if VPS already exists in DB
// 6. Gets VM config to extract resource specs
// 7. Imports missing VPS with proper ownership
func (vm *VPSManager) ImportVPS(ctx context.Context, organizationID string) ([]ImportVPSResult, error) {
	// Try all nodes sequentially until one succeeds for listing VMs (any node can list all VMs in cluster)
	allNodes, err := GetAllProxmoxNodeNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get Proxmox nodes for import: %w", err)
	}

	var proxmoxClient *ProxmoxClient
	var lastErr error
	for _, discoveryNode := range allNodes {
		client, err := vm.GetProxmoxClientForNode(discoveryNode)
		if err != nil {
			logger.Debug("[VPSManager] Failed to get Proxmox client for discovery node %s: %v (trying next node)", discoveryNode, err)
			lastErr = err
			continue
		}
		proxmoxClient = client
		logger.Info("[VPSManager] Using node %s for VM listing", discoveryNode)
		break
	}
	if proxmoxClient == nil {
		return nil, fmt.Errorf("failed to get Proxmox client after trying all %d nodes: %w", len(allNodes), lastErr)
	}

	// List all VMs from Proxmox
	logger.Info("[VPSManager] Listing all VMs from Proxmox for import")
	vms, err := proxmoxClient.ListAllVMsWithDescriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs from Proxmox: %w", err)
	}

	logger.Info("[VPSManager] Found %d VMs in Proxmox, filtering for Obiente Cloud managed VMs", len(vms))

	var results []ImportVPSResult
	importedCount := 0
	skippedCount := 0

	// Process each VM
	for _, proxmoxVM := range vms {
		// Skip VMs without Obiente Cloud description
		if !strings.Contains(proxmoxVM.Description, "Managed by Obiente Cloud") {
			continue
		}

		// Parse description to extract VPS ID and Org ID
		vpsID, orgID, displayName, creatorID, ok := parseVPSDescription(proxmoxVM.Description)
		if !ok {
			logger.Warn("[VPSManager] Failed to parse VPS description for VM %d: %s", proxmoxVM.VMID, proxmoxVM.Description)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("failed to parse VPS description"),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// SECURITY: Verify organization ID matches (prevents importing VPS from other organizations)
		if orgID != organizationID {
			logger.Debug("[VPSManager] Skipping VM %d (VPS %s): belongs to org %s, not %s", proxmoxVM.VMID, vpsID, orgID, organizationID)
			results = append(results, ImportVPSResult{
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// Check if VPS already exists in database
		// Skip VPSes that are soft-deleted, in DELETING status (8), or DELETED status (9)
		var existingVPS database.VPSInstance
		err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&existingVPS).Error
		if err == nil {
			// VPS exists - check if it's in DELETING or DELETED status
			if existingVPS.Status == 8 || existingVPS.Status == 9 {
				// VPS is being deleted or already deleted, skip import
				logger.Debug("[VPSManager] VPS %s is in DELETING/DELETED status (status: %d), skipping import", vpsID, existingVPS.Status)
				results = append(results, ImportVPSResult{
					VPS:     &existingVPS,
					Skipped: true,
				})
				skippedCount++
				continue
			}
			// VPS already exists and is not being deleted, skip
			logger.Debug("[VPSManager] VPS %s already exists in database, skipping", vpsID)
			results = append(results, ImportVPSResult{
				VPS:     &existingVPS,
				Skipped: true,
			})
			skippedCount++
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// Database error
			logger.Warn("[VPSManager] Database error checking for VPS %s: %v", vpsID, err)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("database error: %w", err),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// Also check if another VPS with the same VM ID already exists
		// This prevents importing the same Proxmox VM under a different VPS ID
		vmIDStr := fmt.Sprintf("%d", proxmoxVM.VMID)
		var existingVPSByVMID database.VPSInstance
		err = database.DB.Where("instance_id = ? AND deleted_at IS NULL", vmIDStr).First(&existingVPSByVMID).Error
		if err == nil {
			// VM ID already exists in another VPS record
			logger.Warn("[VPSManager] VM ID %d is already associated with VPS %s, skipping duplicate import for VPS %s",
				proxmoxVM.VMID, existingVPSByVMID.ID, vpsID)
			results = append(results, ImportVPSResult{
				VPS:     &existingVPSByVMID,
				Skipped: true,
				Error:   fmt.Errorf("VM ID %d already associated with VPS %s", proxmoxVM.VMID, existingVPSByVMID.ID),
			})
			skippedCount++
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// Database error
			logger.Warn("[VPSManager] Database error checking for VM ID %d: %v", proxmoxVM.VMID, err)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("database error checking VM ID: %w", err),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// VPS doesn't exist, import it
		logger.Info("[VPSManager] Importing VPS %s (VM %d) from Proxmox", vpsID, proxmoxVM.VMID)

		// Get VM config to extract resource specifications
		vmConfig, err := proxmoxClient.GetVMConfig(ctx, proxmoxVM.NodeName, proxmoxVM.VMID)
		if err != nil {
			logger.Warn("[VPSManager] Failed to get VM config for VM %d: %v", proxmoxVM.VMID, err)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("failed to get VM config: %w", err),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// Extract resource specifications from VM config
		cpuCores := int32(1)
		if cores, ok := vmConfig["cores"].(float64); ok {
			cpuCores = int32(cores)
		} else if cores, ok := vmConfig["cores"].(int); ok {
			cpuCores = int32(cores)
		}

		memoryMB := int64(512)
		if memory, ok := vmConfig["memory"].(float64); ok {
			memoryMB = int64(memory)
		} else if memory, ok := vmConfig["memory"].(int); ok {
			memoryMB = int64(memory)
		}
		memoryBytes := memoryMB * 1024 * 1024

		// Extract disk size from disk configuration
		diskBytes := int64(20 * 1024 * 1024 * 1024) // Default 20GB
		for _, diskKey := range []string{"scsi0", "virtio0", "sata0", "ide0"} {
			if diskConfig, ok := vmConfig[diskKey].(string); ok && diskConfig != "" {
				// Parse disk size from config (e.g., "local-lvm:vm-100-disk-0,size=50G")
				if strings.Contains(diskConfig, "size=") {
					sizePart := strings.Split(diskConfig, "size=")
					if len(sizePart) > 1 {
						sizeStr := strings.TrimSpace(sizePart[1])
						// Remove any trailing commas or other parameters
						if idx := strings.Index(sizeStr, ","); idx != -1 {
							sizeStr = sizeStr[:idx]
						}
						// Parse size (e.g., "50G" -> 50 * 1024^3 bytes)
						var sizeValue float64
						var unit string
						if _, err := fmt.Sscanf(sizeStr, "%f%s", &sizeValue, &unit); err == nil {
							switch strings.ToUpper(unit) {
							case "G", "GB":
								diskBytes = int64(sizeValue * 1024 * 1024 * 1024)
							case "M", "MB":
								diskBytes = int64(sizeValue * 1024 * 1024)
							case "T", "TB":
								diskBytes = int64(sizeValue * 1024 * 1024 * 1024 * 1024)
							}
						}
					}
				}
				break // Use first disk found
			}
		}

		// Extract region from node name using PROXMOX_REGION_NODES mapping
		region := ""
		if proxmoxVM.NodeName != "" {
			// Parse region-to-node mapping and create reverse lookup (node -> region)
			// Need to parse the env var directly to handle multiple nodes per region (comma-separated)
			envValue := os.Getenv("PROXMOX_REGION_NODES")
			if envValue != "" {
				// Parse semicolon-separated region mappings
				regionStrings := strings.Split(envValue, ";")
				for _, regionStr := range regionStrings {
					regionStr = strings.TrimSpace(regionStr)
					if regionStr == "" {
						continue
					}

					// Parse "regionID:nodeName" or "regionID:node1,node2" format
					if strings.Contains(regionStr, ":") {
						parts := strings.SplitN(regionStr, ":", 2)
						if len(parts) == 2 {
							regionID := strings.TrimSpace(parts[0])
							nodeNamesStr := strings.TrimSpace(parts[1])
							// Handle comma-separated node names (multiple nodes per region)
							nodeNames := strings.Split(nodeNamesStr, ",")
							for _, nodeName := range nodeNames {
								nodeName = strings.TrimSpace(nodeName)
								if nodeName == proxmoxVM.NodeName {
									region = regionID
									logger.Info("[VPSManager] Mapped node %s to region %s for VPS %s", proxmoxVM.NodeName, region, vpsID)
									break
								}
							}
							if region != "" {
								break
							}
						}
					}
				}
			}
			if region == "" {
				logger.Debug("[VPSManager] No region mapping found for node %s, leaving region empty for VPS %s", proxmoxVM.NodeName, vpsID)
			}
		}

		// Extract image by checking templates on the node and matching to known patterns
		image := int32(0) // VPS_IMAGE_UNSPECIFIED
		// Try to find templates on the node and match their names to known image patterns
		// This helps identify which OS template was used to create this VM
		templatesResp, templatesErr := proxmoxClient.apiRequest(ctx, "GET", fmt.Sprintf("/nodes/%s/qemu", proxmoxVM.NodeName), nil)
		if templatesErr == nil && templatesResp != nil && templatesResp.StatusCode == http.StatusOK {
			defer templatesResp.Body.Close()
			var templatesData struct {
				Data []struct {
					Vmid     int    `json:"vmid"`
					Name     string `json:"name"`
					Template int    `json:"template"`
				} `json:"data"`
			}
			if err := json.NewDecoder(templatesResp.Body).Decode(&templatesData); err == nil {
				// Check each template to see if it matches known patterns
				// Template names follow patterns: "ubuntu-22.04-standard", "ubuntu-24.04-standard", etc.
				for _, tmpl := range templatesData.Data {
					if tmpl.Template == 1 {
						templateName := strings.ToLower(tmpl.Name)
						// Match template names to image enum values
						if strings.Contains(templateName, "ubuntu-22.04") || strings.Contains(templateName, "ubuntu22.04") {
							image = 1 // UBUNTU_22_04
							logger.Info("[VPSManager] Found template '%s' matching UBUNTU_22_04 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						} else if strings.Contains(templateName, "ubuntu-24.04") || strings.Contains(templateName, "ubuntu24.04") {
							image = 2 // UBUNTU_24_04
							logger.Info("[VPSManager] Found template '%s' matching UBUNTU_24_04 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						} else if strings.Contains(templateName, "debian-12") || strings.Contains(templateName, "debian12") {
							image = 3 // DEBIAN_12
							logger.Info("[VPSManager] Found template '%s' matching DEBIAN_12 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						} else if strings.Contains(templateName, "debian-13") || strings.Contains(templateName, "debian13") {
							image = 4 // DEBIAN_13
							logger.Info("[VPSManager] Found template '%s' matching DEBIAN_13 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						} else if strings.Contains(templateName, "rocky") || strings.Contains(templateName, "rockylinux") {
							image = 5 // ROCKY_LINUX_9
							logger.Info("[VPSManager] Found template '%s' matching ROCKY_LINUX_9 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						} else if strings.Contains(templateName, "alma") || strings.Contains(templateName, "almalinux") {
							image = 6 // ALMA_LINUX_9
							logger.Info("[VPSManager] Found template '%s' matching ALMA_LINUX_9 for VM %d", tmpl.Name, proxmoxVM.VMID)
							break
						}
					}
				}
			}
		}

		// If we couldn't determine from templates, try VM name as fallback
		if image == 0 {
			vmName := strings.ToLower(proxmoxVM.Name)
			if strings.Contains(vmName, "ubuntu-22.04") || strings.Contains(vmName, "ubuntu22") {
				image = 1 // UBUNTU_22_04
				logger.Info("[VPSManager] Matched VM name '%s' to UBUNTU_22_04 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			} else if strings.Contains(vmName, "ubuntu-24.04") || strings.Contains(vmName, "ubuntu24") {
				image = 2 // UBUNTU_24_04
				logger.Info("[VPSManager] Matched VM name '%s' to UBUNTU_24_04 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			} else if strings.Contains(vmName, "debian-12") || strings.Contains(vmName, "debian12") {
				image = 3 // DEBIAN_12
				logger.Info("[VPSManager] Matched VM name '%s' to DEBIAN_12 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			} else if strings.Contains(vmName, "debian-13") || strings.Contains(vmName, "debian13") {
				image = 4 // DEBIAN_13
				logger.Info("[VPSManager] Matched VM name '%s' to DEBIAN_13 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			} else if strings.Contains(vmName, "rocky") || strings.Contains(vmName, "rockylinux") {
				image = 5 // ROCKY_LINUX_9
				logger.Info("[VPSManager] Matched VM name '%s' to ROCKY_LINUX_9 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			} else if strings.Contains(vmName, "alma") || strings.Contains(vmName, "almalinux") {
				image = 6 // ALMA_LINUX_9
				logger.Info("[VPSManager] Matched VM name '%s' to ALMA_LINUX_9 for VM %d", proxmoxVM.Name, proxmoxVM.VMID)
			}
		}

		if image == 0 {
			logger.Debug("[VPSManager] Could not determine image for VM %d (name: %s), leaving as UNSPECIFIED", proxmoxVM.VMID, proxmoxVM.Name)
		}

		// Extract size (default to empty)
		size := ""

		// Map Proxmox status to VPS status
		vpsStatus := mapProxmoxStatusToVPSStatus(proxmoxVM.Status)

		// NodeID is required - skip VMs without node name
		if proxmoxVM.NodeName == "" {
			logger.Warn("[VPSManager] VM %d has no node name - skipping import for VPS %s", proxmoxVM.VMID, vpsID)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("VM %d has no node name - cannot import VPS without node information", proxmoxVM.VMID),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		// Create VPS instance record
		nodeID := &proxmoxVM.NodeName
		vpsInstance := &database.VPSInstance{
			ID:             vpsID,
			Name:           displayName,
			Description:    nil, // Description is not stored in DB, only in Proxmox
			Status:         vpsStatus,
			Region:         region,
			Image:          image,
			ImageID:        nil,
			Size:           size,
			CPUCores:       cpuCores,
			MemoryBytes:    memoryBytes,
			DiskBytes:      diskBytes,
			InstanceID:     &vmIDStr,
			NodeID:         nodeID, // Store Proxmox node name (required)
			SSHKeyID:       nil,
			OrganizationID: organizationID,
			CreatedBy:      creatorID,
			CreatedAt:      time.Now(), // Use current time as we don't know actual creation time
			UpdatedAt:      time.Now(),
			Metadata:       "{}",
			IPv4Addresses:  "[]",
			IPv6Addresses:  "[]",
		}

		// Save to database
		if err := database.DB.Create(vpsInstance).Error; err != nil {
			logger.Warn("[VPSManager] Failed to create VPS record for %s: %v", vpsID, err)
			results = append(results, ImportVPSResult{
				Error:   fmt.Errorf("failed to create VPS record: %w", err),
				Skipped: true,
			})
			skippedCount++
			continue
		}

		logger.Info("[VPSManager] Successfully imported VPS %s (VM %d) from Proxmox", vpsID, proxmoxVM.VMID)
		results = append(results, ImportVPSResult{
			VPS:     vpsInstance,
			Skipped: false,
		})
		importedCount++
	}

	logger.Info("[VPSManager] Import completed: %d imported, %d skipped", importedCount, skippedCount)
	return results, nil
}

// SyncAllVPSStatuses syncs status and IP addresses for all VPS instances from Proxmox
// This is used for periodic background sync to detect deleted VPSs and update IP addresses
// Returns a map of VPS IDs that were marked as DELETED (oldStatus -> newStatus)
func (vm *VPSManager) SyncAllVPSStatuses(ctx context.Context) (map[string]int32, error) {
	// Get all VPS instances that have an instance ID (are provisioned)
	var vpsInstances []database.VPSInstance
	if err := database.DB.Where("instance_id IS NOT NULL AND deleted_at IS NULL").Find(&vpsInstances).Error; err != nil {
		return nil, fmt.Errorf("failed to get VPS instances: %w", err)
	}

	logger.Info("[VPSManager] Syncing status and IPs for %d VPS instances", len(vpsInstances))

	syncedCount := 0
	deletedCount := 0
	ipUpdatedCount := 0
	errorCount := 0
	deletedVPSs := make(map[string]int32) // vpsID -> oldStatus

	for _, vps := range vpsInstances {
		// Get old status for notification purposes
		oldStatus := vps.Status

		// Sync status (this will mark as DELETED if VM doesn't exist)
		if err := vm.SyncVPSStatusFromProxmox(ctx, vps.ID); err != nil {
			// Log error but continue with other VPSs
			logger.Warn("[VPSManager] Failed to sync VPS %s: %v", vps.ID, err)
			errorCount++
			continue
		}

		// Check if status changed to DELETED
		var updatedVPS database.VPSInstance
		if err := database.DB.Where("id = ?", vps.ID).First(&updatedVPS).Error; err == nil {
			if updatedVPS.Status == 9 && oldStatus != 9 { // DELETED
				deletedCount++
				deletedVPSs[vps.ID] = oldStatus
				logger.Info("[VPSManager] VPS %s marked as DELETED during sync", vps.ID)

				// Clear IP addresses and instance ID to prevent stale data when VM ID is reused
				if err := database.DB.Model(&updatedVPS).Updates(map[string]interface{}{
					"ipv4_addresses": "[]",
					"ipv6_addresses": "[]",
					"instance_id":    nil,
				}).Error; err != nil {
					logger.Warn("[VPSManager] Failed to clear IP addresses for deleted VPS %s: %v", vps.ID, err)
				} else {
					logger.Info("[VPSManager] Cleared IP addresses and instance ID for deleted VPS %s", vps.ID)
				}
			} else if updatedVPS.Status != oldStatus {
				syncedCount++
			}

			// For running VPSs, also sync IP addresses
			// Status 1 = RUNNING (from mapProxmoxStatusToVPSStatus)
			if updatedVPS.Status == 1 {
				// Use a timeout for IP fetching to not block the sync
				ipCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, _, err := vm.GetVPSIPAddresses(ipCtx, vps.ID)
				cancel()
				if err != nil {
					logger.Debug("[VPSManager] Failed to sync IPs for VPS %s: %v", vps.ID, err)
				} else {
					ipUpdatedCount++
				}
			}
		}
	}

	logger.Info("[VPSManager] Sync completed: %d status changed, %d IPs updated, %d marked as deleted, %d errors", syncedCount, ipUpdatedCount, deletedCount, errorCount)
	return deletedVPSs, nil
}

// ImportMissingVPSForAllOrgs imports missing VPS instances for all organizations
// This is used for periodic background sync to import VPSs that exist in Proxmox but not in the database
func (vm *VPSManager) ImportMissingVPSForAllOrgs(ctx context.Context) error {
	// Try all nodes sequentially until one succeeds for listing VMs (any node can list all VMs in cluster)
	allNodes, err := GetAllProxmoxNodeNames()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox nodes for import: %w", err)
	}

	var proxmoxClient *ProxmoxClient
	var lastErr error
	for _, discoveryNode := range allNodes {
		client, err := vm.GetProxmoxClientForNode(discoveryNode)
		if err != nil {
			logger.Debug("[VPSManager] Failed to get Proxmox client for discovery node %s: %v (trying next node)", discoveryNode, err)
			lastErr = err
			continue
		}
		proxmoxClient = client
		logger.Info("[VPSManager] Using node %s for VM listing", discoveryNode)
		break
	}
	if proxmoxClient == nil {
		return fmt.Errorf("failed to get Proxmox client after trying all %d nodes: %w", len(allNodes), lastErr)
	}

	// List all VMs from Proxmox
	logger.Info("[VPSManager] Listing all VMs from Proxmox for import")
	vms, err := proxmoxClient.ListAllVMsWithDescriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs from Proxmox: %w", err)
	}

	logger.Info("[VPSManager] Found %d VMs in Proxmox, filtering for Obiente Cloud managed VMs", len(vms))

	// Collect unique organization IDs from VMs that are managed by Obiente Cloud
	orgIDSet := make(map[string]bool)
	for _, proxmoxVM := range vms {
		// Skip VMs without Obiente Cloud description
		if !strings.Contains(proxmoxVM.Description, "Managed by Obiente Cloud") {
			continue
		}

		// Parse description to extract Org ID
		_, orgID, _, _, ok := parseVPSDescription(proxmoxVM.Description)
		if ok && orgID != "" {
			orgIDSet[orgID] = true
		}
	}

	// Convert set to slice
	var orgIDs []string
	for orgID := range orgIDSet {
		orgIDs = append(orgIDs, orgID)
	}

	if len(orgIDs) == 0 {
		logger.Debug("[VPSManager] No organizations found in Proxmox VMs for import")
		return nil
	}

	logger.Info("[VPSManager] Importing missing VPSs for %d organizations found in Proxmox", len(orgIDs))

	totalImported := 0
	totalSkipped := 0

	for _, orgID := range orgIDs {
		results, err := vm.ImportVPS(ctx, orgID)
		if err != nil {
			logger.Warn("[VPSManager] Failed to import VPSs for org %s: %v", orgID, err)
			continue
		}

		for _, result := range results {
			if result.Error != nil {
				logger.Warn("[VPSManager] Import error for org %s: %v", orgID, result.Error)
			} else if result.Skipped {
				totalSkipped++
			} else if result.VPS != nil {
				totalImported++
			}
		}
	}

	logger.Info("[VPSManager] Import for all orgs completed: %d imported, %d skipped", totalImported, totalSkipped)
	return nil
}

// GetVPSLeases retrieves DHCP lease information from the database
func (vm *VPSManager) GetVPSLeases(ctx context.Context, organizationID string, vpsID *string) ([]*vpsv1.VPSLease, error) {
	var leases []database.DHCPLease
	query := database.DB.WithContext(ctx).Where("organization_id = ?", organizationID)

	if vpsID != nil && *vpsID != "" {
		query = query.Where("vps_id = ?", *vpsID)
	}

	if err := query.Find(&leases).Error; err != nil {
		return nil, fmt.Errorf("failed to query DHCP leases: %w", err)
	}

	// Convert database leases to proto format
	result := make([]*vpsv1.VPSLease, 0, len(leases))
	for _, lease := range leases {
		result = append(result, &vpsv1.VPSLease{
			VpsId:          lease.VPSID,
			OrganizationId: lease.OrganizationID,
			MacAddress:     lease.MACAddress,
			IpAddress:      lease.IPAddress,
			ExpiresAt:      timestamppb.New(lease.ExpiresAt),
			IsPublic:       lease.IsPublic,
		})
	}

	return result, nil
}

// RegisterLease creates or updates a DHCP lease in the database
func (vm *VPSManager) RegisterLease(ctx context.Context, req *vpsv1.RegisterLeaseRequest, gatewayNode string) error {
	// Verify VPS exists and belongs to the organization
	var vps database.VPSInstance
	if err := database.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", req.VpsId, req.OrganizationId).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found or organization mismatch: %w", err)
	}

	// Create or update the lease
	lease := database.DHCPLease{
		VPSID:          req.VpsId,
		OrganizationID: req.OrganizationId,
		MACAddress:     req.MacAddress,
		IPAddress:      req.IpAddress,
		IsPublic:       req.IsPublic,
		ExpiresAt:      req.ExpiresAt.AsTime(),
		GatewayNode:    gatewayNode,
	}

	// Check if lease with this MAC already exists
	var existing database.DHCPLease
	result := database.DB.WithContext(ctx).Where("mac_address = ?", req.MacAddress).First(&existing)

	if result.Error == nil {
		// Update existing lease
		lease.ID = existing.ID
		lease.CreatedAt = existing.CreatedAt
		if err := database.DB.WithContext(ctx).Save(&lease).Error; err != nil {
			return fmt.Errorf("failed to update lease: %w", err)
		}
		logger.Info("Updated DHCP lease: VPS=%s MAC=%s IP=%s", req.VpsId, req.MacAddress, req.IpAddress)
	} else {
		// Create new lease
		lease.ID = uuid.New().String()
		if err := database.DB.WithContext(ctx).Create(&lease).Error; err != nil {
			return fmt.Errorf("failed to create lease: %w", err)
		}
		logger.Info("Created DHCP lease: VPS=%s MAC=%s IP=%s", req.VpsId, req.MacAddress, req.IpAddress)
	}

	return nil
}

// SyncLeasesFromGateways pulls allocations from all configured gateways and upserts them into the database.
// This keeps the API's lease view in sync while the API is the connection initiator.
func (vm *VPSManager) SyncLeasesFromGateways(ctx context.Context) error {
	mapping, err := parseNodeGatewayMapping()
	if err != nil {
		return fmt.Errorf("failed to parse gateway mapping: %w", err)
	}

	if len(mapping) == 0 {
		return fmt.Errorf("no gateways configured via VPS_NODE_GATEWAY_ENDPOINTS")
	}

	for nodeName := range mapping {
		client, err := vm.GetGatewayClientForNode(nodeName)
		if err != nil {
			logger.Warn("[LeaseSync] Skipping node %s: %v", nodeName, err)
			continue
		}

		nodeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		allocations, err := client.ListIPs(nodeCtx, "", "")
		cancel()
		if err != nil {
			logger.Warn("[LeaseSync] Failed to list IPs from node %s: %v", nodeName, err)
			continue
		}

		for _, alloc := range allocations {
			if alloc == nil || alloc.LeaseExpires == nil {
				continue
			}

			req := &vpsv1.RegisterLeaseRequest{
				VpsId:          alloc.VpsId,
				OrganizationId: alloc.OrganizationId,
				MacAddress:     alloc.MacAddress,
				IpAddress:      alloc.IpAddress,
				ExpiresAt:      alloc.LeaseExpires,
				IsPublic:       false,
			}

			if err := vm.RegisterLease(ctx, req, nodeName); err != nil {
				logger.Warn("[LeaseSync] Failed to upsert lease for VPS %s (%s) from node %s: %v", alloc.VpsId, alloc.IpAddress, nodeName, err)
			}
		}
	}

	return nil
}

// ReleaseLease removes a DHCP lease from the database
func (vm *VPSManager) ReleaseLease(ctx context.Context, req *vpsv1.ReleaseLeaseRequest, gatewayNode string) error {
	query := database.DB.WithContext(ctx).Where("vps_id = ?", req.VpsId)

	// Optional: verify MAC address if provided
	if req.MacAddress != "" {
		query = query.Where("mac_address = ?", req.MacAddress)
	}

	if err := query.Delete(&database.DHCPLease{}).Error; err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	logger.Info("Released DHCP lease: VPS=%s MAC=%s", req.VpsId, req.MacAddress)
	return nil
}
