package gameservers

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetGameServerMetrics retrieves metrics for a game server
func (s *Service) GetGameServerMetrics(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerMetricsRequest]) (*connect.Response[gameserversv1.GetGameServerMetricsResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
		return nil, err
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

	var dbMetrics []database.GameServerMetrics

	// Query raw metrics (last 24 hours)
	rawStartTime := startTime
	rawEndTime := endTime
	if rawStartTime.Before(cutoffForRaw) {
		rawStartTime = cutoffForRaw
	}
	if rawStartTime.Before(rawEndTime) {
		metricsDB := database.GetMetricsDB()
		query := metricsDB.Where("game_server_id = ? AND timestamp >= ? AND timestamp <= ?", gameServerID, rawStartTime, rawEndTime)
		query = query.Order("timestamp ASC").Limit(10000)

		if err := query.Find(&dbMetrics).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query raw metrics: %w", err))
		}

		logger.Debug("[GetGameServerMetrics] Querying raw metrics for game server %s: start=%s, end=%s, found=%d",
			gameServerID, rawStartTime.Format(time.RFC3339), rawEndTime.Format(time.RFC3339), len(dbMetrics))
	}

	// Query hourly aggregates (older than 24 hours)
	var hourlyAggregates []database.GameServerUsageHourly
	if startTime.Before(cutoffForRaw) {
		hourlyStart := startTime.Truncate(time.Hour)
		hourlyEnd := cutoffForRaw.Truncate(time.Hour)
		if hourlyStart.Before(hourlyEnd) {
			metricsDB := database.GetMetricsDB()
			if err := metricsDB.Where("game_server_id = ? AND hour >= ? AND hour < ?", gameServerID, hourlyStart, hourlyEnd).
				Order("hour ASC").
				Find(&hourlyAggregates).Error; err != nil {
				logger.Warn("[GetGameServerMetrics] Failed to query hourly aggregates: %v", err)
			} else {
				logger.Debug("[GetGameServerMetrics] Querying hourly aggregates for game server %s: start=%s, end=%s, found=%d",
					gameServerID, hourlyStart.Format(time.RFC3339), hourlyEnd.Format(time.RFC3339), len(hourlyAggregates))
			}
		}
	}

	logger.Debug("[GetGameServerMetrics] Returning %d total metrics (%d raw + %d hourly) for game server %s",
		len(dbMetrics)+len(hourlyAggregates), len(dbMetrics), len(hourlyAggregates), gameServerID)

	// Convert to proto
	metrics := make([]*gameserversv1.GameServerMetric, 0, len(dbMetrics)+len(hourlyAggregates))

	// Convert raw metrics
	for _, m := range dbMetrics {
		metrics = append(metrics, &gameserversv1.GameServerMetric{
			GameServerId:     gameServerID,
			Timestamp:        timestamppb.New(m.Timestamp),
			CpuUsagePercent:  &m.CPUUsage,
			MemoryUsageBytes: &m.MemoryUsage,
			NetworkRxBytes:   &m.NetworkRxBytes,
			NetworkTxBytes:   &m.NetworkTxBytes,
			DiskReadBytes:    &m.DiskReadBytes,
			DiskWriteBytes:   &m.DiskWriteBytes,
		})
	}

	// Convert hourly aggregates (use hour start time as timestamp)
	for _, h := range hourlyAggregates {
		cpuUsage := h.AvgCPUUsage
		memoryBytes := int64(h.AvgMemoryUsage)
		metrics = append(metrics, &gameserversv1.GameServerMetric{
			GameServerId:     gameServerID,
			Timestamp:        timestamppb.New(h.Hour),
			CpuUsagePercent:  &cpuUsage,
			MemoryUsageBytes: &memoryBytes,
			NetworkRxBytes:   &h.BandwidthRxBytes,
			NetworkTxBytes:   &h.BandwidthTxBytes,
			DiskReadBytes:    &h.DiskReadBytes,
			DiskWriteBytes:   &h.DiskWriteBytes,
		})
	}

	return connect.NewResponse(&gameserversv1.GetGameServerMetricsResponse{
		Metrics: metrics,
	}), nil
}

// streamLiveGameServerMetrics streams metrics directly from the live metrics streamer
func (s *Service) streamLiveGameServerMetrics(
	ctx context.Context,
	stream *connect.ServerStream[gameserversv1.GameServerMetric],
	gameServerID string,
) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[StreamGameServerMetrics] Panic in streamLiveGameServerMetrics: %v", r)
		}
	}()

	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer == nil {
		// Should not happen, but fallback gracefully
		return connect.NewError(connect.CodeInternal, fmt.Errorf("metrics streamer not available"))
	}

	// Subscribe to live metrics for this game server
	metricChan := metricsStreamer.Subscribe(gameServerID)
	defer metricsStreamer.Unsubscribe(gameServerID, metricChan)

	// Send initial metric from latest live cache
	latestMetrics := metricsStreamer.GetLatestMetrics(gameServerID)
	if len(latestMetrics) > 0 {
		// Get the most recent metric(s) - filter for game server type
		for i := len(latestMetrics) - 1; i >= 0; i-- {
			latest := latestMetrics[i]
			if latest.ResourceType == "gameserver" && latest.ResourceID == gameServerID {
				cpuUsage := latest.CPUUsage
				memoryBytes := latest.MemoryUsage
				diskReadBytes := latest.DiskReadBytes
				diskWriteBytes := latest.DiskWriteBytes
				metric := &gameserversv1.GameServerMetric{
					GameServerId:     gameServerID,
					Timestamp:        timestamppb.New(latest.Timestamp),
					CpuUsagePercent:  &cpuUsage,
					MemoryUsageBytes: &memoryBytes,
					NetworkRxBytes:   &latest.NetworkRxBytes,
					NetworkTxBytes:   &latest.NetworkTxBytes,
					DiskReadBytes:    &diskReadBytes,
					DiskWriteBytes:   &diskWriteBytes,
				}
				if err := stream.Send(metric); err != nil {
					logger.Error("[StreamGameServerMetrics] Failed to send initial metric: %v", err)
					return err
				}
				break
			}
		}
	}

	// Stream new metrics as they arrive
	// Add a heartbeat ticker to detect when metrics aren't being received
	// This prevents the stream from appearing to hang when metrics collection is slow
	heartbeatInterval := 30 * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()
	
	lastMetricTime := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			// Check if we haven't received metrics in a while
			timeSinceLastMetric := time.Since(lastMetricTime)
			if timeSinceLastMetric > 2*heartbeatInterval {
				// No metrics received for 60 seconds - send a heartbeat to keep connection alive
				// and log a warning
				logger.Warn("[StreamGameServerMetrics] No metrics received for %v for game server %s, sending heartbeat", timeSinceLastMetric, gameServerID)
				// Send a zero-value metric as a heartbeat to keep the connection alive
				zeroCPU := 0.0
				zeroMemory := int64(0)
				heartbeatMetric := &gameserversv1.GameServerMetric{
					GameServerId:     gameServerID,
					Timestamp:        timestamppb.Now(),
					CpuUsagePercent:  &zeroCPU,
					MemoryUsageBytes: &zeroMemory,
				}
				if err := stream.Send(heartbeatMetric); err != nil {
					logger.Error("[StreamGameServerMetrics] Failed to send heartbeat: %v", err)
					return err
				}
			}
		case liveMetric, ok := <-metricChan:
			if !ok {
				// Channel closed
				return nil
			}

			// Filter for game server type and matching ID
			if liveMetric.ResourceType != "gameserver" || liveMetric.ResourceID != gameServerID {
				continue
			}

			lastMetricTime = time.Now() // Update last metric time
			
			cpuUsage := liveMetric.CPUUsage
			memoryBytes := liveMetric.MemoryUsage
			diskReadBytes := liveMetric.DiskReadBytes
			diskWriteBytes := liveMetric.DiskWriteBytes
			metric := &gameserversv1.GameServerMetric{
				GameServerId:     gameServerID,
				Timestamp:        timestamppb.New(liveMetric.Timestamp),
				CpuUsagePercent:  &cpuUsage,
				MemoryUsageBytes: &memoryBytes,
				NetworkRxBytes:   &liveMetric.NetworkRxBytes,
				NetworkTxBytes:   &liveMetric.NetworkTxBytes,
				DiskReadBytes:    &diskReadBytes,
				DiskWriteBytes:   &diskWriteBytes,
			}

			if err := stream.Send(metric); err != nil {
				logger.Error("[StreamGameServerMetrics] Failed to send metric: %v", err)
				return err
			}
		}
	}
}

// StreamGameServerMetrics streams real-time metrics for a game server
func (s *Service) StreamGameServerMetrics(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerMetricsRequest], stream *connect.ServerStream[gameserversv1.GameServerMetric]) error {
	// Ensure authenticated - use the returned context
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
		return err
	}

	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer != nil {
		return s.streamLiveGameServerMetrics(ctx, stream, gameServerID)
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
			var dbMetric database.GameServerMetrics
			metricsDB := database.GetMetricsDB()
			query := metricsDB.Where("game_server_id = ?", gameServerID)

			if firstRun {
				query = query.Order("timestamp DESC").Limit(1)
			} else {
				query = query.Where("timestamp > ?", lastSentTimestamp).Order("timestamp ASC").Limit(100)
			}

			if err := query.Find(&dbMetric).Error; err == nil && dbMetric.ID != 0 {
				cpuUsage := dbMetric.CPUUsage
				memoryBytes := dbMetric.MemoryUsage
				metric := &gameserversv1.GameServerMetric{
					GameServerId:     gameServerID,
					Timestamp:        timestamppb.New(dbMetric.Timestamp),
					CpuUsagePercent:  &cpuUsage,
					MemoryUsageBytes: &memoryBytes,
					NetworkRxBytes:   &dbMetric.NetworkRxBytes,
					NetworkTxBytes:   &dbMetric.NetworkTxBytes,
					DiskReadBytes:    &dbMetric.DiskReadBytes,
					DiskWriteBytes:   &dbMetric.DiskWriteBytes,
				}

				if err := stream.Send(metric); err != nil {
					return err
				}

				lastSentTimestamp = dbMetric.Timestamp
				firstRun = false
			}
		}
	}
}

// GetGameServerUsage retrieves aggregated usage for a game server
func (s *Service) GetGameServerUsage(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerUsageRequest]) (*connect.Response[gameserversv1.GetGameServerUsageResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
		return nil, err
	}

	// Get game server to verify organization and get storage
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found: %w", err))
	}
	if gameServer.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("game server does not belong to organization"))
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
		gameServerID,
		"gameserver",
		requestedMonthStart,
		monthEnd,
		rawCutoff,
		gameServer.StorageBytes,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate usage: %w", err))
	}

	// Calculate uptime from game_server_locations (similar to deployment_locations)
	var uptime struct {
		UptimeSeconds int64
	}
	if month == now.Format("2006-01") {
		// Current month: calculate uptime from locations
		database.DB.Table("game_server_locations gsl").
			Select(`
				COALESCE(SUM(EXTRACT(EPOCH FROM (
					CASE 
						WHEN gsl.status = 'running' THEN NOW() - gsl.created_at
						WHEN gsl.updated_at > gsl.created_at THEN gsl.updated_at - gsl.created_at
						ELSE '0 seconds'::interval
					END
				))), 0)::bigint as uptime_seconds
			`).
			Where("gsl.game_server_id = ? AND (gsl.created_at >= ? OR gsl.updated_at >= ?)", gameServerID, monthStart, monthStart).
			Scan(&uptime)
	} else {
		// Historical month: calculate uptime for the specific month
		database.DB.Table("game_server_locations gsl").
			Select(`
				COALESCE(SUM(EXTRACT(EPOCH FROM (
					CASE 
						WHEN gsl.status = 'running' AND gsl.updated_at <= ? THEN ?::timestamp - gsl.created_at
						WHEN gsl.updated_at > gsl.created_at AND gsl.updated_at <= ? THEN gsl.updated_at - gsl.created_at
						WHEN gsl.created_at >= ? AND gsl.created_at < ? THEN ?::timestamp - gsl.created_at
						ELSE '0 seconds'::interval
					END
				))), 0)::bigint as uptime_seconds
			`, monthEnd, monthEnd, monthEnd, requestedMonthStart, monthEnd, monthEnd).
			Where("gsl.game_server_id = ? AND ((gsl.created_at >= ? AND gsl.created_at <= ?) OR (gsl.updated_at >= ? AND gsl.updated_at <= ?))",
				gameServerID, requestedMonthStart, monthEnd, requestedMonthStart, monthEnd).
			Scan(&uptime)
	}
	currentMetrics.UptimeSeconds = uptime.UptimeSeconds

	// Calculate estimated monthly usage
	var estimatedMonthly common.ContainerUsageMetrics
	if month == now.Format("2006-01") {
		estimatedMonthly = common.CalculateEstimatedMonthly(currentMetrics, monthStart, monthEnd)
	} else {
		// Historical month: estimated equals current
		estimatedMonthly = currentMetrics
	}

	// Calculate costs using shared helper
	isCurrentMonth := month == now.Format("2006-01")
	currCPUCost, currMemoryCost, currBandwidthCost, currStorageCost, currTotalCost := common.CalculateCosts(currentMetrics, isCurrentMonth, monthStart, monthEnd)
	estCPUCost, estMemoryCost, estBandwidthCost, estStorageCost, estTotalCost := common.CalculateCosts(estimatedMonthly, false, monthStart, monthEnd)

	// Build response with current and estimated monthly metrics
	currentProto := &gameserversv1.GameServerUsageMetrics{
		CpuCoreSeconds:     currentMetrics.CPUCoreSeconds,
		MemoryByteSeconds:  currentMetrics.MemoryByteSeconds,
		BandwidthRxBytes:   currentMetrics.BandwidthRxBytes,
		BandwidthTxBytes:   currentMetrics.BandwidthTxBytes,
		StorageBytes:       currentMetrics.StorageBytes,
		UptimeSeconds:      currentMetrics.UptimeSeconds,
		EstimatedCostCents:  currTotalCost,
	}
	currCPUCostPtr := int64(currCPUCost)
	currMemoryCostPtr := int64(currMemoryCost)
	currBandwidthCostPtr := int64(currBandwidthCost)
	currStorageCostPtr := int64(currStorageCost)
	currentProto.CpuCostCents = &currCPUCostPtr
	currentProto.MemoryCostCents = &currMemoryCostPtr
	currentProto.BandwidthCostCents = &currBandwidthCostPtr
	currentProto.StorageCostCents = &currStorageCostPtr

	estimatedProto := &gameserversv1.GameServerUsageMetrics{
		CpuCoreSeconds:     estimatedMonthly.CPUCoreSeconds,
		MemoryByteSeconds:  estimatedMonthly.MemoryByteSeconds,
		BandwidthRxBytes:   estimatedMonthly.BandwidthRxBytes,
		BandwidthTxBytes:   estimatedMonthly.BandwidthTxBytes,
		StorageBytes:       estimatedMonthly.StorageBytes,
		UptimeSeconds:      estimatedMonthly.UptimeSeconds,
		EstimatedCostCents:  estTotalCost,
	}
	estCPUCostPtr := int64(estCPUCost)
	estMemoryCostPtr := int64(estMemoryCost)
	estBandwidthCostPtr := int64(estBandwidthCost)
	estStorageCostPtr := int64(estStorageCost)
	estimatedProto.CpuCostCents = &estCPUCostPtr
	estimatedProto.MemoryCostCents = &estMemoryCostPtr
	estimatedProto.BandwidthCostCents = &estBandwidthCostPtr
	estimatedProto.StorageCostCents = &estStorageCostPtr

	response := &gameserversv1.GetGameServerUsageResponse{
		GameServerId:     gameServerID,
		OrganizationId:   orgID,
		Month:            month,
		Current:          currentProto,
		EstimatedMonthly: estimatedProto,
	}

	return connect.NewResponse(response), nil
}
