package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"api/internal/database"
	"api/internal/registry"
)

// DeploymentRouter routes incoming requests to the appropriate deployment containers
type DeploymentRouter struct {
	registry      *registry.ServiceRegistry
	proxyCache    sync.Map // domain -> *httputil.ReverseProxy
	roundRobinIdx sync.Map // domain -> current index for round-robin
	mu            sync.RWMutex
}

// NewDeploymentRouter creates a new deployment router
func NewDeploymentRouter(registry *registry.ServiceRegistry) *DeploymentRouter {
	return &DeploymentRouter{
		registry: registry,
	}
}

// ServeHTTP handles incoming HTTP requests and routes them to deployments
func (dr *DeploymentRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	domain := r.Host

	log.Printf("[Router] Incoming request for domain: %s, path: %s", domain, r.URL.Path)

	// Get deployment routing configuration
	routing, err := dr.registry.GetDeploymentByDomain(domain)
	if err != nil {
		log.Printf("[Router] No deployment found for domain %s: %v", domain, err)
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Get deployment locations
	locations, err := dr.registry.GetDeploymentLocations(routing.DeploymentID)
	if err != nil || len(locations) == 0 {
		log.Printf("[Router] No active instances found for deployment %s", routing.DeploymentID)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Filter only healthy instances
	healthyLocations := []database.DeploymentLocation{}
	for _, loc := range locations {
		if loc.Status == "running" && (loc.HealthStatus == "healthy" || loc.HealthStatus == "unknown") {
			healthyLocations = append(healthyLocations, loc)
		}
	}

	if len(healthyLocations) == 0 {
		log.Printf("[Router] No healthy instances found for deployment %s", routing.DeploymentID)
		http.Error(w, "All instances unhealthy", http.StatusServiceUnavailable)
		return
	}

	// Select target instance based on load balancing algorithm
	targetLocation := dr.selectTarget(healthyLocations, routing.LoadBalancerAlgo, domain)

	// Create or get reverse proxy
	proxy, err := dr.getOrCreateProxy(targetLocation, routing)
	if err != nil {
		log.Printf("[Router] Failed to create proxy: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add custom headers
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Obiente-Deployment-ID", routing.DeploymentID)
	r.Header.Set("X-Obiente-Node-ID", targetLocation.NodeID)

	log.Printf("[Router] Routing %s to container %s on node %s",
		domain, targetLocation.ContainerID[:12], targetLocation.NodeHostname)

	// Serve the request
	proxy.ServeHTTP(w, r)
}

// selectTarget selects a target instance based on the load balancing algorithm
func (dr *DeploymentRouter) selectTarget(locations []database.DeploymentLocation, algo string, domain string) database.DeploymentLocation {
	switch algo {
	case "round-robin":
		return dr.roundRobin(locations, domain)
	case "least-conn":
		return dr.leastConnections(locations)
	case "ip-hash":
		return dr.ipHash(locations, domain)
	default:
		return dr.roundRobin(locations, domain)
	}
}

// roundRobin implements round-robin load balancing
func (dr *DeploymentRouter) roundRobin(locations []database.DeploymentLocation, domain string) database.DeploymentLocation {
	idx := 0
	if val, ok := dr.roundRobinIdx.Load(domain); ok {
		idx = val.(int)
	}

	target := locations[idx%len(locations)]
	dr.roundRobinIdx.Store(domain, (idx+1)%len(locations))

	return target
}

// leastConnections selects the instance with least CPU usage (proxy for connections)
func (dr *DeploymentRouter) leastConnections(locations []database.DeploymentLocation) database.DeploymentLocation {
	minCPU := float64(100.0)
	selectedIdx := 0

	for i, loc := range locations {
		if loc.CPUUsage < minCPU {
			minCPU = loc.CPUUsage
			selectedIdx = i
		}
	}

	return locations[selectedIdx]
}

// ipHash uses consistent hashing based on client IP
func (dr *DeploymentRouter) ipHash(locations []database.DeploymentLocation, domain string) database.DeploymentLocation {
	// Simple hash of domain (in production, use actual client IP)
	hash := 0
	for _, c := range domain {
		hash += int(c)
	}
	return locations[hash%len(locations)]
}

// getOrCreateProxy creates or retrieves a reverse proxy for a location
func (dr *DeploymentRouter) getOrCreateProxy(location database.DeploymentLocation, routing *database.DeploymentRouting) (*httputil.ReverseProxy, error) {
	cacheKey := fmt.Sprintf("%s:%d", location.NodeIP, location.Port)

	// Try cache first
	if cached, ok := dr.proxyCache.Load(cacheKey); ok {
		return cached.(*httputil.ReverseProxy), nil
	}

	// Create new proxy
	targetURL := &url.URL{
		Scheme: routing.Protocol,
		Host:   fmt.Sprintf("%s:%d", location.NodeIP, location.Port),
	}

	if location.NodeIP == "" || location.NodeIP == "0.0.0.0" {
		targetURL.Host = fmt.Sprintf("localhost:%d", location.Port)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize proxy behavior
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[Router] Proxy error for %s: %v", location.DeploymentID, err)

		// Mark instance as potentially unhealthy
		dr.registry.UpdateDeploymentHealth(location.ContainerID, "unhealthy")

		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Set timeouts
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Cache proxy
	dr.proxyCache.Store(cacheKey, proxy)

	return proxy, nil
}

// UpdateRouting updates routing configuration for a deployment
func (dr *DeploymentRouter) UpdateRouting(ctx context.Context, deploymentID string, domain string, targetPort int) error {
	routing := &database.DeploymentRouting{
		DeploymentID:     deploymentID,
		Domain:           domain,
		TargetPort:       targetPort,
		Protocol:         "http",
		LoadBalancerAlgo: "round-robin",
		SSLEnabled:       true,
		SSLCertResolver:  "letsencrypt",
		Middleware:       "{}", // Empty JSON object for jsonb field
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := database.UpsertDeploymentRouting(routing); err != nil {
		return fmt.Errorf("failed to update routing: %w", err)
	}

	// Clear proxy cache for this domain
	dr.clearCacheForDomain(domain)

	log.Printf("[Router] Updated routing for domain %s to deployment %s", domain, deploymentID)
	return nil
}

// RemoveRouting removes routing configuration for a deployment
func (dr *DeploymentRouter) RemoveRouting(ctx context.Context, deploymentID string) error {
	var routing database.DeploymentRouting
	if err := database.DB.Where("deployment_id = ?", deploymentID).First(&routing).Error; err != nil {
		return err
	}

	if err := database.DB.Delete(&routing).Error; err != nil {
		return fmt.Errorf("failed to remove routing: %w", err)
	}

	// Clear proxy cache
	dr.clearCacheForDomain(routing.Domain)

	log.Printf("[Router] Removed routing for deployment %s", deploymentID)
	return nil
}

// clearCacheForDomain removes cached proxies for a domain
func (dr *DeploymentRouter) clearCacheForDomain(domain string) {
	dr.proxyCache.Range(func(key, value interface{}) bool {
		// In production, maintain a domain -> cache key mapping
		dr.proxyCache.Delete(key)
		return true
	})
}

// GetRoutingStats returns routing statistics
func (dr *DeploymentRouter) GetRoutingStats() *RoutingStats {
	stats := &RoutingStats{
		Timestamp: time.Now(),
	}

	// Count total routes
	var routeCount int64
	database.DB.Model(&database.DeploymentRouting{}).Count(&routeCount)
	stats.TotalRoutes = int(routeCount)

	// Count cached proxies
	cacheCount := 0
	dr.proxyCache.Range(func(key, value interface{}) bool {
		cacheCount++
		return true
	})
	stats.CachedProxies = cacheCount

	return stats
}

// RoutingStats holds routing statistics
type RoutingStats struct {
	TotalRoutes   int       `json:"total_routes"`
	CachedProxies int       `json:"cached_proxies"`
	Timestamp     time.Time `json:"timestamp"`
}
