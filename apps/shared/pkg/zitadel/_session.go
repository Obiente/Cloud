package zitadel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SessionService handles Zitadel Session API v2 operations
// See: https://zitadel.com/docs/apis/resources/session_service_v2
type SessionService struct {
	baseURL      string
	getAuthToken func() (string, error) // Function to get auth token (supports both PAT and Client Credentials)
	httpClient   *http.Client
}

// NewSessionService creates a new SessionService instance
func NewSessionService(baseURL string, getAuthToken func() (string, error)) *SessionService {
	return &SessionService{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		getAuthToken: getAuthToken,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CreateSessionRequest represents a session creation request
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-create-session
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password#create-session-with-user-check
type CreateSessionRequest struct {
	OrganizationID string         `json:"organizationId,omitempty"`
	Checks         *SessionChecks `json:"checks,omitempty"`
}

// SessionChecks represents checks to perform during session creation/update
type SessionChecks struct {
	User *UserCheck `json:"user,omitempty"`
}

// UserCheck represents a user verification check
type UserCheck struct {
	LoginName string `json:"loginName,omitempty"`
	UserID    string `json:"userId,omitempty"`
}

// CreateSessionResponse represents a session creation response
type CreateSessionResponse struct {
	SessionID    string `json:"sessionId"`
	SessionToken string `json:"sessionToken"`
}

// CreateSession creates a new session with optional user check
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-create-session
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password#create-session-with-user-check
// Requires: The service account must be a member of the organization with Org User Manager or Org Owner role
// userID is optional - if provided, uses userId for user check; otherwise uses loginName
func (s *SessionService) CreateSession(orgID string, loginName string, userID string) (*CreateSessionResponse, error) {
	url := fmt.Sprintf("%s/v2/sessions", s.baseURL)

	reqBody := CreateSessionRequest{}
	if orgID != "" {
		reqBody.OrganizationID = orgID
	}
	if userID != "" {
		// Prefer userId if available (more reliable)
		reqBody.Checks = &SessionChecks{
			User: &UserCheck{
				UserID: userID,
			},
		}
	} else if loginName != "" {
		// Fallback to loginName
		reqBody.Checks = &SessionChecks{
			User: &UserCheck{
				LoginName: loginName,
			},
		}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	fmt.Printf("[Zitadel Session] CreateSession - Organization ID: %s\n", orgID)
	fmt.Printf("[Zitadel Session] CreateSession - Request body: %s\n", string(bodyBytes))

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	authToken, err := s.getAuthToken()
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
		fmt.Printf("[Zitadel Session] Set x-zitadel-orgid header: %s\n", orgID)
	} else {
		fmt.Printf("[Zitadel Session] WARNING: No organization ID provided in CreateSession\n")
	}
	fmt.Printf("[Zitadel Session] Request URL: POST %s\n", url)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session creation failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.SessionID == "" {
		return nil, fmt.Errorf("session ID is empty in response")
	}

	return &result, nil
}

// UpdateSessionRequest represents a session update request
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-update-session
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password#update-session-with-password
type UpdateSessionRequest struct {
	Checks struct {
		Password struct {
			Password string `json:"password"`
		} `json:"password"`
	} `json:"checks"`
}

// UpdateSessionWithPassword updates a session with password verification
// The user should already be verified in the session from the create step
// See: https://zitadel.com/docs/guides/integrate/login-ui/username-password#update-session-with-password
func (s *SessionService) UpdateSessionWithPassword(sessionID, password, orgID string) error {
	url := fmt.Sprintf("%s/v2/sessions/%s", s.baseURL, sessionID)

	var reqBody UpdateSessionRequest
	reqBody.Checks.Password.Password = password

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	authToken, err := s.getAuthToken()
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("session update failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateSessionWithUserAndPassword updates a session with both user and password verification
// This is useful when creating a session without user check initially
func (s *SessionService) UpdateSessionWithUserAndPassword(sessionID, userID, password, orgID string) error {
	url := fmt.Sprintf("%s/v2/sessions/%s", s.baseURL, sessionID)

	reqBody := map[string]interface{}{
		"checks": map[string]interface{}{
			"user": map[string]interface{}{
				"userId": userID,
			},
			"password": map[string]interface{}{
				"password": password,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	authToken, err := s.getAuthToken()
	if err != nil {
		return fmt.Errorf("get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("session update failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateOIDCIntentRequest represents an OIDC intent creation request
type CreateOIDCIntentRequest struct {
	ClientID string   `json:"clientId"`
	Scope    []string `json:"scope"`
}

// CreateOIDCIntentResponse represents an OIDC intent creation response
type CreateOIDCIntentResponse struct {
	AuthRequestID string `json:"authRequestId"`
}

// CreateOIDCIntent creates an OIDC intent for a session
// See: https://zitadel.com/docs/apis/resources/session_service_v2/session-service-create-oidc-intent
func (s *SessionService) CreateOIDCIntent(sessionID, clientID string, scopes []string, orgID string) (*CreateOIDCIntentResponse, error) {
	url := fmt.Sprintf("%s/v2/sessions/%s/intents/oidc", s.baseURL, sessionID)

	reqBody := CreateOIDCIntentRequest{
		ClientID: clientID,
		Scope:    scopes,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	authToken, err := s.getAuthToken()
	if err != nil {
		return nil, fmt.Errorf("get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("intent creation failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result CreateOIDCIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
