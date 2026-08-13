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
	"strconv"
	"strings"
	"syscall"
	"time"

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

type restoreRunner interface {
	HandleInterrupt()
	Restore() restoreExecutionResult
	Close() error
	Acknowledged() bool
}

type restorePipeline struct {
	createWorkspace    func() (string, error)
	getStorages        func(context.Context, *mongounarchive.Config) ([]projectstorage.Storage, error)
	selectStorage      func([]projectstorage.Storage, string) (projectstorage.Storage, error)
	getExtractionLimit func() (utils.ArchiveExtractionLimits, error)
	download           func(context.Context, projectstorage.Storage, string, string) error
	extract            func(string, string, utils.ArchiveExtractionLimits) error
	newRestore         func([]string) (restoreRunner, error)
	applyUpdates       func(context.Context, *mongounarchive.Config, []update) error
	deleteDirectory    func(string) error
	deleteFile         func(string) error
	handleInterrupt    func(func()) chan struct{}
}

type cleanupEntry struct {
	path   string
	remove func(string) error
}

type cleanupStack struct {
	entries []cleanupEntry
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
		getStorages: func(ctx context.Context, cfg *mongounarchive.Config) ([]projectstorage.Storage, error) {
			return cfg.GetStorages(ctx)
		},
		selectStorage:      projectstorage.SelectRestoreStorage,
		getExtractionLimit: getArchiveExtractionLimits,
		download: func(ctx context.Context, storage projectstorage.Storage, objectName string, destination string) error {
			return storage.Download(ctx, objectName, destination)
		},
		extract:         utils.UnTar,
		newRestore:      newMongoRestoreRunner,
		applyUpdates:    applyUpdates,
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		handleInterrupt: signals.HandleWithInterrupt,
	}
}

func (p restorePipeline) run(ctx context.Context, cfg *mongounarchive.Config) (retErr error) {
	validatedUpdates, err := parseUpdates(cfg)
	if err != nil {
		return err
	}

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

	storages, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeStorages(storages); closeErr != nil {
			retErr = joinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	storage, err := p.selectStorage(storages, cfg.StorageBackend)
	if err != nil {
		return err
	}

	extractionLimits, err := p.getExtractionLimit()
	if err != nil {
		return err
	}

	lookupCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
	if err != nil {
		return err
	}
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

	mlog.Logvf(mlog.Always, "Downloading archive...")
	downloadCtx, cancel, err := operationContext(ctx, envPrefix+"STORAGE_OPERATION_TIMEOUT")
	if err != nil {
		return err
	}
	err = p.download(downloadCtx, storage, objectName, tarfilePath)
	cancel()
	if err != nil {
		return err
	}
	cleanup.addFile(tarfilePath, p.deleteFile)

	mlog.Logvf(mlog.Always, "Extracting files...")
	err = p.extract(tarfilePath, destPath, extractionLimits)
	if err != nil {
		return err
	}
	cleanup.addDirectory(destPath, p.deleteDirectory)

	options := cfg.GetMongounarchiveOptions(destPath)
	restore, err := p.newRestore(options)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := restore.Close(); closeErr != nil {
			retErr = joinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()

	finishedChan := p.handleInterrupt(restore.HandleInterrupt)
	defer func() {
		if finishedChan != nil {
			close(finishedChan)
		}
	}()

	mlog.Logvf(mlog.Always, "Restoring database...")
	result := restore.Restore()
	if result.Err != nil {
		return result.Err
	}

	if restore.Acknowledged() {
		mlog.Logvf(mlog.Always, "%v document(s) restored successfully. %v document(s) failed to restore.", result.Successes, result.Failures)
	} else {
		mlog.Logvf(mlog.Always, "done")
	}

	if len(validatedUpdates) > 0 {
		mlog.Logvf(mlog.Always, "Applying updates...")
		if err := p.applyUpdates(ctx, cfg, validatedUpdates); err != nil {
			return err
		}
	}

	mlog.Logvf(mlog.Always, "Unarchive completed successfully")
	return nil
}

func createRestoreWorkspace() (string, error) {
	basePath := os.Getenv(envPrefix + "RESTORE_PATH")
	if basePath == "" {
		basePath = filepath.Join(os.TempDir(), defaultWorkspaceDir)
	}

	if err := os.MkdirAll(basePath, 0o700); err != nil {
		return "", err
	}

	return os.MkdirTemp(basePath, "run-")
}

func getArchiveExtractionLimits() (utils.ArchiveExtractionLimits, error) {
	limits := utils.DefaultArchiveExtractionLimits()

	maxEntries, err := readPositiveIntEnv(envPrefix+"ARCHIVE_MAX_ENTRIES", limits.MaxEntries)
	if err != nil {
		return utils.ArchiveExtractionLimits{}, err
	}
	maxEntryBytes, err := readPositiveInt64Env(envPrefix+"ARCHIVE_MAX_ENTRY_BYTES", limits.MaxEntryBytes)
	if err != nil {
		return utils.ArchiveExtractionLimits{}, err
	}
	maxTotalBytes, err := readPositiveInt64Env(envPrefix+"ARCHIVE_MAX_TOTAL_BYTES", limits.MaxTotalBytes)
	if err != nil {
		return utils.ArchiveExtractionLimits{}, err
	}

	limits.MaxEntries = maxEntries
	limits.MaxEntryBytes = maxEntryBytes
	limits.MaxTotalBytes = maxTotalBytes

	if err := limits.Validate(); err != nil {
		return utils.ArchiveExtractionLimits{}, err
	}

	return limits, nil
}

func readPositiveIntEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}

func readPositiveInt64Env(key string, fallback int64) (int64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
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

func closeStorages(storages []projectstorage.Storage) error {
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

func applyUpdates(ctx context.Context, cfg *mongounarchive.Config, updates []update) error {
	connectCtx, cancel, err := operationContext(ctx, envPrefix+"UPDATE_TIMEOUT")
	if err != nil {
		return err
	}
	client, dbClient, err := cfg.GetMongoClient(connectCtx)
	cancel()
	if err != nil {
		return err
	}

	defer func() {
		disconnectCtx, cancel, err := operationContext(ctx, envPrefix+"UPDATE_TIMEOUT")
		if err == nil {
			_ = client.Disconnect(disconnectCtx)
			cancel()
		}
	}()

	return applyValidatedUpdates(ctx, mongoUpdateDatabase{database: dbClient}, updates)
}

func applyValidatedUpdates(ctx context.Context, db updateDatabase, updates []update) error {
	for i, u := range updates {
		coll := db.Collection(u.Collection)
		updateCtx, cancel, err := operationContext(ctx, envPrefix+"UPDATE_TIMEOUT")
		if err != nil {
			return err
		}
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

func operationContext(ctx context.Context, envKey string) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
