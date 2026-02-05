package sftp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/pkg/sftp"
)

// userHandler implements sftp.Handlers with permission checking
type userHandler struct {
	basePath    string
	orgID       string
	resourceType string
	resourceID   string
	userID      string
	permissions []Permission
	auditLogger AuditLogger
}

// newUserHandler creates a new user-specific SFTP handler
func newUserHandler(basePath, orgID, resourceType, resourceID, userID string, permissions []Permission, auditLogger AuditLogger) *userHandler {
	return &userHandler{
		basePath:    basePath,
		orgID:       orgID,
		resourceType: resourceType,
		resourceID:   resourceID,
		userID:      userID,
		permissions: permissions,
		auditLogger: auditLogger,
	}
}

// hasPermission checks if user has a specific permission
func (h *userHandler) hasPermission(perm Permission) bool {
	for _, p := range h.permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// resolvePath converts a relative path to absolute path within user's directory
func (h *userHandler) resolvePath(reqPath string) (string, error) {
	// Build scoped directory: org / resource / user
	resourceDir := filepath.Join(h.basePath, h.orgID)
	if h.resourceType != "" && h.resourceID != "" {
		resourceDir = filepath.Join(resourceDir, h.resourceType, h.resourceID)
	}

	userDir := filepath.Join(resourceDir, h.userID)
	
	// Clean the requested path
	cleanPath := filepath.Clean(reqPath)
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "/"
	}
	
	// Join with user directory
	absPath := filepath.Join(userDir, cleanPath)
	
	// Ensure path is within user directory (prevent directory traversal)
	if !strings.HasPrefix(absPath, userDir) {
		return "", fmt.Errorf("access denied: path outside user directory")
	}
	
	return absPath, nil
}

// logAudit logs an operation to the audit log
func (h *userHandler) logAudit(operation, path string, success bool, errMsg string, bytesWritten, bytesRead int64) {
	if h.auditLogger == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry := AuditEntry{
		UserID:       h.userID,
		OrgID:        h.orgID,
		Operation:    operation,
		Path:         path,
		Success:      success,
		ErrorMessage: errMsg,
		BytesWritten: bytesWritten,
		BytesRead:    bytesRead,
	}

	if err := h.auditLogger.LogOperation(ctx, entry); err != nil {
		logger.Error("[SFTP] Failed to log audit entry: %v", err)
	}
}

// Fileread implements sftp.Handlers
func (h *userHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	if !h.hasPermission(PermissionRead) {
		err := fmt.Errorf("read permission denied")
		h.logAudit("download", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	absPath, err := h.resolvePath(r.Filepath)
	if err != nil {
		h.logAudit("download", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		h.logAudit("download", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	// Get file size for audit log
	stat, _ := file.Stat()
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}

	h.logAudit("download", r.Filepath, true, "", 0, size)
	logger.Debug("[SFTP] User %s downloading: %s", h.userID, r.Filepath)

	return file, nil
}

// Filewrite implements sftp.Handlers
func (h *userHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	if !h.hasPermission(PermissionWrite) {
		err := fmt.Errorf("write permission denied")
		h.logAudit("upload", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	absPath, err := h.resolvePath(r.Filepath)
	if err != nil {
		h.logAudit("upload", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		h.logAudit("upload", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		h.logAudit("upload", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	logger.Debug("[SFTP] User %s uploading: %s", h.userID, r.Filepath)

	// Wrap file to track bytes written
	return &auditWriter{
		file:        file,
		handler:     h,
		path:        r.Filepath,
		bytesWritten: 0,
	}, nil
}

// Filecmd implements sftp.Handlers
func (h *userHandler) Filecmd(r *sftp.Request) error {
	absPath, err := h.resolvePath(r.Filepath)
	if err != nil {
		return err
	}

	switch r.Method {
	case "Setstat":
		// Allow setstat for both read and write (it's for chmod, chown, etc)
		if !h.hasPermission(PermissionWrite) {
			err := fmt.Errorf("write permission denied")
			h.logAudit("setstat", r.Filepath, false, err.Error(), 0, 0)
			return err
		}
		return nil // We don't actually change permissions

	case "Rename":
		if !h.hasPermission(PermissionWrite) {
			err := fmt.Errorf("write permission denied")
			h.logAudit("rename", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		absTarget, err := h.resolvePath(r.Target)
		if err != nil {
			h.logAudit("rename", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		// Ensure target parent directory exists
		if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
			h.logAudit("rename", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		if err := os.Rename(absPath, absTarget); err != nil {
			h.logAudit("rename", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		h.logAudit("rename", fmt.Sprintf("%s -> %s", r.Filepath, r.Target), true, "", 0, 0)
		logger.Debug("[SFTP] User %s renamed: %s -> %s", h.userID, r.Filepath, r.Target)
		return nil

	case "Remove":
		if !h.hasPermission(PermissionWrite) {
			err := fmt.Errorf("write permission denied")
			h.logAudit("delete", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		if err := os.Remove(absPath); err != nil {
			h.logAudit("delete", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		h.logAudit("delete", r.Filepath, true, "", 0, 0)
		logger.Debug("[SFTP] User %s deleted: %s", h.userID, r.Filepath)
		return nil

	case "Rmdir":
		if !h.hasPermission(PermissionWrite) {
			err := fmt.Errorf("write permission denied")
			h.logAudit("delete", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		if err := os.Remove(absPath); err != nil {
			h.logAudit("delete", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		h.logAudit("delete", r.Filepath, true, "", 0, 0)
		logger.Debug("[SFTP] User %s removed directory: %s", h.userID, r.Filepath)
		return nil

	case "Mkdir":
		if !h.hasPermission(PermissionWrite) {
			err := fmt.Errorf("write permission denied")
			h.logAudit("mkdir", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		if err := os.MkdirAll(absPath, 0755); err != nil {
			h.logAudit("mkdir", r.Filepath, false, err.Error(), 0, 0)
			return err
		}

		h.logAudit("mkdir", r.Filepath, true, "", 0, 0)
		logger.Debug("[SFTP] User %s created directory: %s", h.userID, r.Filepath)
		return nil

	case "Link", "Symlink":
		// Don't allow symlinks for security
		err := fmt.Errorf("symlinks not allowed")
		h.logAudit("symlink", r.Filepath, false, err.Error(), 0, 0)
		return err

	default:
		return sftp.ErrSSHFxOpUnsupported
	}
}

// Filelist implements sftp.Handlers
func (h *userHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	if !h.hasPermission(PermissionRead) {
		err := fmt.Errorf("read permission denied")
		h.logAudit("list", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	absPath, err := h.resolvePath(r.Filepath)
	if err != nil {
		h.logAudit("list", r.Filepath, false, err.Error(), 0, 0)
		return nil, err
	}

	switch r.Method {
	case "List":
		files, err := os.ReadDir(absPath)
		if err != nil {
			h.logAudit("list", r.Filepath, false, err.Error(), 0, 0)
			return nil, err
		}

		fileInfos := make([]os.FileInfo, 0, len(files))
		for _, f := range files {
			info, err := f.Info()
			if err == nil {
				fileInfos = append(fileInfos, info)
			}
		}

		h.logAudit("list", r.Filepath, true, "", 0, 0)
		logger.Debug("[SFTP] User %s listed: %s (%d files)", h.userID, r.Filepath, len(fileInfos))

		return listerat(fileInfos), nil

	case "Stat":
		stat, err := os.Stat(absPath)
		if err != nil {
			h.logAudit("stat", r.Filepath, false, err.Error(), 0, 0)
			return nil, err
		}

		h.logAudit("stat", r.Filepath, true, "", 0, 0)
		return listerat([]os.FileInfo{stat}), nil

	case "Readlink":
		// Don't allow readlink for security
		err := fmt.Errorf("symlinks not allowed")
		h.logAudit("readlink", r.Filepath, false, err.Error(), 0, 0)
		return nil, err

	default:
		return nil, sftp.ErrSSHFxOpUnsupported
	}
}

// listerat is a simple implementation of ListerAt
type listerat []os.FileInfo

func (l listerat) ListAt(f []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(f, l[offset:])
	if n < len(f) {
		return n, io.EOF
	}

	return n, nil
}

// auditWriter wraps a file writer to track bytes written
type auditWriter struct {
	file         *os.File
	handler      *userHandler
	path         string
	bytesWritten int64
}

func (w *auditWriter) WriteAt(p []byte, offset int64) (int, error) {
	n, err := w.file.WriteAt(p, offset)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *auditWriter) Close() error {
	err := w.file.Close()
	
	// Log audit entry on close
	if err != nil {
		w.handler.logAudit("upload", w.path, false, err.Error(), w.bytesWritten, 0)
	} else {
		w.handler.logAudit("upload", w.path, true, "", w.bytesWritten, 0)
	}
	
	return err
}
