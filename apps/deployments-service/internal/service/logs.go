package deployments

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
	"github.com/moby/moby/api/types/events"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	remoteBuildLogPollInterval = 500 * time.Millisecond
	remoteBuildLogDrainWindow  = 3 * time.Second
	remoteBuildLogBatchSize    = 500
)

// GetDeploymentLogs retrieves a fixed number of deployment logs
func (s *Service) GetDeploymentLogs(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentLogsRequest]) (*connect.Response[deploymentsv1.GetDeploymentLogsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	lines := req.Msg.GetLines()
	if lines <= 0 {
		lines = 50
	}

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	persistedLogs, err := s.loadPersistedDeploymentLogLines(ctx, deploymentID, "", int(lines))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment logs: %w", err))
	}
	snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(ctx, deploymentID, "", lines)
	if snapshotErr == nil && len(snapshotLogs) > 0 {
		_ = s.persistDeploymentLogLines(ctx, deploymentID, "", "", "swarm_service_snapshot", snapshotLogs)
		persistedLogs = mergeDeploymentLogLines(int(lines), persistedLogs, snapshotLogs)
	}
	if len(persistedLogs) == 0 && s.manager != nil {
		if liveLogs, liveErr := s.manager.GetDeploymentLogs(ctx, deploymentID, fmt.Sprintf("%d", lines)); liveErr == nil && strings.TrimSpace(liveLogs) != "" {
			managerLogs := parseDockerServiceLogOutput(deploymentID, liveLogs)
			_ = s.persistDeploymentLogLines(ctx, deploymentID, "", "", "manager_snapshot", managerLogs)
			persistedLogs = mergeDeploymentLogLines(int(lines), persistedLogs, managerLogs)
		}
	}
	if len(persistedLogs) == 0 {
		persistedLogs, err = s.loadPersistedDiagnosticDeploymentLogLines(ctx, deploymentID, int(lines))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment diagnostic logs: %w", err))
		}
	}
	if len(persistedLogs) == 0 {
		return connect.NewResponse(&deploymentsv1.GetDeploymentLogsResponse{Logs: []string{}}), nil
	}

	logs := make([]string, 0, len(persistedLogs))
	for _, line := range persistedLogs {
		logs = append(logs, line.Line)
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

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	tail := req.Msg.GetTail()
	if tail <= 0 {
		tail = 200
	}

	persistedLogs, err := s.loadPersistedDeploymentLogLines(ctx, deploymentID, serviceName, int(tail))
	if err != nil {
		log.Printf("[StreamDeploymentLogs] Failed to load persisted logs for %s: %v", deploymentID, err)
		persistedLogs = nil
	}
	sentInitialHistory := false

	if containerID == "" {
		snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(ctx, deploymentID, serviceName, tail)
		if snapshotErr == nil && len(snapshotLogs) > 0 {
			_ = s.persistDeploymentLogLines(ctx, deploymentID, serviceName, "", "swarm_service_snapshot", snapshotLogs)
			persistedLogs = mergeDeploymentLogLines(int(tail), persistedLogs, snapshotLogs)
		}
		if err := sendDeploymentLogLines(stream, persistedLogs); err != nil {
			return err
		}
		sentInitialHistory = true
		streamedSwarmLogs, swarmStreamErr := s.streamSwarmServiceLogs(ctx, stream, deploymentID, serviceName, 0)
		if streamedSwarmLogs {
			return swarmStreamErr
		}
	}

	// Find container by container_id or service_name, or use first if neither specified
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(ctx, deploymentID, serviceName, tail)
		if snapshotErr == nil && len(snapshotLogs) > 0 {
			_ = s.persistDeploymentLogLines(ctx, deploymentID, serviceName, "", "swarm_service_snapshot", snapshotLogs)
			if len(persistedLogs) == 0 {
				for _, line := range snapshotLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
			}
			return nil
		}
		if len(persistedLogs) == 0 {
			diagnosticLogs, diagErr := s.loadPersistedDiagnosticDeploymentLogLines(ctx, deploymentID, int(tail))
			if diagErr == nil && len(diagnosticLogs) > 0 {
				for _, line := range diagnosticLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
				return nil
			}
			return connect.NewError(connect.CodeNotFound, err)
		}
		return nil
	}

	if !sentInitialHistory {
		if err := sendDeploymentLogLines(stream, persistedLogs); err != nil {
			return err
		}
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

	// Check if container is running to determine if we should follow logs
	// For stopped containers, we'll get historical logs but won't follow
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	isRunning := false
	if err == nil {
		isRunning = containerInfo.State.Running
	} else {
		snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(ctx, deploymentID, serviceName, tail)
		if snapshotErr == nil && len(snapshotLogs) > 0 {
			_ = s.persistDeploymentLogLines(ctx, deploymentID, serviceName, loc.ContainerID, "swarm_service_snapshot", snapshotLogs)
			if len(persistedLogs) == 0 {
				for _, line := range snapshotLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
			}
			return nil
		}
		if len(persistedLogs) == 0 {
			diagnosticLogs, diagErr := s.loadPersistedDiagnosticDeploymentLogLines(ctx, deploymentID, int(tail))
			if diagErr == nil && len(diagnosticLogs) > 0 {
				for _, line := range diagnosticLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
				return nil
			}
		}
	}

	// For stopped containers, use follow=false to get historical logs
	// For running containers, use follow=true to stream new logs
	follow := isRunning
	liveTail := tail
	if len(persistedLogs) > 0 {
		liveTail = 0
	}

	// Start container logs stream
	reader, err := dcli.ContainerLogs(ctx, loc.ContainerID, fmt.Sprintf("%d", liveTail), follow, nil, nil)
	if err != nil {
		snapshotLogs, snapshotErr := s.fetchSwarmServiceLogsSnapshot(ctx, deploymentID, serviceName, tail)
		if snapshotErr == nil && len(snapshotLogs) > 0 {
			_ = s.persistDeploymentLogLines(ctx, deploymentID, serviceName, loc.ContainerID, "swarm_service_snapshot", snapshotLogs)
			if len(persistedLogs) == 0 {
				for _, line := range snapshotLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
			}
			return nil
		}
		if len(persistedLogs) == 0 {
			diagnosticLogs, diagErr := s.loadPersistedDiagnosticDeploymentLogLines(ctx, deploymentID, int(tail))
			if diagErr == nil && len(diagnosticLogs) > 0 {
				for _, line := range diagnosticLogs {
					if sendErr := stream.Send(line); sendErr != nil {
						return sendErr
					}
				}
				return nil
			}
		}
		if len(persistedLogs) > 0 {
			return nil
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("logs: %w", err))
	}
	defer reader.Close()

	// Start Docker events stream filtered for this deployment
	eventFilters := map[string][]string{
		"type":  {"container", "image"},
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
		_ = readDockerContainerLogLines(reader, func(line *dockerLogLine) bool {
			protoLine := dockerLogLineToProto(deploymentID, line)
			return safeSend(protoLine)
		})
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

type dockerLogLine struct {
	line      string
	timestamp time.Time
	stderr    bool
}

type dockerLogLineBuffer struct {
	data []byte
}

func (b *dockerLogLineBuffer) appendFrame(payload []byte, stderr bool, emit func(*dockerLogLine) bool) bool {
	b.data = append(b.data, payload...)
	for {
		newline := bytes.IndexByte(b.data, '\n')
		if newline < 0 {
			return true
		}
		rawLine := string(b.data[:newline])
		b.data = b.data[newline+1:]
		if !emitDockerLogLine(rawLine, stderr, emit) {
			return false
		}
	}
}

func (b *dockerLogLineBuffer) flush(stderr bool, emit func(*dockerLogLine) bool) bool {
	if len(b.data) == 0 {
		return true
	}
	rawLine := string(b.data)
	b.data = nil
	return emitDockerLogLine(rawLine, stderr, emit)
}

func emitDockerLogLine(rawLine string, stderr bool, emit func(*dockerLogLine) bool) bool {
	rawLine = strings.TrimRight(rawLine, "\r")
	if strings.TrimSpace(rawLine) == "" {
		return true
	}
	timestamp, line := parseTimestampedDockerLogLine(strings.ToValidUTF8(rawLine, ""))
	return emit(&dockerLogLine{
		line:      line,
		timestamp: timestamp,
		stderr:    stderr,
	})
}

func dockerLogLineToProto(deploymentID string, line *dockerLogLine) *deploymentsv1.DeploymentLogLine {
	if line == nil {
		return nil
	}
	return &deploymentsv1.DeploymentLogLine{
		DeploymentId: deploymentID,
		Line:         line.line,
		Timestamp:    timestamppb.New(line.timestamp),
		Stderr:       line.stderr,
		LogLevel:     detectLogLevelFromContent(line.line, line.stderr),
	}
}

func readDockerContainerLogLines(reader io.Reader, emit func(*dockerLogLine) bool) error {
	stdoutBuffer := &dockerLogLineBuffer{}
	stderrBuffer := &dockerLogLineBuffer{}
	header := make([]byte, 8)

	for {
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				stdoutBuffer.flush(false, emit)
				stderrBuffer.flush(true, emit)
				return nil
			}
			return err
		}

		streamType := header[0]
		frameSize := binary.BigEndian.Uint32(header[4:])
		if (streamType != 1 && streamType != 2) || frameSize > 16*1024*1024 {
			return readPlainDockerLogLines(io.MultiReader(bytes.NewReader(header), reader), emit)
		}
		if frameSize == 0 {
			continue
		}

		payload := make([]byte, frameSize)
		if _, err := io.ReadFull(reader, payload); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		stderr := streamType == 2
		buffer := stdoutBuffer
		if stderr {
			buffer = stderrBuffer
		}
		if !buffer.appendFrame(payload, stderr, emit) {
			return nil
		}
	}
}

func readPlainDockerLogLines(reader io.Reader, emit func(*dockerLogLine) bool) error {
	buffer := &dockerLogLineBuffer{}
	chunk := make([]byte, 32*1024)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			if !buffer.appendFrame(chunk[:n], false, emit) {
				return nil
			}
		}
		if err != nil {
			if err == io.EOF {
				buffer.flush(false, emit)
				return nil
			}
			return err
		}
	}
}

func (s *Service) streamSwarmServiceLogs(ctx context.Context, stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], deploymentID, serviceName string, tail int32) (bool, error) {
	serviceNames, err := listSwarmServiceNames(ctx, deploymentID, serviceName)
	if err != nil || len(serviceNames) == 0 {
		return false, nil
	}

	logChan := make(chan *deploymentsv1.DeploymentLogLine, 100)
	var wg sync.WaitGroup
	started := 0

	for _, swarmServiceName := range serviceNames {
		args := []string{"service", "logs", "--timestamps", "--tail", fmt.Sprintf("%d", tail), "--raw", "--follow", swarmServiceName}
		cmd := exec.CommandContext(ctx, "docker", args...)

		stdout, stdoutErr := cmd.StdoutPipe()
		if stdoutErr != nil {
			continue
		}
		stderr, stderrErr := cmd.StderrPipe()
		if stderrErr != nil {
			continue
		}
		if startErr := cmd.Start(); startErr != nil {
			continue
		}
		started++

		sendLine := func(line *dockerLogLine) bool {
			select {
			case logChan <- dockerLogLineToProto(deploymentID, line):
				return true
			case <-ctx.Done():
				return false
			}
		}

		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = readPlainDockerLogLines(stdout, sendLine)
		}()
		go func() {
			defer wg.Done()
			_ = readPlainDockerLogLines(stderr, func(line *dockerLogLine) bool {
				line.stderr = true
				return sendLine(line)
			})
		}()
		go func() {
			defer wg.Done()
			if waitErr := cmd.Wait(); waitErr != nil && ctx.Err() == nil {
				log.Printf("[StreamDeploymentLogs] docker service logs failed for %s: %v", swarmServiceName, waitErr)
			}
		}()
	}
	if started == 0 {
		return false, nil
	}

	go func() {
		wg.Wait()
		close(logChan)
	}()

	var persistBatch []*deploymentsv1.DeploymentLogLine
	flushPersistBatch := func() {
		if len(persistBatch) == 0 {
			return
		}
		_ = s.persistDeploymentLogLines(context.Background(), deploymentID, serviceName, "", "swarm_service_live", persistBatch)
		persistBatch = nil
	}
	defer flushPersistBatch()

	for {
		select {
		case <-ctx.Done():
			flushPersistBatch()
			return true, nil
		case line, ok := <-logChan:
			if !ok {
				flushPersistBatch()
				return true, nil
			}
			if sendErr := stream.Send(line); sendErr != nil {
				if ctx.Err() != nil {
					flushPersistBatch()
					return true, nil
				}
				return true, sendErr
			}
			persistBatch = append(persistBatch, line)
			if len(persistBatch) >= 100 {
				flushPersistBatch()
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

var diagnosticRuntimeLogSources = []string{
	"compose_deploy_failed",
	"orchestrator_deploy_failed",
	"startup_verification_failed",
	"manual_start_failed",
	"manual_start_verification_failed",
	"manual_restart_failed",
	"manual_restart_verification_failed",
}

func (s *Service) loadPersistedDeploymentLogLines(ctx context.Context, deploymentID, serviceName string, limit int) ([]*deploymentsv1.DeploymentLogLine, error) {
	if s.runtimeLogsRepo == nil {
		return nil, nil
	}
	queryLimit := limit
	if queryLimit > 0 {
		queryLimit *= 3
	}
	logs, err := s.runtimeLogsRepo.GetRecentLogsForServiceExcludingSources(ctx, deploymentID, serviceName, queryLimit, diagnosticRuntimeLogSources)
	if err != nil {
		return nil, err
	}

	protoLogs := make([]*deploymentsv1.DeploymentLogLine, 0, len(logs))
	for _, entry := range logs {
		protoLogs = append(protoLogs, &deploymentsv1.DeploymentLogLine{
			DeploymentId: deploymentID,
			Line:         entry.Line,
			Timestamp:    timestamppb.New(entry.Timestamp),
			Stderr:       entry.Stderr,
			LogLevel:     commonv1.LogLevel(entry.LogLevel),
		})
	}
	return mergeDeploymentLogLines(limit, protoLogs), nil
}

func (s *Service) loadPersistedDiagnosticDeploymentLogLines(ctx context.Context, deploymentID string, limit int) ([]*deploymentsv1.DeploymentLogLine, error) {
	if s.runtimeLogsRepo == nil {
		return nil, nil
	}
	logs, err := s.runtimeLogsRepo.GetRecentLogsForSources(ctx, deploymentID, limit, diagnosticRuntimeLogSources)
	if err != nil {
		return nil, err
	}
	protoLogs := make([]*deploymentsv1.DeploymentLogLine, 0, len(logs))
	for _, entry := range logs {
		protoLogs = append(protoLogs, &deploymentsv1.DeploymentLogLine{
			DeploymentId: deploymentID,
			Line:         entry.Line,
			Timestamp:    timestamppb.New(entry.Timestamp),
			Stderr:       entry.Stderr,
			LogLevel:     commonv1.LogLevel(entry.LogLevel),
		})
	}
	return protoLogs, nil
}

func (s *Service) persistDeploymentLogLines(ctx context.Context, deploymentID, serviceName, containerID, source string, lines []*deploymentsv1.DeploymentLogLine) error {
	if s.runtimeLogsRepo == nil || len(lines) == 0 {
		return nil
	}

	entries := make([]database.DeploymentRuntimeLog, 0, len(lines))
	for _, line := range lines {
		if line == nil || strings.TrimSpace(line.Line) == "" {
			continue
		}
		timestamp := time.Now()
		if line.Timestamp != nil {
			timestamp = line.Timestamp.AsTime()
		}
		entries = append(entries, database.DeploymentRuntimeLog{
			DeploymentID: deploymentID,
			ServiceName:  serviceName,
			ContainerID:  containerID,
			NodeID:       "",
			Source:       source,
			Line:         line.Line,
			Timestamp:    timestamp,
			Stderr:       line.Stderr,
			LogLevel:     int32(line.LogLevel),
		})
	}

	return s.runtimeLogsRepo.AddLogsBatch(ctx, entries)
}

func sendDeploymentLogLines(stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], lines []*deploymentsv1.DeploymentLogLine) error {
	for _, line := range lines {
		if line == nil {
			continue
		}
		if err := stream.Send(line); err != nil {
			return err
		}
	}
	return nil
}

func mergeDeploymentLogLines(limit int, groups ...[]*deploymentsv1.DeploymentLogLine) []*deploymentsv1.DeploymentLogLine {
	seen := make(map[string]struct{})
	merged := make([]*deploymentsv1.DeploymentLogLine, 0)
	for _, group := range groups {
		for _, line := range group {
			if line == nil || strings.TrimSpace(line.Line) == "" {
				continue
			}
			key := deploymentLogLineKey(line)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, line)
		}
	}
	sortDeploymentLogLines(merged)
	if limit > 0 && len(merged) > limit {
		merged = merged[len(merged)-limit:]
	}
	return merged
}

func deploymentLogLineKey(line *deploymentsv1.DeploymentLogLine) string {
	ts := int64(0)
	nanos := int32(0)
	if line.Timestamp != nil {
		ts = line.Timestamp.Seconds
		nanos = line.Timestamp.Nanos
	}
	return fmt.Sprintf("%d.%09d|%t|%s", ts, nanos, line.Stderr, line.Line)
}

func (s *Service) fetchSwarmServiceLogsSnapshot(ctx context.Context, deploymentID, serviceName string, tail int32) ([]*deploymentsv1.DeploymentLogLine, error) {
	serviceNames, err := listSwarmServiceNames(ctx, deploymentID, serviceName)
	if err != nil || len(serviceNames) == 0 {
		return nil, err
	}

	var combined []*deploymentsv1.DeploymentLogLine
	for _, swarmServiceName := range serviceNames {
		args := []string{"service", "logs", "--timestamps", "--tail", fmt.Sprintf("%d", tail), "--raw", swarmServiceName}
		cmd := exec.CommandContext(ctx, "docker", args...)
		output, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			continue
		}
		combined = append(combined, parseDockerServiceLogOutput(deploymentID, string(output))...)
	}
	sortDeploymentLogLines(combined)
	if tail > 0 && len(combined) > int(tail) {
		combined = combined[len(combined)-int(tail):]
	}
	return combined, nil
}

func listSwarmServiceNames(ctx context.Context, deploymentID, serviceName string) ([]string, error) {
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{})
	addCandidate := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		candidates = append(candidates, name)
	}

	if serviceName != "" {
		addCandidate(serviceName)
		addCandidate(fmt.Sprintf("deploy-%s-%s", deploymentID, serviceName))
	}

	cmd := exec.CommandContext(ctx, "docker", "service", "ls", "--filter", fmt.Sprintf("label=cloud.obiente.deployment_id=%s", deploymentID), "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil && len(candidates) > 0 {
		return candidates, nil
	}
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		addCandidate(line)
	}
	return candidates, nil
}

func parseDockerServiceLogOutput(deploymentID, output string) []*deploymentsv1.DeploymentLogLine {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	lines := make([]*deploymentsv1.DeploymentLogLine, 0)
	for _, rawLine := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		rawLine = strings.TrimSpace(rawLine)
		if rawLine == "" {
			continue
		}
		timestamp, line := parseTimestampedDockerLogLine(rawLine)
		lines = append(lines, &deploymentsv1.DeploymentLogLine{
			DeploymentId: deploymentID,
			Line:         line,
			Timestamp:    timestamppb.New(timestamp),
			Stderr:       false,
			LogLevel:     detectLogLevelFromContent(line, false),
		})
	}
	sortDeploymentLogLines(lines)
	return lines
}

func sortDeploymentLogLines(lines []*deploymentsv1.DeploymentLogLine) {
	sort.SliceStable(lines, func(i, j int) bool {
		left := time.Time{}
		right := time.Time{}
		if lines[i] != nil && lines[i].Timestamp != nil {
			left = lines[i].Timestamp.AsTime()
		}
		if lines[j] != nil && lines[j].Timestamp != nil {
			right = lines[j].Timestamp.AsTime()
		}
		return left.Before(right)
	})
}

func parseTimestampedDockerLogLine(rawLine string) (time.Time, string) {
	parts := strings.SplitN(rawLine, " ", 2)
	if len(parts) == 2 {
		if ts, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			return ts, parts[1]
		}
	}
	return time.Now(), rawLine
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
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDeploymentRead, ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	buildStreamer := GetBuildLogStreamer(deploymentID)
	var latestBuild *database.BuildHistory

	// Give the async trigger path a short window to create the build record and bind
	// the local streamer before deciding how to stream logs.
	waitDeadline := time.Now().Add(5 * time.Second)
	for {
		builds, _, listErr := s.buildHistoryRepo.ListBuilds(ctx, deploymentID, orgID, 1, 0)
		if listErr == nil && len(builds) > 0 {
			latestBuild = builds[0]
		}
		if latestBuild != nil || buildStreamer.CurrentBuildID() != "" || time.Now().After(waitDeadline) || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(100 * time.Millisecond):
		}
	}

	if latestBuild != nil {
		localBuildID := buildStreamer.CurrentBuildID()
		if isBuildStreamingActive(latestBuild.Status) && latestBuild.ID != "" && latestBuild.ID != localBuildID {
			return s.streamRemoteBuildLogs(ctx, stream, deploymentID, latestBuild.ID)
		}
		if !isBuildStreamingActive(latestBuild.Status) {
			_, err := s.streamRemoteBuildLogsSnapshot(ctx, stream, deploymentID, latestBuild.ID)
			return err
		}
	}

	// Subscribe FIRST to avoid race conditions where logs are written
	// between getting buffered logs and subscribing
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

func isBuildStreamingActive(status int32) bool {
	return status == int32(deploymentsv1.BuildStatus_BUILD_PENDING) || status == int32(deploymentsv1.BuildStatus_BUILD_BUILDING)
}

func buildLogEntryToProto(deploymentID string, logEntry *database.BuildLog) *deploymentsv1.DeploymentLogLine {
	return &deploymentsv1.DeploymentLogLine{
		DeploymentId: deploymentID,
		Line:         logEntry.Line,
		Timestamp:    timestamppb.New(logEntry.Timestamp),
		Stderr:       logEntry.Stderr,
		LogLevel:     detectLogLevelFromContent(logEntry.Line, logEntry.Stderr),
	}
}

func sendBuildLogEntries(stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], deploymentID string, logs []*database.BuildLog) (int32, error) {
	var lastLineNumber int32 = -1
	for _, logEntry := range logs {
		if logEntry == nil {
			continue
		}
		if err := stream.Send(buildLogEntryToProto(deploymentID, logEntry)); err != nil {
			return lastLineNumber, err
		}
		lastLineNumber = logEntry.LineNumber
	}
	return lastLineNumber, nil
}

func (s *Service) streamRemoteBuildLogsSnapshot(ctx context.Context, stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], deploymentID, buildID string) (int32, error) {
	if buildID == "" {
		return -1, nil
	}

	buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
	lastLineNumber := int32(-1)
	for {
		logs, err := buildLogsRepo.GetBuildLogsAfterLine(ctx, buildID, lastLineNumber, remoteBuildLogBatchSize)
		if err != nil {
			return lastLineNumber, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load build logs: %w", err))
		}
		if len(logs) == 0 {
			return lastLineNumber, nil
		}
		sentUntil, err := sendBuildLogEntries(stream, deploymentID, logs)
		if err != nil {
			if ctx.Err() != nil {
				return lastLineNumber, nil
			}
			return lastLineNumber, nil
		}
		lastLineNumber = sentUntil
		if len(logs) < remoteBuildLogBatchSize {
			return lastLineNumber, nil
		}
	}
}

func (s *Service) streamRemoteBuildLogs(ctx context.Context, stream *connect.ServerStream[deploymentsv1.DeploymentLogLine], deploymentID, buildID string) error {
	lastLineNumber, err := s.streamRemoteBuildLogsSnapshot(ctx, stream, deploymentID, buildID)
	if err != nil {
		return err
	}

	buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
	drainingSince := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(remoteBuildLogPollInterval):
		}

		logs, err := buildLogsRepo.GetBuildLogsAfterLine(ctx, buildID, lastLineNumber, remoteBuildLogBatchSize)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to poll build logs: %w", err))
		}
		if len(logs) > 0 {
			sentUntil, sendErr := sendBuildLogEntries(stream, deploymentID, logs)
			if sendErr != nil {
				if ctx.Err() != nil {
					return nil
				}
				return nil
			}
			lastLineNumber = sentUntil
		}

		build, err := s.buildHistoryRepo.GetBuildByID(ctx, buildID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check build status: %w", err))
		}

		if isBuildStreamingActive(build.Status) {
			drainingSince = time.Time{}
			continue
		}

		if drainingSince.IsZero() {
			drainingSince = time.Now()
			continue
		}

		if len(logs) == 0 && time.Since(drainingSince) >= remoteBuildLogDrainWindow {
			finalLogs, err := buildLogsRepo.GetBuildLogsAfterLine(ctx, buildID, lastLineNumber, remoteBuildLogBatchSize)
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to flush final build logs: %w", err))
			}
			if len(finalLogs) > 0 {
				if _, err := sendBuildLogEntries(stream, deploymentID, finalLogs); err != nil {
					return nil
				}
			}
			return nil
		}
	}
}
