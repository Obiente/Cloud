package billing

import (
	"fmt"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/gorm"
)

func TestLatestBillingBoundary(t *testing.T) {
	for _, tc := range []struct {
		now  string
		day  int
		want string
	}{
		{"2026-10-27", 26, "2026-10-26"}, {"2026-10-25", 26, "2026-09-26"},
		{"2026-02-28", 31, "2026-02-28"}, {"2028-02-29", 31, "2028-02-29"},
		{"2026-03-01", 30, "2026-02-28"}, {"2027-01-01", 31, "2026-12-31"},
		{"2026-10-27", 27, "2026-10-27"},
	} {
		now, _ := time.Parse("2006-01-02", tc.now)
		got, err := latestBillingBoundary(now, tc.day)
		if err != nil || got.Format("2006-01-02") != tc.want {
			t.Fatalf("%s day %d: %v %v", tc.now, tc.day, got, err)
		}
	}
	for _, day := range []int{0, 32} {
		if _, err := latestBillingBoundary(time.Now(), day); err == nil {
			t.Fatal("invalid billing day accepted")
		}
	}
}

func monthlyRecoveryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := billingWebhookTestDB(t)
	if err := db.AutoMigrate(&database.Deployment{}, &database.GameServer{}, &database.DatabaseInstance{}, &database.VPSInstance{}, &database.VPSPublicIP{}); err != nil {
		t.Fatal(err)
	}
	createMonthlyUsageTables(t, db)
	previous := database.MetricsDB
	database.MetricsDB = db
	t.Cleanup(func() { database.MetricsDB = previous })
	return db
}

func createMonthlyUsageTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, table := range []string{"deployment_usage_hourly", "game_server_usage_hourly", "vps_usage_hourly", "database_usage_hourly"} {
		if err := db.Exec("CREATE TABLE IF NOT EXISTS " + table + " (organization_id text, hour timestamptz, avg_cpu_usage double precision, avg_memory_usage bigint, bandwidth_rx_bytes bigint, bandwidth_tx_bytes bigint)").Error; err != nil {
			t.Fatal(err)
		}
	}
}

func seedOverdueAccount(t *testing.T, db *gorm.DB, created time.Time, day int) {
	t.Helper()
	if err := db.Create(&database.Organization{ID: "org-synthetic", Slug: "synthetic", Status: "active", CreatedAt: created}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.BillingAccount{ID: "account", OrganizationID: "org-synthetic", Status: "ACTIVE", BillingDate: &day}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestMonthlyBillingCatchesUpAfterOutageAndMonthEnd(t *testing.T) {
	for _, tc := range []struct {
		now string
		day int
		end string
	}{{"2026-10-27", 26, "2026-10-26"}, {"2026-03-02", 31, "2026-02-28"}} {
		t.Run(tc.now, func(t *testing.T) {
			db := monthlyRecoveryDB(t)
			now, _ := time.Parse("2006-01-02", tc.now)
			seedOverdueAccount(t, db, now.AddDate(0, -2, 0), tc.day)
			for range 2 {
				if err := processMonthlyBillingAt(now); err != nil {
					t.Fatal(err)
				}
			}
			var bills []database.MonthlyBill
			if err := db.Find(&bills).Error; err != nil {
				t.Fatal(err)
			}
			if len(bills) != 1 || bills[0].BillingPeriodEnd.Format("2006-01-02") != tc.end {
				t.Fatalf("bills=%+v", bills)
			}
			if !bills[0].BillingPeriodEnd.After(bills[0].BillingPeriodStart) {
				t.Fatal("empty billing period")
			}
		})
	}
}

func TestMonthlyBillingRetriesUnavailableUsage(t *testing.T) {
	db := monthlyRecoveryDB(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	seedOverdueAccount(t, db, now.AddDate(0, -1, 0), 26)
	if err := db.Exec("DROP TABLE game_server_usage_hourly").Error; err != nil {
		t.Fatal(err)
	}
	if err := processMonthlyBillingAt(now); err == nil {
		t.Fatal("missing usage was treated as zero")
	}
	var count int64
	db.Model(&database.MonthlyBill{}).Count(&count)
	if count != 0 {
		t.Fatal("incomplete bill persisted")
	}
	createMonthlyUsageTables(t, db)
	if err := processMonthlyBillingAt(now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	db.Model(&database.MonthlyBill{}).Count(&count)
	if count != 1 {
		t.Fatalf("recovered bills=%d", count)
	}
}

func TestConcurrentMonthlyBillingDoesNotDuplicatePeriods(t *testing.T) {
	db := monthlyRecoveryDB(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	seedOverdueAccount(t, db, now.AddDate(0, -1, 0), 26)
	errs := make(chan error, 2)
	start := make(chan struct{})
	for range 2 {
		go func() { <-start; errs <- processMonthlyBillingAt(now) }()
	}
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&database.MonthlyBill{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal(fmt.Sprintf("duplicate periods: %d", count))
	}
}
