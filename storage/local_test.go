package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/utils"
)

func TestLocalStorageUploadRejectsTraversal(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	sourcePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := s.Upload(context.Background(), "../escape.tar.gz", sourcePath); err == nil {
		t.Fatal("Upload() expected traversal error")
	}
}

func TestLocalStorageDownloadRejectsTraversal(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	destPath := filepath.Join(t.TempDir(), "out.tar.gz")

	if err := s.Download(context.Background(), "../escape.tar.gz", destPath); err == nil {
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
	targetPath, err := s.Upload(context.Background(), objectName, sourcePath)
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
	if err := s.Download(context.Background(), objectName, destPath); err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestLocalStorageUploadRejectsSymlinkComponent(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	sourcePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	linkPath := filepath.Join(s.LocalPath, "linked")
	if err := os.Symlink(t.TempDir(), linkPath); err != nil {
		t.Skipf("Symlink() unsupported: %v", err)
	}

	if _, err := s.Upload(context.Background(), filepath.Join("linked", "archive.tar.gz"), sourcePath); err == nil {
		t.Fatal("Upload() expected symlink component error")
	}
}

func TestLocalStorageDownloadRejectsSymlinkComponent(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	linkTarget := t.TempDir()
	linkPath := filepath.Join(s.LocalPath, "linked")
	if err := os.Symlink(linkTarget, linkPath); err != nil {
		t.Skipf("Symlink() unsupported: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linkTarget, "archive.tar.gz"), []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "archive.tar.gz")
	if err := s.Download(context.Background(), filepath.Join("linked", "archive.tar.gz"), destPath); err == nil {
		t.Fatal("Download() expected symlink component error")
	}
}

func TestLocalStorageUploadRejectsSameFileAndPreservesContent(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	objectName := filepath.Join("nested", "archive.tar.gz")
	sourcePath := filepath.Join(s.LocalPath, objectName)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("original"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := s.Upload(context.Background(), objectName, sourcePath); !errors.Is(err, utils.ErrSameFile) {
		t.Fatalf("Upload() error = %v, want ErrSameFile", err)
	}
	assertLocalFileContent(t, sourcePath, "original")
}

func TestLocalStorageDownloadRejectsHardLinkedDestinationAndPreservesContent(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	objectName := filepath.Join("nested", "archive.tar.gz")
	sourcePath := filepath.Join(s.LocalPath, objectName)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("original"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "archive.tar.gz")
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Link(sourcePath, destPath); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	if err := s.Download(context.Background(), objectName, destPath); !errors.Is(err, utils.ErrSameFile) {
		t.Fatalf("Download() error = %v, want ErrSameFile", err)
	}
	assertLocalFileContent(t, sourcePath, "original")
	assertLocalFileContent(t, destPath, "original")
}

func TestLocalStorageNestedObjectRoundTrip(t *testing.T) {
	s := &LocalStorage{LocalPath: t.TempDir()}
	sourcePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(sourcePath, []byte("round-trip"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	objectName := filepath.Join("nested", "deep", "archive.tar.gz")
	if _, err := s.Upload(context.Background(), objectName, sourcePath); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "nested", "archive.tar.gz")
	if err := s.Download(context.Background(), objectName, destPath); err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	assertLocalFileContent(t, destPath, "round-trip")
}

func TestLocalStorageEmptyStoreReturnsClearError(t *testing.T) {
	s := &LocalStorage{}
	if err := s.Init(t.TempDir(), 0, DefaultBackupPrefix); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err := s.GetTargetObjectName(context.Background(), "")
	if err == nil {
		t.Fatal("GetTargetObjectName() expected error")
	}
	if strings.Contains(err.Error(), `"."`) {
		t.Fatalf("GetTargetObjectName() error = %q, want clear no-object error", err)
	}
	if !strings.Contains(err.Error(), "no objects found") {
		t.Fatalf("GetTargetObjectName() error = %q, want no-object message", err)
	}
}

func TestLocalStorageGetTargetObjectNameUsesManagedPrefix(t *testing.T) {
	s := &LocalStorage{}
	if err := s.Init(t.TempDir(), 0, DefaultBackupPrefix); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	managedDir := filepath.Join(s.LocalPath, filepath.FromSlash("mongo-archive"))
	if err := os.MkdirAll(managedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.LocalPath, "legacy.tar.gz"), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(managedDir, "not-a-backup.tar.gz"), []byte("bad"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	managedFile := filepath.Join(managedDir, "9987654320999-2026-08-12T010203.456Z.tar.gz")
	if err := os.WriteFile(managedFile, []byte("managed"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := s.GetTargetObjectName(context.Background(), "")
	if err != nil {
		t.Fatalf("GetTargetObjectName() error = %v", err)
	}
	if got != "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz" {
		t.Fatalf("GetTargetObjectName() = %q", got)
	}
}

func TestLocalStorageMixedPrefixesKeepLatestSelectionAndRetentionIsolated(t *testing.T) {
	now := time.Now()
	objects := map[string]time.Time{
		"mongo-archive/9987654321000-2026-08-20T010203.456Z.tar.gz":    now.Add(-72 * time.Hour),
		"mongo-archive/9987654320999-2026-08-24T010203.456Z.tar.gz":    now.Add(-time.Hour),
		"postgres-archive/9987654321000-2026-08-20T010203.456Z.tar.gz": now.Add(-72 * time.Hour),
		"postgres-archive/9987654320999-2026-08-24T020304.567Z.tar.gz": now,
		"custom/nested/9987654320998-2026-08-24T030405.678Z.tar.gz":    now.Add(time.Hour),
	}
	tests := []struct {
		name       string
		prefix     string
		wantLatest string
		wantOld    string
	}{
		{
			name:       "MongoDB",
			prefix:     MongoDefaultBackupPrefix,
			wantLatest: "mongo-archive/9987654320999-2026-08-24T010203.456Z.tar.gz",
			wantOld:    "mongo-archive/9987654321000-2026-08-20T010203.456Z.tar.gz",
		},
		{
			name:       "PostgreSQL",
			prefix:     PostgreSQLDefaultBackupPrefix,
			wantLatest: "postgres-archive/9987654320999-2026-08-24T020304.567Z.tar.gz",
			wantOld:    "postgres-archive/9987654321000-2026-08-20T010203.456Z.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for name, modifiedAt := range objects {
				path := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
				if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				if err := os.Chtimes(path, modifiedAt, modifiedAt); err != nil {
					t.Fatalf("Chtimes() error = %v", err)
				}
			}

			s := &LocalStorage{}
			if err := s.Init(root, 1, tt.prefix); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			got, err := s.GetTargetObjectName(context.Background(), "")
			if err != nil {
				t.Fatalf("GetTargetObjectName() error = %v", err)
			}
			if got != tt.wantLatest {
				t.Fatalf("GetTargetObjectName() = %q, want %q", got, tt.wantLatest)
			}
			if err := s.DeleteOldObjects(context.Background(), tt.wantLatest); err != nil {
				t.Fatalf("DeleteOldObjects() error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tt.wantOld))); !os.IsNotExist(err) {
				t.Fatalf("expired object %q still exists or stat failed: %v", tt.wantOld, err)
			}
			for name := range objects {
				if name == tt.wantOld {
					continue
				}
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
					t.Fatalf("out-of-scope object %q was changed: %v", name, err)
				}
			}
		})
	}
}

func assertLocalFileContent(t *testing.T, filePath string, want string) {
	t.Helper()
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("ReadFile() = %q, want %q", string(got), want)
	}
}
