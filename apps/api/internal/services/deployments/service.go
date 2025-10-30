package deployments

import (
	"context"
	"fmt"
	"io"
	"strings"
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
		_ = s.manager.CreateDeployment(ctx, cfg) // best-effort; DB state already created, errors can be reconciled
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

	// Update deployment fields
	if req.Msg.Name != nil {
		dbDeployment.Name = req.Msg.GetName()
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

	// Update status fields
	dbDeployment.Status = int32(deploymentsv1.DeploymentStatus_BUILDING)
	dbDeployment.HealthStatus = "pending"
	dbDeployment.LastDeployedAt = time.Now()

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
	// TODO: enforce quota if starting increases replicas/resources
	if s.manager != nil {
		// No-op here; Start applies to already created/running containers in our current model
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID)) }
	dbDep.Status = int32(deploymentsv1.DeploymentStatus_RUNNING)
	if err := s.repo.Update(ctx, dbDep); err != nil { return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start deployment: %w", err)) }
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
	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", req.Msg.GetTail()))
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
	protoDep := dbDeploymentToProto(dbDep)
	return connect.NewResponse(&deploymentsv1.GetDeploymentEnvVarsResponse{EnvVars: protoDep.EnvVars}), nil
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
	// Update env vars in proto, then convert back to DB
	protoDep := dbDeploymentToProto(dbDep)
	protoDep.EnvVars = req.Msg.GetEnvVars()
	updatedDB := protoToDBDeployment(protoDep, dbDep.OrganizationID, dbDep.CreatedBy)
	updatedDB.ID = dbDep.ID
	updatedDB.CreatedAt = dbDep.CreatedAt
	if err := s.repo.Update(ctx, updatedDB); err != nil {
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
	// Basic validation: check for required compose fields
	var validationErr string
	if composeYaml != "" {
		// Basic YAML structure checks
		if !strings.Contains(composeYaml, "services:") && !strings.Contains(composeYaml, "version:") {
			// Allow compose files without version (v3.8+), but require services
			if !strings.Contains(composeYaml, "services:") {
				validationErr = "Docker Compose must include a 'services:' section"
			}
		}
	}
	
	dbDep.ComposeYaml = composeYaml
	if err := s.repo.Update(ctx, dbDep); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update compose: %w", err))
	}
	
	// Reload to get updated state
	dbDep, _ = s.repo.GetByID(ctx, deploymentID)
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentComposeResponse{
		Deployment: dbDeploymentToProto(dbDep),
	})
	if validationErr != "" {
		res.Msg.ValidationError = &validationErr
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

func (s *Service) StreamTerminal(ctx context.Context, stream *connect.BidiStream[deploymentsv1.TerminalInput, deploymentsv1.TerminalOutput]) error {
	// Receive first message to get deployment info
	input, err := stream.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive terminal setup: %w", err)
	}

	deploymentID := input.GetDeploymentId()
	orgID := input.GetOrganizationId()

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

	// Create terminal connection
	cols := int(input.GetCols())
	rows := int(input.GetRows())
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	conn, err := dcli.ContainerExec(ctx, loc.ContainerID, cols, rows)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create terminal: %w", err))
	}
	defer conn.Close()

	// Handle bidirectional communication
	errChan := make(chan error, 2)

	// Read from container stdout/stderr and send to client
	go func() {
		defer close(errChan)
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if sendErr := stream.Send(&deploymentsv1.TerminalOutput{
					Output: buf[:n],
					Exit:   false,
				}); sendErr != nil {
					errChan <- sendErr
					return
				}
			}
			if err != nil {
				// Terminal closed
				_ = stream.Send(&deploymentsv1.TerminalOutput{
					Output: []byte("\r\n[Terminal session closed]\r\n"),
					Exit:   true,
				})
				return
			}
		}
	}()

	// Read from client and write to container stdin
	go func() {
		for {
			msg, err := stream.Receive()
			if err != nil {
				errChan <- err
				return
			}
			// Update terminal size if provided
			if msg.GetCols() > 0 && msg.GetRows() > 0 {
				// TODO: Resize terminal (would need exec ID)
			}
			// Write input to container
			if len(msg.GetInput()) > 0 {
				if _, writeErr := conn.Write(msg.GetInput()); writeErr != nil {
					errChan <- writeErr
					return
				}
			}
		}
	}()

	// Wait for either goroutine to finish
	if err := <-errChan; err != nil && err != io.EOF {
		return err
	}

	return nil
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

	loc := locations[0]
	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	path := req.Msg.GetPath()
	if path == "" {
		path = "/"
	}

	// Use Docker exec to run ls command
	// TODO: Implement proper file listing using Docker Copy API or exec
	// For now, return empty list as placeholder
	files := []*deploymentsv1.ContainerFile{}

	return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
		Files:       files,
		CurrentPath: path,
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

	loc := locations[0]
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path is required"))
	}

	// TODO: Implement file reading using Docker Copy API
	// For now, return placeholder
	return connect.NewResponse(&deploymentsv1.GetContainerFileResponse{
		Content:  "",
		Encoding: "text",
		Size:     0,
	}), nil
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

