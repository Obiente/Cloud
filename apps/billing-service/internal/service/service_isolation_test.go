package billing

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	billingv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/billing/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBillingServiceTenantIsolation(t *testing.T) {
	db := newBillingServiceTestDB(t)
	service := &Service{billingEnabled: true}
	seedBillingServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	account, err := service.GetBillingAccount(ctx, connect.NewRequest(&billingv1.GetBillingAccountRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("get org-a billing account: %v", err)
	}
	if got := account.Msg.Account.GetOrganizationId(); got != "org-a" {
		t.Fatalf("billing account organization = %q, want org-a", got)
	}

	_, err = service.GetBillingAccount(ctx, connect.NewRequest(&billingv1.GetBillingAccountRequest{
		OrganizationId: "org-b",
	}))
	if err == nil {
		t.Fatal("cross-org billing account get succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org get code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	bills, err := service.ListBills(ctx, connect.NewRequest(&billingv1.ListBillsRequest{
		OrganizationId: "org-a",
		Limit:          proto.Int32(10),
	}))
	if err != nil {
		t.Fatalf("list org-a bills: %v", err)
	}
	if got := billIDs(bills.Msg.Bills); !slices.Equal(got, []string{"bill-org-a"}) {
		t.Fatalf("org-a bills returned %v, want only org-a bill", got)
	}

	_, err = service.ListBills(ctx, connect.NewRequest(&billingv1.ListBillsRequest{
		OrganizationId: "org-b",
	}))
	if err == nil {
		t.Fatal("cross-org bill list succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org bill list code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	_, err = service.UpdateBillingAccount(ctx, connect.NewRequest(&billingv1.UpdateBillingAccountRequest{
		OrganizationId: "org-b",
		BillingEmail:   proto.String("cross-org@example.com"),
	}))
	if err == nil {
		t.Fatal("cross-org billing account update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgBAccount database.BillingAccount
	if err := db.First(&orgBAccount, "organization_id = ?", "org-b").Error; err != nil {
		t.Fatalf("fetch org-b billing account: %v", err)
	}
	if orgBAccount.BillingEmail == nil || *orgBAccount.BillingEmail != "billing-b@example.com" {
		t.Fatalf("cross-org update changed org-b billing email to %v", orgBAccount.BillingEmail)
	}

	updateOrgA, err := service.UpdateBillingAccount(ctx, connect.NewRequest(&billingv1.UpdateBillingAccountRequest{
		OrganizationId: "org-a",
		BillingEmail:   proto.String("billing-a-updated@example.com"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.Account.GetBillingEmail(); got != "billing-a-updated@example.com" {
		t.Fatalf("same-org billing email = %q, want updated email", got)
	}
}

func newBillingServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.Organization{},
		&database.OrganizationMember{},
		&database.BillingAccount{},
		&database.MonthlyBill{},
	); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func seedBillingServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	billingA := "billing-a@example.com"
	billingB := "billing-b@example.com"
	records := []any{
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.BillingAccount{ID: "billing-org-a", OrganizationID: "org-a", Status: "ACTIVE", BillingEmail: &billingA, CreatedAt: now, UpdatedAt: now},
		&database.BillingAccount{ID: "billing-org-b", OrganizationID: "org-b", Status: "ACTIVE", BillingEmail: &billingB, CreatedAt: now, UpdatedAt: now},
		testMonthlyBill("bill-org-a", "org-a", now),
		testMonthlyBill("bill-org-b", "org-b", now),
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func testMonthlyBill(id, orgID string, now time.Time) *database.MonthlyBill {
	return &database.MonthlyBill{
		ID:                 id,
		OrganizationID:     orgID,
		BillingPeriodStart: now.AddDate(0, -1, 0),
		BillingPeriodEnd:   now,
		AmountCents:        1234,
		Status:             "PENDING",
		DueDate:            now.AddDate(0, 0, 14),
		UsageBreakdown:     "{}",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func billIDs(bills []*billingv1.MonthlyBill) []string {
	ids := make([]string, 0, len(bills))
	for _, bill := range bills {
		ids = append(ids, bill.GetId())
	}
	slices.Sort(ids)
	return ids
}
