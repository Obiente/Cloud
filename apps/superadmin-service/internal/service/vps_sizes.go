package superadmin

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ListVPSSizes lists all VPS sizes in the catalog (superadmin only)
func (s *Service) ListVPSSizes(ctx context.Context, req *connect.Request[superadminv1.ListVPSSizesRequest]) (*connect.Response[superadminv1.ListVPSSizesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_sizes.read") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	region := req.Msg.GetRegion()
	includeUnavailable := req.Msg.GetIncludeUnavailable()

	sizes, err := database.ListAllVPSSizeCatalog(region, includeUnavailable)
	if err != nil {
		logger.Error("[SuperAdmin] Failed to list VPS sizes: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS sizes: %w", err))
	}

	protoSizes := make([]*commonv1.VPSSize, 0, len(sizes))
	for _, size := range sizes {
		sizeProto := &commonv1.VPSSize{
			Id:                  size.ID,
			Name:                size.Name,
			CpuCores:            size.CPUCores,
			MemoryBytes:         size.MemoryBytes,
			DiskBytes:           size.DiskBytes,
			BandwidthBytesMonth: size.BandwidthBytesMonth,
			MinimumPaymentCents: size.MinimumPaymentCents,
			Available:           size.Available,
			Region:              size.Region,
			CreatedAt:           timestamppb.New(size.CreatedAt),
			UpdatedAt:           timestamppb.New(size.UpdatedAt),
		}
		if size.Description != "" {
			sizeProto.Description = &size.Description
		}
		protoSizes = append(protoSizes, sizeProto)
	}

	return connect.NewResponse(&superadminv1.ListVPSSizesResponse{
		Sizes: protoSizes,
	}), nil
}

// CreateVPSSize creates a new VPS size in the catalog (superadmin only)
func (s *Service) CreateVPSSize(ctx context.Context, req *connect.Request[superadminv1.CreateVPSSizeRequest]) (*connect.Response[superadminv1.CreateVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_sizes.create") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Validate required fields
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if req.Msg.GetCpuCores() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cpu_cores must be greater than 0"))
	}
	if req.Msg.GetMemoryBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("memory_bytes must be greater than 0"))
	}
	if req.Msg.GetDiskBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("disk_bytes must be greater than 0"))
	}

	// Check if size already exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("VPS size with id %s already exists", req.Msg.GetId()))
	} else if err != gorm.ErrRecordNotFound {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing size: %w", err))
	}

	// Create new size
	size := &database.VPSSizeCatalog{
		ID:                  req.Msg.GetId(),
		Name:                req.Msg.GetName(),
		Description:         req.Msg.GetDescription(),
		CPUCores:            req.Msg.GetCpuCores(),
		MemoryBytes:         req.Msg.GetMemoryBytes(),
		DiskBytes:           req.Msg.GetDiskBytes(),
		BandwidthBytesMonth: req.Msg.GetBandwidthBytesMonth(),
		MinimumPaymentCents: req.Msg.GetMinimumPaymentCents(),
		Available:           req.Msg.GetAvailable(),
		Region:              req.Msg.GetRegion(),
	}

	if err := database.CreateVPSSizeCatalog(size); err != nil {
		logger.Error("[SuperAdmin] Failed to create VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS size: %w", err))
	}

	sizeProto := &commonv1.VPSSize{
		Id:                  size.ID,
		Name:                size.Name,
		CpuCores:            size.CPUCores,
		MemoryBytes:         size.MemoryBytes,
		DiskBytes:           size.DiskBytes,
		BandwidthBytesMonth: size.BandwidthBytesMonth,
		MinimumPaymentCents: size.MinimumPaymentCents,
		Available:           size.Available,
		Region:              size.Region,
		CreatedAt:           timestamppb.New(size.CreatedAt),
		UpdatedAt:           timestamppb.New(size.UpdatedAt),
	}
	if size.Description != "" {
		sizeProto.Description = &size.Description
	}
	return connect.NewResponse(&superadminv1.CreateVPSSizeResponse{
		Size: sizeProto,
	}), nil
}

// UpdateVPSSize updates an existing VPS size in the catalog (superadmin only)
func (s *Service) UpdateVPSSize(ctx context.Context, req *connect.Request[superadminv1.UpdateVPSSizeRequest]) (*connect.Response[superadminv1.UpdateVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_sizes.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Check if size exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS size with id %s not found", req.Msg.GetId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find VPS size: %w", err))
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.Msg.Name != nil {
		updates["name"] = req.Msg.GetName()
	}
	if req.Msg.Description != nil {
		updates["description"] = req.Msg.GetDescription()
	}
	if req.Msg.CpuCores != nil {
		if req.Msg.GetCpuCores() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cpu_cores must be greater than 0"))
		}
		updates["cpu_cores"] = req.Msg.GetCpuCores()
	}
	if req.Msg.MemoryBytes != nil {
		if req.Msg.GetMemoryBytes() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("memory_bytes must be greater than 0"))
		}
		updates["memory_bytes"] = req.Msg.GetMemoryBytes()
	}
	if req.Msg.DiskBytes != nil {
		if req.Msg.GetDiskBytes() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("disk_bytes must be greater than 0"))
		}
		updates["disk_bytes"] = req.Msg.GetDiskBytes()
	}
	if req.Msg.BandwidthBytesMonth != nil {
		updates["bandwidth_bytes_month"] = req.Msg.GetBandwidthBytesMonth()
	}
	if req.Msg.MinimumPaymentCents != nil {
		updates["minimum_payment_cents"] = req.Msg.GetMinimumPaymentCents()
	}
	if req.Msg.Available != nil {
		updates["available"] = req.Msg.GetAvailable()
	}
	if req.Msg.Region != nil {
		updates["region"] = req.Msg.GetRegion()
	}

	if len(updates) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no fields to update"))
	}

	// Update size
	if err := database.UpdateVPSSizeCatalog(req.Msg.GetId(), updates); err != nil {
		logger.Error("[SuperAdmin] Failed to update VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VPS size: %w", err))
	}

	// Fetch updated size
	var updated database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&updated).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch updated size: %w", err))
	}

	sizeProto := &commonv1.VPSSize{
		Id:                  updated.ID,
		Name:                updated.Name,
		CpuCores:            updated.CPUCores,
		MemoryBytes:         updated.MemoryBytes,
		DiskBytes:           updated.DiskBytes,
		BandwidthBytesMonth: updated.BandwidthBytesMonth,
		MinimumPaymentCents: updated.MinimumPaymentCents,
		Available:           updated.Available,
		Region:              updated.Region,
		CreatedAt:           timestamppb.New(updated.CreatedAt),
		UpdatedAt:           timestamppb.New(updated.UpdatedAt),
	}
	if updated.Description != "" {
		sizeProto.Description = &updated.Description
	}
	return connect.NewResponse(&superadminv1.UpdateVPSSizeResponse{
		Size: sizeProto,
	}), nil
}

// DeleteVPSSize deletes a VPS size from the catalog (superadmin only)
func (s *Service) DeleteVPSSize(ctx context.Context, req *connect.Request[superadminv1.DeleteVPSSizeRequest]) (*connect.Response[superadminv1.DeleteVPSSizeResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_sizes.delete") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Check if size exists
	var existing database.VPSSizeCatalog
	if err := database.DB.Where("id = ?", req.Msg.GetId()).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS size with id %s not found", req.Msg.GetId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find VPS size: %w", err))
	}

	// Check if any VPS instances are using this size
	var count int64
	if err := database.DB.Table("vps_instances").Where("size = ? AND deleted_at IS NULL", req.Msg.GetId()).Count(&count).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check VPS instances: %w", err))
	}
	if count > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete VPS size: %d VPS instances are using this size", count))
	}

	// Delete size
	if err := database.DeleteVPSSizeCatalog(req.Msg.GetId()); err != nil {
		logger.Error("[SuperAdmin] Failed to delete VPS size: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS size: %w", err))
	}

	return connect.NewResponse(&superadminv1.DeleteVPSSizeResponse{
		Success: true,
	}), nil
}
