package gameservers

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
)

// HandleUploadFile handles a single-file multipart upload and streams it directly into Docker as a tar entry.
// Expected query params:
// - gameServerId (required)
// - destPath (required)
// - volumeName (optional)
// Client must POST multipart/form-data with a single file field named "file".
func (s *Service) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Authenticate HTTP request and set user in context
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var err error
	ctx, _, err = auth.AuthenticateAndSetContext(ctx, authHeader)
	if err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	gameServerId := r.URL.Query().Get("gameServerId")
	if gameServerId == "" {
		http.Error(w, "gameServerId is required", http.StatusBadRequest)
		return
	}

	destPath := r.URL.Query().Get("destPath")
	if destPath == "" {
		http.Error(w, "destPath is required", http.StatusBadRequest)
		return
	}

	volumeName := r.URL.Query().Get("volumeName")

	// Permission check
	if err := s.checkGameServerPermission(ctx, gameServerId, "update"); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Multipart reader
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "invalid multipart request", http.StatusBadRequest)
		return
	}

	// We'll accept a single file per request. The client should include fileName and fileSize as query params if available.
	fileName := r.URL.Query().Get("fileName")
	fileSizeStr := r.URL.Query().Get("fileSize")
	var fileSize int64 = -1
	if fileSizeStr != "" {
		if v, err := strconv.ParseInt(fileSizeStr, 10, 64); err == nil {
			fileSize = v
		}
	}

	// Create a pipe: we'll write a tar into pw while reading the multipart file, and the pipe reader will be consumed by docker client.
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	// Start goroutine to stream tar into docker
	done := make(chan error, 1)
	go func() {
		defer pr.Close()
		// Create docker client
		dcli, derr := docker.New()
		if derr != nil {
			done <- fmt.Errorf("docker client: %w", derr)
			return
		}
		defer dcli.Close()

		// Find container
		containerID, cerr := s.findContainerForGameServer(ctx, gameServerId, dcli)
		if cerr != nil {
			done <- cerr
			return
		}

		if volumeName != "" {
			// Find host path for volume
			volumes, err := dcli.GetContainerVolumes(ctx, containerID)
			if err != nil {
				done <- err
				return
			}
			var target string
			for _, v := range volumes {
				if v.Name == volumeName {
					target = v.Source
					break
				}
			}
			if target == "" {
				done <- fmt.Errorf("volume not found: %s", volumeName)
				return
			}

			// Stream-extract tar directly into volume
			if err := dcli.UploadVolumeFromTar(target, pr); err != nil {
				done <- err
				return
			}
		} else {
			// Upload tar to container path using streaming CopyToContainer
			if err := dcli.ContainerUploadFromTar(ctx, containerID, destPath, pr); err != nil {
				done <- err
				return
			}
		}

		done <- nil
	}()

	// Now read multipart parts and write tar headers + file contents
	var wroteAny bool
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			pw.CloseWithError(err)
			http.Error(w, "error reading multipart", http.StatusBadRequest)
			return
		}

		if part.FormName() != "file" {
			// ignore other fields
			io.Copy(io.Discard, part)
			continue
		}

		// Determine filename
		fname := fileName
		if fname == "" {
			fname = part.FileName()
		}
		if fname == "" {
			pw.CloseWithError(fmt.Errorf("file name required"))
			http.Error(w, "file name required", http.StatusBadRequest)
			return
		}

		// If fileSize unknown, try to read Content-Length header from part
		size := fileSize
		if size <= 0 {
			if v := part.Header.Get("Content-Length"); v != "" {
				if vs, err := strconv.ParseInt(v, 10, 64); err == nil {
					size = vs
				}
			}
		}

		// Write tar header
		hdr := &tar.Header{
			Name: fname,
			Mode: 0644,
			Size: size,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			pw.CloseWithError(err)
			http.Error(w, "error writing tar header", http.StatusInternalServerError)
			return
		}

		// Stream copy file content into tar writer
		if _, err := io.Copy(tw, part); err != nil {
			pw.CloseWithError(err)
			http.Error(w, "error streaming file", http.StatusInternalServerError)
			return
		}

		wroteAny = true
		// Close this part and continue (we accept only single file but will gracefully handle extras)
		part.Close()
	}

	// Close tar writer to signal EOF to docker copy
	if wroteAny {
		if err := tw.Close(); err != nil {
			pw.CloseWithError(err)
			http.Error(w, "error finalizing tar", http.StatusInternalServerError)
			return
		}
	}
	pw.Close()

	// Wait for docker goroutine to finish
	if err := <-done; err != nil {
		http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}
