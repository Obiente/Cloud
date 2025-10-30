package auth

import (
	"context"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	"api/internal/auth"

	"connectrpc.com/connect"
)

type Service struct {
	authv1connect.UnimplementedAuthServiceHandler
}

func NewService() authv1connect.AuthServiceHandler { return &Service{} }

func (s *Service) GetCurrentUser(ctx context.Context, _ *connect.Request[authv1.GetCurrentUserRequest]) (*connect.Response[authv1.GetCurrentUserResponse], error) {
	ui, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewResponse(&authv1.GetCurrentUserResponse{User: nil}), nil
	}
    return connect.NewResponse(&authv1.GetCurrentUserResponse{User: ui}), nil
}
