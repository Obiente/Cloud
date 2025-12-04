package billing

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/pricing"

	"gorm.io/gorm"
)

// UsageBreakdown represents the cost breakdown for a monthly bill
type UsageBreakdown struct {
	CPUCostCents      int64 `json:"cpu_cost_cents"`
	MemoryCostCents   int64 `json:"memory_cost_cents"`
	BandwidthCostCents int64 `json:"bandwidth_cost_cents"`
	StorageCostCents  int64 `json:"storage_cost_cents"`
	PublicIPCostCents int64 `json:"public_ip_cost_cents"` // Flat rate cost for public IPs
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

	// Get game server usage from hourly aggregates and add to deployment usage
	var gameServerHourlyUsage struct {
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
	}
	metricsDB.Table("game_server_usage_hourly gsuh").
		Select(`
			COALESCE(CAST(SUM((gsuh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(gsuh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(gsuh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(gsuh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("gsuh.organization_id = ? AND gsuh.hour >= ? AND gsuh.hour < ?", orgID, billingPeriodStart, billingPeriodEnd).
		Scan(&gameServerHourlyUsage)

	// Get VPS usage from hourly aggregates and add to deployment/game server usage
	var vpsHourlyUsage struct {
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
	}
	metricsDB.Table("vps_usage_hourly vuh").
		Select(`
			COALESCE(CAST(SUM((vuh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(vuh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(vuh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(vuh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("vuh.organization_id = ? AND vuh.hour >= ? AND vuh.hour < ?", orgID, billingPeriodStart, billingPeriodEnd).
		Scan(&vpsHourlyUsage)

	// Combine deployment, game server, and VPS hourly usage
	hourlyUsage.CPUCoreSeconds += gameServerHourlyUsage.CPUCoreSeconds + vpsHourlyUsage.CPUCoreSeconds
	hourlyUsage.MemoryByteSeconds += gameServerHourlyUsage.MemoryByteSeconds + vpsHourlyUsage.MemoryByteSeconds
	hourlyUsage.BandwidthRxBytes += gameServerHourlyUsage.BandwidthRxBytes + vpsHourlyUsage.BandwidthRxBytes
	hourlyUsage.BandwidthTxBytes += gameServerHourlyUsage.BandwidthTxBytes + vpsHourlyUsage.BandwidthTxBytes

	// Get storage bytes (snapshot from deployments, game servers, and VPS tables)
	var deploymentStorage struct {
		StorageBytes int64
	}
	database.DB.Table("deployments d").
		Select("COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
		Where("d.organization_id = ?", orgID).
		Scan(&deploymentStorage)

	var gameServerStorage struct {
		StorageBytes int64
	}
	database.DB.Table("game_servers gs").
		Select("COALESCE(SUM(gs.storage_bytes), 0) as storage_bytes").
		Where("gs.organization_id = ?", orgID).
		Scan(&gameServerStorage)

	var vpsStorage struct {
		StorageBytes int64
	}
	database.DB.Table("vps_instances vps").
		Select("COALESCE(SUM(vps.disk_bytes), 0) as storage_bytes").
		Where("vps.organization_id = ? AND vps.deleted_at IS NULL", orgID).
		Scan(&vpsStorage)

	storageSum := struct {
		StorageBytes int64
	}{
		StorageBytes: deploymentStorage.StorageBytes + gameServerStorage.StorageBytes + vpsStorage.StorageBytes,
	}

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

	// Calculate public IP costs (flat rate, prorated based on billing period)
	var publicIPCost int64
	var publicIPAssignments []struct {
		MonthlyCostCents int64
		AssignedAt       time.Time
	}
	database.DB.Table("vps_public_ips ip").
		Select("ip.monthly_cost_cents, ip.assigned_at").
		Joins("INNER JOIN vps_instances vps ON vps.id = ip.vps_id").
		Where("ip.organization_id = ? AND ip.vps_id IS NOT NULL AND vps.deleted_at IS NULL", orgID).
		Where("ip.assigned_at IS NOT NULL AND ip.assigned_at <= ?", billingPeriodEnd).
		Scan(&publicIPAssignments)
	
	for _, assignment := range publicIPAssignments {
		// Calculate prorated cost based on when IP was assigned
		assignmentDate := assignment.AssignedAt
		if assignmentDate.Before(billingPeriodStart) {
			assignmentDate = billingPeriodStart
		}
		
		periodDuration := billingPeriodEnd.Sub(assignmentDate)
		daysInMonth := float64(time.Date(billingPeriodEnd.Year(), billingPeriodEnd.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
		daysInPeriod := periodDuration.Hours() / 24.0
		
		if daysInPeriod > 0 && daysInMonth > 0 {
			proratedCost := int64(float64(assignment.MonthlyCostCents) * (daysInPeriod / daysInMonth))
			publicIPCost += proratedCost
		} else {
			publicIPCost += assignment.MonthlyCostCents
		}
	}

	totalCostCents := cpuCost + memoryCost + bandwidthCost + storageCost + publicIPCost

	// Create usage breakdown
	breakdown := UsageBreakdown{
		CPUCostCents:       cpuCost,
		MemoryCostCents:    memoryCost,
		BandwidthCostCents: bandwidthCost,
		StorageCostCents:   storageCost,
		PublicIPCostCents:  publicIPCost,
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

// GenerateCurrentBillEarly generates a bill for the current billing period ending at the current date/time
// This allows users to create and pay bills before their scheduled billing date
func GenerateCurrentBillEarly(orgID string) (*database.MonthlyBill, bool, error) {
	now := time.Now()
	
	// Get the billing account to find the billing date
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		return nil, false, fmt.Errorf("billing account not found: %w", err)
	}

	if billingAccount.BillingDate == nil {
		return nil, false, fmt.Errorf("billing date not set for organization %s", orgID)
	}

	billingDay := *billingAccount.BillingDate

	// Calculate billing period
	var org database.Organization
	if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
		return nil, false, fmt.Errorf("organization not found: %w", err)
	}

	// Find the last bill to determine the start of this billing period
	var lastBill database.MonthlyBill
	var billingPeriodStart time.Time
	var billingPeriodEnd time.Time

	if err := database.DB.Where("organization_id = ? AND status IN ?", orgID, []string{"PAID", "PENDING"}).
		Order("billing_period_end DESC").First(&lastBill).Error; err == nil {
		// Use the end of the last billing period as the start of this one
		billingPeriodStart = lastBill.BillingPeriodEnd
	} else {
		// First bill: start from org creation date
		billingPeriodStart = org.CreatedAt.UTC().Truncate(24 * time.Hour)
	}

	// Calculate the next billing date
	nextBillingDate := time.Date(now.Year(), now.Month(), billingDay, 0, 0, 0, 0, time.UTC)
	if nextBillingDate.Before(now) || nextBillingDate.Equal(now) {
		// If the billing date has passed this month, use next month
		nextBillingDate = nextBillingDate.AddDate(0, 1, 0)
	}

	// Use the earlier of: now or next billing date
	// This ensures we don't create bills for future periods
	if now.Before(nextBillingDate) {
		billingPeriodEnd = now.UTC().Truncate(time.Hour) // Round to hour for consistency
	} else {
		billingPeriodEnd = nextBillingDate
	}

	// Check if a bill already exists for this period (or overlapping period)
	var existingBill database.MonthlyBill
	if err := database.DB.Where("organization_id = ? AND billing_period_start = ? AND billing_period_end = ?",
		orgID, billingPeriodStart, billingPeriodEnd).First(&existingBill).Error; err == nil {
		log.Printf("[Generate Current Bill] Bill already exists for org %s for period %s to %s", orgID,
			billingPeriodStart.Format("2006-01-02"), billingPeriodEnd.Format("2006-01-02"))
		return &existingBill, true, nil
	}

	// Also check if there's a bill that covers this period (e.g., if billing date already passed)
	var overlappingBill database.MonthlyBill
	if err := database.DB.Where("organization_id = ? AND billing_period_start <= ? AND billing_period_end >= ?",
		orgID, billingPeriodStart, billingPeriodEnd).First(&overlappingBill).Error; err == nil {
		log.Printf("[Generate Current Bill] Overlapping bill exists for org %s", orgID)
		return &overlappingBill, true, nil
	}

	// Calculate usage for the billing period
	metricsDB := database.GetMetricsDB()
	if metricsDB == nil {
		return nil, false, fmt.Errorf("metrics database not available")
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

	// Get game server usage from hourly aggregates and add to deployment usage
	var gameServerHourlyUsage struct {
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
	}
	metricsDB.Table("game_server_usage_hourly gsuh").
		Select(`
			COALESCE(CAST(SUM((gsuh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(gsuh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(gsuh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(gsuh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("gsuh.organization_id = ? AND gsuh.hour >= ? AND gsuh.hour < ?", orgID, billingPeriodStart, billingPeriodEnd).
		Scan(&gameServerHourlyUsage)

	// Get VPS usage from hourly aggregates and add to deployment/game server usage
	var vpsHourlyUsage struct {
		CPUCoreSeconds    int64
		MemoryByteSeconds int64
		BandwidthRxBytes  int64
		BandwidthTxBytes  int64
	}
	metricsDB.Table("vps_usage_hourly vuh").
		Select(`
			COALESCE(CAST(SUM((vuh.avg_cpu_usage / 100.0) * 3600) AS BIGINT), 0) as cpu_core_seconds,
			COALESCE(CAST(SUM(vuh.avg_memory_usage * 3600) AS BIGINT), 0) as memory_byte_seconds,
			COALESCE(SUM(vuh.bandwidth_rx_bytes), 0) as bandwidth_rx_bytes,
			COALESCE(SUM(vuh.bandwidth_tx_bytes), 0) as bandwidth_tx_bytes
		`).
		Where("vuh.organization_id = ? AND vuh.hour >= ? AND vuh.hour < ?", orgID, billingPeriodStart, billingPeriodEnd).
		Scan(&vpsHourlyUsage)

	// Combine deployment, game server, and VPS hourly usage
	hourlyUsage.CPUCoreSeconds += gameServerHourlyUsage.CPUCoreSeconds + vpsHourlyUsage.CPUCoreSeconds
	hourlyUsage.MemoryByteSeconds += gameServerHourlyUsage.MemoryByteSeconds + vpsHourlyUsage.MemoryByteSeconds
	hourlyUsage.BandwidthRxBytes += gameServerHourlyUsage.BandwidthRxBytes + vpsHourlyUsage.BandwidthRxBytes
	hourlyUsage.BandwidthTxBytes += gameServerHourlyUsage.BandwidthTxBytes + vpsHourlyUsage.BandwidthTxBytes

	// Get storage bytes (snapshot from deployments, game servers, and VPS tables)
	var deploymentStorage struct {
		StorageBytes int64
	}
	database.DB.Table("deployments d").
		Select("COALESCE(SUM(d.storage_bytes), 0) as storage_bytes").
		Where("d.organization_id = ?", orgID).
		Scan(&deploymentStorage)

	var gameServerStorage struct {
		StorageBytes int64
	}
	database.DB.Table("game_servers gs").
		Select("COALESCE(SUM(gs.storage_bytes), 0) as storage_bytes").
		Where("gs.organization_id = ?", orgID).
		Scan(&gameServerStorage)

	var vpsStorage struct {
		StorageBytes int64
	}
	database.DB.Table("vps_instances vps").
		Select("COALESCE(SUM(vps.disk_bytes), 0) as storage_bytes").
		Where("vps.organization_id = ? AND vps.deleted_at IS NULL", orgID).
		Scan(&vpsStorage)

	storageSum := struct {
		StorageBytes int64
	}{
		StorageBytes: deploymentStorage.StorageBytes + gameServerStorage.StorageBytes + vpsStorage.StorageBytes,
	}

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

	// Calculate public IP costs (flat rate, prorated based on billing period)
	var publicIPCost int64
	var publicIPAssignments []struct {
		MonthlyCostCents int64
		AssignedAt       time.Time
	}
	database.DB.Table("vps_public_ips ip").
		Select("ip.monthly_cost_cents, ip.assigned_at").
		Joins("INNER JOIN vps_instances vps ON vps.id = ip.vps_id").
		Where("ip.organization_id = ? AND ip.vps_id IS NOT NULL AND vps.deleted_at IS NULL", orgID).
		Where("ip.assigned_at IS NOT NULL AND ip.assigned_at <= ?", billingPeriodEnd).
		Scan(&publicIPAssignments)
	
	for _, assignment := range publicIPAssignments {
		// Calculate prorated cost based on when IP was assigned
		assignmentDate := assignment.AssignedAt
		if assignmentDate.Before(billingPeriodStart) {
			assignmentDate = billingPeriodStart
		}
		
		periodDuration := billingPeriodEnd.Sub(assignmentDate)
		daysInMonth := float64(time.Date(billingPeriodEnd.Year(), billingPeriodEnd.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
		daysInPeriod := periodDuration.Hours() / 24.0
		
		if daysInPeriod > 0 && daysInMonth > 0 {
			proratedCost := int64(float64(assignment.MonthlyCostCents) * (daysInPeriod / daysInMonth))
			publicIPCost += proratedCost
		} else {
			publicIPCost += assignment.MonthlyCostCents
		}
	}

	totalCostCents := cpuCost + memoryCost + bandwidthCost + storageCost + publicIPCost

	// Create usage breakdown
	breakdown := UsageBreakdown{
		CPUCostCents:       cpuCost,
		MemoryCostCents:    memoryCost,
		BandwidthCostCents: bandwidthCost,
		StorageCostCents:   storageCost,
		PublicIPCostCents:  publicIPCost,
		TotalCostCents:     totalCostCents,
	}

	breakdownJSON, err := json.Marshal(breakdown)
	if err != nil {
		return nil, false, fmt.Errorf("marshal breakdown: %w", err)
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

	// Create the bill (but don't auto-pay - let user pay it manually)
	if err := database.DB.Create(bill).Error; err != nil {
		return nil, false, fmt.Errorf("create bill: %w", err)
	}

	log.Printf("[Generate Current Bill] Created bill %s for org %s: %d cents (period %s to %s)",
		billID, orgID, totalCostCents, billingPeriodStart.Format("2006-01-02"), billingPeriodEnd.Format("2006-01-02"))

	return bill, false, nil
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

