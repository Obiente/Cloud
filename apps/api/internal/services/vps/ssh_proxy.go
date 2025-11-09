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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"api/internal/database"
	"api/internal/logger"

	"golang.org/x/crypto/ssh"
)

// SSHProxyServer handles SSH jump host functionality for VPS access
type SSHProxyServer struct {
	listener       net.Listener
	hostKey        ssh.Signer
	authorizedKeys map[string]bool // VPS ID -> authorized
	mu             sync.RWMutex
	port           int
}

// NewSSHProxyServer creates a new SSH proxy server
func NewSSHProxyServer(port int) (*SSHProxyServer, error) {
	// Generate or load host key
	hostKey, err := getOrGenerateHostKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key: %w", err)
	}

	server := &SSHProxyServer{
		hostKey:        hostKey,
		authorizedKeys: make(map[string]bool),
		port:           port,
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

	// Create SSH server config
	config := &ssh.ServerConfig{
		PublicKeyCallback: s.authenticate,
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
		go s.handleChannel(ctx, channel, requests, sshConn.User())
	}
}

// authenticate authenticates SSH connections using VPS ID
func (s *SSHProxyServer) authenticate(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// Extract VPS ID from username (format: vps-{vps_id})
	username := conn.User()
	if !strings.HasPrefix(username, "vps-") {
		return nil, fmt.Errorf("invalid username format")
	}

	vpsID := strings.TrimPrefix(username, "vps-")

	// Verify VPS exists and user has access
	// TODO: Implement proper authentication using API tokens or SSH keys
	// For now, we'll allow any valid VPS ID
	s.mu.RLock()
	authorized := s.authorizedKeys[vpsID]
	s.mu.RUnlock()

	if !authorized {
		// Check if VPS exists in database
		var vps database.VPSInstance
		if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
			return nil, fmt.Errorf("VPS not found or access denied")
		}
		s.mu.Lock()
		s.authorizedKeys[vpsID] = true
		s.mu.Unlock()
	}

	logger.Info("[SSHProxy] Authenticated connection for VPS %s", vpsID)
	return &ssh.Permissions{
		Extensions: map[string]string{
			"vps_id": vpsID,
		},
	}, nil
}

// handleChannel handles SSH channel requests
func (s *SSHProxyServer) handleChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, username string) {
	defer channel.Close()

	// Extract VPS ID from username
	vpsID := strings.TrimPrefix(username, "vps-")

	// Get VPS instance to find its internal IP
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		logger.Error("[SSHProxy] VPS %s not found: %v", vpsID, err)
		channel.Write([]byte("VPS not found\r\n"))
		return
	}

	// Parse IP addresses from JSON
	var ipv4Addresses []string
	if vps.IPv4Addresses != "" {
		// TODO: Parse JSON array
		// For now, use instance ID to construct internal IP
		// In a real implementation, we'd query Proxmox for the actual IP
	}

	// Determine target IP (use first IPv4 or construct from instance ID)
	targetIP := ""
	if len(ipv4Addresses) > 0 {
		targetIP = ipv4Addresses[0]
	} else if vps.InstanceID != nil {
		// Construct internal IP from instance ID (placeholder)
		// In production, query Proxmox for actual IP
		targetIP = fmt.Sprintf("10.0.0.%s", *vps.InstanceID)
	} else {
		channel.Write([]byte("VPS IP address not available\r\n"))
		return
	}

	// Handle requests
	for req := range requests {
		switch req.Type {
		case "shell", "exec":
			// Forward to VPS
			if err := s.forwardToVPS(ctx, channel, targetIP, req); err != nil {
				logger.Error("[SSHProxy] Failed to forward to VPS: %v", err)
				channel.Write([]byte(fmt.Sprintf("Connection failed: %v\r\n", err)))
			}
			return
		case "pty-req":
			// Accept PTY request
			req.Reply(true, nil)
		case "window-change":
			// Accept window change
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

// forwardToVPS forwards SSH connection to the actual VPS
func (s *SSHProxyServer) forwardToVPS(ctx context.Context, channel ssh.Channel, targetIP string, req *ssh.Request) error {
	// Use SSH to connect to the VPS
	// In production, this would use the VPS's internal IP and root credentials
	// For now, this is a placeholder that shows the connection would be established

	// Construct SSH command to connect to VPS
	// ssh -o StrictHostKeyChecking=no root@<targetIP>
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"root@"+targetIP,
	)

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH command: %w", err)
	}

	// Forward data bidirectionally
	go func() {
		io.Copy(stdin, channel)
		stdin.Close()
	}()

	go func() {
		io.Copy(channel, stdout)
	}()

	go func() {
		io.Copy(channel.Stderr(), stderr)
	}()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("SSH command failed: %w", err)
	}

	return nil
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
