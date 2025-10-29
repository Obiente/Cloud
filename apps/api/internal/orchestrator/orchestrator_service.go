package orchestrator

import (
	"context"
	"log"
	"time"

	"api/internal/registry"
)

// OrchestratorService is the main orchestration service that runs continuously
type OrchestratorService struct {
	deploymentManager *DeploymentManager
	serviceRegistry   *registry.ServiceRegistry
	healthChecker     *registry.HealthChecker
	syncInterval      time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewOrchestratorService creates a new orchestrator service
func NewOrchestratorService(strategy string, maxDeploymentsPerNode int, syncInterval time.Duration) (*OrchestratorService, error) {
	deploymentManager, err := NewDeploymentManager(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, err
	}

	serviceRegistry, err := registry.NewServiceRegistry()
	if err != nil {
		return nil, err
	}

	healthChecker := registry.NewHealthChecker(serviceRegistry, 1*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())

	return &OrchestratorService{
		deploymentManager: deploymentManager,
		serviceRegistry:   serviceRegistry,
		healthChecker:     healthChecker,
		syncInterval:      syncInterval,
		ctx:               ctx,
		cancel:            cancel,
	}, nil
}

// Start begins all background orchestration tasks
func (os *OrchestratorService) Start() {
	log.Println("[Orchestrator] Starting orchestration service...")

	// Start periodic sync with Docker
	os.serviceRegistry.StartPeriodicSync(os.ctx, os.syncInterval)
	log.Printf("[Orchestrator] Started periodic sync (interval: %v)", os.syncInterval)

	// Start health checking
	os.healthChecker.Start(os.ctx)
	log.Println("[Orchestrator] Started health checker")

	// Start metrics collection
	go os.collectMetrics()
	log.Println("[Orchestrator] Started metrics collection")

	// Start cleanup tasks
	go os.cleanupTasks()
	log.Println("[Orchestrator] Started cleanup tasks")

	log.Println("[Orchestrator] Orchestration service started successfully")
}

// Stop gracefully stops the orchestrator service
func (os *OrchestratorService) Stop() {
	log.Println("[Orchestrator] Stopping orchestration service...")
	os.cancel()

	if err := os.deploymentManager.Close(); err != nil {
		log.Printf("[Orchestrator] Error closing deployment manager: %v", err)
	}
	if err := os.serviceRegistry.Close(); err != nil {
		log.Printf("[Orchestrator] Error closing service registry: %v", err)
	}

	log.Println("[Orchestrator] Orchestration service stopped")
}

// collectMetrics periodically collects metrics from all deployments
func (os *OrchestratorService) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get all deployments
			locations, err := os.serviceRegistry.GetAllDeployments()
			if err != nil {
				log.Printf("[Orchestrator] Failed to get deployments for metrics: %v", err)
				continue
			}

			log.Printf("[Orchestrator] Collecting metrics for %d deployments", len(locations))

			// Collect metrics for each deployment
			// In a real implementation, this would use Docker stats API
			// and store metrics in the database

		case <-os.ctx.Done():
			return
		}
	}
}

// cleanupTasks runs periodic cleanup operations
func (os *OrchestratorService) cleanupTasks() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("[Orchestrator] Running cleanup tasks...")

			// Clean old metrics (keep last 30 days)
			// if err := database.CleanOldMetrics(30); err != nil {
			// 	log.Printf("[Orchestrator] Failed to clean old metrics: %v", err)
			// }

			// Remove stale deployment locations
			// (handled by periodic sync)

			log.Println("[Orchestrator] Cleanup tasks completed")

		case <-os.ctx.Done():
			return
		}
	}
}

// GetDeploymentManager returns the deployment manager instance
func (os *OrchestratorService) GetDeploymentManager() *DeploymentManager {
	return os.deploymentManager
}

// GetServiceRegistry returns the service registry instance
func (os *OrchestratorService) GetServiceRegistry() *registry.ServiceRegistry {
	return os.serviceRegistry
}

// GetHealthChecker returns the health checker instance
func (os *OrchestratorService) GetHealthChecker() *registry.HealthChecker {
	return os.healthChecker
}
