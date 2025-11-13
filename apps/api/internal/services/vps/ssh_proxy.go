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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// extractIPFromAddr extracts the IP address from a net.Addr, removing port if present.
func extractIPFromAddr(addr net.Addr) string {
	addrStr := addr.String()
	if host, _, err := net.SplitHostPort(addrStr); err == nil {
		return host
	}
	return addrStr
}

// setClientIPEnv sets SSH environment variables to forward the client's real IP address.
func setClientIPEnv(channel ssh.Channel, clientIP, serverIP string) {
	sshClient := fmt.Sprintf("%s 0 22", clientIP)
	sshConnection := fmt.Sprintf("%s 0 %s 22", clientIP, serverIP)
	
	setEnv := func(name, value string) {
		payload := make([]byte, 0, 4+len(name)+4+len(value))
		payload = append(payload, byte(len(name)>>24), byte(len(name)>>16), byte(len(name)>>8), byte(len(name)))
		payload = append(payload, []byte(name)...)
		payload = append(payload, byte(len(value)>>24), byte(len(value)>>16), byte(len(value)>>8), byte(len(value)))
		payload = append(payload, []byte(value)...)
		
		ok, err := channel.SendRequest("env", false, payload)
		if err != nil {
			logger.Debug("[SSHProxy] Failed to set env %s: %v", name, err)
		} else if !ok {
			logger.Debug("[SSHProxy] VPS rejected env %s (may need AcceptEnv in sshd_config)", name)
		} else {
			logger.Debug("[SSHProxy] Set env %s=%s", name, value)
		}
	}
	
	setEnv("SSH_CLIENT", sshClient)
	setEnv("SSH_CONNECTION", sshConnection)
	setEnv("SSH_CLIENT_REAL", clientIP)
}

// SSHProxyServer handles SSH bastion/jump host functionality for VPS access.
// Users connect via SSH to the API server, which then forwards SSH channels to the VPS.
type SSHProxyServer struct {
	listener      net.Listener
	hostKey       ssh.Signer
	port          int
	vpsService    *Service
	gatewayClient *orchestrator.VPSGatewayClient
	
	// Connection tracking for graceful shutdown
	activeConnections sync.WaitGroup
	shutdownOnce      sync.Once
	shutdownChan      chan struct{}
	isDraining        bool
	drainingMu        sync.RWMutex
}

// NewSSHProxyServer creates a new SSH proxy server
func NewSSHProxyServer(port int, vpsService *Service) (*SSHProxyServer, error) {
	// Generate or load host key
	hostKey, err := getOrGenerateHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key: %w", err)
	}

	gatewayClient, err := orchestrator.NewVPSGatewayClient("")
	if err != nil {
		logger.Warn("[SSHProxy] Failed to initialize VPS gateway client: %v", err)
		gatewayClient = nil
	}

	server := &SSHProxyServer{
		hostKey:       hostKey,
		port:          port,
		vpsService:    vpsService,
		gatewayClient: gatewayClient,
		shutdownChan:  make(chan struct{}),
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
					logger.Info("[SSHProxy] Listener closed, stopping accept loop")
					return
				case <-s.shutdownChan:
					logger.Info("[SSHProxy] Shutdown requested, stopping accept loop")
					return
				default:
					logger.Error("[SSHProxy] Failed to accept connection: %v", err)
					continue
				}
			}

			// Check if we're shutting down before accepting new connections
			select {
			case <-s.shutdownChan:
				logger.Info("[SSHProxy] Shutdown in progress, rejecting new connection from %s", conn.RemoteAddr())
				conn.Close()
				continue
			default:
			}

			logger.Info("[SSHProxy] Accepted connection from %s", conn.RemoteAddr())
			s.activeConnections.Add(1)
			go func() {
				defer s.activeConnections.Done()
				s.handleConnection(ctx, conn)
			}()
		}
	}()

	return nil
}

// IsDraining returns true if the SSH proxy server is in draining mode (graceful shutdown)
func (s *SSHProxyServer) IsDraining() bool {
	s.drainingMu.RLock()
	defer s.drainingMu.RUnlock()
	return s.isDraining
}

// Stop stops the SSH proxy server gracefully.
// It stops accepting new connections and waits for active connections to close.
// If timeout is provided, it will wait up to that duration for connections to close.
// When draining starts, the health check will fail, telling Docker Swarm to stop routing
// new connections while keeping the container running until all connections close.
func (s *SSHProxyServer) Stop(timeout time.Duration) error {
	var stopErr error
	s.shutdownOnce.Do(func() {
		logger.Info("[SSHProxy] Initiating graceful shutdown (draining)...")
		
		// Mark as draining - this will cause health check to fail
		// Docker Swarm will stop routing new connections but keep container running
		s.drainingMu.Lock()
		s.isDraining = true
		s.drainingMu.Unlock()
		
		// Signal shutdown to stop accepting new connections
		close(s.shutdownChan)
		
		// Close the listener to stop accepting new connections
		if s.listener != nil {
			if err := s.listener.Close(); err != nil {
				logger.Warn("[SSHProxy] Error closing listener: %v", err)
			}
		}
		
		// Wait for active connections to close
		if timeout > 0 {
			logger.Info("[SSHProxy] Draining: Waiting up to %v for active connections to close...", timeout)
			done := make(chan struct{})
			go func() {
				s.activeConnections.Wait()
				close(done)
			}()
			
			select {
			case <-done:
				logger.Info("[SSHProxy] All connections closed gracefully")
			case <-time.After(timeout):
				logger.Warn("[SSHProxy] Timeout waiting for connections to close, some connections may be terminated")
			}
		} else {
			logger.Info("[SSHProxy] Draining: Waiting for active connections to close (no timeout)...")
			s.activeConnections.Wait()
			logger.Info("[SSHProxy] All connections closed")
		}
		
		logger.Info("[SSHProxy] Graceful shutdown complete")
	})
	return stopErr
}

// getActiveConnectionCount returns an approximate count of active connections.
// This is approximate because connections may close between the check and the return.
func (s *SSHProxyServer) getActiveConnectionCount() int {
	// We can't get exact count from sync.WaitGroup, but we can check if it's zero
	done := make(chan struct{})
	go func() {
		s.activeConnections.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return 0
	default:
		// We can't get exact count, so return -1 to indicate unknown
		return -1
	}
}

// handleConnection handles an incoming SSH connection using a bastion pattern.
// It extracts the VPS ID from the client's username and forwards SSH channels to the VPS.
func (s *SSHProxyServer) handleConnection(ctx context.Context, clientConn net.Conn) {
	logger.Info("[SSHProxy] Handling connection from %s", clientConn.RemoteAddr())

	if s.gatewayClient == nil {
		logger.Error("[SSHProxy] Gateway not available, cannot proxy SSH connection")
		clientConn.Close()
		return
	}

	username, serverConn, chans, reqs, authInfo, err := s.extractVPSIDAndEstablishConnection(ctx, clientConn)
	if err != nil {
		logger.Warn("[SSHProxy] Failed to extract username and establish connection: %v", err)
		clientConn.Close()
		return
	}
	defer func() {
		// Close server connection when done
		if serverConn != nil {
			serverConn.Close()
		}
		// Close client connection when done
		if err := clientConn.Close(); err != nil {
			logger.Debug("[SSHProxy] Error closing connection: %v", err)
		}
	}()

	// Parse username to extract VPS ID and target user
	vpsID, targetUser := parseUsername(username)
	if !strings.HasPrefix(vpsID, "vps-") {
		logger.Warn("[SSHProxy] Invalid VPS ID format: %s (expected 'vps-' prefix)", vpsID)
		return
	}

	logger.Info("[SSHProxy] Extracted VPS ID: %s, target user: %s", vpsID, targetUser)

	// Get VPS IP or hostname
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for %s: %v", vpsID, err)
		return
	}

	logger.Info("[SSHProxy] Forwarding SSH channels to VPS %s at %s as user %s", vpsID, vpsIP, targetUser)

	// Extract client IP from connection
	// Note: For SSH connections, this is the direct TCP connection IP.
	// If SSH is proxied through Traefik with PROXY protocol, the real client IP would be available.
	// For now, we use the connection's RemoteAddr which may be the Docker network IP.
	clientIP := extractIPFromAddr(clientConn.RemoteAddr())
	logger.Debug("[SSHProxy] Client IP: %s", clientIP)

	// Create audit log for SSH connection
	go createSSHAuditLog(vpsID, targetUser, authInfo, clientIP)

	// Handle global requests in background
	go s.handleGlobalRequests(ctx, reqs, vpsID, vpsIP, authInfo)
	
	// Forward channels - this blocks until all channels are closed
	s.forwardChannelsToVPS(ctx, serverConn, chans, vpsID, vpsIP, targetUser, authInfo, clientIP)
	
	logger.Info("[SSHProxy] All channels closed, connection ending for VPS %s", vpsID)
}

// parseUsername extracts VPS ID and target user from username
// Supports formats: "user@vps-xxx" (standard SSH format), "vps-xxx" (defaults to root)
func parseUsername(username string) (vpsID, targetUser string) {
	// Try standard SSH format: user@vps-xxx
	// We look for the last @ that separates user from vps-id
	// This handles cases like: root@vps-xxx, user@vps-xxx@domain (though domain is ignored)
	if idx := strings.LastIndex(username, "@"); idx != -1 {
		// Check if the part after @ looks like a VPS ID (starts with "vps-")
		afterAt := username[idx+1:]
		if strings.HasPrefix(afterAt, "vps-") {
			// Format: user@vps-xxx
			targetUser = username[:idx]
			vpsID = afterAt
			// If there's another @ after vps-id (like user@vps-xxx@domain), ignore the domain part
			if domainIdx := strings.Index(vpsID, "@"); domainIdx != -1 {
				vpsID = vpsID[:domainIdx]
			}
			return
		}
		// If it doesn't start with vps-, might be old format: vps-xxx@user
		// Check if the part before @ looks like a VPS ID
		beforeAt := username[:idx]
		if strings.HasPrefix(beforeAt, "vps-") {
			// Old format: vps-xxx@user (backwards compatibility)
			vpsID = beforeAt
			targetUser = afterAt
			// If there's another @ after user (like vps-xxx@user@domain), ignore the domain part
			if domainIdx := strings.Index(targetUser, "@"); domainIdx != -1 {
				targetUser = targetUser[:domainIdx]
			}
			return
		}
	}

	// Try : format for backwards compatibility (e.g., vps-xxx:user)
	if idx := strings.LastIndex(username, ":"); idx != -1 {
		vpsID = username[:idx]
		targetUser = username[idx+1:]
		return
	}

	// No separator, use entire username as VPS ID, default to root
	vpsID = username
	targetUser = "root"
	return
}

// connectSSHToVPSForChannelForwarding creates an SSH client connection to the target VPS via gateway.
// Uses the bastion SSH key for authentication.
func (s *SSHProxyServer) connectSSHToVPSForChannelForwarding(ctx context.Context, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo) (*ssh.Client, error) {
	// Create TCP connection to VPS via gateway
	targetConn, err := s.gatewayClient.CreateTCPConnection(ctx, vpsIP, 22)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP connection via gateway: %w", err)
	}

	var authMethods []ssh.AuthMethod
	
	// Use bastion SSH key for bastion authentication
	// The bastion key is provisioned on the VPS via cloud-init, so it's always available
	// If it doesn't exist, create it automatically (for backwards compatibility)
	bastionKey, err := database.GetVPSBastionKey(vpsID)
	if err != nil {
		// If key doesn't exist, try to create it
		// Get VPS to find organization ID
		var vps database.VPSInstance
		if err2 := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err2 != nil {
			return nil, fmt.Errorf("failed to get VPS %s: %w", vpsID, err2)
		}
		
		// Create bastion key
		bastionKey, err = database.CreateVPSBastionKey(vpsID, vps.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bastion SSH key for VPS %s: %w", vpsID, err)
		}
		logger.Info("[SSHProxy] Auto-created bastion key for VPS %s", vpsID)
	}

	signer, err := ssh.ParsePrivateKey([]byte(bastionKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse bastion SSH private key: %w", err)
	}

	authMethods = append(authMethods, ssh.PublicKeys(signer))

	// Note: We don't use the client's password here because:
	// 1. For public key auth: Client's password is their API token, not VPS password
	// 2. For password auth: Client's password is their API token, not VPS password
	// 3. The bastion key is sufficient since it's provisioned on the VPS
	// 4. If client has agent forwarding, they can authenticate using their forwarded agent
	// 5. If client doesn't have agent forwarding, the session channel is already authenticated
	//    via the bastion key used for the bastion connection

	sshConfig := &ssh.ClientConfig{
		User:            targetUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth:            authMethods,
	}
	
	logger.Debug("[SSHProxy] Attempting to connect to VPS %s as user %s (using bastion key)", vpsIP, targetUser)

	clientConn, chans, reqs, err := ssh.NewClientConn(targetConn, vpsIP, sshConfig)
		if err != nil {
		targetConn.Close()
		return nil, fmt.Errorf("failed to create SSH client connection to VPS: %w", err)
	}

	go ssh.DiscardRequests(reqs)
	client := ssh.NewClient(clientConn, chans, nil)

	logger.Info("[SSHProxy] Successfully connected to VPS %s as user %s (authenticated with bastion key)", vpsIP, targetUser)
	return client, nil
}

// extractVPSIDAndEstablishConnection establishes an SSH connection with the client and extracts the username.
// It validates authentication before accepting the connection.
func (s *SSHProxyServer) extractVPSIDAndEstablishConnection(ctx context.Context, conn net.Conn) (string, *ssh.ServerConn, <-chan ssh.NewChannel, <-chan *ssh.Request, *clientAuthInfo, error) {
	logger.Debug("[SSHProxy] Starting SSH handshake with client...")
	
	var extractedUsername string
	authInfo := &clientAuthInfo{}
	
	config := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-ObienteCloud",
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			logger.Debug("[SSHProxy] Validating public key authentication for user: %s", extractedUsername)
			
			// Parse username to get VPS ID
			vpsID, _ := parseUsername(extractedUsername)
			if !strings.HasPrefix(vpsID, "vps-") {
				logger.Warn("[SSHProxy] Invalid VPS ID in username: %s", extractedUsername)
				return nil, fmt.Errorf("invalid VPS ID format")
			}
			
			// Get VPS to find organization
			var vps database.VPSInstance
			if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
				logger.Warn("[SSHProxy] VPS not found: %s", vpsID)
				return nil, fmt.Errorf("VPS not found")
			}
			
			// Get SSH keys for this VPS (includes org-wide and VPS-specific keys)
			sshKeys, err := database.GetSSHKeysForVPS(vps.OrganizationID, vpsID)
			if err != nil {
				logger.Error("[SSHProxy] Failed to get SSH keys: %v", err)
				return nil, fmt.Errorf("failed to get SSH keys")
			}
			
			// Validate the public key against stored keys
			keyFingerprint := ssh.FingerprintSHA256(key)
			keyAuthorized := false
			for _, storedKey := range sshKeys {
				// Parse stored public key
				parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(storedKey.PublicKey))
				if err != nil {
					logger.Debug("[SSHProxy] Failed to parse stored key %s: %v", storedKey.ID, err)
					continue
				}
				
				// Compare keys
				if ssh.FingerprintSHA256(parsedKey) == keyFingerprint {
					keyAuthorized = true
					logger.Info("[SSHProxy] Public key authenticated for VPS %s (key: %s)", vpsID, storedKey.Name)
					break
				}
			}
			
			if !keyAuthorized {
				logger.Warn("[SSHProxy] Public key not authorized for VPS %s (fingerprint: %s)", vpsID, keyFingerprint)
				return nil, fmt.Errorf("public key not authorized")
			}
			
			authInfo.publicKey = key
			authInfo.authMethod = "publickey"
			authInfo.vpsID = vpsID
			authInfo.organizationID = vps.OrganizationID
			logger.Debug("[SSHProxy] Public key authentication successful for user: %s", extractedUsername)
			return &ssh.Permissions{}, nil
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			passwordStr := string(password)
			logger.Debug("[SSHProxy] Validating password/API token authentication for user: %s", extractedUsername)
			
			// Parse username to get VPS ID
			vpsID, _ := parseUsername(extractedUsername)
			if !strings.HasPrefix(vpsID, "vps-") {
				logger.Warn("[SSHProxy] Invalid VPS ID in username: %s", extractedUsername)
				return nil, fmt.Errorf("invalid VPS ID format")
			}
			
			// Get VPS to find organization
			var vps database.VPSInstance
			if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
				logger.Warn("[SSHProxy] VPS not found: %s", vpsID)
				return nil, fmt.Errorf("VPS not found")
			}
			
			// Validate API token (password is used as API token)
			// Format as "Bearer <token>" for AuthenticateAndSetContext
			authHeader := "Bearer " + passwordStr
			_, userInfo, err := auth.AuthenticateAndSetContext(ctx, authHeader)
	if err != nil {
				logger.Warn("[SSHProxy] API token validation failed: %v", err)
				return nil, fmt.Errorf("invalid API token")
			}
			
			// Check if user has access to this VPS
			// Check if user is admin
			if auth.HasRole(userInfo, auth.RoleAdmin) {
				logger.Info("[SSHProxy] Admin user authenticated via API token for VPS %s", vpsID)
			} else if vps.CreatedBy == userInfo.Id {
				logger.Info("[SSHProxy] VPS owner authenticated via API token for VPS %s", vpsID)
						} else {
				// Check organization membership
				var count int64
				if err := database.DB.Model(&database.OrganizationMember{}).
					Where("organization_id = ? AND user_id = ? AND status = ?", vps.OrganizationID, userInfo.Id, "active").
					Count(&count).Error; err != nil {
					logger.Warn("[SSHProxy] Failed to check organization membership: %v", err)
					return nil, fmt.Errorf("access denied")
				}
				
				if count == 0 {
					logger.Warn("[SSHProxy] User %s does not have access to VPS %s", userInfo.Id, vpsID)
					return nil, fmt.Errorf("access denied")
				}
				
				logger.Info("[SSHProxy] Organization member authenticated via API token for VPS %s", vpsID)
			}
			
			authInfo.password = passwordStr
			authInfo.authMethod = "password"
			authInfo.userID = userInfo.Id
			authInfo.vpsID = vpsID
			authInfo.organizationID = vps.OrganizationID
			logger.Debug("[SSHProxy] Password/API token authentication successful for user: %s", extractedUsername)
			return &ssh.Permissions{}, nil
		},
		KeyboardInteractiveCallback: func(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			logger.Warn("[SSHProxy] Keyboard-interactive authentication not supported for user: %s", extractedUsername)
			return nil, fmt.Errorf("keyboard-interactive authentication not supported")
		},
	}
	
	if s.hostKey != nil {
		config.AddHostKey(s.hostKey)
	}
	
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	defer conn.SetReadDeadline(time.Time{})
	
	serverConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	
	if extractedUsername == "" {
		extractedUsername = serverConn.User()
		logger.Debug("[SSHProxy] Extracted username from connection metadata: %s", extractedUsername)
	}
	
	if extractedUsername != "" {
		logger.Info("[SSHProxy] Successfully established SSH connection and extracted username: %s (auth method: %s)", extractedUsername, authInfo.authMethod)
		return extractedUsername, serverConn, chans, reqs, authInfo, nil
	}
	
	if serverConn != nil {
		serverConn.Close()
	}
	
	return "", nil, nil, nil, nil, fmt.Errorf("could not extract username from SSH protocol")
}

// clientAuthInfo stores the client's authentication credentials and authorization info.
type clientAuthInfo struct {
	publicKey      ssh.PublicKey
	password       string
	authMethod     string
	userID         string
	vpsID          string
	organizationID string
	hasAgentForwarding bool // Track if client has agent forwarding enabled
}

// handleGlobalRequests handles global SSH requests from the client.
func (s *SSHProxyServer) handleGlobalRequests(ctx context.Context, reqs <-chan *ssh.Request, vpsID, vpsIP string, authInfo *clientAuthInfo) {
	for {
	select {
	case <-ctx.Done():
			return
		case req, ok := <-reqs:
			if !ok {
				return
			}

			logger.Debug("[SSHProxy] Received global request: %s", req.Type)
			
			if req.Type == "auth-agent-req@openssh.com" {
				logger.Info("[SSHProxy] Client requested agent forwarding")
				authInfo.hasAgentForwarding = true
				if req.WantReply {
				req.Reply(true, nil)
			}
			} else if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// forwardChannelsToVPS forwards SSH channels from the client to the VPS.
func (s *SSHProxyServer) forwardChannelsToVPS(ctx context.Context, serverConn *ssh.ServerConn, chans <-chan ssh.NewChannel, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo, clientIP string) {
	vpsClient, err := s.connectSSHToVPSForChannelForwarding(ctx, vpsID, vpsIP, targetUser, authInfo)
	if err != nil {
		logger.Error("[SSHProxy] Failed to connect to VPS for channel forwarding: %v", err)
		serverConn.Close()
		return
	}
	defer vpsClient.Close()
	
	logger.Info("[SSHProxy] Connected to VPS, forwarding channels...")
	
	for {
		select {
		case <-ctx.Done():
			return
		case newChannel, ok := <-chans:
			if !ok {
					return
			}

			logger.Debug("[SSHProxy] Received new channel from client: %s", newChannel.ChannelType())
			go s.forwardChannel(ctx, newChannel, vpsClient, serverConn, clientIP, vpsIP)
		}
	}
}

// forwardChannel forwards a single SSH channel from client to VPS.
func (s *SSHProxyServer) forwardChannel(ctx context.Context, newChannel ssh.NewChannel, vpsClient *ssh.Client, serverConn *ssh.ServerConn, clientIP, vpsIP string) {
	channelType := newChannel.ChannelType()
	logger.Debug("[SSHProxy] Forwarding channel type: %s", channelType)
	
	clientChannel, clientReqs, err := newChannel.Accept()
	if err != nil {
		logger.Error("[SSHProxy] Failed to accept channel from client: %v", err)
		return
	}
	defer clientChannel.Close()

	if channelType == "auth-agent@openssh.com" {
		logger.Info("[SSHProxy] Forwarding agent forwarding channel to VPS")
		vpsChannel, vpsReqs, err := vpsClient.OpenChannel("auth-agent@openssh.com", nil)
		if err != nil {
			logger.Error("[SSHProxy] Failed to open agent forwarding channel on VPS: %v", err)
					return
				}
		defer vpsChannel.Close()
		
		done := make(chan struct{})
		go func() {
			io.Copy(vpsChannel, clientChannel)
			vpsChannel.CloseWrite()
			done <- struct{}{}
		}()
		go func() {
			io.Copy(clientChannel, vpsChannel)
			clientChannel.CloseWrite()
			done <- struct{}{}
		}()
		
		go func() {
			for req := range clientReqs {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}()
		go func() {
			for req := range vpsReqs {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}()
		
		<-done
		return
	}
	
	if channelType == "session" {
		logger.Debug("[SSHProxy] Forwarding session channel to VPS")
		vpsChannel, vpsReqs, err := vpsClient.OpenChannel("session", nil)
	if err != nil {
			logger.Error("[SSHProxy] Failed to open session channel on VPS: %v", err)
		return
	}
					defer vpsChannel.Close()

		// Set environment variables to forward client's real IP
		if clientIP != "" {
			setClientIPEnv(vpsChannel, clientIP, vpsIP)
		}

		done := make(chan struct{})
		go func() {
			io.Copy(vpsChannel, clientChannel)
			vpsChannel.CloseWrite()
			done <- struct{}{}
		}()
		go func() {
			io.Copy(clientChannel, vpsChannel)
			clientChannel.CloseWrite()
			done <- struct{}{}
		}()
		
					go func() {
			for req := range clientReqs {
							ok, err := vpsChannel.SendRequest(req.Type, req.WantReply, req.Payload)
							if req.WantReply {
									req.Reply(ok, nil)
								}
				if err != nil {
					logger.Debug("[SSHProxy] Error forwarding request: %v", err)
							}
						}
					}()
					go func() {
						for req := range vpsReqs {
				ok, payload, err := serverConn.SendRequest(req.Type, req.WantReply, req.Payload)
				if req.WantReply {
					req.Reply(ok, payload)
				}
				if err != nil {
					logger.Debug("[SSHProxy] Error forwarding request: %v", err)
				}
			}
		}()

		<-done
		return
	}

	logger.Debug("[SSHProxy] Rejecting unsupported channel type: %s", channelType)
	newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
}

// getVPSIP gets the VPS IP address using multiple fallback methods.
func (s *SSHProxyServer) getVPSIP(ctx context.Context, vpsID string) (string, error) {
	var vps database.VPSInstance
	vpsExists := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error == nil
	
	if !vpsExists {
		logger.Debug("[SSHProxy] VPS %s not found in database, using VPS ID as hostname", vpsID)
		return vpsID, nil
	}

	var vpsIP string
	var gatewayIP string

	if s.gatewayClient != nil {
		logger.Info("[SSHProxy] Attempting to get IP address from gateway for VPS %s", vpsID)
		allocations, err := s.gatewayClient.ListIPs(ctx, vps.OrganizationID, vpsID)
		if err == nil && len(allocations) > 0 {
			gatewayIP = allocations[0].IpAddress
			logger.Info("[SSHProxy] Got VPS IP from gateway: %s", gatewayIP)
		} else if err != nil {
			logger.Warn("[SSHProxy] Failed to get IP from gateway for VPS %s: %v", vpsID, err)
		} else {
			logger.Info("[SSHProxy] Gateway returned no IP allocations for VPS %s", vpsID)
		}
	}

	var actualVPSIP string
	logger.Info("[SSHProxy] Attempting to get actual IP address for VPS %s", vpsID)
	vpsManager, err := orchestrator.NewVPSManager()
	if err == nil {
		defer vpsManager.Close()
		ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, vpsID)
		if err == nil && len(ipv4) > 0 {
			actualVPSIP = ipv4[0]
			logger.Info("[SSHProxy] Got actual VPS IP from VPS manager: %s", actualVPSIP)
		} else if err != nil {
			logger.Warn("[SSHProxy] Failed to get IP from VPS manager for VPS %s: %v", vpsID, err)
		}
		} else {
		logger.Warn("[SSHProxy] Failed to create VPS manager: %v", err)
	}

	if actualVPSIP == "" {
		logger.Info("[SSHProxy] No IP from VPS manager, trying to get internal IP from Proxmox for VPS %s", vpsID)
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
							actualVPSIP = ipv4[0]
							logger.Info("[SSHProxy] Got actual VPS IP from Proxmox: %s", actualVPSIP)
						} else if err != nil {
							logger.Warn("[SSHProxy] Failed to get IP from Proxmox for VM %d on node %s: %v", vmIDInt, nodeName, err)
						}
					}
				}
			}
		}
	}

	if actualVPSIP != "" {
		if gatewayIP != "" && gatewayIP != actualVPSIP {
			logger.Warn("[SSHProxy] IP mismatch detected for VPS %s: gateway reports %s but VPS actually has %s", vpsID, gatewayIP, actualVPSIP)
		}
		vpsIP = actualVPSIP
	} else if gatewayIP != "" {
		logger.Info("[SSHProxy] Using gateway IP %s", gatewayIP)
		vpsIP = gatewayIP
	}

	if vpsIP == "" {
		logger.Warn("[SSHProxy] VPS %s has no IP address, attempting connection using hostname", vpsID)
		if vps.InstanceID == nil {
			return "", fmt.Errorf("VPS has no instance ID")
		}
		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt > 0 {
			logger.Info("[SSHProxy] Attempting connection using hostname: %s", vpsID)
			return vpsID, nil
		}
		return "", fmt.Errorf("VPS has invalid instance ID")
	}

	return vpsIP, nil
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

// createSSHAuditLog creates an audit log entry for an SSH connection
func createSSHAuditLog(vpsID, targetUser string, authInfo *clientAuthInfo, clientIP string) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[SSHProxy] Panic creating audit log for SSH connection: %v", r)
		}
	}()

	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use MetricsDB (TimescaleDB) for audit logs
	if database.MetricsDB == nil {
		logger.Warn("[SSHProxy] Metrics database not available, skipping audit log for SSH connection")
		return
	}

	// Determine user ID
	// For password/API token auth, we have the userID
	// For public key auth, we don't know which user owns the key (SSHKey doesn't track user_id)
	// So we use "system" for public key auth, but include the key fingerprint in request data
	userID := authInfo.userID
	if userID == "" {
		// Public key auth - use "system" since we can't identify the user
		userID = "system"
	}

	// Determine organization ID
	var orgID *string
	if authInfo.organizationID != "" {
		orgID = &authInfo.organizationID
	}

	// Create request data
	var requestData string
	if authInfo.publicKey != nil {
		keyFingerprint := ssh.FingerprintSHA256(authInfo.publicKey)
		requestData = fmt.Sprintf(`{"vps_id":"%s","target_user":"%s","auth_method":"%s","has_agent_forwarding":%t,"key_fingerprint":"%s"}`,
			vpsID, targetUser, authInfo.authMethod, authInfo.hasAgentForwarding, keyFingerprint)
	} else {
		requestData = fmt.Sprintf(`{"vps_id":"%s","target_user":"%s","auth_method":"%s","has_agent_forwarding":%t}`,
			vpsID, targetUser, authInfo.authMethod, authInfo.hasAgentForwarding)
	}

	// Create audit log entry
	auditLog := database.AuditLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: orgID,
		Action:         "SSHConnect",
		Service:        "SSHProxyService",
		ResourceType:   stringPtr("vps"),
		ResourceID:     &vpsID,
		IPAddress:      clientIP,
		UserAgent:      fmt.Sprintf("SSH/%s", authInfo.authMethod),
		RequestData:    requestData,
		ResponseStatus: 200,
		ErrorMessage:   nil,
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		logger.Warn("[SSHProxy] Failed to create audit log for SSH connection: %v", err)
	} else {
		logger.Debug("[SSHProxy] Created audit log for SSH connection: user=%s, vps=%s, ip=%s", userID, vpsID, clientIP)
	}
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
