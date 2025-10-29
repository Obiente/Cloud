package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Constants for JWT validation
const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	JWKSRefreshInterval = 30 * time.Minute
)

var (
	ErrNoToken         = errors.New("no token provided")
	ErrInvalidToken    = errors.New("invalid or malformed token")
	ErrTokenExpired    = errors.New("token expired")
	ErrInvalidAudience = errors.New("invalid audience")
	ErrInvalidIssuer   = errors.New("invalid issuer")
	ErrMissingUserID   = errors.New("user ID not found in token")
)

// UserInfo holds the authenticated user information
type UserInfo struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	Name           string   `json:"name"`
	Picture        string   `json:"picture"`
	GivenName      string   `json:"given_name"`
	FamilyName     string   `json:"family_name"`
	OrganizationID string   `json:"organization_id"`
	Roles          []string `json:"roles"`
	Locale         string   `json:"locale"`
}

// contextKey is a private type for context keys
type contextKey int

const userInfoKey contextKey = 0

// JWKSCache manages caching and refreshing the JWKS
type JWKSCache struct {
	keySet    jwk.Set
	mutex     sync.RWMutex
	lastFetch time.Time
	jwksURL   string
}

// NewJWKSCache creates a new JWKS cache with auto-refresh capability
func NewJWKSCache(jwksURL string) *JWKSCache {
	cache := &JWKSCache{
		jwksURL: jwksURL,
	}

	// Initial fetch
	err := cache.refreshKeys()
	if err != nil {
		log.Printf("Warning: Initial JWKS fetch failed: %v\n", err)
	}

	// Start background refresh
	go cache.startAutoRefresh()

	return cache
}

// GetKeySet returns the current JWKS, fetching if needed
func (c *JWKSCache) GetKeySet() (jwk.Set, error) {
	c.mutex.RLock()
	keySet := c.keySet
	lastFetch := c.lastFetch
	c.mutex.RUnlock()

	// If we haven't fetched keys yet or they're stale, refresh them
	if keySet == nil || time.Since(lastFetch) > JWKSRefreshInterval {
		err := c.refreshKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
		}

		c.mutex.RLock()
		keySet = c.keySet
		c.mutex.RUnlock()
	}

	return keySet, nil
}

// refreshKeys fetches a fresh copy of the JWKS
func (c *JWKSCache) refreshKeys() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, c.jwksURL)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	c.keySet = keySet
	c.lastFetch = time.Now()
	c.mutex.Unlock()

	log.Println("JWKS refreshed successfully")
	return nil
}

// startAutoRefresh periodically refreshes the JWKS
func (c *JWKSCache) startAutoRefresh() {
	ticker := time.NewTicker(JWKSRefreshInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		err := c.refreshKeys()
		if err != nil {
			log.Printf("Error refreshing JWKS: %v\n", err)
		}
	}
}

// AuthConfig holds the configuration for the auth middleware
type AuthConfig struct {
	JWKSCache      *JWKSCache
	Issuer         string
	Audience       string
	SkipPaths      []string
	ExposeUserInfo bool // Whether to expose user info in response headers
}

// NewAuthConfig creates a new auth config with default values
func NewAuthConfig() *AuthConfig {
	// Get values from environment
	jwksURL := os.Getenv("ZITADEL_JWKS_URL")
	if jwksURL == "" {
		zitadelURL := os.Getenv("ZITADEL_URL")
		if zitadelURL != "" {
			jwksURL = zitadelURL + "/.well-known/jwks.json"
		} else {
			jwksURL = "https://obiente.cloud/.well-known/jwks.json" // Default fallback
			log.Println("Warning: ZITADEL_JWKS_URL or ZITADEL_URL not set, using default")
		}
	}

	issuer := os.Getenv("ZITADEL_ISSUER")
	if issuer == "" {
		issuer = "https://obiente.cloud" // Default fallback
		log.Println("Warning: ZITADEL_ISSUER not set, using default")
	}

	audience := os.Getenv("ZITADEL_CLIENT_ID")
	if audience == "" {
		audience = "339499954043158530" // Default fallback
		log.Println("Warning: ZITADEL_CLIENT_ID not set, using default")
	}

	return &AuthConfig{
		JWKSCache:      NewJWKSCache(jwksURL),
		Issuer:         issuer,
		Audience:       audience,
		SkipPaths:      []string{"/health", "/metrics", "/.well-known"},
		ExposeUserInfo: false,
	}
}

// MiddlewareInterceptor creates a Connect interceptor for JWT authentication
func MiddlewareInterceptor(config *AuthConfig) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
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

			// Get the key set for validation
			keySet, err := config.JWKSCache.GetKeySet()
			if err != nil {
				log.Printf("Error getting JWKS: %v", err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error validating token: %w", err))
			}

			// Parse and verify token
			parsedToken, err := ValidateToken(token, keySet, config.Issuer, config.Audience)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
			}

			// Extract user info
			userInfo, err := extractUserInfo(parsedToken)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("failed to extract user info: %w", err))
			}

			// Create a new context with user info
			ctx = context.WithValue(ctx, userInfoKey, userInfo)

			// Call the next handler
			resp, err := next(ctx, req)

			// Optionally add user info to response headers (for debugging)
			if config.ExposeUserInfo && resp != nil && err == nil {
				resp.Header().Set("X-User-ID", userInfo.ID)
				resp.Header().Set("X-User-Email", userInfo.Email)
			}

			return resp, err
		}
	}
}

// ValidateToken validates a JWT token against the provided JWKS
func ValidateToken(tokenString string, keySet jwk.Set, issuer, audience string) (jwt.Token, error) {
	// Verify token signature
	payload, err := jws.Verify([]byte(tokenString), jws.WithKeySet(keySet))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Parse token
	token, err := jwt.Parse(payload,
		jwt.WithValidate(true),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Check expiration
	if token.Expiration().Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	return token, nil
}

// GetUserFromContext extracts user info from context
func GetUserFromContext(ctx context.Context) (*UserInfo, error) {
	userInfo, ok := ctx.Value(userInfoKey).(*UserInfo)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return userInfo, nil
}

// extractUserInfo extracts user information from JWT token
func extractUserInfo(token jwt.Token) (*UserInfo, error) {
	// Extract the subject (user ID)
	sub, ok := token.Get("sub")
	if !ok {
		return nil, ErrMissingUserID
	}

	subStr, ok := sub.(string)
	if !ok {
		return nil, fmt.Errorf("invalid subject format in token")
	}

	userInfo := &UserInfo{
		ID: subStr,
	}

	// Extract optional fields
	if email, ok := token.Get("email"); ok {
		if emailStr, ok := email.(string); ok {
			userInfo.Email = emailStr
		}
	}

	if name, ok := token.Get("name"); ok {
		if nameStr, ok := name.(string); ok {
			userInfo.Name = nameStr
		}
	}

	if givenName, ok := token.Get("given_name"); ok {
		if givenNameStr, ok := givenName.(string); ok {
			userInfo.GivenName = givenNameStr
		}
	}

	if familyName, ok := token.Get("family_name"); ok {
		if familyNameStr, ok := familyName.(string); ok {
			userInfo.FamilyName = familyNameStr
		}
	}

	if picture, ok := token.Get("picture"); ok {
		if pictureStr, ok := picture.(string); ok {
			userInfo.Picture = pictureStr
		}
	}

	if locale, ok := token.Get("locale"); ok {
		if localeStr, ok := locale.(string); ok {
			userInfo.Locale = localeStr
		}
	}

	// Extract organization ID (if present in a custom claim)
	if orgID, ok := token.Get("organization_id"); ok {
		if orgIDStr, ok := orgID.(string); ok {
			userInfo.OrganizationID = orgIDStr
		}
	}

	// Extract roles
	userInfo.Roles = extractRoles(token)

	return userInfo, nil
}

// extractRoles extracts roles from various potential claim formats
func extractRoles(token jwt.Token) []string {
	roles := []string{}

	// Try standard roles claim
	if rolesIface, ok := token.Get("roles"); ok {
		if rolesArray, ok := rolesIface.([]interface{}); ok {
			for _, role := range rolesArray {
				if roleStr, ok := role.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}
	}

	// Try Zitadel-specific project roles format
	zitadelRolesKey := "urn:zitadel:iam:org:project:roles"
	if zitadelRoles, ok := token.Get(zitadelRolesKey); ok {
		if zitadelRolesArray, ok := zitadelRoles.([]interface{}); ok {
			for _, role := range zitadelRolesArray {
				if roleStr, ok := role.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}
	}

	// Try Zitadel project-specific roles
	zitadelProjectsKey := "urn:zitadel:iam:org:projects"
	if projectsIface, ok := token.Get(zitadelProjectsKey); ok {
		if projects, ok := projectsIface.(map[string]interface{}); ok {
			for _, projectRoles := range projects {
				if rolesArray, ok := projectRoles.([]interface{}); ok {
					for _, role := range rolesArray {
						if roleStr, ok := role.(string); ok {
							roles = append(roles, roleStr)
						}
					}
				}
			}
		}
	}

	return roles
}

// AuthenticateHTTPRequest authenticates an HTTP request outside of Connect RPC
// This can be used for regular HTTP handlers that need authentication
func AuthenticateHTTPRequest(config *AuthConfig, w http.ResponseWriter, r *http.Request) (*UserInfo, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get(AuthorizationHeader)
	if authHeader == "" || !strings.HasPrefix(authHeader, BearerPrefix) {
		return nil, ErrNoToken
	}

	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return nil, ErrNoToken
	}

	// Get the key set for validation
	keySet, err := config.JWKSCache.GetKeySet()
	if err != nil {
		return nil, fmt.Errorf("error getting JWKS: %w", err)
	}

	// Parse and verify token
	parsedToken, err := ValidateToken(token, keySet, config.Issuer, config.Audience)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract user info
	return extractUserInfo(parsedToken)
}
