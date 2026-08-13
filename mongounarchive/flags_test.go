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

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
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
		t.Setenv(envPrefix+"UPDATE_MAX_BYTES", "8")

		_, err := (&Config{UpdateOptions: UpdateOptions{Updates: `[{"a":1}]`}}).GetUpdates()
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("GetUpdates() error = %v, want maximum size rejection", err)
		}
	})

	t.Run("file", func(t *testing.T) {
		t.Setenv(envPrefix+"UPDATE_MAX_BYTES", "8")
		path := filepath.Join(t.TempDir(), "updates.json")
		if err := os.WriteFile(path, []byte(`[{"a":1}]`), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := (&Config{UpdateOptions: UpdateOptions{UpdatesFile: path}}).GetUpdates()
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("GetUpdates() error = %v, want maximum size rejection", err)
		}
	})
}

func TestParseFlagsRejectsDryRunWithUpdates(t *testing.T) {
	_, _, err := parseFlags(newRestoreTestFlagSet("mongo-unarchive"), restoreMapEnv{}, []string{"--dry-run", "--updates=[]"})
	if err == nil || !strings.Contains(err.Error(), "--dry-run") {
		t.Fatalf("parseFlags() error = %v, want dry-run validation failure", err)
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
