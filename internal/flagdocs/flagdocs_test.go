package flagdocs

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	if got, want := normalizeMarkdownTables(string(content)), normalizeMarkdownTables(Markdown()); got != want {
		t.Fatalf("flags.md is out of date; regenerate it from the current flag definitions")
	}
}

func normalizeMarkdownTables(content string) string {
	lines := strings.Split(content, "\n")
	normalized := make([]string, 0, len(lines))
	previousBlank := false
	for _, line := range lines {
		normalizedLine := normalizeMarkdownTableLine(line)
		if strings.TrimSpace(normalizedLine) == "" {
			if previousBlank {
				continue
			}
			previousBlank = true
			normalized = append(normalized, "")
			continue
		}

		previousBlank = false
		normalized = append(normalized, normalizedLine)
	}

	return strings.TrimRight(strings.Join(normalized, "\n"), "\n")
}

func normalizeMarkdownTableLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return strings.TrimRight(line, " \t")
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) < 3 {
		return strings.TrimRight(line, " \t")
	}

	normalized := make([]string, 0, len(parts)-2)
	for _, part := range parts[1 : len(parts)-1] {
		normalized = append(normalized, strings.TrimSpace(part))
	}

	return "| " + strings.Join(normalized, " | ") + " |"
}
