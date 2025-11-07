package organizations

import (
	"errors"
	"fmt"
	"log"

	"api/internal/database"

	"gorm.io/gorm"
)

// EnsurePlanAssigned ensures an organization has a plan assigned.
// If no plan is assigned, it assigns the default "Starter" plan (plan-starter).
// This function is idempotent and safe to call multiple times.
// It's exported so other packages (like quota) can use it.
func EnsurePlanAssigned(orgID string) error {
	// Check if organization exists
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("organization not found")
		}
		return fmt.Errorf("get organization: %w", err)
	}

	// Check if organization already has a plan assigned
	var quota database.OrgQuota
	if err := database.DB.First(&quota, "organization_id = ?", orgID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("get quota: %w", err)
		}
		// No quota exists, we'll create one
	} else {
		// Quota exists, check if it has a plan assigned
		if quota.PlanID != "" {
			// Plan already assigned, nothing to do
			return nil
		}
	}

	// Find the default "Starter" plan (plan-starter)
	var defaultPlan database.OrganizationPlan
	if err := database.DB.First(&defaultPlan, "id = ?", "plan-starter").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If plan-starter doesn't exist, try to find any plan with minimum_payment_cents = 0 (free tier)
			if err := database.DB.Where("minimum_payment_cents = ?", 0).Order("created_at ASC").First(&defaultPlan).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// No plans exist at all, this shouldn't happen if seeding works
					log.Printf("[EnsurePlanAssigned] Warning: No default plan found for organization %s, skipping plan assignment", orgID)
					return nil
				}
				return fmt.Errorf("find default plan: %w", err)
			}
		} else {
			return fmt.Errorf("get default plan: %w", err)
		}
	}

	// Create or update OrgQuota with the default plan
	if quota.OrganizationID == "" {
		// Create new quota
		quota = database.OrgQuota{
			OrganizationID: orgID,
			PlanID:         defaultPlan.ID,
		}
		if err := database.DB.Create(&quota).Error; err != nil {
			return fmt.Errorf("create quota: %w", err)
		}
		log.Printf("[EnsurePlanAssigned] Assigned plan %s (%s) to organization %s", defaultPlan.Name, defaultPlan.ID, orgID)
	} else {
		// Update existing quota
		quota.PlanID = defaultPlan.ID
		if err := database.DB.Save(&quota).Error; err != nil {
			return fmt.Errorf("update quota: %w", err)
		}
		log.Printf("[EnsurePlanAssigned] Assigned plan %s (%s) to organization %s", defaultPlan.Name, defaultPlan.ID, orgID)
	}

	return nil
}

