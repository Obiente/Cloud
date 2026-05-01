package databases

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseServiceTenantIsolation(t *testing.T) {
	db := newDatabaseServiceTestDB(t)
	service := &Service{
		permissionChecker: auth.NewPermissionChecker(),
		repo:              database.NewDatabaseRepository(db, nil),
	}

	seedDatabaseServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	listOrgA, err := service.ListDatabases(ctx, connect.NewRequest(&databasesv1.ListDatabasesRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a databases: %v", err)
	}
	orgAIDs := databaseIDs(listOrgA.Msg.Databases)
	if !slices.Equal(orgAIDs, []string{"db-org-a-owner", "db-org-a-peer"}) {
		t.Fatalf("org-a list returned %v, want only org-a databases", orgAIDs)
	}
	if got := listOrgA.Msg.Pagination.GetTotal(); got != 2 {
		t.Fatalf("org-a total = %d, want 2", got)
	}

	_, err = service.ListDatabases(ctx, connect.NewRequest(&databasesv1.ListDatabasesRequest{
		OrganizationId: "org-b",
	}))
	if err == nil {
		t.Fatal("cross-org list succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org list code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	_, err = service.UpdateDatabase(ctx, connect.NewRequest(&databasesv1.UpdateDatabaseRequest{
		OrganizationId: "org-a",
		DatabaseId:     "db-org-b-owner",
		Name:           proto.String("cross-org-edit"),
	}))
	if err == nil {
		t.Fatal("cross-org update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgBDatabase database.DatabaseInstance
	if err := db.First(&orgBDatabase, "id = ?", "db-org-b-owner").Error; err != nil {
		t.Fatalf("fetch org-b database: %v", err)
	}
	if orgBDatabase.Name != "Org B Database" {
		t.Fatalf("cross-org update changed org-b database name to %q", orgBDatabase.Name)
	}

	updateOrgA, err := service.UpdateDatabase(ctx, connect.NewRequest(&databasesv1.UpdateDatabaseRequest{
		OrganizationId: "org-a",
		DatabaseId:     "db-org-a-owner",
		Name:           proto.String("Org A Database Updated"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.Database.GetName(); got != "Org A Database Updated" {
		t.Fatalf("same-org update name = %q, want updated name", got)
	}

	_, err = service.DeleteDatabase(ctx, connect.NewRequest(&databasesv1.DeleteDatabaseRequest{
		OrganizationId: "org-a",
		DatabaseId:     "db-org-a-owner",
	}))
	if err != nil {
		t.Fatalf("same-org delete: %v", err)
	}
	waitForDatabaseDeleted(t, db, "db-org-a-owner")

	listAfterDelete, err := service.ListDatabases(ctx, connect.NewRequest(&databasesv1.ListDatabasesRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a databases after delete: %v", err)
	}
	if got := databaseIDs(listAfterDelete.Msg.Databases); !slices.Equal(got, []string{"db-org-a-peer"}) {
		t.Fatalf("org-a list after delete returned %v, want remaining database only", got)
	}
}

func newDatabaseServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.DatabaseInstance{},
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

func seedDatabaseServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	records := []any{
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-a-peer", OrganizationID: "org-a", UserID: "user-org-a-peer", Role: auth.SystemRoleIDMember, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		testDatabase("db-org-a-owner", "Org A Database", "org-a", "user-org-a", now),
		testDatabase("db-org-a-peer", "Org A Peer Database", "org-a", "user-org-a-peer", now),
		testDatabase("db-org-b-owner", "Org B Database", "org-b", "user-org-b", now),
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func testDatabase(id, name, orgID, createdBy string, now time.Time) *database.DatabaseInstance {
	version := "16"
	host := id + ".db.test"
	port := int32(5432)
	return &database.DatabaseInstance{
		ID:             id,
		Name:           name,
		Status:         int32(databasesv1.DatabaseStatus_RUNNING),
		Type:           int32(databasesv1.DatabaseType_POSTGRESQL),
		Version:        &version,
		Size:           "test-size",
		CPUCores:       1,
		MemoryBytes:    1024 * 1024 * 1024,
		DiskBytes:      10 * 1024 * 1024 * 1024,
		MaxConnections: 100,
		Host:           &host,
		Port:           &port,
		Metadata:       "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}
}

func databaseIDs(databases []*databasesv1.DatabaseInstance) []string {
	ids := make([]string, 0, len(databases))
	for _, database := range databases {
		ids = append(ids, database.GetId())
	}
	slices.Sort(ids)
	return ids
}

func waitForDatabaseDeleted(t *testing.T, db *gorm.DB, databaseID string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	var dbInstance database.DatabaseInstance
	for {
		if err := db.First(&dbInstance, "id = ?", databaseID).Error; err != nil {
			t.Fatalf("fetch deleted database: %v", err)
		}
		if dbInstance.DeletedAt != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("same-org delete left database deleted_at nil")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
