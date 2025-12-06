package orchestrator

// This package contains orchestrator-service specific code.
// It imports the shared orchestrator package for DeploymentManager and other shared components.

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	shared "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/registry"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

// Core orchestrator service types and initialization

type OrchestratorService struct {
	deploymentManager *shared.DeploymentManager
	serviceRegistry   *registry.ServiceRegistry
	healthChecker     *registry.HealthChecker
	metricsStreamer   *shared.MetricsStreamer
	rollbackMonitor   *RollbackMonitor
	syncInterval      time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
}

// containerStats is now defined in shared package as ContainerStats

type MicroserviceConfig struct {
	Name     string
	Port     int
	BaseHost string // Base hostname without node prefix (e.g., "auth-service")
}

func NewOrchestratorService(strategy string, maxDeploymentsPerNode int, syncInterval time.Duration) (*OrchestratorService, error) {
	deploymentManager, err := shared.NewDeploymentManager(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, err
	}

	serviceRegistry, err := registry.NewServiceRegistry()
	if err != nil {
		return nil, err
	}

	healthChecker := registry.NewHealthChecker(serviceRegistry, 1*time.Minute)
	metricsStreamer := shared.NewMetricsStreamer(serviceRegistry)

	// Create rollback monitor (may fail if Docker is not available, but that's OK)
	rollbackMonitor, err := NewRollbackMonitor()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to create rollback monitor: %v (rollback notifications will be disabled)", err)
		rollbackMonitor = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Register global metrics streamer for access from other services
	shared.SetGlobalMetricsStreamer(metricsStreamer)

	service := &OrchestratorService{
		deploymentManager: deploymentManager,
		serviceRegistry:   serviceRegistry,
		healthChecker:     healthChecker,
		metricsStreamer:   metricsStreamer,
		rollbackMonitor:   rollbackMonitor,
		syncInterval:      syncInterval,
		ctx:               ctx,
		cancel:            cancel,
	}

	// Register global orchestrator service for access from other services
	// Note: We need to convert to shared.OrchestratorService interface
	shared.SetGlobalOrchestratorService(service)

	return service, nil
}

func (os *OrchestratorService) Start() {
	logger.Info("[Orchestrator] Starting orchestration service...")

	// Start periodic sync with Docker
	os.serviceRegistry.StartPeriodicSync(os.ctx, os.syncInterval)
	logger.Debug("[Orchestrator] Started periodic sync (interval: %v)", os.syncInterval)

	// Start health checking
	os.healthChecker.Start(os.ctx)
	logger.Debug("[Orchestrator] Started health checker")

	// Start metrics streaming (handles live collection and periodic storage)
	os.metricsStreamer.Start()
	logger.Debug("[Orchestrator] Started metrics streaming")

	// Backfill missing hourly aggregates on startup
	go os.backfillMissingHourlyAggregates()
	logger.Debug("[Orchestrator] Started backfill of missing hourly aggregates")

	// Start cleanup tasks
	go os.cleanupTasks()
	logger.Debug("[Orchestrator] Started cleanup tasks")

	// Start usage aggregation (hourly)
	go os.aggregateUsage()
	logger.Debug("[Orchestrator] Started usage aggregation")

	// Start VPS metrics collection (every 5 minutes)
	go os.collectVPSMetrics()
	logger.Debug("[Orchestrator] Started VPS metrics collection")

	// Start storage updates (every 5 minutes)
	go os.updateStoragePeriodically()
	logger.Debug("[Orchestrator] Started periodic storage updates")

	// Start build history cleanup (daily)
	go os.cleanupBuildHistory()
	logger.Debug("[Orchestrator] Started build history cleanup")

	// Start stray container cleanup (every 6 hours)
	go os.cleanupStrayContainers()
	logger.Debug("[Orchestrator] Started stray container cleanup")

	// Start rollback monitor (if available)
	if os.rollbackMonitor != nil {
		os.rollbackMonitor.Start()
		logger.Debug("[Orchestrator] Started rollback monitor")
	}

	// Start microservice Traefik label sync (every 30 seconds)
	go os.syncMicroserviceTraefikLabels()
	logger.Debug("[Orchestrator] Started microservice Traefik label sync")

	// Start periodic node metadata sync (every 5 minutes) to update resource usage
	go os.syncNodeMetadataPeriodically()
	logger.Debug("[Orchestrator] Started periodic node metadata sync")

	// Restore running deployments from database on startup
	go os.restoreRunningDeployments()
	logger.Debug("[Orchestrator] Started restoration of running deployments")

	logger.Info("[Orchestrator] Orchestration service started successfully")
}

// syncNodeMetadataPeriodically periodically syncs node metadata to update resource usage
func (os *OrchestratorService) syncNodeMetadataPeriodically() {
	// Run every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	ctx, cancel := context.WithTimeout(os.ctx, 30*time.Second)
	err := os.deploymentManager.SyncNodeMetadata(ctx)
	cancel()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to sync node metadata on startup: %v", err)
	} else {
		logger.Debug("[Orchestrator] Node metadata synced on startup")
	}

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(os.ctx, 30*time.Second)
			err := os.deploymentManager.SyncNodeMetadata(ctx)
			cancel()
			if err != nil {
				logger.Warn("[Orchestrator] Failed to sync node metadata: %v", err)
			} else {
				logger.Debug("[Orchestrator] Node metadata synced successfully")
			}
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) Stop() {
	logger.Info("[Orchestrator] Stopping orchestration service...")
	os.cancel()

	if err := os.deploymentManager.Close(); err != nil {
		logger.Warn("[Orchestrator] Error closing deployment manager: %v", err)
	}
	if err := os.serviceRegistry.Close(); err != nil {
		logger.Warn("[Orchestrator] Error closing service registry: %v", err)
	}

	os.metricsStreamer.Stop()
	logger.Debug("[Orchestrator] Stopped metrics streamer")

	if os.rollbackMonitor != nil {
		os.rollbackMonitor.Stop()
		logger.Debug("[Orchestrator] Stopped rollback monitor")
	}

	logger.Info("[Orchestrator] Orchestration service stopped")
}

func (os *OrchestratorService) GetDeploymentManager() *shared.DeploymentManager {
	return os.deploymentManager
}

func (os *OrchestratorService) GetServiceRegistry() interface{} {
	return os.serviceRegistry
}

func (os *OrchestratorService) GetHealthChecker() interface{} {
	return os.healthChecker
}

func (os *OrchestratorService) GetMetricsStreamer() *shared.MetricsStreamer {
	return os.metricsStreamer
}

func (os *OrchestratorService) getEnvOrDefault(key string, defaultValue string) string {
	if value := os.getEnv(key); value != "" {
		return value
	}
	return defaultValue
}

func (orch *OrchestratorService) getEnv(key string) string {
	// This is a simple wrapper - in production, we'd use os.Getenv directly
	// But this allows for easier testing
	return orch.getEnvFromOS(key)
}

func (orch *OrchestratorService) getEnvFromOS(key string) string {
	return os.Getenv(key)
}

// restoreRunningDeployments queries the database for all deployments marked as RUNNING
// and attempts to start them. This ensures deployments are restored after orchestrator restarts.
// This function runs in a goroutine and processes deployments concurrently to avoid blocking.
func (os *OrchestratorService) restoreRunningDeployments() {
	// Give the system a moment to fully initialize before restoring deployments
	time.Sleep(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger.Info("[Orchestrator] Restoring running deployments from database...")

	// Query database for all deployments with status = RUNNING
	var deployments []database.Deployment
	runningStatus := int32(deploymentsv1.DeploymentStatus_RUNNING)
	if err := database.DB.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", runningStatus).
		Find(&deployments).Error; err != nil {
		logger.Error("[Orchestrator] Failed to query running deployments from database: %v", err)
		return
	}

	if len(deployments) == 0 {
		logger.Info("[Orchestrator] No running deployments found in database to restore")
		return
	}

	logger.Info("[Orchestrator] Found %d running deployment(s) to restore", len(deployments))

	// Process deployments concurrently with a limit to avoid overwhelming the system
	const maxConcurrency = 5
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	for _, deployment := range deployments {
		wg.Add(1)
		go func(dep database.Deployment) {
			defer wg.Done()

			// Acquire semaphore to limit concurrency
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logger.Info("[Orchestrator] Restoring deployment %s (name: %s, domain: %s)",
				dep.ID, dep.Name, dep.Domain)

			// Use a shorter timeout per deployment to avoid blocking too long
			deployCtx, deployCancel := context.WithTimeout(ctx, 2*time.Minute)
			defer deployCancel()

			if err := os.deploymentManager.StartDeployment(deployCtx, dep.ID); err != nil {
				logger.Warn("[Orchestrator] Failed to restore deployment %s: %v", dep.ID, err)
				mu.Lock()
				failureCount++
				mu.Unlock()
			} else {
				logger.Info("[Orchestrator] Successfully restored deployment %s", dep.ID)
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(deployment)
	}

	// Wait for all deployments to be processed
	wg.Wait()

	logger.Info("[Orchestrator] Deployment restoration completed: %d succeeded, %d failed out of %d total",
		successCount, failureCount, len(deployments))
}
