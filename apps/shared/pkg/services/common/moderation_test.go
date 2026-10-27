package common

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModerationOrganizationHelpers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous; raw.Close() })
	if err := db.AutoMigrate(&database.Organization{}, &database.OrganizationMember{}, &database.UserBan{}, &database.SuperadminRoleBinding{}, &database.SuperadminRole{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&database.Organization{ID: "org", Slug: "org", Status: "banned"})
	db.Create(&database.OrganizationMember{ID: "member", OrganizationID: "org", UserID: "user", Role: auth.SystemRoleIDOwner, Status: "active"})
	for _, role := range []string{"", auth.RoleAdmin, auth.RoleSuperAdmin} {
		user := &authv1.User{Id: "user", Roles: []string{role}}
		ctx := auth.WithUser(context.Background(), user)
		if err := VerifyOrgAccess(ctx, "org", user); (err == nil) != (role != "") {
			t.Fatalf("membership shortcut %s: %v", role, err)
		}
		if err := AuthorizeOrgReadRoles(ctx, "org", user, "owner"); (err == nil) != (role != "") {
			t.Fatalf("read helper %s: %v", role, err)
		}
		if err := AuthorizeOrgRoles(ctx, "org", user, "owner"); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("mutation allowed: %v", err)
		}
	}
	db.Model(&database.Organization{}).Where("id = ?", "org").Update("status", "active")
	db.Model(&database.OrganizationMember{}).Where("id = ?", "member").Update("status", "inactive")
	user := &authv1.User{Id: "user"}
	ctx := auth.WithUser(context.Background(), user)
	if err := VerifyOrgAccess(ctx, "org", user); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("inactive member: %v", err)
	}
	db.Create(&database.UserBan{ID: "ban", UserID: "user", Type: "banned", BannedAt: time.Now()})
	if _, err := EnsureAuthenticated(ctx, connect.NewRequest(&authv1.GetPublicConfigRequest{})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("existing context ignored ban: %v", err)
	}
}
