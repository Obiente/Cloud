package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
)

func TestModerationAcrossHTTPAndOpenStream(t *testing.T) {
	for _, admin := range []bool{false, true} {
		name := "tenant"
		if admin {
			name = "platform-admin"
		}
		t.Run(name, func(t *testing.T) {
			db := moderationDB(t)
			if err := db.AutoMigrate(&database.GameServer{}); err != nil {
				t.Fatal(err)
			}
			db.Create(&database.Organization{ID: "org-http", Slug: "org-http", Status: "active"})
			db.Create(&database.GameServer{ID: "gs-http", OrganizationID: "org-http"})
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"sub":"http-user","email":"user@example.test"}`))
			}))
			defer identity.Close()
			cfg := &AuthConfig{UserInfoURL: identity.URL, HTTPClient: identity.Client(), UserInfoCache: &UserInfoCache{}}
			if admin {
				cfg.SuperAdmins = map[string]struct{}{"user@example.test": {}}
			}
			const listPath = "/obiente.cloud.deployments.v1.DeploymentService/ListDeployments"
			const streamPath = "/obiente.cloud.gameservers.v1.GameServerService/StreamGameServerLogs"
			mux := http.NewServeMux()
			mux.Handle(listPath, connect.NewUnaryHandler(listPath, func(ctx context.Context, req *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
				return connect.NewResponse(&deploymentsv1.ListDeploymentsResponse{}), nil
			}, connect.WithInterceptors(MiddlewareInterceptor(cfg))))
			advance := make(chan struct{})
			mux.Handle(streamPath, connect.NewServerStreamHandler(streamPath, func(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerLogsRequest], stream *connect.ServerStream[gameserversv1.GameServerLogLine]) error {
				for {
					if err := stream.Send(&gameserversv1.GameServerLogLine{}); err != nil {
						return err
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-advance:
					}
				}
			}, connect.WithInterceptors(MiddlewareInterceptor(cfg))))
			server := httptest.NewServer(mux)
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			listClient := connect.NewClient[deploymentsv1.ListDeploymentsRequest, deploymentsv1.ListDeploymentsResponse](server.Client(), server.URL+listPath)
			listReq := connect.NewRequest(&deploymentsv1.ListDeploymentsRequest{OrganizationId: "org-http"})
			listReq.Header().Set(AuthorizationHeader, "Bearer test-session")
			if _, err := listClient.CallUnary(ctx, listReq); err != nil {
				t.Fatal(err)
			}
			streamClient := connect.NewClient[gameserversv1.StreamGameServerLogsRequest, gameserversv1.GameServerLogLine](server.Client(), server.URL+streamPath)
			streamReq := connect.NewRequest(&gameserversv1.StreamGameServerLogsRequest{GameServerId: "gs-http"})
			streamReq.Header().Set(AuthorizationHeader, "Bearer test-session")
			stream, err := streamClient.CallServerStream(ctx, streamReq)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			if !stream.Receive() {
				t.Fatal(stream.Err())
			}
			if err := db.Model(&database.Organization{}).Where("id = ?", "org-http").Update("status", "banned").Error; err != nil {
				t.Fatal(err)
			}
			_, err = listClient.CallUnary(ctx, listReq)
			if admin {
				if err != nil {
					t.Fatalf("admin inspection denied: %v", err)
				}
				if err := db.Create(&database.UserBan{ID: "http-ban", UserID: "http-user", Type: "banned", BannedAt: time.Now()}).Error; err != nil {
					t.Fatal(err)
				}
				_, err = listClient.CallUnary(ctx, listReq)
			}
			scope := "organization"
			if admin {
				scope = "account"
			}
			assertHTTPRestriction(t, err, scope)
			// Exercise the real stream revalidation interval and Connect trailers.
			time.Sleep(1100 * time.Millisecond)
			advance <- struct{}{}
			if stream.Receive() {
				t.Fatal("stream delivered data after restriction")
			}
			assertHTTPRestriction(t, stream.Err(), scope)
		})
	}
}

func assertHTTPRestriction(t *testing.T, err error, scope string) {
	t.Helper()
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) || rpcErr.Code() != connect.CodePermissionDenied || rpcErr.Meta().Get("X-Obiente-Restriction") != scope {
		t.Fatalf("expected %s restriction over HTTP, got %v", scope, err)
	}
}
