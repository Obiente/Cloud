package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	vpsorch "github.com/obiente/cloud/apps/vps-service/orchestrator"
)

// collectVPSMetrics periodically collects VPS metrics from Proxmox and stores them in vps_metrics table
func (os *OrchestratorService) collectVPSMetrics() {
	// Collect every 5 minutes (same as deployments)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	os.collectVPSMetricsOnce()

	for {
		select {
		case <-ticker.C:
			os.collectVPSMetricsOnce()
		case <-os.ctx.Done():
			return
		}
	}
}

// collectVPSMetricsOnce collects metrics for all running VPS instances
func (os *OrchestratorService) collectVPSMetricsOnce() {
	ctx, cancel := context.WithTimeout(os.ctx, 2*time.Minute)
	defer cancel()

	// Get all running VPS instances
	var vpsInstances []database.VPSInstance
	if err := database.DB.Where("deleted_at IS NULL AND instance_id IS NOT NULL").
		Find(&vpsInstances).Error; err != nil {
		logger.Warn("[Orchestrator] Failed to get VPS instances for metrics collection: %v", err)
		return
	}

	if len(vpsInstances) == 0 {
		return
	}

	// Get Proxmox config
	proxmoxConfig, err := vpsorch.GetProxmoxConfig()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to get Proxmox config for VPS metrics: %v", err)
		return
	}

	proxmoxClient, err := vpsorch.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		logger.Warn("[Orchestrator] Failed to create Proxmox client for VPS metrics: %v", err)
		return
	}

	// Get nodes
	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		logger.Warn("[Orchestrator] Failed to get Proxmox nodes for VPS metrics: %v", err)
		return
	}

	nodeName := nodes[0] // Use first node

	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		logger.Warn("[Orchestrator] Metrics database not available for VPS metrics collection")
		return
	}

	now := time.Now()
	collectedCount := 0
	failedCount := 0

	for _, vps := range vpsInstances {
		if vps.InstanceID == nil {
			continue
		}

		// Parse VM ID
		var vmID int
		if _, err := fmt.Sscanf(*vps.InstanceID, "%d", &vmID); err != nil || vmID == 0 {
			logger.Debug("[Orchestrator] Invalid VM ID for VPS %s: %s", vps.ID, *vps.InstanceID)
			continue
		}

		// Get VM status first to check if it's running
		status, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmID)
		if err != nil {
			logger.Debug("[Orchestrator] Failed to get VM status for VPS %s (VMID %d): %v", vps.ID, vmID, err)
			failedCount++
			continue
		}

		// Only collect metrics for running VMs
		if status != "running" {
			continue
		}

		// Get metrics from Proxmox
		metrics, err := proxmoxClient.GetVMMetrics(ctx, nodeName, vmID)
		if err != nil {
			logger.Debug("[Orchestrator] Failed to get VM metrics for VPS %s (VMID %d): %v", vps.ID, vmID, err)
			failedCount++
			continue
		}

		// Parse metrics
		cpuUsage := 0.0
		if cpu, ok := metrics["cpu"].(float64); ok {
			cpuUsage = cpu * 100 // Proxmox returns CPU as fraction (0.0-1.0)
		}

		memoryUsed := int64(0)
		memoryTotal := vps.MemoryBytes
		if mem, ok := metrics["mem"].(float64); ok {
			memoryUsed = int64(mem)
		}
		if maxmem, ok := metrics["maxmem"].(float64); ok {
			memoryTotal = int64(maxmem)
		}

		diskUsed := int64(0)
		diskTotal := vps.DiskBytes
		if diskUsedVal, ok := metrics["disk"].(float64); ok {
			diskUsed = int64(diskUsedVal)
		}

		networkRxBytes := int64(0)
		networkTxBytes := int64(0)
		if netin, ok := metrics["netin"].(float64); ok {
			networkRxBytes = int64(netin)
		}
		if netout, ok := metrics["netout"].(float64); ok {
			networkTxBytes = int64(netout)
		}

		// Note: Proxmox status/current endpoint doesn't provide IOPS, so we'll use 0
		diskReadIOPS := 0.0
		diskWriteIOPS := 0.0

		// Store metric
		vpsMetric := database.VPSMetrics{
			VPSInstanceID:  vps.ID,
			InstanceID:     *vps.InstanceID,
			NodeID:         nodeName,
			CPUUsage:       cpuUsage,
			MemoryUsed:     memoryUsed,
			MemoryTotal:    memoryTotal,
			DiskUsed:       diskUsed,
			DiskTotal:      diskTotal,
			NetworkRxBytes: networkRxBytes,
			NetworkTxBytes: networkTxBytes,
			DiskReadIOPS:   diskReadIOPS,
			DiskWriteIOPS:  diskWriteIOPS,
			Timestamp:      now,
		}

		if err := metricsDB.Create(&vpsMetric).Error; err != nil {
			logger.Warn("[Orchestrator] Failed to store VPS metric for %s: %v", vps.ID, err)
			failedCount++
			continue
		}

		// Also publish to live metrics streamer for real-time streaming
		metricsStreamer := orchestrator.GetGlobalMetricsStreamer()
		if metricsStreamer != nil {
			// Calculate disk read/write bytes (Proxmox doesn't provide these directly, use 0 for now)
			diskReadBytes := int64(0)
			diskWriteBytes := int64(0)

			metricsStreamer.AddVPSMetrics(
				vps.ID,
				*vps.InstanceID,
				nodeName,
				cpuUsage,
				memoryUsed,
				memoryTotal,
				networkRxBytes,
				networkTxBytes,
				diskReadBytes,
				diskWriteBytes,
			)
		}

		collectedCount++
	}

	if collectedCount > 0 || failedCount > 0 {
		logger.Debug("[Orchestrator] Collected VPS metrics: %d succeeded, %d failed", collectedCount, failedCount)
	}
}
