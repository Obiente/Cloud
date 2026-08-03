package deployments

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	deploymentsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1/deploymentsv1connect"

	"connectrpc.com/connect"
)

const internalServiceSecretHeader = "x-internal-service-secret"
const deploymentsInternalServiceSecretEnv = "DEPLOYMENTS_INTERNAL_SERVICE_SECRET"

// InternalServiceAuthInterceptor authenticates node-to-node deployment calls.
// A valid internal call runs as the system user so permission checks still have
// an explicit principal instead of being bypassed.
type InternalServiceAuthInterceptor struct {
	secret string
}

func NewInternalServiceAuthInterceptor(secret string) *InternalServiceAuthInterceptor {
	return &InternalServiceAuthInterceptor{secret: secret}
}

func (i *InternalServiceAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		secret := req.Header().Get(internalServiceSecretHeader)
		if secret == "" {
			return next(ctx, req)
		}

		internalCtx, err := i.authenticateForwardedTrigger(ctx, secret, req.Spec().Procedure, req.Header().Get(orchestrator.ForwardTargetNodeHeader))
		if err != nil {
			return nil, err
		}
		return next(internalCtx, req)
	}
}

func (i *InternalServiceAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if conn.RequestHeader().Get(internalServiceSecretHeader) == "" {
			return next(ctx, conn)
		}
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("internal credentials are not accepted for streaming deployment procedures"))
	}
}

func (i *InternalServiceAuthInterceptor) authenticateForwardedTrigger(ctx context.Context, secret, procedure, targetNode string) (context.Context, error) {
	if i.secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(i.secret)) != 1 {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid internal service credentials"))
	}
	if procedure != deploymentsv1connect.DeploymentServiceTriggerDeploymentProcedure &&
		procedure != deploymentsv1connect.DeploymentServiceDeleteDeploymentProcedure {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("internal credentials are only accepted for deployment triggers and runtime deletion"))
	}
	if strings.TrimSpace(targetNode) == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("internal deployment triggers require a forwarding target"))
	}
	return auth.WithSystemUser(ctx), nil
}

func (i *InternalServiceAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (i *InternalServiceAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
