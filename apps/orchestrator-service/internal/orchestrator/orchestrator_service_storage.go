package orchestrator

import (
	"context"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/lifecycle"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	shared "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"time"
)

func (os *OrchestratorService) updateStoragePeriodically() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		var ids []string
		node := os.deploymentManager.GetNodeID()
		if err := database.DB.WithContext(os.ctx).Table("deployments d").Where("d.deleted_at IS NULL AND (EXISTS (SELECT 1 FROM deployment_locations loc WHERE loc.deployment_id = d.id AND loc.node_id = ? AND loc.status IN ?) OR EXISTS (SELECT 1 FROM deployment_storage_measurements m WHERE m.deployment_id = d.id AND m.node_id = ?))", node, []string{"running", "stopped"}, node).Pluck("d.id", &ids).Error; err != nil {
			logger.Warn("Cannot load deployment storage inventory: %v", err)
		} else {
			lifecycle.EachConcurrent(os.ctx, ids, 5, func(ctx context.Context, id string) {
				probeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				defer cancel()
				if err := shared.MeasureDeploymentStorage(probeCtx, node, id); err != nil {
					logger.Warn("Storage measurement pending for deployment %s: %v", id, err)
				}
			})
		}
		select {
		case <-os.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
