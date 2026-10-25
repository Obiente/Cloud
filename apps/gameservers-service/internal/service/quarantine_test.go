package gameservers

import (
	"connectrpc.com/connect"
	"context"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"testing"
	"time"
)

func TestSuspendedGameServerOwnerCannotMutate(t *testing.T) {
	db := newGameServerServiceTestDB(t)
	seedGameServerServiceIsolationData(t, db)
	svc := NewService(context.Background(), database.NewGameServerRepository(db, nil), nil)
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Email: "user-org-a@example.com"})
	r := database.GameServerSuspension{ID: "test-suspension", GameServerID: "gs-org-a-owner", OrganizationID: "org-a", SuspendedBy: "test-actor", SuspendedAt: time.Now()}
	if err := db.Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{"start", "restart", "update", "delete"} {
		t.Run(permission, func(t *testing.T) {
			err := svc.checkGameServerPermission(ctx, "gs-org-a-owner", permission)
			if connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("mutation allowed: %v", err)
			}
		})
	}
	for _, permission := range []string{"read", "stop"} {
		if err := svc.checkGameServerPermission(ctx, "gs-org-a-owner", permission); err != nil {
			t.Fatalf("safe access blocked: %s: %v", permission, err)
		}
	}
}
