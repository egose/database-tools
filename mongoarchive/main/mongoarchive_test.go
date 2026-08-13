package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func (s *recordingStorage) Download(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *recordingStorage) GetTargetObjectName(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
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

func (s *blockingArchiveStorage) Download(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *blockingArchiveStorage) GetTargetObjectName(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *blockingArchiveStorage) DeleteOldObjects(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *blockingArchiveStorage) Close() error { return nil }

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
	expression  string
	overlap     cronOverlapPolicy
	task        func()
	running     atomic.Bool
	scheduleErr error
}

func (s *fakeCronScheduler) Schedule(expression string, task func(), overlap cronOverlapPolicy) error {
	s.expression = expression
	s.task = task
	s.overlap = overlap
	return s.scheduleErr
}

func (s *fakeCronScheduler) Start() {}

func (s *fakeCronScheduler) Shutdown() error {
	return nil
}

func (s *fakeCronScheduler) trigger() {
	if s.task == nil {
		return
	}
	if s.overlap == cronSkipOverlappingRuns {
		if !s.running.CompareAndSwap(false, true) {
			return
		}
		go func() {
			defer s.running.Store(false)
			s.task()
		}()
		return
	}
	go s.task()
}

func TestUploadBackupToStoragesUploadsBeforeRetention(t *testing.T) {
	backend := &recordingStorage{}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	if err := uploadBackupToStorages(context.Background(), []storage.Storage{backend}, objectName, "/tmp/archive.tar.gz"); err != nil {
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

	err := uploadBackupToStorages(context.Background(), []storage.Storage{backend}, objectName, "/tmp/archive.tar.gz")
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

	if err := uploadBackupToStorages(context.Background(), []storage.Storage{first, second}, objectName, "/tmp/archive.tar.gz"); err != nil {
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

	err := uploadBackupToStorages(context.Background(), []storage.Storage{first, second}, objectName, "/tmp/archive.tar.gz")
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

func TestUploadBackupToStoragesMultiBackendStopsBeforeLaterMutationWhenFirstUploadFails(t *testing.T) {
	uploadErr := errors.New("upload failed")
	var callLog []string
	first := &recordingStorage{name: "first", callLog: &callLog, uploadErr: uploadErr}
	second := &recordingStorage{name: "second", callLog: &callLog}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.Storage{first, second}, objectName, "/tmp/archive.tar.gz")
	if !errors.Is(err, uploadErr) {
		t.Fatalf("uploadBackupToStorages() error = %v, want wrapped %v", err, uploadErr)
	}
	if !strings.Contains(err.Error(), "retention was not run on any backend") {
		t.Fatalf("uploadBackupToStorages() error = %v, want explicit retention state", err)
	}

	want := []string{"first:upload:" + objectName}
	if !reflect.DeepEqual(callLog, want) {
		t.Fatalf("uploadBackupToStorages() call log = %#v, want %#v", callLog, want)
	}
}

func TestUploadBackupToStoragesReturnsDeletionFailure(t *testing.T) {
	deleteErr := errors.New("delete failed")
	backend := &recordingStorage{deleteErr: deleteErr}
	objectName := "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"

	err := uploadBackupToStorages(context.Background(), []storage.Storage{backend}, objectName, "/tmp/archive.tar.gz")
	if !errors.Is(err, deleteErr) {
		t.Fatalf("uploadBackupToStorages() error = %v, want wrapped %v", err, deleteErr)
	}
}

func TestUploadBackupToStoragesHonorsConfiguredDeadline(t *testing.T) {
	t.Setenv(envPrefix+"STORAGE_OPERATION_TIMEOUT", "20ms")

	start := time.Now()
	err := uploadBackupToStorages(context.Background(), []storage.Storage{&blockingArchiveStorage{}}, "backup.tar.gz", "/tmp/archive.tar.gz")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("uploadBackupToStorages() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("uploadBackupToStorages() elapsed = %v, want prompt timeout", elapsed)
	}
}

func TestArchivePipelinePropagatesCancellationToUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 1)
	pipeline := archivePipeline{
		createWorkspace: func() (string, error) { return t.TempDir(), nil },
		newFilename:     func() (string, string) { return "backup.tar.gz", "dumpdir" },
		newDump: func([]string) (archiveDump, func(), error) {
			return &fakeArchiveDump{}, func() {}, nil
		},
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.Storage, error) {
			return []storage.Storage{&recordingStorage{}}, nil
		},
		tar:             func(string, string) error { return nil },
		buildObjectName: func(string, string) (string, error) { return "backup.tar.gz", nil },
		upload: func(ctx context.Context, _ []storage.Storage, _ string, _ string) error {
			started <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: func(func()) chan struct{} { return nil },
		notify:          func(context.Context, *mongoarchive.Config, bool, string) {},
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
				createWorkspace: func() (string, error) {
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
				getStorages: func(context.Context, *mongoarchive.Config) ([]storage.Storage, error) {
					return []storage.Storage{&recordingStorage{}}, nil
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
				upload: func(context.Context, []storage.Storage, string, string) error {
					return tt.uploadErr
				},
				deleteDirectory: utils.DeleteDirectory,
				deleteFile:      utils.DeleteFile,
				handleInterrupt: func(func()) chan struct{} { return nil },
				notify:          func(context.Context, *mongoarchive.Config, bool, string) {},
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
		createWorkspace: func() (string, error) {
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
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.Storage, error) {
			return []storage.Storage{&recordingStorage{}}, nil
		},
		tar: func(_ string, destination string) error {
			tarFile = destination
			return os.WriteFile(destination, []byte("tar"), 0o600)
		},
		buildObjectName: func(string, string) (string, error) {
			return "backup.tar.gz", nil
		},
		upload: func(context.Context, []storage.Storage, string, string) error {
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
		notify:          func(context.Context, *mongoarchive.Config, bool, string) {},
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
		createWorkspace: func() (string, error) {
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
		getStorages: func(context.Context, *mongoarchive.Config) ([]storage.Storage, error) {
			return []storage.Storage{backend}, nil
		},
		tar: func(_ string, destination string) error {
			return os.WriteFile(destination, []byte("tar"), 0o600)
		},
		buildObjectName: func(string, string) (string, error) {
			return "backup.tar.gz", nil
		},
		upload: func(context.Context, []storage.Storage, string, string) error {
			return nil
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: func(func()) chan struct{} { return nil },
		notify: func(_ context.Context, _ *mongoarchive.Config, success bool, filenameOrError string) {
			notifications = append(notifications, fmt.Sprintf("%t:%s", success, filenameOrError))
		},
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
	workspace, err := createArchiveWorkspace()
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

	workspace, err := createArchiveWorkspace()
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

func TestCronRuntimeReturnsSetupErrors(t *testing.T) {
	t.Run("invalid timezone", func(t *testing.T) {
		runtime := cronRuntime{
			newScheduler: func(*time.Location) (cronScheduler, error) {
				return &fakeCronScheduler{}, nil
			},
			runTask:         func(context.Context, *mongoarchive.Config) error { return nil },
			notify:          func(context.Context, *mongoarchive.Config, bool, string) {},
			waitForShutdown: func() {},
			now:             time.Now,
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
			runTask:         func(context.Context, *mongoarchive.Config) error { return nil },
			notify:          func(context.Context, *mongoarchive.Config, bool, string) {},
			waitForShutdown: func() {},
			now:             time.Now,
		}

		err := runtime.run(context.Background(), &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: time.UTC, CronExpression: "bad cron"}})
		if !errors.Is(err, cronErr) {
			t.Fatalf("run() error = %v, want wrapped %v", err, cronErr)
		}
	})
}

func TestCronRuntimeSkipsOverlappingRuns(t *testing.T) {
	scheduler := &fakeCronScheduler{}
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
		runTask: func(context.Context, *mongoarchive.Config) error {
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
		notify: func(context.Context, *mongoarchive.Config, bool, string) {},
		waitForShutdown: func() {
			scheduler.trigger()
			<-started
			scheduler.trigger()
			close(release)
			<-finished
		},
		now: func() time.Time {
			return time.Unix(0, 0)
		},
	}

	err := runtime.run(context.Background(), &mongoarchive.Config{ScheduleOptions: mongoarchive.ScheduleOptions{Location: time.UTC, CronExpression: "* * * * *"}})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if scheduler.overlap != cronSkipOverlappingRuns {
		t.Fatalf("overlap policy = %q, want %q", scheduler.overlap, cronSkipOverlappingRuns)
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
