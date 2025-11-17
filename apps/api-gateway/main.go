package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

// Service routing configuration
// Base service routes (internal service names)
var baseServiceRoutes = map[string]string{
	"/obiente.cloud.auth.v1.AuthService/":                  "auth-service:3002",
	"/obiente.cloud.organizations.v1.OrganizationService/": "organizations-service:3003",
	"/obiente.cloud.billing.v1.BillingService/":            "billing-service:3004",
	"/obiente.cloud.deployments.v1.DeploymentService/":     "deployments-service:3005",
	"/obiente.cloud.gameservers.v1.GameServerService/":     "gameservers-service:3006",
	"/obiente.cloud.vps.v1.VPSService/":                    "vps-service:3008",
	"/obiente.cloud.superadmin.v1.SuperadminService/":      "superadmin-service:3011",
	"/obiente.cloud.support.v1.SupportService/":            "support-service:3009",
	"/obiente.cloud.audit.v1.AuditService/":                "audit-service:3010",
	"/webhooks/stripe":                                     "billing-service:3004",
	"/dns/push":                                            "dns-service:8053",         // DNS delegation push endpoint
	"/dns/push/batch":                                      "dns-service:8053",         // DNS delegation batch push endpoint
	"/terminal/ws":                                         "deployments-service:3005", // Deployment terminals
	"/gameservers/terminal/ws":                             "gameservers-service:3006", // Game server terminals
	"/vps/":                                                "vps-service:3008",         // VPS terminals and other VPS endpoints
	"/vps/ssh/":                                            "vps-service:3008",         // VPS SSH proxy
}

// Service name to domain mapping (for Traefik routing)
var serviceDomains = map[string]string{
	"auth-service:3002":         "auth-service",
	"organizations-service:3003": "organizations-service",
	"billing-service:3004":      "billing-service",
	"deployments-service:3005":  "deployments-service",
	"gameservers-service:3006":  "gameservers-service",
	"vps-service:3008":          "vps-service",
	"superadmin-service:3011":   "superadmin-service",
	"support-service:3009":     "support-service",
	"audit-service:3010":        "audit-service",
	"dns-service:8053":          "dns-service", // Note: DNS service may need special handling
}

// buildServiceRoutes constructs the service routes based on routing mode
func buildServiceRoutes() map[string]string {
	useTraefik := os.Getenv("USE_TRAEFIK_ROUTING")
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}

	routes := make(map[string]string)

	for path, serviceAddr := range baseServiceRoutes {
		if useTraefik == "true" || useTraefik == "1" {
			// Use Traefik routing with HTTPS
			serviceDomain, ok := serviceDomains[serviceAddr]
			if !ok {
				// Fallback: extract service name from address
				parts := strings.Split(serviceAddr, ":")
				serviceDomain = parts[0]
			}
			routes[path] = fmt.Sprintf("https://%s.%s", serviceDomain, domain)
		} else {
			// Use internal routing (default)
			routes[path] = fmt.Sprintf("http://%s", serviceAddr)
		}
	}

	return routes
}

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Initialize logger
	logger.Init()

	logger.Info("=== API Gateway Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	// Build service routes based on routing mode
	serviceRoutes := buildServiceRoutes()
	useTraefik := os.Getenv("USE_TRAEFIK_ROUTING")
	if useTraefik == "true" || useTraefik == "1" {
		logger.Info("Routing mode: Traefik (HTTPS)")
	} else {
		logger.Info("Routing mode: Internal (HTTP)")
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create reverse proxy handler
	proxy := &ReverseProxy{
		routes: serviceRoutes,
	}

	// Initialize health checker for backend services
	proxy.initHealthChecker()
	logger.Info("✓ Health checker initialized for backend services")

	// Register all service routes
	for path, targetURL := range serviceRoutes {
		mux.Handle(path, proxy)
		logger.Info("✓ Route registered: %s -> %s", path, targetURL)
	}

	// Health check endpoint - always returns healthy so Docker doesn't restart the gateway
	// The gateway itself is healthy as long as it's running, even if backends are unhealthy
	// Backend health is monitored but does not block routing (allows degraded services to still receive traffic)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"api-gateway"}`))
	})

	// Detailed health endpoint - shows backend service health status (for monitoring/debugging)
	mux.HandleFunc("/health/detailed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Check health of all backend services
		proxy.healthMutex.RLock()
		allHealthy := true
		unhealthyServices := []string{}
		healthyServices := []string{}
		checkedCount := 0
		for serviceURL, healthy := range proxy.healthStatus {
			checkedCount++
			if !healthy {
				allHealthy = false
				unhealthyServices = append(unhealthyServices, serviceURL)
			} else {
				healthyServices = append(healthyServices, serviceURL)
			}
		}
		proxy.healthMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		
		// If we haven't checked any services yet (startup), show that
		if checkedCount == 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy","service":"api-gateway","message":"health checks not yet initialized","backends_checked":0}`))
			return
		}
		
		// Format service lists as JSON arrays
		unhealthyList := "["
		for i, svc := range unhealthyServices {
			if i > 0 {
				unhealthyList += ","
			}
			unhealthyList += fmt.Sprintf(`"%s"`, svc)
		}
		unhealthyList += "]"
		
		healthyList := "["
		for i, svc := range healthyServices {
			if i > 0 {
				healthyList += ","
			}
			healthyList += fmt.Sprintf(`"%s"`, svc)
		}
		healthyList += "]"
		
		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}
		
		detailedJSON := fmt.Sprintf(`{"status":"%s","service":"api-gateway","all_backends_healthy":%t,"healthy_backends":%s,"unhealthy_backends":%s,"total_backends":%d}`,
			map[bool]string{true: "healthy", false: "degraded"}[allHealthy],
			allHealthy,
			healthyList,
			unhealthyList,
			checkedCount)
		
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(detailedJSON))
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// Try to find matching route
			for path := range serviceRoutes {
				if strings.HasPrefix(r.URL.Path, path) {
					proxy.ServeHTTP(w, r)
					return
				}
			}
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("api-gateway"))
	})

	// Wrap with h2c for HTTP/2
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	// Apply middleware
	var handler http.Handler = h2cHandler
	handler = middleware.CORSHandler(handler)
	handler = middleware.RequestLogger(handler)
	// Note: Auth interceptor is applied per-route in the proxy

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Set up graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== API Gateway Ready - Listening on %s ===", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for interrupt or server error
	select {
	case err := <-serverErr:
		logger.Fatalf("server failed: %v", err)
	case <-shutdownCtx.Done():
		logger.Info("\n=== Shutting down gracefully ===")
		shutdownTimeout := 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("Error during server shutdown: %v", err)
		} else {
			logger.Info(gracefulShutdownMessage)
		}
	}
}

// ReverseProxy handles routing requests to backend services
type ReverseProxy struct {
	routes       map[string]string
	healthStatus map[string]bool // Tracks health status of each backend service
	healthMutex  sync.RWMutex
}

// initHealthChecker starts background health checks for all backend services
func (p *ReverseProxy) initHealthChecker() {
	p.healthStatus = make(map[string]bool)
	
	// Start health checking goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
		defer ticker.Stop()
		
		for range ticker.C {
			p.checkAllServicesHealth()
		}
	}()
	
	// Do initial health check
	p.checkAllServicesHealth()
}

// checkAllServicesHealth checks health of all backend services
func (p *ReverseProxy) checkAllServicesHealth() {
	for _, targetURL := range p.routes {
		// Extract service URL (e.g., "http://auth-service:3002" -> "http://auth-service:3002/health")
		target, err := url.Parse(targetURL)
		if err != nil {
			continue
		}
		
		healthURL := fmt.Sprintf("%s://%s/health", target.Scheme, target.Host)
		isHealthy := p.checkServiceHealth(healthURL)
		
		p.healthMutex.Lock()
		p.healthStatus[targetURL] = isHealthy
		p.healthMutex.Unlock()
		
		if !isHealthy {
			logger.Warn("[API Gateway] Service %s is unhealthy", targetURL)
		}
	}
}

// checkServiceHealth checks if a service is healthy by calling its /health endpoint
func (p *ReverseProxy) checkServiceHealth(healthURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	
	// Configure TLS for HTTPS health checks (Traefik routing)
	skipTLSVerify := os.Getenv("SKIP_TLS_VERIFY")
	shouldSkipVerify := skipTLSVerify == "true" || skipTLSVerify == "1"
	
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: shouldSkipVerify, // Skip TLS verification for internal Traefik certs
		},
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// isServiceHealthy checks if a service is currently healthy
func (p *ReverseProxy) isServiceHealthy(targetURL string) bool {
	p.healthMutex.RLock()
	defer p.healthMutex.RUnlock()
	
	// Default to healthy if we haven't checked yet (optimistic)
	if healthy, exists := p.healthStatus[targetURL]; exists {
		return healthy
	}
	return true // Assume healthy until we check
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Find matching route
	var targetURL string
	for path, target := range p.routes {
		if strings.HasPrefix(r.URL.Path, path) {
			targetURL = target
			break
		}
	}

	if targetURL == "" {
		http.NotFound(w, r)
		return
	}

	// Log health status for monitoring, but don't block routing
	// This allows degraded services to still receive traffic
	if !p.isServiceHealthy(targetURL) {
		logger.Warn("[API Gateway] Service %s is unhealthy, but routing anyway (degraded mode)", targetURL)
	}

	// Parse target URL
	target, err := url.Parse(targetURL)
	if err != nil {
		logger.Error("[API Gateway] Invalid target URL %s: %v", targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check if this is a WebSocket upgrade request
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		p.handleWebSocket(w, r, target)
		return
	}

	// Create reverse proxy request
	proxyURL := *r.URL
	proxyURL.Scheme = target.Scheme
	proxyURL.Host = target.Host

	// Create HTTP client with TLS configuration for HTTPS (Traefik routing)
	skipTLSVerify := os.Getenv("SKIP_TLS_VERIFY")
	shouldSkipVerify := skipTLSVerify == "true" || skipTLSVerify == "1"
	
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: shouldSkipVerify, // Skip TLS verification for internal Traefik certs
		},
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), r.Body)
	if err != nil {
		logger.Error("[API Gateway] Failed to create request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy headers, but skip certain headers that should not be forwarded
	skipHeaders := map[string]bool{
		"Connection":        true,
		"Upgrade":           true,
		"Transfer-Encoding": true,
		"Te":                true, // Trailer encoding
		"Trailer":           true,
	}

	for key, values := range r.Header {
		// Skip headers that should not be forwarded
		if skipHeaders[key] {
			continue
		}
		// Host header should be set by the target URL, not copied
		if strings.EqualFold(key, "Host") {
			continue
		}
		// Copy all other headers
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Explicitly set Host header to target host
	req.Host = target.Host

	// Log Authorization header presence for debugging (without logging the actual token)
	if authHeader := req.Header.Get("Authorization"); authHeader != "" {
		logger.Debug("[API Gateway] Forwarding request to %s with Authorization header present", targetURL)
	} else {
		logger.Debug("[API Gateway] Forwarding request to %s without Authorization header", targetURL)
	}

	// Forward request
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("[API Gateway] Failed to forward request to %s: %v", targetURL, err)
		http.Error(w, fmt.Sprintf("Service Unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy response headers, but preserve CORS headers set by the gateway middleware
	corsHeaders := map[string]bool{
		"Access-Control-Allow-Origin":      true,
		"Access-Control-Allow-Methods":     true,
		"Access-Control-Allow-Headers":     true,
		"Access-Control-Allow-Credentials": true,
		"Access-Control-Expose-Headers":    true,
		"Access-Control-Max-Age":           true,
	}

	for key, values := range resp.Header {
		// Skip CORS headers from backend - gateway middleware handles these
		if corsHeaders[key] {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Error("[API Gateway] Failed to copy response body: %v", err)
	}
}

// handleWebSocket handles WebSocket upgrade requests by proxying the connection
func (p *ReverseProxy) handleWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logger.Error("[API Gateway] WebSocket hijacking not supported")
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		logger.Error("[API Gateway] Failed to hijack connection: %v", err)
		return
	}
	defer clientConn.Close()

	// Connect to backend service
	backendAddr := target.Host
	if !strings.Contains(backendAddr, ":") {
		backendAddr += ":80"
		if target.Scheme == "https" {
			backendAddr = strings.Replace(backendAddr, ":80", ":443", 1)
		}
	}

	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		logger.Error("[API Gateway] Failed to connect to backend %s: %v", backendAddr, err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer backendConn.Close()

	// Rewrite the request to point to backend
	proxyURL := *r.URL
	proxyURL.Scheme = target.Scheme
	proxyURL.Host = target.Host

	// Write the request to backend
	reqStr := fmt.Sprintf("%s %s HTTP/1.1\r\n", r.Method, proxyURL.RequestURI())
	backendConn.Write([]byte(reqStr))

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			backendConn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value)))
		}
	}
	backendConn.Write([]byte("\r\n"))

	// Copy request body if present
	if r.Body != nil {
		io.Copy(backendConn, r.Body)
	}

	// Bidirectionally copy data between client and backend
	go func() {
		io.Copy(backendConn, clientConn)
		backendConn.Close()
	}()

	io.Copy(clientConn, backendConn)
}
