package postgresunarchive

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/egose/database-tools/internal/postgresclient"
	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
)

const (
	envPrefix           = "POSTGRESUNARCHIVE__"
	fallbackEnvPrefix   = "POSTGRES__"
	defaultWorkspaceDir = "postgresunarchive"
)

type Config struct {
	Connection postgresclient.ConnectionOptions
	toolconfig.StorageOptions
	ObjectName string
	RuntimeOptions
	Keep bool
}

type RuntimeOptions struct {
	WorkspaceBasePath       string
	StorageOperationTimeout time.Duration
	ArchiveExtractionLimits utils.ArchiveExtractionLimits
}

var flagDefs = struct {
	host, port, user, database, sslMode, uri, password toolconfig.StringFlagDef
	objectName                                         toolconfig.StringFlagDef
	keep, version                                      toolconfig.BoolFlagDef
}{
	host:       toolconfig.StringFlagDef{Name: "host", EnvKey: "HOST", Usage: "PostgreSQL server host"},
	port:       toolconfig.StringFlagDef{Name: "port", EnvKey: "PORT", Usage: "PostgreSQL server port"},
	user:       toolconfig.StringFlagDef{Name: "user", EnvKey: "USER", Usage: "PostgreSQL user"},
	database:   toolconfig.StringFlagDef{Name: "database", EnvKey: "DATABASE", Usage: "existing PostgreSQL database to restore into"},
	sslMode:    toolconfig.StringFlagDef{Name: "ssl-mode", EnvKey: "SSL_MODE", Usage: "libpq SSL mode (disable, allow, prefer, require, verify-ca, verify-full)"},
	uri:        toolconfig.StringFlagDef{Name: "uri", EnvKey: "URI", Usage: "PostgreSQL connection URI"},
	password:   toolconfig.StringFlagDef{Name: "password", EnvKey: "PASSWORD", Usage: "PostgreSQL password"},
	objectName: toolconfig.StringFlagDef{Name: "object-name", EnvKey: "OBJECT_NAME", Usage: "object name to restore; omit to select the latest eligible PostgreSQL archive"},
	keep:       toolconfig.BoolFlagDef{Name: "keep", EnvKey: "KEEP", Usage: "keep the private per-run workspace"},
	version:    toolconfig.BoolFlagDef{Name: "version", Usage: "show the version"},
}

func ParseFlags() (*Config, bool, error) {
	return parseFlags(flag.CommandLine, utils.NewEnv(envPrefix, fallbackEnvPrefix, ""), os.Args[1:])
}

func parseFlags(fs *flag.FlagSet, env toolconfig.EnvReader, args []string) (*Config, bool, error) {
	host, port, user := flagDefs.host.Bind(fs, env), flagDefs.port.Bind(fs, env), flagDefs.user.Bind(fs, env)
	database, sslMode := flagDefs.database.Bind(fs, env), flagDefs.sslMode.Bind(fs, env)
	uri, password := flagDefs.uri.Bind(fs, env), flagDefs.password.Bind(fs, env)
	storageBindings := toolconfig.BindStorageFlagsWithDefaultPrefix(fs, env, storage.PostgreSQLDefaultBackupPrefix)
	storageBackend := toolconfig.BindRestoreStorageBackendFlag(fs, env)
	objectName, keep, showVersion := flagDefs.objectName.Bind(fs, env), flagDefs.keep.Bind(fs, env), flagDefs.version.Bind(fs, env)
	if err := fs.Parse(args); err != nil {
		return nil, false, err
	}
	if *showVersion {
		return &Config{}, true, nil
	}

	parsedPort, err := parsePort(*port)
	if err != nil {
		return nil, false, err
	}
	cfg := &Config{
		Connection: postgresclient.ConnectionOptions{
			Host: *host, Port: parsedPort, User: *user, Database: *database,
			SSLMode: postgresclient.SSLMode(*sslMode), URI: *uri, Password: *password,
		},
		ObjectName: *objectName,
		Keep:       *keep,
	}
	storageBindings.Apply(&cfg.StorageOptions)
	cfg.StorageBackend = *storageBackend
	cfg.RuntimeOptions, err = parseRuntimeOptions(env)
	if err != nil {
		return nil, false, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, false, err
	}
	return cfg, false, nil
}

func parsePort(raw string) (uint16, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 16)
	if err != nil || value == 0 {
		return 0, errors.New("port must be an integer from 1 through 65535")
	}
	return uint16(value), nil
}

func parseRuntimeOptions(env toolconfig.EnvReader) (RuntimeOptions, error) {
	storageTimeout, err := toolconfig.ReadOptionalDuration(env, "STORAGE_OPERATION_TIMEOUT")
	if err != nil {
		return RuntimeOptions{}, err
	}
	limits := utils.DefaultArchiveExtractionLimits()
	limits.MaxEntries, err = toolconfig.ReadPositiveInt(env, "ARCHIVE_MAX_ENTRIES", limits.MaxEntries)
	if err != nil {
		return RuntimeOptions{}, err
	}
	limits.MaxEntryBytes, err = toolconfig.ReadPositiveInt64(env, "ARCHIVE_MAX_ENTRY_BYTES", limits.MaxEntryBytes)
	if err != nil {
		return RuntimeOptions{}, err
	}
	limits.MaxTotalBytes, err = toolconfig.ReadPositiveInt64(env, "ARCHIVE_MAX_TOTAL_BYTES", limits.MaxTotalBytes)
	if err != nil {
		return RuntimeOptions{}, err
	}
	if err := limits.Validate(); err != nil {
		return RuntimeOptions{}, err
	}
	return RuntimeOptions{
		WorkspaceBasePath:       toolconfig.ReadWorkspaceBase(env, "RESTORE_PATH", filepath.Join(os.TempDir(), defaultWorkspaceDir)),
		StorageOperationTimeout: storageTimeout,
		ArchiveExtractionLimits: limits,
	}, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("PostgreSQL restore configuration is required")
	}
	if err := c.Connection.Validate(); err != nil {
		return err
	}
	database, err := c.Connection.DatabaseName()
	if err != nil {
		return err
	}
	if database == "" {
		return errors.New("existing PostgreSQL target database is required through --database or the URI path")
	}
	if c.StorageOperationTimeout < 0 {
		return errors.New("STORAGE_OPERATION_TIMEOUT must be greater than zero")
	}
	if c.ArchiveExtractionLimits != (utils.ArchiveExtractionLimits{}) {
		if err := c.ArchiveExtractionLimits.Validate(); err != nil {
			return err
		}
	}
	return c.StorageOptions.Validate()
}

func (c *Config) GetStorages(ctx context.Context) ([]storage.RestoreBackend, error) {
	options := c.StorageOptions
	options.BackupPrefix = storage.NormalizeBackupPrefixWithDefault(options.BackupPrefix, storage.PostgreSQLDefaultBackupPrefix)
	return options.GetRestoreStorages(ctx, 0)
}

func (c *Config) GetArchiveExtractionLimits() utils.ArchiveExtractionLimits {
	if c.ArchiveExtractionLimits == (utils.ArchiveExtractionLimits{}) {
		return utils.DefaultArchiveExtractionLimits()
	}
	return c.ArchiveExtractionLimits
}

func (c *Config) HasKeep() bool { return c.Keep }

func FlagDocumentation() toolconfig.CommandDoc {
	flags := []toolconfig.FlagDoc{
		flagDefs.host.Doc(envPrefix), flagDefs.port.Doc(envPrefix), flagDefs.user.Doc(envPrefix),
		flagDefs.database.Doc(envPrefix), flagDefs.sslMode.Doc(envPrefix), flagDefs.uri.Doc(envPrefix), flagDefs.password.Doc(envPrefix),
	}
	flags = append(flags, toolconfig.StorageFlagDocs(envPrefix)...)
	flags = append(flags, toolconfig.RestoreStorageBackendFlagDoc(envPrefix), flagDefs.objectName.Doc(envPrefix), flagDefs.keep.Doc(envPrefix), flagDefs.version.Doc(envPrefix))
	limits := utils.DefaultArchiveExtractionLimits()
	return toolconfig.CommandDoc{
		Name:  "postgres-unarchive",
		Flags: flags,
		EnvVars: []toolconfig.EnvDoc{
			{EnvVar: envPrefix + "RESTORE_PATH", Description: "Base directory for private per-run restore workspaces"},
			{EnvVar: envPrefix + "ARCHIVE_MAX_ENTRIES", DefaultValue: strconv.Itoa(limits.MaxEntries), Description: "Maximum number of archive entries"},
			{EnvVar: envPrefix + "ARCHIVE_MAX_ENTRY_BYTES", DefaultValue: strconv.FormatInt(limits.MaxEntryBytes, 10), Description: "Maximum bytes in one archive entry"},
			{EnvVar: envPrefix + "ARCHIVE_MAX_TOTAL_BYTES", DefaultValue: strconv.FormatInt(limits.MaxTotalBytes, 10), Description: "Maximum total extracted bytes"},
			{EnvVar: envPrefix + "STORAGE_OPERATION_TIMEOUT", Description: "Optional timeout for storage lookup and download operations"},
		},
	}
}
