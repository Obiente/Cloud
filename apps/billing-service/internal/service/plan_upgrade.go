package billing

import (
	"errors"
	"fmt"
	"log"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	"gorm.io/gorm"
)

// checkAndUpgradePlan checks if the organization should be upgraded to a higher plan
// based on their total_paid_cents and automatically upgrades them if eligible.
// This function is idempotent and safe to call multiple times.
func checkAndUpgradePlan(orgID string, tx *gorm.DB) error {
	// Get organization
	var org database.Organization
	if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Get current quota/plan
	var quota database.OrgQuota
	if err := tx.First(&quota, "organization_id = ?", orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No quota exists yet, we'll create one if we find a matching plan
			quota = database.OrgQuota{OrganizationID: orgID}
		} else {
			return fmt.Errorf("get quota: %w", err)
		}
	}

	// Find the best plan the organization qualifies for based on total_paid_cents
	// We want the plan with the highest minimum_payment_cents that is <= total_paid_cents
	var bestPlan database.OrganizationPlan
	var bestPlanFound bool

	var allPlans []database.OrganizationPlan
	if err := tx.Order("minimum_payment_cents DESC").Find(&allPlans).Error; err != nil {
		return fmt.Errorf("list plans: %w", err)
	}

	for _, plan := range allPlans {
		// Skip plans with no minimum payment requirement (they're not auto-upgrade plans)
		if plan.MinimumPaymentCents <= 0 {
			continue
		}

		// Check if organization qualifies for this plan
		if org.TotalPaidCents >= plan.MinimumPaymentCents {
			// Check if this plan is better than the current one
			if !bestPlanFound || plan.MinimumPaymentCents > bestPlan.MinimumPaymentCents {
				bestPlan = plan
				bestPlanFound = true
			}
		}
	}

	// If we found a better plan and the organization isn't already on it, upgrade
	if bestPlanFound {
		currentPlanID := quota.PlanID
		if currentPlanID != bestPlan.ID {
			// Upgrade to the new plan
			quota.PlanID = bestPlan.ID

			// If quota doesn't exist yet, create it
			if quota.OrganizationID == "" {
				quota.OrganizationID = orgID
				if err := tx.Create(&quota).Error; err != nil {
					return fmt.Errorf("create quota: %w", err)
				}
			} else {
				if err := tx.Save(&quota).Error; err != nil {
					return fmt.Errorf("update quota: %w", err)
				}
			}

			log.Printf("[Plan Upgrade] Organization %s upgraded from plan %s to plan %s (total paid: %d cents, minimum required: %d cents)",
				orgID, currentPlanID, bestPlan.ID, org.TotalPaidCents, bestPlan.MinimumPaymentCents)
		}
	}

	return nil
}

// updateTotalPaidAndUpgrade updates the organization's total_paid_cents and checks for plan upgrades
func updateTotalPaidAndUpgrade(orgID string, amountCents int64, tx *gorm.DB) error {
	// Get organization
	var org database.Organization
	if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Update total paid
	oldTotalPaid := org.TotalPaidCents
	org.TotalPaidCents += amountCents
	if err := tx.Save(&org).Error; err != nil {
		return fmt.Errorf("update total paid: %w", err)
	}

	log.Printf("[Billing] Organization %s total paid updated: %d -> %d cents (+%d)",
		orgID, oldTotalPaid, org.TotalPaidCents, amountCents)

	// Check and upgrade plan if eligible
	if err := checkAndUpgradePlan(orgID, tx); err != nil {
		log.Printf("[Plan Upgrade] Warning: failed to check/upgrade plan for org %s: %v", orgID, err)
		// Don't fail the transaction if upgrade check fails
	}

	return nil
}
