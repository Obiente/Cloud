package deployments

import (
	"context"
	"fmt"
	"log"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListBuilds lists all builds for a deployment
func (s *Service) ListBuilds(ctx context.Context, req *connect.Request[deploymentsv1.ListBuildsRequest]) (*connect.Response[deploymentsv1.ListBuildsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 50 // Default
	}
	offset := int(req.Msg.GetOffset())

	builds, total, err := s.buildHistoryRepo.ListBuilds(ctx, deploymentID, orgID, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list builds: %w", err))
	}

	protoBuilds := make([]*deploymentsv1.Build, 0, len(builds))
	for _, build := range builds {
		protoBuilds = append(protoBuilds, dbBuildToProto(build))
	}

	return connect.NewResponse(&deploymentsv1.ListBuildsResponse{
		Builds: protoBuilds,
		Total:   int32(total),
	}), nil
}

// GetBuild gets details of a specific build
func (s *Service) GetBuild(ctx context.Context, req *connect.Request[deploymentsv1.GetBuildRequest]) (*connect.Response[deploymentsv1.GetBuildResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	buildID := req.Msg.GetBuildId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	build, err := s.buildHistoryRepo.GetBuildByID(ctx, buildID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %w", err))
	}

	// Verify build belongs to the deployment and organization
	if build.DeploymentID != deploymentID || build.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found"))
	}

	return connect.NewResponse(&deploymentsv1.GetBuildResponse{
		Build: dbBuildToProto(build),
	}), nil
}

// GetBuildLogs gets logs for a specific build
func (s *Service) GetBuildLogs(ctx context.Context, req *connect.Request[deploymentsv1.GetBuildLogsRequest]) (*connect.Response[deploymentsv1.GetBuildLogsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	buildID := req.Msg.GetBuildId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify build exists and belongs to deployment
	build, err := s.buildHistoryRepo.GetBuildByID(ctx, buildID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %w", err))
	}
	if build.DeploymentID != deploymentID || build.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found"))
	}

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())

	// Use TimescaleDB repository for build logs
	buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
	logs, total, err := buildLogsRepo.GetBuildLogs(ctx, buildID, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get build logs: %w", err))
	}

	protoLogs := make([]*deploymentsv1.DeploymentLogLine, 0, len(logs))
	for _, logEntry := range logs {
		protoLogs = append(protoLogs, &deploymentsv1.DeploymentLogLine{
			DeploymentId: deploymentID,
			Line:         logEntry.Line,
			Timestamp:    timestamppb.New(logEntry.Timestamp),
			Stderr:       logEntry.Stderr,
			LogLevel:     commonv1.LogLevel_LOG_LEVEL_INFO, // Default, could be enhanced
		})
	}

	return connect.NewResponse(&deploymentsv1.GetBuildLogsResponse{
		Logs:  protoLogs,
		Total: int32(total),
	}), nil
}

// RevertToBuild reverts a deployment to a previous build
func (s *Service) RevertToBuild(ctx context.Context, req *connect.Request[deploymentsv1.RevertToBuildRequest]) (*connect.Response[deploymentsv1.RevertToBuildResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	buildID := req.Msg.GetBuildId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentDeploy, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get the build to revert to
	build, err := s.buildHistoryRepo.GetBuildByID(ctx, buildID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %w", err))
	}

	// Verify build belongs to the deployment and organization
	if build.DeploymentID != deploymentID || build.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found"))
	}

	// Only allow reverting to successful builds
	if build.Status != 3 { // BUILD_SUCCESS = 3
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("can only revert to successful builds"))
	}

	// Get the deployment
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %w", err))
	}

	// Restore deployment configuration from build snapshot
	if build.RepositoryURL != nil {
		dbDeployment.RepositoryURL = build.RepositoryURL
	}
	dbDeployment.Branch = build.Branch
	if build.BuildCommand != nil {
		dbDeployment.BuildCommand = build.BuildCommand
	}
	if build.InstallCommand != nil {
		dbDeployment.InstallCommand = build.InstallCommand
	}
	if build.StartCommand != nil {
		dbDeployment.StartCommand = build.StartCommand
	}
	if build.DockerfilePath != nil {
		dbDeployment.DockerfilePath = build.DockerfilePath
	}
	if build.ComposeFilePath != nil {
		dbDeployment.ComposeFilePath = build.ComposeFilePath
	}
	dbDeployment.BuildStrategy = build.BuildStrategy

	// Restore build results
	if build.ImageName != nil {
		dbDeployment.Image = build.ImageName
	}
	if build.ComposeYaml != nil {
		dbDeployment.ComposeYaml = *build.ComposeYaml
	}

	// Update deployment
	if err := s.repo.Update(ctx, dbDeployment); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment: %w", err))
	}

	// Trigger a new build with the reverted configuration
	// This creates a new build that will use the restored configuration
	newBuildID, err := s.triggerBuildFromRevert(ctx, deploymentID, buildID)
	if err != nil {
		log.Printf("[RevertToBuild] Failed to trigger new build: %v", err)
		// Continue anyway - the configuration has been restored
	}

	protoDeployment := dbDeploymentToProto(dbDeployment)

	return connect.NewResponse(&deploymentsv1.RevertToBuildResponse{
		Deployment: protoDeployment,
		NewBuildId: newBuildID,
	}), nil
}

// triggerBuildFromRevert triggers a new build after reverting configuration
func (s *Service) triggerBuildFromRevert(ctx context.Context, deploymentID, revertedBuildID string) (string, error) {
	// Automatically trigger a new deployment with the reverted configuration
	// This will create a new build record and start the build process
	_, err := s.TriggerDeployment(ctx, connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{
		DeploymentId: deploymentID,
	}))
	
	if err != nil {
		return "", fmt.Errorf("failed to trigger deployment: %w", err)
	}
	
	// Return empty string - the new build ID will be created by TriggerDeployment
	// In the future, we could return the build ID if we modify TriggerDeployment to return it
	return "", nil
}

// dbBuildToProto converts database BuildHistory to proto Build
func dbBuildToProto(build *database.BuildHistory) *deploymentsv1.Build {
	protoBuild := &deploymentsv1.Build{
		Id:             build.ID,
		DeploymentId:   build.DeploymentID,
		OrganizationId: build.OrganizationID,
		BuildNumber:    build.BuildNumber,
		Status:         deploymentsv1.BuildStatus(build.Status),
		StartedAt:      timestamppb.New(build.StartedAt),
		BuildTime:      build.BuildTime,
		TriggeredBy:    build.TriggeredBy,
		Branch:         build.Branch,
		BuildStrategy:  deploymentsv1.BuildStrategy(build.BuildStrategy),
		CreatedAt:      timestamppb.New(build.CreatedAt),
		UpdatedAt:      timestamppb.New(build.UpdatedAt),
	}

	if build.CompletedAt != nil {
		protoBuild.CompletedAt = timestamppb.New(*build.CompletedAt)
	}
	if build.RepositoryURL != nil {
		protoBuild.RepositoryUrl = build.RepositoryURL
	}
	if build.CommitSHA != nil {
		protoBuild.CommitSha = build.CommitSHA
	}
	if build.BuildCommand != nil {
		protoBuild.BuildCommand = build.BuildCommand
	}
	if build.InstallCommand != nil {
		protoBuild.InstallCommand = build.InstallCommand
	}
	if build.StartCommand != nil {
		protoBuild.StartCommand = build.StartCommand
	}
	if build.DockerfilePath != nil {
		protoBuild.DockerfilePath = build.DockerfilePath
	}
	if build.ComposeFilePath != nil {
		protoBuild.ComposeFilePath = build.ComposeFilePath
	}
	if build.ImageName != nil {
		protoBuild.ImageName = build.ImageName
	}
	if build.ComposeYaml != nil {
		protoBuild.ComposeYaml = build.ComposeYaml
	}
	if build.Size != nil {
		protoBuild.Size = build.Size
	}
	if build.Error != nil {
		protoBuild.Error = build.Error
	}

	return protoBuild
}

// DeleteBuild deletes a build from history
func (s *Service) DeleteBuild(ctx context.Context, req *connect.Request[deploymentsv1.DeleteBuildRequest]) (*connect.Response[deploymentsv1.DeleteBuildResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	buildID := req.Msg.GetBuildId()

	// Check permissions - need deployments.manage permission
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentManage, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify build exists and belongs to deployment
	build, err := s.buildHistoryRepo.GetBuildByID(ctx, buildID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %w", err))
	}
	if build.DeploymentID != deploymentID || build.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found"))
	}

	// Delete logs from TimescaleDB first (logs are stored separately from builds)
	buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
	if err := buildLogsRepo.DeleteBuildLogs(ctx, buildID); err != nil {
		logger.Warn("[DeleteBuild] Failed to delete logs for build %s: %v", buildID, err)
		// Continue anyway - try to delete the build
	}

	// Delete the build from PostgreSQL
	if err := s.buildHistoryRepo.DeleteBuild(ctx, buildID, orgID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete build: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.DeleteBuildResponse{
		Success: true,
	}), nil
}

