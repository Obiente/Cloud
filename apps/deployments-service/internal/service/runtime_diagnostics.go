package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const deploymentDiagnosticsTailLines = 120

type deploymentDiagnosticSink interface {
	Write([]byte) (int, error)
	WriteStderr([]byte) (int, error)
}

func (s *Service) captureDeploymentFailureDiagnostics(ctx context.Context, deploymentID, source, failureReason string, sink deploymentDiagnosticSink) {
	if s.runtimeLogsRepo == nil && sink == nil {
		return
	}

	diagnosticsCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	entries := make([]database.DeploymentRuntimeLog, 0, 256)
	appendEntry := func(serviceName, containerID, line string, stderr bool, level commonv1.LogLevel, timestamp time.Time) {
		line = strings.TrimSpace(strings.ToValidUTF8(line, ""))
		if line == "" {
			return
		}
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		if sink != nil {
			payload := []byte(line + "\n")
			if stderr || level == commonv1.LogLevel_LOG_LEVEL_ERROR {
				_, _ = sink.WriteStderr(payload)
			} else {
				_, _ = sink.Write(payload)
			}
		}

		entries = append(entries, database.DeploymentRuntimeLog{
			DeploymentID: deploymentID,
			ServiceName:  serviceName,
			ContainerID:  containerID,
			Source:       source,
			Line:         line,
			Timestamp:    timestamp,
			Stderr:       stderr,
			LogLevel:     int32(level),
		})
	}

	if failureReason != "" {
		appendEntry("", "", fmt.Sprintf("[runtime] Failure reason: %s", failureReason), true, commonv1.LogLevel_LOG_LEVEL_ERROR, time.Now())
		appendEntry("", "", "[runtime] Capturing container and service diagnostics for this run...", false, commonv1.LogLevel_LOG_LEVEL_INFO, time.Now())
	}

	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil || len(locations) == 0 {
		if refreshed, refreshErr := database.ValidateAndRefreshLocations(deploymentID); refreshErr == nil && len(refreshed) > 0 {
			locations = refreshed
		}
	}

	dcli, err := docker.New()
	if err != nil {
		appendEntry("", "", fmt.Sprintf("[runtime] Unable to inspect deployment containers: %v", err), true, commonv1.LogLevel_LOG_LEVEL_WARN, time.Now())
		s.flushDiagnosticEntries(diagnosticsCtx, entries)
		return
	}
	defer dcli.Close()

	capturedLogLines := 0
	for _, location := range locations {
		if location.ContainerID == "" {
			continue
		}

		containerInfo, inspectErr := dcli.ContainerInspect(diagnosticsCtx, location.ContainerID)
		serviceName := inferDeploymentServiceName(location.ContainerID, nil)
		if inspectErr != nil {
			appendEntry(serviceName, location.ContainerID, fmt.Sprintf("[runtime] Failed to inspect container %s: %v", shortenContainerID(location.ContainerID), inspectErr), true, commonv1.LogLevel_LOG_LEVEL_WARN, time.Now())
			continue
		}

		var labels map[string]string
		if containerInfo.Config != nil {
			labels = containerInfo.Config.Labels
		}
		serviceName = inferDeploymentServiceName(location.ContainerID, labels)
		appendEntry(serviceName, location.ContainerID, formatContainerStateSummary(serviceName, location.ContainerID, containerInfo), !containerInfo.State.Running, logLevelForContainerState(containerInfo), time.Now())

		snapshotLines, snapshotErr := captureContainerLogSnapshot(diagnosticsCtx, dcli, deploymentID, serviceName, location.ContainerID, deploymentDiagnosticsTailLines)
		if snapshotErr != nil {
			appendEntry(serviceName, location.ContainerID, fmt.Sprintf("[runtime] Failed to capture logs for %s: %v", serviceName, snapshotErr), true, commonv1.LogLevel_LOG_LEVEL_WARN, time.Now())
			continue
		}
		if len(snapshotLines) == 0 {
			continue
		}

		appendEntry(serviceName, location.ContainerID, fmt.Sprintf("[runtime] Last %d log line(s) from service %s:", len(snapshotLines), serviceName), false, commonv1.LogLevel_LOG_LEVEL_INFO, time.Now())
		for _, snapshotLine := range snapshotLines {
			prefixed := fmt.Sprintf("[runtime][%s] %s", serviceName, snapshotLine.Line)
			ts := time.Now()
			if snapshotLine.Timestamp != nil {
				ts = snapshotLine.Timestamp.AsTime()
			}
			appendEntry(serviceName, location.ContainerID, prefixed, snapshotLine.Stderr, snapshotLine.LogLevel, ts)
			capturedLogLines++
		}
	}

	if taskLines, taskErr := captureSwarmTaskDiagnostics(diagnosticsCtx, deploymentID); taskErr == nil && len(taskLines) > 0 {
		appendEntry("", "", "[runtime] Swarm task status snapshot:", false, commonv1.LogLevel_LOG_LEVEL_INFO, time.Now())
		for _, taskLine := range taskLines {
			appendEntry(taskLine.ServiceName, "", taskLine.Line, taskLine.Stderr, taskLine.LogLevel, taskLine.Timestamp)
			capturedLogLines++
		}
	} else if taskErr != nil {
		appendEntry("", "", fmt.Sprintf("[runtime] Failed to capture Swarm task diagnostics: %v", taskErr), true, commonv1.LogLevel_LOG_LEVEL_WARN, time.Now())
	}

	if capturedLogLines == 0 {
		snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(diagnosticsCtx, deploymentID, "", deploymentDiagnosticsTailLines)
		if snapshotErr == nil && len(snapshotLogs) > 0 {
			appendEntry("", "", "[runtime] Captured service snapshot from Docker service logs:", false, commonv1.LogLevel_LOG_LEVEL_INFO, time.Now())
			for _, line := range snapshotLogs {
				ts := time.Now()
				if line.Timestamp != nil {
					ts = line.Timestamp.AsTime()
				}
				appendEntry("", "", fmt.Sprintf("[runtime] %s", line.Line), line.Stderr, line.LogLevel, ts)
				capturedLogLines++
			}
		}
	}

	if capturedLogLines == 0 && s.manager != nil {
		if liveLogs, liveErr := s.manager.GetDeploymentLogs(diagnosticsCtx, deploymentID, fmt.Sprintf("%d", deploymentDiagnosticsTailLines)); liveErr == nil && strings.TrimSpace(liveLogs) != "" {
			appendEntry("", "", "[runtime] Captured deployment log snapshot from the orchestrator:", false, commonv1.LogLevel_LOG_LEVEL_INFO, time.Now())
			for _, line := range parseDockerServiceLogOutput(deploymentID, liveLogs) {
				ts := time.Now()
				if line.Timestamp != nil {
					ts = line.Timestamp.AsTime()
				}
				appendEntry("", "", fmt.Sprintf("[runtime] %s", line.Line), line.Stderr, line.LogLevel, ts)
				capturedLogLines++
			}
		}
	}

	if capturedLogLines == 0 {
		appendEntry("", "", "[runtime] No additional container or service log output was available for this failed run.", true, commonv1.LogLevel_LOG_LEVEL_WARN, time.Now())
	}

	s.flushDiagnosticEntries(diagnosticsCtx, entries)
}

func (s *Service) flushDiagnosticEntries(ctx context.Context, entries []database.DeploymentRuntimeLog) {
	if s.runtimeLogsRepo == nil || len(entries) == 0 {
		return
	}
	if err := s.runtimeLogsRepo.AddLogsBatch(ctx, entries); err != nil {
		logger.Warn("[RuntimeDiagnostics] Failed to persist diagnostic logs: %v", err)
	}
}

func captureContainerLogSnapshot(ctx context.Context, dcli *docker.Client, deploymentID, serviceName, containerID string, tail int) ([]*deploymentsv1.DeploymentLogLine, error) {
	reader, err := dcli.ContainerLogs(ctx, containerID, fmt.Sprintf("%d", tail), false, nil, nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}

	now := time.Now()
	return splitRuntimeLogSnapshot(deploymentID, string(payload), false, now), nil
}

func splitRuntimeLogSnapshot(deploymentID, output string, stderr bool, timestamp time.Time) []*deploymentsv1.DeploymentLogLine {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	rawLines := strings.Split(output, "\n")
	lines := make([]*deploymentsv1.DeploymentLogLine, 0, len(rawLines))

	for _, rawLine := range rawLines {
		line := strings.TrimSpace(strings.ToValidUTF8(rawLine, ""))
		if line == "" {
			continue
		}
		lines = append(lines, &deploymentsv1.DeploymentLogLine{
			DeploymentId: deploymentID,
			Line:         line,
			Timestamp:    timestamppb.New(timestamp),
			Stderr:       stderr,
			LogLevel:     detectLogLevelFromContent(line, stderr),
		})
	}

	return lines
}

func inferDeploymentServiceName(containerID string, labels map[string]string) string {
	if labels != nil {
		if serviceName := labels["cloud.obiente.service_name"]; serviceName != "" {
			return serviceName
		}
		if serviceName := labels["com.docker.compose.service"]; serviceName != "" {
			return serviceName
		}
		if serviceName := labels["com.docker.swarm.service.name"]; serviceName != "" {
			return serviceName
		}
		if name := labels["name"]; name != "" {
			return name
		}
	}

	if containerID == "" {
		return "unknown"
	}
	return shortenContainerID(containerID)
}

type swarmTaskDiagnosticLine struct {
	ServiceName string
	Line        string
	Timestamp   time.Time
	Stderr      bool
	LogLevel    commonv1.LogLevel
}

type dockerServicePSRow struct {
	ID           string `json:"ID"`
	Name         string `json:"Name"`
	Image        string `json:"Image"`
	Node         string `json:"Node"`
	DesiredState string `json:"DesiredState"`
	CurrentState string `json:"CurrentState"`
	Error        string `json:"Error"`
	Ports        string `json:"Ports"`
}

func captureSwarmTaskDiagnostics(ctx context.Context, deploymentID string) ([]swarmTaskDiagnosticLine, error) {
	stackName := fmt.Sprintf("deploy-%s", deploymentID)
	serviceNames := make([]string, 0)
	seenServices := make(map[string]struct{})
	addServices := func(output []byte) {
		for _, rawLine := range strings.Split(string(output), "\n") {
			serviceName := strings.TrimSpace(rawLine)
			if serviceName == "" {
				continue
			}
			if _, exists := seenServices[serviceName]; exists {
				continue
			}
			seenServices[serviceName] = struct{}{}
			serviceNames = append(serviceNames, serviceName)
		}
	}

	stackListCmd := exec.CommandContext(ctx, "docker", "service", "ls", "--filter", fmt.Sprintf("label=com.docker.stack.namespace=%s", stackName), "--format", "{{.Name}}")
	stackListOutput, stackErr := stackListCmd.Output()
	if stackErr == nil {
		addServices(stackListOutput)
	}

	deploymentListCmd := exec.CommandContext(ctx, "docker", "service", "ls", "--filter", fmt.Sprintf("label=cloud.obiente.deployment_id=%s", deploymentID), "--format", "{{.Name}}")
	deploymentListOutput, deploymentErr := deploymentListCmd.Output()
	if deploymentErr == nil {
		addServices(deploymentListOutput)
	}

	if len(serviceNames) == 0 {
		if stackErr != nil {
			return nil, fmt.Errorf("list services for stack %s: %w", stackName, stackErr)
		}
		if deploymentErr != nil {
			return nil, fmt.Errorf("list services for deployment %s: %w", deploymentID, deploymentErr)
		}
	}

	lines := make([]swarmTaskDiagnosticLine, 0)
	for _, serviceName := range serviceNames {
		psCmd := exec.CommandContext(ctx, "docker", "service", "ps", "--no-trunc", "--format", "{{json .}}", serviceName)
		psOutput, psErr := psCmd.Output()
		if psErr != nil {
			lines = append(lines, swarmTaskDiagnosticLine{
				ServiceName: serviceName,
				Line:        fmt.Sprintf("[runtime][%s] Failed to inspect service tasks: %v", serviceName, psErr),
				Timestamp:   time.Now(),
				Stderr:      true,
				LogLevel:    commonv1.LogLevel_LOG_LEVEL_WARN,
			})
			continue
		}

		for _, rawLine := range strings.Split(strings.TrimSpace(string(psOutput)), "\n") {
			if strings.TrimSpace(rawLine) == "" {
				continue
			}

			var row dockerServicePSRow
			if err := json.Unmarshal([]byte(rawLine), &row); err != nil {
				lines = append(lines, swarmTaskDiagnosticLine{
					ServiceName: serviceName,
					Line:        fmt.Sprintf("[runtime][%s] Failed to parse task status row: %s", serviceName, strings.TrimSpace(rawLine)),
					Timestamp:   time.Now(),
					Stderr:      true,
					LogLevel:    commonv1.LogLevel_LOG_LEVEL_WARN,
				})
				continue
			}

			message := fmt.Sprintf("[runtime][%s] task %s on %s: desired=%s current=%s", serviceName, row.ID, coalesce(row.Node, "unknown-node"), row.DesiredState, row.CurrentState)
			stderr := false
			level := commonv1.LogLevel_LOG_LEVEL_INFO

			if strings.TrimSpace(row.Error) != "" {
				message = fmt.Sprintf("%s error=%s", message, strings.TrimSpace(row.Error))
				stderr = true
				level = commonv1.LogLevel_LOG_LEVEL_ERROR
			} else if current := strings.ToLower(row.CurrentState); strings.Contains(current, "failed") || strings.Contains(current, "rejected") || strings.Contains(current, "shutdown") || strings.Contains(current, "complete") || strings.Contains(current, "pending") {
				level = commonv1.LogLevel_LOG_LEVEL_WARN
			}

			lines = append(lines, swarmTaskDiagnosticLine{
				ServiceName: serviceName,
				Line:        message,
				Timestamp:   time.Now(),
				Stderr:      stderr,
				LogLevel:    level,
			})
		}
	}

	return lines, nil
}

func coalesce(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shortenContainerID(containerID string) string {
	if len(containerID) > 12 {
		return containerID[:12]
	}
	if containerID == "" {
		return "unknown"
	}
	return containerID
}

func logLevelForContainerState(containerInfo container.InspectResponse) commonv1.LogLevel {
	if containerInfo.State == nil {
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	if containerInfo.State.Running {
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	if containerInfo.State.ExitCode != 0 || containerInfo.State.OOMKilled || containerInfo.State.Error != "" {
		return commonv1.LogLevel_LOG_LEVEL_ERROR
	}
	return commonv1.LogLevel_LOG_LEVEL_WARN
}

func formatContainerStateSummary(serviceName, containerID string, containerInfo container.InspectResponse) string {
	if containerInfo.State == nil {
		return fmt.Sprintf("[runtime] Service %s container %s state is unavailable", serviceName, shortenContainerID(containerID))
	}

	summary := fmt.Sprintf(
		"[runtime] Service %s container %s status=%s running=%t exit_code=%d restarting=%t oom_killed=%t",
		serviceName,
		shortenContainerID(containerID),
		containerInfo.State.Status,
		containerInfo.State.Running,
		containerInfo.State.ExitCode,
		containerInfo.State.Restarting,
		containerInfo.State.OOMKilled,
	)

	if containerInfo.State.Error != "" {
		summary += fmt.Sprintf(" error=%q", containerInfo.State.Error)
	}
	if containerInfo.State.FinishedAt != "" && !strings.HasPrefix(containerInfo.State.FinishedAt, "0001-01-01") {
		summary += fmt.Sprintf(" finished_at=%s", containerInfo.State.FinishedAt)
	}
	if containerInfo.State.StartedAt != "" && !strings.HasPrefix(containerInfo.State.StartedAt, "0001-01-01") {
		summary += fmt.Sprintf(" started_at=%s", containerInfo.State.StartedAt)
	}

	return summary
}

func (s *Service) notifyDeploymentFailure(ctx context.Context, deployment *database.Deployment, buildID string, buildNumber int32, title, message string, metadata map[string]string) {
	if deployment == nil || deployment.OrganizationID == "" {
		return
	}

	actionURL := fmt.Sprintf("/deployments/%s?tab=logs", deployment.ID)
	actionLabel := "View Logs"
	if buildID != "" {
		actionURL = fmt.Sprintf("/deployments/%s/builds/%s", deployment.ID, buildID)
		actionLabel = "View Diagnostics"
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["deployment_id"] = deployment.ID
	if buildID != "" {
		metadata["build_id"] = buildID
	}
	if buildNumber > 0 {
		metadata["build_number"] = fmt.Sprintf("%d", buildNumber)
	}
	metadata["has_runtime_diagnostics"] = "true"

	if err := notifications.CreateNotificationForOrganization(
		ctx,
		deployment.OrganizationID,
		notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT,
		notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH,
		title,
		message,
		&actionURL,
		&actionLabel,
		metadata,
		nil,
	); err != nil {
		logger.Warn("[RuntimeDiagnostics] Failed to create deployment notification for %s: %v", deployment.ID, err)
	}
}
