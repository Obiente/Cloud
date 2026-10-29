package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

const maxFilePreviewBytes = 64 << 20
const maxDirectoryArchiveBytes = 512 << 20

// Docker's archive API also works with shell-less images and stopped containers.
// Keep the ls fast path for images that provide it, since archives include nested
// contents. Bound the fallback's transfer and lifetime instead of buffering it.
func (c *Client) listContainerArchive(ctx context.Context, id, directory string) ([]FileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := c.api.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{SourcePath: directory})
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}
	defer result.Content.Close()
	if !result.Stat.Mode.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}
	return readDirectoryArchive(result.Content, directory)
}

func readDirectoryArchive(content io.Reader, directory string) ([]FileInfo, error) {
	limited := &io.LimitedReader{R: content, N: maxDirectoryArchiveBytes + 1}
	reader := tar.NewReader(limited)
	root, err := reader.Next()
	if err != nil {
		return nil, fmt.Errorf("read directory metadata: %w", err)
	}
	if root.Typeflag != tar.TypeDir {
		return nil, fmt.Errorf("invalid directory archive")
	}
	prefix := strings.TrimSuffix(path.Clean(root.Name), "/")
	files := make([]FileInfo, 0)
	for {
		header, err := reader.Next()
		if limited.N <= 0 {
			return nil, fmt.Errorf("directory is too large to browse without container file tools")
		}
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read directory archive: %w", err)
		}
		name := path.Clean(header.Name)
		if prefix != "." && prefix != "" {
			if !strings.HasPrefix(name, prefix+"/") {
				return nil, fmt.Errorf("invalid archive entry %q under %q", header.Name, root.Name)
			}
			name = strings.TrimPrefix(name, prefix+"/")
		} else {
			name = strings.TrimPrefix(name, "./")
			if root.Name == "/" {
				name = strings.TrimPrefix(name, "/")
			}
		}
		if name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
			return nil, fmt.Errorf("invalid archive entry %q under %q", header.Name, root.Name)
		}
		if strings.Contains(name, "/") {
			continue
		}
		if len(files) >= 100000 {
			return nil, fmt.Errorf("directory contains too many entries")
		}
		info := header.FileInfo()
		owner, group := header.Uname, header.Gname
		if owner == "" {
			owner = strconv.Itoa(header.Uid)
		}
		if group == "" {
			group = strconv.Itoa(header.Gid)
		}
		files = append(files, FileInfo{Name: name, Path: path.Join(directory, name), IsDirectory: info.IsDir(), Size: header.Size, Permissions: info.Mode().String(), Mode: uint32(header.Mode), Owner: owner, Group: group, ModifiedAt: header.ModTime, IsSymlink: header.Typeflag == tar.TypeSymlink, SymlinkTarget: header.Linkname})
	}
}

func (c *Client) ContainerReadFile(ctx context.Context, id, filePath string) ([]byte, error) {
	// Archive export and extraction need the entire file. Their caller owns the
	// lifetime; preview-specific limits must not truncate these operations.
	return c.readContainerFile(ctx, id, filePath, 0)
}

func (c *Client) ContainerReadFilePreview(ctx context.Context, id, filePath string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return c.readContainerFile(ctx, id, filePath, maxFilePreviewBytes)
}

func (c *Client) readContainerFile(ctx context.Context, id, filePath string, maxBytes int64) ([]byte, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	filePath = path.Clean("/" + filePath)
	for range 16 {
		result, err := c.api.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{SourcePath: filePath})
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		if result.Stat.Mode&os.ModeSymlink != 0 {
			result.Content.Close()
			target := result.Stat.LinkTarget
			if !path.IsAbs(target) {
				target = path.Join(path.Dir(filePath), target)
			}
			filePath = path.Clean(target)
			continue
		}
		if !result.Stat.Mode.IsRegular() || (maxBytes > 0 && result.Stat.Size > maxBytes) {
			result.Content.Close()
			return nil, fmt.Errorf("file must be regular and within the preview size limit")
		}
		reader := tar.NewReader(result.Content)
		header, err := reader.Next()
		if err != nil {
			result.Content.Close()
			return nil, fmt.Errorf("read file metadata: %w", err)
		}
		if !header.FileInfo().Mode().IsRegular() || (maxBytes > 0 && header.Size > maxBytes) {
			result.Content.Close()
			return nil, fmt.Errorf("invalid file archive")
		}
		var body io.Reader = reader
		if maxBytes > 0 {
			body = io.LimitReader(reader, maxBytes+1)
		}
		data, err := io.ReadAll(body)
		result.Content.Close()
		if err != nil {
			return nil, err
		}
		if maxBytes > 0 && int64(len(data)) > maxBytes {
			return nil, fmt.Errorf("file is too large")
		}
		return data, nil
	}
	return nil, fmt.Errorf("too many symbolic links")
}

func (c *Client) ContainerStat(ctx context.Context, id, filePath string) (*FileInfo, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	result, err := c.api.ContainerStatPath(ctx, id, client.ContainerStatPathOptions{Path: filePath})
	if err != nil {
		return nil, err
	}
	s := result.Stat
	return &FileInfo{Name: path.Base(filePath), Path: filePath, Size: s.Size, IsDirectory: s.Mode.IsDir(), Mode: uint32(s.Mode.Perm()), Permissions: s.Mode.String(), ModifiedAt: s.Mtime, IsSymlink: s.Mode&os.ModeSymlink != 0, SymlinkTarget: s.LinkTarget}, nil
}

// Volume reads go through the already-authorized container, not a path on the
// API service's host. The mount may live on a different Docker daemon.
func mountedVolumePath(volume VolumeMount, requested string) (string, error) {
	if !path.IsAbs(volume.MountPoint) || strings.ContainsRune(requested, 0) {
		return "", fmt.Errorf("invalid volume path")
	}
	for _, part := range strings.Split(requested, "/") {
		if part == ".." {
			return "", fmt.Errorf("invalid volume path")
		}
	}
	return path.Join(volume.MountPoint, strings.TrimPrefix(path.Clean("/"+requested), "/")), nil
}

func (c *Client) ListMountedVolumeFiles(ctx context.Context, id string, volume VolumeMount, requested string) ([]FileInfo, error) {
	target, err := mountedVolumePath(volume, requested)
	if err != nil {
		return nil, err
	}
	files, err := c.ContainerListFiles(ctx, id, target)
	if err != nil {
		return nil, err
	}
	for i := range files {
		files[i].Path = path.Join("/", requested, files[i].Name)
	}
	return files, nil
}

func (c *Client) ReadMountedVolumeFile(ctx context.Context, id string, volume VolumeMount, requested string) ([]byte, *FileInfo, error) {
	target, err := mountedVolumePath(volume, requested)
	if err != nil {
		return nil, nil, err
	}
	data, err := c.ContainerReadFilePreview(ctx, id, target)
	if err != nil {
		return nil, nil, err
	}
	info, err := c.ContainerStat(ctx, id, target)
	if err != nil {
		return nil, nil, err
	}
	info.Path = path.Clean("/" + requested)
	return data, info, nil
}
