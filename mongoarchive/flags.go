package mongoarchive

import (
	"errors"
	"flag"
	"os"
	"time"
	"strconv"

	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	"github.com/go-co-op/gocron"
	mlog "github.com/mongodb/mongo-tools/common/log"
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

	uriPtr      *string
	uriPrunePtr *bool

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

	gcpBucketPtr       *string
	gcpCredsFilePtr    *string
	gcpProjectIDPtr    *string
	gcpPrivateKeyIdPtr *string
	gcpPrivateKeyPtr   *string
	gcpClientEmailPtr  *string
	gcpClientIDPtr     *string

	localPathPtr *string
	expiryDaysPtr *string

	rocketChatWebhookUrlPtr *string

	cronPtr           *bool
	cronExpressionPtr *string
	tzPtr             *string

	keepPtr *bool

	loc *time.Location
	expiryDays int
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

	// namespace options:
	dbPtr = flag.String("db", os.Getenv(envPrefix+"DB"), "database to use")
	collectionPtr = flag.String("collection", os.Getenv(envPrefix+"COLLECTION"), "collection to use")

	// uri options:
	uriPtr = flag.String("uri", os.Getenv(envPrefix+"URI"), "MongoDB uri connection string")
	uriPrunePtr = flag.Bool("uri-prune", os.Getenv(envPrefix+"URI_PRUNE") == "true", "prune MongoDB uri connection string")

	// query options:
	queryPtr = flag.String("query", os.Getenv(envPrefix+"QUERY"), "query filter, as a v2 Extended JSON string")
	queryFilePtr = flag.String("query-file", os.Getenv(envPrefix+"QUERY_FILE"), "path to a file containing a query filter (v2 Extended JSON)")
	readPreferencePtr = flag.String("read-preference", os.Getenv(envPrefix+"READ_PREFERENCE"), "specify either a preference mode (e.g. 'nearest') or a preference json object")
	forceTableScanPtr = flag.Bool("force-table-scan", os.Getenv(envPrefix+"FORCE_TABLE_SCAN") == "true", "force a table scan")

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

	localPathPtr = flag.String("local-path", os.Getenv(envPrefix+"LOCAL_PATH"), "Local directory path to store backups")
	expiryDaysPtr = flag.String("expiry-days", os.Getenv(envPrefix+"EXPIRY_DAYS"), "The maximum age, in days, for archives to be retained")

	rocketChatWebhookUrlPtr = flag.String("rocketchat-webhook-url", os.Getenv(envPrefix+"ROCKETCHAT_WEBHOOK_URL"), "Rocket Chat Webhook URL")

	// cron options:
	cronPtr = flag.Bool("cron", os.Getenv(envPrefix+"CRON") == "true", "run a cron schedular and block current execution path")
	cronExpressionPtr = flag.String("cron-expression", os.Getenv(envPrefix+"CRON_EXPRESSION"), "a string describes individual details of the cron schedule")

	// See https://www.gnu.org/software/libc/manual/html_node/TZ-Variable.html
	tzPtr = flag.String("tz", os.Getenv("TZ"), "user-specified time zone")

	keepPtr = flag.Bool("keep", os.Getenv(envPrefix+"KEEP") == "true", "keep data dump")

	flag.Parse()
	parseTZ()
	parseExpiry()
}

func parseTZ() {
	if *tzPtr != "" {
		loc, _ = time.LoadLocation(*tzPtr)
	} else {
		loc = time.Local
	}

	mlog.Logvf(mlog.Always, "Use Time Zone: %v", loc)
}

func parseExpiry() {
	if *expiryDaysPtr != "" {
		num, err := strconv.Atoi(*expiryDaysPtr)
		if err != nil {
			expiryDays = num
		}
	} else {
		expiryDays = 0
	}
}

func GetTZ() *time.Location {
	return loc
}

func GetMongodumpOptions() []string {
	options := []string{
		"--gzip",
	}

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

	if *dbPtr != "" {
		options = append(options, "--db="+*dbPtr)
	}

	if *collectionPtr != "" {
		options = append(options, "--collection="+*collectionPtr)
	}

	if *uriPtr != "" {
		uri := *uriPtr
		if *uriPrunePtr {
			uri = utils.PruneMongoDBURI(uri)
		}

		options = append(options, "--uri="+uri)
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

	if *forceTableScanPtr {
		options = append(options, "--forceTableScan")
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

func getLocal() (*storage.LocalStorage, error) {
	storage := new(storage.LocalStorage)
	err := storage.Init(*localPathPtr, expiryDays)
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

func getRocketChat() (*notification.RocketChat, error) {
	rc := new(notification.RocketChat)
	err := rc.Init(*rocketChatWebhookUrlPtr)
	return rc, err
}

func GetNotifications() []notification.Notification {
	notifications := make([]notification.Notification, 0)

	if useRocketChat() {
		rc, _ := getRocketChat()
		if rc != nil {
			mlog.Logvf(mlog.Always, "Found Notification Option: %v", "RocketChat")
			notifications = append(notifications, rc)
		}
	}

	return notifications
}

func GetCronScheduler() *gocron.Scheduler {
	var cronExpression string
	if cronExpressionPtr != nil {
		cronExpression = *cronExpressionPtr
	} else {
		cronExpression = "0 2 * * *"
	}

	mlog.Logvf(mlog.Always, "Use Cron Expression: %v", cronExpression)
	return gocron.NewScheduler(loc).Cron(cronExpression)
}

func HasCron() bool {
	return *cronPtr
}

func HasKeep() bool {
	return *keepPtr
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

func useLocal() bool {
	return *localPathPtr != ""
}

func useRocketChat() bool {
	return *rocketChatWebhookUrlPtr != ""
}
