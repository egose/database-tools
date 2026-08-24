package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

type recordingStorage struct {
	name      string
	calls     []string
	callLog   *[]string
	uploadErr error
	deleteErr error
	closeErr  error
}

type blockingArchiveStorage struct{}

func (s *recordingStorage) Upload(_ context.Context, name string, _ string) (string, error) {
	s.record("upload:" + name)
	if s.uploadErr != nil {
		return "", s.uploadErr
	}
	return "verified", nil
}

func (s *recordingStorage) DeleteOldObjects(_ context.Context, name string) error {
	s.record("delete:" + name)
	return s.deleteErr
}

func (s *recordingStorage) Close() error { return s.closeErr }

func (s *recordingStorage) record(call string) {
	s.calls = append(s.calls, call)
	if s.callLog == nil {
		return
	}
	entry := call
	if s.name != "" {
		entry = s.name + ":" + call
	}
	*s.callLog = append(*s.callLog, entry)
}

func (s *blockingArchiveStorage) Upload(ctx context.Context, _ string, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (s *blockingArchiveStorage) DeleteOldObjects(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *blockingArchiveStorage) Close() error { return nil }

func TestArchivePipelineRejectsIncompleteStorageBeforeSideEffects(t *testing.T) {
	workspaceCreated := false
	storagesConstructed := false
	pipeline := archivePipeline{
		createWorkspace: func(string) (string, error) {
			workspaceCreated = true
			return t.TempDir(), nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			storagesConstructed = true
			return []storage.ArchiveBackend{&recordingStorage{}}, nil
		},
	}

	err := pipeline.run(context.Background(), &mongoarchive.Config{
		StorageOptions: toolconfig.StorageOptions{AWSBucket: "archive-bucket"},
	})
	if err == nil {
		t.Fatal("run() expected error")
	}
	if !strings.Contains(err.Error(), "AWS_ACCESS_KEY_ID") || strings.Contains(err.Error(), "archive-bucket") {
		t.Fatalf("run() error = %q, want missing identifiers without supplied value", err)
	}
	if workspaceCreated {
		t.Fatal("run() created workspace before rejecting incomplete storage config")
	}
	if storagesConstructed {
		t.Fatal("run() constructed storage before rejecting incomplete storage config")
	}
}

func TestArchivePipelineRejectsInvalidRuntimeBeforeSideEffects(t *testing.T) {
	workspaceCreated := false
	storagesConstructed := false
	pipeline := archivePipeline{
		createWorkspace: func(string) (string, error) {
			workspaceCreated = true
			return t.TempDir(), nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			storagesConstructed = true
			return []storage.ArchiveBackend{&recordingStorage{}}, nil
		},
	}

	err := pipeline.run(context.Background(), &mongoarchive.Config{
		RuntimeOptions: mongoarchive.RuntimeOptions{StorageOperationTimeout: -time.Second},
	})
	if err == nil || !strings.Contains(err.Error(), "STORAGE_OPERATION_TIMEOUT") {
		t.Fatalf("run() error = %v, want runtime validation error", err)
	}
	if workspaceCreated {
		t.Fatal("run() created workspace before rejecting invalid runtime config")
	}
	if storagesConstructed {
		t.Fatal("run() constructed storage before rejecting invalid runtime config")
	}
}

type fakeArchiveDump struct {
	initErr error
	dumpErr error
	onDump  func()
}

func (d *fakeArchiveDump) Init() error {
	return d.initErr
}

func (d *fakeArchiveDump) Dump() error {
	if d.onDump != nil {
		d.onDump()
	}
	return d.dumpErr
}

func (d *fakeArchiveDump) HandleInterrupt() {}

type fakeCronScheduler struct {
	mu          sync.Mutex
	expression  string
	overlap     cronOverlapPolicy
	task        func()
	running     atomic.Bool
	scheduleErr error
	shutdowns   atomic.Int32
	started     chan struct{}
	startOnce   sync.Once
}

type countingNotificationSender struct {
	sends atomic.Int32
}

func (s *countingNotificationSender) Send(context.Context, bool, string) {
	s.sends.Add(1)
}

func (s *fakeCronScheduler) Schedule(expression string, task func(), overlap cronOverlapPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expression = expression
	s.task = task
	s.overlap = overlap
	return s.scheduleErr
}

func (s *fakeCronScheduler) Start() {
	if s.started != nil {
		s.startOnce.Do(func() { close(s.started) })
	}
}

func (s *fakeCronScheduler) Shutdown() error {
	s.shutdowns.Add(1)
	return nil
}

func (s *fakeCronScheduler) trigger() {
	s.mu.Lock()
	task := s.task
	overlap := s.overlap
	s.mu.Unlock()

	if task == nil {
		return
	}
	if overlap == cronSkipOverlappingRuns {
		if !s.running.CompareAndSwap(false, true) {
			return
		}
		go func() {
			defer s.running.Store(false)
			task()
		}()
		return
	}
	go task()
}

func TestArchiveRuntimeConstructsNotificationsOnceForOneShotRun(t *testing.T) {
	sender := &countingNotificationSender{}
	var constructions atomic.Int32
	runtime := archiveRuntime{
		newNotifications: func(*mongoarchive.Config) (notificationSender, error) {
			constructions.Add(1)
			return sender, nil
		},
		runTask: func(ctx context.Context, _ *mongoarchive.Config, got notificationSender) error {
			if got != sender {
				t.Fatalf("runTask() notification sender = %T, want constructed sender", got)
			}
			got.Send(ctx, true, "backup.tar.gz")
			return nil
		},
		runCronJob: func(context.Context, *mongoarchive.Config, notificationSender) error {
			t.Fatal("runCronJob() called for one-shot run")
			return nil
		},
	}

	if err := runtime.run(context.Background(), &mongoarchive.Config{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := constructions.Load(); got != 1 {
		t.Fatalf("notification constructions = %d, want 1", got)
	}
	if got := sender.sends.Load(); got != 1 {
		t.Fatalf("notification sends = %d, want 1", got)
	}
}

func TestArchiveRuntimeConstructsNotificationsOnceForRepeatedCronRuns(t *testing.T) {
	sender := &countingNotificationSender{}
	var constructions atomic.Int32
	runtime := archiveRuntime{
		newNotifications: func(*mongoarchive.Config) (notificationSender, error) {
			constructions.Add(1)
			return sender, nil
		},
		runTask: func(context.Context, *mongoarchive.Config, notificationSender) error {
			t.Fatal("runTask() called directly for cron run")
			return nil
		},
		runCronJob: func(ctx context.Context, _ *mongoarchive.Config, got notificationSender) error {
			if got != sender {
				t.Fatalf("runCronJob() notification sender = %T, want constructed sender", got)
			}
			got.Send(ctx, false, "first failure")
			got.Send(ctx, false, "second failure")
			return nil
		},
	}
	cfg := &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Cron: true}}

	if err := runtime.run(context.Background(), cfg); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := constructions.Load(); got != 1 {
		t.Fatalf("notification constructions = %d, want 1", got)
	}
	if got := sender.sends.Load(); got != 2 {
		t.Fatalf("notification sends = %d, want 2", got)
	}
}

func TestUploadBackupToStoragesUploadsBeforeRetention(t *testing.T) {
	backend := &recordingStorage{}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	if err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{backend}, objectName, "/tmp/archive.tar.gz", 0); err != nil {
		t.Fatalf("uploadBackupToStorages() error = %v", err)
	}

	want := []string{"upload:" + objectName, "delete:" + objectName}
	if !reflect.DeepEqual(backend.calls, want) {
		t.Fatalf("uploadBackupToStorages() calls = %#v, want %#v", backend.calls, want)
	}
}

func TestUploadBackupToStoragesSkipsRetentionAfterUploadFailure(t *testing.T) {
	backend := &recordingStorage{uploadErr: errors.New("upload failed")}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{backend}, objectName, "/tmp/archive.tar.gz", 0)
	if err == nil {
		t.Fatal("uploadBackupToStorages() expected error")
	}

	want := []string{"upload:" + objectName}
	if !reflect.DeepEqual(backend.calls, want) {
		t.Fatalf("uploadBackupToStorages() calls = %#v, want %#v", backend.calls, want)
	}
}

func TestUploadBackupToStoragesMultiBackendRunsRetentionAfterAllUploads(t *testing.T) {
	var callLog []string
	first := &recordingStorage{name: "first", callLog: &callLog}
	second := &recordingStorage{name: "second", callLog: &callLog}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	if err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{first, second}, objectName, "/tmp/archive.tar.gz", 0); err != nil {
		t.Fatalf("uploadBackupToStorages() error = %v", err)
	}

	want := []string{
		"first:upload:" + objectName,
		"second:upload:" + objectName,
		"first:delete:" + objectName,
		"second:delete:" + objectName,
	}
	if !reflect.DeepEqual(callLog, want) {
		t.Fatalf("uploadBackupToStorages() call log = %#v, want %#v", callLog, want)
	}
}

func TestUploadBackupToStoragesMultiBackendReturnsExplicitPartialUploadFailure(t *testing.T) {
	uploadErr := errors.New("upload failed")
	var callLog []string
	first := &recordingStorage{name: "first", callLog: &callLog}
	second := &recordingStorage{name: "second", callLog: &callLog, uploadErr: uploadErr}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{first, second}, objectName, "/tmp/archive.tar.gz", 0)
	if !errors.Is(err, uploadErr) {
		t.Fatalf("uploadBackupToStorages() error = %v, want wrapped %v", err, uploadErr)
	}
	if !strings.Contains(err.Error(), "retention was not run on any backend") {
		t.Fatalf("uploadBackupToStorages() error = %v, want explicit retention state", err)
	}
	if !strings.Contains(err.Error(), "backend #1") || !strings.Contains(err.Error(), "backend #2") {
		t.Fatalf("uploadBackupToStorages() error = %v, want backend-specific partial upload details", err)
	}

	want := []string{
		"first:upload:" + objectName,
		"second:upload:" + objectName,
	}
	if !reflect.DeepEqual(callLog, want) {
		t.Fatalf("uploadBackupToStorages() call log = %#v, want %#v", callLog, want)
	}
}

func TestUploadBackupToStoragesMultiBackendAttemptsLaterUploadWhenFirstUploadFails(t *testing.T) {
	uploadErr := errors.New("upload failed")
	var callLog []string
	first := &recordingStorage{name: "first", callLog: &callLog, uploadErr: uploadErr}
	second := &recordingStorage{name: "second", callLog: &callLog}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{first, second}, objectName, "/tmp/archive.tar.gz", 0)
	if !errors.Is(err, uploadErr) {
		t.Fatalf("uploadBackupToStorages() error = %v, want wrapped %v", err, uploadErr)
	}
	if !strings.Contains(err.Error(), "retention was not run on any backend") {
		t.Fatalf("uploadBackupToStorages() error = %v, want explicit retention state", err)
	}

	want := []string{
		"first:upload:" + objectName,
		"second:upload:" + objectName,
	}
	if !reflect.DeepEqual(callLog, want) {
		t.Fatalf("uploadBackupToStorages() call log = %#v, want %#v", callLog, want)
	}
}

func TestUploadBackupToStoragesReturnsDeletionFailure(t *testing.T) {
	deleteErr := errors.New("delete failed")
	backend := &recordingStorage{deleteErr: deleteErr}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{backend}, objectName, "/tmp/archive.tar.gz", 0)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("uploadBackupToStorages() error = %v, want wrapped %v", err, deleteErr)
	}
}

func TestUploadBackupToStoragesHonorsConfiguredDeadline(t *testing.T) {
	start := time.Now()
	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{&blockingArchiveStorage{}}, "backup.tar.gz", "/tmp/archive.tar.gz", 20*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("uploadBackupToStorages() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("uploadBackupToStorages() elapsed = %v, want prompt timeout", elapsed)
	}
}

func TestUploadBackupToStoragesUsesParsedTimeoutInsteadOfEnvironment(t *testing.T) {
	t.Setenv(envPrefix+"STORAGE_OPERATION_TIMEOUT", "not-a-duration")

	err := uploadBackupToStorages(context.Background(), []storage.ArchiveBackend{&blockingArchiveStorage{}}, "backup.tar.gz", "/tmp/archive.tar.gz", time.Nanosecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("uploadBackupToStorages() error = %v, want parsed timeout deadline", err)
	}
}

func TestArchivePipelinePropagatesCancellationToUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 1)
	pipeline := archivePipeline{
		createWorkspace: func(string) (string, error) { return t.TempDir(), nil },
		newFilename:     func() (string, string) { return "backup.tar.gz", "dumpdir" },
		newDump: func([]string) (archiveDump, func(), error) {
			return &fakeArchiveDump{}, func() {}, nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			return []storage.ArchiveBackend{&recordingStorage{}}, nil
		},
		tar:             func(string, string) error { return nil },
		buildObjectName: func(string, string) (string, error) { return "backup.tar.gz", nil },
		upload: func(ctx context.Context, _ []storage.ArchiveBackend, _ string, _ string, _ time.Duration) error {
			started <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: func(func()) chan struct{} { return nil },
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.run(ctx, &mongoarchive.Config{})
	}()

	<-started
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context canceled", err)
	}
}

func TestArchivePipelineCleanupByStage(t *testing.T) {
	dumpErr := errors.New("dump failed")
	tarErr := errors.New("archive failed")
	uploadErr := errors.New("upload failed")

	tests := []struct {
		name          string
		keep          bool
		dumpErr       error
		tarErr        error
		uploadErr     error
		wantErr       error
		wantWorkspace bool
		wantDumpDir   bool
		wantTarFile   bool
	}{
		{
			name:    "dump failure cleans up partial output",
			dumpErr: dumpErr,
			wantErr: dumpErr,
		},
		{
			name:    "archive failure cleans up partial tarball",
			tarErr:  tarErr,
			wantErr: tarErr,
		},
		{
			name:      "upload failure cleans up artifacts",
			uploadErr: uploadErr,
			wantErr:   uploadErr,
		},
		{
			name:          "keep preserves artifacts after upload failure",
			keep:          true,
			uploadErr:     uploadErr,
			wantErr:       uploadErr,
			wantWorkspace: true,
			wantDumpDir:   true,
			wantTarFile:   true,
		},
		{
			name:          "keep preserves artifacts after success",
			keep:          true,
			wantWorkspace: true,
			wantDumpDir:   true,
			wantTarFile:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			var workspace string
			var dumpDir string
			var tarFile string

			pipeline := archivePipeline{
				createWorkspace: func(string) (string, error) {
					var err error
					workspace, err = os.MkdirTemp(root, "run-")
					return workspace, err
				},
				newFilename: func() (string, string) {
					return "backup.tar.gz", "dumpdir"
				},
				newDump: func(options []string) (archiveDump, func(), error) {
					dumpDir = archiveOutPath(t, options)
					return &fakeArchiveDump{
						dumpErr: tt.dumpErr,
						onDump: func() {
							if err := os.MkdirAll(dumpDir, 0o700); err != nil {
								t.Fatalf("MkdirAll(%q) error = %v", dumpDir, err)
							}
							dataFile := filepath.Join(dumpDir, "archive.bson")
							if err := os.WriteFile(dataFile, []byte("dump"), 0o600); err != nil {
								t.Fatalf("WriteFile(%q) error = %v", dataFile, err)
							}
						},
					}, func() {}, nil
				},
				getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
					return []storage.ArchiveBackend{&recordingStorage{}}, nil
				},
				tar: func(rootPath string, destination string) error {
					tarFile = destination
					if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
						return err
					}
					if err := os.WriteFile(destination, []byte("tar"), 0o600); err != nil {
						return err
					}
					return tt.tarErr
				},
				buildObjectName: func(string, string) (string, error) {
					return "backup.tar.gz", nil
				},
				upload: func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
					return tt.uploadErr
				},
				deleteDirectory: utils.DeleteDirectory,
				deleteFile:      utils.DeleteFile,
				handleInterrupt: func(func()) chan struct{} { return nil },
			}

			cfg := &mongoarchive.Config{Keep: tt.keep}
			err := pipeline.run(context.Background(), cfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, want wrapped %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("run() unexpected error = %v", err)
			}

			assertPathState(t, workspace, tt.wantWorkspace)
			assertPathState(t, dumpDir, tt.wantDumpDir)
			assertPathState(t, tarFile, tt.wantTarFile)
		})
	}
}

func TestArchivePipelineAggregatesCleanupFailure(t *testing.T) {
	primaryErr := errors.New("upload failed")
	cleanupErr := errors.New("remove tar failed")
	root := t.TempDir()
	var tarFile string

	pipeline := archivePipeline{
		createWorkspace: func(string) (string, error) {
			return os.MkdirTemp(root, "run-")
		},
		newFilename: func() (string, string) {
			return "backup.tar.gz", "dumpdir"
		},
		newDump: func(options []string) (archiveDump, func(), error) {
			dumpDir := archiveOutPath(t, options)
			return &fakeArchiveDump{onDump: func() {
				if err := os.MkdirAll(dumpDir, 0o700); err != nil {
					t.Fatalf("MkdirAll(%q) error = %v", dumpDir, err)
				}
			}}, func() {}, nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			return []storage.ArchiveBackend{&recordingStorage{}}, nil
		},
		tar: func(_ string, destination string) error {
			tarFile = destination
			return os.WriteFile(destination, []byte("tar"), 0o600)
		},
		buildObjectName: func(string, string) (string, error) {
			return "backup.tar.gz", nil
		},
		upload: func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
			return primaryErr
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile: func(path string) error {
			if path == tarFile {
				return cleanupErr
			}
			return utils.DeleteFile(path)
		},
		handleInterrupt: func(func()) chan struct{} { return nil },
	}

	err := pipeline.run(context.Background(), &mongoarchive.Config{})
	if !errors.Is(err, primaryErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, primaryErr)
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("run() error = %v, want wrapped cleanup %v", err, cleanupErr)
	}
	if !strings.Contains(err.Error(), "cleanup failed") {
		t.Fatalf("run() error = %v, want cleanup context", err)
	}
}

func TestArchivePipelineSkipsSuccessNotificationAfterCloseFailure(t *testing.T) {
	closeErr := errors.New("close failed")
	backend := &recordingStorage{closeErr: closeErr}
	root := t.TempDir()
	var notifications []string

	pipeline := archivePipeline{
		createWorkspace: func(string) (string, error) {
			return os.MkdirTemp(root, "run-")
		},
		newFilename: func() (string, string) {
			return "backup.tar.gz", "dumpdir"
		},
		newDump: func(options []string) (archiveDump, func(), error) {
			dumpDir := archiveOutPath(t, options)
			return &fakeArchiveDump{onDump: func() {
				if err := os.MkdirAll(dumpDir, 0o700); err != nil {
					t.Fatalf("MkdirAll(%q) error = %v", dumpDir, err)
				}
			}}, func() {}, nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			return []storage.ArchiveBackend{backend}, nil
		},
		tar: func(_ string, destination string) error {
			return os.WriteFile(destination, []byte("tar"), 0o600)
		},
		buildObjectName: func(string, string) (string, error) {
			return "backup.tar.gz", nil
		},
		upload: func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
			return nil
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: func(func()) chan struct{} { return nil },
		notify: notificationSenderFunc(func(_ context.Context, success bool, filenameOrError string) {
			notifications = append(notifications, fmt.Sprintf("%t:%s", success, filenameOrError))
		}),
	}

	err := pipeline.run(context.Background(), &mongoarchive.Config{})
	if !errors.Is(err, closeErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, closeErr)
	}
	if len(notifications) != 0 {
		t.Fatalf("run() notifications = %#v, want no success notification on cleanup failure", notifications)
	}
}

func TestCreateArchiveWorkspaceUsesPortablePrivateDefaults(t *testing.T) {
	t.Setenv(envPrefix+"DUMP_PATH", "")
	workspace, err := createArchiveWorkspace("")
	if err != nil {
		t.Fatalf("createArchiveWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	wantPrefix := filepath.Join(os.TempDir(), defaultWorkspaceDir) + string(os.PathSeparator)
	if !strings.HasPrefix(workspace, wantPrefix) {
		t.Fatalf("createArchiveWorkspace() = %q, want prefix %q", workspace, wantPrefix)
	}

	info, err := os.Stat(workspace)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", workspace, err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("workspace permissions = %#o, want 0700", info.Mode().Perm())
	}
}

func TestCreateArchiveWorkspaceUsesOverrideBasePath(t *testing.T) {
	basePath := t.TempDir()
	t.Setenv(envPrefix+"DUMP_PATH", basePath)

	workspace, err := createArchiveWorkspace(basePath)
	if err != nil {
		t.Fatalf("createArchiveWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	wantPrefix := basePath + string(os.PathSeparator)
	if !strings.HasPrefix(workspace, wantPrefix) {
		t.Fatalf("createArchiveWorkspace() = %q, want prefix %q", workspace, wantPrefix)
	}
}

func TestCreateArchiveWorkspaceUsesParsedBasePathInsteadOfEnvironment(t *testing.T) {
	parsedBasePath := t.TempDir()
	envBasePath := t.TempDir()
	t.Setenv(envPrefix+"DUMP_PATH", envBasePath)

	workspace, err := createArchiveWorkspace(parsedBasePath)
	if err != nil {
		t.Fatalf("createArchiveWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	if !strings.HasPrefix(workspace, parsedBasePath+string(os.PathSeparator)) {
		t.Fatalf("createArchiveWorkspace() = %q, want parsed base path %q", workspace, parsedBasePath)
	}
	if strings.HasPrefix(workspace, envBasePath+string(os.PathSeparator)) {
		t.Fatalf("createArchiveWorkspace() = %q, unexpectedly used process environment", workspace)
	}
}

func TestCronRuntimeReturnsSetupErrors(t *testing.T) {
	t.Run("invalid timezone", func(t *testing.T) {
		runtime := cronRuntime{
			newScheduler: func(*time.Location) (cronScheduler, error) {
				return &fakeCronScheduler{}, nil
			},
			runTask: func(context.Context, *mongoarchive.Config, notificationSender) error { return nil },
			notify:  notificationSenderFunc(func(context.Context, bool, string) {}),
			now:     time.Now,
		}

		err := runtime.run(context.Background(), &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: nil, CronExpression: "* * * * *"}})
		if err == nil {
			t.Fatal("run() expected error")
		}
	})

	t.Run("invalid cron expression", func(t *testing.T) {
		cronErr := errors.New("bad cron expression")
		runtime := cronRuntime{
			newScheduler: func(*time.Location) (cronScheduler, error) {
				return &fakeCronScheduler{scheduleErr: cronErr}, nil
			},
			runTask: func(context.Context, *mongoarchive.Config, notificationSender) error { return nil },
			notify:  notificationSenderFunc(func(context.Context, bool, string) {}),
			now:     time.Now,
		}

		err := runtime.run(context.Background(), &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: time.UTC, CronExpression: "bad cron"}})
		if !errors.Is(err, cronErr) {
			t.Fatalf("run() error = %v, want wrapped %v", err, cronErr)
		}
	})
}

func TestCronRuntimeContextCancellationStopsSchedulerAndCancelsActiveTask(t *testing.T) {
	scheduler := &fakeCronScheduler{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	activeStarted := make(chan struct{})
	activeCanceled := make(chan error, 1)

	runtime := cronRuntime{
		newScheduler: func(*time.Location) (cronScheduler, error) {
			return scheduler, nil
		},
		runTask: func(ctx context.Context, _ *mongoarchive.Config, _ notificationSender) error {
			close(activeStarted)
			<-ctx.Done()
			activeCanceled <- ctx.Err()
			return ctx.Err()
		},
		notify: notificationSenderFunc(func(context.Context, bool, string) {}),
		now:    time.Now,
	}

	runDone := make(chan error, 1)
	go func() {
		runDone <- runtime.run(ctx, &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: time.UTC, CronExpression: "* * * * *"}})
	}()

	select {
	case <-scheduler.started:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not start")
	}
	scheduler.trigger()
	select {
	case <-activeStarted:
	case <-time.After(time.Second):
		t.Fatal("scheduled task did not start")
	}

	cancel()

	select {
	case err := <-activeCanceled:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("active task context error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("active task did not observe cancellation")
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run() error = %v, want nil for ordinary cancellation shutdown", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run() did not return after context cancellation")
	}

	if got := scheduler.shutdowns.Load(); got != 1 {
		t.Fatalf("scheduler shutdowns = %d, want 1", got)
	}
}

func TestCronRuntimeSkipsOverlappingRuns(t *testing.T) {
	scheduler := &fakeCronScheduler{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 1)
	finished := make(chan struct{}, 1)
	release := make(chan struct{})
	var runs atomic.Int32
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	runtime := cronRuntime{
		newScheduler: func(*time.Location) (cronScheduler, error) {
			return scheduler, nil
		},
		runTask: func(context.Context, *mongoarchive.Config, notificationSender) error {
			runs.Add(1)
			current := concurrent.Add(1)
			if current > maxConcurrent.Load() {
				maxConcurrent.Store(current)
			}
			started <- struct{}{}
			<-release
			concurrent.Add(-1)
			finished <- struct{}{}
			return nil
		},
		notify: notificationSenderFunc(func(context.Context, bool, string) {}),
		now: func() time.Time {
			return time.Unix(0, 0)
		},
	}

	runDone := make(chan error, 1)
	go func() {
		runDone <- runtime.run(ctx, &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: time.UTC, CronExpression: "* * * * *"}})
	}()

	select {
	case <-scheduler.started:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not start")
	}
	scheduler.trigger()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first scheduled task did not start")
	}
	scheduler.trigger()
	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("scheduled task did not finish")
	}
	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run() did not return after context cancellation")
	}

	scheduler.mu.Lock()
	overlap := scheduler.overlap
	scheduler.mu.Unlock()
	if overlap != cronSkipOverlappingRuns {
		t.Fatalf("overlap policy = %q, want %q", overlap, cronSkipOverlappingRuns)
	}
	if runs.Load() != 1 {
		t.Fatalf("run count = %d, want 1", runs.Load())
	}
	if maxConcurrent.Load() != 1 {
		t.Fatalf("max concurrent runs = %d, want 1", maxConcurrent.Load())
	}
}

func archiveOutPath(t *testing.T, options []string) string {
	t.Helper()
	for _, option := range options {
		if strings.HasPrefix(option, "--out=") {
			return strings.TrimPrefix(option, "--out=")
		}
	}
	t.Fatal("missing --out option")
	return ""
}

func assertPathState(t *testing.T, path string, wantExists bool) {
	t.Helper()
	if path == "" {
		if wantExists {
			t.Fatal("assertPathState() missing path for expected artifact")
		}
		return
	}

	_, err := os.Stat(path)
	gotExists := err == nil
	if gotExists != wantExists {
		t.Fatalf("path %q exists = %v, want %v (err=%v)", path, gotExists, wantExists, err)
	}
}
