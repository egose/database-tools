package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/archivedelivery"
	"github.com/egose/database-tools/internal/postgresarchiveformat"
	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/postgresarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

const testFilename = "9987654320999-2026-08-24T010203.456Z.tar.gz"

type fakeRunner struct {
	calls      *[]string
	version    string
	versionErr error
	dumpErr    error
	dump       func(context.Context, string) error
}

func (r *fakeRunner) Version(context.Context, postgresclient.Client) (string, error) {
	*r.calls = append(*r.calls, "version")
	return r.version, r.versionErr
}
func (r *fakeRunner) Dump(ctx context.Context, _ postgresclient.ConnectionOptions, path string) error {
	*r.calls = append(*r.calls, "dump")
	if r.dump != nil {
		if err := r.dump(ctx, path); err != nil {
			return err
		}
	}
	return r.dumpErr
}

type recordingBackend struct {
	name      string
	calls     *[]string
	uploadErr error
	deleteErr error
	closeErr  error
}

func (b *recordingBackend) Upload(_ context.Context, name, _ string) (string, error) {
	*b.calls = append(*b.calls, b.name+":upload:"+name)
	return "verified", b.uploadErr
}
func (b *recordingBackend) DeleteOldObjects(_ context.Context, name string) error {
	*b.calls = append(*b.calls, b.name+":delete:"+name)
	return b.deleteErr
}
func (b *recordingBackend) Close() error        { return b.closeErr }
func (b *recordingBackend) BackendName() string { return b.name }

func validConfig() *postgresarchive.Config {
	return &postgresarchive.Config{Connection: postgresclient.ConnectionOptions{Database: "inventory"}}
}

func testPipeline(t *testing.T, calls *[]string) archivePipeline {
	t.Helper()
	return archivePipeline{
		createWorkspace: func(string) (string, error) {
			*calls = append(*calls, "workspace")
			return os.MkdirTemp(t.TempDir(), "run-")
		},
		newFilename: func() (string, string) { return testFilename, "ignored" },
		newRunner:   func(string) dumpRunner { return &fakeRunner{calls: calls, version: "17.4"} },
		getStorages: func(context.Context, *postgresarchive.Config) ([]storage.ArchiveBackend, error) {
			*calls = append(*calls, "storage")
			return []storage.ArchiveBackend{&recordingBackend{name: "local", calls: calls}}, nil
		},
		newManifest: func(created time.Time, database, clientVersion string) (postgresarchiveformat.Manifest, error) {
			*calls = append(*calls, "manifest")
			return postgresarchiveformat.NewManifest(created, database, clientVersion)
		},
		writeManifest: func(string, postgresarchiveformat.Manifest) error {
			*calls = append(*calls, "write-manifest")
			return nil
		},
		tar: func(string, string) error { *calls = append(*calls, "tar"); return nil },
		buildObjectName: func(prefix, filename, defaultPrefix string) (string, error) {
			*calls = append(*calls, "object-name")
			return storage.BuildBackupObjectNameWithDefault(prefix, filename, defaultPrefix)
		},
		deliver: func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
			*calls = append(*calls, "deliver")
			return nil
		},
		deleteDirectory: utils.DeleteDirectory,
		now:             func() time.Time { return time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC) },
	}
}

func TestArchivePipelineValidatesBeforeWorkspaceStorageRunnerOrNotifier(t *testing.T) {
	var calls []string
	notified := false
	pipeline := testPipeline(t, &calls)
	pipeline.notify = notificationSenderFunc(func(context.Context, bool, string) { notified = true })
	secret := "not-a-real-secret" // pragma: allowlist secret
	err := pipeline.run(context.Background(), &postgresarchive.Config{Connection: postgresclient.ConnectionOptions{
		URI: "postgresql://backup:" + secret + "@db/inventory", Host: "other",
	}})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("run() error = %v", err)
	}
	if len(calls) != 0 || notified {
		t.Fatalf("invalid config side effects = %v, notified=%v", calls, notified)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("validation leaked secret: %v", err)
	}
}

func TestArchivePipelineStageFailuresNeverDeliverEarly(t *testing.T) {
	errs := map[string]error{
		"version": errors.New("version failed"), "dump": errors.New("dump failed"),
		"manifest": errors.New("manifest failed"), "write-manifest": errors.New("manifest write failed"),
		"tar": errors.New("tar failed"), "upload": errors.New("upload failed"), "retention": errors.New("retention failed"),
	}
	for _, stage := range []string{"version", "dump", "manifest", "write-manifest", "tar", "upload", "retention"} {
		t.Run(stage, func(t *testing.T) {
			var calls []string
			pipeline := testPipeline(t, &calls)
			runner := &fakeRunner{calls: &calls, version: "17.4"}
			if stage == "version" {
				runner.versionErr = errs[stage]
			}
			if stage == "dump" {
				runner.dumpErr = errs[stage]
			}
			pipeline.newRunner = func(string) dumpRunner { return runner }
			if stage == "manifest" {
				pipeline.newManifest = func(time.Time, string, string) (postgresarchiveformat.Manifest, error) {
					calls = append(calls, "manifest")
					return postgresarchiveformat.Manifest{}, errs[stage]
				}
			}
			if stage == "write-manifest" {
				pipeline.writeManifest = func(string, postgresarchiveformat.Manifest) error {
					calls = append(calls, "write-manifest")
					return errs[stage]
				}
			}
			if stage == "tar" {
				pipeline.tar = func(string, string) error { calls = append(calls, "tar"); return errs[stage] }
			}
			if stage == "upload" || stage == "retention" {
				pipeline.deliver = func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
					calls = append(calls, "deliver")
					return errs[stage]
				}
			}
			err := pipeline.run(context.Background(), validConfig())
			if !errors.Is(err, errs[stage]) {
				t.Fatalf("run() error = %v, want %v", err, errs[stage])
			}
			deliverIndex := indexOf(calls, "deliver")
			if stage != "upload" && stage != "retention" && deliverIndex != -1 {
				t.Fatalf("calls = %v, delivered before %s succeeded", calls, stage)
			}
			if stage == "dump" && indexOf(calls, "manifest") != -1 {
				t.Fatalf("manifest created after failed dump: %v", calls)
			}
			if stage == "manifest" && indexOf(calls, "tar") != -1 {
				t.Fatalf("tar created after manifest failure: %v", calls)
			}
		})
	}
}

func TestArchivePipelineSuccessCreatesValidatedPayloadAndPostgreSQLObject(t *testing.T) {
	var calls []string
	var objectName string
	var archivePath string
	var successValue string
	pipeline := testPipeline(t, &calls)
	pipeline.newRunner = func(string) dumpRunner {
		return &fakeRunner{calls: &calls, version: "17.4", dump: func(_ context.Context, path string) error {
			return os.WriteFile(path, []byte("PGDMPcustom-dump"), 0o600)
		}}
	}
	pipeline.writeManifest = writeManifest
	pipeline.tar = utils.Tar
	pipeline.deliver = func(_ context.Context, _ []storage.ArchiveBackend, object, path string, _ time.Duration) error {
		objectName, archivePath = object, path
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		manifest, err := postgresarchiveformat.ValidateArchive(file)
		if err != nil {
			return err
		}
		if manifest.SourceDatabase != "inventory" || manifest.PostgreSQLClientVersion != "17.4" {
			return errors.New("incorrect manifest metadata")
		}
		return nil
	}
	pipeline.notify = notificationSenderFunc(func(_ context.Context, success bool, value string) {
		if success {
			successValue = value
		}
	})
	if err := pipeline.run(context.Background(), validConfig()); err != nil {
		t.Fatal(err)
	}
	if objectName != storage.PostgreSQLDefaultBackupPrefix+testFilename {
		t.Fatalf("object name = %q", objectName)
	}
	if archivePath == "" || successValue != testFilename {
		t.Fatalf("archive/notification = %q/%q", archivePath, successValue)
	}
	if indexOf(calls, "dump") > indexOf(calls, "manifest") {
		t.Fatalf("manifest was created before dump: %v", calls)
	}
}

func TestArchivePipelineKeepAndCleanupFailureAggregation(t *testing.T) {
	t.Run("keep", func(t *testing.T) {
		var calls []string
		var workspace string
		primary := errors.New("upload failed")
		pipeline := testPipeline(t, &calls)
		pipeline.createWorkspace = func(string) (string, error) {
			workspace = filepath.Join(t.TempDir(), "kept")
			return workspace, os.Mkdir(workspace, 0o700)
		}
		pipeline.newRunner = func(string) dumpRunner {
			return &fakeRunner{calls: &calls, version: "17", dump: func(_ context.Context, path string) error { return os.WriteFile(path, []byte("PGDMPdump"), 0o600) }}
		}
		pipeline.writeManifest = writeManifest
		pipeline.tar = func(_, path string) error { return os.WriteFile(path, []byte("tar"), 0o600) }
		pipeline.deliver = func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error { return primary }
		cfg := validConfig()
		cfg.Keep = true
		if err := pipeline.run(context.Background(), cfg); !errors.Is(err, primary) {
			t.Fatalf("run() = %v", err)
		}
		if _, err := os.Stat(filepath.Join(workspace, "payload", postgresarchiveformat.DumpPath)); err != nil {
			t.Fatalf("kept dump missing: %v", err)
		}
		if _, err := os.Stat(filepath.Join(workspace, testFilename)); err != nil {
			t.Fatalf("kept archive missing: %v", err)
		}
	})

	t.Run("joined cleanup error", func(t *testing.T) {
		var calls []string
		primary, cleanupErr := errors.New("upload failed"), errors.New("remove failed")
		pipeline := testPipeline(t, &calls)
		pipeline.deliver = func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error { return primary }
		pipeline.deleteDirectory = func(string) error { return cleanupErr }
		err := pipeline.run(context.Background(), validConfig())
		if !errors.Is(err, primary) || !errors.Is(err, cleanupErr) || !strings.Contains(err.Error(), "cleanup failed") {
			t.Fatalf("run() error = %v", err)
		}
	})
}

func TestArchivePipelineCancellationReachesDumpAndPreventsDelivery(t *testing.T) {
	var calls []string
	started := make(chan struct{})
	pipeline := testPipeline(t, &calls)
	pipeline.newRunner = func(string) dumpRunner {
		return &fakeRunner{calls: &calls, version: "17", dump: func(ctx context.Context, _ string) error { close(started); <-ctx.Done(); return ctx.Err() }}
	}
	delivered := false
	pipeline.deliver = func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error {
		delivered = true
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pipeline.run(ctx, validConfig()) }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v", err)
	}
	if delivered {
		t.Fatal("canceled dump was delivered")
	}
}

func TestArchivePipelineUsesSharedStrictMultiBackendDelivery(t *testing.T) {
	var calls []string
	uploadErr := errors.New("second upload failed")
	backends := []storage.ArchiveBackend{
		&recordingBackend{name: "local", calls: &calls},
		&recordingBackend{name: "aws", calls: &calls, uploadErr: uploadErr},
	}
	pipeline := testPipeline(t, &calls)
	pipeline.getStorages = func(context.Context, *postgresarchive.Config) ([]storage.ArchiveBackend, error) { return backends, nil }
	pipeline.deliver = func(ctx context.Context, got []storage.ArchiveBackend, object, path string, timeout time.Duration) error {
		return archivedelivery.Deliver(ctx, got, object, path, timeout, nil)
	}
	err := pipeline.run(context.Background(), validConfig())
	if !errors.Is(err, uploadErr) || !strings.Contains(err.Error(), "retention was not run") {
		t.Fatalf("run() error = %v", err)
	}
	for _, call := range calls {
		if strings.Contains(call, ":delete:") {
			t.Fatalf("retention ran after partial upload: %v", calls)
		}
	}
	if indexContaining(calls, "local:upload:") == -1 || indexContaining(calls, "aws:upload:") == -1 {
		t.Fatalf("not all uploads attempted: %v", calls)
	}
}

type blockingBackend struct{ calls *[]string }

func (b *blockingBackend) Upload(ctx context.Context, _, _ string) (string, error) {
	*b.calls = append(*b.calls, "upload")
	<-ctx.Done()
	return "", ctx.Err()
}
func (*blockingBackend) DeleteOldObjects(context.Context, string) error { return nil }
func (*blockingBackend) Close() error                                   { return nil }

func TestArchivePipelineAppliesStorageOperationTimeout(t *testing.T) {
	var calls []string
	backend := &blockingBackend{calls: &calls}
	pipeline := testPipeline(t, &calls)
	pipeline.getStorages = func(context.Context, *postgresarchive.Config) ([]storage.ArchiveBackend, error) {
		return []storage.ArchiveBackend{backend}, nil
	}
	pipeline.deliver = func(ctx context.Context, got []storage.ArchiveBackend, object, path string, timeout time.Duration) error {
		return archivedelivery.Deliver(ctx, got, object, path, timeout, nil)
	}
	cfg := validConfig()
	cfg.StorageOperationTimeout = time.Millisecond
	if err := pipeline.run(context.Background(), cfg); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("run() error = %v, want deadline exceeded", err)
	}
	if calls[len(calls)-1] != "upload" {
		t.Fatalf("calls = %v", calls)
	}
}

type fakeScheduler struct {
	mu        sync.Mutex
	task      func()
	overlap   cronOverlapPolicy
	running   atomic.Bool
	started   chan struct{}
	shutdowns atomic.Int32
}

func (s *fakeScheduler) Schedule(_ string, task func(), overlap cronOverlapPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task, s.overlap = task, overlap
	return nil
}
func (s *fakeScheduler) Start()          { close(s.started) }
func (s *fakeScheduler) Shutdown() error { s.shutdowns.Add(1); return nil }
func (s *fakeScheduler) trigger() {
	s.mu.Lock()
	task, overlap := s.task, s.overlap
	s.mu.Unlock()
	if overlap == cronSkipOverlappingRuns && !s.running.CompareAndSwap(false, true) {
		return
	}
	go func() { defer s.running.Store(false); task() }()
}

func TestCronRuntimeSkipsOverlapAndRootCancellationStopsActiveRun(t *testing.T) {
	scheduler := &fakeScheduler{started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	active := make(chan struct{})
	canceled := make(chan error, 1)
	var runs atomic.Int32
	runtime := cronRuntime{
		newScheduler: func(*time.Location) (cronScheduler, error) { return scheduler, nil },
		runTask: func(ctx context.Context, _ *postgresarchive.Config, _ notificationSender) error {
			runs.Add(1)
			close(active)
			<-ctx.Done()
			canceled <- ctx.Err()
			return ctx.Err()
		},
		notify: notificationSenderFunc(func(context.Context, bool, string) {}), now: time.Now,
	}
	cfg := validConfig()
	cfg.ScheduleOptions = postgresarchive.ScheduleOptions{Cron: true, CronExpression: "* * * * *", Location: time.UTC}
	done := make(chan error, 1)
	go func() { done <- runtime.run(ctx, cfg) }()
	<-scheduler.started
	scheduler.trigger()
	<-active
	scheduler.trigger()
	cancel()
	if err := <-canceled; !errors.Is(err, context.Canceled) {
		t.Fatalf("active error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("cron run = %v", err)
	}
	if runs.Load() != 1 || scheduler.overlap != cronSkipOverlappingRuns || scheduler.shutdowns.Load() != 1 {
		t.Fatalf("cron state runs=%d overlap=%q shutdowns=%d", runs.Load(), scheduler.overlap, scheduler.shutdowns.Load())
	}
}

type lifecycleSender struct {
	sends  atomic.Int32
	closes atomic.Int32
}

func (s *lifecycleSender) Send(context.Context, bool, string) { s.sends.Add(1) }
func (s *lifecycleSender) Close()                             { s.closes.Add(1) }

func TestArchiveRuntimeValidatesBeforeNotifierAndOwnsNotifierOnce(t *testing.T) {
	sender := &lifecycleSender{}
	var constructions atomic.Int32
	runtime := archiveRuntime{
		newNotifications: func(*postgresarchive.Config) (notificationSender, error) { constructions.Add(1); return sender, nil },
		runTask: func(ctx context.Context, _ *postgresarchive.Config, got notificationSender) error {
			got.Send(ctx, true, testFilename)
			return nil
		},
		runCronJob: func(context.Context, *postgresarchive.Config, notificationSender) error { return nil },
	}
	if err := runtime.run(context.Background(), validConfig()); err != nil {
		t.Fatal(err)
	}
	if constructions.Load() != 1 || sender.sends.Load() != 1 || sender.closes.Load() != 1 {
		t.Fatalf("lifecycle constructions=%d sends=%d closes=%d", constructions.Load(), sender.sends.Load(), sender.closes.Load())
	}
	invalid := &postgresarchive.Config{}
	if err := runtime.run(context.Background(), invalid); err == nil {
		t.Fatal("invalid config accepted")
	}
	if constructions.Load() != 1 {
		t.Fatal("notifier constructed before invalid config rejection")
	}
}

func TestArchiveRuntimeReportsFailuresWithPostgreSQLWording(t *testing.T) {
	sender := &recordingSender{}
	failure := errors.New("dump unavailable")
	runtime := archiveRuntime{
		newNotifications: func(*postgresarchive.Config) (notificationSender, error) { return sender, nil },
		runTask:          func(context.Context, *postgresarchive.Config, notificationSender) error { return failure },
	}
	if err := runtime.run(context.Background(), validConfig()); !errors.Is(err, failure) {
		t.Fatalf("run() = %v", err)
	}
	if !reflect.DeepEqual(sender.values, []string{"PostgreSQL archive: dump unavailable"}) {
		t.Fatalf("notifications = %v", sender.values)
	}
}

type recordingSender struct{ values []string }

func (s *recordingSender) Send(_ context.Context, _ bool, value string) {
	s.values = append(s.values, value)
}

type fakeNotification struct {
	err         error
	calls       atomic.Int32
	sawDeadline atomic.Bool
}

func (n *fakeNotification) Send(ctx context.Context, _ bool, _ *time.Location, _ string) error {
	n.calls.Add(1)
	_, hasDeadline := ctx.Deadline()
	n.sawDeadline.Store(hasDeadline)
	return n.err
}

var _ notification.Notification = (*fakeNotification)(nil)

func TestNotificationGroupUsesTimeoutAndSendFailuresRemainNonfatal(t *testing.T) {
	first := &fakeNotification{err: errors.New("notification failed")}
	second := &fakeNotification{}
	group := &notificationGroup{notifications: []notification.Notification{first, second}, timeout: time.Second, location: time.UTC}
	group.Send(context.Background(), false, "PostgreSQL archive: dump failed")
	if first.calls.Load() != 1 || second.calls.Load() != 1 {
		t.Fatalf("notification calls = %d/%d", first.calls.Load(), second.calls.Load())
	}
	if !first.sawDeadline.Load() || !second.sawDeadline.Load() {
		t.Fatal("notification timeout was not applied")
	}
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}
func indexContaining(values []string, target string) int {
	for i, value := range values {
		if strings.Contains(value, target) {
			return i
		}
	}
	return -1
}
