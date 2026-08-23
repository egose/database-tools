package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/egose/database-tools/internal/toolruntime"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/notification"
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
	createWorkspace func(string) (string, error)
	newFilename     func() (string, string)
	newDump         func([]string) (archiveDump, func(), error)
	getStorages     func(context.Context, *mongoarchive.Config) ([]storage.ArchiveBackend, error)
	tar             func(string, string) error
	buildObjectName func(string, string) (string, error)
	upload          func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error
	deleteDirectory func(string) error
	deleteFile      func(string) error
	handleInterrupt func(func()) chan struct{}
	notify          notificationSender
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
	newScheduler func(*time.Location) (cronScheduler, error)
	runTask      func(context.Context, *mongoarchive.Config, notificationSender) error
	notify       notificationSender
	now          func() time.Time
}

type archiveRuntime struct {
	newNotifications func(*mongoarchive.Config) (notificationSender, error)
	runTask          func(context.Context, *mongoarchive.Config, notificationSender) error
	runCronJob       func(context.Context, *mongoarchive.Config, notificationSender) error
}

type notificationSender interface {
	Send(context.Context, bool, string)
}

type notificationSenderFunc func(context.Context, bool, string)

func (f notificationSenderFunc) Send(ctx context.Context, success bool, filenameOrError string) {
	f(ctx, success, filenameOrError)
}

type notificationGroup struct {
	notifications []notification.Notification
	timeout       time.Duration
	location      *time.Location
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

	err = newArchiveRuntime().run(ctx, cfg)

	if err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err.Error())
		os.Exit(1)
	}
}

// See https://github.com/go-co-op/gocron
func runCronJob(ctx context.Context, cfg *mongoarchive.Config) error {
	notifications, err := newNotificationGroup(cfg)
	if err != nil {
		return err
	}
	defer notifications.Close()
	return runCronJobWithNotifications(ctx, cfg, notifications)
}

func runTask(ctx context.Context, cfg *mongoarchive.Config) error {
	notifications, err := newNotificationGroup(cfg)
	if err != nil {
		return err
	}
	defer notifications.Close()
	return runTaskWithNotifications(ctx, cfg, notifications)
}

func newArchiveRuntime() archiveRuntime {
	return archiveRuntime{
		newNotifications: func(cfg *mongoarchive.Config) (notificationSender, error) {
			return newNotificationGroup(cfg)
		},
		runTask:    runTaskWithNotifications,
		runCronJob: runCronJobWithNotifications,
	}
}

func (r archiveRuntime) run(ctx context.Context, cfg *mongoarchive.Config) error {
	// Notification clients are process-lifecycle resources: one-shot runs use one
	// group, and cron mode reuses the same group for every scheduled execution.
	notifications, err := r.newNotifications(cfg)
	if err != nil {
		return err
	}
	if notifications == nil {
		notifications = notificationSenderFunc(func(context.Context, bool, string) {})
	}
	defer closeNotificationSender(notifications)

	if cfg.HasCron() {
		return r.runCronJob(ctx, cfg, notifications)
	}

	err = r.runTask(ctx, cfg, notifications)
	if err != nil {
		notifications.Send(ctx, false, err.Error())
	}
	return err
}

func runCronJobWithNotifications(ctx context.Context, cfg *mongoarchive.Config, notifications notificationSender) error {
	return newCronRuntime(notifications).run(ctx, cfg)
}

func runTaskWithNotifications(ctx context.Context, cfg *mongoarchive.Config, notifications notificationSender) error {
	return newArchivePipeline(notifications).run(ctx, cfg)
}

func newArchivePipeline(notifications notificationSender) archivePipeline {
	return archivePipeline{
		createWorkspace: createArchiveWorkspace,
		newFilename:     utils.GetNewFilename,
		newDump:         newMongoDumpRunner,
		getStorages: func(ctx context.Context, cfg *mongoarchive.Config) ([]storage.ArchiveBackend, error) {
			return cfg.GetStorages(ctx)
		},
		tar:             utils.Tar,
		buildObjectName: storage.BuildBackupObjectName,
		upload:          uploadBackupToStorages,
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: signals.HandleWithInterrupt,
		notify:          notifications,
	}
}

func newCronRuntime(notifications notificationSender) cronRuntime {
	return cronRuntime{
		newScheduler: newGocronScheduler,
		runTask:      runTaskWithNotifications,
		notify:       notifications,
		now:          time.Now,
	}
}

func (p archivePipeline) run(ctx context.Context, cfg *mongoarchive.Config) (retErr error) {
	filename := ""
	dumpDirName := ""
	defer func() {
		if retErr == nil && filename != "" {
			sendWithNotificationSender(p.notify, ctx, true, filename)
		}
	}()

	if err := cfg.StorageOptions.Validate(); err != nil {
		return err
	}
	if err := cfg.RuntimeOptions.Validate(); err != nil {
		return err
	}

	storageBackends, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	if len(storageBackends) == 0 {
		return fmt.Errorf("no storage backends configured")
	}
	defer func() {
		if closeErr := toolruntime.CloseAll(storageBackends); closeErr != nil {
			retErr = toolruntime.JoinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	workspace, err := p.createWorkspace(cfg.WorkspaceBasePath)
	if err != nil {
		return err
	}

	cleanup := toolruntime.CleanupStack{}
	cleanup.AddDirectory(workspace, p.deleteDirectory)
	defer func() {
		if cleanupErr := cleanup.Run(cfg.HasKeep()); cleanupErr != nil {
			retErr = toolruntime.JoinPrimaryAndCleanupErrors(retErr, cleanupErr)
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
	cleanup.AddDirectory(destPath, p.deleteDirectory)

	if err := p.tar(destPath, tarfilePath); err != nil {
		return err
	}
	cleanup.AddFile(tarfilePath, p.deleteFile)

	objectName, err := p.buildObjectName(cfg.BackupPrefix, filename)
	if err != nil {
		return err
	}

	if err := p.upload(ctx, storageBackends, objectName, tarfilePath, cfg.StorageOperationTimeout); err != nil {
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

		if err := r.runTask(ctx, cfg, r.notify); err != nil {
			mlog.Logvf(mlog.Always, "Task failed: %v", err)
			sendWithNotificationSender(r.notify, ctx, false, err.Error())
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

	<-ctx.Done()
	mlog.Logvf(mlog.Always, "Shutting down scheduler...")
	return nil
}

func createArchiveWorkspace(basePath string) (string, error) {
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

func (e *multiBackendArchiveError) Error() string {
	return e.message
}

func (e *multiBackendArchiveError) Unwrap() error {
	return e.cause
}

func uploadBackupToStorages(ctx context.Context, storages []storage.ArchiveBackend, objectName string, tarfilePath string, operationTimeout time.Duration) error {
	if len(storages) <= 1 {
		return uploadBackupToSingleStorage(ctx, storages, objectName, tarfilePath, operationTimeout)
	}

	uploadedBackends := make([]string, 0, len(storages))
	for i, s := range storages {
		backendName := describeStorageBackend(i, s)
		uploadCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
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
		deleteCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		err := s.DeleteOldObjects(deleteCtx, objectName)
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

func uploadBackupToSingleStorage(ctx context.Context, storages []storage.ArchiveBackend, objectName string, tarfilePath string, operationTimeout time.Duration) error {
	for _, s := range storages {
		uploadCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		result, err := s.Upload(uploadCtx, objectName, tarfilePath)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to upload to %T: %w", s, err)
		}
		mlog.Logvf(mlog.Always, "Successfully uploaded backup to %T: %v", s, result)

		deleteCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		err = s.DeleteOldObjects(deleteCtx, objectName)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to delete old objects in %T: %w", s, err)
		}
	}

	return nil
}

func describeStorageBackend(index int, storageBackend storage.ArchiveBackend) string {
	if named, ok := storageBackend.(storage.BackendIdentifier); ok {
		if name, err := storage.BackendName(named); err == nil {
			return fmt.Sprintf("backend #%d (%s)", index+1, name)
		}
	}
	return fmt.Sprintf("backend #%d (%T)", index+1, storageBackend)
}

func formatCompletedBackends(backends []string, empty string) string {
	if len(backends) == 0 {
		return empty
	}

	return strings.Join(backends, ", ")
}

func newNotificationGroup(cfg *mongoarchive.Config) (*notificationGroup, error) {
	notifications, err := cfg.GetNotifications()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notifications: %w", err)
	}
	return &notificationGroup{
		notifications: notifications,
		timeout:       cfg.NotificationTimeout,
		location:      cfg.GetTZ(),
	}, nil
}

func (g *notificationGroup) Send(ctx context.Context, success bool, filenameOrError string) {
	if g == nil {
		return
	}
	for _, notification := range g.notifications {
		notificationCtx, cancel := toolruntime.OperationContext(ctx, g.timeout)
		if err := notification.Send(notificationCtx, success, g.location, filenameOrError); err != nil {
			mlog.Logvf(mlog.Always, "Failed to send notification via %T: %v", notification, err)
		}
		cancel()
	}
}

func (g *notificationGroup) Close() {
	if g == nil {
		return
	}
	for _, notification := range g.notifications {
		closer, ok := notification.(interface{ Close() error })
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil {
			mlog.Logvf(mlog.Always, "Failed to close notification %T: %v", notification, err)
		}
	}
}

func closeNotificationSender(sender notificationSender) {
	if closer, ok := sender.(interface{ Close() }); ok {
		closer.Close()
	}
}

func sendWithNotificationSender(sender notificationSender, ctx context.Context, success bool, filenameOrError string) {
	if sender == nil {
		return
	}
	sender.Send(ctx, success, filenameOrError)
}
