package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/egose/database-tools/internal/toolruntime"
	"github.com/egose/database-tools/mongounarchive"
	projectstorage "github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"

	mlog "github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/signals"
	"github.com/mongodb/mongo-tools/mongorestore"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

var version string

const (
	progressBarLength   = 24
	progressBarWaitTime = time.Second * 3
	envPrefix           = "MONGOUNARCHIVE__"
	defaultWorkspaceDir = "mongounarchive"
)

type restoreExecutionResult struct {
	Successes int64
	Failures  int64
	Err       error
}

type restorePartialResultError struct {
	Successes int64
	Failures  int64
}

func (e restorePartialResultError) Error() string {
	return fmt.Sprintf("restore completed with partial failures: %d document(s) restored successfully, %d document(s) failed to restore", e.Successes, e.Failures)
}

type restoreRunner interface {
	HandleInterrupt()
	Restore() restoreExecutionResult
	Close() error
	Acknowledged() bool
}

type restorePipeline struct {
	createWorkspace    func(string) (string, error)
	getStorages        func(context.Context, *mongounarchive.Config) ([]projectstorage.RestoreBackend, error)
	selectStorage      func([]projectstorage.RestoreBackend, string) (projectstorage.RestoreBackend, error)
	getExtractionLimit func(*mongounarchive.Config) (utils.ArchiveExtractionLimits, error)
	download           func(context.Context, projectstorage.RestoreStorage, string, string) error
	extract            func(string, string, utils.ArchiveExtractionLimits) error
	newRestore         func([]string) (restoreRunner, error)
	applyUpdates       func(context.Context, *mongounarchive.Config, []update) error
	deleteDirectory    func(string) error
	deleteFile         func(string) error
	handleInterrupt    func(func()) chan struct{}
	logAlways          func(string, ...any)
}

type mongoRestoreRunner struct {
	restore *mongorestore.MongoRestore
}

type update struct {
	Collection string         `json:"collection"`
	Filter     map[string]any `json:"filter"`
	Update     map[string]any `json:"update"`
}

type updateCollection interface {
	UpdateMany(context.Context, any, any, ...mongooptions.Lister[mongooptions.UpdateManyOptions]) (*mongo.UpdateResult, error)
}

type updateDatabase interface {
	Collection(string, ...mongooptions.Lister[mongooptions.CollectionOptions]) updateCollection
}

type mongoUpdateDatabase struct {
	database *mongo.Database
}

func (d mongoUpdateDatabase) Collection(name string, opts ...mongooptions.Lister[mongooptions.CollectionOptions]) updateCollection {
	return d.database.Collection(name, opts...)
}

func main() {
	cfg, showVersion, err := mongounarchive.ParseFlags()
	if err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err)
		os.Exit(1)
	}
	if showVersion {
		fmt.Println("mongo-unarchive version:", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runTask(ctx, cfg); err != nil {
		mlog.Logvf(mlog.Always, "Failed: %v", err)
		os.Exit(1)
	}
}

func runTask(ctx context.Context, cfg *mongounarchive.Config) error {
	return newRestorePipeline().run(ctx, cfg)
}

func newRestorePipeline() restorePipeline {
	return restorePipeline{
		createWorkspace: createRestoreWorkspace,
		getStorages: func(ctx context.Context, cfg *mongounarchive.Config) ([]projectstorage.RestoreBackend, error) {
			return cfg.GetStorages(ctx)
		},
		selectStorage: projectstorage.SelectRestoreStorage,
		getExtractionLimit: func(cfg *mongounarchive.Config) (utils.ArchiveExtractionLimits, error) {
			limits := cfg.GetArchiveExtractionLimits()
			if err := limits.Validate(); err != nil {
				return utils.ArchiveExtractionLimits{}, err
			}
			return limits, nil
		},
		download: func(ctx context.Context, storage projectstorage.RestoreStorage, objectName string, destination string) error {
			return storage.Download(ctx, objectName, destination)
		},
		extract:         utils.UnTar,
		newRestore:      newMongoRestoreRunner,
		applyUpdates:    applyUpdates,
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: signals.HandleWithInterrupt,
		logAlways: func(format string, args ...any) {
			mlog.Logvf(mlog.Always, format, args...)
		},
	}
}

func (p restorePipeline) logf(format string, args ...any) {
	if p.logAlways != nil {
		p.logAlways(format, args...)
		return
	}
	mlog.Logvf(mlog.Always, format, args...)
}

func (p restorePipeline) run(ctx context.Context, cfg *mongounarchive.Config) (retErr error) {
	if err := cfg.StorageOptions.Validate(); err != nil {
		return err
	}
	if err := cfg.RuntimeOptions.Validate(); err != nil {
		return err
	}

	validatedUpdates, err := parseUpdates(cfg)
	if err != nil {
		return err
	}

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

	storages, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := toolruntime.CloseAll(storages); closeErr != nil {
			retErr = toolruntime.JoinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	storage, err := p.selectStorage(storages, cfg.StorageBackend)
	if err != nil {
		return err
	}

	extractionLimits, err := p.getExtractionLimit(cfg)
	if err != nil {
		return err
	}

	lookupCtx, cancel := toolruntime.OperationContext(ctx, cfg.StorageOperationTimeout)
	objectName, err := storage.GetTargetObjectName(lookupCtx, cfg.GetObjectName())
	cancel()
	if err != nil {
		return err
	}

	tarfilePath, err := utils.ResolvePathWithinRoot(workspace, objectName)
	if err != nil {
		return err
	}

	destPath := filepath.Join(workspace, utils.GetFileNameWithoutExtension(objectName))

	p.logf("Downloading archive...")
	downloadCtx, cancel := toolruntime.OperationContext(ctx, cfg.StorageOperationTimeout)
	err = p.download(downloadCtx, storage, objectName, tarfilePath)
	cancel()
	if err != nil {
		return err
	}
	cleanup.AddFile(tarfilePath, p.deleteFile)

	p.logf("Extracting files...")
	err = p.extract(tarfilePath, destPath, extractionLimits)
	if err != nil {
		return err
	}
	cleanup.AddDirectory(destPath, p.deleteDirectory)

	options := cfg.GetMongounarchiveOptions(destPath)
	restore, err := p.newRestore(options)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := restore.Close(); closeErr != nil {
			retErr = toolruntime.JoinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	finishedChan := p.handleInterrupt(restore.HandleInterrupt)
	defer func() {
		if finishedChan != nil {
			close(finishedChan)
		}
	}()

	p.logf("Restoring database...")
	result := restore.Restore()
	if result.Err != nil {
		return result.Err
	}
	if result.Failures > 0 {
		return restorePartialResultError{Successes: result.Successes, Failures: result.Failures}
	}

	if restore.Acknowledged() {
		p.logf("%v document(s) restored successfully. %v document(s) failed to restore.", result.Successes, result.Failures)
	} else {
		p.logf("done")
	}

	if len(validatedUpdates) > 0 {
		p.logf("Applying updates...")
		if err := p.applyUpdates(ctx, cfg, validatedUpdates); err != nil {
			return err
		}
	}

	p.logf("Unarchive completed successfully")
	return nil
}

func createRestoreWorkspace(basePath string) (string, error) {
	if basePath == "" {
		basePath = filepath.Join(os.TempDir(), defaultWorkspaceDir)
	}

	if err := os.MkdirAll(basePath, 0o700); err != nil {
		return "", err
	}

	return os.MkdirTemp(basePath, "run-")
}

func newMongoRestoreRunner(options []string) (restoreRunner, error) {
	opts, err := mongorestore.ParseOptions(options, "", "")
	if err != nil {
		return nil, err
	}

	restore, err := mongorestore.New(opts)
	if err != nil {
		return nil, err
	}

	return &mongoRestoreRunner{restore: restore}, nil
}

func (r *mongoRestoreRunner) HandleInterrupt() {
	r.restore.HandleInterrupt()
}

func (r *mongoRestoreRunner) Restore() restoreExecutionResult {
	result := r.restore.Restore()
	return restoreExecutionResult{
		Successes: result.Successes,
		Failures:  result.Failures,
		Err:       result.Err,
	}
}

func (r *mongoRestoreRunner) Close() error {
	r.restore.Close()
	return nil
}

func (r *mongoRestoreRunner) Acknowledged() bool {
	return r.restore.ToolOptions.WriteConcern.Acknowledged()
}

func applyUpdates(ctx context.Context, cfg *mongounarchive.Config, updates []update) error {
	connectCtx, cancel := toolruntime.OperationContext(ctx, cfg.UpdateTimeout)
	client, dbClient, err := cfg.GetMongoClient(connectCtx)
	cancel()
	if err != nil {
		return err
	}

	defer func() {
		disconnectCtx, cancel := toolruntime.OperationContext(ctx, cfg.UpdateTimeout)
		_ = client.Disconnect(disconnectCtx)
		cancel()
	}()

	return applyValidatedUpdates(ctx, mongoUpdateDatabase{database: dbClient}, updates, cfg.UpdateTimeout)
}

func applyValidatedUpdates(ctx context.Context, db updateDatabase, updates []update, operationTimeout time.Duration) error {
	for i, u := range updates {
		coll := db.Collection(u.Collection)
		updateCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		result, err := coll.UpdateMany(updateCtx, u.Filter, u.Update)
		cancel()
		if err != nil {
			return err
		}

		mlog.Logvf(mlog.Always, "Update[%d]: matched count: %d", i, result.MatchedCount)
		mlog.Logvf(mlog.Always, "Update[%d]: modified count: %d", i, result.ModifiedCount)
	}

	return nil
}

func parseUpdates(cfg *mongounarchive.Config) ([]update, error) {
	if !cfg.HasUpdates() {
		return nil, nil
	}
	if cfg.DryRun {
		return nil, errors.New("--dry-run cannot be combined with --updates or --updates-file")
	}

	input, err := cfg.GetUpdates()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()

	updates := []update{}
	if err := decoder.Decode(&updates); err != nil {
		return nil, fmt.Errorf("decode updates: %w", err)
	}
	if err := ensureOnlyJSONValue(decoder); err != nil {
		return nil, err
	}

	for i := range updates {
		updates[i].Collection = strings.TrimSpace(updates[i].Collection)
		if updates[i].Collection == "" {
			return nil, fmt.Errorf("updates[%d].collection must be non-empty", i)
		}
		if len(updates[i].Filter) == 0 {
			return nil, fmt.Errorf("updates[%d].filter must be a non-empty document", i)
		}
		if len(updates[i].Update) == 0 {
			return nil, fmt.Errorf("updates[%d].update must be a non-empty document", i)
		}
	}

	return updates, nil
}

func ensureOnlyJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("decode updates: unexpected trailing data")
		}
		return fmt.Errorf("decode updates: %w", err)
	}
	return nil
}
