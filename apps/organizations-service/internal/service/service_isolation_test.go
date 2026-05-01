package organizations

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	organizationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/organizations/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrganizationServiceTenantIsolation(t *testing.T) {
	db := newOrganizationServiceTestDB(t)
	service := NewService(Config{}).(*Service)
	seedOrganizationServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	list, err := service.ListOrganizations(ctx, connect.NewRequest(&organizationsv1.ListOrganizationsRequest{}))
	if err != nil {
		t.Fatalf("list organizations: %v", err)
	}
	if got := organizationIDs(list.Msg.Organizations); !slices.Equal(got, []string{"org-a"}) {
		t.Fatalf("list returned %v, want only user's organization", got)
	}

	_, err = service.GetOrganization(ctx, connect.NewRequest(&organizationsv1.GetOrganizationRequest{
		OrganizationId: "org-b",
	}))
	if err == nil {
		t.Fatal("cross-org get succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org get code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	_, err = service.UpdateOrganization(ctx, connect.NewRequest(&organizationsv1.UpdateOrganizationRequest{
		OrganizationId: "org-b",
		Name:           proto.String("Cross Org Edit"),
	}))
	if err == nil {
		t.Fatal("cross-org update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgB database.Organization
	if err := db.First(&orgB, "id = ?", "org-b").Error; err != nil {
		t.Fatalf("fetch org-b: %v", err)
	}
	if orgB.Name != "Org B" {
		t.Fatalf("cross-org update changed org-b name to %q", orgB.Name)
	}

	updateOrgA, err := service.UpdateOrganization(ctx, connect.NewRequest(&organizationsv1.UpdateOrganizationRequest{
		OrganizationId: "org-a",
		Name:           proto.String("Org A Updated"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.Organization.GetName(); got != "Org A Updated" {
		t.Fatalf("same-org update name = %q, want updated name", got)
	}
}

func newOrganizationServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.Organization{},
		&database.OrganizationMember{},
		&database.OrganizationPlan{},
		&database.OrgQuota{},
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

func seedOrganizationServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	records := []any{
		&database.OrganizationPlan{ID: "plan-starter", Name: "Starter", Description: "Starter"},
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Plan: "starter", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Plan: "starter", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func organizationIDs(organizations []*organizationsv1.Organization) []string {
	ids := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		ids = append(ids, organization.GetId())
	}
	slices.Sort(ids)
	return ids
}
