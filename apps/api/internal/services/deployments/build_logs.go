package deployments

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"api/internal/database"
	"api/internal/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildLogStreamer manages streaming of build logs for a deployment
type BuildLogStreamer struct {
	deploymentID string
	buildID      string // Optional: ID of the build record in database
	buildLogsRepo *database.BuildLogsRepository // Repository for saving logs to TimescaleDB
	subscribers  map[chan *deploymentsv1.DeploymentLogLine]struct{}
	mu           sync.RWMutex
	logs         []*deploymentsv1.DeploymentLogLine
	maxLogs      int
	lineNumber   int32 // Sequential line number for database storage
	
	// Batching for TimescaleDB
	pendingLogs []struct {
		Line      string
		Stderr    bool
		LineNumber int32
		Timestamp time.Time
	}
	batchMutex  sync.Mutex
	batchTimer  *time.Timer
	stopBatching chan struct{}
	closed      bool // Track if Close() has been called
	closeMutex  sync.Mutex // Protect close operation
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

	// Use TimescaleDB for build logs (MetricsDB connection)
	buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
	
	streamer := &BuildLogStreamer{
		deploymentID:     deploymentID,
		buildLogsRepo:    buildLogsRepo,
		subscribers:      make(map[chan *deploymentsv1.DeploymentLogLine]struct{}),
		logs:             make([]*deploymentsv1.DeploymentLogLine, 0),
		maxLogs:          10000, // Keep last 10k lines
		lineNumber:       0,
		pendingLogs:      make([]struct {
			Line      string
			Stderr    bool
			LineNumber int32
			Timestamp time.Time
		}, 0, 100), // Pre-allocate with capacity for batching
		stopBatching: make(chan struct{}),
	}
	
	// Start batch flusher
	go streamer.startBatchFlusher()
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
func detectLogLevel(line string, isStderr bool) commonv1.LogLevel {
	// Use shared detection function (duplicated here to avoid circular imports)
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
	// Nixpacks writes progress to stderr but it's informational
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
	// But for build tools, stderr is often just progress, so default to INFO
	if isStderr {
		// Be conservative: if it's on stderr and looks suspicious, mark as WARN
		// Otherwise treat as INFO (common for build tools)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "fail") {
			return commonv1.LogLevel_LOG_LEVEL_WARN
		}
		return commonv1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Default to INFO for stdout
	return commonv1.LogLevel_LOG_LEVEL_INFO
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
		currentRepo := s.buildLogsRepo
		lineNumber := s.lineNumber
		if currentBuildID != "" {
			s.lineNumber++
		}
		s.mu.Unlock()

		// Add to batch queue for database if build ID is set (async to avoid blocking)
		if currentBuildID != "" && currentRepo != nil {
			s.addToBatch(l, stderr, lineNumber)
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
				logger.Debug("[BuildLogStreamer] Timeout sending buffered log to subscriber")
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
				logger.Debug("[BuildLogStreamer] Warning: Attempted to close already-closed channel (this is safe to ignore)")
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

// DetectPortFromLogs parses the first N log lines to detect a port number
// Looks for common patterns like "Listening on port 3000", "server running on :8080", etc.
// Returns 0 if no port is found
func (s *BuildLogStreamer) DetectPortFromLogs(maxLines int) int {
	if maxLines <= 0 {
		maxLines = 100 // Check first 100 lines by default
	}

	s.mu.RLock()
	logsToCheck := make([]*deploymentsv1.DeploymentLogLine, 0, maxLines)
	if len(s.logs) > maxLines {
		logsToCheck = append(logsToCheck, s.logs[:maxLines]...)
	} else {
		logsToCheck = append(logsToCheck, s.logs...)
	}
	s.mu.RUnlock()

	return detectPortFromLogLines(logsToCheck)
}

// detectPortFromLogLines parses log lines for port patterns
func detectPortFromLogLines(logLines []*deploymentsv1.DeploymentLogLine) int {
	if len(logLines) == 0 {
		return 0
	}

	// Common port detection patterns
	portPatterns := []*regexp.Regexp{
		// Astro: "local: http://localhost:4321" or "!  local: http://localhost:4321"
		regexp.MustCompile(`(?i)(?:local|network)\s*:\s*https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0|.*?):(\d+)`),
		// "Listening on port 3000"
		regexp.MustCompile(`(?i)(?:listening|listen|running|started|server).*?(?:on|at|port|:)\s*(?:port\s*:?\s*)?(\d+)`),
		// "Server running on :8080"
		regexp.MustCompile(`(?i)(?:server|app|application).*?(?:running|started|listening).*?(?:on|at|:)\s*(?:port\s*:?\s*)?(\d+)`),
		// "Port: 3000" or "PORT=3000"
		regexp.MustCompile(`(?i)(?:^|\s)(?:port|PORT)\s*[:=]\s*(\d+)`),
		// ":8080" or "localhost:8080" or "0.0.0.0:8080"
		regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0|::)\s*:\s*(\d+)`),
		// "http://localhost:3000" or "http://0.0.0.0:8080"
		regexp.MustCompile(`(?i)https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0|.*?):(\d+)`),
		// "binding to port 3000"
		regexp.MustCompile(`(?i)(?:binding|bound|binding\s+to).*?(?:port|:)\s*(\d+)`),
		// Next.js: "Ready on http://localhost:3000"
		regexp.MustCompile(`(?i)(?:ready|started).*?https?://.*?:(\d+)`),
		// Rails: "Listening on tcp://0.0.0.0:3000"
		regexp.MustCompile(`(?i)tcp://.*?:(\d+)`),
		// Python: "Running on http://127.0.0.1:8000"
		regexp.MustCompile(`(?i)(?:running|uvicorn|gunicorn).*?https?://.*?:(\d+)`),
		// Go: "Listening on :8080"
		regexp.MustCompile(`(?i)(?:listening|listen).*?:\s*(\d+)`),
		// Node/Express: "App listening on port 3000"
		regexp.MustCompile(`(?i)(?:app|server).*?(?:listening|listen).*?(?:on|port)\s*(\d+)`),
	}

	// Check each log line
	for _, logLine := range logLines {
		if logLine == nil || logLine.Line == "" {
			continue
		}

		line := logLine.Line
		// Try each pattern
		for _, pattern := range portPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				// Extract port number from match
				portStr := matches[1]
				if port, err := strconv.Atoi(portStr); err == nil {
					if port > 0 && port < 65536 {
						return port
					}
				}
			}
		}
	}

	return 0
}

// addToBatch adds a log line to the batch queue for efficient database writes
func (s *BuildLogStreamer) addToBatch(line string, stderr bool, lineNumber int32) {
	s.batchMutex.Lock()
	defer s.batchMutex.Unlock()

	// Add to pending batch
	s.pendingLogs = append(s.pendingLogs, struct {
		Line      string
		Stderr    bool
		LineNumber int32
		Timestamp time.Time
	}{
		Line:      line,
		Stderr:    stderr,
		LineNumber: lineNumber,
		Timestamp: time.Now(),
	})

	// Get buildID and repo safely (need to check if buildID is set)
	s.mu.RLock()
	buildID := s.buildID
	repo := s.buildLogsRepo
	s.mu.RUnlock()

	if buildID == "" || repo == nil {
		return // Can't flush without buildID
	}

	// Flush immediately if batch is large enough (100 items)
	if len(s.pendingLogs) >= 100 {
		s.flushBatch(buildID, repo)
	} else {
		// Reset timer for auto-flush (flush after 500ms of inactivity)
		if s.batchTimer != nil {
			s.batchTimer.Stop()
		}
		s.batchTimer = time.AfterFunc(500*time.Millisecond, func() {
			s.batchMutex.Lock()
			defer s.batchMutex.Unlock()
			if len(s.pendingLogs) > 0 {
				s.mu.RLock()
				bid := s.buildID
				r := s.buildLogsRepo
				s.mu.RUnlock()
				if bid != "" && r != nil {
					s.flushBatch(bid, r)
				}
			}
		})
	}
}

// flushBatch writes the pending batch to the database
func (s *BuildLogStreamer) flushBatch(buildID string, repo *database.BuildLogsRepository) {
	if len(s.pendingLogs) == 0 {
		return
	}

	// Copy batch for async write
	batch := make([]struct {
		Line      string
		Stderr    bool
		LineNumber int32
		Timestamp time.Time
	}, len(s.pendingLogs))
	copy(batch, s.pendingLogs)
	s.pendingLogs = s.pendingLogs[:0] // Clear batch

	// Write asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := repo.AddBuildLogsBatch(ctx, buildID, batch); err != nil {
			logger.Debug("[BuildLogStreamer] Failed to save batch of %d logs to database: %v", len(batch), err)
		}
	}()
}

// startBatchFlusher periodically flushes any remaining logs
func (s *BuildLogStreamer) startBatchFlusher() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			buildID := s.buildID
			repo := s.buildLogsRepo
			s.mu.RUnlock()
			
			s.batchMutex.Lock()
			pendingCount := len(s.pendingLogs)
			s.batchMutex.Unlock()

			if buildID != "" && repo != nil && pendingCount > 0 {
				s.batchMutex.Lock()
				s.flushBatch(buildID, repo)
				s.batchMutex.Unlock()
			}
		case <-s.stopBatching:
			// Final flush before stopping
			s.mu.RLock()
			buildID := s.buildID
			repo := s.buildLogsRepo
			s.mu.RUnlock()
			
			s.batchMutex.Lock()
			if buildID != "" && repo != nil && len(s.pendingLogs) > 0 {
				s.flushBatch(buildID, repo)
			}
			s.batchMutex.Unlock()
			return
		}
	}
}

// Close cleans up the streamer
// Safe to call multiple times - uses sync to prevent double-close
func (s *BuildLogStreamer) Close() {
	s.closeMutex.Lock()
	defer s.closeMutex.Unlock()
	
	// Check if already closed
	if s.closed {
		logger.Debug("[BuildLogStreamer] Close() called on already-closed streamer, ignoring")
		return
	}
	
	s.closed = true
	
	// Stop batch flusher and do final flush
	// Safe to close - we've checked that we haven't closed before
	close(s.stopBatching)
	
	s.mu.RLock()
	buildID := s.buildID
	s.mu.RUnlock()
	
	s.mu.RLock()
	repo := s.buildLogsRepo
	s.mu.RUnlock()
	
	s.batchMutex.Lock()
	if buildID != "" && repo != nil && len(s.pendingLogs) > 0 {
		s.flushBatch(buildID, repo)
	}
	if s.batchTimer != nil {
		s.batchTimer.Stop()
	}
	s.batchMutex.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscribers safely
	for ch := range s.subscribers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel already closed, ignore panic
					logger.Debug("[BuildLogStreamer] Warning: Attempted to close already-closed channel during Close()")
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
