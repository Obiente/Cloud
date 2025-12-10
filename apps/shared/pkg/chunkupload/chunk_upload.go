package chunkupload

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
)

// Session tracks in-flight file chunks for a resource
type Session struct {
	mu             sync.Mutex
	ResourceID     string
	FileName       string
	FileSize       int64
	TotalChunks    int32
	ReceivedChunks map[int32][]byte // chunk index -> data
	BytesReceived  int64
	LastActivityAt time.Time
	TempFilePath   string // optional temporary file for assembly
}

// Manager manages chunk upload sessions
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	timeout  time.Duration
	ticker   *time.Ticker
	done     chan struct{}
}

// NewManager creates a new chunk upload session manager
func NewManager(timeout time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[string]*Session),
		timeout:  timeout,
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine
	m.ticker = time.NewTicker(5 * time.Minute)
	go m.cleanupLoop()

	return m
}

// Stop stops the cleanup goroutine
func (m *Manager) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	close(m.done)
}

// cleanupLoop periodically removes stale sessions
func (m *Manager) cleanupLoop() {
	for {
		select {
		case <-m.ticker.C:
			m.cleanupStaleSessions()
		case <-m.done:
			return
		}
	}
}

// cleanupStaleSessions removes sessions that have timed out
func (m *Manager) cleanupStaleSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, sess := range m.sessions {
		if now.Sub(sess.LastActivityAt) > m.timeout {
			if sess.TempFilePath != "" {
				os.Remove(sess.TempFilePath)
			}
			log.Printf("[ChunkUploadManager] Cleaned up stale session: %s", key)
			delete(m.sessions, key)
		}
	}
}

// SessionKey generates a unique key for a resource's file upload
func SessionKey(resourceID, fileName string) string {
	return resourceID + ":" + fileName
}

// GetOrCreateSession gets an existing session or creates a new one
func (m *Manager) GetOrCreateSession(resourceID string, upload *commonv1.ChunkedUploadPayload) (*Session, error) {
	key := SessionKey(resourceID, upload.FileName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[key]; ok {
		// Validate consistency
		if sess.TotalChunks != upload.TotalChunks || sess.FileSize != upload.FileSize {
			return nil, fmt.Errorf("inconsistent chunk metadata for ongoing upload")
		}
		sess.mu.Lock()
		sess.LastActivityAt = time.Now()
		sess.mu.Unlock()
		return sess, nil
	}

	// Create new session
	sess := &Session{
		ResourceID:     resourceID,
		FileName:       upload.FileName,
		FileSize:       upload.FileSize,
		TotalChunks:    upload.TotalChunks,
		ReceivedChunks: make(map[int32][]byte),
		BytesReceived:  0,
		LastActivityAt: time.Now(),
	}

	m.sessions[key] = sess
	return sess, nil
}

// StoreChunk stores a chunk of data for a session
func (m *Manager) StoreChunk(resourceID string, upload *commonv1.ChunkedUploadPayload, chunkIndex int32) (*Session, error) {
	sess, err := m.GetOrCreateSession(resourceID, upload)
	if err != nil {
		return nil, err
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	sess.ReceivedChunks[chunkIndex] = upload.ChunkData
	sess.BytesReceived += int64(len(upload.ChunkData))

	return sess, nil
}

// AssembleChunks assembles all chunks into a single byte slice
func (m *Manager) AssembleChunks(resourceID, fileName string, totalChunks int32) ([]byte, error) {
	key := SessionKey(resourceID, fileName)

	m.mu.RLock()
	sess, ok := m.sessions[key]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no session found for resource")
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Assemble chunks in order
	var completeData []byte
	for i := int32(0); i < totalChunks; i++ {
		if chunk, ok := sess.ReceivedChunks[i]; ok {
			completeData = append(completeData, chunk...)
		}
	}

	return completeData, nil
}

// IsComplete checks if all chunks have been received
func (m *Manager) IsComplete(resourceID, fileName string, totalChunks int32) bool {
	key := SessionKey(resourceID, fileName)

	m.mu.RLock()
	sess, ok := m.sessions[key]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	return int32(len(sess.ReceivedChunks)) == totalChunks
}

// RemoveSession removes a session after successful upload
func (m *Manager) RemoveSession(resourceID, fileName string) {
	key := SessionKey(resourceID, fileName)

	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[key]; ok {
		if sess.TempFilePath != "" {
			os.Remove(sess.TempFilePath)
		}
		delete(m.sessions, key)
	}
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
