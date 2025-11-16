package registry

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

// HealthChecker periodically checks the health of all deployments
type HealthChecker struct {
	registry   *ServiceRegistry
	httpClient *http.Client
	interval   time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(registry *ServiceRegistry, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		interval: interval,
	}
}

// Start begins periodic health checking
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				hc.checkAllDeployments(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// checkAllDeployments checks health of all running deployments
func (hc *HealthChecker) checkAllDeployments(ctx context.Context) {
	locations, err := hc.registry.GetAllDeployments()
	if err != nil {
		log.Printf("[HealthChecker] Failed to get deployments: %v", err)
		return
	}

	log.Printf("[HealthChecker] Checking health of %d deployments", len(locations))

	for _, location := range locations {
		go hc.checkDeploymentHealth(ctx, &location)
	}
}

// checkDeploymentHealth checks the health of a single deployment
func (hc *HealthChecker) checkDeploymentHealth(ctx context.Context, location *database.DeploymentLocation) {
	if location.Port == 0 {
		return // No port configured, skip health check
	}

	// Construct health check URL
	healthURL := fmt.Sprintf("http://%s:%d/health", location.NodeIP, location.Port)
	if location.NodeIP == "" {
		healthURL = fmt.Sprintf("http://localhost:%d/health", location.Port)
	}

	// Perform health check
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		log.Printf("[HealthChecker] Failed to create request for %s: %v", location.DeploymentID, err)
		return
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		// Health check failed
		hc.registry.UpdateDeploymentHealth(location.ContainerID, "unhealthy")
		log.Printf("[HealthChecker] Deployment %s is unhealthy: %v", location.DeploymentID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		hc.registry.UpdateDeploymentHealth(location.ContainerID, "healthy")
	} else {
		hc.registry.UpdateDeploymentHealth(location.ContainerID, "unhealthy")
		log.Printf("[HealthChecker] Deployment %s returned status %d", location.DeploymentID, resp.StatusCode)
	}
}

// CheckDeployment performs an immediate health check on a specific deployment
func (hc *HealthChecker) CheckDeployment(ctx context.Context, deploymentID string) (bool, error) {
	locations, err := hc.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return false, err
	}

	if len(locations) == 0 {
		return false, fmt.Errorf("no locations found for deployment %s", deploymentID)
	}

	// Check first location (could check all and aggregate)
	location := locations[0]

	if location.Port == 0 {
		return false, fmt.Errorf("no port configured for deployment")
	}

	healthURL := fmt.Sprintf("http://%s:%d/health", location.NodeIP, location.Port)
	if location.NodeIP == "" {
		healthURL = fmt.Sprintf("http://localhost:%d/health", location.Port)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
