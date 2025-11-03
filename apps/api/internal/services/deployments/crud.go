package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	"api/internal/auth"
	"api/internal/database"

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

	// Create filters with user ID
	filters := &database.DeploymentFilters{
		UserID: userInfo.Id,
		// Admin users can see all deployments
		IncludeAll: auth.HasRole(userInfo, auth.RoleAdmin),
	}

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

	// Convert DB models to proto models and enrich with actual container status
	items := make([]*deploymentsv1.Deployment, 0, len(dbDeployments))
	for _, dbDep := range dbDeployments {
		deployment := dbDeploymentToProto(dbDep)
		
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
		Pagination: &organizationsv1.Pagination{
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
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.create"}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
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
	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.obiente.cloud", req.Msg.GetName()),
		CustomDomains:  []string{},
		Type:           deploymentsv1.DeploymentType_DEPLOYMENT_TYPE_UNSPECIFIED, // Will be auto-detected
		BuildStrategy:  deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED,  // Will be auto-detected
		Status:         deploymentsv1.DeploymentStatus_STOPPED,                    // Start as STOPPED
		HealthStatus:   "pending",
		Environment:    environment,
		Groups:         groups, // Set groups from request
		Branch:         "main", // Default branch
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
	if err := s.checkDeploymentPermission(ctx, deploymentID, "view"); err != nil {
		return nil, err
	}

	// Get deployment by ID
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Convert to proto and enrich with actual container status
	deployment := dbDeploymentToProto(dbDeployment)
	
	// Get actual container status from Docker (not DB)
	// Only for compose deployments (when BuildStrategy is PLAIN_COMPOSE or COMPOSE_REPO)
	if dbDeployment.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE) ||
	   dbDeployment.BuildStrategy == int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) {
		running, total, err := s.getDeploymentContainerStatus(ctx, deploymentID)
		if err == nil {
			deployment.ContainersRunning = proto.Int32(running)
			deployment.ContainersTotal = proto.Int32(total)
			
			// Sync deployment status with actual container status
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
	if req.Msg.Domain != nil {
		dbDeployment.Domain = req.Msg.GetDomain()
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
	// Handle custom_domains (repeated string -> JSON array)
	if len(req.Msg.GetCustomDomains()) > 0 {
		customDomainsJSON, err := json.Marshal(req.Msg.GetCustomDomains())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal custom domains: %w", err))
		}
		dbDeployment.CustomDomains = string(customDomainsJSON)
	} else if req.Msg.CustomDomains != nil {
		// Empty array was explicitly set
		dbDeployment.CustomDomains = "[]"
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
