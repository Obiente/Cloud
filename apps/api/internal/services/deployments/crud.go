package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/quota"

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

	// Convert DB models to proto models
	items := make([]*deploymentsv1.Deployment, 0, len(dbDeployments))
	for _, dbDep := range dbDeployments {
		items = append(items, dbDeploymentToProto(dbDep))
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

	// Quota check
	reqRes := quota.RequestedResources{
		Replicas:    int(req.Msg.GetReplicas()),
		MemoryBytes: req.Msg.GetMemoryBytes(),
		CPUshares:   req.Msg.GetCpuShares(),
	}
	if err := s.quotaChecker.CanAllocate(ctx, orgID, reqRes); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	id := fmt.Sprintf("deploy-%d", time.Now().Unix())

	// Determine build strategy
	buildStrategy := req.Msg.GetBuildStrategy()
	var detectDir string
	if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED {
		// Auto-detect if repository URL is provided
		if repoURL := req.Msg.GetRepositoryUrl(); repoURL != "" {
			branch := req.Msg.GetBranch()
			if branch == "" {
				branch = "main" // Default branch
			}
			// Clone repo temporarily to detect
			buildDir, err := ensureBuildDir(id + "-detect")
			if err == nil {
				if err := cloneRepository(ctx, repoURL, branch, buildDir); err == nil {
					detected, _ := s.buildRegistry.AutoDetect(ctx, buildDir)
					buildStrategy = detected
					detectDir = buildDir // Keep for type inference
				}
			}
		}

		// Default to PLAIN_COMPOSE if image is provided (legacy behavior)
		if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED && req.Msg.GetImage() != "" {
			buildStrategy = deploymentsv1.BuildStrategy_PLAIN_COMPOSE
		} else if buildStrategy == deploymentsv1.BuildStrategy_BUILD_STRATEGY_UNSPECIFIED {
			buildStrategy = deploymentsv1.BuildStrategy_NIXPACKS // Default fallback
		}
	}

	// Infer deployment type from build strategy if not specified
	deploymentType := req.Msg.GetType()
	if deploymentType == deploymentsv1.DeploymentType_DEPLOYMENT_TYPE_UNSPECIFIED {
		if detectDir != "" {
			// Use detection directory if available
			deploymentType = s.buildRegistry.InferDeploymentType(ctx, buildStrategy, detectDir)
			os.RemoveAll(detectDir) // Cleanup after type inference
		} else if repoURL := req.Msg.GetRepositoryUrl(); repoURL != "" {
			branch := req.Msg.GetBranch()
			if branch == "" {
				branch = "main" // Default branch
			}
			// Clone repo temporarily for type inference
			buildDir, err := ensureBuildDir(id + "-type-detect")
			if err == nil {
				if err := cloneRepository(ctx, repoURL, branch, buildDir); err == nil {
					deploymentType = s.buildRegistry.InferDeploymentType(ctx, buildStrategy, buildDir)
					os.RemoveAll(buildDir) // Cleanup
				}
			}
		} else {
			// Default based on build strategy
			deploymentType = s.buildRegistry.InferDeploymentType(ctx, buildStrategy, "")
		}
	}

	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.obiente.cloud", req.Msg.GetName()),
		CustomDomains:  []string{},
		Type:           deploymentType,
		BuildStrategy:  buildStrategy,
		Status:         deploymentsv1.DeploymentStatus_STOPPED, // Start as STOPPED when no repository
		HealthStatus:   "pending",
		Environment:    deploymentsv1.Environment_PRODUCTION,
		LastDeployedAt: timestamppb.Now(),
		BandwidthUsage: 0,
		StorageUsage:   0,
		BuildTime:      0,
		Size:           "--",
		CreatedAt:      timestamppb.Now(),
		Image:          proto.String(req.Msg.GetImage()),
		Port:           proto.Int32(req.Msg.GetPort()),
		Replicas:       proto.Int32(req.Msg.GetReplicas()),
		EnvVars:        req.Msg.GetEnv(),
	}
	if branch := req.Msg.GetBranch(); branch != "" {
		deployment.Branch = branch
	} else {
		deployment.Branch = "main" // Default branch, but won't be used until repo is set
	}
	if repo := req.Msg.GetRepositoryUrl(); repo != "" {
		deployment.RepositoryUrl = proto.String(repo)
		// Only set status to BUILDING if we have a repository to build from
		deployment.Status = deploymentsv1.DeploymentStatus_BUILDING
	}
	if integrationID := req.Msg.GetGithubIntegrationId(); integrationID != "" {
		deployment.GithubIntegrationId = proto.String(integrationID)
	}
	if build := req.Msg.GetBuildCommand(); build != "" {
		deployment.BuildCommand = proto.String(build)
	}
	if install := req.Msg.GetInstallCommand(); install != "" {
		deployment.InstallCommand = proto.String(install)
	}

	dbDeployment := protoToDBDeployment(deployment, orgID, userInfo.Id)
	if err := s.repo.Create(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	// Only build and deploy if we have a repository URL
	hasRepository := req.Msg.GetRepositoryUrl() != ""
	if hasRepository && s.manager != nil && s.buildRegistry != nil {
		strategy, err := s.buildRegistry.Get(buildStrategy)
		if err != nil {
			log.Printf("[CreateDeployment] Invalid build strategy %v: %v", buildStrategy, err)
			_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid build strategy: %w", err))
		}

		// Prepare build config
		branch := req.Msg.GetBranch()
		if branch == "" {
			branch = "main" // Default branch
		}
		buildConfig := &BuildConfig{
			DeploymentID:    id,
			RepositoryURL:   req.Msg.GetRepositoryUrl(),
			Branch:          branch,
			BuildCommand:    req.Msg.GetBuildCommand(),
			InstallCommand:  req.Msg.GetInstallCommand(),
			DockerfilePath:  req.Msg.GetDockerfilePath(),
			ComposeFilePath: req.Msg.GetComposeFilePath(),
			EnvVars:         req.Msg.GetEnv(),
			Port:            int(req.Msg.GetPort()),
			MemoryBytes:     req.Msg.GetMemoryBytes(),
			CPUShares:       req.Msg.GetCpuShares(),
		}

		// For PLAIN_COMPOSE with direct image, skip build step
		if buildStrategy == deploymentsv1.BuildStrategy_PLAIN_COMPOSE && req.Msg.GetImage() != "" && req.Msg.GetRepositoryUrl() == "" {
			// Direct image deployment (legacy behavior)
			cfg := &orchestrator.DeploymentConfig{
				DeploymentID: id,
				Image:        req.Msg.GetImage(),
				Domain:       deployment.GetDomain(),
				Port:         int(req.Msg.GetPort()),
				EnvVars:      req.Msg.GetEnv(),
				Labels:       req.Msg.GetLabels(),
				Memory:       req.Msg.GetMemoryBytes(),
				CPUShares:    req.Msg.GetCpuShares(),
				Replicas:     int(req.Msg.GetReplicas()),
			}
			if err := s.manager.CreateDeployment(ctx, cfg); err != nil {
				log.Printf("[CreateDeployment] WARNING: Failed to create containers: %v", err)
				_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
			} else {
				if err := s.verifyContainersRunning(ctx, id); err != nil {
					log.Printf("[CreateDeployment] WARNING: Containers not running: %v", err)
					_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
				} else {
					_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_RUNNING))
				}
			}
		} else {
			// Build using strategy
			result, err := strategy.Build(ctx, dbDeployment, buildConfig)
			if err != nil || !result.Success {
				log.Printf("[CreateDeployment] Build failed for deployment %s: %v", id, err)
				if result != nil && result.Error != nil {
					err = result.Error
				}
				_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build failed: %w", err))
			}

			// Deploy using build result
			if err := deployResultToOrchestrator(ctx, s.manager, dbDeployment, result); err != nil {
				log.Printf("[CreateDeployment] Deployment failed: %v", err)
				_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("deployment failed: %w", err))
			}

			// Update deployment with build results
			if result.ImageName != "" {
				dbDeployment.Image = &result.ImageName
			}
			if result.ComposeYaml != "" {
				dbDeployment.ComposeYaml = result.ComposeYaml
			}
			if result.Port > 0 {
				port := int32(result.Port)
				dbDeployment.Port = &port
			}
			s.repo.Update(ctx, dbDeployment)

			// Verify containers are running
			if err := s.verifyContainersRunning(ctx, id); err != nil {
				log.Printf("[CreateDeployment] WARNING: Containers not running: %v", err)
				_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
			} else {
				_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_RUNNING))
			}
		}
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

	// Convert to proto and return
	deployment := dbDeploymentToProto(dbDeployment)
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
