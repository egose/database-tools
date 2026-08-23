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

func TestParseFlagsParsesRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	cfg, _, err := parseFlags(newTestFlagSet("mongo-archive"), mapEnv{
		"DUMP_PATH":                 "/archive/work",
		"STORAGE_OPERATION_TIMEOUT": "2s",
		"NOTIFICATION_TIMEOUT":      "3s",
	}, nil)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if cfg.WorkspaceBasePath != "/archive/work" {
		t.Fatalf("WorkspaceBasePath = %q, want injected value", cfg.WorkspaceBasePath)
	}
	if cfg.StorageOperationTimeout != 2*time.Second || cfg.NotificationTimeout != 3*time.Second {
		t.Fatalf("timeouts = %v/%v, want 2s/3s", cfg.StorageOperationTimeout, cfg.NotificationTimeout)
	}
}

func TestParseFlagsRejectsRestoreOnlyStorageBackendFlag(t *testing.T) {
	t.Parallel()

	_, _, err := parseFlags(newTestFlagSet("mongo-archive"), mapEnv{}, []string{"--storage-backend=local"})
	if err == nil || !strings.Contains(err.Error(), "storage-backend") {
		t.Fatalf("parseFlags() error = %v, want unknown storage-backend flag", err)
	}
}

func TestParseFlagsRejectsInvalidRuntimeEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  mapEnv
		want string
	}{
		{name: "storage timeout", env: mapEnv{"STORAGE_OPERATION_TIMEOUT": "bad"}, want: "STORAGE_OPERATION_TIMEOUT"},
		{name: "notification timeout", env: mapEnv{"NOTIFICATION_TIMEOUT": "0s"}, want: "NOTIFICATION_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := parseFlags(newTestFlagSet("mongo-archive"), tt.env, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseFlags() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseFlagsRejectsIncompleteStorageConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     mapEnv
		missing []string
		omitted []string
	}{
		{
			name:    "local plus AWS bucket only",
			env:     mapEnv{"LOCAL_PATH": t.TempDir(), "AWS_BUCKET": "archive-bucket"},
			missing: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
			omitted: []string{"archive-bucket"},
		},
		{
			name:    "local plus Azure account name only",
			env:     mapEnv{"LOCAL_PATH": t.TempDir(), "AZ_ACCOUNT_NAME": "archive-account"},
			missing: []string{"AZ_ACCOUNT_KEY", "AZ_CONTAINER_NAME"},
			omitted: []string{"archive-account"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseFlags(newTestFlagSet("mongo-archive"), tt.env, nil)
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
