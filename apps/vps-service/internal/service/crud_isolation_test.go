package vps

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVPSServiceTenantIsolation(t *testing.T) {
	db := newVPSServiceTestDB(t)
	service := &Service{
		permissionChecker: auth.NewPermissionChecker(),
	}

	seedVPSServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	listOrgA, err := service.ListVPS(ctx, connect.NewRequest(&vpsv1.ListVPSRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a VPS instances: %v", err)
	}
	orgAIDs := vpsIDs(listOrgA.Msg.VpsInstances)
	if !slices.Equal(orgAIDs, []string{"vps-org-a-owner", "vps-org-a-peer"}) {
		t.Fatalf("org-a list returned %v, want only org-a VPS instances", orgAIDs)
	}
	if got := listOrgA.Msg.Pagination.GetTotal(); got != 2 {
		t.Fatalf("org-a total = %d, want 2", got)
	}

	_, err = service.ListVPS(ctx, connect.NewRequest(&vpsv1.ListVPSRequest{
		OrganizationId: "org-b",
	}))
	if err == nil {
		t.Fatal("cross-org list succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org list code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	_, err = service.UpdateVPS(ctx, connect.NewRequest(&vpsv1.UpdateVPSRequest{
		VpsId: "vps-org-b-owner",
		Name:  proto.String("cross-org-edit"),
	}))
	if err == nil {
		t.Fatal("cross-org update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgBVPS database.VPSInstance
	if err := db.First(&orgBVPS, "id = ?", "vps-org-b-owner").Error; err != nil {
		t.Fatalf("fetch org-b VPS: %v", err)
	}
	if orgBVPS.Name != "Org B VPS" {
		t.Fatalf("cross-org update changed org-b VPS name to %q", orgBVPS.Name)
	}

	updateOrgA, err := service.UpdateVPS(ctx, connect.NewRequest(&vpsv1.UpdateVPSRequest{
		VpsId: "vps-org-a-owner",
		Name:  proto.String("Org A VPS Updated"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.Vps.GetName(); got != "Org A VPS Updated" {
		t.Fatalf("same-org update name = %q, want updated name", got)
	}

	_, err = service.DeleteVPS(ctx, connect.NewRequest(&vpsv1.DeleteVPSRequest{
		OrganizationId: "org-a",
		VpsId:          "vps-org-a-owner",
	}))
	if err != nil {
		t.Fatalf("same-org delete: %v", err)
	}

	var deletedVPS database.VPSInstance
	if err := db.First(&deletedVPS, "id = ?", "vps-org-a-owner").Error; err != nil {
		t.Fatalf("fetch deleted VPS: %v", err)
	}
	if deletedVPS.DeletedAt == nil {
		t.Fatal("same-org delete left VPS deleted_at nil")
	}
	if deletedVPS.IPv4Addresses != "[]" || deletedVPS.IPv6Addresses != "[]" {
		t.Fatalf("same-org delete left stale IPs: ipv4=%q ipv6=%q", deletedVPS.IPv4Addresses, deletedVPS.IPv6Addresses)
	}

	listAfterDelete, err := service.ListVPS(ctx, connect.NewRequest(&vpsv1.ListVPSRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a VPS instances after delete: %v", err)
	}
	if got := vpsIDs(listAfterDelete.Msg.VpsInstances); !slices.Equal(got, []string{"vps-org-a-peer"}) {
		t.Fatalf("org-a list after delete returned %v, want remaining VPS only", got)
	}
}

func newVPSServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.VPSInstance{},
		&database.Notification{},
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

func seedVPSServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	records := []any{
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-a-peer", OrganizationID: "org-a", UserID: "user-org-a-peer", Role: auth.SystemRoleIDMember, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		testVPS("vps-org-a-owner", "Org A VPS", "org-a", "user-org-a", now),
		testVPS("vps-org-a-peer", "Org A Peer VPS", "org-a", "user-org-a-peer", now),
		testVPS("vps-org-b-owner", "Org B VPS", "org-b", "user-org-b", now),
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func testVPS(id, name, orgID, createdBy string, now time.Time) *database.VPSInstance {
	return &database.VPSInstance{
		ID:             id,
		Name:           name,
		Status:         int32(vpsv1.VPSStatus_STOPPED),
		Region:         "test-region",
		Image:          int32(vpsv1.VPSImage_UBUNTU_24_04),
		Size:           "test-size",
		CPUCores:       1,
		MemoryBytes:    1024 * 1024 * 1024,
		DiskBytes:      20 * 1024 * 1024 * 1024,
		IPv4Addresses:  "[]",
		IPv6Addresses:  "[]",
		Metadata:       "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}
}

func vpsIDs(instances []*vpsv1.VPSInstance) []string {
	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.GetId())
	}
	slices.Sort(ids)
	return ids
}
