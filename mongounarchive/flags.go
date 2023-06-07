package mongounarchive

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path"

	"github.com/junminahn/mongo-tools-ext/common"
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

	uriPtr *string

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
	sslCAFilePtr = flag.String("sslCAFile", os.Getenv(envPrefix+"SSL_CA_FILE"), "the .pem file containing the root certificate chain from the certificate authority")
	sslPEMKeyFilePtr = flag.String("sslPEMKeyFile", os.Getenv(envPrefix+"SSL_PEM_KEY_FILE"), "the .pem file containing the certificate and key")
	sslPEMKeyPasswordPtr = flag.String("sslPEMKeyPassword", os.Getenv(envPrefix+"SSL_PEM_KEY_PASSWORD"), "the password to decrypt the sslPEMKeyFile, if necessary")
	sslCRLFilePtr = flag.String("sslCRLFile", os.Getenv(envPrefix+"SSL_CRL_File"), "the .pem file containing the certificate revocation list")
	sslAllowInvalidCertificatesPtr = flag.Bool("sslAllowInvalidCertificates", os.Getenv(envPrefix+"SSL_ALLOW_INVALID_CERTIFICATES") == "true", "bypass the validation for server certificates")
	sslAllowInvalidHostnamesPtr = flag.Bool("sslAllowInvalidHostnames", os.Getenv(envPrefix+"SSL_ALLOW_INVALID_HOSTNAMES") == "true", "bypass the validation for server name")
	sslFIPSModePtr = flag.Bool("sslFIPSMode", os.Getenv(envPrefix+"SSL_FIPS_MODE") == "true", "use FIPS mode of the installed openssl library")

	// authentication options:
	usernamePtr = flag.String("username", os.Getenv(envPrefix+"USERNAME"), "username for authentication")
	passwordPtr = flag.String("password", os.Getenv(envPrefix+"PASSWORD"), "password for authentication")
	authenticationDatabasePtr = flag.String("authenticationDatabase", os.Getenv(envPrefix+"AUTHENTICATION_DATABASE"), "database that holds the user's credentials")
	authenticationMechanismPtr = flag.String("authenticationMechanism", os.Getenv(envPrefix+"AUTHENTICATION_MECHANISM"), "authentication mechanism to use")

	// kerberos options:
	gssapiServiceNamePtr = flag.String("gssapiServiceName", os.Getenv(envPrefix+"GSSAPI_SERVICE_NAME"), "service name to use when authenticating using GSSAPI/Kerberos (default: mongodb)")
	gssapiHostNamePtr = flag.String("gssapiHostName", os.Getenv(envPrefix+"GSSAPI_HOST_NAME"), "hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>)")

	// uri options:
	uriPtr = flag.String("uri", os.Getenv(envPrefix+"URI"), "MongoDB uri connection string")

	// namespace options:
	dbPtr = flag.String("db", os.Getenv(envPrefix+"DB"), "database to use")
	collectionPtr = flag.String("collection", os.Getenv(envPrefix+"COLLECTION"), "collection to use")
	nsExcludePtr = flag.String("nsExclude", os.Getenv(envPrefix+"NS_EXCLUDE"), "exclude matching namespaces")
	nsIncludePtr = flag.String("nsInclude", os.Getenv(envPrefix+"NS_INCLUDE"), "include matching namespaces")
	nsFromPtr = flag.String("nsFrom", os.Getenv(envPrefix+"NS_FROM"), "rename matching namespaces, must have matching nsTo")
	nsToPtr = flag.String("nsTo", os.Getenv(envPrefix+"NS_TO"), "rename matched namespaces, must have matching nsFrom")

	// restore options:
	dropPtr = flag.Bool("drop", os.Getenv(envPrefix+"DROP") == "true", "drop each collection before import")
	dryRunPtr = flag.Bool("dryRun", os.Getenv(envPrefix+"DRY_RUN") == "true", "view summary without importing anything. recommended with verbosity")
	writeConcernPtr = flag.String("writeConcern", os.Getenv(envPrefix+"WRITE_CONCERN"), "write concern options")
	noIndexRestorePtr = flag.Bool("noIndexRestore", os.Getenv(envPrefix+"NO_INDEX_RESTORE") == "true", "don't restore indexes")
	noOptionsRestorePtr = flag.Bool("noOptionsRestore", os.Getenv(envPrefix+"NO_OPTIONS_RESTORE") == "true", "don't restore collection options")
	keepIndexVersionPtr = flag.Bool("keepIndexVersion", os.Getenv(envPrefix+"KEEP_INDEX_VERSION") == "true", "don't update index version")
	maintainInsertionOrderPtr = flag.Bool("maintainInsertionOrder", os.Getenv(envPrefix+"MAINTAIN_INSERTION_ORDER") == "true", "restore the documents in the order of their appearance in the input source. By default the insertions will be performed in an arbitrary order. Setting this flag also enables the behavior of --stopOnError and restricts NumInsertionWorkersPerCollection to 1")
	numParallelCollectionsPtr = flag.String("numParallelCollections", os.Getenv(envPrefix+"NUM_PARALLEL_COLLECTIONS"), "number of collections to restore in parallel (default: 4)")
	numInsertionWorkersPerCollectionPtr = flag.String("numInsertionWorkersPerCollection", os.Getenv(envPrefix+"NUM_INSERTION_WORKERS_PER_COLLECTION"), "number of insert operations to run concurrently per collection (default: 1)")
	stopOnErrorPtr = flag.Bool("stopOnError", os.Getenv(envPrefix+"STOP_ON_ERROR") == "true", "halt after encountering any error during insertion. By default, mongorestore will attempt to continue through document validation and DuplicateKey errors, but with this option enabled, the tool will stop instead. A small number of documents may be inserted after encountering an error even with this option enabled; use --maintainInsertionOrder to halt immediately after an error")
	bypassDocumentValidationPtr = flag.Bool("bypassDocumentValidation", os.Getenv(envPrefix+"BYPASS_DOCUMENT_VALIDATION") == "true", "bypass document validation")
	preserveUUIDPtr = flag.Bool("preserveUUID", os.Getenv(envPrefix+"PRESERVE_UUID") == "true", "preserve original collection UUIDs (off by default, requires drop)")

	azAccountNamePtr = flag.String("azAccountName", os.Getenv(envPrefix+"AZ_ACCOUNT_NAME"), "Azure Blob Storage Account Name")
	azAccountKeyPtr = flag.String("azAccountKey", os.Getenv(envPrefix+"AZ_ACCOUNT_KEY"), "Azure Blob Storage Account Key")
	azContainerNamePtr = flag.String("azContainerName", os.Getenv(envPrefix+"AZ_CONTAINER_NAME"), "Azure Blob Storage Container Name")

	awsAccessKeyIdPtr = flag.String("awsAccessKeyId", os.Getenv(envPrefix+"AWS_ACCESS_KEY_ID"), "AWS access key associated with an IAM account")
	awsSecretAccessKeyPtr = flag.String("awsSecretAccessKey", os.Getenv(envPrefix+"AWS_SECRET_ACCESS_KEY"), "AWS secret key associated with the access key")
	awsRegionPtr = flag.String("awsRegion", os.Getenv(envPrefix+"AWS_REGION"), "AWS Region whose servers you want to send your requests to")
	awsBucketPtr = flag.String("awsBucket", os.Getenv(envPrefix+"AWS_BUCKET"), "AWS S3 bucket name")

	objectNamePtr = flag.String("objectName", os.Getenv(envPrefix+"OBJECT_NAME"), "Object name of the archived file in the storage (optional)")
	dirPtr = flag.String("dir", os.Getenv(envPrefix+"DIR"), "directory name that contains the dumped files")

	updatesPtr = flag.String("updates", os.Getenv(envPrefix+"UPDATES"), "array of update specifications in JSON string")
	updatesFilePtr = flag.String("updatesFile", os.Getenv(envPrefix+"UPDATES_FILE"), "path to a file containing an array of update specifications")

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
		options = append(options, "--uri="+*uriPtr)
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

func getAzBlob() (*common.AzBlob, error) {
	az := new(common.AzBlob)
	err := az.Init(*azAccountNamePtr, *azAccountKeyPtr, *azContainerNamePtr)
	if err != nil {
		return nil, err
	}

	return az, nil
}

func getAwsS3() (*common.AwsS3, error) {
	s3 := new(common.AwsS3)
	err := s3.Init(*awsAccessKeyIdPtr, *awsSecretAccessKeyPtr, *awsRegionPtr, *awsBucketPtr)
	if err != nil {
		return nil, err
	}

	return s3, nil
}

func GetStorage() (common.Storage, error) {
	if useAzure() == true {
		return getAzBlob()
	}

	if useAWS() == true {
		return getAwsS3()
	}

	return nil, errors.New("No storage provider detected.")
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
