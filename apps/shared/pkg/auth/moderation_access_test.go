package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
	"google.golang.org/protobuf/proto"
)

func TestModerationSeparatesPlatformInspectionAndTenantAccess(t *testing.T) {
	for _, role := range []string{"", RoleAdmin, RoleSuperAdmin} {
		for _, status := range []string{"banned", "suspended"} {
			t.Run(role+"/"+status, func(t *testing.T) {
				db := moderationDB(t)
				db.Create(&database.Organization{ID: "restricted-org", Slug: "restricted-org", Status: status})
				ctx := WithUser(context.Background(), &authv1.User{Id: "test-user", Roles: []string{role}})
				for _, permission := range []string{PermissionDeploymentRead, PermissionDeploymentLogs, PermissionGameServersRead, PermissionDatabaseRead, PermissionVPSRead, PermissionOrganizationRead} {
					err := CheckOrganizationAccess(ctx, "restricted-org", permission)
					if (err == nil) != (role != "") {
						t.Fatalf("%s: %v", permission, err)
					}
				}
				for _, permission := range []string{PermissionDeploymentStart, PermissionGameServersUpdate, PermissionDatabaseCreate, PermissionVPSDelete} {
					if err := CheckOrganizationAccess(ctx, "restricted-org", permission); connect.CodeOf(err) != connect.CodePermissionDenied {
						t.Fatalf("mutation allowed: %s, %v", permission, err)
					}
				}
				db.Create(&database.UserBan{ID: "ban", UserID: "test-user", Type: "banned", BannedAt: time.Now()})
				if err := CheckOrganizationAccess(ctx, "restricted-org", PermissionDeploymentRead); connect.CodeOf(err) != connect.CodePermissionDenied {
					t.Fatalf("banned administrator allowed: %v", err)
				}
			})
		}
	}
}

func TestModerationFailureMetadataAndDatabaseOutage(t *testing.T) {
	db := moderationDB(t)
	db.Create(&database.Organization{ID: "restricted-org", Slug: "restricted-org", Status: "banned"})
	ctx := WithUser(context.Background(), &authv1.User{Id: "test-user"})
	err := CheckOrganizationAccess(ctx, "restricted-org", PermissionDeploymentRead)
	rpcErr, ok := err.(*connect.Error)
	if !ok || rpcErr.Meta().Get("X-Obiente-Restriction") != "organization" || rpcErr.Meta().Get("X-Obiente-Organization-ID") != "restricted-org" {
		t.Fatalf("missing restriction metadata: %v", err)
	}
	admin := WithUser(context.Background(), &authv1.User{Id: "test-admin", Roles: []string{RoleAdmin}})
	err = CheckOrganizationAccess(admin, "restricted-org", PermissionDeploymentStart)
	if e := err.(*connect.Error); e.Meta().Get("X-Obiente-Restriction") != "" {
		t.Fatal("denied admin mutation must preserve inspection view")
	}
	raw, _ := db.DB()
	raw.Close()
	if err := CheckOrganizationAccess(admin, "restricted-org", PermissionDeploymentRead); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("database outage: %v", err)
	}
}

func TestAuthenticationDoesNotTrustInternalHeaderOrPriorSession(t *testing.T) {
	db := moderationDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"sub":"test-user","email":"user@example.test"}`))
	}))
	defer server.Close()
	cfg := &AuthConfig{UserInfoURL: server.URL, HTTPClient: server.Client(), UserInfoCache: &UserInfoCache{}}
	called := 0
	handler := MiddlewareInterceptor(cfg).WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called++
		return connect.NewResponse(&authv1.GetPublicConfigResponse{}), nil
	})
	req := connect.NewRequest(&authv1.GetPublicConfigRequest{})
	req.Header().Set("Authorization", "Bearer same-session-token")
	if _, err := handler(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	db.Create(&database.UserBan{ID: "ban", UserID: "test-user", Type: "banned", BannedAt: time.Now()})
	req.Header().Set("x-internal-service-secret", "untrusted-header")
	if _, err := handler(context.Background(), req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("ban bypassed: %v", err)
	}
	if called != 1 {
		t.Fatalf("protected handler reached %d times", called)
	}
}

type moderationStreamConn struct {
	procedure string
	message   proto.Message
	sent      int
}

func (s *moderationStreamConn) Spec() connect.Spec           { return connect.Spec{Procedure: s.procedure} }
func (s *moderationStreamConn) Peer() connect.Peer           { return connect.Peer{} }
func (s *moderationStreamConn) RequestHeader() http.Header   { return http.Header{} }
func (s *moderationStreamConn) ResponseHeader() http.Header  { return http.Header{} }
func (s *moderationStreamConn) ResponseTrailer() http.Header { return http.Header{} }
func (s *moderationStreamConn) Receive(msg any) error {
	proto.Merge(msg.(proto.Message), s.message)
	return nil
}
func (s *moderationStreamConn) Send(any) error { s.sent++; return nil }

func TestOpenStreamStopsDeliveringAfterOrganizationOrAccountBan(t *testing.T) {
	for _, role := range []string{"", RoleAdmin} {
		t.Run(role, func(t *testing.T) {
			db := moderationDB(t)
			if err := db.AutoMigrate(&database.GameServer{}); err != nil {
				t.Fatal(err)
			}
			db.Create(&database.Organization{ID: "org-stream", Slug: "org-stream", Status: "active"})
			db.Create(&database.GameServer{ID: "gs-stream", OrganizationID: "org-stream"})
			ctx := WithUser(context.Background(), &authv1.User{Id: "test-user", Roles: []string{role}})
			conn := &moderationStreamConn{procedure: "/obiente.cloud.gameservers.v1.GameServerService/StreamGameServerLogs", message: &gameserversv1.StreamGameServerLogsRequest{GameServerId: "gs-stream"}}
			now := time.Now()
			stream := &moderatedStream{StreamingHandlerConn: conn, ctx: ctx, now: func() time.Time { return now }}
			if err := stream.Receive(&gameserversv1.StreamGameServerLogsRequest{}); err != nil {
				t.Fatal(err)
			}
			if err := stream.Send(&gameserversv1.GameServerLogLine{}); err != nil {
				t.Fatal(err)
			}
			db.Model(&database.Organization{}).Where("id = ?", "org-stream").Update("status", "banned")
			now = now.Add(time.Second)
			err := stream.Send(&gameserversv1.GameServerLogLine{})
			if (err == nil) != (role == RoleAdmin) {
				t.Fatalf("org ban after stream opened: %v", err)
			}
			db.Create(&database.UserBan{ID: "ban", UserID: "test-user", Type: "banned", BannedAt: time.Now()})
			now = now.Add(time.Second)
			before := conn.sent
			if err := stream.Send(&gameserversv1.GameServerLogLine{}); connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("user ban after stream opened: %v", err)
			}
			if conn.sent != before {
				t.Fatal("data delivered after account ban")
			}
		})
	}
}

func TestOrganizationGuardRejectsRequestBeforeListHandler(t *testing.T) {
	db := moderationDB(t)
	db.Create(&database.Organization{ID: "restricted-org", Slug: "restricted-org", Status: "banned"})
	ctx := WithUser(context.Background(), &authv1.User{Id: "test-user"})
	req := connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{OrganizationId: "restricted-org"})
	if err := checkRequestOrganizationAccess(ctx, req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("list request passed moderation: %v", err)
	}
}

func TestModerationIdleConnectionGuard(t *testing.T) {
	db := moderationDB(t)
	db.Create(&database.Organization{ID: "org-watch", Slug: "org-watch", Status: "active"})
	ctx := WithUser(context.Background(), &authv1.User{Id: "watch-user"})
	checked, revoked := make(chan struct{}, 1), make(chan struct{})
	stop := watchAccess(ctx, time.Millisecond, func(checkCtx context.Context) error {
		err := CheckOrganizationAccess(checkCtx, "org-watch", PermissionDeploymentManage)
		select {
		case checked <- struct{}{}:
		default:
		}
		return err
	}, func() { close(revoked) })
	defer stop()
	select {
	case <-checked:
	case <-time.After(time.Second):
		t.Fatal("guard never checked")
	}
	if err := db.Model(&database.Organization{}).Where("id = ?", "org-watch").Update("status", "banned").Error; err != nil {
		t.Fatal(err)
	}
	select {
	case <-revoked:
	case <-time.After(time.Second):
		t.Fatal("idle connection survived ban")
	}
}

func TestModerationDelegatedInspectionIsPermissionScoped(t *testing.T) {
	db := moderationDB(t)
	if err := db.AutoMigrate(&database.SuperadminRole{}, &database.SuperadminRoleBinding{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&database.Organization{ID: "restricted-org", Slug: "restricted-org", Status: "banned"})
	db.Create(&database.SuperadminRole{ID: "reviewer", Name: "Deployment reviewer", Permissions: `["superadmin.deployments.read"]`})
	db.Create(&database.SuperadminRoleBinding{ID: "binding", UserID: "reviewer-user", RoleID: "reviewer"})
	ctx := WithUser(context.Background(), &authv1.User{Id: "reviewer-user"})
	checker := NewPermissionChecker()
	if err := checker.CheckScopedPermission(ctx, "restricted-org", ScopedPermission{Permission: PermissionDeploymentRead}); err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{PermissionDeploymentStart, PermissionGameServersRead, PermissionOrganizationRead} {
		if err := checker.CheckScopedPermission(ctx, "restricted-org", ScopedPermission{Permission: permission}); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("unrelated permission %s: %v", permission, err)
		}
	}
}
