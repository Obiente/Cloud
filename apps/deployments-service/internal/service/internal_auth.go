package deployments

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	"connectrpc.com/connect"
)

const internalServiceSecretHeader = "x-internal-service-secret"

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
		if i.secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(i.secret)) != 1 {
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid internal service credentials"))
		}
		return next(auth.WithSystemUser(ctx), req)
	}
}

func (i *InternalServiceAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		secret := conn.RequestHeader().Get(internalServiceSecretHeader)
		if secret == "" {
			return next(ctx, conn)
		}
		if i.secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(i.secret)) != 1 {
			return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid internal service credentials"))
		}
		return next(auth.WithSystemUser(ctx), conn)
	}
}

func (i *InternalServiceAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (i *InternalServiceAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
