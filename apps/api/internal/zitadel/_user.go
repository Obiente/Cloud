package zitadel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// UserService handles Zitadel User API v2 operations
// See: https://zitadel.com/docs/apis/resources/user_service_v2
type UserService struct {
	baseURL      string
	getAuthToken func() (string, error) // Function to get auth token (supports both PAT and Client Credentials)
	httpClient   *http.Client
}

// NewUserService creates a new UserService instance
func NewUserService(baseURL string, getAuthToken func() (string, error)) *UserService {
	return &UserService{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		getAuthToken: getAuthToken,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// User represents a Zitadel user
type User struct {
	UserID         string
	OrganizationID string
	Email          string
	Username       string
}

// FindByEmail searches for a user by email address
// Requires: Org User Manager or Org Owner permission
func (s *UserService) FindByEmail(email string) (*User, error) {
	url := fmt.Sprintf("%s/v2/users", s.baseURL)

	// Search query for email
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

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user search failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			UserID  string `json:"userId"`
			Details struct {
				ResourceOwner string `json:"resourceOwner"`
			} `json:"details"`
			Human struct {
				Email struct {
					Email string `json:"email"`
				} `json:"email"`
			} `json:"human"`
			Username string `json:"username"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Result) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	userData := result.Result[0]
	user := &User{
		UserID:         userData.UserID,
		OrganizationID: userData.Details.ResourceOwner,
		Username:       userData.Username,
	}

	if userData.Human.Email.Email != "" {
		user.Email = userData.Human.Email.Email
	}

	return user, nil
}
