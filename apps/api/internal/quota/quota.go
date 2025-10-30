package quota

import (
	"context"
	"fmt"

	"api/internal/database"
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
	quota, err := c.getQuota(organizationID)
	if err != nil { return fmt.Errorf("quota: load: %w", err) }

	// Effective limits come entirely from OrgQuota (custom per org)
	effDeployMax := valueOr(quota.DeploymentsMaxOverride, 0)
	effMem := valueOr64(quota.MemoryBytesOverride, 0)
	effCPU := valueOr(quota.CPUCoresOverride, 0)
	// Zero means unlimited; allow if not set
	curReplicas, curMemBytes, curCPUcores, err := c.currentAllocations(organizationID)
	if err != nil { return fmt.Errorf("quota: current allocations: %w", err) }

	if effDeployMax > 0 && curReplicas+req.Replicas > effDeployMax {
		return fmt.Errorf("quota exceeded: replicas %d > max %d", curReplicas+req.Replicas, effDeployMax)
	}
	if effMem > 0 && curMemBytes+req.MemoryBytes*int64(req.Replicas) > effMem {
		return fmt.Errorf("quota exceeded: memory")
	}
	if effCPU > 0 && curCPUcores+int(req.CPUshares) > effCPU {
		return fmt.Errorf("quota exceeded: cpu")
	}
	return nil
}

func (c *Checker) getQuota(orgID string) (*database.OrgQuota, error) {
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
		// default unlimited if not configured
		quota = database.OrgQuota{OrganizationID: orgID}
	}
	return &quota, nil
}

func (c *Checker) currentAllocations(orgID string) (replicas int, memBytes int64, cpuCores int, err error) {
	// Count running replicas
	var count int64
	if err = database.DB.Model(&database.DeploymentLocation{}).
		Where("status = ?", "running").
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

func valueOr(p *int, d int) int { if p == nil { return d }; return *p }
func valueOr64(p *int64, d int64) int64 { if p == nil { return d }; return *p }
