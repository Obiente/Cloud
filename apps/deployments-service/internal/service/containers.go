package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// attemptAutomaticRedeployment attempts to automatically start/redeploy a deployment
// that should be running but has no containers
func (s *Service) attemptAutomaticRedeployment(ctx context.Context, deploymentID string) error {
	// Get deployment from database to check status
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Only attempt redeployment if deployment should be running
	status := deploymentsv1.DeploymentStatus(dbDep.Status)
	if status != deploymentsv1.DeploymentStatus_RUNNING && status != deploymentsv1.DeploymentStatus_DEPLOYING {
		return fmt.Errorf("deployment status is %s, not attempting redeployment", getStatusName(dbDep.Status))
	}

	log.Printf("[attemptAutomaticRedeployment] Deployment %s should be running but has no containers, attempting automatic redeployment", deploymentID)

	// First, try to start existing containers (they might just be stopped)
	if s.manager != nil {
		if err := s.manager.StartDeployment(ctx, deploymentID); err == nil {
			log.Printf("[attemptAutomaticRedeployment] Successfully started existing containers for deployment %s", deploymentID)
			// Wait a bit for containers to start
			time.Sleep(1 * time.Second)
			return nil
		}
		log.Printf("[attemptAutomaticRedeployment] No existing containers to start for deployment %s, will create new ones", deploymentID)
	}

	// Check if this is a compose-based deployment
	if dbDep.ComposeYaml != "" {
		// Deploy using Docker Compose
		if s.manager != nil {
			if err := s.manager.DeployComposeFile(ctx, deploymentID, dbDep.ComposeYaml); err != nil {
				log.Printf("[attemptAutomaticRedeployment] Failed to deploy compose file for deployment %s: %v", deploymentID, err)
				return fmt.Errorf("failed to deploy compose file: %w", err)
			}
			log.Printf("[attemptAutomaticRedeployment] Successfully deployed compose file for deployment %s", deploymentID)
		} else {
			return fmt.Errorf("compose deployment requires orchestrator")
		}
	} else {
		// Regular container-based deployment
		if s.manager != nil {
			// Get deployment config from database
			image := ""
			if dbDep.Image != nil {
				image = *dbDep.Image
			}
			
			// If no image is configured, check if deployment needs to be built
			if image == "" {
				// Check if deployment has a repository URL or build strategy that requires building
				if dbDep.RepositoryURL != nil && *dbDep.RepositoryURL != "" {
					log.Printf("[attemptAutomaticRedeployment] Deployment %s has no image but has repository URL - deployment needs to be built first", deploymentID)
					return fmt.Errorf("deployment image not found - deployment needs to be built. Please trigger a deployment build first")
				}
				// If no repository URL, this might be a deployment that uses an external image
				// In that case, we can't automatically redeploy
				log.Printf("[attemptAutomaticRedeployment] Deployment %s has no image configured", deploymentID)
				return fmt.Errorf("deployment has no image configured")
			}
			
			// Get port from routing configuration if available, otherwise use deployment port
			port := 8080
			if dbDep.Port != nil {
				port = int(*dbDep.Port)
			}
			
			// Check routing configuration for target port (takes precedence)
			routings, err := database.GetDeploymentRoutings(deploymentID)
			if err == nil && len(routings) > 0 {
				// Track if we found a routing rule
				foundRouting := false
				// Find routing rule for "default" service (or first one if no service name specified)
				for _, routing := range routings {
					if routing.ServiceName == "" || routing.ServiceName == "default" {
						port = routing.TargetPort
						log.Printf("[attemptAutomaticRedeployment] Using target port %d from routing configuration (default service) for deployment %s", port, deploymentID)
						foundRouting = true
						break
					}
				}
				// If no default service routing found, use first routing's target port
				if !foundRouting {
					port = routings[0].TargetPort
					log.Printf("[attemptAutomaticRedeployment] Using target port %d from first routing rule for deployment %s", port, deploymentID)
				}
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
				log.Printf("[attemptAutomaticRedeployment] Failed to create containers for deployment %s: %v", deploymentID, err)
				// Check if error is about missing image
				if strings.Contains(err.Error(), "No such image") || strings.Contains(err.Error(), "image not found") {
					// Check if there's a latest successful build - if so, trigger automatic rebuild
					latestBuild, buildErr := s.buildHistoryRepo.GetLatestSuccessfulBuild(ctx, deploymentID)
					if buildErr == nil && latestBuild != nil {
						log.Printf("[attemptAutomaticRedeployment] Image %s not found but found successful build #%d, triggering automatic rebuild", image, latestBuild.BuildNumber)
						
						// Trigger rebuild asynchronously
						go func() {
							// Create a system context with admin user to bypass permission checks
							systemCtx := s.createSystemContext()
							
							// Create a request with organization ID from deployment
							req := connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{
								DeploymentId:   deploymentID,
								OrganizationId: dbDep.OrganizationID,
							})
							
							// Trigger the deployment (this will handle the build asynchronously)
							_, triggerErr := s.TriggerDeployment(systemCtx, req)
							if triggerErr != nil {
								log.Printf("[attemptAutomaticRedeployment] Failed to trigger automatic rebuild for deployment %s: %v", deploymentID, triggerErr)
							} else {
								log.Printf("[attemptAutomaticRedeployment] Successfully triggered automatic rebuild for deployment %s", deploymentID)
							}
						}()
						
						return fmt.Errorf("image %s not found - automatically triggered rebuild based on latest successful build #%d. Please wait for the build to complete", image, latestBuild.BuildNumber)
					}
					
					// No successful build found
					return fmt.Errorf("image %s not found - deployment needs to be built. Please trigger a deployment build first", image)
				}
				return fmt.Errorf("failed to create containers: %w", err)
			}
			log.Printf("[attemptAutomaticRedeployment] Successfully created containers for deployment %s", deploymentID)

			// Start the containers
			if err := s.manager.StartDeployment(ctx, deploymentID); err != nil {
				log.Printf("[attemptAutomaticRedeployment] Failed to start containers for deployment %s: %v", deploymentID, err)
				return fmt.Errorf("failed to start containers: %w", err)
			}
		} else {
			return fmt.Errorf("deployment has no containers and orchestrator is not available")
		}
	}

	// Wait a bit for containers to be created and started
	time.Sleep(2 * time.Second)

	return nil
}

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

	// If still no containers found, check if deployment should be running and attempt automatic redeployment
	if len(locations) == 0 {
		// Attempt automatic redeployment if deployment should be running
		if err := s.attemptAutomaticRedeployment(ctx, deploymentID); err != nil {
			log.Printf("[findContainerForDeployment] Automatic redeployment failed for deployment %s: %v", deploymentID, err)
			// Continue to check again after redeployment attempt
		}

		// Try to get locations again after redeployment attempt
		locations, err = database.ValidateAndRefreshLocations(deploymentID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate locations after redeployment: %w", err)
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("no containers for deployment")
		}
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

// getDeploymentHealthStatus gets the health status from Docker containers
// Returns the health status: "none", "starting", "healthy", "unhealthy", or empty string if no health check
func (s *Service) getDeploymentHealthStatus(ctx context.Context, deploymentID string) (string, error) {
	dcli, err := docker.New()
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}
	defer dcli.Close()

	// Get all locations for this deployment
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		// Fallback to validate and refresh if GetAllDeploymentLocations fails
		locations, err = database.ValidateAndRefreshLocations(deploymentID)
		if err != nil {
			return "", fmt.Errorf("failed to get locations: %w", err)
		}
	}

	if len(locations) == 0 {
		return "", nil // No containers, no health status
	}

	// Check health status of all containers
	// If any container is unhealthy, deployment is unhealthy
	// If all containers are healthy, deployment is healthy
	// If any container is starting, deployment is starting
	// If no health check, return empty string
	hasHealthCheck := false
	hasUnhealthy := false
	hasStarting := false
	allHealthy := true

	for _, loc := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			// Container might not exist, skip it
			continue
		}

		// Check health status from Docker
		if containerInfo.State != nil && containerInfo.State.Health != nil {
			hasHealthCheck = true
			healthStatus := containerInfo.State.Health.Status
			
			switch healthStatus {
			case "unhealthy":
				hasUnhealthy = true
				allHealthy = false
			case "starting":
				hasStarting = true
				allHealthy = false
			case "healthy":
				// Container is healthy, continue checking others
			case "none":
				// No health check configured for this container
			}
		}
	}

	// Determine overall health status
	if !hasHealthCheck {
		return "", nil // No health checks configured
	}
	
	if hasUnhealthy {
		return "unhealthy", nil
	}
	if hasStarting {
		return "starting", nil
	}
	if allHealthy {
		return "healthy", nil
	}
	
	return "unknown", nil
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

	// Check if we need to forward to another node
	if shouldForward, targetNodeID := s.shouldForwardToNode(loc); shouldForward {
		log.Printf("[StreamContainerLogs] Container %s is on node %s, forwarding request", loc.ContainerID[:12], targetNodeID)
		return s.forwardStreamContainerLogs(ctx, req, stream, targetNodeID)
	}

	tail := req.Msg.GetTail()
	if tail <= 0 {
		tail = 200
	}

	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", tail), true, nil, nil) // follow=true for streaming
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

// forwardStreamContainerLogs forwards a streaming container log request to another node
func (s *Service) forwardStreamContainerLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamContainerLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], targetNodeID string) error {
	if s.forwarder == nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("node forwarder not available"))
	}

	// Serialize request
	reqBody, err := json.Marshal(req.Msg)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal request: %w", err))
	}

	// Forward the request
	path := "/obiente.cloud.deployments.v1.DeploymentService/StreamContainerLogs"
	headers := map[string]string{
		"Authorization": req.Header().Get("Authorization"),
	}

	resp, err := s.forwarder.ForwardConnectRPCRequest(ctx, targetNodeID, "POST", path, bytes.NewReader(reqBody), headers)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to forward request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		code := connect.CodeInternal
		if resp.StatusCode == http.StatusUnauthorized {
			code = connect.CodeUnauthenticated
		} else if resp.StatusCode == http.StatusForbidden {
			code = connect.CodePermissionDenied
		} else if resp.StatusCode == http.StatusNotFound {
			code = connect.CodeNotFound
		}
		return connect.NewError(code, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes)))
	}

	// Stream the response
	decoder := json.NewDecoder(resp.Body)
	for {
		var logLine deploymentsv1.DeploymentLogLine
		if err := decoder.Decode(&logLine); err != nil {
			if err == io.EOF {
				return nil
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode response: %w", err))
		}

		if err := stream.Send(&logLine); err != nil {
			return err
		}
	}
}

// forwardUnaryRequest forwards a unary ConnectRPC request to another node
func (s *Service) forwardUnaryRequest(ctx context.Context, reqBody []byte, targetNodeID string, path string, headers map[string]string, respType interface{}) ([]byte, error) {
	if s.forwarder == nil {
		return nil, fmt.Errorf("node forwarder not available")
	}

	// Forward the request
	resp, err := s.forwarder.ForwardConnectRPCRequest(ctx, targetNodeID, "POST", path, bytes.NewReader(reqBody), headers)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return bodyBytes, nil
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

	// Check if we need to forward to another node
	if shouldForward, targetNodeID := s.shouldForwardToNode(loc); shouldForward {
		log.Printf("[StartContainer] Container %s is on node %s, forwarding request", loc.ContainerID[:12], targetNodeID)
		reqBody, _ := json.Marshal(req.Msg)
		headers := map[string]string{"Authorization": req.Header().Get("Authorization")}
		bodyBytes, err := s.forwardUnaryRequest(ctx, reqBody, targetNodeID, "/obiente.cloud.deployments.v1.DeploymentService/StartContainer", headers, &deploymentsv1.StartContainerResponse{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to forward request: %w", err))
		}
		var response deploymentsv1.StartContainerResponse
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode response: %w", err))
		}
		return connect.NewResponse(&response), nil
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

	// Check if we need to forward to another node
	if shouldForward, targetNodeID := s.shouldForwardToNode(loc); shouldForward {
		log.Printf("[StopContainer] Container %s is on node %s, forwarding request", loc.ContainerID[:12], targetNodeID)
		reqBody, _ := json.Marshal(req.Msg)
		headers := map[string]string{"Authorization": req.Header().Get("Authorization")}
		bodyBytes, err := s.forwardUnaryRequest(ctx, reqBody, targetNodeID, "/obiente.cloud.deployments.v1.DeploymentService/StopContainer", headers, &deploymentsv1.StopContainerResponse{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to forward request: %w", err))
		}
		var response deploymentsv1.StopContainerResponse
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode response: %w", err))
		}
		return connect.NewResponse(&response), nil
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

	// Check if we need to forward to another node
	if shouldForward, targetNodeID := s.shouldForwardToNode(loc); shouldForward {
		log.Printf("[RestartContainer] Container %s is on node %s, forwarding request", loc.ContainerID[:12], targetNodeID)
		reqBody, _ := json.Marshal(req.Msg)
		headers := map[string]string{"Authorization": req.Header().Get("Authorization")}
		bodyBytes, err := s.forwardUnaryRequest(ctx, reqBody, targetNodeID, "/obiente.cloud.deployments.v1.DeploymentService/RestartContainer", headers, &deploymentsv1.RestartContainerResponse{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to forward request: %w", err))
		}
		var response deploymentsv1.RestartContainerResponse
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode response: %w", err))
		}
		return connect.NewResponse(&response), nil
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
