package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"api/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code
// It also implements http.Flusher and http.Hijacker if the underlying writer does
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher if the underlying ResponseWriter does
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker if the underlying ResponseWriter does
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// RequestLogger logs all incoming requests with detailed information
func RequestLogger(next http.Handler) http.Handler {
	// Initialize logger if not already done
	logger.Init()
	debug := logger.IsDebug()

	logger.Debug("[Middleware] RequestLogger initialized (debug=%v, LOG_LEVEL=%s)", debug, logger.GetLevel())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log incoming request (only at info level or above)
		logger.Debug("[Request] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if debug {
			// Log important headers
			logger.Debug("  Origin: %s", r.Header.Get("Origin"))
			logger.Debug("  Content-Type: %s", r.Header.Get("Content-Type"))
			logger.Debug("  Authorization: %s", maskAuth(r.Header.Get("Authorization")))
			logger.Debug("  User-Agent: %s", r.Header.Get("User-Agent"))
			logger.Debug("  Connect-Protocol-Version: %s", r.Header.Get("Connect-Protocol-Version"))

			// Log all headers in debug mode
			logger.Debugln("  All Headers:")
			for name, values := range r.Header {
				for _, value := range values {
					if name == "Authorization" {
						value = maskAuth(value)
					}
					logger.Debug("    %s: %s", name, value)
				}
			}
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log response (only at info level or above)
		duration := time.Since(start)
		// Log errors and warnings at their respective levels
		if wrapped.statusCode >= 500 {
			logger.Error("[Response] %s %s -> %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
		} else if wrapped.statusCode >= 400 {
			logger.Warn("[Response] %s %s -> %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
		} else {
			logger.Debug("[Response] %s %s -> %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
		}

		if debug {
			// Log response headers
			logger.Debugln("  Response Headers:")
			for name, values := range wrapped.Header() {
				for _, value := range values {
					logger.Debug("    %s: %s", name, value)
				}
			}
		}
	})
}

// maskAuth masks authorization header for security
func maskAuth(auth string) string {
	if auth == "" {
		return "<none>"
	}
	if len(auth) > 20 {
		return auth[:15] + "..."
	}
	return auth
}

// CORSDebugLogger specifically logs CORS-related information
func CORSDebugLogger(next http.Handler) http.Handler {
	logger.Debugln("[Middleware] CORSDebugLogger initialized")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only log if there's an Origin header (CORS request) and debug is enabled
		if origin != "" && logger.IsDebug() {
			logger.Debug("[CORS] Request: %s %s", r.Method, r.URL.Path)
			logger.Debug("[CORS]   Origin: %s", origin)
			logger.Debug("[CORS]   Method: %s", r.Method)
			logger.Debug("[CORS]   Access-Control-Request-Method: %s", r.Header.Get("Access-Control-Request-Method"))
			logger.Debug("[CORS]   Access-Control-Request-Headers: %s", r.Header.Get("Access-Control-Request-Headers"))
		}

		// Wrap response to log CORS headers
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Log CORS response headers (only in debug mode)
		if origin != "" && logger.IsDebug() {
			logger.Debug("[CORS] Response: %d", wrapped.statusCode)
			logger.Debug("[CORS]   Access-Control-Allow-Origin: %s", wrapped.Header().Get("Access-Control-Allow-Origin"))
			logger.Debug("[CORS]   Access-Control-Allow-Credentials: %s", wrapped.Header().Get("Access-Control-Allow-Credentials"))
			logger.Debug("[CORS]   Access-Control-Allow-Methods: %s", wrapped.Header().Get("Access-Control-Allow-Methods"))
			logger.Debug("[CORS]   Access-Control-Allow-Headers: %s", wrapped.Header().Get("Access-Control-Allow-Headers"))
		}
	})
}
