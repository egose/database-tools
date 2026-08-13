package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
)

var ErrSameFile = errors.New("source and destination refer to the same file")

const (
	defaultArchiveMaxEntries    = 100000
	defaultArchiveMaxEntryBytes = int64(32 << 30)
	defaultArchiveMaxTotalBytes = int64(256 << 30)
)

type ArchiveExtractionLimits struct {
	MaxEntries    int
	MaxEntryBytes int64
	MaxTotalBytes int64
}

func DefaultArchiveExtractionLimits() ArchiveExtractionLimits {
	return ArchiveExtractionLimits{
		MaxEntries:    defaultArchiveMaxEntries,
		MaxEntryBytes: defaultArchiveMaxEntryBytes,
		MaxTotalBytes: defaultArchiveMaxTotalBytes,
	}
}

func Tar(root string, destPath string) error {
	err := DeleteFile(destPath)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(destPath), 0o700)
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

func UnTar(filePath string, destPath string, limits ArchiveExtractionLimits) error {
	if err := limits.Validate(); err != nil {
		return err
	}

	destParent := filepath.Dir(destPath)
	if err := os.MkdirAll(destParent, 0o700); err != nil {
		return err
	}

	stagingDir, err := os.MkdirTemp(destParent, "."+filepath.Base(destPath)+".partial-")
	if err != nil {
		return err
	}

	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	if err := extractTarGz(filePath, stagingDir, limits); err != nil {
		return err
	}

	if err := DeleteDirectory(destPath); err != nil {
		return err
	}

	if err := os.Rename(stagingDir, destPath); err != nil {
		return err
	}

	shouldCleanup = false
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

// WriteFileAtomically writes through a randomized same-directory temporary file
// and renames it into place after a successful close. On platforms where
// replacing an existing file via rename is not supported, the rename fails and
// the temporary file is still removed.
func WriteFileAtomically(filePath string, write func(*os.File) error) (retErr error) {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	cleanPath := filepath.Clean(filePath)
	parentDir := filepath.Dir(cleanPath)
	if err := ensureNoSymlinkComponents(parentDir); err != nil {
		return err
	}
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(cleanPath); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(parentDir, "."+filepath.Base(cleanPath)+".partial-")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	closed := false
	shouldCleanup := true
	defer func() {
		if !closed {
			if closeErr := tempFile.Close(); retErr == nil && closeErr != nil {
				retErr = closeErr
			}
		}
		if shouldCleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(0o600); err != nil {
		return err
	}
	if err := write(tempFile); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		closed = true
		return err
	}
	closed = true

	if err := os.Rename(tempPath, cleanPath); err != nil {
		return err
	}

	shouldCleanup = false
	return nil
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

func ResolvePathWithinRoot(root string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("path name cannot be empty")
	}

	if filepath.IsAbs(name) {
		return "", fmt.Errorf("absolute paths are not allowed: %q", name)
	}

	cleanRoot := filepath.Clean(root)
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == ".." {
		return "", fmt.Errorf("invalid relative path: %q", name)
	}

	targetPath := filepath.Join(cleanRoot, cleanName)
	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to validate path %q: %w", name, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root directory: %q", name)
	}
	if err := ensureNoSymlinkComponents(targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func CopyFileAtomically(sourceFile string, destFile string) (retErr error) {
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		return err
	}
	if err := ensureDistinctFiles(sourceInfo, destFile); err != nil {
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

	return WriteFileAtomically(destFile, func(dest *os.File) error {
		_, err := io.Copy(dest, source)
		return err
	})
}

func ensureDistinctFiles(sourceInfo os.FileInfo, destFile string) error {
	destInfo, err := os.Stat(destFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if os.SameFile(sourceInfo, destInfo) {
		return ErrSameFile
	}
	return nil
}

func ensureNoSymlinkComponents(targetPath string) error {
	cleanPath := filepath.Clean(targetPath)
	volume := filepath.VolumeName(cleanPath)
	remainder := strings.TrimPrefix(cleanPath, volume)
	current := volume
	if filepath.IsAbs(cleanPath) {
		current += string(filepath.Separator)
		remainder = strings.TrimPrefix(remainder, string(filepath.Separator))
	}

	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		if current == "" || current == volume+string(filepath.Separator) {
			current += part
		} else {
			current = filepath.Join(current, part)
		}

		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path contains symlink component: %q", current)
		}
	}

	return nil
}

func (l ArchiveExtractionLimits) Validate() error {
	if l.MaxEntries <= 0 {
		return fmt.Errorf("archive max entries must be positive")
	}
	if l.MaxEntryBytes <= 0 {
		return fmt.Errorf("archive max entry bytes must be positive")
	}
	if l.MaxTotalBytes <= 0 {
		return fmt.Errorf("archive max total bytes must be positive")
	}
	if l.MaxEntryBytes > l.MaxTotalBytes {
		return fmt.Errorf("archive max entry bytes cannot exceed max total bytes")
	}
	return nil
}

func extractTarGz(filePath string, destRoot string, limits ArchiveExtractionLimits) error {
	source, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer source.Close()

	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	entryCount := 0
	var totalBytes int64

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		entryCount++
		if entryCount > limits.MaxEntries {
			return fmt.Errorf("archive contains too many entries: %d exceeds limit %d", entryCount, limits.MaxEntries)
		}

		targetPath, err := resolveArchiveEntryPath(destRoot, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := validateArchiveEntrySize(header.Name, header.Size, limits, totalBytes); err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
				return err
			}

			if err := writeArchiveFile(targetPath, tarReader, header.Size); err != nil {
				return err
			}

			totalBytes += header.Size
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("archive entry %q uses unsupported link type", header.Name)
		case tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			return fmt.Errorf("archive entry %q uses unsupported special file type", header.Name)
		default:
			return fmt.Errorf("archive entry %q uses unsupported type %d", header.Name, header.Typeflag)
		}
	}
}

func resolveArchiveEntryPath(root string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("archive entry path cannot be empty")
	}
	if path.IsAbs(name) {
		return "", fmt.Errorf("absolute archive entry paths are not allowed: %q", name)
	}

	cleanName := path.Clean(name)
	if cleanName == "." || cleanName == ".." {
		return "", fmt.Errorf("invalid archive entry path: %q", name)
	}
	if strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("archive entry path escapes destination: %q", name)
	}

	return ResolvePathWithinRoot(root, filepath.FromSlash(cleanName))
}

func validateArchiveEntrySize(name string, size int64, limits ArchiveExtractionLimits, totalBytes int64) error {
	if size < 0 {
		return fmt.Errorf("archive entry %q has invalid negative size", name)
	}
	if size > limits.MaxEntryBytes {
		return fmt.Errorf("archive entry %q exceeds max entry size of %d bytes", name, limits.MaxEntryBytes)
	}
	if totalBytes > limits.MaxTotalBytes-size {
		return fmt.Errorf("archive entry %q exceeds max extracted size of %d bytes", name, limits.MaxTotalBytes)
	}
	return nil
}

func writeArchiveFile(targetPath string, reader io.Reader, size int64) error {
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.CopyN(targetFile, reader, size); err != nil {
		return err
	}

	return nil
}

func getChildren(targetDir string) ([]string, error) {
	return listDirectChildren(targetDir, os.ReadDir)
}

func listDirectChildren(targetDir string, readDir func(string) ([]os.DirEntry, error)) ([]string, error) {
	entries, err := readDir(targetDir)
	if err != nil {
		return nil, err
	}

	children := make([]string, 0, len(entries))
	for _, entry := range entries {
		children = append(children, filepath.Join(targetDir, entry.Name()))
	}

	return children, nil
}
