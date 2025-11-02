package deployments

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	conn        io.ReadWriteCloser
	containerID string
	createdAt   time.Time
}

// terminalSessions stores active terminal sessions keyed by deploymentID
var terminalSessions = make(map[string]*TerminalSession)
var terminalSessionsMutex sync.RWMutex

// ensureTerminalSession returns an active terminal session for the given deployment,
// creating one if necessary. It returns the session, a cleanup function, and a boolean
// indicating whether a new session was created.
func (s *Service) ensureTerminalSession(ctx context.Context, deploymentID, orgID string, cols, rows int, containerID, serviceName string) (*TerminalSession, func(), bool, error) {
	// Normalize terminal dimensions
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeNotFound, err)
	}

	// Check if container is running - Docker exec requires running containers
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, nil, false, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("container is stopped. Type 'start' to start the container first."),
		)
	}

	terminalSessionsMutex.Lock()
	session, exists := terminalSessions[deploymentID]
	if !exists {
		conn, err := dcli.ContainerExec(ctx, loc.ContainerID, cols, rows)
		if err != nil {
			terminalSessionsMutex.Unlock()
			return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create terminal: %w", err))
		}

		session = &TerminalSession{
			conn:        conn,
			containerID: loc.ContainerID,
			createdAt:   time.Now(),
		}
		terminalSessions[deploymentID] = session
	}
	terminalSessionsMutex.Unlock()

	cleanup := func() {
		terminalSessionsMutex.Lock()
		if s, exists := terminalSessions[deploymentID]; exists && s == session {
			delete(terminalSessions, deploymentID)
			session.conn.Close()
		}
		terminalSessionsMutex.Unlock()
	}

	return session, cleanup, !exists, nil
}

// StreamTerminal implements bidirectional streaming for terminal access
// This provides better input/output synchronization compared to separate RPCs
func (s *Service) StreamTerminal(ctx context.Context, stream *connect.BidiStream[deploymentsv1.TerminalInput, deploymentsv1.TerminalOutput]) error {
	var deploymentID string
	var orgID string
	var session *TerminalSession
	var initialized bool
	var cleanupSession func()

	// Channel to send container output
	outputChan := make(chan []byte, 100)
	errorChan := make(chan error, 1)

	// Handle input from client in a goroutine
	go func() {
		defer close(outputChan)
		defer close(errorChan)

		for {
			input, err := stream.Receive()
			if err != nil {
				if err == io.EOF {
					// Client closed the stream
					return
				}
				errorChan <- err
				return
			}
			if input == nil {
				continue
			}

			// First message initializes the session
			if !initialized {
				deploymentID = input.GetDeploymentId()
				orgID = input.GetOrganizationId()
				containerID := input.GetContainerId()
				serviceName := input.GetServiceName()

				// Check permissions
				if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
					errorChan <- connect.NewError(connect.CodePermissionDenied, err)
					return
				}

				cols := int(input.GetCols())
				rows := int(input.GetRows())
				var created bool
				var err error
				session, cleanupSession, created, err = s.ensureTerminalSession(ctx, deploymentID, orgID, cols, rows, containerID, serviceName)
				if err != nil {
					errorChan <- err
					return
				}

				// Send a newline to wake up the shell and trigger a prompt when session is new
				if created {
					log.Printf("[Terminal] Sending initial newline to shell")
					if _, err := session.conn.Write([]byte("\r\n")); err != nil {
						log.Printf("[Terminal] Warning: Failed to send initial newline: %v", err)
					} else {
						log.Printf("[Terminal] Successfully sent initial newline")
					}
				}

				// Start reading from container in background
				go func(currentSession *TerminalSession, cleanup func()) {
					defer cleanup()

					buf := make([]byte, 4096)
					log.Printf("[Terminal] Starting to read from container connection")
					for {
						select {
						case <-ctx.Done():
							log.Printf("[Terminal] Context cancelled, stopping read loop")
							return
						default:
						}

						n, err := currentSession.conn.Read(buf)
						if n > 0 {
							log.Printf("[Terminal] Read %d bytes from container: %q", n, string(buf[:min(n, 100)]))
							select {
							case outputChan <- buf[:n]:
								log.Printf("[Terminal] Sent %d bytes to output channel", n)
							case <-ctx.Done():
								log.Printf("[Terminal] Context cancelled while sending output")
								return
							}
						}
						if err != nil {
							if err == io.EOF {
								log.Printf("[Terminal] Received EOF from container")
								select {
								case outputChan <- []byte("\r\n[Terminal session closed]\r\n"):
								case <-ctx.Done():
								}
							} else {
								log.Printf("[Terminal] Read error: %v", err)
							}
							errorChan <- err
							return
						}
					}
				}(session, cleanupSession)

				initialized = true
			}

			// Handle input
			if len(input.GetInput()) > 0 && session != nil {
				log.Printf("[Terminal] Writing %d bytes of input to container", len(input.GetInput()))
				n, err := session.conn.Write(input.GetInput())
				if err != nil {
					log.Printf("[Terminal] Error writing input: %v", err)
					errorChan <- fmt.Errorf("failed to write input: %w", err)
					return
				}
				log.Printf("[Terminal] Successfully wrote %d bytes to container", n)
			}

			// Handle resize
			if input.GetCols() > 0 && input.GetRows() > 0 && session != nil {
				// TODO: Implement terminal resize via exec resize API
				// For now, just log it
				log.Printf("[Terminal] Resize requested: %dx%d (not implemented)", input.GetCols(), input.GetRows())
			}
		}
	}()

	// Send output to client
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errorChan:
			return err
		case output, ok := <-outputChan:
			if !ok {
				// Channel closed, send final exit message
				_ = stream.Send(&deploymentsv1.TerminalOutput{
					Output: []byte("\r\n[Terminal session closed]\r\n"),
					Exit:   true,
				})
				return nil
			}
			// Log output for debugging
			log.Printf("[Terminal] Sending output: %d bytes", len(output))
			if err := stream.Send(&deploymentsv1.TerminalOutput{
				Output: output,
				Exit:   false,
			}); err != nil {
				log.Printf("[Terminal] Error sending output: %v", err)
				return err
			}
		}
	}
}

// StreamTerminalOutput streams terminal output from a deployment container
// This is a server stream that works well with gRPC-Web in browsers
func (s *Service) StreamTerminalOutput(ctx context.Context, req *connect.Request[deploymentsv1.StreamTerminalOutputRequest], stream *connect.ServerStream[deploymentsv1.TerminalOutput]) error {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	// Find container for this deployment
	// Validate and refresh locations to ensure we have valid container IDs
	locations, err := database.ValidateAndRefreshLocations(deploymentID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to validate locations: %w", err))
	}
	if len(locations) == 0 {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}

	loc := locations[0]
	dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Get or create terminal connection
	cols := int(req.Msg.GetCols())
	rows := int(req.Msg.GetRows())
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	// Check if session exists
	terminalSessionsMutex.Lock()
	session, exists := terminalSessions[deploymentID]
	if !exists {
		// Create new terminal connection
		conn, err := dcli.ContainerExec(ctx, loc.ContainerID, cols, rows)
		if err != nil {
			terminalSessionsMutex.Unlock()
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create terminal: %w", err))
		}
		session = &TerminalSession{
			conn:        conn,
			containerID: loc.ContainerID,
			createdAt:   time.Now(),
		}
		terminalSessions[deploymentID] = session
	}
	terminalSessionsMutex.Unlock()

	// Clean up session when stream ends
	defer func() {
		terminalSessionsMutex.Lock()
		if s, exists := terminalSessions[deploymentID]; exists && s == session {
			delete(terminalSessions, deploymentID)
			session.conn.Close()
		}
		terminalSessionsMutex.Unlock()
	}()

	// Read from container stdout/stderr and send to client
	buf := make([]byte, 4096)
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := session.conn.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&deploymentsv1.TerminalOutput{
				Output: buf[:n],
				Exit:   false,
			}); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if err == io.EOF {
				// Terminal closed
				_ = stream.Send(&deploymentsv1.TerminalOutput{
					Output: []byte("\r\n[Terminal session closed]\r\n"),
					Exit:   true,
				})
			}
			return err
		}
	}
}

// SendTerminalInput sends input to an active terminal session
func (s *Service) SendTerminalInput(ctx context.Context, req *connect.Request[deploymentsv1.SendTerminalInputRequest]) (*connect.Response[deploymentsv1.SendTerminalInputResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get session
	terminalSessionsMutex.RLock()
	session, exists := terminalSessions[deploymentID]
	terminalSessionsMutex.RUnlock()

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no active terminal session for deployment"))
	}

	// Write input to container
	if len(req.Msg.GetInput()) > 0 {
		if _, err := session.conn.Write(req.Msg.GetInput()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write input: %w", err))
		}
	}

	// Handle resize if dimensions changed
	if req.Msg.GetCols() > 0 && req.Msg.GetRows() > 0 {
		// TODO: Implement terminal resize via exec resize API
		log.Printf("[Terminal] Resize requested: %dx%d (not implemented)", req.Msg.GetCols(), req.Msg.GetRows())
	}

	return connect.NewResponse(&deploymentsv1.SendTerminalInputResponse{
		Success: true,
	}), nil
}
