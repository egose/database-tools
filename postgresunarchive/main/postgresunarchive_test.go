package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/postgresarchiveformat"
	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/postgresunarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

const testObjectName = "postgres-archive/9987654320999-2026-08-24T010203.456Z.tar.gz"

type fakeBackend struct {
	name          string
	objectName    string
	lookupErr     error
	downloadErr   error
	closeErr      error
	lookup        func(context.Context, string) (string, error)
	download      func(context.Context, string, string) error
	requestedName string
	downloaded    string
	closed        bool
}

func (b *fakeBackend) BackendName() string {
	if b.name == "" {
		return storage.BackendLocal
	}
	return b.name
}
func (b *fakeBackend) GetTargetObjectName(ctx context.Context, requested string) (string, error) {
	b.requestedName = requested
	if b.lookup != nil {
		return b.lookup(ctx, requested)
	}
	if b.lookupErr != nil {
		return "", b.lookupErr
	}
	if b.objectName != "" {
		return b.objectName, nil
	}
	return testObjectName, nil
}
func (b *fakeBackend) Download(ctx context.Context, objectName, destination string) error {
	b.downloaded = objectName
	if b.download != nil {
		return b.download(ctx, objectName, destination)
	}
	return b.downloadErr
}
func (b *fakeBackend) Close() error { b.closed = true; return b.closeErr }

type fakeRunner struct {
	called     bool
	ctx        context.Context
	connection postgresclient.ConnectionOptions
	path       string
	err        error
	restore    func(context.Context) error
}

type partialRestoreError struct{ cause error }

func (err partialRestoreError) Error() string            { return err.cause.Error() }
func (err partialRestoreError) Unwrap() error            { return err.cause }
func (partialRestoreError) PartialChangesPossible() bool { return true }

func (r *fakeRunner) Restore(ctx context.Context, connection postgresclient.ConnectionOptions, path string) error {
	r.called, r.ctx, r.connection, r.path = true, ctx, connection, path
	if r.restore != nil {
		return r.restore(ctx)
	}
	return r.err
}

func validConfig() *postgresunarchive.Config {
	return &postgresunarchive.Config{Connection: postgresclient.ConnectionOptions{Database: "inventory"}}
}

func testPipeline(t *testing.T, backend *fakeBackend, runner *fakeRunner) restorePipeline {
	t.Helper()
	return restorePipeline{
		createWorkspace: func(string) (string, error) { return os.MkdirTemp(t.TempDir(), "run-") },
		getStorages: func(context.Context, *postgresunarchive.Config) ([]storage.RestoreBackend, error) {
			return []storage.RestoreBackend{backend}, nil
		},
		selectStorage: storage.SelectRestoreStorage,
		download: func(ctx context.Context, storage storage.RestoreStorage, objectName, destination string) error {
			return storage.Download(ctx, objectName, destination)
		},
		validateArchive: validateArchiveFile,
		extract:         utils.UnTar,
		validateDump:    validateExtractedDump,
		newRunner:       func(string) restoreRunner { return runner },
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
	}
}

func TestRestorePipelineValidatesBeforeStorageWorkspaceOrProcess(t *testing.T) {
	storageCalled, workspaceCalled := false, false
	runner := &fakeRunner{}
	pipeline := restorePipeline{
		getStorages: func(context.Context, *postgresunarchive.Config) ([]storage.RestoreBackend, error) {
			storageCalled = true
			return nil, nil
		},
		createWorkspace: func(string) (string, error) { workspaceCalled = true; return "", nil },
	}
	secret := "not-a-real-secret" // pragma: allowlist secret
	err := pipeline.run(context.Background(), &postgresunarchive.Config{Connection: postgresclient.ConnectionOptions{
		URI: "postgresql://restore:" + secret + "@db/inventory", Host: "other",
	}})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("run() error = %v", err)
	}
	if storageCalled || workspaceCalled || runner.called || strings.Contains(err.Error(), secret) {
		t.Fatalf("invalid configuration caused side effects or leaked a secret: storage=%v workspace=%v runner=%v error=%v", storageCalled, workspaceCalled, runner.called, err)
	}
}

func TestRestorePipelineExplicitObjectAndBackendSelection(t *testing.T) {
	first, selected := &fakeBackend{name: storage.BackendAWS}, &fakeBackend{name: storage.BackendLocal}
	runner := &fakeRunner{}
	pipeline := testPipeline(t, selected, runner)
	pipeline.getStorages = func(context.Context, *postgresunarchive.Config) ([]storage.RestoreBackend, error) {
		return []storage.RestoreBackend{first, selected}, nil
	}
	pipeline.download = func(_ context.Context, backend storage.RestoreStorage, objectName, destination string) error {
		if backend != selected {
			t.Fatalf("selected backend = %T, want local", backend)
		}
		return writeValidArchive(destination)
	}
	cfg := validConfig()
	cfg.StorageBackend = storage.BackendLocal
	cfg.ObjectName = testObjectName
	if err := pipeline.run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if selected.requestedName != testObjectName || first.requestedName != "" || !runner.called {
		t.Fatalf("selection state: first=%q selected=%q runner=%v", first.requestedName, selected.requestedName, runner.called)
	}
}

func TestRestorePipelineRequiresExplicitSelectionForMultipleBackends(t *testing.T) {
	workspaceCalled := false
	pipeline := restorePipeline{
		getStorages: func(context.Context, *postgresunarchive.Config) ([]storage.RestoreBackend, error) {
			return []storage.RestoreBackend{&fakeBackend{name: storage.BackendAWS}, &fakeBackend{name: storage.BackendLocal}}, nil
		},
		selectStorage:   storage.SelectRestoreStorage,
		createWorkspace: func(string) (string, error) { workspaceCalled = true; return "", nil },
	}
	err := pipeline.run(context.Background(), validConfig())
	if err == nil || !strings.Contains(err.Error(), "specify --storage-backend") || workspaceCalled {
		t.Fatalf("run() error = %v, workspace=%v", err, workspaceCalled)
	}
}

func TestLatestSelectionIsRestrictedToPostgreSQLPrefix(t *testing.T) {
	root := t.TempDir()
	mongoName := "mongo-archive/9987654320999-2026-08-24T020203.456Z.tar.gz"
	postgresName := testObjectName
	writeObject := func(name string, modified time.Time) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("archive"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	writeObject(postgresName, time.Unix(1, 0))
	writeObject(mongoName, time.Unix(2, 0))
	cfg := validConfig()
	cfg.StorageOptions = toolconfig.StorageOptions{LocalPath: root}
	backends, err := cfg.GetStorages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := backends[0].GetTargetObjectName(context.Background(), "")
	if err != nil || got != postgresName {
		t.Fatalf("latest object = %q, error %v, want %q", got, err, postgresName)
	}
	got, err = backends[0].GetTargetObjectName(context.Background(), filepath.Base(postgresName))
	if err != nil || got != postgresName {
		t.Fatalf("explicit generated filename = %q, error %v, want %q", got, err, postgresName)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(postgresName))); err != nil {
		t.Fatal(err)
	}
	if _, err := backends[0].GetTargetObjectName(context.Background(), ""); err == nil {
		t.Fatal("MongoDB-only namespace was eligible for PostgreSQL latest selection")
	}
}

func TestInvalidPayloadsFailBeforeProcessInvocation(t *testing.T) {
	valid, err := postgresarchiveformat.NewManifest(time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC), "source", "17.4")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		entries []tarEntry
	}{
		{name: "missing manifest", entries: []tarEntry{{postgresarchiveformat.DumpPath, "PGDMPdump"}}},
		{name: "malformed manifest", entries: []tarEntry{{postgresarchiveformat.ManifestPath, "{"}, {postgresarchiveformat.DumpPath, "PGDMPdump"}}},
		{name: "unsupported version", entries: payloadEntries(manifestJSON(t, mutateManifest(valid, func(m *postgresarchiveformat.Manifest) { m.FormatVersion++ })), "PGDMPdump")},
		{name: "wrong family", entries: payloadEntries(manifestJSON(t, mutateManifest(valid, func(m *postgresarchiveformat.Manifest) { m.DatabaseFamily = "mongodb" })), "PGDMPdump")},
		{name: "inconsistent dump format", entries: payloadEntries(manifestJSON(t, valid), "not-a-custom-dump")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend, runner := &fakeBackend{}, &fakeRunner{}
			pipeline := testPipeline(t, backend, runner)
			pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
				return writeArchive(destination, test.entries)
			}
			err := pipeline.run(context.Background(), validConfig())
			if err == nil || runner.called {
				t.Fatalf("run() error = %v, runner called = %v", err, runner.called)
			}
			if !strings.Contains(err.Error(), "validate PostgreSQL archive") {
				t.Fatalf("run() error = %v", err)
			}
		})
	}
}

func TestDefensiveExtractionRunsBeforePayloadValidationAndProcess(t *testing.T) {
	tests := []struct {
		name   string
		limits utils.ArchiveExtractionLimits
		entry  string
		want   string
	}{
		{name: "entry count limit", limits: utils.ArchiveExtractionLimits{MaxEntries: 1, MaxEntryBytes: 4096, MaxTotalBytes: 8192}, entry: postgresarchiveformat.ManifestPath, want: "too many entries"},
		{name: "path traversal", limits: utils.DefaultArchiveExtractionLimits(), entry: "../manifest.json", want: "escapes destination"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend, runner := &fakeBackend{}, &fakeRunner{}
			validated := false
			pipeline := testPipeline(t, backend, runner)
			pipeline.validateArchive = func(string) error { validated = true; return nil }
			pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
				return writeArchive(destination, []tarEntry{{test.entry, `{}`}, {postgresarchiveformat.DumpPath, "PGDMPdump"}})
			}
			cfg := validConfig()
			cfg.ArchiveExtractionLimits = test.limits
			err := pipeline.run(context.Background(), cfg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("run() error = %v, want %q", err, test.want)
			}
			if validated || runner.called {
				t.Fatalf("unsafe extraction reached validation/process: validated=%v runner=%v", validated, runner.called)
			}
		})
	}
}

func TestRestorePipelineFailuresTimeoutCancellationCleanupKeepAndSuccess(t *testing.T) {
	downloadErr, extractErr, restoreErr := errors.New("download failed"), errors.New("extract failed"), errors.New("restore failed")
	tests := []struct {
		name       string
		keep       bool
		download   error
		extract    error
		restore    error
		want       error
		wantKept   bool
		wantRunner bool
	}{
		{name: "download failure", download: downloadErr, want: downloadErr},
		{name: "extraction failure", extract: extractErr, want: extractErr},
		{name: "process setup failure has no partial warning", restore: restoreErr, want: restoreErr, wantRunner: true},
		{name: "keep after process failure", keep: true, restore: partialRestoreError{restoreErr}, want: restoreErr, wantRunner: true, wantKept: true},
		{name: "success", wantRunner: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend, runner := &fakeBackend{}, &fakeRunner{err: test.restore}
			var workspace string
			pipeline := testPipeline(t, backend, runner)
			pipeline.createWorkspace = func(string) (string, error) {
				var err error
				workspace, err = os.MkdirTemp(t.TempDir(), "run-")
				return workspace, err
			}
			pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
				if test.download != nil {
					return test.download
				}
				return writeValidArchive(destination)
			}
			if test.extract != nil {
				pipeline.extract = func(string, string, utils.ArchiveExtractionLimits) error { return test.extract }
			}
			cfg := validConfig()
			cfg.Keep = test.keep
			err := pipeline.run(context.Background(), cfg)
			if !errors.Is(err, test.want) || runner.called != test.wantRunner {
				t.Fatalf("run() error = %v, runner=%v", err, runner.called)
			}
			var partial interface{ PartialChangesPossible() bool }
			partialExpected := errors.As(test.restore, &partial)
			if strings.Contains(errString(err), "partial database changes may exist") != partialExpected {
				t.Fatalf("process error partial-change warning accuracy = %v, want %v: %v", !partialExpected, partialExpected, err)
			}
			if _, statErr := os.Stat(workspace); (statErr == nil) != test.wantKept {
				t.Fatalf("workspace kept = %v, want %v (error %v)", statErr == nil, test.wantKept, statErr)
			}
		})
	}
}

func TestRestorePipelineCancellationReachesProcessAndWarnsOfPartialChanges(t *testing.T) {
	backend, runner := &fakeBackend{}, &fakeRunner{}
	started := make(chan struct{})
	runner.restore = func(ctx context.Context) error { close(started); <-ctx.Done(); return partialRestoreError{ctx.Err()} }
	pipeline := testPipeline(t, backend, runner)
	pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
		return writeValidArchive(destination)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pipeline.run(ctx, validConfig()) }()
	<-started
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "partial database changes may exist") {
		t.Fatalf("run() error = %v", err)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestRestorePipelineAppliesStorageTimeoutAndClosesBackend(t *testing.T) {
	for _, stage := range []string{"lookup", "download"} {
		t.Run(stage, func(t *testing.T) {
			backend, runner := &fakeBackend{}, &fakeRunner{}
			if stage == "lookup" {
				backend.lookup = func(ctx context.Context, _ string) (string, error) { <-ctx.Done(); return "", ctx.Err() }
			} else {
				backend.download = func(ctx context.Context, _, _ string) error { <-ctx.Done(); return ctx.Err() }
			}
			pipeline := testPipeline(t, backend, runner)
			cfg := validConfig()
			cfg.StorageOperationTimeout = time.Millisecond
			err := pipeline.run(context.Background(), cfg)
			if !errors.Is(err, context.DeadlineExceeded) || !backend.closed || runner.called {
				t.Fatalf("run() error=%v closed=%v runner=%v", err, backend.closed, runner.called)
			}
		})
	}
}

func TestRestorePipelinePreservesPrimaryAndCleanupErrors(t *testing.T) {
	primary, cleanupErr, closeErr := errors.New("restore failed"), errors.New("workspace cleanup failed"), errors.New("backend close failed")
	backend, runner := &fakeBackend{closeErr: closeErr}, &fakeRunner{err: primary}
	pipeline := testPipeline(t, backend, runner)
	pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
		return writeValidArchive(destination)
	}
	pipeline.deleteDirectory = func(string) error { return cleanupErr }
	err := pipeline.run(context.Background(), validConfig())
	for _, want := range []error{primary, cleanupErr, closeErr} {
		if !errors.Is(err, want) {
			t.Fatalf("run() error = %v, missing %v", err, want)
		}
	}
}

func TestRestoreSuccessUsesValidatedDumpInPrivateWorkspace(t *testing.T) {
	backend, runner := &fakeBackend{}, &fakeRunner{}
	var workspace string
	pipeline := testPipeline(t, backend, runner)
	pipeline.createWorkspace = func(base string) (string, error) {
		var err error
		workspace, err = createRestoreWorkspace(base)
		return workspace, err
	}
	pipeline.download = func(_ context.Context, _ storage.RestoreStorage, _, destination string) error {
		return writeValidArchive(destination)
	}
	cfg := validConfig()
	cfg.WorkspaceBasePath = t.TempDir()
	if err := pipeline.run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if !runner.called || runner.connection != cfg.Connection || filepath.Base(runner.path) != postgresarchiveformat.DumpPath {
		t.Fatalf("runner = %#v", runner)
	}
	if !strings.HasPrefix(runner.path, workspace+string(os.PathSeparator)) {
		t.Fatalf("dump path %q is outside workspace %q", runner.path, workspace)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace was not cleaned: %v", err)
	}
}

func TestCreateRestoreWorkspaceIsPrivate(t *testing.T) {
	workspace, err := createRestoreWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	info, err := os.Stat(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("workspace permissions = %#o, want 0700", info.Mode().Perm())
	}
}

type tarEntry struct{ name, body string }

func payloadEntries(manifest, dump string) []tarEntry {
	return []tarEntry{{postgresarchiveformat.ManifestPath, manifest}, {postgresarchiveformat.DumpPath, dump}}
}

func writeValidArchive(destination string) error {
	manifest, err := postgresarchiveformat.NewManifest(time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC), "source", "17.4")
	if err != nil {
		return err
	}
	return writeArchive(destination, payloadEntries(manifestJSONNoTest(manifest), "PGDMPcustom-dump"))
}

func mutateManifest(manifest postgresarchiveformat.Manifest, mutate func(*postgresarchiveformat.Manifest)) postgresarchiveformat.Manifest {
	mutate(&manifest)
	return manifest
}

func manifestJSON(t *testing.T, manifest postgresarchiveformat.Manifest) string {
	t.Helper()
	value := manifestJSONNoTest(manifest)
	if value == "" {
		t.Fatal("manifest encoding failed")
	}
	return value
}

func manifestJSONNoTest(manifest postgresarchiveformat.Manifest) string {
	var output strings.Builder
	if err := postgresarchiveformat.WriteManifest(&output, manifest); err != nil {
		// Invalid manifests must still be serializable for negative archive tests.
		return `{"format_version":` + strconv.Itoa(manifest.FormatVersion) + `,"database_family":"` + manifest.DatabaseFamily + `","dump_format":"` + manifest.DumpFormat + `","created_at":"` + manifest.CreatedAt.Format(time.RFC3339Nano) + `","source_database":"` + manifest.SourceDatabase + `","postgresql_client_version":"` + manifest.PostgreSQLClientVersion + `"}`
	}
	return output.String()
}

func writeArchive(destination string, entries []tarEntry) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); retErr == nil && err != nil {
			retErr = err
		}
	}()
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o600, Size: int64(len(entry.body)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		if _, err := io.WriteString(tarWriter, entry.body); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzipWriter.Close()
}
