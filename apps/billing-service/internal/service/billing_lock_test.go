package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestBillingCompletesWithSingleConnectionPool(t *testing.T) {
	for _, early := range []bool{false, true} {
		t.Run(fmt.Sprintf("early=%v", early), func(t *testing.T) {
			db := monthlyRecoveryDB(t)
			now := time.Now().UTC()
			seedOverdueAccount(t, db, now.AddDate(0, -2, 0), 1)
			pool, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			pool.SetMaxOpenConns(1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = withBillingLock(ctx, "org-synthetic", func(tx *gorm.DB) error {
				if early {
					_, _, err := generateCurrentBillEarlyLocked(tx, "org-synthetic")
					return err
				}
				return processOrganizationBillingLocked(tx, "org-synthetic", now)
			})
			if err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := db.Model(&database.MonthlyBill{}).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("bills=%d err=%v", count, err)
			}
		})
	}
}

func TestConcurrentBillingDoesNotExhaustPool(t *testing.T) {
	db := monthlyRecoveryDB(t)
	now := time.Now().UTC()
	day := 1
	for i := range 4 {
		id := fmt.Sprintf("org-%d", i)
		if err := db.Create(&database.Organization{ID: id, Slug: id, Status: "active", CreatedAt: now.AddDate(0, -2, 0)}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&database.BillingAccount{ID: fmt.Sprintf("account-%d", i), OrganizationID: id, Status: "ACTIVE", BillingDate: &day}).Error; err != nil {
			t.Fatal(err)
		}
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := make(chan error, 4)
	for i := range 4 {
		go func(i int) {
			id := fmt.Sprintf("org-%d", i)
			results <- withBillingLock(ctx, id, func(tx *gorm.DB) error { _, _, err := generateCurrentBillEarlyLocked(tx, id); return err })
		}(i)
	}
	for range 4 {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	var count int64
	if err := db.Model(&database.MonthlyBill{}).Count(&count).Error; err != nil || count != 4 {
		t.Fatalf("bills=%d err=%v", count, err)
	}
}

func TestBillingCancellationReleasesConnectionAndLock(t *testing.T) {
	db := monthlyRecoveryDB(t)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := withBillingLock(ctx, "org-synthetic", func(tx *gorm.DB) error { return tx.Exec("SELECT pg_sleep(10)").Error })
	if err == nil || ctx.Err() == nil {
		t.Fatalf("query ignored cancellation: %v", err)
	}
	retry, cancelRetry := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelRetry()
	if err := withBillingLock(retry, "org-synthetic", func(tx *gorm.DB) error { return tx.Exec("SELECT 1").Error }); err != nil {
		t.Fatal(err)
	}
}

func TestSeparateMetricsQueriesInheritBillingDeadline(t *testing.T) {
	db := monthlyRecoveryDB(t)
	previous := database.MetricsDB
	// A separate pool for the same fixture represents a separate metrics server.
	dsn := db.Dialector.(*postgres.Dialector).Config.DSN
	metrics, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.MetricsDB = metrics
	t.Cleanup(func() { database.MetricsDB = previous; pool, _ := metrics.DB(); pool.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = withBillingLock(ctx, "org-synthetic", func(tx *gorm.DB) error {
		metrics, err := billingMetricsDB(tx)
		if err != nil {
			return err
		}
		return metrics.Exec("SELECT pg_sleep(10)").Error
	})
	if err == nil || ctx.Err() == nil {
		t.Fatalf("metrics ignored deadline: %v", err)
	}
}
