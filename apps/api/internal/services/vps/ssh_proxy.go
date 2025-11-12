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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"

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
	vpsService     *Service                       // Reference to VPS service for connectSSH
	gatewayClient  *orchestrator.VPSGatewayClient // Optional gateway client for SSH proxying
	connectionPool *SSHConnectionPool             // Connection pool for persistent SSH connections
}

// NewSSHProxyServer creates a new SSH proxy server
func NewSSHProxyServer(port int, vpsService *Service) (*SSHProxyServer, error) {
	// Generate or load host key
	hostKey, err := getOrGenerateHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key: %w", err)
	}

	// Initialize gateway client (optional - will be nil if gateway is not configured)
	// Uses VPS_GATEWAY_URL from environment or can be discovered from node metadata
	gatewayClient, err := orchestrator.NewVPSGatewayClient("")
	if err != nil {
		logger.Warn("[SSHProxy] Failed to initialize VPS gateway client (gateway may not be configured): %v", err)
		gatewayClient = nil // Continue without gateway - will use direct SSH connection
	}

	// Initialize connection pool if gateway is available
	var connectionPool *SSHConnectionPool
	if gatewayClient != nil {
		connectionPool = NewSSHConnectionPool(gatewayClient)
	}

	server := &SSHProxyServer{
		hostKey:        hostKey,
		authorizedKeys: make(map[string]bool),
		port:           port,
		vpsService:     vpsService,
		gatewayClient:  gatewayClient,
		connectionPool: connectionPool,
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

	// Create SSH server config with authentication methods
	// IMPORTANT: Public key authentication is tried FIRST, then password
	// This ensures SSH keys are prioritized over password authentication
	// Note: SSH protocol order is client-determined, but we can influence it
	// by making public key auth succeed when possible and password auth as fallback
	config := &ssh.ServerConfig{
		// Public key auth is tried first (preferred method)
		// The callback will be called for each key the client offers
		PublicKeyCallback: s.authenticatePublicKey,
		// Password auth is fallback (API token as password)
		// This is used when public key auth fails or client doesn't offer keys
		PasswordCallback: s.authenticatePassword,
		// Set server version
		ServerVersion: "SSH-2.0-ObienteCloud",
		// Enable keyboard-interactive as alternative password method
		// This ensures password input works properly when PasswordCallback doesn't work
		// Some SSH clients prefer keyboard-interactive for password input
		KeyboardInteractiveCallback: func(conn ssh.ConnMetadata, challenge ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			// Prompt for password using keyboard-interactive
			// This method is more reliable for password input in some clients
			answers, err := challenge("", "", []string{"Password: "}, []bool{false})
			if err != nil || len(answers) == 0 {
				return nil, fmt.Errorf("keyboard-interactive authentication failed")
			}
			// Use the password authentication logic
			return s.authenticatePassword(conn, []byte(answers[0]))
		},
	}

	if s.hostKey == nil {
		logger.Error("[SSHProxy] Host key is nil, cannot handle connection")
		return
	}
	config.AddHostKey(s.hostKey)

	// Perform SSH handshake
	// This will call PublicKeyCallback for each key the client offers
	// If no keys match, it will fall back to PasswordCallback or KeyboardInteractiveCallback
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		logger.Warn("[SSHProxy] SSH handshake failed for %s: %v", conn.RemoteAddr(), err)
		// Log more details about authentication failure
		if strings.Contains(err.Error(), "no auth passed") {
			logger.Info("[SSHProxy] Authentication failed: client did not provide valid credentials (no matching SSH keys and password/auth failed)")
		}
		return
	}

	// Clear the read deadline after successful handshake
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		logger.Debug("[SSHProxy] Failed to clear read deadline: %v", err)
	}

	// Log authentication method used
	authMethod := "unknown"
	if sshConn.Permissions != nil {
		if method, ok := sshConn.Permissions.Extensions["auth_method"]; ok {
			authMethod = method
		} else {
			authMethod = "password" // Default if no auth_method extension
		}
	}
	logger.Info("[SSHProxy] SSH handshake successful for user %s from %s (auth method: %s)", sshConn.User(), conn.RemoteAddr(), authMethod)
	defer sshConn.Close()

	// Handle channels
	// Username is the VPS ID (which already includes "vps-" prefix)
	vpsID := sshConn.User()

	// Handle global requests (for port forwarding, etc.)
	go s.handleGlobalRequests(ctx, reqs, sshConn, vpsID)

	// Extract SSH key ID from permissions if available
	var keyID string
	var sshPublicKey string
	if sshConn.Permissions != nil {
		if kID, ok := sshConn.Permissions.Extensions["key_id"]; ok {
			keyID = kID
			// Get SSH key from database
			var sshKey database.SSHKey
			if err := database.DB.Where("id = ?", kID).First(&sshKey).Error; err == nil {
				sshPublicKey = sshKey.PublicKey
				logger.Debug("[SSHProxy] Extracted SSH public key for connection: %s", kID)
			}
		}
	}

	for newChannel := range chans {
		channelType := newChannel.ChannelType()
		
		// Route channels by type
		switch channelType {
		case "session":
			channel, requests, err := newChannel.Accept()
			if err != nil {
				logger.Error("[SSHProxy] Failed to accept session channel for user %s: %v", sshConn.User(), err)
				continue
			}
			go s.handleSessionChannel(ctx, channel, requests, vpsID, sshPublicKey, keyID)
		case "direct-tcpip":
			go s.handleDirectTCPIPChannel(ctx, newChannel, vpsID, keyID)
		case "forwarded-tcpip":
			go s.handleForwardedTCPIPChannel(ctx, newChannel, vpsID, keyID)
		case "auth-agent@openssh.com":
			channel, _, err := newChannel.Accept()
			if err != nil {
				logger.Error("[SSHProxy] Failed to accept agent channel for user %s: %v", sshConn.User(), err)
				continue
			}
			go s.handleAgentChannel(ctx, channel, vpsID, keyID)
		case "x11":
			channel, _, err := newChannel.Accept()
			if err != nil {
				logger.Error("[SSHProxy] Failed to accept X11 channel for user %s: %v", sshConn.User(), err)
				continue
			}
			go s.handleX11Channel(ctx, channel, vpsID, keyID)
		default:
			// Forward unknown channels transparently
			channel, requests, err := newChannel.Accept()
			if err != nil {
				logger.Warn("[SSHProxy] Failed to accept channel type %s: %v", channelType, err)
				newChannel.Reject(ssh.UnknownChannelType, "channel type not supported")
				continue
			}
			go s.handleGenericChannel(ctx, channel, requests, vpsID, keyID, channelType)
		}
	}
}

// authenticatePassword authenticates SSH connections using API token as password
// NOTE: We authenticate at the proxy level for security/audit, but we allow connections
// even if the VPS doesn't exist in the database - the VPS's own SSH server will handle
// the actual authentication to the VPS itself.
func (s *SSHProxyServer) authenticatePassword(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	// Username is the VPS ID (which already includes "vps-" prefix)
	username := conn.User()
	if !strings.HasPrefix(username, "vps-") {
		return nil, fmt.Errorf("invalid username format: expected VPS ID starting with 'vps-'")
	}

	vpsID := username // VPS ID already includes "vps-" prefix
	apiToken := string(password)

	// Validate API token for security/audit purposes
	// Use AuthenticateHTTPRequest helper which validates tokens
	ctx := context.Background()
	authConfig := auth.NewAuthConfig()

	// Create a mock HTTP request for token validation
	mockReq, _ := http.NewRequestWithContext(ctx, "GET", "/", nil)
	mockReq.Header.Set("Authorization", "Bearer "+apiToken)

	// Validate token
	userInfo, err := auth.AuthenticateHTTPRequest(authConfig, mockReq)
	if err != nil {
		logger.Debug("[SSHProxy] Token validation failed for VPS %s: %v", vpsID, err)
		// Even if token validation fails, we allow the connection
		// The VPS's SSH server will handle authentication
		// This allows users to connect directly with VPS credentials
		logger.Debug("[SSHProxy] Allowing connection to VPS %s without API token (VPS will authenticate)", vpsID)
		return &ssh.Permissions{
			Extensions: map[string]string{
				"vps_id":      vpsID,
				"auth_method": "password",
				"vps_exists":  "unknown", // Can't determine if VPS exists without token
			},
		}, nil
	}

	// Token is valid - check if VPS exists (for logging/audit)
	var vps database.VPSInstance
	vpsExists := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error == nil
	if !vpsExists {
		logger.Debug("[SSHProxy] VPS %s not found in database for user %s, allowing connection (VPS will authenticate)", vpsID, userInfo.Id)
		return &ssh.Permissions{
			Extensions: map[string]string{
				"vps_id":      vpsID,
				"user_id":     userInfo.Id,
				"auth_method": "password",
				"vps_exists":  "false",
			},
		}, nil
	}

	// VPS exists and token is valid
	logger.Info("[SSHProxy] Authenticated user %s for VPS %s via password (API token)", userInfo.Id, vpsID)
	return &ssh.Permissions{
		Extensions: map[string]string{
			"vps_id":      vpsID,
			"user_id":     userInfo.Id,
			"auth_method": "password",
			"vps_exists":  "true",
		},
	}, nil
}

// authenticatePublicKey authenticates SSH connections using public key
// This is called for EACH key the client offers during authentication
// IMPORTANT: The SSH protocol order is client-determined, but this callback
// will be called for every key the client tries, allowing us to accept valid keys immediately
// NOTE: We authenticate at the proxy level for security/audit, but we allow connections
// even if the VPS doesn't exist in the database - the VPS's own SSH server will handle
// the actual authentication to the VPS itself.
func (s *SSHProxyServer) authenticatePublicKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Username is the VPS ID (which already includes "vps-" prefix)
	username := conn.User()
	if !strings.HasPrefix(username, "vps-") {
		return nil, fmt.Errorf("invalid username format: expected VPS ID starting with 'vps-'")
	}

	vpsID := username // VPS ID already includes "vps-" prefix

	// Try to get VPS instance to find organization and SSH keys
	// If VPS doesn't exist in database, we still allow the connection - the VPS's SSH server will authenticate
	var vps database.VPSInstance
	var orgID string
	vpsExists := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error == nil
	if vpsExists {
		orgID = vps.OrganizationID
	} else {
		logger.Debug("[SSHProxy] VPS %s not found in database, allowing connection (VPS SSH will authenticate)", vpsID)
		// Allow connection even if VPS doesn't exist - VPS's SSH server will handle authentication
		// We still need to authenticate the user at the proxy level for security/audit
		// For now, we'll accept any valid SSH key format (we can't verify against database)
		// The actual VPS authentication will happen when we forward to the VPS
		return &ssh.Permissions{
			Extensions: map[string]string{
				"vps_id":      vpsID,
				"auth_method": "publickey",
				"vps_exists":  "false", // Flag to indicate VPS not in database
			},
		}, nil
	}

	// VPS exists in database - check SSH keys
	// Get SSH keys for the VPS (includes both VPS-specific and org-wide keys)
	// This ensures we check all available keys, not just org-wide ones
	sshKeys, err := database.GetSSHKeysForVPS(orgID, vpsID)
	if err != nil {
		logger.Debug("[SSHProxy] Failed to get SSH keys for VPS %s: %v, allowing connection anyway", vpsID, err)
		// Allow connection even if we can't get SSH keys - VPS will authenticate
		return &ssh.Permissions{
			Extensions: map[string]string{
				"vps_id":      vpsID,
				"auth_method": "publickey",
				"vps_exists":  "true",
			},
		}, nil
	}

	// Calculate fingerprint of the provided key
	providedFingerprint := ssh.FingerprintSHA256(key)
	logger.Debug("[SSHProxy] Checking public key with fingerprint %s for VPS %s", providedFingerprint, vpsID)

	// Check if any of the VPS's SSH keys match
	for _, vpsKey := range sshKeys {
		// Parse the stored public key
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(vpsKey.PublicKey))
		if err != nil {
			logger.Debug("[SSHProxy] Failed to parse SSH key %s: %v", vpsKey.ID, err)
			continue // Skip invalid keys
		}

		// Compare fingerprints
		storedFingerprint := ssh.FingerprintSHA256(parsedKey)
		if storedFingerprint == providedFingerprint {
			logger.Info("[SSHProxy] Authenticated via SSH key %s (fingerprint: %s) for VPS %s", vpsKey.Name, providedFingerprint, vpsID)
			return &ssh.Permissions{
				Extensions: map[string]string{
					"vps_id":      vpsID,
					"key_id":      vpsKey.ID,
					"auth_method": "publickey",
					"vps_exists":  "true",
				},
			}, nil
		}
	}

	// Key doesn't match any in database, but we still allow the connection
	// The VPS's SSH server will handle the actual authentication
	logger.Debug("[SSHProxy] Public key with fingerprint %s not found in database for VPS %s, allowing connection (VPS will authenticate)", providedFingerprint, vpsID)
	return &ssh.Permissions{
		Extensions: map[string]string{
			"vps_id":      vpsID,
			"auth_method": "publickey",
			"vps_exists":  "true",
		},
	}, nil
}

// getVPSIP gets the VPS IP address using multiple fallback methods
// If VPS doesn't exist in database, uses VPS ID as hostname (gateway will resolve)
func (s *SSHProxyServer) getVPSIP(ctx context.Context, vpsID string) (string, error) {
	// Get VPS instance
	var vps database.VPSInstance
	vpsExists := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error == nil
	
	// If VPS doesn't exist in database, try to use VPS ID as hostname
	// The gateway's dnsmasq can resolve VPS hostnames
	if !vpsExists {
		logger.Debug("[SSHProxy] VPS %s not found in database, using VPS ID as hostname (gateway will resolve)", vpsID)
		// Use VPS ID as hostname - gateway's dnsmasq will resolve it
		return vpsID, nil
	}

	var vpsIP string

	// Try to get IP from gateway first (if available)
	if s.gatewayClient != nil {
		logger.Info("[SSHProxy] Attempting to get IP address from gateway for VPS %s", vpsID)
		allocations, err := s.gatewayClient.ListIPs(ctx, vps.OrganizationID, vpsID)
		if err == nil && len(allocations) > 0 {
			// Gateway manages DHCP, so use the allocated IP
			vpsIP = allocations[0].IpAddress
			logger.Info("[SSHProxy] Got VPS IP from gateway: %s", vpsIP)
			return vpsIP, nil
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
				return vpsIP, nil
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
							return vpsIP, nil
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

	// If no IP found, try to use VM hostname (gateway can resolve hostnames)
	if vpsIP == "" {
		logger.Warn("[SSHProxy] VPS %s has no IP address from guest agent. Instance ID: %v, Status: %s. Will attempt connection using hostname.", vpsID, vps.InstanceID, vps.Status)

		// We need the VM ID to construct a hostname
		if vps.InstanceID == nil {
			return "", fmt.Errorf("VPS has no instance ID")
		}

		// Try to use the VM's hostname (typically the VPS ID or VM name)
		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt > 0 {
			// Use the VPS ID as hostname (cloud-init typically sets this)
			vpsHostname := vpsID
			logger.Info("[SSHProxy] Attempting connection using hostname: %s (gateway will resolve)", vpsHostname)
			return vpsHostname, nil
		} else {
			return "", fmt.Errorf("VPS has invalid instance ID")
		}
	}

	return vpsIP, nil
}

// getSSHSigner gets an SSH signer from the database for the given key ID
func (s *SSHProxyServer) getSSHSigner(keyID string) (ssh.Signer, error) {
	if keyID == "" {
		return nil, fmt.Errorf("key ID is required")
	}

	// Get SSH key from database
	var sshKey database.SSHKey
	if err := database.DB.Where("id = ?", keyID).First(&sshKey).Error; err != nil {
		return nil, fmt.Errorf("SSH key not found: %w", err)
	}

	// Note: We only have the public key in the database, not the private key
	// For authentication to VPS, we need the user's private key
	// This is a limitation - we'll need to handle this differently
	// For now, return an error indicating we need the private key
	return nil, fmt.Errorf("private key required for SSH authentication (not stored in database)")
}

// handleSessionChannel handles SSH session channel requests (shell, exec, subsystem)
func (s *SSHProxyServer) handleSessionChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, vpsID string, sshPublicKey string, keyID string) {
	defer channel.Close()

	// Get VPS IP address
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for %s: %v", vpsID, err)
		if _, writeErr := channel.Write([]byte(fmt.Sprintf("Failed to get VPS IP: %v\r\n", err))); writeErr != nil {
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
			// Try to use connection pool if available
			if s.connectionPool != nil && keyID != "" {
				// Try to get SSH signer (may fail if private key not available)
				sshSigner, err := s.getSSHSigner(keyID)
				if err == nil {
					// Get or create persistent connection
					vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
					if err == nil {
						// Use persistent connection for shell/exec
						if err := s.forwardSessionViaPool(ctx, channel, requests, vpsConn, req.Type, cols, rows); err != nil {
							logger.Error("[SSHProxy] Failed to forward session via pool: %v", err)
							// Fall back to gateway TCP forwarding
							if s.gatewayClient != nil {
								if err := s.forwardToVPSViaGateway(ctx, channel, vpsIP, "", cols, rows, sshPublicKey); err != nil {
									logger.Error("[SSHProxy] Gateway ProxySSH also failed: %v", err)
									return
								}
								return
							}
						} else {
							return
						}
					}
				}
			}

			// Fall back to gateway TCP forwarding (current approach)
			if s.gatewayClient != nil {
				logger.Info("[SSHProxy] Gateway available, attempting connection via gateway ProxySSH for VPS at %s", vpsIP)
				if err := s.forwardToVPSViaGateway(ctx, channel, vpsIP, "", cols, rows, sshPublicKey); err != nil {
					logger.Error("[SSHProxy] Gateway ProxySSH failed for VPS at %s: %v", vpsIP, err)
					if _, writeErr := channel.Write([]byte(fmt.Sprintf("Connection via gateway failed: %v\r\n", err))); writeErr != nil {
						logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
					}
					return
				}
				return
			}

			// Gateway not available
			logger.Error("[SSHProxy] Gateway not available for VPS %s. Direct connection not supported for security reasons.", vpsID)
			if _, writeErr := channel.Write([]byte("Gateway not available. Please connect directly to the VPS.\r\n")); writeErr != nil {
				logger.Warn("[SSHProxy] Failed to write error message to channel: %v", writeErr)
			}
			return
		case "subsystem":
			// Handle SFTP/SCP subsystem requests
			var subsystemMsg struct {
				Name string
			}
			if err := ssh.Unmarshal(req.Payload, &subsystemMsg); err != nil {
				req.Reply(false, nil)
				continue
			}

			// Try to use connection pool if available
			if s.connectionPool != nil && keyID != "" {
				sshSigner, err := s.getSSHSigner(keyID)
				if err == nil {
					vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
					if err == nil {
						// Create session and request subsystem
						vpsSession, err := vpsConn.sshClient.NewSession()
						if err == nil {
							if err := vpsSession.RequestSubsystem(subsystemMsg.Name); err == nil {
								req.Reply(true, nil)
								// Set up pipes
								stdin, _ := vpsSession.StdinPipe()
								stdout, _ := vpsSession.StdoutPipe()
								stderr, _ := vpsSession.StderrPipe()
								// Forward stdin/stdout/stderr
								go io.Copy(stdin, channel)
								go io.Copy(channel, stdout)
								go io.Copy(channel.Stderr(), stderr)
								// Wait for session to end
								vpsSession.Wait()
								vpsSession.Close()
								return
							}
							vpsSession.Close()
						}
					}
				}
			}

			// Fall back to gateway TCP forwarding
			if s.gatewayClient != nil {
				logger.Info("[SSHProxy] Forwarding subsystem %s via gateway for VPS at %s", subsystemMsg.Name, vpsIP)
				if err := s.forwardToVPSViaGateway(ctx, channel, vpsIP, "", cols, rows, sshPublicKey); err != nil {
					logger.Error("[SSHProxy] Gateway ProxySSH failed for subsystem: %v", err)
					req.Reply(false, []byte(fmt.Sprintf("Failed to forward subsystem: %v", err)))
					return
				}
				req.Reply(true, nil)
				return
			}

			req.Reply(false, []byte("Subsystem forwarding not available"))
			return
		default:
			req.Reply(false, nil)
		}
	}
}

// forwardToVPS forwards SSH connection to the actual VPS
// Uses gateway ProxySSH if available, otherwise falls back to direct SSH connection
// vpsIP can be either an IP address or a hostname (when IP is not available)
func (s *SSHProxyServer) forwardToVPS(ctx context.Context, channel ssh.Channel, vpsIP, rootPassword string, cols, rows int) error {
	if s.vpsService == nil {
		logger.Error("[SSHProxy] VPS service is nil in forwardToVPS")
		return fmt.Errorf("VPS service not available")
	}

	// Prefer gateway ProxySSH if available
	if s.gatewayClient != nil {
		logger.Info("[SSHProxy] Using gateway ProxySSH for VPS at %s", vpsIP)
		// Note: sshPublicKey is not available in this path since it's only passed from handleChannel
		// This is a fallback path, so we'll pass empty string
		return s.forwardToVPSViaGateway(ctx, channel, vpsIP, rootPassword, cols, rows, "")
	}

	// Fallback to direct SSH connection (no jump host)
	logger.Info("[SSHProxy] Gateway not available, using direct SSH connection to VPS at %s", vpsIP)
	sshConn, err := s.vpsService.connectSSH(ctx, vpsIP, rootPassword, cols, rows, "", "")
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

// forwardToVPSViaGateway forwards SSH connection through gateway ProxySSH
func (s *SSHProxyServer) forwardToVPSViaGateway(ctx context.Context, channel ssh.Channel, vpsIP, rootPassword string, cols, rows int, sshPublicKey string) error {
	// Generate connection ID
	connectionID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())

	// Create gateway ProxySSH stream
	logger.Info("[SSHProxy] Creating ProxySSH stream to gateway for VPS at %s", vpsIP)
	stream, err := s.gatewayClient.ProxySSH(ctx)
	if err != nil {
		logger.Error("[SSHProxy] Failed to create gateway ProxySSH stream: %v", err)
		return fmt.Errorf("failed to create gateway ProxySSH stream: %w", err)
	}
	logger.Info("[SSHProxy] ProxySSH stream created successfully")

	// Send connect request with SSH public key
	// TODO: After regenerating proto files, uncomment SshPublicKey field
	req := &vpsgatewayv1.ProxySSHRequest{
		ConnectionId: connectionID,
		Type:         "connect",
		Target:       vpsIP,
		Port:         22,
		// SshPublicKey: sshPublicKey, // TODO: Uncomment after proto regeneration
	}
	if sshPublicKey != "" {
		logger.Info("[SSHProxy] Forwarding SSH connection with public key (key will be in SSH protocol stream)")
	}
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send connect request: %w", err)
	}

	errChan := make(chan error, 2)

	// Handle responses from gateway
	go func() {
		for {
			resp, err := stream.Receive()
			if err == io.EOF {
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("gateway stream error: %w", err)
				return
			}

			switch resp.Type {
			case "connected":
				logger.Info("[SSHProxy] Gateway connected to VPS at %s", vpsIP)
			case "data":
				// Forward data from gateway to channel
				if _, err := channel.Write(resp.Data); err != nil {
					errChan <- fmt.Errorf("failed to write to channel: %w", err)
					return
				}
			case "error":
				errChan <- fmt.Errorf("gateway error: %s", resp.Error)
				return
			case "closed":
				logger.Info("[SSHProxy] Gateway closed connection to VPS at %s", vpsIP)
				return
			}
		}
	}()

	// Forward data from channel to gateway
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := channel.Read(buf)
			if n > 0 {
				if sendErr := stream.Send(&vpsgatewayv1.ProxySSHRequest{
					ConnectionId: connectionID,
					Type:         "data",
					Data:         buf[:n],
				}); sendErr != nil {
					errChan <- fmt.Errorf("failed to send data to gateway: %w", sendErr)
					return
				}
			}
			if err == io.EOF {
				// Send close request
				stream.Send(&vpsgatewayv1.ProxySSHRequest{
					ConnectionId: connectionID,
					Type:         "close",
				})
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("failed to read from channel: %w", err)
				return
			}
		}
	}()

	// Wait for connection or error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// forwardSessionViaPool forwards a session channel using a pooled SSH connection
func (s *SSHProxyServer) forwardSessionViaPool(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, vpsConn *PooledSSHConnection, reqType string, cols, rows int) error {
	// Create new session on the pooled connection
	session, err := vpsConn.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Request PTY if needed
	if reqType == "shell" {
		if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}); err != nil {
			return fmt.Errorf("failed to request PTY: %w", err)
		}
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start shell or exec
	if reqType == "shell" {
		if err := session.Shell(); err != nil {
			stdin.Close()
			return fmt.Errorf("failed to start shell: %w", err)
		}
	} else if reqType == "exec" {
		// For exec, we need to get the command from the request
		// This is handled in the request loop, so we'll start shell for now
		// TODO: Extract command from exec request
		if err := session.Shell(); err != nil {
			stdin.Close()
			return fmt.Errorf("failed to start shell: %w", err)
		}
	}

	// Forward data bidirectionally
	errChan := make(chan error, 3)

	go func() {
		_, err := io.Copy(stdin, channel)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("stdin copy error: %w", err)
		}
		stdin.Close()
	}()

	go func() {
		_, err := io.Copy(channel, stdout)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("stdout copy error: %w", err)
		}
	}()

	go func() {
		_, err := io.Copy(channel.Stderr(), stderr)
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("stderr copy error: %w", err)
		}
	}()

	// Handle window size changes
	go func() {
		for req := range requests {
			if req.Type == "window-change" {
				var windowSize struct {
					Width  uint32
					Height uint32
				}
				if err := ssh.Unmarshal(req.Payload, &windowSize); err == nil {
					session.WindowChange(int(windowSize.Height), int(windowSize.Width))
				}
				req.Reply(true, nil)
			}
		}
	}()

	// Wait for session to end
	session.Wait()
	return nil
}

// handleDirectTCPIPChannel handles local port forwarding (ssh -L)
func (s *SSHProxyServer) handleDirectTCPIPChannel(ctx context.Context, newChannel ssh.NewChannel, vpsID string, keyID string) {
	// Parse channel data
	var directMsg struct {
		Host string
		Port uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &directMsg); err != nil {
		logger.Error("[SSHProxy] Failed to parse direct-tcpip channel data: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, "invalid channel data")
		return
	}

	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for direct-tcpip: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, "failed to get VPS IP")
		return
	}

	// Accept client channel
	clientChannel, clientReqs, err := newChannel.Accept()
	if err != nil {
		logger.Error("[SSHProxy] Failed to accept direct-tcpip channel: %v", err)
		return
	}
	defer clientChannel.Close()

	// Try to use connection pool
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				// Open channel on VPS connection
				vpsChannel, vpsReqs, err := vpsConn.sshClient.OpenChannel("direct-tcpip", newChannel.ExtraData())
				if err == nil {
					defer vpsChannel.Close()
					// Forward requests
					go ssh.DiscardRequests(clientReqs)
					go ssh.DiscardRequests(vpsReqs)
					// Forward data bidirectionally
					go io.Copy(clientChannel, vpsChannel)
					io.Copy(vpsChannel, clientChannel)
					return
				}
			}
		}
	}

	// Fall back: reject if no connection pool
	logger.Warn("[SSHProxy] Connection pool not available for direct-tcpip, rejecting")
	clientChannel.Write([]byte("Port forwarding requires connection pool\r\n"))
}

// handleForwardedTCPIPChannel handles remote port forwarding (ssh -R)
func (s *SSHProxyServer) handleForwardedTCPIPChannel(ctx context.Context, newChannel ssh.NewChannel, vpsID string, keyID string) {
	// Parse channel data
	var forwardedMsg struct {
		Host string
		Port uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &forwardedMsg); err != nil {
		logger.Error("[SSHProxy] Failed to parse forwarded-tcpip channel data: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, "invalid channel data")
		return
	}

	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for forwarded-tcpip: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, "failed to get VPS IP")
		return
	}

	// Accept client channel
	clientChannel, clientReqs, err := newChannel.Accept()
	if err != nil {
		logger.Error("[SSHProxy] Failed to accept forwarded-tcpip channel: %v", err)
		return
	}
	defer clientChannel.Close()

	// Try to use connection pool
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				// Open channel on VPS connection
				vpsChannel, vpsReqs, err := vpsConn.sshClient.OpenChannel("forwarded-tcpip", newChannel.ExtraData())
				if err == nil {
					defer vpsChannel.Close()
					// Forward requests
					go ssh.DiscardRequests(clientReqs)
					go ssh.DiscardRequests(vpsReqs)
					// Forward data bidirectionally
					go io.Copy(clientChannel, vpsChannel)
					io.Copy(vpsChannel, clientChannel)
					return
				}
			}
		}
	}

	// Fall back: reject if no connection pool
	logger.Warn("[SSHProxy] Connection pool not available for forwarded-tcpip, rejecting")
	clientChannel.Write([]byte("Remote port forwarding requires connection pool\r\n"))
}

// handleAgentChannel handles SSH agent forwarding (ssh -A)
func (s *SSHProxyServer) handleAgentChannel(ctx context.Context, channel ssh.Channel, vpsID string, keyID string) {
	defer channel.Close()

	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for agent channel: %v", err)
		return
	}

	// Try to use connection pool
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				// Open agent channel on VPS connection
				vpsChannel, _, err := vpsConn.sshClient.OpenChannel("auth-agent@openssh.com", nil)
				if err == nil {
					defer vpsChannel.Close()
					// Forward data bidirectionally
					go io.Copy(channel, vpsChannel)
					io.Copy(vpsChannel, channel)
					return
				}
			}
		}
	}

	// Fall back: close channel if no connection pool
	logger.Warn("[SSHProxy] Connection pool not available for agent forwarding")
}

// handleX11Channel handles X11 forwarding (ssh -X)
func (s *SSHProxyServer) handleX11Channel(ctx context.Context, channel ssh.Channel, vpsID string, keyID string) {
	defer channel.Close()

	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for X11 channel: %v", err)
		return
	}

	// Try to use connection pool
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				// Open X11 channel on VPS connection
				vpsChannel, _, err := vpsConn.sshClient.OpenChannel("x11", nil)
				if err == nil {
					defer vpsChannel.Close()
					// Forward data bidirectionally
					go io.Copy(channel, vpsChannel)
					io.Copy(vpsChannel, channel)
					return
				}
			}
		}
	}

	// Fall back: close channel if no connection pool
	logger.Warn("[SSHProxy] Connection pool not available for X11 forwarding")
}

// handleGenericChannel handles unknown channel types transparently
func (s *SSHProxyServer) handleGenericChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, vpsID string, keyID string, channelType string) {
	defer channel.Close()

	logger.Info("[SSHProxy] Handling generic channel type: %s for VPS %s", channelType, vpsID)

	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for generic channel: %v", err)
		return
	}

	// Try to use connection pool
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			vpsConn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				// Open channel on VPS connection
				vpsChannel, vpsReqs, err := vpsConn.sshClient.OpenChannel(channelType, nil)
				if err == nil {
					defer vpsChannel.Close()
					// Forward requests
					go func() {
						for req := range requests {
							// Forward request to VPS
							ok, err := vpsChannel.SendRequest(req.Type, req.WantReply, req.Payload)
							if req.WantReply {
								if err != nil {
									req.Reply(false, nil)
								} else {
									req.Reply(ok, nil)
								}
							}
						}
					}()
					go func() {
						for req := range vpsReqs {
							// Forward requests from VPS (if any)
							// Most channels don't have requests, but handle them if they do
							req.Reply(false, nil)
						}
					}()
					// Forward data bidirectionally
					go io.Copy(channel, vpsChannel)
					io.Copy(vpsChannel, channel)
					return
				}
			}
		}
	}

	// Fall back: close channel if no connection pool
	logger.Warn("[SSHProxy] Connection pool not available for generic channel type %s", channelType)
}

// handleGlobalRequests handles global SSH requests (port forwarding, etc.)
func (s *SSHProxyServer) handleGlobalRequests(ctx context.Context, reqs <-chan *ssh.Request, sshConn *ssh.ServerConn, vpsID string) {
	// Get VPS IP
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for global requests: %v", err)
		// Discard requests if we can't get VPS IP
		go ssh.DiscardRequests(reqs)
		return
	}

	// Extract key ID from permissions
	var keyID string
	if sshConn.Permissions != nil {
		if kID, ok := sshConn.Permissions.Extensions["key_id"]; ok {
			keyID = kID
		}
	}

	// Get persistent connection if available
	var vpsConn *PooledSSHConnection
	if s.connectionPool != nil && keyID != "" {
		sshSigner, err := s.getSSHSigner(keyID)
		if err == nil {
			conn, err := s.connectionPool.GetOrCreateConnection(ctx, vpsID, vpsIP, keyID, sshSigner)
			if err == nil {
				vpsConn = conn
			}
		}
	}

	if vpsConn == nil {
		// No connection pool available, discard requests
		logger.Debug("[SSHProxy] Connection pool not available for global requests, discarding")
		go ssh.DiscardRequests(reqs)
		return
	}

	// Handle global requests
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			s.handleTCPIPForward(ctx, req, vpsConn, sshConn)
		case "cancel-tcpip-forward":
			s.handleCancelTCPIPForward(ctx, req, vpsConn, sshConn)
		case "streamlocal-forward@openssh.com":
			s.handleStreamLocalForward(ctx, req, vpsConn, sshConn)
		case "cancel-streamlocal-forward@openssh.com":
			s.handleCancelStreamLocalForward(ctx, req, vpsConn, sshConn)
		default:
			// Forward unknown requests
			s.forwardGlobalRequest(ctx, req, vpsConn, sshConn)
		}
	}
}

// handleTCPIPForward handles remote port forwarding requests
func (s *SSHProxyServer) handleTCPIPForward(ctx context.Context, req *ssh.Request, vpsConn *PooledSSHConnection, sshConn *ssh.ServerConn) {
	// Parse request
	var bindAddr struct {
		Addr string
		Port uint32
	}
	if err := ssh.Unmarshal(req.Payload, &bindAddr); err != nil {
		logger.Error("[SSHProxy] Failed to parse tcpip-forward request: %v", err)
		req.Reply(false, nil)
		return
	}

	// Forward request to VPS
	ok, payload, err := vpsConn.sshClient.SendRequest("tcpip-forward", req.WantReply, req.Payload)
	if req.WantReply {
		if err != nil {
			req.Reply(false, nil)
		} else {
			req.Reply(ok, payload)
		}
	}

	if ok {
		logger.Info("[SSHProxy] Forwarded remote port forwarding request: %s:%d", bindAddr.Addr, bindAddr.Port)
	}
}

// handleCancelTCPIPForward handles cancel remote port forwarding requests
func (s *SSHProxyServer) handleCancelTCPIPForward(ctx context.Context, req *ssh.Request, vpsConn *PooledSSHConnection, sshConn *ssh.ServerConn) {
	// Forward cancel request to VPS
	ok, payload, err := vpsConn.sshClient.SendRequest("cancel-tcpip-forward", req.WantReply, req.Payload)
	if req.WantReply {
		if err != nil {
			req.Reply(false, nil)
		} else {
			req.Reply(ok, payload)
		}
	}

	if ok {
		logger.Info("[SSHProxy] Canceled remote port forwarding")
	}
}

// handleStreamLocalForward handles Unix socket forwarding requests
func (s *SSHProxyServer) handleStreamLocalForward(ctx context.Context, req *ssh.Request, vpsConn *PooledSSHConnection, sshConn *ssh.ServerConn) {
	// Forward request to VPS
	ok, payload, err := vpsConn.sshClient.SendRequest("streamlocal-forward@openssh.com", req.WantReply, req.Payload)
	if req.WantReply {
		if err != nil {
			req.Reply(false, nil)
		} else {
			req.Reply(ok, payload)
		}
	}

	if ok {
		logger.Info("[SSHProxy] Forwarded Unix socket forwarding request")
	}
}

// handleCancelStreamLocalForward handles cancel Unix socket forwarding requests
func (s *SSHProxyServer) handleCancelStreamLocalForward(ctx context.Context, req *ssh.Request, vpsConn *PooledSSHConnection, sshConn *ssh.ServerConn) {
	// Forward cancel request to VPS
	ok, payload, err := vpsConn.sshClient.SendRequest("cancel-streamlocal-forward@openssh.com", req.WantReply, req.Payload)
	if req.WantReply {
		if err != nil {
			req.Reply(false, nil)
		} else {
			req.Reply(ok, payload)
		}
	}

	if ok {
		logger.Info("[SSHProxy] Canceled Unix socket forwarding")
	}
}

// forwardGlobalRequest forwards unknown global requests
func (s *SSHProxyServer) forwardGlobalRequest(ctx context.Context, req *ssh.Request, vpsConn *PooledSSHConnection, sshConn *ssh.ServerConn) {
	// Forward request to VPS
	ok, payload, err := vpsConn.sshClient.SendRequest(req.Type, req.WantReply, req.Payload)
	if req.WantReply {
		if err != nil {
			req.Reply(false, nil)
		} else {
			req.Reply(ok, payload)
		}
	}

	logger.Debug("[SSHProxy] Forwarded global request: %s (success: %v)", req.Type, ok)
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
