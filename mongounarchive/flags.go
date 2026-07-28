package mongounarchive

import (
	"flag"
	"os"
	"path"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	envPrefix         = "MONGOUNARCHIVE__"
	fallbackEnvPrefix = "MONGO__"
)

type Config struct {
	toolconfig.MongoOptions
	toolconfig.StorageOptions
	NSExclude                        string
	NSInclude                        string
	NSFrom                           string
	NSTo                             string
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
	ObjectName                       string
	Dir                              string
	Updates                          string
	UpdatesFile                      string
	Keep                             bool
}

func ParseFlags() (*Config, bool) {
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")
	cfg := &Config{}

	mongoBindings := toolconfig.BindMongoFlags(env)
	nsExclude := flag.String("ns-exclude", env.GetValue("NS_EXCLUDE"), "exclude matching namespaces")
	nsInclude := flag.String("ns-include", env.GetValue("NS_INCLUDE"), "include matching namespaces")
	nsFrom := flag.String("ns-from", env.GetValue("NS_FROM"), "rename matching namespaces, must have matching nsTo")
	nsTo := flag.String("ns-to", env.GetValue("NS_TO"), "rename matched namespaces, must have matching nsFrom")
	drop := flag.Bool("drop", env.GetValue("DROP") == "true", "drop each collection before import")
	dryRun := flag.Bool("dry-run", env.GetValue("DRY_RUN") == "true", "view summary without importing anything. recommended with verbosity")
	writeConcern := flag.String("write-concern", env.GetValue("WRITE_CONCERN"), "write concern options")
	noIndexRestore := flag.Bool("no-index-restore", env.GetValue("NO_INDEX_RESTORE") == "true", "don't restore indexes")
	noOptionsRestore := flag.Bool("no-options-restore", env.GetValue("NO_OPTIONS_RESTORE") == "true", "don't restore collection options")
	keepIndexVersion := flag.Bool("keep-index-version", env.GetValue("KEEP_INDEX_VERSION") == "true", "don't update index version")
	maintainInsertionOrder := flag.Bool("maintain-insertion-order", env.GetValue("MAINTAIN_INSERTION_ORDER") == "true", "restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1")
	numParallelCollections := flag.String("num-parallel-collections", env.GetValue("NUM_PARALLEL_COLLECTIONS"), "number of collections to restore in parallel (default: 4)")
	numInsertionWorkersPerCollection := flag.String("num-insertion-workers-per-collection", env.GetValue("NUM_INSERTION_WORKERS_PER_COLLECTION"), "number of insert operations to run concurrently per collection (default: 1)")
	stopOnError := flag.Bool("stop-on-error", env.GetValue("STOP_ON_ERROR") == "true", "halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error")
	bypassDocumentValidation := flag.Bool("bypass-document-validation", env.GetValue("BYPASS_DOCUMENT_VALIDATION") == "true", "bypass document validation")
	preserveUUID := flag.Bool("preserve-uuid", env.GetValue("PRESERVE_UUID") == "true", "preserve original collection UUIDs (off by default, requires drop)")
	storageBindings := toolconfig.BindStorageFlags(env)
	objectName := flag.String("object-name", env.GetValue("OBJECT_NAME"), "Object name of the archived file in the storage (optional)")
	dir := flag.String("dir", env.GetValue("DIR"), "directory name that contains the dumped files")
	updates := flag.String("updates", env.GetValue("UPDATES"), "array of update specifications in JSON string")
	updatesFile := flag.String("updates-file", env.GetValue("UPDATES_FILE"), "path to a file containing an array of update specifications")
	keep := flag.Bool("keep", env.GetValue("KEEP") == "true", "keep data dump")
	showVersion := flag.Bool("version", false, "Show the version")

	flag.Parse()

	mongoBindings.Apply(&cfg.MongoOptions)
	cfg.NSExclude = *nsExclude
	cfg.NSInclude = *nsInclude
	cfg.NSFrom = *nsFrom
	cfg.NSTo = *nsTo
	cfg.Drop = *drop
	cfg.DryRun = *dryRun
	cfg.WriteConcern = *writeConcern
	cfg.NoIndexRestore = *noIndexRestore
	cfg.NoOptionsRestore = *noOptionsRestore
	cfg.KeepIndexVersion = *keepIndexVersion
	cfg.MaintainInsertionOrder = *maintainInsertionOrder
	cfg.NumParallelCollections = *numParallelCollections
	cfg.NumInsertionWorkersPerCollection = *numInsertionWorkersPerCollection
	cfg.StopOnError = *stopOnError
	cfg.BypassDocumentValidation = *bypassDocumentValidation
	cfg.PreserveUUID = *preserveUUID
	storageBindings.Apply(&cfg.StorageOptions)
	cfg.ObjectName = *objectName
	cfg.Dir = *dir
	cfg.Updates = *updates
	cfg.UpdatesFile = *updatesFile
	cfg.Keep = *keep

	if showVersion != nil && *showVersion {
		return cfg, true
	}

	return cfg, false
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

func (c *Config) GetStorages() []storage.Storage {
	return c.StorageOptions.GetStorages(0)
}

func (c *Config) GetObjectName() string {
	return c.ObjectName
}

func (c *Config) GetMongoClient() (*mongo.Client, *mongo.Database, error) {
	uri, err := c.MongoOptions.MongoConnectionURI()
	if err != nil {
		return nil, nil, err
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}

	return client, client.Database(c.DB), nil
}

func (c *Config) GetUpdates() ([]byte, error) {
	if c.Updates != "" {
		return []byte(c.Updates), nil
	}
	if c.UpdatesFile != "" {
		return os.ReadFile(c.UpdatesFile)
	}
	return []byte(""), nil
}

func (c *Config) HasUpdates() bool {
	return c.Updates != "" || c.UpdatesFile != ""
}

func (c *Config) HasKeep() bool {
	return c.Keep
}
