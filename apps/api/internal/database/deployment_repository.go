package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DeploymentRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewDeploymentRepository(db *gorm.DB, cache *RedisCache) *DeploymentRepository {
	return &DeploymentRepository{
		db:    db,
		cache: cache,
	}
}

func (r *DeploymentRepository) Create(ctx context.Context, deployment *Deployment) error {
	return r.db.WithContext(ctx).Create(deployment).Error
}

func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*Deployment, error) {
	cacheKey := fmt.Sprintf("deployment:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var deployment Deployment
			if err := json.Unmarshal([]byte(cachedData), &deployment); err == nil {
				return &deployment, nil
			}
		}
	}

	var deployment Deployment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&deployment).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, deployment, 5*time.Minute)
	}

	return &deployment, nil
}

func (r *DeploymentRepository) GetAll(ctx context.Context, organizationID string, filters *DeploymentFilters) ([]*Deployment, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", organizationID)

	if filters != nil {
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.Limit > 0 {
			query = query.Limit(int(filters.Limit))
		}
		if filters.Offset > 0 {
			query = query.Offset(int(filters.Offset))
		}
	}

	var deployments []*Deployment
	if err := query.Find(&deployments).Error; err != nil {
		return nil, err
	}

	return deployments, nil
}

func (r *DeploymentRepository) Update(ctx context.Context, deployment *Deployment) error {
	deployment.LastDeployedAt = time.Now()

	// Clear cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", deployment.ID))
	}

	return r.db.WithContext(ctx).Save(deployment).Error
}

func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id string, status int32) error {
	// Clear cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *DeploymentRepository) Delete(ctx context.Context, id string) error {
	// Clear cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return r.db.WithContext(ctx).Delete(&Deployment{}, "id = ?", id).Error
}

func (r *DeploymentRepository) Count(ctx context.Context, organizationID string) (int64, error) {
	var count int64
	return count, r.db.WithContext(ctx).Model(&Deployment{}).
		Where("organization_id = ?", organizationID).
		Count(&count).Error
}

type DeploymentFilters struct {
	Status *int32
	Limit  int
	Offset int
}
