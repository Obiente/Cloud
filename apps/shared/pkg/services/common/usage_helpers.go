package common

import (
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/pricing"
)

// ContainerUsageMetrics represents aggregated usage metrics for a container-based resource
type ContainerUsageMetrics struct {
	CPUCoreSeconds    int64
	MemoryByteSeconds  int64
	BandwidthRxBytes   int64
	BandwidthTxBytes   int64
	StorageBytes       int64
	UptimeSeconds      int64
}

// CalculateUsageFromHourlyAndRaw calculates usage metrics from hourly aggregates and raw metrics
// This is a shared function that can be used by both deployments and gameservers
func CalculateUsageFromHourlyAndRaw(
	resourceID string,
	resourceType string, // "deployment" or "gameserver"
	monthStart time.Time,
	monthEnd time.Time,
	rawCutoff time.Time,
	storageBytes int64, // Storage from the resource table
) (ContainerUsageMetrics, error) {
	var metrics ContainerUsageMetrics
	metrics.StorageBytes = storageBytes

	metricsDB := database.GetMetricsDB()
	now := time.Now()
	isCurrentMonth := monthStart.Year() == now.Year() && monthStart.Month() == now.Month()

	if isCurrentMonth {
		// Current month: calculate from hourly aggregates (older) + raw metrics (recent)
		
		// Get usage from hourly aggregates for the period before raw cutoff
		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}

		tableName := "deployment_usage_hourly"
		idColumn := "deployment_id"
		if resourceType == "gameserver" {
			tableName = "game_server_usage_hourly"
			idColumn = "game_server_id"
		}

		query := metricsDB.Table(tableName + " duh").
			Select(`
				COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
				COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh."+idColumn+" = ? AND duh.hour >= ? AND duh.hour < ?", resourceID, monthStart, rawCutoff)

		if err := query.Scan(&hourlyUsage).Error; err != nil {
			return metrics, err
		}

		// Get recent usage from raw metrics (last 24 hours - not yet aggregated)
		rawTableName := "deployment_metrics"
		if resourceType == "gameserver" {
			rawTableName = "game_server_metrics"
		}

		// Calculate CPU and Memory from raw metrics (grouped by timestamp)
		type metricTimestamp struct {
			CPUUsage  float64
			MemorySum int64
			Timestamp time.Time
		}
		var metricTimestamps []metricTimestamp
		rawQuery := metricsDB.Table(rawTableName + " dm").
			Select(`
				AVG(dm.cpu_usage) as cpu_usage,
				SUM(dm.memory_usage) as memory_sum,
				dm.timestamp as timestamp
			`).
			Where("dm."+idColumn+" = ? AND dm.timestamp >= ?", resourceID, rawCutoff).
			Group("dm.timestamp").
			Order("dm.timestamp ASC")

		if err := rawQuery.Scan(&metricTimestamps).Error; err != nil {
			return metrics, err
		}

		// Calculate byte-seconds from timestamped metrics
		metricInterval := int64(5)
		var recentCPUCoreSeconds int64
		var recentMemoryByteSeconds int64

		if len(metricTimestamps) > 0 {
			// First timestamp: use time from rawCutoff to first timestamp, or default interval
			firstTimestamp := metricTimestamps[0].Timestamp
			firstInterval := int64(firstTimestamp.Sub(rawCutoff).Seconds())
			if firstInterval <= 0 {
				firstInterval = metricInterval
			} else if firstInterval > 3600 {
				firstInterval = metricInterval // Sanity check
			}
			recentCPUCoreSeconds += int64((metricTimestamps[0].CPUUsage / 100.0) * float64(firstInterval))
			recentMemoryByteSeconds += metricTimestamps[0].MemorySum * firstInterval

			// Subsequent timestamps: use actual interval between timestamps
			for i := 1; i < len(metricTimestamps); i++ {
				interval := metricInterval
				intervalSeconds := int64(metricTimestamps[i].Timestamp.Sub(metricTimestamps[i-1].Timestamp).Seconds())
				if intervalSeconds > 0 && intervalSeconds <= 3600 {
					interval = intervalSeconds
				}
				// Use memory from the PREVIOUS timestamp for this interval
				recentCPUCoreSeconds += int64((metricTimestamps[i-1].CPUUsage / 100.0) * float64(interval))
				recentMemoryByteSeconds += metricTimestamps[i-1].MemorySum * interval
			}
		}

		// Get bandwidth from raw metrics
		var recentBandwidth struct {
			BandwidthRxBytes int64
			BandwidthTxBytes int64
		}
		metricsDB.Table(rawTableName + " dm").
			Select(`
				COALESCE(SUM(dm.network_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(dm.network_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("dm."+idColumn+" = ? AND dm.timestamp >= ?", resourceID, rawCutoff).
			Scan(&recentBandwidth)

		// Combine: hourly aggregates (older) + raw metrics (recent)
		metrics.CPUCoreSeconds = hourlyUsage.CPUCoreSeconds + recentCPUCoreSeconds
		metrics.MemoryByteSeconds = hourlyUsage.MemoryByteSeconds + recentMemoryByteSeconds
		metrics.BandwidthRxBytes = hourlyUsage.BandwidthRxBytes + recentBandwidth.BandwidthRxBytes
		metrics.BandwidthTxBytes = hourlyUsage.BandwidthTxBytes + recentBandwidth.BandwidthTxBytes
	} else {
		// Historical month: calculate from hourly aggregates only
		tableName := "deployment_usage_hourly"
		idColumn := "deployment_id"
		if resourceType == "gameserver" {
			tableName = "game_server_usage_hourly"
			idColumn = "game_server_id"
		}

		var hourlyUsage struct {
			CPUCoreSeconds    int64
			MemoryByteSeconds int64
			BandwidthRxBytes  int64
			BandwidthTxBytes  int64
		}
		query := metricsDB.Table(tableName + " duh").
			Select(`
				COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
				COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
				COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
				COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
			`).
			Where("duh."+idColumn+" = ? AND duh.hour >= ? AND duh.hour <= ?", resourceID, monthStart, monthEnd)

		if err := query.Scan(&hourlyUsage).Error; err != nil {
			return metrics, err
		}

		metrics.CPUCoreSeconds = hourlyUsage.CPUCoreSeconds
		metrics.MemoryByteSeconds = hourlyUsage.MemoryByteSeconds
		metrics.BandwidthRxBytes = hourlyUsage.BandwidthRxBytes
		metrics.BandwidthTxBytes = hourlyUsage.BandwidthTxBytes
	}

	return metrics, nil
}

// CalculateEstimatedMonthly projects current usage to full month
func CalculateEstimatedMonthly(current ContainerUsageMetrics, monthStart time.Time, monthEnd time.Time) ContainerUsageMetrics {
	now := time.Now()
	elapsed := now.Sub(monthStart)
	monthDuration := monthEnd.Sub(monthStart)
	elapsedRatio := float64(elapsed) / float64(monthDuration)

	if elapsedRatio > 0 {
		return ContainerUsageMetrics{
			CPUCoreSeconds:    int64(float64(current.CPUCoreSeconds) / elapsedRatio),
			MemoryByteSeconds: int64(float64(current.MemoryByteSeconds) / elapsedRatio),
			BandwidthRxBytes:  current.BandwidthRxBytes, // Bandwidth is cumulative
			BandwidthTxBytes:  current.BandwidthTxBytes,
			StorageBytes:      current.StorageBytes,
			UptimeSeconds:     int64(float64(current.UptimeSeconds) / elapsedRatio),
		}
	}
	return current
}

// CalculateCosts calculates costs for usage metrics using the pricing model
func CalculateCosts(metrics ContainerUsageMetrics, isCurrentMonth bool, monthStart time.Time, monthEnd time.Time) (int64, int64, int64, int64, int64) {
	pricingModel := pricing.GetPricing()
	bandwidthBytes := metrics.BandwidthRxBytes + metrics.BandwidthTxBytes

	cpuCost := pricingModel.CalculateCPUCost(metrics.CPUCoreSeconds)
	memoryCost := pricingModel.CalculateMemoryCost(metrics.MemoryByteSeconds)
	bandwidthCost := pricingModel.CalculateBandwidthCost(bandwidthBytes)

	// Storage cost needs prorating for current month
	var storageCost int64
	if isCurrentMonth {
		now := time.Now()
		elapsed := now.Sub(monthStart)
		monthDuration := monthEnd.Sub(monthStart)
		elapsedRatio := float64(elapsed) / float64(monthDuration)
		storageFullMonth := pricingModel.CalculateStorageCost(metrics.StorageBytes)
		storageCost = int64(float64(storageFullMonth) * elapsedRatio)
	} else {
		storageCost = pricingModel.CalculateStorageCost(metrics.StorageBytes)
	}

	totalCost := cpuCost + memoryCost + bandwidthCost + storageCost

	return cpuCost, memoryCost, bandwidthCost, storageCost, totalCost
}

