package billing

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/gorm"
)

// Lock and billing queries share one transaction and one pool connection.
// Transaction-scoped locks are released on rollback, cancellation or commit.
func withBillingLock(ctx context.Context, orgID string, operation func(*gorm.DB) error) error {
	return database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "monthly-billing:"+orgID).Error; err != nil {
				return err
			}
		}
		return operation(tx)
	})
}

func billingMetricsDB(tx *gorm.DB) (*gorm.DB, error) {
	metrics := database.GetMetricsDB()
	if metrics == nil {
		return nil, fmt.Errorf("metrics database not available")
	}
	metricsPool, err := metrics.DB()
	if err != nil {
		return nil, err
	}
	applicationPool, err := database.DB.DB()
	if err != nil {
		return nil, err
	}
	if metricsPool == applicationPool {
		// Self-hosted setups may share the primary database for metrics.
		return tx, nil
	}
	return metrics.WithContext(tx.Statement.Context), nil
}
