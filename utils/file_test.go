package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathWithinRootAllowsNestedPaths(t *testing.T) {
	got, err := ResolvePathWithinRoot("/tmp/root", "nested/archive.tar.gz")
	if err != nil {
		t.Fatalf("ResolvePathWithinRoot() returned error: %v", err)
	}

	want := "/tmp/root/nested/archive.tar.gz"
	if got != want {
		t.Fatalf("ResolvePathWithinRoot() = %q, want %q", got, want)
	}
}

func TestResolvePathWithinRootRejectsTraversal(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "../escape.tar.gz"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected traversal error")
	}
}

func TestResolvePathWithinRootRejectsAbsolutePath(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "/etc/passwd"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected absolute path error")
	}
}

func TestTarCreatesDestinationParentDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dump")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	childPath := filepath.Join(root, "data.bson")
	if err := os.WriteFile(childPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "archives", "archive.tar.gz")
	if err := Tar(root, archivePath); err != nil {
		t.Fatalf("Tar() error = %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestUnTarCreatesDestinationDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dump")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	childPath := filepath.Join(root, "data.bson")
	if err := os.WriteFile(childPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := Tar(root, archivePath); err != nil {
		t.Fatalf("Tar() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "nested")
	if err := UnTar(archivePath, destPath); err != nil {
		t.Fatalf("UnTar() error = %v", err)
	}

	restoredPath := filepath.Join(destPath, "data.bson")
	if _, err := os.Stat(restoredPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}
