package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDefaultArchiveExtractionLimitsAreValid(t *testing.T) {
	if err := DefaultArchiveExtractionLimits().Validate(); err != nil {
		t.Fatalf("DefaultArchiveExtractionLimits().Validate() error = %v", err)
	}
}

func TestResolvePathWithinRootAllowsNestedPaths(t *testing.T) {
	got, err := ResolvePathWithinRoot("/tmp/root", "nested/archive.tar.gz")
	if err != nil {
		t.Fatalf("ResolvePathWithinRoot() returned error: %v", err)
	}

	want := "/tmp/root/nested/archive.tar.gz"
	if got != want {
		t.Fatalf("ResolvePathWithinRoot() = %q, want %q", got, want)
	}
}

func TestResolvePathWithinRootRejectsTraversal(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "../escape.tar.gz"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected traversal error")
	}
}

func TestResolvePathWithinRootRejectsAbsolutePath(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "/etc/passwd"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected absolute path error")
	}
}

func TestResolvePathWithinRootRejectsSymlinkComponent(t *testing.T) {
	root := t.TempDir()
	linkTarget := t.TempDir()
	if err := os.Symlink(linkTarget, filepath.Join(root, "linked")); err != nil {
		t.Skipf("Symlink() unsupported: %v", err)
	}

	if _, err := ResolvePathWithinRoot(root, filepath.Join("linked", "archive.tar.gz")); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected symlink component error")
	}
}

func TestTarCreatesDestinationParentDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dump")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	childPath := filepath.Join(root, "data.bson")
	if err := os.WriteFile(childPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "archives", "archive.tar.gz")
	if err := Tar(root, archivePath); err != nil {
		t.Fatalf("Tar() error = %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestUnTarCreatesDestinationDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dump")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	childPath := filepath.Join(root, "data.bson")
	if err := os.WriteFile(childPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := Tar(root, archivePath); err != nil {
		t.Fatalf("Tar() error = %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "restore", "nested")
	if err := UnTar(archivePath, destPath, DefaultArchiveExtractionLimits()); err != nil {
		t.Fatalf("UnTar() error = %v", err)
	}

	restoredPath := filepath.Join(destPath, "data.bson")
	if _, err := os.Stat(restoredPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	assertMode(t, destPath, 0o700)
	assertMode(t, restoredPath, 0o600)
}

func TestUnTarRejectsTraversalEntriesWithoutTouchingOutsideSentinel(t *testing.T) {
	archivePath := writeTarGzArchive(t, []tarEntry{{name: "../escape.txt", body: []byte("owned")}})
	root := t.TempDir()
	destPath := filepath.Join(root, "restore")
	sentinelPath := filepath.Join(root, "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("safe"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := UnTar(archivePath, destPath, DefaultArchiveExtractionLimits())
	if err == nil {
		t.Fatal("UnTar() expected traversal error")
	}

	assertFileContent(t, sentinelPath, "safe")
	assertPathAbsent(t, destPath)
}

func TestUnTarRejectsAbsoluteEntries(t *testing.T) {
	archivePath := writeTarGzArchive(t, []tarEntry{{name: "/etc/passwd", body: []byte("owned")}})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, DefaultArchiveExtractionLimits()); err == nil {
		t.Fatal("UnTar() expected absolute path error")
	}

	assertPathAbsent(t, destPath)
}

func TestUnTarRejectsSymlinkEntries(t *testing.T) {
	archivePath := writeTarGzArchive(t, []tarEntry{{name: "link", typeflag: tar.TypeSymlink, linkname: "../escape.txt"}})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, DefaultArchiveExtractionLimits()); err == nil {
		t.Fatal("UnTar() expected symlink error")
	}

	assertPathAbsent(t, destPath)
}

func TestUnTarRejectsSpecialFileEntries(t *testing.T) {
	archivePath := writeTarGzArchive(t, []tarEntry{{name: "fifo", typeflag: tar.TypeFifo}})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, DefaultArchiveExtractionLimits()); err == nil {
		t.Fatal("UnTar() expected special file error")
	}

	assertPathAbsent(t, destPath)
}

func TestUnTarRejectsOversizedEntries(t *testing.T) {
	limits := DefaultArchiveExtractionLimits()
	limits.MaxEntryBytes = 4
	archivePath := writeTarGzArchive(t, []tarEntry{{name: "data.bson", body: []byte("12345")}})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, limits); err == nil {
		t.Fatal("UnTar() expected size limit error")
	}

	assertPathAbsent(t, destPath)
}

func TestUnTarRejectsTooManyEntries(t *testing.T) {
	limits := DefaultArchiveExtractionLimits()
	limits.MaxEntries = 1
	archivePath := writeTarGzArchive(t, []tarEntry{
		{name: "first/data.bson", body: []byte("1")},
		{name: "second/data.bson", body: []byte("2")},
	})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, limits); err == nil {
		t.Fatal("UnTar() expected entry count error")
	}

	assertPathAbsent(t, destPath)
	assertNoTempExtractionDirs(t, filepath.Dir(destPath))
}

func TestUnTarRejectsTotalExtractedBytesLimit(t *testing.T) {
	limits := DefaultArchiveExtractionLimits()
	limits.MaxTotalBytes = 4
	archivePath := writeTarGzArchive(t, []tarEntry{
		{name: "first.bson", body: []byte("12")},
		{name: "second.bson", body: []byte("345")},
	})
	destPath := filepath.Join(t.TempDir(), "restore")

	if err := UnTar(archivePath, destPath, limits); err == nil {
		t.Fatal("UnTar() expected total size limit error")
	}

	assertPathAbsent(t, destPath)
}

func TestWriteFileAtomicallyUsesPrivateParentPermissions(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "nested", "archive.tar.gz")
	if err := WriteFileAtomically(targetPath, func(file *os.File) error {
		_, err := file.Write([]byte("data"))
		return err
	}); err != nil {
		t.Fatalf("WriteFileAtomically() error = %v", err)
	}

	assertMode(t, filepath.Dir(targetPath), 0o700)
	assertMode(t, targetPath, 0o600)
}

func TestWriteFileAtomicallyRemovesPartialFileOnFailure(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "nested", "archive.tar.gz")
	err := WriteFileAtomically(targetPath, func(file *os.File) error {
		if _, err := file.Write([]byte("partial")); err != nil {
			return err
		}
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("WriteFileAtomically() expected error")
	}

	assertPathAbsent(t, targetPath)
	assertNoPartialFiles(t, filepath.Dir(targetPath), filepath.Base(targetPath))
}

func TestWriteFileAtomicallyRejectsSymlinkDestination(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	linkPath := filepath.Join(root, "linked")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("Symlink() unsupported: %v", err)
	}

	targetPath := filepath.Join(linkPath, "archive.tar.gz")
	if err := WriteFileAtomically(targetPath, func(file *os.File) error {
		_, err := file.Write([]byte("data"))
		return err
	}); err == nil {
		t.Fatal("WriteFileAtomically() expected symlink component error")
	}

	assertPathAbsent(t, filepath.Join(outside, "archive.tar.gz"))
}

func TestListDirectChildrenReadsOnlyRootDirectory(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(root, name, "nested", "deep"), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "root.bson"), []byte("data"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	readCalls := 0
	children, err := listDirectChildren(root, func(path string) ([]os.DirEntry, error) {
		readCalls++
		if path != root {
			t.Fatalf("listDirectChildren() read unexpected path %q", path)
		}
		return os.ReadDir(path)
	})
	if err != nil {
		t.Fatalf("listDirectChildren() error = %v", err)
	}
	if readCalls != 1 {
		t.Fatalf("listDirectChildren() readCalls = %d, want 1", readCalls)
	}

	want := []string{
		filepath.Join(root, "alpha"),
		filepath.Join(root, "beta"),
		filepath.Join(root, "root.bson"),
	}
	if got := sortStrings(children); !equalStrings(got, want) {
		t.Fatalf("listDirectChildren() = %#v, want %#v", got, want)
	}
}

func BenchmarkListDirectChildren(b *testing.B) {
	root := b.TempDir()
	for i := range 64 {
		childDir := filepath.Join(root, fmt.Sprintf("child-%03d", i))
		if err := os.MkdirAll(childDir, 0o755); err != nil {
			b.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(childDir, "data.bson"), []byte("data"), 0o600); err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
	}

	deepRoot := filepath.Join(root, "child-000", "nested")
	for i := range 1024 {
		leafDir := filepath.Join(deepRoot, fmt.Sprintf("branch-%04d", i))
		if err := os.MkdirAll(leafDir, 0o755); err != nil {
			b.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(leafDir, "data.bson"), []byte("data"), 0o600); err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		children, err := getChildren(root)
		if err != nil {
			b.Fatalf("getChildren() error = %v", err)
		}
		if len(children) != 64 {
			b.Fatalf("getChildren() len = %d, want 64", len(children))
		}
	}
}

type tarEntry struct {
	name     string
	body     []byte
	typeflag byte
	linkname string
}

func writeTarGzArchive(t *testing.T, entries []tarEntry) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}

		size := int64(len(entry.body))
		if typeflag != tar.TypeReg && typeflag != tar.TypeRegA {
			size = 0
		}

		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o600,
			Size:     size,
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		if typeflag == tar.TypeDir {
			header.Mode = 0o700
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader() error = %v", err)
		}

		if size > 0 {
			if _, err := tarWriter.Write(entry.body); err != nil {
				t.Fatalf("Write() error = %v", err)
			}
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return archivePath
}

func sortStrings(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func assertPathAbsent(t *testing.T, targetPath string) {
	t.Helper()
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be absent, stat err = %v", targetPath, err)
	}
}

func assertFileContent(t *testing.T, filePath string, want string) {
	t.Helper()
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("ReadFile() = %q, want %q", string(got), want)
	}
}

func assertMode(t *testing.T, targetPath string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("mode for %q = %o, want %o", targetPath, info.Mode().Perm(), want)
	}
}

func assertNoTempExtractionDirs(t *testing.T, parent string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "restore" && filepath.Ext(entry.Name()) == "" && len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			t.Fatalf("unexpected leftover extraction directory %q", entry.Name())
		}
	}
}

func assertNoPartialFiles(t *testing.T, parent string, finalName string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	prefix := "." + finalName + ".partial-"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Fatalf("unexpected leftover partial file %q", entry.Name())
		}
	}
}
