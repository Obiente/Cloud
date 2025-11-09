package billing

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"api/internal/database"
	"api/internal/pricing"

	"gorm.io/gorm"
)

// UsageBreakdown represents the cost breakdown for a monthly bill
type UsageBreakdown struct {
	CPUCostCents      int64 `json:"cpu_cost_cents"`
	MemoryCostCents   int64 `json:"memory_cost_cents"`
	BandwidthCostCents int64 `json:"bandwidth_cost_cents"`
	StorageCostCents  int64 `json:"storage_cost_cents"`
	TotalCostCents    int64 `json:"total_cost_cents"`
}

// ProcessMonthlyBilling processes monthly billing for all organizations that should be billed today
// This should be called daily to check if any organizations need to be billed
func ProcessMonthlyBilling() error {
	log.Printf("[Monthly Billing] Starting monthly billing process")

	now := time.Now()
	today := now.Day()

	// Get all billing accounts that should be billed today
	var billingAccounts []database.BillingAccount
	if err := database.DB.Where("billing_date = ? AND status = ?", today, "ACTIVE").Find(&billingAccounts).Error; err != nil {
		return fmt.Errorf("get billing accounts: %w", err)
	}

	if len(billingAccounts) == 0 {
		log.Printf("[Monthly Billing] No organizations to bill today (billing date: %d)", today)
		return nil
	}

	log.Printf("[Monthly Billing] Found %d organizations to bill today", len(billingAccounts))

	var orgsProcessed int
	var orgsSkipped int

	for _, billingAccount := range billingAccounts {
		err := processOrganizationBilling(billingAccount.OrganizationID, now)
		if err != nil {
			log.Printf("[Monthly Billing] Error billing org %s: %v", billingAccount.OrganizationID, err)
			orgsSkipped++
			continue
		}
		orgsProcessed++
	}

	log.Printf("[Monthly Billing] Completed: %d orgs processed, %d orgs skipped",
		orgsProcessed, orgsSkipped)

	return nil
}

// processOrganizationBilling processes billing for a single organization
func processOrganizationBilling(orgID string, billingDate time.Time) error {
	// Check if we've already created a bill for this billing period
	// Calculate the billing period: from last billing date to today
	var lastBill database.MonthlyBill
	var billingPeriodStart time.Time
	var billingPeriodEnd time.Time

	// Get the billing account to find the billing date
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		return fmt.Errorf("billing account not found: %w", err)
	}

	if billingAccount.BillingDate == nil {
		return fmt.Errorf("billing date not set for organization %s", orgID)
	}

	billingDay := *billingAccount.BillingDate

	// Calculate billing period
	// If this is the first bill, start from org creation or last month's billing date
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	// Find the last bill to determine the start of this billing period
	if err := database.DB.Where("organization_id = ? AND status IN ?", orgID, []string{"PAID", "PENDING"}).
		Order("billing_period_end DESC").First(&lastBill).Error; err == nil {
		// Use the end of the last billing period as the start of this one
		billingPeriodStart = lastBill.BillingPeriodEnd
	} else {
		// First bill: start from org creation date
		billingPeriodStart = org.CreatedAt.UTC().Truncate(24 * time.Hour)
	}

	billingPeriodEnd = time.Date(billingDate.Year(), billingDate.Month(), billingDay, 0, 0, 0, 0, time.UTC)

	// Check if a bill already exists for this period
	var existingBill database.MonthlyBill
	if err := database.DB.Where("organization_id = ? AND billing_period_start = ? AND billing_period_end = ?",
		orgID, billingPeriodStart, billingPeriodEnd).First(&existingBill).Error; err == nil {
		log.Printf("[Monthly Billing] Bill already exists for org %s for period %s to %s", orgID,
			billingPeriodStart.Format("2006-01-02"), billingPeriodEnd.Format("2006-01-02"))
		return nil
	}

	// Calculate usage for the billing period directly from metrics
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		return fmt.Errorf("metrics database not available")
	}

	// Get usage from hourly aggregates for the billing period
	var hourlyUsage struct {
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
	}
	metricsDB.Table("deployment_usage_hourly duh").
		Select(`
			COALESCE(CAST(SUM((duh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(duh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(duh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(duh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("duh.organization_id = ? AND duh.hour >= ? AND duh.hour < ?", orgID, billingPeriodStart, billingPeriodEnd).
		Scan(&hourlyUsage)

	// Get storage bytes (snapshot from deployments table)
	var storageSum struct {
		StorageBytes int64
	}
	database.DB.Table("deployments d").
		Select("COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
		Where("d.organization_id = ?", orgID).
		Scan(&storageSum)

	// Calculate costs using pricing model
	pricingModel := pricing.GetPricing()
	
	var cpuCost, memoryCost, bandwidthCost, storageCost int64
	
	cpuCost = pricingModel.CalculateCPUCost(hourlyUsage.CPUCoreSeconds)
	memoryCost = pricingModel.CalculateMemoryCost(hourlyUsage.MemoryByteSeconds)
	bandwidthBytes := hourlyUsage.BandwidthRxBytes + hourlyUsage.BandwidthTxBytes
	bandwidthCost = pricingModel.CalculateBandwidthCost(bandwidthBytes)
	// Storage is monthly cost, prorate based on billing period duration
	storageCostFullMonth := pricingModel.CalculateStorageCost(storageSum.StorageBytes)
	periodDuration := billingPeriodEnd.Sub(billingPeriodStart)
	daysInMonth := float64(time.Date(billingPeriodEnd.Year(), billingPeriodEnd.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
	daysInPeriod := periodDuration.Hours() / 24.0
	if daysInPeriod > 0 && daysInMonth > 0 {
		storageCost = int64(float64(storageCostFullMonth) * (daysInPeriod / daysInMonth))
	} else {
		storageCost = storageCostFullMonth
	}

	totalCostCents := cpuCost + memoryCost + bandwidthCost + storageCost

	// Create usage breakdown
	breakdown := UsageBreakdown{
		CPUCostCents:       cpuCost,
		MemoryCostCents:    memoryCost,
		BandwidthCostCents: bandwidthCost,
		StorageCostCents:   storageCost,
		TotalCostCents:     totalCostCents,
	}

	breakdownJSON, err := json.Marshal(breakdown)
	if err != nil {
		return fmt.Errorf("marshal breakdown: %w", err)
	}

	// Create the bill
	billID := fmt.Sprintf("bill-%d", time.Now().UnixNano())
	dueDate := billingPeriodEnd.AddDate(0, 0, 7) // Due 7 days after billing period ends

	bill := &database.MonthlyBill{
		ID:                billID,
		OrganizationID:    orgID,
		BillingPeriodStart: billingPeriodStart,
		BillingPeriodEnd:   billingPeriodEnd,
		AmountCents:       totalCostCents,
		Status:            "PENDING",
		DueDate:           dueDate,
		UsageBreakdown:    string(breakdownJSON),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Process payment in a transaction
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create the bill
		if err := tx.Create(bill).Error; err != nil {
			return fmt.Errorf("create bill: %w", err)
		}

		// Try to pay from credits
		if totalCostCents > 0 {
			var org database.Organization
			if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
				return fmt.Errorf("organization not found: %w", err)
			}

			if org.Credits >= totalCostCents {
				// Deduct from credits
				oldBalance := org.Credits
				org.Credits -= totalCostCents
				if err := tx.Save(&org).Error; err != nil {
					return fmt.Errorf("update credits: %w", err)
				}

				// Mark bill as paid
				now := time.Now()
				bill.Status = "PAID"
				bill.PaidAt = &now
				if err := tx.Save(bill).Error; err != nil {
					return fmt.Errorf("update bill status: %w", err)
				}

				// Record credit transaction
				note := fmt.Sprintf("Monthly bill payment for period %s to %s", 
					billingPeriodStart.Format("2006-01-02"), billingPeriodEnd.Format("2006-01-02"))
				transactionID := fmt.Sprintf("ct-%d", time.Now().UnixNano())
				transaction := &database.CreditTransaction{
					ID:             transactionID,
					OrganizationID: orgID,
					AmountCents:    -totalCostCents, // Negative for deduction
					BalanceAfter:   org.Credits,
					Type:           "usage",
					Source:         "system",
					Note:           &note,
					CreatedAt:      time.Now(),
				}
				if err := tx.Create(transaction).Error; err != nil {
					return fmt.Errorf("create transaction: %w", err)
				}

				log.Printf("[Monthly Billing] Paid bill %s for org %s from credits (%d -> %d cents)",
					billID, orgID, oldBalance, org.Credits)
			} else {
				// Not enough credits - bill remains PENDING
				log.Printf("[Monthly Billing] Insufficient credits for org %s (have %d, need %d). Bill %s remains PENDING",
					orgID, org.Credits, totalCostCents, billID)
			}
		} else {
			// Zero amount bill - mark as paid
			now := time.Now()
			bill.Status = "PAID"
			bill.PaidAt = &now
			if err := tx.Save(bill).Error; err != nil {
				return fmt.Errorf("update bill status: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[Monthly Billing] Created bill %s for org %s: %d cents (status: %s)",
		billID, orgID, totalCostCents, bill.Status)

	return nil
}

// PayBillPrematurely allows paying a pending bill before its due date
func PayBillPrematurely(billID string, orgID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Get the bill
		var bill database.MonthlyBill
		if err := tx.Where("id = ? AND organization_id = ?", billID, orgID).First(&bill).Error; err != nil {
			return fmt.Errorf("bill not found: %w", err)
		}

		if bill.Status != "PENDING" {
			return fmt.Errorf("bill is not pending (status: %s)", bill.Status)
		}

		// Get organization
		var org database.Organization
		if err := tx.First(&org, "id = ?", orgID).Error; err != nil {
			return fmt.Errorf("organization not found: %w", err)
		}

		if org.Credits < bill.AmountCents {
			return fmt.Errorf("insufficient credits (have %d, need %d)", org.Credits, bill.AmountCents)
		}

		// Deduct from credits
		oldBalance := org.Credits
		org.Credits -= bill.AmountCents
		if err := tx.Save(&org).Error; err != nil {
			return fmt.Errorf("update credits: %w", err)
		}

		// Mark bill as paid
		now := time.Now()
		bill.Status = "PAID"
		bill.PaidAt = &now
		if err := tx.Save(&bill).Error; err != nil {
			return fmt.Errorf("update bill status: %w", err)
		}

		// Record credit transaction
		note := fmt.Sprintf("Premature payment for bill %s (period %s to %s)",
			billID, bill.BillingPeriodStart.Format("2006-01-02"), bill.BillingPeriodEnd.Format("2006-01-02"))
		transactionID := fmt.Sprintf("ct-%d", time.Now().UnixNano())
		transaction := &database.CreditTransaction{
			ID:             transactionID,
			OrganizationID: orgID,
			AmountCents:    -bill.AmountCents,
			BalanceAfter:   org.Credits,
			Type:           "usage",
			Source:         "system",
			Note:           &note,
			CreatedAt:      time.Now(),
		}
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		log.Printf("[Monthly Billing] Paid bill %s prematurely for org %s from credits (%d -> %d cents)",
			billID, orgID, oldBalance, org.Credits)

		return nil
	})
}

