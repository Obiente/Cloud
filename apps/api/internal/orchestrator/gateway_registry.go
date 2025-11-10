package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"api/internal/logger"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"

	"google.golang.org/protobuf/proto"
)

// GatewayConnection represents a connected gateway
type GatewayConnection struct {
	GatewayID      string
	Version        string
	GatewayIP      string
	RegisteredAt   time.Time
	LastHeartbeat  time.Time
	MessageChan    chan *vpsgatewayv1.GatewayMessage
	RequestChan    chan *vpsgatewayv1.GatewayRequest
	RequestMap     map[string]chan *vpsgatewayv1.GatewayResponse
	RequestMapLock sync.RWMutex
	mu             sync.RWMutex
}

// GatewayRegistry manages connected gateways
type GatewayRegistry struct {
	gateways       map[string]*GatewayConnection
	gatewayMetrics map[string]string // gatewayID -> metrics text
	mu             sync.RWMutex
	apiSecret      string
}

var globalRegistry *GatewayRegistry
var globalRegistryOnce sync.Once

// GetGlobalGatewayRegistry returns the global gateway registry singleton
func GetGlobalGatewayRegistry() *GatewayRegistry {
	globalRegistryOnce.Do(func() {
		apiSecret := ""
		// Get API secret from environment (same as gateway uses)
		if secret := os.Getenv("VPS_GATEWAY_API_SECRET"); secret != "" {
			apiSecret = secret
		}
		globalRegistry = NewGatewayRegistry(apiSecret)
	})
	return globalRegistry
}

// NewGatewayRegistry creates a new gateway registry
func NewGatewayRegistry(apiSecret string) *GatewayRegistry {
	return &GatewayRegistry{
		gateways:       make(map[string]*GatewayConnection),
		gatewayMetrics: make(map[string]string),
		apiSecret:      apiSecret,
	}
}

// RegisterGateway registers a new gateway connection
func (r *GatewayRegistry) RegisterGateway(ctx context.Context, gatewayID, version, gatewayIP string) (*GatewayConnection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if gateway already exists
	if existing, ok := r.gateways[gatewayID]; ok {
		logger.Warn("[GatewayRegistry] Gateway %s already registered, replacing connection", gatewayID)
		close(existing.MessageChan)
		close(existing.RequestChan)
	}

	conn := &GatewayConnection{
		GatewayID:     gatewayID,
		Version:       version,
		GatewayIP:     gatewayIP,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		MessageChan:   make(chan *vpsgatewayv1.GatewayMessage, 100),
		RequestChan:   make(chan *vpsgatewayv1.GatewayRequest, 100),
		RequestMap:    make(map[string]chan *vpsgatewayv1.GatewayResponse),
	}

	r.gateways[gatewayID] = conn
	logger.Info("[GatewayRegistry] Gateway %s registered (version: %s, IP: %s)", gatewayID, version, gatewayIP)

	return conn, nil
}

// UnregisterGateway removes a gateway connection
func (r *GatewayRegistry) UnregisterGateway(gatewayID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conn, ok := r.gateways[gatewayID]; ok {
		close(conn.MessageChan)
		close(conn.RequestChan)
		delete(r.gateways, gatewayID)
		delete(r.gatewayMetrics, gatewayID)
		logger.Info("[GatewayRegistry] Gateway %s unregistered", gatewayID)
	}
}

// GetGateway returns a gateway connection by ID
func (r *GatewayRegistry) GetGateway(gatewayID string) (*GatewayConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.gateways[gatewayID]
	return conn, ok
}

// GetAnyGateway returns any connected gateway (useful when there's only one)
func (r *GatewayRegistry) GetAnyGateway() (*GatewayConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, conn := range r.gateways {
		return conn, true
	}
	return nil, false
}

// ListGateways returns all registered gateways
func (r *GatewayRegistry) ListGateways() []*GatewayConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gateways := make([]*GatewayConnection, 0, len(r.gateways))
	for _, conn := range r.gateways {
		gateways = append(gateways, conn)
	}
	return gateways
}

// UpdateHeartbeat updates the last heartbeat time for a gateway
func (c *GatewayConnection) UpdateHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastHeartbeat = time.Now()
}

// SendRequest sends an RPC request to the gateway and waits for a response
func (c *GatewayConnection) SendRequest(ctx context.Context, method string, req proto.Message) (proto.Message, error) {
	// Serialize request
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Generate request ID
	requestID := fmt.Sprintf("%s-%d", c.GatewayID, time.Now().UnixNano())

	// Create response channel
	responseChan := make(chan *vpsgatewayv1.GatewayResponse, 1)
	c.RequestMapLock.Lock()
	c.RequestMap[requestID] = responseChan
	c.RequestMapLock.Unlock()

	// Clean up response channel after request completes
	defer func() {
		c.RequestMapLock.Lock()
		delete(c.RequestMap, requestID)
		close(responseChan)
		c.RequestMapLock.Unlock()
	}()

	// Send request
	gatewayReq := &vpsgatewayv1.GatewayRequest{
		RequestId: requestID,
		Method:    method,
		Payload:   reqBytes,
	}

	select {
	case c.RequestChan <- gatewayReq:
		// Request sent
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for response
	select {
	case resp := <-responseChan:
		if !resp.Success {
			return nil, fmt.Errorf("gateway error: %s", resp.Error)
		}

		// Deserialize response based on method
		var respMsg proto.Message
		switch method {
		case "AllocateIP":
			respMsg = &vpsgatewayv1.AllocateIPResponse{}
		case "ReleaseIP":
			respMsg = &vpsgatewayv1.ReleaseIPResponse{}
		case "ListIPs":
			respMsg = &vpsgatewayv1.ListIPsResponse{}
		case "GetGatewayInfo":
			respMsg = &vpsgatewayv1.GetGatewayInfoResponse{}
		default:
			return nil, fmt.Errorf("unknown method: %s", method)
		}

		if err := proto.Unmarshal(resp.Payload, respMsg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return respMsg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleResponse handles a response from the gateway
func (c *GatewayConnection) HandleResponse(resp *vpsgatewayv1.GatewayResponse) {
	c.RequestMapLock.RLock()
	responseChan, ok := c.RequestMap[resp.RequestId]
	c.RequestMapLock.RUnlock()

	if ok {
		select {
		case responseChan <- resp:
		default:
			logger.Warn("[GatewayRegistry] Response channel full for request %s", resp.RequestId)
		}
	} else {
		logger.Warn("[GatewayRegistry] No pending request found for response ID %s", resp.RequestId)
	}
}

// ProcessMetrics processes metrics from the gateway and forwards them to Prometheus
func (r *GatewayRegistry) ProcessMetrics(gatewayID string, metricsText string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store metrics text for this gateway
	// The /metrics endpoint will append these metrics to the response
	r.gatewayMetrics[gatewayID] = metricsText
	logger.Debug("[GatewayRegistry] Updated metrics for gateway %s (%d bytes)", gatewayID, len(metricsText))
}

// GetGatewayMetrics returns all stored gateway metrics as a single text block
func (r *GatewayRegistry) GetGatewayMetrics() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.gatewayMetrics) == 0 {
		return ""
	}

	var buf strings.Builder
	for gatewayID, metricsText := range r.gatewayMetrics {
		// Add a comment to identify which gateway these metrics are from
		buf.WriteString(fmt.Sprintf("# Gateway: %s\n", gatewayID))
		buf.WriteString(metricsText)
		buf.WriteString("\n")
	}

	return buf.String()
}
