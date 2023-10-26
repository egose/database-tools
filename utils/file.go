package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
)

func Tar(root string, destPath string) error {
	err := DeleteFile(destPath)
	if err != nil {
		return err
	}

	paths, err := getChildren(root)
	if err != nil {
		return err
	}

	err = archiver.Archive(paths, destPath)
	if err != nil {
		return err
	}

	return nil
}

func UnTar(filePath string, destPath string) error {
	err := DeleteDirectory(destPath)
	if err != nil {
		return err
	}

	err = archiver.Unarchive(filePath, destPath)
	if err != nil {
		return err
	}

	return nil
}

func DeleteFile(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		err := os.Remove(filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); err == nil {
		err := os.RemoveAll(dirPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadFileToBuffer(filePath string) ([]byte, error) {
	buffer, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func CreateFile(filePath string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	destFile, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return destFile, nil
}

func GetFileNameWithoutExtension(filePath string) string {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)
	fileNameWithoutExt := strings.TrimSuffix(fileName, ext)
	if ext != "" {
		// Remove the last extension if it exists
		fileNameWithoutExt = strings.TrimSuffix(fileNameWithoutExt, filepath.Ext(fileNameWithoutExt))
	}
	return fileNameWithoutExt
}

func getChildren(targetDir string) ([]string, error) {
	var children []string

	err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the target directory itself
		if path == targetDir {
			return nil
		}

		// Check if the entry is a direct child of the target directory
		parentDir := filepath.Dir(path)
		if parentDir == targetDir {
			children = append(children, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return children, nil
}
