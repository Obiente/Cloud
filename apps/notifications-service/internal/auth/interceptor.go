package auth

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
)

const (
	// InternalServiceSecretHeader is the header key for internal service authentication
	InternalServiceSecretHeader = "x-internal-service-secret"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// InternalServiceCallKey is the context key for marking internal service calls
	InternalServiceCallKey contextKey = "internal_service_call"
)

// InternalServiceAuthInterceptor validates internal service-to-service calls
// This allows services to call internal endpoints without user authentication
type InternalServiceAuthInterceptor struct {
	secret string
}

// NewInternalServiceAuthInterceptor creates a new internal service auth interceptor
func NewInternalServiceAuthInterceptor(secret string) *InternalServiceAuthInterceptor {
	return &InternalServiceAuthInterceptor{
		secret: secret,
	}
}

// WrapUnary validates authentication for unary RPCs
func (i *InternalServiceAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Check if this is an internal service call
		secret := req.Header().Get(InternalServiceSecretHeader)
		if secret != "" {
			if secret != i.secret {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid %s", InternalServiceSecretHeader))
			}
			// Valid internal service call - set a flag in context
			ctx = context.WithValue(ctx, InternalServiceCallKey, true)
		}
		return next(ctx, req)
	}
}

// WrapStreamingHandler validates authentication for streaming RPCs
func (i *InternalServiceAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		secret := conn.RequestHeader().Get(InternalServiceSecretHeader)
		if secret != "" {
			if secret != i.secret {
				return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid %s", InternalServiceSecretHeader))
			}
			ctx = context.WithValue(ctx, InternalServiceCallKey, true)
		}
		return next(ctx, conn)
	}
}

// WrapUnaryClient is required by connect.Interceptor but not used for server-side
func (i *InternalServiceAuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

// WrapStreamingClient is required by connect.Interceptor but not used for server-side
func (i *InternalServiceAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// IsInternalServiceCall checks if the context indicates an internal service call
func IsInternalServiceCall(ctx context.Context) bool {
	val := ctx.Value(InternalServiceCallKey)
	if val == nil {
		return false
	}
	b, ok := val.(bool)
	return ok && b
}

