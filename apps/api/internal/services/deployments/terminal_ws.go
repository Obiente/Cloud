package deployments

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"api/internal/auth"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type terminalWSMessage struct {
	Type           string `json:"type"`
	DeploymentID   string `json:"deploymentId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	Token          string `json:"token,omitempty"`
	Input          []int  `json:"input,omitempty"`
	Cols           int    `json:"cols,omitempty"`
	Rows           int    `json:"rows,omitempty"`
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

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Origin checking can be added here if needed; for now, rely on auth token validation.
		InsecureSkipVerify: true,
	})
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

	if initMsg.Token == "" {
		sendError("Authentication token is required")
		conn.Close(websocket.StatusPolicyViolation, "missing token")
		return
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

	session, cleanup, created, err := s.ensureTerminalSession(ctx, initMsg.DeploymentID, initMsg.OrganizationID, initMsg.Cols, initMsg.Rows)
	if err != nil {
		sendError(err.Error())
		conn.Close(websocket.StatusInternalError, "failed to initialize terminal")
		return
	}

	var cleanupOnce sync.Once
	cleanupFn := func() {
		cleanupOnce.Do(cleanup)
	}
	defer cleanupFn()

	if created {
		log.Printf("[Terminal WS] New terminal session created, sending initial newline")
		if _, err := session.conn.Write([]byte("\r\n")); err != nil {
			log.Printf("[Terminal WS] Failed to write initial newline: %v", err)
		}
	}

	if err := writeJSON(map[string]string{"type": "connected"}); err != nil {
		log.Printf("[Terminal WS] Failed to send connected message: %v", err)
		conn.Close(websocket.StatusInternalError, "failed to send connected")
		return
	}

	outputCtx, outputCancel := context.WithCancel(ctx)
	outputDone := make(chan struct{})

	// Forward container output to websocket client
	go func() {
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
			// TODO: implement container resize support if needed
			log.Printf("[Terminal WS] Resize event received: %dx%d (not implemented)", msg.Cols, msg.Rows)
		case "ping":
			_ = writeJSON(map[string]string{"type": "pong"})
		default:
			sendError("Unknown message type")
		}
	}
}
