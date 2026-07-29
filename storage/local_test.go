package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageUploadRejectsTraversal(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	sourcePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := s.Upload("../escape.tar.gz", sourcePath); err == nil {
		t.Fatal("Upload() expected traversal error")
	}
}

func TestLocalStorageDownloadRejectsTraversal(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	destPath := filepath.Join(t.TempDir(), "out.tar.gz")

	if err := s.Download("../escape.tar.gz", destPath); err == nil {
		t.Fatal("Download() expected traversal error")
	}
}

func TestLocalStorageUploadCreatesDestinationParentDirectory(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	sourcePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	objectName := filepath.Join("nested", "archive.tar.gz")
	targetPath, err := s.Upload(objectName, sourcePath)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestLocalStorageDownloadCreatesDestinationParentDirectory(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	objectName := filepath.Join("nested", "archive.tar.gz")
	sourcePath := filepath.Join(s.LocalPath, objectName)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "archive.tar.gz")
	if err := s.Download(objectName, destPath); err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}
