package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/mongounarchive"
	projectstorage "github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type restoreStorageStub struct {
	name       string
	objectName string
}

func (s *restoreStorageStub) Download(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *restoreStorageStub) GetTargetObjectName(_ context.Context, name string) (string, error) {
	if s.objectName != "" {
		return s.objectName, nil
	}
	return name, nil
}

func (s *restoreStorageStub) Close() error { return nil }

func (s *restoreStorageStub) BackendName() string {
	if s.name != "" {
		return s.name
	}
	return projectstorage.BackendLocal
}

func TestRestorePipelineRejectsIncompleteStorageBeforeSideEffects(t *testing.T) {
	workspaceCreated := false
	storagesConstructed := false
	pipeline := restorePipeline{
		createWorkspace: func(string) (string, error) {
			workspaceCreated = true
			return t.TempDir(), nil
		},
		getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			storagesConstructed = true
			return []projectstorage.RestoreBackend{&restoreStorageStub{}}, nil
		},
	}

	err := pipeline.run(context.Background(), &mongounarchive.Config{
		StorageOptions: toolconfig.StorageOptions{AZAccountName: "restore-account"},
	})
	if err == nil {
		t.Fatal("run() expected error")
	}
	if !strings.Contains(err.Error(), "AZ_ACCOUNT_KEY") || strings.Contains(err.Error(), "restore-account") {
		t.Fatalf("run() error = %q, want missing identifiers without supplied value", err)
	}
	if workspaceCreated {
		t.Fatal("run() created workspace before rejecting incomplete storage config")
	}
	if storagesConstructed {
		t.Fatal("run() constructed storage before rejecting incomplete storage config")
	}
}

func TestRestorePipelineRejectsInvalidRuntimeBeforeSideEffects(t *testing.T) {
	workspaceCreated := false
	storagesConstructed := false
	pipeline := restorePipeline{
		createWorkspace: func(string) (string, error) {
			workspaceCreated = true
			return t.TempDir(), nil
		},
		getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			storagesConstructed = true
			return []projectstorage.RestoreBackend{&restoreStorageStub{}}, nil
		},
	}

	limits := utils.DefaultArchiveExtractionLimits()
	limits.MaxEntries = 0
	err := pipeline.run(context.Background(), &mongounarchive.Config{
		RuntimeOptions: mongounarchive.RuntimeOptions{ArchiveExtractionLimits: limits},
	})
	if err == nil {
		t.Fatal("run() expected runtime validation error")
	}
	if workspaceCreated {
		t.Fatal("run() created workspace before rejecting invalid runtime config")
	}
	if storagesConstructed {
		t.Fatal("run() constructed storage before rejecting invalid runtime config")
	}
}

type fakeRestoreRunner struct {
	result       restoreExecutionResult
	acknowledged bool
	closeErr     error
	closed       bool
}

func (r *fakeRestoreRunner) HandleInterrupt() {}

func (r *fakeRestoreRunner) Restore() restoreExecutionResult {
	return r.result
}

func (r *fakeRestoreRunner) Close() error {
	r.closed = true
	return r.closeErr
}

func (r *fakeRestoreRunner) Acknowledged() bool {
	return r.acknowledged
}

type fakeUpdateCollection struct {
	updateMany func(context.Context, any, any, ...mongooptions.Lister[mongooptions.UpdateManyOptions]) (*mongo.UpdateResult, error)
}

func (c fakeUpdateCollection) UpdateMany(ctx context.Context, filter any, update any, opts ...mongooptions.Lister[mongooptions.UpdateManyOptions]) (*mongo.UpdateResult, error) {
	return c.updateMany(ctx, filter, update, opts...)
}

type fakeUpdateDatabase struct {
	collection updateCollection
	name       string
}

func (d *fakeUpdateDatabase) Collection(name string, _ ...mongooptions.Lister[mongooptions.CollectionOptions]) updateCollection {
	d.name = name
	return d.collection
}

func TestRestorePipelineCleanupByStage(t *testing.T) {
	downloadErr := errors.New("download failed")
	extractErr := errors.New("extract failed")
	restoreErr := errors.New("restore failed")
	updateErr := errors.New("update failed")

	tests := []struct {
		name          string
		keep          bool
		downloadErr   error
		extractErr    error
		restoreErr    error
		updateErr     error
		updates       string
		wantErr       error
		wantWorkspace bool
		wantTarFile   bool
		wantDumpDir   bool
	}{
		{
			name:        "download failure cleans up workspace",
			downloadErr: downloadErr,
			wantErr:     downloadErr,
		},
		{
			name:       "extract failure cleans up downloaded archive",
			extractErr: extractErr,
			wantErr:    extractErr,
		},
		{
			name:       "restore failure cleans up extracted files",
			restoreErr: restoreErr,
			wantErr:    restoreErr,
		},
		{
			name:      "update failure cleans up restore artifacts",
			updateErr: updateErr,
			updates:   `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`,
			wantErr:   updateErr,
		},
		{
			name:          "keep preserves artifacts after update failure",
			keep:          true,
			updateErr:     updateErr,
			updates:       `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`,
			wantErr:       updateErr,
			wantWorkspace: true,
			wantTarFile:   true,
			wantDumpDir:   true,
		},
		{
			name:          "keep preserves artifacts after success",
			keep:          true,
			wantWorkspace: true,
			wantTarFile:   true,
			wantDumpDir:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			storageStub := &restoreStorageStub{objectName: "backups/archive.tar.gz"}
			var workspace string
			var tarFile string
			var dumpDir string

			pipeline := restorePipeline{
				createWorkspace: func(string) (string, error) {
					var err error
					workspace, err = os.MkdirTemp(root, "run-")
					return workspace, err
				},
				getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
					return []projectstorage.RestoreBackend{storageStub}, nil
				},
				selectStorage: func(storages []projectstorage.RestoreBackend, backend string) (projectstorage.RestoreBackend, error) {
					return storages[0], nil
				},
				getExtractionLimit: func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
					return utils.DefaultArchiveExtractionLimits(), nil
				},
				download: func(_ context.Context, _ projectstorage.RestoreStorage, _ string, destination string) error {
					tarFile = destination
					if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
						return err
					}
					if err := os.WriteFile(destination, []byte("archive"), 0o600); err != nil {
						return err
					}
					return tt.downloadErr
				},
				extract: func(_ string, destination string, _ utils.ArchiveExtractionLimits) error {
					dumpDir = destination
					if err := os.MkdirAll(destination, 0o700); err != nil {
						return err
					}
					dataFile := filepath.Join(destination, "data.bson")
					if err := os.WriteFile(dataFile, []byte("restore"), 0o600); err != nil {
						return err
					}
					return tt.extractErr
				},
				newRestore: func([]string) (restoreRunner, error) {
					return &fakeRestoreRunner{
						acknowledged: true,
						result: restoreExecutionResult{
							Successes: 1,
							Err:       tt.restoreErr,
						},
					}, nil
				},
				applyUpdates: func(context.Context, *mongounarchive.Config, []update) error {
					return tt.updateErr
				},
				deleteDirectory: utils.DeleteDirectory,
				deleteFile:      utils.DeleteFile,
				handleInterrupt: func(func()) chan struct{} { return nil },
			}

			cfg := &mongounarchive.Config{
				StorageOptions: toolconfig.StorageOptions{StorageBackend: "local"},
				UpdateOptions:  mongounarchive.UpdateOptions{Updates: tt.updates},
				Keep:           tt.keep,
			}
			err := pipeline.run(context.Background(), cfg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, want wrapped %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("run() unexpected error = %v", err)
			}

			assertPathState(t, workspace, tt.wantWorkspace)
			assertPathState(t, tarFile, tt.wantTarFile)
			assertPathState(t, dumpDir, tt.wantDumpDir)
		})
	}
}

func TestRestorePipelinePartialResultFailsBeforeUpdatesAndSuccessLog(t *testing.T) {
	logs := []string{}
	updatesApplied := false
	pipeline := newRestorePipelineTestDouble(t, restoreExecutionResult{Successes: 7, Failures: 2}, true)
	pipeline.applyUpdates = func(context.Context, *mongounarchive.Config, []update) error {
		updatesApplied = true
		return nil
	}
	pipeline.logAlways = func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	err := pipeline.run(context.Background(), &mongounarchive.Config{
		StorageOptions: toolconfig.StorageOptions{StorageBackend: "local"},
		UpdateOptions:  mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`},
	})
	var partialErr restorePartialResultError
	if !errors.As(err, &partialErr) {
		t.Fatalf("run() error = %T %[1]v, want restorePartialResultError", err)
	}
	if partialErr.Successes != 7 || partialErr.Failures != 2 {
		t.Fatalf("partial error = %+v, want successes=7 failures=2", partialErr)
	}
	if !strings.Contains(err.Error(), "7 document(s) restored successfully") || !strings.Contains(err.Error(), "2 document(s) failed to restore") {
		t.Fatalf("run() error = %v, want successes and failures", err)
	}
	if updatesApplied {
		t.Fatal("run() applied updates after partial restore result")
	}
	for _, log := range logs {
		if strings.Contains(log, "Unarchive completed successfully") {
			t.Fatalf("run() logged completion success after partial restore: %q", log)
		}
	}
}

func TestRestorePipelineZeroFailureResultAppliesUpdatesAndSucceeds(t *testing.T) {
	tests := []struct {
		name         string
		acknowledged bool
		wantLog      string
	}{
		{name: "acknowledged", acknowledged: true, wantLog: "3 document(s) restored successfully. 0 document(s) failed to restore."},
		{name: "unacknowledged", acknowledged: false, wantLog: "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := []string{}
			updatesApplied := false
			pipeline := newRestorePipelineTestDouble(t, restoreExecutionResult{Successes: 3}, tt.acknowledged)
			pipeline.applyUpdates = func(context.Context, *mongounarchive.Config, []update) error {
				updatesApplied = true
				return nil
			}
			pipeline.logAlways = func(format string, args ...any) {
				logs = append(logs, fmt.Sprintf(format, args...))
			}

			err := pipeline.run(context.Background(), &mongounarchive.Config{
				StorageOptions: toolconfig.StorageOptions{StorageBackend: "local"},
				UpdateOptions:  mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`},
			})
			if err != nil {
				t.Fatalf("run() error = %v, want nil", err)
			}
			if !updatesApplied {
				t.Fatal("run() did not apply updates after zero-failure restore")
			}
			if !containsLog(logs, tt.wantLog) {
				t.Fatalf("logs = %v, want %q", logs, tt.wantLog)
			}
			if !containsLog(logs, "Unarchive completed successfully") {
				t.Fatalf("logs = %v, want completion success", logs)
			}
		})
	}
}

func TestRestorePipelineAggregatesCleanupFailure(t *testing.T) {
	primaryErr := errors.New("restore failed")
	cleanupErr := errors.New("remove archive failed")
	root := t.TempDir()
	storageStub := &restoreStorageStub{objectName: "backups/archive.tar.gz"}
	var tarFile string

	pipeline := restorePipeline{
		createWorkspace: func(string) (string, error) {
			return os.MkdirTemp(root, "run-")
		},
		getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			return []projectstorage.RestoreBackend{storageStub}, nil
		},
		selectStorage: func(storages []projectstorage.RestoreBackend, backend string) (projectstorage.RestoreBackend, error) {
			return storages[0], nil
		},
		getExtractionLimit: func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
			return utils.DefaultArchiveExtractionLimits(), nil
		},
		download: func(_ context.Context, _ projectstorage.RestoreStorage, _ string, destination string) error {
			tarFile = destination
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				return err
			}
			return os.WriteFile(destination, []byte("archive"), 0o600)
		},
		extract: func(_ string, destination string, _ utils.ArchiveExtractionLimits) error {
			return os.MkdirAll(destination, 0o700)
		},
		newRestore: func([]string) (restoreRunner, error) {
			return &fakeRestoreRunner{acknowledged: true, result: restoreExecutionResult{Err: primaryErr}}, nil
		},
		applyUpdates:    func(context.Context, *mongounarchive.Config, []update) error { return nil },
		deleteDirectory: utils.DeleteDirectory,
		deleteFile: func(path string) error {
			if path == tarFile {
				return cleanupErr
			}
			return utils.DeleteFile(path)
		},
		handleInterrupt: func(func()) chan struct{} { return nil },
	}

	err := pipeline.run(context.Background(), &mongounarchive.Config{StorageOptions: toolconfig.StorageOptions{StorageBackend: "local"}})
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

func TestRestorePipelinePropagatesCancellationToUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storageStub := &restoreStorageStub{objectName: "backups/archive.tar.gz"}
	started := make(chan struct{}, 1)
	pipeline := restorePipeline{
		createWorkspace: func(string) (string, error) { return t.TempDir(), nil },
		getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			return []projectstorage.RestoreBackend{storageStub}, nil
		},
		selectStorage: func(storages []projectstorage.RestoreBackend, backend string) (projectstorage.RestoreBackend, error) {
			return storages[0], nil
		},
		getExtractionLimit: func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
			return utils.DefaultArchiveExtractionLimits(), nil
		},
		download: func(_ context.Context, _ projectstorage.RestoreStorage, _ string, destination string) error {
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				return err
			}
			return os.WriteFile(destination, []byte("archive"), 0o600)
		},
		extract: func(_ string, destination string, _ utils.ArchiveExtractionLimits) error {
			return os.MkdirAll(destination, 0o700)
		},
		newRestore: func([]string) (restoreRunner, error) {
			return &fakeRestoreRunner{acknowledged: true, result: restoreExecutionResult{Successes: 1}}, nil
		},
		applyUpdates: func(ctx context.Context, _ *mongounarchive.Config, _ []update) error {
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
		errCh <- pipeline.run(ctx, &mongounarchive.Config{StorageOptions: toolconfig.StorageOptions{StorageBackend: "local"}, UpdateOptions: mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`}})
	}()

	<-started
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context canceled", err)
	}
}

func TestRestorePipelineRejectsInvalidUpdatesBeforeDownload(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *mongounarchive.Config
		wantErr string
	}{
		{
			name:    "dry run with updates",
			cfg:     &mongounarchive.Config{RestoreExecutionOptions: mongounarchive.RestoreExecutionOptions{DryRun: true}, UpdateOptions: mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`}},
			wantErr: "--dry-run cannot be combined",
		},
		{
			name:    "unknown field",
			cfg:     &mongounarchive.Config{UpdateOptions: mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}},"extra":true}]`}},
			wantErr: "unknown field",
		},
		{
			name:    "empty filter",
			cfg:     &mongounarchive.Config{UpdateOptions: mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{},"update":{"$set":{"active":false}}}]`}},
			wantErr: "filter must be a non-empty document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloaded := false
			pipeline := restorePipeline{
				createWorkspace: func(string) (string, error) { return t.TempDir(), nil },
				getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
					return []projectstorage.RestoreBackend{&restoreStorageStub{objectName: "archive.tar.gz"}}, nil
				},
				selectStorage: func(storages []projectstorage.RestoreBackend, backend string) (projectstorage.RestoreBackend, error) {
					return storages[0], nil
				},
				getExtractionLimit: func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
					return utils.DefaultArchiveExtractionLimits(), nil
				},
				download: func(context.Context, projectstorage.RestoreStorage, string, string) error {
					downloaded = true
					return nil
				},
				extract: func(string, string, utils.ArchiveExtractionLimits) error { return nil },
				newRestore: func([]string) (restoreRunner, error) {
					return &fakeRestoreRunner{acknowledged: true, result: restoreExecutionResult{Successes: 1}}, nil
				},
				applyUpdates:    func(context.Context, *mongounarchive.Config, []update) error { return nil },
				deleteDirectory: utils.DeleteDirectory,
				deleteFile:      utils.DeleteFile,
				handleInterrupt: func(func()) chan struct{} { return nil },
			}

			err := pipeline.run(context.Background(), tt.cfg)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("run() error = %v, want %q", err, tt.wantErr)
			}
			if downloaded {
				t.Fatal("run() reached download for invalid updates")
			}
		})
	}
}

func TestApplyValidatedUpdatesHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{}, 1)
	db := &fakeUpdateDatabase{
		collection: fakeUpdateCollection{updateMany: func(ctx context.Context, filter any, update any, _ ...mongooptions.Lister[mongooptions.UpdateManyOptions]) (*mongo.UpdateResult, error) {
			started <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- applyValidatedUpdates(ctx, db, []update{{Collection: "users", Filter: map[string]any{"active": true}, Update: map[string]any{"$set": map[string]any{"active": false}}}}, 0)
	}()

	<-started
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("applyValidatedUpdates() error = %v, want context canceled", err)
	}
	if db.name != "users" {
		t.Fatalf("Collection() name = %q, want users", db.name)
	}
}

func TestApplyValidatedUpdatesUsesParsedTimeoutInsteadOfEnvironment(t *testing.T) {
	t.Setenv(envPrefix+"UPDATE_TIMEOUT", "not-a-duration")

	db := &fakeUpdateDatabase{
		collection: fakeUpdateCollection{updateMany: func(ctx context.Context, filter any, update any, _ ...mongooptions.Lister[mongooptions.UpdateManyOptions]) (*mongo.UpdateResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}},
	}

	err := applyValidatedUpdates(context.Background(), db, []update{{Collection: "users", Filter: map[string]any{"active": true}, Update: map[string]any{"$set": map[string]any{"active": false}}}}, time.Nanosecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("applyValidatedUpdates() error = %v, want parsed timeout deadline", err)
	}
}

func TestParseUpdatesRejectsTrailingData(t *testing.T) {
	_, err := parseUpdates(&mongounarchive.Config{UpdateOptions: mongounarchive.UpdateOptions{Updates: `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}] {}`}})
	if err == nil || !strings.Contains(err.Error(), "unexpected trailing data") {
		t.Fatalf("parseUpdates() error = %v, want trailing data rejection", err)
	}
}

func TestCreateRestoreWorkspaceUsesPortablePrivateDefaults(t *testing.T) {
	t.Setenv(envPrefix+"RESTORE_PATH", "")
	workspace, err := createRestoreWorkspace("")
	if err != nil {
		t.Fatalf("createRestoreWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	wantPrefix := filepath.Join(os.TempDir(), defaultWorkspaceDir) + string(os.PathSeparator)
	if !strings.HasPrefix(workspace, wantPrefix) {
		t.Fatalf("createRestoreWorkspace() = %q, want prefix %q", workspace, wantPrefix)
	}

	info, err := os.Stat(workspace)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", workspace, err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("workspace permissions = %#o, want 0700", info.Mode().Perm())
	}
}

func TestCreateRestoreWorkspaceUsesOverrideBasePath(t *testing.T) {
	basePath := t.TempDir()
	t.Setenv(envPrefix+"RESTORE_PATH", basePath)

	workspace, err := createRestoreWorkspace(basePath)
	if err != nil {
		t.Fatalf("createRestoreWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	wantPrefix := basePath + string(os.PathSeparator)
	if !strings.HasPrefix(workspace, wantPrefix) {
		t.Fatalf("createRestoreWorkspace() = %q, want prefix %q", workspace, wantPrefix)
	}
}

func TestCreateRestoreWorkspaceUsesParsedBasePathInsteadOfEnvironment(t *testing.T) {
	parsedBasePath := t.TempDir()
	envBasePath := t.TempDir()
	t.Setenv(envPrefix+"RESTORE_PATH", envBasePath)

	workspace, err := createRestoreWorkspace(parsedBasePath)
	if err != nil {
		t.Fatalf("createRestoreWorkspace() error = %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	if !strings.HasPrefix(workspace, parsedBasePath+string(os.PathSeparator)) {
		t.Fatalf("createRestoreWorkspace() = %q, want parsed base path %q", workspace, parsedBasePath)
	}
	if strings.HasPrefix(workspace, envBasePath+string(os.PathSeparator)) {
		t.Fatalf("createRestoreWorkspace() = %q, unexpectedly used process environment", workspace)
	}
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

func newRestorePipelineTestDouble(t *testing.T, result restoreExecutionResult, acknowledged bool) restorePipeline {
	t.Helper()
	storageStub := &restoreStorageStub{objectName: "backups/archive.tar.gz"}
	return restorePipeline{
		createWorkspace: func(string) (string, error) { return t.TempDir(), nil },
		getStorages: func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			return []projectstorage.RestoreBackend{storageStub}, nil
		},
		selectStorage: func(storages []projectstorage.RestoreBackend, backend string) (projectstorage.RestoreBackend, error) {
			return storages[0], nil
		},
		getExtractionLimit: func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
			return utils.DefaultArchiveExtractionLimits(), nil
		},
		download: func(_ context.Context, _ projectstorage.RestoreStorage, _ string, destination string) error {
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				return err
			}
			return os.WriteFile(destination, []byte("archive"), 0o600)
		},
		extract: func(_ string, destination string, _ utils.ArchiveExtractionLimits) error {
			return os.MkdirAll(destination, 0o700)
		},
		newRestore: func([]string) (restoreRunner, error) {
			return &fakeRestoreRunner{acknowledged: acknowledged, result: result}, nil
		},
		applyUpdates:    func(context.Context, *mongounarchive.Config, []update) error { return nil },
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: func(func()) chan struct{} { return nil },
	}
}

func containsLog(logs []string, want string) bool {
	for _, log := range logs {
		if log == want {
			return true
		}
	}
	return false
}
