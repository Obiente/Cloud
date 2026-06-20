package gameservers

import (
	"crypto/sha1"
	"os"
	"path/filepath"
	"testing"

	"gameservers-service/internal/catalog/modrinth"
)

func TestMatchingVersionFilePrefersHash(t *testing.T) {
	files := []modrinth.VersionFile{
		{
			Filename: "Plugin.jar",
			Hashes:   map[string]string{"sha1": "old"},
		},
		{
			Filename: "Plugin-Renamed.jar",
			Hashes:   map[string]string{"sha1": "expected"},
		},
	}

	file := matchingVersionFile(files, "expected", "sha1", "Plugin.jar")
	if file == nil || file.Filename != "Plugin-Renamed.jar" {
		t.Fatalf("expected hash match, got %#v", file)
	}
}

func TestFileHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.jar")
	if err := os.WriteFile(path, []byte("plugin-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	sum, err := fileHash(path, sha1.New())
	if err != nil {
		t.Fatalf("fileHash returned error: %v", err)
	}
	if sum != "d093414c4a97fac29482fb0dfb22872a2d8ddd90" {
		t.Fatalf("unexpected hash %s", sum)
	}
}
