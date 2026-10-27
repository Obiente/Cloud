package organizations

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	organizationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/organizations/v1"
)

func TestModerationOrganizationDiscovery(t *testing.T) {
	db := newOrganizationServiceTestDB(t)
	seedOrganizationServiceIsolationData(t, db)
	db.Model(&database.Organization{}).Where("id = ?", "org-a").Update("status", "banned")
	svc := NewService(Config{}).(*Service)
	for _, role := range []string{"", auth.RoleAdmin, auth.RoleSuperAdmin} {
		ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Roles: []string{role}})
		response, err := svc.ListOrganizations(ctx, connect.NewRequest(&organizationsv1.ListOrganizationsRequest{OnlyMine: protoBool(true)}))
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(organizationIDs(response.Msg.Organizations), "org-a") != (role != "") {
			t.Fatalf("restricted org visible to %q: %v", role, response.Msg.Organizations)
		}
	}
	db.Create(&database.UserBan{ID: "ban", UserID: "user-org-a", Type: "banned", BannedAt: time.Now()})
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Roles: []string{auth.RoleSuperAdmin}})
	if _, err := svc.ListOrganizations(ctx, connect.NewRequest(&organizationsv1.ListOrganizationsRequest{})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("banned admin could discover orgs: %v", err)
	}
}

func protoBool(value bool) *bool { return &value }
