package deployments

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func TestModerationListAndDetail(t *testing.T) {
	for _, status := range []string{"banned", "suspended"} {
		for _, role := range []string{"", auth.RoleAdmin, auth.RoleSuperAdmin} {
			t.Run(status+"/"+role, func(t *testing.T) {
				db := newDeploymentServiceTestDB(t)
				seedDeploymentServiceIsolationData(t, db)
				if err := db.Model(&database.Organization{}).Where("id = ?", "org-a").Update("status", status).Error; err != nil {
					t.Fatal(err)
				}
				svc := NewService(context.Background(), database.NewDeploymentRepository(db, nil), nil, nil)
				ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Roles: []string{role}})
				list, err := svc.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{OrganizationId: "org-a"}))
				allowed := role != ""
				if allowed {
					if err != nil || len(list.Msg.Deployments) == 0 {
						t.Fatalf("admin list: %v, %v", list, err)
					}
				} else if connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("restricted member list allowed: %v", err)
				}
				detail, err := svc.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{DeploymentId: "dep-org-a-owner"}))
				if allowed {
					if err != nil || detail.Msg.Deployment == nil {
						t.Fatalf("admin details: %v, %v", detail, err)
					}
				} else if connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("restricted member detail allowed: %v", err)
				}
				if err := db.Create(&database.UserBan{ID: "current-ban", UserID: "user-org-a", Type: "banned", BannedAt: time.Now()}).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := svc.ListDeployments(ctx, connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{OrganizationId: "org-a"})); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("banned account list allowed: %v", err)
				}
				if _, err := svc.GetDeployment(ctx, connect.NewRequest(&deploymentsv1.GetDeploymentRequest{DeploymentId: "dep-org-a-owner"})); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("banned account details allowed: %v", err)
				}
			})
		}
	}
}
