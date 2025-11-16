package deployments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

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

	// Check if we need to forward to another node
	if shouldForward, targetNodeID := s.shouldForwardToNode(loc); shouldForward {
		log.Printf("[StreamDeploymentLogs] Container %s is on node %s, forwarding request", loc.ContainerID[:12], targetNodeID)
		return s.forwardStreamDeploymentLogs(ctx, req, stream, targetNodeID)
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
	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", tail), follow, nil, nil)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("logs: %w", err))
	}
	defer reader.Close()
	
	// Start Docker events stream filtered for this deployment
	eventFilters := map[string][]string{
		"type": {"container", "image"},
		"label": {fmt.Sprintf("cloud.obiente.deployment_id=%s", deploymentID)},
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
	
	// Close channel after all goroutines finish
	go func() {
		wg.Wait()
		close(logChan)
	}()
	
	// Helper function to safely send to channel
	safeSend := func(line *deploymentsv1.DeploymentLogLine) bool {
		select {
		case logChan <- line:
			return true
		case <-ctx.Done():
			return false
		}
	}
	
	// Stream container logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				// Docker logs can contain invalid UTF-8 sequences (binary data).
				// Sanitize to valid UTF-8 before sending to protobuf, which requires valid UTF-8.
				// Invalid sequences are replaced with the Unicode replacement character (U+FFFD).
				sanitizedLine := strings.ToValidUTF8(string(buf[:n]), "")
				// Detect log level from content (container logs can also have structured log levels)
				logLevel := detectLogLevelFromContent(sanitizedLine, false)
				line := &deploymentsv1.DeploymentLogLine{
					DeploymentId: deploymentID,
					Line:         sanitizedLine,
					Timestamp:    timestamppb.Now(),
					Stderr:       false, // Container logs don't distinguish stderr/stdout in this context
					LogLevel:     logLevel,
				}
				if !safeSend(line) {
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
						if !safeSend(eventLine) {
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
				// Channel closed, all goroutines finished
				return nil
			}
			if sendErr := stream.Send(line); sendErr != nil {
				return sendErr
			}
		}
	}
}

// detectLogLevelFromContent detects log level from log line content
// This is a shared function for both build logs and container logs
func detectLogLevelFromContent(line string, isStderr bool) commonv1.LogLevel {
	lineLower := strings.ToLower(strings.TrimSpace(line))
	
	// Check for explicit log level markers (case-insensitive)
	if strings.Contains(lineLower, "[error]") || strings.Contains(lineLower, "error:") ||
		strings.Contains(lineLower, "fatal:") || strings.Contains(lineLower, "failed") ||
		strings.HasPrefix(lineLower, "error") || strings.Contains(lineLower, " ❌ ") {
		return commonv1.LogLevel_LOG_LEVEL_ERROR
	}
	
	if strings.Contains(lineLower, "[warn]") || strings.Contains(lineLower, "[warning]") ||
		strings.Contains(lineLower, "warning:") || strings.Contains(lineLower, "⚠️") ||
		strings.HasPrefix(lineLower, "warn") {
		return commonv1.LogLevel_LOG_LEVEL_WARN
	}
	
	if strings.Contains(lineLower, "[debug]") || strings.Contains(lineLower, "[trace]") ||
		strings.HasPrefix(lineLower, "debug") || strings.HasPrefix(lineLower, "trace") {
		return commonv1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Nixpacks/Railpack specific patterns - these are INFO even if on stderr
	if strings.Contains(lineLower, "nixpacks") || strings.Contains(lineLower, "railpack") ||
		strings.Contains(lineLower, "building") || strings.Contains(lineLower, "setup") ||
		strings.Contains(lineLower, "install") || strings.Contains(lineLower, "build") ||
		strings.Contains(lineLower, "start") || strings.Contains(lineLower, "transferring") ||
		strings.Contains(lineLower, "loading") || strings.Contains(lineLower, "resolving") ||
		strings.Contains(lineLower, "[internal]") || strings.Contains(lineLower, "[stage-") ||
		strings.Contains(lineLower, "sha256:") || strings.Contains(lineLower, "done") ||
		strings.Contains(lineLower, "dockerfile:") || strings.Contains(lineLower, "context:") ||
		strings.Contains(lineLower, "metadata") {
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Docker build output patterns - usually INFO
	if strings.Contains(lineLower, "[") && strings.Contains(lineLower, "]") &&
		(strings.Contains(lineLower, "step") || strings.Contains(lineLower, "from") ||
		strings.Contains(lineLower, "running") || strings.Contains(lineLower, "executing")) {
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Default: if stderr is true and no pattern matched, it might be an error
	if isStderr {
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "fail") {
			return commonv1.LogLevel_LOG_LEVEL_WARN
		}
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Default to INFO for stdout
	return commonv1.LogLevel_LOG_LEVEL_INFO
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
			if event.Actor.Attributes["cloud.obiente.deployment_id"] == deploymentID {
				isOurContainer = true
			}
		}
		
		if !isOurContainer {
			return nil
		}
		
		// Format container events
		containerName := containerID
		if len(event.Actor.Attributes) > 0 {
			if serviceName := event.Actor.Attributes["cloud.obiente.service_name"]; serviceName != "" {
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

// forwardStreamDeploymentLogs forwards a streaming log request to another node
func (s *Service) forwardStreamDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], targetNodeID string) error {
	if s.forwarder == nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("node forwarder not available"))
	}

	// Serialize request
	reqBody, err := json.Marshal(req.Msg)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal request: %w", err))
	}

	// Forward the request
	path := "/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentLogs"
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

	// Send buffered logs first
	bufferedLogs := buildStreamer.GetLogs()
	
	// If no buffered logs, try to load from database (for builds happening on other nodes)
	if len(bufferedLogs) == 0 {
		// Get latest active build for this deployment
		builds, _, err := s.buildHistoryRepo.ListBuilds(ctx, deploymentID, orgID, 1, 0)
		if err == nil && len(builds) > 0 {
			build := builds[0]
			// Check if build is currently building (status 2 = BUILD_BUILDING)
			if build.Status == 2 {
				// Load logs from database for this build
				buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
				dbLogs, _, err := buildLogsRepo.GetBuildLogs(ctx, build.ID, 10000, 0)
				if err == nil {
					// Convert database logs to proto format
					for _, logEntry := range dbLogs {
						logLine := &deploymentsv1.DeploymentLogLine{
							DeploymentId: deploymentID,
							Line:         logEntry.Line,
							Timestamp:    timestamppb.New(logEntry.Timestamp),
							Stderr:       logEntry.Stderr,
							LogLevel:     commonv1.LogLevel_LOG_LEVEL_INFO,
						}
						bufferedLogs = append(bufferedLogs, logLine)
					}
				}
			}
		}
	}

	// Send buffered logs
	for _, logLine := range bufferedLogs {
		// Check context before sending
		if ctx.Err() != nil {
			return nil
		}
		if err := stream.Send(logLine); err != nil {
			// If context is cancelled, return nil for proper closure
			if ctx.Err() != nil {
				return nil
			}
			// For other errors, still return nil to ensure proper stream closure
			return nil
		}
	}

	// Subscribe to build logs for real-time updates
	logChan := buildStreamer.Subscribe()
	defer func() {
		// Unsubscribe when done
		buildStreamer.Unsubscribe(logChan)
	}()

	// Stream new logs to client
	for {
		select {
		case <-ctx.Done():
			// Context cancelled (client disconnected or timeout) - return nil for proper stream close
			return nil
		case logLine, ok := <-logChan:
			if !ok {
				// Channel closed, stream ended normally - return nil for proper stream close
				return nil
			}
			if err := stream.Send(logLine); err != nil {
				// Check if context was cancelled (client disconnected)
				// In that case, return nil for proper stream closure
				if ctx.Err() != nil {
					return nil
				}
				// For other errors from Send(), return nil as well to ensure proper stream closure
				// This handles cases where the client disconnected but context isn't cancelled yet
				return nil
			}
		}
	}
}
