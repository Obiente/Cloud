package gameservers

import (
	"context"
	"errors"
	"fmt"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/lifecycle"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"time"
)

type StorageInfo = docker.StorageInfo

func CalculateStorage(ctx context.Context, _ string, id string) (*StorageInfo, error) {
	return docker.CalculateStorage(ctx, []string{id})
}

// updateGameServerStorage calculates and updates storage usage for a game server
func (s *Service) updateGameServerStorage(ctx context.Context, gameServerID string) error {
	// Read the current container identity rather than a cached runtime snapshot.
	var gameServer database.GameServer
	if err := database.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", gameServerID).First(&gameServer).Error; err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	// Calculate storage
	containerID := ""
	if gameServer.ContainerID != nil {
		containerID = *gameServer.ContainerID
	}

	storageInfo, measureErr := CalculateStorage(ctx, gameServer.DockerImage, containerID)
	bytes := int64(0)
	if measureErr == nil {
		bytes = storageInfo.TotalStorage
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(measureErr, s.repo.RecordStorage(persistCtx, gameServerID, containerID, bytes, measureErr))

}

// StartStorageMonitor refreshes both running and stopped local servers. Stopped
// volumes remain billable and must not retain an indefinitely old snapshot.
func (s *Service) StartStorageMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		if s.manager != nil && ctx.Err() == nil {
			var ids []string
			if err := database.DB.WithContext(ctx).Table("game_servers gs").Joins("JOIN game_server_locations loc ON loc.game_server_id = gs.id").Where("gs.deleted_at IS NULL AND loc.node_id = ?", s.manager.GetNodeID()).Distinct().Pluck("gs.id", &ids).Error; err == nil {
				lifecycle.EachConcurrent(ctx, ids, 5, func(ctx context.Context, id string) {
					probeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
					defer cancel()
					if err := s.updateGameServerStorage(probeCtx, id); err != nil {
						logger.Warn("Storage measurement pending for game server %s: %v", id, err)
					}
				})
			} else {
				logger.Warn("Cannot load storage inventory: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
