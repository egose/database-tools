package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

type LocalStorage struct {
	LocalPath  string
	ExpiryDays int
}

func (this *LocalStorage) Init(localPath string, expiryDays int) error {
	this.LocalPath = localPath
	this.ExpiryDays = expiryDays

	return nil
}

func (this *LocalStorage) GetTargetObjectName(objectName string) (string, error) {
	if objectName == "" {
		return this.getLastUpdatedFile()
	}

	return objectName, nil
}

func (this *LocalStorage) Upload(objectName string, filePath string) (string, error) {
	targetPath, err := utils.ResolvePathWithinRoot(this.LocalPath, objectName)
	if err != nil {
		return "", err
	}
	err = copyFile(filePath, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	return targetPath, nil
}

func (this *LocalStorage) Download(objectName string, filePath string) error {
	sourceFile, err := utils.ResolvePathWithinRoot(this.LocalPath, objectName)
	if err != nil {
		return err
	}

	err = copyFile(sourceFile, filePath)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

func (this *LocalStorage) DeleteOldObjects() error {
	// If expiry days is not set, do not delete backups
	if this.ExpiryDays == 0 {
		return nil
	}

	err := filepath.Walk(this.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		mlog.Logvf(mlog.Info, "Checking old objects: %s", path)

		if info.IsDir() {
			return nil
		}

		if isExpired(info.ModTime(), this.ExpiryDays, time.Now()) {
			if err := os.Remove(path); err != nil {
				return err
			}
			mlog.Logvf(mlog.Info, "Deleted file: %s", filepath.Base(path))
		}

		return nil
	})

	return err
}

func (this *LocalStorage) getLastUpdatedFile() (string, error) {
	var lastModTime time.Time
	var lastFile string

	err := filepath.Walk(this.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if lastFile == "" || info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()
				lastFile = path
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return filepath.Base(lastFile), nil
}

func copyFile(sourceFile string, destFile string) error {
	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := utils.CreateFile(destFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
