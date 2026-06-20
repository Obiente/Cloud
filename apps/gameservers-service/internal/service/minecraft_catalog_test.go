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

func TestMinecraftProjectNameCandidates(t *testing.T) {
	tests := map[string]string{
		"ViaBackwards-4.3.0.jar":                           "ViaBackwards",
		"LuckPerms-Bukkit-5.4.30.jar":                      "LuckPerms",
		"FastAsyncWorldEdit-Bukkit-2.4.1-SNAPSHOT-239.jar": "FastAsyncWorldEdit",
		"CMILib1.5.9.7.jar":                                "CMILib",
		"Jobs5.1.0.0.jar":                                  "Jobs",
		"squaremap-paper-mc1.19-1.1.5.jar":                 "squaremap",
		"worldguard-bukkit-7.0.7-dist.jar":                 "worldguard",
	}

	for filename, expected := range tests {
		candidates := minecraftProjectNameCandidates(filename)
		if len(candidates) == 0 || candidates[0] != expected {
			t.Fatalf("expected first candidate %q for %s, got %#v", expected, filename, candidates)
		}
	}
}

func TestMinecraftVersionFromFilename(t *testing.T) {
	tests := map[string]string{
		"ViaBackwards-4.3.0.jar":           "4.3.0",
		"CMILib1.5.9.7.jar":                "1.5.9.7",
		"ProtocolLib.jar":                  "",
		"squaremap-paper-mc1.19-1.1.5.jar": "1.1.5",
	}

	for filename, expected := range tests {
		if got := minecraftVersionFromFilename(filename); got != expected {
			t.Fatalf("expected version %q for %s, got %q", expected, filename, got)
		}
	}
}

func TestMinecraftProjectMatchesFilename(t *testing.T) {
	candidates := minecraftProjectNameCandidates("ViaBackwards-4.3.0.jar")
	if !minecraftProjectMatchesFilename(modrinth.Project{Slug: "viabackwards", Title: "ViaBackwards"}, candidates) {
		t.Fatal("expected ViaBackwards to match filename candidates")
	}
	if minecraftProjectMatchesFilename(modrinth.Project{Slug: "viaversion", Title: "ViaVersion"}, candidates) {
		t.Fatal("did not expect ViaVersion to match ViaBackwards filename candidates")
	}
}

func TestIsStableMinecraftVersion(t *testing.T) {
	tests := []struct {
		name    string
		version modrinth.Version
		stable  bool
	}{
		{
			name:    "release type is stable",
			version: modrinth.Version{VersionType: "release", VersionNumber: "5.10.0"},
			stable:  true,
		},
		{
			name:    "missing type with normal version is stable",
			version: modrinth.Version{VersionNumber: "5.10.0"},
			stable:  true,
		},
		{
			name:    "beta type is not stable",
			version: modrinth.Version{VersionType: "beta", VersionNumber: "5.10.1"},
			stable:  false,
		},
		{
			name:    "alpha type is not stable",
			version: modrinth.Version{VersionType: "alpha", VersionNumber: "5.10.1"},
			stable:  false,
		},
		{
			name:    "snapshot marker is not stable",
			version: modrinth.Version{VersionType: "release", VersionNumber: "5.10.1-SNAPSHOT+1011"},
			stable:  false,
		},
		{
			name:    "release candidate marker is not stable",
			version: modrinth.Version{VersionType: "release", Name: "v5.10.1-rc.1"},
			stable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStableMinecraftVersion(tt.version); got != tt.stable {
				t.Fatalf("expected stable=%v, got %v", tt.stable, got)
			}
		})
	}
}
