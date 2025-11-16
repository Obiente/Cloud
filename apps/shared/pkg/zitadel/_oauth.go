package zitadel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// OAuthService handles OAuth token operations
type OAuthService struct {
	baseURL    string
	clientID   string
	httpClient *http.Client
}

// NewOAuthService creates a new OAuthService instance
func NewOAuthService(baseURL string, clientID string) *OAuthService {
	return &OAuthService{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// TokenResponse represents an OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// ExchangeAuthRequest exchanges an auth request ID for OAuth tokens
// This is used after creating an OIDC intent in a session
func (s *OAuthService) ExchangeAuthRequest(authRequestID string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth/v2/token", s.baseURL)
	
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authRequestID)
	data.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob") // Out-of-band for service accounts
	if s.clientID != "" {
		data.Set("client_id", s.clientID) // OAuth client ID is required for token exchange
	}
	
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &tokenResp, nil
}

