package gameservers

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"

	"connectrpc.com/connect"
)

// attachReadWriteCloser combines a reader (stdout/stderr) and writer (stdin)
// Handles both TTY mode (raw output) and non-TTY mode (8-byte headers)
type attachReadWriteCloser struct {
	reader  io.ReadCloser
	writer  io.WriteCloser
	closeFn func() error
	closed  bool
	isTTY   bool // Whether this is a TTY connection (raw output, no headers)
	mu      sync.Mutex
}

func (a *attachReadWriteCloser) Read(p []byte) (n int, err error) {
	if a.closed {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	
	if a.isTTY {
		// TTY mode: raw output, no headers, read directly
		return a.reader.Read(p)
	}
	
	// Non-TTY mode: Docker multiplexes stdout/stderr with 8-byte headers
	// Format: [stream_type(1)][reserved(3)][payload_length(4 bytes, big-endian)]
	header := make([]byte, 8)
	_, err = io.ReadFull(a.reader, header)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return 0, io.EOF
		}
		return 0, err
	}

	// Read payload length (bytes 4-7, big-endian)
	payloadLength := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))
	if payloadLength == 0 {
		// Empty frame, try next frame
		return a.Read(p)
	}

	// Read the payload - read into p if it fits, otherwise read into temp buffer
	if payloadLength <= len(p) {
		// Payload fits in buffer - read directly
		n, err = io.ReadFull(a.reader, p[:payloadLength])
		if err != nil && err != io.ErrUnexpectedEOF {
			return 0, err
		}
		return n, nil
	} else {
		payload := make([]byte, payloadLength)
		_, err = io.ReadFull(a.reader, payload)
		if err != nil && err != io.ErrUnexpectedEOF {
			return 0, err
		}
		copy(p, payload[:len(p)])
		return len(p), nil
	}
}

func (a *attachReadWriteCloser) Write(p []byte) (n int, err error) {
	if a.closed {
		return 0, io.ErrClosedPipe
	}
	n, err = a.writer.Write(p)
	if err != nil {
		return n, err
	}
	
	// Try to flush if the writer supports it (important for TTY mode)
	if flusher, ok := a.writer.(interface{ Flush() error }); ok {
		if flushErr := flusher.Flush(); flushErr != nil {
			log.Printf("[GameServer Terminal] Failed to flush after write: %v", flushErr)
			// Don't fail the write if flush fails
		}
	}
	
	return n, err
}

func (a *attachReadWriteCloser) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	
	var errs []error
	if a.reader != nil {
		if err := a.reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if a.writer != nil {
		if err := a.writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if a.closeFn != nil {
		if err := a.closeFn(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

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

	containerInfo, err := dcli.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		log.Printf("[GameServer Terminal] Container %s is not running, cannot attach", *gameServer.ContainerID)
		return nil, nil, false, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("container is stopped. Type 'start' to start the container first"),
		)
	}

	// Check if container has TTY enabled (from Config.Tty)
	// If TTY is enabled, we need to attach with TTY mode for proper terminal support
	containerHasTTY := containerInfo.Config.Tty

	terminalSessionsMutex.Lock()
	session, exists := terminalSessions[gameServerID]
	if !exists {
		reader, writer, closeFn, err := dcli.ContainerAttach(ctx, *gameServer.ContainerID, docker.ContainerAttachOptions{
			Stdin:  true,
			Stdout: true,
			Stderr: true,
			Tty:    containerHasTTY, // Match container's TTY setting
		})
		if err != nil {
			terminalSessionsMutex.Unlock()
			log.Printf("[GameServer Terminal] Failed to attach to container %s: %v", *gameServer.ContainerID, err)
			return nil, nil, false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to attach to container: %w", err))
		}

		conn := &attachReadWriteCloser{
			reader:  reader,
			writer:  writer,
			closeFn: closeFn,
			isTTY:   containerHasTTY, // Track TTY mode for reading
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

// CloseTerminalSession closes and removes an existing terminal session for a game server.
// This is useful when you need to force a new attach (e.g., after server restart).
func (s *Service) CloseTerminalSession(gameServerID string) {
	terminalSessionsMutex.Lock()
	defer terminalSessionsMutex.Unlock()
	
	if session, exists := terminalSessions[gameServerID]; exists {
		log.Printf("[GameServer Terminal] Closing existing terminal session for game server %s", gameServerID)
		if session.conn != nil {
			session.conn.Close()
		}
		delete(terminalSessions, gameServerID)
	}
}
