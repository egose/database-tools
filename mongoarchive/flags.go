package mongoarchive

import (
	"errors"
	"flag"
	"strconv"
	"time"

	"github.com/egose/database-tools/notification"
	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

const (
	envPrefix         = "MONGOARCHIVE__"
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

	dbPtr         *string
	collectionPtr *string

	uriPtr      *string
	uriPrunePtr *bool

	queryPtr          *string
	queryFilePtr      *string
	readPreferencePtr *string
	forceTableScanPtr *bool

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

	localPathPtr  *string
	expiryDaysPtr *string

	rocketChatWebhookUrlPtr          *string
	rocketChatWebhookPrefixPtr       *string
	rocketChatNotifyOnFailureOnlyPtr *bool

	cronPtr           *bool
	cronExpressionPtr *string
	tzPtr             *string

	keepPtr *bool

	loc            *time.Location
	expiryDays     int
	cronExpression string
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

	// namespace options:
	dbPtr = flag.String("db", env.GetValue("DB"), "database to use")
	collectionPtr = flag.String("collection", env.GetValue("COLLECTION"), "collection to use")

	// uri options:
	uriPtr = flag.String("uri", env.GetValue("URI"), "MongoDB uri connection string")
	uriPrunePtr = flag.Bool("uri-prune", env.GetValue("URI_PRUNE") == "true", "prune MongoDB uri connection string")

	// query options:
	queryPtr = flag.String("query", env.GetValue("QUERY"), "query filter, as a v2 Extended JSON string")
	queryFilePtr = flag.String("query-file", env.GetValue("QUERY_FILE"), "path to a file containing a query filter (v2 Extended JSON)")
	readPreferencePtr = flag.String("read-preference", env.GetValue("READ_PREFERENCE"), "specify either a preference mode (e.g. 'nearest') or a preference json object")
	forceTableScanPtr = flag.Bool("force-table-scan", env.GetValue("FORCE_TABLE_SCAN") == "true", "force a table scan")

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
	expiryDaysPtr = flag.String("expiry-days", env.GetValue("EXPIRY_DAYS"), "The maximum age, in days, for archives to be retained")

	rocketChatWebhookUrlPtr = flag.String("rocketchat-webhook-url", env.GetValue("ROCKETCHAT_WEBHOOK_URL"), "Rocket Chat Webhook URL")
	rocketChatWebhookPrefixPtr = flag.String("rocketchat-webhook-prefix", env.GetValue("ROCKETCHAT_WEBHOOK_PREFIX"), "Rocket Chat Webhook Prefix")
	rocketChatNotifyOnFailureOnlyPtr = flag.Bool("rocketchat-notify-on-failure-only", env.GetValue("ROCKETCHAT_NOTIFY_ON_FAILURE_ONLY") == "true", "Send Rocket Chat notifications only when something goes wrong during the execution")

	// cron options:
	cronPtr = flag.Bool("cron", env.GetValue("CRON") == "true", "run a cron schedular and block current execution path")
	cronExpressionPtr = flag.String("cron-expression", env.GetValue("CRON_EXPRESSION"), "a string describes individual details of the cron schedule")

	// See https://www.gnu.org/software/libc/manual/html_node/TZ-Variable.html
	tzPtr = flag.String("tz", env.GetValue("TZ"), "user-specified time zone")

	keepPtr = flag.Bool("keep", env.GetValue("KEEP") == "true", "keep data dump")

	flag.Parse()
	parseTZ()
	parseExpiry()
	parseCronExpression()
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

func parseCronExpression() {
	if cronExpressionPtr != nil || *cronExpressionPtr != "" {
		cronExpression = *cronExpressionPtr
	} else {
		cronExpression = "0 2 * * *"
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
	err := rc.Init(*rocketChatWebhookUrlPtr, *rocketChatWebhookPrefixPtr, *rocketChatNotifyOnFailureOnlyPtr)
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

func GetLocation() *time.Location {
	return loc
}

func GetCronExpression() string {
	return cronExpression
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
	return *awsAccessKeyIdPtr != "" && *awsSecretAccessKeyPtr != "" && *awsBucketPtr != ""
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
