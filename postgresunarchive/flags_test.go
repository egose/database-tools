package postgresunarchive

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
	fs := flag.NewFlagSet("postgres-unarchive", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func TestParseFlagsUsesExactEnvironmentPrecedenceAndCLIOverride(t *testing.T) {
	for _, key := range []string{"POSTGRESUNARCHIVE__HOST", "POSTGRES__HOST", "HOST", "POSTGRESUNARCHIVE__DATABASE"} {
		t.Setenv(key, "")
	}
	t.Setenv("POSTGRESUNARCHIVE__DATABASE", "inventory")
	t.Setenv("HOST", "unprefixed")
	t.Setenv("POSTGRES__HOST", "shared")
	t.Setenv("POSTGRESUNARCHIVE__HOST", "restore")
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")

	for _, test := range []struct {
		name string
		set  func()
		args []string
		want string
	}{
		{name: "command specific", set: func() {}, want: "restore"},
		{name: "shared", set: func() { t.Setenv("POSTGRESUNARCHIVE__HOST", "") }, want: "shared"},
		{name: "unprefixed", set: func() { t.Setenv("POSTGRES__HOST", "") }, want: "unprefixed"},
		{name: "CLI", set: func() {}, args: []string{"--host=cli"}, want: "cli"},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.set()
			cfg, _, err := parseFlags(testFlagSet(), env, test.args)
			if err != nil || cfg.Connection.Host != test.want {
				t.Fatalf("host = %q, error = %v, want %q", cfg.Connection.Host, err, test.want)
			}
		})
	}
}

func TestParseFlagsBuildsMinimalTypedConfiguration(t *testing.T) {
	cfg, show, err := parseFlags(testFlagSet(), mapEnv{}, []string{
		"--host=db.internal", "--port=5544", "--user=restore", "--database=inventory",
		"--ssl-mode=verify-full", "--password=test-password", "--backup-prefix=tenant/postgres", // pragma: allowlist secret
		"--storage-backend=local", "--object-name=tenant/postgres/archive.tar.gz", "--keep",
	})
	if err != nil || show {
		t.Fatalf("parseFlags() = show %v, error %v", show, err)
	}
	want := postgresclient.ConnectionOptions{Host: "db.internal", Port: 5544, User: "restore", Database: "inventory", SSLMode: postgresclient.SSLModeVerifyFull, Password: "test-password"}
	if cfg.Connection != want || cfg.BackupPrefix != "tenant/postgres" || cfg.StorageBackend != "local" || cfg.ObjectName != "tenant/postgres/archive.tar.gz" || !cfg.Keep {
		t.Fatalf("configuration = %#v", cfg)
	}
	for _, forbidden := range []string{"--clean", "--create", "--single-transaction", "--jobs=2", "--args=--clean"} {
		if _, _, err := parseFlags(testFlagSet(), mapEnv{}, []string{forbidden}); err == nil {
			t.Fatalf("unsupported option %q was accepted", forbidden)
		}
	}
}

func TestParseFlagsUsesPostgreSQLPrefixAndTypedRuntime(t *testing.T) {
	cfg, _, err := parseFlags(testFlagSet(), mapEnv{
		"DATABASE": "inventory", "RESTORE_PATH": "/private/restore", "STORAGE_OPERATION_TIMEOUT": "2s",
		"ARCHIVE_MAX_ENTRIES": "2", "ARCHIVE_MAX_ENTRY_BYTES": "4096", "ARCHIVE_MAX_TOTAL_BYTES": "8192",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupPrefix != storage.PostgreSQLDefaultBackupPrefix {
		t.Fatalf("BackupPrefix = %q", cfg.BackupPrefix)
	}
	if cfg.WorkspaceBasePath != "/private/restore" || cfg.StorageOperationTimeout != 2*time.Second {
		t.Fatalf("runtime = %#v", cfg.RuntimeOptions)
	}
	if cfg.ArchiveExtractionLimits != (utils.ArchiveExtractionLimits{MaxEntries: 2, MaxEntryBytes: 4096, MaxTotalBytes: 8192}) {
		t.Fatalf("limits = %#v", cfg.ArchiveExtractionLimits)
	}
}

func TestParseFlagsRejectsInvalidConfigurationWithoutLeakingSecrets(t *testing.T) {
	secret := "not-a-real-secret" // pragma: allowlist secret
	uri := "postgresql://restore:" + secret + "@db.example/inventory?sslmode=require"
	tests := []struct {
		name string
		env  mapEnv
		args []string
		want string
	}{
		{name: "missing database", want: "target database is required"},
		{name: "invalid port", args: []string{"--database=inventory", "--port=70000"}, want: "port"},
		{name: "invalid SSL mode", args: []string{"--database=inventory", "--ssl-mode=unsafe"}, want: "SSL mode"},
		{name: "URI and discrete", args: []string{"--uri=" + uri, "--host=other"}, want: "cannot be combined"},
		{name: "invalid storage timeout", env: mapEnv{"DATABASE": "inventory", "STORAGE_OPERATION_TIMEOUT": "bad"}, want: "STORAGE_OPERATION_TIMEOUT"},
		{name: "invalid extraction limit", env: mapEnv{"DATABASE": "inventory", "ARCHIVE_MAX_ENTRIES": "0"}, want: "ARCHIVE_MAX_ENTRIES"},
		{name: "incomplete backend", env: mapEnv{"DATABASE": "inventory", "AWS_BUCKET": secret}, want: "AWS_ACCESS_KEY_ID"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseFlags(testFlagSet(), test.env, test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), uri) {
				t.Fatalf("error leaked a secret: %v", err)
			}
		})
	}
}

func TestParseFlagsVersionIgnoresOperationalConfiguration(t *testing.T) {
	_, show, err := parseFlags(testFlagSet(), mapEnv{"PORT": "invalid", "STORAGE_OPERATION_TIMEOUT": "invalid"}, []string{"--version"})
	if err != nil || !show {
		t.Fatalf("--version = show %v, error %v", show, err)
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
