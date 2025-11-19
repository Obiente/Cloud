package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/email"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
)

// RollbackMonitor monitors Docker Swarm services for rollback events
type RollbackMonitor struct {
	dockerClient *docker.Client
	mailer       email.Sender
	consoleURL   string
	supportEmail string
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewRollbackMonitor creates a new rollback monitor
func NewRollbackMonitor() (*RollbackMonitor, error) {
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	mailer := email.NewSenderFromEnv()
	consoleURL := os.Getenv("DASHBOARD_URL")
	if consoleURL == "" {
		consoleURL = "https://obiente.cloud"
	}

	supportEmail := os.Getenv("SUPPORT_EMAIL")
	if supportEmail == "" {
		supportEmail = "support@obiente.cloud"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RollbackMonitor{
		dockerClient: dcli,
		mailer:       mailer,
		consoleURL:   consoleURL,
		supportEmail: supportEmail,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

// Start begins monitoring for rollback events
func (rm *RollbackMonitor) Start() {
	logger.Info("[RollbackMonitor] Starting rollback monitor...")

	// Monitor Docker events for service updates
	go rm.monitorServiceEvents()

	logger.Info("[RollbackMonitor] Rollback monitor started")
}

// Stop stops the rollback monitor
func (rm *RollbackMonitor) Stop() {
	logger.Info("[RollbackMonitor] Stopping rollback monitor...")
	rm.cancel()
	if rm.dockerClient != nil {
		rm.dockerClient.Close()
	}
	logger.Info("[RollbackMonitor] Rollback monitor stopped")
}

// monitorServiceEvents monitors Docker Swarm service events for rollbacks
func (rm *RollbackMonitor) monitorServiceEvents() {
	// Poll for service updates periodically
	// Docker Swarm doesn't emit explicit "rollback" events, so we need to detect rollbacks
	// by monitoring service task states and comparing with previous versions
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Track last known service versions
	lastServiceVersions := make(map[string]uint64)

	for {
		select {
		case <-ticker.C:
			rm.checkForRollbacks(lastServiceVersions)
		case <-rm.ctx.Done():
			return
		}
	}
}

// checkForRollbacks checks all Obiente-managed services for rollback events
func (rm *RollbackMonitor) checkForRollbacks(lastServiceVersions map[string]uint64) {
	ctx, cancel := context.WithTimeout(rm.ctx, 30*time.Second)
	defer cancel()

	// Get all Obiente-managed services
	// Services are labeled with cloud.obiente.managed=true and cloud.obiente.deployment_id
	// Use docker service ls to get services
	// Note: We'll use exec to run docker commands since the Docker API doesn't directly expose service version info
	cmd := exec.CommandContext(ctx, "docker", "service", "ls",
		"--filter", "label=cloud.obiente.managed=true",
		"--filter", "label=cloud.obiente.deployment_id",
		"--format", "{{.Name}}\t{{.ID}}",
	)

	output, err := cmd.Output()
	if err != nil {
		logger.Debug("[RollbackMonitor] Failed to list services: %v", err)
		return
	}

	// Parse service list
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		serviceName := strings.TrimSpace(parts[0])
		serviceID := strings.TrimSpace(parts[1])

		// Get service details to check for rollback
		rm.checkServiceRollback(ctx, serviceName, serviceID, lastServiceVersions)
	}
}

// checkServiceRollback checks if a service has been rolled back
func (rm *RollbackMonitor) checkServiceRollback(ctx context.Context, serviceName, serviceID string, lastServiceVersions map[string]uint64) {
	// Get service inspect to check update status and version
	// Docker Swarm services have an UpdateStatus field that indicates rollback
	cmd := exec.CommandContext(ctx, "docker", "service", "inspect", serviceName,
		"--format", "{{.UpdateStatus.State}}\t{{.UpdateStatus.Message}}\t{{.Version.Index}}",
	)

	output, err := cmd.Output()
	if err != nil {
		logger.Debug("[RollbackMonitor] Failed to inspect service %s: %v", serviceName, err)
		return
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(parts) < 3 {
		return
	}

	updateState := strings.TrimSpace(parts[0])
	updateMessage := strings.TrimSpace(parts[1])
	versionStr := strings.TrimSpace(parts[2])

	// Check if this is a rollback event
	// Docker Swarm sets UpdateStatus.State to "rollback_started" or "rollback_completed"
	// and UpdateStatus.Message contains "rolled back"
	if updateState == "rollback_started" || updateState == "rollback_completed" ||
		strings.Contains(strings.ToLower(updateMessage), "rolled back") ||
		strings.Contains(strings.ToLower(updateMessage), "rollback") {

		// Check if we've already notified for this rollback
		// Compare version to see if this is a new rollback
		var currentVersion uint64
		fmt.Sscanf(versionStr, "%d", &currentVersion)

		lastVersion, seen := lastServiceVersions[serviceName]
		if seen && currentVersion <= lastVersion {
			// Already processed this rollback
			return
		}

		// Update last known version
		lastServiceVersions[serviceName] = currentVersion

		// Extract deployment ID from service labels
		labelCmd := exec.CommandContext(ctx, "docker", "service", "inspect", serviceName,
			"--format", "{{index .Spec.Labels \"cloud.obiente.deployment_id\"}}",
		)

		labelOutput, err := labelCmd.Output()
		if err != nil {
			logger.Debug("[RollbackMonitor] Failed to get deployment ID for service %s: %v", serviceName, err)
			return
		}

		deploymentID := strings.TrimSpace(string(labelOutput))
		if deploymentID == "" {
			logger.Debug("[RollbackMonitor] Service %s has no deployment_id label", serviceName)
			return
		}

		// Get service name from labels
		serviceNameLabelCmd := exec.CommandContext(ctx, "docker", "service", "inspect", serviceName,
			"--format", "{{index .Spec.Labels \"cloud.obiente.service_name\"}}",
		)

		serviceNameLabelOutput, _ := serviceNameLabelCmd.Output()
		displayServiceName := strings.TrimSpace(string(serviceNameLabelOutput))
		if displayServiceName == "" {
			displayServiceName = "default"
		}

		// Send notification
		logger.Info("[RollbackMonitor] Detected rollback for deployment %s, service %s", deploymentID, displayServiceName)
		rm.sendRollbackNotification(ctx, deploymentID, displayServiceName, serviceName, updateMessage)
	}
}

// sendRollbackNotification sends an email notification about a deployment rollback
func (rm *RollbackMonitor) sendRollbackNotification(ctx context.Context, deploymentID, serviceName, swarmServiceName, reason string) {
	if !rm.mailer.Enabled() {
		logger.Debug("[RollbackMonitor] Email disabled, skipping rollback notification for deployment %s", deploymentID)
		return
	}

	// Get deployment details
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		logger.Warn("[RollbackMonitor] Failed to get deployment %s: %v", deploymentID, err)
		return
	}

	// Get organization members (owners and admins) to notify
	var members []database.OrganizationMember
	if err := database.DB.Where("organization_id = ? AND role IN (?, ?) AND status = ?",
		deployment.OrganizationID, "owner", "admin", "active").Find(&members).Error; err != nil {
		logger.Warn("[RollbackMonitor] Failed to get organization members: %v", err)
		return
	}

	if len(members) == 0 {
		logger.Debug("[RollbackMonitor] No members to notify for deployment %s", deploymentID)
		return
	}

	// Get user emails
	var emails []string
	resolver := organizations.GetUserProfileResolver()
	for _, member := range members {
		if member.UserID == "" || strings.HasPrefix(member.UserID, "pending:") {
			continue
		}

		userProfile, err := resolver.Resolve(ctx, member.UserID)
		if err != nil {
			logger.Debug("[RollbackMonitor] Failed to resolve user profile for %s: %v", member.UserID, err)
			continue
		}

		if userProfile != nil && userProfile.Email != "" {
			emails = append(emails, userProfile.Email)
		}
	}

	if len(emails) == 0 {
		logger.Debug("[RollbackMonitor] No email addresses found for deployment %s", deploymentID)
		return
	}

	// Get organization name
	var org database.Organization
	orgName := "your organization"
	if err := database.DB.Where("id = ?", deployment.OrganizationID).First(&org).Error; err == nil {
		orgName = org.Name
	}

	// Get deployment name
	deploymentName := deploymentID
	if deployment.Name != "" {
		deploymentName = deployment.Name
	}

	// Build email
	subject := fmt.Sprintf("Deployment Rollback: %s", deploymentName)
	template := email.TemplateData{
		Subject:     subject,
		PreviewText: fmt.Sprintf("Your deployment %s was automatically rolled back due to a failure.", deploymentName),
		Greeting:    fmt.Sprintf("Hi %s,", orgName),
		Heading:     "Deployment Rollback Notification",
		IntroLines: []string{
			fmt.Sprintf("Your deployment '%s' was automatically rolled back to a previous version due to a failure during update.", deploymentName),
			"Docker Swarm detected that the new version was not healthy and automatically reverted to the previous working version.",
		},
		Highlights: []email.Highlight{
			{Label: "Deployment", Value: deploymentName},
			{Label: "Service", Value: serviceName},
			{Label: "Reason", Value: reason},
		},
		Sections: []email.Section{
			{
				Title: "What happened?",
				Lines: []string{
					"A new version of your deployment was deployed, but it failed health checks or crashed during startup.",
					"Docker Swarm automatically detected this failure and rolled back to the previous working version.",
					"Your deployment is now running the previous stable version.",
				},
			},
			{
				Title: "What should you do?",
				Lines: []string{
					"Review your deployment logs to identify the issue with the new version.",
					"Fix the problem in your code or configuration.",
					"Redeploy once the issue is resolved.",
				},
			},
		},
		CTA: &email.CTA{
			Label: "View Deployment",
			URL:   fmt.Sprintf("%s/deployments/%s", rm.consoleURL, deploymentID),
		},
		Category:     email.CategoryNotification,
		SupportEmail: rm.supportEmail,
		BaseURL:      rm.consoleURL,
	}

	message := &email.Message{
		To:       emails,
		Subject:  subject,
		Template: &template,
		Category: email.CategoryNotification,
		Metadata: map[string]string{
			"deployment_id": deploymentID,
			"service_name":  serviceName,
			"reason":        reason,
		},
	}

	if err := rm.mailer.Send(ctx, message); err != nil {
		logger.Warn("[RollbackMonitor] Failed to send rollback notification for deployment %s: %v", deploymentID, err)
	} else {
		logger.Info("[RollbackMonitor] Sent rollback notification to %d recipient(s) for deployment %s", len(emails), deploymentID)
	}
}
