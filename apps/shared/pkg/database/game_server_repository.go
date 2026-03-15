package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type GameServerRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

const gameServerPortAllocationLockKey int64 = 6773616

func NewGameServerRepository(db *gorm.DB, cache *RedisCache) *GameServerRepository {
	return &GameServerRepository{
		db:    db,
		cache: cache,
	}
}

func (r *GameServerRepository) WithPortAllocationLock(ctx context.Context, fn func(repo *GameServerRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector != nil && tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", gameServerPortAllocationLockKey).Error; err != nil {
				return fmt.Errorf("failed to acquire game server port allocation lock: %w", err)
			}
		}

		txRepo := &GameServerRepository{
			db:    tx,
			cache: r.cache,
		}

		return fn(txRepo)
	})
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
	// For list queries, we don't cache individual items but could cache the list result
	// However, lists are often filtered/paginated, so caching is less effective
	// We'll rely on individual item caching from GetByID calls

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

	// Cache individual items for faster subsequent GetByID calls
	if r.cache != nil && len(gameServers) > 0 {
		pairs := make(map[string]interface{})
		for _, gs := range gameServers {
			pairs[fmt.Sprintf("gameserver:%s", gs.ID)] = gs
		}
		// Use MSet for batch caching (more efficient)
		r.cache.MSet(ctx, pairs, 5*time.Minute)
	}

	return gameServers, nil
}

func (r *GameServerRepository) Update(ctx context.Context, gameServer *GameServer) error {
	// Use Select to explicitly update fields
	if err := r.db.WithContext(ctx).
		Model(gameServer).
		Select(
			"name", "description", "game_type", "status",
			"memory_bytes", "cpu_cores", "port", "extra_ports",
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

// ParseGameServerExtraPorts parses stored extra_ports and returns sanitized ports.
// Supports JSON arrays ("[25566,25567]"), quoted JSON strings, and PostgreSQL array format ("{25566,25567}").
func ParseGameServerExtraPorts(raw string) []int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") {
		return []int32{}
	}

	sanitize := func(ports []int32) []int32 {
		seen := make(map[int32]struct{}, len(ports))
		normalized := make([]int32, 0, len(ports))
		for _, port := range ports {
			if port <= 0 || port > 65535 {
				continue
			}
			if _, exists := seen[port]; exists {
				continue
			}
			seen[port] = struct{}{}
			normalized = append(normalized, port)
		}
		return normalized
	}

	parseDelimited := func(value string) []int32 {
		parts := strings.Split(value, ",")
		ports := make([]int32, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.Trim(strings.TrimSpace(part), "\"")
			if trimmed == "" {
				continue
			}
			number, err := strconv.ParseInt(trimmed, 10, 32)
			if err != nil {
				continue
			}
			ports = append(ports, int32(number))
		}
		return sanitize(ports)
	}

	var ports []int32
	if err := json.Unmarshal([]byte(raw), &ports); err == nil {
		return sanitize(ports)
	}

	var encoded string
	if err := json.Unmarshal([]byte(raw), &encoded); err == nil {
		encoded = strings.TrimSpace(encoded)
		if err := json.Unmarshal([]byte(encoded), &ports); err == nil {
			return sanitize(ports)
		}
		raw = encoded
	}

	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		inner := strings.TrimSpace(raw[1 : len(raw)-1])
		if inner == "" {
			return []int32{}
		}
		return parseDelimited(inner)
	}

	if strings.Contains(raw, ",") {
		return parseDelimited(raw)
	}

	number, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return []int32{}
	}
	return sanitize([]int32{int32(number)})
}

func (r *GameServerRepository) getUsedPortsSet(ctx context.Context, excludeGameServerID string) (map[int32]struct{}, error) {
	type gameServerPortRow struct {
		ID         string
		Port       int32
		ExtraPorts string
	}

	var rows []gameServerPortRow
	if err := r.db.WithContext(ctx).
		Table("game_servers").
		Select("id, port, extra_ports").
		Where("deleted_at IS NULL").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	used := make(map[int32]struct{}, len(rows)*2)
	for _, row := range rows {
		if excludeGameServerID != "" && row.ID == excludeGameServerID {
			continue
		}

		if row.Port > 0 && row.Port <= 65535 {
			used[row.Port] = struct{}{}
		}

		for _, extraPort := range ParseGameServerExtraPorts(row.ExtraPorts) {
			used[extraPort] = struct{}{}
		}
	}

	return used, nil
}

// IsPortAvailable checks whether a port is unused by any non-deleted game server.
func (r *GameServerRepository) IsPortAvailable(ctx context.Context, port int32, excludeGameServerID string) (bool, error) {
	if port <= 0 || port > 65535 {
		return false, fmt.Errorf("port %d out of valid range (1-65535)", port)
	}

	used, err := r.getUsedPortsSet(ctx, excludeGameServerID)
	if err != nil {
		return false, err
	}

	_, inUse := used[port]
	return !inUse, nil
}

// GetAvailablePorts finds N available ports starting from basePort.
func (r *GameServerRepository) GetAvailablePorts(ctx context.Context, basePort int32, count int, reserved []int32, excludeGameServerID string) ([]int32, error) {
	if count <= 0 {
		return []int32{}, nil
	}
	if count > 65535 {
		return nil, fmt.Errorf("invalid requested port count: %d", count)
	}

	if basePort < 1 {
		basePort = 1
	}

	used, err := r.getUsedPortsSet(ctx, excludeGameServerID)
	if err != nil {
		return nil, err
	}

	for _, reservedPort := range reserved {
		if reservedPort > 0 && reservedPort <= 65535 {
			used[reservedPort] = struct{}{}
		}
	}

	ports := make([]int32, 0, count)
	for candidate := basePort; candidate <= 65535 && len(ports) < count; candidate++ {
		if _, inUse := used[candidate]; inUse {
			continue
		}
		ports = append(ports, candidate)
		used[candidate] = struct{}{}
	}

	if len(ports) != count {
		return nil, fmt.Errorf("unable to allocate %d ports (allocated %d)", count, len(ports))
	}

	return ports, nil
}

// GetAvailablePort finds an available port starting from a base port
func (r *GameServerRepository) GetAvailablePort(ctx context.Context, basePort int32) (int32, error) {
	ports, err := r.GetAvailablePorts(ctx, basePort, 1, nil, "")
	if err != nil {
		return 0, err
	}

	return ports[0], nil
}
