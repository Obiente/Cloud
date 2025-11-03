package deployments

import (
	"context"
	"fmt"
	"log"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// findContainerForDeployment finds a container by container_id or service_name
// preferRunning: if true, prefer running containers when container_id/service_name not specified
func (s *Service) findContainerForDeployment(ctx context.Context, deploymentID, containerID, serviceName string, dcli *docker.Client) (*database.DeploymentLocation, error) {
	// Get all locations (not just running ones) for file operations
	// This ensures we can access containers even if they're not marked as "running" in the DB
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %w", err)
	}

	// If no locations found, try to validate and refresh (this discovers containers from Docker)
	if len(locations) == 0 {
		locations, err = database.ValidateAndRefreshLocations(deploymentID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate locations: %w", err)
		}
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("no containers for deployment")
	}

	// If container_id is provided, find by ID
	// Docker supports matching by prefix, so we should support that too
	if containerID != "" {
		// Normalize the input container ID (lowercase, no spaces)
		normalizedInputID := strings.ToLower(strings.TrimSpace(containerID))
		
		for _, loc := range locations {
			// Normalize stored container ID
			normalizedStoredID := strings.ToLower(strings.TrimSpace(loc.ContainerID))
			
			// Exact match or prefix match (Docker-style)
			if normalizedStoredID == normalizedInputID ||
				strings.HasPrefix(normalizedStoredID, normalizedInputID) ||
				strings.HasPrefix(normalizedInputID, normalizedStoredID) {
				return &loc, nil
			}
		}
		
		// If not found, try refreshing locations and search again
		// This helps when containers were recently created or the database is stale
		refreshedLocations, refreshErr := database.ValidateAndRefreshLocations(deploymentID)
		if refreshErr == nil && len(refreshedLocations) > 0 {
			for _, loc := range refreshedLocations {
				normalizedStoredID := strings.ToLower(strings.TrimSpace(loc.ContainerID))
				
				if normalizedStoredID == normalizedInputID ||
					strings.HasPrefix(normalizedStoredID, normalizedInputID) ||
					strings.HasPrefix(normalizedInputID, normalizedStoredID) {
					return &loc, nil
				}
			}
		}
		
		// Extract short ID for error message
		shortID := containerID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		return nil, fmt.Errorf("container %s not found for deployment %s", shortID, deploymentID)
	}

	// If service_name is provided, find by service name (check container labels)
	// Prefer running containers for the service
	if serviceName != "" {
		var runningContainer *database.DeploymentLocation
		var anyContainer *database.DeploymentLocation

		for _, loc := range locations {
			containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
			if err != nil {
				continue
			}

			// Check both label formats
			labelServiceName := ""
			if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
				labelServiceName = containerInfo.Config.Labels["cloud.obiente.service_name"]
				if labelServiceName == "" {
					labelServiceName = containerInfo.Config.Labels["com.docker.compose.service"]
				}
			}

			if labelServiceName == serviceName {
				if anyContainer == nil {
					anyContainer = &loc
				}
				// Prefer running containers
				if containerInfo.State.Running && runningContainer == nil {
					runningContainer = &loc
				}
			}
		}

		if runningContainer != nil {
			return runningContainer, nil
		}
		if anyContainer != nil {
			return anyContainer, nil
		}
		return nil, fmt.Errorf("service %s not found for deployment %s", serviceName, deploymentID)
	}

	// Default: prefer a running container, but return any container if none are running
	// Check actual Docker status (not DB status) for accuracy
	var runningContainer *database.DeploymentLocation
	var anyContainer *database.DeploymentLocation

	for _, loc := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			// Container might not exist, but keep it as fallback
			if anyContainer == nil {
				anyContainer = &loc
			}
			continue
		}

		// Track any valid container as fallback
		if anyContainer == nil {
			anyContainer = &loc
		}

		// Prefer running containers - check actual Docker status
		if containerInfo.State.Running && runningContainer == nil {
			runningContainer = &loc
		}
	}

	// Prefer running container, fallback to any container, fallback to first location
	if runningContainer != nil {
		return runningContainer, nil
	}
	if anyContainer != nil {
		return anyContainer, nil
	}

	// Return first container (may not be running, but user can still access volumes)
	return &locations[0], nil
}

// ListDeploymentContainers lists all containers for a deployment
func (s *Service) ListDeploymentContainers(ctx context.Context, req *connect.Request[deploymentsv1.ListDeploymentContainersRequest]) (*connect.Response[deploymentsv1.ListDeploymentContainersResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Get all locations for this deployment (including stopped containers)
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		// Fallback to validate and refresh if GetAllDeploymentLocations fails
		locations, err = database.ValidateAndRefreshLocations(deploymentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get locations: %w", err))
		}
	}

	containers := make([]*deploymentsv1.DeploymentContainer, 0, len(locations))

	for _, loc := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			log.Printf("[ListDeploymentContainers] Failed to inspect container %s: %v", loc.ContainerID[:12], err)
			continue
		}

		// Extract service name from labels
		serviceName := ""
		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			serviceName = containerInfo.Config.Labels["cloud.obiente.service_name"]
			if serviceName == "" {
				serviceName = containerInfo.Config.Labels["com.docker.compose.service"]
			}
			if serviceName == "" {
				serviceName = "default"
			}
		}

		status := containerInfo.State.Status
		if containerInfo.State.Running {
			status = "running"
		}

		container := &deploymentsv1.DeploymentContainer{
			ContainerId: loc.ContainerID,
			Status:      status,
			NodeId:      &loc.NodeID,
			Port:        func() *int32 { p := int32(loc.Port); return &p }(),
			CreatedAt:   timestamppb.New(loc.CreatedAt),
			UpdatedAt:   timestamppb.New(loc.UpdatedAt),
		}

		if serviceName != "" && serviceName != "default" {
			container.ServiceName = &serviceName
		}
		if loc.NodeHostname != "" {
			container.NodeHostname = &loc.NodeHostname
		}

		containers = append(containers, container)
	}

	return connect.NewResponse(&deploymentsv1.ListDeploymentContainersResponse{
		Containers: containers,
	}), nil
}

// getDeploymentContainerStatus gets the actual running/total container counts from Docker
// This checks Docker directly, not the database, for accurate status
func (s *Service) getDeploymentContainerStatus(ctx context.Context, deploymentID string) (runningCount int32, totalCount int32, err error) {
	dcli, err := docker.New()
	if err != nil {
		return 0, 0, fmt.Errorf("docker client: %w", err)
	}
	defer dcli.Close()

	// Get all locations for this deployment
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		// Fallback to validate and refresh if GetAllDeploymentLocations fails
		locations, err = database.ValidateAndRefreshLocations(deploymentID)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get locations: %w", err)
		}
	}

	totalCount = int32(len(locations))
	if totalCount == 0 {
		return 0, 0, nil
	}

	// Inspect each container to check actual Docker status
	for _, loc := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			// Container might not exist, skip it
			continue
		}

		// Check actual running status from Docker
		if containerInfo.State.Running {
			runningCount++
		}
	}

	return runningCount, totalCount, nil
}

// StreamContainerLogs streams logs from a specific container
func (s *Service) StreamContainerLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamContainerLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine]) error {
	// Ensure user is authenticated for streaming RPCs
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	containerID := req.Msg.GetContainerId()

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find the specific container
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, "", dcli)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}

	tail := req.Msg.GetTail()
	if tail <= 0 {
		tail = 200
	}

	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", tail), true) // follow=true for streaming
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("logs: %w", err))
	}
	defer reader.Close()

	buf := make([]byte, 4096)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			// Sanitize to valid UTF-8
			sanitizedLine := strings.ToValidUTF8(string(buf[:n]), "")
			// Detect log level from content
			logLevel := detectLogLevelFromContent(sanitizedLine, false)
			line := &deploymentsv1.DeploymentLogLine{
				DeploymentId: deploymentID,
				Line:         sanitizedLine,
				Timestamp:    timestamppb.Now(),
				Stderr:       false,
				LogLevel:     logLevel,
			}
			if sendErr := stream.Send(line); sendErr != nil {
				return sendErr
			}
		}
		if readErr != nil {
			break
		}
	}
	return nil
}

// StartContainer starts a specific container
func (s *Service) StartContainer(ctx context.Context, req *connect.Request[deploymentsv1.StartContainerRequest]) (*connect.Response[deploymentsv1.StartContainerResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	containerID := req.Msg.GetContainerId()

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.manage", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find the specific container
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, "", dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := dcli.StartContainer(ctx, loc.ContainerID); err != nil {
		return connect.NewResponse(&deploymentsv1.StartContainerResponse{
			Success: false,
			Error:   &[]string{err.Error()}[0],
		}), nil
	}

	// Update location status
	database.DB.Model(&database.DeploymentLocation{}).
		Where("container_id = ?", loc.ContainerID).
		Update("status", "running")

	return connect.NewResponse(&deploymentsv1.StartContainerResponse{
		Success: true,
	}), nil
}

// StopContainer stops a specific container
func (s *Service) StopContainer(ctx context.Context, req *connect.Request[deploymentsv1.StopContainerRequest]) (*connect.Response[deploymentsv1.StopContainerResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	containerID := req.Msg.GetContainerId()

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.manage", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find the specific container
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, "", dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := dcli.StopContainer(ctx, loc.ContainerID, 30*time.Second); err != nil {
		return connect.NewResponse(&deploymentsv1.StopContainerResponse{
			Success: false,
			Error:   &[]string{err.Error()}[0],
		}), nil
	}

	// Update location status
	database.DB.Model(&database.DeploymentLocation{}).
		Where("container_id = ?", loc.ContainerID).
		Update("status", "stopped")

	return connect.NewResponse(&deploymentsv1.StopContainerResponse{
		Success: true,
	}), nil
}

// RestartContainer restarts a specific container
func (s *Service) RestartContainer(ctx context.Context, req *connect.Request[deploymentsv1.RestartContainerRequest]) (*connect.Response[deploymentsv1.RestartContainerResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	containerID := req.Msg.GetContainerId()

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.manage", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find the specific container
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, "", dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err := dcli.RestartContainer(ctx, loc.ContainerID, 30*time.Second); err != nil {
		return connect.NewResponse(&deploymentsv1.RestartContainerResponse{
			Success: false,
			Error:   &[]string{err.Error()}[0],
		}), nil
	}

	// Update location status
	database.DB.Model(&database.DeploymentLocation{}).
		Where("container_id = ?", loc.ContainerID).
		Update("status", "running")

	return connect.NewResponse(&deploymentsv1.RestartContainerResponse{
		Success: true,
	}), nil
}
