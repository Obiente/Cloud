package gameservers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/chunkupload"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
)

// ChunkUploadManager handles chunk upload session management
var (
	chunkUploadManager     *chunkupload.Manager
	chunkUploadManagerOnce sync.Once
)

func getChunkUploadManager() *chunkupload.Manager {
	chunkUploadManagerOnce.Do(func() {
		chunkUploadManager = chunkupload.NewManager(30 * time.Minute)
	})
	return chunkUploadManager
}

func init() {
	// Cleanup is handled by the manager's goroutine
}

// ChunkUploadGameServerFiles handles a single file chunk upload request.
// Multiple requests for the same file (different chunk_index) are reassembled in order.
// When all chunks are received, the file is uploaded to the target destination.
func (s *Service) ChunkUploadGameServerFiles(ctx context.Context, req *connect.Request[gameserversv1.ChunkUploadGameServerFilesRequest]) (*connect.Response[gameserversv1.ChunkUploadGameServerFilesResponse], error) {
	msg := req.Msg
	gameServerId := msg.GameServerId
	upload := msg.Upload

	// Permission check
	if err := s.checkGameServerPermission(ctx, gameServerId, "update"); err != nil {
		return nil, err
	}

	// Validate request
	if gameServerId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	// Validate upload payload using shared validator
	if err := chunkupload.ValidatePayload(upload); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Get or create session
	sess, err := getChunkUploadManager().GetOrCreateSession(gameServerId, upload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Store the chunk (use returned session so BytesReceived is up-to-date)
	sess, err = getChunkUploadManager().StoreChunk(gameServerId, upload, upload.ChunkIndex)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	bytesReceived := sess.BytesReceived

	// Check if this is the last chunk
	isLastChunk := upload.ChunkIndex == upload.TotalChunks-1
	allChunksReceived := getChunkUploadManager().IsComplete(gameServerId, upload.FileName, upload.TotalChunks)

	resp := &gameserversv1.ChunkUploadGameServerFilesResponse{
		Result: &commonv1.ChunkedUploadResponsePayload{
			Success:       true,
			FileName:      upload.FileName,
			BytesReceived: bytesReceived,
		},
	}

	// If this is the last chunk and we have all chunks, assemble and upload
	if isLastChunk && allChunksReceived {
		// Attempt to upload the assembled file
		if err := s.uploadAssembledFile(ctx, gameServerId, upload); err != nil {
			resp.Result.Success = false
			errorMsg := fmt.Sprintf("failed to upload assembled file: %v", err)
			resp.Result.Error = &errorMsg

			// Emit a single audit log for the failed upload (async)
			go func() {
				userID := "system"
				if u, _ := auth.GetUserFromContext(ctx); u != nil && u.Id != "" {
					userID = u.Id
				}
				ip := "unknown"
				if req != nil {
					if h := req.Header().Get("X-Forwarded-For"); h != "" {
						ip = h
					} else if h := req.Header().Get("X-Real-IP"); h != "" {
						ip = h
					}
				}
				action := "UploadGameServerFile"
				service := "GameServerService"
				rt := "game_server"
				logger.Debug("[ChunkUpload] Emitting audit log (failure): gameServer=%s file=%s", gameServerId, upload.FileName)
				// Marshal request data for audit log
				requestDataBytes, marshalErr := json.Marshal(msg)
				requestData := "{}"
				if marshalErr != nil {
					logger.Error("[ChunkUpload] Failed to marshal request data: %v", marshalErr)
				} else {
					requestData = string(requestDataBytes)
				}
				if err := middleware.CreateAuditLog(context.Background(), middleware.AuditEntry{
					UserID:         userID,
					OrganizationID: nil,
					Action:         action,
					Service:        service,
					ResourceType:   &rt,
					ResourceID:     &gameServerId,
					IPAddress:      ip,
					UserAgent:      req.Header().Get("User-Agent"),
					RequestData:    requestData,
					ResponseStatus: 500,
					ErrorMessage:   &errorMsg,
					DurationMs:     0,
				}); err != nil {
					logger.Error("[ChunkUpload] CreateAuditLog error (failure): %v", err)
				}
			}()

			return connect.NewResponse(resp), nil
		}

		// Emit a single audit log for the successful upload (async)
		go func() {
			userID := "system"
			if u, _ := auth.GetUserFromContext(ctx); u != nil && u.Id != "" {
				userID = u.Id
			}
			ip := "unknown"
			if req != nil {
				if h := req.Header().Get("X-Forwarded-For"); h != "" {
					ip = h
				} else if h := req.Header().Get("X-Real-IP"); h != "" {
					ip = h
				}
			}
			action := "UploadGameServerFile"
			service := "GameServerService"
			rt := "game_server"
			logger.Debug("[ChunkUpload] Emitting audit log (success): gameServer=%s file=%s", gameServerId, upload.FileName)
			if err := middleware.CreateAuditLog(context.Background(), middleware.AuditEntry{
				UserID:         userID,
				OrganizationID: nil,
				Action:         action,
				Service:        service,
				ResourceType:   &rt,
				ResourceID:     &gameServerId,
				IPAddress:      ip,
				UserAgent:      req.Header().Get("User-Agent"),
				RequestData:    "{}",
				ResponseStatus: 200,
				ErrorMessage:   nil,
				DurationMs:     0,
			}); err != nil {
				logger.Error("[ChunkUpload] CreateAuditLog error (success): %v", err)
			}
		}()

		// Clean up the session after successful upload
		getChunkUploadManager().RemoveSession(gameServerId, upload.FileName)
	}

	return connect.NewResponse(resp), nil
}

// uploadAssembledFile assembles all chunks from the session and uploads the complete file.
func (s *Service) uploadAssembledFile(ctx context.Context, gameServerId string, upload *commonv1.ChunkedUploadPayload) error {
	// Assemble file from chunks using shared manager
	completeData, err := getChunkUploadManager().AssembleChunks(gameServerId, upload.FileName, upload.TotalChunks)
	if err != nil {
		return err
	}

	if int64(len(completeData)) != int64(upload.FileSize) {
		return fmt.Errorf("assembled file size mismatch: expected %d, got %d", upload.FileSize, len(completeData))
	}

	// Create docker client
	dcli, err := docker.New()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer dcli.Close()

	// Find container
	containerID, err := s.findContainerForGameServer(ctx, gameServerId, dcli)
	if err != nil {
		return err
	}

	// Prepare files map
	files := map[string][]byte{
		upload.FileName: completeData,
	}

	// Log target info for debugging
	volumeNameStr := ""
	if upload.VolumeName != nil {
		volumeNameStr = *upload.VolumeName
	}
	log.Printf("uploadAssembledFile: container=%s, destPath=%q, volumeName=%q, file=%s", containerID, upload.DestinationPath, volumeNameStr, upload.FileName)

	// If a volume name was provided, upload directly to the host volume path
	volumeName := volumeNameStr

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
		destPath := upload.DestinationPath
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
		if err := dcli.ContainerUploadFiles(ctx, containerID, upload.DestinationPath, files); err != nil {
			return fmt.Errorf("failed to upload to container: %w", err)
		}
	}

	return nil
}
