package deployments

import (
	"context"
	"fmt"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"api/docker"

	"connectrpc.com/connect"
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

// StreamDeploymentLogs streams deployment logs from containers
func (s *Service) StreamDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.StreamDeploymentLogsRequest], stream *connect.ServerStream[deploymentsv1.DeploymentLogLine]) error {
	// Ensure user is authenticated for streaming RPCs (interceptor may not run)
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	// Find a container for this deployment and stream logs
	// Validate and refresh locations to ensure we have valid container IDs
	locations, err := database.ValidateAndRefreshLocations(deploymentID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to validate locations: %w", err))
	}
	if len(locations) == 0 {
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
