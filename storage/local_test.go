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
