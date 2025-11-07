package organizations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"

	"google.golang.org/protobuf/proto"
)

const userProfileCacheTTL = 5 * time.Minute

type cachedProfile struct {
	user      *authv1.User
	expiresAt time.Time
}

type userProfileResolver struct {
	httpClient    *http.Client
	baseURL       string
	token         string
	organizationID string

	mu    sync.RWMutex
	cache map[string]*cachedProfile
}

var resolverOnce sync.Once
var resolverInstance *userProfileResolver

func getUserProfileResolver() *userProfileResolver {
	resolverOnce.Do(func() {
		resolverInstance = newUserProfileResolver()
	})
	return resolverInstance
}

// GetUserProfileResolver returns the singleton user profile resolver instance
func GetUserProfileResolver() *userProfileResolver {
	return getUserProfileResolver()
}

// IsConfigured returns true if the resolver is properly configured with baseURL and token
func (r *userProfileResolver) IsConfigured() bool {
	return r != nil && r.baseURL != "" && r.token != ""
}

func newUserProfileResolver() *userProfileResolver {
	baseURL := strings.TrimSuffix(os.Getenv("ZITADEL_URL"), "/")
	token := strings.TrimSpace(os.Getenv("ZITADEL_MANAGEMENT_TOKEN"))
	organizationID := strings.TrimSpace(os.Getenv("ZITADEL_ORGANIZATION_ID"))
	if baseURL == "" || token == "" {
		return &userProfileResolver{cache: make(map[string]*cachedProfile)}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return &userProfileResolver{
		httpClient:     client,
		baseURL:        baseURL,
		token:          token,
		organizationID: organizationID,
		cache:          make(map[string]*cachedProfile),
	}
}

func (r *userProfileResolver) Resolve(ctx context.Context, userID string) (*authv1.User, error) {
	if r == nil || userID == "" || r.token == "" || r.baseURL == "" {
		return nil, fmt.Errorf("profile resolver not configured")
	}

	r.mu.RLock()
	if cached, ok := r.cache[userID]; ok && time.Now().Before(cached.expiresAt) {
		r.mu.RUnlock()
		return cloneUser(cached.user), nil
	}
	r.mu.RUnlock()

	// Use Zitadel Management API v2 for user lookup
	// The v2 API endpoint is /v2/users/{userId}
	url := fmt.Sprintf("%s/v2/users/%s", r.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json")
	
	// Add organization context header if available
	// This helps Zitadel route the request to the correct organization context
	if r.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", r.organizationID)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	defer resp.Body.Close()

	// Read response body once
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile request failed: %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Decode as direct user object (Zitadel v2 API format)
	var raw managementUserResponse
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("decode profile: %w (response: %s)", err, string(bodyBytes))
	}

	user := raw.toAuthUser()
	if user == nil {
		return nil, fmt.Errorf("profile decode produced empty user")
	}

	r.mu.Lock()
	r.cache[userID] = &cachedProfile{user: user, expiresAt: time.Now().Add(userProfileCacheTTL)}
	r.mu.Unlock()

	return cloneUser(user), nil
}

// UpdateProfile updates a user's profile in Zitadel
func (r *userProfileResolver) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*authv1.User, error) {
	if r == nil || userID == "" || r.token == "" || r.baseURL == "" {
		return nil, fmt.Errorf("profile resolver not configured")
	}

	// Build update request body for v2 API
	// The v2 API requires specifying the user type (human) in the request body
	updateBody := make(map[string]interface{})
	human := make(map[string]interface{})
	
	if profile, ok := updates["profile"].(map[string]interface{}); ok {
		human["profile"] = profile
	}
	if preferredLanguage, ok := updates["preferredLanguage"].(string); ok {
		human["preferredLanguage"] = preferredLanguage
	}
	
	if len(human) > 0 {
		updateBody["human"] = human
	}

	// Serialize request body
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return nil, fmt.Errorf("marshal update: %w", err)
	}

	// Create PATCH request to v2 API endpoint
	// The v2 API uses /v2/users/:userId (without /management prefix)
	// The request body must wrap human updates in a "human" object
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, 
		fmt.Sprintf("%s/v2/users/%s", r.baseURL, userID),
		strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if r.organizationID != "" {
		req.Header.Set("x-zitadel-orgid", r.organizationID)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("profile update failed: %s, body: %s", resp.Status, string(body))
	}

	// Invalidate cache
	r.mu.Lock()
	delete(r.cache, userID)
	r.mu.Unlock()

	// Fetch updated profile
	return r.Resolve(ctx, userID)
}

func cloneUser(in *authv1.User) *authv1.User {
	if in == nil {
		return nil
	}
	clone := proto.Clone(in)
	if user, ok := clone.(*authv1.User); ok {
		return user
	}
	return in
}

// managementUserResponse handles Zitadel v2 API GetUserByID response
// The actual response structure wraps the user in a "user" field
// Field names use camelCase: givenName/familyName (not firstName/lastName), isVerified (not isEmailVerified)
type managementUserResponse struct {
	Details struct {
		Sequence      string `json:"sequence"`
		ChangeDate    string `json:"changeDate"`
		ResourceOwner string `json:"resourceOwner"`
		CreationDate  string `json:"creationDate"`
	} `json:"details"`
	User struct {
		UserID             string   `json:"userId"`
		Details            struct {
			Sequence      string `json:"sequence"`
			ChangeDate    string `json:"changeDate"`
			ResourceOwner string `json:"resourceOwner"`
			CreationDate  string `json:"creationDate"`
		} `json:"details"`
		State              string   `json:"state"`
		Username           string   `json:"username"`
		LoginNames         []string `json:"loginNames"`
		PreferredLoginName string   `json:"preferredLoginName"`
		Human              struct {
			Profile struct {
				GivenName         string `json:"givenName"`         // Note: givenName, not firstName
				FamilyName        string `json:"familyName"`         // Note: familyName, not lastName
				DisplayName       string `json:"displayName"`
				NickName          string `json:"nickName"`
				PreferredLanguage string `json:"preferredLanguage"`
				Gender            string `json:"gender"`
			} `json:"profile"`
			Email struct {
				Email      string `json:"email"`
				IsVerified bool   `json:"isVerified"` // Note: isVerified, not isEmailVerified
			} `json:"email"`
			Phone struct {
			} `json:"phone"`
		} `json:"human"`
	} `json:"user"`
}

func (m managementUserResponse) toAuthUser() *authv1.User {
	// The response is always wrapped in a "user" field
	if m.User.UserID == "" {
		return nil
	}

	userID := m.User.UserID
	user := &authv1.User{Id: userID}
	
	// Extract email
	email := m.User.Human.Email.Email
	if email != "" {
		user.Email = email
		user.EmailVerified = m.User.Human.Email.IsVerified // Note: isVerified, not isEmailVerified
	}

	// Determine display name (prefer displayName, then nickName, then givenName + familyName)
	switch {
	case m.User.Human.Profile.DisplayName != "":
		user.Name = m.User.Human.Profile.DisplayName
	case m.User.Human.Profile.NickName != "":
		user.Name = m.User.Human.Profile.NickName
	default:
		// Use givenName and familyName (not firstName/lastName)
		fullName := strings.TrimSpace(strings.Join([]string{m.User.Human.Profile.GivenName, m.User.Human.Profile.FamilyName}, " "))
		if fullName != "" {
			user.Name = fullName
		}
	}

	// Fallback to email-derived name if no name found
	if user.Name == "" && email != "" {
		user.Name = deriveNameFromEmail(email)
	}

	// Set preferred username
	if m.User.PreferredLoginName != "" {
		user.PreferredUsername = m.User.PreferredLoginName
	} else if m.User.Username != "" {
		user.PreferredUsername = m.User.Username
	} else if len(m.User.LoginNames) > 0 {
		user.PreferredUsername = m.User.LoginNames[0]
	}

	// Set locale
	if lang := m.User.Human.Profile.PreferredLanguage; lang != "" {
		user.Locale = lang
	}

	return user
}
