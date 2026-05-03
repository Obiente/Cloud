package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
	"github.com/obiente/cloud/apps/shared/pkg/zitadel"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	authv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1/authv1connect"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (s *Service) ConnectOrganizationGitHubApp(ctx context.Context, req *connect.Request[authv1.ConnectOrganizationGitHubAppRequest]) (*connect.Response[authv1.ConnectOrganizationGitHubAppResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}
	if req.Msg.GetInstallationId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("installation_id is required"))
	}

	if err := s.verifyOrgAdminPermission(ctx, orgID, user); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialized"))
	}

	installationID := req.Msg.GetInstallationId()
	installation, err := verifyGitHubAppInstallation(ctx, installationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("failed to verify GitHub App installation: %w", err))
	}
	if err := verifyGitHubAppInstallationForUser(ctx, req.Msg.GetSetupCode(), installationID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("failed to verify GitHub installer access: %w", err))
	}

	now := time.Now()
	accountLogin := strings.TrimSpace(installation.Account.Login)
	if accountLogin == "" {
		accountLogin = fmt.Sprintf("installation-%d", installationID)
	}
	accountType := strings.TrimSpace(installation.Account.Type)
	if accountType == "" {
		accountType = "Organization"
	}
	scope := "github_app"
	if selection := strings.TrimSpace(installation.RepositorySelection); selection != "" {
		scope += ":" + selection
	}

	integration := database.GitHubIntegration{
		ID:                      uuid.New().String(),
		UserID:                  nil,
		OrganizationID:          &orgID,
		Token:                   "",
		RefreshToken:            nil,
		Username:                accountLogin,
		Scope:                   scope,
		AuthType:                "github_app",
		GitHubAppInstallationID: &installationID,
		GitHubAppAccountLogin:   &accountLogin,
		GitHubAppAccountType:    &accountType,
		TokenExpiresAt:          nil,
		ConnectedAt:             now,
		UpdatedAt:               now,
	}

	if err := s.upsertGitHubIntegration("organization_id", integration); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save GitHub App integration: %w", err))
	}

	return connect.NewResponse(&authv1.ConnectOrganizationGitHubAppResponse{
		Success:        true,
		AccountLogin:   accountLogin,
		InstallationId: installationID,
	}), nil
}

type githubAppInstallation struct {
	ID                  int64  `json:"id"`
	RepositorySelection string `json:"repository_selection"`
	Account             struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
}

type githubAppUserTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type githubUserInstallationsResponse struct {
	Installations []struct {
		ID int64 `json:"id"`
	} `json:"installations"`
}

func verifyGitHubAppInstallation(ctx context.Context, installationID int64) (*githubAppInstallation, error) {
	if installationID <= 0 {
		return nil, fmt.Errorf("installation_id is required")
	}

	appJWT, err := createGitHubAppJWT(time.Now())
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub App installation verification request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub App installation verification failed: %d - %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var installation githubAppInstallation
	if err := json.Unmarshal(body, &installation); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub App installation response: %w", err)
	}
	if installation.ID != installationID {
		return nil, fmt.Errorf("GitHub App installation ID mismatch")
	}

	return &installation, nil
}

func verifyGitHubAppInstallationForUser(ctx context.Context, setupCode string, installationID int64) error {
	setupCode = strings.TrimSpace(setupCode)
	if setupCode == "" {
		return fmt.Errorf("GitHub App user authorization code is required")
	}

	userToken, err := exchangeGitHubAppUserCode(ctx, setupCode)
	if err != nil {
		return err
	}
	if strings.TrimSpace(userToken) == "" {
		return fmt.Errorf("GitHub App user authorization returned no access token")
	}

	for page := 1; page <= 10; page++ {
		found, hasMore, err := userCanAccessGitHubInstallation(ctx, userToken, installationID, page)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		if !hasMore {
			break
		}
	}

	return fmt.Errorf("GitHub user is not associated with installation %d", installationID)
}

func exchangeGitHubAppUserCode(ctx context.Context, setupCode string) (string, error) {
	clientID := strings.TrimSpace(os.Getenv("GITHUB_APP_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GITHUB_APP_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("GITHUB_APP_CLIENT_ID and GITHUB_APP_CLIENT_SECRET are required for secure GitHub App installation verification")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", setupCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub App user authorization token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub App user authorization token request failed: %d - %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp githubAppUserTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode GitHub App user authorization response: %w", err)
	}
	if tokenResp.Error != "" {
		if tokenResp.ErrorDescription != "" {
			return "", fmt.Errorf("GitHub App user authorization failed: %s: %s", tokenResp.Error, tokenResp.ErrorDescription)
		}
		return "", fmt.Errorf("GitHub App user authorization failed: %s", tokenResp.Error)
	}

	return tokenResp.AccessToken, nil
}

func userCanAccessGitHubInstallation(ctx context.Context, userToken string, installationID int64, page int) (bool, bool, error) {
	reqURL := fmt.Sprintf("https://api.github.com/user/installations?per_page=100&page=%d", page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, false, err
	}
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, false, fmt.Errorf("GitHub user installation lookup failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, false, fmt.Errorf("GitHub user installation lookup failed: %d - %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var installationsResp githubUserInstallationsResponse
	if err := json.Unmarshal(body, &installationsResp); err != nil {
		return false, false, fmt.Errorf("failed to decode GitHub user installations response: %w", err)
	}

	for _, installation := range installationsResp.Installations {
		if installation.ID == installationID {
			return true, false, nil
		}
	}

	return false, len(installationsResp.Installations) == 100, nil
}

func createGitHubAppJWT(now time.Time) (string, error) {
	appID := strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
	if appID == "" {
		return "", fmt.Errorf("GITHUB_APP_ID is required for GitHub App installations")
	}

	key, err := loadGitHubAppPrivateKey()
	if err != nil {
		return "", err
	}

	header, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]interface{}{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign GitHub App JWT: %w", err)
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func loadGitHubAppPrivateKey() (*rsa.PrivateKey, error) {
	keyPEM := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY"))
	if keyPEM == "" {
		if encoded := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_BASE64")); encoded != "" {
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, fmt.Errorf("failed to decode GITHUB_APP_PRIVATE_KEY_BASE64: %w", err)
			}
			keyPEM = string(decoded)
		}
	}
	if keyPEM == "" {
		if path := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")); path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read GITHUB_APP_PRIVATE_KEY_PATH: %w", err)
			}
			keyPEM = string(data)
		}
	}
	if keyPEM == "" {
		return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY, GITHUB_APP_PRIVATE_KEY_BASE64, or GITHUB_APP_PRIVATE_KEY_PATH is required for GitHub App installations")
	}

	keyPEM = strings.ReplaceAll(keyPEM, `\n`, "\n")
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode GitHub App private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub App private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("GitHub App private key must be an RSA private key")
	}
	return key, nil
}

func (s *Service) DisconnectOrganizationGitHubApp(ctx context.Context, req *connect.Request[authv1.DisconnectOrganizationGitHubAppRequest]) (*connect.Response[authv1.DisconnectOrganizationGitHubAppResponse], error) {
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

	return connect.NewResponse(&authv1.DisconnectOrganizationGitHubAppResponse{
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
			AuthType:    githubIntegrationAuthType(integration),
		}
		if integration.GitHubAppInstallationID != nil {
			info.GithubAppInstallationId = *integration.GitHubAppInstallationID
		}
		if integration.GitHubAppAccountLogin != nil {
			info.GithubAppAccountLogin = *integration.GitHubAppAccountLogin
		}
		if integration.GitHubAppAccountType != nil {
			info.GithubAppAccountType = *integration.GitHubAppAccountType
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

func githubIntegrationAuthType(integration database.GitHubIntegration) string {
	authType := strings.TrimSpace(integration.AuthType)
	if authType == "" {
		return "github_app"
	}
	return authType
}

func (s *Service) upsertGitHubIntegration(conflictColumn string, integration database.GitHubIntegration) error {
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: conflictColumn}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"token":                      integration.Token,
			"refresh_token":              integration.RefreshToken,
			"username":                   integration.Username,
			"scope":                      integration.Scope,
			"auth_type":                  integration.AuthType,
			"github_app_installation_id": integration.GitHubAppInstallationID,
			"github_app_account_login":   integration.GitHubAppAccountLogin,
			"github_app_account_type":    integration.GitHubAppAccountType,
			"token_expires_at":           integration.TokenExpiresAt,
			"updated_at":                 integration.UpdatedAt,
			"connected_at":               integration.ConnectedAt,
			"user_id":                    integration.UserID,
			"organization_id":            integration.OrganizationID,
		}),
	}).Create(&integration).Error
}

// verifyOrgAdminPermission verifies that the user is an admin or owner of the organization.
// Deprecated: use common.AuthorizeOrgAdmin instead.
func (s *Service) verifyOrgAdminPermission(ctx context.Context, orgID string, user *authv1.User) error {
	return common.AuthorizeOrgAdmin(ctx, orgID, user)
}
