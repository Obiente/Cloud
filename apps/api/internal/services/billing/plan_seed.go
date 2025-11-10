package billing

import (
	"fmt"
	"log"

	"api/internal/database"
)

// SeedDefaultPlans creates default plans if no plans exist
func SeedDefaultPlans() error {
	// Check if any plans exist
	var count int64
	if err := database.DB.Model(&database.OrganizationPlan{}).Count(&count).Error; err != nil {
		return fmt.Errorf("check existing plans: %w", err)
	}

	if count > 0 {
		log.Printf("[Plan Seed] Plans already exist (%d plans), skipping seed", count)
		return nil
	}

	log.Printf("[Plan Seed] No plans found, creating default plans...")

	defaultPlans := []database.OrganizationPlan{
		{
			ID:                      "plan-starter",
			Name:                    "Starter",
			Description:             "Perfect for small projects and getting started. Includes basic resource limits.",
			CPUCores:                2,
			MemoryBytes:             2 * 1024 * 1024 * 1024, // 2 GB
			DeploymentsMax:          3,
			MaxVpsInstances:         2,                      // 2 VPS instances for starter plan
			BandwidthBytesMonth:     50 * 1024 * 1024 * 1024, // 50 GB/month
			StorageBytes:            10 * 1024 * 1024 * 1024, // 10 GB
			MinimumPaymentCents:     0,                       // Free tier, no minimum payment
			MonthlyFreeCreditsCents: 0,                       // No free credits for default plans
		},
		{
			ID:                      "plan-pro",
			Name:                    "Pro",
			Description:             "For growing businesses and production workloads. Higher resource limits.",
			CPUCores:                8,
			MemoryBytes:             16 * 1024 * 1024 * 1024, // 16 GB
			DeploymentsMax:          20,
			MaxVpsInstances:         10,                     // 10 VPS instances for pro plan
			BandwidthBytesMonth:     500 * 1024 * 1024 * 1024, // 500 GB/month
			StorageBytes:            100 * 1024 * 1024 * 1024, // 100 GB
			MinimumPaymentCents:     10000,                    // $100 minimum payment to auto-upgrade
			MonthlyFreeCreditsCents: 0,                        // No free credits for default plans
		},
		{
			ID:                      "plan-enterprise",
			Name:                    "Enterprise",
			Description:             "For large-scale applications and enterprise customers. Maximum resources and premium support.",
			CPUCores:                32,
			MemoryBytes:             64 * 1024 * 1024 * 1024, // 64 GB
			DeploymentsMax:          100,
			MaxVpsInstances:         50,                      // 50 VPS instances for enterprise plan
			BandwidthBytesMonth:     5 * 1024 * 1024 * 1024 * 1024, // 5 TB/month
			StorageBytes:            1 * 1024 * 1024 * 1024 * 1024, // 1 TB
			MinimumPaymentCents:     50000,                         // $500 minimum payment to auto-upgrade
			MonthlyFreeCreditsCents: 0,                             // No free credits for default plans
		},
	}

	for _, plan := range defaultPlans {
		if err := database.DB.Create(&plan).Error; err != nil {
			return fmt.Errorf("create plan %s: %w", plan.Name, err)
		}
		log.Printf("[Plan Seed] Created plan: %s (ID: %s)", plan.Name, plan.ID)
	}

	log.Printf("[Plan Seed] Successfully created %d default plans", len(defaultPlans))
	return nil
}
