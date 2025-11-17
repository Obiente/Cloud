package database

import (
	"context"
	"fmt"
	"time"
)

// CacheWarmer provides utilities for warming caches
type CacheWarmer struct {
	cache *RedisCache
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *RedisCache) *CacheWarmer {
	return &CacheWarmer{
		cache: cache,
	}
}

// WarmDeployments pre-loads deployments into cache for an organization
func (cw *CacheWarmer) WarmDeployments(ctx context.Context, repo *DeploymentRepository, organizationID string, limit int) error {
	if cw.cache == nil || repo == nil {
		return nil
	}

	filters := &DeploymentFilters{
		Limit: limit,
	}
	_, err := repo.GetAll(ctx, organizationID, filters)
	if err != nil {
		return err
	}

	// GetAll already caches individual items via MSet
	// This is just a convenience method to trigger the warming
	return nil
}

// WarmGameServers pre-loads game servers into cache for an organization
func (cw *CacheWarmer) WarmGameServers(ctx context.Context, repo *GameServerRepository, organizationID string, limit int) error {
	if cw.cache == nil || repo == nil {
		return nil
	}

	filters := &GameServerFilters{
		Limit: int64(limit),
	}
	_, err := repo.GetAll(ctx, organizationID, filters)
	if err != nil {
		return err
	}

	// GetAll already caches individual items via MSet
	return nil
}

// WarmVPSInstances pre-loads VPS instances into cache for an organization
func (cw *CacheWarmer) WarmVPSInstances(ctx context.Context, repo *VPSRepository, organizationID string, limit int) error {
	if cw.cache == nil || repo == nil {
		return nil
	}

	filters := &VPSFilters{
		Limit: int64(limit),
	}
	_, err := repo.GetAll(ctx, organizationID, filters)
	if err != nil {
		return err
	}

	// GetAll already caches individual items via MSet
	return nil
}

// InvalidateOrganizationCache clears all caches related to an organization
func (cw *CacheWarmer) InvalidateOrganizationCache(ctx context.Context, organizationID string) error {
	if cw.cache == nil {
		return nil
	}

	// Clear organization cache
	cw.cache.Delete(ctx, fmt.Sprintf("organization:%s", organizationID))
	// Note: Slug cache would need the actual slug to clear, pattern deletion is expensive

	// Individual resource caches are cleared by their respective repositories on update/delete
	// We don't scan all keys here to avoid performance issues on large datasets
	return nil
}

// GetCacheStats returns basic cache statistics (if supported by Redis)
func (cw *CacheWarmer) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	if cw.cache == nil || cw.cache.client == nil {
		return nil, fmt.Errorf("cache not initialized")
	}

	info, err := cw.cache.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["info"] = info
	return stats, nil
}

// RefreshCache refreshes a cached item by re-fetching from database
func (cw *CacheWarmer) RefreshCache(ctx context.Context, cacheKey string, fetchFn func() (interface{}, error), ttl time.Duration) error {
	if cw.cache == nil {
		return nil
	}

	item, err := fetchFn()
	if err != nil {
		return err
	}

	return cw.cache.Set(ctx, cacheKey, item, ttl)
}

