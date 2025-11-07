package common

import (
	"context"
	"fmt"
	"strings"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	"api/internal/auth"

	"connectrpc.com/connect"
)

// EnsureAuthenticated ensures the user is authenticated for streaming RPCs.
// This is needed because unary interceptors may not run for streaming RPCs.
// Returns the context with user info and nil error if authenticated, or an error if not.
func EnsureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	// Check if user is already in context (interceptor ran)
	if userInfo, err := auth.GetUserFromContext(ctx); err == nil && userInfo != nil {
		return ctx, nil
	}

	// Extract token from Authorization header
	authHeader := req.Header().Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Use AuthenticateAndSetContext helper which handles token validation and context setup
	ctx, userInfo, err := auth.AuthenticateAndSetContext(ctx, authHeader)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if userInfo == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	return ctx, nil
}

// GetUserFromContextOrError gets the user from context or returns an authentication error.
func GetUserFromContextOrError(ctx context.Context) (*authv1.User, error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	return user, nil
}

