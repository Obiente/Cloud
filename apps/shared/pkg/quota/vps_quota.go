package quota

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
)

// CanAllocateVPS validates if the organization can allocate a new VPS instance
func (c *Checker) CanAllocateVPS(ctx context.Context, organizationID string) error {
	// Ensure organization has a plan assigned (defaults to Starter plan)
	_ = organizations.EnsurePlanAssigned(organizationID)

	quota, err := c.getQuota(organizationID)
	if err != nil {
		return fmt.Errorf("quota: load: %w", err)
	}

	// Get plan limits first (these are the maximum boundary)
	planVPSMax := c.getPlanVPSMax(organizationID)

	// Get effective limits: use overrides if set, but cap them to plan limits
	effVPSMax := planVPSMax
	if quota.MaxVpsInstancesOverride != nil {
		overrideVPSMax := *quota.MaxVpsInstancesOverride
		if overrideVPSMax > 0 {
			// Cap override to plan limit (plan limit is the maximum)
			if planVPSMax > 0 && overrideVPSMax > planVPSMax {
				effVPSMax = planVPSMax
			} else {
				effVPSMax = overrideVPSMax
			}
		}
		// If override is 0, keep plan limit (0 means use plan default, not unlimited)
	}

	// Zero means unlimited
	currentVPSCount, err := c.currentVPSCount(organizationID)
	if err != nil {
		return fmt.Errorf("quota: current VPS count: %w", err)
	}

	if effVPSMax > 0 && currentVPSCount >= effVPSMax {
		return fmt.Errorf("quota exceeded: maximum VPS instances (%d) reached", effVPSMax)
	}

	return nil
}

// getPlanVPSMax gets the maximum VPS instances limit for an organization's plan
func (c *Checker) getPlanVPSMax(orgID string) int {
	var quota database.OrgQuota
	if err := database.DB.Where("organization_id = ?", orgID).First(&quota).Error; err != nil {
		return 0 // No plan
	}

	if quota.PlanID == "" {
		return 0 // No plan assigned
	}

	var plan database.OrganizationPlan
	if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
		return 0 // Plan not found
	}

	return plan.MaxVpsInstances
}

// currentVPSCount counts the number of active VPS instances for an organization
func (c *Checker) currentVPSCount(orgID string) (int, error) {
	var count int64
	if err := database.DB.Model(&database.VPSInstance{}).
		Where("organization_id = ? AND deleted_at IS NULL", orgID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count VPS instances: %w", err)
	}
	return int(count), nil
}

// GetVPSQuota returns the current VPS count and the maximum allowed for an organization
func (c *Checker) GetVPSQuota(ctx context.Context, organizationID string) (current int, max int, err error) {
	_ = organizations.EnsurePlanAssigned(organizationID)

	quota, err := c.getQuota(organizationID)
	if err != nil {
		return 0, 0, fmt.Errorf("quota: load: %w", err)
	}

	planVPSMax := c.getPlanVPSMax(organizationID)
	effVPSMax := planVPSMax
	if quota.MaxVpsInstancesOverride != nil {
		overrideVPSMax := *quota.MaxVpsInstancesOverride
		if overrideVPSMax > 0 {
			if planVPSMax > 0 && overrideVPSMax > planVPSMax {
				effVPSMax = planVPSMax
			} else {
				effVPSMax = overrideVPSMax
			}
		}
	}

	currentCount, err := c.currentVPSCount(organizationID)
	if err != nil {
		return 0, 0, fmt.Errorf("quota: current VPS count: %w", err)
	}

	return currentCount, effVPSMax, nil
}
