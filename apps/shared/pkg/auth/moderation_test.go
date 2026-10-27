package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func moderationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&database.Organization{}, &database.OrganizationMember{}, &database.UserBan{}, &database.OrgRole{}, &database.OrgRoleBinding{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous; raw.Close() })
	return db
}

func TestModerationBeforePermissionShortcuts(t *testing.T) {
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	for _, tc := range []struct {
		name, orgStatus, memberStatus, ban string
		orgExpiry, banExpiry, revoked      *time.Time
		allow                              bool
	}{
		{"active_owner", "active", "active", "", nil, nil, nil, true},
		{"active_member_with_permission", "active", "active", "", nil, nil, nil, true},
		{"active_member_without_permission", "active", "active", "", nil, nil, nil, false},
		{"suspended_org", "suspended", "active", "", nil, nil, nil, false},
		{"expired_org_suspension", "suspended", "active", "", &past, nil, nil, true},
		{"banned_org", "banned", "active", "", &past, nil, nil, false},
		{"banned_user", "active", "active", "banned", nil, &past, nil, false},
		{"suspended_user", "active", "active", "suspended", nil, &future, nil, false},
		{"expired_user_suspension", "active", "active", "suspended", nil, &past, nil, true},
		{"revoked_ban", "active", "active", "banned", nil, nil, &past, true},
		{"inactive_owner_with_binding", "active", "inactive", "", nil, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moderationDB(t)
			role := SystemRoleIDOwner
			if strings.HasPrefix(tc.name, "active_member") {
				role = "none"
			}
			permissions := `["deployment.create"]`
			if tc.name == "active_member_without_permission" {
				permissions = `[]`
			}
			for _, row := range []interface{}{
				&database.Organization{ID: "org-test", Slug: "org-test", Status: tc.orgStatus, SuspensionExpires: tc.orgExpiry},
				&database.OrganizationMember{ID: "member-test", OrganizationID: "org-test", UserID: "user-test", Role: role, Status: tc.memberStatus},
				&database.OrgRole{ID: "role-test", OrganizationID: "org-test", Name: "Limited test role", Permissions: permissions},
				&database.OrgRoleBinding{ID: "binding-test", OrganizationID: "org-test", UserID: "user-test", RoleID: "role-test"},
			} {
				if err := db.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tc.ban != "" {
				if err := db.Create(&database.UserBan{ID: "ban-test", UserID: "user-test", Type: tc.ban, BannedBy: "admin-test", BannedAt: time.Now(), ExpiresAt: tc.banExpiry, RevokedAt: tc.revoked}).Error; err != nil {
					t.Fatal(err)
				}
			}
			ctx := WithUser(context.Background(), &authv1.User{Id: "user-test"})
			err := NewPermissionChecker().CheckScopedPermission(ctx, "org-test", ScopedPermission{Permission: PermissionDeploymentCreate, ResourceType: "deployment"})
			if (err == nil) != tc.allow {
				t.Fatalf("allow=%v, error=%v", tc.allow, err)
			}
		})
	}
}

func TestTokenValidationConsultsCurrentModeration(t *testing.T) {
	db := moderationDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sub":"user-test","email":"user@example.test"}`))
	}))
	defer server.Close()
	cfg := &AuthConfig{UserInfoURL: server.URL, HTTPClient: server.Client(), UserInfoCache: &UserInfoCache{}}
	if _, err := cfg.validateToken(context.Background(), "synthetic-token"); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.UserBan{ID: "ban-test", UserID: "user-test", Type: "banned", BannedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.validateToken(context.Background(), "synthetic-token"); err == nil {
		t.Fatal("current ban was ignored")
	}
	raw, _ := db.DB()
	raw.Close()
	if _, err := cfg.validateToken(context.Background(), "synthetic-token"); err == nil {
		t.Fatal("database failure granted access")
	}
}

func TestOwnerCannotUseAnotherOrganizationsResourceID(t *testing.T) {
	db := moderationDB(t)
	if err := db.AutoMigrate(&database.Deployment{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []interface{}{
		&database.Organization{ID: "org-test", Slug: "org-test", Status: "active"},
		&database.OrganizationMember{ID: "member-test", OrganizationID: "org-test", UserID: "user-test", Role: SystemRoleIDOwner, Status: "active"},
		&database.Deployment{ID: "deployment-other", OrganizationID: "org-other"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := WithUser(context.Background(), &authv1.User{Id: "user-test"})
	if err := NewPermissionChecker().CheckScopedPermission(ctx, "org-test", ScopedPermission{Permission: PermissionDeploymentStart, ResourceType: "deployment", ResourceID: "deployment-other"}); err == nil {
		t.Fatal("owner crossed organization boundary")
	}
}
