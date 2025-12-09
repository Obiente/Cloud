package gameservers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
)

// chunkUploadSessions tracks in-flight file uploads keyed by (gameServerId, fileName)
// to reassemble chunks and detect completion.
var (
	chunkUploadSessionsMux sync.RWMutex
	chunkUploadSessions    = make(map[string]*chunkUploadSession)
)

type chunkUploadSession struct {
	mu             sync.Mutex
	gameServerId   string
	fileName       string
	destPath       string
	volumeName     string
	fileSize       int64
	totalChunks    int32
	receivedChunks map[int32][]byte // chunk index -> data
	fileMode       string
	lastActivityAt time.Time
	tempFilePath   string // temporary file to assemble chunks
}

const (
	chunkUploadTimeout = 30 * time.Minute
	sessionCleanupTick = 5 * time.Minute
)

func init() {
	// Periodically clean up stale sessions
	go func() {
		ticker := time.NewTicker(sessionCleanupTick)
		defer ticker.Stop()
		for range ticker.C {
			cleanupStaleSessions()
		}
	}()
}

func cleanupStaleSessions() {
	chunkUploadSessionsMux.Lock()
	defer chunkUploadSessionsMux.Unlock()

	now := time.Now()
	for key, sess := range chunkUploadSessions {
		if now.Sub(sess.lastActivityAt) > chunkUploadTimeout {
			if sess.tempFilePath != "" {
				os.Remove(sess.tempFilePath)
			}
			delete(chunkUploadSessions, key)
		}
	}
}

func getChunkUploadSessionKey(gameServerId, fileName string) string {
	return gameServerId + ":" + fileName
}

func getOrCreateChunkSession(req *gameserversv1.ChunkUploadGameServerFilesRequest) *chunkUploadSession {
	key := getChunkUploadSessionKey(req.GameServerId, req.FileName)

	chunkUploadSessionsMux.Lock()
	defer chunkUploadSessionsMux.Unlock()

	if sess, ok := chunkUploadSessions[key]; ok {
		sess.mu.Lock()
		sess.lastActivityAt = time.Now()
		sess.mu.Unlock()
		return sess
	}

	// Create new session with temp file for assembly
	tempFile, _ := os.CreateTemp("", "chunk-upload-*")
	tempFilePath := ""
	if tempFile != nil {
		tempFilePath = tempFile.Name()
		tempFile.Close()
	}

	volumeName := ""
	if req.VolumeName != nil {
		volumeName = *req.VolumeName
	}
	fileMode := ""
	if req.FileMode != nil {
		fileMode = *req.FileMode
	}

	sess := &chunkUploadSession{
		gameServerId:   req.GameServerId,
		fileName:       req.FileName,
		destPath:       req.DestinationPath,
		volumeName:     volumeName,
		fileSize:       req.FileSize,
		totalChunks:    req.TotalChunks,
		receivedChunks: make(map[int32][]byte),
		fileMode:       fileMode,
		lastActivityAt: time.Now(),
		tempFilePath:   tempFilePath,
	}

	chunkUploadSessions[key] = sess
	return sess
}

// ChunkUploadGameServerFiles handles a single file chunk upload request.
// Multiple requests for the same file (different chunk_index) are reassembled in order.
// When all chunks are received, the file is uploaded to the target destination.
func (s *Service) ChunkUploadGameServerFiles(ctx context.Context, req *connect.Request[gameserversv1.ChunkUploadGameServerFilesRequest]) (*connect.Response[gameserversv1.ChunkUploadGameServerFilesResponse], error) {
	msg := req.Msg

	// Permission check
	if err := s.checkGameServerPermission(ctx, msg.GameServerId, "update"); err != nil {
		return nil, err
	}

	// Validate request
	if msg.GameServerId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if msg.FileName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("file_name is required"))
	}
	if msg.FileSize <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("file_size must be > 0"))
	}
	if msg.TotalChunks <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("total_chunks must be > 0"))
	}
	if msg.ChunkIndex < 0 || msg.ChunkIndex >= msg.TotalChunks {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid chunk_index: must be in range [0, %d)", msg.TotalChunks))
	}
	if len(msg.ChunkData) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("chunk_data is required"))
	}

	// Get or create session for this file
	sess := getOrCreateChunkSession(msg)

	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Check for duplicate chunk (idempotence)
	if _, exists := sess.receivedChunks[msg.ChunkIndex]; exists {
		// Return success (idempotent)
		bytesReceived := int64(0)
		for _, data := range sess.receivedChunks {
			bytesReceived += int64(len(data))
		}
		return connect.NewResponse(&gameserversv1.ChunkUploadGameServerFilesResponse{
			Success:       true,
			FileName:      msg.FileName,
			BytesReceived: bytesReceived,
		}), nil
	}

	// Write chunk to temp file at the correct offset
	if sess.tempFilePath != "" {
		f, err := os.OpenFile(sess.tempFilePath, os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			defer f.Close()
			offset := int64(msg.ChunkIndex) * (msg.FileSize / int64(msg.TotalChunks))
			if msg.ChunkIndex == msg.TotalChunks-1 {
				// Last chunk may be smaller
				offset = msg.FileSize - int64(len(msg.ChunkData))
			}
			if _, err := f.WriteAt(msg.ChunkData, offset); err != nil {
				log.Printf("Failed to write chunk to temp file: %v", err)
			}
		}
	}

	// Store chunk in memory for reference
	sess.receivedChunks[msg.ChunkIndex] = msg.ChunkData

	// Calculate total bytes received
	bytesReceived := int64(0)
	for _, data := range sess.receivedChunks {
		bytesReceived += int64(len(data))
	}

	// Check if all chunks received
	if int32(len(sess.receivedChunks)) == msg.TotalChunks {
		// All chunks received, upload the file to target
		defer func() {
			// Clean up session and temp file
			chunkUploadSessionsMux.Lock()
			defer chunkUploadSessionsMux.Unlock()
			key := getChunkUploadSessionKey(msg.GameServerId, msg.FileName)
			if sess.tempFilePath != "" {
				os.Remove(sess.tempFilePath)
			}
			delete(chunkUploadSessions, key)
		}()

		// Assemble chunks in order and upload
		if err := s.uploadAssembledFile(ctx, msg); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload file: %w", err))
		}
	}

	return connect.NewResponse(&gameserversv1.ChunkUploadGameServerFilesResponse{
		Success:       true,
		FileName:      msg.FileName,
		BytesReceived: bytesReceived,
	}), nil
}

// uploadAssembledFile assembles all chunks from the session and uploads the complete file.
func (s *Service) uploadAssembledFile(ctx context.Context, req *gameserversv1.ChunkUploadGameServerFilesRequest) error {
	// Get the session to read all chunks
	key := getChunkUploadSessionKey(req.GameServerId, req.FileName)

	chunkUploadSessionsMux.RLock()
	sess, ok := chunkUploadSessions[key]
	chunkUploadSessionsMux.RUnlock()

	if !ok || len(sess.receivedChunks) == 0 {
		return fmt.Errorf("no chunks found for upload")
	}

	// Create docker client
	dcli, err := docker.New()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer dcli.Close()

	// Find container
	containerID, err := s.findContainerForGameServer(ctx, req.GameServerId, dcli)
	if err != nil {
		return err
	}

	// Assemble file from chunks (in order)
	var completeData []byte
	for i := int32(0); i < req.TotalChunks; i++ {
		if chunk, ok := sess.receivedChunks[i]; ok {
			completeData = append(completeData, chunk...)
		}
	}

	if int64(len(completeData)) != req.FileSize {
		return fmt.Errorf("assembled file size mismatch: expected %d, got %d", req.FileSize, len(completeData))
	}

	// Upload single file using the docker client
	files := map[string][]byte{
		req.FileName: completeData,
	}

	// Log target info for debugging
	log.Printf("uploadAssembledFile: container=%s, destPath=%q, volumeName=%q, file=%s", containerID, req.DestinationPath, req.GetVolumeName(), req.FileName)

	// If a volume name was provided, upload directly to the host volume path
	volumeName := ""
	if req.VolumeName != nil {
		volumeName = *req.VolumeName
	}

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to list container volumes: %w", err)
		}

		var targetVolume *docker.VolumeMount
		for _, v := range volumes {
			if v.Name == volumeName {
				targetVolume = &v
				break
			}
		}
		if targetVolume == nil {
			return fmt.Errorf("volume not found: %s", volumeName)
		}

		// Combine destination path with file names to build paths inside the volume
		uploadFiles := make(map[string][]byte)
		destPath := req.DestinationPath
		if destPath == "" {
			destPath = "/"
		}
		for fname, content := range files {
			fullPath := filepath.Join(destPath, fname)
			uploadFiles[fullPath] = content
		}

		if err := dcli.UploadVolumeFiles(targetVolume.Source, uploadFiles); err != nil {
			return fmt.Errorf("failed to upload to volume: %w", err)
		}
	} else {
		if err := dcli.ContainerUploadFiles(ctx, containerID, req.DestinationPath, files); err != nil {
			return fmt.Errorf("failed to upload to container: %w", err)
		}
	}

	return nil
}
