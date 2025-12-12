package vps

import (
	"context"
	"errors"
	"fmt"
	"time"

	vpsorch "github.com/obiente/cloud/apps/vps-service/orchestrator"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/redis"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

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
	// Get regions from environment variable (similar to NODE_IPS)
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

// StreamVPSLogs streams VPS provisioning logs from Redis
func (s *Service) StreamVPSLogs(ctx context.Context, req *connect.Request[vpsv1.StreamVPSLogsRequest], stream *connect.ServerStream[vpsv1.VPSLogLine]) error {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return err
	}

	// Read and send buffered logs first from Redis
	streamer := redis.NewLogStreamer(vpsID)
	bufferedLogs, lastID, err := streamer.ReadBufferedLogs(ctx, "0", 1000)
	if err != nil {
		logger.Error("[StreamVPSLogs] Failed to read buffered logs from Redis for VPS %s: %v", vpsID, err)
		// Don't fail, just continue with empty buffer
		lastID = "0"
	}

	for _, logEntry := range bufferedLogs {
		// Check context before sending
		if ctx.Err() != nil {
			return nil
		}
		// Convert Redis log entry to protobuf VPSLogLine
		logLine := &vpsv1.VPSLogLine{
			Line:       logEntry.Line,
			Stderr:     logEntry.Stderr,
			LineNumber: logEntry.LineNumber,
			Timestamp:  timestamppb.New(logEntry.Timestamp),
		}
		if err := stream.Send(logLine); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}

	// Stream new logs from Redis
	logChan, errChan := streamer.Stream(ctx, lastID)

	// Start keepalive ticker to prevent connection timeout
	keepaliveTicker := time.NewTicker(5 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errChan:
			if err != nil {
				logger.Error("[StreamVPSLogs] Error streaming logs from Redis for VPS %s: %v", vpsID, err)
			}
			return nil
		case <-keepaliveTicker.C:
			// Send keepalive heartbeat
			heartbeat := &vpsv1.VPSLogLine{
				Line:       "",
				Stderr:     false,
				LineNumber: 0,
				Timestamp:  timestamppb.Now(),
			}
			if err := stream.Send(heartbeat); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return nil
			}
		case logEntry, ok := <-logChan:
			if !ok {
				return nil
			}
			// Convert Redis log entry to protobuf VPSLogLine
			logLine := &vpsv1.VPSLogLine{
				Line:       logEntry.Line,
				Stderr:     logEntry.Stderr,
				LineNumber: logEntry.LineNumber,
				Timestamp:  timestamppb.New(logEntry.Timestamp),
			}
			if err := stream.Send(logLine); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return nil
			}
		}
	}
}

// StreamVPSMetrics streams real-time VPS metrics
func (s *Service) StreamVPSMetrics(ctx context.Context, req *connect.Request[vpsv1.StreamVPSMetricsRequest], stream *connect.ServerStream[vpsv1.VPSMetric]) error {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return err
	}

	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer != nil {
		return s.streamLiveVPSMetrics(ctx, stream, vpsID)
	}

	// Fallback to database polling if streamer is not available
	intervalSeconds := 5 // Default 5 seconds
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	var lastSentTimestamp time.Time
	firstRun := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Get latest metric from database
			var vpsMetric database.VPSMetrics
			query := database.GetMetricsDB().Where("vps_instance_id = ?", vpsID).
				Order("timestamp DESC").
				Limit(1)
			
			if !firstRun {
				query = query.Where("timestamp > ?", lastSentTimestamp)
			}

			if err := query.First(&vpsMetric).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) && firstRun {
					// No metrics yet, continue waiting
					firstRun = false
					continue
				}
				// Error or no new metrics, continue
				continue
			}

			firstRun = false
			lastSentTimestamp = vpsMetric.Timestamp

			metric := &vpsv1.VPSMetric{
				VpsId:            vpsID,
				Timestamp:        timestamppb.New(vpsMetric.Timestamp),
				CpuUsagePercent:  vpsMetric.CPUUsage,
				MemoryUsedBytes:  vpsMetric.MemoryUsed,
				MemoryTotalBytes: vpsMetric.MemoryTotal,
				DiskUsedBytes:    vpsMetric.DiskUsed,
				DiskTotalBytes:   vpsMetric.DiskTotal,
				NetworkRxBytes:   vpsMetric.NetworkRxBytes,
				NetworkTxBytes:   vpsMetric.NetworkTxBytes,
				DiskReadIops:     vpsMetric.DiskReadIOPS,
				DiskWriteIops:    vpsMetric.DiskWriteIOPS,
			}

			if err := stream.Send(metric); err != nil {
				return err
			}
		}
	}
}

// streamLiveVPSMetrics streams metrics directly from the live metrics streamer
func (s *Service) streamLiveVPSMetrics(
	ctx context.Context,
	stream *connect.ServerStream[vpsv1.VPSMetric],
	vpsID string,
) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[StreamVPSMetrics] Panic in streamLiveVPSMetrics: %v", r)
		}
	}()

	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer == nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("metrics streamer not available"))
	}

	// Subscribe to live metrics for this VPS
	metricChan := metricsStreamer.Subscribe(vpsID)
	defer metricsStreamer.Unsubscribe(vpsID, metricChan)

	// Send initial metric from latest live cache
	latestMetrics := metricsStreamer.GetLatestMetrics(vpsID)
	if len(latestMetrics) > 0 {
		// Get the most recent metric(s) - filter for VPS type
		for i := len(latestMetrics) - 1; i >= 0; i-- {
			latest := latestMetrics[i]
			if latest.ResourceType == "vps" && latest.ResourceID == vpsID {
				cpuUsage := latest.CPUUsage
				memoryUsedBytes := latest.MemoryUsage
			diskReadIops := 0.0
			diskWriteIops := 0.0
			metric := &vpsv1.VPSMetric{
				VpsId:            vpsID,
				Timestamp:        timestamppb.New(latest.Timestamp),
				CpuUsagePercent:  cpuUsage,
				MemoryUsedBytes:  memoryUsedBytes,
				NetworkRxBytes:   latest.NetworkRxBytes,
				NetworkTxBytes:   latest.NetworkTxBytes,
				DiskReadIops:     diskReadIops,
				DiskWriteIops:    diskWriteIops,
			}
				if err := stream.Send(metric); err != nil {
					logger.Error("[StreamVPSMetrics] Failed to send initial metric: %v", err)
					return err
				}
				break
			}
		}
	}

	// Stream new metrics as they arrive
	heartbeatInterval := 30 * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	lastMetricTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			// Check if we're receiving metrics regularly
			if time.Since(lastMetricTime) > 2*heartbeatInterval {
				logger.Warn("[StreamVPSMetrics] No metrics received for %v, stream may be stale", time.Since(lastMetricTime))
			}
		case liveMetric, ok := <-metricChan:
			if !ok {
				// Channel closed
				return nil
			}

			// Filter for VPS type and matching ID
			if liveMetric.ResourceType != "vps" || liveMetric.ResourceID != vpsID {
				continue
			}

			lastMetricTime = time.Now()

			cpuUsage := liveMetric.CPUUsage
			memoryUsedBytes := liveMetric.MemoryUsage
			networkRxBytes := liveMetric.NetworkRxBytes
			networkTxBytes := liveMetric.NetworkTxBytes

			diskReadIops := 0.0
			diskWriteIops := 0.0
			metric := &vpsv1.VPSMetric{
				VpsId:            vpsID,
				Timestamp:        timestamppb.New(liveMetric.Timestamp),
				CpuUsagePercent:  cpuUsage,
				MemoryUsedBytes:  memoryUsedBytes,
				NetworkRxBytes:   networkRxBytes,
				NetworkTxBytes:   networkTxBytes,
				DiskReadIops:     diskReadIops,
				DiskWriteIops:    diskWriteIops,
			}

			if err := stream.Send(metric); err != nil {
				logger.Error("[StreamVPSMetrics] Failed to send metric: %v", err)
				return err
			}
		}
	}
}

// GetVPSMetrics retrieves VPS instance metrics (real-time or historical)
func (s *Service) GetVPSMetrics(ctx context.Context, req *connect.Request[vpsv1.GetVPSMetricsRequest]) (*connect.Response[vpsv1.GetVPSMetricsResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Parse time range
	startTime := time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	endTime := time.Now()

	if req.Msg.GetStartTime() != nil {
		startTime = req.Msg.GetStartTime().AsTime()
	}
	if req.Msg.GetEndTime() != nil {
		endTime = req.Msg.GetEndTime().AsTime()
	}

	// Keep last 24 hours of raw metrics, older data is aggregated hourly
	cutoffForRaw := time.Now().Add(-24 * time.Hour)

	var dbMetrics []database.VPSMetrics
	metricsDB := database.GetMetricsDB()

	// Query raw metrics (last 24 hours)
	rawStartTime := startTime
	rawEndTime := endTime
	if rawStartTime.Before(cutoffForRaw) {
		rawStartTime = cutoffForRaw
	}
	if rawStartTime.Before(rawEndTime) && metricsDB != nil {
		query := metricsDB.Where("vps_instance_id = ? AND timestamp >= ? AND timestamp <= ?", vpsID, rawStartTime, rawEndTime)
		query = query.Order("timestamp ASC").Limit(10000)

		if err := query.Find(&dbMetrics).Error; err != nil {
			logger.Warn("[GetVPSMetrics] Failed to query raw metrics: %v", err)
		}
	}

	// Query hourly aggregates (older than 24 hours)
	var hourlyAggregates []database.VPSUsageHourly
	if startTime.Before(cutoffForRaw) && metricsDB != nil {
		hourlyStart := startTime.Truncate(time.Hour)
		hourlyEnd := cutoffForRaw.Truncate(time.Hour)
		if hourlyStart.Before(hourlyEnd) {
			if err := metricsDB.Where("vps_instance_id = ? AND hour >= ? AND hour < ?", vpsID, hourlyStart, hourlyEnd).
				Order("hour ASC").
				Find(&hourlyAggregates).Error; err != nil {
				logger.Warn("[GetVPSMetrics] Failed to query hourly aggregates: %v", err)
			}
		}
	}

	// Convert to proto
	metrics := make([]*vpsv1.VPSMetric, 0, len(dbMetrics)+len(hourlyAggregates))

	// Convert raw metrics
	for _, m := range dbMetrics {
		metrics = append(metrics, &vpsv1.VPSMetric{
			VpsId:            vpsID,
			Timestamp:        timestamppb.New(m.Timestamp),
			CpuUsagePercent:  m.CPUUsage,
			MemoryUsedBytes:  m.MemoryUsed,
			MemoryTotalBytes: m.MemoryTotal,
			DiskUsedBytes:    m.DiskUsed,
			DiskTotalBytes:   m.DiskTotal,
			NetworkRxBytes:   m.NetworkRxBytes,
			NetworkTxBytes:   m.NetworkTxBytes,
			DiskReadIops:     m.DiskReadIOPS,
			DiskWriteIops:    m.DiskWriteIOPS,
		})
	}

	// Convert hourly aggregates (use hour start time as timestamp)
	for _, h := range hourlyAggregates {
		cpuUsage := h.AvgCPUUsage
		memoryBytes := int64(h.AvgMemoryUsage)
		metrics = append(metrics, &vpsv1.VPSMetric{
			VpsId:            vpsID,
			Timestamp:        timestamppb.New(h.Hour),
			CpuUsagePercent:  cpuUsage,
			MemoryUsedBytes:  memoryBytes,
			MemoryTotalBytes: memoryBytes, // Use average as total for hourly aggregates
			NetworkRxBytes:   h.BandwidthRxBytes,
			NetworkTxBytes:   h.BandwidthTxBytes,
			DiskReadIops:     0, // Not available in hourly aggregates
			DiskWriteIops:    0, // Not available in hourly aggregates
		})
	}

	// If no time range specified or end time is recent, also get current metric from Proxmox
	if req.Msg.GetStartTime() == nil || req.Msg.GetEndTime() == nil || endTime.After(time.Now().Add(-5*time.Minute)) {
		if vps.InstanceID != nil {
			vmIDInt := 0
			fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
			if vmIDInt > 0 {
				// Get node name from VPS (required)
				nodeName := ""
				if vps.NodeID != nil && *vps.NodeID != "" {
					nodeName = *vps.NodeID
				} else {
					logger.Warn("[GetVPSMetrics] VPS %s has no node ID - skipping current metrics from Proxmox", vpsID)
				}
				if nodeName != "" {
					// Get VPS manager to get Proxmox client for the node
					vpsManager, err := vpsorch.NewVPSManager()
					if err == nil {
						defer vpsManager.Close()
						proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
						if err == nil {
							// Get current metrics from Proxmox
							proxmoxMetrics, err := proxmoxClient.GetVMMetrics(ctx, nodeName, vmIDInt)
							if err == nil {
								// Get disk size from VM config (fallback to database value)
								diskTotalBytes := vps.DiskBytes
								if diskSize, err := proxmoxClient.GetVMDiskSize(ctx, nodeName, vmIDInt); err == nil {
									diskTotalBytes = diskSize
								}

								// Parse metrics from Proxmox response
								cpuUsagePercent := 0.0
								if cpu, ok := proxmoxMetrics["cpu"].(float64); ok {
									cpuUsagePercent = cpu * 100 // Proxmox returns CPU as fraction (0.0-1.0)
								}

								memoryUsedBytes := int64(0)
								memoryTotalBytes := vps.MemoryBytes
								if mem, ok := proxmoxMetrics["mem"].(float64); ok {
									memoryUsedBytes = int64(mem)
								}
								if maxmem, ok := proxmoxMetrics["maxmem"].(float64); ok {
									memoryTotalBytes = int64(maxmem)
								}

								diskUsedBytes := int64(0)
								if diskUsed, ok := proxmoxMetrics["disk"].(float64); ok {
									diskUsedBytes = int64(diskUsed)
								}

								networkRxBytes := int64(0)
								networkTxBytes := int64(0)
								if netin, ok := proxmoxMetrics["netin"].(float64); ok {
									networkRxBytes = int64(netin)
								}
								if netout, ok := proxmoxMetrics["netout"].(float64); ok {
									networkTxBytes = int64(netout)
								}

								// Add current metric
								metrics = append(metrics, &vpsv1.VPSMetric{
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
								})
							}
						}
					}
				}
			}
		}
	}

	return connect.NewResponse(&vpsv1.GetVPSMetricsResponse{
		Metrics: metrics,
	}), nil
}

// GetVPSUsage retrieves aggregated usage for a VPS instance
func (s *Service) GetVPSUsage(ctx context.Context, req *connect.Request[vpsv1.GetVPSUsageRequest]) (*connect.Response[vpsv1.GetVPSUsageResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return nil, err
	}

	// Get VPS to verify organization and get storage
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("VPS does not belong to organization"))
	}

	// Determine month (default to current month)
	month := req.Msg.GetMonth()
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}

	// Calculate estimated monthly usage based on current month progress
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	// Parse requested month for historical queries
	requestedMonthStart := monthStart
	if month != now.Format("2006-01") {
		// Parse historical month
		t, err := time.Parse("2006-01", month)
		if err == nil {
			requestedMonthStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			monthEnd = requestedMonthStart.AddDate(0, 1, 0).Add(-time.Second)
		}
	}

	// Calculate usage from hourly aggregates and raw metrics using shared helper
	rawCutoff := time.Now().Add(-24 * time.Hour)
	if rawCutoff.Before(monthStart) {
		rawCutoff = monthStart
	}

	currentMetrics, err := common.CalculateUsageFromHourlyAndRaw(
		vpsID,
		"vps",
		requestedMonthStart,
		monthEnd,
		rawCutoff,
		vps.DiskBytes,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate usage: %w", err))
	}

	// Calculate uptime from vps_usage_hourly
	var uptime struct {
		UptimeSeconds int64
	}
	metricsDB := database.GetMetricsDB()
	if metricsDB != nil {
		metricsDB.Table("vps_usage_hourly vuh").
			Select("COALESCE(SUM(vuh.uptime_seconds), 0) as uptime_seconds").
			Where("vuh.vps_instance_id = ? AND vuh.hour >= ? AND vuh.hour <= ?", vpsID, requestedMonthStart, monthEnd).
			Scan(&uptime)
	}
	currentMetrics.UptimeSeconds = uptime.UptimeSeconds

	// Calculate estimated monthly usage
	var estimatedMonthly common.ContainerUsageMetrics
	if month == now.Format("2006-01") {
		estimatedMonthly = common.CalculateEstimatedMonthly(currentMetrics, monthStart, monthEnd)
	} else {
		// Historical month: estimated equals current (already full month)
		estimatedMonthly = currentMetrics
	}

	// Calculate costs
	isCurrentMonth := month == now.Format("2006-01")
	currCPUCost, currMemoryCost, currBandwidthCost, currStorageCost, currTotalCost := common.CalculateCosts(
		currentMetrics,
		isCurrentMonth,
		monthStart,
		monthEnd,
	)

	estCPUCost, estMemoryCost, estBandwidthCost, estStorageCost, estTotalCost := common.CalculateCosts(
		estimatedMonthly,
		false, // Estimated is always full month
		monthStart,
		monthEnd,
	)

	// Build response
	currCPUCostPtr := int64(currCPUCost)
	currMemoryCostPtr := int64(currMemoryCost)
	currBandwidthCostPtr := int64(currBandwidthCost)
	currStorageCostPtr := int64(currStorageCost)
	currentUsageMetrics := &vpsv1.VPSUsageMetrics{
		CpuCoreSeconds:     currentMetrics.CPUCoreSeconds,
		MemoryByteSeconds:  currentMetrics.MemoryByteSeconds,
		BandwidthRxBytes:   currentMetrics.BandwidthRxBytes,
		BandwidthTxBytes:   currentMetrics.BandwidthTxBytes,
		DiskBytes:          currentMetrics.StorageBytes,
		UptimeSeconds:      currentMetrics.UptimeSeconds,
		EstimatedCostCents: currTotalCost,
		CpuCostCents:       &currCPUCostPtr,
		MemoryCostCents:   &currMemoryCostPtr,
		BandwidthCostCents: &currBandwidthCostPtr,
		StorageCostCents:  &currStorageCostPtr,
	}

	estCPUCostPtr := int64(estCPUCost)
	estMemoryCostPtr := int64(estMemoryCost)
	estBandwidthCostPtr := int64(estBandwidthCost)
	estStorageCostPtr := int64(estStorageCost)
	estimatedUsageMetrics := &vpsv1.VPSUsageMetrics{
		CpuCoreSeconds:     estimatedMonthly.CPUCoreSeconds,
		MemoryByteSeconds:  estimatedMonthly.MemoryByteSeconds,
		BandwidthRxBytes:   estimatedMonthly.BandwidthRxBytes,
		BandwidthTxBytes:   estimatedMonthly.BandwidthTxBytes,
		DiskBytes:          estimatedMonthly.StorageBytes,
		UptimeSeconds:      estimatedMonthly.UptimeSeconds,
		EstimatedCostCents: estTotalCost,
		CpuCostCents:       &estCPUCostPtr,
		MemoryCostCents:    &estMemoryCostPtr,
		BandwidthCostCents: &estBandwidthCostPtr,
		StorageCostCents:   &estStorageCostPtr,
	}

	response := &vpsv1.GetVPSUsageResponse{
		VpsId:             vpsID,
		Month:             month,
		Current:           currentUsageMetrics,
		EstimatedMonthly:  estimatedUsageMetrics,
		EstimatedCostCents: estTotalCost,
	}

	return connect.NewResponse(response), nil
}

// ImportVPS imports missing VPS instances from Proxmox that belong to the organization
func (s *Service) ImportVPS(ctx context.Context, req *connect.Request[vpsv1.ImportVPSRequest]) (*connect.Response[vpsv1.ImportVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	// Create VPS manager
	vpsManager, err := vpsorch.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}

	// Import VPS from Proxmox
	results, err := vpsManager.ImportVPS(ctx, orgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to import VPS: %w", err))
	}

	// Process results
	var importedVPS []*vpsv1.VPSInstance
	var errors []string
	importedCount := int32(0)
	skippedCount := int32(0)

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error.Error())
			skippedCount++
		} else if result.Skipped {
			skippedCount++
		} else if result.VPS != nil {
			importedVPS = append(importedVPS, vpsToProto(result.VPS))
			importedCount++
		}
	}

	response := &vpsv1.ImportVPSResponse{
		ImportedCount: importedCount,
		ImportedVps:   importedVPS,
		SkippedCount:  skippedCount,
		Errors:        errors,
	}

	return connect.NewResponse(response), nil
}
