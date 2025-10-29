package auth

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	"api/internal/auth"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	authv1connect.UnimplementedAuthServiceHandler

	user *authv1.User
}

func NewService() authv1connect.AuthServiceHandler {
	return &Service{
		user: &authv1.User{
			Id:        "user_mock_123",
			Email:     "developer@obiente.cloud",
			Name:      "Obiente Developer",
			AvatarUrl: "https://cdn.obiente.cloud/assets/avatar/mock.png",
			CreatedAt: timestamppb.New(time.Date(2024, time.January, 15, 8, 30, 0, 0, time.UTC)),
			Timezone:  "UTC",
		},
	}
}

func (s *Service) GetCurrentUser(ctx context.Context, _ *connect.Request[authv1.GetCurrentUserRequest]) (*connect.Response[authv1.GetCurrentUserResponse], error) {
	// Try to get user from context first (from auth middleware)
	userInfo, err := auth.GetUserFromContext(ctx)
	if err == nil {
		// User found in context, use the real user info
		user := &authv1.User{
			Id:        userInfo.Sub,
			Email:     userInfo.Email,
			Name:      userInfo.Name,
			AvatarUrl: userInfo.Picture,
			CreatedAt: timestamppb.Now(), // We don't have created_at in the token
			Timezone:  "UTC",             // Default timezone
		}
		res := connect.NewResponse(&authv1.GetCurrentUserResponse{User: user})
		return res, nil
	}

	// Fall back to mock user if not in context or in development mode
	if os.Getenv("NODE_ENV") != "production" {
		res := connect.NewResponse(&authv1.GetCurrentUserResponse{User: s.cloneUser()})
		return res, nil
	}

	// In production with no valid user, return unauthenticated error
	return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
}

func (s *Service) InitiateLogin(_ context.Context, req *connect.Request[authv1.InitiateLoginRequest]) (*connect.Response[authv1.InitiateLoginResponse], error) {
	state := fmt.Sprintf("mock-state-%d", time.Now().UnixNano())
	redirect := req.Msg.GetRedirectUri()
	loginURL := fmt.Sprintf("https://auth.obiente.cloud/login?state=%s", url.QueryEscape(state))
	if redirect != "" {
		loginURL = fmt.Sprintf("%s&redirect_uri=%s", loginURL, url.QueryEscape(redirect))
	}

	res := connect.NewResponse(&authv1.InitiateLoginResponse{
		LoginUrl: loginURL,
		State:    state,
	})
	return res, nil
}

func (s *Service) HandleCallback(_ context.Context, req *connect.Request[authv1.HandleCallbackRequest]) (*connect.Response[authv1.HandleCallbackResponse], error) {
	res := connect.NewResponse(&authv1.HandleCallbackResponse{
		AccessToken:  fmt.Sprintf("mock-access-token-%s", req.Msg.GetCode()),
		RefreshToken: fmt.Sprintf("mock-refresh-token-%s", req.Msg.GetState()),
		ExpiresIn:    3600,
		User:         s.cloneUser(),
	})
	return res, nil
}

func (s *Service) RefreshToken(_ context.Context, req *connect.Request[authv1.RefreshTokenRequest]) (*connect.Response[authv1.RefreshTokenResponse], error) {
	res := connect.NewResponse(&authv1.RefreshTokenResponse{
		AccessToken:  fmt.Sprintf("mock-access-token-%d", time.Now().Unix()),
		RefreshToken: req.Msg.GetRefreshToken(),
		ExpiresIn:    3600,
	})
	return res, nil
}

func (s *Service) Logout(_ context.Context, _ *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	res := connect.NewResponse(&authv1.LogoutResponse{Success: true})
	return res, nil
}

func (s *Service) cloneUser() *authv1.User {
	if s.user == nil {
		return nil
	}

	return &authv1.User{
		Id:        s.user.GetId(),
		Email:     s.user.GetEmail(),
		Name:      s.user.GetName(),
		AvatarUrl: s.user.GetAvatarUrl(),
		CreatedAt: timestamppb.New(s.user.GetCreatedAt().AsTime()),
		Timezone:  s.user.GetTimezone(),
	}
}
