package backups

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var ErrSizeLimit = errors.New("backup exceeds configured size limit")

// The root must already exist on the mounted shared volume. Never create it:
// doing so would disguise a missing mount as working persistent storage.
type Store struct {
	Directory string
	MaxBytes  int64
}

func (s Store) Open() (*os.Root, error) {
	if s.Directory == "" || !filepath.IsAbs(s.Directory) {
		return nil, fmt.Errorf("shared backup volume is not configured")
	}
	root, err := os.OpenRoot(s.Directory)
	if err != nil {
		return nil, fmt.Errorf("shared backup volume is unavailable: %w", err)
	}
	marker, err := root.ReadFile(".obiente-backup-volume")
	if err != nil || string(marker) != "obiente-cloud-backups-v1\n" {
		_ = root.Close()
		return nil, fmt.Errorf("shared backup volume marker is missing or invalid")
	}
	return root, nil
}

func fileName(id string) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid backup identifier")
	}
	return id + ".sql.gz", nil
}

func (s Store) Write(ctx context.Context, id string, produce func(io.Writer) error) (size int64, checksum string, err error) {
	name, err := fileName(id)
	if err != nil {
		return 0, "", err
	}
	root, err := s.Open()
	if err != nil {
		return 0, "", err
	}
	defer root.Close()
	tmp := name + ".partial"
	// A pending record owns its temporary file across process restarts.
	if err = root.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, "", err
	}
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = root.Remove(tmp)
		}
	}()
	hash := sha256.New()
	zipped := gzip.NewWriter(io.MultiWriter(f, hash))
	limit := s.MaxBytes
	if limit <= 0 {
		limit = 1 << 30
	}
	writer := &limitedWriter{writer: zipped, remaining: limit, ctx: ctx}
	if err = produce(writer); err != nil {
		return 0, "", err
	}
	if writer.remaining == limit {
		return 0, "", fmt.Errorf("empty backup output")
	}
	if err = ctx.Err(); err != nil {
		return 0, "", err
	}
	if err = zipped.Close(); err != nil {
		return 0, "", err
	}
	if err = f.Sync(); err != nil {
		return 0, "", err
	}
	info, err := f.Stat()
	if err != nil {
		return 0, "", err
	}
	if err = f.Close(); err != nil {
		return 0, "", err
	}
	if err = root.Rename(tmp, name); err != nil {
		return 0, "", err
	}
	// fsync the directory before publishing completed metadata.
	dir, err := root.Open(".")
	if err != nil {
		return 0, "", err
	}
	err = dir.Sync()
	_ = dir.Close()
	if err != nil {
		return 0, "", err
	}
	return info.Size(), hex.EncodeToString(hash.Sum(nil)), nil
}

// Read verifies the complete compressed file and decompression before the
// caller can modify a database. It returns the same open file to avoid swaps.
func (s Store) Read(ctx context.Context, id, checksum string, size int64) (io.ReadCloser, error) {
	name, err := fileName(id)
	if err != nil {
		return nil, err
	}
	root, err := s.Open()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (io.ReadCloser, error) { _ = f.Close(); return nil, err }
	info, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() || info.Size() != size || len(checksum) != 64 {
		return fail(fmt.Errorf("backup metadata or file size is invalid"))
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, &contextReader{ctx: ctx, reader: f}); err != nil {
		return fail(err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != checksum {
		return fail(fmt.Errorf("backup checksum does not match"))
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	zipped, err := gzip.NewReader(f)
	if err != nil {
		return fail(err)
	}
	limit := s.MaxBytes
	if limit <= 0 {
		limit = 1 << 30
	}
	n, err := io.Copy(io.Discard, io.LimitReader(&contextReader{ctx: ctx, reader: zipped}, limit+1))
	_ = zipped.Close()
	if err != nil {
		return fail(err)
	}
	if n > limit {
		return fail(ErrSizeLimit)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	zipped, err = gzip.NewReader(f)
	if err != nil {
		return fail(err)
	}
	return &backupReader{Reader: &contextReader{ctx: ctx, reader: zipped}, zipped: zipped, file: f}, nil
}

func (s Store) Delete(id string) error {
	name, err := fileName(id)
	if err != nil {
		return err
	}
	root, err := s.Open()
	if err != nil {
		return err
	}
	defer root.Close()
	for _, name := range []string{name, name + ".partial"} {
		if err = root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

type limitedWriter struct {
	writer    io.Writer
	remaining int64
	ctx       context.Context
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if int64(len(p)) > w.remaining {
		return 0, ErrSizeLimit
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

type backupReader struct {
	io.Reader
	zipped *gzip.Reader
	file   *os.File
}

func (r *backupReader) Close() error { return errors.Join(r.zipped.Close(), r.file.Close()) }
