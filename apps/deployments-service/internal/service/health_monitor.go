package deployments

import (
	"context"
	"log"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

// checkAndRedeployDeployments checks all deployments that should be running
// and automatically redeploys any that are missing containers
func (s *Service) checkAndRedeployDeployments(ctx context.Context) {
	log.Printf("[HealthMonitor] Checking deployments that should be running...")

	// Query all deployments with status RUNNING or DEPLOYING that are not deleted
	var deployments []database.Deployment
	statusRunning := int32(deploymentsv1.DeploymentStatus_RUNNING)
	statusDeploying := int32(deploymentsv1.DeploymentStatus_DEPLOYING)
	
	err := database.DB.WithContext(ctx).
		Where("(status = ? OR status = ?) AND deleted_at IS NULL", statusRunning, statusDeploying).
		Find(&deployments).Error
	
	if err != nil {
		log.Printf("[HealthMonitor] Failed to query deployments: %v", err)
		return
	}

	log.Printf("[HealthMonitor] Found %d deployments that should be running", len(deployments))

	redeployedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, deployment := range deployments {
		// Check if deployment has containers
		locations, err := database.GetAllDeploymentLocations(deployment.ID)
		if err != nil {
			log.Printf("[HealthMonitor] Failed to get locations for deployment %s: %v", deployment.ID, err)
			errorCount++
			continue
		}

		// Validate and refresh locations to discover containers from Docker
		locations, err = database.ValidateAndRefreshLocations(deployment.ID)
		if err != nil {
			log.Printf("[HealthMonitor] Failed to validate locations for deployment %s: %v", deployment.ID, err)
			errorCount++
			continue
		}

		// If no containers found, attempt automatic redeployment
		if len(locations) == 0 {
			log.Printf("[HealthMonitor] Deployment %s (%s) should be running but has no containers, attempting automatic redeployment", deployment.ID, deployment.Name)
			
			// Use system context for internal operations
			systemCtx := s.createSystemContext()
			
			if err := s.attemptAutomaticRedeployment(systemCtx, deployment.ID); err != nil {
				log.Printf("[HealthMonitor] Failed to redeploy deployment %s: %v", deployment.ID, err)
				errorCount++
			} else {
				log.Printf("[HealthMonitor] Successfully triggered redeployment for deployment %s", deployment.ID)
				redeployedCount++
			}
		} else {
			// Deployment has containers, skip
			skippedCount++
		}
	}

	log.Printf("[HealthMonitor] Check complete: %d redeployed, %d skipped (had containers), %d errors", redeployedCount, skippedCount, errorCount)
}

// StartHealthMonitor starts a background service that periodically checks
// and redeploys deployments that should be running but don't have containers
func (s *Service) StartHealthMonitor(ctx context.Context, interval time.Duration) {
	log.Printf("[HealthMonitor] Starting health monitor service (interval: %v)", interval)
	
	// Run immediately on startup
	s.checkAndRedeployDeployments(ctx)
	
	// Then run periodically
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("[HealthMonitor] Health monitor service shutting down")
			return
		case <-ticker.C:
			s.checkAndRedeployDeployments(ctx)
		}
	}
}

