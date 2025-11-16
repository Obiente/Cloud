package zitadel

import (
	"fmt"
)

// LoginV2 implements the Zitadel v4 Session API flow for username/password authentication
// This is the recommended approach for custom login UIs per Zitadel documentation
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
type LoginV2 struct {
	client     *Client
	sessionSvc *SessionService
	userSvc    *UserService
	oauthSvc   *OAuthService
}

// NewLoginV2 creates a new LoginV2 instance
func NewLoginV2(client *Client) *LoginV2 {
	return &LoginV2{
		client:     client,
		sessionSvc: NewSessionService(client.baseURL, client.getAuthToken),
		userSvc:    NewUserService(client.baseURL, client.getAuthToken),
		oauthSvc:   NewOAuthService(client.baseURL, client.clientID),
	}
}

// Authenticate performs the complete authentication flow:
// 1. Find user by email to determine their organization
// 2. Create a session in that organization
// 3. Update session with user credentials
// 4. Create OIDC intent
// 5. Exchange for OAuth tokens
func (l *LoginV2) Authenticate(email, password string) (*LoginResponse, error) {
	// Step 1: Find user to determine their organization
	user, err := l.userSvc.FindByEmail(email)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("User not found: %v", err),
		}, fmt.Errorf("user lookup: %w", err)
	}

	orgID := user.OrganizationID
	if orgID == "" {
		// Fallback to configured organization ID
		orgID = l.client.organizationID
	}

	if orgID == "" {
		return &LoginResponse{
			Success: false,
			Message: "Could not determine user's organization. Please configure ZITADEL_ORGANIZATION_ID.",
		}, fmt.Errorf("organization ID not found")
	}

	// Step 2: Create session with organization context
	// Client Credentials tokens require explicit organization context
	fmt.Printf("[Zitadel LoginV2] Creating session for organization: %s\n", orgID)
	session, err := l.sessionSvc.CreateSession(orgID, "", "")
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Session creation failed: %v", err),
		}, fmt.Errorf("create session: %w", err)
	}
	fmt.Printf("[Zitadel LoginV2] Session created successfully: %s\n", session.SessionID)

	// Step 3: Update session with BOTH user and password verification in one call
	// This is more efficient and may avoid membership issues
	// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password#update-session-with-password
	if err := l.sessionSvc.UpdateSessionWithUserAndPassword(session.SessionID, user.UserID, password, orgID); err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Authentication failed: %v", err),
		}, fmt.Errorf("update session: %w", err)
	}

	// Step 4: Create OIDC intent
	scopes := []string{"openid", "profile", "email", "offline_access"}
	intent, err := l.sessionSvc.CreateOIDCIntent(session.SessionID, l.client.clientID, scopes, orgID)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("OIDC intent creation failed: %v", err),
		}, fmt.Errorf("create intent: %w", err)
	}

	// Step 5: Exchange auth request ID for OAuth tokens
	tokens, err := l.oauthSvc.ExchangeAuthRequest(intent.AuthRequestID)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Token exchange failed: %v", err),
		}, fmt.Errorf("exchange tokens: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}
