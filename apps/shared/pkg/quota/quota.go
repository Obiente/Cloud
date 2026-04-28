package quota

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
)

type RequestedResources struct {
	Replicas            int
	MemoryBytes         int64
	CPUshares           int64
	ExcludeDeploymentID string
}

type Checker struct{}

func NewChecker() *Checker { return &Checker{} }

// CanAllocate validates if the organization can allocate requested resources on top of current running allocations.
func (c *Checker) CanAllocate(ctx context.Context, organizationID string, req RequestedResources) error {
	// Ensure organization has a plan assigned (defaults to Starter plan)
	// This is called when resources are requested, so it's a good place to ensure plan assignment
	_ = organizations.EnsurePlanAssigned(organizationID)

	quota, err := c.getQuota(organizationID)
	if err != nil {
		return fmt.Errorf("quota: load: %w", err)
	}

	// Get plan limits first (these are the maximum boundary)
	planDeployMax, planMem, planCPU := c.getPlanLimitsFromQuota(quota)

	// Get effective limits: use overrides if set, but cap them to plan limits
	// Plan limits are the final boundary - org overrides cannot exceed them
	effDeployMax := planDeployMax
	if quota.DeploymentsMaxOverride != nil {
		overrideDeployMax := *quota.DeploymentsMaxOverride
		if overrideDeployMax > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planDeployMax > 0 && overrideDeployMax > planDeployMax {
				effDeployMax = planDeployMax
			} else {
				effDeployMax = overrideDeployMax
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	effMem := planMem
	if quota.MemoryBytesOverride != nil {
		overrideMem := *quota.MemoryBytesOverride
		if overrideMem > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planMem > 0 && overrideMem > planMem {
				effMem = planMem
			} else {
				effMem = overrideMem
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	effCPU := planCPU
	if quota.CPUCoresOverride != nil {
		overrideCPU := *quota.CPUCoresOverride
		if overrideCPU > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planCPU > 0 && overrideCPU > planCPU {
				effCPU = planCPU
			} else {
				effCPU = overrideCPU
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	// Zero means unlimited; allow if not set
	curReplicas, curMemBytes, curCPUcores, err := c.currentAllocations(organizationID, req.ExcludeDeploymentID)
	if err != nil {
		return fmt.Errorf("quota: current allocations: %w", err)
	}

	if effDeployMax > 0 && curReplicas+req.Replicas > effDeployMax {
		return fmt.Errorf("quota exceeded: replicas %d > max %d", curReplicas+req.Replicas, effDeployMax)
	}
	if effMem > 0 && curMemBytes+req.MemoryBytes*int64(req.Replicas) > effMem {
		return fmt.Errorf("quota exceeded: memory %d bytes > max %d bytes", curMemBytes+req.MemoryBytes*int64(req.Replicas), effMem)
	}
	// Convert Docker CPUshares to cores, multiplied by replicas (matching currentAllocations)
	totalCPUshares := req.CPUshares * int64(req.Replicas)
	reqCores := int(totalCPUshares) / 1024
	if totalCPUshares%1024 != 0 {
		reqCores++ // round up partial cores
	}
	if effCPU > 0 && curCPUcores+reqCores > effCPU {
		return fmt.Errorf("quota exceeded: cpu %d cores > max %d cores", curCPUcores+reqCores, effCPU)
	}
	return nil
}

func (c *Checker) getQuota(orgID string) (*database.OrgQuota, error) {
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
		// No quota exists, create empty one
		quota = database.OrgQuota{OrganizationID: orgID}
	}
	return &quota, nil
}

// getPlanLimitsFromQuota gets plan limits using an already-loaded OrgQuota
func (c *Checker) getPlanLimitsFromQuota(q *database.OrgQuota) (deploymentsMax int, memoryBytes int64, cpuCores int) {
	if q.PlanID == "" {
		return 0, 0, 0
	}
	var plan database.OrganizationPlan
	if err := database.DB.First(&plan, "id = ?", q.PlanID).Error; err != nil {
		return 0, 0, 0
	}
	return plan.DeploymentsMax, plan.MemoryBytes, plan.CPUCores
}

func (c *Checker) currentAllocations(orgID string, excludeDeploymentID string) (replicas int, memBytes int64, cpuCores int, err error) {
	// Count running replicas from deployment locations
	var count int64
	locationQuery := database.DB.Model(&database.DeploymentLocation{}).
		Where("deployment_locations.status = ?", "running").
		Joins("JOIN deployments d ON d.id = deployment_locations.deployment_id").
		Where("d.organization_id = ?", orgID)
	if excludeDeploymentID != "" {
		locationQuery = locationQuery.Where("d.id <> ?", excludeDeploymentID)
	}
	if err = locationQuery.Count(&count).Error; err != nil {
		return
	}
	// Sum memory and CPU across active deployments, multiplied by their replica count.
	// Only count deployments that are running, building, or deploying (not stopped/failed).
	// RUNNING=3, BUILDING=2, DEPLOYING=6
	type agg struct {
		Mem int64
		CPU int64
	}
	var a agg
	deploymentQuery := database.DB.Model(&database.Deployment{}).
		Select("COALESCE(SUM(COALESCE(memory_bytes,0) * COALESCE(replicas,1)),0) as mem, COALESCE(SUM(COALESCE(cpu_shares,0) * COALESCE(replicas,1)),0) as cpu").
		Where("organization_id = ? AND deleted_at IS NULL AND status IN (2,3,6)", orgID)
	if excludeDeploymentID != "" {
		deploymentQuery = deploymentQuery.Where("id <> ?", excludeDeploymentID)
	}
	if err = deploymentQuery.Scan(&a).Error; err != nil {
		return
	}
	// Convert Docker CPU shares to cores (1024 shares = 1 core)
	cpuCores = int(a.CPU) / 1024
	if a.CPU%1024 != 0 {
		cpuCores++ // round up partial cores
	}
	return int(count), a.Mem, cpuCores, nil
}

// GetEffectiveLimits returns the effective memory and CPU limits for an organization
// Plan limits are the maximum boundary - org overrides cannot exceed them
// Returns (memoryBytes, cpuCores, error)
// Zero values mean unlimited
func GetEffectiveLimits(organizationID string) (memoryBytes int64, cpuCores int, err error) {
	// Get organization quota
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", organizationID).First(&quota).Error; err != nil {
		// No quota exists - no limits (unlimited)
		return 0, 0, nil
	}

	// Get plan limits first (these are the maximum boundary)
	var plan database.OrganizationPlan
	planMem := int64(0)
	planCPU := 0
	if quota.PlanID != "" {
		if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err == nil {
			planMem = plan.MemoryBytes
			planCPU = plan.CPUCores
		}
	}

	// Get effective limits: use overrides if set, but cap them to plan limits
	// Plan limits are the final boundary - org overrides cannot exceed them
	effMem := planMem
	if quota.MemoryBytesOverride != nil {
		overrideMem := *quota.MemoryBytesOverride
		if overrideMem > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planMem > 0 && overrideMem > planMem {
				effMem = planMem
			} else {
				effMem = overrideMem
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	effCPU := planCPU
	if quota.CPUCoresOverride != nil {
		overrideCPU := *quota.CPUCoresOverride
		if overrideCPU > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planCPU > 0 && overrideCPU > planCPU {
				effCPU = planCPU
			} else {
				effCPU = overrideCPU
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	// Zero means unlimited
	return effMem, effCPU, nil
}

func valueOr(p *int, d int) int {
	if p == nil {
		return d
	}
	return *p
}
func valueOr64(p *int64, d int64) int64 {
	if p == nil {
		return d
	}
	return *p
}
