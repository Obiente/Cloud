package vps

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type vpsTerminalWSMessage struct {
	Type           string `json:"type"`
	VPSID          string `json:"vpsId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	Token          string `json:"token,omitempty"`
	Input          []int  `json:"input,omitempty"`
	Cols           int    `json:"cols,omitempty"`
	Rows           int    `json:"rows,omitempty"`
	Command        string `json:"command,omitempty"`
}

type vpsTerminalWSOutput struct {
	Type    string `json:"type"`
	Data    []int  `json:"data,omitempty"`
	Exit    bool   `json:"exit,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// SSHConnection wraps an SSH connection and session for terminal access
type SSHConnection struct {
	conn    *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
}

// HandleVPSTerminalWebSocket handles WebSocket connections for VPS terminal access
// Uses SSH for terminal access (gateway handles routing for VPSes without public IPs)
func (s *Service) HandleVPSTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract VPS ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var vpsID string
	for i, part := range pathParts {
		if part == "vps" && i+1 < len(pathParts) {
			vpsID = pathParts[i+1]
			break
		}
	}

	if vpsID == "" {
		http.Error(w, "VPS ID not found in URL path", http.StatusBadRequest)
		return
	}

	// Validate origin using CORS configuration
	origin := r.Header.Get("Origin")
	if !middleware.IsOriginAllowed(origin) {
		log.Printf("[VPS Terminal WS] Origin %s not allowed", origin)
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Prepare origin patterns for WebSocket library
	// Check if wildcard CORS is configured - if so, allow all origins
	acceptOptions := &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled, // Disable compression for better performance with binary data
	}
	corsConfig := middleware.DefaultCORSConfig()
	isWildcard := len(corsConfig.AllowedOrigins) == 1 && corsConfig.AllowedOrigins[0] == "*"
	
	if isWildcard {
		// Wildcard CORS configured - allow all origins in WebSocket library
		acceptOptions.OriginPatterns = []string{"*"}
	} else if origin != "" {
		// Specific origins configured - use the validated origin
		acceptOptions.OriginPatterns = []string{origin}
	} else {
		// Empty origin - might be same-origin request, allow all
		acceptOptions.OriginPatterns = []string{"*"}
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to accept websocket connection: %v", err)
		return
	}

	var writeMu sync.Mutex
	writeJSON := func(msg interface{}) error {
		ctxWrite, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		writeMu.Lock()
		defer writeMu.Unlock()
		return wsjson.Write(ctxWrite, conn, msg)
	}

	writeBinary := func(data []byte) error {
		ctxWrite, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.Write(ctxWrite, websocket.MessageBinary, data)
	}

	sendError := func(message string) {
		_ = writeJSON(vpsTerminalWSOutput{
			Type:    "error",
			Message: message,
		})
	}

	closed := false
	defer func() {
		if !closed {
			conn.Close(websocket.StatusNormalClosure, "")
		}
	}()

	// Read the initial message
	var initMsg vpsTerminalWSMessage
	if err := wsjson.Read(ctx, conn, &initMsg); err != nil {
		log.Printf("[VPS Terminal WS] Failed to read init message: %v", err)
		sendError("Failed to read init message")
		conn.Close(websocket.StatusProtocolError, "missing init message")
		return
	}

	if strings.ToLower(initMsg.Type) != "init" {
		sendError("First message must be of type 'init'")
		conn.Close(websocket.StatusUnsupportedData, "expected init message")
		return
	}

	authDisabled := os.Getenv("DISABLE_AUTH") == "true"
	if initMsg.Token == "" {
		if authDisabled {
			initMsg.Token = "dev-dummy-token"
		} else {
			sendError("Authentication token is required")
			conn.Close(websocket.StatusPolicyViolation, "missing token")
			return
		}
	}

	if initMsg.VPSID == "" {
		initMsg.VPSID = vpsID
	}

	if initMsg.VPSID == "" || initMsg.OrganizationID == "" {
		sendError("vpsId and organizationId are required")
		conn.Close(websocket.StatusPolicyViolation, "missing identifiers")
		return
	}

	ctx, userInfo, err := auth.AuthenticateAndSetContext(ctx, "Bearer "+strings.TrimSpace(initMsg.Token))
	if err != nil {
		log.Printf("[VPS Terminal WS] Authentication failed: %v", err)
		sendError("Authentication required")
		conn.Close(websocket.StatusPolicyViolation, "auth failed")
		return
	}

	// Verify permissions
	if err := s.checkVPSPermission(ctx, initMsg.VPSID, "vps.view"); err != nil {
		sendError("Permission denied")
		conn.Close(websocket.StatusPolicyViolation, "permission denied")
		return
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", initMsg.VPSID).First(&vps).Error; err != nil {
		sendError(fmt.Sprintf("VPS instance %s not found", initMsg.VPSID))
		conn.Close(websocket.StatusInternalError, "VPS not found")
		return
	}

	// Extract client IP from request
	clientIP := getClientIPFromRequest(r)

	// Check VPS status
	blockedStatuses := []int32{
		int32(vpsv1.VPSStatus_STOPPING),
		int32(vpsv1.VPSStatus_FAILED),
		int32(vpsv1.VPSStatus_DELETING),
		int32(vpsv1.VPSStatus_DELETED),
	}
	isBlocked := false
	for _, blockedStatus := range blockedStatuses {
		if vps.Status == blockedStatus {
			isBlocked = true
			break
		}
	}

	if isBlocked {
		statusMsg := fmt.Sprintf("VPS terminal access is not available (status: %d).\r\n", vps.Status)
		data := make([]int, len(statusMsg))
		for i, b := range []byte(statusMsg) {
			data[i] = int(b)
		}
		_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: data})
		_ = writeJSON(vpsTerminalWSOutput{Type: "connected"})
		// Keep connection open to display status message, but don't attempt terminal connection
		for {
			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
					return
				}
				return
			}
		}
	}

	// Normalize terminal dimensions
	cols := initMsg.Cols
	if cols <= 0 {
		cols = 80
	}
	rows := initMsg.Rows
	if rows <= 0 {
		rows = 24
	}

	// Get Proxmox client
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		sendError("Failed to get Proxmox config")
		conn.Close(websocket.StatusInternalError, "Proxmox config error")
		return
	}

	proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		sendError("Failed to create Proxmox client")
		conn.Close(websocket.StatusInternalError, "Proxmox client error")
		return
	}

	vmIDInt := 0
	if vps.InstanceID != nil {
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	}
	if vmIDInt == 0 {
		sendError("Invalid VM ID")
		conn.Close(websocket.StatusInternalError, "Invalid VM ID")
		return
	}

	// Find which node the VM is running on
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		sendError(fmt.Sprintf("Failed to find Proxmox node for VM: %v", err))
		conn.Close(websocket.StatusInternalError, "VM node not found")
		return
	}

	// Use SSH for terminal access (xterm.js requires text-based terminal, not VNC)
	// SSH uses the web terminal SSH key (generated per VPS)
	log.Printf("[VPS Terminal WS] Using SSH for terminal access (xterm.js compatible)")

	// Get web terminal SSH key for this VPS
	terminalKey, err := database.GetVPSTerminalKey(initMsg.VPSID)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to get terminal SSH key: %v, falling back to password", err)
		// Fall back to password if key not found (for backwards compatibility)
		terminalKey = nil
	}

	// Try SSH with web terminal key or password fallback
	var sshConn *SSHConnection
	vpsManager, err := orchestrator.NewVPSManager()
	if err == nil {
		defer vpsManager.Close()
		ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, initMsg.VPSID)
		if err == nil && len(ipv4) > 0 {
			vpsIP := ipv4[0]
			if terminalKey != nil {
				// Use web terminal SSH key
				sshConn, err = s.connectSSHWithKey(ctx, initMsg.VPSID, vpsIP, terminalKey.PrivateKey, cols, rows)
				if err != nil {
					log.Printf("[VPS Terminal WS] SSH connection with terminal key failed: %v", err)
				}
			} else {
				// Fall back to password authentication
				rootPassword, err := s.getVPSRootPassword(ctx, initMsg.VPSID)
				if err == nil && rootPassword != "" {
					sshConn, err = s.connectSSH(ctx, initMsg.VPSID, vpsIP, rootPassword, cols, rows)
					if err != nil {
						log.Printf("[VPS Terminal WS] SSH connection with password failed: %v", err)
					}
				}
			}
		} else {
			// No public IP, try to get internal IP from Proxmox
			log.Printf("[VPS Terminal WS] No public IP found, attempting to get internal IP from Proxmox")
			ipv4, _, err := proxmoxClient.GetVMIPAddresses(ctx, nodeName, vmIDInt)
			if err == nil && len(ipv4) > 0 {
				vpsIP := ipv4[0]
				if terminalKey != nil {
					// Use web terminal SSH key
					sshConn, err = s.connectSSHWithKey(ctx, initMsg.VPSID, vpsIP, terminalKey.PrivateKey, cols, rows)
					if err != nil {
						log.Printf("[VPS Terminal WS] SSH connection with terminal key failed: %v", err)
						sshConn = nil
					}
				} else {
					// Fall back to password authentication
					rootPassword, err := s.getVPSRootPassword(ctx, initMsg.VPSID)
					if err == nil && rootPassword != "" {
						sshConn, err = s.connectSSH(ctx, initMsg.VPSID, vpsIP, rootPassword, cols, rows)
						if err != nil {
							log.Printf("[VPS Terminal WS] SSH connection with password failed: %v", err)
							sshConn = nil
						}
					}
				}
			}
		}
	}

	// If SSH connection succeeded, use it
	if sshConn != nil {
		if terminalKey != nil {
			log.Printf("[VPS Terminal WS] Using SSH connection with web terminal key")
		} else {
			log.Printf("[VPS Terminal WS] Using SSH connection with password (fallback)")
		}
		defer func() {
			if sshConn != nil {
				if sshConn.session != nil {
					sshConn.session.Close()
				}
				if sshConn.conn != nil {
					sshConn.conn.Close()
				}
			}
		}()

		// Send connected message
		if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
			log.Printf("[VPS Terminal WS] Failed to send connected message: %v", err)
			return
		}

		// Create audit log for successful web terminal connection
		go createWebTerminalAuditLog(initMsg.VPSID, userInfo, vps.OrganizationID, clientIP, r.UserAgent())

		outputCtx, outputCancel := context.WithCancel(ctx)
		defer outputCancel()

		// Forward SSH output to websocket (as binary for better performance)
		var outputWg sync.WaitGroup
		outputWg.Add(2)

		// Forward stdout
		go func() {
			defer outputWg.Done()
			buf := make([]byte, 4096)
			for {
				select {
				case <-outputCtx.Done():
					return
				default:
				}

				n, err := sshConn.stdout.Read(buf)
				if n > 0 {
					// Send as binary for better performance with xterm
					if err := writeBinary(buf[:n]); err != nil {
						log.Printf("[VPS Terminal WS] Failed to forward output: %v", err)
						return
					}
				}

				if err != nil {
					if err == io.EOF {
						_ = writeJSON(vpsTerminalWSOutput{Type: "closed", Reason: "Terminal session ended", Exit: true})
						conn.Close(websocket.StatusNormalClosure, "terminal closed")
						closed = true
					} else {
						log.Printf("[VPS Terminal WS] SSH read error: %v", err)
						_ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: "Terminal stream error"})
					}
					return
				}
			}
		}()

		// Forward stderr
		go func() {
			defer outputWg.Done()
			buf := make([]byte, 4096)
			for {
				select {
				case <-outputCtx.Done():
					return
				default:
				}

				n, err := sshConn.stderr.Read(buf)
				if n > 0 {
					if err := writeBinary(buf[:n]); err != nil {
						log.Printf("[VPS Terminal WS] Failed to forward stderr: %v", err)
						return
					}
				}

				if err != nil {
					if err != io.EOF {
						log.Printf("[VPS Terminal WS] SSH stderr read error: %v", err)
					}
					return
				}
			}
		}()

		// Handle input and resize
		for {
			select {
			case <-outputCtx.Done():
				return
			default:
			}

			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
					return
				}
				log.Printf("[VPS Terminal WS] Read error: %v", err)
				return
			}

			switch strings.ToLower(msg.Type) {
			case "input":
				if len(msg.Input) == 0 {
					continue
				}

				inputBytes := make([]byte, len(msg.Input))
				for i, v := range msg.Input {
					inputBytes[i] = byte(v)
				}
				if _, err := sshConn.stdin.Write(inputBytes); err != nil {
					log.Printf("[VPS Terminal WS] Failed to write input: %v", err)
					return
				}

			case "resize":
				if sshConn.session != nil && msg.Cols > 0 && msg.Rows > 0 {
					cols = msg.Cols
					rows = msg.Rows
					if err := sshConn.session.WindowChange(rows, cols); err != nil {
						log.Printf("[VPS Terminal WS] Failed to resize terminal: %v", err)
					}
				}

			case "ping":
				_ = writeJSON(map[string]string{"type": "pong"})

			default:
				log.Printf("[VPS Terminal WS] Unknown message type: %s", msg.Type)
			}
		}
	}

	// All connection methods failed
	sendError("Failed to establish terminal connection. Please ensure the VPS is running and SSH is accessible.")
	conn.Close(websocket.StatusInternalError, "Terminal connection failed")
}

// handleProxmoxTermProxy handles terminal access via Proxmox termproxy WebSocket
// termproxy is designed for text terminals and is compatible with xterm.js
// It requires sending an authentication message after WebSocket connection: "username:ticket\n"
func (s *Service) handleProxmoxTermProxy(
	ctx context.Context,
	clientConn *websocket.Conn,
	termProxyInfo *orchestrator.TermProxyInfo,
	proxmoxClient *orchestrator.ProxmoxClient,
	proxmoxConfig *orchestrator.ProxmoxConfig,
	nodeName string,
	vmID int,
	vps *database.VPSInstance,
	writeJSON func(interface{}) error,
	writeBinary func([]byte) error,
	cols, rows int,
) {
	// Parse the WebSocket URL to extract the base URL for cookie domain
	wsURL, err := url.Parse(termProxyInfo.WebSocketURL)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to parse termproxy URL: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Invalid termproxy URL: %v", err))
		return
	}

	// Reuse the ProxmoxClient's HTTP client to preserve authentication state
	baseHTTPClient := proxmoxClient.GetHTTPClient()
	
	// Create a new HTTP client with the same transport but our own cookie jar
	jar := &cookieJar{
		cookies: make(map[string]*http.Cookie),
	}
	
	// Copy the transport from the base client
	var tr *http.Transport
	if baseHTTPClient.Transport != nil {
		if baseTr, ok := baseHTTPClient.Transport.(*http.Transport); ok {
			tr = baseTr
		}
	}
	if tr == nil {
		// Fallback: create new transport with same config
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   baseHTTPClient.Timeout,
		Jar:       jar,
	}
	
	// Proxmox WebSocket authentication per API documentation:
	// The vncwebsocket endpoint (used by termproxy) requires:
	// 1. A valid vncticket in the URL (from termproxy call)
	// 2. Authentication for the WebSocket upgrade request itself
	headers := make(http.Header)
	
	if proxmoxConfig.TokenID != "" && proxmoxConfig.Secret != "" {
		// API token authentication - use Authorization header for WebSocket upgrade
		authHeader := proxmoxClient.GetAuthHeader()
		if authHeader != "" {
			headers.Set("Authorization", authHeader)
			log.Printf("[VPS Terminal WS] Using API token authentication (Authorization header + vncticket)")
		}
	} else {
		// Password-based auth - use PVEAuthCookie cookie for WebSocket upgrade
		authCookie, err := proxmoxClient.GetOrCreateTicketForWebSocket(ctx)
		if err != nil {
			log.Printf("[VPS Terminal WS] Failed to get ticket for WebSocket: %v", err)
		} else if authCookie != "" {
			cookie := &http.Cookie{
				Name:  "PVEAuthCookie",
				Value: authCookie,
				Path:  "/",
			}
			jar.SetCookies(wsURL, []*http.Cookie{cookie})
			log.Printf("[VPS Terminal WS] Using password-based auth (PVEAuthCookie cookie + vncticket)")
		}
	}

	// Connect to Proxmox termproxy WebSocket
	dialOptions := &websocket.DialOptions{
		HTTPClient: httpClient,
	}
	if len(headers) > 0 {
		dialOptions.HTTPHeader = headers
	}

	log.Printf("[VPS Terminal WS] Connecting to termproxy WebSocket: %s", termProxyInfo.WebSocketURL)

	proxmoxConn, _, err := websocket.Dial(ctx, termProxyInfo.WebSocketURL, dialOptions)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to connect to Proxmox termproxy: %v (URL: %s)", err, termProxyInfo.WebSocketURL)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Failed to connect to termproxy: %v. Falling back to SSH.", err))
		return
	}
	defer proxmoxConn.Close(websocket.StatusInternalError, "")

	log.Printf("[VPS Terminal WS] Connected to Proxmox termproxy WebSocket for VM %d", vmID)

	// Send termproxy authentication message: "username:ticket\n"
	// Per Proxmox documentation and source code, the username should be WITHOUT token ID
	// Format: username@realm:ticket\n (both for password auth and API tokens)
	// The token ID is not included in the termproxy authentication message
	// Reference: Proxmox termproxy source code expects username@realm format
	authUser := termProxyInfo.User
	// Remove token ID if present (termproxy auth doesn't use token ID in username)
	if idx := strings.Index(authUser, "!"); idx != -1 {
		authUser = authUser[:idx]
		log.Printf("[VPS Terminal WS] Removed token ID from username for termproxy auth: %s -> %s", termProxyInfo.User, authUser)
	}
	
	authMsg := fmt.Sprintf("%s:%s\n", authUser, termProxyInfo.Ticket)
	ticketPreview := termProxyInfo.Ticket
	if len(ticketPreview) > 20 {
		ticketPreview = ticketPreview[:20]
	}
	log.Printf("[VPS Terminal WS] Sending termproxy authentication: %s:%s...", authUser, ticketPreview)
	if err := proxmoxConn.Write(ctx, websocket.MessageText, []byte(authMsg)); err != nil {
		log.Printf("[VPS Terminal WS] Failed to send termproxy authentication: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Failed to authenticate termproxy: %v", err))
		return
	}
	
	// Send connected message to client immediately after sending auth
	// Proxmox termproxy will start sending data once authenticated
	if err := writeJSON(vpsTerminalWSOutput{Type: "connected"}); err != nil {
		log.Printf("[VPS Terminal WS] Failed to send connected message: %v", err)
		return
	}

	// Proxy messages bidirectionally using binary for better performance
	errChan := make(chan error, 2)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Helper to safely send errors
	sendError := func(err error) {
		select {
		case errChan <- err:
		case <-ctx.Done():
		default:
			// Channel might be full or closed, ignore
		}
	}

	// Store any initial data we might have read during auth
	var initialData []byte

	// Forward binary data from Proxmox termproxy to client (as binary)
	go func() {
		defer func() {
			cancel() // Signal other goroutine to stop
		}()
		
		// If we have initial data from auth check, send it first
		if len(initialData) > 0 {
			if err := writeBinary(initialData); err != nil {
				log.Printf("[VPS Terminal WS] Failed to forward initial data: %v", err)
				sendError(fmt.Errorf("failed to forward initial data: %w", err))
				return
			}
		}
		
		for {
			messageType, data, err := proxmoxConn.Read(ctx)
			if err != nil {
				// Check if connection was closed
				if websocket.CloseStatus(err) != -1 {
					closeStatus := websocket.CloseStatus(err)
					log.Printf("[VPS Terminal WS] Proxmox termproxy connection closed with status %d: %v", closeStatus, err)
					sendError(nil)
					return
				}
				// EOF or other read error
				if err == io.EOF {
					log.Printf("[VPS Terminal WS] Proxmox termproxy connection closed (EOF)")
				} else {
					log.Printf("[VPS Terminal WS] Error reading from Proxmox termproxy: %v", err)
				}
				sendError(fmt.Errorf("failed to read from Proxmox termproxy: %w", err))
				return
			}

			// Log message type for debugging
			if messageType != websocket.MessageBinary && messageType != websocket.MessageText {
				log.Printf("[VPS Terminal WS] Received unexpected message type: %d", messageType)
			}

			// Send data directly to client (xterm handles binary efficiently)
			if err := writeBinary(data); err != nil {
				log.Printf("[VPS Terminal WS] Failed to write to client: %v", err)
				sendError(fmt.Errorf("failed to write to client: %w", err))
				return
			}
		}
	}()

	// Forward input from client to Proxmox termproxy
	go func() {
		defer func() {
			cancel() // Signal other goroutine to stop
		}()
		for {
			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, clientConn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					sendError(nil)
					return
				}
				sendError(fmt.Errorf("failed to read from client: %w", err))
				return
			}

			switch strings.ToLower(msg.Type) {
			case "input":
				// Convert input array to bytes
				if len(msg.Input) > 0 {
					inputBytes := make([]byte, len(msg.Input))
					for i, v := range msg.Input {
						inputBytes[i] = byte(v)
					}
					// Write binary data to Proxmox termproxy WebSocket
					if err := proxmoxConn.Write(ctx, websocket.MessageBinary, inputBytes); err != nil {
						sendError(fmt.Errorf("failed to write to Proxmox termproxy: %w", err))
						return
					}
				}

			case "resize":
				// Terminal resize is handled automatically by Proxmox
				_ = msg.Cols
				_ = msg.Rows

			case "ping":
				_ = writeJSON(map[string]string{"type": "pong"})

			default:
				log.Printf("[VPS Terminal WS] Unknown message type from client: %s", msg.Type)
			}
		}
	}()

	// Wait for an error or context cancellation
	select {
	case <-ctx.Done():
		log.Printf("[VPS Terminal WS] Context cancelled for termproxy WebSocket")
	case err := <-errChan:
		if err != nil {
			log.Printf("[VPS Terminal WS] termproxy WebSocket proxy error: %v", err)
			_ = writeJSON(vpsTerminalWSOutput{
				Type:    "error",
				Message: fmt.Sprintf("termproxy error: %v", err),
			})
		}
	}
}

// createSSHClientViaGateway creates an SSH client connection to a VPS via the gateway
func (s *Service) createSSHClientViaGateway(ctx context.Context, vpsID, vpsIP string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create VPS manager: %w", err)
	}
	defer vpsManager.Close()

	gatewayClient := vpsManager.GetGatewayClient()
	if gatewayClient == nil {
		return nil, fmt.Errorf("gateway client not available")
	}

	// Create TCP connection via gateway
	targetConn, err := gatewayClient.CreateTCPConnection(ctx, vpsIP, 22)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP connection via gateway: %w", err)
	}

	// Create SSH connection over the gateway TCP connection
	clientConn, chans, reqs, err := ssh.NewClientConn(targetConn, vpsIP, sshConfig)
	if err != nil {
		targetConn.Close()
		return nil, fmt.Errorf("failed to create SSH client connection: %w", err)
	}

	client := ssh.NewClient(clientConn, chans, reqs)
	return client, nil
}

// setupSSHSession sets up a PTY session with pipes for terminal access
func setupSSHSession(client *ssh.Client, cols, rows int) (*ssh.Session, io.WriteCloser, io.Reader, io.Reader, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request PTY
	if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.IGNCR:         0,
		ssh.ICRNL:         1,
		ssh.ONLCR:         1,
	}); err != nil {
		session.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start shell
	if err := session.Shell(); err != nil {
		stdin.Close()
		session.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to start shell: %w", err)
	}

	return session, stdin, stdout, stderr, nil
}

// connectSSHWithKey establishes an SSH connection to the VPS using a private key
// Always uses gateway for connection
func (s *Service) connectSSHWithKey(ctx context.Context, vpsID, vpsIP, privateKeyPEM string, cols, rows int) (*SSHConnection, error) {
	// Parse private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	// Create SSH client via gateway
	client, err := s.createSSHClientViaGateway(ctx, vpsID, vpsIP, sshConfig)
	if err != nil {
		return nil, err
	}

	// Set up session with PTY and pipes
	session, stdin, stdout, stderr, err := setupSSHSession(client, cols, rows)
	if err != nil {
		client.Close()
		return nil, err
	}

	return &SSHConnection{
		conn:    client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// connectSSH establishes an SSH connection to the VPS using password authentication
// Always uses gateway for connection
func (s *Service) connectSSH(ctx context.Context, vpsID, vpsIP, rootPassword string, cols, rows int) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.Password(rootPassword),
		},
	}

	// Create SSH client via gateway
	client, err := s.createSSHClientViaGateway(ctx, vpsID, vpsIP, sshConfig)
	if err != nil {
		return nil, err
	}

	// Set up session with PTY and pipes
	session, stdin, stdout, stderr, err := setupSSHSession(client, cols, rows)
	if err != nil {
		client.Close()
		return nil, err
	}

	return &SSHConnection{
		conn:    client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}


// getVPSRootPassword retrieves the root password from Proxmox VM config
func (s *Service) getVPSRootPassword(ctx context.Context, vpsID string) (string, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return "", fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return "", fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return "", fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return "", fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	// Get VM config from Proxmox
	vmConfig, err := proxmoxClient.GetVMConfig(ctx, nodes[0], vmIDInt)
	if err != nil {
		return "", fmt.Errorf("failed to get VM config: %w", err)
	}

	// Try to get password from cipassword (cloud-init password)
	if cipassword, ok := vmConfig["cipassword"].(string); ok && cipassword != "" {
		return cipassword, nil
	}

	return "", fmt.Errorf("root password not found in VM config")
}

// cookieJar is a simple cookie jar implementation for WebSocket connections
type cookieJar struct {
	cookies map[string]*http.Cookie
	mu      sync.Mutex
}

func (j *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, cookie := range cookies {
		j.cookies[cookie.Name] = cookie
	}
}

func (j *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	var result []*http.Cookie
	for _, cookie := range j.cookies {
		result = append(result, cookie)
	}
	return result
}

// getClientIPFromRequest extracts the client IP address from an HTTP request
// Traefik is configured with forwardedHeaders middleware to properly forward the real client IP
func getClientIPFromRequest(r *http.Request) string {
	// Try CF-Connecting-IP (Cloudflare)
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}

	// Try True-Client-IP (used by some proxies)
	if trueClientIP := r.Header.Get("True-Client-IP"); trueClientIP != "" {
		return strings.TrimSpace(trueClientIP)
	}

	// Try X-Forwarded-For header (Traefik sets this with forwardedHeaders middleware)
	// Format: "client-ip, proxy1-ip, proxy2-ip, ..."
	// The first IP is the original client IP
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Try X-Real-IP header (nginx and some proxies)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip := strings.TrimSpace(realIP)
		if ip != "" {
			return ip
		}
	}

	// Try X-Client-IP (some proxies)
	if clientIP := r.Header.Get("X-Client-IP"); clientIP != "" {
		ip := strings.TrimSpace(clientIP)
		if ip != "" {
			return ip
		}
	}

	// Fallback: use RemoteAddr
	if r.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			return host
		}
		return r.RemoteAddr
	}

	return "unknown"
}

// createWebTerminalAuditLog creates an audit log entry for a web terminal connection
func createWebTerminalAuditLog(vpsID string, userInfo *authv1.User, organizationID string, clientIP, userAgent string) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[VPS Terminal WS] Panic creating audit log for web terminal connection: %v", r)
		}
	}()

	// Use background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use MetricsDB (TimescaleDB) for audit logs
	if database.MetricsDB == nil {
		logger.Warn("[VPS Terminal WS] Metrics database not available, skipping audit log for web terminal connection")
		return
	}

	// Determine user ID
	userID := "system"
	if userInfo != nil {
		userID = userInfo.Id
	}

	// Determine organization ID
	var orgID *string
	if organizationID != "" {
		orgID = &organizationID
	}

	// Create request data
	requestData := fmt.Sprintf(`{"vps_id":"%s","connection_type":"web_terminal"}`, vpsID)

	// Set user agent
	if userAgent == "" {
		userAgent = "unknown"
	}

	// Create audit log entry
	auditLog := database.AuditLog{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: orgID,
		Action:         "WebTerminalConnect",
		Service:        "VPSTerminalService",
		ResourceType:   stringPtrForTerminal("vps"),
		ResourceID:     &vpsID,
		IPAddress:      clientIP,
		UserAgent:      userAgent,
		RequestData:    requestData,
		ResponseStatus: 200,
		ErrorMessage:   nil,
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		logger.Warn("[VPS Terminal WS] Failed to create audit log for web terminal connection: %v", err)
	} else {
		logger.Debug("[VPS Terminal WS] Created audit log for web terminal connection: user=%s, vps=%s, ip=%s", userID, vpsID, clientIP)
	}
}

// stringPtrForTerminal returns a pointer to a string
func stringPtrForTerminal(s string) *string {
	return &s
}
