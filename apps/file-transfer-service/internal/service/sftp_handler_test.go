package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathStaysWithinRoot(t *testing.T) {
	root := t.TempDir()
	handler := newSFTPHandler(&Session{RootPath: root})

	resolved, err := handler.resolvePath("../../etc/passwd")
	if err != nil {
		t.Fatalf("resolvePath returned error for cleaned traversal: %v", err)
	}

	want := filepath.Join(root, "etc", "passwd")
	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

func TestResolvePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	handler := newSFTPHandler(&Session{RootPath: root})
	if _, err := handler.resolvePath("/outside/file.txt"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestIsWithinRootAllowsRootAndChildren(t *testing.T) {
	root := t.TempDir()
	if !isWithinRoot(root, root) {
		t.Fatal("root should be within itself")
	}
	if !isWithinRoot(root, filepath.Join(root, "nested", "file.txt")) {
		t.Fatal("child path should be within root")
	}
	if isWithinRoot(root, filepath.Dir(root)) {
		t.Fatal("parent path should not be within root")
	}
}
