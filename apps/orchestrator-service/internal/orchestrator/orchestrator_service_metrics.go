package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	dockerclient "github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	shared "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
)

// Metrics operations for orchestrator service

func (os *OrchestratorService) aggregateUsage() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Aggregating usage...")
			now := time.Now()
			month := now.Format("2006-01")

			// Calculate the start of the current month for accurate aggregation
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

			// Get all organizations with deployments (active or historical for the month)
			type orgRow struct {
				OrganizationID string
				Count          int
			}
			var orgRows []orgRow
			// Get organizations with running deployments now, plus those with usage records this month
			database.DB.Table("deployment_locations dl").
				Select("d.organization_id, COUNT(*) as count").
				Joins("JOIN deployments d ON d.id = dl.deployment_id").
				Where("dl.status = ?", "running").
				Group("d.organization_id").Scan(&orgRows)

			// Also include organizations that have usage records this month (for historical tracking)
			var existingOrgs []string
			database.DB.Table("usage_monthly").
				Where("month = ?", month).
				Distinct("organization_id").
				Pluck("organization_id", &existingOrgs)

			orgMap := make(map[string]int)
			for _, r := range orgRows {
				orgMap[r.OrganizationID] = r.Count
			}
			for _, orgID := range existingOrgs {
				if _, exists := orgMap[orgID]; !exists {
					orgMap[orgID] = 0
				}
			}

			for orgID := range orgMap {
				// Get all deployments for this organization (running or that ran this month)
				var deploymentIDs []string
				database.DB.Table("deployments d").
					Select("d.id").
					Where("d.organization_id = ?", orgID).
					Pluck("d.id", &deploymentIDs)

				if len(deploymentIDs) == 0 {
					continue
				}

				// Get actual CPU/memory allocations from deployments table
				type deploymentAlloc struct {
					ID          string
					CPUShares   *int64
					MemoryBytes *int64
				}
				var allocs []deploymentAlloc
				database.DB.Table("deployments").
					Select("id, cpu_shares, memory_bytes").
					Where("id IN ?", deploymentIDs).
					Scan(&allocs)

				allocMap := make(map[string]struct {
					cpuShares   int64
					memoryBytes int64
				})
				for _, a := range allocs {
					cpu := int64(1) // Default 1 core if not specified
					if a.CPUShares != nil && *a.CPUShares > 0 {
						// CPU shares: typically 1024 = 1 core, but we'll use as-is
						cpu = *a.CPUShares / 1024
						if cpu < 1 {
							cpu = 1
						}
					}
					mem := int64(512 * 1024 * 1024) // Default 512MB
					if a.MemoryBytes != nil && *a.MemoryBytes > 0 {
						mem = *a.MemoryBytes
					}
					allocMap[a.ID] = struct {
						cpuShares   int64
						memoryBytes int64
					}{cpu, mem}
				}

				// Calculate CPU core-seconds and memory byte-seconds from actual runtime
				// We track from deployment_locations: sum all runtime periods this month
				type locationRuntime struct {
					DeploymentID string
					CreatedAt    time.Time
					UpdatedAt    time.Time
					Status       string
				}
				var locations []locationRuntime
				database.DB.Table("deployment_locations dl").
					Select("dl.deployment_id, dl.created_at, dl.updated_at, dl.status").
					Joins("JOIN deployments d ON d.id = dl.deployment_id").
					Where("d.organization_id = ? AND (dl.created_at >= ? OR dl.updated_at >= ?)", orgID, monthStart, monthStart).
					Order("dl.deployment_id, dl.created_at").
					Scan(&locations)

				var totalCPUSeconds int64

				// Group by deployment and calculate total runtime
				// Sum all runtime periods for each deployment this month
				runtimeByDeployment := make(map[string]int64)
				for _, loc := range locations {
					// Calculate the time this location was active during the month
					locationStart := loc.CreatedAt
					if locationStart.Before(monthStart) {
						locationStart = monthStart // Only count from month start
					}

					locationEnd := now
					if loc.Status != "running" {
						// If stopped, use the update time (when it stopped)
						if loc.UpdatedAt.After(locationStart) {
							locationEnd = loc.UpdatedAt
						} else {
							// Updated before month start or before location start - skip
							continue
						}
					}

					// Ensure we don't count future times
					if locationEnd.After(now) {
						locationEnd = now
					}

					if locationEnd.After(locationStart) {
						runtimeSeconds := int64(locationEnd.Sub(locationStart).Seconds())
						runtimeByDeployment[loc.DeploymentID] += runtimeSeconds
					}
				}

				// Calculate CPU totals using allocations (CPU is based on allocation)
				for deploymentID, runtimeSeconds := range runtimeByDeployment {
					alloc, exists := allocMap[deploymentID]
					if !exists {
						// Default allocation if not found
						alloc = struct {
							cpuShares   int64
							memoryBytes int64
						}{1, 512 * 1024 * 1024} // 1 core, 512MB default
					}
					totalCPUSeconds += alloc.cpuShares * runtimeSeconds
				}

				// Calculate memory byte-seconds from actual metrics (not allocated memory)
				// Aggregate from raw metrics (recent) + hourly aggregates (older) for the month
				var orgMemoryByteSeconds int64

				// Sum from raw metrics (last 24 hours)
				// Calculate memory byte-seconds by summing memory per distinct timestamp * interval
				rawCutoff := time.Now().Add(-24 * time.Hour)
				if rawCutoff.Before(monthStart) {
					rawCutoff = monthStart
				}

				type orgMemoryPerTimestamp struct {
					MemorySum int64
					Timestamp time.Time
				}
				var orgMemoryTimestamps []orgMemoryPerTimestamp
				metricsDB := database.GetMetricsDB()
				metricsDB.Table("deployment_metrics dm").
					Select(`
						SUM(dm.memory_usage) as memory_sum,
						dm.timestamp as timestamp
					`).
					Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawCutoff).
					Group("dm.timestamp").
					Order("dm.timestamp ASC").
					Scan(&orgMemoryTimestamps)

				// Calculate byte-seconds: for each timestamp, use memory * 5 seconds (average interval)
				intervalSeconds := int64(5) // Average interval between metric samples
				for _, m := range orgMemoryTimestamps {
					orgMemoryByteSeconds += m.MemorySum * intervalSeconds
				}

				// Sum from hourly aggregates (older than 24 hours, within month)
				if rawCutoff.After(monthStart) {
					var hourlyMemorySum int64
					metricsDB.Table("deployment_usage_hourly duh").
						Select("COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds").
						Where("duh.deployment_id IN ? AND duh.hour >= ? AND duh.hour < ?", deploymentIDs, monthStart, rawCutoff).
						Scan(&hourlyMemorySum)
					orgMemoryByteSeconds += hourlyMemorySum
				}

				// Aggregate bandwidth from raw metrics (recent) + hourly aggregates (older) for the month
				type bandwidthRow struct {
					RxBytes int64
					TxBytes int64
				}

				// Sum from raw metrics (last 24 hours)
				var rawBandwidth bandwidthRow
				rawCutoffBandwidth := time.Now().Add(-24 * time.Hour)
				if rawCutoffBandwidth.Before(monthStart) {
					rawCutoffBandwidth = monthStart
				}
				metricsDB.Table("deployment_metrics dm").
					Select("COALESCE(SUM(dm.network_rx_bytes), 0) as rx_bytes, COALESCE(SUM(dm.network_tx_bytes), 0) as tx_bytes").
					Where("dm.deployment_id IN ? AND dm.timestamp >= ?", deploymentIDs, rawCutoffBandwidth).
					Scan(&rawBandwidth)

				// Sum from hourly aggregates (older than 24 hours, within month)
				var hourlyBandwidth bandwidthRow
				if rawCutoffBandwidth.After(monthStart) {
					metricsDB.Table("deployment_usage_hourly duh").
						Select("COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as rx_bytes, COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as tx_bytes").
						Where("duh.deployment_id IN ? AND duh.hour >= ? AND duh.hour < ?", deploymentIDs, monthStart, rawCutoffBandwidth).
						Scan(&hourlyBandwidth)
				}

			}
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) aggregateGameServerMetrics(gameServerID string, aggregateCutoff time.Time) (aggregated int, deleted int64) {
	// Get org ID once
	var orgID string
	database.DB.Table("game_servers").
		Select("organization_id").
		Where("id = ?", gameServerID).
		Pluck("organization_id", &orgID)

	if orgID == "" {
		return 0, 0
	}

	// Find the oldest metric for this game server
	var oldestTime time.Time
	metricsDB := database.GetMetricsDB()
	metricsDB.Table("game_server_metrics").
		Select("MIN(timestamp)").
		Where("game_server_id = ? AND timestamp < ?", gameServerID, aggregateCutoff).
		Scan(&oldestTime)

	if oldestTime.IsZero() {
		return 0, 0
	}

	// Aggregate hour by hour
	currentHour := oldestTime.Truncate(time.Hour)
	deletedInGameServer := int64(0)
	aggregatedCount := 0

	for currentHour.Before(aggregateCutoff) {
		nextHour := currentHour.Add(1 * time.Hour)

		// Check if hourly aggregate already exists
		var existingHourly database.GameServerUsageHourly
		err := metricsDB.Where("game_server_id = ? AND hour = ?", gameServerID, currentHour).
			First(&existingHourly).Error

		// Check if raw metrics still exist for this hour (if not, we can't recalculate)
		var rawMetricsCount int64
		metricsDB.Table("game_server_metrics").
			Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?", gameServerID, currentHour, nextHour).
			Count(&rawMetricsCount)

		// Only create/recalculate if raw metrics exist
		if rawMetricsCount > 0 {
			// Delete existing aggregate if present (to allow recalculation)
			if err == nil {
				metricsDB.Where("game_server_id = ? AND hour = ?", gameServerID, currentHour).
					Delete(&database.GameServerUsageHourly{})
			}

			// Aggregate metrics for this hour
			type hourlyAgg struct {
				AvgCPUUsage    float64
				SumMemoryUsage float64
				AvgMemoryUsage float64
				SumNetworkRx   int64
				SumNetworkTx   int64
				SumDiskRead    int64
				SumDiskWrite   int64
				Count          int64
				TimestampCount int64
			}
			var agg hourlyAgg

			err := metricsDB.Table("game_server_metrics").
				Select(`
					AVG(cpu_usage) as avg_cpu_usage,
					AVG(memory_usage) as avg_memory_usage,
					SUM(memory_usage) as sum_memory_usage,
					COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
					COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
					COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
					COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
					COUNT(*) as count,
					COUNT(DISTINCT timestamp) as timestamp_count
				`).
				Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
				Scan(&agg).Error

			if err != nil {
				logger.Warn("[Orchestrator] Failed to query game server metrics for %s at %s: %v", gameServerID, currentHour, err)
				currentHour = nextHour
				continue
			}

			if agg.Count == 0 {
				currentHour = nextHour
				continue
			}

			// Calculate CPU core-seconds and memory byte-seconds using actual timestamp intervals
			type metricTimestamp struct {
				CPUUsage  float64
				MemorySum int64
				Timestamp time.Time
			}
			var timestamps []metricTimestamp

			err = metricsDB.Table("game_server_metrics").
				Select("cpu_usage, memory_usage, timestamp").
				Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
				Order("timestamp ASC").
				Scan(&timestamps).Error

			var cpuCoreSeconds float64
			var memoryByteSeconds int64

			if err == nil && len(timestamps) > 1 {
				// Calculate intervals between timestamps
				for i := 0; i < len(timestamps)-1; i++ {
					interval := int64(timestamps[i+1].Timestamp.Sub(timestamps[i].Timestamp).Seconds())
					if interval <= 0 {
						interval = 5 // Default 5-second interval
					}

					// Validate CPU usage before calculating (filter out invalid values)
					if timestamps[i].CPUUsage >= 0 && timestamps[i].CPUUsage <= 10000 {
						cpuCoreSeconds += (timestamps[i].CPUUsage / 100.0) * float64(interval)
					}
					memoryByteSeconds += timestamps[i].MemorySum * interval
				}

				// Handle last timestamp: if it's not at the end of the hour, use remaining time
				if len(timestamps) > 0 {
					lastTimestamp := timestamps[len(timestamps)-1].Timestamp
					timeRemaining := int64(nextHour.Sub(lastTimestamp).Seconds())
					if timeRemaining > 0 && timeRemaining <= 3600 {
						lastCPU := timestamps[len(timestamps)-1].CPUUsage
						lastMemory := timestamps[len(timestamps)-1].MemorySum
						// Validate CPU usage before calculating
						if lastCPU >= 0 && lastCPU <= 10000 {
							cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
						}
						memoryByteSeconds += lastMemory * timeRemaining
					}
				}

				// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
				if cpuCoreSeconds > 0 {
					agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
				}

				// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
				if memoryByteSeconds > 0 {
					agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
				} else if agg.TimestampCount > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					estimatedTimestamps := float64(agg.Count) / 1.0
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else {
					agg.AvgMemoryUsage = 0
				}
			} else {
				// Fallback calculation
				if agg.TimestampCount > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					estimatedTimestamps := float64(agg.Count) / 1.0
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				}
			}

			hourlyUsage := database.GameServerUsageHourly{
				GameServerID:     gameServerID,
				OrganizationID:   orgID,
				Hour:             currentHour,
				AvgCPUUsage:      agg.AvgCPUUsage,
				AvgMemoryUsage:   agg.AvgMemoryUsage,
				BandwidthRxBytes: agg.SumNetworkRx,
				BandwidthTxBytes: agg.SumNetworkTx,
				DiskReadBytes:    agg.SumDiskRead,
				DiskWriteBytes:   agg.SumDiskWrite,
				SampleCount:      agg.Count,
			}

			if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
				logger.Warn("[Orchestrator] Failed to create hourly aggregate for game server %s at %s: %v", gameServerID, currentHour, err)
			} else {
				aggregatedCount++

				// Delete the raw metrics for this hour in batch
				result := metricsDB.Where("game_server_id = ? AND timestamp >= ? AND timestamp < ?",
					gameServerID, currentHour, nextHour).
					Delete(&database.GameServerMetrics{})

				if result.Error == nil {
					deletedInGameServer += result.RowsAffected
				}
			}
		}

		currentHour = nextHour
	}

	return aggregatedCount, deletedInGameServer
}

func (os *OrchestratorService) aggregateVPSMetrics(vpsID string, aggregateCutoff time.Time) (aggregated int, deleted int64) {
	// Get org ID once
	var orgID string
	database.DB.Table("vps_instances").
		Select("organization_id").
		Where("id = ?", vpsID).
		Pluck("organization_id", &orgID)

	if orgID == "" {
		return 0, 0
	}

	// Find the oldest metric for this VPS
	var oldestTime time.Time
	metricsDB := database.GetMetricsDB()
	metricsDB.Table("vps_metrics").
		Select("MIN(timestamp)").
		Where("vps_instance_id = ? AND timestamp < ?", vpsID, aggregateCutoff).
		Scan(&oldestTime)

	if oldestTime.IsZero() {
		return 0, 0
	}

	// Aggregate hour by hour
	currentHour := oldestTime.Truncate(time.Hour)
	deletedInVPS := int64(0)
	aggregatedCount := 0

	for currentHour.Before(aggregateCutoff) {
		nextHour := currentHour.Add(1 * time.Hour)

		// Check if hourly aggregate already exists
		var existingHourly database.VPSUsageHourly
		err := metricsDB.Where("vps_instance_id = ? AND hour = ?", vpsID, currentHour).
			First(&existingHourly).Error

		// Check if raw metrics still exist for this hour (if not, we can't recalculate)
		var rawMetricsCount int64
		metricsDB.Table("vps_metrics").
			Where("vps_instance_id = ? AND timestamp >= ? AND timestamp < ?", vpsID, currentHour, nextHour).
			Count(&rawMetricsCount)

		// Only create/recalculate if raw metrics exist
		if rawMetricsCount > 0 {
			// Delete existing aggregate if present (to allow recalculation)
			if err == nil {
				metricsDB.Where("vps_instance_id = ? AND hour = ?", vpsID, currentHour).
					Delete(&database.VPSUsageHourly{})
			}

			// Aggregate metrics for this hour
			type hourlyAgg struct {
				AvgCPUUsage    float64
				SumMemoryUsage float64
				AvgMemoryUsage float64
				SumNetworkRx    int64
				SumNetworkTx    int64
				SumDiskRead     int64
				SumDiskWrite    int64
				Count           int64
				TimestampCount  int64
			}
			var agg hourlyAgg

			err := metricsDB.Table("vps_metrics").
				Select(`
					AVG(cpu_usage) as avg_cpu_usage,
					AVG(memory_used) as avg_memory_usage,
					SUM(memory_used) as sum_memory_usage,
					COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
					COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
					COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
					COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
					COUNT(*) as count,
					COUNT(DISTINCT timestamp) as timestamp_count
				`).
				Where("vps_instance_id = ? AND timestamp >= ? AND timestamp < ?",
					vpsID, currentHour, nextHour).
				Scan(&agg).Error

			if err != nil {
				logger.Warn("[Orchestrator] Failed to query VPS metrics for %s at %s: %v", vpsID, currentHour, err)
				currentHour = nextHour
				continue
			}

			if agg.Count == 0 {
				currentHour = nextHour
				continue
			}

			// Calculate CPU core-seconds and memory byte-seconds using actual timestamp intervals
			type metricTimestamp struct {
				CPUUsage  float64
				MemorySum int64
				Timestamp time.Time
			}
			var timestamps []metricTimestamp

			err = metricsDB.Table("vps_metrics").
				Select("cpu_usage, memory_used, timestamp").
				Where("vps_instance_id = ? AND timestamp >= ? AND timestamp < ?",
					vpsID, currentHour, nextHour).
				Order("timestamp ASC").
				Scan(&timestamps).Error

			var cpuCoreSeconds float64
			var memoryByteSeconds int64

			if err == nil && len(timestamps) > 1 {
				// Calculate intervals between timestamps
				for i := 0; i < len(timestamps)-1; i++ {
					interval := int64(timestamps[i+1].Timestamp.Sub(timestamps[i].Timestamp).Seconds())
					if interval <= 0 {
						interval = 5 // Default 5-second interval
					}

					// Validate CPU usage before calculating (filter out invalid values)
					if timestamps[i].CPUUsage >= 0 && timestamps[i].CPUUsage <= 10000 {
						cpuCoreSeconds += (timestamps[i].CPUUsage / 100.0) * float64(interval)
					}
					memoryByteSeconds += timestamps[i].MemorySum * interval
				}

				// Handle last timestamp: if it's not at the end of the hour, use remaining time
				if len(timestamps) > 0 {
					lastTimestamp := timestamps[len(timestamps)-1]
					remainingSeconds := int64(nextHour.Sub(lastTimestamp.Timestamp).Seconds())
					if remainingSeconds > 0 && remainingSeconds <= 3600 {
						if lastTimestamp.CPUUsage >= 0 && lastTimestamp.CPUUsage <= 10000 {
							cpuCoreSeconds += (lastTimestamp.CPUUsage / 100.0) * float64(remainingSeconds)
						}
						memoryByteSeconds += lastTimestamp.MemorySum * remainingSeconds
					}
				}
			} else {
				// Fallback: use average CPU and memory with estimated intervals
				metricInterval := int64(5) // Default 5-second interval
				if agg.TimestampCount > 0 {
					estimatedIntervals := float64(agg.TimestampCount) * float64(metricInterval)
					cpuCoreSeconds = (agg.AvgCPUUsage / 100.0) * estimatedIntervals
					memoryByteSeconds = int64(agg.AvgMemoryUsage * estimatedIntervals)
				} else if agg.Count > 0 {
					estimatedIntervals := float64(agg.Count) * float64(metricInterval)
					cpuCoreSeconds = (agg.AvgCPUUsage / 100.0) * estimatedIntervals
					memoryByteSeconds = int64(agg.AvgMemoryUsage * estimatedIntervals)
				}
			}

			// Calculate average memory usage (byte-seconds per hour / 3600)
			if cpuCoreSeconds > 0 || memoryByteSeconds > 0 {
				agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
			} else {
				// Fallback calculation
				if agg.TimestampCount > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					estimatedTimestamps := float64(agg.Count) / 1.0
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else {
					agg.AvgMemoryUsage = 0
				}
			}

			// Calculate uptime seconds (sum of intervals)
			uptimeSeconds := int64(0)
			if len(timestamps) > 1 {
				for i := 0; i < len(timestamps)-1; i++ {
					interval := int64(timestamps[i+1].Timestamp.Sub(timestamps[i].Timestamp).Seconds())
					if interval <= 0 {
						interval = 5
					}
					uptimeSeconds += interval
				}
				if len(timestamps) > 0 {
					lastTimestamp := timestamps[len(timestamps)-1]
					remainingSeconds := int64(nextHour.Sub(lastTimestamp.Timestamp).Seconds())
					if remainingSeconds > 0 && remainingSeconds <= 3600 {
						uptimeSeconds += remainingSeconds
					}
				}
			} else if agg.Count > 0 {
				// Fallback: estimate uptime
				uptimeSeconds = int64(agg.Count * 5) // Assume 5-second intervals
			}

			hourlyUsage := database.VPSUsageHourly{
				VPSInstanceID:   vpsID,
				OrganizationID:  orgID,
				Hour:            currentHour,
				AvgCPUUsage:     agg.AvgCPUUsage,
				AvgMemoryUsage:  agg.AvgMemoryUsage,
				BandwidthRxBytes: agg.SumNetworkRx,
				BandwidthTxBytes: agg.SumNetworkTx,
				DiskReadBytes:    agg.SumDiskRead,
				DiskWriteBytes:   agg.SumDiskWrite,
				UptimeSeconds:   uptimeSeconds,
				SampleCount:     agg.Count,
			}

			if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
				logger.Warn("[Orchestrator] Failed to create hourly aggregate for VPS %s at %s: %v", vpsID, currentHour, err)
			} else {
				aggregatedCount++

				// Delete the raw metrics for this hour in batch
				result := metricsDB.Where("vps_instance_id = ? AND timestamp >= ? AND timestamp < ?",
					vpsID, currentHour, nextHour).
					Delete(&database.VPSMetrics{})

				if result.Error == nil {
					deletedInVPS += result.RowsAffected
				}
			}
		}

		currentHour = nextHour
	}

	return aggregatedCount, deletedInVPS
}

func (os *OrchestratorService) aggregateDeploymentMetrics(deploymentID string, aggregateCutoff time.Time) (aggregated int, deleted int64) {
	// Get org ID once
	var orgID string
	database.DB.Table("deployments").
		Select("organization_id").
		Where("id = ?", deploymentID).
		Pluck("organization_id", &orgID)

	// Find the oldest metric for this deployment
	var oldestTime time.Time
	metricsDB := database.GetMetricsDB()
	metricsDB.Table("deployment_metrics").
		Select("MIN(timestamp)").
		Where("deployment_id = ? AND timestamp < ?", deploymentID, aggregateCutoff).
		Scan(&oldestTime)

	if oldestTime.IsZero() {
		return 0, 0
	}

	// Aggregate hour by hour
	currentHour := oldestTime.Truncate(time.Hour)
	deletedInDeployment := int64(0)
	aggregatedCount := 0

	for currentHour.Before(aggregateCutoff) {
		nextHour := currentHour.Add(1 * time.Hour)

		// Check if hourly aggregate already exists
		var existingHourly database.DeploymentUsageHourly
		err := metricsDB.Where("deployment_id = ? AND hour = ?", deploymentID, currentHour).
			First(&existingHourly).Error

		// Check if raw metrics still exist for this hour (if not, we can't recalculate)
		var rawMetricsCount int64
		metricsDB.Table("deployment_metrics").
			Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", deploymentID, currentHour, nextHour).
			Count(&rawMetricsCount)

		// Only create/recalculate if raw metrics exist (allows recalculation of existing aggregates)
		if rawMetricsCount > 0 {
			// Delete existing aggregate if present (to allow recalculation with corrected logic)
			if err == nil {
				metricsDB.Where("deployment_id = ? AND hour = ?", deploymentID, currentHour).
					Delete(&database.DeploymentUsageHourly{})
			}
			// Aggregate metrics for this hour
			// First, get basic aggregates
			type hourlyAgg struct {
				AvgCPUUsage     float64
				SumMemoryUsage  float64
				AvgMemoryUsage  float64
				SumNetworkRx    int64
				SumNetworkTx    int64
				SumDiskRead     int64
				SumDiskWrite    int64
				SumRequestCount int64
				SumErrorCount   int64
				Count           int64
				TimestampCount  int64
			}
			var agg hourlyAgg

			err := metricsDB.Table("deployment_metrics").
				Select(`
					AVG(cpu_usage) as avg_cpu_usage,
					AVG(memory_usage) as avg_memory_usage,
					SUM(memory_usage) as sum_memory_usage,
					COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
					COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
					COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
					COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
					COALESCE(SUM(request_count), 0) as sum_request_count,
					COALESCE(SUM(error_count), 0) as sum_error_count,
					COUNT(*) as count,
					COUNT(DISTINCT timestamp) as timestamp_count
				`).
				Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
					deploymentID, currentHour, nextHour).
				Scan(&agg).Error

			if err == nil && agg.Count > 0 {
				// Calculate CPU core-seconds and memory byte-seconds properly using actual timestamp intervals
				// Fetch timestamps ordered to calculate intervals
				type metricTimestamp struct {
					CPUUsage  float64
					MemorySum int64
					Timestamp time.Time
				}
				var timestamps []metricTimestamp
				metricsDB.Table("deployment_metrics").
					Select(`
						AVG(cpu_usage) as cpu_usage,
						SUM(memory_usage) as memory_sum,
						timestamp
					`).
					Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
						deploymentID, currentHour, nextHour).
					Group("timestamp").
					Order("timestamp ASC").
					Scan(&timestamps)

				// Calculate core-seconds and byte-seconds from actual intervals
				var cpuCoreSeconds float64
				var memoryByteSeconds int64
				metricInterval := int64(5) // Default 5 seconds
				if len(timestamps) > 0 {
					for i, ts := range timestamps {
						interval := metricInterval
						if i > 0 {
							intervalSeconds := int64(ts.Timestamp.Sub(timestamps[i-1].Timestamp).Seconds())
							if intervalSeconds > 0 && intervalSeconds <= 3600 { // Sanity check: max 1 hour
								interval = intervalSeconds
							}
						} else if i == 0 && len(timestamps) == 1 {
							// Single timestamp: use default interval
							interval = metricInterval
						}
						cpuCoreSeconds += (ts.CPUUsage / 100.0) * float64(interval)
						memoryByteSeconds += ts.MemorySum * interval
					}
					// Handle last timestamp: if it's not at the end of the hour, use remaining time
					if len(timestamps) > 0 {
						lastTimestamp := timestamps[len(timestamps)-1].Timestamp
						timeRemaining := int64(nextHour.Sub(lastTimestamp).Seconds())
						if timeRemaining > 0 && timeRemaining <= 3600 {
							lastCPU := timestamps[len(timestamps)-1].CPUUsage
							lastMemory := timestamps[len(timestamps)-1].MemorySum
							cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
							memoryByteSeconds += lastMemory * timeRemaining
						}
					}
				}

				// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
				// This way, when we query with SUM((avg_cpu_usage / 100.0) * 3600), we get correct core-seconds
				if cpuCoreSeconds > 0 {
					agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
				}

				// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
				// This way, when we query with SUM(avg_memory_usage * 3600), we get correct byte-seconds
				if memoryByteSeconds > 0 {
					agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
				} else if agg.TimestampCount > 0 {
					// Fallback: calculate byte-seconds from average memory and default interval
					// If we have timestamps but couldn't calculate intervals, use 5-second default
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
					// Assume 5-second intervals, so byte-seconds = avgMemory * interval * timestamps
					// But we want byte-seconds/hour, so: (avgMemory * 5 * timestamps) / 3600
					metricInterval := int64(5)
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else if agg.Count > 0 {
					// Last resort: use average memory and assume 5-second intervals
					avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
					metricInterval := int64(5)
					// Estimate timestamps: assume metrics are evenly distributed over the hour
					estimatedTimestamps := float64(agg.Count) / 1.0 // 1 metric per timestamp
					fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
					agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
				} else {
					// No metrics at all - set to 0
					agg.AvgMemoryUsage = 0
				}

				hourlyUsage := database.DeploymentUsageHourly{
					DeploymentID:     deploymentID,
					OrganizationID:   orgID,
					Hour:             currentHour,
					AvgCPUUsage:      agg.AvgCPUUsage,
					AvgMemoryUsage:   agg.AvgMemoryUsage,
					BandwidthRxBytes: agg.SumNetworkRx,
					BandwidthTxBytes: agg.SumNetworkTx,
					DiskReadBytes:    agg.SumDiskRead,
					DiskWriteBytes:   agg.SumDiskWrite,
					RequestCount:     agg.SumRequestCount,
					ErrorCount:       agg.SumErrorCount,
					SampleCount:      agg.Count,
				}

				if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
					logger.Warn("[Orchestrator] Failed to create hourly aggregate for %s at %s: %v", deploymentID, currentHour, err)
				} else {
					aggregatedCount++

					// Delete the raw metrics for this hour in batch
					result := metricsDB.Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
						deploymentID, currentHour, nextHour).
						Delete(&database.DeploymentMetrics{})

					if result.Error == nil {
						deletedInDeployment += result.RowsAffected
					}
				}
			}
		}

		currentHour = nextHour
	}

	return aggregatedCount, deletedInDeployment
}

func (os *OrchestratorService) backfillMissingHourlyAggregates() {
	logger.Info("[Orchestrator] Starting backfill of missing hourly aggregates...")

	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		logger.Warn("[Orchestrator] Metrics database not available, skipping backfill")
		return
	}

	// Verify tables exist
	var metricsTableExists bool
	var hourlyTableExists bool
	metricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'deployment_metrics'
		)
	`).Scan(&metricsTableExists)
	metricsDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'deployment_usage_hourly'
		)
	`).Scan(&hourlyTableExists)

	logger.Debug("[Orchestrator] Tables exist: deployment_metrics=%v, deployment_usage_hourly=%v", metricsTableExists, hourlyTableExists)

	if !metricsTableExists {
		logger.Warn("[Orchestrator] deployment_metrics table does not exist, skipping backfill")
		return
	}

	if !hourlyTableExists {
		logger.Warn("[Orchestrator] deployment_usage_hourly table does not exist, cannot create aggregates")
		return
	}

	// Get current time and calculate backfill range (current month start to now)
	// Include ALL hours in the current month, not just older than 24 hours
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	backfillCutoff := now.Truncate(time.Hour) // Current hour (not 24 hours ago)

	logger.Debug("[Orchestrator] Backfill range: %s to %s", monthStart, backfillCutoff)

	// First check if there are any metrics at all
	var totalMetricsCount int64
	metricsDB.Table("deployment_metrics").
		Where("timestamp >= ? AND timestamp < ?", monthStart, backfillCutoff.Add(1*time.Hour)).
		Count(&totalMetricsCount)
	logger.Debug("[Orchestrator] Found %d total raw metrics in backfill range", totalMetricsCount)

	if totalMetricsCount == 0 {
		logger.Warn("[Orchestrator] No raw metrics found in backfill range - deployment_usage_hourly will remain empty")
		return
	}

	// Get all deployments that have metrics in the backfill range (entire current month)
	var deploymentIDs []string
	metricsDB.Table("deployment_metrics").
		Select("DISTINCT deployment_id").
		Where("timestamp >= ? AND timestamp < ?", monthStart, backfillCutoff.Add(1*time.Hour)).
		Pluck("deployment_id", &deploymentIDs)

	if len(deploymentIDs) == 0 {
		logger.Warn("[Orchestrator] No deployments with metrics to backfill (found %d total metrics but no distinct deployments)", totalMetricsCount)
		return
	}

	logger.Info("[Orchestrator] Found %d deployments with metrics for backfill", len(deploymentIDs))

	logger.Info("[Orchestrator] Backfilling aggregates for %d deployments", len(deploymentIDs))

	totalAggregated := 0
	const batchSize = 10 // Process 10 deployments concurrently
	var aggMutex sync.Mutex

	// Process in batches
	for i := 0; i < len(deploymentIDs); i += batchSize {
		end := i + batchSize
		if end > len(deploymentIDs) {
			end = len(deploymentIDs)
		}
		batch := deploymentIDs[i:end]

		var wg sync.WaitGroup
		for _, deploymentID := range batch {
			wg.Add(1)
			go func(depID string) {
				defer wg.Done()

				// Get org ID for this deployment
				var orgID string
				database.DB.Table("deployments").
					Select("organization_id").
					Where("id = ?", depID).
					Pluck("organization_id", &orgID)

				if orgID == "" {
					logger.Warn("[Orchestrator] Deployment %s has no organization_id, skipping", depID)
					return
				}

				// Find hours with raw metrics but missing aggregates
				// Iterate through each hour in the backfill range (entire current month)
				aggregatedForDeployment := 0
				currentHour := monthStart.Truncate(time.Hour)

				for currentHour.Before(backfillCutoff) || currentHour.Equal(backfillCutoff) {
					nextHour := currentHour.Add(1 * time.Hour)

					// Check if aggregate already exists
					var existingHourly database.DeploymentUsageHourly
					err := metricsDB.Where("deployment_id = ? AND hour = ?", depID, currentHour).
						First(&existingHourly).Error

					// If aggregate exists, skip (don't recalculate on startup)
					if err == nil {
						currentHour = nextHour
						continue
					}

					// Check if raw metrics exist for this hour
					var rawMetricsCount int64
					metricsDB.Table("deployment_metrics").
						Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", depID, currentHour, nextHour).
						Count(&rawMetricsCount)

					if rawMetricsCount > 0 {
						// Create aggregate for this hour
						aggregated, err := os.aggregateSingleHour(depID, orgID, currentHour, nextHour)
						if aggregated {
							aggregatedForDeployment++
						} else if err != nil {
							logger.Warn("[Orchestrator] Failed to aggregate hour %s for deployment %s: %v", currentHour, depID, err)
						}
					}

					currentHour = nextHour
				}

				if aggregatedForDeployment > 0 {
					aggMutex.Lock()
					totalAggregated += aggregatedForDeployment
					aggMutex.Unlock()
					logger.Info("[Orchestrator] Backfilled %d hours for deployment %s", aggregatedForDeployment, depID)
				} else {
					// Log why no aggregates were created
					var totalHoursChecked int
					var hoursWithMetrics int
					var hoursWithAggregates int
					checkHour := monthStart.Truncate(time.Hour)
					for checkHour.Before(backfillCutoff) || checkHour.Equal(backfillCutoff) {
						totalHoursChecked++
						var rawCount int64
						metricsDB.Table("deployment_metrics").
							Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?", depID, checkHour, checkHour.Add(1*time.Hour)).
							Count(&rawCount)
						if rawCount > 0 {
							hoursWithMetrics++
						}
						var aggExists database.DeploymentUsageHourly
						if err := metricsDB.Where("deployment_id = ? AND hour = ?", depID, checkHour).First(&aggExists).Error; err == nil {
							hoursWithAggregates++
						}
						checkHour = checkHour.Add(1 * time.Hour)
					}
					logger.Debug("[Orchestrator] Deployment %s: checked %d hours, %d with metrics, %d with aggregates", depID, totalHoursChecked, hoursWithMetrics, hoursWithAggregates)
				}
			}(deploymentID)
		}
		wg.Wait()
	}

	if totalAggregated > 0 {
		logger.Info("[Orchestrator] ✓ Backfill completed successfully: created %d hourly aggregates", totalAggregated)
	} else {
		logger.Warn("[Orchestrator] ⚠️ Backfill completed but no aggregates were created (found %d deployments with %d raw metrics)",
			len(deploymentIDs), totalMetricsCount)
		logger.Warn("[Orchestrator] This may indicate that aggregates already exist or there's an issue with aggregation")
	}
}

func (os *OrchestratorService) aggregateSingleHour(deploymentID, orgID string, hourStart, hourEnd time.Time) (bool, error) {
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		return false, fmt.Errorf("metrics database not available")
	}

	logger.Debug("[Orchestrator] Aggregating hour %s for deployment %s", hourStart, deploymentID)

	// Get basic aggregates
	type hourlyAgg struct {
		AvgCPUUsage     float64
		SumMemoryUsage  float64
		AvgMemoryUsage  float64
		SumNetworkRx    int64
		SumNetworkTx    int64
		SumDiskRead     int64
		SumDiskWrite    int64
		SumRequestCount int64
		SumErrorCount   int64
		Count           int64
		TimestampCount  int64
	}
	var agg hourlyAgg

	err := metricsDB.Table("deployment_metrics").
		Select(`
			AVG(cpu_usage) as avg_cpu_usage,
			AVG(memory_usage) as avg_memory_usage,
			SUM(memory_usage) as sum_memory_usage,
			COALESCE(SUM(network_rx_bytes), 0) as sum_network_rx,
			COALESCE(SUM(network_tx_bytes), 0) as sum_network_tx,
			COALESCE(SUM(disk_read_bytes), 0) as sum_disk_read,
			COALESCE(SUM(disk_write_bytes), 0) as sum_disk_write,
			COALESCE(SUM(request_count), 0) as sum_request_count,
			COALESCE(SUM(error_count), 0) as sum_error_count,
			COUNT(*) as count,
			COUNT(DISTINCT timestamp) as timestamp_count
		`).
		Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
			deploymentID, hourStart, hourEnd).
		Scan(&agg).Error

	if err != nil {
		logger.Warn("[Orchestrator] Failed to query metrics for %s at %s: %v", deploymentID, hourStart, err)
		return false, err
	}

	if agg.Count == 0 {
		logger.Debug("[Orchestrator] No metrics found for %s at %s", deploymentID, hourStart)
		return false, nil
	}

	logger.Debug("[Orchestrator] Found %d metrics for %s at %s", agg.Count, deploymentID, hourStart)

	// Calculate CPU core-seconds and memory byte-seconds using actual timestamp intervals
	type metricTimestamp struct {
		CPUUsage  float64
		MemorySum int64
		Timestamp time.Time
	}
	var timestamps []metricTimestamp
	metricsDB.Table("deployment_metrics").
		Select(`
			AVG(cpu_usage) as cpu_usage,
			SUM(memory_usage) as memory_sum,
			timestamp
		`).
		Where("deployment_id = ? AND timestamp >= ? AND timestamp < ?",
			deploymentID, hourStart, hourEnd).
		Group("timestamp").
		Order("timestamp ASC").
		Scan(&timestamps)

	// Calculate core-seconds and byte-seconds from actual intervals
	var cpuCoreSeconds float64
	var memoryByteSeconds int64
	metricInterval := int64(5) // Default 5 seconds
	if len(timestamps) > 0 {
		// First timestamp: use time from hour start to first timestamp, or default interval
		firstTimestamp := timestamps[0].Timestamp
		firstInterval := int64(firstTimestamp.Sub(hourStart).Seconds())
		if firstInterval <= 0 {
			firstInterval = metricInterval
		} else if firstInterval > 3600 {
			firstInterval = metricInterval // Sanity check
		}
		cpuCoreSeconds += (timestamps[0].CPUUsage / 100.0) * float64(firstInterval)
		memoryByteSeconds += timestamps[0].MemorySum * firstInterval

		// Subsequent timestamps: use actual interval between timestamps
		// For each interval from timestamps[i-1] to timestamps[i], use memory[i-1] (the value at the start of the interval)
		for i := 1; i < len(timestamps); i++ {
			interval := metricInterval
			intervalSeconds := int64(timestamps[i].Timestamp.Sub(timestamps[i-1].Timestamp).Seconds())
			if intervalSeconds > 0 && intervalSeconds <= 3600 { // Sanity check: max 1 hour
				interval = intervalSeconds
			}
			// Use memory from the PREVIOUS timestamp for this interval (memory[i-1] represents memory from timestamps[i-1] to timestamps[i])
			cpuCoreSeconds += (timestamps[i-1].CPUUsage / 100.0) * float64(interval)
			memoryByteSeconds += timestamps[i-1].MemorySum * interval
		}
		// Handle last timestamp: if it's not at the end of the hour, use remaining time
		// Note: We've already counted the interval from timestamps[last-1] to timestamps[last],
		// so we only need to add the remaining time from timestamps[last] to hourEnd
		if len(timestamps) > 0 {
			lastTimestamp := timestamps[len(timestamps)-1].Timestamp
			timeRemaining := int64(hourEnd.Sub(lastTimestamp).Seconds())
			if timeRemaining > 0 && timeRemaining <= 3600 {
				lastCPU := timestamps[len(timestamps)-1].CPUUsage
				lastMemory := timestamps[len(timestamps)-1].MemorySum
				cpuCoreSeconds += (lastCPU / 100.0) * float64(timeRemaining)
				memoryByteSeconds += lastMemory * timeRemaining
			}
		}
	}

	// Store avg_cpu_usage as average CPU for the hour (core-seconds / 3600)
	if cpuCoreSeconds > 0 {
		agg.AvgCPUUsage = (cpuCoreSeconds / 3600.0) * 100.0 // Convert back to percentage
	}

	// Store avg_memory_usage as average memory for the hour (byte-seconds / 3600)
	// This way, when we query with SUM(avg_memory_usage * 3600), we get correct byte-seconds
	if memoryByteSeconds > 0 {
		agg.AvgMemoryUsage = float64(memoryByteSeconds) / 3600.0
	} else if agg.TimestampCount > 0 {
		// Fallback: calculate byte-seconds from average memory and default interval
		// If we have timestamps but couldn't calculate intervals, use 5-second default
		avgMemoryBytes := agg.SumMemoryUsage / float64(agg.TimestampCount)
		// Assume 5-second intervals, so byte-seconds = avgMemory * interval * timestamps
		// But we want byte-seconds/hour, so: (avgMemory * 5 * timestamps) / 3600
		metricInterval := int64(5)
		fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * float64(agg.TimestampCount))
		agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
	} else if agg.Count > 0 {
		// Last resort: use average memory and assume 5-second intervals
		avgMemoryBytes := agg.SumMemoryUsage / float64(agg.Count)
		metricInterval := int64(5)
		// Estimate timestamps: assume metrics are evenly distributed over the hour
		estimatedTimestamps := float64(agg.Count) / 1.0 // 1 metric per timestamp
		fallbackByteSeconds := int64(avgMemoryBytes * float64(metricInterval) * estimatedTimestamps)
		agg.AvgMemoryUsage = float64(fallbackByteSeconds) / 3600.0
	} else {
		// No metrics at all - set to 0
		agg.AvgMemoryUsage = 0
	}

	hourlyUsage := database.DeploymentUsageHourly{
		DeploymentID:     deploymentID,
		OrganizationID:   orgID,
		Hour:             hourStart,
		AvgCPUUsage:      agg.AvgCPUUsage,
		AvgMemoryUsage:   agg.AvgMemoryUsage,
		BandwidthRxBytes: agg.SumNetworkRx,
		BandwidthTxBytes: agg.SumNetworkTx,
		DiskReadBytes:    agg.SumDiskRead,
		DiskWriteBytes:   agg.SumDiskWrite,
		RequestCount:     agg.SumRequestCount,
		ErrorCount:       agg.SumErrorCount,
		SampleCount:      agg.Count,
	}

	if err := metricsDB.Create(&hourlyUsage).Error; err != nil {
		logger.Warn("[Orchestrator] Failed to create hourly aggregate for %s at %s: %v", deploymentID, hourStart, err)
		return false, err
	}

	logger.Debug("[Orchestrator] ✓ Created hourly aggregate for %s at %s (CPU: %.2f%%, Memory: %.2f bytes/sec, SampleCount: %d)",
		deploymentID, hourStart, agg.AvgCPUUsage, agg.AvgMemoryUsage, agg.Count)
	return true, nil
}

func (os *OrchestratorService) getContainerStats(containerID string) (*shared.ContainerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get container stats (stream=false means get one snapshot)
	statsResp, err := os.serviceRegistry.DockerClient().ContainerStats(ctx, containerID, dockerclient.ContainerStatsOptions{Stream: false})
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	// Decode stats JSON response - ContainerStats returns JSON that matches StatsJSON structure
	var statsJSON struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  uint   `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		BlkioStats struct {
			IoServiceBytesRecursive []struct {
				Major uint64 `json:"major"`
				Minor uint64 `json:"minor"`
				Op    string `json:"op"`
				Value uint64 `json:"value"`
			} `json:"io_service_bytes_recursive"`
		} `json:"blkio_stats"`
	}
	if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Calculate CPU usage percentage
	// Docker CPU stats are in nanoseconds. We need to validate the deltas to prevent division by tiny numbers
	cpuUsage := 0.0
	if statsJSON.PreCPUStats.SystemUsage > 0 && statsJSON.CPUStats.SystemUsage > statsJSON.PreCPUStats.SystemUsage {
		cpuDelta := int64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := int64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
		
		// Validate deltas to prevent invalid calculations
		// Minimum systemDelta: 1 millisecond (1,000,000 nanoseconds) to prevent division by tiny numbers
		// This ensures we have at least 1ms of time between measurements
		const minSystemDelta = 1_000_000 // 1ms in nanoseconds
		
		if systemDelta >= minSystemDelta && statsJSON.CPUStats.OnlineCPUs > 0 {
			// Handle counter wraparound (uint64 overflow) - if cpuDelta is negative, counters wrapped
			if cpuDelta < 0 {
				// Counter wraparound detected - skip this calculation
				logger.Warn("[getContainerStats] CPU counter wraparound detected for container %s (cpuDelta: %d, systemDelta: %d), skipping", containerID[:12], cpuDelta, systemDelta)
				cpuUsage = 0.0
			} else {
				// Calculate CPU usage: (cpuDelta / systemDelta) * numCPUs * 100
				// cpuDelta and systemDelta are both in nanoseconds, so the ratio is dimensionless
				// Multiplying by OnlineCPUs gives us the effective CPU usage across all cores
				cpuUsage = (float64(cpuDelta) / float64(systemDelta)) * float64(statsJSON.CPUStats.OnlineCPUs) * 100.0
				
				// Validate the result is physically reasonable
				// Maximum possible CPU usage: OnlineCPUs * 100% (all cores at 100%)
				maxReasonableCPU := float64(statsJSON.CPUStats.OnlineCPUs) * 100.0
				if cpuUsage < 0 {
					cpuUsage = 0.0
				} else if cpuUsage > maxReasonableCPU {
					// Log detailed error for debugging
					logger.Warn("[getContainerStats] Invalid CPU usage %.2f%% (max reasonable: %.2f%%) for container %s - cpuDelta: %d, systemDelta: %d, OnlineCPUs: %d. Skipping this measurement.",
						cpuUsage, maxReasonableCPU, containerID[:12], cpuDelta, systemDelta, statsJSON.CPUStats.OnlineCPUs)
					cpuUsage = 0.0 // Set to 0 instead of clamping to prevent cost calculation errors
				}
			}
		} else if systemDelta > 0 && systemDelta < minSystemDelta {
			// systemDelta too small - likely measurement error or very short time window
			logger.Debug("[getContainerStats] systemDelta too small (%d ns < %d ns) for container %s, skipping CPU calculation", systemDelta, minSystemDelta, containerID[:12])
			cpuUsage = 0.0
		}
	}

	// Calculate network bytes (sum across all interfaces)
	var networkRx, networkTx int64
	for _, netStats := range statsJSON.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
	}

	// Calculate disk I/O (read and write)
	// Docker's BlkioStats typically uses lowercase operation names: "read", "write"
	// Use case-insensitive matching to handle variations
	var diskRead, diskWrite int64
	for _, ioStat := range statsJSON.BlkioStats.IoServiceBytesRecursive {
		op := strings.ToLower(strings.TrimSpace(ioStat.Op))
		switch op {
		case "read":
			diskRead += int64(ioStat.Value)
		case "write":
			diskWrite += int64(ioStat.Value)
		}
	}

	// Log if we're not getting any disk I/O data (for debugging)
	if len(statsJSON.BlkioStats.IoServiceBytesRecursive) == 0 {
		logger.Debug("[getContainerStats] No BlkioStats found for container %s - disk I/O may not be available", containerID[:12])
	} else if diskRead == 0 && diskWrite == 0 {
		// Log operation names we're seeing (for debugging)
		ops := make([]string, 0, len(statsJSON.BlkioStats.IoServiceBytesRecursive))
		for _, ioStat := range statsJSON.BlkioStats.IoServiceBytesRecursive {
			ops = append(ops, ioStat.Op)
		}
		logger.Debug("[getContainerStats] BlkioStats available for container %s but no read/write found. Operations seen: %v", containerID[:12], ops)
	}

	return &shared.ContainerStats{
		CPUUsage:       cpuUsage,
		MemoryUsage:    int64(statsJSON.MemoryStats.Usage),
		NetworkRxBytes: networkRx,
		NetworkTxBytes: networkTx,
		DiskReadBytes:  diskRead,
		DiskWriteBytes: diskWrite,
	}, nil
}
