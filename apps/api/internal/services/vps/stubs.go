package vps

import (
	"context"
	"errors"
	"fmt"

	commonv1 "api/gen/proto/obiente/cloud/common/v1"
	vpsv1 "api/gen/proto/obiente/cloud/vps/v1"
	"api/internal/database"
	"api/internal/orchestrator"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ListVPSSizes returns available VPS sizes/pricing from catalog
func (s *Service) ListVPSSizes(ctx context.Context, req *connect.Request[vpsv1.ListAvailableVPSSizesRequest]) (*connect.Response[vpsv1.ListAvailableVPSSizesResponse], error) {
	region := req.Msg.GetRegion()

	catalogSizes, err := database.ListVPSSizeCatalog(region)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS sizes: %w", err))
	}

	sizes := make([]*commonv1.VPSSize, len(catalogSizes))
	for i, catalogSize := range catalogSizes {
		size := &commonv1.VPSSize{
			Id:                  catalogSize.ID,
			Name:                catalogSize.Name,
			CpuCores:            catalogSize.CPUCores,
			MemoryBytes:         catalogSize.MemoryBytes,
			DiskBytes:           catalogSize.DiskBytes,
			BandwidthBytesMonth: catalogSize.BandwidthBytesMonth,
			MinimumPaymentCents: catalogSize.MinimumPaymentCents,
			Available:           catalogSize.Available,
			Region:              catalogSize.Region,
		}
		if catalogSize.Description != "" {
			size.Description = &catalogSize.Description
		}
		sizes[i] = size
	}

	return connect.NewResponse(&vpsv1.ListAvailableVPSSizesResponse{
		Sizes: sizes,
	}), nil
}

// ListVPSRegions returns available VPS regions/locations from environment variables
func (s *Service) ListVPSRegions(ctx context.Context, req *connect.Request[vpsv1.ListVPSRegionsRequest]) (*connect.Response[vpsv1.ListVPSRegionsResponse], error) {
	// Get regions from environment variable (similar to TRAEFIK_IPS)
	envRegions, err := database.GetVPSRegionsFromEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse VPS regions from environment: %w", err))
	}

	regions := make([]*vpsv1.VPSRegion, len(envRegions))
	for i, envRegion := range envRegions {
		regions[i] = &vpsv1.VPSRegion{
			Id:        envRegion.ID,
			Name:      envRegion.Name,
			Available: envRegion.Available,
		}
	}

	return connect.NewResponse(&vpsv1.ListVPSRegionsResponse{
		Regions: regions,
	}), nil
}

// StreamVPSStatus streams VPS status updates
func (s *Service) StreamVPSStatus(ctx context.Context, req *connect.Request[vpsv1.StreamVPSStatusRequest], stream *connect.ServerStream[vpsv1.VPSStatusUpdate]) error {
	// TODO: Implement status streaming
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

// StreamVPSMetrics streams real-time VPS metrics
func (s *Service) StreamVPSMetrics(ctx context.Context, req *connect.Request[vpsv1.StreamVPSMetricsRequest], stream *connect.ServerStream[vpsv1.VPSMetric]) error {
	// TODO: Implement metrics streaming
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}

// GetVPSMetrics retrieves VPS instance metrics (real-time or historical)
func (s *Service) GetVPSMetrics(ctx context.Context, req *connect.Request[vpsv1.GetVPSMetricsRequest]) (*connect.Response[vpsv1.GetVPSMetricsResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.view"); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID"))
	}

	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox config: %w", err))
	}

	proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create Proxmox client: %w", err))
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID))
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find Proxmox node: %w", err))
	}

	// Get current metrics from Proxmox
	metrics, err := proxmoxClient.GetVMMetrics(ctx, nodes[0], vmIDInt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VM metrics: %w", err))
	}

	// Get disk size from VM config (fallback to database value)
	diskTotalBytes := vps.DiskBytes
	if diskSize, err := proxmoxClient.GetVMDiskSize(ctx, nodes[0], vmIDInt); err == nil {
		diskTotalBytes = diskSize
	}

	// Parse metrics from Proxmox response
	cpuUsagePercent := 0.0
	if cpu, ok := metrics["cpu"].(float64); ok {
		cpuUsagePercent = cpu * 100 // Proxmox returns CPU as fraction (0.0-1.0)
	}

	memoryUsedBytes := int64(0)
	memoryTotalBytes := vps.MemoryBytes
	if mem, ok := metrics["mem"].(float64); ok {
		memoryUsedBytes = int64(mem)
	}
	if maxmem, ok := metrics["maxmem"].(float64); ok {
		memoryTotalBytes = int64(maxmem)
	}

	// Get disk used from guest agent if available, otherwise use 0
	diskUsedBytes := int64(0)
	if diskUsed, ok := metrics["disk"].(float64); ok {
		diskUsedBytes = int64(diskUsed)
	}

	// Network stats (if available)
	networkRxBytes := int64(0)
	networkTxBytes := int64(0)
	if netin, ok := metrics["netin"].(float64); ok {
		networkRxBytes = int64(netin)
	}
	if netout, ok := metrics["netout"].(float64); ok {
		networkTxBytes = int64(netout)
	}

	// Create metric
	metric := &vpsv1.VPSMetric{
		VpsId:            vpsID,
		Timestamp:        timestamppb.Now(),
		CpuUsagePercent:  cpuUsagePercent,
		MemoryUsedBytes:  memoryUsedBytes,
		MemoryTotalBytes: memoryTotalBytes,
		DiskUsedBytes:    diskUsedBytes,
		DiskTotalBytes:   diskTotalBytes,
		NetworkRxBytes:   networkRxBytes,
		NetworkTxBytes:   networkTxBytes,
		DiskReadIops:     0, // Not available from current endpoint
		DiskWriteIops:    0, // Not available from current endpoint
	}

	return connect.NewResponse(&vpsv1.GetVPSMetricsResponse{
		Metrics: []*vpsv1.VPSMetric{metric},
	}), nil
}

// GetVPSUsage retrieves aggregated usage for a VPS instance
func (s *Service) GetVPSUsage(ctx context.Context, req *connect.Request[vpsv1.GetVPSUsageRequest]) (*connect.Response[vpsv1.GetVPSUsageResponse], error) {
	// TODO: Implement usage retrieval
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("not implemented"))
}
