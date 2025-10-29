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

	"connectrpc.com/connect"
)

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

// UserInfo represents the user information from Zitadel userinfo endpoint
type UserInfo struct {
	Sub               string   `json:"sub"`                // User ID
	Name              string   `json:"name"`               // Full name
	GivenName         string   `json:"given_name"`         // First name
	FamilyName        string   `json:"family_name"`        // Last name
	PreferredUsername string   `json:"preferred_username"` // Username
	Email             string   `json:"email"`              // Email
	EmailVerified     bool     `json:"email_verified"`     // Email verified status
	Locale            string   `json:"locale"`             // User locale
	Picture           string   `json:"picture"`            // Profile picture URL
	UpdatedAt         int64    `json:"updated_at"`         // Last update timestamp
	Roles             []string // Custom roles (if needed)
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
	UserInfo  *UserInfo
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
func (c *UserInfoCache) Get(token string) (*UserInfo, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.cache[token]
	if !exists {
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return cached.UserInfo, true
}

// Set stores user info in cache
func (c *UserInfoCache) Set(token string, userInfo *UserInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[token] = &CachedUserInfo{
		UserInfo:  userInfo,
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
	if os.Getenv("DISABLE_AUTH") == "true" {
		log.Println("⚠️  WARNING: Authentication is DISABLED (DISABLE_AUTH=true)")
		log.Println("⚠️  This should ONLY be used in development!")
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
			// If auth is disabled (development only), skip all checks
			if config.UserInfoCache == nil {
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
				resp.Header().Set("X-User-ID", userInfo.Sub)
				resp.Header().Set("X-User-Email", userInfo.Email)
			}

			return resp, err
		}
	}
}

// validateToken validates a token against Zitadel's userinfo endpoint
func (c *AuthConfig) validateToken(ctx context.Context, token string) (*UserInfo, error) {
	// Check cache first
	if cachedUserInfo, found := c.UserInfoCache.Get(token); found {
		log.Printf("✓ Token validated (cached) for user: %s (%s)", cachedUserInfo.Sub, cachedUserInfo.Email)
		return cachedUserInfo, nil
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
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	// Cache the result
	c.UserInfoCache.Set(token, &userInfo)

	log.Printf("✓ Token validated for user: %s (%s)", userInfo.Sub, userInfo.Email)
	return &userInfo, nil
}

// GetUserFromContext extracts user info from context
func GetUserFromContext(ctx context.Context) (*UserInfo, error) {
	userInfo, ok := ctx.Value(userInfoKey).(*UserInfo)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return userInfo, nil
}

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) (string, bool) {
	userInfo, ok := ctx.Value(userInfoKey).(*UserInfo)
	if !ok {
		return "", false
	}
	return userInfo.Sub, true
}

// AuthenticateHTTPRequest authenticates an HTTP request outside of Connect RPC
// This can be used for regular HTTP handlers that need authentication
func AuthenticateHTTPRequest(config *AuthConfig, r *http.Request) (*UserInfo, error) {
	// If auth is disabled, return nil
	if config.UserInfoCache == nil {
		return nil, nil
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
