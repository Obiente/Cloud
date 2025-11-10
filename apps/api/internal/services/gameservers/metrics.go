package gameservers

import (
	"context"
	"fmt"
	"time"

	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"
	"api/internal/pricing"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetGameServerMetrics retrieves metrics for a game server
func (s *Service) GetGameServerMetrics(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerMetricsRequest]) (*connect.Response[gameserversv1.GetGameServerMetricsResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
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
	for {
		select {
		case <-ctx.Done():
			return nil
		case liveMetric, ok := <-metricChan:
			if !ok {
				// Channel closed
				return nil
			}

			// Filter for game server type and matching ID
			if liveMetric.ResourceType != "gameserver" || liveMetric.ResourceID != gameServerID {
				continue
			}

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
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
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
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	// Get game server to determine organization
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found: %w", err))
	}

	// Determine month
	month := req.Msg.GetMonth()
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	// Parse month
	monthTime, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid month format: %w", err))
	}

	monthStart := time.Date(monthTime.Year(), monthTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Query hourly aggregates for the month
	metricsDB := database.GetMetricsDB()
	var hourlyAggregates []database.GameServerUsageHourly
	if err := metricsDB.Where("game_server_id = ? AND hour >= ? AND hour < ?", gameServerID, monthStart, monthEnd).
		Order("hour ASC").
		Find(&hourlyAggregates).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query usage: %w", err))
	}

	// Calculate totals
	var cpuCoreSeconds int64
	var memoryByteSeconds int64
	var bandwidthRxBytes int64
	var bandwidthTxBytes int64
	var storageBytes int64

	for _, h := range hourlyAggregates {
		// CPU: avg_cpu_usage is stored as percentage, convert to core-seconds
		// avg_cpu_usage = (core-seconds / 3600) * 100, so core-seconds = (avg_cpu_usage / 100) * 3600
		cpuCoreSeconds += int64((h.AvgCPUUsage / 100.0) * 3600.0)

		// Memory: avg_memory_usage is stored as bytes/second average, convert to byte-seconds
		// avg_memory_usage = byte-seconds / 3600, so byte-seconds = avg_memory_usage * 3600
		memoryByteSeconds += int64(h.AvgMemoryUsage * 3600.0)

		bandwidthRxBytes += h.BandwidthRxBytes
		bandwidthTxBytes += h.BandwidthTxBytes
	}

	// Get storage from game server
	storageBytes = gameServer.StorageBytes

	// Calculate costs
	pricingModel := pricing.GetPricing()
	cpuCostCents := int64(float64(cpuCoreSeconds) * pricingModel.CPUCostPerCoreSecond * 100)
	memoryCostCents := int64(float64(memoryByteSeconds) * pricingModel.MemoryCostPerByteSecond * 100)
	bandwidthTotalBytes := bandwidthRxBytes + bandwidthTxBytes
	bandwidthCostCents := int64(float64(bandwidthTotalBytes) * pricingModel.BandwidthCostPerByte * 100)
	storageCostCents := int64(float64(storageBytes) * pricingModel.StorageCostPerByteMonth * 100)

	totalCostCents := cpuCostCents + memoryCostCents + bandwidthCostCents + storageCostCents

	return connect.NewResponse(&gameserversv1.GetGameServerUsageResponse{
		GameServerId:       gameServerID,
		Month:              month,
		CpuCoreSeconds:     cpuCoreSeconds,
		MemoryByteSeconds:  memoryByteSeconds,
		BandwidthBytes:     bandwidthTotalBytes,
		StorageBytes:       storageBytes,
		CpuCostCents:       cpuCostCents,
		MemoryCostCents:    memoryCostCents,
		BandwidthCostCents: bandwidthCostCents,
		StorageCostCents:   storageCostCents,
		TotalCostCents:     totalCostCents,
	}), nil
}
