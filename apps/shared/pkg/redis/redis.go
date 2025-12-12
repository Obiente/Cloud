package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Config holds Redis configuration
type Config struct {
	Addr     string
	Password string
	DB       int
}

// InitRedis initializes the Redis client
func InitRedis(cfg Config) error {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}

	client = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("✓ Redis initialized at %s", cfg.Addr)
	return nil
}

// GetClient returns the Redis client
func GetClient() *redis.Client {
	return client
}

// Close closes the Redis client
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// LogEntry represents a single log line
type LogEntry struct {
	Line       string    `json:"line"`
	Stderr     bool      `json:"stderr"`
	LineNumber int32     `json:"line_number"`
	Timestamp  time.Time `json:"timestamp"`
}

// LogWriter writes logs to Redis using streams
type LogWriter struct {
	streamKey  string
	maxLen     int64
	lineNumber int32
}

// NewLogWriter creates a new Redis log writer
// streamKey format: "logs:{resource_type}:{resource_id}" e.g., "logs:vps:vps-123" or "logs:gameserver:gs-456"
func NewLogWriter(streamKey string) *LogWriter {
	return &LogWriter{
		streamKey:  streamKey,
		maxLen:     10000, // Keep last 10k entries per stream
		lineNumber: 0,
	}
}

// WriteLine writes a log line to Redis stream
func (w *LogWriter) WriteLine(line string, stderr bool) error {
	if client == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	w.lineNumber++

	entry := LogEntry{
		Line:       line,
		Stderr:     stderr,
		LineNumber: w.lineNumber,
		Timestamp:  time.Now().UTC(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add to Redis stream with MAXLEN to limit memory usage
	_, err = client.XAdd(ctx, &redis.XAddArgs{
		Stream: w.streamKey,
		MaxLen: w.maxLen,
		Approx: true, // Use approximate trimming for better performance
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to write to Redis stream: %w", err)
	}

	return nil
}

// ReadLogs reads logs from Redis stream
// Returns all logs if lastID is "0", or logs after lastID
func ReadLogs(ctx context.Context, streamKey string, lastID string, count int64) ([]LogEntry, string, error) {
	if client == nil {
		return nil, "", fmt.Errorf("Redis client not initialized")
	}

	if lastID == "" {
		lastID = "0" // Start from beginning
	}

	if count <= 0 {
		count = 100 // Default to 100 entries
	}

	// Read from stream
	streams, err := client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{streamKey, lastID},
		Count:   count,
		Block:   0, // Don't block, return immediately
	}).Result()

	if err == redis.Nil {
		// No data available
		return []LogEntry{}, lastID, nil
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to read from Redis stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return []LogEntry{}, lastID, nil
	}

	// Parse messages
	entries := make([]LogEntry, 0, len(streams[0].Messages))
	newLastID := lastID

	for _, msg := range streams[0].Messages {
		newLastID = msg.ID

		dataStr, ok := msg.Values["data"].(string)
		if !ok {
			logger.Warn("Invalid message format in stream %s: %v", streamKey, msg.Values)
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(dataStr), &entry); err != nil {
			logger.Warn("Failed to unmarshal log entry from stream %s: %v", streamKey, err)
			continue
		}

		entries = append(entries, entry)
	}

	return entries, newLastID, nil
}

// StreamLogs streams logs from Redis using blocking read
// Call with lastID="0" to start from beginning, or pass the last seen ID to continue
// Returns channel for log entries, last ID, and error channel
func StreamLogs(ctx context.Context, streamKey string, lastID string) (<-chan LogEntry, <-chan error) {
	logChan := make(chan LogEntry, 100)
	errChan := make(chan error, 1)

	if client == nil {
		errChan <- fmt.Errorf("Redis client not initialized")
		close(logChan)
		close(errChan)
		return logChan, errChan
	}

	go func() {
		defer close(logChan)
		defer close(errChan)

		currentID := lastID
		if currentID == "" {
			currentID = "0"
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Read with blocking (wait up to 5 seconds for new data)
			streams, err := client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{streamKey, currentID},
				Count:   100,
				Block:   5 * time.Second,
			}).Result()

			if err == redis.Nil {
				// Timeout, no new data - continue waiting
				continue
			}

			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled
					return
				}
				errChan <- fmt.Errorf("failed to read from Redis stream: %w", err)
				return
			}

			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			// Process messages
			for _, msg := range streams[0].Messages {
				currentID = msg.ID

				dataStr, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}

				var entry LogEntry
				if err := json.Unmarshal([]byte(dataStr), &entry); err != nil {
					continue
				}

				select {
				case logChan <- entry:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return logChan, errChan
}

// DeleteStream deletes a log stream
func DeleteStream(ctx context.Context, streamKey string) error {
	if client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	return client.Del(ctx, streamKey).Err()
}

// GetStreamLength returns the number of entries in a stream
func GetStreamLength(ctx context.Context, streamKey string) (int64, error) {
	if client == nil {
		return 0, fmt.Errorf("Redis client not initialized")
	}
	return client.XLen(ctx, streamKey).Result()
}

// SetExpiry sets an expiry time on a stream key
// Useful for auto-cleanup of old logs
func SetExpiry(ctx context.Context, streamKey string, expiry time.Duration) error {
	if client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	return client.Expire(ctx, streamKey, expiry).Err()
}
