package billing

import (
	"fmt"
	"log"
	"time"

	"api/internal/database"

	"gorm.io/gorm"
)

// GrantMonthlyFreeCredits grants monthly free credits to all organizations based on their plan
// This should be called monthly (e.g., via cron job or scheduled task)
func GrantMonthlyFreeCredits() error {
	log.Printf("[Monthly Credits] Starting monthly free credits grant process")

	// Get all organizations with assigned plans
	var quotas []database.OrgQuota
	if err := database.DB.Where("plan_id != '' AND plan_id IS NOT NULL").Find(&quotas).Error; err != nil {
		return fmt.Errorf("get quotas: %w", err)
	}

	if len(quotas) == 0 {
		log.Printf("[Monthly Credits] No organizations with assigned plans found")
		return nil
	}

	// Track statistics
	var totalGranted int64
	var orgsProcessed int
	var orgsSkipped int

	for _, quota := range quotas {
		// Get the plan
		var plan database.OrganizationPlan
		if err := database.DB.First(&plan, "id = ?", quota.PlanID).Error; err != nil {
			log.Printf("[Monthly Credits] Warning: plan %s not found for org %s, skipping", quota.PlanID, quota.OrganizationID)
			orgsSkipped++
			continue
		}

		// Skip if plan has no monthly free credits
		if plan.MonthlyFreeCreditsCents <= 0 {
			continue
		}

		// Grant credits in a transaction
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Get organization
			var org database.Organization
			if err := tx.First(&org, "id = ?", quota.OrganizationID).Error; err != nil {
				return fmt.Errorf("organization not found: %w", err)
			}

			// Check if we've already granted credits this month using the tracking table
			now := time.Now()
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

			var existingGrant database.MonthlyCreditGrant
			if err := tx.Where("organization_id = ? AND plan_id = ? AND grant_month = ?",
				quota.OrganizationID, plan.ID, monthStart).First(&existingGrant).Error; err == nil {
				log.Printf("[Monthly Credits] Credits already granted to org %s for %s (granted at %s), skipping",
					quota.OrganizationID, monthStart.Format("2006-01"), existingGrant.GrantedAt.Format(time.RFC3339))
				return nil // Already granted this month
			}

			// Grant credits
			oldBalance := org.Credits
			org.Credits += plan.MonthlyFreeCreditsCents
			if err := tx.Save(&org).Error; err != nil {
				return fmt.Errorf("update credits: %w", err)
			}

			// Record credit transaction
			monthStr := monthStart.Format("2006-01")
			note := fmt.Sprintf("Monthly free credits for %s (plan: %s)", monthStr, plan.Name)
			transactionID := fmt.Sprintf("ct-%d", time.Now().UnixNano())
			transaction := &database.CreditTransaction{
				ID:             transactionID,
				OrganizationID: quota.OrganizationID,
				AmountCents:    plan.MonthlyFreeCreditsCents,
				BalanceAfter:   org.Credits,
				Type:           "admin_add",
				Source:         "system",
				Note:           &note,
				CreatedAt:      time.Now(),
			}
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("create transaction: %w", err)
			}

			// Record grant in tracking table for metrics and recovery
			grant := &database.MonthlyCreditGrant{
				OrganizationID: quota.OrganizationID,
				PlanID:         plan.ID,
				GrantMonth:     monthStart,
				AmountCents:    plan.MonthlyFreeCreditsCents,
				GrantedAt:      time.Now(),
				CreatedAt:      time.Now(),
			}
			if err := tx.Create(grant).Error; err != nil {
				return fmt.Errorf("create grant record: %w", err)
			}

			log.Printf("[Monthly Credits] Granted %d cents to org %s (plan: %s, balance: %d -> %d)",
				plan.MonthlyFreeCreditsCents, quota.OrganizationID, plan.Name, oldBalance, org.Credits)

			totalGranted += plan.MonthlyFreeCreditsCents
			orgsProcessed++

			return nil
		})

		if err != nil {
			log.Printf("[Monthly Credits] Error granting credits to org %s: %v", quota.OrganizationID, err)
			orgsSkipped++
		}
	}

	log.Printf("[Monthly Credits] Completed: %d orgs processed, %d orgs skipped, %d total cents granted",
		orgsProcessed, orgsSkipped, totalGranted)

	return nil
}
