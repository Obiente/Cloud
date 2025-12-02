package vps

import (
	"strings"
	"sync"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// VPSLogStreamer manages streaming of provisioning logs for a VPS
type VPSLogStreamer struct {
	vpsID       string
	subscribers map[chan *vpsv1.VPSLogLine]struct{}
	mu          sync.RWMutex
	logs        []*vpsv1.VPSLogLine
	maxLogs     int
	lineNumber  int32
}

// VPSLogStreamerRegistry manages VPS log streamers for all VPS instances
type VPSLogStreamerRegistry struct {
	streamers map[string]*VPSLogStreamer
	mu        sync.RWMutex
}

var globalVPSLogRegistry = &VPSLogStreamerRegistry{
	streamers: make(map[string]*VPSLogStreamer),
}

// GetVPSLogStreamer gets or creates a VPS log streamer for a VPS instance
func GetVPSLogStreamer(vpsID string) *VPSLogStreamer {
	globalVPSLogRegistry.mu.Lock()
	defer globalVPSLogRegistry.mu.Unlock()

	if streamer, ok := globalVPSLogRegistry.streamers[vpsID]; ok {
		return streamer
	}

	streamer := &VPSLogStreamer{
		vpsID:       vpsID,
		subscribers: make(map[chan *vpsv1.VPSLogLine]struct{}),
		logs:        make([]*vpsv1.VPSLogLine, 0),
		maxLogs:     10000, // Keep last 10k lines
		lineNumber:  0,
	}

	globalVPSLogRegistry.streamers[vpsID] = streamer
	return streamer
}

// WriteLine writes a log line to the streamer (implements orchestrator.LogWriter)
func (s *VPSLogStreamer) WriteLine(line string, stderr bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Trim whitespace
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Increment line number
	s.lineNumber++

	// Create log line
	logLine := &vpsv1.VPSLogLine{
		Line:       line,
		Stderr:     stderr,
		LineNumber: s.lineNumber,
		Timestamp:  timestamppb.Now(),
	}

	// Add to logs buffer
	s.logs = append(s.logs, logLine)

	// Trim old logs if over limit
	if len(s.logs) > s.maxLogs {
		// Keep the most recent logs
		keepCount := s.maxLogs / 2
		s.logs = s.logs[len(s.logs)-keepCount:]
	}

	// Broadcast to all subscribers
	for subChan := range s.subscribers {
		select {
		case subChan <- logLine:
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// Write implements io.Writer to capture output
func (s *VPSLogStreamer) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			s.WriteLine(line, false)
		}
	}
	return len(p), nil
}

// WriteStderr writes a line to stderr
func (s *VPSLogStreamer) WriteStderr(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			s.WriteLine(line, true)
		}
	}
	return len(p), nil
}

// GetLogs returns all buffered logs
func (s *VPSLogStreamer) GetLogs() []*vpsv1.VPSLogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	logs := make([]*vpsv1.VPSLogLine, len(s.logs))
	copy(logs, s.logs)
	return logs
}

// Subscribe creates a new subscription channel for log updates
func (s *VPSLogStreamer) Subscribe() chan *vpsv1.VPSLogLine {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *vpsv1.VPSLogLine, 100) // Buffered channel
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscription channel
func (s *VPSLogStreamer) Unsubscribe(ch chan *vpsv1.VPSLogLine) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, ch)
	close(ch)
}

// Close closes the streamer and cleans up resources
func (s *VPSLogStreamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscriber channels
	for ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = make(map[chan *vpsv1.VPSLogLine]struct{})

	// Remove from registry
	globalVPSLogRegistry.mu.Lock()
	delete(globalVPSLogRegistry.streamers, s.vpsID)
	globalVPSLogRegistry.mu.Unlock()
}
