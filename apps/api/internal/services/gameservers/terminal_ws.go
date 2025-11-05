package gameservers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"api/docker"
	"api/internal/auth"

	"connectrpc.com/connect"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type gameServerTerminalWSMessage struct {
	Type           string `json:"type"`
	GameServerID   string `json:"gameServerId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	Token          string `json:"token,omitempty"`
	Input          []int  `json:"input,omitempty"`
	Cols           int    `json:"cols,omitempty"`
	Rows           int    `json:"rows,omitempty"`
	Command        string `json:"command,omitempty"` // For special commands like "start"
}

type gameServerTerminalWSOutput struct {
	Type    string `json:"type"`
	Data    []int  `json:"data,omitempty"`
	Exit    bool   `json:"exit,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// HandleTerminalWebSocket upgrades the HTTP connection to a WebSocket and proxies
// terminal input/output directly to the Go API for game servers.
func (s *Service) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Origin checking can be added here if needed; for now, rely on auth token validation.
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("[GameServer Terminal WS] Failed to accept websocket connection: %v", err)
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
		_ = writeJSON(gameServerTerminalWSOutput{
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
	var initMsg gameServerTerminalWSMessage
	if err := wsjson.Read(ctx, conn, &initMsg); err != nil {
		log.Printf("[GameServer Terminal WS] Failed to read init message: %v", err)
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

	if initMsg.GameServerID == "" || initMsg.OrganizationID == "" {
		sendError("gameServerId and organizationId are required")
		conn.Close(websocket.StatusPolicyViolation, "missing identifiers")
		return
	}

	ctx, _, err = auth.AuthenticateAndSetContext(ctx, "Bearer "+strings.TrimSpace(initMsg.Token))
	if err != nil {
		log.Printf("[GameServer Terminal WS] Authentication failed: %v", err)
		sendError("Authentication required")
		conn.Close(websocket.StatusPolicyViolation, "auth failed")
		return
	}

	// Verify permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, initMsg.OrganizationID, auth.ScopedPermission{Permission: "gameservers.view", ResourceType: "gameserver", ResourceID: initMsg.GameServerID}); err != nil {
		sendError("Permission denied")
		conn.Close(websocket.StatusPolicyViolation, "permission denied")
		return
	}

	// Get game server to find container ID
	gameServer, err := s.repo.GetByID(ctx, initMsg.GameServerID)
	if err != nil {
		sendError("Game server not found")
		conn.Close(websocket.StatusInternalError, "game server not found")
		return
	}

	var currentContainerID string
	if gameServer.ContainerID != nil {
		currentContainerID = *gameServer.ContainerID
	}

	// Try to initialize terminal session
	session, cleanup, created, err := s.ensureTerminalSession(ctx, initMsg.GameServerID, initMsg.OrganizationID, initMsg.Cols, initMsg.Rows)

	// If container is stopped, allow user to type "start" command
	if err != nil {
		connectErr, ok := err.(*connect.Error)
		if ok && connectErr.Code() == connect.CodeFailedPrecondition {
			// Container is stopped - this is expected
			log.Printf("[GameServer Terminal WS] Container is stopped for game server %s", initMsg.GameServerID)
		} else {
			log.Printf("[GameServer Terminal WS] Failed to create terminal session: %v", err)
		}
		// Don't close connection - allow user to type "start" command
		session = nil
		cleanup = func() {}
		created = false
	} else {
		log.Printf("[GameServer Terminal WS] Terminal session created successfully for game server %s (attached to main process)", initMsg.GameServerID)
	}

	var cleanupOnce sync.Once
	cleanupFn := func() {
		cleanupOnce.Do(cleanup)
	}
	defer cleanupFn()
	
	// Use a mutex to protect session access from concurrent goroutines
	var sessionMu sync.RWMutex
	getSession := func() *TerminalSession {
		sessionMu.RLock()
		defer sessionMu.RUnlock()
		return session
	}
	setSession := func(newSession *TerminalSession, newCleanup func()) {
		sessionMu.Lock()
		defer sessionMu.Unlock()
		session = newSession
		cleanup = newCleanup
	}

	currentSession := getSession()
	if created && currentSession != nil {
		log.Printf("[GameServer Terminal WS] New terminal session created, sending initial newline")
		if _, err := currentSession.conn.Write([]byte("\r\n")); err != nil {
			log.Printf("[GameServer Terminal WS] Failed to write initial newline: %v", err)
		}
	}

	// Always send connected message - WebSocket connection is established
	// This allows the frontend to enable command input even when container is stopped
	if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
		log.Printf("[GameServer Terminal WS] Failed to send connected message: %v", err)
		conn.Close(websocket.StatusInternalError, "failed to send connected")
		return
	}

	// If container is stopped, send helpful message
	currentSession = getSession()
	if currentSession == nil {
		stoppedMsg := "Game server is stopped. Type 'start' to start the server first.\r\n"
		data := make([]int, len(stoppedMsg))
		for i, b := range []byte(stoppedMsg) {
			data[i] = int(b)
		}
		_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data})
	}

	outputCtx, outputCancel := context.WithCancel(ctx)
	outputDone := make(chan struct{})

	// Buffer for accumulating command input when container is stopped
	var commandBuffer strings.Builder

	// Forward container output to websocket client (only if session exists)
	var outputDoneWg sync.WaitGroup
	currentSession = getSession()
	if currentSession != nil {
		outputDoneWg.Add(1)
		go func() {
			defer outputDoneWg.Done()
			defer close(outputDone)
			defer outputCancel()

			buf := make([]byte, 4096)
			for {
				// Set a read deadline to avoid blocking indefinitely
				// This allows us to detect if the connection is alive
				currentSession := getSession()
				if currentSession == nil {
					return
				}
				if conn, ok := currentSession.conn.(interface{ SetReadDeadline(time.Time) error }); ok {
					conn.SetReadDeadline(time.Now().Add(30 * time.Second))
				}
				
				n, err := currentSession.conn.Read(buf)
				
				// Clear the deadline
				if conn, ok := currentSession.conn.(interface{ SetReadDeadline(time.Time) error }); ok {
					conn.SetReadDeadline(time.Time{})
				}
				
				if n > 0 {
					// Log first 100 bytes of output for debugging
					previewLen := n
					if previewLen > 100 {
						previewLen = 100
					}
					log.Printf("[GameServer Terminal WS] Read %d bytes from game server attach stream: %q", n, string(buf[:previewLen]))
					data := make([]int, n)
					for i := 0; i < n; i++ {
						data[i] = int(buf[i])
					}
					if err := writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data}); err != nil {
						log.Printf("[GameServer Terminal WS] Failed to forward output: %v", err)
						return
					}
				}

				if err != nil {
					if err == io.EOF {
						log.Printf("[GameServer Terminal WS] Container output stream ended (EOF)")
						_ = writeJSON(gameServerTerminalWSOutput{Type: "closed", Reason: "Terminal session ended", Exit: true})
						conn.Close(websocket.StatusNormalClosure, "terminal closed")
						closed = true
					} else {
						log.Printf("[GameServer Terminal WS] Container read error: %v", err)
						_ = writeJSON(gameServerTerminalWSOutput{Type: "error", Message: "Terminal stream error"})
						conn.Close(websocket.StatusInternalError, "terminal error")
						closed = true
					}
					return
				}
			}
		}()
		
		// Also start a log reader as fallback (some game servers don't output to attach stream)
		// Docker logs API reads from the logging driver and may be more reliable than attach
		outputDoneWg.Add(1)
		go func() {
			defer outputDoneWg.Done()
			
			dcli, err := docker.New()
			if err != nil {
				log.Printf("[GameServer Terminal WS] Failed to create docker client for logs: %v", err)
				return
			}
			defer dcli.Close()
			
			// Get the current session to access containerID
			currentSession := getSession()
			if currentSession == nil {
				log.Printf("[GameServer Terminal WS] No session available for logs reader")
				return
			}
			containerID := currentSession.containerID
			
			// Read logs with follow=true to stream new output
			// Use tail=0 to get all logs, or a small number to get recent logs
			logsReader, err := dcli.ContainerLogs(ctx, containerID, "0", true)
			if err != nil {
				log.Printf("[GameServer Terminal WS] Failed to start log stream: %v", err)
				return
			}
			defer logsReader.Close()
			
			// Docker logs API returns frames with 8-byte headers for non-TTY containers
			// Read frame by frame and strip headers
			header := make([]byte, 8)
			frameBuf := make([]byte, 32*1024)
			
			for {
				select {
				case <-outputCtx.Done():
					return
				default:
				}
				
				// Read header
				if _, err := io.ReadFull(logsReader, header); err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						return
					}
					log.Printf("[GameServer Terminal WS] Log stream header read error: %v", err)
					return
				}
				
				// Parse payload length (bytes 4-7, big-endian)
				payloadLength := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))
				if payloadLength == 0 {
					continue // Empty frame, read next
				}
				
				// Read payload
				if payloadLength > len(frameBuf) {
					frameBuf = make([]byte, payloadLength)
				}
				
				n, err := io.ReadFull(logsReader, frameBuf[:payloadLength])
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						// Partial frame, send what we have
						if n > 0 {
							data := make([]int, n)
							for i := 0; i < n; i++ {
								data[i] = int(frameBuf[i])
							}
							log.Printf("[GameServer Terminal WS] Forwarding %d bytes from logs stream (partial)", n)
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data})
						}
						return
					}
					log.Printf("[GameServer Terminal WS] Log stream payload read error: %v", err)
					return
				}
				
				if n > 0 {
					data := make([]int, n)
					for i := 0; i < n; i++ {
						data[i] = int(frameBuf[i])
					}
					log.Printf("[GameServer Terminal WS] Forwarding %d bytes from logs stream", n)
					_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data})
				}
			}
		}()
	} else {
		// No session - just close the channel
		close(outputDone)
	}

	// Listen for client input messages
	for {
		select {
		case <-outputCtx.Done():
			cleanupFn()
			return
		default:
		}

		var msg gameServerTerminalWSMessage
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			log.Printf("[GameServer Terminal WS] Read error: %v", err)
			sendError("Failed to read message")
			conn.Close(websocket.StatusProtocolError, "read error")
			return
		}

		switch strings.ToLower(msg.Type) {
		case "input":
			if len(msg.Input) == 0 {
				continue
			}

			// Check for "start" command when container is stopped
			currentSession := getSession()
			if currentSession == nil {
				inputBytes := make([]byte, len(msg.Input))
				for i, v := range msg.Input {
					inputBytes[i] = byte(v)
				}
				inputStr := string(inputBytes)

				// Accumulate characters in buffer until we get Enter/newline
				currentBuffer := commandBuffer.String()
				commandBuffer.WriteString(inputStr)

				// Get the new buffer content
				newBuffer := commandBuffer.String()

				// Only echo characters that match typing "start" letter by letter (case-insensitive)
				bufferForCheck := strings.ReplaceAll(strings.ReplaceAll(newBuffer, "\r", ""), "\n", "")
				lowerBuffer := strings.ToLower(bufferForCheck)
				expectedStart := "start"

				// Check if the buffer still matches "start" prefix
				if len(lowerBuffer) <= len(expectedStart) && expectedStart[:len(lowerBuffer)] == lowerBuffer {
					// Still matching "start" - echo only the new characters
					if len(newBuffer) > len(currentBuffer) {
						newChars := newBuffer[len(currentBuffer):]
						displayChars := strings.ReplaceAll(strings.ReplaceAll(newChars, "\r", ""), "\n", "")
						if len(displayChars) > 0 {
							displayBytes := []byte(displayChars)
							displayData := make([]int, len(displayBytes))
							for i, b := range displayBytes {
								displayData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: displayData})
						}
					}
				}

				// Check if we have a complete command (ends with \r or \n)
				if strings.Contains(newBuffer, "\r") || strings.Contains(newBuffer, "\n") {
					trimmed := strings.TrimSpace(strings.ToLower(bufferForCheck))
					commandBuffer.Reset()

					// Check if user typed "start" command
					if trimmed == "start" {
						// Check permissions for starting game server
						if err := s.permissionChecker.CheckScopedPermission(ctx, initMsg.OrganizationID, auth.ScopedPermission{Permission: "gameservers.manage", ResourceType: "gameserver", ResourceID: initMsg.GameServerID}); err != nil {
							errMsg := "Permission denied: you need 'gameservers.manage' permission to start game servers.\r\n"
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						if currentContainerID == "" {
							errMsg := "Error: Container ID not found. Please reconnect.\r\n"
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						// Start the container
						manager, err := s.getGameServerManager()
						if err != nil {
							errMsg := fmt.Sprintf("Error: %v\r\n", err)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						statusMsg := fmt.Sprintf("Starting game server %s...\r\n", initMsg.GameServerID)
						statusData := make([]int, len(statusMsg))
						for i, b := range []byte(statusMsg) {
							statusData[i] = int(b)
						}
						_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: statusData})

						if err := manager.StartGameServer(ctx, initMsg.GameServerID); err != nil {
							errMsg := fmt.Sprintf("Error starting game server: %v\r\n", err)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						// Wait a moment for container to be ready
						time.Sleep(500 * time.Millisecond)

						// Close old session before creating new one (to attach to new process)
						// The old session was attached to PID 1 (wrapper script), we need a fresh attach
						oldSession := getSession()
						if oldSession != nil {
							log.Printf("[GameServer Terminal WS] Closing old terminal session before creating new one")
							oldCleanup := cleanup
							if oldCleanup != nil {
								oldCleanup()
							}
							// Also close the old session's connection explicitly
							if oldSession.conn != nil {
								oldSession.conn.Close()
							}
						}
						
						// Force close any existing session in the cache (ensures we get a fresh attach)
						log.Printf("[GameServer Terminal WS] Removing old session from cache to force new attach")
						s.CloseTerminalSession(initMsg.GameServerID)

						// Try to initialize terminal session again (will create a new one)
						var newSession *TerminalSession
						var newCleanup func()
						var newCreated bool
						var newErr error
						newSession, newCleanup, newCreated, newErr = s.ensureTerminalSession(ctx, initMsg.GameServerID, initMsg.OrganizationID, initMsg.Cols, initMsg.Rows)

						if newErr != nil {
							errMsg := fmt.Sprintf("Error connecting to container: %v\r\n", newErr)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						// Success! Update session
						setSession(newSession, newCleanup)
						
						log.Printf("[GameServer Terminal WS] Session updated after server start: containerID=%s, conn=%v", newSession.containerID, newSession.conn != nil)
						
						// Verify the connection is ready
						if newSession.conn == nil {
							log.Printf("[GameServer Terminal WS] ERROR: New session has nil connection!")
							errMsg := "Error: Terminal session connection is invalid.\r\n"
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
							continue
						}

						// Start output forwarding
						outputDone = make(chan struct{})
						outputCancel()
						outputCtx, outputCancel = context.WithCancel(ctx)
						
						// Start attach stream reader
						outputDoneWg.Add(1)
						go func() {
							defer outputDoneWg.Done()
							defer close(outputDone)
							defer outputCancel()

							log.Printf("[GameServer Terminal WS] Attach stream reader goroutine started")
							// Capture the session reference at the start
							sessionConn := newSession.conn
							if sessionConn == nil {
								log.Printf("[GameServer Terminal WS] ERROR: Session connection is nil!")
								return
							}
							
							buf := make([]byte, 4096)
							for {
								// Check if context is cancelled
								select {
								case <-outputCtx.Done():
									log.Printf("[GameServer Terminal WS] Attach stream reader: context cancelled")
									return
								default:
								}
								
								// Read from the captured session connection
								n, err := sessionConn.Read(buf)
								if n > 0 {
									log.Printf("[GameServer Terminal WS] Read %d bytes from attach stream", n)
									data := make([]int, n)
									for i := 0; i < n; i++ {
										data[i] = int(buf[i])
									}
									if err := writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data}); err != nil {
										log.Printf("[GameServer Terminal WS] Failed to forward output: %v", err)
										return
									}
								}

								if err != nil {
									if err == io.EOF {
										log.Printf("[GameServer Terminal WS] Attach stream reader: EOF")
										_ = writeJSON(gameServerTerminalWSOutput{Type: "closed", Reason: "Terminal session ended", Exit: true})
										conn.Close(websocket.StatusNormalClosure, "terminal closed")
										closed = true
									} else {
										log.Printf("[GameServer Terminal WS] Container read error: %v", err)
										_ = writeJSON(gameServerTerminalWSOutput{Type: "error", Message: "Terminal stream error"})
										conn.Close(websocket.StatusInternalError, "terminal error")
										closed = true
									}
									return
								}
							}
						}()
						
						// Also restart logs reader goroutine with new container
						outputDoneWg.Add(1)
						go func() {
							defer outputDoneWg.Done()
							
							dcli, err := docker.New()
							if err != nil {
								log.Printf("[GameServer Terminal WS] Failed to create docker client for logs: %v", err)
								return
							}
							defer dcli.Close()
							
							// Get the current session to access containerID
							currentSession := getSession()
							if currentSession == nil {
								log.Printf("[GameServer Terminal WS] No session available for logs reader")
								return
							}
							containerID := currentSession.containerID
							
							// Read logs with follow=true to stream new output
							logsReader, err := dcli.ContainerLogs(ctx, containerID, "0", true)
							if err != nil {
								log.Printf("[GameServer Terminal WS] Failed to start log stream: %v", err)
								return
							}
							defer logsReader.Close()
							
							// Docker logs API returns frames with 8-byte headers for non-TTY containers
							header := make([]byte, 8)
							frameBuf := make([]byte, 32*1024)
							
							for {
								select {
								case <-outputCtx.Done():
									return
								default:
								}
								
								// Read header
								if _, err := io.ReadFull(logsReader, header); err != nil {
									if err == io.EOF || err == io.ErrUnexpectedEOF {
										return
									}
									log.Printf("[GameServer Terminal WS] Log stream header read error: %v", err)
									return
								}
								
								// Parse payload length (bytes 4-7, big-endian)
								payloadLength := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))
								if payloadLength == 0 {
									continue // Empty frame, read next
								}
								
								// Read payload
								if payloadLength > len(frameBuf) {
									frameBuf = make([]byte, payloadLength)
								}
								
								n, err := io.ReadFull(logsReader, frameBuf[:payloadLength])
								if err != nil {
									if err == io.EOF || err == io.ErrUnexpectedEOF {
										// Partial frame, send what we have
										if n > 0 {
											data := make([]int, n)
											for i := 0; i < n; i++ {
												data[i] = int(frameBuf[i])
											}
											log.Printf("[GameServer Terminal WS] Forwarding %d bytes from logs stream (partial)", n)
											_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data})
										}
										return
									}
									log.Printf("[GameServer Terminal WS] Log stream payload read error: %v", err)
									return
								}
								
								if n > 0 {
									data := make([]int, n)
									for i := 0; i < n; i++ {
										data[i] = int(frameBuf[i])
									}
									log.Printf("[GameServer Terminal WS] Forwarding %d bytes from logs stream", n)
									_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: data})
								}
							}
						}()

						// Send initial newline
						currentSession := getSession()
						if newCreated && currentSession != nil {
							if _, err := currentSession.conn.Write([]byte("\r\n")); err != nil {
								log.Printf("[GameServer Terminal WS] Failed to write initial newline: %v", err)
							}
						}

						successMsg := "Game server started successfully! Terminal connected.\r\n"
						successData := make([]int, len(successMsg))
						for i, b := range []byte(successMsg) {
							successData[i] = int(b)
						}
						_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: successData})
					} else {
						// Not "start" command - show error message
						errMsg := "Unknown command. Type 'start' to start the game server.\r\n"
						errData := make([]int, len(errMsg))
						for i, b := range []byte(errMsg) {
							errData[i] = int(b)
						}
						_ = writeJSON(gameServerTerminalWSOutput{Type: "output", Data: errData})
					}
				}
				// Continue to allow accumulating more characters
				continue
			}

			// Normal input handling when session exists
			// Extract the command string from input bytes
			inputBytes := make([]byte, len(msg.Input))
			for i, v := range msg.Input {
				inputBytes[i] = byte(v)
			}
			commandStr := strings.TrimSpace(string(inputBytes))
			
			log.Printf("[GameServer Terminal WS] Sending command to game server: %q", commandStr)
			
			// Get fresh session reference - session might have been updated after server start
			currentSession = getSession()
			if currentSession == nil {
				log.Printf("[GameServer Terminal WS] No active session, cannot send command")
				sendError("No active terminal session. Please wait for server to start.")
				continue
			}
			
			log.Printf("[GameServer Terminal WS] Session found: containerID=%s, conn=%v", currentSession.containerID, currentSession.conn != nil)
			
			// Verify connection is valid before writing
			if currentSession.conn == nil {
				log.Printf("[GameServer Terminal WS] ERROR: Session has nil connection!")
				sendError("Terminal session connection is invalid. Please reconnect.")
				continue
			}
			
			// Try sending via stdin first (for real-time terminal interaction)
			stdinWritten := false
			if n, err := currentSession.conn.Write(inputBytes); err == nil {
				log.Printf("[GameServer Terminal WS] Successfully wrote %d bytes to game server stdin", n)
				stdinWritten = true
				
				// Try to flush if supported (important for TTY mode to ensure data is sent immediately)
				if flusher, ok := currentSession.conn.(interface{ Flush() error }); ok {
					if flushErr := flusher.Flush(); flushErr != nil {
						log.Printf("[GameServer Terminal WS] Warning: Failed to flush stdin: %v", flushErr)
					} else {
						log.Printf("[GameServer Terminal WS] Flushed stdin successfully")
					}
				}
			} else {
				log.Printf("[GameServer Terminal WS] Failed to write to stdin: %v", err)
			}
			
			// Also try via GameServerManager which uses multiple fallback methods
			// (RCON, named pipes, wrapper scripts, etc.) - many game servers don't read from stdin
			if manager, err := s.getGameServerManager(); err == nil {
				if err := manager.SendGameServerCommand(ctx, initMsg.GameServerID, commandStr); err != nil {
					log.Printf("[GameServer Terminal WS] Failed to send command via GameServerManager: %v", err)
					if !stdinWritten {
						// Both methods failed
						sendError(fmt.Sprintf("Failed to send command: %v", err))
					}
				} else {
					log.Printf("[GameServer Terminal WS] Successfully sent command via GameServerManager fallback methods")
				}
			} else {
				log.Printf("[GameServer Terminal WS] Failed to get GameServerManager: %v", err)
				if !stdinWritten {
					sendError("Failed to send command")
				}
			}
		case "resize":
			// Resize container TTY (only works if container has TTY enabled)
			currentSession := getSession()
			if currentSession != nil && msg.Cols > 0 && msg.Rows > 0 {
				dcli, err := docker.New()
				if err == nil {
					defer dcli.Close()
					if err := dcli.ContainerResize(ctx, currentSession.containerID, msg.Rows, msg.Cols); err != nil {
						log.Printf("[GameServer Terminal WS] Failed to resize container TTY: %v", err)
					} else {
						log.Printf("[GameServer Terminal WS] Resized container TTY to %dx%d", msg.Cols, msg.Rows)
					}
				}
			} else {
				log.Printf("[GameServer Terminal WS] Resize event received but no active session or invalid dimensions: %dx%d", msg.Cols, msg.Rows)
			}
		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})
		default:
			sendError("Unknown message type")
		}
	}
}
