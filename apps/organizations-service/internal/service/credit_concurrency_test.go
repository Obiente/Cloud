package organizations

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	organizationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/organizations/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestOrganizationEditPreservesConcurrentCreditUpdate(t *testing.T) {
	db := newOrganizationServiceTestDB(t)
	seedOrganizationServiceIsolationData(t, db)
	var injected atomic.Bool
	if err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_credit", func(tx *gorm.DB) {
		if tx.Statement.Table == "organizations" && injected.CompareAndSwap(false, true) {
			if err := tx.Session(&gorm.Session{NewDB: true}).Exec("UPDATE organizations SET credits = 1234 WHERE id = ?", "org-a").Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Email: "owner@example.test"})
	_, err := (&Service{}).UpdateOrganization(ctx, connect.NewRequest(&organizationsv1.UpdateOrganizationRequest{OrganizationId: "org-a", Name: proto.String("Renamed")}))
	if err != nil {
		t.Fatal(err)
	}
	var org database.Organization
	if err := db.First(&org, "id = ?", "org-a").Error; err != nil {
		t.Fatal(err)
	}
	if org.Credits != 1234 || org.Name != "Renamed" {
		t.Fatalf("credits=%d name=%s", org.Credits, org.Name)
	}
}

func TestConcurrentAdminCreditAdjustments(t *testing.T) {
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("org_credits_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
		raw, _ := db.DB()
		raw.Close()
		base.Exec("DROP SCHEMA " + schema + " CASCADE")
		raw, _ = base.DB()
		raw.Close()
	})
	if err := db.AutoMigrate(&database.Organization{}, &database.OrgQuota{}, &database.OrganizationPlan{}, &database.CreditTransaction{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.Organization{ID: "org-credit", Slug: "credit", Status: "active", Credits: 1000}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.OrganizationPlan{ID: "plan-starter", Name: "Starter"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.OrgQuota{OrganizationID: "org-credit", PlanID: "plan-starter"}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "admin-test", Roles: []string{auth.RoleSuperAdmin}})
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := (&Service{}).AdminAddCredits(ctx, connect.NewRequest(&organizationsv1.AdminAddCreditsRequest{OrganizationId: "org-credit", AmountCents: 100}))
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := (&Service{}).AdminRemoveCredits(ctx, connect.NewRequest(&organizationsv1.AdminRemoveCreditsRequest{OrganizationId: "org-credit", AmountCents: 100}))
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var org database.Organization
	if err := db.First(&org, "id = ?", "org-credit").Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&database.CreditTransaction{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if org.Credits != 1000 || count != 2 {
		t.Fatalf("balance=%d ledger=%d", org.Credits, count)
	}
}
