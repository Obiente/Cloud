package gameservers

import (
	"context"
	"fmt"

	gameserversv1 "api/gen/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StreamGameServerStatus streams status updates for a game server
func (s *Service) StreamGameServerStatus(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerStatusRequest], stream *connect.ServerStream[gameserversv1.GameServerStatusUpdate]) error {
	// Ensure authenticated
	if _, err := s.ensureAuthenticated(ctx, req); err != nil {
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return err
	}

	// TODO: Implement actual streaming
	// For now, return current status
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	update := &gameserversv1.GameServerStatusUpdate{
		GameServerId: gameServerID,
		Status:       gameserversv1.GameServerStatus(gameServer.Status),
		Timestamp:    timestamppb.Now(),
	}

	return stream.Send(update)
}

// GetGameServerLogs retrieves logs for a game server
func (s *Service) GetGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerLogsRequest]) (*connect.Response[gameserversv1.GetGameServerLogsResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	// TODO: Implement log retrieval from Docker container
	res := connect.NewResponse(&gameserversv1.GetGameServerLogsResponse{
		Lines: []*gameserversv1.GameServerLogLine{},
	})
	return res, nil
}

// StreamGameServerLogs streams logs for a game server
func (s *Service) StreamGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerLogsRequest], stream *connect.ServerStream[gameserversv1.GameServerLogLine]) error {
	// Ensure authenticated
	if _, err := s.ensureAuthenticated(ctx, req); err != nil {
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return err
	}

	// TODO: Implement actual log streaming from Docker container
	return nil
}

// GetGameServerMetrics retrieves metrics for a game server
func (s *Service) GetGameServerMetrics(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerMetricsRequest]) (*connect.Response[gameserversv1.GetGameServerMetricsResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	// TODO: Implement metrics retrieval
	res := connect.NewResponse(&gameserversv1.GetGameServerMetricsResponse{
		Metrics: []*gameserversv1.GameServerMetric{},
	})
	return res, nil
}

// StreamGameServerMetrics streams real-time metrics for a game server
func (s *Service) StreamGameServerMetrics(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerMetricsRequest], stream *connect.ServerStream[gameserversv1.GameServerMetric]) error {
	// Ensure authenticated
	if _, err := s.ensureAuthenticated(ctx, req); err != nil {
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return err
	}

	// TODO: Implement metrics streaming
	return nil
}

// GetGameServerUsage retrieves aggregated usage for a game server
func (s *Service) GetGameServerUsage(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerUsageRequest]) (*connect.Response[gameserversv1.GetGameServerUsageResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	// TODO: Implement usage calculation similar to deployments
	res := connect.NewResponse(&gameserversv1.GetGameServerUsageResponse{
		GameServerId:      gameServerID,
		Month:             req.Msg.GetMonth(),
		CpuCoreSeconds:   0,
		MemoryByteSeconds: 0,
		BandwidthBytes:   0,
		StorageBytes:     0,
		CpuCostCents:     0,
		MemoryCostCents:  0,
		BandwidthCostCents: 0,
		StorageCostCents: 0,
		TotalCostCents:   0,
	})
	return res, nil
}

