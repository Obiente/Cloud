package gameservers

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGameServerServiceTenantIsolation(t *testing.T) {
	db := newGameServerServiceTestDB(t)
	service := NewService(context.Background(), database.NewGameServerRepository(db, nil), nil)

	seedGameServerServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	listOrgA, err := service.ListGameServers(ctx, connect.NewRequest(&gameserversv1.ListGameServersRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a game servers: %v", err)
	}
	orgAIDs := gameServerIDs(listOrgA.Msg.GameServers)
	if !slices.Equal(orgAIDs, []string{"gs-org-a-owner", "gs-org-a-peer"}) {
		t.Fatalf("org-a list returned %v, want only org-a game servers", orgAIDs)
	}

	listOrgB, err := service.ListGameServers(ctx, connect.NewRequest(&gameserversv1.ListGameServersRequest{
		OrganizationId: "org-b",
	}))
	if err != nil {
		t.Fatalf("list org-b game servers: %v", err)
	}
	if got := gameServerIDs(listOrgB.Msg.GameServers); len(got) != 0 {
		t.Fatalf("org-b list returned %v for org-a user, want none", got)
	}

	_, err = service.UpdateGameServer(ctx, connect.NewRequest(&gameserversv1.UpdateGameServerRequest{
		GameServerId: "gs-org-b-owner",
		Name:         proto.String("cross-org-edit"),
	}))
	if err == nil {
		t.Fatal("cross-org update succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-org update code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}

	var orgBGameServer database.GameServer
	if err := db.First(&orgBGameServer, "id = ?", "gs-org-b-owner").Error; err != nil {
		t.Fatalf("fetch org-b game server: %v", err)
	}
	if orgBGameServer.Name != "Org B Game Server" {
		t.Fatalf("cross-org update changed org-b game server name to %q", orgBGameServer.Name)
	}

	updateOrgA, err := service.UpdateGameServer(ctx, connect.NewRequest(&gameserversv1.UpdateGameServerRequest{
		GameServerId: "gs-org-a-owner",
		Name:         proto.String("Org A Game Server Updated"),
	}))
	if err != nil {
		t.Fatalf("same-org update: %v", err)
	}
	if got := updateOrgA.Msg.GameServer.GetName(); got != "Org A Game Server Updated" {
		t.Fatalf("same-org update name = %q, want updated name", got)
	}

	_, err = service.DeleteGameServer(ctx, connect.NewRequest(&gameserversv1.DeleteGameServerRequest{
		GameServerId: "gs-org-a-owner",
	}))
	if err != nil {
		t.Fatalf("same-org delete: %v", err)
	}

	var deletedGameServer database.GameServer
	if err := db.First(&deletedGameServer, "id = ?", "gs-org-a-owner").Error; err != nil {
		t.Fatalf("fetch deleted game server: %v", err)
	}
	if deletedGameServer.DeletedAt == nil {
		t.Fatal("same-org delete left game server deleted_at nil")
	}

	listAfterDelete, err := service.ListGameServers(ctx, connect.NewRequest(&gameserversv1.ListGameServersRequest{
		OrganizationId: "org-a",
	}))
	if err != nil {
		t.Fatalf("list org-a game servers after delete: %v", err)
	}
	if got := gameServerIDs(listAfterDelete.Msg.GameServers); !slices.Equal(got, []string{"gs-org-a-peer"}) {
		t.Fatalf("org-a list after delete returned %v, want remaining game server only", got)
	}
}

func newGameServerServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&database.GameServer{},
		&database.GameServerLocation{},
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

func seedGameServerServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	records := []any{
		&database.Organization{ID: "org-a", Name: "Org A", Slug: "org-a", Status: "active", CreatedAt: now},
		&database.Organization{ID: "org-b", Name: "Org B", Slug: "org-b", Status: "active", CreatedAt: now},
		&database.OrganizationMember{ID: "member-org-a", OrganizationID: "org-a", UserID: "user-org-a", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-a-peer", OrganizationID: "org-a", UserID: "user-org-a-peer", Role: auth.SystemRoleIDMember, Status: "active", JoinedAt: now},
		&database.OrganizationMember{ID: "member-org-b", OrganizationID: "org-b", UserID: "user-org-b", Role: auth.SystemRoleIDOwner, Status: "active", JoinedAt: now},
		testGameServer("gs-org-a-owner", "Org A Game Server", "org-a", "user-org-a", now),
		testGameServer("gs-org-a-peer", "Org A Peer Game Server", "org-a", "user-org-a-peer", now),
		testGameServer("gs-org-b-owner", "Org B Game Server", "org-b", "user-org-b", now),
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed %T: %v", record, err)
		}
	}
}

func testGameServer(id, name, orgID, createdBy string, now time.Time) *database.GameServer {
	return &database.GameServer{
		ID:             id,
		Name:           name,
		GameType:       int32(gameserversv1.GameType_MINECRAFT),
		Status:         int32(gameserversv1.GameServerStatus_STOPPED),
		MemoryBytes:    1024 * 1024 * 1024,
		CPUCores:       1,
		Port:           25565,
		ExtraPorts:     "[]",
		DockerImage:    "itzg/minecraft-server:latest",
		EnvVars:        "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}
}

func gameServerIDs(gameServers []*gameserversv1.GameServer) []string {
	ids := make([]string, 0, len(gameServers))
	for _, gameServer := range gameServers {
		ids = append(ids, gameServer.GetId())
	}
	slices.Sort(ids)
	return ids
}
