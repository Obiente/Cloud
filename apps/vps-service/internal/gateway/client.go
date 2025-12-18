package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

// RequestHandler handles incoming requests from the gateway
type RequestHandler interface {
	// HandleRequest processes a request and returns response payload and error
	HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error)
}

// GatewayClient maintains persistent connections to multiple VPS gateways
type GatewayClient struct {
	apiSecret  string
	instanceID string
	version    string
	handlers   map[string]RequestHandler // method -> handler
	handlersMu sync.RWMutex

	// Sync callback - called when we need to sync allocations to a gateway
	syncCallback func(ctx context.Context, nodeName string) ([]*vpsgatewayv1.DesiredAllocation, error)
	syncMu       sync.RWMutex

	// Track active streams per node for sending requests
	streams   map[string]*connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]
	streamsMu sync.RWMutex

	// Track pending requests for responses
	pendingRequests   map[string]chan *vpsgatewayv1.GatewayResponse
	pendingRequestsMu sync.Mutex
	requestCounter    uint64
}

// parseNodeGatewayMapping parses the VPS_NODE_GATEWAY_ENDPOINTS env var
// Format: "node1:http://url1,node2:http://url2"
// Returns map of node name -> gateway URL
func parseNodeGatewayMapping() (map[string]string, error) {
	envVal := os.Getenv("VPS_NODE_GATEWAY_ENDPOINTS")
	if envVal == "" {
		return nil, fmt.Errorf("VPS_NODE_GATEWAY_ENDPOINTS environment variable is required")
	}

	result := make(map[string]string)
	pairs := strings.Split(envVal, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in VPS_NODE_GATEWAY_ENDPOINTS: %s (expected node:url)", pair)
		}

		nodeName := strings.TrimSpace(parts[0])
		gatewayURL := strings.TrimSpace(parts[1])

		if nodeName == "" {
			return nil, fmt.Errorf("empty node name in VPS_NODE_GATEWAY_ENDPOINTS")
		}

		// Validate URL format
		if !strings.HasPrefix(gatewayURL, "http://") && !strings.HasPrefix(gatewayURL, "https://") {
			return nil, fmt.Errorf("invalid gateway URL for node %s: %s (must start with http:// or https://)", nodeName, gatewayURL)
		}

		if _, err := url.Parse(gatewayURL); err != nil {
			return nil, fmt.Errorf("invalid gateway URL for node %s: %w", nodeName, err)
		}

		result[nodeName] = gatewayURL
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid gateway endpoints found in VPS_NODE_GATEWAY_ENDPOINTS")
	}

	return result, nil
}

// NewGatewayClient creates a new gateway client that connects to all configured gateways
func NewGatewayClient() (*GatewayClient, error) {
	apiSecret := os.Getenv("VPS_GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("VPS_GATEWAY_API_SECRET environment variable is required")
	}

	// Get instance ID (use hostname or generate)
	instanceID, _ := os.Hostname()
	if instanceID == "" {
		instanceID = fmt.Sprintf("vps-api-%d", time.Now().Unix())
	}

	return &GatewayClient{
		apiSecret:       apiSecret,
		instanceID:      instanceID,
		version:         "1.0.0", // TODO: Get from build info
		handlers:        make(map[string]RequestHandler),
		streams:         make(map[string]*connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]),
		pendingRequests: make(map[string]chan *vpsgatewayv1.GatewayResponse),
	}, nil
}

// RegisterHandler registers a handler for a specific request method
// This allows easy extension for new request types
func (c *GatewayClient) RegisterHandler(method string, handler RequestHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[method] = handler
	logger.Info("[GatewayClient] Registered handler for method: %s", method)
}

// SetSyncCallback sets the callback function used to retrieve allocations for syncing
// This function will be called when the client needs to sync allocations to a gateway
// The callback should query the database and return all allocations for the given gateway node
func (c *GatewayClient) SetSyncCallback(callback func(ctx context.Context, nodeName string) ([]*vpsgatewayv1.DesiredAllocation, error)) {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	c.syncCallback = callback
	logger.Info("[GatewayClient] Sync callback registered")
}

// Start connects to all configured gateways and maintains connections
func (c *GatewayClient) Start(ctx context.Context) error {
	// Parse gateway endpoints
	gateways, err := parseNodeGatewayMapping()
	if err != nil {
		return fmt.Errorf("failed to parse gateway endpoints: %w", err)
	}

	logger.Info("[GatewayClient] Starting connections to %d gateways", len(gateways))

	// Start connection to each gateway
	var wg sync.WaitGroup
	for nodeName, gatewayURL := range gateways {
		wg.Add(1)
		go func(node, url string) {
			defer wg.Done()
			c.maintainConnection(ctx, node, url)
		}(nodeName, gatewayURL)
	}

	// Wait for all connections to finish (which should only happen on context cancellation)
	wg.Wait()
	return ctx.Err()
}

func (c *GatewayClient) maintainConnection(ctx context.Context, nodeName, gatewayURL string) {
	logger.Info("[GatewayClient] Starting connection to %s at %s", nodeName, gatewayURL)

	for {
		select {
		case <-ctx.Done():
			logger.Info("[GatewayClient] Stopping connection to %s", nodeName)
			return
		default:
			if err := c.connectAndServe(ctx, nodeName, gatewayURL); err != nil {
				logger.Error("[GatewayClient] Connection to %s error: %v, reconnecting in 5s...", nodeName, err)
				time.Sleep(5 * time.Second)
				continue
			}
		}
	}
}

func (c *GatewayClient) connectAndServe(ctx context.Context, nodeName, gatewayURL string) error {
	// Create HTTP client with h2c support and keep-alive settings
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		// Enable HTTP/2 ping frames to keep connection alive
		ReadIdleTimeout: 45 * time.Second, // Send ping if no reads for 45s
		PingTimeout:     15 * time.Second, // Wait 15s for pong response
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   0, // No timeout for streaming connections
	}

	// Create Connect client
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		gatewayURL,
	)

	logger.Info("[GatewayClient] Connecting to gateway %s at %s", nodeName, gatewayURL)

	// Establish bidirectional stream
	// Note: For streaming calls, we need to set auth header using request options
	stream := client.RegisterGateway(ctx)

	// Set auth header on the stream
	stream.RequestHeader().Set("x-api-secret", c.apiSecret)

	// Send registration - tell the gateway which node it is
	regMsg := &vpsgatewayv1.GatewayMessage{
		Type: "register",
		Registration: &vpsgatewayv1.GatewayRegistration{
			GatewayId:     nodeName, // Tell gateway its node name
			Version:       c.version,
			GatewayIp:     gatewayURL,   // Gateway URL for reference
			GatewayIpDhcp: c.instanceID, // VPS service instance ID
		},
	}

	if err := stream.Send(regMsg); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	logger.Info("[GatewayClient] Sent registration to gateway %s", nodeName)

	// Store stream for sending requests
	c.streamsMu.Lock()
	c.streams[nodeName] = stream
	c.streamsMu.Unlock()

	// Remove stream on disconnect
	defer func() {
		c.streamsMu.Lock()
		delete(c.streams, nodeName)
		c.streamsMu.Unlock()
	}()

	// Start heartbeat sender
	go c.sendHeartbeats(ctx, stream)

	// Start periodic allocation sync (every 5 minutes)
	go c.periodicSync(ctx, stream, nodeName)

	// Handle incoming messages
	for {
		msg, err := stream.Receive()
		if err == io.EOF {
			logger.Info("[GatewayClient] Gateway closed connection")
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive error: %w", err)
		}

		switch msg.Type {
		case "registered":
			logger.Info("[GatewayClient] Successfully registered with gateway %s", nodeName)

			// Trigger initial sync on registration
			go c.syncAllocations(ctx, stream, nodeName)

		case "response":
			// Handle response to our request
			if msg.Response != nil {
				c.handleResponse(msg.Response)
			}

		case "request":
			if msg.Request != nil {
				go c.handleRequest(ctx, stream, msg.Request, nodeName)
			}

		case "heartbeat":
			logger.Debug("[GatewayClient] Received heartbeat from gateway %s", nodeName)

		case "sync_result":
			// Handle sync response from gateway with discovered allocations
			if msg.SyncResult != nil {
				logger.Info("[GatewayClient] Received sync result from gateway %s: added=%d removed=%d discovered=%d", 
					nodeName, msg.SyncResult.Added, msg.SyncResult.Removed, len(msg.SyncResult.DiscoveredAllocations))
				
				if len(msg.SyncResult.DiscoveredAllocations) > 0 {
					// Register discovered leases in database
					go c.registerDiscoveredLeases(ctx, nodeName, msg.SyncResult.DiscoveredAllocations)
				}
			} else {
				logger.Info("[GatewayClient] Received sync result from gateway %s (no data)", nodeName)
			}

		default:
			logger.Warn("[GatewayClient] Unknown message type from %s: %s", nodeName, msg.Type)
		}
	}
}

func (c *GatewayClient) sendHeartbeats(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := &vpsgatewayv1.GatewayMessage{
				Type:      "heartbeat",
				Heartbeat: timestamppb.Now(),
			}
			if err := stream.Send(msg); err != nil {
				logger.Debug("[GatewayClient] Failed to send heartbeat: %v", err)
				return
			}
		}
	}
}

func (c *GatewayClient) handleRequest(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], req *vpsgatewayv1.GatewayRequest, nodeName string) {
	var respPayload []byte
	var respError string

	// Look up handler for this method
	c.handlersMu.RLock()
	handler, ok := c.handlers[req.Method]
	c.handlersMu.RUnlock()

	if !ok {
		respError = fmt.Sprintf("no handler registered for method: %s", req.Method)
	} else {
		// Call handler
		payload, err := handler.HandleRequest(ctx, req.Method, req.Payload)
		if err != nil {
			respError = err.Error()
		} else {
			respPayload = payload
		}
	}

	// Send response
	resp := &vpsgatewayv1.GatewayResponse{
		RequestId: req.RequestId,
		Success:   respError == "",
		Error:     respError,
		Payload:   respPayload,
	}

	msg := &vpsgatewayv1.GatewayMessage{
		Type:     "response",
		Response: resp,
	}

	if err := stream.Send(msg); err != nil {
		logger.Error("[GatewayClient] Failed to send response: %v", err)
	}
}

// periodicSync sends allocation sync requests to the gateway every 5 minutes
func (c *GatewayClient) periodicSync(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], nodeName string) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Debug("[GatewayClient] Triggering periodic sync for gateway %s", nodeName)
			c.syncAllocations(ctx, stream, nodeName)
		}
	}
}

// syncAllocations queries the database and sends allocation sync to the gateway
func (c *GatewayClient) syncAllocations(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], nodeName string) {
	c.syncMu.RLock()
	callback := c.syncCallback
	c.syncMu.RUnlock()

	if callback == nil {
		logger.Warn("[GatewayClient] Sync callback not set, skipping sync for %s", nodeName)
		return
	}

	// Query database for allocations for this gateway node
	allocations, err := callback(ctx, nodeName)
	if err != nil {
		logger.Error("[GatewayClient] Failed to query allocations for %s: %v", nodeName, err)
		return
	}

	logger.Info("[GatewayClient] Syncing %d allocations to gateway %s", len(allocations), nodeName)

	// Send sync message
	msg := &vpsgatewayv1.GatewayMessage{
		Type: "sync_allocations",
		SyncAllocations: &vpsgatewayv1.SyncAllocationsRequest{
			Allocations: allocations,
		},
	}

	if err := stream.Send(msg); err != nil {
		logger.Error("[GatewayClient] Failed to send sync message to %s: %v", nodeName, err)
		return
	}

	logger.Debug("[GatewayClient] Sync message sent to gateway %s", nodeName)
}

// sendRequest sends a request to a gateway over the bidirectional stream and waits for response
func (c *GatewayClient) sendRequest(ctx context.Context, nodeName, method string, payload []byte) (*vpsgatewayv1.GatewayResponse, error) {
	// Get stream for this node
	c.streamsMu.RLock()
	stream, ok := c.streams[nodeName]
	c.streamsMu.RUnlock()
	
	if !ok || stream == nil {
		return nil, fmt.Errorf("no active connection to gateway %s", nodeName)
	}
	
	// Generate unique request ID
	requestID := fmt.Sprintf("vps-req-%d", atomic.AddUint64(&c.requestCounter, 1))
	
	// Create response channel
	respChan := make(chan *vpsgatewayv1.GatewayResponse, 1)
	c.pendingRequestsMu.Lock()
	c.pendingRequests[requestID] = respChan
	c.pendingRequestsMu.Unlock()
	
	// Clean up on exit
	defer func() {
		c.pendingRequestsMu.Lock()
		delete(c.pendingRequests, requestID)
		c.pendingRequestsMu.Unlock()
		close(respChan)
	}()
	
	// Send request
	msg := &vpsgatewayv1.GatewayMessage{
		Type: "request",
		Request: &vpsgatewayv1.GatewayRequest{
			RequestId: requestID,
			Method:    method,
			Payload:   payload,
		},
	}
	
	if err := stream.Send(msg); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Wait for response with timeout
	select {
	case resp := <-respChan:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// handleResponse handles a response from the gateway
func (c *GatewayClient) handleResponse(resp *vpsgatewayv1.GatewayResponse) {
	c.pendingRequestsMu.Lock()
	respChan, ok := c.pendingRequests[resp.RequestId]
	c.pendingRequestsMu.Unlock()
	
	if !ok {
		logger.Warn("[GatewayClient] Received response for unknown request: %s", resp.RequestId)
		return
	}
	
	// Send response to waiting goroutine
	select {
	case respChan <- resp:
	default:
		logger.Warn("[GatewayClient] Response channel full for request: %s", resp.RequestId)
	}
}

// AllocateIP allocates a DHCP IP for a VPS via the bidirectional stream
func (c *GatewayClient) AllocateIP(ctx context.Context, nodeName, vpsID, organizationID, macAddress string) (*vpsgatewayv1.AllocateIPResponse, error) {
	req := &vpsgatewayv1.AllocateIPRequest{
		VpsId:          vpsID,
		OrganizationId: organizationID,
		MacAddress:     macAddress,
	}
	
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	resp, err := c.sendRequest(ctx, nodeName, "AllocateIP", payload)
	if err != nil {
		return nil, err
	}
	
	if !resp.Success {
		return nil, fmt.Errorf("gateway error: %s", resp.Error)
	}
	
	var allocResp vpsgatewayv1.AllocateIPResponse
	if err := proto.Unmarshal(resp.Payload, &allocResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &allocResp, nil
}

// ReleaseIP releases a DHCP IP for a VPS via the bidirectional stream
func (c *GatewayClient) ReleaseIP(ctx context.Context, nodeName, vpsID string) error {
	req := &vpsgatewayv1.ReleaseIPRequest{
		VpsId: vpsID,
	}
	
	payload, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	resp, err := c.sendRequest(ctx, nodeName, "ReleaseIP", payload)
	if err != nil {
		return err
	}
	
	if !resp.Success {
		return fmt.Errorf("gateway error: %s", resp.Error)
	}
	
	return nil
}

// registerDiscoveredLeases registers DHCP leases discovered by gateway during self-healing
// registerDiscoveredLeases registers discovered allocations in the database
func (c *GatewayClient) registerDiscoveredLeases(ctx context.Context, nodeName string, allocations []*vpsgatewayv1.DesiredAllocation) {
	if len(allocations) == 0 {
		return
	}

	logger.Info("[GatewayClient] Registering %d discovered leases in database for gateway %s", len(allocations), nodeName)

	for _, alloc := range allocations {
		// Register in database using shared database package
		lease := database.DHCPLease{
			ID:             uuid.New().String(),
			VPSID:          alloc.VpsId,
			OrganizationID: alloc.OrganizationId,
			IPAddress:      alloc.IpAddress,
			MACAddress:     alloc.MacAddress,
			GatewayNode:    nodeName,
			IsPublic:       alloc.IsPublic,
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		// Use GORM's FirstOrCreate to avoid duplicates
		result := database.DB.WithContext(ctx).
			Where("vps_id = ? AND is_public = ?", alloc.VpsId, alloc.IsPublic).
			FirstOrCreate(&lease)

		if result.Error != nil {
			logger.Error("[GatewayClient] Failed to register discovered lease for VPS %s: %v", alloc.VpsId, result.Error)
			continue
		}

		if result.RowsAffected > 0 {
			logger.Info("[GatewayClient] Registered discovered lease: VPS %s -> IP %s (MAC: %s, Gateway: %s)", 
				alloc.VpsId, alloc.IpAddress, alloc.MacAddress, nodeName)
		} else {
			logger.Debug("[GatewayClient] Lease for VPS %s already exists in database", alloc.VpsId)
		}
	}

	logger.Info("[GatewayClient] Completed registration of %d discovered leases", len(allocations))
}

// ListIPs lists all allocated IPs from a gateway via the bidirectional stream
func (c *GatewayClient) ListIPs(ctx context.Context, nodeName, organizationID, vpsID string) ([]*vpsgatewayv1.IPAllocation, error) {
	req := &vpsgatewayv1.ListIPsRequest{
		OrganizationId: organizationID,
		VpsId:          vpsID,
	}
	
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	resp, err := c.sendRequest(ctx, nodeName, "ListIPs", payload)
	if err != nil {
		return nil, err
	}
	
	if !resp.Success {
		return nil, fmt.Errorf("gateway error: %s", resp.Error)
	}
	
	var listResp vpsgatewayv1.ListIPsResponse
	if err := proto.Unmarshal(resp.Payload, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return listResp.Allocations, nil
}
