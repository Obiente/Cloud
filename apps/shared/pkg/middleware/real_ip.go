package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIPMiddleware is middleware that extracts the real client IP and makes it available
// to downstream handlers. It modifies the request to include the real IP in a custom header
// that downstream services can use for logging and auditing.
func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract real client IP using the GetClientIP utility
		clientIP := GetClientIP(r)

		// Add a custom header that backend services can use
		// This ensures all downstream services have access to the real client IP
		r.Header.Set("X-Obiente-Client-IP", clientIP)

		// Also update RemoteAddr for services that expect it
		// (this is useful for services that use r.RemoteAddr directly)
		// Note: This sets just the IP; the port information is lost
		r.RemoteAddr = clientIP

		next.ServeHTTP(w, r)
	})
}

// TrustedProxyMiddleware returns a middleware that trusts specific proxy IP ranges.
// This should be used in production when you want to limit which proxies can set
// X-Forwarded-For headers. Leave empty for development/internal deployments.
//
// Example usage:
//
//	trustedProxies := []string{"10.0.0.0/8", "172.16.0.0/12"}
//	handler = TrustedProxyMiddleware(handler, trustedProxies)
func TrustedProxyMiddleware(next http.Handler, trustedProxies []string) http.Handler {
	var cidrs []*net.IPNet
	for _, proxy := range trustedProxies {
		_, cidr, err := net.ParseCIDR(proxy)
		if err == nil {
			cidrs = append(cidrs, cidr)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if RemoteAddr (direct connection) is in the trusted proxy list
		remoteIP := r.RemoteAddr
		if idx := strings.Index(remoteIP, ":"); idx != -1 {
			remoteIP = remoteIP[:idx]
		}

		parsedRemoteIP := net.ParseIP(remoteIP)
		isTrusted := false

		if parsedRemoteIP != nil {
			for _, cidr := range cidrs {
				if cidr.Contains(parsedRemoteIP) {
					isTrusted = true
					break
				}
			}
		}

		// Only trust X-Forwarded-For if the direct connection is from a trusted proxy
		if isTrusted {
			next.ServeHTTP(w, r)
		} else {
			// If not a trusted proxy, remove forwarded headers to prevent spoofing
			r.Header.Del("X-Forwarded-For")
			r.Header.Del("X-Forwarded-Proto")
			r.Header.Del("X-Real-IP")
			next.ServeHTTP(w, r)
		}
	})
}
