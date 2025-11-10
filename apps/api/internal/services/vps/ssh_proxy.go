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
}

// NewSSHProxyServer creates a new SSH proxy server
func NewSSHProxyServer(port int, vpsService *Service) (*SSHProxyServer, error) {
	// Generate or load host key
	hostKey, err := getOrGenerateHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key: %w", err)
	}

	server := &SSHProxyServer{
		hostKey:        hostKey,
		authorizedKeys: make(map[string]bool),
		port:           port,
		vpsService:     vpsService,
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
	defer conn.Close()

	// Create SSH server config with password authentication
	// Users provide their API token as the password
	config := &ssh.ServerConfig{
		PasswordCallback: s.authenticatePassword,
		// Also allow public key auth for future use
		PublicKeyCallback: s.authenticatePublicKey,
	}

	config.AddHostKey(s.hostKey)

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		logger.Debug("[SSHProxy] SSH handshake failed: %v", err)
		return
	}
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
			logger.Debug("[SSHProxy] Failed to accept channel: %v", err)
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
		logger.Error("[SSHProxy] VPS %s not found: %v", vpsID, err)
		channel.Write([]byte("VPS not found\r\n"))
		return
	}

	// Get VPS IP address and root password
	var vpsIP string
	var rootPassword string

	// Try to get IP from VPS manager first
	vpsManager, err := orchestrator.NewVPSManager()
	if err == nil {
		defer vpsManager.Close()
		ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, vpsID)
		if err == nil && len(ipv4) > 0 {
			vpsIP = ipv4[0]
		}
	}

	// If no public IP, get internal IP from Proxmox
	if vpsIP == "" {
		proxmoxConfig, err := orchestrator.GetProxmoxConfig()
		if err == nil {
			proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
			if err == nil {
				vmIDInt := 0
				if vps.InstanceID != nil {
					fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
				}
				if vmIDInt > 0 {
					nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
					if err == nil {
						ipv4, _, err := proxmoxClient.GetVMIPAddresses(ctx, nodeName, vmIDInt)
						if err == nil && len(ipv4) > 0 {
							vpsIP = ipv4[0]
						}
					}
				}
			}
		}
	}

	if vpsIP == "" {
		channel.Write([]byte("VPS IP address not available. Please ensure the VPS is running.\r\n"))
		return
	}

	// Get root password
	if s.vpsService != nil {
		rootPassword, err = s.vpsService.getVPSRootPassword(ctx, vpsID)
		if err != nil || rootPassword == "" {
			channel.Write([]byte("Failed to get VPS root password\r\n"))
			return
		}
	} else {
		channel.Write([]byte("VPS service not available\r\n"))
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
			if err := s.forwardToVPS(ctx, channel, vpsIP, rootPassword, cols, rows); err != nil {
				logger.Error("[SSHProxy] Failed to forward to VPS: %v", err)
				channel.Write([]byte(fmt.Sprintf("Connection failed: %v\r\n", err)))
			}
			return
		default:
			req.Reply(false, nil)
		}
	}
}

// forwardToVPS forwards SSH connection to the actual VPS using connectSSH
func (s *SSHProxyServer) forwardToVPS(ctx context.Context, channel ssh.Channel, vpsIP, rootPassword string, cols, rows int) error {
	if s.vpsService == nil {
		return fmt.Errorf("VPS service not available")
	}

	// Get Proxmox host for jump host if needed
	proxmoxHost := ""
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err == nil && proxmoxConfig.APIURL != "" {
		if u, err := url.Parse(proxmoxConfig.APIURL); err == nil {
			proxmoxHost = u.Hostname()
		}
	}

	// Connect to VPS (with jump host if needed)
	sshConn, err := s.vpsService.connectSSH(ctx, vpsIP, rootPassword, cols, rows, proxmoxHost, "root")
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %w", err)
	}
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
			errChan <- fmt.Errorf("stdin copy error: %w", err)
		}
		sshConn.stdin.Close()
	}()

	// Forward stdout
	go func() {
		_, err := io.Copy(channel, sshConn.stdout)
		if err != nil {
			errChan <- fmt.Errorf("stdout copy error: %w", err)
		}
	}()

	// Forward stderr
	go func() {
		_, err := io.Copy(channel.Stderr(), sshConn.stderr)
		if err != nil {
			errChan <- fmt.Errorf("stderr copy error: %w", err)
		}
	}()

	// Wait for connection to close or error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
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
