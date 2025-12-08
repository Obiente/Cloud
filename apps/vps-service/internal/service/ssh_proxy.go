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

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/text"
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

// isInternalIP checks if an IP address is an internal/private network IP
func isInternalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for private/internal IP ranges
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, ::1
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// extractRealClientIP attempts to extract the real client IP from a TCP connection
// For raw TCP connections, we can only get the IP from RemoteAddr
// If the connection is behind a proxy/NAT, this will be the proxy/NAT IP
func extractRealClientIP(conn net.Conn) string {
	clientIP := extractIPFromAddr(conn.RemoteAddr())

	// Check if we got an internal IP (likely Docker network or NAT)
	if isInternalIP(clientIP) {
		logger.Warn("[SSHProxy] Detected internal IP %s - real client IP may be obscured by proxy/NAT. Consider using PROXY protocol if behind Traefik.", clientIP)
	}

	return clientIP
}

// createBox creates a formatted ASCII box with the given title and content lines.
// The box automatically adjusts width based on content and uses proper box-drawing characters.
// Maximum width is limited to 80 characters for terminal compatibility.
func createBox(title string, contentLines []string) string {
	const maxBoxWidth = 80 // Maximum box width for terminal compatibility

	// Calculate maximum width needed
	maxWidth := text.StringWidth(title)
	for _, line := range contentLines {
		// Wrap each line first to get actual display width
		wrapped := text.WrapHard(line, maxBoxWidth-4)
		wrappedLines := strings.Split(wrapped, "\n")
		for _, wrappedLine := range wrappedLines {
			width := text.StringWidth(wrappedLine)
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	// Add padding (2 chars on each side = 4 total)
	boxWidth := maxWidth + 4
	if boxWidth < text.StringWidth(title)+4 {
		boxWidth = text.StringWidth(title) + 4
	}

	// Cap at maximum width
	if boxWidth > maxBoxWidth {
		boxWidth = maxBoxWidth
	}

	// Build the box
	var result strings.Builder
	result.WriteString("\r\n")

	// Top border
	result.WriteString("╔")
	result.WriteString(strings.Repeat("═", boxWidth-2))
	result.WriteString("╗\r\n")

	// Title line (centered)
	titlePadding := (boxWidth - 2 - text.StringWidth(title)) / 2
	result.WriteString("║")
	result.WriteString(strings.Repeat(" ", titlePadding))
	result.WriteString(title)
	result.WriteString(strings.Repeat(" ", boxWidth-2-titlePadding-text.StringWidth(title)))
	result.WriteString("║\r\n")

	// Separator if there's content
	if len(contentLines) > 0 {
		result.WriteString("╠")
		result.WriteString(strings.Repeat("═", boxWidth-2))
		result.WriteString("╣\r\n")
	}

	// Content lines
	for _, line := range contentLines {
		if line == "" {
			// Empty line
			result.WriteString("║")
			result.WriteString(strings.Repeat(" ", boxWidth-2))
			result.WriteString("║\r\n")
		} else {
			// Wrap long lines
			wrapped := text.WrapHard(line, boxWidth-4)
			wrappedLines := strings.Split(wrapped, "\n")
			for _, wrappedLine := range wrappedLines {
				result.WriteString("║")
				result.WriteString(" ")
				result.WriteString(wrappedLine)
				// Pad to box width
				padding := boxWidth - 2 - text.StringWidth(wrappedLine) - 1
				if padding > 0 {
					result.WriteString(strings.Repeat(" ", padding))
				}
				result.WriteString("║\r\n")
			}
		}
	}

	// Bottom border
	result.WriteString("╚")
	result.WriteString(strings.Repeat("═", boxWidth-2))
	result.WriteString("╝\r\n")

	return result.String()
}

// createErrorBox creates a formatted error box with title and content.
func createErrorBox(title string, contentLines []string) string {
	return createBox(title, contentLines)
}

// setClientIPEnv sets SSH environment variables to forward the client's real IP address.
func setClientIPEnv(channel ssh.Channel, clientIP, serverIP string) {
	sshClient := fmt.Sprintf("%s 0 22", clientIP)
	sshConnection := fmt.Sprintf("%s 0 %s 22", clientIP, serverIP)

	var envSetCount int
	var envRejectedCount int

	setEnv := func(name, value string) {
		payload := make([]byte, 0, 4+len(name)+4+len(value))
		payload = append(payload, byte(len(name)>>24), byte(len(name)>>16), byte(len(name)>>8), byte(len(name)))
		payload = append(payload, []byte(name)...)
		payload = append(payload, byte(len(value)>>24), byte(len(value)>>16), byte(len(value)>>8), byte(len(value)))
		payload = append(payload, []byte(value)...)

		ok, err := channel.SendRequest("env", false, payload)
		if err != nil {
			logger.Warn("[SSHProxy] Failed to set env %s: %v", name, err)
		} else if !ok {
			envRejectedCount++
			logger.Warn("[SSHProxy] VPS rejected env %s (may need AcceptEnv SSH_CLIENT SSH_CONNECTION SSH_CLIENT_REAL in sshd_config)", name)
		} else {
			envSetCount++
			logger.Debug("[SSHProxy] Set env %s=%s", name, value)
		}
	}

	setEnv("SSH_CLIENT", sshClient)
	setEnv("SSH_CONNECTION", sshConnection)
	setEnv("SSH_CLIENT_REAL", clientIP)

	if envRejectedCount > 0 {
		logger.Warn("[SSHProxy] VPS rejected %d environment variable(s) - real client IP (%s) may not be available in lastlog", envRejectedCount, clientIP)
	} else {
		logger.Info("[SSHProxy] Successfully set %d environment variable(s) for real client IP: %s", envSetCount, clientIP)
	}
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

			// Wrap connection to parse PROXY protocol if present
			realConn, realIP := s.parseProxyProtocol(conn)
			if realIP != "" {
				logger.Info("[SSHProxy] Accepted connection with PROXY protocol: real IP=%s, connection from=%s", realIP, conn.RemoteAddr())
			} else {
				logger.Info("[SSHProxy] Accepted connection from %s (no PROXY protocol)", conn.RemoteAddr())
			}

			s.activeConnections.Add(1)
			go func() {
				defer s.activeConnections.Done()
				s.handleConnection(ctx, realConn, realIP)
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

// parseProxyProtocol attempts to parse PROXY protocol v1 or v2 from the connection
// Returns the connection (wrapped if PROXY protocol was present) and the real client IP
// If PROXY protocol is not present, returns the original connection and empty string
func (s *SSHProxyServer) parseProxyProtocol(conn net.Conn) (net.Conn, string) {
	// Set a short read timeout to detect PROXY protocol without blocking
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	// Try to read PROXY protocol header (max 108 bytes for v2, or first line for v1)
	buffer := make([]byte, 108)
	n, err := conn.Read(buffer)
	conn.SetReadDeadline(time.Time{}) // Clear deadline

	if err != nil {
		// No data or timeout - no PROXY protocol, return original connection
		return conn, ""
	}

	// Check for PROXY protocol v1 (text-based, starts with "PROXY ")
	if n >= 6 && string(buffer[:6]) == "PROXY " {
		// Parse PROXY protocol v1
		// Format: "PROXY TCP4|TCP6 <srcip> <dstip> <srcport> <dstport>\r\n"
		line := string(buffer[:n])
		// Remove trailing \r\n if present
		line = strings.TrimSuffix(line, "\r\n")
		line = strings.TrimSuffix(line, "\n")
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Extract source IP (real client IP) - it's the 3rd field
			realIP := parts[2]
			// Validate it's a valid IP
			if net.ParseIP(realIP) != nil {
				logger.Debug("[SSHProxy] Parsed PROXY protocol v1: real IP=%s", realIP)
				// Create a wrapper that prepends the read data back
				return &proxyProtocolConn{Conn: conn, prefix: buffer[:n]}, realIP
			}
		}
	}

	// Check for PROXY protocol v2 (binary)
	// V2 starts with: 0x0D 0x0A 0x0D 0x0A 0x00 0x0D 0x0A 0x51 0x55 0x49 0x54 0x0A
	if n >= 12 && buffer[0] == 0x0D && buffer[1] == 0x0A && buffer[2] == 0x0D && buffer[3] == 0x0A &&
		buffer[4] == 0x00 && buffer[5] == 0x0D && buffer[6] == 0x0A && buffer[7] == 0x51 &&
		buffer[8] == 0x55 && buffer[9] == 0x49 && buffer[10] == 0x54 && buffer[11] == 0x0A {
		// Parse PROXY protocol v2 (simplified - just extract IP for now)
		// V2 is complex, for now we'll log and return empty (can be enhanced later)
		logger.Debug("[SSHProxy] Detected PROXY protocol v2 (binary parsing not yet implemented)")
		return &proxyProtocolConn{Conn: conn, prefix: buffer[:n]}, ""
	}

	// No PROXY protocol detected, prepend the read data back
	return &proxyProtocolConn{Conn: conn, prefix: buffer[:n]}, ""
}

// proxyProtocolConn wraps a connection and prepends data that was read for PROXY protocol detection
type proxyProtocolConn struct {
	net.Conn
	prefix     []byte
	prefixRead int
}

func (c *proxyProtocolConn) Read(b []byte) (n int, err error) {
	// First, return the prefix data that was already read
	if c.prefixRead < len(c.prefix) {
		n = copy(b, c.prefix[c.prefixRead:])
		c.prefixRead += n
		if n < len(b) {
			// Still have space, read from underlying connection
			more, err := c.Conn.Read(b[n:])
			return n + more, err
		}
		return n, nil
	}
	// Prefix exhausted, read from underlying connection
	return c.Conn.Read(b)
}

// handleConnection handles an incoming SSH connection using a bastion pattern.
// It extracts the VPS ID from the client's username and forwards SSH channels to the VPS.
// realIP is the client's real IP if extracted from PROXY protocol, otherwise empty.
func (s *SSHProxyServer) handleConnection(ctx context.Context, clientConn net.Conn, realIP string) {
	logger.Info("[SSHProxy] Handling connection from %s", clientConn.RemoteAddr())

	if s.gatewayClient == nil {
		logger.Error("[SSHProxy] Gateway not available, cannot proxy SSH connection")
		clientConn.Close()
		return
	}

	username, serverConn, chans, reqs, authInfo, err := s.extractVPSIDAndEstablishConnection(ctx, clientConn)
	if err != nil {
		logger.Warn("[SSHProxy] Failed to extract username and establish connection: %v", err)

		// Try to extract and resolve VPS ID from error or username to show helpful error
		vpsID := ""
		if username != "" {
			vpsIdentifier, _ := parseUsername(username)
			// Try to resolve identifier (could be alias or full VPS ID) to actual VPS ID
			resolvedID, resolveErr := resolveVPSID(vpsIdentifier)
			if resolveErr == nil {
				vpsID = resolvedID
			} else {
				// If resolution fails, use the identifier as-is (might be a non-existent VPS)
				// But still try to extract from error message if it contains a VPS ID
				vpsID = vpsIdentifier
			}
		}

		// If we still don't have a VPS ID, try to extract from error message
		if vpsID == "" || (!strings.HasPrefix(vpsID, "vps-") && !strings.Contains(err.Error(), "not found")) {
			errStr := err.Error()
			// Check if error contains "VPS not found: <identifier>"
			if strings.Contains(errStr, "VPS not found:") {
				// Extract the identifier from the error message
				parts := strings.Split(errStr, "VPS not found:")
				if len(parts) > 1 {
					identifier := strings.TrimSpace(parts[1])
					// Remove any trailing commas, periods, or other punctuation
					identifier = strings.TrimRight(identifier, ",.")
					identifier = strings.TrimSpace(identifier)
					// For "VPS not found" errors, use the identifier that was attempted
					vpsID = identifier
				}
			} else if strings.Contains(errStr, "vps-") {
				// Try to extract VPS ID from error message
				parts := strings.Fields(errStr)
				for _, part := range parts {
					if strings.HasPrefix(part, "vps-") {
						vpsID = part
						break
					}
				}
			}
		}

		// Extract client IP for audit logging
		clientIP := extractRealClientIP(clientConn)
		if realIP != "" {
			clientIP = realIP
		}

		// Create audit log for failed SSH connection
		go createFailedSSHAuditLog(vpsID, username, err, clientIP)

		// Send formatted error message to client before closing
		if vpsID != "" {
			errorMsg := s.formatConnectionError(err, vpsID)
			// Write error directly to the connection
			// Note: This might not work if SSH handshake has already started,
			// but we try anyway to show the error if possible
			clientConn.Write([]byte("\r\n" + errorMsg + "\r\n"))
			time.Sleep(500 * time.Millisecond) // Give client time to read the error
		} else {
			// Even if we don't have a VPS ID, try to show a generic error
			genericError := createErrorBox("Connection Error", []string{
				"Failed to establish SSH connection.",
				"",
				"Please verify your connection details and try again.",
				"",
				fmt.Sprintf("Error: %s", err.Error()),
			})
			clientConn.Write([]byte("\r\n" + genericError + "\r\n"))
			time.Sleep(500 * time.Millisecond)
		}

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

	// Parse username to extract VPS ID/alias and target user
	vpsIdentifier, targetUser := parseUsername(username)

	// Resolve identifier (could be full VPS ID or alias) to actual VPS ID
	vpsID, err := resolveVPSID(vpsIdentifier)
	if err != nil {
		logger.Warn("[SSHProxy] VPS not found (ID or alias: %s): %v", vpsIdentifier, err)
		return
	}

	logger.Info("[SSHProxy] Resolved VPS identifier %s to VPS ID: %s, target user: %s", vpsIdentifier, vpsID, targetUser)

	// Get VPS instance to access name
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		logger.Error("[SSHProxy] Failed to get VPS instance for %s: %v", vpsID, err)
		return
	}

	// Get VPS IP or hostname
	vpsIP, err := s.getVPSIP(ctx, vpsID)
	if err != nil {
		logger.Error("[SSHProxy] Failed to get VPS IP for %s: %v", vpsID, err)
		return
	}

	logger.Info("[SSHProxy] Forwarding SSH channels to VPS %s at %s as user %s", vpsID, vpsIP, targetUser)

	// Use real IP from PROXY protocol if available, otherwise extract from connection
	var clientIP string
	if realIP != "" {
		clientIP = realIP
		logger.Info("[SSHProxy] Using real client IP from PROXY protocol: %s", clientIP)
	} else {
		// Extract client IP from connection
		// Note: For SSH connections through Docker Swarm ingress, this will be the Docker network IP.
		// PROXY protocol is needed to get the real client IP when behind a load balancer.
		clientIP = extractRealClientIP(clientConn)
		logger.Info("[SSHProxy] Client IP: %s (connection from %s, no PROXY protocol)", clientIP, clientConn.RemoteAddr())
	}

	// Create audit log for SSH connection
	go createSSHAuditLog(vpsID, targetUser, authInfo, clientIP)

	// Handle global requests in background
	go s.handleGlobalRequests(ctx, reqs, vpsID, vpsIP, authInfo)

	// Forward channels - this blocks until all channels are closed
	s.forwardChannelsToVPS(ctx, serverConn, chans, vpsID, vpsIP, targetUser, authInfo, clientIP, vps.Name)

	logger.Info("[SSHProxy] All channels closed, connection ending for VPS %s", vpsID)
}

// parseUsername extracts VPS ID and target user from username
// Supports formats:
// - "user@vps-xxx" (standard SSH format with full VPS ID)
// - "user@alias" (standard SSH format with SSH alias)
// - "user@vps-xxx@hostname" (with hostname, hostname is ignored)
// - "user@alias@hostname" (with hostname, hostname is ignored)
// - "vps-xxx" (defaults to root, full VPS ID)
// - "alias" (defaults to root, SSH alias)
func parseUsername(username string) (vpsID, targetUser string) {
	// Count @ signs to determine format
	atCount := strings.Count(username, "@")

	if atCount >= 2 {
		// Format: user@identifier@hostname (e.g., root@test@localhost)
		// Find first @ to get user, second @ to get identifier
		firstAt := strings.Index(username, "@")
		secondAt := strings.Index(username[firstAt+1:], "@")
		if secondAt != -1 {
			targetUser = username[:firstAt]
			vpsID = username[firstAt+1 : firstAt+1+secondAt]
			return
		}
	} else if atCount == 1 {
		// Format: user@identifier (e.g., root@test or root@vps-xxx)
		idx := strings.Index(username, "@")
		targetUser = username[:idx]
		vpsID = username[idx+1:]
		return
	}

	// Try : format for backwards compatibility (e.g., vps-xxx:user)
	if idx := strings.LastIndex(username, ":"); idx != -1 {
		vpsID = username[:idx]
		targetUser = username[idx+1:]
		return
	}

	// No separator, use entire username as VPS ID or alias, default to root
	vpsID = username
	targetUser = "root"
	return
}

// resolveVPSID resolves a VPS identifier (either full ID or alias) to the actual VPS ID
func resolveVPSID(identifier string) (string, error) {
	return database.ResolveVPSIDFromSSHIdentifierAnyOrg(identifier)
}

// connectSSHToVPSForChannelForwarding creates an SSH client connection to the target VPS via gateway.
// Uses the bastion SSH key for authentication.
// If serverConn is provided, it can be used to send status messages to the client.
func (s *SSHProxyServer) connectSSHToVPSForChannelForwarding(ctx context.Context, serverConn *ssh.ServerConn, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo) (*ssh.Client, error) {
	// Get VPS to find node name for gateway selection
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, fmt.Errorf("failed to get VPS %s: %w", vpsID, err)
	}

	// Get gateway client for the node where VPS is running
	var gatewayClient *orchestrator.VPSGatewayClient
	if vps.NodeID != nil && *vps.NodeID != "" {
		vpsManager, err := orchestrator.NewVPSManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create VPS manager: %w", err)
		}
		defer vpsManager.Close()

		client, err := vpsManager.GetGatewayClientForNode(*vps.NodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get gateway client for node %s: %w", *vps.NodeID, err)
		}
		gatewayClient = client
	} else {
		// Fallback to global gateway client if node not set (for backwards compatibility during migration)
		if s.gatewayClient == nil {
			return nil, fmt.Errorf("VPS %s has no node name and no global gateway client available", vpsID)
		}
		gatewayClient = s.gatewayClient
		logger.Warn("[SSHProxy] VPS %s has no node name, using global gateway client (should be migrated)", vpsID)
	}

	// Create TCP connection to VPS via gateway
	targetConn, err := gatewayClient.CreateTCPConnection(ctx, vpsIP, 22)
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
		// We already have vps from above

		// Create bastion key
		bastionKey, err = database.CreateVPSBastionKey(vpsID, vps.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bastion SSH key for VPS %s: %w", vpsID, err)
		}
		logger.Info("[SSHProxy] Auto-created bastion key for VPS %s (fingerprint: %s)", vpsID, bastionKey.Fingerprint)

		// Regenerate cloud-init to include the new bastion key
		// Only do this if VPS is already provisioned (has instance ID)
		if vps.InstanceID != nil {
			// Parse VM ID
			vmIDInt := 0
			fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
			if vmIDInt > 0 {
				// Get node name from VPS (required)
				nodeName := ""
				if vps.NodeID != nil && *vps.NodeID != "" {
					nodeName = *vps.NodeID
				} else {
					logger.Warn("[SSHProxy] VPS %s has no node ID - skipping cloud-init update after auto-creating bastion key", vpsID)
				}
				if nodeName != "" {
					// Get VPS manager to get Proxmox client for the node
					vpsManager, err := orchestrator.NewVPSManager()
					if err == nil {
						defer vpsManager.Close()
						proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
						if err == nil {
							// Update cloud-init config to include the new bastion key
							// We need to load current config, which will automatically include the new key
							// when GenerateCloudInitUserData is called
							vpsConfig := &orchestrator.VPSConfig{
								VPSID:          vps.ID,
								OrganizationID: vps.OrganizationID,
								CloudInit:      nil, // Use default/current config
							}

							// Generate cloud-init userData (will include the new bastion key)
							userData := orchestrator.GenerateCloudInitUserData(vpsConfig)

							// Get storage for snippets
							storage := "local"
							if storageEnv := os.Getenv("PROXMOX_STORAGE"); storageEnv != "" {
								storage = storageEnv
							}

							// Upload cloud-init snippet
							snippetPath, snippetErr := proxmoxClient.CreateCloudInitSnippet(ctx, nodeName, storage, vmIDInt, userData)
							if snippetErr == nil {
								// Update VM config with cicustom parameter
								if updateErr := proxmoxClient.UpdateVMCicustom(ctx, nodeName, vmIDInt, snippetPath); updateErr == nil {
									// Note: When using snippets, cloud-init changes are automatically applied on the next VM boot.
									logger.Info("[SSHProxy] Successfully updated cloud-init after auto-creating bastion key for VPS %s. Changes will be applied on next boot.", vpsID)
								} else {
									logger.Warn("[SSHProxy] Failed to update VM cicustom after auto-creating bastion key for VPS %s: %v", vpsID, updateErr)
								}
							} else {
								logger.Warn("[SSHProxy] Failed to create cloud-init snippet after auto-creating bastion key for VPS %s: %v", vpsID, snippetErr)
							}
						}
					}
				}
			}
		}
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
	logger.Debug("[SSHProxy] Using bastion key fingerprint: %s", bastionKey.Fingerprint)

	clientConn, chans, reqs, err := ssh.NewClientConn(targetConn, vpsIP, sshConfig)
	if err != nil {
		targetConn.Close()
		logger.Warn("[SSHProxy] Failed to connect to VPS %s with bastion key: %v", vpsID, err)
		logger.Warn("[SSHProxy] Bastion key fingerprint: %s", bastionKey.Fingerprint)
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

	// Store VPS validation result to share between callbacks
	type vpsValidation struct {
		validated  bool
		vpsID      string
		identifier string
		err        error
	}
	vpsValidationResult := &vpsValidation{}

	// Helper function to validate VPS
	validateVPS := func(username string) (string, error) {
		if vpsValidationResult.validated {
			if vpsValidationResult.err != nil {
				return "", vpsValidationResult.err
			}
			return vpsValidationResult.vpsID, nil
		}

		// Parse username to get VPS ID or alias
		vpsIdentifier, _ := parseUsername(username)
		vpsValidationResult.identifier = vpsIdentifier

		// Resolve identifier (could be full VPS ID or alias) to actual VPS ID
		vpsID, err := resolveVPSID(vpsIdentifier)
		if err != nil {
			logger.Warn("[SSHProxy] VPS not found (ID or alias: %s): %v", vpsIdentifier, err)
			vpsValidationResult.validated = true
			vpsValidationResult.err = fmt.Errorf("VPS not found: %s", vpsIdentifier)
			return "", vpsValidationResult.err
		}

		// Verify VPS exists in database
		var vps database.VPSInstance
		if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
			logger.Warn("[SSHProxy] VPS not found in database: %s", vpsID)
			vpsValidationResult.validated = true
			vpsValidationResult.err = fmt.Errorf("VPS not found: %s", vpsIdentifier)
			return "", vpsValidationResult.err
		}

		vpsValidationResult.validated = true
		vpsValidationResult.vpsID = vpsID
		return vpsID, nil
	}

	config := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-ObienteCloud",
		BannerCallback: func(conn ssh.ConnMetadata) string {
			// Validate VPS in banner callback - this is called early
			username := conn.User()
			_, err := validateVPS(username)
			if err != nil {
				// Return error message as banner - this will be displayed to user
				identifier := vpsValidationResult.identifier
				errorBox := createErrorBox("Connection Error", []string{
					"The specified VPS was not found.",
					"",
					"Please verify the VPS ID or SSH alias and ensure you have access to it.",
					"",
					"You can use either:",
					"• Full VPS ID: vps-xxx",
					"• SSH alias: your-alias (if configured)",
					"",
					fmt.Sprintf("Identifier used: %s", identifier),
				})
				return errorBox
			}
			return createBox("Welcome to Obiente Cloud SSH Bastion", []string{})
		},
		NoClientAuthCallback: func(conn ssh.ConnMetadata) (*ssh.Permissions, error) {
			// This callback is called early in the handshake
			// We use it to validate VPS exists and reject immediately if it doesn't
			username := conn.User()
			logger.Debug("[SSHProxy] NoClientAuthCallback for user: %s", username)

			vpsID, err := validateVPS(username)
			if err != nil {
				// Reject immediately to prevent password prompts
				return nil, err
			}

			// Store resolved VPS ID for later use
			authInfo.vpsID = vpsID

			// Allow authentication to proceed
			return nil, nil
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			extractedUsername = conn.User()
			logger.Debug("[SSHProxy] Validating public key authentication for user: %s", extractedUsername)

			// Validate VPS exists (uses cached result if already validated)
			vpsID, err := validateVPS(extractedUsername)
			if err != nil {
				return nil, err
			}

			// Get VPS to find organization
			var vps database.VPSInstance
			if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
				logger.Warn("[SSHProxy] VPS not found: %s", vpsID)
				return nil, fmt.Errorf("VPS not found: %s", vpsValidationResult.identifier)
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

			// Validate VPS exists (uses cached result if already validated)
			vpsID, err := validateVPS(extractedUsername)
			if err != nil {
				return nil, err
			}

			// Get VPS to find organization
			var vps database.VPSInstance
			if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
				logger.Warn("[SSHProxy] VPS not found: %s", vpsID)
				return nil, fmt.Errorf("VPS not found: %s", vpsValidationResult.identifier)
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
			// Check permissions using unified permission checker
			pc := auth.NewPermissionChecker()
			if err := pc.CheckResourcePermission(ctx, "vps", vpsID, "vps.read"); err == nil {
				logger.Info("[SSHProxy] User authenticated via API token for VPS %s", vpsID)
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
		// Try to extract username and VPS ID from the connection attempt
		// The error might contain the VPS ID, or we might have captured it during auth callbacks
		var resolvedVPSID string
		if authInfo.vpsID != "" {
			// We have the resolved VPS ID from auth callback
			resolvedVPSID = authInfo.vpsID
		} else if extractedUsername != "" {
			// Try to resolve from username (might be alias)
			vpsIdentifier, _ := parseUsername(extractedUsername)
			if resolvedID, resolveErr := resolveVPSID(vpsIdentifier); resolveErr == nil {
				resolvedVPSID = resolvedID
			}
		}

		// If we still don't have a VPS ID, try to extract from error message
		if resolvedVPSID == "" {
			errStr := err.Error()
			// Check if error contains "VPS not found: <identifier>"
			if strings.Contains(errStr, "VPS not found:") {
				// Extract the identifier from the error message
				parts := strings.Split(errStr, "VPS not found:")
				if len(parts) > 1 {
					identifier := strings.TrimSpace(parts[1])
					// Remove any trailing commas, periods, or other punctuation
					identifier = strings.TrimRight(identifier, ",.")
					identifier = strings.TrimSpace(identifier)
					// For "VPS not found" errors, use the identifier that was attempted (don't try to resolve)
					resolvedVPSID = identifier
				}
			} else if strings.Contains(errStr, "vps-") {
				// Try to extract VPS ID from error message
				parts := strings.Fields(errStr)
				for _, part := range parts {
					if strings.HasPrefix(part, "vps-") {
						resolvedVPSID = part
						break
					}
				}
			}
		}

		// If we have a resolved VPS ID or identifier, include it in the error for better error messages
		if resolvedVPSID != "" {
			// Check if error is about VPS not found
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "VPS not found") {
				return "", nil, nil, nil, nil, fmt.Errorf("VPS not found: %s", resolvedVPSID)
			}
			// For other errors, wrap with VPS ID context
			return "", nil, nil, nil, nil, fmt.Errorf("failed to establish SSH connection to VPS %s: %w", resolvedVPSID, err)
		}

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
	publicKey          ssh.PublicKey
	password           string
	authMethod         string
	userID             string
	vpsID              string
	organizationID     string
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

// sendMessageToChannel sends a message to the client via a session channel.
// This is used to show connection status, errors, and other messages.
func (s *SSHProxyServer) sendMessageToChannel(channel ssh.Channel, message string, isError bool) {
	if isError {
		channel.Stderr().Write([]byte(message))
	} else {
		channel.Write([]byte(message))
	}
}

// sendConnectionStatus sends connection status messages with loading indicators to a channel
func (s *SSHProxyServer) sendConnectionStatus(channel ssh.Channel, vpsID, targetUser string) {
	statusMessages := []string{
		fmt.Sprintf("Connecting to your VPS (%s) as user %s", vpsID, targetUser),
		".",
		".",
		".",
		"\r\n",
	}

	for _, msg := range statusMessages {
		s.sendMessageToChannel(channel, msg, false)
		time.Sleep(300 * time.Millisecond)
	}
}

// forwardChannelsToVPS forwards SSH channels from the client to the VPS.
func (s *SSHProxyServer) forwardChannelsToVPS(ctx context.Context, serverConn *ssh.ServerConn, chans <-chan ssh.NewChannel, vpsID, vpsIP, targetUser string, authInfo *clientAuthInfo, clientIP string, vpsName string) {
	var vpsClient *ssh.Client
	var connectionErr error
	var connectionEstablished sync.Once
	var firstSessionChannel sync.Once

	// Try to establish VPS connection in background
	connectionReady := make(chan struct{})
	go func() {
		vpsClient, connectionErr = s.connectSSHToVPSForChannelForwarding(ctx, serverConn, vpsID, vpsIP, targetUser, authInfo)
		close(connectionReady)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-connectionReady:
			// Connection attempt completed, continue to handle channels
		case newChannel, ok := <-chans:
			if !ok {
				// Channels closed, send goodbye if we had a connection
				if vpsClient != nil {
					// Goodbye message will be sent when channel closes
				}
				return
			}

			channelType := newChannel.ChannelType()
			logger.Debug("[SSHProxy] Received new channel from client: %s", channelType)

			// For session channels, send connection status before forwarding (only for first one)
			if channelType == "session" {
				handled := false
				firstSessionChannel.Do(func() {
					handled = true
					// Accept the channel first
					clientChannel, clientReqs, err := newChannel.Accept()
					if err != nil {
						logger.Error("[SSHProxy] Failed to accept session channel: %v", err)
						return
					}

					// Wait for connection attempt to complete
					<-connectionReady

					if connectionErr != nil {
						// Send error message
						errorMsg := s.formatConnectionError(connectionErr, vpsID)
						s.sendMessageToChannel(clientChannel, errorMsg, true)
						time.Sleep(2 * time.Second)
						clientChannel.Close()
						serverConn.Close()
						return
					}

					// Send connection status
					s.sendConnectionStatus(clientChannel, vpsID, targetUser)

					// Now forward the channel
					connectionEstablished.Do(func() {
						logger.Info("[SSHProxy] Connected to VPS, forwarding channels...")
						// Use VPS name for display (fallback to ID if name is empty)
						displayName := vpsID
						if vpsName != "" {
							displayName = vpsName
						}
						successMsg := fmt.Sprintf("✓ Successfully connected to VPS %s\r\n", displayName)
						s.sendMessageToChannel(clientChannel, successMsg, false)
						time.Sleep(500 * time.Millisecond)
					})

					// Forward this channel with goodbye message
					go s.forwardChannelWithGoodbye(ctx, clientChannel, clientReqs, vpsClient, serverConn, clientIP, vpsIP)
				})
				// If this is not the first session channel, handle it normally
				if !handled {
					// This is a subsequent session channel, forward normally
					go func() {
						<-connectionReady
						if connectionErr != nil {
							newChannel.Reject(ssh.ConnectionFailed, connectionErr.Error())
							return
						}
						s.forwardChannel(ctx, newChannel, vpsClient, serverConn, clientIP, vpsIP)
					}()
				}
			} else {
				// For non-session channels, wait for connection and forward normally
				go func() {
					<-connectionReady
					if connectionErr != nil {
						newChannel.Reject(ssh.ConnectionFailed, connectionErr.Error())
						return
					}
					s.forwardChannel(ctx, newChannel, vpsClient, serverConn, clientIP, vpsIP)
				}()
			}
		}
	}
}

// forwardChannelWithGoodbye forwards a session channel and sends goodbye message on close
func (s *SSHProxyServer) forwardChannelWithGoodbye(ctx context.Context, clientChannel ssh.Channel, clientReqs <-chan *ssh.Request, vpsClient *ssh.Client, serverConn *ssh.ServerConn, clientIP, vpsIP string) {
	defer func() {
		// Send goodbye message before closing
		goodbyeMsg := createBox("Thank you for using Obiente Cloud!", []string{
			"Connection closed.",
		})
		s.sendMessageToChannel(clientChannel, goodbyeMsg, false)
		time.Sleep(500 * time.Millisecond)
		clientChannel.Close()
	}()

	// Open session channel on VPS
	vpsChannel, vpsReqs, err := vpsClient.OpenChannel("session", nil)
	if err != nil {
		logger.Error("[SSHProxy] Failed to open session channel on VPS: %v", err)
		return
	}
	defer vpsChannel.Close()

	// Set environment variables to forward client's real IP
	if clientIP != "" {
		logger.Debug("[SSHProxy] Setting environment variables for real client IP: %s", clientIP)
		setClientIPEnv(vpsChannel, clientIP, vpsIP)
	} else {
		logger.Warn("[SSHProxy] No client IP available to forward to VPS")
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

	// Forward client requests to VPS
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

	// Forward VPS requests to server
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
}

// formatConnectionError formats an error into a user-friendly message with actionable guidance
func (s *SSHProxyServer) formatConnectionError(err error, vpsID string) string {
	errStr := strings.ToLower(err.Error())

	// Check for specific error types and provide helpful messages

	// 1. Bastion key not configured in database
	if strings.Contains(errStr, "bastion key") || strings.Contains(errStr, "bastion ssh key") {
		return createErrorBox("Connection Error", []string{
			"The bastion SSH key for this VPS is not configured in the database.",
			"",
			"Please visit the VPS SSH tab in the dashboard and",
			"ensure the bastion key is properly configured.",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 2. SSH authentication failure - bastion key missing on VPS
	// This happens when the key exists in DB but not on the VPS (e.g., keys removed, cloud-init not run)
	if strings.Contains(errStr, "unable to authenticate") ||
		strings.Contains(errStr, "no supported methods remain") ||
		(strings.Contains(errStr, "handshake failed") && strings.Contains(errStr, "publickey")) {
		// Get bastion key info for better error message
		bastionKey, keyErr := database.GetVPSBastionKey(vpsID)
		fingerprint := "unknown"
		if keyErr == nil && bastionKey != nil {
			fingerprint = bastionKey.Fingerprint
		}

		return createErrorBox("Authentication Error", []string{
			"The bastion SSH key is not configured on the VPS.",
			"",
			"This usually means:",
			"• The VPS needs to be rebooted to apply cloud-init changes",
			"• Cloud-init hasn't run yet after key rotation",
			"• The SSH keys were manually removed from the VPS",
			"",
			"To fix this:",
			"1. Reboot the VPS to trigger cloud-init",
			"   OR manually run on the VPS:",
			"   cloud-init clean && cloud-init init",
			"",
			"2. Verify the key was added:",
			fmt.Sprintf("   cat ~/.ssh/authorized_keys | grep %s", fingerprint),
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 3. Connection timeout
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "i/o timeout") {
		return createErrorBox("Connection Timeout", []string{
			"Failed to connect to the VPS within the timeout period.",
			"",
			"This may be due to:",
			"• VPS is not running or is unresponsive",
			"• Network connectivity issues",
			"• Firewall blocking SSH connections",
			"• Gateway service unavailable",
			"",
			"Please check:",
			"• VPS status in the dashboard",
			"• Network connectivity",
			"• Firewall rules",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 4. Connection refused (VPS not listening on SSH port)
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "refused") {
		return createErrorBox("Connection Refused", []string{
			"The VPS is not accepting SSH connections.",
			"",
			"This may be due to:",
			"• SSH service is not running on the VPS",
			"• VPS is not running",
			"• Firewall is blocking SSH port 22",
			"",
			"Please check:",
			"• VPS status in the dashboard",
			"• SSH service status on the VPS",
			"• Firewall configuration",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 5. VPS not found
	if strings.Contains(errStr, "vps not found") {
		return createErrorBox("Connection Error", []string{
			"The specified VPS was not found.",
			"",
			"Please verify the VPS ID or SSH alias and ensure you have access to it.",
			"",
			"You can use either:",
			"• Full VPS ID: vps-xxx",
			"• SSH alias: your-alias (if configured)",
			"",
			fmt.Sprintf("Identifier used: %s", vpsID),
		})
	}

	// 6. TCP connection / Gateway errors
	if strings.Contains(errStr, "tcp connection") || strings.Contains(errStr, "gateway") ||
		strings.Contains(errStr, "failed to create tcp connection") {
		return createErrorBox("Connection Error", []string{
			"Failed to establish connection to the VPS via gateway.",
			"",
			"This may be due to:",
			"• VPS is not running",
			"• Network connectivity issues",
			"• Gateway service unavailable",
			"• VPS IP address is incorrect",
			"",
			"Please check:",
			"• VPS status in the dashboard",
			"• Gateway service status",
			"• VPS network configuration",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 7. Host key verification errors (shouldn't happen with InsecureIgnoreHostKey, but handle it)
	if strings.Contains(errStr, "host key") || strings.Contains(errStr, "hostkey") {
		return createErrorBox("Host Key Error", []string{
			"Host key verification failed.",
			"",
			"This is an internal error. Please contact support.",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 8. SSH handshake failures (generic)
	if strings.Contains(errStr, "handshake failed") {
		return createErrorBox("SSH Handshake Failed", []string{
			"The SSH handshake with the VPS failed.",
			"",
			"This may be due to:",
			"• SSH service configuration issues on the VPS",
			"• Authentication method not supported",
			"• Network issues during handshake",
			"",
			"Please check:",
			"• VPS SSH configuration",
			"• Network connectivity",
			"• VPS logs for SSH errors",
			"",
			fmt.Sprintf("VPS ID: %s", vpsID),
		})
	}

	// 9. Generic error message with full error details
	errorLines := []string{
		"Failed to connect to VPS.",
		"",
		fmt.Sprintf("Error: %s", err.Error()),
		"",
		"Please check:",
		"• VPS status in the dashboard",
		"• VPS configuration",
		"• Network connectivity",
		"",
		fmt.Sprintf("VPS ID: %s", vpsID),
	}
	return createErrorBox("Connection Error", errorLines)
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
		vmIDInt := 0
		if vps.InstanceID != nil {
			fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		}
		if vmIDInt > 0 {
			// Get node name from VPS (required)
			nodeName := ""
			if vps.NodeID != nil && *vps.NodeID != "" {
				nodeName = *vps.NodeID
			} else {
				logger.Warn("[SSHProxy] VPS %s has no node ID - cannot get IP from Proxmox", vpsID)
			}
			if nodeName != "" {
				// Get VPS manager to get Proxmox client for the node
				vpsManager, err := orchestrator.NewVPSManager()
				if err == nil {
					defer vpsManager.Close()
					proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
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

// createSSHAuditLog creates an audit log entry for a successful SSH connection
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

	// Create audit log entry for successful connection
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

// createFailedSSHAuditLog creates an audit log entry for a failed SSH connection attempt
func createFailedSSHAuditLog(vpsID, username string, err error, clientIP string) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[SSHProxy] Panic creating audit log for failed SSH connection: %v", r)
		}
	}()

	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use MetricsDB (TimescaleDB) for audit logs
	if database.MetricsDB == nil {
		logger.Warn("[SSHProxy] Metrics database not available, skipping audit log for failed SSH connection")
		return
	}

	// Extract VPS ID from username if not provided
	if vpsID == "" && username != "" {
		vpsIdentifier, _ := parseUsername(username)
		if resolvedID, resolveErr := resolveVPSID(vpsIdentifier); resolveErr == nil {
			vpsID = resolvedID
		} else {
			vpsID = vpsIdentifier // Use identifier as-is if resolution fails
		}
	}

	// Try to get organization ID from VPS if we have a valid VPS ID
	var orgID *string
	if vpsID != "" && strings.HasPrefix(vpsID, "vps-") {
		var vps database.VPSInstance
		if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err == nil {
			orgID = &vps.OrganizationID
		}
	}

	// Determine error status code based on error type
	errorStatus := int32(401) // Default to unauthorized
	errorMsg := err.Error()
	if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "VPS not found") {
		errorStatus = 404 // Not found
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "i/o timeout") {
		errorStatus = 408 // Request timeout
	} else if strings.Contains(errorMsg, "connection refused") {
		errorStatus = 503 // Service unavailable
	} else if strings.Contains(errorMsg, "authentication") || strings.Contains(errorMsg, "unable to authenticate") {
		errorStatus = 401 // Unauthorized
	}

	// Create request data
	requestData := fmt.Sprintf(`{"vps_id":"%s","username":"%s","error_type":"%s"}`, vpsID, username, errorMsg)

	// Create audit log entry for failed connection
	auditLog := database.AuditLog{
		ID:             uuid.New().String(),
		UserID:         "unknown", // Failed connections don't have authenticated user
		OrganizationID: orgID,
		Action:         "SSHConnect",
		Service:        "SSHProxyService",
		ResourceType:   stringPtr("vps"),
		ResourceID:     stringPtr(vpsID),
		IPAddress:      clientIP,
		UserAgent:      "SSH/unknown",
		RequestData:    requestData,
		ResponseStatus: errorStatus,
		ErrorMessage:   &errorMsg,
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		logger.Warn("[SSHProxy] Failed to create audit log for failed SSH connection: %v", err)
	} else {
		logger.Debug("[SSHProxy] Created audit log for failed SSH connection: vps=%s, ip=%s, error=%s", vpsID, clientIP, errorMsg)
	}
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
