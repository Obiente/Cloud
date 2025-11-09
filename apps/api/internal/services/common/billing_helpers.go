package common

import (
	"errors"
	"fmt"
	"time"

	"api/internal/database"

	"gorm.io/gorm"
)

// GetOrCreateBillingAccount gets an existing billing account or creates a new one.
func GetOrCreateBillingAccount(orgID string) (*database.BillingAccount, error) {
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Get organization to set billing date to creation day
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
				return nil, fmt.Errorf("organization not found: %w", err)
			}
			
			// Set billing date to the day the org was created (1-31)
			billingDay := org.CreatedAt.Day()
			billingAccount = database.BillingAccount{
				ID:             GenerateID("ba"),
				OrganizationID: orgID,
				Status:         "ACTIVE",
				BillingDate:    &billingDay,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := database.DB.Create(&billingAccount).Error; err != nil {
				return nil, fmt.Errorf("create billing account: %w", err)
			}
		} else {
			return nil, err
		}
	} else {
		// If billing account exists but billing_date is not set, set it to org creation day
		if billingAccount.BillingDate == nil {
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", orgID).Error; err == nil {
				billingDay := org.CreatedAt.Day()
				billingAccount.BillingDate = &billingDay
				database.DB.Save(&billingAccount)
			}
		}
	}
	return &billingAccount, nil
}

// GetBillingAccount retrieves a billing account, returning nil if not found.
func GetBillingAccount(orgID string) (*database.BillingAccount, error) {
	var billingAccount database.BillingAccount
	if err := database.DB.Where("organization_id = ?", orgID).First(&billingAccount).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &billingAccount, nil
}

// GenerateID generates a unique ID with a prefix.
func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

