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
		}
		// Don't close connection - allow user to type "start" command
		session = nil
		cleanup = func() {}
		created = false
	}

	var cleanupOnce sync.Once
	cleanupFn := func() {
		cleanupOnce.Do(cleanup)
	}
	defer cleanupFn()

	if created && session != nil {
		log.Printf("[GameServer Terminal WS] New terminal session created, sending initial newline")
		if _, err := session.conn.Write([]byte("\r\n")); err != nil {
			log.Printf("[GameServer Terminal WS] Failed to write initial newline: %v", err)
		}
	}

	// Send connected message (or stopped message)
	if session != nil {
		if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
			log.Printf("[GameServer Terminal WS] Failed to send connected message: %v", err)
			conn.Close(websocket.StatusInternalError, "failed to send connected")
			return
		}
	} else {
		// Container is stopped - send helpful message
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
	if session != nil {
		outputDoneWg.Add(1)
		go func() {
			defer outputDoneWg.Done()
			defer close(outputDone)
			defer outputCancel()

			buf := make([]byte, 4096)
			for {
				n, err := session.conn.Read(buf)
				if n > 0 {
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
			if session == nil {
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

						// Try to initialize terminal session again
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
						session = newSession
						cleanup = newCleanup

						// Start output forwarding
						outputDone = make(chan struct{})
						outputCancel()
						outputCtx, outputCancel = context.WithCancel(ctx)
						outputDoneWg.Add(1)
						go func() {
							defer outputDoneWg.Done()
							defer close(outputDone)
							defer outputCancel()

							buf := make([]byte, 4096)
							for {
								n, err := session.conn.Read(buf)
								if n > 0 {
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

						// Send initial newline
						if newCreated && session != nil {
							if _, err := session.conn.Write([]byte("\r\n")); err != nil {
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
			inputBytes := make([]byte, len(msg.Input))
			for i, v := range msg.Input {
				inputBytes[i] = byte(v)
			}
			if _, err := session.conn.Write(inputBytes); err != nil {
				log.Printf("[GameServer Terminal WS] Failed to write input: %v", err)
				sendError("Failed to send input")
				conn.Close(websocket.StatusInternalError, "write error")
				return
			}
		case "resize":
			// TODO: implement container resize support if needed
			log.Printf("[GameServer Terminal WS] Resize event received: %dx%d (not implemented)", msg.Cols, msg.Rows)
		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})
		default:
			sendError("Unknown message type")
		}
	}
}
