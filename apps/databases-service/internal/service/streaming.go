package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	orchestrator "github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const databaseStatusStreamPollInterval = 1500 * time.Millisecond

// StreamDatabaseStatus streams database status updates from shared state.
func (s *Service) StreamDatabaseStatus(ctx context.Context, req *connect.Request[databasesv1.StreamDatabaseStatusRequest], stream *connect.ServerStream[databasesv1.DatabaseStatusUpdate]) error {
	// Ensure authenticated
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	databaseID := req.Msg.GetDatabaseId()
	ticker := time.NewTicker(databaseStatusStreamPollInterval)
	defer ticker.Stop()

	lastStatus := int32(-1)
	lastDeleted := false

	sendUpdate := func(dbInstance *database.DatabaseInstance, deleted bool) error {
		status := databasesv1.DatabaseStatus(dbInstance.Status)
		var message *string
		if deleted {
			status = databasesv1.DatabaseStatus_DELETED
			deletedMessage := "Database deleted"
			message = &deletedMessage
		}

		update := &databasesv1.DatabaseStatusUpdate{
			DatabaseId: databaseID,
			Status:     status,
			Message:    message,
			Timestamp:  timestamppb.Now(),
		}
		if err := stream.Send(update); err != nil {
			return err
		}

		lastStatus = dbInstance.Status
		lastDeleted = deleted
		return nil
	}

	for {
		dbInstance, err := s.repo.GetByIDIncludeDeleted(ctx, databaseID, true)
		if err != nil {
			if lastStatus == -1 {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return nil
		}

		deleted := dbInstance.DeletedAt != nil
		if dbInstance.Status != lastStatus || deleted != lastDeleted {
			if err := sendUpdate(dbInstance, deleted); err != nil {
				return err
			}
			if deleted {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// GetDatabaseMetrics retrieves historical database metrics
func (s *Service) GetDatabaseMetrics(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseMetricsRequest]) (*connect.Response[databasesv1.GetDatabaseMetricsResponse], error) {
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Use StartTime from request, default to last hour
	since := time.Now().Add(-1 * time.Hour)
	if req.Msg.GetStartTime() != nil {
		since = req.Msg.GetStartTime().AsTime()
	}

	rawMetrics, err := database.GetRecentDatabaseMetrics(ctx, req.Msg.GetDatabaseId(), since)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch metrics: %w", err))
	}

	protoMetrics := make([]*databasesv1.DatabaseMetric, 0, len(rawMetrics))
	for _, m := range rawMetrics {
		protoMetrics = append(protoMetrics, &databasesv1.DatabaseMetric{
			DatabaseId:      req.Msg.GetDatabaseId(),
			CpuUsagePercent: m.CPUUsage,
			MemoryUsedBytes: m.MemoryUsage,
			Timestamp:       timestamppb.New(m.Timestamp),
		})
	}

	return connect.NewResponse(&databasesv1.GetDatabaseMetricsResponse{
		Metrics: protoMetrics,
	}), nil
}

// StreamDatabaseMetrics streams real-time database metrics (placeholder)
func (s *Service) StreamDatabaseMetrics(ctx context.Context, req *connect.Request[databasesv1.StreamDatabaseMetricsRequest], stream *connect.ServerStream[databasesv1.DatabaseMetric]) error {
	// Ensure authenticated
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	databaseID := req.Msg.GetDatabaseId()

	// Try to get the global metrics streamer for live data
	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("metrics streaming not available"))
	}

	// Subscribe to live metrics
	metricsCh := metricsStreamer.Subscribe(databaseID)
	defer metricsStreamer.Unsubscribe(databaseID, metricsCh)

	for {
		select {
		case <-ctx.Done():
			return nil
		case liveMetric, ok := <-metricsCh:
			if !ok {
				return nil
			}
			if liveMetric.ResourceID != databaseID || liveMetric.ResourceType != "database" {
				continue
			}
			metric := &databasesv1.DatabaseMetric{
				DatabaseId:      databaseID,
				CpuUsagePercent: liveMetric.CPUUsage,
				MemoryUsedBytes: liveMetric.MemoryUsage,
				Timestamp:       timestamppb.New(liveMetric.Timestamp),
			}
			if err := stream.Send(metric); err != nil {
				return err
			}
		}
	}
}

// ListDatabaseSizes lists available database sizes/pricing.
func (s *Service) ListDatabaseSizes(ctx context.Context, req *connect.Request[databasesv1.ListDatabaseSizesRequest]) (*connect.Response[databasesv1.ListDatabaseSizesResponse], error) {
	res := connect.NewResponse(&databasesv1.ListDatabaseSizesResponse{
		Sizes: listDatabaseSizes(req.Msg.Type),
	})
	return res, nil
}
