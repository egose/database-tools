package flagdocs

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/mongoarchive"
	"github.com/egose/database-tools/mongounarchive"
)

func TestPublicEnvironmentLookupsAreDocumented(t *testing.T) {
	repoRoot := testRepoRoot(t)
	documented := documentedEnvKeys()
	internalOnly := map[string]bool{
		"AZURITE_ACCOUNT_KEY":         true,
		"AZURITE_ACCOUNT_NAME":        true,
		"AZURITE_CONTAINER":           true,
		"AZURITE_URL":                 true,
		"DATABASE_URL":                true,
		"FAKE_GCP_BUCKET":             true,
		"FAKE_GCP_PORT":               true,
		"MINIO_ACCESS_KEY":            true,
		"MINIO_BUCKET":                true,
		"MINIO_SECRET_KEY":            true,
		"MINIO_URL":                   true,
		"STORAGE_EMULATOR_HOST":       true,
		"MONGOARCHIVE__BACKUP_PREFIX": true,
	}
	lookupPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\.GetValue\("([A-Z0-9_]+)"`),
		regexp.MustCompile(`os\.Getenv\("([A-Z0-9_]+)"`),
		regexp.MustCompile(`LookupEnv\("([A-Z0-9_]+)"`),
		regexp.MustCompile(`Read(?:WorkspaceBase|OptionalDuration|PositiveInt|PositiveInt64)\([^,]+,\s*"([A-Z0-9_]+)"`),
	}

	undocumented := map[string]bool{}
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range lookupPatterns {
			for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
				key := match[1]
				if documented[key] || internalOnly[key] {
					continue
				}
				undocumented[key] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(undocumented) > 0 {
		keys := make([]string, 0, len(undocumented))
		for key := range undocumented {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		t.Fatalf("public environment lookups are missing from generated docs or the internal-only allowlist: %s", strings.Join(keys, ", "))
	}
}

func documentedEnvKeys() map[string]bool {
	documented := map[string]bool{}
	for _, command := range []toolconfig.CommandDoc{mongoarchive.FlagDocumentation(), mongounarchive.FlagDocumentation()} {
		for _, flag := range command.Flags {
			addDocumentedEnvKey(documented, flag.EnvVar)
		}
		for _, envVar := range command.EnvVars {
			addDocumentedEnvKey(documented, "`"+envVar.EnvVar+"`")
		}
	}
	return documented
}

func addDocumentedEnvKey(documented map[string]bool, markdownEnv string) {
	if !strings.HasPrefix(markdownEnv, "`") || !strings.HasSuffix(markdownEnv, "`") {
		return
	}
	envVar := strings.Trim(markdownEnv, "`")
	documented[envVar] = true
	for _, prefix := range []string{"MONGOARCHIVE__", "MONGOUNARCHIVE__", "MONGO__"} {
		if strings.HasPrefix(envVar, prefix) {
			documented[strings.TrimPrefix(envVar, prefix)] = true
		}
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
