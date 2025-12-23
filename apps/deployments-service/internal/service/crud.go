package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/quota"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListDeployments lists all deployments for an organization
func (s *Service) ListDeployments(ctx context.Context, req *connect.Request[deploymentsv1.ListDeploymentsRequest]) (*connect.Response[deploymentsv1.ListDeploymentsResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		// Resolve to a real organization the user belongs to
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Get authenticated user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	// Check if user has organization-wide read permission for deployments
	// This allows users with custom roles (like "system admin") to see all deployments
	hasOrgWideRead := false
	permErr := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{
		Permission:   auth.PermissionDeploymentRead,
		ResourceType: "deployment",
		ResourceID:   "", // Empty resource ID means org-wide permission
	})
	if permErr == nil {
		hasOrgWideRead = true
	} else {
		// Log permission check failure for debugging (but don't fail the request yet)
		// User might still have permission to see their own deployments
		log.Printf("[ListDeployments] User %s does not have org-wide deployment.read permission in org %s: %v", userInfo.Id, orgID, permErr)
	}

	// Create filters with user ID
	filters := &database.DeploymentFilters{
		UserID: userInfo.Id,
		// Admin users or users with org-wide read permission can see all deployments
		IncludeAll: auth.IsSuperadmin(ctx, userInfo) || hasOrgWideRead,
	}
	
	log.Printf("[ListDeployments] User %s, Org %s, IncludeAll: %v, hasOrgWideRead: %v", userInfo.Id, orgID, filters.IncludeAll, hasOrgWideRead)

	// Add status filter if provided
	if status := req.Msg.Status; status != nil {
		statusVal := int32(*status)
		filters.Status = &statusVal
	}

	// Get deployments filtered by organization and user ID
	dbDeployments, err := s.repo.GetAll(ctx, orgID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deployments: %w", err))
	}
	
	log.Printf("[ListDeployments] Found %d deployments for user %s in org %s (IncludeAll: %v)", len(dbDeployments), userInfo.Id, orgID, filters.IncludeAll)

	// Convert DB models to proto models and enrich with actual container status
	items := make([]*deploymentsv1.Deployment, 0, len(dbDeployments))
	for _, dbDep := range dbDeployments {
		deployment := dbDeploymentToProto(dbDep)
		
		// If deployment's build time is 0, try to get it from the latest successful build
		if deployment.BuildTime == 0 {
			latestBuild, err := s.buildHistoryRepo.GetLatestSuccessfulBuild(ctx, dbDep.ID)
			if err == nil && latestBuild != nil && latestBuild.BuildTime > 0 {
				deployment.BuildTime = latestBuild.BuildTime
			}
		}
		
		// Get actual container status from Docker (not DB)
		// Only for compose deployments (when BuildStrategy is PLAIN_COMPOSE or COMPOSE_REPO)
		if dbDep.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE) ||
		   dbDep.BuildStrategy == int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) {
			running, total, err := s.getDeploymentContainerStatus(ctx, dbDep.ID)
			if err == nil {
				deployment.ContainersRunning = proto.Int32(running)
				deployment.ContainersTotal = proto.Int32(total)
			}
		}
		
		items = append(items, deployment)
	}

	// Get total count with same filters
	total, err := s.repo.Count(ctx, orgID, filters)
	if err != nil {
		total = int64(len(dbDeployments))
	}

	// Create response with pagination
	res := connect.NewResponse(&deploymentsv1.ListDeploymentsResponse{
		Deployments: items,
		Pagination: &commonv1.Pagination{
			Page:       1,
			PerPage:    int32(len(items)),
			Total:      int32(total),
			TotalPages: 1,
		},
	})
	return res, nil
}

// CreateDeployment creates a new deployment
func (s *Service) CreateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.CreateDeploymentRequest]) (*connect.Response[deploymentsv1.CreateDeploymentResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	// Permission: org-level
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentCreate}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Check quota: verify organization hasn't exceeded max deployments limit
	// This checks if creating a new deployment (1 replica) would exceed the limit
	if err := s.quotaChecker.CanAllocate(ctx, orgID, quota.RequestedResources{
		Replicas:    1, // Creating a new deployment counts as 1 replica
		MemoryBytes: 0, // Memory/CPU will be checked when deployment is actually started
		CPUshares:   0,
	}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("quota check failed: %w", err))
	}

	id := fmt.Sprintf("deploy-%d", time.Now().Unix())

	// Get environment from request, default to PRODUCTION
	environment := req.Msg.GetEnvironment()
	if environment == deploymentsv1.Environment_ENVIRONMENT_UNSPECIFIED {
		environment = deploymentsv1.Environment_PRODUCTION
	}

	// Get groups from request (optional)
	groups := req.Msg.GetGroups()

	// Create deployment with minimal configuration
	// Type and build strategy will be auto-detected when repository is configured
	healthcheckTypeUnspecified := deploymentsv1.HealthCheckType_HEALTHCHECK_TYPE_UNSPECIFIED
	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.my.obiente.cloud", id),
		CustomDomains:  []string{},
		Type:           deploymentsv1.DeploymentType_DEPLOYMENT_TYPE_UNSPECIFIED, // Will be auto-detected
		BuildStrategy:  deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED,  // Will be auto-detected
		Status:         deploymentsv1.DeploymentStatus_STOPPED,                    // Start as STOPPED
		HealthStatus:   "pending",
		Environment:    environment,
		Groups:         groups, // Set groups from request
		Branch:         "main", // Default branch
		HealthcheckType: &healthcheckTypeUnspecified, // Default to auto-detection
		LastDeployedAt: timestamppb.Now(),
		BandwidthUsage: 0,
		StorageUsage:   0,
		BuildTime:      0,
		Size:           "--",
		CreatedAt:      timestamppb.Now(),
		EnvVars:        map[string]string{},
	}

	dbDeployment := protoToDBDeployment(deployment, orgID, userInfo.Id)
	if err := s.repo.Create(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	// Fetch the latest deployment from database to ensure all fields are included in response
	updatedDeployment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[CreateDeployment] Warning: Failed to fetch updated deployment: %v", err)
		// Fallback to the deployment object we created
	} else {
		deployment = dbDeploymentToProto(updatedDeployment)
	}

	res := connect.NewResponse(&deploymentsv1.CreateDeploymentResponse{Deployment: deployment})
	return res, nil
}

// GetDeployment retrieves a deployment by ID
func (s *Service) GetDeployment(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRequest]) (*connect.Response[deploymentsv1.GetDeploymentResponse], error) {
	// Check if user has view permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "read"); err != nil {
		return nil, err
	}

	// Get deployment by ID
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Convert to proto and enrich with actual container status
	deployment := dbDeploymentToProto(dbDeployment)
	
	// If deployment's build time is 0, get it from the latest successful build
	if deployment.BuildTime == 0 {
		latestBuild, err := s.buildHistoryRepo.GetLatestSuccessfulBuild(ctx, deploymentID)
		if err == nil && latestBuild != nil && latestBuild.BuildTime > 0 {
			deployment.BuildTime = latestBuild.BuildTime
		}
	}
	
	// Get actual container status from Docker (not DB)
	// Only for compose deployments (when BuildStrategy is PLAIN_COMPOSE or COMPOSE_REPO)
	if dbDeployment.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE) ||
	   dbDeployment.BuildStrategy == int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) {
		running, total, err := s.getDeploymentContainerStatus(ctx, deploymentID)
		if err == nil {
			deployment.ContainersRunning = proto.Int32(running)
			deployment.ContainersTotal = proto.Int32(total)
			
			// Sync deployment status with actual container status
			// IMPORTANT: Don't sync status if deployment is currently BUILDING or DEPLOYING
			// During builds, containers might not exist yet or be in transitional states
			currentStatus := deploymentsv1.DeploymentStatus(dbDeployment.Status)
			isBuildingOrDeploying := currentStatus == deploymentsv1.DeploymentStatus_BUILDING || 
			                          currentStatus == deploymentsv1.DeploymentStatus_DEPLOYING
			
			if !isBuildingOrDeploying {
				// If deployment has containers but none are running, it should be STOPPED
				// If some containers are running, it should be RUNNING
				if total > 0 {
					if running == 0 {
						// All containers stopped - update status to STOPPED
						deployment.Status = deploymentsv1.DeploymentStatus_STOPPED
						// Optionally update DB to keep it in sync (async to not block response)
						go func() {
							if err := s.repo.UpdateStatus(context.Background(), deploymentID, int32(deploymentsv1.DeploymentStatus_STOPPED)); err != nil {
								log.Printf("[GetDeployment] Failed to sync deployment status to STOPPED: %v", err)
							}
						}()
					} else if running > 0 && dbDeployment.Status == int32(deploymentsv1.DeploymentStatus_STOPPED) {
						// Some containers running but DB says STOPPED - update to RUNNING
						deployment.Status = deploymentsv1.DeploymentStatus_RUNNING
						// Optionally update DB to keep it in sync (async to not block response)
						go func() {
							if err := s.repo.UpdateStatus(context.Background(), deploymentID, int32(deploymentsv1.DeploymentStatus_RUNNING)); err != nil {
								log.Printf("[GetDeployment] Failed to sync deployment status to RUNNING: %v", err)
							}
						}()
					}
				}
			}
		}
		
		// Get health status from Docker containers
		healthStatus, err := s.getDeploymentHealthStatus(ctx, deploymentID)
		if err == nil && healthStatus != "" {
			deployment.HealthStatus = healthStatus
			// Optionally update DB to keep it in sync (async to not block response)
			go func() {
				if err := s.repo.UpdateHealthStatus(context.Background(), deploymentID, healthStatus); err != nil {
					log.Printf("[GetDeployment] Failed to sync health status: %v", err)
				}
			}()
		}
	} else {
		// For image-based deployments, also check health status
		healthStatus, err := s.getDeploymentHealthStatus(ctx, deploymentID)
		if err == nil && healthStatus != "" {
			deployment.HealthStatus = healthStatus
			// Optionally update DB to keep it in sync (async to not block response)
			go func() {
				if err := s.repo.UpdateHealthStatus(context.Background(), deploymentID, healthStatus); err != nil {
					log.Printf("[GetDeployment] Failed to sync health status: %v", err)
				}
			}()
		}
	}
	
	res := connect.NewResponse(&deploymentsv1.GetDeploymentResponse{Deployment: deployment})
	return res, nil
}

// UpdateDeployment updates a deployment's configuration
func (s *Service) UpdateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentResponse], error) {
	// Check if user has edit permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "edit"); err != nil {
		return nil, err
	}

	// Get deployment by ID
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Update deployment fields (only update if provided)
	if req.Msg.Name != nil {
		dbDeployment.Name = req.Msg.GetName()
	}
	if req.Msg.RepositoryUrl != nil {
		repoURL := req.Msg.GetRepositoryUrl()
		if repoURL != "" {
			dbDeployment.RepositoryURL = &repoURL
		} else {
			dbDeployment.RepositoryURL = nil
		}
	}
	if req.Msg.GithubIntegrationId != nil {
		integrationID := req.Msg.GetGithubIntegrationId()
		if integrationID != "" {
			dbDeployment.GitHubIntegrationID = &integrationID
		} else {
			dbDeployment.GitHubIntegrationID = nil
		}
	}
	if req.Msg.Branch != nil {
		branch := req.Msg.GetBranch()
		dbDeployment.Branch = branch
	}
	if req.Msg.BuildCommand != nil {
		build := req.Msg.GetBuildCommand()
		if build != "" {
			dbDeployment.BuildCommand = &build
		} else {
			dbDeployment.BuildCommand = nil
		}
	}
	if req.Msg.InstallCommand != nil {
		install := req.Msg.GetInstallCommand()
		if install != "" {
			dbDeployment.InstallCommand = &install
		} else {
			dbDeployment.InstallCommand = nil
		}
	}
	if req.Msg.StartCommand != nil {
		start := req.Msg.GetStartCommand()
		if start != "" {
			dbDeployment.StartCommand = &start
		} else {
			dbDeployment.StartCommand = nil
		}
	}
	if req.Msg.DockerfilePath != nil {
		dockerfilePath := req.Msg.GetDockerfilePath()
		if dockerfilePath != "" {
			dbDeployment.DockerfilePath = &dockerfilePath
		} else {
			dbDeployment.DockerfilePath = nil
		}
	}
	if req.Msg.ComposeFilePath != nil {
		composeFilePath := req.Msg.GetComposeFilePath()
		if composeFilePath != "" {
			dbDeployment.ComposeFilePath = &composeFilePath
		} else {
			dbDeployment.ComposeFilePath = nil
		}
	}
	if req.Msg.BuildPath != nil {
		buildPath := req.Msg.GetBuildPath()
		if buildPath != "" {
			dbDeployment.BuildPath = &buildPath
		} else {
			dbDeployment.BuildPath = nil
		}
	}
	if req.Msg.BuildOutputPath != nil {
		buildOutputPath := req.Msg.GetBuildOutputPath()
		if buildOutputPath != "" {
			dbDeployment.BuildOutputPath = &buildOutputPath
		} else {
			dbDeployment.BuildOutputPath = nil
		}
	}
	if req.Msg.UseNginx != nil {
		useNginx := req.Msg.GetUseNginx()
		dbDeployment.UseNginx = &useNginx
	}
	if req.Msg.NginxConfig != nil {
		nginxConfig := req.Msg.GetNginxConfig()
		if nginxConfig != "" {
			dbDeployment.NginxConfig = &nginxConfig
		} else {
			dbDeployment.NginxConfig = nil
		}
	}

	// Health check configuration
	if req.Msg.HealthcheckType != nil {
		hcType := int32(req.Msg.GetHealthcheckType())
		dbDeployment.HealthcheckType = &hcType
	}
	if req.Msg.HealthcheckPort != nil {
		hcPort := req.Msg.GetHealthcheckPort()
		if hcPort > 0 {
			dbDeployment.HealthcheckPort = &hcPort
		} else {
			dbDeployment.HealthcheckPort = nil
		}
	}
	if req.Msg.HealthcheckPath != nil {
		hcPath := req.Msg.GetHealthcheckPath()
		if hcPath != "" {
			dbDeployment.HealthcheckPath = &hcPath
		} else {
			dbDeployment.HealthcheckPath = nil
		}
	}
	if req.Msg.HealthcheckExpectedStatus != nil {
		hcStatus := req.Msg.GetHealthcheckExpectedStatus()
		if hcStatus > 0 {
			dbDeployment.HealthcheckExpectedStatus = &hcStatus
		} else {
			dbDeployment.HealthcheckExpectedStatus = nil
		}
	}
	if req.Msg.HealthcheckCustomCommand != nil {
		hcCmd := req.Msg.GetHealthcheckCustomCommand()
		if hcCmd != "" {
			// Sanitize custom healthcheck command to prevent injection
			sanitized := sanitizeHealthcheckCommand(hcCmd)
			if sanitized != "" {
				dbDeployment.HealthcheckCustomCommand = &sanitized
			} else {
				dbDeployment.HealthcheckCustomCommand = nil
			}
		} else {
			dbDeployment.HealthcheckCustomCommand = nil
		}
	}

	if req.Msg.Port != nil {
		port := req.Msg.GetPort()
		dbDeployment.Port = &port
	}
	if req.Msg.BuildStrategy != nil {
		buildStrategy := int32(req.Msg.GetBuildStrategy())
		dbDeployment.BuildStrategy = buildStrategy
	}
	if req.Msg.Environment != nil {
		environment := int32(req.Msg.GetEnvironment())
		dbDeployment.Environment = environment
	}

	// Per-deployment resource limits (overrides)
	// Stored in DB as cpu_shares + memory_bytes for Docker.
	// Clearing semantics: if client sends 0, clear the override (set NULL) to fall back to defaults.
	// Cap values to organization plan limits before storing.
	if req.Msg.CpuLimit != nil || req.Msg.MemoryLimit != nil {
		// Get organization plan limits
		maxMemoryBytes, maxCPUCores, planErr := quota.GetEffectiveLimits(dbDeployment.OrganizationID)
		
		if req.Msg.CpuLimit != nil {
			cpuLimit := req.Msg.GetCpuLimit()
			if cpuLimit <= 0 {
				dbDeployment.CPUShares = nil
			} else {
				// Cap to plan limit if set
				if planErr == nil && maxCPUCores > 0 && cpuLimit > float64(maxCPUCores) {
					cpuLimit = float64(maxCPUCores)
					log.Printf("[UpdateDeployment] Capping CPU limit for deployment %s from %.2f to plan limit %d cores", deploymentID, req.Msg.GetCpuLimit(), maxCPUCores)
				}
				shares := int64(math.Round(cpuLimit * 1024.0))
				if shares < 1 {
					shares = 1
				}
				dbDeployment.CPUShares = &shares
			}
		}
		if req.Msg.MemoryLimit != nil {
			memMB := req.Msg.GetMemoryLimit()
			if memMB <= 0 {
				dbDeployment.MemoryBytes = nil
			} else {
				bytes := memMB * 1024 * 1024
				// Cap to plan limit if set
				if planErr == nil && maxMemoryBytes > 0 && bytes > maxMemoryBytes {
					bytes = maxMemoryBytes
					log.Printf("[UpdateDeployment] Capping memory limit for deployment %s from %d MB to plan limit %d bytes (%d MB)", 
						deploymentID, memMB, maxMemoryBytes, maxMemoryBytes/(1024*1024))
				}
				dbDeployment.MemoryBytes = &bytes
			}
		}
	}
	// Handle groups (repeated string -> JSON array)
	if len(req.Msg.GetGroups()) > 0 {
		groupsJSON, err := json.Marshal(req.Msg.GetGroups())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal groups: %w", err))
		}
		dbDeployment.Groups = string(groupsJSON)
	} else if req.Msg.Groups != nil {
		// Empty array was explicitly set
		dbDeployment.Groups = "[]"
	}
	// Get deployment before making changes (to preserve original domain for validation)
	originalDomain := dbDeployment.Domain
	
	// Handle custom_domains (repeated string -> JSON array)
	// Note: Handle this BEFORE domain validation so we can check against updated custom domains
	// For protobuf repeated fields, the slice is always non-nil, so we process it if it's provided
	// The frontend always sends custom_domains when updating, so we always process it
	customDomains := req.Msg.GetCustomDomains()
	if len(customDomains) > 0 {
		// Preserve existing tokens for domains that already have them
		// Get current deployment state to preserve existing tokens
		var currentCustomDomains []string
		if dbDeployment.CustomDomains != "" {
			if err := json.Unmarshal([]byte(dbDeployment.CustomDomains), &currentCustomDomains); err != nil {
				currentCustomDomains = []string{}
			}
		}
		
		// Map to preserve tokens: domain -> token entry
		tokenMap := make(map[string]string)
		for _, entry := range currentCustomDomains {
			parts := strings.Split(entry, ":")
			if len(parts) >= 3 && parts[1] == "token" {
				domainName := parts[0] // Extract domain from entry
				tokenMap[strings.ToLower(domainName)] = entry
			}
		}
		
		// Process new domains and preserve existing tokens
		processedDomains := []string{}
		for _, domain := range customDomains {
			parts := strings.Split(domain, ":")
			domainName := parts[0] // Extract domain from entry
			domainLower := strings.ToLower(domainName)
			
			// If this domain already has a token, preserve it
			if tokenEntry, exists := tokenMap[domainLower]; exists {
				processedDomains = append(processedDomains, tokenEntry)
				delete(tokenMap, domainLower) // Remove from map so we don't add it twice
			} else {
				// New domain or plain entry - add as-is (token will be created on first verification request)
				processedDomains = append(processedDomains, domain)
			}
		}
		
		// Deduplicate domains (case-insensitive) before validating
		customDomains = DeduplicateCustomDomains(processedDomains)
		
		// Validate custom domains before saving (check conflicts and verify ownership)
		if err := s.ValidateCustomDomains(ctx, deploymentID, customDomains); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("custom domain validation failed: %w", err))
		}
		
		customDomainsJSON, err := json.Marshal(customDomains)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal custom domains: %w", err))
		}
		dbDeployment.CustomDomains = string(customDomainsJSON)
	} else {
		// Empty array - clear custom domains
		dbDeployment.CustomDomains = "[]"
	}
	
	// Validate domain AFTER custom domains are updated (so we can check against newly verified custom domains)
	// Note: Users can only set the domain to the original default domain or a verified custom domain
	if req.Msg.Domain != nil {
		newDomain := req.Msg.GetDomain()
		// Validate that the domain is either the original default domain or a verified custom domain
		if newDomain != "" {
			// Get available domains (will include newly updated custom domains)
			availableDomains := s.getAvailableDomainsForDeployment(dbDeployment)
			// Also include the original default domain in case it's being changed
			domainAllowed := false
			if newDomain == originalDomain {
				domainAllowed = true
			} else {
				for _, allowedDomain := range availableDomains {
					if allowedDomain == newDomain {
						domainAllowed = true
						break
					}
				}
			}
			
			if !domainAllowed {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain %s is not available for this deployment. You can only use the default domain (%s) or verified custom domains", newDomain, originalDomain))
			}
		}
		dbDeployment.Domain = newDomain
	}

	// NOTE: Do NOT update status fields on config save
	// Status changes should only happen via explicit deploy/start/stop actions
	// This allows users to save settings without triggering a build

	// Save changes to database
	if err := s.repo.Update(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment: %w", err))
	}

	// Return updated deployment
	protoDeployment := dbDeploymentToProto(dbDeployment)
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentResponse{Deployment: protoDeployment})
	return res, nil
}

// DeleteDeployment deletes a deployment
func (s *Service) DeleteDeployment(ctx context.Context, req *connect.Request[deploymentsv1.DeleteDeploymentRequest]) (*connect.Response[deploymentsv1.DeleteDeploymentResponse], error) {
	// Check if user has delete permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "delete"); err != nil {
		return nil, err
	}

	// Get deployment before deleting to check if it's compose-based
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Delete all build logs and build history for this deployment
	// Get all build IDs for this deployment before deleting
	buildIDs, deletedCount, err := s.buildHistoryRepo.DeleteBuildsByDeployment(ctx, deploymentID)
	if err != nil {
		log.Printf("[DeleteDeployment] Failed to get builds for deployment %s: %v", deploymentID, err)
		// Continue with deletion even if we can't get builds
	} else if deletedCount > 0 {
		// Delete build logs from TimescaleDB for each build
		buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
		for _, buildID := range buildIDs {
			if err := buildLogsRepo.DeleteBuildLogs(ctx, buildID); err != nil {
				log.Printf("[DeleteDeployment] Failed to delete logs for build %s: %v", buildID, err)
				// Continue deleting other build logs
			}
		}
		log.Printf("[DeleteDeployment] Deleted %d builds and their logs for deployment %s", deletedCount, deploymentID)
	}

	// Remove containers/stack before deleting from DB
	if s.manager != nil {
		if dbDep.ComposeYaml != "" {
			// Remove compose deployment
			if err := s.manager.RemoveComposeDeployment(ctx, deploymentID); err != nil {
				log.Printf("[DeleteDeployment] Failed to remove compose deployment %s: %v", deploymentID, err)
				// Continue with DB deletion even if container removal failed
			}
		} else {
			// Remove regular containers
			_ = s.manager.DeleteDeployment(ctx, deploymentID)
		}
	}

	if err := s.repo.Delete(ctx, deploymentID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	res := connect.NewResponse(&deploymentsv1.DeleteDeploymentResponse{Success: true})
	return res, nil
}

// resolveUserDefaultOrgID returns a membership org id for the authenticated user, if any
func resolveUserDefaultOrgID(ctx context.Context) (string, bool) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil || userInfo == nil {
		return "", false
	}
	// Pick any organization the user belongs to (first by created_at desc)
	type row struct{ OrganizationID string }
	var r row
	if err := database.DB.Raw(`
        SELECT m.organization_id
        FROM organization_members m
        JOIN organizations o ON o.id = m.organization_id
        WHERE m.user_id = ?
        ORDER BY o.created_at DESC
        LIMIT 1
    `, userInfo.Id).Scan(&r).Error; err != nil {
		return "", false
	}
	if r.OrganizationID == "" {
		// No membership found; ensure a personal org exists, then retry lookup
		if id, ok := ensurePersonalOrgForUser(ctx, userInfo.Id); ok {
			return id, true
		}
		return "", false
	}
	return r.OrganizationID, true
}

// ensurePersonalOrgForUser creates a personal org and membership if user has none
func ensurePersonalOrgForUser(ctx context.Context, userID string) (string, bool) {
	// Double-check if any membership exists
	var cnt int64
	if err := database.DB.Model(&database.OrganizationMember{}).Where("user_id = ?", userID).Count(&cnt).Error; err != nil {
		return "", false
	}
	if cnt > 0 {
		// Race: someone created in between; fetch latest
		type row struct{ OrganizationID string }
		var r row
		_ = database.DB.Raw(`
            SELECT m.organization_id FROM organization_members m
            JOIN organizations o ON o.id = m.organization_id
            WHERE m.user_id = ? ORDER BY o.created_at DESC LIMIT 1
        `, userID).Scan(&r).Error
		if r.OrganizationID != "" {
			return r.OrganizationID, true
		}
		return "", false
	}
	now := time.Now()
	orgID := fmt.Sprintf("%s-%d", "org", now.UnixNano())
	org := &database.Organization{ID: orgID, Name: "Personal", Slug: "personal-" + userID, Plan: "personal", Status: "active", CreatedAt: now}
	if err := database.DB.Create(org).Error; err != nil {
		return "", false
	}
	mem := &database.OrganizationMember{ID: fmt.Sprintf("%s-%d", "mem", now.UnixNano()), OrganizationID: orgID, UserID: userID, Role: "owner", Status: "active", JoinedAt: now}
	if err := database.DB.Create(mem).Error; err != nil {
		return "", false
	}
	return orgID, true
}
