package auth

import (
	"connectrpc.com/connect"
	sharedauth "github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	adminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/admin/v1"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestQuotaEditorLoadsAndClearsOverrides(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old; raw.Close() })
	if err := db.AutoMigrate(&database.Organization{}, &database.OrganizationPlan{}, &database.OrgQuota{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []interface{}{&database.Organization{ID: "quota-org", Slug: "quota-org", Status: "active"}, &database.OrganizationPlan{ID: "plan-starter", Name: "Starter", DeploymentsMax: 10}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := sharedauth.WithUser(t.Context(), &authv1.User{Id: "test-admin", Roles: []string{sharedauth.RoleSuperAdmin}})
	s := &AdminService{}
	request := &adminv1.UpsertOrgQuotaRequest{OrganizationId: "quota-org", DeploymentsMaxOverride: 3, MemoryBytesOverride: 4096}
	if _, err := s.UpsertOrgQuota(ctx, connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.GetOrgQuota(ctx, connect.NewRequest(&adminv1.GetOrgQuotaRequest{OrganizationId: "quota-org"}))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Msg.Quota.DeploymentsMaxOverride != 3 || loaded.Msg.Quota.MemoryBytesOverride != 4096 {
		t.Fatalf("saved limits not loaded: %v", loaded.Msg)
	}
	if loaded.Msg.PlanDefaults.DeploymentsMax != 10 || loaded.Msg.Effective.DeploymentsMax != 3 {
		t.Fatal("effective limits missing")
	}
	request.DeploymentsMaxOverride = 20
	if _, err := s.UpsertOrgQuota(ctx, connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	loaded, err = s.GetOrgQuota(ctx, connect.NewRequest(&adminv1.GetOrgQuotaRequest{OrganizationId: "quota-org"}))
	if err != nil || loaded.Msg.Effective.DeploymentsMax != 10 || loaded.Msg.Quota.DeploymentsMaxOverride != 20 {
		t.Fatal("plan cap was not distinguished from override")
	}
	request.DeploymentsMaxOverride = 0
	request.MemoryBytesOverride = 0
	if _, err := s.UpsertOrgQuota(ctx, connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	var quota database.OrgQuota
	if err := db.First(&quota, "organization_id = ?", "quota-org").Error; err != nil {
		t.Fatal(err)
	}
	if quota.DeploymentsMaxOverride != nil || quota.MemoryBytesOverride != nil || quota.PlanID != "plan-starter" {
		t.Fatalf("zero did not restore plan inheritance: %+v", quota)
	}
	request.CpuCoresOverride = -1
	if _, err := s.UpsertOrgQuota(ctx, connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("negative override accepted: %v", err)
	}
}
