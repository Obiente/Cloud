package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/services/organizations"
	"api/internal/zitadel"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Service struct {
	authv1connect.UnimplementedAuthServiceHandler
	db *gorm.DB
}

func NewService() authv1connect.AuthServiceHandler {
	return &Service{
		db: database.DB,
	}
}

func (s *Service) GetPublicConfig(ctx context.Context, _ *connect.Request[authv1.GetPublicConfigRequest]) (*connect.Response[authv1.GetPublicConfigResponse], error) {
	// Read configuration from environment variables
	billingEnabled := os.Getenv("BILLING_ENABLED") != "false" && os.Getenv("BILLING_ENABLED") != "0"
	selfHosted := os.Getenv("SELF_HOSTED") == "true" || os.Getenv("SELF_HOSTED") == "1"
	disableAuth := os.Getenv("DISABLE_AUTH") == "true" || os.Getenv("DISABLE_AUTH") == "1"

	return connect.NewResponse(&authv1.GetPublicConfigResponse{
		BillingEnabled: billingEnabled,
		SelfHosted:     selfHosted,
		DisableAuth:    disableAuth,
	}), nil
}

func (s *Service) Login(ctx context.Context, req *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	email := strings.TrimSpace(req.Msg.GetEmail())
	password := req.Msg.GetPassword()

	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email and password are required"))
	}

	// Use Zitadel client to authenticate
	zitadelClient := zitadel.NewClient()
	loginResp, err := zitadelClient.Login(email, password)
	if err != nil {
		// Handle error - loginResp might be nil or have error details
		errorMsg := "Authentication failed"
		if loginResp != nil && loginResp.Message != "" {
			errorMsg = loginResp.Message
		} else if err != nil {
			errorMsg = err.Error()
		}
		return connect.NewResponse(&authv1.LoginResponse{
			Success: false,
			Message: errorMsg,
		}), nil
	}

	return connect.NewResponse(&authv1.LoginResponse{
		Success:      loginResp.Success,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
	}), nil
}

func (s *Service) GetCurrentUser(ctx context.Context, _ *connect.Request[authv1.GetCurrentUserRequest]) (*connect.Response[authv1.GetCurrentUserResponse], error) {
	ui, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewResponse(&authv1.GetCurrentUserResponse{User: nil}), nil
	}
    return connect.NewResponse(&authv1.GetCurrentUserResponse{User: ui}), nil
}

func (s *Service) UpdateUserProfile(ctx context.Context, req *connect.Request[authv1.UpdateUserProfileRequest]) (*connect.Response[authv1.UpdateUserProfileResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if user.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user ID is required"))
	}

	// Get user profile resolver
	resolver := organizations.GetUserProfileResolver()

	// Build update map
	updates := make(map[string]interface{})
	profile := make(map[string]interface{})

	if req.Msg.GivenName != nil {
		givenName := strings.TrimSpace(req.Msg.GetGivenName())
		if givenName != "" {
			profile["firstName"] = givenName
		}
	}

	if req.Msg.FamilyName != nil {
		familyName := strings.TrimSpace(req.Msg.GetFamilyName())
		if familyName != "" {
			profile["lastName"] = familyName
		}
	}

	if req.Msg.Name != nil {
		displayName := strings.TrimSpace(req.Msg.GetName())
		if displayName != "" {
			profile["displayName"] = displayName
		}
	}

	if req.Msg.PreferredUsername != nil {
		// Note: Username updates might need a different endpoint in Zitadel
		// For now, we'll skip this or handle it separately
	}

	if len(profile) > 0 {
		updates["profile"] = profile
	}

	if req.Msg.Locale != nil {
		locale := strings.TrimSpace(req.Msg.GetLocale())
		if locale != "" {
			updates["preferredLanguage"] = locale
		}
	}

	if len(updates) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no fields to update"))
	}

	// Update profile via Zitadel management API
	updatedUser, err := resolver.UpdateProfile(ctx, user.Id, updates)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update profile: %w", err))
	}

	return connect.NewResponse(&authv1.UpdateUserProfileResponse{
		User: updatedUser,
	}), nil
}

func (s *Service) ConnectGitHub(ctx context.Context, req *connect.Request[authv1.ConnectGitHubRequest]) (*connect.Response[authv1.ConnectGitHubResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	// Check if user already has a GitHub integration
	var existing database.GitHubIntegration
	err = s.db.Where("user_id = ?", user.Id).First(&existing).Error
	
	now := time.Now()
	userID := user.Id
	
	if err == gorm.ErrRecordNotFound {
		// Create new integration
		integration := database.GitHubIntegration{
			ID:          uuid.New().String(),
			UserID:      &userID,
			OrganizationID: nil,
			Token:       req.Msg.GetAccessToken(), // TODO: Encrypt this token before storing
			Username:    req.Msg.GetUsername(),
			Scope:       req.Msg.GetScope(),
			ConnectedAt: now,
			UpdatedAt:   now,
		}
		
		if err := s.db.Create(&integration).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save GitHub integration: %w", err))
		}
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing integration: %w", err))
	} else {
		// Update existing integration
		existing.Token = req.Msg.GetAccessToken() // TODO: Encrypt this token before storing
		existing.Username = req.Msg.GetUsername()
		existing.Scope = req.Msg.GetScope()
		existing.UpdatedAt = now
		
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update GitHub integration: %w", err))
		}
	}

	return connect.NewResponse(&authv1.ConnectGitHubResponse{
		Success:  true,
		Username: req.Msg.GetUsername(),
	}), nil
}

func (s *Service) DisconnectGitHub(ctx context.Context, _ *connect.Request[authv1.DisconnectGitHubRequest]) (*connect.Response[authv1.DisconnectGitHubResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	// Delete GitHub integration
	result := s.db.Where("user_id = ?", user.Id).Delete(&database.GitHubIntegration{})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to disconnect GitHub: %w", result.Error))
	}

	return connect.NewResponse(&authv1.DisconnectGitHubResponse{
		Success: true,
	}), nil
}

func (s *Service) GetGitHubStatus(ctx context.Context, _ *connect.Request[authv1.GetGitHubStatusRequest]) (*connect.Response[authv1.GetGitHubStatusResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	// Check if user has a GitHub integration
	var integration database.GitHubIntegration
	err = s.db.Where("user_id = ?", user.Id).First(&integration).Error
	
	if err == gorm.ErrRecordNotFound {
		return connect.NewResponse(&authv1.GetGitHubStatusResponse{
			Connected: false,
			Username:   "",
		}), nil
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check GitHub status: %w", err))
	}

	return connect.NewResponse(&authv1.GetGitHubStatusResponse{
		Connected: true,
		Username:  integration.Username,
	}), nil
}

func (s *Service) ConnectOrganizationGitHub(ctx context.Context, req *connect.Request[authv1.ConnectOrganizationGitHubRequest]) (*connect.Response[authv1.ConnectOrganizationGitHubResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has permission to manage integrations for this organization
	// Only owners and admins can manage organization integrations
	if err := s.verifyOrgAdminPermission(ctx, orgID, user); err != nil {
		return nil, err
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	// Check if organization already has a GitHub integration
	var existing database.GitHubIntegration
	err = s.db.Where("organization_id = ?", orgID).First(&existing).Error
	
	now := time.Now()
	
	if err == gorm.ErrRecordNotFound {
		// Create new integration
		integration := database.GitHubIntegration{
			ID:            uuid.New().String(),
			UserID:        nil,
			OrganizationID: &orgID,
			Token:         req.Msg.GetAccessToken(), // TODO: Encrypt this token before storing
			Username:      req.Msg.GetUsername(),
			Scope:         req.Msg.GetScope(),
			ConnectedAt:   now,
			UpdatedAt:     now,
		}
		
		if err := s.db.Create(&integration).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save GitHub integration: %w", err))
		}
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing integration: %w", err))
	} else {
		// Update existing integration
		existing.Token = req.Msg.GetAccessToken() // TODO: Encrypt this token before storing
		existing.Username = req.Msg.GetUsername()
		existing.Scope = req.Msg.GetScope()
		existing.UpdatedAt = now
		
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update GitHub integration: %w", err))
		}
	}

	return connect.NewResponse(&authv1.ConnectOrganizationGitHubResponse{
		Success:  true,
		Username: req.Msg.GetUsername(),
	}), nil
}

func (s *Service) DisconnectOrganizationGitHub(ctx context.Context, req *connect.Request[authv1.DisconnectOrganizationGitHubRequest]) (*connect.Response[authv1.DisconnectOrganizationGitHubResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Verify user has permission to manage integrations for this organization
	// Only owners and admins can manage organization integrations
	if err := s.verifyOrgAdminPermission(ctx, orgID, user); err != nil {
		return nil, err
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	// Delete organization GitHub integration
	result := s.db.Where("organization_id = ?", orgID).Delete(&database.GitHubIntegration{})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to disconnect GitHub: %w", result.Error))
	}

	return connect.NewResponse(&authv1.DisconnectOrganizationGitHubResponse{
		Success: true,
	}), nil
}

func (s *Service) ListGitHubIntegrations(ctx context.Context, _ *connect.Request[authv1.ListGitHubIntegrationsRequest]) (*connect.Response[authv1.ListGitHubIntegrationsResponse], error) {
	// Get authenticated user
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	var integrations []database.GitHubIntegration
	
	// Get user's GitHub integration
	var userIntegration database.GitHubIntegration
	if err := s.db.Where("user_id = ?", user.Id).First(&userIntegration).Error; err == nil {
		integrations = append(integrations, userIntegration)
	}

	// TODO: Get organization integrations where user is a member/admin
	// For now, we'll get all organizations the user belongs to and check for integrations
	var orgMembers []database.OrganizationMember
	if err := s.db.Where("user_id = ? AND status = ?", user.Id, "active").Find(&orgMembers).Error; err == nil {
		orgIDs := make([]string, 0, len(orgMembers))
		for _, member := range orgMembers {
			orgIDs = append(orgIDs, member.OrganizationID)
		}
		
		if len(orgIDs) > 0 {
			var orgIntegrations []database.GitHubIntegration
			if err := s.db.Where("organization_id IN ?", orgIDs).Find(&orgIntegrations).Error; err == nil {
				integrations = append(integrations, orgIntegrations...)
			}
		}
	}

	// Convert to proto
	protoIntegrations := make([]*authv1.GitHubIntegrationInfo, 0, len(integrations))
	for _, integration := range integrations {
		isUser := integration.UserID != nil && *integration.UserID == user.Id
		info := &authv1.GitHubIntegrationInfo{
			Id:          integration.ID,
			Username:    integration.Username,
			Scope:       integration.Scope,
			IsUser:      isUser,
			ConnectedAt: timestamppb.New(integration.ConnectedAt),
		}
		
		if !isUser && integration.OrganizationID != nil {
			info.OrganizationId = *integration.OrganizationID
			// Fetch organization name from database
			var org database.Organization
			if err := s.db.Where("id = ?", *integration.OrganizationID).First(&org).Error; err == nil {
				info.OrganizationName = org.Name
			}
		}
		
		protoIntegrations = append(protoIntegrations, info)
	}

	return connect.NewResponse(&authv1.ListGitHubIntegrationsResponse{
		Integrations: protoIntegrations,
	}), nil
}

// verifyOrgAdminPermission verifies that the user is an admin or owner of the organization
func (s *Service) verifyOrgAdminPermission(ctx context.Context, orgID string, user *authv1.User) error {
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Superadmins bypass permission checks
	if auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil
	}

	// Check if user is a member with admin or owner role
	var member database.OrganizationMember
	if err := s.db.Where("organization_id = ? AND user_id = ?", orgID, user.Id).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this organization"))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("membership lookup: %w", err))
	}

	if !strings.EqualFold(member.Status, "active") {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("inactive members cannot perform this action"))
	}

	role := strings.ToLower(member.Role)
	if role != "owner" && role != "admin" {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only organization owners and admins can manage integrations"))
	}

	return nil
}
