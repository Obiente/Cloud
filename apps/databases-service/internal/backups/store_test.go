package backups

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func testStore(t *testing.T) Store {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".obiente-backup-volume"), []byte("obiente-cloud-backups-v1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return Store{Directory: directory, MaxBytes: 1024}
}
func TestStoreRoundTripAndCorruption(t *testing.T) {
	s := testStore(t)
	id := uuid.NewString()
	ctx := context.Background()
	size, sum, err := s.Write(ctx, id, func(w io.Writer) error { _, err := io.WriteString(w, "schema and Unicode: café"); return err })
	if err != nil {
		t.Fatal(err)
	}
	reader, err := s.Read(ctx, id, sum, size)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "schema and Unicode: café" {
		t.Fatalf("round trip: %q, %v", data, err)
	}
	info, err := os.Stat(filepath.Join(s.Directory, id+".sql.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("backup permissions: %v", info.Mode())
	}
	file, err := os.OpenFile(filepath.Join(s.Directory, id+".sql.gz"), os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteAt([]byte("changed"), 12)
	_ = file.Close()
	if reader, err = s.Read(ctx, id, sum, size); err == nil {
		_ = reader.Close()
		t.Fatal("corrupt backup accepted")
	}
	if err = s.Delete(id); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(id); err != nil {
		t.Fatal("deletion retry failed", err)
	}
}
func TestStoreFailuresNeverPublishPartialBackups(t *testing.T) {
	for _, scenario := range []string{"producer", "limit", "cancel", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			s := testStore(t)
			id := uuid.NewString()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, _, err := s.Write(ctx, id, func(w io.Writer) error {
				if scenario == "empty" {
					return nil
				}
				if scenario == "cancel" {
					cancel()
				}
				if scenario == "limit" {
					_, err := io.WriteString(w, strings.Repeat("a", 1025))
					return err
				}
				if _, err := io.WriteString(w, "partial data"); err != nil {
					return err
				}
				return errors.New("producer disconnected")
			})
			if err == nil {
				t.Fatal("failed backup was accepted")
			}
			entries, err := os.ReadDir(s.Directory)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("partial files retained: %v", entries)
			}
		})
	}
}
func TestStoreRequiresMountedVolumeAndContainedPaths(t *testing.T) {
	s := Store{Directory: filepath.Join(t.TempDir(), "missing")}
	if root, err := s.Open(); err == nil {
		root.Close()
		t.Fatal("missing mount accepted")
	}
	s.Directory = t.TempDir()
	if root, err := s.Open(); err == nil {
		root.Close()
		t.Fatal("empty mountpoint accepted")
	}
	s = testStore(t)
	if err := s.Delete("../../outside"); err == nil {
		t.Fatal("path traversal accepted")
	}
	outside := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	if err := os.Symlink(outside, filepath.Join(s.Directory, id+".sql.gz")); err != nil {
		t.Fatal(err)
	}
	if reader, err := s.Read(context.Background(), id, strings.Repeat("0", 64), 4); err == nil {
		reader.Close()
		t.Fatal("symlink escape accepted")
	}
	data, _ := os.ReadFile(outside)
	if string(data) != "keep" {
		t.Fatal("outside file modified")
	}
}
