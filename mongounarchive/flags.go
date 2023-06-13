package mongounarchive

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path"

	"github.com/junminahn/database-tools/storage"
	"github.com/junminahn/database-tools/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	envPrefix = "MONGOUNARCHIVE__"
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

	azAccountNamePtr   *string
	azAccountKeyPtr    *string
	azContainerNamePtr *string

	awsAccessKeyIdPtr     *string
	awsSecretAccessKeyPtr *string
	awsRegionPtr          *string
	awsBucketPtr          *string

	gcpBucketPtr       *string
	gcpCredsFilePtr    *string
	gcpProjectIDPtr    *string
	gcpPrivateKeyIdPtr *string
	gcpPrivateKeyPtr   *string
	gcpClientEmailPtr  *string
	gcpClientIDPtr     *string

	objectNamePtr *string
	dirPtr        *string

	updatesPtr     *string
	updatesFilePtr *string

	keepPtr *bool
)

func ParseFlags() {
	// verbosity options:
	verbose = flag.String("verbose", os.Getenv(envPrefix+"VERBOSE"), "more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N)")
	quietPtr = flag.Bool("quiet", os.Getenv(envPrefix+"QUIET") == "true", "hide all log output")

	// connection options:
	hostPtr = flag.String("host", os.Getenv(envPrefix+"HOST"), "MongoDB host to connect to (setname/host1,host2 for replica sets)")
	portPtr = flag.String("port", os.Getenv(envPrefix+"PORT"), "MongoDB port (can also use --host hostname:port)")

	// ssl options:
	sslPtr = flag.Bool("ssl", os.Getenv(envPrefix+"SSL") == "true", "connect to a mongod or mongos that has ssl enabled")
	sslCAFilePtr = flag.String("ssl-ca-file", os.Getenv(envPrefix+"SSL_CA_FILE"), "the .pem file containing the root certificate chain from the certificate authority")
	sslPEMKeyFilePtr = flag.String("ssl-pem-key-file", os.Getenv(envPrefix+"SSL_PEM_KEY_FILE"), "the .pem file containing the certificate and key")
	sslPEMKeyPasswordPtr = flag.String("ssl-pem-key-password", os.Getenv(envPrefix+"SSL_PEM_KEY_PASSWORD"), "the password to decrypt the sslPEMKeyFile, if necessary")
	sslCRLFilePtr = flag.String("ssl-crl-file", os.Getenv(envPrefix+"SSL_CRL_File"), "the .pem file containing the certificate revocation list")
	sslAllowInvalidCertificatesPtr = flag.Bool("ssl-allow-invalid-certificates", os.Getenv(envPrefix+"SSL_ALLOW_INVALID_CERTIFICATES") == "true", "bypass the validation for server certificates")
	sslAllowInvalidHostnamesPtr = flag.Bool("ssl-allow-invalid-hostnames", os.Getenv(envPrefix+"SSL_ALLOW_INVALID_HOSTNAMES") == "true", "bypass the validation for server name")
	sslFIPSModePtr = flag.Bool("ssl-fips-mode", os.Getenv(envPrefix+"SSL_FIPS_MODE") == "true", "use FIPS mode of the installed openssl library")

	// authentication options:
	usernamePtr = flag.String("username", os.Getenv(envPrefix+"USERNAME"), "username for authentication")
	passwordPtr = flag.String("password", os.Getenv(envPrefix+"PASSWORD"), "password for authentication")
	authenticationDatabasePtr = flag.String("authentication-database", os.Getenv(envPrefix+"AUTHENTICATION_DATABASE"), "database that holds the user's credentials")
	authenticationMechanismPtr = flag.String("authentication-mechanism", os.Getenv(envPrefix+"AUTHENTICATION_MECHANISM"), "authentication mechanism to use")

	// kerberos options:
	gssapiServiceNamePtr = flag.String("gssapi-service-name", os.Getenv(envPrefix+"GSSAPI_SERVICE_NAME"), "service name to use when authenticating using GSSAPI/Kerberos (default: mongodb)")
	gssapiHostNamePtr = flag.String("gssapi-host-name", os.Getenv(envPrefix+"GSSAPI_HOST_NAME"), "hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>)")

	// uri options:
	uriPtr = flag.String("uri", os.Getenv(envPrefix+"URI"), "MongoDB uri connection string")
	uriPrunePtr = flag.Bool("uri-prune", os.Getenv(envPrefix+"URI_PRUNE") == "true", "prune MongoDB uri connection string")

	// namespace options:
	dbPtr = flag.String("db", os.Getenv(envPrefix+"DB"), "database to use")
	collectionPtr = flag.String("collection", os.Getenv(envPrefix+"COLLECTION"), "collection to use")
	nsExcludePtr = flag.String("ns-exclude", os.Getenv(envPrefix+"NS_EXCLUDE"), "exclude matching namespaces")
	nsIncludePtr = flag.String("ns-include", os.Getenv(envPrefix+"NS_INCLUDE"), "include matching namespaces")
	nsFromPtr = flag.String("ns-from", os.Getenv(envPrefix+"NS_FROM"), "rename matching namespaces, must have matching nsTo")
	nsToPtr = flag.String("ns-to", os.Getenv(envPrefix+"NS_TO"), "rename matched namespaces, must have matching nsFrom")

	// restore options:
	dropPtr = flag.Bool("drop", os.Getenv(envPrefix+"DROP") == "true", "drop each collection before import")
	dryRunPtr = flag.Bool("dry-run", os.Getenv(envPrefix+"DRY_RUN") == "true", "view summary without importing anything. recommended with verbosity")
	writeConcernPtr = flag.String("write-concern", os.Getenv(envPrefix+"WRITE_CONCERN"), "write concern options")
	noIndexRestorePtr = flag.Bool("no-index-restore", os.Getenv(envPrefix+"NO_INDEX_RESTORE") == "true", "don't restore indexes")
	noOptionsRestorePtr = flag.Bool("no-options-restore", os.Getenv(envPrefix+"NO_OPTIONS_RESTORE") == "true", "don't restore collection options")
	keepIndexVersionPtr = flag.Bool("keep-index-version", os.Getenv(envPrefix+"KEEP_INDEX_VERSION") == "true", "don't update index version")
	maintainInsertionOrderPtr = flag.Bool("maintain-insertion-order", os.Getenv(envPrefix+"MAINTAIN_INSERTION_ORDER") == "true", "restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1")
	numParallelCollectionsPtr = flag.String("num-parallel-collections", os.Getenv(envPrefix+"NUM_PARALLEL_COLLECTIONS"), "number of collections to restore in parallel (default: 4)")
	numInsertionWorkersPerCollectionPtr = flag.String("num-insertion-workers-per-collection", os.Getenv(envPrefix+"NUM_INSERTION_WORKERS_PER_COLLECTION"), "number of insert operations to run concurrently per collection (default: 1)")
	stopOnErrorPtr = flag.Bool("stop-on-error", os.Getenv(envPrefix+"STOP_ON_ERROR") == "true", "halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error")
	bypassDocumentValidationPtr = flag.Bool("bypass-document-validation", os.Getenv(envPrefix+"BYPASS_DOCUMENT_VALIDATION") == "true", "bypass document validation")
	preserveUUIDPtr = flag.Bool("preserve-uuid", os.Getenv(envPrefix+"PRESERVE_UUID") == "true", "preserve original collection UUIDs (off by default, requires drop)")

	azAccountNamePtr = flag.String("az-account-name", os.Getenv(envPrefix+"AZ_ACCOUNT_NAME"), "Azure Blob Storage Account Name")
	azAccountKeyPtr = flag.String("az-account-key", os.Getenv(envPrefix+"AZ_ACCOUNT_KEY"), "Azure Blob Storage Account Key")
	azContainerNamePtr = flag.String("az-container-name", os.Getenv(envPrefix+"AZ_CONTAINER_NAME"), "Azure Blob Storage Container Name")

	awsAccessKeyIdPtr = flag.String("aws-access-key-id", os.Getenv(envPrefix+"AWS_ACCESS_KEY_ID"), "AWS access key associated with an IAM account")
	awsSecretAccessKeyPtr = flag.String("aws-secret-access-key", os.Getenv(envPrefix+"AWS_SECRET_ACCESS_KEY"), "AWS secret key associated with the access key")
	awsRegionPtr = flag.String("aws-region", os.Getenv(envPrefix+"AWS_REGION"), "AWS Region whose servers you want to send your requests to")
	awsBucketPtr = flag.String("aws-bucket", os.Getenv(envPrefix+"AWS_BUCKET"), "AWS S3 bucket name")

	gcpBucketPtr = flag.String("gcp-bucket", os.Getenv(envPrefix+"GCP_BUCKET"), "GCP storage bucket name")
	gcpCredsFilePtr = flag.String("gcp-creds-file", os.Getenv(envPrefix+"GCP_CREDS_FILE"), "GCP service account's credentials file")
	gcpProjectIDPtr = flag.String("gcp-project-id", os.Getenv(envPrefix+"GCP_PROJECT_ID"), "GCP service account's project id")
	gcpPrivateKeyIdPtr = flag.String("gcp-private-key-id", os.Getenv(envPrefix+"GCP_PRIVATE_KEY_ID"), "GCP service account's private key id")
	gcpPrivateKeyPtr = flag.String("gcp-private-key", os.Getenv(envPrefix+"GCP_PRIVATE_KEY"), "GCP service account's private key")
	gcpClientEmailPtr = flag.String("gcp-client-email", os.Getenv(envPrefix+"GCP_CLIENT_EMAIL"), "GCP service account's client email")
	gcpClientIDPtr = flag.String("gcp-client-id", os.Getenv(envPrefix+"GCP_CLIENT_ID"), "GCP service account's client id")

	objectNamePtr = flag.String("object-name", os.Getenv(envPrefix+"OBJECT_NAME"), "Object name of the archived file in the storage (optional)")
	dirPtr = flag.String("dir", os.Getenv(envPrefix+"DIR"), "directory name that contains the dumped files")

	updatesPtr = flag.String("updates", os.Getenv(envPrefix+"UPDATES"), "array of update specifications in JSON string")
	updatesFilePtr = flag.String("updates-file", os.Getenv(envPrefix+"UPDATES_FILE"), "path to a file containing an array of update specifications")

	keepPtr = flag.Bool("keep", os.Getenv(envPrefix+"KEEP") == "true", "keep data dump")

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

	if *quietPtr == true {
		options = append(options, "--quiet")
	}

	if *hostPtr != "" {
		options = append(options, "--host="+*hostPtr)
	}

	if *portPtr != "" {
		options = append(options, "--port="+*portPtr)
	}

	if *sslPtr == true {
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

	if *sslAllowInvalidCertificatesPtr == true {
		options = append(options, "--sslAllowInvalidCertificates")
	}

	if *sslAllowInvalidHostnamesPtr == true {
		options = append(options, "--sslAllowInvalidHostnames")
	}

	if *sslFIPSModePtr == true {
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
		if *uriPrunePtr == true {
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

	if *dropPtr == true {
		options = append(options, "--drop")
	}

	if *dryRunPtr == true {
		options = append(options, "--dryRun")
	}

	if *writeConcernPtr != "" {
		options = append(options, "--writeConcern="+*writeConcernPtr)
	}

	if *noIndexRestorePtr == true {
		options = append(options, "--noIndexRestore")
	}

	if *noOptionsRestorePtr == true {
		options = append(options, "--noOptionsRestore")
	}

	if *keepIndexVersionPtr == true {
		options = append(options, "--keepIndexVersion")
	}

	if *maintainInsertionOrderPtr == true {
		options = append(options, "--maintainInsertionOrder")
	}

	if *numParallelCollectionsPtr != "" {
		options = append(options, "--numParallelCollections="+*numParallelCollectionsPtr)
	}

	if *numInsertionWorkersPerCollectionPtr != "" {
		options = append(options, "--numInsertionWorkersPerCollection="+*numInsertionWorkersPerCollectionPtr)
	}

	if *stopOnErrorPtr == true {
		options = append(options, "--stopOnError")
	}

	if *bypassDocumentValidationPtr == true {
		options = append(options, "--bypassDocumentValidation")
	}

	if *preserveUUIDPtr == true {
		options = append(options, "--preserveUUID")
	}

	return options
}

func getAzBlob() (*storage.AzBlob, error) {
	az := new(storage.AzBlob)
	err := az.Init(*azAccountNamePtr, *azAccountKeyPtr, *azContainerNamePtr)
	if err != nil {
		return nil, err
	}

	return az, nil
}

func getAwsS3() (*storage.AwsS3, error) {
	s3 := new(storage.AwsS3)
	err := s3.Init(*awsAccessKeyIdPtr, *awsSecretAccessKeyPtr, *awsRegionPtr, *awsBucketPtr)
	if err != nil {
		return nil, err
	}

	return s3, nil
}

func getGCP() (*storage.GcpStorage, error) {
	storage := new(storage.GcpStorage)

	err := storage.Init(*gcpBucketPtr, *gcpCredsFilePtr, *gcpProjectIDPtr, *gcpPrivateKeyIdPtr, *gcpPrivateKeyPtr, *gcpClientEmailPtr, *gcpClientIDPtr)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func GetStorage() (storage.Storage, error) {
	if useAzure() == true {
		return getAzBlob()
	}

	if useAWS() == true {
		return getAwsS3()
	}

	if useGCP() == true {
		return getGCP()
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
		content, err := ioutil.ReadFile(*updatesFilePtr)
		return content, err
	}

	return []byte(""), nil
}

func HasUpdates() bool {
	return *updatesPtr != "" || *updatesFilePtr != ""
}

func HasKeep() bool {
	return *keepPtr == true
}

func useAzure() bool {
	return *azAccountNamePtr != "" && *azAccountKeyPtr != "" && *azContainerNamePtr != ""
}

func useAWS() bool {
	return *awsAccessKeyIdPtr != "" && *awsSecretAccessKeyPtr != "" && *awsRegionPtr != "" && *awsBucketPtr != ""
}

func useGCP() bool {
	return *gcpBucketPtr != ""
}
