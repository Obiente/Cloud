package superadmin

import (
	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func remediationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&database.Organization{}, &database.OrganizationMember{}, &database.Deployment{}, &database.DNSDelegationAPIKey{}, &database.SuperadminRoleBinding{}); err != nil {
		t.Fatal(err)
	}
	old := database.DB
	oldMetrics := database.MetricsDB
	database.DB = db
	database.MetricsDB = nil
	t.Cleanup(func() { database.DB = old; database.MetricsDB = oldMetrics; raw.Close() })
	return db
}
func TestOrganizationOverviewCountsIndependentRelations(t *testing.T) {
	db := remediationDB(t)
	now := time.Now()
	rows := []interface{}{
		&database.Organization{ID: "test-org", Slug: "test-org", Status: "active"},
		&database.OrganizationMember{ID: "owner", OrganizationID: "test-org", UserID: "owner", Role: "system:owner", Status: "active"},
		&database.OrganizationMember{ID: "member", OrganizationID: "test-org", UserID: "member", Role: "system:member", Status: "active"},
		&database.OrganizationMember{ID: "invited", OrganizationID: "test-org", UserID: "pending:test@example.invalid", Status: "invited"},
		&database.Deployment{ID: "one", OrganizationID: "test-org"}, &database.Deployment{ID: "two", OrganizationID: "test-org"}, &database.Deployment{ID: "deleted", OrganizationID: "test-org", DeletedAt: &now},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	counts, err := aggregateCounts()
	if err != nil {
		t.Fatal(err)
	}
	if counts.TotalDeployments != 2 {
		t.Fatalf("deleted deployment counted: %d", counts.TotalDeployments)
	}
	orgs, err := loadOrganizationOverviews()
	if err != nil {
		t.Fatal(err)
	}
	if len(orgs) != 1 || orgs[0].MemberCount != 2 || orgs[0].InviteCount != 1 || orgs[0].DeploymentCount != 2 {
		t.Fatalf("multiplied overview counts: %v", orgs)
	}
	deployments, err := loadDeploymentOverviews()
	if err != nil {
		t.Fatal(err)
	}
	if len(deployments) != 2 {
		t.Fatalf("deleted deployment listed: %d", len(deployments))
	}
}
func TestDNSRevokeByIDRejectsCredentialsForOnlySelectedKey(t *testing.T) {
	db := remediationDB(t)
	for _, id := range []string{"org-key", "team-key", "keep-key"} {
		org := "test-org"
		if id == "team-key" {
			org = ""
		}
		if err := db.Create(&database.DNSDelegationAPIKey{ID: id, KeyHash: database.HashAPIKeyForDelegation("synthetic-" + id), OrganizationID: org, IsActive: true}).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := &Service{}
	ctx := auth.WithUser(t.Context(), &authv1.User{Id: "test-admin", Roles: []string{auth.RoleSuperAdmin}})
	for _, id := range []string{"org-key", "team-key"} {
		if _, err := service.RevokeDNSDelegationAPIKey(ctx, connect.NewRequest(&superadminv1.RevokeDNSDelegationAPIKeyRequest{KeyId: id})); err != nil {
			t.Fatal(err)
		}
		if valid, _ := database.ValidateDNSDelegationAPIKey("synthetic-" + id); valid {
			t.Fatalf("revoked key remains usable: %s", id)
		}
	}
	if valid, err := database.ValidateDNSDelegationAPIKey("synthetic-keep-key"); !valid || err != nil {
		t.Fatalf("unselected key revoked: %v", err)
	}
	unauthorized := auth.WithUser(t.Context(), &authv1.User{Id: "test-user"})
	if _, err := service.RevokeDNSDelegationAPIKey(unauthorized, connect.NewRequest(&superadminv1.RevokeDNSDelegationAPIKeyRequest{KeyId: "keep-key"})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unauthorized revocation: %v", err)
	}
}
func TestAbuseDetectionDoesNotReportHealthyWhenMetricsUnavailable(t *testing.T) {
	remediationDB(t)
	if _, err := DetectAbuse(t.Context()); err == nil {
		t.Fatal("missing telemetry reported as no abuse")
	}
}

func TestPartialAbuseDetectorFailureIsNotHealthy(t *testing.T) {
	db := remediationDB(t)
	database.MetricsDB = db
	if _, err := detectSSHBruteForce(t.Context(), time.Now().Add(-24*time.Hour)); err == nil {
		t.Fatal("missing audit table was ignored")
	}
	if _, err := detectAPIAbuse(t.Context(), time.Now().Add(-24*time.Hour)); err == nil {
		t.Fatal("failed API detector was ignored")
	}
}
