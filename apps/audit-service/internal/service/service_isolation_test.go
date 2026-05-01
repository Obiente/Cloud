package audit

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	auditv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/audit/v1"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditServiceTenantIsolation(t *testing.T) {
	db := newAuditServiceTestDB(t)
	service := NewService(db)
	seedAuditServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a-admin",
		Email: "user-org-a-admin@example.com",
	})

	listOrgA, err := service.ListAuditLogs(ctx, connect.NewRequest(&auditv1.ListAuditLogsRequest{
		OrganizationId: proto.String("org-a"),
		PageSize:       proto.Int32(10),
	}))
	if err != nil {
		t.Fatalf("list org-a audit logs: %v", err)
	}
	if got := auditLogIDs(listOrgA.Msg.AuditLogs); !slices.Equal(got, []string{"audit-org-a"}) {
		t.Fatalf("org-a audit list returned %v, want only org-a audit log", got)
	}
	if got := listOrgA.Msg.TotalCount; got != 1 {
		t.Fatalf("org-a total = %d, want 1", got)
	}

	_, err = service.ListAuditLogs(ctx, connect.NewRequest(&auditv1.ListAuditLogsRequest{
		OrganizationId: proto.String("org-b"),
	}))
	if err == nil {
		t.Fatal("cross-org audit list succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org audit list code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	_, err = service.GetAuditLog(ctx, connect.NewRequest(&auditv1.GetAuditLogRequest{
		AuditLogId: "audit-org-b",
	}))
	if err == nil {
		t.Fatal("cross-org audit get succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org audit get code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}
}

func newAuditServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.AuditLog{},
		&database.Organization{},
		&database.OrganizationMember{},
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

func seedAuditServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	orgA := "org-a"
	orgB := "org-b"
	records := []any{
		&database.Organization{ID: orgA, Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: orgB, Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a-admin", OrganizationID: orgA, UserID: "user-org-a-admin", Role: auth.SystemRoleIDAdmin, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b-admin", OrganizationID: orgB, UserID: "user-org-b-admin", Role: auth.SystemRoleIDAdmin, Status: "active", JoinedAt: now},
		testAuditLog("audit-org-a", "user-org-a-admin", &orgA, now),
		testAuditLog("audit-org-b", "user-org-b-admin", &orgB, now),
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func testAuditLog(id, userID string, orgID *string, now time.Time) *database.AuditLog {
	resourceType := "deployment"
	resourceID := id + "-resource"
	return &database.AuditLog{
		ID:             id,
		UserID:         userID,
		OrganizationID: orgID,
		Action:         "UpdateDeployment",
		Service:        "DeploymentService",
		ResourceType:   &resourceType,
		ResourceID:     &resourceID,
		IPAddress:      "127.0.0.1",
		UserAgent:      "test",
		RequestData:    "{}",
		ResponseStatus: 200,
		DurationMs:     1,
		CreatedAt:      now,
	}
}

func auditLogIDs(logs []*auditv1.AuditLogEntry) []string {
	ids := make([]string, 0, len(logs))
	for _, log := range logs {
		ids = append(ids, log.GetId())
	}
	slices.Sort(ids)
	return ids
}
