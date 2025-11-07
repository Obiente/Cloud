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

// ClientCredentialsService handles OAuth2 Client Credentials Grant authentication
// This authenticates as the OAuth application itself, not as a user
// See: https://zitadel.com/docs/guides/integrate/login/oidc/oauth-recommended-flows#client-credentials-grant
type ClientCredentialsService struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	tokenCache   *clientCredentialsToken
	tokenExpiry  time.Time
}

type clientCredentialsToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	ExpiresAt   time.Time
}

// NewClientCredentialsService creates a new Client Credentials service
func NewClientCredentialsService() *ClientCredentialsService {
	zitadelURL := os.Getenv("ZITADEL_URL")
	if zitadelURL == "" {
		zitadelURL = "https://auth.obiente.cloud"
	}
	zitadelURL = strings.TrimSuffix(zitadelURL, "/")

	clientID := os.Getenv("ZITADEL_CLIENT_ID")
	clientSecret := strings.TrimSpace(os.Getenv("ZITADEL_CLIENT_SECRET"))

	return &ClientCredentialsService{
		baseURL:      zitadelURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAccessToken obtains an access token using Client Credentials Grant
// Tokens are cached until expiry
func (s *ClientCredentialsService) GetAccessToken() (string, error) {
	// Check if we have a valid cached token
	if s.tokenCache != nil && time.Now().Before(s.tokenExpiry) {
		return s.tokenCache.AccessToken, nil
	}

	if s.clientID == "" {
		return "", fmt.Errorf("ZITADEL_CLIENT_ID not configured")
	}
	if s.clientSecret == "" {
		return "", fmt.Errorf("ZITADEL_CLIENT_SECRET not configured")
	}

	tokenURL := fmt.Sprintf("%s/oauth/v2/token", s.baseURL)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	
	// Request scopes for management API access
	// Required scopes for Session API and User API:
	// - openid: Basic OpenID Connect scope
	// - profile: User profile information
	// - urn:zitadel:iam:org:project:id:zitadel:aud: Management API access
	// See: https://zitadel.com/docs/guides/integrate/service-users/client-credentials
	scopes := "openid profile urn:zitadel:iam:org:project:id:zitadel:aud"
	
	// Optional: Add email scope if needed for user info
	// scopes += " email"
	
	// Note: Organization-specific scopes are not typically needed for Client Credentials
	// The service user's organization membership is checked via roles/permissions, not scopes
	fmt.Printf("[Zitadel ClientCredentials] Requesting scopes: %s\n", scopes)
	data.Set("scope", scopes)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Use HTTP Basic Authentication for Client Credentials Grant
	// See: https://zitadel.com/docs/guides/integrate/login/oidc/oauth-recommended-flows#client-credentials-grant
	req.SetBasicAuth(s.clientID, s.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	fmt.Printf("[Zitadel ClientCredentials] Using HTTP Basic Auth with client_id: %s\n", s.clientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp clientCredentialsToken
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access token is empty in response")
	}

	// Cache the token
	s.tokenCache = &tokenResp
	// Set expiry to 90% of the actual expiry time for safety
	expiryDuration := time.Duration(tokenResp.ExpiresIn) * time.Second * 90 / 100
	s.tokenExpiry = time.Now().Add(expiryDuration)

	return tokenResp.AccessToken, nil
}

// IsConfigured checks if client credentials are configured
func (s *ClientCredentialsService) IsConfigured() bool {
	return s.clientID != "" && s.clientSecret != ""
}

