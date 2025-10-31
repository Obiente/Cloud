package deployments

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1 "api/gen/proto/obiente/cloud/organizations/v1"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/quota"
	githubclient "api/internal/services/github"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
	repo              *database.DeploymentRepository
	permissionChecker *auth.PermissionChecker
	manager           *orchestrator.DeploymentManager
	quotaChecker      *quota.Checker
}

func NewService(repo *database.DeploymentRepository, manager *orchestrator.DeploymentManager, qc *quota.Checker) deploymentsv1connect.DeploymentServiceHandler {
	return &Service{
		repo:              repo,
		permissionChecker: auth.NewPermissionChecker(),
		manager:           manager,
		quotaChecker:      qc,
	}
}

// checkDeploymentPermission is a helper to verify user permissions
func (s *Service) checkDeploymentPermission(ctx context.Context, deploymentID string, permission string) error {
	// Get deployment by ID to check ownership
	deployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	
	// Get user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}
	
	// First check if user is admin (always has access)
	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}
	
	// Check if user is the resource owner
    if deployment.CreatedBy == userInfo.Id {
		return nil // Resource owners have full access to their resources
	}
	
	// For more complex permissions (organization-based, team-based, etc.)
	err = s.permissionChecker.CheckPermission(ctx, auth.ResourceTypeDeployment, deploymentID, permission)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: %w", err))
	}
	
	return nil
}

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

func (s *Service) CreateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.CreateDeploymentRequest]) (*connect.Response[deploymentsv1.CreateDeploymentResponse], error) {
    orgID := req.Msg.GetOrganizationId()
    if orgID == "" {
        if eff, ok := resolveUserDefaultOrgID(ctx); ok {
            orgID = eff
        }
    }
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil { return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err)) }

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
	deployment := &deploymentsv1.Deployment{
		Id:             id,
		Name:           req.Msg.GetName(),
		Domain:         fmt.Sprintf("%s.obiente.cloud", req.Msg.GetName()),
		CustomDomains:  []string{},
		Type:           req.Msg.GetType(),
		Branch:         req.Msg.GetBranch(),
		Status:         deploymentsv1.DeploymentStatus_DEPLOYING,
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
	if repo := req.Msg.GetRepositoryUrl(); repo != "" { deployment.RepositoryUrl = proto.String(repo) }
	if build := req.Msg.GetBuildCommand(); build != "" { deployment.BuildCommand = proto.String(build) }
	if install := req.Msg.GetInstallCommand(); install != "" { deployment.InstallCommand = proto.String(install) }

    dbDeployment := protoToDBDeployment(deployment, orgID, userInfo.Id)
	if err := s.repo.Create(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	// Orchestrate container(s)
	if s.manager != nil {
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
			log.Printf("[CreateDeployment] WARNING: Failed to create containers for deployment %s (DB entry created): %v", id, err)
			// Update status to indicate deployment creation failed
			_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_FAILED))
		} else {
			log.Printf("[CreateDeployment] Successfully created containers for deployment %s", id)
			// Update status to running
			_ = s.repo.UpdateStatus(ctx, id, int32(deploymentsv1.DeploymentStatus_RUNNING))
		}
	}

	res := connect.NewResponse(&deploymentsv1.CreateDeploymentResponse{Deployment: deployment})
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
        if r.OrganizationID != "" { return r.OrganizationID, true }
        return "", false
    }
    now := time.Now()
    orgID := fmt.Sprintf("%s-%d", "org", now.UnixNano())
    org := &database.Organization{ID: orgID, Name: "Personal", Slug: "personal-" + userID, Plan: "personal", Status: "active", CreatedAt: now}
    if err := database.DB.Create(org).Error; err != nil { return "", false }
    mem := &database.OrganizationMember{ID: fmt.Sprintf("%s-%d", "mem", now.UnixNano()), OrganizationID: orgID, UserID: userID, Role: "owner", Status: "active", JoinedAt: now}
    if err := database.DB.Create(mem).Error; err != nil { return "", false }
    return orgID, true
}

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
	if req.Msg.Branch != nil {
		dbDeployment.Branch = req.Msg.GetBranch()
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
	if req.Msg.Domain != nil {
		dbDeployment.Domain = req.Msg.GetDomain()
	}
	if req.Msg.Port != nil {
		port := req.Msg.GetPort()
		dbDeployment.Port = &port
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

func (s *Service) TriggerDeployment(ctx context.Context, req *connect.Request[deploymentsv1.TriggerDeploymentRequest]) (*connect.Response[deploymentsv1.TriggerDeploymentResponse], error) {
	// Check if user has deploy permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "deploy"); err != nil {
		return nil, err
	}
	
	// Update deployment status to deploying
	if err := s.repo.UpdateStatus(ctx, deploymentID, int32(deploymentsv1.DeploymentStatus_DEPLOYING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deployment: %w", err))
	}

	// Simulate async deployment
	go func() {
		time.Sleep(10 * time.Second)
		s.repo.UpdateStatus(context.Background(), req.Msg.GetDeploymentId(), int32(deploymentsv1.DeploymentStatus_RUNNING))
	}()

	dbDeployment, _ := s.repo.GetByID(ctx, req.Msg.GetDeploymentId())
	res := connect.NewResponse(&deploymentsv1.TriggerDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Status:       "DEPLOYING",
	})
	if dbDeployment != nil {
		res.Msg.Status = getStatusName(dbDeployment.Status)
	}
	return res, nil
}

func (s *Service) StreamDeploymentStatus(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentStatusRequest], stream *connect.ServerStream[deploymentsv1.DeploymentStatusUpdate]) error {
	updates := []deploymentsv1.DeploymentStatusUpdate{
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "starting",
			Message:      proto.String("Build started"),
			Timestamp:    timestamppb.Now(),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_DEPLOYING,
			HealthStatus: "verifying",
			Message:      proto.String("Running smoke tests"),
			Timestamp:    timestamppb.New(time.Now().Add(5 * time.Second)),
		},
		{
			DeploymentId: req.Msg.GetDeploymentId(),
			Status:       deploymentsv1.DeploymentStatus_RUNNING,
			HealthStatus: "healthy",
			Message:      proto.String("Deployment complete"),
			Timestamp:    timestamppb.New(time.Now().Add(10 * time.Second)),
		},
	}

	for i := range updates {
		if err := stream.Send(&updates[i]); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (s *Service) GetDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentLogsRequest]) (*connect.Response[deploymentsv1.GetDeploymentLogsResponse], error) {
	lines := req.Msg.GetLines()
	if lines <= 0 {
		lines = 50
	}

	logs := make([]string, 0, lines)
	for i := int32(0); i < lines; i++ {
		logs = append(logs, fmt.Sprintf("[%s] Log line %d for deployment %s", time.Now().Format(time.RFC3339), i+1, req.Msg.GetDeploymentId()))
	}

	res := connect.NewResponse(&deploymentsv1.GetDeploymentLogsResponse{Logs: logs})
	return res, nil
}

func (s *Service) StartDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StartDeploymentRequest]) (*connect.Response[deploymentsv1.StartDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.start", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	// Check if deployment has containers created
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		// No containers exist - need to create them first
		// This happens if CreateDeployment failed or was never called
		// We should trigger container creation
		if s.manager != nil {
			// Get deployment config from database
			image := ""
			if dbDep.Image != nil {
				image = *dbDep.Image
			}
			port := 8080
			if dbDep.Port != nil {
				port = int(*dbDep.Port)
			}
			memory := int64(512 * 1024 * 1024) // Default 512MB
			if dbDep.MemoryBytes != nil {
				memory = *dbDep.MemoryBytes
			}
			cpuShares := int64(1024) // Default
			if dbDep.CPUShares != nil {
				cpuShares = *dbDep.CPUShares
			}
			replicas := 1 // Default
			if dbDep.Replicas != nil {
				replicas = int(*dbDep.Replicas)
			}

			// Recreate containers using deployment config
			cfg := &orchestrator.DeploymentConfig{
				DeploymentID: deploymentID,
				Image:        image,
				Domain:       dbDep.Domain,
				Port:         port,
				EnvVars:      parseEnvVars(dbDep.EnvVars),
				Labels:       map[string]string{},
				Memory:       memory,
				CPUShares:    cpuShares,
				Replicas:     replicas,
			}
			if err := s.manager.CreateDeployment(ctx, cfg); err != nil {
				log.Printf("[StartDeployment] Failed to create containers for deployment %s: %v", deploymentID, err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create containers: %w", err))
			}
			log.Printf("[StartDeployment] Successfully created containers for deployment %s", deploymentID)
		} else {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment has no containers and orchestrator is not available"))
		}
	} else {
		// Containers exist - start them
		if s.manager != nil {
			if err := s.manager.StartDeployment(ctx, deploymentID); err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start containers: %w", err))
			}
		}
	}

	// Update deployment status
	dbDep.Status = int32(deploymentsv1.DeploymentStatus_RUNNING)
	if err := s.repo.Update(ctx, dbDep); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}

	res := connect.NewResponse(&deploymentsv1.StartDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

func (s *Service) StopDeployment(ctx context.Context, req *connect.Request[deploymentsv1.StopDeploymentRequest]) (*connect.Response[deploymentsv1.StopDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.stop", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if s.manager != nil { _ = s.manager.StopDeployment(ctx, deploymentID) }
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID)) }
	dbDep.Status = int32(deploymentsv1.DeploymentStatus_STOPPED)
	if err := s.repo.Update(ctx, dbDep); err != nil { return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop deployment: %w", err)) }
	res := connect.NewResponse(&deploymentsv1.StopDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

func (s *Service) RestartDeployment(ctx context.Context, req *connect.Request[deploymentsv1.RestartDeploymentRequest]) (*connect.Response[deploymentsv1.RestartDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.restart", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if s.manager != nil { _ = s.manager.RestartDeployment(ctx, deploymentID) }
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID)) }
	res := connect.NewResponse(&deploymentsv1.RestartDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

func (s *Service) ScaleDeployment(ctx context.Context, req *connect.Request[deploymentsv1.ScaleDeploymentRequest]) (*connect.Response[deploymentsv1.ScaleDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.scale", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	// Quota check: replicas delta
	newReplicas := int(req.Msg.GetReplicas())
	if newReplicas <= 0 { return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("replicas must be > 0")) }
	if err := s.quotaChecker.CanAllocate(ctx, orgID, quota.RequestedResources{Replicas: newReplicas}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if s.manager != nil { _ = s.manager.ScaleDeployment(ctx, deploymentID, newReplicas) }
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID)) }
	res := connect.NewResponse(&deploymentsv1.ScaleDeploymentResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

func (s *Service) StreamDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine]) error {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	// Find a container for this deployment and stream logs
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}
    // Use the first location for now
    loc := locations[0]
    dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()
    reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", req.Msg.GetTail()), true) // follow=true for streaming
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("logs: %w", err))
	}
	defer reader.Close()
	buf := make([]byte, 4096)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			line := &deploymentsv1.DeploymentLogLine{DeploymentId: deploymentID, Line: string(buf[:n]), Timestamp: timestamppb.Now()}
			if sendErr := stream.Send(line); sendErr != nil { return sendErr }
		}
		if readErr != nil {
			break
		}
	}
	return nil
}

// TODO: implement RestartDeployment, ScaleDeployment, StreamDeploymentLogs similarly with permission + quota checks

func (s *Service) DeleteDeployment(ctx context.Context, req *connect.Request[deploymentsv1.DeleteDeploymentRequest]) (*connect.Response[deploymentsv1.DeleteDeploymentResponse], error) {
	// Check if user has delete permission for this deployment
	deploymentID := req.Msg.GetDeploymentId()
	if err := s.checkDeploymentPermission(ctx, deploymentID, "delete"); err != nil {
		return nil, err
	}
	
	if err := s.repo.Delete(ctx, deploymentID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	res := connect.NewResponse(&deploymentsv1.DeleteDeploymentResponse{Success: true})
	return res, nil
}

func (s *Service) GetDeploymentEnvVars(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentEnvVarsRequest]) (*connect.Response[deploymentsv1.GetDeploymentEnvVarsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	
	envFileContent := dbDep.EnvFileContent
	
	return connect.NewResponse(&deploymentsv1.GetDeploymentEnvVarsResponse{
		EnvFileContent: envFileContent,
	}), nil
}

func (s *Service) UpdateDeploymentEnvVars(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentEnvVarsRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentEnvVarsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	
	envFileContent := req.Msg.GetEnvFileContent()
	
	// Parse to generate env_vars map for backward compatibility with existing code
	envVarsMap := parseEnvFileToMap(envFileContent)
	
	// Marshal env vars to JSON
	envJSON, err := json.Marshal(envVarsMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal env vars: %w", err))
	}
	
	// Update only the env vars fields using repository method
	if err := s.repo.UpdateEnvVars(ctx, deploymentID, envFileContent, string(envJSON)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update env vars: %w", err))
	}
	
	// Reload to get updated state
	dbDep, _ = s.repo.GetByID(ctx, deploymentID)
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentEnvVarsResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

func (s *Service) GetDeploymentCompose(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentComposeRequest]) (*connect.Response[deploymentsv1.GetDeploymentComposeResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	return connect.NewResponse(&deploymentsv1.GetDeploymentComposeResponse{ComposeYaml: dbDep.ComposeYaml}), nil
}

func (s *Service) UpdateDeploymentCompose(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentComposeRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentComposeResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	
	composeYaml := req.Msg.GetComposeYaml()
	
	// Perform comprehensive validation using Docker Compose CLI
	validationErrors := ValidateCompose(ctx, composeYaml)
	
	// Check if there are any errors (severity == "error")
	hasErrors := false
	var firstErrorMsg string
	for _, ve := range validationErrors {
		if ve.Severity == "error" {
			hasErrors = true
			if firstErrorMsg == "" {
				firstErrorMsg = ve.Message
			}
			break
		}
	}
	
	// Only save if there are no errors (warnings are OK)
	if !hasErrors {
		dbDep.ComposeYaml = composeYaml
		if err := s.repo.Update(ctx, dbDep); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update compose: %w", err))
		}
	}
	
	// Reload to get updated state
	dbDep, _ = s.repo.GetByID(ctx, deploymentID)
	
	// Convert validation errors to proto
	protoErrors := make([]*deploymentsv1.ComposeValidationError, 0, len(validationErrors))
	for _, ve := range validationErrors {
		protoErrors = append(protoErrors, &deploymentsv1.ComposeValidationError{
			Line:        ve.Line,
			Column:      ve.Column,
			Message:     ve.Message,
			Severity:    ve.Severity,
			StartLine:   ve.StartLine,
			EndLine:     ve.EndLine,
			StartColumn: ve.StartColumn,
			EndColumn:   ve.EndColumn,
		})
	}
	
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentComposeResponse{
		Deployment:       dbDeploymentToProto(dbDep),
		ValidationErrors: protoErrors,
	})
	
	// Set legacy validation_error for backward compatibility
	if firstErrorMsg != "" {
		res.Msg.ValidationError = &firstErrorMsg
	}
	
	return res, nil
}

func (s *Service) ListGitHubRepos(ctx context.Context, req *connect.Request[deploymentsv1.ListGitHubReposRequest]) (*connect.Response[deploymentsv1.ListGitHubReposResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	
	// TODO: Get GitHub token from user profile/auth context
	// For now, return empty list if no token configured
	// In production, this would come from OAuth integration
	ghToken := "" // userInfo.GitHubToken or from session
	
	ghClient := githubclient.NewClient(ghToken)
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}
	
	repos, err := ghClient.ListRepos(ctx, page, perPage)
	if err != nil {
		// If GitHub API fails (e.g., no token), return empty list
		return connect.NewResponse(&deploymentsv1.ListGitHubReposResponse{
			Repos: []*deploymentsv1.GitHubRepo{},
			Total: 0,
		}), nil
	}
	
	protoRepos := make([]*deploymentsv1.GitHubRepo, 0, len(repos))
	for _, r := range repos {
		protoRepos = append(protoRepos, &deploymentsv1.GitHubRepo{
			Id:            fmt.Sprintf("%d", r.ID),
			Name:          r.Name,
			FullName:      r.FullName,
			Description:   r.Description,
			Url:           r.URL,
			IsPrivate:     r.IsPrivate,
			DefaultBranch: r.DefaultBranch,
		})
	}
	
	return connect.NewResponse(&deploymentsv1.ListGitHubReposResponse{
		Repos: protoRepos,
		Total: int32(len(protoRepos)),
	}), nil
}

func (s *Service) GetGitHubBranches(ctx context.Context, req *connect.Request[deploymentsv1.GetGitHubBranchesRequest]) (*connect.Response[deploymentsv1.GetGitHubBranchesResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	
	repoFullName := req.Msg.GetRepoFullName()
	if repoFullName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repo_full_name is required"))
	}
	
	ghToken := "" // TODO: Get from user context
	ghClient := githubclient.NewClient(ghToken)
	
	branches, err := ghClient.ListBranches(ctx, repoFullName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch branches: %w", err))
	}
	
	protoBranches := make([]*deploymentsv1.GitHubBranch, 0, len(branches))
	for i, b := range branches {
		protoBranches = append(protoBranches, &deploymentsv1.GitHubBranch{
			Name:      b.Name,
			IsDefault: i == 0, // First branch is often default
			Sha:       b.Commit.SHA,
		})
	}
	
	return connect.NewResponse(&deploymentsv1.GetGitHubBranchesResponse{
		Branches: protoBranches,
	}), nil
}

func (s *Service) GetGitHubFile(ctx context.Context, req *connect.Request[deploymentsv1.GetGitHubFileRequest]) (*connect.Response[deploymentsv1.GetGitHubFileResponse], error) {
	_, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	
	repoFullName := req.Msg.GetRepoFullName()
	branch := req.Msg.GetBranch()
	path := req.Msg.GetPath()
	
	if repoFullName == "" || branch == "" || path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repo_full_name, branch, and path are required"))
	}
	
	ghToken := "" // TODO: Get from user context
	ghClient := githubclient.NewClient(ghToken)
	
	fileContent, err := ghClient.GetFile(ctx, repoFullName, branch, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to fetch file: %w", err))
	}
	
	return connect.NewResponse(&deploymentsv1.GetGitHubFileResponse{
		Content:  fileContent.Content,
		Encoding: fileContent.Encoding,
		Size:     fileContent.Size,
	}), nil
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	conn        io.ReadWriteCloser
	containerID string
	createdAt   time.Time
}

// terminalSessions stores active terminal sessions keyed by deploymentID
var terminalSessions = make(map[string]*TerminalSession)
var terminalSessionsMutex sync.RWMutex

// StreamTerminalOutput streams terminal output from a deployment container
// This is a server stream that works well with gRPC-Web in browsers
func (s *Service) StreamTerminalOutput(ctx context.Context, req *connect.Request[deploymentsv1.StreamTerminalOutputRequest], stream *connect.ServerStream[deploymentsv1.TerminalOutput]) error {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	// Find container for this deployment
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}

	loc := locations[0]
	dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Get or create terminal connection
	cols := int(req.Msg.GetCols())
	rows := int(req.Msg.GetRows())
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	// Check if session exists
	terminalSessionsMutex.Lock()
	session, exists := terminalSessions[deploymentID]
	if !exists {
		// Create new terminal connection
		conn, err := dcli.ContainerExec(ctx, loc.ContainerID, cols, rows)
		if err != nil {
			terminalSessionsMutex.Unlock()
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create terminal: %w", err))
		}
		session = &TerminalSession{
			conn:        conn,
			containerID: loc.ContainerID,
			createdAt:   time.Now(),
		}
		terminalSessions[deploymentID] = session
	}
	terminalSessionsMutex.Unlock()

	// Clean up session when stream ends
	defer func() {
		terminalSessionsMutex.Lock()
		if s, exists := terminalSessions[deploymentID]; exists && s == session {
			delete(terminalSessions, deploymentID)
			session.conn.Close()
		}
		terminalSessionsMutex.Unlock()
	}()

	// Read from container stdout/stderr and send to client
	buf := make([]byte, 4096)
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := session.conn.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&deploymentsv1.TerminalOutput{
				Output: buf[:n],
				Exit:   false,
			}); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if err == io.EOF {
				// Terminal closed
				_ = stream.Send(&deploymentsv1.TerminalOutput{
					Output: []byte("\r\n[Terminal session closed]\r\n"),
					Exit:   true,
				})
			}
			return err
		}
	}
}

// SendTerminalInput sends input to an active terminal session
func (s *Service) SendTerminalInput(ctx context.Context, req *connect.Request[deploymentsv1.SendTerminalInputRequest]) (*connect.Response[deploymentsv1.SendTerminalInputResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get session
	terminalSessionsMutex.RLock()
	session, exists := terminalSessions[deploymentID]
	terminalSessionsMutex.RUnlock()

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no active terminal session for deployment"))
	}

	// Write input to container
	if len(req.Msg.GetInput()) > 0 {
		if _, err := session.conn.Write(req.Msg.GetInput()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write input: %w", err))
		}
	}

	// Handle resize if dimensions changed
	if req.Msg.GetCols() > 0 && req.Msg.GetRows() > 0 {
		// TODO: Implement terminal resize via exec resize API
		log.Printf("[Terminal] Resize requested: %dx%d (not implemented)", req.Msg.GetCols(), req.Msg.GetRows())
	}

	return connect.NewResponse(&deploymentsv1.SendTerminalInputResponse{
		Success: true,
	}), nil
}

func (s *Service) ListContainerFiles(ctx context.Context, req *connect.Request[deploymentsv1.ListContainerFilesRequest]) (*connect.Response[deploymentsv1.ListContainerFilesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}
	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Use the first location for now
	loc := locations[0]

	// Check container state
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}
	isRunning := containerInfo.State.Running

	// If list_volumes is true, return list of volumes
	if req.Msg.GetListVolumes() {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get volumes: %w", err))
		}

		volumeInfos := make([]*deploymentsv1.VolumeInfo, len(volumes))
		for i, vol := range volumes {
			volumeInfos[i] = &deploymentsv1.VolumeInfo{
				Name:        vol.Name,
				MountPoint:  vol.MountPoint,
				Source:      vol.Source,
				IsPersistent: vol.IsNamed,
			}
		}

		return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
			Volumes:        volumeInfos,
			ContainerRunning: isRunning,
		}), nil
	}

	// If volume_name is specified, list files from volume
	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get volumes: %w", err))
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}

		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		path := req.Msg.GetPath()
		if path == "" {
			path = "/"
		}

		// List files directly from volume (works even if container is stopped)
		fileInfos, err := dcli.ListVolumeFiles(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list volume files: %w", err))
		}

		// Convert to proto format
		files := make([]*deploymentsv1.ContainerFile, len(fileInfos))
		for i, fi := range fileInfos {
			volName := volumeName
			files[i] = &deploymentsv1.ContainerFile{
				Name:        fi.Name,
				Path:        fi.Path,
				IsDirectory: fi.IsDirectory,
				Size:        fi.Size,
				Permissions: fi.Permissions,
				VolumeName:  &volName,
			}
			if fi.ModifiedAt != "" {
				files[i].ModifiedAt = &fi.ModifiedAt
			}
		}

		return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
			Files:          files,
			CurrentPath:    path,
			IsVolume:       true,
			ContainerRunning: isRunning,
		}), nil
	}

	// Otherwise, list files from container filesystem (only works if running)
	if !isRunning {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running. Use volume_name parameter to access persistent volumes"))
	}

	path := req.Msg.GetPath()
	if path == "" {
		path = "/"
	}

	// List files using Docker exec (container must be running)
	fileInfos, err := dcli.ContainerListFiles(ctx, loc.ContainerID, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list files: %w", err))
	}

	// Convert to proto format
	files := make([]*deploymentsv1.ContainerFile, len(fileInfos))
	for i, fi := range fileInfos {
		files[i] = &deploymentsv1.ContainerFile{
			Name:        fi.Name,
			Path:        fi.Path,
			IsDirectory: fi.IsDirectory,
			Size:        fi.Size,
			Permissions: fi.Permissions,
		}
		if fi.ModifiedAt != "" {
			files[i].ModifiedAt = &fi.ModifiedAt
		}
	}

	return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
		Files:          files,
		CurrentPath:    path,
		IsVolume:       false,
		ContainerRunning: isRunning,
	}), nil
}

func (s *Service) GetContainerFile(ctx context.Context, req *connect.Request[deploymentsv1.GetContainerFileRequest]) (*connect.Response[deploymentsv1.GetContainerFileResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path is required"))
	}

	// Use the first location for now
	loc := locations[0]

	// If volume_name is specified, read file from volume
	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get volumes: %w", err))
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}

		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		// Read file directly from volume (works even if container is stopped)
		content, err := dcli.ReadVolumeFile(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read volume file: %w", err))
		}

		return connect.NewResponse(&deploymentsv1.GetContainerFileResponse{
			Content:  string(content),
			Encoding: "text",
			Size:     int64(len(content)),
		}), nil
	}

	// Otherwise, read file from container filesystem (only works if running)
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running. Use volume_name parameter to read files from persistent volumes"))
	}

	// Read file using Docker exec (container must be running)
	content, err := dcli.ContainerReadFile(ctx, loc.ContainerID, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read file: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.GetContainerFileResponse{
		Content:  string(content),
		Encoding: "text",
		Size:     int64(len(content)),
	}), nil
}

func (s *Service) UploadContainerFiles(ctx context.Context, stream *connect.ClientStream[deploymentsv1.UploadContainerFilesRequest]) (*connect.Response[deploymentsv1.UploadContainerFilesResponse], error) {
	var metadata *deploymentsv1.UploadContainerFilesMetadata
	var fileData bytes.Buffer

	// Read all stream messages
	for stream.Receive() {
		msg := stream.Msg()
		
		switch d := msg.Data.(type) {
		case *deploymentsv1.UploadContainerFilesRequest_Metadata:
			metadata = d.Metadata
		case *deploymentsv1.UploadContainerFilesRequest_Chunk:
			fileData.Write(d.Chunk)
		}
	}

	if err := stream.Err(); err != nil && err != io.EOF {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to receive stream: %w", err))
	}

	if metadata == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata is required"))
	}

	deploymentID := metadata.GetDeploymentId()
	orgID := metadata.GetOrganizationId()
	
	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get deployment locations
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Use the first location
	loc := locations[0]
	destPath := metadata.GetDestinationPath()
	if destPath == "" {
		destPath = "/"
	}

	// Extract files from tar archive
	files := make(map[string][]byte)
	tarReader := tar.NewReader(&fileData)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read tar: %w", err))
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read file from tar: %w", err))
		}

		files[hdr.Name] = content
	}

	// Upload files to container
	err = dcli.ContainerUploadFiles(ctx, loc.ContainerID, destPath, files)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload files: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.UploadContainerFilesResponse{
		Success:      true,
		FilesUploaded: int32(len(files)),
	}), nil
}

// parseEnvVars parses environment variables from JSON string stored in database
func parseEnvVars(envVarsJSON string) map[string]string {
	if envVarsJSON == "" {
		return make(map[string]string)
	}
	var envMap map[string]string
	if err := json.Unmarshal([]byte(envVarsJSON), &envMap); err != nil {
		return make(map[string]string)
	}
	return envMap
}

// parseEnvFileToMap parses a .env file content and extracts key-value pairs (ignores comments)
// Used internally to maintain backward compatibility with EnvVars JSON field
func parseEnvFileToMap(envFileContent string) map[string]string {
	envMap := make(map[string]string)
	if envFileContent == "" {
		return envMap
	}

	lines := strings.Split(envFileContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		equalIndex := strings.Index(trimmed, "=")
		if equalIndex == -1 {
			continue
		}

		key := strings.TrimSpace(trimmed[:equalIndex])
		value := strings.TrimSpace(trimmed[equalIndex+1:])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Uppercase key (env var standard)
		envMap[strings.ToUpper(key)] = value
	}

	return envMap
}

func getStatusName(status int32) string {
	switch deploymentsv1.DeploymentStatus(status) {
	case deploymentsv1.DeploymentStatus_CREATED:
		return "CREATED"
	case deploymentsv1.DeploymentStatus_BUILDING:
		return "BUILDING"
	case deploymentsv1.DeploymentStatus_RUNNING:
		return "RUNNING"
	case deploymentsv1.DeploymentStatus_STOPPED:
		return "STOPPED"
	case deploymentsv1.DeploymentStatus_FAILED:
		return "FAILED"
	case deploymentsv1.DeploymentStatus_DEPLOYING:
		return "DEPLOYING"
	default:
		return "UNSPECIFIED"
	}
}

func (s *Service) GetDeploymentRoutings(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRoutingsRequest]) (*connect.Response[deploymentsv1.GetDeploymentRoutingsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	
	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	
	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}
	
	// Get all routing rules
	dbRoutings, err := database.GetDeploymentRoutings(deploymentID)
	if err != nil {
		// If no routing rules exist, return empty list (not an error)
		return connect.NewResponse(&deploymentsv1.GetDeploymentRoutingsResponse{Rules: []*deploymentsv1.RoutingRule{}}), nil
	}
	
	// Convert to proto
	rules := make([]*deploymentsv1.RoutingRule, 0, len(dbRoutings))
	for _, dbRouting := range dbRoutings {
		rules = append(rules, &deploymentsv1.RoutingRule{
			Id:                dbRouting.ID,
			DeploymentId:      dbRouting.DeploymentID,
			Domain:            dbRouting.Domain,
			ServiceName:       dbRouting.ServiceName,
			PathPrefix:        dbRouting.PathPrefix,
			TargetPort:        int32(dbRouting.TargetPort),
			Protocol:          dbRouting.Protocol,
			LoadBalancerAlgo:  dbRouting.LoadBalancerAlgo,
			SslEnabled:        dbRouting.SSLEnabled,
			SslCertResolver:   dbRouting.SSLCertResolver,
		})
	}
	
	return connect.NewResponse(&deploymentsv1.GetDeploymentRoutingsResponse{Rules: rules}), nil
}

func (s *Service) UpdateDeploymentRoutings(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentRoutingsRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentRoutingsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	
	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	
	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}
	
	// Delete all existing routing rules for this deployment
	existingRoutings, _ := database.GetDeploymentRoutings(deploymentID)
	for _, routing := range existingRoutings {
		if err := database.DB.Delete(&routing).Error; err != nil {
			log.Printf("[UpdateDeploymentRoutings] Warning: Failed to delete existing routing %s: %v", routing.ID, err)
		}
	}
	
	// Create new routing rules
	newRules := make([]*deploymentsv1.RoutingRule, 0, len(req.Msg.GetRules()))
	for _, rule := range req.Msg.GetRules() {
		// Generate ID if not provided
		ruleID := rule.GetId()
		if ruleID == "" {
			ruleID = fmt.Sprintf("route-%s-%s-%s-%d", deploymentID, rule.GetDomain(), rule.GetServiceName(), rule.GetTargetPort())
		}
		
		// Set defaults
		serviceName := rule.GetServiceName()
		if serviceName == "" {
			serviceName = "default"
		}
		protocol := rule.GetProtocol()
		if protocol == "" {
			protocol = "http"
		}
		loadBalancerAlgo := rule.GetLoadBalancerAlgo()
		if loadBalancerAlgo == "" {
			loadBalancerAlgo = "round-robin"
		}
		
		dbRouting := &database.DeploymentRouting{
			ID:                ruleID,
			DeploymentID:     deploymentID,
			Domain:            rule.GetDomain(),
			ServiceName:       serviceName,
			PathPrefix:        rule.GetPathPrefix(),
			TargetPort:        int(rule.GetTargetPort()),
			Protocol:          protocol,
			LoadBalancerAlgo:  loadBalancerAlgo,
			SSLEnabled:        rule.GetSslEnabled(),
			SSLCertResolver:   rule.GetSslCertResolver(),
			Middleware:        "{}",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		
		if err := database.UpsertDeploymentRouting(dbRouting); err != nil {
			log.Printf("[UpdateDeploymentRoutings] Warning: Failed to create routing rule for %s: %v", rule.GetDomain(), err)
			continue
		}
		
		// Convert back to proto for response
		newRules = append(newRules, &deploymentsv1.RoutingRule{
			Id:                dbRouting.ID,
			DeploymentId:     dbRouting.DeploymentID,
			Domain:            dbRouting.Domain,
			ServiceName:       dbRouting.ServiceName,
			PathPrefix:        dbRouting.PathPrefix,
			TargetPort:        int32(dbRouting.TargetPort),
			Protocol:          dbRouting.Protocol,
			LoadBalancerAlgo:  dbRouting.LoadBalancerAlgo,
			SslEnabled:        dbRouting.SSLEnabled,
			SslCertResolver:   dbRouting.SSLCertResolver,
		})
	}
	
	return connect.NewResponse(&deploymentsv1.UpdateDeploymentRoutingsResponse{Rules: newRules}), nil
}

func (s *Service) GetDeploymentServiceNames(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentServiceNamesRequest]) (*connect.Response[deploymentsv1.GetDeploymentServiceNamesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	
	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	
	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}
	
	// Extract service names from Docker Compose
	serviceNames, err := ExtractServiceNames(dbDeployment.ComposeYaml)
	if err != nil {
		log.Printf("[GetDeploymentServiceNames] Warning: Failed to parse compose for deployment %s: %v", deploymentID, err)
		// Return default service name on error
		serviceNames = []string{"default"}
	}
	
	return connect.NewResponse(&deploymentsv1.GetDeploymentServiceNamesResponse{
		ServiceNames: serviceNames,
	}), nil
}

