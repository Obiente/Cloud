package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// GetDatabaseUsage retrieves aggregated usage and billing for a database
func (s *Service) GetDatabaseUsage(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseUsageRequest]) (*connect.Response[databasesv1.GetDatabaseUsageResponse], error) {
	databaseID := req.Msg.GetDatabaseId()
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check permissions
	if err := s.checkDatabasePermission(ctx, databaseID, auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database to verify organization and get storage
	dbInstance, err := s.repo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("database does not belong to organization"))
	}

	// Determine month (default to current month)
	month := req.Msg.GetMonth()
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)

	// Parse requested month for historical queries
	requestedMonthStart := monthStart
	if month != now.Format("2006-01") {
		t, err := time.Parse("2006-01", month)
		if err == nil {
			requestedMonthStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			monthEnd = requestedMonthStart.AddDate(0, 1, 0).Add(-time.Second)
		}
	}

	// Calculate usage from hourly aggregates and raw metrics
	rawCutoff := time.Now().Add(-24 * time.Hour)
	if rawCutoff.Before(monthStart) {
		rawCutoff = monthStart
	}

	currentMetrics, err := common.CalculateUsageFromHourlyAndRaw(
		databaseID,
		"database",
		requestedMonthStart,
		monthEnd,
		rawCutoff,
		dbInstance.DiskBytes,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate usage: %w", err))
	}

	// Sum only the portions of running intervals that overlap the requested
	// month. This excludes stopped time and clamps long-running databases to the
	// month boundary.
	requestedMonthEnd := requestedMonthStart.AddDate(0, 1, 0)
	var uptimeIntervals []database.DatabaseUptimeInterval
	if err := database.DB.WithContext(ctx).
		Where("database_id = ? AND started_at < ? AND (ended_at IS NULL OR ended_at > ?)", databaseID, requestedMonthEnd, requestedMonthStart).
		Find(&uptimeIntervals).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to calculate database uptime: %w", err))
	}
	currentMetrics.UptimeSeconds = databaseUptimeSeconds(uptimeIntervals, requestedMonthStart, requestedMonthEnd, now)

	// Calculate estimated monthly usage
	var estimatedMonthly common.ContainerUsageMetrics
	isCurrentMonth := month == now.Format("2006-01")
	if isCurrentMonth {
		estimatedMonthly = common.CalculateEstimatedMonthly(currentMetrics, monthStart, monthEnd)
	} else {
		estimatedMonthly = currentMetrics
	}

	// Calculate costs
	currCPUCost, currMemoryCost, currBandwidthCost, currStorageCost, currTotalCost := common.CalculateCosts(currentMetrics, isCurrentMonth, monthStart, monthEnd)
	estCPUCost, estMemoryCost, estBandwidthCost, estStorageCost, estTotalCost := common.CalculateCosts(estimatedMonthly, false, monthStart, monthEnd)

	// Build response
	currentProto := &databasesv1.DatabaseUsageMetrics{
		CpuCoreSeconds:     currentMetrics.CPUCoreSeconds,
		MemoryByteSeconds:  currentMetrics.MemoryByteSeconds,
		BandwidthRxBytes:   currentMetrics.BandwidthRxBytes,
		BandwidthTxBytes:   currentMetrics.BandwidthTxBytes,
		StorageBytes:       currentMetrics.StorageBytes,
		UptimeSeconds:      currentMetrics.UptimeSeconds,
		EstimatedCostCents: currTotalCost,
	}
	currCPUCostPtr := int64(currCPUCost)
	currMemoryCostPtr := int64(currMemoryCost)
	currBandwidthCostPtr := int64(currBandwidthCost)
	currStorageCostPtr := int64(currStorageCost)
	currentProto.CpuCostCents = &currCPUCostPtr
	currentProto.MemoryCostCents = &currMemoryCostPtr
	currentProto.BandwidthCostCents = &currBandwidthCostPtr
	currentProto.StorageCostCents = &currStorageCostPtr

	estimatedProto := &databasesv1.DatabaseUsageMetrics{
		CpuCoreSeconds:     estimatedMonthly.CPUCoreSeconds,
		MemoryByteSeconds:  estimatedMonthly.MemoryByteSeconds,
		BandwidthRxBytes:   estimatedMonthly.BandwidthRxBytes,
		BandwidthTxBytes:   estimatedMonthly.BandwidthTxBytes,
		StorageBytes:       estimatedMonthly.StorageBytes,
		UptimeSeconds:      estimatedMonthly.UptimeSeconds,
		EstimatedCostCents: estTotalCost,
	}
	estCPUCostPtr := int64(estCPUCost)
	estMemoryCostPtr := int64(estMemoryCost)
	estBandwidthCostPtr := int64(estBandwidthCost)
	estStorageCostPtr := int64(estStorageCost)
	estimatedProto.CpuCostCents = &estCPUCostPtr
	estimatedProto.MemoryCostCents = &estMemoryCostPtr
	estimatedProto.BandwidthCostCents = &estBandwidthCostPtr
	estimatedProto.StorageCostCents = &estStorageCostPtr

	response := &databasesv1.GetDatabaseUsageResponse{
		DatabaseId:       databaseID,
		OrganizationId:   orgID,
		Month:            month,
		Current:          currentProto,
		EstimatedMonthly: estimatedProto,
	}

	return connect.NewResponse(response), nil
}

func databaseUptimeSeconds(intervals []database.DatabaseUptimeInterval, periodStart, periodEnd, now time.Time) int64 {
	var uptime time.Duration
	for _, interval := range intervals {
		start := interval.StartedAt
		if start.Before(periodStart) {
			start = periodStart
		}
		end := now
		if interval.EndedAt != nil {
			end = *interval.EndedAt
		}
		if end.After(periodEnd) {
			end = periodEnd
		}
		if end.After(start) {
			uptime += end.Sub(start)
		}
	}
	return int64(uptime / time.Second)
}
