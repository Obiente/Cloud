package vps

import (
	"context"
	"crypto/tls"
	"encoding/json"
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

	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

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

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled, // Disable compression for better performance with binary data
	})
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

	ctx, _, err = auth.AuthenticateAndSetContext(ctx, "Bearer "+strings.TrimSpace(initMsg.Token))
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

	// Use SSH only for terminal access (gateway handles routing for VPSes without public IPs)
	log.Printf("[VPS Terminal WS] Using SSH for terminal access")

	var sshConn *SSHConnection
	vpsManager, err := orchestrator.NewVPSManager()
	if err == nil {
		defer vpsManager.Close()
		ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, initMsg.VPSID)
		if err == nil && len(ipv4) > 0 {
			vpsIP := ipv4[0]
			rootPassword, err := s.getVPSRootPassword(ctx, initMsg.VPSID)
			if err == nil && rootPassword != "" {
				// Try direct SSH first
				sshConn, err = s.connectSSH(ctx, vpsIP, rootPassword, cols, rows, "", "")
				if err != nil {
					log.Printf("[VPS Terminal WS] Direct SSH connection failed: %v", err)
					// Connection failed - will try to get internal IP if available
				}
			}
		} else {
			// No public IP, try to get internal IP from Proxmox
			log.Printf("[VPS Terminal WS] No public IP found, attempting to get internal IP from Proxmox")
			ipv4, _, err := proxmoxClient.GetVMIPAddresses(ctx, nodeName, vmIDInt)
			if err == nil && len(ipv4) > 0 {
				vpsIP := ipv4[0]
				rootPassword, err := s.getVPSRootPassword(ctx, initMsg.VPSID)
				if err == nil && rootPassword != "" {
					// Use direct connection (gateway will handle routing if configured)
					sshConn, err = s.connectSSH(ctx, vpsIP, rootPassword, cols, rows, "", "")
					if err != nil {
						log.Printf("[VPS Terminal WS] SSH connection failed: %v", err)
						sshConn = nil
					}
				}
			}
		}
	}

	// If SSH connection succeeded, use it
	if sshConn != nil {
		log.Printf("[VPS Terminal WS] Using SSH connection")
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

	// SSH connection failed - send error to client
	sendError("Failed to establish SSH connection to VPS. Please ensure the VPS is running and SSH is accessible.")
	conn.Close(websocket.StatusInternalError, "SSH connection failed")
	return
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
	
	// Wait briefly for authentication response
	// Proxmox may send an initial response or start sending terminal data immediately
	time.Sleep(100 * time.Millisecond)

	// Send connected message to client
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

	// Forward binary data from Proxmox termproxy to client (as binary)
	go func() {
		defer func() {
			cancel() // Signal other goroutine to stop
		}()
		// Small delay to allow connection to stabilize
		time.Sleep(50 * time.Millisecond)
		
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

// handleProxmoxSerialConsole handles terminal access via Proxmox Serial Console WebSocket
// This is the preferred method as it provides boot output and works even when VM is booting
func (s *Service) handleProxmoxSerialConsole(
	ctx context.Context,
	clientConn *websocket.Conn,
	serialWSURL string,
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
	wsURL, err := url.Parse(serialWSURL)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to parse serial console URL: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Invalid serial console URL: %v", err))
		return
	}

	// Reuse the ProxmoxClient's HTTP client to preserve authentication state
	// This ensures cookies and transport settings are consistent
	baseHTTPClient := proxmoxClient.GetHTTPClient()
	
	// Create a new HTTP client with the same transport but our own cookie jar
	// We need to add cookies for password-based auth if needed
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
	// The vncwebsocket endpoint requires:
	// 1. A valid vncticket in the URL (from vncproxy call)
	// 2. Authentication for the WebSocket upgrade request itself
	// Reference: https://pve.proxmox.com/pve-docs-8/api-viewer/index.html#/nodes/{node}/qemu/{vmid}/vncwebsocket
	// 
	// For API tokens: Use Authorization header for WebSocket upgrade
	// For password auth: Use PVEAuthCookie cookie for WebSocket upgrade
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

	// Connect to Proxmox Serial Console WebSocket
	dialOptions := &websocket.DialOptions{
		HTTPClient: httpClient,
	}
	if len(headers) > 0 {
		dialOptions.HTTPHeader = headers
	}

	log.Printf("[VPS Terminal WS] Connecting to serial console WebSocket: %s (auth: %s)", 
		serialWSURL, 
		func() string {
			if proxmoxConfig.TokenID != "" {
				return "API token"
			}
			if authCookie := proxmoxClient.GetAuthCookie(); authCookie != "" {
				return "password (cookie)"
			}
			return "vncticket only"
		}())

	proxmoxConn, _, err := websocket.Dial(ctx, serialWSURL, dialOptions)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to connect to Proxmox Serial Console: %v (URL: %s)", err, serialWSURL)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Failed to connect to serial console: %v. Falling back to SSH or guest agent.", err))
		return
	}
	defer proxmoxConn.Close(websocket.StatusInternalError, "")

	log.Printf("[VPS Terminal WS] Connected to Proxmox Serial Console WebSocket for VM %d", vmID)
	// No authentication message needed - vncticket in URL is sufficient per Proxmox API docs

	// Send connected message to client
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

	// Forward binary data from Proxmox Serial Console to client (as binary)
	// Note: When using vncwebsocket with websocket=1, we may initially receive RFB protocol
	// handshake data ("RFB 003.008") which needs to be filtered out before serial console data
	go func() {
		defer func() {
			cancel() // Signal other goroutine to stop
		}()
		// Small delay to allow connection to stabilize
		time.Sleep(50 * time.Millisecond)
		
		// Track if we've seen the RFB handshake
		seenRFBHandshake := false
		buffer := make([]byte, 0, 4096)
		
		for {
			messageType, data, err := proxmoxConn.Read(ctx)
			if err != nil {
				// Check if connection was closed
				if websocket.CloseStatus(err) != -1 {
					closeStatus := websocket.CloseStatus(err)
					log.Printf("[VPS Terminal WS] Proxmox connection closed with status %d: %v", closeStatus, err)
					sendError(nil)
					return
				}
				// EOF or other read error
				if err == io.EOF {
					log.Printf("[VPS Terminal WS] Proxmox connection closed (EOF) - connection may have been rejected or closed by server")
				} else {
					log.Printf("[VPS Terminal WS] Error reading from Proxmox: %v", err)
				}
				sendError(fmt.Errorf("failed to read from Proxmox Serial Console: %w", err))
				return
			}

			// Log message type for debugging
			if messageType != websocket.MessageBinary && messageType != websocket.MessageText {
				log.Printf("[VPS Terminal WS] Received unexpected message type: %d", messageType)
			}

			// Accumulate data in buffer
			buffer = append(buffer, data...)
			
			// Check for RFB protocol handshake at the start
			if !seenRFBHandshake && len(buffer) >= 12 {
				// RFB handshake is "RFB 003.008\n" (12 bytes)
				if string(buffer[:12]) == "RFB 003.008\n" {
					log.Printf("[VPS Terminal WS] Detected RFB protocol handshake, filtering it out")
					buffer = buffer[12:]
					seenRFBHandshake = true
				} else if len(buffer) > 100 {
					// If we have more than 100 bytes and no RFB handshake, assume it's serial data
					seenRFBHandshake = true
				}
			}
			
			// Once we've processed the RFB handshake (or determined there isn't one),
			// send all accumulated data to the client
			if seenRFBHandshake && len(buffer) > 0 {
				if err := writeBinary(buffer); err != nil {
					log.Printf("[VPS Terminal WS] Failed to write to client: %v", err)
					sendError(fmt.Errorf("failed to write to client: %w", err))
					return
				}
				buffer = buffer[:0] // Clear buffer
			}
		}
	}()

	// Forward input from client to Proxmox Serial Console
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
					// Write binary data to Proxmox Serial Console WebSocket
					if err := proxmoxConn.Write(ctx, websocket.MessageBinary, inputBytes); err != nil {
						sendError(fmt.Errorf("failed to write to Proxmox Serial Console: %w", err))
						return
					}
				}

			case "resize":
				// Serial console resize is handled automatically by Proxmox
				// Terminal dimensions are stored for reference
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
		log.Printf("[VPS Terminal WS] Context cancelled for Serial Console WebSocket")
	case err := <-errChan:
		if err != nil {
			log.Printf("[VPS Terminal WS] Serial Console WebSocket proxy error: %v", err)
			_ = writeJSON(vpsTerminalWSOutput{
				Type:    "error",
				Message: fmt.Sprintf("Serial Console error: %v", err),
			})
		}
	}
}

// connectSSH establishes an SSH connection to the VPS
// Connects directly to VPS via SSH (gateway handles routing if configured)
// jumpHost and jumpUser parameters are deprecated and ignored (gateway is used instead)
func (s *Service) connectSSH(ctx context.Context, vpsIP, rootPassword string, cols, rows int, jumpHost, jumpUser string) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.Password(rootPassword),
		},
	}

	// Direct connection (gateway handles routing if configured)
	// jumpHost and jumpUser parameters are ignored - gateway is used instead
	client, err := ssh.Dial("tcp", net.JoinHostPort(vpsIP, "22"), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VPS via SSH: %w", err)
	}

	// Create session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request PTY with xterm-256color for better compatibility
	if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.IGNCR:         0,
		ssh.ICRNL:         1,
		ssh.ONLCR:         1,
	}); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start shell
	if err := session.Shell(); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	return &SSHConnection{
		conn:    client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// handleProxmoxGuestAgentTerminal handles terminal access via Proxmox guest agent (no IP needed)
func (s *Service) handleProxmoxGuestAgentTerminal(
	ctx context.Context,
	conn *websocket.Conn,
	proxmoxClient *orchestrator.ProxmoxClient,
	nodeName string,
	vmID int,
	vps *database.VPSInstance,
	writeJSON func(interface{}) error,
	initMsg vpsTerminalWSMessage,
) {
	// Send welcome message
	welcomeMsg := fmt.Sprintf("Connected to VPS %s (%s) via Proxmox Guest Agent\r\n", vps.Name, vps.ID)
	welcomeMsg += "Note: Terminal access via guest agent. Some interactive features may be limited.\r\n"
	welcomeMsg += "Type 'exit' to disconnect.\r\n\r\n"
	data := make([]int, len(welcomeMsg))
	for i, b := range []byte(welcomeMsg) {
		data[i] = int(b)
	}
	_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: data})
	_ = writeJSON(vpsTerminalWSOutput{Type: "connected"})

	// Maintain a shell session state
	currentDir := "/root"
	shellPrompt := func() string {
		return fmt.Sprintf("root@%s:%s$ ", vps.Name, currentDir)
	}

	// Send initial prompt
	prompt := shellPrompt()
	promptData := make([]int, len(prompt))
	for i, b := range []byte(prompt) {
		promptData[i] = int(b)
	}
	_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: promptData})

	// Buffer for command input
	var commandBuffer []byte

	// Listen for client input messages
	for {
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

			// Convert input to bytes
			inputBytes := make([]byte, len(msg.Input))
			for i, v := range msg.Input {
				inputBytes[i] = byte(v)
			}

			// Handle special keys
			for _, b := range inputBytes {
				if b == '\r' || b == '\n' {
					// Execute command
					if len(commandBuffer) > 0 {
						command := strings.TrimSpace(string(commandBuffer))
						commandBuffer = commandBuffer[:0]

						// Echo newline
						echoData := []int{13, 10} // \r\n
						_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: echoData})

						// Handle special commands
						if command == "exit" || command == "logout" {
							exitMsg := "Disconnecting...\r\n"
							exitData := make([]int, len(exitMsg))
							for i, b := range []byte(exitMsg) {
								exitData[i] = int(b)
							}
							_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: exitData})
							return
						}

						// Execute command via guest agent
						output, err := s.executeGuestAgentCommand(ctx, proxmoxClient, nodeName, vmID, command, currentDir)
						if err != nil {
							errorMsg := fmt.Sprintf("Error: %v\r\n", err)
							errorData := make([]int, len(errorMsg))
							for i, b := range []byte(errorMsg) {
								errorData[i] = int(b)
							}
							_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: errorData})
						} else {
							// Send command output
							if len(output) > 0 {
								outputData := make([]int, len(output))
								for i, b := range []byte(output) {
									outputData[i] = int(b)
								}
								_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: outputData})
							}
						}

						// Update current directory if command was 'cd'
						if strings.HasPrefix(command, "cd ") {
							newDir := strings.TrimSpace(command[3:])
							if newDir == "" {
								newDir = "/root"
							} else if newDir == "~" {
								newDir = "/root"
							} else if !strings.HasPrefix(newDir, "/") {
								newDir = currentDir + "/" + newDir
							}
							// Verify directory exists
							checkCmd := fmt.Sprintf("test -d %s && echo %s || echo %s", newDir, newDir, currentDir)
							dirOutput, _ := s.executeGuestAgentCommand(ctx, proxmoxClient, nodeName, vmID, checkCmd, currentDir)
							if strings.TrimSpace(dirOutput) != "" {
								currentDir = strings.TrimSpace(dirOutput)
							}
						}

						// Send prompt
						prompt := shellPrompt()
						promptData := make([]int, len(prompt))
						for i, b := range []byte(prompt) {
							promptData[i] = int(b)
						}
						_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: promptData})
					} else {
						// Empty command, just send prompt
						prompt := shellPrompt()
						promptData := make([]int, len(prompt))
						for i, b := range []byte(prompt) {
							promptData[i] = int(b)
						}
						_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: promptData})
					}
				} else if b == 127 || b == 8 { // Backspace
					// Handle backspace
					if len(commandBuffer) > 0 {
						commandBuffer = commandBuffer[:len(commandBuffer)-1]
						// Echo backspace
						bsData := []int{8, 32, 8} // backspace, space, backspace
						_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: bsData})
					}
				} else if b >= 32 && b <= 126 {
					// Printable ASCII characters
					commandBuffer = append(commandBuffer, b)
				}
			}

		case "resize":
			// Terminal resize (not fully supported via guest agent)
			_ = msg.Cols
			_ = msg.Rows

		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})

		default:
			log.Printf("[VPS Terminal WS] Unknown message type: %s", msg.Type)
		}
	}
}

// executeGuestAgentCommand executes a command via Proxmox guest agent
func (s *Service) executeGuestAgentCommand(ctx context.Context, proxmoxClient *orchestrator.ProxmoxClient, nodeName string, vmID int, command string, workingDir string) (string, error) {
	endpoint := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec", nodeName, vmID)

	// Prepare command with working directory
	fullCommand := command
	if workingDir != "" {
		fullCommand = fmt.Sprintf("cd %s && %s", workingDir, command)
	}

	// Create request body
	reqBody := map[string]interface{}{
		"command": []string{"sh", "-c", fullCommand},
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API request
	resp, err := proxmoxClient.APIRequestRaw(ctx, "POST", endpoint, reqBodyJSON)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("command execution failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var execResp struct {
		Data struct {
			Pid int `json:"pid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Wait for command to complete and get output
	statusEndpoint := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status", nodeName, vmID)
	statusBody := map[string]interface{}{
		"pid": execResp.Data.Pid,
	}
	statusBodyJSON, _ := json.Marshal(statusBody)

	maxWait := 30 * time.Second
	startTime := time.Now()
	for time.Since(startTime) < maxWait {
		time.Sleep(100 * time.Millisecond)
		statusResp, err := proxmoxClient.APIRequestRaw(ctx, "POST", statusEndpoint, statusBodyJSON)
		if err != nil {
			continue
		}

		var statusData struct {
			Data struct {
				Exited   int    `json:"exited"`
				OutData  string `json:"out-data"`
				ErrData  string `json:"err-data"`
				ExitCode int    `json:"exit-code"`
			} `json:"data"`
		}

		if err := json.NewDecoder(statusResp.Body).Decode(&statusData); err != nil {
			statusResp.Body.Close()
			continue
		}
		statusResp.Body.Close()

		if statusData.Data.Exited == 1 {
			// Command completed
			output := statusData.Data.OutData
			if statusData.Data.ErrData != "" {
				output += statusData.Data.ErrData
			}
			return output, nil
		}
	}

	return "", fmt.Errorf("command execution timeout")
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
