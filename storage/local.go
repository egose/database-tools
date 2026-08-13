package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

type LocalStorage struct {
	LocalPath    string
	ExpiryDays   int
	BackupPrefix string
}

func (this *LocalStorage) Init(localPath string, expiryDays int, backupPrefix string) error {
	this.LocalPath = localPath
	this.ExpiryDays = expiryDays
	this.BackupPrefix = NormalizeBackupPrefix(backupPrefix)

	return nil
}

func (this *LocalStorage) GetTargetObjectName(ctx context.Context, objectName string) (string, error) {
	if err := contextOrBackground(ctx).Err(); err != nil {
		return "", err
	}

	if objectName == "" {
		return this.getLastUpdatedFile()
	}

	resolved, found, err := resolveExplicitObjectName(this.BackupPrefix, objectName, func(candidate string) (bool, error) {
		targetPath, err := utils.ResolvePathWithinRoot(this.LocalPath, filepath.FromSlash(candidate))
		if err != nil {
			return false, err
		}
		if _, err := os.Stat(targetPath); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	})
	if err != nil {
		return "", err
	}
	if found {
		return resolved, nil
	}

	return "", fmt.Errorf("object %q not found in %q", objectName, this.LocalPath)
}

func (this *LocalStorage) Upload(ctx context.Context, objectName string, filePath string) (string, error) {
	targetPath, err := utils.ResolvePathWithinRoot(this.LocalPath, objectName)
	if err != nil {
		return "", err
	}
	err = copyFile(ctx, filePath, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	sourceInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to verify source file: %w", err)
	}
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to verify uploaded object: %w", err)
	}
	if targetInfo.Size() != sourceInfo.Size() {
		return "", fmt.Errorf("failed to verify uploaded object: size mismatch")
	}

	return targetPath, nil
}

func (this *LocalStorage) Download(ctx context.Context, objectName string, filePath string) error {
	sourceFile, err := utils.ResolvePathWithinRoot(this.LocalPath, objectName)
	if err != nil {
		return err
	}

	err = copyFile(ctx, sourceFile, filePath)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

func (this *LocalStorage) DeleteOldObjects(ctx context.Context, currentObjectName string) error {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	// If expiry days is not set, do not delete backups
	if this.ExpiryDays == 0 {
		return nil
	}

	objects, err := this.listScopedObjects()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, obj := range objects {
		daysOld := now.Sub(obj.ModifiedAt).Hours() / 24
		mlog.Logvf(mlog.Info, "Checking object: %s (%.1f days old)", obj.Name, daysOld)
	}

	return deleteExpiredObjects(objects, this.BackupPrefix, this.ExpiryDays, now, currentObjectName, func(name string) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		targetPath, err := utils.ResolvePathWithinRoot(this.LocalPath, filepath.FromSlash(name))
		if err != nil {
			return err
		}
		if err := os.Remove(targetPath); err != nil {
			return err
		}
		mlog.Logvf(mlog.Info, "Deleted file: %s", filepath.Base(targetPath))
		return nil
	})
}

func (this *LocalStorage) Close() error {
	return nil
}

func (this *LocalStorage) getLastUpdatedFile() (string, error) {
	objects, err := this.listScopedObjects()
	if err != nil {
		return "", err
	}

	latest, ok := latestEligibleObject(objects, this.BackupPrefix)
	if !ok {
		return "", fmt.Errorf("no objects found in %q", this.LocalPath)
	}

	return latest.Name, nil
}

func (this *LocalStorage) listScopedObjects() ([]objectTimestamp, error) {
	scopeRoot, err := this.getScopeRoot()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(scopeRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	objects := make([]objectTimestamp, 0)
	err = filepath.Walk(scopeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(this.LocalPath, path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink objects are not allowed: %q", relPath)
		}

		objects = append(objects, objectTimestamp{
			Name:       filepath.ToSlash(relPath),
			ModifiedAt: info.ModTime(),
		})
		return nil
	})

	return objects, err
}

func (this *LocalStorage) getScopeRoot() (string, error) {
	prefixPath := strings.TrimSuffix(this.BackupPrefix, "/")
	return utils.ResolvePathWithinRoot(this.LocalPath, filepath.FromSlash(prefixPath))
}

func copyFile(ctx context.Context, sourceFile string, destFile string) (retErr error) {
	ctx = contextOrBackground(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		return err
	}
	if destInfo, err := os.Stat(destFile); err == nil && os.SameFile(sourceInfo, destInfo) {
		return utils.ErrSameFile
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := source.Close(); retErr == nil && closeErr != nil {
			retErr = closeErr
		}
	}()

	return utils.WriteFileAtomically(destFile, func(dest *os.File) error {
		buf := make([]byte, 32*1024)
		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			n, readErr := source.Read(buf)
			if n > 0 {
				if _, err := dest.Write(buf[:n]); err != nil {
					return err
				}
			}
			if readErr == nil {
				continue
			}
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	})
}
