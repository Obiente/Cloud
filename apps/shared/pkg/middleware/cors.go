package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           string
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	// Get allowed origins from environment
	originsEnv := os.Getenv("CORS_ORIGIN")
	allowedOrigins := []string{}
	allowCredentials := true

	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
		// Trim spaces
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// Developer-friendly defaults for local environments so dashboard → API calls don't need extra env config
		allowedOrigins = []string{
			"http://localhost",
			"http://127.0.0.1",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3001",
			"http://api.localhost",
			"http://api.localhost:80",
			"http://api.localhost:8080",
			"http://api.localhost:880",
		}

		// When relying on the defaults we only need Authorization headers (no cookies),
		// so disable credentials to keep wildcard/preflight logic simple.
		allowCredentials = false
	}

	// If nothing is configured, fall back to permissive wildcard without credentials
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
		allowCredentials = false
	}

	return &CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Accept",
			"Accept-Encoding",
			"Authorization",
			"Content-Type",
			"Content-Length",
			"Origin",
			"X-Requested-With",
			"X-CSRF-Token",
			// WebSocket headers
			"Upgrade",
			"Connection",
			"Sec-WebSocket-Key",
			"Sec-WebSocket-Version",
			"Sec-WebSocket-Protocol",
			"Sec-WebSocket-Extensions",
			// Connect-RPC specific headers
			"Connect-Protocol-Version",
			"Connect-Timeout-Ms",
			"Grpc-Timeout",
			"X-Grpc-Web",
			"X-User-Agent",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Type",
			// Connect-RPC specific headers
			"Connect-Protocol-Version",
			"Grpc-Status",
			"Grpc-Message",
			"Grpc-Status-Details-Bin",
		},
		AllowCredentials: allowCredentials,
		MaxAge:           "7200", // 2 hours
	}
}

// CORS creates a CORS middleware with the given configuration
func CORS(config *CORSConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultCORSConfig()
	}

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				origin := r.Header.Get("Origin")
				isPreflight := r.Method == http.MethodOptions

				// Check if origin is allowed
				allowedOrigin := ""
				isWildcard := len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*"

				// If wildcard is configured with credentials, automatically fall back to non-credentialed wildcard
				if isWildcard && config.AllowCredentials {
					logger.Warn("[CORS] Wildcard (*) cannot be combined with credentials - disabling credentials for compatibility")
					config.AllowCredentials = false
				}

				// Determine if origin is allowed
				if origin != "" {
					// Cross-origin request - must validate origin
					if isWildcard {
						// Wildcard allows all origins (but only without credentials)
						allowedOrigin = "*"
						logger.Debug("[CORS] Wildcard mode: allowing origin %s", origin)
					} else {
						// Check if the specific origin is in the allowed list
						for _, allowed := range config.AllowedOrigins {
							// Normalize origins for comparison (trim trailing slashes and whitespace)
							normalizedOrigin := strings.TrimSpace(strings.TrimSuffix(origin, "/"))
							normalizedAllowed := strings.TrimSpace(strings.TrimSuffix(allowed, "/"))
							
							if normalizedAllowed == normalizedOrigin {
								// Echo the exact origin (CORS best practice)
								allowedOrigin = origin
								logger.Debug("[CORS] Origin %s matched allowed origin %s", origin, allowed)
								break
							}
						}
						
						if allowedOrigin == "" {
							logger.Debug("[CORS] Origin %s NOT in allowed list: %v", origin, config.AllowedOrigins)
							// CORS Best Practice: Do NOT set CORS headers for disallowed origins
							// This prevents browsers from reading the response
							
							// Handle preflight request with disallowed origin
							if isPreflight {
								// CORS Best Practice: Return 403 Forbidden for preflight requests with disallowed origins
								w.WriteHeader(http.StatusForbidden)
								return
							}
							// For non-preflight requests, continue without CORS headers
							// Browser will block the response, which is correct
							next.ServeHTTP(w, r)
							return
						}
					}
				} else {
					// No Origin header - might be same-origin request
					// CORS Best Practice: Same-origin requests don't need CORS headers
					// Only set CORS headers for cross-origin requests
					// If wildcard is configured without credentials, allow it
					if isWildcard && !config.AllowCredentials {
						allowedOrigin = "*"
					}
				}

				// Set CORS headers only if origin is allowed
				if allowedOrigin != "" {
					// CORS Best Practice: Echo the exact origin (not "*" when credentials are enabled)
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					
					// CORS Best Practice: Always set Vary: Origin when echoing specific origins
					if allowedOrigin != "*" {
						w.Header().Add("Vary", "Origin")
					}
					
					if config.AllowCredentials {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
					
					// Set other CORS headers
					if len(config.AllowedMethods) > 0 {
						w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
					}
					
					if len(config.AllowedHeaders) > 0 {
						w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
					}
					
					if len(config.ExposedHeaders) > 0 {
						w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
					}
					
					if config.MaxAge != "" {
						w.Header().Set("Access-Control-Max-Age", config.MaxAge)
					}
				}

				// Handle preflight OPTIONS request
				if isPreflight {
					// CORS Best Practice: For preflight, return 204 No Content if origin is allowed
					if allowedOrigin != "" {
						w.WriteHeader(http.StatusNoContent)
						return
					} else {
						// Origin not allowed for preflight
						w.WriteHeader(http.StatusForbidden)
						return
					}
				}

				// Continue to next handler
				next.ServeHTTP(w, r)
			})
		}
}

// CORSHandler wraps an http.Handler with CORS middleware
func CORSHandler(handler http.Handler) http.Handler {
	logger.Init()
	config := DefaultCORSConfig()
	logger.Debug("[Middleware] CORS initialized - AllowedOrigins: %v, AllowCredentials: %v",
		config.AllowedOrigins, config.AllowCredentials)
	return CORS(config)(handler)
}

// IsOriginAllowed checks if an origin is allowed based on the CORS configuration.
// This is useful for WebSocket handlers that need to validate origins.
// For WebSocket: cross-origin requests always include an Origin header.
// Same-origin requests may not include an Origin header, which we allow when wildcard is configured.
func IsOriginAllowed(origin string) bool {
	config := DefaultCORSConfig()
	
	// Check if wildcard is configured
	isWildcard := len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*"
	
	if isWildcard {
		// Wildcard allows all origins (including empty origin for same-origin requests)
		return true
	}
	
	if origin == "" {
		// No origin header - might be same-origin request
		// For WebSocket, cross-origin requests always have Origin header
		// Same-origin requests may not have it, but we can't easily distinguish
		// When specific origins are configured, we require Origin header for security
		return false
	}
	
	// Check if the specific origin is in the allowed list
	for _, allowed := range config.AllowedOrigins {
		// Normalize origins for comparison (trim trailing slashes)
		normalizedOrigin := strings.TrimSuffix(origin, "/")
		normalizedAllowed := strings.TrimSuffix(allowed, "/")
		
		if normalizedAllowed == normalizedOrigin || allowed == "*" {
			return true
		}
	}
	
	return false
}
