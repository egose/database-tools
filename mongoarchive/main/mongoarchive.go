package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"github.com/go-co-op/gocron/v2"
	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/progress"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongodump"
)

var version string

const (
	progressBarLength   = 24
	progressBarWaitTime = time.Second * 3
	envPrefix           = "MONGOARCHIVE__"
	defaultWorkspaceDir = "mongoarchive"
)

type archiveDump interface {
	Init() error
	Dump() error
	HandleInterrupt()
}

type archivePipeline struct {
	createWorkspace func() (string, error)
	newFilename     func() (string, string)
	newDump         func([]string) (archiveDump, func(), error)
	getStorages     func(context.Context, *mongoarchive.Config) ([]storage.Storage, error)
	tar             func(string, string) error
	buildObjectName func(string, string) (string, error)
	upload          func(context.Context, []storage.Storage, string, string) error
	deleteDirectory func(string) error
	deleteFile      func(string) error
	handleInterrupt func(func()) chan struct{}
	notify          func(context.Context, *mongoarchive.Config, bool, string)
}

type cleanupEntry struct {
	path   string
	remove func(string) error
}

type cleanupStack struct {
	entries []cleanupEntry
}

type multiBackendArchiveError struct {
	message string
	cause   error
}

type cronOverlapPolicy string

const cronSkipOverlappingRuns cronOverlapPolicy = "skip"

type cronScheduler interface {
	Schedule(string, func(), cronOverlapPolicy) error
	Start()
	Shutdown() error
}

type cronRuntime struct {
	newScheduler    func(*time.Location) (cronScheduler, error)
	runTask         func(context.Context, *mongoarchive.Config) error
	notify          func(context.Context, *mongoarchive.Config, bool, string)
	waitForShutdown func()
	now             func() time.Time
}

type gocronScheduler struct {
	scheduler gocron.Scheduler
}

func main() {
	cfg, showVersion, err := mongoarchive.ParseFlags()
	if err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err)
		os.Exit(1)
	}
	if showVersion {
		fmt.Println("mongo-archive version:", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.HasCron() {
		err = runCronJob(ctx, cfg)
	} else {
		err = runTask(ctx, cfg)
		if err != nil {
			sendNotification(ctx, cfg, false, err.Error())
		}
	}

	if err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err.Error())
		os.Exit(1)
	}
}

// See https://github.com/go-co-op/gocron
func runCronJob(ctx context.Context, cfg *mongoarchive.Config) error {
	return newCronRuntime().run(ctx, cfg)
}

func runTask(ctx context.Context, cfg *mongoarchive.Config) error {
	return newArchivePipeline().run(ctx, cfg)
}

func newArchivePipeline() archivePipeline {
	return archivePipeline{
		createWorkspace: createArchiveWorkspace,
		newFilename:     utils.GetNewFilename,
		newDump:         newMongoDumpRunner,
		getStorages: func(ctx context.Context, cfg *mongoarchive.Config) ([]storage.Storage, error) {
			return cfg.GetStorages(ctx)
		},
		tar:             utils.Tar,
		buildObjectName: storage.BuildBackupObjectName,
		upload:          uploadBackupToStorages,
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: signals.HandleWithInterrupt,
		notify:          sendNotification,
	}
}

func newCronRuntime() cronRuntime {
	return cronRuntime{
		newScheduler: newGocronScheduler,
		runTask:      runTask,
		notify:       sendNotification,
		waitForShutdown: func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigChan)
			<-sigChan
		},
		now: time.Now,
	}
}

func (p archivePipeline) run(ctx context.Context, cfg *mongoarchive.Config) (retErr error) {
	filename := ""
	dumpDirName := ""
	defer func() {
		if retErr == nil && filename != "" {
			p.notify(ctx, cfg, true, filename)
		}
	}()

	storageBackends, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	if len(storageBackends) == 0 {
		return fmt.Errorf("no storage backends configured")
	}
	defer func() {
		if closeErr := closeStorages(storageBackends); closeErr != nil {
			retErr = joinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	workspace, err := p.createWorkspace()
	if err != nil {
		return err
	}

	cleanup := cleanupStack{}
	cleanup.addDirectory(workspace, p.deleteDirectory)
	defer func() {
		if cleanupErr := cleanup.run(cfg.HasKeep()); cleanupErr != nil {
			retErr = joinPrimaryAndCleanupErrors(retErr, cleanupErr)
		}
	}()

	filename, dumpDirName = p.newFilename()
	destPath := filepath.Join(workspace, dumpDirName)
	tarfilePath := filepath.Join(workspace, filename)

	options := cfg.GetMongodumpOptions()
	options = append(options, "--out="+destPath)

	dump, stopProgress, err := p.newDump(options)
	if err != nil {
		return err
	}
	defer stopProgress()

	finishedChan := p.handleInterrupt(dump.HandleInterrupt)
	defer func() {
		if finishedChan != nil {
			close(finishedChan)
		}
	}()

	if err := dump.Init(); err != nil {
		return err
	}

	if err := dump.Dump(); err != nil {
		return err
	}
	cleanup.addDirectory(destPath, p.deleteDirectory)

	if err := p.tar(destPath, tarfilePath); err != nil {
		return err
	}
	cleanup.addFile(tarfilePath, p.deleteFile)

	objectName, err := p.buildObjectName(cfg.BackupPrefix, filename)
	if err != nil {
		return err
	}

	if err := p.upload(ctx, storageBackends, objectName, tarfilePath); err != nil {
		return err
	}

	return nil
}

func (r cronRuntime) run(ctx context.Context, cfg *mongoarchive.Config) error {
	loc := cfg.GetLocation()
	if loc == nil {
		return fmt.Errorf("invalid timezone location")
	}

	exp := cfg.GetCronExpression()
	if exp == "" {
		return fmt.Errorf("empty cron expression")
	}

	mlog.Logvf(mlog.Always, "Using Cron Expression: %v", exp)

	s, err := r.newScheduler(loc)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}
	defer func() {
		if err := s.Shutdown(); err != nil {
			mlog.Logvf(mlog.Always, "Failed to shut down scheduler: %v", err)
		}
	}()

	err = s.Schedule(exp, func() {
		startTime := r.now()
		mlog.Logvf(mlog.Always, "Task started at: %v", startTime)

		if err := r.runTask(ctx, cfg); err != nil {
			mlog.Logvf(mlog.Always, "Task failed: %v", err)
			r.notify(ctx, cfg, false, err.Error())
		} else {
			mlog.Logvf(mlog.Always, "Task completed successfully at: %v (Duration: %v)", r.now(), r.now().Sub(startTime))
		}
	}, cronSkipOverlappingRuns)
	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	s.Start()
	mlog.Logvf(mlog.Always, "Scheduler started.")
	mlog.Logvf(mlog.Always, "Scheduler overlap policy: skip overlapping runs for the same job.")

	r.waitForShutdown()
	mlog.Logvf(mlog.Always, "Shutting down scheduler...")
	return nil
}

func createArchiveWorkspace() (string, error) {
	basePath := os.Getenv(envPrefix + "DUMP_PATH")
	if basePath == "" {
		basePath = filepath.Join(os.TempDir(), defaultWorkspaceDir)
	}

	if err := os.MkdirAll(basePath, 0o700); err != nil {
		return "", err
	}

	return os.MkdirTemp(basePath, "run-")
}

func newMongoDumpRunner(options []string) (archiveDump, func(), error) {
	opts, err := mongodump.ParseOptions(options, "", "")
	if err != nil {
		return nil, func() {}, err
	}

	mlog.SetVerbosity(opts.Verbosity)
	opts.URI.LogUnsupportedOptions()

	progressManager := progress.NewBarWriter(mlog.Writer(0), progressBarWaitTime, progressBarLength, false)
	progressManager.Start()

	dump := mongodump.MongoDump{
		ToolOptions:     opts.ToolOptions,
		InputOptions:    opts.InputOptions,
		OutputOptions:   opts.OutputOptions,
		ProgressManager: progressManager,
	}

	return &dump, progressManager.Stop, nil
}

func newGocronScheduler(loc *time.Location) (cronScheduler, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return nil, err
	}

	return &gocronScheduler{scheduler: scheduler}, nil
}

func (s *gocronScheduler) Schedule(expression string, task func(), overlap cronOverlapPolicy) error {
	jobOptions := []gocron.JobOption{}
	if overlap == cronSkipOverlappingRuns {
		jobOptions = append(jobOptions, gocron.WithSingletonMode(gocron.LimitModeReschedule))
	}

	_, err := s.scheduler.NewJob(gocron.CronJob(expression, false), gocron.NewTask(task), jobOptions...)
	return err
}

func (s *gocronScheduler) Start() {
	s.scheduler.Start()
}

func (s *gocronScheduler) Shutdown() error {
	return s.scheduler.Shutdown()
}

func (c *cleanupStack) addFile(path string, remove func(string) error) {
	c.entries = append(c.entries, cleanupEntry{path: path, remove: remove})
}

func (c *cleanupStack) addDirectory(path string, remove func(string) error) {
	c.entries = append(c.entries, cleanupEntry{path: path, remove: remove})
}

func (c *cleanupStack) run(keep bool) error {
	if keep {
		return nil
	}

	var cleanupErrors []error
	for i := len(c.entries) - 1; i >= 0; i-- {
		entry := c.entries[i]
		if err := entry.remove(entry.path); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cleanup %q: %w", entry.path, err))
		}
	}

	return errors.Join(cleanupErrors...)
}

func (e *multiBackendArchiveError) Error() string {
	return e.message
}

func (e *multiBackendArchiveError) Unwrap() error {
	return e.cause
}

func closeStorages(storages []storage.Storage) error {
	var closeErrors []error
	for _, storageBackend := range storages {
		if err := storageBackend.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close %T: %w", storageBackend, err))
		}
	}

	return errors.Join(closeErrors...)
}

func joinPrimaryAndCleanupErrors(primary error, cleanup error) error {
	if primary == nil {
		return fmt.Errorf("cleanup failed: %w", cleanup)
	}
	if cleanup == nil {
		return primary
	}
	return errors.Join(primary, fmt.Errorf("cleanup failed: %w", cleanup))
}

func uploadBackupToStorages(ctx context.Context, storages []storage.Storage, objectName string, tarfilePath string) error {
	if len(storages) <= 1 {
		return uploadBackupToSingleStorage(ctx, storages, objectName, tarfilePath)
	}

	uploadedBackends := make([]string, 0, len(storages))
	for i, s := range storages {
		backendName := describeStorageBackend(i, s)
		uploadCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
		if err != nil {
			return err
		}
		result, err := s.Upload(uploadCtx, objectName, tarfilePath)
		cancel()
		if err != nil {
			partialState := "before any backend upload completed"
			if len(uploadedBackends) > 0 {
				partialState = "after successful uploads to " + formatCompletedBackends(uploadedBackends, "none")
			}
			return &multiBackendArchiveError{
				message: fmt.Sprintf(
					"archive upload failed %s; retention was not run on any backend: failed to upload to %s: %v",
					partialState,
					backendName,
					err,
				),
				cause: err,
			}
		}
		uploadedBackends = append(uploadedBackends, backendName)
		mlog.Logvf(mlog.Always, "Successfully uploaded backup to %s: %v", backendName, result)
	}

	mlog.Logvf(mlog.Always, "Verified archive upload across %d storage backends; starting retention.", len(uploadedBackends))

	retainedBackends := make([]string, 0, len(storages))
	for i, s := range storages {
		backendName := describeStorageBackend(i, s)
		deleteCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
		if err != nil {
			return err
		}
		err = s.DeleteOldObjects(deleteCtx, objectName)
		cancel()
		if err != nil {
			return &multiBackendArchiveError{
				message: fmt.Sprintf(
					"archive retention failed after successful retention on %s; archive upload completed on all configured backends: failed to delete old objects in %s: %v",
					formatCompletedBackends(retainedBackends, "none"),
					backendName,
					err,
				),
				cause: err,
			}
		}
		retainedBackends = append(retainedBackends, backendName)
	}

	return nil
}

func uploadBackupToSingleStorage(ctx context.Context, storages []storage.Storage, objectName string, tarfilePath string) error {
	for _, s := range storages {
		uploadCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
		if err != nil {
			return err
		}
		result, err := s.Upload(uploadCtx, objectName, tarfilePath)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to upload to %T: %w", s, err)
		}
		mlog.Logvf(mlog.Always, "Successfully uploaded backup to %T: %v", s, result)

		deleteCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
		if err != nil {
			return err
		}
		err = s.DeleteOldObjects(deleteCtx, objectName)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to delete old objects in %T: %w", s, err)
		}
	}

	return nil
}

func describeStorageBackend(index int, storageBackend storage.Storage) string {
	return fmt.Sprintf("backend #%d (%T)", index+1, storageBackend)
}

func formatCompletedBackends(backends []string, empty string) string {
	if len(backends) == 0 {
		return empty
	}

	return strings.Join(backends, ", ")
}

func sendNotification(ctx context.Context, cfg *mongoarchive.Config, success bool, filenameOrError string) {
	notifications, err := cfg.GetNotifications()
	if err != nil {
		mlog.Logvf(mlog.Always, "Failed to initialize notifications: %v", err)
		return
	}
	for _, notification := range notifications {
		notificationCtx, cancel, err := operationContext(ctx, envPrefix+"NOTIFICATION_TIMEOUT")
		if err != nil {
			mlog.Logvf(mlog.Always, "Failed to prepare notification context for %T: %v", notification, err)
			continue
		}
		if err := notification.Send(notificationCtx, success, cfg.GetTZ(), filenameOrError); err != nil {
			mlog.Logvf(mlog.Always, "Failed to send notification via %T: %v", notification, err)
		}
		cancel()
	}
}

func operationContext(ctx context.Context, envKey string) (context.Context, context.CancelFunc, error) {
	ctx = storageContextOrBackground(ctx)
	raw := os.Getenv(envKey)
	if raw == "" {
		opCtx, cancel := context.WithCancel(ctx)
		return opCtx, cancel, nil
	}

	timeout, err := time.ParseDuration(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("%s must be a valid duration: %w", envKey, err)
	}
	if timeout <= 0 {
		return nil, nil, fmt.Errorf("%s must be greater than zero", envKey)
	}

	opCtx, cancel := context.WithTimeout(ctx, timeout)
	return opCtx, cancel, nil
}

func storageContextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}
