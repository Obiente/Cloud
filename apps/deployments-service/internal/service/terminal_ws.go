package deployments

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

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"

	"connectrpc.com/connect"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type terminalWSMessage struct {
	Type           string `json:"type"`
	DeploymentID   string `json:"deploymentId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	Token          string `json:"token,omitempty"`
	ContainerID    string `json:"containerId,omitempty"`
	ServiceName    string `json:"serviceName,omitempty"`
	Input          []int  `json:"input,omitempty"`
	Cols           int    `json:"cols,omitempty"`
	Rows           int    `json:"rows,omitempty"`
	Command        string `json:"command,omitempty"` // For special commands like "start"
}

type terminalWSOutput struct {
	Type    string `json:"type"`
	Data    []int  `json:"data,omitempty"`
	Exit    bool   `json:"exit,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// HandleTerminalWebSocket upgrades the HTTP connection to a WebSocket and proxies
// terminal input/output directly to the Go API, bypassing the Nuxt proxy.
func (s *Service) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate origin using CORS configuration
	origin := r.Header.Get("Origin")
	if !middleware.IsOriginAllowed(origin) {
		log.Printf("[Terminal WS] Origin %s not allowed", origin)
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Prepare origin patterns for WebSocket library
	// Check if wildcard CORS is configured - if so, allow all origins
	acceptOptions := &websocket.AcceptOptions{}
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
		log.Printf("[Terminal WS] Failed to accept websocket connection: %v", err)
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
		_ = writeJSON(terminalWSOutput{
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
	var initMsg terminalWSMessage
	if err := wsjson.Read(ctx, conn, &initMsg); err != nil {
		log.Printf("[Terminal WS] Failed to read init message: %v", err)
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
	if initMsg.DeploymentID == "" || initMsg.OrganizationID == "" {
		sendError("deploymentId and organizationId are required")
		conn.Close(websocket.StatusPolicyViolation, "missing identifiers")
		return
	}

	ctx, _, err = auth.AuthenticateAndSetContext(ctx, "Bearer "+strings.TrimSpace(initMsg.Token))
	if err != nil {
		log.Printf("[Terminal WS] Authentication failed: %v", err)
		sendError("Authentication required")
		conn.Close(websocket.StatusPolicyViolation, "auth failed")
		return
	}

	// Verify permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, initMsg.OrganizationID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: initMsg.DeploymentID}); err != nil {
		sendError("Permission denied")
		conn.Close(websocket.StatusPolicyViolation, "permission denied")
		return
	}

	// Track container info for "start" command
	var currentContainerID string

	// Check if we need to forward to another node before creating session
	dcli, dcliErr := docker.New()
	var targetNodeID string
	var shouldForwardWS bool
	if dcliErr == nil {
		loc, findErr := s.findContainerForDeployment(ctx, initMsg.DeploymentID, initMsg.ContainerID, initMsg.ServiceName, dcli)
		if findErr == nil {
			if shouldForward, nodeID := s.shouldForwardToNode(loc); shouldForward {
				shouldForwardWS = true
				targetNodeID = nodeID
				currentContainerID = loc.ContainerID
			}
		}
		dcli.Close()
	}

	// If we need to forward, proxy the WebSocket connection to the target node
	if shouldForwardWS {
		s.forwardTerminalWebSocket(ctx, w, r, targetNodeID, initMsg)
		return
	}

	// Try to initialize terminal session
	session, cleanup, created, err := s.ensureTerminalSession(ctx, initMsg.DeploymentID, initMsg.OrganizationID, initMsg.Cols, initMsg.Rows, initMsg.ContainerID, initMsg.ServiceName)

	// If container is stopped, we need to find it first to get its ID
	if err != nil {
		connectErr, ok := err.(*connect.Error)
		if ok && connectErr.Code() == connect.CodeFailedPrecondition {
			// Container is stopped - find it to get the container ID for "start" command
			dcli, dcliErr := docker.New()
			if dcliErr == nil {
				loc, findErr := s.findContainerForDeployment(ctx, initMsg.DeploymentID, initMsg.ContainerID, initMsg.ServiceName, dcli)
				if findErr == nil {
					currentContainerID = loc.ContainerID
				}
				dcli.Close()
			}
		}
		// Don't close connection - allow user to type "start" command
		// We'll set session to nil to indicate no active session
		session = nil
		cleanup = func() {}
		created = false
	} else {
		// Session created successfully - store container info
		if session != nil {
			currentContainerID = session.containerID
		}
	}

	var cleanupOnce sync.Once
	cleanupFn := func() {
		cleanupOnce.Do(cleanup)
	}
	defer cleanupFn()

	if created && session != nil {
		log.Printf("[Terminal WS] New terminal session created, sending initial newline")
		if _, err := session.conn.Write([]byte("\r\n")); err != nil {
			log.Printf("[Terminal WS] Failed to write initial newline: %v", err)
		}
	}

	// Send connected message (or stopped message)
	if session != nil {
		if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
			log.Printf("[Terminal WS] Failed to send connected message: %v", err)
			conn.Close(websocket.StatusInternalError, "failed to send connected")
			return
		}
	} else {
		// Container is stopped - send helpful message
		stoppedMsg := "Container is stopped. Type 'start' to start the container first.\r\n"
		data := make([]int, len(stoppedMsg))
		for i, b := range []byte(stoppedMsg) {
			data[i] = int(b)
		}
		_ = writeJSON(terminalWSOutput{Type: "output", Data: data})
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
					if err := writeJSON(terminalWSOutput{Type: "output", Data: data}); err != nil {
						log.Printf("[Terminal WS] Failed to forward output: %v", err)
						return
					}
				}

				if err != nil {
					if err == io.EOF {
						_ = writeJSON(terminalWSOutput{Type: "closed", Reason: "Terminal session ended", Exit: true})
						conn.Close(websocket.StatusNormalClosure, "terminal closed")
						closed = true
					} else {
						log.Printf("[Terminal WS] Container read error: %v", err)
						_ = writeJSON(terminalWSOutput{Type: "error", Message: "Terminal stream error"})
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

		var msg terminalWSMessage
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			log.Printf("[Terminal WS] Read error: %v", err)
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
				// Remove newlines/carriage returns for comparison
				bufferForCheck := strings.ReplaceAll(strings.ReplaceAll(newBuffer, "\r", ""), "\n", "")
				lowerBuffer := strings.ToLower(bufferForCheck)
				expectedStart := "start"

				// Check if the buffer still matches "start" prefix (character by character)
				if len(lowerBuffer) <= len(expectedStart) && expectedStart[:len(lowerBuffer)] == lowerBuffer {
					// Still matching "start" - echo only the new characters
					// Find what was newly added (after currentBuffer)
					if len(newBuffer) > len(currentBuffer) {
						newChars := newBuffer[len(currentBuffer):]
						// Remove control characters for display
						displayChars := strings.ReplaceAll(strings.ReplaceAll(newChars, "\r", ""), "\n", "")
						if len(displayChars) > 0 {
							displayBytes := []byte(displayChars)
							displayData := make([]int, len(displayBytes))
							for i, b := range displayBytes {
								displayData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: displayData})
						}
					}
				}
				// If it doesn't match "start", don't echo anything

				// Check if we have a complete command (ends with \r or \n)
				if strings.Contains(newBuffer, "\r") || strings.Contains(newBuffer, "\n") {
					// We have a complete command
					trimmed := strings.TrimSpace(strings.ToLower(bufferForCheck))
					// Reset buffer
					commandBuffer.Reset()

					// Check if user typed "start" command (case-insensitive, allow with newline)
					if trimmed == "start" {
						// Check permissions for starting container
						if err := s.permissionChecker.CheckScopedPermission(ctx, initMsg.OrganizationID, auth.ScopedPermission{Permission: "deployments.manage", ResourceType: "deployment", ResourceID: initMsg.DeploymentID}); err != nil {
							errMsg := "Permission denied: you need 'deployments.manage' permission to start containers.\r\n"
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
							continue
						}

						if currentContainerID == "" {
							errMsg := "Error: Container ID not found. Please reconnect.\r\n"
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
							continue
						}

						// Start the container
						dcli, err := docker.New()
						if err != nil {
							errMsg := fmt.Sprintf("Error starting container: %v\r\n", err)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
							dcli.Close()
							continue
						}

						statusMsg := fmt.Sprintf("Starting container %s...\r\n", currentContainerID[:12])
						statusData := make([]int, len(statusMsg))
						for i, b := range []byte(statusMsg) {
							statusData[i] = int(b)
						}
						_ = writeJSON(terminalWSOutput{Type: "output", Data: statusData})

						if err := dcli.StartContainer(ctx, currentContainerID); err != nil {
							errMsg := fmt.Sprintf("Error starting container: %v\r\n", err)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
							dcli.Close()
							continue
						}

						dcli.Close()

						// Wait a moment for container to be ready
						time.Sleep(500 * time.Millisecond)

						// Try to initialize terminal session again
						var newSession *TerminalSession
						var newCleanup func()
						var newCreated bool
						var newErr error
						newSession, newCleanup, newCreated, newErr = s.ensureTerminalSession(ctx, initMsg.DeploymentID, initMsg.OrganizationID, initMsg.Cols, initMsg.Rows, initMsg.ContainerID, initMsg.ServiceName)

						if newErr != nil {
							errMsg := fmt.Sprintf("Error connecting to container: %v\r\n", newErr)
							errData := make([]int, len(errMsg))
							for i, b := range []byte(errMsg) {
								errData[i] = int(b)
							}
							_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
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
									if err := writeJSON(terminalWSOutput{Type: "output", Data: data}); err != nil {
										log.Printf("[Terminal WS] Failed to forward output: %v", err)
										return
									}
								}

								if err != nil {
									if err == io.EOF {
										_ = writeJSON(terminalWSOutput{Type: "closed", Reason: "Terminal session ended", Exit: true})
										conn.Close(websocket.StatusNormalClosure, "terminal closed")
										closed = true
									} else {
										log.Printf("[Terminal WS] Container read error: %v", err)
										_ = writeJSON(terminalWSOutput{Type: "error", Message: "Terminal stream error"})
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
								log.Printf("[Terminal WS] Failed to write initial newline: %v", err)
							}
						}

						successMsg := "Container started successfully! Terminal connected.\r\n"
						successData := make([]int, len(successMsg))
						for i, b := range []byte(successMsg) {
							successData[i] = int(b)
						}
						_ = writeJSON(terminalWSOutput{Type: "output", Data: successData})
					} else {
						// Not "start" command - show error message
						errMsg := "Unknown command. Type 'start' to start the container.\r\n"
						errData := make([]int, len(errMsg))
						for i, b := range []byte(errMsg) {
							errData[i] = int(b)
						}
						_ = writeJSON(terminalWSOutput{Type: "output", Data: errData})
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
				log.Printf("[Terminal WS] Failed to write input: %v", err)
				sendError("Failed to send input")
				conn.Close(websocket.StatusInternalError, "write error")
				return
			}
		case "resize":
			// Resize container TTY (only works if container has TTY enabled)
			if session != nil && msg.Cols > 0 && msg.Rows > 0 {
				dcli, err := docker.New()
				if err == nil {
					defer dcli.Close()
					if err := dcli.ContainerResize(ctx, session.containerID, msg.Rows, msg.Cols); err != nil {
						log.Printf("[Terminal WS] Failed to resize container TTY: %v", err)
					} else {
						log.Printf("[Terminal WS] Resized container TTY to %dx%d", msg.Cols, msg.Rows)
					}
				}
			} else {
				log.Printf("[Terminal WS] Resize event received but no active session or invalid dimensions: %dx%d", msg.Cols, msg.Rows)
			}
		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})
		default:
			sendError("Unknown message type")
		}
	}
}

// forwardTerminalWebSocket forwards a terminal WebSocket connection to another node
func (s *Service) forwardTerminalWebSocket(ctx context.Context, w http.ResponseWriter, r *http.Request, targetNodeID string, initMsg terminalWSMessage) {
	if s.forwarder == nil {
		http.Error(w, "Node forwarder not available", http.StatusInternalServerError)
		return
	}

	// Validate origin using CORS configuration (origin should already be validated, but double-check for security)
	origin := r.Header.Get("Origin")
	if !middleware.IsOriginAllowed(origin) {
		log.Printf("[Terminal WS Forward] Origin %s not allowed", origin)
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	// Prepare origin patterns for WebSocket library
	// Check if wildcard CORS is configured - if so, allow all origins
	acceptOptions := &websocket.AcceptOptions{}
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

	// Get the original WebSocket connection from the client
	clientConn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		log.Printf("[Terminal WS Forward] Failed to accept client WebSocket: %v", err)
		return
	}
	defer clientConn.Close(websocket.StatusInternalError, "")

	// Prepare headers for forwarding
	headers := make(http.Header)
	if auth := r.Header.Get("Authorization"); auth != "" {
		headers.Set("Authorization", auth)
	}
	// Also include the token from init message
	if initMsg.Token != "" {
		headers.Set("Authorization", "Bearer "+strings.TrimSpace(initMsg.Token))
	}

	// Forward WebSocket connection to target node
	targetPath := "/terminal/ws"
	targetConn, err := s.forwarder.ForwardWebSocket(ctx, targetNodeID, targetPath, headers)
	if err != nil {
		log.Printf("[Terminal WS Forward] Failed to connect to target node %s: %v", targetNodeID, err)
		_ = wsjson.Write(ctx, clientConn, terminalWSOutput{
			Type:    "error",
			Message: fmt.Sprintf("Failed to connect to node %s: %v", targetNodeID, err),
		})
		return
	}
	defer targetConn.Close(websocket.StatusInternalError, "")

	log.Printf("[Terminal WS Forward] Successfully connected to target node %s, proxying messages", targetNodeID)

	// Send init message to target node
	if err := wsjson.Write(ctx, targetConn, initMsg); err != nil {
		log.Printf("[Terminal WS Forward] Failed to send init message to target: %v", err)
		return
	}

	// Read initial response from target (connected/error message)
	var initResponse terminalWSOutput
	if err := wsjson.Read(ctx, targetConn, &initResponse); err != nil {
		log.Printf("[Terminal WS Forward] Failed to read init response: %v", err)
		return
	}
	// Forward initial response to client
	if err := wsjson.Write(ctx, clientConn, initResponse); err != nil {
		log.Printf("[Terminal WS Forward] Failed to forward init response: %v", err)
		return
	}

	// Proxy messages bidirectionally
	errChan := make(chan error, 2)

	// Forward messages from client to target node
	go func() {
		for {
			var msg terminalWSMessage
			if err := wsjson.Read(ctx, clientConn, &msg); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from client: %w", err)
				return
			}
			if err := wsjson.Write(ctx, targetConn, msg); err != nil {
				errChan <- fmt.Errorf("failed to write to target: %w", err)
				return
			}
		}
	}()

	// Forward messages from target node to client
	go func() {
		for {
			var output terminalWSOutput
			if err := wsjson.Read(ctx, targetConn, &output); err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errChan <- nil
					return
				}
				errChan <- fmt.Errorf("failed to read from target: %w", err)
				return
			}
			if err := wsjson.Write(ctx, clientConn, output); err != nil {
				errChan <- fmt.Errorf("failed to write to client: %w", err)
				return
			}
		}
	}()

	// Wait for an error or context cancellation
	select {
	case <-ctx.Done():
		log.Printf("[Terminal WS Forward] Context cancelled")
	case err := <-errChan:
		if err != nil {
			log.Printf("[Terminal WS Forward] Proxy error: %v", err)
		}
	}
}