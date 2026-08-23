package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type archiveRestoreStorage interface {
	ArchiveStorage
	RestoreStorage
}

type providerContractSubject struct {
	name    string
	backend archiveRestoreStorage
	root    string
	cloud   *fakeProviderClient
}

type fakeProviderClient struct {
	objects                map[string][]byte
	pages                  [][]objectTimestamp
	failUploadVerification bool
	failDelete             bool
	failDownload           bool
	deleted                []string
}

func newProviderContractSubjects(t *testing.T) []providerContractSubject {
	t.Helper()

	newCloud := func(name string) providerContractSubject {
		client := &fakeProviderClient{objects: map[string][]byte{}}
		subject := providerContractSubject{name: name, cloud: client}
		switch name {
		case "aws":
			subject.backend = &AwsS3{
				Bucket:          "bucket",
				BackupPrefix:    DefaultBackupPrefix,
				uploadObject:    client.upload,
				downloadObject:  client.download,
				objectExists:    client.exists,
				listObjectPages: client.listPages,
				deleteObject:    client.delete,
			}
		case "azure":
			subject.backend = &AzBlob{
				ContainerName:   "container",
				BackupPrefix:    DefaultBackupPrefix,
				uploadObject:    client.upload,
				downloadObject:  client.download,
				objectExists:    client.exists,
				listObjectPages: client.listPages,
				deleteObject:    client.delete,
			}
		case "gcp":
			subject.backend = &GcpStorage{
				Bucket:          "bucket",
				BackupPrefix:    DefaultBackupPrefix,
				uploadObject:    client.upload,
				downloadObject:  client.download,
				objectExists:    client.exists,
				listObjectPages: client.listPages,
				deleteObject:    client.delete,
			}
		default:
			t.Fatalf("unknown cloud provider %q", name)
		}
		return subject
	}

	localRoot := t.TempDir()
	local := &LocalStorage{}
	if err := local.Init(localRoot, 0, DefaultBackupPrefix); err != nil {
		t.Fatalf("LocalStorage.Init() error = %v", err)
	}

	return []providerContractSubject{
		{name: "local", backend: local, root: localRoot},
		newCloud("aws"),
		newCloud("azure"),
		newCloud("gcp"),
	}
}

func TestProviderContractMissingExplicitObjectFails(t *testing.T) {
	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			_, err := subject.backend.GetTargetObjectName(context.Background(), "missing.tar.gz")
			if err == nil {
				t.Fatal("GetTargetObjectName() expected missing-object error")
			}
			if !strings.Contains(err.Error(), "missing.tar.gz") {
				t.Fatalf("GetTargetObjectName() error = %q, want missing object name", err)
			}
		})
	}
}

func TestProviderContractPaginatedLatestSelection(t *testing.T) {
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	oldObject := "mongo-archive/9987654321000-2026-08-20T010203.456Z.tar.gz"
	newObject := "mongo-archive/9987654320999-2026-08-22T010203.456Z.tar.gz"
	malformedObject := "mongo-archive/not-a-backup.tar.gz"

	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			subject.putPages(t, [][]objectTimestamp{
				{{Name: oldObject, ModifiedAt: now.Add(-48 * time.Hour)}, {Name: malformedObject, ModifiedAt: now.Add(48 * time.Hour)}},
				{{Name: newObject, ModifiedAt: now}},
			})

			got, err := subject.backend.GetTargetObjectName(context.Background(), "")
			if err != nil {
				t.Fatalf("GetTargetObjectName() error = %v", err)
			}
			if got != newObject {
				t.Fatalf("GetTargetObjectName() = %q, want %q", got, newObject)
			}
		})
	}
}

func TestProviderContractUploadVerificationFailure(t *testing.T) {
	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			source := writeContractSource(t, "archive")
			subject.setUploadVerificationFailure()

			_, err := subject.backend.Upload(context.Background(), "mongo-archive/9987654320999-2026-08-22T010203.456Z.tar.gz", source)
			if err == nil {
				t.Fatal("Upload() expected verification failure")
			}
			if !strings.Contains(err.Error(), "verify") {
				t.Fatalf("Upload() error = %q, want verification failure", err)
			}
		})
	}
}

func TestProviderContractRetentionDeletionFailure(t *testing.T) {
	now := time.Now()
	oldObject := "mongo-archive/9987654321000-2026-08-20T010203.456Z.tar.gz"
	currentObject := "mongo-archive/9987654320999-2026-08-22T010203.456Z.tar.gz"

	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			subject.setExpiryDays(1)
			subject.putPages(t, [][]objectTimestamp{{
				{Name: oldObject, ModifiedAt: now.Add(-72 * time.Hour)},
				{Name: currentObject, ModifiedAt: now},
			}})
			subject.setRetentionDeletionFailure()

			err := subject.backend.DeleteOldObjects(context.Background(), currentObject)
			if err == nil {
				t.Fatal("DeleteOldObjects() expected deletion failure")
			}
			if !strings.Contains(err.Error(), oldObject) {
				t.Fatalf("DeleteOldObjects() error = %q, want failed object name", err)
			}
		})
	}
}

func TestProviderContractCancellation(t *testing.T) {
	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := subject.backend.Upload(ctx, "mongo-archive/9987654320999-2026-08-22T010203.456Z.tar.gz", writeContractSource(t, "archive"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Upload() error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestProviderContractPartialDownloadCleanup(t *testing.T) {
	for _, subject := range newProviderContractSubjects(t) {
		t.Run(subject.name, func(t *testing.T) {
			subject.setDownloadFailure()
			destDir := t.TempDir()
			destPath := filepath.Join(destDir, "restore.tar.gz")

			err := subject.backend.Download(context.Background(), "mongo-archive/9987654320999-2026-08-22T010203.456Z.tar.gz", destPath)
			if err == nil {
				t.Fatal("Download() expected failure")
			}
			if _, statErr := os.Stat(destPath); !os.IsNotExist(statErr) {
				t.Fatalf("Download() left final file, stat error = %v", statErr)
			}
			entries, readErr := os.ReadDir(destDir)
			if readErr != nil {
				t.Fatalf("ReadDir() error = %v", readErr)
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), ".restore.tar.gz.partial-") {
					t.Fatalf("Download() left partial file %q", entry.Name())
				}
			}
		})
	}
}

func (s providerContractSubject) putPages(t *testing.T, pages [][]objectTimestamp) {
	t.Helper()
	if s.cloud != nil {
		s.cloud.pages = pages
		for _, page := range pages {
			for _, obj := range page {
				s.cloud.objects[obj.Name] = []byte("object")
			}
		}
		return
	}

	for _, page := range pages {
		for _, obj := range page {
			path, err := utilsResolveLocalContractPath(s.root, obj.Name)
			if err != nil {
				t.Fatalf("resolve local object path: %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(path, []byte("object"), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if err := os.Chtimes(path, obj.ModifiedAt, obj.ModifiedAt); err != nil {
				t.Fatalf("Chtimes() error = %v", err)
			}
		}
	}
}

func (s providerContractSubject) setUploadVerificationFailure() {
	if s.cloud != nil {
		s.cloud.failUploadVerification = true
		return
	}

	local := s.backend.(*LocalStorage)
	local.statFile = func(path string) (os.FileInfo, error) {
		info, err := os.Stat(path)
		if err != nil || !strings.Contains(filepath.ToSlash(path), "mongo-archive/") {
			return info, err
		}
		return sizeOverrideFileInfo{FileInfo: info, size: info.Size() + 1}, nil
	}
}

func (s providerContractSubject) setRetentionDeletionFailure() {
	if s.cloud != nil {
		s.cloud.failDelete = true
		return
	}
	s.backend.(*LocalStorage).deleteObject = func(string) error { return errors.New("delete failed") }
}

func (s providerContractSubject) setDownloadFailure() {
	if s.cloud != nil {
		s.cloud.failDownload = true
		return
	}
	s.backend.(*LocalStorage).downloadObject = func(ctx context.Context, source string, dest *os.File) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, _ = dest.WriteString("partial")
		return errors.New("download failed")
	}
}

func (s providerContractSubject) setExpiryDays(days int) {
	switch b := s.backend.(type) {
	case *LocalStorage:
		b.ExpiryDays = days
	case *AwsS3:
		b.ExpiryDays = days
	case *AzBlob:
		b.ExpiryDays = days
	case *GcpStorage:
		b.ExpiryDays = days
	}
}

func (f *fakeProviderClient) upload(ctx context.Context, name string, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	f.objects[name] = data
	if f.failUploadVerification {
		return "", errors.New("failed to verify uploaded object: verification failed")
	}
	return "etag", nil
}

func (f *fakeProviderClient) download(ctx context.Context, name string, dest *os.File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.failDownload {
		_, _ = dest.WriteString("partial")
		return errors.New("download failed")
	}
	data, ok := f.objects[name]
	if !ok {
		return errors.New("object not found")
	}
	_, err := dest.Write(data)
	return err
}

func (f *fakeProviderClient) exists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, ok := f.objects[name]
	return ok, nil
}

func (f *fakeProviderClient) listPages(ctx context.Context, handle func([]objectTimestamp) bool) error {
	for _, page := range f.pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !handle(page) {
			return nil
		}
	}
	return nil
}

func (f *fakeProviderClient) delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, name)
	if f.failDelete {
		return errors.New("delete failed")
	}
	delete(f.objects, name)
	return nil
}

func writeContractSource(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.tar.gz")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func utilsResolveLocalContractPath(root, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if strings.HasPrefix(cleanName, "..") {
		return "", errors.New("invalid path")
	}
	return filepath.Join(root, cleanName), nil
}

type sizeOverrideFileInfo struct {
	os.FileInfo
	size int64
}

func (f sizeOverrideFileInfo) Size() int64 { return f.size }
