package deployments

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeploymentServiceTenantIsolation(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	service := NewService(context.Background(), database.NewDeploymentRepository(db, nil), nil, nil)

	seedDeploymentServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	listOrgA, err := service.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a deployments: %v", err)
	}

	orgAIDs := deploymentIDs(listOrgA.Msg.Deployments)
	if !slices.Equal(orgAIDs, []string{"dep-org-a-owner", "dep-org-a-peer"}) {
		t.Fatalf("org-a list returned %v, want only org-a deployments", orgAIDs)
	}
	if got := listOrgA.Msg.Pagination.GetTotal(); got != 2 {
		t.Fatalf("org-a total = %d, want 2", got)
	}

	listOrgB, err := service.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{
		OrganizationId: "org-b",
	}))
	if err != nil {
		t.Fatalf("list org-b deployments: %v", err)
	}
	if got := deploymentIDs(listOrgB.Msg.Deployments); len(got) != 0 {
		t.Fatalf("org-b list returned %v for org-a user, want none", got)
	}

	_, err = service.UpdateDeployment(ctx, connect.NewRequest(&deploymentsv1.UpdateDeploymentRequest{
		DeploymentId: "dep-org-b-owner",
		Name:         proto.String("cross-org-edit"),
	}))
	if err == nil {
		t.Fatal("cross-org update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgBDeployment database.Deployment
	if err := db.First(&orgBDeployment, "id = ?", "dep-org-b-owner").Error; err != nil {
		t.Fatalf("fetch org-b deployment: %v", err)
	}
	if orgBDeployment.Name != "Org B Deployment" {
		t.Fatalf("cross-org update changed org-b name to %q", orgBDeployment.Name)
	}

	updateOrgA, err := service.UpdateDeployment(ctx, connect.NewRequest(&deploymentsv1.UpdateDeploymentRequest{
		DeploymentId: "dep-org-a-owner",
		Name:         proto.String("Org A Updated"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.Deployment.GetName(); got != "Org A Updated" {
		t.Fatalf("same-org update name = %q, want updated name", got)
	}

	_, err = service.DeleteDeployment(ctx, connect.NewRequest(&deploymentsv1.DeleteDeploymentRequest{
		OrganizationId: "org-a",
		DeploymentId:   "dep-org-a-owner",
	}))
	if err != nil {
		t.Fatalf("same-org delete: %v", err)
	}

	var deletedDeployment database.Deployment
	if err := db.First(&deletedDeployment, "id = ?", "dep-org-a-owner").Error; err != nil {
		t.Fatalf("fetch deleted deployment: %v", err)
	}
	if deletedDeployment.DeletedAt == nil {
		t.Fatal("same-org delete left deployment deleted_at nil")
	}

	listAfterDelete, err := service.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a deployments after delete: %v", err)
	}
	if got := deploymentIDs(listAfterDelete.Msg.Deployments); !slices.Equal(got, []string{"dep-org-a-peer"}) {
		t.Fatalf("org-a list after delete returned %v, want remaining deployment only", got)
	}
}

func newDeploymentServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.Deployment{},
		&database.BuildHistory{},
		&database.Organization{},
		&database.OrganizationMember{},
		&database.OrgRole{},
		&database.OrgRoleBinding{},
	); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := database.DB
	previousMetricsDB := database.MetricsDB
	database.DB = db
	database.MetricsDB = db
	t.Cleanup(func() {
		database.DB = previousDB
		database.MetricsDB = previousMetricsDB
	})

	return db
}

func seedDeploymentServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	autoDeploy := false
	deployments := []*database.Deployment{
		testDeployment("dep-org-a-owner", "Org A Deployment", "org-a", "user-org-a", now, &autoDeploy),
		testDeployment("dep-org-a-peer", "Org A Peer Deployment", "org-a", "user-org-a-peer", now, &autoDeploy),
		testDeployment("dep-org-b-owner", "Org B Deployment", "org-b", "user-org-b", now, &autoDeploy),
	}

	records := []any{
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-a-peer", OrganizationID: "org-a", UserID: "user-org-a-peer", Role: auth.SystemRoleIDMember, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
	for _, deployment := range deployments {
		if err := db.Create(deployment).Error; err != nil {
			t.Fatalf("seed deployment %s: %v", deployment.ID, err)
		}
	}
}

func testDeployment(id, name, orgID, createdBy string, now time.Time, autoDeploy *bool) *database.Deployment {
	return &database.Deployment{
		ID:             id,
		Name:           name,
		Domain:         id + ".my.obiente.cloud",
		CustomDomains:  "[]",
		Type:           int32(deploymentsv1.DeploymentType_NODE),
		BuildStrategy:  int32(deploymentsv1.BuildStrategy_RAILPACK),
		Branch:         "main",
		AutoDeploy:     autoDeploy,
		Status:         int32(deploymentsv1.DeploymentStatus_STOPPED),
		HealthStatus:   "unknown",
		Environment:    int32(deploymentsv1.Environment_PRODUCTION),
		Groups:         "[]",
		BuildTime:      1,
		Size:           "--",
		EnvVars:        "{}",
		CreatedAt:      now,
		LastDeployedAt: now,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}
}

func deploymentIDs(deployments []*deploymentsv1.Deployment) []string {
	ids := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		ids = append(ids, deployment.GetId())
	}
	slices.Sort(ids)
	return ids
}
