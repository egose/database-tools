package mongounarchive

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

func TestGetMongounarchiveOptionsDoesNotDropByDefault(t *testing.T) {
	cfg := &Config{MongoOptions: toolconfig.MongoOptions{DB: "testdb"}, RestoreExecutionOptions: RestoreExecutionOptions{Drop: false}}

	options := cfg.GetMongounarchiveOptions("/tmp/restore")
	for _, option := range options {
		if option == "--drop" {
			t.Fatalf("GetMongounarchiveOptions() unexpectedly included %q", option)
		}
	}
}

func TestGetMongoConnectionURIBuildsFromFlags(t *testing.T) {
	cfg := &Config{
		MongoOptions: toolconfig.MongoOptions{
			Host:                    "rs0/db1.example.com,db2.example.com",
			Port:                    "27018",
			Username:                "user",
			Password:                "secret",
			AuthenticationDatabase:  "admin",
			AuthenticationMechanism: "SCRAM-SHA-256",
			SSL:                     true,
			GSSAPIServiceName:       "mongodb",
		},
	}

	uri, err := cfg.MongoOptions.MongoConnectionURI()
	if err != nil {
		t.Fatalf("getMongoConnectionURI() returned error: %v", err)
	}

	wants := []string{
		"mongodb://user:secret@db1.example.com:27018,db2.example.com:27018/?", // pragma: allowlist secret
		"authMechanism=SCRAM-SHA-256",
		"authSource=admin",
		"replicaSet=rs0",
		"tls=true",
	}

	for _, want := range wants {
		if !strings.Contains(uri, want) {
			t.Fatalf("getMongoConnectionURI() = %q, missing %q", uri, want)
		}
	}
}

func TestGetStoragesUsesConfiguredLocalBackend(t *testing.T) {
	cfg := &Config{StorageOptions: toolconfig.StorageOptions{LocalPath: t.TempDir()}}

	storages, err := cfg.GetStorages(context.Background())
	if err != nil {
		t.Fatalf("GetStorages() error = %v", err)
	}
	if len(storages) != 1 {
		t.Fatalf("GetStorages() len = %d, want 1", len(storages))
	}

	if _, ok := storages[0].(*storage.LocalStorage); !ok {
		t.Fatalf("GetStorages()[0] = %T, want *storage.LocalStorage", storages[0])
	}
}

func TestGetUpdatesRejectsOversizedInlineAndFileInputs(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		_, err := (&Config{UpdateOptions: UpdateOptions{Updates: `[{"a":1}]`}, RuntimeOptions: RuntimeOptions{UpdateMaxBytes: 8}}).GetUpdates()
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("GetUpdates() error = %v, want maximum size rejection", err)
		}
	})

	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "updates.json")
		if err := os.WriteFile(path, []byte(`[{"a":1}]`), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := (&Config{UpdateOptions: UpdateOptions{UpdatesFile: path}, RuntimeOptions: RuntimeOptions{UpdateMaxBytes: 8}}).GetUpdates()
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("GetUpdates() error = %v, want maximum size rejection", err)
		}
	})
}

func TestParseFlagsParsesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), restoreMapEnv{
		"RESTORE_PATH":              "/restore/work",
		"STORAGE_OPERATION_TIMEOUT": "2s",
		"UPDATE_TIMEOUT":            "3s",
		"ARCHIVE_MAX_ENTRIES":       "12",
		"ARCHIVE_MAX_ENTRY_BYTES":   "4096",
		"ARCHIVE_MAX_TOTAL_BYTES":   "8192",
		"UPDATE_MAX_BYTES":          "2048",
	}, nil)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg.WorkspaceBasePath != "/restore/work" {
		t.Fatalf("WorkspaceBasePath = %q, want injected value", cfg.WorkspaceBasePath)
	}
	if cfg.StorageOperationTimeout != 2*time.Second || cfg.UpdateTimeout != 3*time.Second {
		t.Fatalf("timeouts = %v/%v, want 2s/3s", cfg.StorageOperationTimeout, cfg.UpdateTimeout)
	}
	if cfg.ArchiveExtractionLimits.MaxEntries != 12 || cfg.ArchiveExtractionLimits.MaxEntryBytes != 4096 || cfg.ArchiveExtractionLimits.MaxTotalBytes != 8192 {
		t.Fatalf("ArchiveExtractionLimits = %+v, want injected values", cfg.ArchiveExtractionLimits)
	}
	if cfg.UpdateMaxBytes != 2048 {
		t.Fatalf("UpdateMaxBytes = %d, want 2048", cfg.UpdateMaxBytes)
	}
}

func TestParseFlagsRejectsInvalidRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  restoreMapEnv
		want string
	}{
		{name: "storage timeout", env: restoreMapEnv{"STORAGE_OPERATION_TIMEOUT": "bad"}, want: "STORAGE_OPERATION_TIMEOUT"},
		{name: "update timeout", env: restoreMapEnv{"UPDATE_TIMEOUT": "0s"}, want: "UPDATE_TIMEOUT"},
		{name: "archive max entries", env: restoreMapEnv{"ARCHIVE_MAX_ENTRIES": "0"}, want: "ARCHIVE_MAX_ENTRIES"},
		{name: "update max bytes", env: restoreMapEnv{"UPDATE_MAX_BYTES": "nope"}, want: "UPDATE_MAX_BYTES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), tt.env, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseFlags() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestGetArchiveExtractionLimitsReturnsParsedConfig(t *testing.T) {
	t.Parallel()

	limits := utils.ArchiveExtractionLimits{MaxEntries: 12, MaxEntryBytes: 4096, MaxTotalBytes: 8192}
	cfg := &Config{RuntimeOptions: RuntimeOptions{ArchiveExtractionLimits: limits}}
	if got := cfg.GetArchiveExtractionLimits(); got != limits {
		t.Fatalf("GetArchiveExtractionLimits() = %+v, want %+v", got, limits)
	}
}

func TestParseFlagsRejectsDryRunWithUpdates(t *testing.T) {
	_, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), restoreMapEnv{}, []string{"--dry-run", "--updates=[]"})
	if err == nil || !strings.Contains(err.Error(), "--dry-run") {
		t.Fatalf("parseFlags() error = %v, want dry-run validation failure", err)
	}
}

func TestParseFlagsRejectsIncompleteStorageConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     restoreMapEnv
		missing []string
		omitted []string
	}{
		{
			name:    "local plus AWS bucket only",
			env:     restoreMapEnv{"LOCAL_PATH": t.TempDir(), "AWS_BUCKET": "restore-bucket"},
			missing: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
			omitted: []string{"restore-bucket"},
		},
		{
			name:    "local plus Azure account name only",
			env:     restoreMapEnv{"LOCAL_PATH": t.TempDir(), "AZ_ACCOUNT_NAME": "restore-account"},
			missing: []string{"AZ_ACCOUNT_KEY", "AZ_CONTAINER_NAME"},
			omitted: []string{"restore-account"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), tt.env, nil)
			if err == nil {
				t.Fatal("parseFlags() expected error")
			}
			for _, missing := range tt.missing {
				if !strings.Contains(err.Error(), missing) {
					t.Fatalf("parseFlags() error = %q, missing %q", err, missing)
				}
			}
			for _, value := range tt.omitted {
				if strings.Contains(err.Error(), value) {
					t.Fatalf("parseFlags() error = %q, leaked supplied value %q", err, value)
				}
			}
		})
	}
}

func TestParseFlagsRunsInParallelWithoutGlobalState(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), restoreMapEnv{"UPDATES": `[{"collection":"users","filter":{"active":true},"update":{"$set":{"active":false}}}]`}, []string{"--storage-backend=local"})
			if err != nil {
				t.Errorf("parseFlags() error = %v", err)
				return
			}
			if !cfg.HasUpdates() || cfg.StorageBackend != "local" {
				t.Errorf("parseFlags() returned unexpected config: %+v", cfg)
			}
		}()
	}
	wg.Wait()
}

type restoreMapEnv map[string]string

func (e restoreMapEnv) GetValue(key string, defaults ...string) string {
	if value, ok := e[key]; ok && value != "" {
		return value
	}
	for _, fallback := range defaults {
		if fallback != "" {
			return fallback
		}
	}
	return ""
}

func newRestoreTestFlagSet(name string) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	return flagSet
}
