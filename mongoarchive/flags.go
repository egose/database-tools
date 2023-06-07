package mongoarchive

import (
	"errors"
	"flag"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/junminahn/mongo-tools-ext/common"
)

const (
	envPrefix = "MONGOARCHIVE__"
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

	dbPtr         *string
	collectionPtr *string

	uriPtr *string

	queryPtr          *string
	queryFilePtr      *string
	readPreferencePtr *string
	forceTableScanPtr *bool

	azAccountNamePtr   *string
	azAccountKeyPtr    *string
	azContainerNamePtr *string

	awsAccessKeyIdPtr     *string
	awsSecretAccessKeyPtr *string
	awsRegionPtr          *string
	awsBucketPtr          *string

	cronPtr           *bool
	cronExpressionPtr *string
	tzPtr             *string

	keepPtr *bool

	loc *time.Location
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

	// namespace options:
	dbPtr = flag.String("db", os.Getenv(envPrefix+"DB"), "database to use")
	collectionPtr = flag.String("collection", os.Getenv(envPrefix+"COLLECTION"), "collection to use")

	// uri options:
	uriPtr = flag.String("uri", os.Getenv(envPrefix+"URI"), "MongoDB uri connection string")

	// query options:
	queryPtr = flag.String("query", os.Getenv(envPrefix+"QUERY"), "query filter, as a v2 Extended JSON string")
	queryFilePtr = flag.String("queryFile", os.Getenv(envPrefix+"QUERY_FILE"), "path to a file containing a query filter (v2 Extended JSON)")
	readPreferencePtr = flag.String("readPreference", os.Getenv(envPrefix+"READ_PREFERENCE"), "specify either a preference mode (e.g. 'nearest') or a preference json object")
	forceTableScanPtr = flag.Bool("forceTableScan", os.Getenv(envPrefix+"FORCE_TABLE_SCAN") == "true", "force a table scan")

	azAccountNamePtr = flag.String("azAccountName", os.Getenv(envPrefix+"AZ_ACCOUNT_NAME"), "Azure Blob Storage Account Name")
	azAccountKeyPtr = flag.String("azAccountKey", os.Getenv(envPrefix+"AZ_ACCOUNT_KEY"), "Azure Blob Storage Account Key")
	azContainerNamePtr = flag.String("azContainerName", os.Getenv(envPrefix+"AZ_CONTAINER_NAME"), "Azure Blob Storage Container Name")

	awsAccessKeyIdPtr = flag.String("awsAccessKeyId", os.Getenv(envPrefix+"AWS_ACCESS_KEY_ID"), "AWS access key associated with an IAM account")
	awsSecretAccessKeyPtr = flag.String("awsSecretAccessKey", os.Getenv(envPrefix+"AWS_SECRET_ACCESS_KEY"), "AWS secret key associated with the access key")
	awsRegionPtr = flag.String("awsRegion", os.Getenv(envPrefix+"AWS_REGION"), "AWS Region whose servers you want to send your requests to")
	awsBucketPtr = flag.String("awsBucket", os.Getenv(envPrefix+"AWS_BUCKET"), "AWS S3 bucket name")

	// cron options:
	cronPtr = flag.Bool("cron", os.Getenv(envPrefix+"CRON") == "true", "run a cron schedular and block current execution path")
	cronExpressionPtr = flag.String("cronExpression", os.Getenv(envPrefix+"CRON_EXPRESSION"), "a string describes individual details of the cron schedule")

	// See https://www.gnu.org/software/libc/manual/html_node/TZ-Variable.html
	tzPtr = flag.String("tz", os.Getenv("TZ"), "user-specified time zone")

	keepPtr = flag.Bool("keep", os.Getenv(envPrefix+"KEEP") == "true", "keep data dump")

	flag.Parse()
	parseTZ()
}

func parseTZ() {
	if *tzPtr != "" {
		loc, _ = time.LoadLocation(*tzPtr)
	} else {
		loc = time.Local
	}
}

func GetMongodumpOptions() []string {
	options := []string{
		"--gzip",
	}

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

	if *dbPtr != "" {
		options = append(options, "--db="+*dbPtr)
	}

	if *collectionPtr != "" {
		options = append(options, "--collection="+*collectionPtr)
	}

	if *uriPtr != "" {
		options = append(options, "--uri="+*uriPtr)
	}

	if *queryPtr != "" {
		options = append(options, "--query="+*queryPtr)
	}

	if *queryFilePtr != "" {
		options = append(options, "--queryFile="+*queryFilePtr)
	}

	if *readPreferencePtr != "" {
		options = append(options, "--readPreference="+*readPreferencePtr)
	}

	if *forceTableScanPtr == true {
		options = append(options, "--forceTableScan")
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

func GetCronScheduler() *gocron.Scheduler {
	var cronExpression string
	if cronExpressionPtr != nil {
		cronExpression = *cronExpressionPtr
	} else {
		cronExpression = "0 2 * * *"
	}
	return gocron.NewScheduler(loc).Cron(cronExpression)
}

func HasCron() bool {
	return *cronPtr == true
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
