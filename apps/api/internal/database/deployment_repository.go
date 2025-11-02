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
	return r.GetByIDIncludeDeleted(ctx, id, false)
}

func (r *DeploymentRepository) GetByIDIncludeDeleted(ctx context.Context, id string, includeDeleted bool) (*Deployment, error) {
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
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.First(&deployment).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, deployment, 5*time.Minute)
	}

	return &deployment, nil
}

func (r *DeploymentRepository) GetAll(ctx context.Context, organizationID string, filters *DeploymentFilters) ([]*Deployment, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filters != nil {
		// Apply status filter if provided
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
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

	// Use Select to explicitly update fields, including zero values like empty strings
	return r.db.WithContext(ctx).
		Model(deployment).
		Select(
			"name", "domain", "custom_domains", "type", "build_strategy",
			"repository_url", "branch", "build_command", "install_command",
			"dockerfile_path", "compose_file_path", "github_integration_id",
			"status", "health_status", "environment",
			"image", "port", "replicas", "memory_bytes", "cpu_shares",
			"env_vars", "env_file_content", "compose_yaml",
			"last_deployed_at", "updated_at",
		).
		Updates(deployment).Error
}

// UpdateEnvVars updates only the environment variables fields
func (r *DeploymentRepository) UpdateEnvVars(ctx context.Context, id string, envFileContent string, envVarsJSON string) error {
	// Clear cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"env_file_content": envFileContent,
			"env_vars":         envVarsJSON,
			"last_deployed_at": time.Now(),
		}).Error
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

	// Soft delete: set DeletedAt timestamp
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&Deployment{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error
}

func (r *DeploymentRepository) Count(ctx context.Context, organizationID string, filters *DeploymentFilters) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Deployment{}).
		Where("organization_id = ?", organizationID)
	
	// Apply additional filters if provided
	if filters != nil {
		// Apply status filter
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		
		// Apply user ID filter if provided and not in "include all" mode
		if filters.UserID != "" && !filters.IncludeAll {
			query = query.Where("created_by = ?", filters.UserID)
		}
	}
	
	var count int64
	return count, query.Count(&count).Error
}

type DeploymentFilters struct {
	Status     *int32
	Limit      int
	Offset     int
	UserID     string // Filter by creator user ID
	IncludeAll bool   // Include all deployments regardless of creator (for admins)
}
