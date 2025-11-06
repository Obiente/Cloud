package zitadel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client handles Zitadel API v2 interactions
//
// Required Zitadel Service Account Organization Permissions:
//   - Org User Manager: Required for searching users by email and managing user sessions
//     This permission allows:
//   - Searching users via POST /v2/users (User Service v2)
//   - Creating sessions via /v2/sessions
//
// Alternative permissions that may work:
// - Org Owner: Full organization access (includes user management)
// - Org Admin Impersonator: Can impersonate users (may include session creation)
//
// Required Scopes for Management Token:
// - urn:zitadel:iam:org:project:id:zitadel:aud (Management API access)
//
// To configure in Zitadel Console:
// 1. Go to Organizations → Select your organization
// 2. Go to Members → Find your Service Account
// 3. Click "Grant" or "Edit" to assign organization permissions
// 4. Grant "Org User Manager" permission (or "Org Owner" for full access)
// 5. Go to Projects → Select your project
// 6. Generate Personal Access Token with scope: urn:zitadel:iam:org:project:id:zitadel:aud
type Client struct {
	baseURL         string
	clientID        string
	managementToken string
	organizationID  string // Optional: Organization ID for API requests
	httpClient      *http.Client
}

// NewClient creates a new Zitadel API v2 client
func NewClient() *Client {
	zitadelURL := os.Getenv("ZITADEL_URL")
	if zitadelURL == "" {
		zitadelURL = "https://auth.obiente.cloud"
	}
	zitadelURL = strings.TrimSuffix(zitadelURL, "/")

	clientID := os.Getenv("ZITADEL_CLIENT_ID")
	managementToken := strings.TrimSpace(os.Getenv("ZITADEL_MANAGEMENT_TOKEN"))
	organizationID := strings.TrimSpace(os.Getenv("ZITADEL_ORGANIZATION_ID"))

	return &Client{
		baseURL:         zitadelURL,
		clientID:        clientID,
		managementToken: managementToken,
		organizationID:  organizationID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LoginResponse represents the response from a login operation
type LoginResponse struct {
	Success      bool
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32
	Message      string
}

// Login authenticates a user with email and password using Zitadel Session API v2
// Implements the recommended Session API flow per Zitadel documentation
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
//
// Note: Zitadel does NOT support ROPC grant (deprecated in OAuth 2.1).
// The Session API flow is the recommended approach for custom login UIs.
func (c *Client) Login(email, password string) (*LoginResponse, error) {
	// Use Session API flow (Zitadel's recommended approach)
	// This requires a service account with Org User Manager permission
	if c.managementToken == "" {
		return &LoginResponse{
			Success: false,
			Message: "Management token required for Session API authentication. Configure ZITADEL_MANAGEMENT_TOKEN.",
		}, fmt.Errorf("management token not configured")
	}

	return c.authenticateWithSessionAPI(email, password)
}

// getOAuthTokensForUser uses OAuth2 Resource Owner Password Credentials grant
func (c *Client) getOAuthTokensForUser(email, password string) (*LoginResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", c.baseURL)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", email)
	data.Set("password", password)
	data.Set("client_id", c.clientID)
	data.Set("scope", "openid profile email offline_access")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed: %s, body: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int32(tokenResp.ExpiresIn),
	}, nil
}

// authenticateWithSessionAPI implements the Session API flow for username/password authentication
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
// Requires: Org User Manager (or Org Owner) organization permission
func (c *Client) authenticateWithSessionAPI(email, password string) (*LoginResponse, error) {
	// Step 1: Create a new session
	sessionID, err := c.createSession(email, password)
	if err != nil {
		return nil, fmt.Errorf("session creation: %w", err)
	}

	// Step 2: Complete the session and exchange for OAuth tokens
	return c.completeSessionAndGetTokens(sessionID)
}

// findUserByEmail searches for a user by email using Zitadel API v2
// Requires: Org User Manager (or Org Owner) organization permission
// Endpoint: POST /v2/users (see https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users)
func (c *Client) findUserByEmail(email string) (string, error) {
	// Zitadel User Service v2 uses POST /v2/users endpoint for searching
	searchURL := fmt.Sprintf("%s/v2/users", c.baseURL)

	// Request body structure for Zitadel User Service v2 search
	// See: https://zitadel.com/docs/apis/resources/user_service_v2/user-service-list-users
	searchBody := map[string]interface{}{
		"queries": []map[string]interface{}{
			{
				"emailQuery": map[string]interface{}{
					"emailAddress": email,
					"method":       "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE",
				},
			},
		},
		"limit": 1,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return "", fmt.Errorf("marshal search: %w", err)
	}

	req, err := http.NewRequest("POST", searchURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Add organization context header if specified
	if c.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", c.organizationID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Provide more helpful error message for 404
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("search endpoint not found (404): %s. Verify the endpoint path and ensure your service account has Org User Manager permission. Response: %s", searchURL, string(body))
		}
		return "", fmt.Errorf("search failed: %s, body: %s", resp.Status, string(body))
	}

	var searchResult struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(searchResult.Result) == 0 {
		return "", fmt.Errorf("user not found")
	}

	return searchResult.Result[0].ID, nil
}

// createSession creates a session using Zitadel Session API v2
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-create-session
// Requires: Org User Manager (or Org Owner) organization permission
func (c *Client) createSession(email, password string) (string, error) {
	// Step 1: Create an empty session first
	sessionURL := fmt.Sprintf("%s/v2/sessions", c.baseURL)

	// Create empty session (no checks initially)
	sessionBody := map[string]interface{}{}

	bodyBytes, err := json.Marshal(sessionBody)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	req, err := http.NewRequest("POST", sessionURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Add organization context header if specified
	if c.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", c.organizationID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("session creation failed (404): %s. Ensure your service account is a member of the organization with 'Org User Manager' or 'Org Owner' permission. Response: %s", resp.Status, string(body))
		}
		return "", fmt.Errorf("session creation failed: %s, body: %s", resp.Status, string(body))
	}

	var sessionResult struct {
		SessionID    string `json:"sessionId"`
		SessionToken string `json:"sessionToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResult); err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}

	if sessionResult.SessionID == "" {
		return "", fmt.Errorf("no session ID in response")
	}

	sessionID := sessionResult.SessionID

	// Step 2: Update session with user credentials check
	updateURL := fmt.Sprintf("%s/v2/sessions/%s", c.baseURL, sessionID)

	updateBody := map[string]interface{}{
		"checks": map[string]interface{}{
			"user": map[string]interface{}{
				"loginName": email,
				"password": map[string]interface{}{
					"password": password,
				},
			},
		},
		"challenges": []string{"PASSWORD"},
	}

	updateBytes, err := json.Marshal(updateBody)
	if err != nil {
		return "", fmt.Errorf("marshal update: %w", err)
	}

	updateReq, err := http.NewRequest("PUT", updateURL, strings.NewReader(string(updateBytes)))
	if err != nil {
		return "", fmt.Errorf("create update request: %w", err)
	}
	updateReq.Header.Set("Authorization", "Bearer "+c.managementToken)
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := c.httpClient.Do(updateReq)
	if err != nil {
		return "", fmt.Errorf("session update request: %w", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		return "", fmt.Errorf("session update failed: %s, body: %s", updateResp.Status, string(body))
	}

	return sessionID, nil
}

// completeSessionAndGetTokens completes the session and exchanges it for OAuth tokens
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password
func (c *Client) completeSessionAndGetTokens(sessionID string) (*LoginResponse, error) {
	// Step 1: Set intent to authenticate and get session token
	intentURL := fmt.Sprintf("%s/v2/sessions/%s/intents/oidc", c.baseURL, sessionID)

	intentBody := map[string]interface{}{
		"clientId": c.clientID,
		"scope":    []string{"openid", "profile", "email", "offline_access"},
	}

	bodyBytes, err := json.Marshal(intentBody)
	if err != nil {
		return nil, fmt.Errorf("marshal intent: %w", err)
	}

	req, err := http.NewRequest("POST", intentURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create intent request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementToken)
	req.Header.Set("Content-Type", "application/json")
	// Add organization context header if specified
	if c.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", c.organizationID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("intent request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("intent creation failed: %s, body: %s", resp.Status, string(body))
	}

	var intentResult struct {
		AuthRequestID string `json:"authRequestId"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&intentResult); err != nil {
		return nil, fmt.Errorf("decode intent: %w", err)
	}

	if intentResult.AuthRequestID == "" {
		return nil, fmt.Errorf("no auth request ID in response")
	}

	// Step 2: Exchange auth request for tokens
	return c.exchangeAuthRequestForTokens(intentResult.AuthRequestID)
}

// exchangeAuthRequestForTokens exchanges an auth request ID for OAuth tokens
func (c *Client) exchangeAuthRequestForTokens(authRequestID string) (*LoginResponse, error) {
	// Use the OAuth token endpoint with the auth request ID
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", c.baseURL)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authRequestID)
	data.Set("client_id", c.clientID)
	data.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob") // Out-of-band redirect for service accounts

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s, body: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &LoginResponse{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int32(tokenResp.ExpiresIn),
	}, nil
}
