package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
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
	"auth-service:3002":          "auth-service",
	"organizations-service:3003": "organizations-service",
	"billing-service:3004":       "billing-service",
	"deployments-service:3005":   "deployments-service",
	"gameservers-service:3006":   "gameservers-service",
	"vps-service:3008":           "vps-service",
	"superadmin-service:3011":    "superadmin-service",
	"support-service:3009":       "support-service",
	"audit-service:3010":         "audit-service",
	"dns-service:8053":           "dns-service",
}

func buildServiceRoutes() map[string]string {
	useTraefik := os.Getenv("USE_TRAEFIK_ROUTING")
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}

	routes := make(map[string]string)

	for path, serviceAddr := range baseServiceRoutes {
		if useTraefik == "true" || useTraefik == "1" {
			serviceDomain, ok := serviceDomains[serviceAddr]
			if !ok {
				parts := strings.Split(serviceAddr, ":")
				serviceDomain = parts[0]
			}

			// Always use shared domain (no node subdomains) for proper load balancing
			targetDomain := fmt.Sprintf("%s.%s", serviceDomain, domain)
			routes[path] = fmt.Sprintf("https://%s", targetDomain)
		} else {
			routes[path] = fmt.Sprintf("http://%s", serviceAddr)
		}
	}

	return routes
}

// buildHealthCheckURLs creates a mapping from routing URLs to health check URLs
// When using Traefik routing, health checks bypass Traefik and go directly to services
// This ensures health checks are independent of Traefik's routing decisions
func buildHealthCheckURLs(routingURLs map[string]string, baseRoutes map[string]string) (map[string]string, map[string]string) {
	healthCheckURLs := make(map[string]string)
	baseServiceAddrs := make(map[string]string)
	useTraefik := os.Getenv("USE_TRAEFIK_ROUTING")

	for path, routingURL := range routingURLs {
		baseAddr, exists := baseRoutes[path]
		if !exists {
			continue
		}

		baseServiceAddrs[routingURL] = baseAddr

		if useTraefik == "true" || useTraefik == "1" {
			// Health check directly to service (bypasses Traefik) for independent status
			healthCheckURLs[routingURL] = fmt.Sprintf("http://%s", baseAddr)
		} else {
			healthCheckURLs[routingURL] = routingURL
		}
	}

	return healthCheckURLs, baseServiceAddrs
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	logger.Init()

	logger.Info("=== API Gateway Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	serviceRoutes := buildServiceRoutes()
	useTraefik := os.Getenv("USE_TRAEFIK_ROUTING")
	if useTraefik == "true" || useTraefik == "1" {
		logger.Info("Routing mode: Traefik (HTTPS)")
	} else {
		logger.Info("Routing mode: Internal (HTTP)")
	}

	// Health checks bypass Traefik when using Traefik routing to prevent feedback loops
	healthCheckURLs, baseServiceAddrs := buildHealthCheckURLs(serviceRoutes, baseServiceRoutes)

	mux := http.NewServeMux()
	proxy := &ReverseProxy{
		routes:           serviceRoutes,
		healthCheckURLs: healthCheckURLs,
		baseServiceAddrs: baseServiceAddrs,
	}

	proxy.initHealthChecker()
	logger.Info("✓ Health checker initialized for backend services")
	
	if useTraefik == "true" || useTraefik == "1" {
		logger.Info("✓ Health checks use direct service addresses (bypassing Traefik) for independent status")
	} else {
		logger.Info("✓ Health checks use same URLs as routing")
	}

	for path, targetURL := range serviceRoutes {
		mux.Handle(path, proxy)
		logger.Info("✓ Route registered: %s -> %s", path, targetURL)
	}

	// Verify terminal/ws route is registered
	if terminalRoute, ok := serviceRoutes["/terminal/ws"]; ok {
		logger.Info("✓ Terminal WebSocket route verified: /terminal/ws -> %s", terminalRoute)
	} else {
		logger.Error("✗ Terminal WebSocket route NOT FOUND in serviceRoutes!")
	}

	// Health check endpoint - always returns healthy (gateway health independent of backends)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"api-gateway"}`))
	})

	// Detailed health endpoint for monitoring/debugging
	mux.HandleFunc("/health/detailed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		proxy.healthMutex.RLock()
		allHealthy := true
		unhealthyServices := []string{}
		healthyServices := []string{}
		serviceDetails := make(map[string]interface{})
		checkedCount := 0
		for serviceURL, serviceHealth := range proxy.healthStatus {
			checkedCount++
			if serviceHealth == nil || !serviceHealth.Healthy {
				allHealthy = false
				unhealthyServices = append(unhealthyServices, serviceURL)
			} else {
				healthyServices = append(healthyServices, serviceURL)
			}
			if serviceHealth != nil {
				serviceDetails[serviceURL] = map[string]interface{}{
					"healthy":       serviceHealth.Healthy,
					"replica_count": serviceHealth.ReplicaCount,
					"replicas":      serviceHealth.Replicas,
				}
			}
		}
		proxy.healthMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")

		if checkedCount == 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy","service":"api-gateway","message":"health checks not yet initialized","backends_checked":0}`))
			return
		}

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

		response := map[string]interface{}{
			"status":               map[bool]string{true: "healthy", false: "degraded"}[allHealthy],
			"service":              "api-gateway",
			"all_backends_healthy": allHealthy,
			"healthy_backends":     healthyServices,
			"unhealthy_backends":   unhealthyServices,
			"total_backends":       checkedCount,
			"services":             serviceDetails,
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
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

	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	var handler http.Handler = h2cHandler
	handler = middleware.CORSHandler(handler)
	handler = middleware.RequestLogger(handler)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== API Gateway Ready - Listening on %s ===", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

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

// ReplicaHealth tracks health status of individual replicas
type ReplicaHealth struct {
	ReplicaID string
	Healthy   bool
	LastSeen  time.Time
}

// ServiceHealth tracks health status for a service and its replicas
type ServiceHealth struct {
	Healthy      bool
	Replicas     map[string]*ReplicaHealth // Map of replica ID -> health status
	ReplicaCount int                       // Number of unique healthy replicas
}

// ReverseProxy handles routing requests to backend services
type ReverseProxy struct {
	routes            map[string]string // Path -> routing URL (may go through Traefik)
	healthCheckURLs   map[string]string // Routing URL -> health check URL (always direct to service)
	baseServiceAddrs  map[string]string // Routing URL -> base service address (for health checks)
	healthStatus      map[string]*ServiceHealth // Tracks health status of each backend service and its replicas
	healthMutex       sync.RWMutex
}

// initHealthChecker starts background health checks for all backend services
// Health checks use direct service addresses (bypassing Traefik when using Traefik routing)
// This ensures health status is independent of Traefik's routing decisions
func (p *ReverseProxy) initHealthChecker() {
	p.healthStatus = make(map[string]*ServiceHealth)
	if p.healthCheckURLs == nil {
		p.healthCheckURLs = make(map[string]string)
	}
	if p.baseServiceAddrs == nil {
		p.baseServiceAddrs = make(map[string]string)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Remove stale replicas (not seen for 2 minutes)
		cleanupTicker := time.NewTicker(30 * time.Second)
		defer cleanupTicker.Stop()

		for {
			select {
			case <-ticker.C:
				p.checkAllServicesHealth()
			case <-cleanupTicker.C:
				p.cleanupStaleReplicas()
			}
		}
	}()

	p.checkAllServicesHealth()
}

// checkAllServicesHealth checks health of all backend services in parallel
// Tracks individual replica IDs to determine actual replica count
// Uses direct service addresses for health checks (bypassing Traefik when using Traefik routing)
// This ensures health checks are independent of Traefik's routing decisions
func (p *ReverseProxy) checkAllServicesHealth() {
	var wg sync.WaitGroup

	for _, routingURL := range p.routes {
		healthCheckURL, exists := p.healthCheckURLs[routingURL]
		if !exists {
			healthCheckURL = routingURL
		}

		target, err := url.Parse(healthCheckURL)
		if err != nil {
			logger.Warn("[API Gateway] Failed to parse health check URL %s: %v", healthCheckURL, err)
			continue
		}

		healthURL := fmt.Sprintf("%s://%s/health", target.Scheme, target.Host)
		logger.Debug("[API Gateway] Health checking service: routing=%s, health_check=%s", routingURL, healthURL)

		wg.Add(1)
		go func(url, routing string) {
			defer wg.Done()
			p.checkServiceHealthWithReplicas(url, routing)
		}(healthURL, routingURL)
	}

	wg.Wait()
}

// cleanupStaleReplicas removes replicas that haven't been seen recently
// This handles cases where replicas are removed during upgrades
func (p *ReverseProxy) cleanupStaleReplicas() {
	p.healthMutex.Lock()
	defer p.healthMutex.Unlock()

	staleThreshold := 2 * time.Minute // Remove replicas not seen for 2 minutes
	now := time.Now()

	for serviceURL, serviceHealth := range p.healthStatus {
		for replicaID, replica := range serviceHealth.Replicas {
			if now.Sub(replica.LastSeen) > staleThreshold {
				logger.Debug("[API Gateway] Removing stale replica %s from service %s (not seen for %v)", replicaID, serviceURL, now.Sub(replica.LastSeen))
				delete(serviceHealth.Replicas, replicaID)
			}
		}
		serviceHealth.ReplicaCount = len(serviceHealth.Replicas)
		serviceHealth.Healthy = false
		for _, replica := range serviceHealth.Replicas {
			if replica.Healthy {
				serviceHealth.Healthy = true
				break
			}
		}
	}
}

// checkServiceHealthWithReplicas checks service health by sampling multiple replicas
// Tracks replica IDs to determine actual replica count and detect changes
func (p *ReverseProxy) checkServiceHealthWithReplicas(healthURL string, serviceURL string) {
	// Determine number of checks based on current known replica count
	// Start with a reasonable number, then adjust based on discovered replicas
	p.healthMutex.RLock()
	serviceHealth, exists := p.healthStatus[serviceURL]
	knownReplicaCount := 0
	if exists && serviceHealth != nil {
		knownReplicaCount = serviceHealth.ReplicaCount
	}
	p.healthMutex.RUnlock()

	// Sample known count + 2 to discover new replicas (min 3, max 10)
	numChecks := knownReplicaCount + 2
	if numChecks < 3 {
		numChecks = 3
	}
	if numChecks > 10 {
		numChecks = 10
	}

	// DNS service typically has 1 replica, so only check once
	if strings.Contains(healthURL, "dns-service") {
		numChecks = 1
	}

	discoveredReplicas := make(map[string]*ReplicaHealth)
	var lastError error
	successfulChecks := 0
	var replicaMutex sync.Mutex

	// Run health checks in parallel to discover replicas faster
	var checkWg sync.WaitGroup
	checkWg.Add(numChecks)

	for i := 0; i < numChecks; i++ {
		go func(attempt int) {
			defer checkWg.Done()

			healthy, replicaID, err := p.checkServiceHealth(healthURL)
			if err != nil {
				replicaMutex.Lock()
				lastError = err
				replicaMutex.Unlock()
				logger.Debug("[API Gateway] Health check failed for %s (attempt %d/%d): %v", healthURL, attempt+1, numChecks, err)
				return
			}

			replicaMutex.Lock()
			successfulChecks++

			if replicaID != "" {
				if _, exists := discoveredReplicas[replicaID]; !exists {
					discoveredReplicas[replicaID] = &ReplicaHealth{
						ReplicaID: replicaID,
						Healthy:   healthy,
						LastSeen:  time.Now(),
					}
				} else {
					discoveredReplicas[replicaID].Healthy = healthy
					discoveredReplicas[replicaID].LastSeen = time.Now()
				}
			} else if healthy {
				// Backwards compatibility: no replica ID returned
				tempID := fmt.Sprintf("unknown-%d", attempt)
				if _, exists := discoveredReplicas[tempID]; !exists {
					discoveredReplicas[tempID] = &ReplicaHealth{
						ReplicaID: tempID,
						Healthy:   true,
						LastSeen:  time.Now(),
					}
				}
			}
			replicaMutex.Unlock()
		}(i)
	}

	checkWg.Wait()

	p.healthMutex.Lock()
	defer p.healthMutex.Unlock()

	if p.healthStatus[serviceURL] == nil {
		p.healthStatus[serviceURL] = &ServiceHealth{
			Replicas: make(map[string]*ReplicaHealth),
		}
	}

	serviceHealth = p.healthStatus[serviceURL]

	for replicaID, replica := range discoveredReplicas {
		if existing, exists := serviceHealth.Replicas[replicaID]; exists {
			existing.Healthy = replica.Healthy
			existing.LastSeen = replica.LastSeen
		} else {
			serviceHealth.Replicas[replicaID] = replica
			logger.Debug("[API Gateway] Discovered new replica %s for service %s", replicaID, serviceURL)
		}
	}

	serviceHealth.ReplicaCount = len(serviceHealth.Replicas)
	serviceHealth.Healthy = false

	if successfulChecks > 0 {
		for _, replica := range serviceHealth.Replicas {
			if replica.Healthy {
				serviceHealth.Healthy = true
				break
			}
		}
		// Backwards compatibility: successful checks but no replica IDs
		if !serviceHealth.Healthy && successfulChecks > 0 && len(discoveredReplicas) == 0 {
			logger.Debug("[API Gateway] Service %s had successful health checks but no replica IDs, assuming healthy", serviceURL)
			serviceHealth.Healthy = true
		}
	} else {
		// All checks failed - keep existing status if we have tracked replicas
		if serviceHealth.ReplicaCount == 0 {
			serviceHealth.Healthy = false
		} else {
			for _, replica := range serviceHealth.Replicas {
				if replica.Healthy {
					serviceHealth.Healthy = true
					break
				}
			}
		}
	}

	if !serviceHealth.Healthy && lastError != nil && serviceHealth.ReplicaCount == 0 {
		logger.Warn("[API Gateway] Service %s is unhealthy (all health checks failed, no replicas tracked): %v", serviceURL, lastError)
	} else if serviceHealth.ReplicaCount > 0 {
		logger.Debug("[API Gateway] Service %s: %d replica(s) tracked, healthy: %v", serviceURL, serviceHealth.ReplicaCount, serviceHealth.Healthy)
	}
}

// HealthResponse represents the health check response from services
type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	ReplicaID string                 `json:"replica_id"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// checkServiceHealth checks service health. Returns (isHealthy, replicaID, error).
// Health checks are informational and don't block routing. When using Traefik routing,
// health checks bypass Traefik to ensure independent status.
func (p *ReverseProxy) checkServiceHealth(healthURL string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	targetURL, err := url.Parse(healthURL)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse health URL: %w", err)
	}

	var transport *http.Transport
	if targetURL.Scheme == "https" {
		skipTLSVerify := os.Getenv("SKIP_TLS_VERIFY")
		shouldSkipVerify := skipTLSVerify == "true" || skipTLSVerify == "1"
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: shouldSkipVerify,
			},
			DialContext: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).DialContext,
		}
	} else {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).DialContext,
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 and 503 as valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return false, "", fmt.Errorf("unexpected status code: %d (expected 200 or 503)", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		// Backwards compatibility: no JSON response
		isHealthy := resp.StatusCode == http.StatusOK
		return isHealthy, "", nil
	}

	isHealthy := healthResp.Status == "healthy"
	return isHealthy, healthResp.ReplicaID, nil
}

// isServiceHealthy checks if a service is currently healthy (informational only, doesn't block routing)
func (p *ReverseProxy) isServiceHealthy(targetURL string) bool {
	p.healthMutex.RLock()
	defer p.healthMutex.RUnlock()

	// Optimistic: default to healthy to allow routing on startup
	if serviceHealth, exists := p.healthStatus[targetURL]; exists && serviceHealth != nil {
		if serviceHealth.ReplicaCount > 0 {
			return serviceHealth.Healthy
		}
		return true
	}
	return true
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgradeHeader := r.Header.Get("Upgrade")
	connectionHeader := r.Header.Get("Connection")
	isWebSocketRequest := upgradeHeader != "" || strings.Contains(strings.ToLower(connectionHeader), "upgrade")

	if isWebSocketRequest {
		logger.Info("[API Gateway] WebSocket request: method=%s, path=%s, Upgrade=%s, Connection=%s, routes_count=%d",
			r.Method, r.URL.Path, upgradeHeader, connectionHeader, len(p.routes))
	} else {
		logger.Debug("[API Gateway] Request: method=%s, path=%s", r.Method, r.URL.Path)
	}

	// Match longer/more specific paths first (e.g., /dns/push/batch before /dns/push)
	var targetURL string
	var matchedPath string

	type routeEntry struct {
		path   string
		target string
	}
	routes := make([]routeEntry, 0, len(p.routes))
	for path, target := range p.routes {
		routes = append(routes, routeEntry{path, target})
	}

	// Sort by path length (descending) to match longer/more specific paths first
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].path) > len(routes[j].path)
	})

	logger.Debug("[API Gateway] Checking routes for path: %s (total routes: %d)", r.URL.Path, len(routes))
	for _, route := range routes {
		matches := strings.HasPrefix(r.URL.Path, route.path)
		logger.Debug("[API Gateway] Checking route: %s -> %s (matches: %v)", route.path, route.target, matches)
		if matches {
			targetURL = route.target
			matchedPath = route.path
			logger.Debug("[API Gateway] Route matched: %s -> %s", route.path, route.target)
			break
		}
	}

	if targetURL == "" {
		logger.Warn("[API Gateway] No route found for path: %s (checked %d routes)", r.URL.Path, len(routes))
		availableRoutes := make([]string, 0, len(routes))
		for _, route := range routes {
			availableRoutes = append(availableRoutes, route.path)
		}
		logger.Debug("[API Gateway] Available routes: %v", availableRoutes)
		http.NotFound(w, r)
		return
	}

	logger.Debug("[API Gateway] Routing %s -> %s (matched path: %s)", r.URL.Path, targetURL, matchedPath)

	// Health status is informational - Traefik handles routing decisions
	if !p.isServiceHealthy(targetURL) {
		logger.Warn("[API Gateway] Service %s appears unhealthy, but routing anyway - Traefik will handle load balancing", targetURL)
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		logger.Error("[API Gateway] Invalid target URL %s: %v", targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check for WebSocket upgrade
	upgradeHeaderValue := strings.ToLower(r.Header.Get("Upgrade"))
	connectionHeaderValue := strings.ToLower(r.Header.Get("Connection"))
	isWebSocket := upgradeHeaderValue == "websocket" &&
		(strings.Contains(connectionHeaderValue, "upgrade") || connectionHeaderValue == "upgrade")

	if isWebSocket {
		logger.Debug("[API Gateway] WebSocket upgrade detected for %s -> %s", r.URL.Path, targetURL)
		p.handleWebSocket(w, r, target)
		return
	}

	proxyURL := *r.URL
	proxyURL.Scheme = target.Scheme
	proxyURL.Host = target.Host

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

	req, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL.String(), r.Body)
	if err != nil {
		logger.Error("[API Gateway] Failed to create request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	skipHeaders := map[string]bool{
		"Connection":        true,
		"Upgrade":           true,
		"Transfer-Encoding": true,
		"Te":                true, // Trailer encoding
		"Trailer":           true,
	}

	for key, values := range r.Header {
		if skipHeaders[key] {
			continue
		}
		if strings.EqualFold(key, "Host") {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Host = target.Host

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

	// Skip CORS headers from backend - gateway middleware handles these
	corsHeaders := map[string]bool{
		"Access-Control-Allow-Origin":      true,
		"Access-Control-Allow-Methods":     true,
		"Access-Control-Allow-Headers":     true,
		"Access-Control-Allow-Credentials": true,
		"Access-Control-Expose-Headers":    true,
		"Access-Control-Max-Age":           true,
	}

	for key, values := range resp.Header {
		if corsHeaders[key] {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Error("[API Gateway] Failed to copy response body: %v", err)
	}
}

// handleWebSocket handles WebSocket upgrade requests by proxying the connection
func (p *ReverseProxy) handleWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	logger.Info("[API Gateway] Handling WebSocket upgrade: %s -> %s", r.URL.Path, target.String())

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

	logger.Debug("[API Gateway] Connection hijacked, connecting to backend: %s", target.Host)

	backendAddr := target.Host
	if !strings.Contains(backendAddr, ":") {
		if target.Scheme == "https" {
			backendAddr += ":443"
		} else {
			backendAddr += ":80"
		}
	}

	var backendConn net.Conn
	if target.Scheme == "https" {
		skipTLSVerify := os.Getenv("SKIP_TLS_VERIFY")
		shouldSkipVerify := skipTLSVerify == "true" || skipTLSVerify == "1"

		tcpConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
		if err != nil {
			logger.Error("[API Gateway] Failed to connect to backend %s: %v", backendAddr, err)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}

		hostname := target.Host
		if idx := strings.Index(hostname, ":"); idx != -1 {
			hostname = hostname[:idx]
		}

		tlsConfig := &tls.Config{
			ServerName:         hostname,
			InsecureSkipVerify: shouldSkipVerify,
		}
		backendConn = tls.Client(tcpConn, tlsConfig)
	} else {
		backendConn, err = net.DialTimeout("tcp", backendAddr, 10*time.Second)
		if err != nil {
			logger.Error("[API Gateway] Failed to connect to backend %s: %v", backendAddr, err)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
	}
	defer backendConn.Close()

	proxyURL := *r.URL
	proxyURL.Scheme = target.Scheme
	proxyURL.Host = target.Host

	reqStr := fmt.Sprintf("%s %s HTTP/1.1\r\n", r.Method, proxyURL.RequestURI())
	if _, err := backendConn.Write([]byte(reqStr)); err != nil {
		logger.Error("[API Gateway] Failed to write request line: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	hostHeaderSet := false
	for key, values := range r.Header {
		if strings.EqualFold(key, "Host") {
			backendConn.Write([]byte(fmt.Sprintf("Host: %s\r\n", target.Host)))
			hostHeaderSet = true
			continue
		}
		for _, value := range values {
			if _, err := backendConn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
				logger.Error("[API Gateway] Failed to write header %s: %v", key, err)
				clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
				return
			}
		}
	}
	if !hostHeaderSet {
		backendConn.Write([]byte(fmt.Sprintf("Host: %s\r\n", target.Host)))
	}
	backendConn.Write([]byte("\r\n"))

	if r.Body != nil {
		if _, err := io.Copy(backendConn, r.Body); err != nil {
			logger.Error("[API Gateway] Failed to copy request body: %v", err)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
	}

	// Forward WebSocket handshake response to client
	buf := make([]byte, 4096)
	n, err := backendConn.Read(buf)
	if err != nil && err != io.EOF {
		logger.Error("[API Gateway] Failed to read WebSocket handshake response: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	if n > 0 {
		if _, err := clientConn.Write(buf[:n]); err != nil {
			logger.Error("[API Gateway] Failed to forward WebSocket handshake response: %v", err)
			return
		}
	}

	go func() {
		io.Copy(backendConn, clientConn)
		backendConn.Close()
	}()

	io.Copy(clientConn, backendConn)
}
