package deployments

import (
	"bufio"
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/database"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildLogStreamer manages streaming of build logs for a deployment
type BuildLogStreamer struct {
	deploymentID string
	buildID      string // Optional: ID of the build record in database
	buildHistoryRepo *database.BuildHistoryRepository // Repository for saving logs
	subscribers  map[chan *deploymentsv1.DeploymentLogLine]struct{}
	mu           sync.RWMutex
	logs         []*deploymentsv1.DeploymentLogLine
	maxLogs      int
	lineNumber   int32 // Sequential line number for database storage
}

// BuildLogStreamerRegistry manages build log streamers for all deployments
type BuildLogStreamerRegistry struct {
	streamers map[string]*BuildLogStreamer
	mu        sync.RWMutex
}

var globalBuildLogRegistry = &BuildLogStreamerRegistry{
	streamers: make(map[string]*BuildLogStreamer),
}

// GetBuildLogStreamer gets or creates a build log streamer for a deployment
func GetBuildLogStreamer(deploymentID string) *BuildLogStreamer {
	globalBuildLogRegistry.mu.Lock()
	defer globalBuildLogRegistry.mu.Unlock()

	if streamer, ok := globalBuildLogRegistry.streamers[deploymentID]; ok {
		return streamer
	}

	streamer := &BuildLogStreamer{
		deploymentID:     deploymentID,
		buildHistoryRepo: database.NewBuildHistoryRepository(database.DB),
		subscribers:      make(map[chan *deploymentsv1.DeploymentLogLine]struct{}),
		logs:             make([]*deploymentsv1.DeploymentLogLine, 0),
		maxLogs:          10000, // Keep last 10k lines
		lineNumber:       0,
	}
	globalBuildLogRegistry.streamers[deploymentID] = streamer
	return streamer
}

// SetBuildID sets the build ID for database persistence
func (s *BuildLogStreamer) SetBuildID(buildID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buildID = buildID
	s.lineNumber = 0 // Reset line number for new build
}

// Write implements io.Writer to capture build output
func (s *BuildLogStreamer) Write(p []byte) (n int, err error) {
	return s.writeLine(string(p), false)
}

// WriteStderr writes a line to stderr
func (s *BuildLogStreamer) WriteStderr(p []byte) (n int, err error) {
	return s.writeLine(string(p), true)
}

// detectLogLevel intelligently detects the log level from log line content
// This is important because build tools like nixpacks write normal output to stderr
// but those lines are not actually errors - they're informational build progress
func detectLogLevel(line string, isStderr bool) deploymentsv1.LogLevel {
	// Use shared detection function (duplicated here to avoid circular imports)
	lineLower := strings.ToLower(strings.TrimSpace(line))
	
	// Check for explicit log level markers (case-insensitive)
	if strings.Contains(lineLower, "[error]") || strings.Contains(lineLower, "error:") ||
		strings.Contains(lineLower, "fatal:") || strings.Contains(lineLower, "failed") ||
		strings.HasPrefix(lineLower, "error") || strings.Contains(lineLower, " ❌ ") {
		return deploymentsv1.LogLevel_LOG_LEVEL_ERROR
	}
	
	if strings.Contains(lineLower, "[warn]") || strings.Contains(lineLower, "[warning]") ||
		strings.Contains(lineLower, "warning:") || strings.Contains(lineLower, "⚠️") ||
		strings.HasPrefix(lineLower, "warn") {
		return deploymentsv1.LogLevel_LOG_LEVEL_WARN
	}
	
	if strings.Contains(lineLower, "[debug]") || strings.Contains(lineLower, "[trace]") ||
		strings.HasPrefix(lineLower, "debug") || strings.HasPrefix(lineLower, "trace") {
		return deploymentsv1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Nixpacks/Railpacks specific patterns - these are INFO even if on stderr
	// Nixpacks writes progress to stderr but it's informational
	if strings.Contains(lineLower, "nixpacks") || strings.Contains(lineLower, "railpacks") ||
		strings.Contains(lineLower, "building") || strings.Contains(lineLower, "setup") ||
		strings.Contains(lineLower, "install") || strings.Contains(lineLower, "build") ||
		strings.Contains(lineLower, "start") || strings.Contains(lineLower, "transferring") ||
		strings.Contains(lineLower, "loading") || strings.Contains(lineLower, "resolving") ||
		strings.Contains(lineLower, "[internal]") || strings.Contains(lineLower, "[stage-") ||
		strings.Contains(lineLower, "sha256:") || strings.Contains(lineLower, "done") ||
		strings.Contains(lineLower, "dockerfile:") || strings.Contains(lineLower, "context:") ||
		strings.Contains(lineLower, "metadata") {
		return deploymentsv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Docker build output patterns - usually INFO
	if strings.Contains(lineLower, "[") && strings.Contains(lineLower, "]") &&
		(strings.Contains(lineLower, "step") || strings.Contains(lineLower, "from") ||
		strings.Contains(lineLower, "running") || strings.Contains(lineLower, "executing")) {
		return deploymentsv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Default: if stderr is true and no pattern matched, it might be an error
	// But for build tools, stderr is often just progress, so default to INFO
	if isStderr {
		// Be conservative: if it's on stderr and looks suspicious, mark as WARN
		// Otherwise treat as INFO (common for build tools)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "fail") {
			return deploymentsv1.LogLevel_LOG_LEVEL_WARN
		}
		return deploymentsv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Default to INFO for stdout
	return deploymentsv1.LogLevel_LOG_LEVEL_INFO
}

// writeLine writes a log line and broadcasts it to all subscribers
func (s *BuildLogStreamer) writeLine(line string, stderr bool) (int, error) {
	if line == "" {
		return 0, nil
	}

	// Split multi-line output (handle newlines)
	lines := []string{}
	scanner := bufio.NewScanner(strings.NewReader(line))
	for scanner.Scan() {
		text := scanner.Text()
		if text != "" {
			lines = append(lines, text)
		}
	}
	if len(lines) == 0 && line != "" {
		// If no newlines, treat entire string as one line
		lines = []string{strings.TrimRight(line, "\n\r")}
	}

	now := timestamppb.Now()

	for _, l := range lines {
		// Detect log level from content
		logLevel := detectLogLevel(l, stderr)
		
		logLine := &deploymentsv1.DeploymentLogLine{
			DeploymentId: s.deploymentID,
			Line:         l,
			Timestamp:    now,
			Stderr:       stderr, // Keep for backward compatibility
			LogLevel:     logLevel,
		}

		s.mu.Lock()
		// Store in buffer (with size limit)
		s.logs = append(s.logs, logLine)
		if len(s.logs) > s.maxLogs {
			s.logs = s.logs[len(s.logs)-s.maxLogs:]
		}

		// Get build info and increment line number if saving to DB
		currentBuildID := s.buildID
		currentRepo := s.buildHistoryRepo
		lineNumber := s.lineNumber
		if currentBuildID != "" {
			s.lineNumber++
		}
		s.mu.Unlock()

		// Save to database if build ID is set (async to avoid blocking)
		if currentBuildID != "" && currentRepo != nil {
			go func(buildID string, repo *database.BuildHistoryRepository, line string, isStderr bool, lineNum int32) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := repo.AddBuildLog(ctx, buildID, line, isStderr, lineNum); err != nil {
					log.Printf("[BuildLogStreamer] Failed to save log to database: %v", err)
				}
			}(currentBuildID, currentRepo, l, stderr, lineNumber)
		}

		s.mu.RLock()
		// Broadcast to all subscribers
		for ch := range s.subscribers {
			select {
			case ch <- logLine:
			default:
				// Skip if channel is full (non-blocking)
			}
		}
		s.mu.RUnlock()
	}

	return len(line), nil
}

// Subscribe creates a new subscription channel for build logs
func (s *BuildLogStreamer) Subscribe() chan *deploymentsv1.DeploymentLogLine {
	ch := make(chan *deploymentsv1.DeploymentLogLine, 100) // Buffered to avoid blocking

	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers[ch] = struct{}{}

	// Send buffered logs to new subscriber
	go func() {
		s.mu.RLock()
		bufferedLogs := make([]*deploymentsv1.DeploymentLogLine, len(s.logs))
		copy(bufferedLogs, s.logs)
		s.mu.RUnlock()

		for _, logLine := range bufferedLogs {
			select {
			case ch <- logLine:
			case <-time.After(1 * time.Second):
				log.Printf("[BuildLogStreamer] Timeout sending buffered log to subscriber")
				return
			}
		}
	}()

	return ch
}

// Unsubscribe removes a subscription
func (s *BuildLogStreamer) Unsubscribe(ch chan *deploymentsv1.DeploymentLogLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if channel is still in subscribers (not already unsubscribed/closed)
	if _, exists := s.subscribers[ch]; !exists {
		// Already unsubscribed, ignore
		return
	}
	
	delete(s.subscribers, ch)
	
	// Safely close channel with recover to handle already-closed channels
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel already closed, ignore panic
				log.Printf("[BuildLogStreamer] Warning: Attempted to close already-closed channel (this is safe to ignore)")
			}
		}()
		close(ch)
	}()
}

// GetLogs returns all buffered logs
func (s *BuildLogStreamer) GetLogs() []*deploymentsv1.DeploymentLogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]*deploymentsv1.DeploymentLogLine, len(s.logs))
	copy(logs, s.logs)
	return logs
}

// Close cleans up the streamer
func (s *BuildLogStreamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscribers safely
	for ch := range s.subscribers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel already closed, ignore panic
					log.Printf("[BuildLogStreamer] Warning: Attempted to close already-closed channel during Close()")
				}
			}()
			close(ch)
		}()
	}
	s.subscribers = make(map[chan *deploymentsv1.DeploymentLogLine]struct{})

	// Remove from registry
	globalBuildLogRegistry.mu.Lock()
	delete(globalBuildLogRegistry.streamers, s.deploymentID)
	globalBuildLogRegistry.mu.Unlock()
}

// MultiWriter is a writer that writes to multiple writers
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a new multi-writer
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all writers
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}
