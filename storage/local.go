package storage

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/egose/database-tools/utils"
)

type LocalStorage struct {
	LocalPath string
}

func (this *LocalStorage) Init(localPath string) error {
	this.LocalPath = localPath

	return nil
}

func (this *LocalStorage) GetTargetObjectName(objectName string) (string, error) {
	if objectName == "" {
		return this.getLastUpdatedFile()
	}

	return objectName, nil
}

func (this *LocalStorage) Upload(objectName string, buffer []byte) (string, error) {
	targetPath := path.Join(this.LocalPath, objectName)
	err := storeBytesToFile(buffer, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	return targetPath, nil
}

func (this *LocalStorage) Download(objectName string, filePath string) error {
	sourceFile := path.Join(this.LocalPath, objectName)

	dest, err := utils.CreateFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	err = copyFile(sourceFile, filePath)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
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

	destination, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func storeBytesToFile(data []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}
