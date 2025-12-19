package chunkupload

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	sharedredis "github.com/obiente/cloud/apps/shared/pkg/redis"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	redispkg "github.com/redis/go-redis/v9"
)

// Session tracks in-flight file chunks metadata (stored in Redis)
type Session struct {
	ResourceID     string    `json:"resource_id"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	TotalChunks    int32     `json:"total_chunks"`
	BytesReceived  int64     `json:"bytes_received"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

// Manager manages chunk upload sessions using Redis. All chunks and metadata
// are stored in Redis with automatic TTL-based expiry.
type Manager struct {
	timeout time.Duration
}

// NewManager creates a new chunk upload session manager using Redis.
// Panics if Redis is not initialized.
func NewManager(timeout time.Duration) *Manager {
	if sharedredis.GetClient() == nil {
		logger.Fatalf("[ChunkUploadManager] Redis client not initialized - required for chunk uploads")
	}

	logger.Info("[ChunkUploadManager] Initialized with Redis-backed storage (TTL: %v)", timeout)
	return &Manager{
		timeout: timeout,
	}
}

// SessionKey generates a unique key for a resource's file upload
func SessionKey(resourceID, fileName string) string {
	return resourceID + ":" + fileName
}

// redis keys helpers
func metaKey(base string) string {
	return "chunksess:" + base
}
func chunkKey(base string, index int32) string {
	return fmt.Sprintf("chunkdata:%s:%d", base, index)
}
func idxSetKey(base string) string {
	return "chunkidx:" + base
}

// GetOrCreateSession gets an existing session or creates a new one
func (m *Manager) GetOrCreateSession(resourceID string, upload *commonv1.ChunkedUploadPayload) (*Session, error) {
	key := SessionKey(resourceID, upload.FileName)
	ctx := context.Background()
	client := sharedredis.GetClient()
	if client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	mk := metaKey(key)
	data, err := client.Get(ctx, mk).Result()
	if err == nil {
		var sess Session
		if err := json.Unmarshal([]byte(data), &sess); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session metadata: %w", err)
		}
		// validate
		if sess.TotalChunks != upload.TotalChunks || sess.FileSize != upload.FileSize {
			return nil, fmt.Errorf("inconsistent chunk metadata for ongoing upload")
		}
		// update last activity and TTL
		sess.LastActivityAt = time.Now()
		b, _ := json.Marshal(&sess)
		client.Set(ctx, mk, b, m.timeout+5*time.Minute)
		return &sess, nil
	}

	// create new metadata
	sess := &Session{
		ResourceID:     resourceID,
		FileName:       upload.FileName,
		FileSize:       upload.FileSize,
		TotalChunks:    upload.TotalChunks,
		BytesReceived:  0,
		LastActivityAt: time.Now(),
	}
	b, _ := json.Marshal(sess)
	if err := client.Set(ctx, mk, b, m.timeout+5*time.Minute).Err(); err != nil {
		return nil, fmt.Errorf("failed to persist session metadata: %w", err)
	}
	return sess, nil
}

// StoreChunk stores a chunk of data for a session
func (m *Manager) StoreChunk(resourceID string, upload *commonv1.ChunkedUploadPayload, chunkIndex int32) (*Session, error) {
	sess, err := m.GetOrCreateSession(resourceID, upload)
	if err != nil {
		return nil, err
	}

	key := SessionKey(resourceID, upload.FileName)
	ctx := context.Background()
	client := sharedredis.GetClient()
	if client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	ck := chunkKey(key, chunkIndex)
	mk := metaKey(key)

	// Use WATCH + TxPipelined to atomically set chunk, add index, and update metadata
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = client.Watch(ctx, func(tx *redispkg.Tx) error {
			// Fetch existing metadata (if any)
			data, err := tx.Get(ctx, mk).Result()
			var meta Session
			if err != nil && err != redispkg.Nil {
				return err
			}
			if err == nil {
				if err := json.Unmarshal([]byte(data), &meta); err != nil {
					return err
				}
				// validate
				if meta.TotalChunks != upload.TotalChunks || meta.FileSize != upload.FileSize {
					return fmt.Errorf("inconsistent chunk metadata for ongoing upload")
				}
			} else {
				// no metadata exists; create baseline
				meta = Session{
					ResourceID:     resourceID,
					FileName:       upload.FileName,
					FileSize:       upload.FileSize,
					TotalChunks:    upload.TotalChunks,
					BytesReceived:  0,
					LastActivityAt: time.Now(),
				}
			}

			// Update bytes received and last activity
			meta.BytesReceived += int64(len(upload.ChunkData))
			meta.LastActivityAt = time.Now()
			b, _ := json.Marshal(&meta)

			// Execute transactional pipeline
			_, err = tx.TxPipelined(ctx, func(pipe redispkg.Pipeliner) error {
				pipe.Set(ctx, ck, upload.ChunkData, m.timeout+5*time.Minute)
				pipe.SAdd(ctx, idxSetKey(key), strconv.FormatInt(int64(chunkIndex), 10))
				pipe.Set(ctx, mk, b, m.timeout+5*time.Minute)
				pipe.Expire(ctx, idxSetKey(key), m.timeout+5*time.Minute)
				return nil
			})

			if err != nil {
				return err
			}

			// Success, return nil to break Watch loop
			return nil
		}, mk)

		if lastErr == nil {
			// log stored chunk
			logger.Debug("[ChunkUpload] Stored chunk %d for %s", chunkIndex, key)
			// return updated session (read back metadata)
			data, err := client.Get(ctx, mk).Result()
			if err == nil {
				var updated Session
				if err := json.Unmarshal([]byte(data), &updated); err == nil {
					return &updated, nil
				}
			}
			// fallback to returned sess modified locally
			sess.BytesReceived += int64(len(upload.ChunkData))
			sess.LastActivityAt = time.Now()
			return sess, nil
		}

		// If txn failed due to concurrent modification, retry a short time
		if lastErr == redispkg.TxFailedErr {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to store chunk transactionally: %w", lastErr)
	}
	return sess, nil
}

// AssembleChunks assembles all chunks into a single byte slice
func (m *Manager) AssembleChunks(resourceID, fileName string, totalChunks int32) ([]byte, error) {
	key := SessionKey(resourceID, fileName)
	ctx := context.Background()
	client := sharedredis.GetClient()
	if client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	// Ensure we actually have all chunk indices recorded. Retry briefly in case of ordering latency.
	var completeData []byte
	var idxCount int64
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		idxCount, err = client.SCard(ctx, idxSetKey(key)).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk index set: %w", err)
		}
		if int32(idxCount) != totalChunks {
			if attempt < 2 {
				// short wait for eventual consistency between writers
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("incomplete upload: have %d of %d chunks", idxCount, totalChunks)
		}
		break
	}

	for i := int32(0); i < totalChunks; i++ {
		ck := chunkKey(key, i)
		b, err := client.Get(ctx, ck).Bytes()
		if err != nil {
			logger.Error("[ChunkUpload] missing chunk %d for %s: %v", i, key, err)
			return nil, fmt.Errorf("missing chunk %d: %w", i, err)
		}
		completeData = append(completeData, b...)
		logger.Debug("[ChunkUpload] Retrieved chunk %d for %s (%d bytes)", i, key, len(b))
	}

	return completeData, nil
}

// IsComplete checks if all chunks have been received
func (m *Manager) IsComplete(resourceID, fileName string, totalChunks int32) bool {
	key := SessionKey(resourceID, fileName)
	ctx := context.Background()
	client := sharedredis.GetClient()
	if client == nil {
		return false
	}
	cnt, err := client.SCard(ctx, idxSetKey(key)).Result()
	if err != nil {
		return false
	}
	return int32(cnt) == totalChunks
}

// RemoveSession removes a session after successful upload
func (m *Manager) RemoveSession(resourceID, fileName string) {
	key := SessionKey(resourceID, fileName)
	ctx := context.Background()
	client := sharedredis.GetClient()
	if client == nil {
		return
	}

	// delete metadata, chunk keys and index set
	client.Del(ctx, metaKey(key))
	// remove index set and associated chunk keys
	idxs, err := client.SMembers(ctx, idxSetKey(key)).Result()
	if err == nil {
		for _, idStr := range idxs {
			if i, err := strconv.ParseInt(idStr, 10, 32); err == nil {
				client.Del(ctx, chunkKey(key, int32(i)))
			}
		}
	}
	client.Del(ctx, idxSetKey(key))
}

// ValidatePayload validates required fields in a chunk upload payload
func ValidatePayload(upload *commonv1.ChunkedUploadPayload) error {
	if upload == nil {
		return fmt.Errorf("upload payload is required")
	}
	if upload.FileName == "" {
		return fmt.Errorf("file_name is required")
	}
	if upload.FileSize <= 0 {
		return fmt.Errorf("file_size must be positive")
	}
	if upload.TotalChunks <= 0 {
		return fmt.Errorf("total_chunks must be positive")
	}
	if upload.ChunkIndex < 0 || upload.ChunkIndex >= upload.TotalChunks {
		return fmt.Errorf("chunk_index out of range")
	}
	return nil
}

// CreateUploadResponse creates a standard chunked upload response
func CreateUploadResponse(fileName string, bytesReceived int64, success bool, errMsg string) *commonv1.ChunkedUploadResponsePayload {
	resp := &commonv1.ChunkedUploadResponsePayload{
		Success:       success,
		FileName:      fileName,
		BytesReceived: bytesReceived,
	}

	if errMsg != "" {
		resp.Error = &errMsg
	}

	return resp
}
