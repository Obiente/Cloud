package databases

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StreamDatabaseStatus streams database status updates (placeholder)
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

	// Placeholder implementation - would poll database status and send updates
	// For now, just send current status once
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}

	update := &databasesv1.DatabaseStatusUpdate{
		DatabaseId: req.Msg.GetDatabaseId(),
		Status:     databasesv1.DatabaseStatus(dbInstance.Status),
		Timestamp:  timestamppb.Now(),
	}

	return stream.Send(update)
}

// GetDatabaseMetrics gets database metrics (placeholder)
func (s *Service) GetDatabaseMetrics(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseMetricsRequest]) (*connect.Response[databasesv1.GetDatabaseMetricsResponse], error) {
	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Placeholder implementation
	res := connect.NewResponse(&databasesv1.GetDatabaseMetricsResponse{
		Metrics: []*databasesv1.DatabaseMetric{},
	})
	return res, nil
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

	// Placeholder implementation
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("streaming metrics not yet implemented"))
}

// ListDatabaseSizes lists available database sizes/pricing (placeholder)
func (s *Service) ListDatabaseSizes(ctx context.Context, req *connect.Request[databasesv1.ListDatabaseSizesRequest]) (*connect.Response[databasesv1.ListDatabaseSizesResponse], error) {
	// Placeholder implementation - would return available sizes from catalog
	res := connect.NewResponse(&databasesv1.ListDatabaseSizesResponse{
		Sizes: []*databasesv1.DatabaseSize{
			{
				Id:                 "small",
				Name:               "Small",
				Type:               databasesv1.DatabaseType_POSTGRESQL,
				CpuCores:           1,
				MemoryBytes:        2147483648,  // 2GB
				DiskBytes:          10737418240, // 10GB
				MaxConnections:     100,
				PriceCentsPerMonth: 1000, // $10/month
			},
			{
				Id:                 "medium",
				Name:               "Medium",
				Type:               databasesv1.DatabaseType_POSTGRESQL,
				CpuCores:           2,
				MemoryBytes:        4294967296,  // 4GB
				DiskBytes:          53687091200, // 50GB
				MaxConnections:     200,
				PriceCentsPerMonth: 2000, // $20/month
			},
			{
				Id:                 "large",
				Name:               "Large",
				Type:               databasesv1.DatabaseType_POSTGRESQL,
				CpuCores:           4,
				MemoryBytes:        8589934592,   // 8GB
				DiskBytes:          107374182400, // 100GB
				MaxConnections:     500,
				PriceCentsPerMonth: 4000, // $40/month
			},
		},
	})
	return res, nil
}

