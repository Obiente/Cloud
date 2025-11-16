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
	allowedOrigins := []string{"*"}
	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
		// Trim spaces
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
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
		AllowCredentials: true,
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

			// Check if origin is allowed
			allowedOrigin := ""

			// Check if wildcard is configured
			isWildcard := len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*"
			
			if isWildcard {
				// When credentials are enabled, we MUST echo the origin, not use "*"
				// Browsers reject "Access-Control-Allow-Origin: *" with credentials
				if config.AllowCredentials && origin != "" {
					// Always echo the origin when wildcard is configured and credentials are enabled
					allowedOrigin = origin
					logger.Debug("[CORS] Wildcard mode with credentials: echoing origin %s", origin)
				} else if !config.AllowCredentials {
					allowedOrigin = "*"
					logger.Debug("[CORS] Wildcard mode without credentials: using *")
				} else if origin == "" {
					// No origin header - might be same-origin request
					// For streaming endpoints, we should still allow the request
					// Note: Cannot use "*" with credentials, but since there's no origin header,
					// this might be a same-origin request, so we'll allow it
					allowedOrigin = "*"
					logger.Debug("[CORS] Wildcard mode: no origin header, allowing with *")
				}
			} else if origin != "" {
				// Check if the specific origin is in the allowed list
				for _, allowed := range config.AllowedOrigins {
					// Normalize origins for comparison (trim trailing slashes)
					normalizedOrigin := strings.TrimSuffix(origin, "/")
					normalizedAllowed := strings.TrimSuffix(allowed, "/")
					
					if normalizedAllowed == normalizedOrigin || allowed == "*" {
						allowedOrigin = origin
						logger.Debug("[CORS] Origin %s matched allowed origin %s", origin, allowed)
						break
					}
				}
				
				if allowedOrigin == "" {
					logger.Debug("[CORS] Origin %s NOT in allowed list: %v", origin, config.AllowedOrigins)
				}
			} else {
				// No Origin header - might be same-origin request or browser didn't send it
				// For streaming endpoints and API calls, we should still allow if wildcard is configured
				if isWildcard && !config.AllowCredentials {
					allowedOrigin = "*"
					logger.Debug("[CORS] No origin header but wildcard configured, allowing")
				} else if r.Method == http.MethodOptions {
					logger.Debug("[CORS] OPTIONS request with no Origin header")
				}
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

				// Vary header is important when echoing origin
				if allowedOrigin != "*" {
					w.Header().Add("Vary", "Origin")
				}

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

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
			} else if origin != "" {
				// Origin was provided but didn't match - log for debugging
				logger.Debug("[CORS] Origin %s not allowed, not setting CORS headers", origin)
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
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
