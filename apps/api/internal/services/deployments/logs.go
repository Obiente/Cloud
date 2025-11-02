package deployments

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
	"github.com/moby/moby/api/types/events"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetDeploymentLogs retrieves a fixed number of deployment logs
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

// StreamDeploymentLogs streams deployment logs from containers and Docker events
func (s *Service) StreamDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine]) error {
	// Ensure user is authenticated for streaming RPCs (interceptor may not run)
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	
	dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()
	
	// Find container by container_id or service_name, or use first if neither specified
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}
	
	// Get all containers for this deployment to filter events
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment locations: %w", err))
	}
	
	// Build set of container IDs for this deployment
	containerIDs := make(map[string]bool)
	for _, location := range locations {
		containerIDs[location.ContainerID] = true
	}
	
	// Also get images used by these containers to track image pull events
	imageNames := make(map[string]bool)
	for _, location := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, location.ContainerID)
		if err == nil && containerInfo.Config != nil {
			if containerInfo.Config.Image != "" {
				imageNames[containerInfo.Config.Image] = true
			}
		}
	}
	
	tail := req.Msg.GetTail()
	if tail <= 0 {
		tail = 200
	}
	
	// Check if container is running to determine if we should follow logs
	// For stopped containers, we'll get historical logs but won't follow
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	isRunning := false
	if err == nil {
		isRunning = containerInfo.State.Running
	}
	
	// For stopped containers, use follow=false to get historical logs
	// For running containers, use follow=true to stream new logs
	follow := isRunning
	
	// Start container logs stream
	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", tail), follow)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("logs: %w", err))
	}
	defer reader.Close()
	
	// Start Docker events stream filtered for this deployment
	eventFilters := map[string][]string{
		"type": {"container", "image"},
		"label": {fmt.Sprintf("com.obiente.deployment_id=%s", deploymentID)},
	}
	eventChan, eventErrChan, cleanup, err := dcli.Events(ctx, eventFilters)
	if err != nil {
		// If events fail, continue with just logs
		log.Printf("[StreamDeploymentLogs] Failed to start event stream: %v", err)
		eventChan = nil
	}
	if cleanup != nil {
		defer cleanup()
	}
	
	// Create a channel to send log lines
	logChan := make(chan *deploymentsv1.DeploymentLogLine, 100)
	var wg sync.WaitGroup
	
	// Stream container logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(logChan)
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				// Docker logs can contain invalid UTF-8 sequences (binary data).
				// Sanitize to valid UTF-8 before sending to protobuf, which requires valid UTF-8.
				// Invalid sequences are replaced with the Unicode replacement character (U+FFFD).
				sanitizedLine := strings.ToValidUTF8(string(buf[:n]), "")
				line := &deploymentsv1.DeploymentLogLine{
					DeploymentId: deploymentID,
					Line:         sanitizedLine,
					Timestamp:    timestamppb.Now(),
				}
				select {
				case logChan <- line:
				case <-ctx.Done():
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()
	
	// Stream Docker events
	if eventChan != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-eventChan:
					if !ok {
						return
					}
					
					// Format and send relevant events
					eventLine := s.formatDockerEvent(deploymentID, event, containerIDs, imageNames)
					if eventLine != nil {
						select {
						case logChan <- eventLine:
						case <-ctx.Done():
							return
						}
					}
				case err := <-eventErrChan:
					if err != nil {
						log.Printf("[StreamDeploymentLogs] Event stream error: %v", err)
					}
					return
				}
			}
		}()
	}
	
	// Send merged logs and events to client
	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-logChan:
			if !ok {
				// Wait for goroutines to finish
				wg.Wait()
				return nil
			}
			if sendErr := stream.Send(line); sendErr != nil {
				return sendErr
			}
		}
	}
}

// formatDockerEvent formats a Docker event into a log line
func (s *Service) formatDockerEvent(deploymentID string, event events.Message, containerIDs map[string]bool, imageNames map[string]bool) *deploymentsv1.DeploymentLogLine {
	var line string
	var isRelevant bool
	
	switch event.Type {
	case "container":
		// Check if this container belongs to our deployment
		containerID := event.Actor.ID
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		
		// Check by container ID
		isOurContainer := false
		for id := range containerIDs {
			if id == event.Actor.ID || strings.HasPrefix(id, event.Actor.ID) || strings.HasPrefix(event.Actor.ID, id) {
				isOurContainer = true
				break
			}
		}
		
		// Also check by deployment label
		if len(event.Actor.Attributes) > 0 {
			if event.Actor.Attributes["com.obiente.deployment_id"] == deploymentID {
				isOurContainer = true
			}
		}
		
		if !isOurContainer {
			return nil
		}
		
		// Format container events
		containerName := containerID
		if len(event.Actor.Attributes) > 0 {
			if serviceName := event.Actor.Attributes["com.obiente.service_name"]; serviceName != "" {
				containerName = serviceName
			} else if serviceName := event.Actor.Attributes["com.docker.compose.service"]; serviceName != "" {
				containerName = serviceName
			} else if name := event.Actor.Attributes["name"]; name != "" {
				containerName = name
			}
		}
		
		switch event.Action {
		case "create":
			line = fmt.Sprintf("[docker] Creating container %s", containerName)
		case "start":
			line = fmt.Sprintf("[docker] Starting container %s", containerName)
		case "stop":
			line = fmt.Sprintf("[docker] Stopping container %s", containerName)
		case "die", "kill":
			line = fmt.Sprintf("[docker] Container %s stopped", containerName)
		case "restart":
			line = fmt.Sprintf("[docker] Restarting container %s", containerName)
		case "pause":
			line = fmt.Sprintf("[docker] Pausing container %s", containerName)
		case "unpause":
			line = fmt.Sprintf("[docker] Resuming container %s", containerName)
		case "health_status":
			if len(event.Actor.Attributes) > 0 {
				status := event.Actor.Attributes["status"]
				line = fmt.Sprintf("[docker] Container %s health status: %s", containerName, status)
			} else {
				return nil
			}
		default:
			// Ignore other container events
			return nil
		}
		isRelevant = true
		
	case "image":
		// Check if this image is used by our deployment
		imageName := event.Actor.ID
		if len(event.Actor.Attributes) > 0 {
			if name := event.Actor.Attributes["name"]; name != "" {
				imageName = name
			}
		}
		
		// Check if this image is used by any of our containers
		isOurImage := false
		for img := range imageNames {
			if img == imageName || strings.HasPrefix(imageName, img) || strings.HasPrefix(img, imageName) {
				isOurImage = true
				break
			}
		}
		
		// Also check by matching image name patterns
		if !isOurImage && imageName != "" {
			for img := range imageNames {
				// Match if image names share the same base (e.g., "myapp:latest" and "myapp:v1")
				imgBase := strings.Split(img, ":")[0]
				eventBase := strings.Split(imageName, ":")[0]
				if imgBase != "" && imgBase == eventBase {
					isOurImage = true
					break
				}
			}
		}
		
		if !isOurImage {
			return nil
		}
		
		// Format image events
		displayName := imageName
		if len(displayName) > 60 {
			displayName = displayName[:57] + "..."
		}
		
		switch event.Action {
		case "pull":
			line = fmt.Sprintf("[docker] Pulling image %s", displayName)
		case "tag":
			if len(event.Actor.Attributes) > 0 {
				if target := event.Actor.Attributes["target"]; target != "" {
					line = fmt.Sprintf("[docker] Tagging image %s as %s", displayName, target)
				} else {
					line = fmt.Sprintf("[docker] Tagging image %s", displayName)
				}
			} else {
				line = fmt.Sprintf("[docker] Tagging image %s", displayName)
			}
		case "untag":
			line = fmt.Sprintf("[docker] Untagging image %s", displayName)
		case "delete":
			line = fmt.Sprintf("[docker] Deleting image %s", displayName)
		default:
			// Ignore other image events
			return nil
		}
		isRelevant = true
	}
	
	if !isRelevant {
		return nil
	}
	
	// Create timestamp from event time
	var timestamp time.Time
	if event.Time != 0 {
		timestamp = time.Unix(event.Time, event.TimeNano)
	} else {
		timestamp = time.Now()
	}
	
	return &deploymentsv1.DeploymentLogLine{
		DeploymentId: deploymentID,
		Line:         line,
		Timestamp:    timestamppb.New(timestamp),
		Stderr:       false, // Events are informational, not errors
	}
}

// StreamBuildLogs streams build logs for a deployment
func (s *Service) StreamBuildLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamBuildLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine]) error {
	// Ensure user is authenticated for streaming RPCs (interceptor may not run)
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get or create build log streamer
	buildStreamer := GetBuildLogStreamer(deploymentID)

	// Subscribe to build logs
	logChan := buildStreamer.Subscribe()
	defer func() {
		// Unsubscribe when done
		buildStreamer.Unsubscribe(logChan)
	}()

	// Send buffered logs first
	bufferedLogs := buildStreamer.GetLogs()
	for _, logLine := range bufferedLogs {
		if err := stream.Send(logLine); err != nil {
			return err
		}
	}

	// Stream new logs to client
	for {
		select {
		case <-ctx.Done():
			return nil // Return nil on context cancel for proper stream close
		case logLine, ok := <-logChan:
			if !ok {
				// Channel closed, stream ended normally
				return nil
			}
			if err := stream.Send(logLine); err != nil {
				// Connection closed or error, stop streaming
				return nil
			}
		}
	}
}
