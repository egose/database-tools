package flagdocs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFlagsDocumentationIsCurrent(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	content, err := os.ReadFile(filepath.Join(repoRoot, "flags.md"))
	if err != nil {
		t.Fatalf("ReadFile(flags.md) error = %v", err)
	}

	if got, want := string(content), Markdown(); got != want {
		t.Fatalf("flags.md is out of date; regenerate it from the current flag definitions")
	}
}
