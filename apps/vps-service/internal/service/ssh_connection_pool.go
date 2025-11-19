package vps

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsorch "vps-service/orchestrator"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"

	"connectrpc.com/connect"
	"golang.org/x/crypto/ssh"
)

// SSHConnectionPool manages persistent SSH connections to VPS instances
type SSHConnectionPool struct {
	connections map[string]*PooledSSHConnection // key: "vpsID:keyID" or "vpsID" if no key
	mu          sync.RWMutex
	gatewayClient *vpsorch.VPSGatewayClient
	idleTimeout   time.Duration
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// PooledSSHConnection represents a persistent SSH connection to a VPS
type PooledSSHConnection struct {
	vpsID      string
	keyID      string // SSH key ID used for authentication (empty if using password)
	sshClient  *ssh.Client
	vpsIP      string
	createdAt  time.Time
	lastUsed   time.Time
	channels   map[uint32]*ForwardedChannel
	mu         sync.RWMutex
}

// ForwardedChannel tracks a forwarded channel
type ForwardedChannel struct {
	clientChannel ssh.Channel
	vpsChannel    ssh.Channel
	channelType   string
}

// NewSSHConnectionPool creates a new SSH connection pool
func NewSSHConnectionPool(gatewayClient *vpsorch.VPSGatewayClient) *SSHConnectionPool {
	pool := &SSHConnectionPool{
		connections:   make(map[string]*PooledSSHConnection),
		gatewayClient: gatewayClient,
		idleTimeout:   5 * time.Minute,
		cleanupTicker:  time.NewTicker(1 * time.Minute),
		stopCleanup:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go pool.cleanupIdleConnections()

	return pool
}

// GetOrCreateConnection gets an existing connection or creates a new one
// keyID can be empty if no specific key is needed
func (p *SSHConnectionPool) GetOrCreateConnection(ctx context.Context, vpsID, vpsIP, keyID string, sshSigner ssh.Signer) (*PooledSSHConnection, error) {
	// Create connection key
	connKey := p.connectionKey(vpsID, keyID)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if connection exists and is healthy
	if conn, exists := p.connections[connKey]; exists {
		if conn.IsHealthy() {
			conn.lastUsed = time.Now()
			logger.Debug("[SSHConnectionPool] Reusing existing connection for VPS %s (key: %s)", vpsID, keyID)
			return conn, nil
		}
		// Connection unhealthy, close it
		logger.Info("[SSHConnectionPool] Closing unhealthy connection for VPS %s", vpsID)
		conn.Close()
		delete(p.connections, connKey)
	}

	// Create new connection
	logger.Info("[SSHConnectionPool] Creating new SSH connection for VPS %s (key: %s)", vpsID, keyID)
	sshClient, err := p.createSSHConnectionViaGateway(ctx, vpsID, vpsIP, sshSigner)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	conn := &PooledSSHConnection{
		vpsID:     vpsID,
		keyID:     keyID,
		sshClient: sshClient,
		vpsIP:     vpsIP,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		channels:  make(map[uint32]*ForwardedChannel),
	}

	p.connections[connKey] = conn
	return conn, nil
}

// streamConn implements net.Conn by wrapping a ProxySSH bidirectional stream
type streamConn struct {
	stream       *connect.BidiStreamForClient[vpsgatewayv1.ProxySSHRequest, vpsgatewayv1.ProxySSHResponse]
	connectionID string
	readBuf      []byte
	readMu       sync.Mutex
	writeMu      sync.Mutex
	closed       bool
	closeMu      sync.Mutex
}

func (c *streamConn) Read(b []byte) (n int, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// If we have buffered data, return it
	if len(c.readBuf) > 0 {
		n = copy(b, c.readBuf)
		c.readBuf = c.readBuf[n:]
		return n, nil
	}

	// Read from stream
	resp, err := c.stream.Receive()
	if err == io.EOF {
		return 0, io.EOF
	}
	if err != nil {
		return 0, err
	}

	if resp.Type == "data" {
		n = copy(b, resp.Data)
		// Buffer any remaining data
		if len(resp.Data) > n {
			c.readBuf = append(c.readBuf, resp.Data[n:]...)
		}
		return n, nil
	} else if resp.Type == "closed" || resp.Type == "error" {
		return 0, io.EOF
	}

	return 0, nil
}

func (c *streamConn) Write(b []byte) (n int, err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.closeMu.Lock()
	closed := c.closed
	c.closeMu.Unlock()

	if closed {
		return 0, io.ErrClosedPipe
	}

	err = c.stream.Send(&vpsgatewayv1.ProxySSHRequest{
		ConnectionId: c.connectionID,
		Type:         "data",
		Data:         b,
	})
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *streamConn) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	// Send close request
	c.stream.Send(&vpsgatewayv1.ProxySSHRequest{
		ConnectionId: c.connectionID,
		Type:         "close",
	})
	return nil
}

func (c *streamConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (c *streamConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (c *streamConn) SetDeadline(t time.Time) error      { return nil }
func (c *streamConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *streamConn) SetWriteDeadline(t time.Time) error { return nil }

// createSSHConnectionViaGateway creates an SSH client connection via the gateway
func (p *SSHConnectionPool) createSSHConnectionViaGateway(ctx context.Context, vpsID, vpsIP string, sshSigner ssh.Signer) (*ssh.Client, error) {
	if p.gatewayClient == nil {
		return nil, fmt.Errorf("gateway client not available")
	}

	if sshSigner == nil {
		return nil, fmt.Errorf("SSH signer required for authentication")
	}

	// Create ProxySSH stream
	stream, err := p.gatewayClient.ProxySSH(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway stream: %w", err)
	}

	// Create connection ID
	connectionID := fmt.Sprintf("pool-%d", time.Now().UnixNano())

	// Send connect request
	connectReq := &vpsgatewayv1.ProxySSHRequest{
		ConnectionId: connectionID,
		Type:         "connect",
		Target:       vpsIP,
		Port:         22,
	}

	if err := stream.Send(connectReq); err != nil {
		return nil, fmt.Errorf("failed to send connect request: %w", err)
	}

	// Wait for connected response
	resp, err := stream.Receive()
	if err != nil {
		return nil, fmt.Errorf("failed to receive connect response: %w", err)
	}

	if resp.Type != "connected" {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}

	// Create stream-based connection
	conn := &streamConn{
		stream:       stream,
		connectionID: connectionID,
		readBuf:      make([]byte, 0),
	}

	// Perform SSH handshake over the stream connection
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshSigner),
		},
	}

	// Create SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, vpsIP, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SSH client connection: %w", err)
	}

	client := ssh.NewClient(sshConn, chans, reqs)
	return client, nil
}

// IsHealthy checks if the connection is still healthy
func (c *PooledSSHConnection) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.sshClient == nil {
		return false
	}

	// Try to create a test session to check if connection is alive
	session, err := c.sshClient.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// Close closes the SSH connection
func (c *PooledSSHConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close all forwarded channels
	for _, ch := range c.channels {
		if ch.clientChannel != nil {
			ch.clientChannel.Close()
		}
		if ch.vpsChannel != nil {
			ch.vpsChannel.Close()
		}
	}
	c.channels = make(map[uint32]*ForwardedChannel)

	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// connectionKey creates a key for the connection map
func (p *SSHConnectionPool) connectionKey(vpsID, keyID string) string {
	if keyID == "" {
		return vpsID
	}
	return fmt.Sprintf("%s:%s", vpsID, keyID)
}

// cleanupIdleConnections periodically closes idle connections
func (p *SSHConnectionPool) cleanupIdleConnections() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.mu.Lock()
			now := time.Now()
			for key, conn := range p.connections {
				if now.Sub(conn.lastUsed) > p.idleTimeout {
					logger.Info("[SSHConnectionPool] Closing idle connection for VPS %s", conn.vpsID)
					conn.Close()
					delete(p.connections, key)
				}
			}
			p.mu.Unlock()
		case <-p.stopCleanup:
			return
		}
	}
}

// Close closes the connection pool
func (p *SSHConnectionPool) Close() error {
	close(p.stopCleanup)
	p.cleanupTicker.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.Close()
	}
	p.connections = make(map[string]*PooledSSHConnection)

	return nil
}
