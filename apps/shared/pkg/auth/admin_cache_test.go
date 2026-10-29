package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
)

type testUserInfoStore map[string]string

func (s testUserInfoStore) Get(_ context.Context, key string) (string, error) {
	if value, ok := s[key]; ok {
		return value, nil
	}
	return "", errors.New("cache miss")
}

func (s testUserInfoStore) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s[key] = string(encoded)
	return nil
}

func TestCachedAdminDemotionRemovesBothPlatformRoles(t *testing.T) {
	for _, scenario := range []string{"removed", "empty_config", "missing_email", "legacy_admin_only"} {
		t.Run(scenario, func(t *testing.T) {
			store := testUserInfoStore{}
			cache := &UserInfoCache{redisCache: store}
			user := &authv1.User{Id: "user-synthetic", Email: "admin@example.test", Roles: []string{RoleAdmin, RoleSuperAdmin, RoleOwner}}
			config := &AuthConfig{UserInfoCache: cache, SuperAdmins: map[string]struct{}{"someone-else@example.test": {}}}
			switch scenario {
			case "empty_config":
				config.SuperAdmins = nil
			case "missing_email":
				user.Email = ""
			case "legacy_admin_only":
				user.Roles = []string{RoleAdmin, RoleOwner}
			}
			cache.Set(context.Background(), "synthetic-token", user)
			resolved, err := config.resolveToken(context.Background(), "synthetic-token")
			if err != nil {
				t.Fatal(err)
			}
			if HasRole(resolved, RoleAdmin) || HasRole(resolved, RoleSuperAdmin) {
				t.Fatalf("retained platform roles: %v", resolved.Roles)
			}
			if !HasRole(resolved, RoleOwner) {
				t.Fatal("unrelated tenant role was removed")
			}
			stored, found := cache.Get(context.Background(), "synthetic-token")
			if !found || HasRole(stored, RoleAdmin) || HasRole(stored, RoleSuperAdmin) {
				t.Fatal("elevated cache entry was not replaced")
			}
		})
	}
}

func TestConfiguredAdminRolesMatchOnFreshAndCachedIdentity(t *testing.T) {
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sub":"user-synthetic","email":"Admin@Example.Test","email_verified":true}`))
	}))
	defer identity.Close()
	cache := &UserInfoCache{redisCache: testUserInfoStore{}}
	config := &AuthConfig{UserInfoCache: cache, SuperAdmins: map[string]struct{}{"admin@example.test": {}}, UserInfoURL: identity.URL, HTTPClient: identity.Client()}
	for range 2 {
		user, err := config.resolveToken(context.Background(), "synthetic-token")
		if err != nil {
			t.Fatal(err)
		}
		if !HasRole(user, RoleSuperAdmin) || !HasRole(user, RoleAdmin) {
			t.Fatalf("configured admin lacks roles: %v", user.Roles)
		}
	}
	config.SuperAdmins = map[string]struct{}{}
	user, err := config.resolveToken(context.Background(), "synthetic-token")
	if err != nil {
		t.Fatal(err)
	}
	if HasRole(user, RoleAdmin) || HasRole(user, RoleSuperAdmin) {
		t.Fatal("configuration removal retained cached authority")
	}
	config.SuperAdmins["admin@example.test"] = struct{}{}
	user, err = config.resolveToken(context.Background(), "synthetic-token")
	if err != nil {
		t.Fatal(err)
	}
	if !HasRole(user, RoleSuperAdmin) || !HasRole(user, RoleAdmin) {
		t.Fatal("configuration addition did not apply to cached identity")
	}
}
