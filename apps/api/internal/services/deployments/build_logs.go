package deployments

import (
	"bufio"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildLogStreamer manages streaming of build logs for a deployment
type BuildLogStreamer struct {
	deploymentID string
	subscribers  map[chan *deploymentsv1.DeploymentLogLine]struct{}
	mu           sync.RWMutex
	logs         []*deploymentsv1.DeploymentLogLine
	maxLogs      int
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
		deploymentID: deploymentID,
		subscribers:  make(map[chan *deploymentsv1.DeploymentLogLine]struct{}),
		logs:         make([]*deploymentsv1.DeploymentLogLine, 0),
		maxLogs:      10000, // Keep last 10k lines
	}
	globalBuildLogRegistry.streamers[deploymentID] = streamer
	return streamer
}

// Write implements io.Writer to capture build output
func (s *BuildLogStreamer) Write(p []byte) (n int, err error) {
	return s.writeLine(string(p), false)
}

// WriteStderr writes a line to stderr
func (s *BuildLogStreamer) WriteStderr(p []byte) (n int, err error) {
	return s.writeLine(string(p), true)
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
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, l := range lines {
		logLine := &deploymentsv1.DeploymentLogLine{
			DeploymentId: s.deploymentID,
			Line:         l,
			Timestamp:    now,
			Stderr:       stderr,
		}

		// Store in buffer (with size limit)
		s.logs = append(s.logs, logLine)
		if len(s.logs) > s.maxLogs {
			s.logs = s.logs[len(s.logs)-s.maxLogs:]
		}

		// Broadcast to all subscribers
		for ch := range s.subscribers {
			select {
			case ch <- logLine:
			default:
				// Skip if channel is full (non-blocking)
			}
		}
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
	delete(s.subscribers, ch)
	close(ch)
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

	// Close all subscribers
	for ch := range s.subscribers {
		close(ch)
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
