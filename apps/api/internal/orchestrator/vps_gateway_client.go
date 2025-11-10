package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"api/internal/logger"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
)

// VPSGatewayClient handles communication with the vps-gateway service
type VPSGatewayClient struct {
	client    vpsgatewayv1connect.VPSGatewayServiceClient // nil if using registry
	apiSecret string
	baseURL   string
	registry  *GatewayRegistry // Used for reverse connection pattern
}

// NewVPSGatewayClient creates a new vps-gateway client
func NewVPSGatewayClient() (*VPSGatewayClient, error) {
	baseURL := os.Getenv("VPS_GATEWAY_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("VPS_GATEWAY_URL environment variable is required")
	}

	apiSecret := os.Getenv("VPS_GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("VPS_GATEWAY_API_SECRET environment variable is required")
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create Connect client
	client := vpsgatewayv1connect.NewVPSGatewayServiceClient(
		httpClient,
		baseURL,
		connect.WithInterceptors(newGatewayAuthInterceptor(apiSecret)),
	)

	return &VPSGatewayClient{
		client:    client,
		apiSecret: apiSecret,
		baseURL:   baseURL,
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
	// Use registry if available (reverse connection)
	if c.registry != nil {
		gatewayConn, ok := c.registry.GetAnyGateway()
		if !ok {
			return nil, fmt.Errorf("no gateway connected")
		}

		req := &vpsgatewayv1.AllocateIPRequest{
			VpsId:          vpsID,
			OrganizationId: organizationID,
			MacAddress:     macAddress,
		}

		resp, err := gatewayConn.SendRequest(ctx, "AllocateIP", req)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate IP from gateway: %w", err)
		}

		allocResp := resp.(*vpsgatewayv1.AllocateIPResponse)
		logger.Info("[VPSGateway] Allocated IP %s for VPS %s (org: %s)", allocResp.IpAddress, vpsID, organizationID)
		return allocResp, nil
	}

	// Legacy direct connection
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
