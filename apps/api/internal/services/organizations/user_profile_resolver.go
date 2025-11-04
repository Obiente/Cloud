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
	httpClient *http.Client
	baseURL    string
	token      string

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

func newUserProfileResolver() *userProfileResolver {
	baseURL := strings.TrimSuffix(os.Getenv("ZITADEL_URL"), "/")
	token := strings.TrimSpace(os.Getenv("ZITADEL_MANAGEMENT_TOKEN"))
	if baseURL == "" || token == "" {
		return &userProfileResolver{cache: make(map[string]*cachedProfile)}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return &userProfileResolver{
		httpClient: client,
		baseURL:    baseURL,
		token:      token,
		cache:      make(map[string]*cachedProfile),
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/management/v1/users/%s", r.baseURL, userID), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile request failed: %s", resp.Status)
	}

	var raw managementUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
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

type managementUserResponse struct {
	User struct {
		Id                 string   `json:"id"`
		PreferredLoginName string   `json:"preferredLoginName"`
		Username           string   `json:"username"`
		LoginNames         []string `json:"loginNames"`
		Human              struct {
			Profile struct {
				FirstName   string `json:"firstName"`
				LastName    string `json:"lastName"`
				DisplayName string `json:"displayName"`
				NickName    string `json:"nickName"`
				AvatarKey   string `json:"avatarKey"`
			} `json:"profile"`
			Email struct {
				Email           string `json:"email"`
				IsEmailVerified bool   `json:"isEmailVerified"`
			} `json:"email"`
			PreferredLanguage string `json:"preferredLanguage"`
		} `json:"human"`
	} `json:"user"`
}

func (m managementUserResponse) toAuthUser() *authv1.User {
	if m.User.Id == "" {
		return nil
	}
	user := &authv1.User{Id: m.User.Id}
	email := m.User.Human.Email.Email
	if email != "" {
		user.Email = email
		user.EmailVerified = m.User.Human.Email.IsEmailVerified
	}

	switch {
	case m.User.Human.Profile.DisplayName != "":
		user.Name = m.User.Human.Profile.DisplayName
	case m.User.Human.Profile.NickName != "":
		user.Name = m.User.Human.Profile.NickName
	default:
		fullName := strings.TrimSpace(strings.Join([]string{m.User.Human.Profile.FirstName, m.User.Human.Profile.LastName}, " "))
		if fullName != "" {
			user.Name = fullName
		}
	}

	if user.Name == "" && email != "" {
		user.Name = deriveNameFromEmail(email)
	}

	if m.User.PreferredLoginName != "" {
		user.PreferredUsername = m.User.PreferredLoginName
	} else if m.User.Username != "" {
		user.PreferredUsername = m.User.Username
	} else if len(m.User.LoginNames) > 0 {
		user.PreferredUsername = m.User.LoginNames[0]
	}

	if lang := m.User.Human.PreferredLanguage; lang != "" {
		user.Locale = lang
	}

	if key := m.User.Human.Profile.AvatarKey; key != "" {
		// Zitadel avatar keys need to be resolved via CDN; for now just store key.
		user.AvatarUrl = key
	}

	return user
}
