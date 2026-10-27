package gameservers

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
)

func TestModerationListAndDetail(t *testing.T) {
	for _, status := range []string{"banned", "suspended"} {
		for _, role := range []string{"", auth.RoleAdmin, auth.RoleSuperAdmin} {
			t.Run(status+"/"+role, func(t *testing.T) {
				db := newGameServerServiceTestDB(t)
				seedGameServerServiceIsolationData(t, db)
				if err := db.Model(&database.Organization{}).Where("id = ?", "org-a").Update("status", status).Error; err != nil {
					t.Fatal(err)
				}
				svc := NewService(context.Background(), database.NewGameServerRepository(db, nil), nil)
				ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Roles: []string{role}})
				list, err := svc.ListGameServers(ctx, connect.NewRequest(&gameserversv1.ListGameServersRequest{OrganizationId: "org-a"}))
				allowed := role != ""
				if allowed {
					if err != nil || len(list.Msg.GameServers) == 0 {
						t.Fatalf("admin list: %v, %v", list, err)
					}
				} else if connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("restricted member list allowed: %v", err)
				}
				detail, err := svc.GetGameServer(ctx, connect.NewRequest(&gameserversv1.GetGameServerRequest{GameServerId: "gs-org-a-owner"}))
				if allowed {
					if err != nil || detail.Msg.GameServer == nil {
						t.Fatalf("admin details: %v, %v", detail, err)
					}
				} else if connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("restricted member detail allowed: %v", err)
				}
				if err := db.Create(&database.UserBan{ID: "current-ban", UserID: "user-org-a", Type: "banned", BannedAt: time.Now()}).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := svc.ListGameServers(ctx, connect.NewRequest(&gameserversv1.ListGameServersRequest{OrganizationId: "org-a"})); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("banned account list allowed: %v", err)
				}
				if _, err := svc.GetGameServer(ctx, connect.NewRequest(&gameserversv1.GetGameServerRequest{GameServerId: "gs-org-a-owner"})); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("banned account details allowed: %v", err)
				}
			})
		}
	}
}
