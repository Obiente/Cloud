package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// APIClient handles connection to the API server
type APIClient struct {
	client      vpsgatewayv1connect.VPSGatewayServiceClient
	apiURL      string
	apiSecret   string
	gatewayID   string
	version     string
	dhcpManager *dhcp.Manager
}

// NewAPIClient creates a new API client
func NewAPIClient(dhcpManager *dhcp.Manager) (*APIClient, error) {
	apiURL := os.Getenv("GATEWAY_API_URL")
	if apiURL == "" {
		return nil, fmt.Errorf("GATEWAY_API_URL environment variable is required")
	}

	apiSecret := os.Getenv("GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("GATEWAY_API_SECRET environment variable is required")
	}

	// Get gateway ID (use hostname or generate UUID)
	gatewayID, _ := os.Hostname()
	if gatewayID == "" {
		gatewayID = fmt.Sprintf("gateway-%d", time.Now().Unix())
	}

	// Get gateway IP (from DHCP config)
	gatewayIP := os.Getenv("GATEWAY_DHCP_GATEWAY")
	if gatewayIP == "" {
		gatewayIP = "10.15.3.1" // Default
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Connect client with auth interceptor
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		apiURL,
		connect.WithInterceptors(newAPIAuthInterceptor(apiSecret)),
	)

	return &APIClient{
		client:      client,
		apiURL:      apiURL,
		apiSecret:   apiSecret,
		gatewayID:   gatewayID,
		version:     "1.0.0", // TODO: Get from build info
		dhcpManager: dhcpManager,
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

// Connect connects to the API and maintains the bidirectional stream
func (c *APIClient) Connect(ctx context.Context) error {
	logger.Info("[APIClient] Connecting to API at %s", c.apiURL)

	// Create bidirectional stream
	stream := c.client.RegisterGateway(ctx)

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
		return fmt.Errorf("failed to send registration: %w", err)
	}

	// Start goroutine to send metrics periodically
	go c.sendMetricsLoop(ctx, stream)

	// Start goroutine to send heartbeats
	go c.sendHeartbeatLoop(ctx, stream)

	// Handle incoming messages from API
	for {
		msg, err := stream.Receive()
		if err == io.EOF {
			logger.Info("[APIClient] API closed connection")
			return nil
		}
		if err != nil {
			return fmt.Errorf("error receiving from API: %w", err)
		}

		switch msg.Type {
		case "registered":
			logger.Info("[APIClient] Successfully registered with API")

		case "request":
			if msg.Request != nil {
				go c.handleRequest(ctx, stream, msg.Request)
			}

		default:
			logger.Warn("[APIClient] Unknown message type: %s", msg.Type)
		}
	}
}

// sendMetricsLoop sends Prometheus metrics to the API periodically
func (c *APIClient) sendMetricsLoop(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]) {
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
func (c *APIClient) sendHeartbeatLoop(ctx context.Context, stream *connect.BidiStreamForClient[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]) {
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
