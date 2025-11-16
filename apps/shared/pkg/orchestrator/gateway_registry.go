package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

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

// GatewayMetadata stores gateway metadata in Redis
type GatewayMetadata struct {
	GatewayID     string    `json:"gateway_id"`
	Version       string    `json:"version"`
	GatewayIP     string    `json:"gateway_ip"`
	RegisteredAt  time.Time `json:"registered_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	APIInstanceID string   `json:"api_instance_id"` // Which API instance has the connection
}

// GatewayRegistry manages connected gateways
type GatewayRegistry struct {
	gateways       map[string]*GatewayConnection
	gatewayMetrics map[string]string // gatewayID -> metrics text
	mu             sync.RWMutex
	apiSecret      string
	apiInstanceID  string // Unique ID for this API instance
	redisClient    *database.RedisCache
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
		
		// Get API instance ID (use hostname or generate)
		apiInstanceID := os.Getenv("HOSTNAME")
		if apiInstanceID == "" {
			apiInstanceID = fmt.Sprintf("api-%d", time.Now().Unix())
		}
		
		globalRegistry = NewGatewayRegistry(apiSecret, apiInstanceID)
	})
	return globalRegistry
}

// NewGatewayRegistry creates a new gateway registry
// Note: In forward connection pattern, API connects to gateway, so registry is mainly for tracking gateway metadata
func NewGatewayRegistry(apiSecret, apiInstanceID string) *GatewayRegistry {
	registry := &GatewayRegistry{
		gateways:       make(map[string]*GatewayConnection),
		gatewayMetrics: make(map[string]string),
		apiSecret:      apiSecret,
		apiInstanceID:  apiInstanceID,
		redisClient:    database.RedisClient,
	}
	
	// Start background task to sync gateway metadata from Redis (if needed)
	if registry.redisClient != nil {
		go registry.syncGatewayMetadata()
	}
	
	return registry
}

// registerAPIInstance is deprecated - no longer used in forward connection pattern
// API instances no longer need to register themselves since gateway is the server
// This method is kept for reference but is not called
func (r *GatewayRegistry) registerAPIInstance() {
	// No-op in forward connection pattern - API connects to gateway, not vice versa
}

// getContainerIP gets the container's own IP address from the Docker overlay network
// In Swarm, this should be the IP on the overlay network (not the host network)
func (r *GatewayRegistry) getContainerIP() string {
	// Try to get IP from environment variable first (can be set in docker-compose)
	if ip := os.Getenv("CONTAINER_IP"); ip != "" {
		return ip
	}
	
	// In Docker Swarm, the hostname resolves to the container's overlay network IP
	// This is the most reliable way to get the overlay network IP
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		addrs, err := net.LookupHost(hostname)
		if err == nil && len(addrs) > 0 {
			// In Swarm, hostname resolves to overlay network IP
			// This is the IP that other containers on the same overlay network can reach
			return addrs[0]
		}
	}
	
	// Fallback: Try to get IP from overlay network interface
	// Overlay networks typically use interfaces like 'eth0' or 'veth*'
	// Look for IPs that are likely from the Docker overlay network (10.x.x.x, 172.16-31.x.x)
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// Skip loopback and docker0 bridge
			if iface.Flags&net.FlagLoopback != 0 || iface.Name == "docker0" {
				continue
			}
			
			if iface.Flags&net.FlagUp != 0 {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok {
							ip := ipnet.IP
							if ip.To4() != nil && !ip.IsLoopback() {
								// Check if it's likely a Docker overlay network IP
								// Docker overlay networks typically use 10.x.x.x or 172.16-31.x.x
								if ip[0] == 10 || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) {
									return ip.String()
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Last resort: Get any non-loopback IPv4 address
	interfaces, err = net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
							if ipnet.IP.To4() != nil {
								// Return first IPv4 address
								return ipnet.IP.String()
							}
						}
					}
				}
			}
		}
	}
	
	return ""
}

// syncGatewayMetadata periodically syncs gateway metadata from Redis
func (r *GatewayRegistry) syncGatewayMetadata() {
	ticker := time.NewTicker(10 * time.Second) // Sync every 10 seconds
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		
		// Get all gateway keys from Redis
		// Format: gateway:metadata:{gatewayID}
		// We'll use a pattern to find all gateway metadata
		// For now, we'll just update heartbeats for known gateways
		
		r.mu.RLock()
		gatewayIDs := make([]string, 0, len(r.gateways))
		for id := range r.gateways {
			gatewayIDs = append(gatewayIDs, id)
		}
		r.mu.RUnlock()
		
		// Update heartbeats in Redis for our local connections
		for _, gatewayID := range gatewayIDs {
			r.mu.RLock()
			conn, ok := r.gateways[gatewayID]
			r.mu.RUnlock()
			
			if ok {
				metadata := GatewayMetadata{
					GatewayID:     conn.GatewayID,
					Version:       conn.Version,
					GatewayIP:     conn.GatewayIP,
					RegisteredAt:  conn.RegisteredAt,
					LastHeartbeat: conn.LastHeartbeat,
					APIInstanceID: r.apiInstanceID,
				}
				
				key := fmt.Sprintf("gateway:metadata:%s", gatewayID)
				if r.redisClient != nil {
					// Store with 30 second TTL (will be refreshed on heartbeat)
					r.redisClient.Set(ctx, key, metadata, 30*time.Second)
				}
			}
		}
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

	// Store metadata in Redis so other API instances know about this gateway
	if r.redisClient != nil {
		metadata := GatewayMetadata{
			GatewayID:     gatewayID,
			Version:       version,
			GatewayIP:     gatewayIP,
			RegisteredAt:  time.Now(),
			LastHeartbeat: time.Now(),
			APIInstanceID: r.apiInstanceID,
		}
		
		key := fmt.Sprintf("gateway:metadata:%s", gatewayID)
		ctx := context.Background()
		// Store with 30 second TTL (will be refreshed on heartbeat)
		if err := r.redisClient.Set(ctx, key, metadata, 30*time.Second); err != nil {
			logger.Warn("[GatewayRegistry] Failed to store gateway metadata in Redis: %v", err)
		}
	}

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
// First checks local registry, then Redis if no local gateway found
func (r *GatewayRegistry) GetAnyGateway() (*GatewayConnection, bool) {
	r.mu.RLock()
	// First, try local registry
	for _, conn := range r.gateways {
		r.mu.RUnlock()
		return conn, true
	}
	r.mu.RUnlock()
	
	// If no local gateway, check Redis for any gateway metadata
	// This allows API instances to know about gateways connected to other instances
	if r.redisClient != nil {
		// For now, we can't return a connection from Redis since the actual connection
		// is only with one API instance. But we can at least know that a gateway exists.
		// TODO: Implement request forwarding to the API instance that has the connection
		logger.Debug("[GatewayRegistry] No local gateway found, but gateways may exist on other API instances (check Redis)")
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

// UpdateHeartbeatWithRegistry updates heartbeat and refreshes Redis metadata
func (r *GatewayRegistry) UpdateHeartbeatWithRegistry(gatewayID string) {
	r.mu.RLock()
	conn, ok := r.gateways[gatewayID]
	r.mu.RUnlock()
	
	if !ok {
		return
	}
	
	conn.UpdateHeartbeat()
	
	// Refresh Redis metadata
	if r.redisClient != nil {
		metadata := GatewayMetadata{
			GatewayID:     conn.GatewayID,
			Version:       conn.Version,
			GatewayIP:     conn.GatewayIP,
			RegisteredAt:  conn.RegisteredAt,
			LastHeartbeat: conn.LastHeartbeat,
			APIInstanceID: r.apiInstanceID,
		}
		
		key := fmt.Sprintf("gateway:metadata:%s", gatewayID)
		ctx := context.Background()
		// Store with 30 second TTL (will be refreshed on next heartbeat)
		if err := r.redisClient.Set(ctx, key, metadata, 30*time.Second); err != nil {
			logger.Debug("[GatewayRegistry] Failed to update gateway heartbeat in Redis: %v", err)
		}
	}
}

// GetGatewayMetadataFromRedis retrieves gateway metadata from Redis
func (r *GatewayRegistry) GetGatewayMetadataFromRedis(ctx context.Context, gatewayID string) (*GatewayMetadata, error) {
	if r.redisClient == nil {
		return nil, fmt.Errorf("redis not available")
	}
	
	key := fmt.Sprintf("gateway:metadata:%s", gatewayID)
	data, err := r.redisClient.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	
	var metadata GatewayMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway metadata: %w", err)
	}
	
	return &metadata, nil
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
