package gateway

import (
	"context"
	"fmt"
	"io"

	"api/internal/logger"
	"api/internal/orchestrator"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
)

// GatewayService handles gateway registration and communication
type GatewayService struct {
	vpsgatewayv1connect.UnimplementedVPSGatewayServiceHandler
	registry *orchestrator.GatewayRegistry
}

// NewGatewayService creates a new gateway service
func NewGatewayService(registry *orchestrator.GatewayRegistry) *GatewayService {
	return &GatewayService{
		registry: registry,
	}
}

// RegisterGateway handles gateway registration via bidirectional stream
func (s *GatewayService) RegisterGateway(
	ctx context.Context,
	stream *connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage],
) error {
	var gatewayConn *orchestrator.GatewayConnection
	var gatewayID string

	// Handle incoming messages from gateway
	go func() {
		for {
			msg, err := stream.Receive()
			if err == io.EOF {
				logger.Info("[GatewayService] Gateway %s disconnected", gatewayID)
				if gatewayConn != nil {
					s.registry.UnregisterGateway(gatewayID)
				}
				return
			}
			if err != nil {
				logger.Error("[GatewayService] Error receiving from gateway: %v", err)
				return
			}

			switch msg.Type {
			case "register":
				if msg.Registration == nil {
					logger.Error("[GatewayService] Registration message missing registration data")
					continue
				}
				reg := msg.Registration
				gatewayID = reg.GatewayId

				// Register gateway
				conn, err := s.registry.RegisterGateway(
					ctx,
					reg.GatewayId,
					reg.Version,
					reg.GatewayIp,
				)
				if err != nil {
					logger.Error("[GatewayService] Failed to register gateway %s: %v", reg.GatewayId, err)
					// Send error response
					_ = stream.Send(&vpsgatewayv1.GatewayMessage{
						Type: "error",
					})
					return
				}
				gatewayConn = conn

				// Send registration confirmation
				if err := stream.Send(&vpsgatewayv1.GatewayMessage{
					Type: "registered",
				}); err != nil {
					logger.Error("[GatewayService] Failed to send registration confirmation: %v", err)
					return
				}

				// Start forwarding messages from registry to gateway
				go s.forwardMessagesToGateway(stream, gatewayConn)

			case "metrics":
				if gatewayConn != nil {
					s.registry.ProcessMetrics(gatewayID, msg.Metrics)
				}

			case "response":
				if gatewayConn != nil && msg.Response != nil {
					gatewayConn.HandleResponse(msg.Response)
				}

			case "heartbeat":
				if gatewayConn != nil {
					// Update heartbeat and refresh Redis metadata
					s.registry.UpdateHeartbeatWithRegistry(gatewayID)
				}

			default:
				logger.Warn("[GatewayService] Unknown message type: %s", msg.Type)
			}
		}
	}()

	// Keep connection alive
	<-ctx.Done()
	return nil
}

// forwardMessagesToGateway forwards messages from the registry to the gateway
func (s *GatewayService) forwardMessagesToGateway(
	stream *connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage],
	conn *orchestrator.GatewayConnection,
) {
	for req := range conn.RequestChan {
		msg := &vpsgatewayv1.GatewayMessage{
			Type:    "request",
			Request: req,
		}
		if err := stream.Send(msg); err != nil {
			logger.Error("[GatewayService] Failed to send request to gateway: %v", err)
			return
		}
	}
}

// AllocateIP, ReleaseIP, etc. will be handled by the registry forwarding requests to the gateway
// These methods are kept for backward compatibility but will use the registry

func (s *GatewayService) AllocateIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.AllocateIPRequest],
) (*connect.Response[vpsgatewayv1.AllocateIPResponse], error) {
	// Get any connected gateway
	gatewayConn, ok := s.registry.GetAnyGateway()
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no gateway connected"))
	}

	// Send request through registry
	resp, err := gatewayConn.SendRequest(ctx, "AllocateIP", req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp.(*vpsgatewayv1.AllocateIPResponse)), nil
}

func (s *GatewayService) ReleaseIP(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.ReleaseIPRequest],
) (*connect.Response[vpsgatewayv1.ReleaseIPResponse], error) {
	gatewayConn, ok := s.registry.GetAnyGateway()
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no gateway connected"))
	}

	resp, err := gatewayConn.SendRequest(ctx, "ReleaseIP", req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp.(*vpsgatewayv1.ReleaseIPResponse)), nil
}

func (s *GatewayService) ListIPs(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.ListIPsRequest],
) (*connect.Response[vpsgatewayv1.ListIPsResponse], error) {
	gatewayConn, ok := s.registry.GetAnyGateway()
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no gateway connected"))
	}

	resp, err := gatewayConn.SendRequest(ctx, "ListIPs", req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp.(*vpsgatewayv1.ListIPsResponse)), nil
}

func (s *GatewayService) GetGatewayInfo(
	ctx context.Context,
	req *connect.Request[vpsgatewayv1.GetGatewayInfoRequest],
) (*connect.Response[vpsgatewayv1.GetGatewayInfoResponse], error) {
	gatewayConn, ok := s.registry.GetAnyGateway()
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no gateway connected"))
	}

	resp, err := gatewayConn.SendRequest(ctx, "GetGatewayInfo", req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp.(*vpsgatewayv1.GetGatewayInfoResponse)), nil
}
