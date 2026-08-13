package mongounarchive

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	envPrefix                   = "MONGOUNARCHIVE__"
	fallbackEnvPrefix           = "MONGO__"
	defaultUpdateMaxBytes int64 = 1 << 20
)

type Config struct {
	toolconfig.MongoOptions
	toolconfig.StorageOptions
	RestoreNamespaceOptions
	RestoreExecutionOptions
	RestoreSourceOptions
	UpdateOptions
	Keep bool
}

type RestoreNamespaceOptions struct {
	NSExclude string
	NSInclude string
	NSFrom    string
	NSTo      string
}

type RestoreExecutionOptions struct {
	Drop                             bool
	DryRun                           bool
	WriteConcern                     string
	NoIndexRestore                   bool
	NoOptionsRestore                 bool
	KeepIndexVersion                 bool
	MaintainInsertionOrder           bool
	NumParallelCollections           string
	NumInsertionWorkersPerCollection string
	StopOnError                      bool
	BypassDocumentValidation         bool
	PreserveUUID                     bool
}

type RestoreSourceOptions struct {
	ObjectName string
	Dir        string
}

type UpdateOptions struct {
	Updates     string
	UpdatesFile string
}

var restoreFlagDefs = struct {
	nsExclude                        toolconfig.StringFlagDef
	nsInclude                        toolconfig.StringFlagDef
	nsFrom                           toolconfig.StringFlagDef
	nsTo                             toolconfig.StringFlagDef
	drop                             toolconfig.BoolFlagDef
	dryRun                           toolconfig.BoolFlagDef
	writeConcern                     toolconfig.StringFlagDef
	noIndexRestore                   toolconfig.BoolFlagDef
	noOptionsRestore                 toolconfig.BoolFlagDef
	keepIndexVersion                 toolconfig.BoolFlagDef
	maintainInsertionOrder           toolconfig.BoolFlagDef
	numParallelCollections           toolconfig.StringFlagDef
	numInsertionWorkersPerCollection toolconfig.StringFlagDef
	stopOnError                      toolconfig.BoolFlagDef
	bypassDocumentValidation         toolconfig.BoolFlagDef
	preserveUUID                     toolconfig.BoolFlagDef
	objectName                       toolconfig.StringFlagDef
	dir                              toolconfig.StringFlagDef
	updates                          toolconfig.StringFlagDef
	updatesFile                      toolconfig.StringFlagDef
	keep                             toolconfig.BoolFlagDef
	version                          toolconfig.BoolFlagDef
}{
	nsExclude:                        toolconfig.StringFlagDef{Name: "ns-exclude", EnvKey: "NS_EXCLUDE", Usage: "exclude matching namespaces"},
	nsInclude:                        toolconfig.StringFlagDef{Name: "ns-include", EnvKey: "NS_INCLUDE", Usage: "include matching namespaces"},
	nsFrom:                           toolconfig.StringFlagDef{Name: "ns-from", EnvKey: "NS_FROM", Usage: "rename matching namespaces, must have matching nsTo"},
	nsTo:                             toolconfig.StringFlagDef{Name: "ns-to", EnvKey: "NS_TO", Usage: "rename matched namespaces, must have matching nsFrom"},
	drop:                             toolconfig.BoolFlagDef{Name: "drop", EnvKey: "DROP", Usage: "drop each collection before import"},
	dryRun:                           toolconfig.BoolFlagDef{Name: "dry-run", EnvKey: "DRY_RUN", Usage: "view summary without importing anything; cannot be combined with updates"},
	writeConcern:                     toolconfig.StringFlagDef{Name: "write-concern", EnvKey: "WRITE_CONCERN", Usage: "write concern options"},
	noIndexRestore:                   toolconfig.BoolFlagDef{Name: "no-index-restore", EnvKey: "NO_INDEX_RESTORE", Usage: "don't restore indexes"},
	noOptionsRestore:                 toolconfig.BoolFlagDef{Name: "no-options-restore", EnvKey: "NO_OPTIONS_RESTORE", Usage: "don't restore collection options"},
	keepIndexVersion:                 toolconfig.BoolFlagDef{Name: "keep-index-version", EnvKey: "KEEP_INDEX_VERSION", Usage: "don't update index version"},
	maintainInsertionOrder:           toolconfig.BoolFlagDef{Name: "maintain-insertion-order", EnvKey: "MAINTAIN_INSERTION_ORDER", Usage: "restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1"},
	numParallelCollections:           toolconfig.StringFlagDef{Name: "num-parallel-collections", EnvKey: "NUM_PARALLEL_COLLECTIONS", Usage: "number of collections to restore in parallel (default: 4)"},
	numInsertionWorkersPerCollection: toolconfig.StringFlagDef{Name: "num-insertion-workers-per-collection", EnvKey: "NUM_INSERTION_WORKERS_PER_COLLECTION", Usage: "number of insert operations to run concurrently per collection (default: 1)"},
	stopOnError:                      toolconfig.BoolFlagDef{Name: "stop-on-error", EnvKey: "STOP_ON_ERROR", Usage: "halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error"},
	bypassDocumentValidation:         toolconfig.BoolFlagDef{Name: "bypass-document-validation", EnvKey: "BYPASS_DOCUMENT_VALIDATION", Usage: "bypass document validation"},
	preserveUUID:                     toolconfig.BoolFlagDef{Name: "preserve-uuid", EnvKey: "PRESERVE_UUID", Usage: "preserve original collection UUIDs (off by default, requires drop)"},
	objectName:                       toolconfig.StringFlagDef{Name: "object-name", EnvKey: "OBJECT_NAME", Usage: "Object name of the archived file in the storage (optional)"},
	dir:                              toolconfig.StringFlagDef{Name: "dir", EnvKey: "DIR", Usage: "directory name that contains the dumped files"},
	updates:                          toolconfig.StringFlagDef{Name: "updates", EnvKey: "UPDATES", Usage: "array of update specifications in JSON string"},
	updatesFile:                      toolconfig.StringFlagDef{Name: "updates-file", EnvKey: "UPDATES_FILE", Usage: "path to a file containing an array of update specifications"},
	keep:                             toolconfig.BoolFlagDef{Name: "keep", EnvKey: "KEEP", Usage: "keep data dump"},
	version:                          toolconfig.BoolFlagDef{Name: "version", Usage: "Show the version"},
}

func ParseFlags() (*Config, bool, error) {
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")
	return parseFlags(flag.CommandLine, env, os.Args[1:])
}

func parseFlags(flagSet *flag.FlagSet, env toolconfig.EnvReader, args []string) (*Config, bool, error) {
	cfg := &Config{}

	mongoBindings := toolconfig.BindMongoFlags(flagSet, env)
	nsExclude := restoreFlagDefs.nsExclude.Bind(flagSet, env)
	nsInclude := restoreFlagDefs.nsInclude.Bind(flagSet, env)
	nsFrom := restoreFlagDefs.nsFrom.Bind(flagSet, env)
	nsTo := restoreFlagDefs.nsTo.Bind(flagSet, env)
	drop := restoreFlagDefs.drop.Bind(flagSet, env)
	dryRun := restoreFlagDefs.dryRun.Bind(flagSet, env)
	writeConcern := restoreFlagDefs.writeConcern.Bind(flagSet, env)
	noIndexRestore := restoreFlagDefs.noIndexRestore.Bind(flagSet, env)
	noOptionsRestore := restoreFlagDefs.noOptionsRestore.Bind(flagSet, env)
	keepIndexVersion := restoreFlagDefs.keepIndexVersion.Bind(flagSet, env)
	maintainInsertionOrder := restoreFlagDefs.maintainInsertionOrder.Bind(flagSet, env)
	numParallelCollections := restoreFlagDefs.numParallelCollections.Bind(flagSet, env)
	numInsertionWorkersPerCollection := restoreFlagDefs.numInsertionWorkersPerCollection.Bind(flagSet, env)
	stopOnError := restoreFlagDefs.stopOnError.Bind(flagSet, env)
	bypassDocumentValidation := restoreFlagDefs.bypassDocumentValidation.Bind(flagSet, env)
	preserveUUID := restoreFlagDefs.preserveUUID.Bind(flagSet, env)
	storageBindings := toolconfig.BindStorageFlags(flagSet, env)
	objectName := restoreFlagDefs.objectName.Bind(flagSet, env)
	dir := restoreFlagDefs.dir.Bind(flagSet, env)
	updates := restoreFlagDefs.updates.Bind(flagSet, env)
	updatesFile := restoreFlagDefs.updatesFile.Bind(flagSet, env)
	keep := restoreFlagDefs.keep.Bind(flagSet, env)
	showVersion := restoreFlagDefs.version.Bind(flagSet, env)

	if err := flagSet.Parse(args); err != nil {
		return nil, false, err
	}

	mongoBindings.Apply(&cfg.MongoOptions)
	cfg.RestoreNamespaceOptions = RestoreNamespaceOptions{
		NSExclude: *nsExclude,
		NSInclude: *nsInclude,
		NSFrom:    *nsFrom,
		NSTo:      *nsTo,
	}
	cfg.RestoreExecutionOptions = RestoreExecutionOptions{
		Drop:                             *drop,
		DryRun:                           *dryRun,
		WriteConcern:                     *writeConcern,
		NoIndexRestore:                   *noIndexRestore,
		NoOptionsRestore:                 *noOptionsRestore,
		KeepIndexVersion:                 *keepIndexVersion,
		MaintainInsertionOrder:           *maintainInsertionOrder,
		NumParallelCollections:           *numParallelCollections,
		NumInsertionWorkersPerCollection: *numInsertionWorkersPerCollection,
		StopOnError:                      *stopOnError,
		BypassDocumentValidation:         *bypassDocumentValidation,
		PreserveUUID:                     *preserveUUID,
	}
	storageBindings.Apply(&cfg.StorageOptions)
	cfg.RestoreSourceOptions = RestoreSourceOptions{ObjectName: *objectName, Dir: *dir}
	cfg.UpdateOptions = UpdateOptions{Updates: *updates, UpdatesFile: *updatesFile}
	cfg.Keep = *keep

	if showVersion != nil && *showVersion {
		return cfg, true, nil
	}

	if err := cfg.Validate(); err != nil {
		return nil, false, err
	}

	return cfg, false, nil
}

func (c *Config) GetMongounarchiveOptions(destPath string) []string {
	options := []string{"--gzip"}

	tdir := ""
	if c.Dir != "" {
		tdir = c.Dir
	} else if c.DB != "" {
		tdir = c.DB
	}

	options = append(options, "--dir="+path.Join(destPath, tdir))
	options = c.MongoOptions.AppendToolOptions(options)
	if c.NSExclude != "" {
		options = append(options, "--nsExclude="+c.NSExclude)
	}
	if c.NSInclude != "" {
		options = append(options, "--nsInclude="+c.NSInclude)
	}
	if c.NSFrom != "" {
		options = append(options, "--nsFrom="+c.NSFrom)
	}
	if c.NSTo != "" {
		options = append(options, "--nsTo="+c.NSTo)
	}
	if c.Drop {
		options = append(options, "--drop")
	}
	if c.DryRun {
		options = append(options, "--dryRun")
	}
	if c.WriteConcern != "" {
		options = append(options, "--writeConcern="+c.WriteConcern)
	}
	if c.NoIndexRestore {
		options = append(options, "--noIndexRestore")
	}
	if c.NoOptionsRestore {
		options = append(options, "--noOptionsRestore")
	}
	if c.KeepIndexVersion {
		options = append(options, "--keepIndexVersion")
	}
	if c.MaintainInsertionOrder {
		options = append(options, "--maintainInsertionOrder")
	}
	if c.NumParallelCollections != "" {
		options = append(options, "--numParallelCollections="+c.NumParallelCollections)
	}
	if c.NumInsertionWorkersPerCollection != "" {
		options = append(options, "--numInsertionWorkersPerCollection="+c.NumInsertionWorkersPerCollection)
	}
	if c.StopOnError {
		options = append(options, "--stopOnError")
	}
	if c.BypassDocumentValidation {
		options = append(options, "--bypassDocumentValidation")
	}
	if c.PreserveUUID {
		options = append(options, "--preserveUUID")
	}

	return options
}

func (c *Config) GetStorages(ctx context.Context) ([]storage.Storage, error) {
	return c.StorageOptions.GetStorages(ctx, 0)
}

func (c *Config) GetObjectName() string {
	return c.ObjectName
}

func (c *Config) GetMongoClient(ctx context.Context) (*mongo.Client, *mongo.Database, error) {
	clientOptions, err := c.MongoOptions.MongoClientOptions()
	if err != nil {
		return nil, nil, err
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}

	return client, client.Database(c.DB), nil
}

func (c *Config) GetUpdates() ([]byte, error) {
	maxBytes, err := readPositiveInt64Env(envPrefix+"UPDATE_MAX_BYTES", defaultUpdateMaxBytes)
	if err != nil {
		return nil, err
	}

	if c.Updates != "" {
		if int64(len(c.Updates)) > maxBytes {
			return nil, fmt.Errorf("updates exceeds maximum size of %d bytes", maxBytes)
		}
		return []byte(c.Updates), nil
	}
	if c.UpdatesFile != "" {
		return readLimitedFile(c.UpdatesFile, maxBytes)
	}
	return []byte(""), nil
}

func (c *Config) HasUpdates() bool {
	return c.Updates != "" || c.UpdatesFile != ""
}

func (c *Config) HasKeep() bool {
	return c.Keep
}

func (c *Config) Validate() error {
	if c.DryRun && c.HasUpdates() {
		return errors.New("--dry-run cannot be combined with --updates or --updates-file")
	}
	if !c.HasUpdates() {
		return nil
	}
	_, err := c.GetUpdates()
	return err
}

func FlagDocumentation() toolconfig.CommandDoc {
	flags := append([]toolconfig.FlagDoc{}, toolconfig.MongoFlagDocs(envPrefix)...)
	flags = append(flags,
		restoreFlagDefs.nsExclude.Doc(envPrefix),
		restoreFlagDefs.nsInclude.Doc(envPrefix),
		restoreFlagDefs.nsFrom.Doc(envPrefix),
		restoreFlagDefs.nsTo.Doc(envPrefix),
		restoreFlagDefs.drop.Doc(envPrefix),
		restoreFlagDefs.dryRun.Doc(envPrefix),
		restoreFlagDefs.writeConcern.Doc(envPrefix),
		restoreFlagDefs.noIndexRestore.Doc(envPrefix),
		restoreFlagDefs.noOptionsRestore.Doc(envPrefix),
		restoreFlagDefs.keepIndexVersion.Doc(envPrefix),
		restoreFlagDefs.maintainInsertionOrder.Doc(envPrefix),
		restoreFlagDefs.numParallelCollections.Doc(envPrefix),
		restoreFlagDefs.numInsertionWorkersPerCollection.Doc(envPrefix),
		restoreFlagDefs.stopOnError.Doc(envPrefix),
		restoreFlagDefs.bypassDocumentValidation.Doc(envPrefix),
		restoreFlagDefs.preserveUUID.Doc(envPrefix),
	)
	flags = append(flags, toolconfig.StorageFlagDocs(envPrefix)...)
	flags = append(flags,
		restoreFlagDefs.objectName.Doc(envPrefix),
		restoreFlagDefs.dir.Doc(envPrefix),
		restoreFlagDefs.updates.Doc(envPrefix),
		restoreFlagDefs.updatesFile.Doc(envPrefix),
		restoreFlagDefs.keep.Doc(envPrefix),
		restoreFlagDefs.version.Doc(envPrefix),
	)

	return toolconfig.CommandDoc{
		Name:  "mongo-unarchive",
		Flags: flags,
		EnvVars: []toolconfig.EnvDoc{
			{EnvVar: envPrefix + "RESTORE_PATH", Description: "Base directory for per-run restore workspaces before extraction"},
			{EnvVar: envPrefix + "UPDATE_MAX_BYTES", DefaultValue: strconv.FormatInt(defaultUpdateMaxBytes, 10), Description: "Maximum size in bytes allowed for inline or file-based update specifications"},
			{EnvVar: envPrefix + "STORAGE_OPERATION_TIMEOUT", Description: "Optional timeout applied to storage lookup and download operations"},
			{EnvVar: envPrefix + "UPDATE_TIMEOUT", Description: "Optional timeout applied to MongoDB update connections and update operations"},
		},
	}
}

func readPositiveInt64Env(key string, fallback int64) (int64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return value, nil
}

func readLimitedFile(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("updates file %q exceeds maximum size of %d bytes", path, maxBytes)
	}

	return data, nil
}
