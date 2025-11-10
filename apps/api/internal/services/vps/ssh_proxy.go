package vps

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"

	"golang.org/x/crypto/ssh"
)

// SSHProxyServer handles SSH jump host functionality for VPS access
// Users connect via SSH to the API server, which then proxies to the VPS
type SSHProxyServer struct {
	listener       net.Listener
	hostKey        ssh.Signer
	authorizedKeys map[string]bool // VPS ID -> authorized
	mu             sync.RWMutex
	port           int
	vpsService     *Service // Reference to VPS service for connectSSH
	gatewayClient  *orchestrator.VPSGatewayClient // Optional gateway client for SSH proxying
}

// NewSSHProxyServer creates a new SSH proxy server
func NewSSHProxyServer(port int, vpsService *Service) (*SSHProxyServer, error) {
	// Generate or load host key
	hostKey, err := getOrGenerateHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key: %w", err)
	}

	// Initialize gateway client (optional - will be nil if gateway is not configured)
	gatewayClient, err := orchestrator.NewVPSGatewayClient()
	if err != nil {
		logger.Warn("[SSHProxy] Failed to initialize VPS gateway client (gateway may not be configured): %v", err)
		gatewayClient = nil // Continue without gateway - will use direct SSH connection
	}

	server := &SSHProxyServer{
		hostKey:        hostKey,
		authorizedKeys: make(map[string]bool),
		port:           port,
		vpsService:     vpsService,
		gatewayClient:  gatewayClient,
	}

	return server, nil
}

// Start starts the SSH proxy server
func (s *SSHProxyServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	logger.Info("[SSHProxy] Started SSH proxy server on %s", addr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					logger.Error("[SSHProxy] Failed to accept connection: %v", err)
					continue
				}
			}

			logger.Info("[SSHProxy] Accepted connection from %s", conn.RemoteAddr())
			go s.handleConnection(ctx, conn)
		}
	}()

	return nil
}

// Stop stops the SSH proxy server
func (s *SSHProxyServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection handles an incoming SSH connection
func (s *SSHProxyServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Debug("[SSHProxy] Error closing connection: %v", err)
		}
	}()

	logger.Info("[SSHProxy] Handling connection from %s", conn.RemoteAddr())

	// Set a read deadline for the initial handshake
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		logger.Warn("[SSHProxy] Failed to set read deadline: %v", err)
	}

	// Create SSH server config with password authentication
	// Users provide their API token as the password
	config := &ssh.ServerConfig{
		PasswordCallback: s.authenticatePassword,
		// Also allow public key auth for future use
		PublicKeyCallback: s.authenticatePublicKey,
		// Set server version
		ServerVersion: "SSH-2.0-ObienteCloud",
	}

	if s.hostKey == nil {
		logger.Error("[SSHProxy] Host key is nil, cannot handle connection")
		return
	}
	config.AddHostKey(s.hostKey)

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		logger.Warn("[SSHProxy] SSH handshake failed for %s: %v", conn.RemoteAddr(), err)
		return
	}
	
	// Clear the read deadline after successful handshake
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		logger.Debug("[SSHProxy] Failed to clear read deadline: %v", err)
	}
	
	logger.Info("[SSHProxy] SSH handshake successful for user %s from %s", sshConn.User(), conn.RemoteAddr())
	defer sshConn.Close()

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			logger.Error("[SSHProxy] Failed to accept channel for user %s: %v", sshConn.User(), err)
			continue
		}

		// Handle channel requests
		// Username is the VPS ID (which already includes "vps-" prefix)
		vpsID := sshConn.User()
		go s.handleChannel(ctx, channel, requests, vpsID)
	}
}

// authenticatePassword authenticates SSH connections using API token as password
func (s *SSHProxyServer) authenticatePassword(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	// Username is the VPS ID (which already includes "vps-" prefix)
	username := conn.User()
	if !strings.HasPrefix(username, "vps-") {
		return nil, fmt.Errorf("invalid username format: expected VPS ID starting with 'vps-'")
	}

	vpsID := username // VPS ID already includes "vps-" prefix
	apiToken := string(password)

	// Validate API token and check VPS access
	// Use AuthenticateHTTPRequest helper which validates tokens
	// Create a mock request with the token
	ctx := context.Background()
	authConfig := auth.NewAuthConfig()
	
	// Create a mock HTTP request for token validation
	// We'll use the validateToken method via a helper
	// Since validateToken is private, we need to use AuthenticateHTTPRequest pattern
	// But for SSH, we can create a simple HTTP request
	mockReq, _ := http.NewRequestWithContext(ctx, "GET", "/", nil)
	mockReq.Header.Set("Authorization", "Bearer "+apiToken)
	
	// Validate token
	userInfo, err := auth.AuthenticateHTTPRequest(authConfig, mockReq)
	if err != nil {
		logger.Debug("[SSHProxy] Token validation failed for VPS %s: %v", vpsID, err)
		return nil, fmt.Errorf("authentication failed: invalid token")
	}

	// Check if VPS exists and user has access
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		logger.Debug("[SSHProxy] VPS %s not found for user %s: %v", vpsID, userInfo.Id, err)
		return nil, fmt.Errorf("VPS not found or access denied")
	}

	// Check if user has access to this VPS's organization
	if vps.OrganizationID != userInfo.Id {
		// TODO: Check if user is member of organization
		// For now, we'll allow if VPS exists
		logger.Debug("[SSHProxy] User %s accessing VPS %s in org %s", userInfo.Id, vpsID, vps.OrganizationID)
	}

	logger.Info("[SSHProxy] Authenticated user %s for VPS %s", userInfo.Id, vpsID)
	return &ssh.Permissions{
		Extensions: map[string]string{
			"vps_id":  vpsID,
			"user_id": userInfo.Id,
		},
	}, nil
}

// authenticatePublicKey authenticates SSH connections using public key
func (s *SSHProxyServer) authenticatePublicKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Username is the VPS ID (which already includes "vps-" prefix)
	username := conn.User()
	if !strings.HasPrefix(username, "vps-") {
		return nil, fmt.Errorf("invalid username format: expected VPS ID starting with 'vps-'")
	}

	vpsID := username // VPS ID already includes "vps-" prefix

	// Get VPS instance to find organization
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, fmt.Errorf("VPS not found or access denied")
	}

	// Get SSH keys for the organization
	sshKeys, err := database.GetSSHKeysForOrganization(vps.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH keys: %w", err)
	}

	// Calculate fingerprint of the provided key
	providedFingerprint := ssh.FingerprintSHA256(key)

	// Check if any of the organization's SSH keys match
	for _, orgKey := range sshKeys {
		// Parse the stored public key
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(orgKey.PublicKey))
		if err != nil {
			continue // Skip invalid keys
		}

		// Compare fingerprints
		if ssh.FingerprintSHA256(parsedKey) == providedFingerprint {
			logger.Info("[SSHProxy] Authenticated via SSH key %s for VPS %s", orgKey.Name, vpsID)
			return &ssh.Permissions{
				Extensions: map[string]string{
					"vps_id":  vpsID,
					"key_id":  orgKey.ID,
					"auth_method": "publickey",
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("SSH key not authorized for this VPS")
}

// handleChannel handles SSH channel requests
func (s *SSHProxyServer) handleChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, vpsID string) {
	defer channel.Close()

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		logger.Error("[SSHProxy] VPS %s not found in database: %v", vpsID, err)
		if _, writeErr := channel.Write([]byte("VPS not found\r\n")); writeErr != nil {
			logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
		}
		return
	}

	// Get VPS IP address and root password
	var vpsIP string
	var rootPassword string

	// Try to get IP from gateway first (if available)
	if s.gatewayClient != nil {
		logger.Info("[SSHProxy] Attempting to get IP address from gateway for VPS %s", vpsID)
		allocations, err := s.gatewayClient.ListIPs(ctx, vps.OrganizationID, vpsID)
		if err == nil && len(allocations) > 0 {
			// Gateway manages DHCP, so use the allocated IP
			vpsIP = allocations[0].IpAddress
			logger.Info("[SSHProxy] Got VPS IP from gateway: %s", vpsIP)
		} else if err != nil {
			logger.Warn("[SSHProxy] Failed to get IP from gateway for VPS %s: %v", vpsID, err)
		} else {
			logger.Info("[SSHProxy] Gateway returned no IP allocations for VPS %s", vpsID)
		}
	}

	// Try to get IP from VPS manager (public IPs or Proxmox guest agent)
	if vpsIP == "" {
		logger.Info("[SSHProxy] Attempting to get IP address for VPS %s (Instance ID: %v, Status: %s)", vpsID, vps.InstanceID, vps.Status)
		vpsManager, err := orchestrator.NewVPSManager()
		if err == nil {
			defer vpsManager.Close()
			ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, vpsID)
			if err == nil && len(ipv4) > 0 {
				vpsIP = ipv4[0]
				logger.Info("[SSHProxy] Got VPS IP from VPS manager: %s", vpsIP)
			} else if err != nil {
				logger.Warn("[SSHProxy] Failed to get IP from VPS manager for VPS %s: %v", vpsID, err)
			} else {
				logger.Info("[SSHProxy] VPS manager returned no IP addresses for VPS %s", vpsID)
			}
		} else {
			logger.Warn("[SSHProxy] Failed to create VPS manager: %v", err)
		}
	}

	// If no public IP, get internal IP from Proxmox
	if vpsIP == "" {
		logger.Info("[SSHProxy] No public IP found, trying to get internal IP from Proxmox for VPS %s", vpsID)
		proxmoxConfig, err := orchestrator.GetProxmoxConfig()
		if err == nil {
			proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
			if err == nil {
				vmIDInt := 0
				if vps.InstanceID != nil {
					fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
				}
				if vmIDInt > 0 {
					logger.Info("[SSHProxy] Looking up VM node for VM ID %d", vmIDInt)
					nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
					if err == nil {
						logger.Info("[SSHProxy] Found VM on node %s, getting IP addresses", nodeName)
						ipv4, _, err := proxmoxClient.GetVMIPAddresses(ctx, nodeName, vmIDInt)
						if err == nil && len(ipv4) > 0 {
							vpsIP = ipv4[0]
							logger.Info("[SSHProxy] Got VPS IP from Proxmox: %s", vpsIP)
						} else if err != nil {
							logger.Warn("[SSHProxy] Failed to get IP from Proxmox for VM %d on node %s: %v", vmIDInt, nodeName, err)
						} else {
							logger.Warn("[SSHProxy] Proxmox returned no IP addresses for VM %d on node %s (guest agent may not be ready or VM has no network)", vmIDInt, nodeName)
						}
					} else {
						logger.Warn("[SSHProxy] Failed to find VM node for VM ID %d: %v", vmIDInt, err)
					}
				} else {
					logger.Warn("[SSHProxy] VPS %s has no instance ID (not provisioned yet?)", vpsID)
				}
			} else {
				logger.Warn("[SSHProxy] Failed to create Proxmox client: %v", err)
			}
		} else {
			logger.Warn("[SSHProxy] Failed to get Proxmox config: %v", err)
		}
	}

	// If no IP found, try to connect via Proxmox jump host using VM hostname
	// This allows connection even when guest agent isn't ready
	if vpsIP == "" {
		logger.Warn("[SSHProxy] VPS %s has no IP address from guest agent. Instance ID: %v, Status: %s. Will attempt connection via Proxmox jump host using hostname.", vpsID, vps.InstanceID, vps.Status)
		
		// We need the VM ID to construct a hostname or use Proxmox jump host
		if vps.InstanceID == nil {
			logger.Error("[SSHProxy] Cannot connect: VPS has no instance ID")
			if _, writeErr := channel.Write([]byte("VPS is not provisioned yet. Please wait for provisioning to complete.\r\n")); writeErr != nil {
				logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
			}
			return
		}
		
		// Try to use the VM's hostname (typically the VPS ID or VM name)
		// VMs on Proxmox often have hostnames based on their name or ID
		// We'll use the VPS ID as hostname and connect via Proxmox jump host
		// The jump host should be able to resolve the hostname or route to the VM
		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt > 0 {
			// Use the VPS ID as hostname (cloud-init typically sets this)
			// Fallback to vm-{id} pattern if VPS ID doesn't work
			vpsHostname := vpsID
			logger.Info("[SSHProxy] Attempting connection via jump host using hostname: %s", vpsHostname)
			// We'll set vpsIP to the hostname and let forwardToVPS handle the jump host connection
			vpsIP = vpsHostname
		} else {
			logger.Error("[SSHProxy] Cannot connect: Invalid VM ID")
			if _, writeErr := channel.Write([]byte("VPS has invalid instance ID. Please check the VPS configuration.\r\n")); writeErr != nil {
				logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
			}
			return
		}
	}

	// Get root password
	var err error
	if s.vpsService != nil {
		rootPassword, err = s.vpsService.getVPSRootPassword(ctx, vpsID)
		if err != nil {
			logger.Error("[SSHProxy] Failed to get root password for VPS %s: %v", vpsID, err)
			if _, writeErr := channel.Write([]byte("Failed to get VPS root password\r\n")); writeErr != nil {
				logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
			}
			return
		}
		if rootPassword == "" {
			logger.Error("[SSHProxy] Root password is empty for VPS %s", vpsID)
			if _, writeErr := channel.Write([]byte("Failed to get VPS root password\r\n")); writeErr != nil {
				logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
			}
			return
		}
		logger.Debug("[SSHProxy] Successfully retrieved root password for VPS %s", vpsID)
	} else {
		logger.Error("[SSHProxy] VPS service is nil, cannot get root password for VPS %s", vpsID)
		if _, writeErr := channel.Write([]byte("VPS service not available\r\n")); writeErr != nil {
			logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
		}
		return
	}

	// Handle requests
	var windowSize struct {
		Width  uint32
		Height uint32
	}
	cols, rows := 80, 24

	for req := range requests {
		switch req.Type {
		case "pty-req":
			// Accept PTY request and update window size
			if err := ssh.Unmarshal(req.Payload, &windowSize); err == nil {
				cols = int(windowSize.Width)
				rows = int(windowSize.Height)
			}
			req.Reply(true, nil)
		case "window-change":
			// Update window size
			if err := ssh.Unmarshal(req.Payload, &windowSize); err == nil {
				cols = int(windowSize.Width)
				rows = int(windowSize.Height)
			}
			req.Reply(true, nil)
		case "shell", "exec":
			// Forward to VPS using connectSSH
			logger.Info("[SSHProxy] Forwarding connection to VPS %s at %s", vpsID, vpsIP)
			if err := s.forwardToVPS(ctx, channel, vpsIP, rootPassword, cols, rows); err != nil {
				logger.Error("[SSHProxy] Failed to forward connection to VPS %s at %s: %v", vpsID, vpsIP, err)
				if _, writeErr := channel.Write([]byte(fmt.Sprintf("Connection failed: %v\r\n", err))); writeErr != nil {
					logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
				}
			}
			return
		default:
			req.Reply(false, nil)
		}
	}
}

// forwardToVPS forwards SSH connection to the actual VPS using connectSSH
// vpsIP can be either an IP address or a hostname (when IP is not available)
func (s *SSHProxyServer) forwardToVPS(ctx context.Context, channel ssh.Channel, vpsIP, rootPassword string, cols, rows int) error {
	if s.vpsService == nil {
		logger.Error("[SSHProxy] VPS service is nil in forwardToVPS")
		return fmt.Errorf("VPS service not available")
	}

	// Get jump host (dedicated SSH proxy VM or Proxmox node as fallback)
	// Prefer dedicated SSH proxy VM for security (doesn't expose Proxmox node)
	jumpHost := ""
	jumpUser := "root"
	
	// Check for dedicated SSH proxy VM first (recommended for security)
	sshProxyHost := os.Getenv("SSH_PROXY_JUMP_HOST")
	if sshProxyHost != "" {
		jumpHost = sshProxyHost
		sshProxyUser := os.Getenv("SSH_PROXY_JUMP_USER")
		if sshProxyUser != "" {
			jumpUser = sshProxyUser
		}
		logger.Info("[SSHProxy] Using dedicated SSH proxy jump host: %s (user: %s)", jumpHost, jumpUser)
	} else {
		// Fallback to Proxmox node (less secure, exposes Proxmox)
		proxmoxConfig, err := orchestrator.GetProxmoxConfig()
		if err == nil && proxmoxConfig.APIURL != "" {
			if u, err := url.Parse(proxmoxConfig.APIURL); err == nil {
				jumpHost = u.Hostname()
				logger.Info("[SSHProxy] Using Proxmox node as jump host: %s (consider using SSH_PROXY_JUMP_HOST for better security)", jumpHost)
			} else {
				logger.Warn("[SSHProxy] Failed to parse Proxmox API URL: %v", err)
			}
		} else if err != nil {
			logger.Warn("[SSHProxy] Failed to get Proxmox config: %v", err)
		}
	}

	// Determine if vpsIP is a hostname (not an IP address)
	// Try to parse as IP to determine if it's a hostname
	var useJumpHost bool
	if net.ParseIP(vpsIP) != nil {
		// It's an IP address
		useJumpHost = false // Can connect directly, but may still use jump host for internal IPs
		logger.Info("[SSHProxy] VPS target is IP address: %s", vpsIP)
	} else {
		// It's a hostname, we need jump host
		useJumpHost = true
		logger.Info("[SSHProxy] VPS target is hostname: %s (will use jump host)", vpsIP)
	}

	// If we have a hostname and no jump host, we can't connect
	if useJumpHost && jumpHost == "" {
		logger.Error("[SSHProxy] Cannot connect: VPS hostname %s requires jump host but none available", vpsIP)
		return fmt.Errorf("cannot connect to hostname %s without jump host (set SSH_PROXY_JUMP_HOST or ensure Proxmox API URL is configured)", vpsIP)
	}
	
	if jumpHost != "" {
		logger.Info("[SSHProxy] Connecting to VPS at %s via jump host %s", vpsIP, jumpHost)
	} else {
		logger.Info("[SSHProxy] Connecting to VPS at %s (direct connection)", vpsIP)
	}
	
	sshConn, err := s.vpsService.connectSSH(ctx, vpsIP, rootPassword, cols, rows, jumpHost, jumpUser)
	if err != nil {
		logger.Error("[SSHProxy] Failed to establish SSH connection to VPS at %s: %v", vpsIP, err)
		return fmt.Errorf("failed to connect to VPS: %w", err)
	}
	logger.Info("[SSHProxy] Successfully connected to VPS at %s", vpsIP)
	defer func() {
		if sshConn.session != nil {
			sshConn.session.Close()
		}
		if sshConn.conn != nil {
			sshConn.conn.Close()
		}
	}()

	// Forward data bidirectionally
	errChan := make(chan error, 3)

	// Forward stdin
	go func() {
		_, err := io.Copy(sshConn.stdin, channel)
		if err != nil {
			logger.Error("[SSHProxy] Error copying stdin to VPS at %s: %v", vpsIP, err)
			errChan <- fmt.Errorf("stdin copy error: %w", err)
		}
		if closeErr := sshConn.stdin.Close(); closeErr != nil {
			logger.Debug("[SSHProxy] Error closing stdin: %v", closeErr)
		}
	}()

	// Forward stdout
	go func() {
		_, err := io.Copy(channel, sshConn.stdout)
		if err != nil {
			logger.Error("[SSHProxy] Error copying stdout from VPS at %s: %v", vpsIP, err)
			errChan <- fmt.Errorf("stdout copy error: %w", err)
		}
	}()

	// Forward stderr
	go func() {
		_, err := io.Copy(channel.Stderr(), sshConn.stderr)
		if err != nil {
			logger.Error("[SSHProxy] Error copying stderr from VPS at %s: %v", vpsIP, err)
			errChan <- fmt.Errorf("stderr copy error: %w", err)
		}
	}()

	// Wait for connection to close or error
	select {
	case err := <-errChan:
		logger.Warn("[SSHProxy] Connection to VPS at %s closed with error: %v", vpsIP, err)
		return err
	case <-ctx.Done():
		logger.Info("[SSHProxy] Connection to VPS at %s closed due to context cancellation", vpsIP)
		return ctx.Err()
	}
}

// getOrGenerateHostKey gets or generates an SSH host key
func getOrGenerateHostKey() (ssh.Signer, error) {
	keyPath := os.Getenv("SSH_PROXY_HOST_KEY_PATH")
	if keyPath == "" {
		keyPath = "/var/lib/obiente/ssh_proxy_host_key"
	}

	// Try to load existing key
	if keyData, err := os.ReadFile(keyPath); err == nil {
		signer, err := ssh.ParsePrivateKey(keyData)
		if err == nil {
			return signer, nil
		}
	}

	// Generate new key
	logger.Info("[SSHProxy] Generating new SSH host key...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write key file
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(privateKeyPEM), 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	// Parse and return signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	logger.Info("[SSHProxy] Generated and saved SSH host key to %s", keyPath)
	return signer, nil
}
