package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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
	// Check if debug logging is enabled
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	debug := logLevel == "debug" || logLevel == "trace"

	log.Printf("[Middleware] RequestLogger initialized (debug=%v, LOG_LEVEL=%s)", debug, logLevel)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log incoming request
		log.Printf("[Request] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		if debug {
			// Log important headers
			log.Printf("  Origin: %s", r.Header.Get("Origin"))
			log.Printf("  Content-Type: %s", r.Header.Get("Content-Type"))
			log.Printf("  Authorization: %s", maskAuth(r.Header.Get("Authorization")))
			log.Printf("  User-Agent: %s", r.Header.Get("User-Agent"))
			log.Printf("  Connect-Protocol-Version: %s", r.Header.Get("Connect-Protocol-Version"))

			// Log all headers in debug mode
			log.Println("  All Headers:")
			for name, values := range r.Header {
				for _, value := range values {
					if name == "Authorization" {
						value = maskAuth(value)
					}
					log.Printf("    %s: %s", name, value)
				}
			}
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		log.Printf("[Response] %s %s -> %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)

		if debug {
			// Log response headers
			log.Println("  Response Headers:")
			for name, values := range wrapped.Header() {
				for _, value := range values {
					log.Printf("    %s: %s", name, value)
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
	log.Println("[Middleware] CORSDebugLogger initialized")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only log if there's an Origin header (CORS request)
		if origin != "" {
			log.Printf("[CORS] Request: %s %s", r.Method, r.URL.Path)
			log.Printf("[CORS]   Origin: %s", origin)
			log.Printf("[CORS]   Method: %s", r.Method)
			log.Printf("[CORS]   Access-Control-Request-Method: %s", r.Header.Get("Access-Control-Request-Method"))
			log.Printf("[CORS]   Access-Control-Request-Headers: %s", r.Header.Get("Access-Control-Request-Headers"))
		}

		// Wrap response to log CORS headers
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Log CORS response headers
		if origin != "" {
			log.Printf("[CORS] Response: %d", wrapped.statusCode)
			log.Printf("[CORS]   Access-Control-Allow-Origin: %s", wrapped.Header().Get("Access-Control-Allow-Origin"))
			log.Printf("[CORS]   Access-Control-Allow-Credentials: %s", wrapped.Header().Get("Access-Control-Allow-Credentials"))
			log.Printf("[CORS]   Access-Control-Allow-Methods: %s", wrapped.Header().Get("Access-Control-Allow-Methods"))
			log.Printf("[CORS]   Access-Control-Allow-Headers: %s", wrapped.Header().Get("Access-Control-Allow-Headers"))
		}
	})
}
