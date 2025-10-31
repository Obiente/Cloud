package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// isAuthDisabled checks if authentication is disabled via environment variable
func isAuthDisabled() bool {
	return os.Getenv("DISABLE_AUTH") == "true"
}

// getMockDevUser returns a mock development user with appropriate permissions
func getMockDevUser() *authv1.User {
	return &authv1.User{
		Id:                "mem-development",
		Email:             "dev@obiente.local",
		Name:              "Development User",
		GivenName:         "Development",
		FamilyName:        "User",
		PreferredUsername: "dev",
		EmailVerified:     true,
		Locale:            "en",
		AvatarUrl:         "",
		Roles:             []string{RoleAdmin, RoleOwner}, // Give full permissions for development
		UpdatedAt:         timestamppb.Now(),
	}
}

// Constants for auth validation
const (
	AuthorizationHeader   = "Authorization"
	BearerPrefix          = "Bearer "
	UserInfoCacheDuration = 5 * time.Minute // Cache userinfo for 5 minutes
)

// Common errors
var (
	ErrNoToken        = errors.New("no authorization token provided")
	ErrInvalidToken   = errors.New("invalid authorization token")
	ErrUserInfoFailed = errors.New("failed to fetch user info")
)

// Use protobuf-defined user type for userinfo
// We will decode Zitadel userinfo JSON into an internal struct and map to authv1.User
type zitadelUserInfo struct {
    Sub               string `json:"sub"`
    Name              string `json:"name"`
    GivenName         string `json:"given_name"`
    FamilyName        string `json:"family_name"`
    PreferredUsername string `json:"preferred_username"`
    Email             string `json:"email"`
    EmailVerified     bool   `json:"email_verified"`
    Locale            string `json:"locale"`
    Picture           string `json:"picture"`
    UpdatedAt         int64  `json:"updated_at"`
}

// contextKey is a private type for context keys
type contextKey int

const userInfoKey contextKey = 0

// UserInfoCache manages caching of validated tokens
type UserInfoCache struct {
    cache map[string]*CachedUserInfo
	mutex sync.RWMutex
}

// CachedUserInfo stores user info with expiration
type CachedUserInfo struct {
    User      *authv1.User
    ExpiresAt time.Time
}

// NewUserInfoCache creates a new user info cache
func NewUserInfoCache() *UserInfoCache {
	cache := &UserInfoCache{
		cache: make(map[string]*CachedUserInfo),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves cached user info if valid
func (c *UserInfoCache) Get(token string) (*authv1.User, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[token]
	if !exists {
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

    return cached.User, true
}

// Set stores user info in cache
func (c *UserInfoCache) Set(token string, user *authv1.User) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[token] = &CachedUserInfo{
        User:      user,
		ExpiresAt: time.Now().Add(UserInfoCacheDuration),
	}
}

// cleanup removes expired entries
func (c *UserInfoCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for token, cached := range c.cache {
			if now.After(cached.ExpiresAt) {
				delete(c.cache, token)
			}
		}
		c.mutex.Unlock()
	}
}

// AuthConfig holds configuration for authentication
type AuthConfig struct {
	UserInfoCache  *UserInfoCache
	UserInfoURL    string
	HTTPClient     *http.Client
	SkipPaths      []string
	ExposeUserInfo bool // Whether to expose user info in response headers
}

// NewAuthConfig creates a new auth config with default values
func NewAuthConfig() *AuthConfig {
	// Check if auth is disabled for development
	if isAuthDisabled() {
		log.Println("⚠️  WARNING: Authentication is DISABLED (DISABLE_AUTH=true)")
		log.Println("⚠️  This should ONLY be used in development!")
		log.Println("⚠️  Using mock development user: mem-development")
		return &AuthConfig{
			UserInfoCache:  nil,
			UserInfoURL:    "",
			HTTPClient:     nil,
			SkipPaths:      []string{"/"},  // Skip all paths
			ExposeUserInfo: false,
		}
	}

	// Get Zitadel URL from environment
	zitadelURL := os.Getenv("ZITADEL_URL")
	if zitadelURL == "" {
		zitadelURL = "https://auth.obiente.cloud" // Default fallback
		log.Println("⚠️  Warning: ZITADEL_URL not set, using default")
	}

	// Build userinfo URL
	userInfoURL := strings.TrimSuffix(zitadelURL, "/") + "/oidc/v1/userinfo"

	// Create HTTP client with optional TLS skip verification for development
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	if os.Getenv("SKIP_TLS_VERIFY") == "true" {
		log.Println("⚠️  Warning: TLS verification disabled (SKIP_TLS_VERIFY=true)")
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	log.Printf("🔐 Auth Configuration:")
	log.Printf("  Zitadel URL: %s", zitadelURL)
	log.Printf("  UserInfo URL: %s", userInfoURL)
	log.Printf("  Skip TLS Verify: %s", os.Getenv("SKIP_TLS_VERIFY"))

	return &AuthConfig{
		UserInfoCache:  NewUserInfoCache(),
		UserInfoURL:    userInfoURL,
		HTTPClient:     httpClient,
		SkipPaths:      []string{"/health", "/metrics", "/.well-known"},
		ExposeUserInfo: false,
	}
}

// MiddlewareInterceptor creates a Connect interceptor for token authentication
func MiddlewareInterceptor(config *AuthConfig) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// If auth is disabled (development only), inject mock user and continue
			if config.UserInfoCache == nil && isAuthDisabled() {
				mockUser := getMockDevUser()
				ctx = context.WithValue(ctx, userInfoKey, mockUser)
				return next(ctx, req)
			}
			
			// If auth config is nil but DISABLE_AUTH is not set, this is an error state
			if config.UserInfoCache == nil {
				log.Println("⚠️  WARNING: Auth config is nil but DISABLE_AUTH is not set. This should not happen.")
				return next(ctx, req)
			}

			// Skip auth for specified paths
			for _, path := range config.SkipPaths {
				if strings.HasPrefix(req.Spec().Procedure, path) {
					return next(ctx, req)
				}
			}

			// Skip auth for authentication-related endpoints
			if strings.Contains(req.Spec().Procedure, "AuthService") {
				// Skip most auth service methods, but not GetCurrentUser
				if !strings.Contains(req.Spec().Procedure, "GetCurrentUser") {
					return next(ctx, req)
				}
			}

			// Extract token from Authorization header
			authHeader := req.Header().Get(AuthorizationHeader)
			if authHeader == "" || !strings.HasPrefix(authHeader, BearerPrefix) {
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
			}

			token := strings.TrimPrefix(authHeader, BearerPrefix)
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
			}

			// Validate token against userinfo endpoint
			userInfo, err := config.validateToken(ctx, token)
			if err != nil {
				log.Printf("Token validation failed: %v", err)
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("error validating token: %w", err))
			}

			// Add user info to context
			ctx = context.WithValue(ctx, userInfoKey, userInfo)

			// Call next handler
			resp, err := next(ctx, req)

            // Optionally add user info to response headers (for debugging)
			if config.ExposeUserInfo && resp != nil && err == nil {
                resp.Header().Set("X-User-ID", userInfo.Id)
                resp.Header().Set("X-User-Email", userInfo.Email)
			}

			return resp, err
		}
	}
}

// validateToken validates a token against Zitadel's userinfo endpoint
func (c *AuthConfig) validateToken(ctx context.Context, token string) (*authv1.User, error) {
	// Check cache first
    if cachedUser, found := c.UserInfoCache.Get(token); found {
        log.Printf("✓ Token validated (cached) for user: %s (%s)", cachedUser.Id, cachedUser.Email)
        return cachedUser, nil
	}

	// Create request to userinfo endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", c.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	// Add bearer token
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfoFailed, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("UserInfo request failed: status=%d, body=%s", resp.StatusCode, string(body))
		
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("%w: status %d", ErrUserInfoFailed, resp.StatusCode)
	}

	// Parse response
    var zui zitadelUserInfo
    if err := json.NewDecoder(resp.Body).Decode(&zui); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

    // Map to proto user
    user := &authv1.User{
        Id:                zui.Sub,
        Email:             zui.Email,
        Name:              zui.Name,
        AvatarUrl:         zui.Picture,
        GivenName:         zui.GivenName,
        FamilyName:        zui.FamilyName,
        PreferredUsername: zui.PreferredUsername,
        EmailVerified:     zui.EmailVerified,
        Locale:            zui.Locale,
    }
    if zui.UpdatedAt > 0 {
        user.UpdatedAt = timestamppb.New(time.Unix(zui.UpdatedAt, 0))
    }

    // Cache the result
    c.UserInfoCache.Set(token, user)

    log.Printf("✓ Token validated for user: %s (%s)", user.Id, user.Email)
    return user, nil
}

// GetUserFromContext extracts user info from context
// When DISABLE_AUTH=true, returns a mock dev user if no user is in context
func GetUserFromContext(ctx context.Context) (*authv1.User, error) {
    userInfo, ok := ctx.Value(userInfoKey).(*authv1.User)
	if !ok {
		// If auth is disabled, return mock user as fallback
		if isAuthDisabled() {
			return getMockDevUser(), nil
		}
		return nil, errors.New("user not found in context")
	}
	return userInfo, nil
}

// UserIDFromContext extracts user ID from context
// When DISABLE_AUTH=true, returns mock dev user ID if no user is in context
func UserIDFromContext(ctx context.Context) (string, bool) {
    userInfo, ok := ctx.Value(userInfoKey).(*authv1.User)
	if !ok {
		// If auth is disabled, return mock user ID as fallback
		if isAuthDisabled() {
			return getMockDevUser().Id, true
		}
		return "", false
	}
    return userInfo.Id, true
}

// AuthenticateHTTPRequest authenticates an HTTP request outside of Connect RPC
// This can be used for regular HTTP handlers that need authentication
// When DISABLE_AUTH=true, returns a mock dev user
func AuthenticateHTTPRequest(config *AuthConfig, r *http.Request) (*authv1.User, error) {
	// If auth is disabled, return mock dev user
	if config.UserInfoCache == nil && isAuthDisabled() {
		return getMockDevUser(), nil
	}
	
	// If auth config is nil but DISABLE_AUTH is not set, return error
	if config.UserInfoCache == nil {
		return nil, errors.New("authentication not configured")
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get(AuthorizationHeader)
	if authHeader == "" || !strings.HasPrefix(authHeader, BearerPrefix) {
		return nil, ErrNoToken
	}

	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return nil, ErrNoToken
	}

	// Validate token
	return config.validateToken(r.Context(), token)
}
