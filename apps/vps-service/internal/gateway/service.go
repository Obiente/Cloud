package gateway

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	sharedorchestrator "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"
	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"
)

// Service implements a lightweight handler to accept gateway RegisterGateway streams
// and process incoming PushLeases messages.
type Service struct {
	vpsgatewayv1connect.UnimplementedVPSGatewayServiceHandler
	vpsManager *orchestrator.VPSManager
}

func NewService(vpsManager *orchestrator.VPSManager) *Service {
	return &Service{vpsManager: vpsManager}
}

// RegisterGateway accepts bidirectional gateway streams and handles PushLeases
func (s *Service) RegisterGateway(ctx context.Context, stream *connect.BidiStream[vpsgatewayv1.GatewayMessage, vpsgatewayv1.GatewayMessage]) error {
	var gatewayID string
	registry := sharedorchestrator.GetGlobalGatewayRegistry()

	logger.Info("[GatewayListener] New gateway connection (waiting for register message)")

	// Read messages and handle requests
	for {
		msg, err := stream.Receive()
		if err == io.EOF {
			logger.Info("[GatewayListener] connection closed by gateway %s", gatewayID)
			if gatewayID != "" {
				registry.UnregisterGateway(gatewayID)
			}
			return nil
		}
		if err != nil {
			logger.Error("[GatewayListener] receive error: %v", err)
			if gatewayID != "" {
				registry.UnregisterGateway(gatewayID)
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("receive error: %w", err))
		}

		switch msg.Type {
		case "register":
			if msg.Registration == nil {
				logger.Warn("[GatewayListener] register message missing payload")
				continue
			}
			reg := msg.Registration
			gatewayID = reg.GatewayId
			// Register in registry so other parts of API can send requests
			if _, err := registry.RegisterGateway(ctx, reg.GatewayId, reg.Version, reg.GatewayIp); err != nil {
				logger.Warn("[GatewayListener] failed to register gateway %s: %v", reg.GatewayId, err)
			} else {
				logger.Info("[GatewayListener] Gateway %s registered", reg.GatewayId)
			}

			// Send confirmation
			if err := stream.Send(&vpsgatewayv1.GatewayMessage{Type: "registered"}); err != nil {
				logger.Warn("[GatewayListener] failed to send registered confirmation: %v", err)
			}

		case "metrics":
			if gatewayID != "" {
				registry.ProcessMetrics(gatewayID, msg.Metrics)
			}

		case "heartbeat":
			if gatewayID != "" {
				registry.UpdateHeartbeatWithRegistry(gatewayID)
			}

		case "request":
			if msg.Request == nil {
				continue
			}
			req := msg.Request

			// Handle different request methods from gateways
			if req.Method == "PushLeases" {
				var leasesResp vpsgatewayv1.GetLeasesResponse
				if err := proto.Unmarshal(req.Payload, &leasesResp); err != nil {
					logger.Warn("[GatewayListener] failed to unmarshal PushLeases payload: %v", err)
					// send error response
					_ = stream.Send(&vpsgatewayv1.GatewayMessage{Type: "response", Response: &vpsgatewayv1.GatewayResponse{RequestId: req.RequestId, Success: false, Error: err.Error()}})
					continue
				}

				// Upsert each lease using VPSManager.RegisterLease
				for _, lease := range leasesResp.Leases {
					if lease == nil {
						continue
					}

					// Fill defaults where necessary
					expires := lease.ExpiresAt
					if expires == nil || expires.AsTime().IsZero() {
						expires = timestamppb.New(time.Now().Add(24 * time.Hour))
					}

					// LeaseRecord from gateway only contains MAC/IP/Hostname/ExpiresAt.
					// Try to resolve VPS ID and Organization using any available hints:
					// 1) Existing DHCP lease in DB (match by MAC or IP)
					// 2) Public IP assignment (VPSPublicIP table)
					vpsID := ""
					orgID := ""
					isPublic := false

					// 1) Try matching existing DHCPLease by MAC
					if mac := lease.GetMacAddress(); mac != "" {
						var existing database.DHCPLease
						if err := database.DB.WithContext(ctx).Where("mac_address = ?", mac).First(&existing).Error; err == nil {
							vpsID = existing.VPSID
							orgID = existing.OrganizationID
							isPublic = existing.IsPublic
						}
					}

					// 2) If not found by MAC, try matching by IP address
					if vpsID == "" && lease.GetIpAddress() != "" {
						var existing database.DHCPLease
						if err := database.DB.WithContext(ctx).Where("ip_address = ?", lease.GetIpAddress()).First(&existing).Error; err == nil {
							vpsID = existing.VPSID
							orgID = existing.OrganizationID
							isPublic = existing.IsPublic
						}
					}

					// 3) Fallback: check public IP assignments
					if vpsID == "" && lease.GetIpAddress() != "" {
						var pub database.VPSPublicIP
						if err := database.DB.WithContext(ctx).Where("ip_address = ?", lease.GetIpAddress()).First(&pub).Error; err == nil {
							if pub.VPSID != nil {
								vpsID = *pub.VPSID
							}
							if pub.OrganizationID != nil {
								orgID = *pub.OrganizationID
							}
							isPublic = true
						}
					}

					// If we still can't resolve a VPS or organization, skip this lease.
					if vpsID == "" || orgID == "" {
						logger.Debug("[GatewayListener] Skipping lease with missing org/VPS for IP %s (mac=%s)", lease.GetIpAddress(), lease.GetMacAddress())
						continue
					}

					registerReq := &vpsv1.RegisterLeaseRequest{
						VpsId:          vpsID,
						OrganizationId: orgID,
						MacAddress:     lease.GetMacAddress(),
						IpAddress:      lease.GetIpAddress(),
						ExpiresAt:      expires,
						IsPublic:       isPublic,
					}

					if s.vpsManager != nil {
						// Call RegisterLease but don't fail the whole batch on single errors
						if err := s.vpsManager.RegisterLease(ctx, registerReq, gatewayID); err != nil {
							logger.Warn("[GatewayListener] failed to register lease %s for VPS %s: %v", lease.GetIpAddress(), vpsID, err)
						}
					} else {
						logger.Debug("[GatewayListener] vpsManager not available, skipping lease upsert")
					}
				}

				// Acknowledge success
				_ = stream.Send(&vpsgatewayv1.GatewayMessage{Type: "response", Response: &vpsgatewayv1.GatewayResponse{RequestId: req.RequestId, Success: true}})

			} else if req.Method == "FindVPSByLease" {
				// DEPRECATED: This code path is NOT USED in production
				// This handles FindVPSByLease in the WRONG direction (Gateway→VPS inbound stream)
				// The actual implementation is in internal/gateway/handlers.go (VPS→Gateway outbound stream)
				// Architecture: VPS service connects TO gateways via bidirectional stream
				//              Gateways send requests over that stream, VPS responds
				//              Handlers in handlers.go process those requests
				// This RegisterGateway() endpoint exists for reverse connection (not currently used)
				
				logger.Warn("[GatewayListener] Received FindVPSByLease on deprecated inbound stream (ID: %s) - should use outbound stream handler instead", req.RequestId)
				
				_ = stream.Send(&vpsgatewayv1.GatewayMessage{
					Type: "response",
					Response: &vpsgatewayv1.GatewayResponse{
						RequestId: req.RequestId,
						Success:   false,
						Error:     "FindVPSByLease not supported on inbound stream - use VPS→Gateway connection",
					},
				})

			} else {
				// Unknown request method from gateway; respond with error
				_ = stream.Send(&vpsgatewayv1.GatewayMessage{Type: "response", Response: &vpsgatewayv1.GatewayResponse{RequestId: req.RequestId, Success: false, Error: fmt.Sprintf("unknown method: %s", req.Method)}})
			}

		case "response":
			// Responses from gateway correspond to API-initiated requests; route to registry so waiting callers get them
			if msg.Response != nil && gatewayID != "" {
				if conn, ok := registry.GetGateway(gatewayID); ok {
					conn.HandleResponse(msg.Response)
				}
			}

		default:
			logger.Debug("[GatewayListener] unknown gateway message type: %s", msg.Type)
		}
	}
}
