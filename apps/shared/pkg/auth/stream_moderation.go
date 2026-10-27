package auth

import (
	"context"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type moderationInterceptor struct {
	connect.UnaryInterceptorFunc
	config *AuthConfig
}

func (i *moderationInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if conn.RequestHeader().Get("x-internal-service-secret") != "" && i.config.AllowInternalServiceAuth {
			return next(ctx, conn) // The subsequent internal interceptor must authenticate.
		}
		if isAuthDisabled() && i.config.UserInfoCache == nil {
			return next(WithUser(ctx, getMockDevUser()), conn)
		}
		if i.config.UserInfoCache == nil {
			return connect.NewError(connect.CodeUnavailable, ErrUserInfoFailed)
		}
		header := conn.RequestHeader().Get(AuthorizationHeader)
		if !strings.HasPrefix(header, BearerPrefix) || strings.TrimPrefix(header, BearerPrefix) == "" {
			return connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
		}
		user, err := i.config.validateToken(ctx, strings.TrimPrefix(header, BearerPrefix))
		if err != nil {
			if connect.CodeOf(err) == connect.CodePermissionDenied || connect.CodeOf(err) == connect.CodeUnavailable {
				return err
			}
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		ctx = WithUser(ctx, user)
		return next(ctx, &moderatedStream{StreamingHandlerConn: conn, ctx: ctx})
	}
}

type moderatedStream struct {
	connect.StreamingHandlerConn
	ctx       context.Context
	request   proto.Message
	mu        sync.Mutex
	nextCheck time.Time
	now       func() time.Time
}

func (s *moderatedStream) Receive(message any) error {
	if err := EnsureActiveUser(s.ctx); err != nil {
		return err
	}
	if err := s.StreamingHandlerConn.Receive(message); err != nil {
		return err
	}
	if msg, ok := message.(proto.Message); ok {
		permission := streamModerationPermission(s.Spec().Procedure)
		if err := checkMessageModeration(s.ctx, msg.ProtoReflect(), permission); err != nil {
			return err
		}
		s.mu.Lock()
		if s.request == nil {
			s.request = proto.Clone(msg)
			s.nextCheck = time.Time{}
		}
		s.mu.Unlock()
	}
	return s.checkAccess()
}

func (s *moderatedStream) Send(message any) error {
	if err := s.checkAccess(); err != nil {
		return err
	}
	return s.StreamingHandlerConn.Send(message)
}

// Recheck active streams at least once per second, without issuing database
// queries for every log line. Normal resource permissions remain in the handler.
func (s *moderatedStream) checkAccess() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	if now.Before(s.nextCheck) {
		return nil
	}
	if err := EnsureActiveUser(s.ctx); err != nil {
		return err
	}
	if s.request == nil {
		return nil
	}
	permission := streamModerationPermission(s.Spec().Procedure)
	if err := checkMessageModeration(s.ctx, s.request.ProtoReflect(), permission); err != nil {
		return err
	}
	s.nextCheck = now.Add(time.Second)
	return nil
}

func streamModerationPermission(procedure string) string {
	if strings.HasSuffix(procedure, "/StreamTerminal") {
		return PermissionDeploymentManage
	}
	permission, _, _ := inferPermissionFromProcedure(procedure)
	return permission
}

func checkMessageModeration(ctx context.Context, msg protoreflect.Message, permission string) error {
	var accessErr error
	msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() || field.IsMap() {
			return true
		}
		if field.Kind() == protoreflect.MessageKind {
			accessErr = checkMessageModeration(ctx, value.Message(), permission)
			return accessErr == nil
		}
		if field.Kind() != protoreflect.StringKind || value.String() == "" {
			return true
		}
		if field.Name() == "organization_id" {
			accessErr = CheckOrganizationAccess(ctx, value.String(), permission)
			return accessErr == nil
		}
		table := map[protoreflect.Name]string{
			"deployment_id": "deployments", "game_server_id": "game_servers",
			"database_id": "database_instances", "vps_id": "vps_instances",
		}[field.Name()]
		if table == "" {
			return true
		}
		var resource struct{ OrganizationID string }
		if err := database.DB.WithContext(ctx).Table(table).Select("organization_id").Where("id = ?", value.String()).Take(&resource).Error; err != nil {
			accessErr = ModerationError(err, "")
		} else {
			accessErr = CheckOrganizationAccess(ctx, resource.OrganizationID, permission)
		}
		return accessErr == nil
	})
	return accessErr
}
