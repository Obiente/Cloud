package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// StreamKeyGenerator generates Redis stream keys for different resource types
type StreamKeyGenerator func(resourceID string) string

// StreamKey returns Redis stream key for any resource
// Since resource IDs are already prefixed (e.g., vps-123, gs-456), we just use logs:{id}
func StreamKey(resourceID string) string {
	return fmt.Sprintf("logs:%s", resourceID)
}

// LogStreamer is a helper for streaming logs from Redis to gRPC/Connect streams
type LogStreamer struct {
	streamKey    string
	resourceID   string
	autoExpiry   time.Duration // Auto-expire logs after this duration
	enableExpiry bool
}

// NewLogStreamer creates a new log streamer helper
func NewLogStreamer(resourceID string) *LogStreamer {
	return &LogStreamer{
		streamKey:    StreamKey(resourceID),
		resourceID:   resourceID,
		enableExpiry: false,
	}
}

// WithAutoExpiry enables auto-expiry of logs after the specified duration
func (ls *LogStreamer) WithAutoExpiry(duration time.Duration) *LogStreamer {
	ls.enableExpiry = true
	ls.autoExpiry = duration
	return ls
}

// GetStreamKey returns the Redis stream key
func (ls *LogStreamer) GetStreamKey() string {
	return ls.streamKey
}

// WriteLog writes a log line to Redis
func (ls *LogStreamer) WriteLog(ctx context.Context, line string, stderr bool) error {
	writer := NewLogWriter(ls.streamKey)

	if err := writer.WriteLine(line, stderr); err != nil {
		return fmt.Errorf("failed to write log to Redis: %w", err)
	}

	// Set expiry if enabled (only on first write to avoid overhead)
	if ls.enableExpiry {
		if err := SetExpiry(ctx, ls.streamKey, ls.autoExpiry); err != nil {
			logger.Warn("Failed to set expiry on stream %s: %v", ls.streamKey, err)
		}
	}

	return nil
}

// ReadBufferedLogs reads historical logs from Redis
func (ls *LogStreamer) ReadBufferedLogs(ctx context.Context, lastID string, count int64) ([]LogEntry, string, error) {
	return ReadLogs(ctx, ls.streamKey, lastID, count)
}

// Stream starts streaming logs from Redis
// Returns a channel for log entries and an error channel
func (ls *LogStreamer) Stream(ctx context.Context, lastID string) (<-chan LogEntry, <-chan error) {
	return StreamLogs(ctx, ls.streamKey, lastID)
}

// Cleanup deletes the log stream from Redis
func (ls *LogStreamer) Cleanup(ctx context.Context) error {
	if err := DeleteStream(ctx, ls.streamKey); err != nil {
		return fmt.Errorf("failed to cleanup stream %s: %w", ls.streamKey, err)
	}
	logger.Info("Cleaned up Redis log stream: %s", ls.streamKey)
	return nil
}

// GetLength returns the number of log entries in the stream
func (ls *LogStreamer) GetLength(ctx context.Context) (int64, error) {
	return GetStreamLength(ctx, ls.streamKey)
}

// logWriterAdapter adapts LogStreamer to the orchestrator LogWriter interface
type logWriterAdapter struct {
	streamer *LogStreamer
}

// AsLogWriter returns an adapter that implements the orchestrator.LogWriter interface
// This allows LogStreamer to be used with provisioning orchestrators
func (ls *LogStreamer) AsLogWriter() interface {
	WriteLine(line string, stderr bool)
} {
	return &logWriterAdapter{streamer: ls}
}

// WriteLine implements the orchestrator.LogWriter interface (no error return)
func (lwa *logWriterAdapter) WriteLine(line string, stderr bool) {
	// Use background context for orchestrator log writes
	// Errors are logged but not returned to match the orchestrator interface
	if err := lwa.streamer.WriteLog(context.Background(), line, stderr); err != nil {
		logger.Error("Failed to write log to Redis: %v", err)
	}
}

// LogWriterAdapter adapts Redis LogWriter for use with provisioning orchestrators
type LogWriterAdapter struct {
	writer *LogWriter
}

// NewLogWriterAdapter creates a new adapter for the log writer
func NewLogWriterAdapter(streamKey string) *LogWriterAdapter {
	return &LogWriterAdapter{
		writer: NewLogWriter(streamKey),
	}
}

// Write implements io.Writer interface
func (lwa *LogWriterAdapter) Write(p []byte) (n int, err error) {
	line := string(p)
	if err := lwa.writer.WriteLine(line, false); err != nil {
		return 0, err
	}
	return len(p), nil
}

// WriteLine writes a single log line
func (lwa *LogWriterAdapter) WriteLine(line string, stderr bool) error {
	return lwa.writer.WriteLine(line, stderr)
}
