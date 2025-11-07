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
			// Create new billing account
			billingAccount = database.BillingAccount{
				ID:             GenerateID("ba"),
				OrganizationID: orgID,
				Status:         "ACTIVE",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := database.DB.Create(&billingAccount).Error; err != nil {
				return nil, fmt.Errorf("create billing account: %w", err)
			}
		} else {
			return nil, err
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

