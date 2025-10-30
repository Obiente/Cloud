package orchestrator

import (
	"context"
	"log"
	"time"

	"api/internal/registry"
	"api/internal/database"
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

	// Start usage aggregation (hourly)
	go os.aggregateUsage()
	log.Println("[Orchestrator] Started usage aggregation")

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
			// Placeholder
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
			log.Println("[Orchestrator] Cleanup tasks completed")
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) aggregateUsage() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Println("[Orchestrator] Aggregating usage...")
			// Very simplified aggregation: update peak deployments per org
			type row struct{ OrganizationID string; Count int }
			var rows []row
			database.DB.Table("deployment_locations dl").
				Select("d.organization_id, COUNT(*) as count").
				Joins("JOIN deployments d ON d.id = dl.deployment_id").
				Where("dl.status = ?", "running").
				Group("d.organization_id").Scan(&rows)
			month := time.Now().Format("2006-01")
			week := time.Now().Format("2006-01") + "-W" // simplistic placeholder
			for _, r := range rows {
				_ = database.DB.Where("organization_id = ? AND month = ?", r.OrganizationID, month).
					Assign(&database.UsageMonthly{DeploymentsActivePeak: r.Count}).
					FirstOrCreate(&database.UsageMonthly{OrganizationID: r.OrganizationID, Month: month})
				_ = database.DB.Where("organization_id = ? AND week = ?", r.OrganizationID, week).
					Assign(&database.UsageWeekly{DeploymentsActivePeak: r.Count}).
					FirstOrCreate(&database.UsageWeekly{OrganizationID: r.OrganizationID, Week: week})
			}
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
