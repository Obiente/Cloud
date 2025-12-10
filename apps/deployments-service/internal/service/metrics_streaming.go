package deployments

import (
	"context"
	"log"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// streamLiveMetrics streams metrics directly from the live metrics streamer
func (s *Service) streamLiveMetrics(
	ctx context.Context,
	stream *connect.ServerStream[deploymentsv1.DeploymentMetric],
	deploymentID string,
	targetContainerID string,
	shouldAggregate bool,
) error {
	metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
	if metricsStreamer == nil {
		// Should not happen, but fallback gracefully
		return connect.NewError(connect.CodeInternal, nil)
	}

	// Subscribe to live metrics for this deployment
	metricChan := metricsStreamer.Subscribe(deploymentID)
	defer metricsStreamer.Unsubscribe(deploymentID, metricChan)

	// Send initial metric from latest live cache
	latestMetrics := metricsStreamer.GetLatestMetrics(deploymentID)
	if len(latestMetrics) > 0 {
		// Get the most recent metric(s)
		latest := latestMetrics[len(latestMetrics)-1]

		// Apply container filter if specified
		if targetContainerID == "" || latest.ContainerID == targetContainerID {
			var metric *deploymentsv1.DeploymentMetric

			if shouldAggregate {
				// Aggregate all metrics at this timestamp
				metricsAtTime := make([]orchestrator.LiveMetric, 0)
				for _, m := range latestMetrics {
					if m.Timestamp.Equal(latest.Timestamp) {
						metricsAtTime = append(metricsAtTime, m)
					}
				}

				var sumCPU float64
				var sumMemory int64
				var sumNetworkRx int64
				var sumNetworkTx int64
				var sumDiskRead int64
				var sumDiskWrite int64

				for _, m := range metricsAtTime {
					sumCPU += m.CPUUsage
					sumMemory += m.MemoryUsage
					sumNetworkRx += m.NetworkRxBytes
					sumNetworkTx += m.NetworkTxBytes
					sumDiskRead += m.DiskReadBytes
					sumDiskWrite += m.DiskWriteBytes
				}

				avgCPU := sumCPU / float64(len(metricsAtTime))

				metric = &deploymentsv1.DeploymentMetric{
					DeploymentId:     deploymentID,
					Timestamp:        timestamppb.New(latest.Timestamp),
					CpuUsagePercent:  avgCPU,
					MemoryUsageBytes: sumMemory,
					NetworkRxBytes:   sumNetworkRx,
					NetworkTxBytes:   sumNetworkTx,
					DiskReadBytes:    sumDiskRead,
					DiskWriteBytes:   sumDiskWrite,
				}
			} else {
				// Single container metric
				metric = &deploymentsv1.DeploymentMetric{
					DeploymentId:     deploymentID,
					Timestamp:        timestamppb.New(latest.Timestamp),
					CpuUsagePercent:  latest.CPUUsage,
					MemoryUsageBytes: latest.MemoryUsage,
					NetworkRxBytes:   latest.NetworkRxBytes,
					NetworkTxBytes:   latest.NetworkTxBytes,
					DiskReadBytes:    latest.DiskReadBytes,
					DiskWriteBytes:   latest.DiskWriteBytes,
				}
			}

			if metric != nil {
				if err := stream.Send(metric); err != nil {
					return err
				}
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
				// No metrics received for 60 seconds - log a warning
				// Note: We don't send a heartbeat for deployments as they may legitimately have no activity
				log.Printf("[streamLiveMetrics] No metrics received for %v for deployment %s", timeSinceLastMetric, deploymentID)
			}
		case liveMetric, ok := <-metricChan:
			if !ok {
				// Channel closed
				return nil
			}

			lastMetricTime = time.Now() // Update last metric time
			
			// Apply container filter
			if targetContainerID != "" && liveMetric.ContainerID != targetContainerID {
				continue
			}

			// If aggregating, we need to collect all metrics at this timestamp
			if shouldAggregate {
				// Get all metrics at the same timestamp
				allMetricsAtTime := metricsStreamer.GetLatestMetrics(deploymentID)
				metricsAtTime := make([]orchestrator.LiveMetric, 0)
				for _, m := range allMetricsAtTime {
					if m.Timestamp.Equal(liveMetric.Timestamp) {
						metricsAtTime = append(metricsAtTime, m)
					}
				}

				if len(metricsAtTime) > 0 {
					var sumCPU float64
					var sumMemory int64
					var sumNetworkRx int64
					var sumNetworkTx int64
					var sumDiskRead int64
					var sumDiskWrite int64

					for _, m := range metricsAtTime {
						sumCPU += m.CPUUsage
						sumMemory += m.MemoryUsage
						sumNetworkRx += m.NetworkRxBytes
						sumNetworkTx += m.NetworkTxBytes
						sumDiskRead += m.DiskReadBytes
						sumDiskWrite += m.DiskWriteBytes
					}

					avgCPU := sumCPU / float64(len(metricsAtTime))

					metric := &deploymentsv1.DeploymentMetric{
						DeploymentId:     deploymentID,
						Timestamp:        timestamppb.New(liveMetric.Timestamp),
						CpuUsagePercent:  avgCPU,
						MemoryUsageBytes: sumMemory,
						NetworkRxBytes:   sumNetworkRx,
						NetworkTxBytes:   sumNetworkTx,
						DiskReadBytes:    sumDiskRead,
						DiskWriteBytes:   sumDiskWrite,
					}

					if err := stream.Send(metric); err != nil {
						// Log the error but don't return immediately - might be transient
						log.Printf("[streamLiveMetrics] Error sending metric to stream: %v", err)
						// Check if it's a context cancellation
						if ctx.Err() != nil {
							return ctx.Err()
						}
						// For other errors, return them
						return err
					}
				}
			} else {
				// Single container metric
				metric := &deploymentsv1.DeploymentMetric{
					DeploymentId:     deploymentID,
					Timestamp:        timestamppb.New(liveMetric.Timestamp),
					CpuUsagePercent:  liveMetric.CPUUsage,
					MemoryUsageBytes: liveMetric.MemoryUsage,
					NetworkRxBytes:   liveMetric.NetworkRxBytes,
					NetworkTxBytes:   liveMetric.NetworkTxBytes,
					DiskReadBytes:    liveMetric.DiskReadBytes,
					DiskWriteBytes:   liveMetric.DiskWriteBytes,
				}

				if err := stream.Send(metric); err != nil {
					// Log the error but don't return immediately - might be transient
					log.Printf("[streamLiveMetrics] Error sending metric to stream: %v", err)
					// Check if it's a context cancellation
					if ctx.Err() != nil {
						return ctx.Err()
					}
					// For other errors, return them
					return err
				}
			}
		}
	}
}
