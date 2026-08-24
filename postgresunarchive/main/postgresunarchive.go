package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/egose/database-tools/internal/postgresarchiveformat"
	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolruntime"
	"github.com/egose/database-tools/postgresunarchive"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

var version string

type restoreRunner interface {
	Restore(context.Context, postgresclient.ConnectionOptions, string) error
}

type restorePipeline struct {
	createWorkspace func(string) (string, error)
	getStorages     func(context.Context, *postgresunarchive.Config) ([]storage.RestoreBackend, error)
	selectStorage   func([]storage.RestoreBackend, string) (storage.RestoreBackend, error)
	download        func(context.Context, storage.RestoreStorage, string, string) error
	validateArchive func(string) error
	extract         func(string, string, utils.ArchiveExtractionLimits) error
	validateDump    func(string) error
	newRunner       func(string) restoreRunner
	deleteDirectory func(string) error
	deleteFile      func(string) error
	logAlways       func(string, ...any)
}

func main() {
	cfg, showVersion, err := postgresunarchive.ParseFlags()
	if err != nil {
		mlog.Logvf(mlog.Always, "PostgreSQL restore failed: %v", err)
		os.Exit(1)
	}
	if showVersion {
		fmt.Println("postgres-unarchive version:", version)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runTask(ctx, cfg); err != nil {
		mlog.Logvf(mlog.Always, "PostgreSQL restore failed: %v", err)
		os.Exit(1)
	}
}

func runTask(ctx context.Context, cfg *postgresunarchive.Config) error {
	return newRestorePipeline().run(ctx, cfg)
}

func newRestorePipeline() restorePipeline {
	return restorePipeline{
		createWorkspace: createRestoreWorkspace,
		getStorages: func(ctx context.Context, cfg *postgresunarchive.Config) ([]storage.RestoreBackend, error) {
			return cfg.GetStorages(ctx)
		},
		selectStorage: storage.SelectRestoreStorage,
		download: func(ctx context.Context, backend storage.RestoreStorage, objectName, destination string) error {
			return backend.Download(ctx, objectName, destination)
		},
		validateArchive: validateArchiveFile,
		extract:         utils.UnTar,
		validateDump:    validateExtractedDump,
		newRunner: func(workspace string) restoreRunner {
			return postgresclient.NewRunner(postgresclient.WithTemporaryDirectory(workspace))
		},
		deleteDirectory: utils.DeleteDirectory,
		deleteFile:      utils.DeleteFile,
		logAlways: func(format string, args ...any) {
			mlog.Logvf(mlog.Always, format, args...)
		},
	}
}

func (p restorePipeline) run(ctx context.Context, cfg *postgresunarchive.Config) (retErr error) {
	if err := cfg.Validate(); err != nil {
		return err
	}
	limits := cfg.GetArchiveExtractionLimits()
	if err := limits.Validate(); err != nil {
		return err
	}

	backends, err := p.getStorages(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := toolruntime.CloseAll(backends); closeErr != nil {
			retErr = toolruntime.JoinPrimaryAndCleanupErrors(retErr, closeErr)
		}
	}()
	backend, err := p.selectStorage(backends, cfg.StorageBackend)
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

	lookupCtx, cancel := toolruntime.OperationContext(ctx, cfg.StorageOperationTimeout)
	objectName, err := backend.GetTargetObjectName(lookupCtx, cfg.ObjectName)
	cancel()
	if err != nil {
		return err
	}
	archivePath := filepath.Join(workspace, "archive.tar.gz")
	p.logf("Downloading PostgreSQL archive %q...", objectName)
	downloadCtx, cancel := toolruntime.OperationContext(ctx, cfg.StorageOperationTimeout)
	err = p.download(downloadCtx, backend, objectName, archivePath)
	cancel()
	if err != nil {
		return err
	}
	cleanup.AddFile(archivePath, p.deleteFile)

	payloadPath := filepath.Join(workspace, "payload")
	p.logf("Extracting PostgreSQL archive...")
	if err := p.extract(archivePath, payloadPath, limits); err != nil {
		return fmt.Errorf("extract PostgreSQL archive: %w", err)
	}
	cleanup.AddDirectory(payloadPath, p.deleteDirectory)
	if err := p.validateArchive(archivePath); err != nil {
		return fmt.Errorf("validate PostgreSQL archive: %w", err)
	}
	dumpPath := filepath.Join(payloadPath, postgresarchiveformat.DumpPath)
	if err := p.validateDump(dumpPath); err != nil {
		return fmt.Errorf("validate extracted PostgreSQL dump: %w", err)
	}

	p.logf("Restoring existing PostgreSQL database...")
	runner := p.newRunner(workspace)
	if err := runner.Restore(ctx, cfg.Connection, dumpPath); err != nil {
		var partial interface{ PartialChangesPossible() bool }
		if errors.As(err, &partial) && partial.PartialChangesPossible() {
			return fmt.Errorf("PostgreSQL restore failed; partial database changes may exist: %w", err)
		}
		return fmt.Errorf("PostgreSQL restore did not complete: %w", err)
	}
	p.logf("PostgreSQL restore completed successfully")
	return nil
}

func (p restorePipeline) logf(format string, args ...any) {
	if p.logAlways != nil {
		p.logAlways(format, args...)
	}
}

func createRestoreWorkspace(base string) (string, error) {
	if base == "" {
		base = filepath.Join(os.TempDir(), "postgresunarchive")
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "run-")
}

func validateArchiveFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = postgresarchiveformat.ValidateArchive(file)
	return err
}

func validateExtractedDump(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("database.dump must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	magic := make([]byte, len("PGDMP"))
	if _, err := io.ReadFull(file, magic); err != nil || string(magic) != "PGDMP" {
		return errors.New("database.dump is not PostgreSQL custom format")
	}
	return nil
}
