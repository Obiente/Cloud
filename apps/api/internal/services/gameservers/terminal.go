package gameservers

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"api/docker"

	"connectrpc.com/connect"
)

// TerminalSession represents an active terminal session for a game server
type TerminalSession struct {
	conn        io.ReadWriteCloser
	containerID string
	createdAt   time.Time
}

// terminalSessions stores active terminal sessions keyed by gameServerID
var terminalSessions = make(map[string]*TerminalSession)
var terminalSessionsMutex sync.RWMutex

// ensureTerminalSession returns an active terminal session for the given game server,
// creating one if necessary. It returns the session, a cleanup function, and a boolean
// indicating whether a new session was created.
func (s *Service) ensureTerminalSession(ctx context.Context, gameServerID, orgID string, cols, rows int) (*TerminalSession, func(), bool, error) {
	// Normalize terminal dimensions
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	// Get game server from database
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found: %w", err))
	}

	if gameServer.ContainerID == nil {
		return nil, nil, false, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server has no container ID"))
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Check if container is running - Docker exec requires running containers
	containerInfo, err := dcli.ContainerInspect(ctx, *gameServer.ContainerID)
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
	session, exists := terminalSessions[gameServerID]
	if !exists {
		conn, err := dcli.ContainerExec(ctx, *gameServer.ContainerID, cols, rows)
		if err != nil {
			terminalSessionsMutex.Unlock()
			return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create terminal: %w", err))
		}

		session = &TerminalSession{
			conn:        conn,
			containerID: *gameServer.ContainerID,
			createdAt:   time.Now(),
		}
		terminalSessions[gameServerID] = session
	}
	terminalSessionsMutex.Unlock()

	cleanup := func() {
		terminalSessionsMutex.Lock()
		if s, exists := terminalSessions[gameServerID]; exists && s == session {
			delete(terminalSessions, gameServerID)
			session.conn.Close()
		}
		terminalSessionsMutex.Unlock()
	}

	return session, cleanup, !exists, nil
}
