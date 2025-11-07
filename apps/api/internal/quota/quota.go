package quota

import (
	"context"
	"fmt"

	"api/internal/database"
	"api/internal/services/organizations"
)

type RequestedResources struct {
	Replicas    int
	MemoryBytes int64
	CPUshares   int64
}

type Checker struct{}

func NewChecker() *Checker { return &Checker{} }

// CanAllocate validates if the organization can allocate requested resources on top of current running allocations.
func (c *Checker) CanAllocate(ctx context.Context, organizationID string, req RequestedResources) error {
	// Ensure organization has a plan assigned (defaults to Starter plan)
	// This is called when resources are requested, so it's a good place to ensure plan assignment
	_ = organizations.EnsurePlanAssigned(organizationID)
	
	quota, err := c.getQuota(organizationID)
	if err != nil { return fmt.Errorf("quota: load: %w", err) }

	// Get effective limits: use overrides if set, otherwise use plan limits
	effDeployMax := valueOr(quota.DeploymentsMaxOverride, 0)
	effMem := valueOr64(quota.MemoryBytesOverride, 0)
	effCPU := valueOr(quota.CPUCoresOverride, 0)
	
	// If no overrides, get from plan
	if effDeployMax == 0 && effMem == 0 && effCPU == 0 {
		planDeployMax, planMem, planCPU := c.getPlanLimits(organizationID)
		if effDeployMax == 0 {
			effDeployMax = planDeployMax
		}
		if effMem == 0 {
			effMem = planMem
		}
		if effCPU == 0 {
			effCPU = planCPU
		}
	}
	
	// Zero means unlimited; allow if not set
	curReplicas, curMemBytes, curCPUcores, err := c.currentAllocations(organizationID)
	if err != nil { return fmt.Errorf("quota: current allocations: %w", err) }

	if effDeployMax > 0 && curReplicas+req.Replicas > effDeployMax {
		return fmt.Errorf("quota exceeded: replicas %d > max %d", curReplicas+req.Replicas, effDeployMax)
	}
	if effMem > 0 && curMemBytes+req.MemoryBytes*int64(req.Replicas) > effMem {
		return fmt.Errorf("quota exceeded: memory %d bytes > max %d bytes", curMemBytes+req.MemoryBytes*int64(req.Replicas), effMem)
	}
	if effCPU > 0 && curCPUcores+int(req.CPUshares) > effCPU {
		return fmt.Errorf("quota exceeded: cpu %d cores > max %d cores", curCPUcores+int(req.CPUshares), effCPU)
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

// getPlanLimits gets plan limits for an organization
func (c *Checker) getPlanLimits(orgID string) (deploymentsMax int, memoryBytes int64, cpuCores int) {
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
		return 0, 0, 0 // No plan
	}
	
	if quota.PlanID == "" {
		return 0, 0, 0 // No plan assigned
	}
	
	var plan database.OrganizationPlan
	if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
		return 0, 0, 0 // Plan not found
	}
	
	return plan.DeploymentsMax, plan.MemoryBytes, plan.CPUCores
}

func (c *Checker) currentAllocations(orgID string) (replicas int, memBytes int64, cpuCores int, err error) {
	// Count running replicas
	var count int64
    if err = database.DB.Model(&database.DeploymentLocation{}).
        Where("deployment_locations.status = ?", "running").
        Joins("JOIN deployments d ON d.id = deployment_locations.deployment_id").
        Where("d.organization_id = ?", orgID).
        Count(&count).Error; err != nil { return }
	// Sum requested memory and CPU across org deployments (nil treated as 0)
	type agg struct{ Mem int64; CPU int64 }
	var a agg
	if err = database.DB.Model(&database.Deployment{}).
		Select("COALESCE(SUM(COALESCE(memory_bytes,0)),0) as mem, COALESCE(SUM(COALESCE(cpu_shares,0)),0) as cpu").
		Where("organization_id = ?", orgID).Scan(&a).Error; err != nil { return }
	return int(count), a.Mem, int(a.CPU), nil
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

func valueOr(p *int, d int) int { if p == nil { return d }; return *p }
func valueOr64(p *int64, d int64) int64 { if p == nil { return d }; return *p }
