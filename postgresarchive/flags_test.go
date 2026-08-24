package postgresarchive

import (
	"context"
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

type mapEnv map[string]string

func (env mapEnv) GetValue(key string, defaults ...string) string {
	if value := env[key]; value != "" {
		return value
	}
	for _, value := range defaults {
		if value != "" {
			return value
		}
	}
	return ""
}

func testFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("postgres-archive", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func TestParseFlagsUsesExactEnvironmentPrecedenceAndCLIOverride(t *testing.T) {
	for _, key := range []string{"POSTGRESARCHIVE__HOST", "POSTGRES__HOST", "HOST"} {
		t.Setenv(key, "")
	}
	t.Setenv("POSTGRESARCHIVE__DATABASE", "inventory")
	t.Setenv("HOST", "unprefixed")
	t.Setenv("POSTGRES__HOST", "shared")
	t.Setenv("POSTGRESARCHIVE__HOST", "archive")
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")

	cfg, _, err := parseFlags(testFlagSet(), env, nil)
	if err != nil || cfg.Connection.Host != "archive" {
		t.Fatalf("specific precedence = %q, %v", cfg.Connection.Host, err)
	}
	t.Setenv("POSTGRESARCHIVE__HOST", "")
	cfg, _, err = parseFlags(testFlagSet(), env, nil)
	if err != nil || cfg.Connection.Host != "shared" {
		t.Fatalf("shared precedence = %q, %v", cfg.Connection.Host, err)
	}
	t.Setenv("POSTGRES__HOST", "")
	cfg, _, err = parseFlags(testFlagSet(), env, nil)
	if err != nil || cfg.Connection.Host != "unprefixed" {
		t.Fatalf("unprefixed precedence = %q, %v", cfg.Connection.Host, err)
	}
	cfg, _, err = parseFlags(testFlagSet(), env, []string{"--host=cli"})
	if err != nil || cfg.Connection.Host != "cli" {
		t.Fatalf("CLI precedence = %q, %v", cfg.Connection.Host, err)
	}
}

func TestParseFlagsBuildsTypedConnectionAndPostgreSQLPrefix(t *testing.T) {
	cfg, _, err := parseFlags(testFlagSet(), mapEnv{}, []string{
		"--host=db.internal", "--port=5544", "--user=backup", "--database=inventory",
		"--ssl-mode=verify-full", "--password=test-password", "--backup-prefix=tenant/postgres", // pragma: allowlist secret
	})
	if err != nil {
		t.Fatal(err)
	}
	want := postgresclient.ConnectionOptions{Host: "db.internal", Port: 5544, User: "backup", Database: "inventory", SSLMode: postgresclient.SSLModeVerifyFull, Password: "test-password"}
	if cfg.Connection != want {
		t.Fatalf("Connection = %#v, want %#v", cfg.Connection, want)
	}
	if cfg.BackupPrefix != "tenant/postgres" {
		t.Fatalf("BackupPrefix = %q", cfg.BackupPrefix)
	}

	defaultCfg, _, err := parseFlags(testFlagSet(), mapEnv{"DATABASE": "inventory"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if defaultCfg.BackupPrefix != storage.PostgreSQLDefaultBackupPrefix {
		t.Fatalf("default BackupPrefix = %q, want %q", defaultCfg.BackupPrefix, storage.PostgreSQLDefaultBackupPrefix)
	}
}

func TestParseFlagsURIAndDiscreteValidationIsSecretSafe(t *testing.T) {
	secret := "not-a-real-secret" // pragma: allowlist secret
	uri := "postgresql://backup:" + secret + "@db.example/inventory?sslmode=require"
	cfg, _, err := parseFlags(testFlagSet(), mapEnv{}, []string{"--uri=" + uri})
	if err != nil {
		t.Fatalf("URI config error = %v", err)
	}
	if database, err := cfg.Connection.DatabaseName(); err != nil || database != "inventory" {
		t.Fatalf("DatabaseName() = %q, %v", database, err)
	}

	_, _, err = parseFlags(testFlagSet(), mapEnv{}, []string{"--uri=" + uri, "--host=other"})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("mixed config error = %v", err)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), uri) {
		t.Fatalf("mixed config error leaked secret: %v", err)
	}
}

func TestParseFlagsRejectsInvalidConfigurationBeforeUse(t *testing.T) {
	tests := []struct {
		name string
		env  mapEnv
		args []string
		want string
	}{
		{name: "missing database", want: "database is required"},
		{name: "invalid port", args: []string{"--database=inventory", "--port=70000"}, want: "port"},
		{name: "invalid SSL mode", args: []string{"--database=inventory", "--ssl-mode=unsafe"}, want: "SSL mode"},
		{name: "negative retention", args: []string{"--database=inventory", "--expiry-days=-1"}, want: "expiry-days"},
		{name: "invalid timeout", env: mapEnv{"DATABASE": "inventory", "STORAGE_OPERATION_TIMEOUT": "bad"}, want: "STORAGE_OPERATION_TIMEOUT"},
		{name: "incomplete storage", env: mapEnv{"DATABASE": "inventory", "AWS_BUCKET": "private-bucket"}, want: "AWS_ACCESS_KEY_ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseFlags(testFlagSet(), tt.env, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseFlags() error = %v, want %q", err, tt.want)
			}
			if strings.Contains(err.Error(), "private-bucket") {
				t.Fatalf("error exposed supplied storage value: %v", err)
			}
		})
	}
}

func TestParseFlagsVersionIsSideEffectFreeAndRuntimeOptionsAreTyped(t *testing.T) {
	if _, show, err := parseFlags(testFlagSet(), mapEnv{}, []string{"--version"}); err != nil || !show {
		t.Fatalf("--version = show %v, error %v", show, err)
	}
	cfg, show, err := parseFlags(testFlagSet(), mapEnv{
		"DATABASE": "inventory", "DUMP_PATH": "/private/work", "STORAGE_OPERATION_TIMEOUT": "2s", "NOTIFICATION_TIMEOUT": "3s",
	}, []string{"--keep", "--cron", "--cron-expression=*/5 * * * *", "--tz=UTC", "--expiry-days=7"})
	if err != nil || show {
		t.Fatalf("parseFlags() = show %v, error %v", show, err)
	}
	if cfg.WorkspaceBasePath != "/private/work" || cfg.StorageOperationTimeout != 2*time.Second || cfg.NotificationTimeout != 3*time.Second {
		t.Fatalf("runtime options = %#v", cfg.RuntimeOptions)
	}
	if !cfg.Keep || !cfg.Cron || cfg.CronExpression != "*/5 * * * *" || cfg.Location != time.UTC || cfg.ExpiryDays != 7 {
		t.Fatalf("archive controls = %#v", cfg)
	}
}

func TestGetStoragesAlwaysUsesPostgreSQLDefaultOrConfiguredPrefix(t *testing.T) {
	for _, test := range []struct{ configured, want string }{
		{want: storage.PostgreSQLDefaultBackupPrefix},
		{configured: "tenant/postgres", want: "tenant/postgres/"},
	} {
		cfg := &Config{StorageOptions: toolconfig.StorageOptions{LocalPath: t.TempDir(), BackupPrefix: test.configured}}
		backends, err := cfg.GetStorages(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		local, ok := backends[0].(*storage.LocalStorage)
		if !ok || local.BackupPrefix != test.want {
			t.Fatalf("local storage = %#v, want prefix %q", backends[0], test.want)
		}
	}
}
