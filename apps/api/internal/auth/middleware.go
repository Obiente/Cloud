package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	authv1 "api/gen/proto/obiente/cloud/auth/v1"
	"api/internal/logger"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// isAuthDisabled checks if authentication is disabled via environment variable
func isAuthDisabled() bool {
	return os.Getenv("DISABLE_AUTH") == "true"
}

// getMockDevUser returns a mock development user with appropriate permissions
func getMockDevUser() *authv1.User {
	user := &authv1.User{
		Id:                "mem-development",
		Email:             "dev@obiente.local",
		Name:              "Development User",
		GivenName:         "Development",
		FamilyName:        "User",
		PreferredUsername: "dev",
		EmailVerified:     true,
		Locale:            "en",
		AvatarUrl:         "",
		Roles:             []string{},
		UpdatedAt:         timestamppb.Now(),
	}

	superAdmins := loadSuperAdminEmails()
	lowerEmail := strings.ToLower(user.Email)
	if _, ok := superAdmins[lowerEmail]; ok {
		ensureRole(user, RoleSuperAdmin)
		logger.Debug("Mock dev user promoted to superadmin via SUPERADMIN_EMAILS")
	}

	ensureRole(user, RoleAdmin)
	ensureRole(user, RoleOwner)

	return user
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
	SuperAdmins    map[string]struct{}
}

// NewAuthConfig creates a new auth config with default values
func NewAuthConfig() *AuthConfig {
	superAdmins := loadSuperAdminEmails()

	// Check if auth is disabled for development
	if isAuthDisabled() {
		logger.Warn("⚠️  WARNING: Authentication is DISABLED (DISABLE_AUTH=true)")
		logger.Warn("⚠️  This should ONLY be used in development!")
		logger.Warn("⚠️  Using mock development user: mem-development")
		return &AuthConfig{
			UserInfoCache:  nil,
			UserInfoURL:    "",
			HTTPClient:     nil,
			SkipPaths:      []string{"/"}, // Skip all paths
			ExposeUserInfo: false,
			SuperAdmins:    superAdmins,
		}
	}

	// Get Zitadel URL from environment
	zitadelURL := os.Getenv("ZITADEL_URL")
	if zitadelURL == "" {
		zitadelURL = "https://auth.obiente.cloud" // Default fallback
		logger.Warn("⚠️  Warning: ZITADEL_URL not set, using default")
	}

	// Build userinfo URL
	userInfoURL := strings.TrimSuffix(zitadelURL, "/") + "/oidc/v1/userinfo"

	// Create HTTP client with optional TLS skip verification for development
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	if os.Getenv("SKIP_TLS_VERIFY") == "true" {
		logger.Warn("⚠️  Warning: TLS verification disabled (SKIP_TLS_VERIFY=true)")
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	logger.Debug("🔐 Auth Configuration:")
	logger.Debug("  Zitadel URL: %s", zitadelURL)
	logger.Debug("  UserInfo URL: %s", userInfoURL)
	logger.Debug("  Skip TLS Verify: %s", os.Getenv("SKIP_TLS_VERIFY"))

	return &AuthConfig{
		UserInfoCache:  NewUserInfoCache(),
		UserInfoURL:    userInfoURL,
		HTTPClient:     httpClient,
		SkipPaths:      []string{"/health", "/metrics", "/.well-known"},
		ExposeUserInfo: true, // Enable to allow audit interceptor to extract user ID
		SuperAdmins:    superAdmins,
	}
}

// MiddlewareInterceptor creates a Connect interceptor for token authentication
// This interceptor works for both unary and streaming RPCs
func MiddlewareInterceptor(config *AuthConfig) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Skip authentication for Login method (public endpoint)
			if procedure == "/obiente.cloud.auth.v1.AuthService/Login" {
				logger.Debug("[Auth] Skipping authentication for Login procedure")
				return next(ctx, req)
			}

			// If auth is disabled (development only), inject mock user and continue
			if config.UserInfoCache == nil && isAuthDisabled() {
				mockUser := getMockDevUser()
				if mockUser != nil && config.SuperAdmins != nil {
					if _, ok := config.SuperAdmins[strings.ToLower(mockUser.Email)]; ok {
						ensureRole(mockUser, RoleSuperAdmin)
						ensureRole(mockUser, RoleAdmin)
					}
				}
				ctx = context.WithValue(ctx, userInfoKey, mockUser)
				logger.Debug("[Auth] Auth disabled - using mock user for: %s", procedure)
				return next(ctx, req)
			}

			// If auth config is nil but DISABLE_AUTH is not set, this is an error state
			if config.UserInfoCache == nil {
				logger.Warn("⚠️  WARNING: Auth config is nil but DISABLE_AUTH is not set. This should not happen.")
				return next(ctx, req)
			}

			// Extract token from Authorization header
			authHeader := req.Header().Get(AuthorizationHeader)
			hasAuthHeader := authHeader != "" && strings.HasPrefix(authHeader, BearerPrefix)
			logger.Debug("[Auth] Procedure: %s, Has auth header: %v", procedure, hasAuthHeader)

			if !hasAuthHeader {
				logger.Debug("[Auth] Missing Authorization header for: %s", procedure)
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
			}

			token := strings.TrimPrefix(authHeader, BearerPrefix)
			if token == "" {
				logger.Debug("[Auth] Empty token after Bearer prefix for: %s", procedure)
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
			}

			// Validate token against userinfo endpoint
			userInfo, err := config.validateToken(ctx, token)
			if err != nil {
				logger.Debug("[Auth] Token validation failed for %s: %v", procedure, err)
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("error validating token: %w", err))
			}

			logger.Debug("[Auth] Token validated successfully for %s, user: %s", procedure, userInfo.Id)

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
		logger.Debug("✓ Token validated (cached) for user: %s (%s)", cachedUser.Id, cachedUser.Email)
		
		// Always check superadmin emails for cached users, as SUPERADMIN_EMAILS might have changed
		// This ensures superadmin status is applied even when using cached user data
		if cachedUser.Email != "" && c.SuperAdmins != nil {
			lowerEmail := strings.ToLower(cachedUser.Email)
			logger.Debug("[SuperAdmin] Checking cached user email: %s (lowercase: %s)", cachedUser.Email, lowerEmail)
			logger.Debug("[SuperAdmin] SuperAdmins map has %d entries", len(c.SuperAdmins))
			rolesChanged := false
			if _, ok := c.SuperAdmins[lowerEmail]; ok {
				// User is in superadmin list - ensure they have the role
				hadRole := false
				for _, role := range cachedUser.Roles {
					if role == RoleSuperAdmin {
						hadRole = true
						break
					}
				}
				if !hadRole {
					logger.Info("[SuperAdmin] ✓ Granting superadmin role to cached user: %s", cachedUser.Email)
					ensureRole(cachedUser, RoleSuperAdmin)
					ensureRole(cachedUser, RoleAdmin)
					rolesChanged = true
				}
			} else {
				// User is NOT in superadmin list - remove the role if they had it
				// (in case SUPERADMIN_EMAILS was updated to remove them)
				newRoles := removeRole(cachedUser.Roles, RoleSuperAdmin)
				if len(newRoles) != len(cachedUser.Roles) {
					logger.Info("[SuperAdmin] Removing superadmin role from cached user: %s", cachedUser.Email)
					cachedUser.Roles = newRoles
					rolesChanged = true
				}
			}
			
			// Update cache if roles changed to ensure persistence
			if rolesChanged {
				c.UserInfoCache.Set(token, cachedUser)
				logger.Debug("[SuperAdmin] Updated cache with new roles for user: %s", cachedUser.Email)
			}
		} else {
			if cachedUser.Email == "" {
				logger.Debug("[SuperAdmin] Cached user has no email, skipping superadmin check")
			}
			if c.SuperAdmins == nil {
				logger.Debug("[SuperAdmin] SuperAdmins map is nil, skipping superadmin check")
			}
		}
		
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
		logger.Debug("UserInfo request failed: status=%d, body=%s", resp.StatusCode, string(body))

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

	// Apply superadmin roles if configured via email match
	if user.Email != "" && c.SuperAdmins != nil {
		lowerEmail := strings.ToLower(user.Email)
		logger.Debug("[SuperAdmin] Checking new user email: %s (lowercase: %s)", user.Email, lowerEmail)
		logger.Debug("[SuperAdmin] SuperAdmins map has %d entries", len(c.SuperAdmins))
		if _, ok := c.SuperAdmins[lowerEmail]; ok {
			logger.Info("[SuperAdmin] ✓ Granting superadmin role to user: %s", user.Email)
			ensureRole(user, RoleSuperAdmin)
			ensureRole(user, RoleAdmin)
		} else {
			logger.Debug("[SuperAdmin] User %s is NOT in superadmin list", user.Email)
		}
	} else {
		if user.Email == "" {
			logger.Debug("[SuperAdmin] User has no email, skipping superadmin check")
		}
		if c.SuperAdmins == nil {
			logger.Debug("[SuperAdmin] SuperAdmins map is nil, skipping superadmin check")
		}
	}

	// Cache the result
	c.UserInfoCache.Set(token, user)

	logger.Debug("✓ Token validated for user: %s (%s)", user.Id, user.Email)
	return user, nil
}

func ensureRole(user *authv1.User, role string) {
	if user == nil {
		return
	}
	for _, existing := range user.Roles {
		if existing == role {
			return
		}
	}
	user.Roles = append(user.Roles, role)
}

// removeRole removes a role from a roles slice
func removeRole(roles []string, roleToRemove string) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		if role != roleToRemove {
			result = append(result, role)
		}
	}
	return result
}

func loadSuperAdminEmails() map[string]struct{} {
	superAdmins := make(map[string]struct{})
	envValue := os.Getenv("SUPERADMIN_EMAILS")
	logger.Debug("[SuperAdmin] Loading SUPERADMIN_EMAILS from environment: %q", envValue)
	
	for _, raw := range strings.Split(envValue, ",") {
		email := strings.TrimSpace(raw)
		email = strings.Trim(email, "\"'")
		email = strings.ToLower(email)
		if email == "" {
			continue
		}
		superAdmins[email] = struct{}{}
		logger.Debug("[SuperAdmin] Added superadmin email: %s", email)
	}
	if len(superAdmins) > 0 {
		logger.Info("[SuperAdmin] Configured %d superadmin email(s)", len(superAdmins))
		for email := range superAdmins {
			logger.Info("[SuperAdmin]   - %s", email)
		}
	} else {
		logger.Warn("[SuperAdmin] No superadmin emails configured (SUPERADMIN_EMAILS is empty or invalid)")
	}
	return superAdmins
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

// AuthenticateAndSetContext authenticates a request and adds the user to context
// This is useful for streaming RPCs where the interceptor may not run
func AuthenticateAndSetContext(ctx context.Context, authHeader string) (context.Context, *authv1.User, error) {
	if isAuthDisabled() {
		mockUser := getMockDevUser()
		if mockUser == nil {
			return nil, nil, errors.New("mock user not available")
		}
		superAdmins := loadSuperAdminEmails()
		if _, ok := superAdmins[strings.ToLower(mockUser.Email)]; ok {
			ensureRole(mockUser, RoleSuperAdmin)
			ensureRole(mockUser, RoleAdmin)
		}
		ctx = context.WithValue(ctx, userInfoKey, mockUser)
		return ctx, mockUser, nil
	}

	if authHeader == "" || !strings.HasPrefix(authHeader, BearerPrefix) {
		return nil, nil, ErrNoToken
	}

	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return nil, nil, ErrNoToken
	}

	// Get auth config
	config := NewAuthConfig()

	// Validate token
	userInfo, err := config.validateToken(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("error validating token: %w", err)
	}

	// Add user to context
	ctx = context.WithValue(ctx, userInfoKey, userInfo)

	return ctx, userInfo, nil
}

// WithSystemUser creates a context with a system user that has admin permissions
// This is used for internal operations that need to bypass permission checks
func WithSystemUser(ctx context.Context) context.Context {
	systemUser := &authv1.User{
		Id:                "system",
		Email:             "system@obiente.local",
		Name:              "System",
		GivenName:         "System",
		FamilyName:        "User",
		PreferredUsername: "system",
		EmailVerified:     true,
		Locale:            "en",
		AvatarUrl:         "",
		Roles:             []string{RoleAdmin, RoleSuperAdmin},
		UpdatedAt:         timestamppb.Now(),
	}
	return context.WithValue(ctx, userInfoKey, systemUser)
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
