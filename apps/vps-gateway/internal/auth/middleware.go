package auth

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

const (
	// MetadataKey is the key for the API secret in gRPC metadata
	MetadataKey = "x-api-secret"
)

// AuthInterceptor validates the shared secret from gRPC metadata
type AuthInterceptor struct {
	secret string
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(secret string) *AuthInterceptor {
	return &AuthInterceptor{
		secret: secret,
	}
}

// WrapUnary validates authentication for unary RPCs
func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := a.validateSecret(req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

// WrapStream validates authentication for streaming RPCs
func (a *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := a.validateSecret(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

// WrapUnaryClient is required by connect.Interceptor but not used for server-side
func (a *AuthInterceptor) WrapUnaryClient(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

// WrapStreamingClient is required by connect.Interceptor but not used for server-side
func (a *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// validateSecret checks if the secret in the header matches the configured secret
func (a *AuthInterceptor) validateSecret(header http.Header) error {
	secret := header.Get(MetadataKey)
	if secret == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing %s header", MetadataKey))
	}
	if secret != a.secret {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid %s", MetadataKey))
	}
	return nil
}

