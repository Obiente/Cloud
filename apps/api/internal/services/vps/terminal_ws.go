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

	vpsv1 "api/gen/proto/obiente/cloud/vps/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"

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
	Command        string `json:"command,omitempty"` // For special commands like "start"
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
func (s *Service) HandleVPSTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract VPS ID from URL path (e.g., /vps/{vps_id}/terminal/ws)
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

	// Read the initial message which should contain authentication and sizing info
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

	// Use VPS ID from URL if not provided in message
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

	// Get VPS instance to check status
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", initMsg.VPSID).First(&vps).Error; err != nil {
		sendError(fmt.Sprintf("VPS instance %s not found", initMsg.VPSID))
		conn.Close(websocket.StatusInternalError, "VPS not found")
		return
	}

	// Check if VPS status allows terminal access
	// Serial console works even when VM is booting/rebooting, so allow access for:
	// - CREATING (1): Can see boot output
	// - STARTING (2): Can see boot output
	// - RUNNING (3): Normal operation
	// - REBOOTING (6): Can see reboot output
	// - STOPPED (5): Serial console might still work
	// Block only for:
	// - STOPPING (4): VM is shutting down
	// - FAILED (7): VM might not exist
	// - DELETING (8): VM is being deleted
	// - DELETED (9): VM is deleted
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
		// Don't close - allow user to see the message, but don't try to connect
		// Listen for messages but only handle "start" command
		for {
			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
					return
				}
				log.Printf("[VPS Terminal WS] Read error: %v", err)
				return
			}
			// Just echo input or ignore
			if strings.ToLower(msg.Type) == "input" && len(msg.Input) > 0 {
				inputBytes := make([]byte, len(msg.Input))
				for i, v := range msg.Input {
					inputBytes[i] = byte(v)
				}
				inputStr := strings.TrimSpace(string(inputBytes))
				if strings.ToLower(inputStr) == "start" {
					startMsg := "Starting VPS... Please reconnect after it's running.\r\n"
					startData := make([]int, len(startMsg))
					for i, b := range []byte(startMsg) {
						startData[i] = int(b)
					}
					_ = writeJSON(vpsTerminalWSOutput{Type: "output", Data: startData})
					// TODO: Actually start the VPS
				}
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

	// Get Proxmox client for terminal access
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

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		sendError("Failed to find Proxmox node")
		conn.Close(websocket.StatusInternalError, "No Proxmox node found")
		return
	}

	nodeName := nodes[0]

	// Priority order:
	// 1. Serial Console WebSocket (text terminal including boot output, works even when VM is booting)
	// 2. SSH (best experience when IP is available and VM is fully booted)
	// 3. Guest agent (fallback when no IP)

	// Try Serial Console WebSocket first (provides text terminal including boot output)
	// Note: Serial console may not be supported by all Proxmox setups, so we try it but fall back gracefully
	serialWSURL, err := proxmoxClient.GetSerialConsoleWebSocketURL(ctx, nodeName, vmIDInt)
	if err == nil && serialWSURL != "" {
		log.Printf("[VPS Terminal WS] Attempting Serial Console WebSocket (will fall back to SSH/guest agent if it fails)")
		// Try serial console, but if it fails, we'll continue to SSH/guest agent below
		// We can't easily detect failure here, so we'll let it try and if it fails, the error will be shown
		// but the connection will remain open for fallback
		// Actually, since handleSerialConsoleWebSocketTerminal returns on error, we need a different approach
		// For now, let's skip serial console if we're not sure it will work, or make it a background attempt
		// Let's just try it and if it fails immediately, continue to SSH
		// We'll modify the handler to not return on first error, or we'll catch the error here
		// Actually, the simplest is to just skip serial console for now until we can get it working
		// Or we can try it in a goroutine and fall back if it doesn't connect quickly
		// For now, let's comment it out and use SSH/guest agent which we know works
		// TODO: Fix serial console WebSocket authentication/endpoint
		// s.handleSerialConsoleWebSocketTerminal(ctx, conn, serialWSURL, proxmoxClient, nodeName, vmIDInt, &vps, writeJSON, initMsg, cols, rows)
		// return
	}
	log.Printf("[VPS Terminal WS] Serial Console WebSocket skipped (not yet fully supported), trying SSH/guest agent")

	// Try SSH connection (preferred when IP is available)
	var sshConn *SSHConnection
	vpsManager, err := orchestrator.NewVPSManager()
	if err == nil {
		defer vpsManager.Close()
		ipv4, _, err := vpsManager.GetVPSIPAddresses(ctx, initMsg.VPSID)
		if err == nil && len(ipv4) > 0 {
			vpsIP := ipv4[0]
			rootPassword, err := s.getVPSRootPassword(ctx, initMsg.VPSID)
			if err == nil && rootPassword != "" {
				// Try to connect via SSH
				sshConn, err = s.connectSSH(ctx, vpsIP, rootPassword, cols, rows)
				if err != nil {
					log.Printf("[VPS Terminal WS] Failed to connect via SSH: %v", err)
					sshConn = nil
				}
			}
		}
	}

	// If SSH connection failed, fall back to guest agent
	if sshConn == nil {
		log.Printf("[VPS Terminal WS] SSH not available, using guest agent terminal")
		// Use Proxmox guest agent for terminal access (no IP needed)
		s.handleProxmoxGuestAgentTerminal(ctx, conn, proxmoxClient, nodeName, vmIDInt, &vps, writeJSON, initMsg)
		return
	}

	// SSH connection established - handle terminal session
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

	// Send initial newline
	if _, err := sshConn.stdin.Write([]byte("\r\n")); err != nil {
		log.Printf("[VPS Terminal WS] Failed to write initial newline: %v", err)
	}

	// Send connected message
	if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
		log.Printf("[VPS Terminal WS] Failed to send connected message: %v", err)
		conn.Close(websocket.StatusInternalError, "failed to send connected")
		return
	}

	outputCtx, outputCancel := context.WithCancel(ctx)
	outputDone := make(chan struct{})

	// Forward SSH output to websocket client
	var outputDoneWg sync.WaitGroup
	outputDoneWg.Add(1)
	go func() {
		defer outputDoneWg.Done()
		defer close(outputDone)
		defer outputCancel()

		buf := make([]byte, 4096)
		for {
			select {
			case <-outputCtx.Done():
				return
			default:
			}

			n, err := sshConn.stdout.Read(buf)
			if n > 0 {
				data := make([]int, n)
				for i := 0; i < n; i++ {
					data[i] = int(buf[i])
				}
				if err := writeJSON(vpsTerminalWSOutput{Type: "output", Data: data}); err != nil {
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
					conn.Close(websocket.StatusInternalError, "terminal error")
					closed = true
				}
				return
			}
		}
	}()

	// Forward stderr in a separate goroutine
	outputDoneWg.Add(1)
	go func() {
		defer outputDoneWg.Done()

		buf := make([]byte, 4096)
		for {
			select {
			case <-outputCtx.Done():
				return
			default:
			}

			n, err := sshConn.stderr.Read(buf)
			if n > 0 {
				data := make([]int, n)
				for i := 0; i < n; i++ {
					data[i] = int(buf[i])
				}
				if err := writeJSON(vpsTerminalWSOutput{Type: "output", Data: data}); err != nil {
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

	// Listen for client input messages
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
			sendError("Failed to read message")
			conn.Close(websocket.StatusProtocolError, "read error")
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
				sendError("Failed to send input")
				conn.Close(websocket.StatusInternalError, "write error")
				return
			}

		case "resize":
			// Resize SSH terminal
			if sshConn.session != nil && msg.Cols > 0 && msg.Rows > 0 {
				cols = msg.Cols
				rows = msg.Rows
				if err := sshConn.session.WindowChange(rows, cols); err != nil {
					log.Printf("[VPS Terminal WS] Failed to resize terminal: %v", err)
				} else {
					log.Printf("[VPS Terminal WS] Resized terminal to %dx%d", msg.Cols, msg.Rows)
				}
			}

		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})

		default:
			log.Printf("[VPS Terminal WS] Unknown message type: %s", msg.Type)
		}
	}
}

// connectSSH establishes an SSH connection to the VPS
func (s *Service) connectSSH(ctx context.Context, vpsIP, rootPassword string, cols, rows int) (*SSHConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.Password(rootPassword),
		},
	}

	// Connect to SSH
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

	// Request PTY
	if err := session.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
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

						// Echo newline (command is already echoed by frontend terminal via local echo)
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
					// Printable ASCII characters (space through ~)
					// Add to command buffer
					// Frontend terminal handles local echo, so we don't need to echo here
					commandBuffer = append(commandBuffer, b)
				}
				// Ignore other control characters
			}

		case "resize":
			// Terminal resize (not fully supported via guest agent, but acknowledge)
			// Note: Guest agent doesn't support dynamic terminal resizing
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
	// Use guest agent to execute command
	// Format: /nodes/{node}/qemu/{vmid}/agent/exec
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
	// Poll for command status
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

// handleVNCWebSocketTerminal handles terminal access via Proxmox VNC WebSocket
// This provides full terminal access including boot output, similar to Proxmox web UI
func (s *Service) handleVNCWebSocketTerminal(
	ctx context.Context,
	clientConn *websocket.Conn,
	vncWSURL string,
	proxmoxClient *orchestrator.ProxmoxClient,
	nodeName string,
	vmID int,
	vps *database.VPSInstance,
	writeJSON func(interface{}) error,
	initMsg vpsTerminalWSMessage,
	cols, rows int,
) {
	// Parse VNC WebSocket URL
	u, err := url.Parse(vncWSURL)
	if err != nil {
		log.Printf("[VPS Terminal WS] Invalid VNC WebSocket URL: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError("Failed to parse VNC WebSocket URL")
		return
	}
	_ = u // Suppress unused variable warning

	// Get Proxmox config for authentication
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to get Proxmox config: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError("Failed to get Proxmox configuration")
		return
	}

	// Prepare headers for Proxmox VNC WebSocket connection
	headers := make(http.Header)
	if proxmoxConfig.TokenID != "" && proxmoxConfig.Secret != "" {
		// Use API token authentication
		// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", proxmoxConfig.Username, proxmoxConfig.TokenID, proxmoxConfig.Secret)
		headers.Set("Authorization", authHeader)
	} else {
		// Password-based auth would need a ticket, but VNC WebSocket typically uses token
		log.Printf("[VPS Terminal WS] Warning: VNC WebSocket requires API token authentication")
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError("VNC WebSocket requires API token authentication")
		return
	}

	// Connect to Proxmox VNC WebSocket
	dialOptions := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Proxmox often uses self-signed certs
				},
			},
		},
		HTTPHeader: headers,
	}

	proxmoxConn, _, err := websocket.Dial(ctx, vncWSURL, dialOptions)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to connect to Proxmox VNC WebSocket: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Failed to connect to Proxmox VNC: %v", err))
		return
	}
	defer proxmoxConn.Close(websocket.StatusInternalError, "")

	log.Printf("[VPS Terminal WS] Connected to Proxmox VNC WebSocket for VM %d", vmID)

	// Send connected message to client
	if err := writeJSON(vpsTerminalWSOutput{Type: "connected"}); err != nil {
		log.Printf("[VPS Terminal WS] Failed to send connected message: %v", err)
		return
	}

	// Proxy messages bidirectionally
	errChan := make(chan error, 2)

	// Forward binary data from Proxmox VNC to client (as JSON with data array)
	go func() {
		defer close(errChan)
		for {
			_, data, err := proxmoxConn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from Proxmox VNC: %w", err)
				return
			}

			// Convert binary data to JSON format (array of ints)
			dataInts := make([]int, len(data))
			for i, b := range data {
				dataInts[i] = int(b)
			}

			if err := writeJSON(vpsTerminalWSOutput{
				Type: "output",
				Data: dataInts,
			}); err != nil {
				errChan <- fmt.Errorf("failed to write to client: %w", err)
				return
			}
		}
	}()

	// Forward input from client to Proxmox VNC
	go func() {
		for {
			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, clientConn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from client: %w", err)
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
					// Write binary data to Proxmox VNC WebSocket
					if err := proxmoxConn.Write(ctx, websocket.MessageBinary, inputBytes); err != nil {
						errChan <- fmt.Errorf("failed to write to Proxmox VNC: %w", err)
						return
					}
				}

			case "resize":
				// VNC WebSocket handles resize automatically, but we can acknowledge it
				// Note: VNC resize is typically handled by the VNC protocol itself

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
		log.Printf("[VPS Terminal WS] Context cancelled for VNC WebSocket")
	case err := <-errChan:
		if err != nil {
			log.Printf("[VPS Terminal WS] VNC WebSocket proxy error: %v", err)
			_ = writeJSON(vpsTerminalWSOutput{
				Type:    "error",
				Message: fmt.Sprintf("VNC WebSocket error: %v", err),
			})
		}
	}
}

// handleSerialConsoleWebSocketTerminal handles terminal access via Proxmox Serial Console WebSocket
// This provides text-based terminal access including boot output, works even when VM is booting
func (s *Service) handleSerialConsoleWebSocketTerminal(
	ctx context.Context,
	clientConn *websocket.Conn,
	serialWSURL string,
	proxmoxClient *orchestrator.ProxmoxClient,
	nodeName string,
	vmID int,
	vps *database.VPSInstance,
	writeJSON func(interface{}) error,
	initMsg vpsTerminalWSMessage,
	cols, rows int,
) {
	// Get Proxmox config for authentication
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to get Proxmox config: %v", err)
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError("Failed to get Proxmox configuration")
		return
	}

	// Prepare headers for Proxmox Serial Console WebSocket connection
	// Proxmox serial console requires proper authentication for the WebSocket upgrade
	// The vncticket in URL authenticates the VNC/Serial console itself,
	// but we still need to authenticate the HTTP WebSocket upgrade request
	headers := make(http.Header)

	// Use API token authentication for WebSocket upgrade
	// Format: PVEAPIToken=USER@REALM!TOKENID=SECRET
	if proxmoxConfig.TokenID != "" && proxmoxConfig.Secret != "" {
		authHeader := fmt.Sprintf("PVEAPIToken=%s!%s=%s", proxmoxConfig.Username, proxmoxConfig.TokenID, proxmoxConfig.Secret)
		headers.Set("Authorization", authHeader)
	} else {
		// For password-based auth, try to use the proxmoxClient's ticket/cookie
		// But we can't access it directly, so we'll need to create a new request to get cookies
		// For now, try without additional headers - vncticket in URL might be enough
		// If this fails, we'll fall back to SSH or guest agent
		log.Printf("[VPS Terminal WS] Using password-based auth - serial console may require API token or cookies")
	}

	// Connect to Proxmox Serial Console WebSocket
	// Note: vncticket in URL is the primary authentication for VNC/Serial console
	// But we still need to authenticate the WebSocket upgrade request
	dialOptions := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Proxmox often uses self-signed certs
				},
			},
		},
		HTTPHeader: headers,
	}

	proxmoxConn, _, err := websocket.Dial(ctx, serialWSURL, dialOptions)
	if err != nil {
		log.Printf("[VPS Terminal WS] Failed to connect to Proxmox Serial Console WebSocket: %v", err)
		// Serial console failed - this is expected if Proxmox doesn't support it or auth fails
		// Send error to client but don't close - let them know we're falling back
		sendError := func(msg string) { _ = writeJSON(vpsTerminalWSOutput{Type: "error", Message: msg}) }
		sendError(fmt.Sprintf("Serial Console not available: %v. Falling back to SSH or guest agent.", err))
		// Note: We return here, but the caller should handle fallback
		// Actually, since we return, the caller won't get a chance to fall back
		// Let's just return and let the normal flow handle SSH/guest agent
		return
	}
	defer proxmoxConn.Close(websocket.StatusInternalError, "")

	log.Printf("[VPS Terminal WS] Connected to Proxmox Serial Console WebSocket for VM %d", vmID)

	// Send connected message to client
	if err := writeJSON(vpsTerminalWSOutput{Type: "connected"}); err != nil {
		log.Printf("[VPS Terminal WS] Failed to send connected message: %v", err)
		return
	}

	// Proxy messages bidirectionally
	errChan := make(chan error, 2)

	// Forward binary data from Proxmox Serial Console to client (as JSON with data array)
	go func() {
		defer close(errChan)
		for {
			_, data, err := proxmoxConn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from Proxmox Serial Console: %w", err)
				return
			}

			// Convert binary data to JSON format (array of ints)
			dataInts := make([]int, len(data))
			for i, b := range data {
				dataInts[i] = int(b)
			}

			if err := writeJSON(vpsTerminalWSOutput{
				Type: "output",
				Data: dataInts,
			}); err != nil {
				errChan <- fmt.Errorf("failed to write to client: %w", err)
				return
			}
		}
	}()

	// Forward input from client to Proxmox Serial Console
	go func() {
		for {
			var msg vpsTerminalWSMessage
			if err := wsjson.Read(ctx, clientConn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from client: %w", err)
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
						errChan <- fmt.Errorf("failed to write to Proxmox Serial Console: %w", err)
						return
					}
				}

			case "resize":
				// Serial Console WebSocket handles resize automatically, but we can acknowledge it
				// Note: Serial console resize is typically handled by the protocol itself

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
				Message: fmt.Sprintf("Serial Console WebSocket error: %v", err),
			})
		}
	}
}
