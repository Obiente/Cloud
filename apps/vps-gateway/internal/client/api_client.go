package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/redis"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// APIConnection represents a connection to a single API instance
type APIConnection struct {
	client        vpsgatewayv1connect.VPSGatewayServiceClient
	apiURL        string
	apiInstanceID string
	ctx           context.Context
	cancel        context.CancelFunc
	connected     bool
	mu            sync.RWMutex
}

// APIClient handles connections to all API servers
type APIClient struct {
	connections map[string]*APIConnection // apiInstanceID -> connection
	apiSecret   string
	gatewayID   string
	version     string
	dhcpManager *dhcp.Manager
	redisClient *redis.Client
	mu          sync.RWMutex
}

// NewAPIClient creates a new API client that connects to all API instances
func NewAPIClient(dhcpManager *dhcp.Manager) (*APIClient, error) {
	apiSecret := os.Getenv("GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("GATEWAY_API_SECRET environment variable is required")
	}

	// Get gateway ID (use hostname or generate UUID)
	gatewayID, _ := os.Hostname()
	if gatewayID == "" {
		gatewayID = fmt.Sprintf("gateway-%d", time.Now().Unix())
	}

	// Initialize Redis client (optional - for API instance discovery)
	var redisClient *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		client, err := redis.NewClient()
		if err != nil {
			logger.Warn("[APIClient] Failed to connect to Redis: %v (will use GATEWAY_API_URL fallback)", err)
		} else {
			redisClient = client
		}
	}

	return &APIClient{
		connections: make(map[string]*APIConnection),
		apiSecret:   apiSecret,
		gatewayID:   gatewayID,
		version:     "1.0.0", // TODO: Get from build info
		dhcpManager: dhcpManager,
		redisClient: redisClient,
	}, nil
}

// apiAuthInterceptor adds the API secret header to requests
type apiAuthInterceptor struct {
	secret string
}

func newAPIAuthInterceptor(secret string) connect.Interceptor {
	return &apiAuthInterceptor{secret: secret}
}

func (i *apiAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-api-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *apiAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("x-api-secret", i.secret)
		return next(ctx, req)
	}
}

func (i *apiAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("x-api-secret", i.secret)
		return conn
	}
}

func (i *apiAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// Connect connects to all API instances and maintains bidirectional streams
func (c *APIClient) Connect(ctx context.Context) error {
	// Start discovery loop to find and connect to all API instances
	go c.discoverAndConnectLoop(ctx)

	// Keep running
	<-ctx.Done()
	return ctx.Err()
}

// discoverAndConnectLoop continuously discovers API instances and connects to them
func (c *APIClient) discoverAndConnectLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Discover every 10 seconds
	defer ticker.Stop()

	// Initial discovery
	c.discoverAndConnect(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.discoverAndConnect(ctx)
		}
	}
}

// discoverAndConnect discovers API instances from Redis and connects to them
func (c *APIClient) discoverAndConnect(ctx context.Context) {
	var apiInstances []APInstanceInfo

	// Try to discover from Redis
	if c.redisClient != nil {
		instances, err := c.discoverAPIInstancesFromRedis(ctx)
		if err != nil {
			logger.Debug("[APIClient] Failed to discover API instances from Redis: %v", err)
		} else {
			apiInstances = instances
		}
	}

	// Fallback to GATEWAY_API_URL if Redis discovery failed
	if len(apiInstances) == 0 {
		apiURL := os.Getenv("GATEWAY_API_URL")
		if apiURL != "" {
			apiInstances = []APInstanceInfo{
				{
					InstanceID: "default",
					APIURL:     apiURL,
				},
			}
			logger.Info("[APIClient] Using GATEWAY_API_URL fallback: %s", apiURL)
		} else {
			logger.Warn("[APIClient] No API instances discovered and GATEWAY_API_URL not set")
			return
		}
	}

	logger.Info("[APIClient] Discovered %d API instance(s)", len(apiInstances))

	// Deduplicate by URL (multiple instances may register with same service name)
	// This is necessary because in Swarm, all instances may register with 'http://api:3001'
	// and we only want one connection per unique URL
	seenURLs := make(map[string]bool)
	uniqueInstances := make([]APInstanceInfo, 0)

	for _, instance := range apiInstances {
		if !seenURLs[instance.APIURL] {
			seenURLs[instance.APIURL] = true
			uniqueInstances = append(uniqueInstances, instance)
		} else {
			logger.Debug("[APIClient] Skipping duplicate URL: %s (instance: %s)", instance.APIURL, instance.InstanceID)
		}
	}

	logger.Info("[APIClient] Connecting to %d unique API URL(s) (deduplicated from %d instances)", len(uniqueInstances), len(apiInstances))

	// Connect to all unique URLs
	for _, instance := range uniqueInstances {
		c.mu.RLock()
		// Check if we already have a connection to this URL (by checking any connection with same URL)
		alreadyConnected := false
		for _, conn := range c.connections {
			if conn.apiURL == instance.APIURL && conn.isConnected() {
				alreadyConnected = true
				break
			}
		}
		c.mu.RUnlock()

		if alreadyConnected {
			logger.Debug("[APIClient] Already connected to %s, skipping", instance.APIURL)
			continue
		}

		// Start connection in goroutine
		go c.connectToAPIInstance(ctx, instance)
	}
}

// APInstanceInfo represents an API instance
type APInstanceInfo struct {
	InstanceID string
	APIURL     string
}

// discoverAPIInstancesFromRedis discovers API instances from Redis
func (c *APIClient) discoverAPIInstancesFromRedis(ctx context.Context) ([]APInstanceInfo, error) {
	keys, err := c.redisClient.Keys(ctx, "api:instance:*")
	if err != nil {
		return nil, err
	}

	instances := make([]APInstanceInfo, 0, len(keys))
	for _, key := range keys {
		data, err := c.redisClient.Get(ctx, key)
		if err != nil {
			logger.Debug("[APIClient] Failed to get API instance info for %s: %v", key, err)
			continue
		}

		var instanceInfo map[string]interface{}
		if err := json.Unmarshal([]byte(data), &instanceInfo); err != nil {
			logger.Debug("[APIClient] Failed to unmarshal API instance info: %v", err)
			continue
		}

		instanceID, _ := instanceInfo["instance_id"].(string)
		apiURL, _ := instanceInfo["api_url"].(string)

		if instanceID != "" && apiURL != "" {
			instances = append(instances, APInstanceInfo{
				InstanceID: instanceID,
				APIURL:     apiURL,
			})
		}
	}

	return instances, nil
}

// connectToAPIInstance connects to a single API instance
func (c *APIClient) connectToAPIInstance(ctx context.Context, instance APInstanceInfo) {
	// Create connection context
	connCtx, cancel := context.WithCancel(ctx)

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Connect client with auth interceptor
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		instance.APIURL,
		connect.WithInterceptors(newAPIAuthInterceptor(c.apiSecret)),
	)

	// Create connection object
	conn := &APIConnection{
		client:        client,
		apiURL:        instance.APIURL,
		apiInstanceID: instance.InstanceID,
		ctx:           connCtx,
		cancel:        cancel,
		connected:     false,
	}

	// Store connection
	c.mu.Lock()
	c.connections[instance.InstanceID] = conn
	c.mu.Unlock()

	// Update connection status in Redis
	c.updateConnectionStatus(ctx, instance.InstanceID, false)

	// Connect and maintain stream
	for {
		logger.Info("[APIClient] Connecting to API instance %s at %s", instance.InstanceID, instance.APIURL)

		// Create bidirectional stream
		stream := client.RegisterGateway(connCtx)

		// Get DHCP configuration for registration
		poolStart, poolEnd, subnetMask, gateway, _ := c.dhcpManager.GetConfig()

		// Send registration message
		regMsg := &vpsgatewayv1.GatewayMessage{
			Type: "register",
			Registration: &vpsgatewayv1.GatewayRegistration{
				GatewayId:     c.gatewayID,
				Version:       c.version,
				GatewayIp:     gateway,
				DhcpPoolStart: poolStart,
				DhcpPoolEnd:   poolEnd,
				SubnetMask:    subnetMask,
				GatewayIpDhcp: gateway,
			},
		}

		if err := stream.Send(regMsg); err != nil {
			logger.Error("[APIClient] Failed to send registration to %s at %s: %v", instance.InstanceID, instance.APIURL, err)
			logger.Debug("[APIClient] This may be normal if the API instance is on a different network (overlay network IP not reachable from gateway host network)")
			conn.setConnected(false)
			c.updateConnectionStatus(ctx, instance.InstanceID, false)
			time.Sleep(5 * time.Second)
			continue
		}

		// Mark as connected
		conn.setConnected(true)
		c.updateConnectionStatus(ctx, instance.InstanceID, true)
		logger.Info("[APIClient] Successfully connected to API instance %s", instance.InstanceID)

		// Start goroutine to send metrics periodically
		go c.sendMetricsLoop(connCtx, stream, instance.InstanceID)

		// Start goroutine to send heartbeats
		go c.sendHeartbeatLoop(connCtx, stream, instance.InstanceID)

		// Start goroutine to push leases periodically
		go c.sendLeasesLoop(connCtx, stream, instance.InstanceID)

		// Handle incoming messages from API
		for {
			msg, err := stream.Receive()
			if err == io.EOF {
				logger.Info("[APIClient] API instance %s closed connection, reconnecting in 5 seconds...", instance.InstanceID)
				conn.setConnected(false)
				c.updateConnectionStatus(ctx, instance.InstanceID, false)
				time.Sleep(5 * time.Second)
				break // Break inner loop to reconnect
			}
			if err != nil {
				logger.Error("[APIClient] Error receiving from API instance %s: %v, reconnecting in 5 seconds...", instance.InstanceID, err)
				conn.setConnected(false)
				c.updateConnectionStatus(ctx, instance.InstanceID, false)
				time.Sleep(5 * time.Second)
				break // Break inner loop to reconnect
			}

			switch msg.Type {
			case "registered":
				logger.Info("[APIClient] Successfully registered with API instance %s", instance.InstanceID)

			case "request":
				if msg.Request != nil {
					go c.handleRequest(connCtx, stream, msg.Request)
				}

			default:
				logger.Warn("[APIClient] Unknown message type from %s: %s", instance.InstanceID, msg.Type)
			}
		}
	}
}

// updateConnectionStatus updates connection status in Redis
func (c *APIClient) updateConnectionStatus(ctx context.Context, apiInstanceID string, connected bool) {
	if c.redisClient == nil {
		return
	}

	status := map[string]interface{}{
		"gateway_id":      c.gatewayID,
		"api_instance_id": apiInstanceID,
		"connected":       connected,
		"updated_at":      time.Now(),
	}

	key := fmt.Sprintf("gateway:connection:%s:%s", c.gatewayID, apiInstanceID)
	if err := c.redisClient.Set(ctx, key, status, 60*time.Second); err != nil {
		logger.Debug("[APIClient] Failed to update connection status in Redis: %v", err)
	}
}

// isConnected returns whether the connection is currently connected
func (c *APIConnection) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// setConnected sets the connection status
func (c *APIConnection) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = connected
}

// sendMetricsLoop sends Prometheus metrics to the API periodically
func (c *APIClient) sendMetricsLoop(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], apiInstanceID string) {
	ticker := time.NewTicker(15 * time.Second) // Send metrics every 15 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get metrics in Prometheus text format
			metricsText, err := metrics.GetMetricsText()
			if err != nil {
				logger.Error("[APIClient] Failed to get metrics: %v", err)
				continue
			}

			msg := &vpsgatewayv1.GatewayMessage{
				Type:    "metrics",
				Metrics: metricsText,
			}

			if err := stream.Send(msg); err != nil {
				logger.Error("[APIClient] Failed to send metrics: %v", err)
				return
			}
		}
	}
}

// sendHeartbeatLoop sends heartbeat messages to keep connection alive
func (c *APIClient) sendHeartbeatLoop(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], apiInstanceID string) {
	ticker := time.NewTicker(30 * time.Second) // Send heartbeat every 30 seconds
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
				logger.Error("[APIClient] Failed to send heartbeat: %v", err)
				return
			}
		}
	}
}

// sendLeasesLoop periodically pushes active DHCP leases to the API over the bidi stream
func (c *APIClient) sendLeasesLoop(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage], apiInstanceID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	sendNow := func() {
		leases, err := c.dhcpManager.GetActiveLeases()
		if err != nil {
			logger.Debug("[APIClient] Failed to get active leases: %v", err)
			return
		}

		protoLeases := make([]*vpsgatewayv1.LeaseRecord, 0, len(leases))
		for _, lease := range leases {
			protoLeases = append(protoLeases, &vpsgatewayv1.LeaseRecord{
				MacAddress: lease.MAC,
				IpAddress:  lease.IP.String(),
				Hostname:   lease.Hostname,
				ExpiresAt:  timestamppb.New(lease.ExpiresAt),
			})
		}

		payloadResp := &vpsgatewayv1.GetLeasesResponse{Leases: protoLeases}
		payloadBytes, err := proto.Marshal(payloadResp)
		if err != nil {
			logger.Error("[APIClient] Failed to marshal leases payload: %v", err)
			return
		}

		req := &vpsgatewayv1.GatewayRequest{
			RequestId: fmt.Sprintf("%s-pushleases-%d", c.gatewayID, time.Now().UnixNano()),
			Method:    "PushLeases",
			Payload:   payloadBytes,
		}

		msg := &vpsgatewayv1.GatewayMessage{Type: "request", Request: req}
		if err := stream.Send(msg); err != nil {
			logger.Error("[APIClient] Failed to send PushLeases to %s: %v", apiInstanceID, err)
			return
		}
		logger.Debug("[APIClient] Sent PushLeases (%d leases) to API %s", len(protoLeases), apiInstanceID)
	}

	// Send immediately once on connect
	sendNow()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendNow()
		}
	}
}

// handleRequest handles an RPC request from the API
func (c *APIClient) handleRequest(
	ctx context.Context,
	stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage],
	req *vpsgatewayv1.GatewayRequest,
) {
	var resp proto.Message
	var err error

	switch req.Method {
	case "AllocateIP":
		var allocReq vpsgatewayv1.AllocateIPRequest
		if unmarshalErr := proto.Unmarshal(req.Payload, &allocReq); unmarshalErr != nil {
			resp = &vpsgatewayv1.AllocateIPResponse{}
			err = fmt.Errorf("failed to unmarshal request: %w", unmarshalErr)
		} else {
			// Call DHCP manager
			alloc, allocErr := c.dhcpManager.AllocateIP(
				ctx,
				allocReq.VpsId,
				allocReq.OrganizationId,
				allocReq.MacAddress,
				allocReq.PreferredIp,
				false, // allowPublicIP: false for regular DHCP allocations
			)
			if allocErr != nil {
				resp = &vpsgatewayv1.AllocateIPResponse{}
				err = allocErr
			} else {
				// Get config for subnet, gateway, DNS
				_, _, subnetMask, gatewayIP, dnsServers := c.dhcpManager.GetConfig()
				resp = &vpsgatewayv1.AllocateIPResponse{
					IpAddress:    alloc.IPAddress.String(),
					SubnetMask:   subnetMask,
					Gateway:      gatewayIP,
					DnsServers:   dnsServers,
					LeaseExpires: timestamppb.New(alloc.LeaseExpires),
				}
			}
		}

	case "ReleaseIP":
		var releaseReq vpsgatewayv1.ReleaseIPRequest
		if unmarshalErr := proto.Unmarshal(req.Payload, &releaseReq); unmarshalErr != nil {
			err = fmt.Errorf("failed to unmarshal request: %w", unmarshalErr)
		} else {
			releaseErr := c.dhcpManager.ReleaseIP(ctx, releaseReq.VpsId, releaseReq.IpAddress)
			if releaseErr != nil {
				err = fmt.Errorf("failed to release IP: %w", releaseErr)
			}
		}
		resp = &vpsgatewayv1.ReleaseIPResponse{
			Success: err == nil,
			Message: func() string {
				if err != nil {
					return err.Error()
				}
				return "IP released successfully"
			}(),
		}

	case "ListIPs":
		var listReq vpsgatewayv1.ListIPsRequest
		if unmarshalErr := proto.Unmarshal(req.Payload, &listReq); unmarshalErr != nil {
			resp = &vpsgatewayv1.ListIPsResponse{}
			err = fmt.Errorf("failed to unmarshal request: %w", unmarshalErr)
		} else {
			allocations, listErr := c.dhcpManager.ListIPs(ctx, listReq.OrganizationId, listReq.VpsId)
			if listErr != nil {
				resp = &vpsgatewayv1.ListIPsResponse{}
				err = listErr
			} else {
				protoAllocs := make([]*vpsgatewayv1.IPAllocation, len(allocations))
				for i, alloc := range allocations {
					protoAllocs[i] = &vpsgatewayv1.IPAllocation{
						VpsId:          alloc.VPSID,
						OrganizationId: alloc.OrganizationID,
						IpAddress:      alloc.IPAddress.String(),
						MacAddress:     alloc.MACAddress,
						AllocatedAt:    timestamppb.New(alloc.AllocatedAt),
						LeaseExpires:   timestamppb.New(alloc.LeaseExpires),
					}
				}
				resp = &vpsgatewayv1.ListIPsResponse{
					Allocations: protoAllocs,
				}
			}
		}

	case "GetGatewayInfo":
		poolStart, poolEnd, subnetMask, gatewayIP, dnsServers := c.dhcpManager.GetConfig()
		totalIPs, allocatedIPs, dhcpStatus := c.dhcpManager.GetStats()
		resp = &vpsgatewayv1.GetGatewayInfoResponse{
			Version:        c.version,
			DhcpPoolStart:  poolStart,
			DhcpPoolEnd:    poolEnd,
			SubnetMask:     subnetMask,
			GatewayIp:      gatewayIP,
			DnsServers:     dnsServers,
			TotalIps:       int32(totalIPs),
			AllocatedIps:   int32(allocatedIPs),
			DhcpStatus:     dhcpStatus,
			SshProxyStatus: "running", // TODO: Get from SSH proxy
		}

	default:
		err = fmt.Errorf("unknown method: %s", req.Method)
		resp = nil
	}

	// Serialize response
	var respBytes []byte
	if resp != nil {
		respBytes, err = proto.Marshal(resp)
		if err != nil {
			logger.Error("[APIClient] Failed to marshal response: %v", err)
			return
		}
	}

	// Send response
	responseMsg := &vpsgatewayv1.GatewayMessage{
		Type: "response",
		Response: &vpsgatewayv1.GatewayResponse{
			RequestId: req.RequestId,
			Success:   err == nil,
			Payload:   respBytes,
			Error: func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		},
	}

	if err := stream.Send(responseMsg); err != nil {
		logger.Error("[APIClient] Failed to send response: %v", err)
	}
}
