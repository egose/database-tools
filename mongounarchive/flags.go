package mongounarchive

import (
	"context"
	"errors"
	"flag"
	"os"
	"path"

	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	envPrefix         = "MONGOUNARCHIVE__"
	fallbackEnvPrefix = "MONGO__"
)

var (
	verbose  *string
	quietPtr *bool

	hostPtr *string
	portPtr *string

	sslPtr                         *bool
	sslCAFilePtr                   *string
	sslPEMKeyFilePtr               *string
	sslPEMKeyPasswordPtr           *string
	sslCRLFilePtr                  *string
	sslAllowInvalidCertificatesPtr *bool
	sslAllowInvalidHostnamesPtr    *bool
	sslFIPSModePtr                 *bool

	usernamePtr                *string
	passwordPtr                *string
	authenticationDatabasePtr  *string
	authenticationMechanismPtr *string

	gssapiServiceNamePtr *string
	gssapiHostNamePtr    *string

	uriPtr      *string
	uriPrunePtr *bool

	dbPtr         *string
	collectionPtr *string
	nsExcludePtr  *string
	nsIncludePtr  *string
	nsFromPtr     *string
	nsToPtr       *string

	dropPtr                             *bool
	dryRunPtr                           *bool
	writeConcernPtr                     *string
	noIndexRestorePtr                   *bool
	noOptionsRestorePtr                 *bool
	keepIndexVersionPtr                 *bool
	maintainInsertionOrderPtr           *bool
	numParallelCollectionsPtr           *string
	numInsertionWorkersPerCollectionPtr *string
	stopOnErrorPtr                      *bool
	bypassDocumentValidationPtr         *bool
	preserveUUIDPtr                     *bool

	azEndpointPtr      *string
	azAccountNamePtr   *string
	azAccountKeyPtr    *string
	azContainerNamePtr *string

	awsEndpointPtr         *string
	awsAccessKeyIdPtr      *string
	awsSecretAccessKeyPtr  *string
	awsRegionPtr           *string
	awsBucketPtr           *string
	awsS3ForcePathStylePtr *bool

	gcpEndpointPtr     *string
	gcpBucketPtr       *string
	gcpCredsFilePtr    *string
	gcpProjectIDPtr    *string
	gcpPrivateKeyIdPtr *string
	gcpPrivateKeyPtr   *string
	gcpClientEmailPtr  *string
	gcpClientIDPtr     *string

	localPathPtr *string

	objectNamePtr *string
	dirPtr        *string

	updatesPtr     *string
	updatesFilePtr *string

	keepPtr *bool
)

func ParseFlags() {
	env := utils.NewEnv(envPrefix, fallbackEnvPrefix, "")

	// verbosity options:
	verbose = flag.String("verbose", env.GetValue("VERBOSE"), "more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N)")
	quietPtr = flag.Bool("quiet", env.GetValue("QUIET") == "true", "hide all log output")

	// connection options:
	hostPtr = flag.String("host", env.GetValue("HOST"), "MongoDB host to connect to (setname/host1,host2 for replica sets)")
	portPtr = flag.String("port", env.GetValue("PORT"), "MongoDB port (can also use --host hostname:port)")

	// ssl options:
	sslPtr = flag.Bool("ssl", env.GetValue("SSL") == "true", "connect to a mongod or mongos that has ssl enabled")
	sslCAFilePtr = flag.String("ssl-ca-file", env.GetValue("SSL_CA_FILE"), "the .pem file containing the root certificate chain from the certificate authority")
	sslPEMKeyFilePtr = flag.String("ssl-pem-key-file", env.GetValue("SSL_PEM_KEY_FILE"), "the .pem file containing the certificate and key")
	sslPEMKeyPasswordPtr = flag.String("ssl-pem-key-password", env.GetValue("SSL_PEM_KEY_PASSWORD"), "the password to decrypt the sslPEMKeyFile, if necessary")
	sslCRLFilePtr = flag.String("ssl-crl-file", env.GetValue("SSL_CRL_File"), "the .pem file containing the certificate revocation list")
	sslAllowInvalidCertificatesPtr = flag.Bool("ssl-allow-invalid-certificates", env.GetValue("SSL_ALLOW_INVALID_CERTIFICATES") == "true", "bypass the validation for server certificates")
	sslAllowInvalidHostnamesPtr = flag.Bool("ssl-allow-invalid-hostnames", env.GetValue("SSL_ALLOW_INVALID_HOSTNAMES") == "true", "bypass the validation for server name")
	sslFIPSModePtr = flag.Bool("ssl-fips-mode", env.GetValue("SSL_FIPS_MODE") == "true", "use FIPS mode of the installed openssl library")

	// authentication options:
	usernamePtr = flag.String("username", env.GetValue("USERNAME"), "username for authentication")
	passwordPtr = flag.String("password", env.GetValue("PASSWORD"), "password for authentication")
	authenticationDatabasePtr = flag.String("authentication-database", env.GetValue("AUTHENTICATION_DATABASE"), "database that holds the user's credentials")
	authenticationMechanismPtr = flag.String("authentication-mechanism", env.GetValue("AUTHENTICATION_MECHANISM"), "authentication mechanism to use")

	// kerberos options:
	gssapiServiceNamePtr = flag.String("gssapi-service-name", env.GetValue("GSSAPI_SERVICE_NAME"), "service name to use when authenticating using GSSAPI/Kerberos (default: mongodb)")
	gssapiHostNamePtr = flag.String("gssapi-host-name", env.GetValue("GSSAPI_HOST_NAME"), "hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>)")

	// uri options:
	uriPtr = flag.String("uri", env.GetValue("URI"), "MongoDB uri connection string")
	uriPrunePtr = flag.Bool("uri-prune", env.GetValue("URI_PRUNE") == "true", "prune MongoDB uri connection string")

	// namespace options:
	dbPtr = flag.String("db", env.GetValue("DB"), "database to use")
	collectionPtr = flag.String("collection", env.GetValue("COLLECTION"), "collection to use")
	nsExcludePtr = flag.String("ns-exclude", env.GetValue("NS_EXCLUDE"), "exclude matching namespaces")
	nsIncludePtr = flag.String("ns-include", env.GetValue("NS_INCLUDE"), "include matching namespaces")
	nsFromPtr = flag.String("ns-from", env.GetValue("NS_FROM"), "rename matching namespaces, must have matching nsTo")
	nsToPtr = flag.String("ns-to", env.GetValue("NS_TO"), "rename matched namespaces, must have matching nsFrom")

	// restore options:
	dropPtr = flag.Bool("drop", env.GetValue("DROP") == "true", "drop each collection before import")
	dryRunPtr = flag.Bool("dry-run", env.GetValue("DRY_RUN") == "true", "view summary without importing anything. recommended with verbosity")
	writeConcernPtr = flag.String("write-concern", env.GetValue("WRITE_CONCERN"), "write concern options")
	noIndexRestorePtr = flag.Bool("no-index-restore", env.GetValue("NO_INDEX_RESTORE") == "true", "don't restore indexes")
	noOptionsRestorePtr = flag.Bool("no-options-restore", env.GetValue("NO_OPTIONS_RESTORE") == "true", "don't restore collection options")
	keepIndexVersionPtr = flag.Bool("keep-index-version", env.GetValue("KEEP_INDEX_VERSION") == "true", "don't update index version")
	maintainInsertionOrderPtr = flag.Bool("maintain-insertion-order", env.GetValue("MAINTAIN_INSERTION_ORDER") == "true", "restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1")
	numParallelCollectionsPtr = flag.String("num-parallel-collections", env.GetValue("NUM_PARALLEL_COLLECTIONS"), "number of collections to restore in parallel (default: 4)")
	numInsertionWorkersPerCollectionPtr = flag.String("num-insertion-workers-per-collection", env.GetValue("NUM_INSERTION_WORKERS_PER_COLLECTION"), "number of insert operations to run concurrently per collection (default: 1)")
	stopOnErrorPtr = flag.Bool("stop-on-error", env.GetValue("STOP_ON_ERROR") == "true", "halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error")
	bypassDocumentValidationPtr = flag.Bool("bypass-document-validation", env.GetValue("BYPASS_DOCUMENT_VALIDATION") == "true", "bypass document validation")
	preserveUUIDPtr = flag.Bool("preserve-uuid", env.GetValue("PRESERVE_UUID") == "true", "preserve original collection UUIDs (off by default, requires drop)")

	azEndpointPtr = flag.String("az-endpoint", env.GetValue("AZ_ENDPOINT", ""), "specify the emulator hostname and Azure Blob Storage port")
	azAccountNamePtr = flag.String("az-account-name", env.GetValue("AZ_ACCOUNT_NAME"), "Azure Blob Storage Account Name")
	azAccountKeyPtr = flag.String("az-account-key", env.GetValue("AZ_ACCOUNT_KEY"), "Azure Blob Storage Account Key")
	azContainerNamePtr = flag.String("az-container-name", env.GetValue("AZ_CONTAINER_NAME"), "Azure Blob Storage Container Name")

	awsEndpointPtr = flag.String("aws-endpoint", env.GetValue("AWS_ENDPOINT", ""), "AWS endpoint URL (hostname only or fully qualified URI)")
	awsAccessKeyIdPtr = flag.String("aws-access-key-id", env.GetValue("AWS_ACCESS_KEY_ID"), "AWS access key associated with an IAM account")
	awsSecretAccessKeyPtr = flag.String("aws-secret-access-key", env.GetValue("AWS_SECRET_ACCESS_KEY"), "AWS secret key associated with the access key")
	awsRegionPtr = flag.String("aws-region", env.GetValue("AWS_REGION", "us-east-1"), "AWS Region whose servers you want to send your requests to")
	awsBucketPtr = flag.String("aws-bucket", env.GetValue("AWS_BUCKET"), "AWS S3 bucket name")
	awsS3ForcePathStylePtr = flag.Bool("aws-s3-force-path-style", env.GetValue("AWS_S3_FORCE_PATH_STYLE") == "true", "force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`)")

	gcpEndpointPtr = flag.String("gcp-endpoint", env.GetValue("GCP_ENDPOINT", ""), "GCP endpoint URL")
	gcpBucketPtr = flag.String("gcp-bucket", env.GetValue("GCP_BUCKET"), "GCP storage bucket name")
	gcpCredsFilePtr = flag.String("gcp-creds-file", env.GetValue("GCP_CREDS_FILE"), "GCP service account's credentials file")
	gcpProjectIDPtr = flag.String("gcp-project-id", env.GetValue("GCP_PROJECT_ID"), "GCP service account's project id")
	gcpPrivateKeyIdPtr = flag.String("gcp-private-key-id", env.GetValue("GCP_PRIVATE_KEY_ID"), "GCP service account's private key id")
	gcpPrivateKeyPtr = flag.String("gcp-private-key", env.GetValue("GCP_PRIVATE_KEY"), "GCP service account's private key")
	gcpClientEmailPtr = flag.String("gcp-client-email", env.GetValue("GCP_CLIENT_EMAIL"), "GCP service account's client email")
	gcpClientIDPtr = flag.String("gcp-client-id", env.GetValue("GCP_CLIENT_ID"), "GCP service account's client id")

	localPathPtr = flag.String("local-path", env.GetValue("LOCAL_PATH"), "Local directory path to store backups")

	objectNamePtr = flag.String("object-name", env.GetValue("OBJECT_NAME"), "Object name of the archived file in the storage (optional)")
	dirPtr = flag.String("dir", env.GetValue("DIR"), "directory name that contains the dumped files")

	updatesPtr = flag.String("updates", env.GetValue("UPDATES"), "array of update specifications in JSON string")
	updatesFilePtr = flag.String("updates-file", env.GetValue("UPDATES_FILE"), "path to a file containing an array of update specifications")

	keepPtr = flag.Bool("keep", env.GetValue("KEEP") == "true", "keep data dump")

	flag.Parse()
}

func GetMongounarchiveOptions(destPath string) []string {
	options := []string{
		"--gzip",
		"--drop",
	}

	tdir := ""
	if *dirPtr != "" {
		tdir = *dirPtr
	} else if *dbPtr != "" {
		tdir = *dbPtr
	}

	options = append(options, "--dir="+path.Join(destPath, tdir))

	if *verbose != "" {
		options = append(options, "--verbose="+*verbose)
	}

	if *quietPtr {
		options = append(options, "--quiet")
	}

	if *hostPtr != "" {
		options = append(options, "--host="+*hostPtr)
	}

	if *portPtr != "" {
		options = append(options, "--port="+*portPtr)
	}

	if *sslPtr {
		options = append(options, "--ssl")
	}

	if *sslCAFilePtr != "" {
		options = append(options, "--sslCAFile="+*sslCAFilePtr)
	}

	if *sslPEMKeyFilePtr != "" {
		options = append(options, "--sslPEMKeyFile="+*sslPEMKeyFilePtr)
	}

	if *sslPEMKeyPasswordPtr != "" {
		options = append(options, "--sslPEMKeyPassword="+*sslPEMKeyPasswordPtr)
	}

	if *sslCRLFilePtr != "" {
		options = append(options, "--sslCRLFile="+*sslCRLFilePtr)
	}

	if *sslAllowInvalidCertificatesPtr {
		options = append(options, "--sslAllowInvalidCertificates")
	}

	if *sslAllowInvalidHostnamesPtr {
		options = append(options, "--sslAllowInvalidHostnames")
	}

	if *sslFIPSModePtr {
		options = append(options, "--sslFIPSMode")
	}

	if *usernamePtr != "" {
		options = append(options, "--username="+*usernamePtr)
	}

	if *passwordPtr != "" {
		options = append(options, "--password="+*passwordPtr)
	}

	if *authenticationDatabasePtr != "" {
		options = append(options, "--authenticationDatabase="+*authenticationDatabasePtr)
	}

	if *authenticationMechanismPtr != "" {
		options = append(options, "--authenticationMechanism="+*authenticationMechanismPtr)
	}

	if *gssapiServiceNamePtr != "" {
		options = append(options, "--gssapiServiceName="+*gssapiServiceNamePtr)
	}

	if *gssapiHostNamePtr != "" {
		options = append(options, "--gssapiHostName="+*gssapiHostNamePtr)
	}

	if *uriPtr != "" {
		uri := *uriPtr
		if *uriPrunePtr {
			uri = utils.PruneMongoDBURI(uri)
		}

		options = append(options, "--uri="+uri)
	}

	if *dbPtr != "" {
		options = append(options, "--db="+*dbPtr)
	}

	if *collectionPtr != "" {
		options = append(options, "--collection="+*collectionPtr)
	}

	if *nsExcludePtr != "" {
		options = append(options, "--nsExclude="+*nsExcludePtr)
	}

	if *nsIncludePtr != "" {
		options = append(options, "--nsInclude="+*nsIncludePtr)
	}

	if *nsFromPtr != "" {
		options = append(options, "--nsFrom="+*nsFromPtr)
	}

	if *nsToPtr != "" {
		options = append(options, "--nsTo="+*nsToPtr)
	}

	if *dropPtr {
		options = append(options, "--drop")
	}

	if *dryRunPtr {
		options = append(options, "--dryRun")
	}

	if *writeConcernPtr != "" {
		options = append(options, "--writeConcern="+*writeConcernPtr)
	}

	if *noIndexRestorePtr {
		options = append(options, "--noIndexRestore")
	}

	if *noOptionsRestorePtr {
		options = append(options, "--noOptionsRestore")
	}

	if *keepIndexVersionPtr {
		options = append(options, "--keepIndexVersion")
	}

	if *maintainInsertionOrderPtr {
		options = append(options, "--maintainInsertionOrder")
	}

	if *numParallelCollectionsPtr != "" {
		options = append(options, "--numParallelCollections="+*numParallelCollectionsPtr)
	}

	if *numInsertionWorkersPerCollectionPtr != "" {
		options = append(options, "--numInsertionWorkersPerCollection="+*numInsertionWorkersPerCollectionPtr)
	}

	if *stopOnErrorPtr {
		options = append(options, "--stopOnError")
	}

	if *bypassDocumentValidationPtr {
		options = append(options, "--bypassDocumentValidation")
	}

	if *preserveUUIDPtr {
		options = append(options, "--preserveUUID")
	}

	return options
}

func getAzBlob() (*storage.AzBlob, error) {
	az := new(storage.AzBlob)
	err := az.Init(*azAccountNamePtr, *azAccountKeyPtr, *azContainerNamePtr, *azEndpointPtr)
	if err != nil {
		return nil, err
	}

	return az, nil
}

func getAwsS3() (*storage.AwsS3, error) {
	s3 := new(storage.AwsS3)
	err := s3.Init(*awsEndpointPtr, *awsAccessKeyIdPtr, *awsSecretAccessKeyPtr, *awsRegionPtr, *awsBucketPtr, *awsS3ForcePathStylePtr)
	if err != nil {
		return nil, err
	}

	return s3, nil
}

func getGCP() (*storage.GcpStorage, error) {
	storage := new(storage.GcpStorage)

	err := storage.Init(*gcpEndpointPtr, *gcpBucketPtr, *gcpCredsFilePtr, *gcpProjectIDPtr, *gcpPrivateKeyIdPtr, *gcpPrivateKeyPtr, *gcpClientEmailPtr, *gcpClientIDPtr)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func getLocal() (*storage.LocalStorage, error) {
	storage := new(storage.LocalStorage)
	err := storage.Init(*localPathPtr, 0)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func GetStorage() (storage.Storage, error) {
	if useAzure() {
		mlog.Logvf(mlog.Always, "Found Storage Option: %v", "Azure")
		return getAzBlob()
	}

	if useAWS() {
		mlog.Logvf(mlog.Always, "Found Storage Option: %v", "AWS")
		return getAwsS3()
	}

	if useGCP() {
		mlog.Logvf(mlog.Always, "Found Storage Option: %v", "GCP")
		return getGCP()
	}

	if useLocal() {
		mlog.Logvf(mlog.Always, "Found Storage Option: %v", "Local")
		return getLocal()
	}

	return nil, errors.New("no storage provider detected")
}

func GetObjectName() string {
	return *objectNamePtr
}

func GetMongoClient() (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(*uriPtr))
	if err != nil {
		return nil, nil, err
	}

	dbClient := client.Database(*dbPtr)
	return client, dbClient, nil
}

func GetUpdates() ([]byte, error) {
	if *updatesPtr != "" {
		return []byte(*updatesPtr), nil
	} else if *updatesFilePtr != "" {
		content, err := os.ReadFile(*updatesFilePtr)
		return content, err
	}

	return []byte(""), nil
}

func HasUpdates() bool {
	return *updatesPtr != "" || *updatesFilePtr != ""
}

func HasKeep() bool {
	return *keepPtr
}

func useAzure() bool {
	return *azAccountNamePtr != "" && *azAccountKeyPtr != "" && *azContainerNamePtr != ""
}

func useAWS() bool {
	return *awsAccessKeyIdPtr != "" && *awsSecretAccessKeyPtr != "" && *awsBucketPtr != ""
}

func useGCP() bool {
	return *gcpBucketPtr != ""
}

func useLocal() bool {
	return *localPathPtr != ""
}
