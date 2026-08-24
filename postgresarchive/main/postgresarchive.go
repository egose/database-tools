package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/egose/database-tools/internal/archivedelivery"
	"github.com/egose/database-tools/internal/postgresarchiveformat"
	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolruntime"
	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/postgresarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"github.com/go-co-op/gocron/v2"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

var version string

type dumpRunner interface {
	Dump(context.Context, postgresclient.ConnectionOptions, string) error
	Version(context.Context, postgresclient.Client) (string, error)
}

type archivePipeline struct {
	createWorkspace func(string) (string, error)
	newFilename     func() (string, string)
	newRunner       func(string) dumpRunner
	getStorages     func(context.Context, *postgresarchive.Config) ([]storage.ArchiveBackend, error)
	newManifest     func(time.Time, string, string) (postgresarchiveformat.Manifest, error)
	writeManifest   func(string, postgresarchiveformat.Manifest) error
	tar             func(string, string) error
	buildObjectName func(string, string, string) (string, error)
	deliver         func(context.Context, []storage.ArchiveBackend, string, string, time.Duration) error
	deleteDirectory func(string) error
	now             func() time.Time
	notify          notificationSender
}

type notificationSender interface {
	Send(context.Context, bool, string)
}
type notificationSenderFunc func(context.Context, bool, string)

func (f notificationSenderFunc) Send(ctx context.Context, success bool, value string) {
	f(ctx, success, value)
}

type notificationGroup struct {
	notifications []notification.Notification
	timeout       time.Duration
	location      *time.Location
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
	runTask      func(context.Context, *postgresarchive.Config, notificationSender) error
	notify       notificationSender
	now          func() time.Time
}

type archiveRuntime struct {
	newNotifications func(*postgresarchive.Config) (notificationSender, error)
	runTask          func(context.Context, *postgresarchive.Config, notificationSender) error
	runCronJob       func(context.Context, *postgresarchive.Config, notificationSender) error
}

type gocronScheduler struct{ scheduler gocron.Scheduler }

func main() {
	cfg, showVersion, err := postgresarchive.ParseFlags()
	if err != nil {
		mlog.Logvf(mlog.Always, "PostgreSQL archive failed: %v", err)
		os.Exit(1)
	}
	if showVersion {
		fmt.Println("postgres-archive version:", version)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newArchiveRuntime().run(ctx, cfg); err != nil {
		mlog.Logvf(mlog.Always, "PostgreSQL archive failed: %v", err)
		os.Exit(1)
	}
}

func newArchiveRuntime() archiveRuntime {
	return archiveRuntime{
		newNotifications: func(cfg *postgresarchive.Config) (notificationSender, error) {
			return newNotificationGroup(cfg)
		},
		runTask: func(ctx context.Context, cfg *postgresarchive.Config, notify notificationSender) error {
			return newArchivePipeline(notify).run(ctx, cfg)
		},
		runCronJob: func(ctx context.Context, cfg *postgresarchive.Config, notify notificationSender) error {
			return newCronRuntime(notify).run(ctx, cfg)
		},
	}
}

func (r archiveRuntime) run(ctx context.Context, cfg *postgresarchive.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	notify, err := r.newNotifications(cfg)
	if err != nil {
		return fmt.Errorf("initialize notifications: %w", err)
	}
	if notify == nil {
		notify = notificationSenderFunc(func(context.Context, bool, string) {})
	}
	defer closeNotificationSender(notify)
	if cfg.HasCron() {
		return r.runCronJob(ctx, cfg, notify)
	}
	err = r.runTask(ctx, cfg, notify)
	if err != nil {
		notify.Send(ctx, false, "PostgreSQL archive: "+err.Error())
	}
	return err
}

func newArchivePipeline(notify notificationSender) archivePipeline {
	return archivePipeline{
		createWorkspace: createArchiveWorkspace,
		newFilename:     utils.GetNewFilename,
		newRunner: func(workspace string) dumpRunner {
			return postgresclient.NewRunner(postgresclient.WithTemporaryDirectory(workspace))
		},
		getStorages: func(ctx context.Context, cfg *postgresarchive.Config) ([]storage.ArchiveBackend, error) {
			return cfg.GetStorages(ctx)
		},
		newManifest:     postgresarchiveformat.NewManifest,
		writeManifest:   writeManifest,
		tar:             utils.Tar,
		buildObjectName: storage.BuildBackupObjectNameWithDefault,
		deliver: func(ctx context.Context, backends []storage.ArchiveBackend, objectName, path string, timeout time.Duration) error {
			return archivedelivery.Deliver(ctx, backends, objectName, path, timeout, func(format string, args ...any) { mlog.Logvf(mlog.Always, format, args...) })
		},
		deleteDirectory: utils.DeleteDirectory,
		now:             time.Now,
		notify:          notify,
	}
}

func (p archivePipeline) run(ctx context.Context, cfg *postgresarchive.Config) (retErr error) {
	filename := ""
	defer func() {
		if retErr == nil && filename != "" && p.notify != nil {
			p.notify.Send(ctx, true, filename)
		}
	}()
	if err := cfg.Validate(); err != nil {
		return err
	}

	backends, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	if len(backends) == 0 {
		return fmt.Errorf("no storage backends configured")
	}
	defer func() {
		if closeErr := toolruntime.CloseAll(backends); closeErr != nil {
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

	filename, _ = p.newFilename()
	payloadPath := filepath.Join(workspace, "payload")
	if err := os.Mkdir(payloadPath, 0o700); err != nil {
		return fmt.Errorf("create PostgreSQL archive payload: %w", err)
	}
	dumpPath := filepath.Join(payloadPath, postgresarchiveformat.DumpPath)
	manifestPath := filepath.Join(payloadPath, postgresarchiveformat.ManifestPath)
	archivePath := filepath.Join(workspace, filename)

	runner := p.newRunner(workspace)
	clientVersion, err := runner.Version(ctx, postgresclient.PGDump)
	if err != nil {
		return err
	}
	if err := runner.Dump(ctx, cfg.Connection, dumpPath); err != nil {
		return err
	}
	database, err := cfg.Connection.DatabaseName()
	if err != nil {
		return err
	}
	manifest, err := p.newManifest(p.now(), database, clientVersion)
	if err != nil {
		return fmt.Errorf("create PostgreSQL archive manifest: %w", err)
	}
	if err := p.writeManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("write PostgreSQL archive manifest: %w", err)
	}
	if err := p.tar(payloadPath, archivePath); err != nil {
		return fmt.Errorf("create PostgreSQL archive transport: %w", err)
	}
	objectName, err := p.buildObjectName(cfg.BackupPrefix, filename, storage.PostgreSQLDefaultBackupPrefix)
	if err != nil {
		return err
	}
	if err := p.deliver(ctx, backends, objectName, archivePath, cfg.StorageOperationTimeout); err != nil {
		return err
	}
	return nil
}

func writeManifest(path string, manifest postgresarchiveformat.Manifest) error {
	return utils.WriteFileAtomically(path, func(file *os.File) error { return postgresarchiveformat.WriteManifest(file, manifest) })
}

func createArchiveWorkspace(base string) (string, error) {
	if base == "" {
		base = filepath.Join(os.TempDir(), "postgresarchive")
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "run-")
}

func newCronRuntime(notify notificationSender) cronRuntime {
	return cronRuntime{
		newScheduler: newGocronScheduler,
		runTask: func(ctx context.Context, cfg *postgresarchive.Config, notify notificationSender) error {
			return newArchivePipeline(notify).run(ctx, cfg)
		},
		notify: notify,
		now:    time.Now,
	}
}

func (r cronRuntime) run(ctx context.Context, cfg *postgresarchive.Config) error {
	if cfg.GetLocation() == nil {
		return fmt.Errorf("invalid timezone location")
	}
	if cfg.GetCronExpression() == "" {
		return fmt.Errorf("empty cron expression")
	}
	scheduler, err := r.newScheduler(cfg.GetLocation())
	if err != nil {
		return fmt.Errorf("create scheduler: %w", err)
	}
	defer func() {
		if err := scheduler.Shutdown(); err != nil {
			mlog.Logvf(mlog.Always, "Failed to shut down PostgreSQL archive scheduler: %v", err)
		}
	}()
	err = scheduler.Schedule(cfg.GetCronExpression(), func() {
		started := r.now()
		if err := r.runTask(ctx, cfg, r.notify); err != nil {
			mlog.Logvf(mlog.Always, "PostgreSQL archive task failed: %v", err)
			r.notify.Send(ctx, false, "PostgreSQL archive: "+err.Error())
			return
		}
		mlog.Logvf(mlog.Always, "PostgreSQL archive task completed successfully at %v (duration %v)", r.now(), r.now().Sub(started))
	}, cronSkipOverlappingRuns)
	if err != nil {
		return fmt.Errorf("schedule PostgreSQL archive job: %w", err)
	}
	scheduler.Start()
	<-ctx.Done()
	return nil
}

func newGocronScheduler(location *time.Location) (cronScheduler, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(location))
	if err != nil {
		return nil, err
	}
	return &gocronScheduler{scheduler: scheduler}, nil
}

func (s *gocronScheduler) Schedule(expression string, task func(), overlap cronOverlapPolicy) error {
	options := []gocron.JobOption{}
	if overlap == cronSkipOverlappingRuns {
		options = append(options, gocron.WithSingletonMode(gocron.LimitModeReschedule))
	}
	_, err := s.scheduler.NewJob(gocron.CronJob(expression, false), gocron.NewTask(task), options...)
	return err
}
func (s *gocronScheduler) Start()          { s.scheduler.Start() }
func (s *gocronScheduler) Shutdown() error { return s.scheduler.Shutdown() }

func newNotificationGroup(cfg *postgresarchive.Config) (*notificationGroup, error) {
	notifications, err := cfg.NotificationOptions.GetNotifications()
	if err != nil {
		return nil, err
	}
	return &notificationGroup{notifications: notifications, timeout: cfg.NotificationTimeout, location: cfg.GetLocation()}, nil
}

func (g *notificationGroup) Send(ctx context.Context, success bool, value string) {
	if g == nil {
		return
	}
	for _, item := range g.notifications {
		notifyCtx, cancel := toolruntime.OperationContext(ctx, g.timeout)
		if err := item.Send(notifyCtx, success, g.location, value); err != nil {
			mlog.Logvf(mlog.Always, "Failed to send PostgreSQL archive notification via %T: %v", item, err)
		}
		cancel()
	}
}

func (g *notificationGroup) Close() {
	if g == nil {
		return
	}
	for _, item := range g.notifications {
		if closer, ok := item.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				mlog.Logvf(mlog.Always, "Failed to close PostgreSQL archive notification %T: %v", item, err)
			}
		}
	}
}

func closeNotificationSender(sender notificationSender) {
	if closer, ok := sender.(interface{ Close() }); ok {
		closer.Close()
	}
}
