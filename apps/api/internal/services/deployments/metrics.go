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
				var latestTimestamp time.Time
				if err := query.Select("MAX(timestamp)").Scan(&latestTimestamp).Error; err == nil && !latestTimestamp.IsZero() {
					// Get all metrics at that timestamp
					if err := database.DB.Where("deployment_id = ? AND timestamp = ?", deploymentID, latestTimestamp).
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
						lastSentTimestamp = latestTimestamp

						metric := &deploymentsv1.DeploymentMetric{
							DeploymentId:     deploymentID,
							Timestamp:        timestamppb.New(latestTimestamp),
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

	// Get usage from DeploymentUsage table
	var usage database.DeploymentUsage
	err = database.DB.Where("deployment_id = ? AND month = ?", deploymentID, month).First(&usage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return zero usage if no record exists
			usage = database.DeploymentUsage{
				DeploymentID:      deploymentID,
				OrganizationID:    orgID,
				Month:             month,
				CPUCoreSeconds:    0,
				MemoryByteSeconds: 0,
				BandwidthRxBytes:  0,
				BandwidthTxBytes:  0,
				StorageBytes:      0,
				RequestCount:      0,
				ErrorCount:        0,
				UptimeSeconds:     0,
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query usage: %w", err))
		}
	}

	// Calculate estimated monthly usage based on current month progress
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	var estimatedMonthly *deploymentsv1.DeploymentUsageMetrics
	if month == now.Format("2006-01") {
		// Current month: project based on elapsed time
		elapsed := now.Sub(monthStart)
		monthDuration := monthEnd.Sub(monthStart)
		elapsedRatio := float64(elapsed) / float64(monthDuration)

		if elapsedRatio > 0 {
			estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
				CpuCoreSeconds:    int64(float64(usage.CPUCoreSeconds) / elapsedRatio),
				MemoryByteSeconds: int64(float64(usage.MemoryByteSeconds) / elapsedRatio),
				BandwidthRxBytes:  usage.BandwidthRxBytes, // Bandwidth is cumulative, not time-based for estimation
				BandwidthTxBytes:  usage.BandwidthTxBytes,
				StorageBytes:      usage.StorageBytes, // Storage is not time-based
				RequestCount:      usage.RequestCount,
				ErrorCount:        usage.ErrorCount,
				UptimeSeconds:     int64(float64(usage.UptimeSeconds) / elapsedRatio),
			}
		} else {
			estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
				CpuCoreSeconds:    usage.CPUCoreSeconds,
				MemoryByteSeconds: usage.MemoryByteSeconds,
				BandwidthRxBytes:  usage.BandwidthRxBytes,
				BandwidthTxBytes:  usage.BandwidthTxBytes,
				StorageBytes:      usage.StorageBytes,
				RequestCount:      usage.RequestCount,
				ErrorCount:        usage.ErrorCount,
				UptimeSeconds:     usage.UptimeSeconds,
			}
		}
	} else {
		// Historical month: estimated equals current
		estimatedMonthly = &deploymentsv1.DeploymentUsageMetrics{
			CpuCoreSeconds:    usage.CPUCoreSeconds,
			MemoryByteSeconds: usage.MemoryByteSeconds,
			BandwidthRxBytes:  usage.BandwidthRxBytes,
			BandwidthTxBytes:  usage.BandwidthTxBytes,
			StorageBytes:      usage.StorageBytes,
			RequestCount:      usage.RequestCount,
			ErrorCount:        usage.ErrorCount,
			UptimeSeconds:     usage.UptimeSeconds,
		}
	}

	// Calculate estimated cost using centralized pricing model
	pricingModel := pricing.GetPricing()
	estBandwidthBytes := estimatedMonthly.BandwidthRxBytes + estimatedMonthly.BandwidthTxBytes
	estimatedMonthly.EstimatedCostCents = pricingModel.CalculateTotalCost(
		estimatedMonthly.CpuCoreSeconds,
		estimatedMonthly.MemoryByteSeconds,
		estBandwidthBytes,
		estimatedMonthly.StorageBytes,
	)

	// Calculate current cost using centralized pricing model
	currBandwidthBytes := usage.BandwidthRxBytes + usage.BandwidthTxBytes
	currentCostCents := pricingModel.CalculateTotalCost(
		usage.CPUCoreSeconds,
		usage.MemoryByteSeconds,
		currBandwidthBytes,
		usage.StorageBytes,
	)

	currentMetrics := &deploymentsv1.DeploymentUsageMetrics{
		CpuCoreSeconds:     usage.CPUCoreSeconds,
		MemoryByteSeconds:  usage.MemoryByteSeconds,
		BandwidthRxBytes:   usage.BandwidthRxBytes,
		BandwidthTxBytes:   usage.BandwidthTxBytes,
		StorageBytes:       usage.StorageBytes,
		RequestCount:       usage.RequestCount,
		ErrorCount:         usage.ErrorCount,
		UptimeSeconds:      usage.UptimeSeconds,
		EstimatedCostCents: currentCostCents, // Current usage cost (calculated server-side)
	}

	response := &deploymentsv1.GetDeploymentUsageResponse{
		DeploymentId:     deploymentID,
		OrganizationId:   orgID,
		Month:            month,
		Current:          currentMetrics,
		EstimatedMonthly: estimatedMonthly,
	}

	return connect.NewResponse(response), nil
}
