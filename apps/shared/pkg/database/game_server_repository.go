package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type GameServerRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewGameServerRepository(db *gorm.DB, cache *RedisCache) *GameServerRepository {
	return &GameServerRepository{
		db:    db,
		cache: cache,
	}
}

func (r *GameServerRepository) Create(ctx context.Context, gameServer *GameServer) error {
	if err := r.db.WithContext(ctx).Create(gameServer).Error; err != nil {
		return err
	}

	// Cache the newly created game server
	if r.cache != nil {
		cacheKey := fmt.Sprintf("gameserver:%s", gameServer.ID)
		r.cache.Set(ctx, cacheKey, gameServer, 5*time.Minute)
	}

	return nil
}

func (r *GameServerRepository) GetByID(ctx context.Context, id string) (*GameServer, error) {
	return r.GetByIDIncludeDeleted(ctx, id, false)
}

func (r *GameServerRepository) GetByIDIncludeDeleted(ctx context.Context, id string, includeDeleted bool) (*GameServer, error) {
	cacheKey := fmt.Sprintf("gameserver:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var gameServer GameServer
			if err := json.Unmarshal([]byte(cachedData), &gameServer); err == nil {
				return &gameServer, nil
			}
		}
	}

	var gameServer GameServer
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.First(&gameServer).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, gameServer, 5*time.Minute)
	}

	return &gameServer, nil
}

type GameServerFilters struct {
	UserID     string
	IncludeAll bool
	Status     *int32
	GameType   *int32
	Limit      int64
	Offset     int64
}

func (r *GameServerRepository) GetAll(ctx context.Context, organizationID string, filters *GameServerFilters) ([]*GameServer, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filters != nil {
		// Apply status filter if provided
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}

		// Apply game type filter if provided
		if filters.GameType != nil {
			query = query.Where("game_type = ?", *filters.GameType)
		}

		// Apply user ID filter if provided and not in "include all" mode
		if filters.UserID != "" && !filters.IncludeAll {
			query = query.Where("created_by = ?", filters.UserID)
		}

		// Apply pagination
		if filters.Limit > 0 {
			query = query.Limit(int(filters.Limit))
		}
		if filters.Offset > 0 {
			query = query.Offset(int(filters.Offset))
		}
	}

	var gameServers []*GameServer
	if err := query.Find(&gameServers).Error; err != nil {
		return nil, err
	}

	return gameServers, nil
}

func (r *GameServerRepository) Update(ctx context.Context, gameServer *GameServer) error {
	// Use Select to explicitly update fields
	if err := r.db.WithContext(ctx).
		Model(gameServer).
		Select(
			"name", "description", "game_type", "status",
			"memory_bytes", "cpu_cores", "port",
			"docker_image", "start_command", "env_vars",
			"server_version", "container_id", "container_name",
			"storage_bytes", "bandwidth_usage",
			"player_count", "max_players",
			"last_started_at", "updated_at",
		).
		Updates(gameServer).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("gameserver:%s", gameServer.ID)
		// Fetch the updated game server to ensure we have all fields
		var updatedGameServer GameServer
		if err := r.db.WithContext(ctx).Where("id = ?", gameServer.ID).First(&updatedGameServer).Error; err == nil {
			r.cache.Set(ctx, cacheKey, updatedGameServer, 5*time.Minute)
		} else {
			// If fetch fails, at least clear the cache to avoid stale data
			r.cache.Delete(ctx, cacheKey)
		}
	}

	return nil
}

func (r *GameServerRepository) UpdateStatus(ctx context.Context, id string, status int32) error {
	if err := r.db.WithContext(ctx).Model(&GameServer{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("gameserver:%s", id))
	}

	return nil
}

func (r *GameServerRepository) UpdateContainerInfo(ctx context.Context, id string, containerID *string, containerName *string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if containerID != nil {
		updates["container_id"] = *containerID
	}
	if containerName != nil {
		updates["container_name"] = *containerName
	}

	if err := r.db.WithContext(ctx).Model(&GameServer{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("gameserver:%s", id))
	}

	return nil
}

func (r *GameServerRepository) UpdateStorage(ctx context.Context, id string, storageBytes int64) error {
	if err := r.db.WithContext(ctx).Model(&GameServer{}).
		Where("id = ?", id).
		Update("storage_bytes", storageBytes).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("gameserver:%s", id))
	}

	return nil
}

func (r *GameServerRepository) Delete(ctx context.Context, id string) error {
	// Soft delete
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&GameServer{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("gameserver:%s", id))
	}

	return nil
}

func (r *GameServerRepository) HardDelete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Unscoped().Delete(&GameServer{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("gameserver:%s", id))
	}

	return nil
}

// GetAvailablePort finds an available port starting from a base port
func (r *GameServerRepository) GetAvailablePort(ctx context.Context, basePort int32) (int32, error) {
	// Find the highest port in use
	var maxPort int32
	var gameServer GameServer
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("port DESC").
		First(&gameServer).Error
	
	if err == gorm.ErrRecordNotFound {
		// No game servers exist, return base port
		return basePort, nil
	} else if err != nil {
		return 0, err
	}

	maxPort = gameServer.Port
	if maxPort < basePort {
		return basePort, nil
	}

	// Return next available port
	return maxPort + 1, nil
}

