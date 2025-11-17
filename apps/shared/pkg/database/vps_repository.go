package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type VPSRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewVPSRepository(db *gorm.DB, cache *RedisCache) *VPSRepository {
	return &VPSRepository{
		db:    db,
		cache: cache,
	}
}

func (r *VPSRepository) Create(ctx context.Context, vps *VPSInstance) error {
	if err := r.db.WithContext(ctx).Create(vps).Error; err != nil {
		return err
	}

	// Cache the newly created VPS
	if r.cache != nil {
		cacheKey := fmt.Sprintf("vps:%s", vps.ID)
		r.cache.Set(ctx, cacheKey, vps, 5*time.Minute)
	}

	return nil
}

func (r *VPSRepository) GetByID(ctx context.Context, id string) (*VPSInstance, error) {
	return r.GetByIDIncludeDeleted(ctx, id, false)
}

func (r *VPSRepository) GetByIDIncludeDeleted(ctx context.Context, id string, includeDeleted bool) (*VPSInstance, error) {
	cacheKey := fmt.Sprintf("vps:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var vps VPSInstance
			if err := json.Unmarshal([]byte(cachedData), &vps); err == nil {
				return &vps, nil
			}
		}
	}

	var vps VPSInstance
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.First(&vps).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, vps, 5*time.Minute)
	}

	return &vps, nil
}

func (r *VPSRepository) GetAll(ctx context.Context, organizationID string, filters *VPSFilters) ([]*VPSInstance, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filters != nil {
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.Region != "" {
			query = query.Where("region = ?", filters.Region)
		}
		if filters.UserID != "" && !filters.IncludeAll {
			query = query.Where("created_by = ?", filters.UserID)
		}
		if filters.Limit > 0 {
			query = query.Limit(int(filters.Limit))
		}
		if filters.Offset > 0 {
			query = query.Offset(int(filters.Offset))
		}
	}

	var vpsInstances []*VPSInstance
	if err := query.Find(&vpsInstances).Error; err != nil {
		return nil, err
	}

	// Cache individual items for faster subsequent GetByID calls
	if r.cache != nil && len(vpsInstances) > 0 {
		pairs := make(map[string]interface{})
		for _, vps := range vpsInstances {
			pairs[fmt.Sprintf("vps:%s", vps.ID)] = vps
		}
		// Use MSet for batch caching (more efficient)
		r.cache.MSet(ctx, pairs, 5*time.Minute)
	}

	return vpsInstances, nil
}

func (r *VPSRepository) Update(ctx context.Context, vps *VPSInstance) error {
	if err := r.db.WithContext(ctx).
		Model(vps).
		Updates(vps).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("vps:%s", vps.ID)
		// Fetch the updated VPS to ensure we have all fields
		var updatedVPS VPSInstance
		if err := r.db.WithContext(ctx).Where("id = ?", vps.ID).First(&updatedVPS).Error; err == nil {
			r.cache.Set(ctx, cacheKey, updatedVPS, 5*time.Minute)
		} else {
			// If fetch fails, at least clear the cache to avoid stale data
			r.cache.Delete(ctx, cacheKey)
		}
	}

	return nil
}

func (r *VPSRepository) UpdateStatus(ctx context.Context, id string, status int32) error {
	if err := r.db.WithContext(ctx).Model(&VPSInstance{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("vps:%s", id))
	}

	return nil
}

func (r *VPSRepository) Delete(ctx context.Context, id string) error {
	// Hard delete
	if err := r.db.WithContext(ctx).Delete(&VPSInstance{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("vps:%s", id))
	}

	return nil
}

type VPSFilters struct {
	Status   *int32
	Region   string
	Limit    int64
	Offset   int64
	UserID   string
	IncludeAll bool
}

