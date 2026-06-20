package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pkgsftp "github.com/pkg/sftp"
)

type sftpHandler struct {
	session *Session
	root    string
}

func newSFTPHandler(session *Session) *sftpHandler {
	return &sftpHandler{
		session: session,
		root:    filepath.Clean(session.RootPath),
	}
}

func (h *sftpHandler) Fileread(r *pkgsftp.Request) (io.ReaderAt, error) {
	if !hasPermission(h.session.Permissions, PermissionRead) {
		return nil, fmt.Errorf("read permission denied")
	}
	resolved, err := h.resolvePath(r.Filepath)
	if err != nil {
		return nil, err
	}
	return os.Open(resolved)
}

func (h *sftpHandler) Filewrite(r *pkgsftp.Request) (io.WriterAt, error) {
	if !hasPermission(h.session.Permissions, PermissionWrite) {
		return nil, fmt.Errorf("write permission denied")
	}
	resolved, err := h.resolvePath(r.Filepath)
	if err != nil {
		return nil, err
	}
	if err := h.ensureParentInsideRoot(resolved); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0750); err != nil {
		return nil, err
	}
	return os.OpenFile(resolved, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
}

func (h *sftpHandler) Filecmd(r *pkgsftp.Request) error {
	switch r.Method {
	case "Setstat":
		if !hasPermission(h.session.Permissions, PermissionWrite) {
			return fmt.Errorf("write permission denied")
		}
		return nil
	case "Rename":
		if !hasPermission(h.session.Permissions, PermissionWrite) {
			return fmt.Errorf("write permission denied")
		}
		source, err := h.resolvePath(r.Filepath)
		if err != nil {
			return err
		}
		target, err := h.resolvePath(r.Target)
		if err != nil {
			return err
		}
		if err := h.ensureParentInsideRoot(target); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
			return err
		}
		return os.Rename(source, target)
	case "Remove":
		if !hasPermission(h.session.Permissions, PermissionWrite) {
			return fmt.Errorf("write permission denied")
		}
		resolved, err := h.resolvePath(r.Filepath)
		if err != nil {
			return err
		}
		return os.Remove(resolved)
	case "Rmdir":
		if !hasPermission(h.session.Permissions, PermissionWrite) {
			return fmt.Errorf("write permission denied")
		}
		resolved, err := h.resolvePath(r.Filepath)
		if err != nil {
			return err
		}
		return os.Remove(resolved)
	case "Mkdir":
		if !hasPermission(h.session.Permissions, PermissionWrite) {
			return fmt.Errorf("write permission denied")
		}
		resolved, err := h.resolvePath(r.Filepath)
		if err != nil {
			return err
		}
		if err := h.ensureParentInsideRoot(resolved); err != nil {
			return err
		}
		return os.MkdirAll(resolved, 0750)
	case "Link", "Symlink":
		return fmt.Errorf("symlinks are not supported")
	default:
		return pkgsftp.ErrSSHFxOpUnsupported
	}
}

func (h *sftpHandler) Filelist(r *pkgsftp.Request) (pkgsftp.ListerAt, error) {
	if !hasPermission(h.session.Permissions, PermissionRead) {
		return nil, fmt.Errorf("read permission denied")
	}
	resolved, err := h.resolvePath(r.Filepath)
	if err != nil {
		return nil, err
	}

	switch r.Method {
	case "List":
		entries, err := os.ReadDir(resolved)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err == nil {
				infos = append(infos, info)
			}
		}
		return listerAt(infos), nil
	case "Stat":
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, err
		}
		return listerAt([]os.FileInfo{info}), nil
	case "Readlink":
		return nil, fmt.Errorf("symlinks are not supported")
	default:
		return nil, pkgsftp.ErrSSHFxOpUnsupported
	}
}

func (h *sftpHandler) resolvePath(requestPath string) (string, error) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(requestPath, "\\", "/"))
	cleaned = strings.Trim(cleaned, "\x00\r\n")
	cleaned = filepath.ToSlash(filepath.Clean("/" + cleaned))
	relative := strings.TrimPrefix(cleaned, "/")
	if relative == "." {
		relative = ""
	}

	candidate := filepath.Join(h.root, filepath.FromSlash(relative))
	if !isWithinRoot(h.root, candidate) {
		return "", fmt.Errorf("path escapes transfer root")
	}
	if err := h.ensureExistingPathInsideRoot(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func (h *sftpHandler) ensureParentInsideRoot(path string) error {
	return h.ensureExistingPathInsideRoot(filepath.Dir(path))
}

func (h *sftpHandler) ensureExistingPathInsideRoot(candidate string) error {
	root, err := filepath.Abs(h.root)
	if err != nil {
		return err
	}
	current := root
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return err
		}
		if !isWithinRoot(root, resolved) {
			return fmt.Errorf("path escapes transfer root")
		}
	}
	return nil
}

func isWithinRoot(root, candidate string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}

type listerAt []os.FileInfo

func (l listerAt) ListAt(out []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(out, l[offset:])
	if n < len(out) {
		return n, io.EOF
	}
	return n, nil
}
