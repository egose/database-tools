package mongoarchive

import (
	"context"
	"flag"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
)

func TestGetMongodumpOptionsPrunesURI(t *testing.T) {
	cfg := &Config{
		MongoOptions: toolconfig.MongoOptions{
			URI:      "mongodb://user:secret@host1:27017,host2:27018/testdb?replicaSet=rs0", // pragma: allowlist secret
			URIPrune: true,
		},
	}

	options := cfg.GetMongodumpOptions()
	joined := strings.Join(options, " ")

	if strings.Contains(joined, "secret") {
		t.Fatalf("GetMongodumpOptions() leaked credentials: %q", joined)
	}
	if !strings.Contains(joined, "--uri=mongodb://host1:27017,host2:27018/testdb?replicaSet=rs0") {
		t.Fatalf("GetMongodumpOptions() did not preserve pruned URI hosts/query: %q", joined)
	}
}

func TestConfigCronDefaults(t *testing.T) {
	cfg := &Config{}

	if got := cfg.GetCronExpression(); got != defaultCronExpr {
		t.Fatalf("GetCronExpression() = %q, want %q", got, defaultCronExpr)
	}
	if got, err := parseLocation(""); err != nil || got != time.Local {
		t.Fatalf("parseLocation(\"\") = %v, want %v", got, time.Local)
	}
}

func TestParseFlagsRejectsInvalidRetentionAndNotificationConfig(t *testing.T) {
	t.Run("expiry", func(t *testing.T) {
		_, _, err := parseFlags(newTestFlagSet("mongo-archive"), mapEnv{}, []string{"--expiry-days=-1"})
		if err == nil || !strings.Contains(err.Error(), "expiry-days") {
			t.Fatalf("parseFlags() error = %v, want expiry-days rejection", err)
		}
	})

	t.Run("notification", func(t *testing.T) {
		_, _, err := parseFlags(newTestFlagSet("mongo-archive"), mapEnv{}, []string{"--slack-webhook-url=http://localhost/webhook"})
		if err == nil || !strings.Contains(err.Error(), "Slack webhook URL") {
			t.Fatalf("parseFlags() error = %v, want Slack webhook validation failure", err)
		}
	})
}

func TestParseFlagsRunsInParallelWithoutGlobalState(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg, _, err := parseFlags(newTestFlagSet("mongo-archive"), mapEnv{"QUERY": "{}"}, []string{"--db=testdb", "--expiry-days=0", "--smtp-host=smtp.example.com", "--smtp-from=from@example.com", "--smtp-to=to@example.com"})
			if err != nil {
				t.Errorf("parseFlags() error = %v", err)
				return
			}
			if cfg.DB != "testdb" || cfg.Query != "{}" || cfg.ExpiryDays != 0 {
				t.Errorf("parseFlags() returned unexpected config: %+v", cfg)
			}
		}(i)
	}
	wg.Wait()
}

type mapEnv map[string]string

func (e mapEnv) GetValue(key string, defaults ...string) string {
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

func newTestFlagSet(name string) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	return flagSet
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

func TestGetStoragesPropagatesBackupPrefix(t *testing.T) {
	cfg := &Config{StorageOptions: toolconfig.StorageOptions{LocalPath: t.TempDir(), BackupPrefix: "custom-prefix"}}

	storages, err := cfg.GetStorages(context.Background())
	if err != nil {
		t.Fatalf("GetStorages() error = %v", err)
	}
	localStorage, ok := storages[0].(*storage.LocalStorage)
	if !ok {
		t.Fatalf("GetStorages()[0] = %T, want *storage.LocalStorage", storages[0])
	}
	if localStorage.BackupPrefix != "custom-prefix/" {
		t.Fatalf("BackupPrefix = %q, want %q", localStorage.BackupPrefix, "custom-prefix/")
	}
}
