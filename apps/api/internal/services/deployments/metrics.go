package deployments

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/pricing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// GetDeploymentMetrics retrieves historical metrics for a deployment
func (s *Service) GetDeploymentMetrics(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentMetricsRequest]) (*connect.Response[deploymentsv1.GetDeploymentMetricsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists (allow deleted deployments for viewing metrics)
	var dbDeployment *database.Deployment
	var err error
	if orgID != "" {
		// Verify organization ownership
		dbDeployment, err = s.repo.GetByIDIncludeDeleted(ctx, deploymentID, true)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
		}
		if dbDeployment.OrganizationID != orgID {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
		}
	}

	// Determine time range
	var startTime time.Time
	var endTime time.Time = time.Now()

	if req.Msg.StartTime != nil {
		startTime = req.Msg.StartTime.AsTime()
	} else {
		// Default to last 24 hours
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if req.Msg.EndTime != nil {
		endTime = req.Msg.EndTime.AsTime()
	}

	// Get container/service filters
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	shouldAggregate := req.Msg.GetAggregate()

	// If container_id or service_name is specified, we need to resolve it
	var targetContainerID string
	if containerID != "" {
		targetContainerID = containerID
	} else if serviceName != "" {
		// Resolve service_name to container_id
		dcli, err := docker.New()
		if err == nil {
			defer dcli.Close()
			locations, err := database.GetDeploymentLocations(deploymentID)
			if err == nil {
				for _, loc := range locations {
					containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
					if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
						labelServiceName := containerInfo.Config.Labels["com.obiente.service_name"]
						if labelServiceName == "" {
							labelServiceName = containerInfo.Config.Labels["com.docker.compose.service"]
						}
						if labelServiceName == serviceName {
							targetContainerID = loc.ContainerID
							break
						}
					}
				}
			}
		}
	}

	// Get raw metrics (for recent data, typically last 24 hours)
	cutoffForRaw := time.Now().Add(-24 * time.Hour)

	var dbMetrics []database.DeploymentMetrics
	// Query raw metrics if the time range overlaps with recent data (within 24 hours)
	rawStartTime := startTime
	rawEndTime := endTime
	if rawStartTime.Before(cutoffForRaw) {
		rawStartTime = cutoffForRaw
	}
	if rawStartTime.Before(rawEndTime) {
		query := database.DB.Where("deployment_id = ? AND timestamp >= ? AND timestamp <= ?", deploymentID, rawStartTime, rawEndTime)

		// Apply container filter if specified
		if targetContainerID != "" {
			query = query.Where("container_id = ?", targetContainerID)
		}

		if req.Msg.GetLatestOnly() {
			query = query.Order("timestamp DESC").Limit(1)
		} else {
			query = query.Order("timestamp ASC").Limit(10000) // Reasonable limit
		}

		if err := query.Find(&dbMetrics).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query raw metrics: %w", err))
		}
	}

	// If querying older than 24 hours, also get hourly aggregates
	var hourlyAggregates []database.DeploymentUsageHourly
	if startTime.Before(cutoffForRaw) {
		hourlyStart := startTime.Truncate(time.Hour)
		hourlyEnd := cutoffForRaw.Truncate(time.Hour)
		if hourlyStart.Before(hourlyEnd) {
			if err := database.DB.Where("deployment_id = ? AND hour >= ? AND hour < ?", deploymentID, hourlyStart, hourlyEnd).
				Order("hour ASC").
				Find(&hourlyAggregates).Error; err != nil {
				// Log error but don't fail - we can still return raw metrics
				log.Printf("[GetDeploymentMetrics] Failed to query hourly aggregates: %v", err)
			}
		}
	}

	// Build a map of container ID to service name for enriching metrics
	containerServiceMap := make(map[string]string)
	if targetContainerID == "" || shouldAggregate {
		// We need service names, so fetch them
		dcli, err := docker.New()
		if err == nil {
			defer dcli.Close()
			locations, err := database.GetDeploymentLocations(deploymentID)
			if err == nil {
				for _, loc := range locations {
					containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
					if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
						serviceName := containerInfo.Config.Labels["com.obiente.service_name"]
						if serviceName == "" {
							serviceName = containerInfo.Config.Labels["com.docker.compose.service"]
						}
						if serviceName == "" {
							serviceName = "default"
						}
						containerServiceMap[loc.ContainerID] = serviceName
					}
				}
			}
		}
	}

	// Convert raw metrics to proto
	metrics := make([]*deploymentsv1.DeploymentMetric, 0, len(dbMetrics)+len(hourlyAggregates))

	// If we should aggregate (or no filter specified), group by timestamp first
	// When no container filter is specified, aggregate by default unless explicitly set to false
	shouldAggregateMetrics := shouldAggregate || (targetContainerID == "" && !req.Msg.GetAggregate())
	if shouldAggregateMetrics && len(dbMetrics) > 0 {
		// Group metrics by timestamp
		metricsByTimestamp := make(map[int64][]database.DeploymentMetrics)
		for _, m := range dbMetrics {
			ts := m.Timestamp.Unix()
			metricsByTimestamp[ts] = append(metricsByTimestamp[ts], m)
		}

		// Aggregate each timestamp group
		for ts, group := range metricsByTimestamp {
			var sumCPU float64
			var sumMemory int64
			var sumNetworkRx int64
			var sumNetworkTx int64
			var sumDiskRead int64
			var sumDiskWrite int64
			var sumRequestCount int64
			var sumErrorCount int64
			count := len(group)

			for _, m := range group {
				sumCPU += m.CPUUsage
				sumMemory += m.MemoryUsage
				sumNetworkRx += m.NetworkRxBytes
				sumNetworkTx += m.NetworkTxBytes
				sumDiskRead += m.DiskReadBytes
				sumDiskWrite += m.DiskWriteBytes
				sumRequestCount += m.RequestCount
				sumErrorCount += m.ErrorCount
			}

			avgCPU := sumCPU / float64(count)

			metric := &deploymentsv1.DeploymentMetric{
				DeploymentId:     deploymentID,
				Timestamp:        timestamppb.New(time.Unix(ts, 0)),
				CpuUsagePercent:  avgCPU,
				MemoryUsageBytes: sumMemory, // Total memory across containers
				NetworkRxBytes:   sumNetworkRx,
				NetworkTxBytes:   sumNetworkTx,
				DiskReadBytes:    sumDiskRead,
				DiskWriteBytes:   sumDiskWrite,
			}
			if sumRequestCount > 0 {
				metric.RequestCount = &sumRequestCount
			}
			if sumErrorCount > 0 {
				metric.ErrorCount = &sumErrorCount
			}
			metrics = append(metrics, metric)
		}
	} else {
		// Return individual container metrics
		for _, m := range dbMetrics {
			metric := &deploymentsv1.DeploymentMetric{
				DeploymentId:     m.DeploymentID,
				Timestamp:        timestamppb.New(m.Timestamp),
				CpuUsagePercent:  m.CPUUsage,
				MemoryUsageBytes: m.MemoryUsage,
				NetworkRxBytes:   m.NetworkRxBytes,
				NetworkTxBytes:   m.NetworkTxBytes,
				DiskReadBytes:    m.DiskReadBytes,
				DiskWriteBytes:   m.DiskWriteBytes,
			}
			if m.ContainerID != "" {
				metric.ContainerId = &m.ContainerID
				if svcName, ok := containerServiceMap[m.ContainerID]; ok {
					metric.ServiceName = &svcName
				}
			}
			if m.RequestCount > 0 {
				metric.RequestCount = &m.RequestCount
			}
			if m.ErrorCount > 0 {
				metric.ErrorCount = &m.ErrorCount
			}
			metrics = append(metrics, metric)
		}
	}

	// Convert hourly aggregates to proto (one data point per hour)
	for _, h := range hourlyAggregates {
		reqCount := h.RequestCount
		errCount := h.ErrorCount
		metrics = append(metrics, &deploymentsv1.DeploymentMetric{
			DeploymentId:     h.DeploymentID,
			Timestamp:        timestamppb.New(h.Hour),
			CpuUsagePercent:  h.AvgCPUUsage,
			MemoryUsageBytes: h.AvgMemoryUsage,
			NetworkRxBytes:   h.BandwidthRxBytes, // Hourly totals
			NetworkTxBytes:   h.BandwidthTxBytes,
			DiskReadBytes:    h.DiskReadBytes, // Hourly totals
			DiskWriteBytes:   h.DiskWriteBytes,
			RequestCount:     &reqCount,
			ErrorCount:       &errCount,
		})
	}

	// Sort all metrics by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		tsI := metrics[i].Timestamp.AsTime()
		tsJ := metrics[j].Timestamp.AsTime()
		return tsI.Before(tsJ)
	})

	if req.Msg.GetLatestOnly() && len(metrics) > 0 {
		// Return only the latest metric
		metrics = []*deploymentsv1.DeploymentMetric{metrics[len(metrics)-1]}
	}

	return connect.NewResponse(&deploymentsv1.GetDeploymentMetricsResponse{
		Metrics: metrics,
	}), nil
}

// StreamDeploymentMetrics streams real-time deployment metrics
func (s *Service) StreamDeploymentMetrics(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentMetricsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentMetric]) error {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Ensure user is authenticated for streaming RPCs (interceptor may not run)
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get container/service filters
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	shouldAggregate := req.Msg.GetAggregate()

	// Resolve container filter similar to GetDeploymentMetrics
	var targetContainerID string
	if containerID != "" {
		targetContainerID = containerID
	} else if serviceName != "" {
		dcli, err := docker.New()
		if err == nil {
			defer dcli.Close()
			locations, err := database.GetDeploymentLocations(deploymentID)
			if err == nil {
				for _, loc := range locations {
					containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
					if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
						labelServiceName := containerInfo.Config.Labels["com.obiente.service_name"]
						if labelServiceName == "" {
							labelServiceName = containerInfo.Config.Labels["com.docker.compose.service"]
						}
						if labelServiceName == serviceName {
							targetContainerID = loc.ContainerID
							break
						}
					}
				}
			}
		}
	}

	intervalSeconds := int(req.Msg.GetIntervalSeconds())
	if intervalSeconds <= 0 {
		intervalSeconds = 5 // Default 5 seconds
	}
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	// Track the last metric timestamp we sent
	var lastSentTimestamp time.Time
	// On first run, send the most recent metric if available, then stream new ones
	firstRun := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var dbMetric database.DeploymentMetrics
			query := database.DB.Where("deployment_id = ?", deploymentID)

			// Apply container filter if specified
			if targetContainerID != "" {
				query = query.Where("container_id = ?", targetContainerID)
			}

			if firstRun {
				// First run: Get the most recent metric regardless of time
				// Try raw metrics first, then hourly aggregates
				err := query.Order("timestamp DESC").
					First(&dbMetric).Error

				// If no raw metrics, try hourly aggregates
				if errors.Is(err, gorm.ErrRecordNotFound) {
					var hourlyMetric database.DeploymentUsageHourly
					hourlyErr := database.DB.Where("deployment_id = ?", deploymentID).
						Order("hour DESC").
						First(&hourlyMetric).Error

					if hourlyErr == nil {
						// Convert hourly aggregate to metric format
						lastSentTimestamp = hourlyMetric.Hour
						firstRun = false
						reqCount := hourlyMetric.RequestCount
						errCount := hourlyMetric.ErrorCount

						metric := &deploymentsv1.DeploymentMetric{
							DeploymentId:     hourlyMetric.DeploymentID,
							Timestamp:        timestamppb.New(hourlyMetric.Hour),
							CpuUsagePercent:  hourlyMetric.AvgCPUUsage,
							MemoryUsageBytes: hourlyMetric.AvgMemoryUsage,
							NetworkRxBytes:   hourlyMetric.BandwidthRxBytes,
							NetworkTxBytes:   hourlyMetric.BandwidthTxBytes,
							DiskReadBytes:    hourlyMetric.DiskReadBytes,
							DiskWriteBytes:   hourlyMetric.DiskWriteBytes,
						}
						if reqCount > 0 {
							metric.RequestCount = &reqCount
						}
						if errCount > 0 {
							metric.ErrorCount = &errCount
						}

						if err := stream.Send(metric); err != nil {
							return err
						}
						continue
					}
				}

				if err == nil {
					lastSentTimestamp = dbMetric.Timestamp
					firstRun = false

					metric := &deploymentsv1.DeploymentMetric{
						DeploymentId:     dbMetric.DeploymentID,
						Timestamp:        timestamppb.New(dbMetric.Timestamp),
						CpuUsagePercent:  dbMetric.CPUUsage,
						MemoryUsageBytes: dbMetric.MemoryUsage,
						NetworkRxBytes:   dbMetric.NetworkRxBytes,
						NetworkTxBytes:   dbMetric.NetworkTxBytes,
						DiskReadBytes:    dbMetric.DiskReadBytes,
						DiskWriteBytes:   dbMetric.DiskWriteBytes,
					}
					if dbMetric.RequestCount > 0 {
						metric.RequestCount = &dbMetric.RequestCount
					}
					if dbMetric.ErrorCount > 0 {
						metric.ErrorCount = &dbMetric.ErrorCount
					}

					if err := stream.Send(metric); err != nil {
						return err
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("[StreamDeploymentMetrics] Error querying metrics for %s: %v", deploymentID, err)
				}
				// Continue to next tick if no metrics found yet
				continue
			}

			// Subsequent runs: Get metrics newer than the last one we sent
			nextQuery := query.Where("timestamp > ?", lastSentTimestamp).Order("timestamp DESC")

			// Determine if we should aggregate (aggregate if no container filter and aggregate is true or not specified)
			shouldAggregateMetrics := shouldAggregate || (targetContainerID == "" && !req.Msg.GetAggregate())
			if shouldAggregateMetrics {
				// Aggregate mode: get all containers' metrics for the latest timestamp
				var latestMetrics []database.DeploymentMetrics
				// Get the latest timestamp first
				// Get the latest timestamp using raw SQL to avoid ORDER BY issues with aggregates
				var latestTimestamp *time.Time
				var timestampValue time.Time
				query := `SELECT MAX(timestamp) as max_timestamp FROM deployment_metrics WHERE deployment_id = $1`
				args := []interface{}{deploymentID}
				argIndex := 2
				
				if targetContainerID != "" {
					query += ` AND container_id = $` + fmt.Sprintf("%d", argIndex)
					args = append(args, targetContainerID)
					argIndex++
				}
				if !lastSentTimestamp.IsZero() {
					query += ` AND timestamp > $` + fmt.Sprintf("%d", argIndex)
					args = append(args, lastSentTimestamp)
					argIndex++
				}
				
				if err := database.DB.Raw(query, args...).Scan(&timestampValue).Error; err == nil && !timestampValue.IsZero() {
					latestTimestamp = &timestampValue
				}
				
				if latestTimestamp != nil && !latestTimestamp.IsZero() {
					// Get all metrics at that timestamp
					if err := database.DB.Where("deployment_id = ? AND timestamp = ?", deploymentID, *latestTimestamp).
						Find(&latestMetrics).Error; err == nil && len(latestMetrics) > 0 {
						// Aggregate across all containers
						var sumCPU float64
						var sumMemory int64
						var sumNetworkRx int64
						var sumNetworkTx int64
						var sumDiskRead int64
						var sumDiskWrite int64
						var sumRequestCount int64
						var sumErrorCount int64

						for _, m := range latestMetrics {
							sumCPU += m.CPUUsage
							sumMemory += m.MemoryUsage
							sumNetworkRx += m.NetworkRxBytes
							sumNetworkTx += m.NetworkTxBytes
							sumDiskRead += m.DiskReadBytes
							sumDiskWrite += m.DiskWriteBytes
							sumRequestCount += m.RequestCount
							sumErrorCount += m.ErrorCount
						}

						avgCPU := sumCPU / float64(len(latestMetrics))
						lastSentTimestamp = *latestTimestamp

						metric := &deploymentsv1.DeploymentMetric{
							DeploymentId:     deploymentID,
							Timestamp:        timestamppb.New(*latestTimestamp),
							CpuUsagePercent:  avgCPU,
							MemoryUsageBytes: sumMemory,
							NetworkRxBytes:   sumNetworkRx,
							NetworkTxBytes:   sumNetworkTx,
							DiskReadBytes:    sumDiskRead,
							DiskWriteBytes:   sumDiskWrite,
						}
						if sumRequestCount > 0 {
							metric.RequestCount = &sumRequestCount
						}
						if sumErrorCount > 0 {
							metric.ErrorCount = &sumErrorCount
						}

						if err := stream.Send(metric); err != nil {
							return err
						}
						continue
					}
				}
			} else {
				// Single container mode: just get the latest metric
				err := nextQuery.First(&dbMetric).Error

				if err != nil {
					// If no record found, that's fine - just wait for next tick
					// Metrics may not be collected yet or no new data since last check
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						// Log actual errors (not "record not found")
						log.Printf("[StreamDeploymentMetrics] Error querying metrics for %s: %v", deploymentID, err)
					}
					continue
				}

				lastSentTimestamp = dbMetric.Timestamp

				metric := &deploymentsv1.DeploymentMetric{
					DeploymentId:     dbMetric.DeploymentID,
					Timestamp:        timestamppb.New(dbMetric.Timestamp),
					CpuUsagePercent:  dbMetric.CPUUsage,
					MemoryUsageBytes: dbMetric.MemoryUsage,
					NetworkRxBytes:   dbMetric.NetworkRxBytes,
					NetworkTxBytes:   dbMetric.NetworkTxBytes,
					DiskReadBytes:    dbMetric.DiskReadBytes,
					DiskWriteBytes:   dbMetric.DiskWriteBytes,
				}
				if dbMetric.ContainerID != "" {
					metric.ContainerId = &dbMetric.ContainerID
				}
				if dbMetric.RequestCount > 0 {
					metric.RequestCount = &dbMetric.RequestCount
				}
				if dbMetric.ErrorCount > 0 {
					metric.ErrorCount = &dbMetric.ErrorCount
				}

				if err := stream.Send(metric); err != nil {
					return err
				}
			}
		}
	}
}

// GetDeploymentUsage retrieves aggregated usage for a deployment
func (s *Service) GetDeploymentUsage(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentUsageRequest]) (*connect.Response[deploymentsv1.GetDeploymentUsageResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions (allow viewing usage for deleted deployments)
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists (include deleted for historical usage)
	dbDeployment, err := s.repo.GetByIDIncludeDeleted(ctx, deploymentID, true)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
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

	// Calculate usage from deployment_usage_hourly (single source of truth)
	// This works for both current and historical months
	var currentCPUCoreSeconds int64
	var currentMemoryByteSeconds int64
	var currentBandwidthRxBytes int64
	var currentBandwidthTxBytes int64
	var currentRequestCount int64
	var currentErrorCount int64
	var currentStorageBytes int64
	var currentUptimeSeconds int64

	if month == now.Format("2006-01") {
		// Current month: calculate live from hourly aggregates (full month) + raw metrics (recent)
		// Hourly aggregates are reliable and pre-calculated, raw metrics for most recent period
		rawCutoff := time.Now().Add(-24 * time.Hour)
		if rawCutoff.Before(monthStart) {
			rawCutoff = monthStart
		}

		// Get usage from hourly aggregates for full month (most efficient and accurate)
		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}
		database.DB.Table("deployment_usage_hourly duh").
			Select(`
				COALESCE(SUM((duh.avg_cpu_usage / 100.0) * 3600), 0) as cpu_core_seconds,
				COALESCE(SUM(duh.avg_memory_usage * 3600), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh.deployment_id = ? AND duh.hour >= ? AND duh.hour < ?", deploymentID, monthStart, rawCutoff).
			Scan(&hourlyUsage)

		// Get recent usage from raw metrics (last 24 hours - not yet aggregated)
		type recentMetric struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
			RequestCount      int64
			ErrorCount        int64
		}
		var recentUsage recentMetric
		
		// Calculate CPU and Memory from raw metrics (grouped by timestamp)
		type metricTimestamp struct {
			CPUUsage    float64
			MemorySum   int64
			Timestamp   time.Time
		}
		var metricTimestamps []metricTimestamp
		database.DB.Table("deployment_metrics dm").
			Select(`
				AVG(dm.cpu_usage) as cpu_usage,
				SUM(dm.memory_usage) as memory_sum,
				dm.timestamp as timestamp
			`).
			Where("dm.deployment_id = ? AND dm.timestamp >= ?", deploymentID, rawCutoff).
			Group("dm.timestamp").
			Order("dm.timestamp ASC").
			Scan(&metricTimestamps)
		
		// Calculate byte-seconds from timestamped metrics
		metricInterval := int64(5)
		for i, m := range metricTimestamps {
			interval := metricInterval
			if i > 0 {
				interval = int64(m.Timestamp.Sub(metricTimestamps[i-1].Timestamp).Seconds())
				if interval <= 0 {
					interval = metricInterval
				}
			}
			recentUsage.CPUCoreSeconds += int64((m.CPUUsage / 100.0) * float64(interval))
			recentUsage.MemoryByteSeconds += m.MemorySum * interval
		}

		// Get bandwidth and request counts from raw metrics
		database.DB.Table("deployment_metrics dm").
			Select(`
				COALESCE(SUM(dm.network_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(dm.network_tx_bytes), 0) as bandwidth_tx_bytes,
				COALESCE(SUM(dm.request_count), 0) as request_count,
				COALESCE(SUM(dm.error_count), 0) as error_count
			`).
			Where("dm.deployment_id = ? AND dm.timestamp >= ?", deploymentID, rawCutoff).
			Scan(&recentUsage)

		// Combine: hourly aggregates (older) + raw metrics (recent) = live current month usage
		currentCPUCoreSeconds = hourlyUsage.CPUCoreSeconds + recentUsage.CPUCoreSeconds
		currentMemoryByteSeconds = hourlyUsage.MemoryByteSeconds + recentUsage.MemoryByteSeconds
		currentBandwidthRxBytes = hourlyUsage.BandwidthRxBytes + recentUsage.BandwidthRxBytes
		currentBandwidthTxBytes = hourlyUsage.BandwidthTxBytes + recentUsage.BandwidthTxBytes
		currentRequestCount = recentUsage.RequestCount
		currentErrorCount = recentUsage.ErrorCount
		
		// Get storage from deployments table
		var storage struct {
			StorageBytes int64
		}
		database.DB.Table("deployments").
			Select("COALESCE(storage_bytes, 0) as storage_bytes").
			Where("id = ?", deploymentID).
			Scan(&storage)
		currentStorageBytes = storage.StorageBytes
		
		// Calculate uptime from deployment_locations
		var uptime struct {
			UptimeSeconds int64
		}
		database.DB.Table("deployment_locations dl").
			Select(`
				COALESCE(SUM(EXTRACT(EPOCH FROM (
					CASE 
						WHEN dl.status = 'running' THEN NOW() - dl.created_at
						WHEN dl.updated_at > dl.created_at THEN dl.updated_at - dl.created_at
						ELSE '0 seconds'::interval
					END
				))), 0)::bigint as uptime_seconds
			`).
			Where("dl.deployment_id = ? AND (dl.created_at >= ? OR dl.updated_at >= ?)", deploymentID, monthStart, monthStart).
			Scan(&uptime)
		currentUptimeSeconds = uptime.UptimeSeconds
	} else {
		// Historical month: calculate from deployment_usage_hourly
		// Get usage from hourly aggregates for the entire requested month
		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}
		database.DB.Table("deployment_usage_hourly duh").
			Select(`
				COALESCE(SUM((duh.avg_cpu_usage / 100.0) * 3600), 0) as cpu_core_seconds,
				COALESCE(SUM(duh.avg_memory_usage * 3600), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh.deployment_id = ? AND duh.hour >= ? AND duh.hour <= ?", deploymentID, requestedMonthStart, monthEnd).
			Scan(&hourlyUsage)
		
		currentCPUCoreSeconds = hourlyUsage.CPUCoreSeconds
		currentMemoryByteSeconds = hourlyUsage.MemoryByteSeconds
		currentBandwidthRxBytes = hourlyUsage.BandwidthRxBytes
		currentBandwidthTxBytes = hourlyUsage.BandwidthTxBytes
		
		// Get request/error counts from raw metrics for the month
		var reqCount struct {
			RequestCount int64
			ErrorCount   int64
		}
		database.DB.Table("deployment_metrics dm").
			Select(`
				COALESCE(SUM(dm.request_count), 0) as request_count,
				COALESCE(SUM(dm.error_count), 0) as error_count
			`).
			Where("dm.deployment_id = ? AND dm.timestamp >= ? AND dm.timestamp <= ?", deploymentID, requestedMonthStart, monthEnd).
			Scan(&reqCount)
		currentRequestCount = reqCount.RequestCount
		currentErrorCount = reqCount.ErrorCount
		
		// Get storage from deployments table (historical snapshot not available, use current)
		var storage struct {
			StorageBytes int64
		}
		database.DB.Table("deployments").
			Select("COALESCE(storage_bytes, 0) as storage_bytes").
			Where("id = ?", deploymentID).
			Scan(&storage)
		currentStorageBytes = storage.StorageBytes
		
		// Calculate uptime from deployment_locations for historical month
		var uptime struct {
			UptimeSeconds int64
		}
		database.DB.Table("deployment_locations dl").
			Select(`
				COALESCE(SUM(EXTRACT(EPOCH FROM (
					CASE 
						WHEN dl.status = 'running' AND dl.updated_at <= ? THEN ?::timestamp - dl.created_at
						WHEN dl.updated_at > dl.created_at AND dl.updated_at <= ? THEN dl.updated_at - dl.created_at
						WHEN dl.created_at >= ? AND dl.created_at < ? THEN ?::timestamp - dl.created_at
						ELSE '0 seconds'::interval
					END
				))), 0)::bigint as uptime_seconds
			`, monthEnd, monthEnd, monthEnd, requestedMonthStart, monthEnd, monthEnd).
			Where("dl.deployment_id = ? AND ((dl.created_at >= ? AND dl.created_at <= ?) OR (dl.updated_at >= ? AND dl.updated_at <= ?))", 
				deploymentID, requestedMonthStart, monthEnd, requestedMonthStart, monthEnd).
			Scan(&uptime)
		currentUptimeSeconds = uptime.UptimeSeconds
	}

	var estimatedMonthly *deploymentsv1.DeploymentUsageMetrics
	if month == now.Format("2006-01") {
		// Current month: project based on elapsed time using live calculated values
		elapsed := now.Sub(monthStart)
		monthDuration := monthEnd.Sub(monthStart)
		elapsedRatio := float64(elapsed) / float64(monthDuration)

		if elapsedRatio > 0 {
			estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
				CpuCoreSeconds:    int64(float64(currentCPUCoreSeconds) / elapsedRatio),
				MemoryByteSeconds: int64(float64(currentMemoryByteSeconds) / elapsedRatio),
				BandwidthRxBytes:  currentBandwidthRxBytes, // Bandwidth is cumulative, use current value for estimate
				BandwidthTxBytes:  currentBandwidthTxBytes,
				StorageBytes:      currentStorageBytes, // Storage is snapshot from deployments table
				RequestCount:      currentRequestCount,
				ErrorCount:        currentErrorCount,
				UptimeSeconds:     int64(float64(currentUptimeSeconds) / elapsedRatio), // Use calculated uptime
			}
		} else {
			estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
				CpuCoreSeconds:    currentCPUCoreSeconds,
				MemoryByteSeconds: currentMemoryByteSeconds,
				BandwidthRxBytes:  currentBandwidthRxBytes,
				BandwidthTxBytes:  currentBandwidthTxBytes,
				StorageBytes:      currentStorageBytes,
				RequestCount:      currentRequestCount,
				ErrorCount:        currentErrorCount,
				UptimeSeconds:     currentUptimeSeconds,
			}
		}
	} else {
		// Historical month: estimated equals current (from calculated data)
		estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
			CpuCoreSeconds:    currentCPUCoreSeconds,
			MemoryByteSeconds: currentMemoryByteSeconds,
			BandwidthRxBytes:  currentBandwidthRxBytes,
			BandwidthTxBytes:  currentBandwidthTxBytes,
			StorageBytes:      currentStorageBytes,
			RequestCount:      currentRequestCount,
			ErrorCount:        currentErrorCount,
			UptimeSeconds:     currentUptimeSeconds,
		}
	}

	// Calculate estimated cost using centralized pricing model
	pricingModel := pricing.GetPricing()
	estBandwidthBytes := estimatedMonthly.BandwidthRxBytes + estimatedMonthly.BandwidthTxBytes
	
	// Calculate per-resource costs for estimated monthly
	estCPUCost := pricingModel.CalculateCPUCost(estimatedMonthly.CpuCoreSeconds)
	estMemoryCost := pricingModel.CalculateMemoryCost(estimatedMonthly.MemoryByteSeconds)
	estBandwidthCost := pricingModel.CalculateBandwidthCost(estBandwidthBytes)
	estStorageCost := pricingModel.CalculateStorageCost(estimatedMonthly.StorageBytes) // Full month for estimate
	estimatedMonthly.EstimatedCostCents = estCPUCost + estMemoryCost + estBandwidthCost + estStorageCost
	
	// Set per-resource cost breakdown for estimated monthly
	cpuCostPtr := int64(estCPUCost)
	memoryCostPtr := int64(estMemoryCost)
	bandwidthCostPtr := int64(estBandwidthCost)
	storageCostPtr := int64(estStorageCost)
	estimatedMonthly.CpuCostCents = &cpuCostPtr
	estimatedMonthly.MemoryCostCents = &memoryCostPtr
	estimatedMonthly.BandwidthCostCents = &bandwidthCostPtr
	estimatedMonthly.StorageCostCents = &storageCostPtr

	// Calculate current cost using centralized pricing model with live calculated values
	// Note: Storage is billed monthly - for current cost, we need to prorate it
	// CPU/Memory are already time-based (core-seconds, byte-seconds) so no prorating needed
	// Bandwidth is one-time cost per byte transferred, no prorating needed
	// Storage is monthly cost per byte, so must prorate based on elapsed time
	currBandwidthBytes := currentBandwidthRxBytes + currentBandwidthTxBytes
	
	// Calculate elapsed ratio for storage prorating
	var elapsedRatio float64
	if month == now.Format("2006-01") {
		elapsed := now.Sub(monthStart)
		monthDuration := monthEnd.Sub(monthStart)
		elapsedRatio = float64(elapsed) / float64(monthDuration)
	} else {
		// Historical month: use full month (1.0) for prorating
		elapsedRatio = 1.0
	}
	
	// Calculate per-resource costs for current usage (using live calculated values)
	currCPUCost := pricingModel.CalculateCPUCost(currentCPUCoreSeconds)
	currMemoryCost := pricingModel.CalculateMemoryCost(currentMemoryByteSeconds)
	currBandwidthCost := pricingModel.CalculateBandwidthCost(currBandwidthBytes)
	currStorageFullMonth := pricingModel.CalculateStorageCost(currentStorageBytes)
	currStorageCost := int64(float64(currStorageFullMonth) * elapsedRatio) // Prorate for current month
	currentCostCents := currCPUCost + currMemoryCost + currBandwidthCost + currStorageCost

	currentMetrics := &deploymentsv1.DeploymentUsageMetrics{
		CpuCoreSeconds:     currentCPUCoreSeconds,
		MemoryByteSeconds:  currentMemoryByteSeconds,
		BandwidthRxBytes:   currentBandwidthRxBytes,
		BandwidthTxBytes:   currentBandwidthTxBytes,
		StorageBytes:       currentStorageBytes, // Storage from deployments table
		RequestCount:       currentRequestCount,
		ErrorCount:         currentErrorCount,
		UptimeSeconds:      currentUptimeSeconds, // Uptime calculated from deployment_locations
		EstimatedCostCents: currentCostCents, // Current usage cost (calculated server-side with live data)
	}
	
	// Set per-resource cost breakdown for current usage
	currCPUCostPtr := int64(currCPUCost)
	currMemoryCostPtr := int64(currMemoryCost)
	currBandwidthCostPtr := int64(currBandwidthCost)
	currStorageCostPtr := currStorageCost
	currentMetrics.CpuCostCents = &currCPUCostPtr
	currentMetrics.MemoryCostCents = &currMemoryCostPtr
	currentMetrics.BandwidthCostCents = &currBandwidthCostPtr
	currentMetrics.StorageCostCents = &currStorageCostPtr

	response := &deploymentsv1.GetDeploymentUsageResponse{
		DeploymentId:     deploymentID,
		OrganizationId:   orgID,
		Month:            month,
		Current:          currentMetrics,
		EstimatedMonthly: estimatedMonthly,
	}

	return connect.NewResponse(response), nil
}
