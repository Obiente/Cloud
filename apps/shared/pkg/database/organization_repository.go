package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type OrganizationRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewOrganizationRepository(db *gorm.DB, cache *RedisCache) *OrganizationRepository {
	return &OrganizationRepository{
		db:    db,
		cache: cache,
	}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *Organization) error {
	if err := r.db.WithContext(ctx).Create(org).Error; err != nil {
		return err
	}

	// Cache the newly created organization
	if r.cache != nil {
		cacheKey := fmt.Sprintf("organization:%s", org.ID)
		r.cache.Set(ctx, cacheKey, org, 10*time.Minute) // Longer TTL for organizations
	}

	return nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id string) (*Organization, error) {
	cacheKey := fmt.Sprintf("organization:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var org Organization
			if err := json.Unmarshal([]byte(cachedData), &org); err == nil {
				return &org, nil
			}
		}
	}

	var org Organization
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&org).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, org, 10*time.Minute) // Longer TTL for organizations
	}

	return &org, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, org *Organization) error {
	if err := r.db.WithContext(ctx).
		Model(org).
		Updates(org).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("organization:%s", org.ID)
		// Fetch the updated organization to ensure we have all fields
		var updatedOrg Organization
		if err := r.db.WithContext(ctx).Where("id = ?", org.ID).First(&updatedOrg).Error; err == nil {
			r.cache.Set(ctx, cacheKey, updatedOrg, 10*time.Minute)
		} else {
			// If fetch fails, at least clear the cache to avoid stale data
			r.cache.Delete(ctx, cacheKey)
		}
	}

	return nil
}

func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	// Note: Organizations are typically not deleted, but if they are:
	if err := r.db.WithContext(ctx).Delete(&Organization{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("organization:%s", id))
		// Also clear any organization member caches
		r.cache.DeletePattern(ctx, fmt.Sprintf("org:members:%s:*", id))
		// Clear related resource caches
		r.cache.DeletePattern(ctx, fmt.Sprintf("deployment:*:org:%s", id))
		r.cache.DeletePattern(ctx, fmt.Sprintf("gameserver:*:org:%s", id))
		r.cache.DeletePattern(ctx, fmt.Sprintf("vps:*:org:%s", id))
	}

	return nil
}

// GetBySlug retrieves an organization by slug (for URL-friendly lookups)
func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	cacheKey := fmt.Sprintf("organization:slug:%s", slug)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var org Organization
			if err := json.Unmarshal([]byte(cachedData), &org); err == nil {
				return &org, nil
			}
		}
	}

	var org Organization
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error; err != nil {
		return nil, err
	}

	// Cache by both ID and slug
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, org, 10*time.Minute)
		r.cache.Set(ctx, fmt.Sprintf("organization:%s", org.ID), org, 10*time.Minute)
	}

	return &org, nil
}

