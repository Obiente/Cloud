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
	if err := r.db.WithContext(ctx).Create(deployment).Error; err != nil {
		return err
	}

	// Cache the newly created deployment
	if r.cache != nil {
		cacheKey := fmt.Sprintf("deployment:%s", deployment.ID)
		r.cache.Set(ctx, cacheKey, deployment, 5*time.Minute)
	}

	return nil
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

	// Use Select to explicitly update fields, including zero values like empty strings
	if err := r.db.WithContext(ctx).
		Model(deployment).
		Select(
			"name", "domain", "custom_domains", "type", "build_strategy",
			"repository_url", "branch", "build_command", "install_command", "start_command",
			"dockerfile_path", "compose_file_path", "build_path", "build_output_path",
			"use_nginx", "nginx_config", "github_integration_id",
			"status", "health_status", "environment", "groups",
			"image", "port", "replicas", "memory_bytes", "cpu_shares",
			"env_vars", "env_file_content", "compose_yaml",
			"build_time", "size", "storage_bytes", "bandwidth_usage",
			"last_deployed_at", "updated_at",
		).
		Updates(deployment).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("deployment:%s", deployment.ID)
		// Fetch the updated deployment to ensure we have all fields
		var updatedDeployment Deployment
		if err := r.db.WithContext(ctx).Where("id = ?", deployment.ID).First(&updatedDeployment).Error; err == nil {
			r.cache.Set(ctx, cacheKey, updatedDeployment, 5*time.Minute)
		} else {
			// If fetch fails, at least clear the cache to avoid stale data
			r.cache.Delete(ctx, cacheKey)
		}
	}

	return nil
}

// UpdateEnvVars updates only the environment variables fields
func (r *DeploymentRepository) UpdateEnvVars(ctx context.Context, id string, envFileContent string, envVarsJSON string) error {
	if err := r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"env_file_content": envFileContent,
			"env_vars":         envVarsJSON,
			"last_deployed_at": time.Now(),
		}).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return nil
}

func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id string, status int32) error {
	if err := r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return nil
}

func (r *DeploymentRepository) UpdateHealthStatus(ctx context.Context, id string, healthStatus string) error {
	if err := r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Update("health_status", healthStatus).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return nil
}

func (r *DeploymentRepository) UpdateStorage(ctx context.Context, id string, storageBytes int64) error {
	// Store bytes as string (client will format it)
	sizeStr := fmt.Sprintf("%d", storageBytes)

	if err := r.db.WithContext(ctx).Model(&Deployment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"storage_bytes": storageBytes,
			"size":          sizeStr,
		}).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}

	return nil
}

func (r *DeploymentRepository) Delete(ctx context.Context, id string) error {
	// Soft delete: set DeletedAt timestamp
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&Deployment{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		// Clear deployment cache
		r.cache.Delete(ctx, fmt.Sprintf("deployment:%s", id))
		// Also clear DNS cache for this deployment (DNS service caches IPs by deployment ID)
		r.cache.Delete(ctx, fmt.Sprintf("dns:deployment:%s", id))
	}

	return nil
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
