package orchestrator

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"api/internal/logger"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
)

// VPSGatewayClient handles communication with the vps-gateway service
// In forward connection pattern, API connects to gateway's gRPC server
type VPSGatewayClient struct {
	client    vpsgatewayv1connect.VPSGatewayServiceClient
	apiSecret string
	baseURL   string
}

// NewVPSGatewayClient creates a new vps-gateway client
// gatewayURL should be the full URL to the gateway (e.g., "http://gateway-ip:1537")
// If gatewayURL is empty, uses VPS_GATEWAY_URL from environment
func NewVPSGatewayClient(gatewayURL string) (*VPSGatewayClient, error) {
	if gatewayURL == "" {
		gatewayURL = os.Getenv("VPS_GATEWAY_URL")
		if gatewayURL == "" {
			return nil, fmt.Errorf("gateway URL required (provide as parameter or set VPS_GATEWAY_URL)")
		}
	}

	apiSecret := os.Getenv("VPS_GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("VPS_GATEWAY_API_SECRET environment variable is required")
	}

	// Create HTTP client with timeout and HTTP/2 (h2c) support
	// The gateway server uses h2c.NewHandler which supports cleartext HTTP/2
	// Connect RPC bidirectional streaming requires HTTP/2
	// Use http2.Transport with AllowHTTP for cleartext HTTP/2 (h2c) connections
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			// For h2c, we dial without TLS (cleartext)
			return net.Dial(network, addr)
		},
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Create Connect client
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		gatewayURL,
		connect.WithInterceptors(newGatewayAuthInterceptor(apiSecret)),
	)

	return &VPSGatewayClient{
		client:    client,
		apiSecret: apiSecret,
		baseURL:   gatewayURL,
	}, nil
}

// gatewayAuthInterceptor adds the API secret header to requests
type gatewayAuthInterceptor struct {
	secret string
}

func newGatewayAuthInterceptor(secret string) connect.Interceptor {
	return &gatewayAuthInterceptor{secret: secret}
}

func (i *gatewayAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-api-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *gatewayAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		conn.RequestHeader().Set("x-api-secret", i.secret)
		return next(ctx, conn)
	}
}

func (i *gatewayAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-api-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *gatewayAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("x-api-secret", i.secret)
		return conn
	}
}

// AllocateIP allocates a DHCP IP address for a VPS instance
func (c *VPSGatewayClient) AllocateIP(ctx context.Context, vpsID, organizationID, macAddress string) (*vpsgatewayv1.AllocateIPResponse, error) {
	req := connect.NewRequest(&vpsgatewayv1.AllocateIPRequest{
		VpsId:          vpsID,
		OrganizationId: organizationID,
		MacAddress:     macAddress,
	})

	resp, err := c.client.AllocateIP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP from gateway: %w", err)
	}

	logger.Info("[VPSGateway] Allocated IP %s for VPS %s (org: %s)", resp.Msg.IpAddress, vpsID, organizationID)
	return resp.Msg, nil
}

// ReleaseIP releases a DHCP IP address for a VPS instance
func (c *VPSGatewayClient) ReleaseIP(ctx context.Context, vpsID string) error {
	req := connect.NewRequest(&vpsgatewayv1.ReleaseIPRequest{
		VpsId: vpsID,
	})

	resp, err := c.client.ReleaseIP(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to release IP from gateway: %w", err)
	}

	if !resp.Msg.Success {
		return fmt.Errorf("gateway returned failure for IP release")
	}

	logger.Info("[VPSGateway] Released IP for VPS %s", vpsID)
	return nil
}

// ListIPs lists all allocated IP addresses, optionally filtered
func (c *VPSGatewayClient) ListIPs(ctx context.Context, organizationID, vpsID string) ([]*vpsgatewayv1.IPAllocation, error) {
	req := connect.NewRequest(&vpsgatewayv1.ListIPsRequest{
		OrganizationId: organizationID,
		VpsId:          vpsID,
	})

	resp, err := c.client.ListIPs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list IPs from gateway: %w", err)
	}

	return resp.Msg.Allocations, nil
}

// GetGatewayInfo returns gateway status and configuration
func (c *VPSGatewayClient) GetGatewayInfo(ctx context.Context) (*vpsgatewayv1.GetGatewayInfoResponse, error) {
	req := connect.NewRequest(&vpsgatewayv1.GetGatewayInfoRequest{})

	resp, err := c.client.GetGatewayInfo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway info: %w", err)
	}

	return resp.Msg, nil
}

// ProxySSH proxies an SSH connection through the gateway
// This returns a bidirectional stream that can be used to forward SSH traffic
func (c *VPSGatewayClient) ProxySSH(ctx context.Context) (*connect.BidiStreamForClient[vpsgatewayv1.ProxySSHRequest, vpsgatewayv1.ProxySSHResponse], error) {
	stream := c.client.ProxySSH(ctx)
	return stream, nil
}

// CreateTCPConnection creates a raw TCP connection to a VPS via gateway
// Returns a net.Conn that can be used for SSH handshake
// The connection is backed by the gateway's ProxySSH stream
func (c *VPSGatewayClient) CreateTCPConnection(ctx context.Context, target string, port int) (net.Conn, error) {
	if port == 0 {
		port = 22
	}
	
	// Create ProxySSH stream
	stream, err := c.ProxySSH(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create ProxySSH stream: %w", err)
	}
	
	// Generate connection ID
	connectionID := fmt.Sprintf("tcp-%d", time.Now().UnixNano())
	
	// Send connect request
	req := &vpsgatewayv1.ProxySSHRequest{
		ConnectionId: connectionID,
		Type:         "connect",
		Target:       target,
		Port:         int32(port),
	}
	
	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send connect request: %w", err)
	}
	
	// Wait for connected response FIRST (before starting read goroutine)
	// This ensures we don't have a race where readFromStream consumes the "connected" message
	resp, err := stream.Receive()
	if err != nil {
		return nil, fmt.Errorf("failed to receive connect response: %w", err)
	}
	
	if resp.Type != "connected" {
		return nil, fmt.Errorf("unexpected response type: %s (expected connected)", resp.Type)
	}
	
	// Create a connection wrapper that uses the stream
	conn := &gatewayTCPConnection{
		stream:       stream,
		connectionID: connectionID,
		target:       target,
		port:         port,
		readChan:     make(chan []byte, 100),
		readErrChan:  make(chan error, 1),
		ctx:          ctx,
	}
	
	// Start goroutine to read from stream AFTER receiving connected
	// This ensures we've consumed the "connected" message before the goroutine starts
	go conn.readFromStream()
	
	return conn, nil
}

// gatewayTCPConnection wraps a ProxySSH stream as a net.Conn
type gatewayTCPConnection struct {
	stream       *connect.BidiStreamForClient[vpsgatewayv1.ProxySSHRequest, vpsgatewayv1.ProxySSHResponse]
	connectionID string
	target       string
	port         int
	readChan     chan []byte
	readErrChan  chan error
	readBuf      []byte
	readBufPos   int
	closed       bool
	mu           sync.Mutex
	ctx          context.Context
}

func (c *gatewayTCPConnection) Read(b []byte) (n int, err error) {
	// Check for buffered data first (with lock)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, io.EOF
	}
	
	// If we have buffered data, use it
	if len(c.readBuf) > c.readBufPos {
		n = copy(b, c.readBuf[c.readBufPos:])
		c.readBufPos += n
		if c.readBufPos >= len(c.readBuf) {
			c.readBuf = nil
			c.readBufPos = 0
		}
		c.mu.Unlock()
		return n, nil
	}
	c.mu.Unlock()
	
	// Wait for data from stream (without holding lock to allow readFromStream to send)
	select {
	case data := <-c.readChan:
		c.mu.Lock()
		n = copy(b, data)
		if n < len(data) {
			// Buffer remaining data
			c.readBuf = data[n:]
			c.readBufPos = 0
		}
		c.mu.Unlock()
		return n, nil
	case err := <-c.readErrChan:
		return 0, err
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	}
}

func (c *gatewayTCPConnection) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return 0, io.EOF
	}
	
	req := &vpsgatewayv1.ProxySSHRequest{
		ConnectionId: c.connectionID,
		Type:         "data",
		Data:         b,
	}
	
	if err := c.stream.Send(req); err != nil {
		logger.Debug("[VPSGatewayClient] Failed to send data to gateway stream for connection %s: %v", c.connectionID, err)
		// Mark as closed if stream error
		c.closed = true
		return 0, fmt.Errorf("failed to send data to gateway: %w", err)
	}
	
	return len(b), nil
}

func (c *gatewayTCPConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	
	// Send close request
	req := &vpsgatewayv1.ProxySSHRequest{
		ConnectionId: c.connectionID,
		Type:         "close",
	}
	c.stream.Send(req)
	
	return nil
}

func (c *gatewayTCPConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
}

func (c *gatewayTCPConnection) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP(c.target),
		Port: c.port,
	}
}

func (c *gatewayTCPConnection) SetDeadline(t time.Time) error {
	// Not implemented - stream doesn't support deadlines
	return nil
}

func (c *gatewayTCPConnection) SetReadDeadline(t time.Time) error {
	// Not implemented - stream doesn't support deadlines
	return nil
}

func (c *gatewayTCPConnection) SetWriteDeadline(t time.Time) error {
	// Not implemented - stream doesn't support deadlines
	return nil
}

func (c *gatewayTCPConnection) readFromStream() {
	for {
		resp, err := c.stream.Receive()
		if err != nil {
			if err != io.EOF {
				select {
				case c.readErrChan <- err:
				case <-c.ctx.Done():
					return
				}
			} else {
				select {
				case c.readErrChan <- io.EOF:
				case <-c.ctx.Done():
					return
				}
			}
			return
		}
		
		switch resp.Type {
		case "connected":
			// Already handled in CreateTCPConnection, but ignore it here
			// Data might arrive immediately after connected
			continue
		case "data":
			// Non-blocking send with timeout to prevent deadlock if channel is full
			select {
			case c.readChan <- resp.Data:
			case <-c.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				// Channel is full and no one is reading - this indicates a problem
				c.readErrChan <- fmt.Errorf("read channel full, connection may be stuck")
				return
			}
		case "error":
			select {
			case c.readErrChan <- fmt.Errorf("gateway error: %s", resp.Error):
			case <-c.ctx.Done():
				return
			}
			return
		case "closed":
			select {
			case c.readErrChan <- io.EOF:
			case <-c.ctx.Done():
				return
			}
			return
		}
	}
}
