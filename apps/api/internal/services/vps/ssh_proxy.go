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
	"time"

	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"

	"golang.org/x/crypto/ssh"
)

// SSHProxyServer handles SSH bastion/jump host functionality for VPS access.
// Users connect via SSH to the API server, which then forwards SSH channels to the VPS.
type SSHProxyServer struct {
	listener      net.Listener
	hostKey       ssh.Signer
	port          int
	vpsService    *Service
	gatewayClient *orchestrator.VPSGatewayClient
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

// handleConnection handles an incoming SSH connection using a bastion pattern.
// It extracts the VPS ID from the client's username and forwards SSH channels to the VPS.
func (s *SSHProxyServer) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer func() {
		if err := clientConn.Close(); err != nil {
			logger.Debug("[SSHProxy] Error closing connection: %v", err)
		}
	}()

	logger.Info("[SSHProxy] Handling connection from %s", clientConn.RemoteAddr())

	if s.gatewayClient == nil {
		logger.Error("[SSHProxy] Gateway not available, cannot proxy SSH connection")
		return
	}

	username, serverConn, chans, reqs, authInfo, err := s.extractVPSIDAndEstablishConnection(ctx, clientConn)
	if err != nil {
		logger.Warn("[SSHProxy] Failed to extract username and establish connection: %v", err)
		return
	}

	// Parse username to extract VPS ID and target user
	vpsID, targetUser := parseUsername(username)
	if !strings.HasPrefix(vpsID, "vps-") {
		logger.Warn("[SSHProxy] Invalid VPS ID format: %s (expected 'vps-' prefix)", vpsID)
		if serverConn != nil {
			serverConn.Close()
		}
		return
	}

	logger.Info("[SSHProxy] Extracted VPS ID: %s, target user: %s", vpsID, targetUser)

	// Get VPS IP or hostname
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for %s: %v", vpsID, err)
		if serverConn != nil {
			serverConn.Close()
		}
		return
	}

	logger.Info("[SSHProxy] Forwarding SSH channels to VPS %s at %s as user %s", vpsID, vpsIP, targetUser)

	go s.handleGlobalRequests(ctx, reqs, vpsID, vpsIP)
	s.forwardChannelsToVPS(ctx, serverConn, chans, vpsID, vpsIP, targetUser, authInfo)
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
// Uses the web terminal SSH key for authentication.
func (s *SSHProxyServer) connectSSHToVPSForChannelForwarding(ctx context.Context, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo) (*ssh.Client, error) {
	// Create TCP connection to VPS via gateway
	targetConn, err := s.gatewayClient.CreateTCPConnection(ctx, vpsIP, 22)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP connection via gateway: %w", err)
	}

	var authMethods []ssh.AuthMethod
	
	terminalKey, err := database.GetVPSTerminalKey(vpsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get web terminal SSH key for VPS %s: %w", vpsID, err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(terminalKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse web terminal SSH private key: %w", err)
	}

	authMethods = append(authMethods, ssh.PublicKeys(signer))

	if authInfo.authMethod == "password" && authInfo.password != "" {
		authMethods = append(authMethods, ssh.Password(authInfo.password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            targetUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth:            authMethods,
	}
	
	logger.Debug("[SSHProxy] Attempting to connect to VPS %s as user %s", vpsIP, targetUser)

	clientConn, chans, reqs, err := ssh.NewClientConn(targetConn, vpsIP, sshConfig)
	if err != nil {
		targetConn.Close()
		return nil, fmt.Errorf("failed to create SSH client connection to VPS: %w", err)
	}

	go ssh.DiscardRequests(reqs)
	client := ssh.NewClient(clientConn, chans, nil)

	logger.Info("[SSHProxy] Successfully connected to VPS %s as user %s", vpsIP, targetUser)
	return client, nil
}

// extractVPSIDAndEstablishConnection establishes an SSH connection with the client and extracts the username.
func (s *SSHProxyServer) extractVPSIDAndEstablishConnection(ctx context.Context, conn net.Conn) (string, *ssh.ServerConn, <-chan ssh.NewChannel, <-chan *ssh.Request, *clientAuthInfo, error) {
	logger.Debug("[SSHProxy] Starting SSH handshake with client...")
	
	var extractedUsername string
	authInfo := &clientAuthInfo{}
	
	config := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-ObienteCloud",
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			authInfo.publicKey = key
			authInfo.authMethod = "publickey"
			logger.Debug("[SSHProxy] Extracted username from PublicKeyCallback: %s", extractedUsername)
			return &ssh.Permissions{}, nil
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			authInfo.password = string(password)
			authInfo.authMethod = "password"
			logger.Debug("[SSHProxy] Extracted username from PasswordCallback: %s", extractedUsername)
			return &ssh.Permissions{}, nil
		},
		KeyboardInteractiveCallback: func(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			authInfo.authMethod = "keyboard-interactive"
			logger.Debug("[SSHProxy] Extracted username from KeyboardInteractiveCallback: %s", extractedUsername)
			return &ssh.Permissions{}, nil
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

// clientAuthInfo stores the client's authentication credentials.
type clientAuthInfo struct {
	publicKey  ssh.PublicKey
	password   string
	authMethod string
}

// handleGlobalRequests handles global SSH requests from the client.
func (s *SSHProxyServer) handleGlobalRequests(ctx context.Context, reqs <-chan *ssh.Request, vpsID, vpsIP string) {
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
func (s *SSHProxyServer) forwardChannelsToVPS(ctx context.Context, serverConn *ssh.ServerConn, chans <-chan ssh.NewChannel, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo) {
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
			go s.forwardChannel(ctx, newChannel, vpsClient, serverConn)
		}
	}
}

// forwardChannel forwards a single SSH channel from client to VPS.
func (s *SSHProxyServer) forwardChannel(ctx context.Context, newChannel ssh.NewChannel, vpsClient *ssh.Client, serverConn *ssh.ServerConn) {
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
