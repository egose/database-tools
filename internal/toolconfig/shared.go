package toolconfig

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoFlagBindings struct {
	Verbose                     *string
	Quiet                       *bool
	Host                        *string
	Port                        *string
	SSL                         *bool
	SSLCAFile                   *string
	SSLPEMKeyFile               *string
	SSLPEMKeyPassword           *string
	SSLCRLFile                  *string
	SSLAllowInvalidCertificates *bool
	SSLAllowInvalidHostnames    *bool
	SSLFIPSMode                 *bool
	Username                    *string
	Password                    *string
	AuthenticationDatabase      *string
	AuthenticationMechanism     *string
	GSSAPIServiceName           *string
	GSSAPIHostName              *string
	DB                          *string
	Collection                  *string
	URI                         *string
	URIPrune                    *bool
}

var mongoFlagDefs = struct {
	verbose                     StringFlagDef
	quiet                       BoolFlagDef
	host                        StringFlagDef
	port                        StringFlagDef
	ssl                         BoolFlagDef
	sslCAFile                   StringFlagDef
	sslPEMKeyFile               StringFlagDef
	sslPEMKeyPassword           StringFlagDef
	sslCRLFile                  StringFlagDef
	sslAllowInvalidCertificates BoolFlagDef
	sslAllowInvalidHostnames    BoolFlagDef
	sslFIPSMode                 BoolFlagDef
	username                    StringFlagDef
	password                    StringFlagDef
	authenticationDatabase      StringFlagDef
	authenticationMechanism     StringFlagDef
	gssapiServiceName           StringFlagDef
	gssapiHostName              StringFlagDef
	db                          StringFlagDef
	collection                  StringFlagDef
	uri                         StringFlagDef
	uriPrune                    BoolFlagDef
}{
	verbose:                     StringFlagDef{Name: "verbose", EnvKey: "VERBOSE", Usage: "more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N)"},
	quiet:                       BoolFlagDef{Name: "quiet", EnvKey: "QUIET", Usage: "hide all log output"},
	host:                        StringFlagDef{Name: "host", EnvKey: "HOST", Usage: "MongoDB host to connect to (setname/host1,host2 for replica sets)"},
	port:                        StringFlagDef{Name: "port", EnvKey: "PORT", Usage: "MongoDB port (can also use --host hostname:port)"},
	ssl:                         BoolFlagDef{Name: "ssl", EnvKey: "SSL", Usage: "connect to a mongod or mongos that has ssl enabled"},
	sslCAFile:                   StringFlagDef{Name: "ssl-ca-file", EnvKey: "SSL_CA_FILE", Usage: "the .pem file containing the root certificate chain from the certificate authority"},
	sslPEMKeyFile:               StringFlagDef{Name: "ssl-pem-key-file", EnvKey: "SSL_PEM_KEY_FILE", Usage: "the .pem file containing the certificate and key"},
	sslPEMKeyPassword:           StringFlagDef{Name: "ssl-pem-key-password", EnvKey: "SSL_PEM_KEY_PASSWORD", Usage: "the password to decrypt the sslPEMKeyFile, if necessary"},
	sslCRLFile:                  StringFlagDef{Name: "ssl-crl-file", EnvKey: "SSL_CRL_FILE", Usage: "the .pem file containing the certificate revocation list"},
	sslAllowInvalidCertificates: BoolFlagDef{Name: "ssl-allow-invalid-certificates", EnvKey: "SSL_ALLOW_INVALID_CERTIFICATES", Usage: "bypass the validation for server certificates"},
	sslAllowInvalidHostnames:    BoolFlagDef{Name: "ssl-allow-invalid-hostnames", EnvKey: "SSL_ALLOW_INVALID_HOSTNAMES", Usage: "bypass the validation for server name"},
	sslFIPSMode:                 BoolFlagDef{Name: "ssl-fips-mode", EnvKey: "SSL_FIPS_MODE", Usage: "use FIPS mode of the installed openssl library"},
	username:                    StringFlagDef{Name: "username", EnvKey: "USERNAME", Usage: "username for authentication"},
	password:                    StringFlagDef{Name: "password", EnvKey: "PASSWORD", Usage: "password for authentication"},
	authenticationDatabase:      StringFlagDef{Name: "authentication-database", EnvKey: "AUTHENTICATION_DATABASE", Usage: "database that holds the user's credentials"},
	authenticationMechanism:     StringFlagDef{Name: "authentication-mechanism", EnvKey: "AUTHENTICATION_MECHANISM", Usage: "authentication mechanism to use"},
	gssapiServiceName:           StringFlagDef{Name: "gssapi-service-name", EnvKey: "GSSAPI_SERVICE_NAME", Usage: "service name to use when authenticating using GSSAPI/Kerberos (default: mongodb)"},
	gssapiHostName:              StringFlagDef{Name: "gssapi-host-name", EnvKey: "GSSAPI_HOST_NAME", Usage: "hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>)"},
	db:                          StringFlagDef{Name: "db", EnvKey: "DB", Usage: "database to use"},
	collection:                  StringFlagDef{Name: "collection", EnvKey: "COLLECTION", Usage: "collection to use"},
	uri:                         StringFlagDef{Name: "uri", EnvKey: "URI", Usage: "MongoDB uri connection string"},
	uriPrune:                    BoolFlagDef{Name: "uri-prune", EnvKey: "URI_PRUNE", Usage: "prune MongoDB uri connection string"},
}

func BindMongoFlags(fs FlagBinder, env EnvReader) MongoFlagBindings {
	return MongoFlagBindings{
		Verbose:                     mongoFlagDefs.verbose.Bind(fs, env),
		Quiet:                       mongoFlagDefs.quiet.Bind(fs, env),
		Host:                        mongoFlagDefs.host.Bind(fs, env),
		Port:                        mongoFlagDefs.port.Bind(fs, env),
		SSL:                         mongoFlagDefs.ssl.Bind(fs, env),
		SSLCAFile:                   mongoFlagDefs.sslCAFile.Bind(fs, env),
		SSLPEMKeyFile:               mongoFlagDefs.sslPEMKeyFile.Bind(fs, env),
		SSLPEMKeyPassword:           mongoFlagDefs.sslPEMKeyPassword.Bind(fs, env),
		SSLCRLFile:                  mongoFlagDefs.sslCRLFile.Bind(fs, env),
		SSLAllowInvalidCertificates: mongoFlagDefs.sslAllowInvalidCertificates.Bind(fs, env),
		SSLAllowInvalidHostnames:    mongoFlagDefs.sslAllowInvalidHostnames.Bind(fs, env),
		SSLFIPSMode:                 mongoFlagDefs.sslFIPSMode.Bind(fs, env),
		Username:                    mongoFlagDefs.username.Bind(fs, env),
		Password:                    mongoFlagDefs.password.Bind(fs, env),
		AuthenticationDatabase:      mongoFlagDefs.authenticationDatabase.Bind(fs, env),
		AuthenticationMechanism:     mongoFlagDefs.authenticationMechanism.Bind(fs, env),
		GSSAPIServiceName:           mongoFlagDefs.gssapiServiceName.Bind(fs, env),
		GSSAPIHostName:              mongoFlagDefs.gssapiHostName.Bind(fs, env),
		DB:                          mongoFlagDefs.db.Bind(fs, env),
		Collection:                  mongoFlagDefs.collection.Bind(fs, env),
		URI:                         mongoFlagDefs.uri.Bind(fs, env),
		URIPrune:                    mongoFlagDefs.uriPrune.Bind(fs, env),
	}
}

func MongoFlagDocs(envPrefix string) []FlagDoc {
	return []FlagDoc{
		mongoFlagDefs.verbose.Doc(envPrefix),
		mongoFlagDefs.quiet.Doc(envPrefix),
		mongoFlagDefs.host.Doc(envPrefix),
		mongoFlagDefs.port.Doc(envPrefix),
		mongoFlagDefs.ssl.Doc(envPrefix),
		mongoFlagDefs.sslCAFile.Doc(envPrefix),
		mongoFlagDefs.sslPEMKeyFile.Doc(envPrefix),
		mongoFlagDefs.sslPEMKeyPassword.Doc(envPrefix),
		mongoFlagDefs.sslCRLFile.Doc(envPrefix),
		mongoFlagDefs.sslAllowInvalidCertificates.Doc(envPrefix),
		mongoFlagDefs.sslAllowInvalidHostnames.Doc(envPrefix),
		mongoFlagDefs.sslFIPSMode.Doc(envPrefix),
		mongoFlagDefs.username.Doc(envPrefix),
		mongoFlagDefs.password.Doc(envPrefix),
		mongoFlagDefs.authenticationDatabase.Doc(envPrefix),
		mongoFlagDefs.authenticationMechanism.Doc(envPrefix),
		mongoFlagDefs.gssapiServiceName.Doc(envPrefix),
		mongoFlagDefs.gssapiHostName.Doc(envPrefix),
		mongoFlagDefs.db.Doc(envPrefix),
		mongoFlagDefs.collection.Doc(envPrefix),
		mongoFlagDefs.uri.Doc(envPrefix),
		mongoFlagDefs.uriPrune.Doc(envPrefix),
	}
}

func (b MongoFlagBindings) Apply(target *MongoOptions) {
	target.Verbose = *b.Verbose
	target.Quiet = *b.Quiet
	target.Host = *b.Host
	target.Port = *b.Port
	target.SSL = *b.SSL
	target.SSLCAFile = *b.SSLCAFile
	target.SSLPEMKeyFile = *b.SSLPEMKeyFile
	target.SSLPEMKeyPassword = *b.SSLPEMKeyPassword
	target.SSLCRLFile = *b.SSLCRLFile
	target.SSLAllowInvalidCertificates = *b.SSLAllowInvalidCertificates
	target.SSLAllowInvalidHostnames = *b.SSLAllowInvalidHostnames
	target.SSLFIPSMode = *b.SSLFIPSMode
	target.Username = *b.Username
	target.Password = *b.Password
	target.AuthenticationDatabase = *b.AuthenticationDatabase
	target.AuthenticationMechanism = *b.AuthenticationMechanism
	target.GSSAPIServiceName = *b.GSSAPIServiceName
	target.GSSAPIHostName = *b.GSSAPIHostName
	target.DB = *b.DB
	target.Collection = *b.Collection
	target.URI = *b.URI
	target.URIPrune = *b.URIPrune
}

type StorageFlagBindings struct {
	AZEndpoint          *string
	AZAccountName       *string
	AZAccountKey        *string
	AZContainerName     *string
	AWSEndpoint         *string
	AWSAccessKeyID      *string
	AWSSecretAccessKey  *string
	AWSRegion           *string
	AWSBucket           *string
	AWSS3ForcePathStyle *bool
	GCPEndpoint         *string
	GCPBucket           *string
	GCPCredsFile        *string
	GCPProjectID        *string
	GCPPrivateKeyID     *string
	GCPPrivateKey       *string
	GCPClientEmail      *string
	GCPClientID         *string
	LocalPath           *string
	BackupPrefix        *string
}

var storageFlagDefs = struct {
	azEndpoint          StringFlagDef
	azAccountName       StringFlagDef
	azAccountKey        StringFlagDef
	azContainerName     StringFlagDef
	awsEndpoint         StringFlagDef
	awsAccessKeyID      StringFlagDef
	awsSecretAccessKey  StringFlagDef
	awsRegion           StringFlagDef
	awsBucket           StringFlagDef
	awsS3ForcePathStyle BoolFlagDef
	gcpEndpoint         StringFlagDef
	gcpBucket           StringFlagDef
	gcpCredsFile        StringFlagDef
	gcpProjectID        StringFlagDef
	gcpPrivateKeyID     StringFlagDef
	gcpPrivateKey       StringFlagDef
	gcpClientEmail      StringFlagDef
	gcpClientID         StringFlagDef
	localPath           StringFlagDef
	backupPrefix        StringFlagDef
}{
	azEndpoint:          StringFlagDef{Name: "az-endpoint", EnvKey: "AZ_ENDPOINT", Usage: "specify the emulator hostname and Azure Blob Storage port"},
	azAccountName:       StringFlagDef{Name: "az-account-name", EnvKey: "AZ_ACCOUNT_NAME", Usage: "Azure Blob Storage Account Name"},
	azAccountKey:        StringFlagDef{Name: "az-account-key", EnvKey: "AZ_ACCOUNT_KEY", Usage: "Azure Blob Storage Account Key"},
	azContainerName:     StringFlagDef{Name: "az-container-name", EnvKey: "AZ_CONTAINER_NAME", Usage: "Azure Blob Storage Container Name"},
	awsEndpoint:         StringFlagDef{Name: "aws-endpoint", EnvKey: "AWS_ENDPOINT", Usage: "AWS endpoint URL (hostname only or fully qualified URI)"},
	awsAccessKeyID:      StringFlagDef{Name: "aws-access-key-id", EnvKey: "AWS_ACCESS_KEY_ID", Usage: "AWS access key associated with an IAM account"},
	awsSecretAccessKey:  StringFlagDef{Name: "aws-secret-access-key", EnvKey: "AWS_SECRET_ACCESS_KEY", Usage: "AWS secret key associated with the access key"},
	awsRegion:           StringFlagDef{Name: "aws-region", EnvKey: "AWS_REGION", Usage: "AWS Region whose servers you want to send your requests to", Defaults: []string{"us-east-1"}},
	awsBucket:           StringFlagDef{Name: "aws-bucket", EnvKey: "AWS_BUCKET", Usage: "AWS S3 bucket name"},
	awsS3ForcePathStyle: BoolFlagDef{Name: "aws-s3-force-path-style", EnvKey: "AWS_S3_FORCE_PATH_STYLE", Usage: "force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`)"},
	gcpEndpoint:         StringFlagDef{Name: "gcp-endpoint", EnvKey: "GCP_ENDPOINT", Usage: "GCP endpoint URL"},
	gcpBucket:           StringFlagDef{Name: "gcp-bucket", EnvKey: "GCP_BUCKET", Usage: "GCP storage bucket name"},
	gcpCredsFile:        StringFlagDef{Name: "gcp-creds-file", EnvKey: "GCP_CREDS_FILE", Usage: "GCP service account's credentials file"},
	gcpProjectID:        StringFlagDef{Name: "gcp-project-id", EnvKey: "GCP_PROJECT_ID", Usage: "GCP service account's project id"},
	gcpPrivateKeyID:     StringFlagDef{Name: "gcp-private-key-id", EnvKey: "GCP_PRIVATE_KEY_ID", Usage: "GCP service account's private key id"},
	gcpPrivateKey:       StringFlagDef{Name: "gcp-private-key", EnvKey: "GCP_PRIVATE_KEY", Usage: "GCP service account's private key"},
	gcpClientEmail:      StringFlagDef{Name: "gcp-client-email", EnvKey: "GCP_CLIENT_EMAIL", Usage: "GCP service account's client email"},
	gcpClientID:         StringFlagDef{Name: "gcp-client-id", EnvKey: "GCP_CLIENT_ID", Usage: "GCP service account's client id"},
	localPath:           StringFlagDef{Name: "local-path", EnvKey: "LOCAL_PATH", Usage: "Local directory path to store backups"},
	backupPrefix:        StringFlagDef{Name: "backup-prefix", EnvKey: "BACKUP_PREFIX", Usage: "Prefix/namespace used for managed backup objects", Defaults: []string{storage.DefaultBackupPrefix}},
}

var restoreStorageBackendFlagDef = StringFlagDef{Name: "storage-backend", EnvKey: "STORAGE_BACKEND", Usage: "Storage backend to use for restore when multiple backends are configured (azure, aws, gcp, local)"}

func BindStorageFlags(fs FlagBinder, env EnvReader) StorageFlagBindings {
	return StorageFlagBindings{
		AZEndpoint:          storageFlagDefs.azEndpoint.Bind(fs, env),
		AZAccountName:       storageFlagDefs.azAccountName.Bind(fs, env),
		AZAccountKey:        storageFlagDefs.azAccountKey.Bind(fs, env),
		AZContainerName:     storageFlagDefs.azContainerName.Bind(fs, env),
		AWSEndpoint:         storageFlagDefs.awsEndpoint.Bind(fs, env),
		AWSAccessKeyID:      storageFlagDefs.awsAccessKeyID.Bind(fs, env),
		AWSSecretAccessKey:  storageFlagDefs.awsSecretAccessKey.Bind(fs, env),
		AWSRegion:           storageFlagDefs.awsRegion.Bind(fs, env),
		AWSBucket:           storageFlagDefs.awsBucket.Bind(fs, env),
		AWSS3ForcePathStyle: storageFlagDefs.awsS3ForcePathStyle.Bind(fs, env),
		GCPEndpoint:         storageFlagDefs.gcpEndpoint.Bind(fs, env),
		GCPBucket:           storageFlagDefs.gcpBucket.Bind(fs, env),
		GCPCredsFile:        storageFlagDefs.gcpCredsFile.Bind(fs, env),
		GCPProjectID:        storageFlagDefs.gcpProjectID.Bind(fs, env),
		GCPPrivateKeyID:     storageFlagDefs.gcpPrivateKeyID.Bind(fs, env),
		GCPPrivateKey:       storageFlagDefs.gcpPrivateKey.Bind(fs, env),
		GCPClientEmail:      storageFlagDefs.gcpClientEmail.Bind(fs, env),
		GCPClientID:         storageFlagDefs.gcpClientID.Bind(fs, env),
		LocalPath:           storageFlagDefs.localPath.Bind(fs, env),
		BackupPrefix:        storageFlagDefs.backupPrefix.Bind(fs, env),
	}
}

func BindRestoreStorageBackendFlag(fs FlagBinder, env EnvReader) *string {
	return restoreStorageBackendFlagDef.Bind(fs, env)
}

func StorageFlagDocs(envPrefix string) []FlagDoc {
	return []FlagDoc{
		storageFlagDefs.azEndpoint.Doc(envPrefix),
		storageFlagDefs.azAccountName.Doc(envPrefix),
		storageFlagDefs.azAccountKey.Doc(envPrefix),
		storageFlagDefs.azContainerName.Doc(envPrefix),
		storageFlagDefs.awsEndpoint.Doc(envPrefix),
		storageFlagDefs.awsAccessKeyID.Doc(envPrefix),
		storageFlagDefs.awsSecretAccessKey.Doc(envPrefix),
		storageFlagDefs.awsRegion.Doc(envPrefix),
		storageFlagDefs.awsBucket.Doc(envPrefix),
		storageFlagDefs.awsS3ForcePathStyle.Doc(envPrefix),
		storageFlagDefs.gcpEndpoint.Doc(envPrefix),
		storageFlagDefs.gcpBucket.Doc(envPrefix),
		storageFlagDefs.gcpCredsFile.Doc(envPrefix),
		storageFlagDefs.gcpProjectID.Doc(envPrefix),
		storageFlagDefs.gcpPrivateKeyID.Doc(envPrefix),
		storageFlagDefs.gcpPrivateKey.Doc(envPrefix),
		storageFlagDefs.gcpClientEmail.Doc(envPrefix),
		storageFlagDefs.gcpClientID.Doc(envPrefix),
		storageFlagDefs.localPath.Doc(envPrefix),
		storageFlagDefs.backupPrefix.Doc(envPrefix),
	}
}

func RestoreStorageBackendFlagDoc(envPrefix string) FlagDoc {
	return restoreStorageBackendFlagDef.Doc(envPrefix)
}

func (b StorageFlagBindings) Apply(target *StorageOptions) {
	target.AZEndpoint = *b.AZEndpoint
	target.AZAccountName = *b.AZAccountName
	target.AZAccountKey = *b.AZAccountKey
	target.AZContainerName = *b.AZContainerName
	target.AWSEndpoint = *b.AWSEndpoint
	target.AWSAccessKeyID = *b.AWSAccessKeyID
	target.AWSSecretAccessKey = *b.AWSSecretAccessKey
	target.AWSRegion = *b.AWSRegion
	target.AWSBucket = *b.AWSBucket
	target.AWSS3ForcePathStyle = *b.AWSS3ForcePathStyle
	target.GCPEndpoint = *b.GCPEndpoint
	target.GCPBucket = *b.GCPBucket
	target.GCPCredsFile = *b.GCPCredsFile
	target.GCPProjectID = *b.GCPProjectID
	target.GCPPrivateKeyID = *b.GCPPrivateKeyID
	target.GCPPrivateKey = *b.GCPPrivateKey
	target.GCPClientEmail = *b.GCPClientEmail
	target.GCPClientID = *b.GCPClientID
	target.LocalPath = *b.LocalPath
	target.BackupPrefix = *b.BackupPrefix
}

type MongoOptions struct {
	Verbose                     string
	Quiet                       bool
	Host                        string
	Port                        string
	SSL                         bool
	SSLCAFile                   string
	SSLPEMKeyFile               string
	SSLPEMKeyPassword           string
	SSLCRLFile                  string
	SSLAllowInvalidCertificates bool
	SSLAllowInvalidHostnames    bool
	SSLFIPSMode                 bool
	Username                    string
	Password                    string
	AuthenticationDatabase      string
	AuthenticationMechanism     string
	GSSAPIServiceName           string
	GSSAPIHostName              string
	DB                          string
	Collection                  string
	URI                         string
	URIPrune                    bool
}

func (m MongoOptions) AppendToolOptions(options []string) []string {
	if m.Verbose != "" {
		options = append(options, "--verbose="+m.Verbose)
	}
	if m.Quiet {
		options = append(options, "--quiet")
	}
	if m.Host != "" {
		options = append(options, "--host="+m.Host)
	}
	if m.Port != "" {
		options = append(options, "--port="+m.Port)
	}
	if m.SSL {
		options = append(options, "--ssl")
	}
	if m.SSLCAFile != "" {
		options = append(options, "--sslCAFile="+m.SSLCAFile)
	}
	if m.SSLPEMKeyFile != "" {
		options = append(options, "--sslPEMKeyFile="+m.SSLPEMKeyFile)
	}
	if m.SSLPEMKeyPassword != "" {
		options = append(options, "--sslPEMKeyPassword="+m.SSLPEMKeyPassword)
	}
	if m.SSLCRLFile != "" {
		options = append(options, "--sslCRLFile="+m.SSLCRLFile)
	}
	if m.SSLAllowInvalidCertificates {
		options = append(options, "--sslAllowInvalidCertificates")
	}
	if m.SSLAllowInvalidHostnames {
		options = append(options, "--sslAllowInvalidHostnames")
	}
	if m.SSLFIPSMode {
		options = append(options, "--sslFIPSMode")
	}
	if m.Username != "" {
		options = append(options, "--username="+m.Username)
	}
	if m.Password != "" {
		options = append(options, "--password="+m.Password)
	}
	if m.AuthenticationDatabase != "" {
		options = append(options, "--authenticationDatabase="+m.AuthenticationDatabase)
	}
	if m.AuthenticationMechanism != "" {
		options = append(options, "--authenticationMechanism="+m.AuthenticationMechanism)
	}
	if m.GSSAPIServiceName != "" {
		options = append(options, "--gssapiServiceName="+m.GSSAPIServiceName)
	}
	if m.GSSAPIHostName != "" {
		options = append(options, "--gssapiHostName="+m.GSSAPIHostName)
	}
	if m.DB != "" {
		options = append(options, "--db="+m.DB)
	}
	if m.Collection != "" {
		options = append(options, "--collection="+m.Collection)
	}
	if m.URI != "" {
		uri := m.URI
		if m.URIPrune {
			uri = utils.PruneMongoDBURI(uri)
		}
		options = append(options, "--uri="+uri)
	}

	return options
}

func (m MongoOptions) MongoConnectionURI() (string, error) {
	if m.URI != "" {
		return m.URI, nil
	}

	hosts := strings.TrimSpace(m.Host)
	if hosts == "" {
		hosts = "localhost"
	}

	replicaSet := ""
	if parts := strings.SplitN(hosts, "/", 2); len(parts) == 2 {
		replicaSet = strings.TrimSpace(parts[0])
		hosts = parts[1]
	}

	hostList := strings.Split(hosts, ",")
	for i, host := range hostList {
		host = strings.TrimSpace(host)
		if host == "" {
			return "", fmt.Errorf("invalid host list")
		}
		if m.Port != "" && !strings.Contains(host, ":") {
			host += ":" + m.Port
		}
		hostList[i] = host
	}

	uri := &url.URL{Scheme: "mongodb", Host: strings.Join(hostList, ","), Path: "/"}
	if m.Username != "" {
		if m.Password != "" {
			uri.User = url.UserPassword(m.Username, m.Password)
		} else {
			uri.User = url.User(m.Username)
		}
	}

	query := url.Values{}
	if replicaSet != "" {
		query.Set("replicaSet", replicaSet)
	}
	if m.AuthenticationDatabase != "" {
		query.Set("authSource", m.AuthenticationDatabase)
	}
	if m.AuthenticationMechanism != "" {
		query.Set("authMechanism", m.AuthenticationMechanism)
	}
	if m.SSL {
		query.Set("tls", "true")
	}
	if m.SSLAllowInvalidCertificates || m.SSLAllowInvalidHostnames {
		query.Set("tlsInsecure", "true")
	}

	authProps := make([]string, 0, 2)
	if m.GSSAPIServiceName != "" {
		authProps = append(authProps, "SERVICE_NAME:"+m.GSSAPIServiceName)
	}
	if m.GSSAPIHostName != "" {
		authProps = append(authProps, "SERVICE_HOST:"+m.GSSAPIHostName)
	}
	if len(authProps) > 0 {
		query.Set("authMechanismProperties", strings.Join(authProps, ","))
	}

	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

func (m MongoOptions) MongoClientOptions() (*mongooptions.ClientOptions, error) {
	uri, err := m.MongoConnectionURI()
	if err != nil {
		return nil, err
	}

	if m.SSLCRLFile != "" {
		return nil, errors.New("ssl-crl-file is not supported for update operations via the MongoDB Go driver")
	}
	if m.SSLFIPSMode {
		return nil, errors.New("ssl-fips-mode is not supported for update operations via the MongoDB Go driver")
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	query := parsedURI.Query()
	if m.SSLCAFile != "" {
		query.Set("tlsCAFile", m.SSLCAFile)
	}
	if m.SSLPEMKeyFile != "" {
		query.Set("tlsCertificateKeyFile", m.SSLPEMKeyFile)
	}
	if m.SSLPEMKeyPassword != "" {
		query.Set("tlsCertificateKeyFilePassword", m.SSLPEMKeyPassword)
	}
	parsedURI.RawQuery = query.Encode()

	return mongooptions.Client().ApplyURI(parsedURI.String()), nil
}

type StorageOptions struct {
	AZEndpoint          string
	AZAccountName       string
	AZAccountKey        string
	AZContainerName     string
	AWSEndpoint         string
	AWSAccessKeyID      string
	AWSSecretAccessKey  string
	AWSRegion           string
	AWSBucket           string
	AWSS3ForcePathStyle bool
	GCPEndpoint         string
	GCPBucket           string
	GCPCredsFile        string
	GCPProjectID        string
	GCPPrivateKeyID     string
	GCPPrivateKey       string
	GCPClientEmail      string
	GCPClientID         string
	LocalPath           string
	BackupPrefix        string
	StorageBackend      string
}

type storageSetting struct {
	identifier string
	value      string
}

type storageBackendOption[T storage.Lifecycle] struct {
	name    string
	enabled func() bool
	getter  func() (T, error)
}

func getConfiguredBackends[T storage.Lifecycle](options []storageBackendOption[T], validate func() error) ([]T, error) {
	var zero T

	if err := validate(); err != nil {
		return nil, err
	}

	storages := make([]T, 0)
	foundNames := make([]string, 0)
	initErrors := make([]error, 0)
	for _, opt := range options {
		if opt.enabled() {
			storageBackend, err := opt.getter()
			if err != nil {
				initErrors = append(initErrors, fmt.Errorf("%s storage initialization failed: %w", opt.name, err))
				continue
			}
			if any(storageBackend) != any(zero) {
				storages = append(storages, storageBackend)
				foundNames = append(foundNames, opt.name)
			}
		}
	}

	if len(initErrors) > 0 {
		for _, storageBackend := range storages {
			_ = storageBackend.Close()
		}
		return nil, errors.Join(initErrors...)
	}

	for _, name := range foundNames {
		mlog.Logvf(mlog.Always, "Found Storage Option: %v", name)
	}

	return storages, nil
}

func (s StorageOptions) GetArchiveStorages(ctx context.Context, expiryDays int) ([]storage.ArchiveBackend, error) {
	options := []storageBackendOption[storage.ArchiveBackend]{
		{"Local", s.useLocal, func() (storage.ArchiveBackend, error) { return s.getLocalStorage(expiryDays) }},
		{"Azure", s.useAzure, func() (storage.ArchiveBackend, error) { return s.getAzBlobStorage(expiryDays) }},
		{"AWS", s.useAWS, func() (storage.ArchiveBackend, error) { return s.getAwsS3Storage(expiryDays) }},
		{"GCP", s.useGCP, func() (storage.ArchiveBackend, error) { return s.getGcpStorage(ctx, expiryDays) }},
	}

	return getConfiguredBackends(options, s.Validate)
}

func (s StorageOptions) GetRestoreStorages(ctx context.Context, expiryDays int) ([]storage.RestoreBackend, error) {
	options := []storageBackendOption[storage.RestoreBackend]{
		{"Local", s.useLocal, func() (storage.RestoreBackend, error) { return s.getLocalStorage(expiryDays) }},
		{"Azure", s.useAzure, func() (storage.RestoreBackend, error) { return s.getAzBlobStorage(expiryDays) }},
		{"AWS", s.useAWS, func() (storage.RestoreBackend, error) { return s.getAwsS3Storage(expiryDays) }},
		{"GCP", s.useGCP, func() (storage.RestoreBackend, error) { return s.getGcpStorage(ctx, expiryDays) }},
	}

	return getConfiguredBackends(options, s.Validate)
}

func (s StorageOptions) Validate() error {
	return errors.Join(
		s.validateAzure(),
		s.validateAWS(),
		s.validateGCP(),
	)
}

func (s StorageOptions) validateAzure() error {
	settings := []storageSetting{
		{"AZ_ACCOUNT_NAME", s.AZAccountName},
		{"AZ_ACCOUNT_KEY", s.AZAccountKey},
		{"AZ_CONTAINER_NAME", s.AZContainerName},
	}
	if !anySettingPresent(append(settings, storageSetting{"AZ_ENDPOINT", s.AZEndpoint})) {
		return nil
	}

	return missingRequiredSettingsError("Azure", settings)
}

func (s StorageOptions) validateAWS() error {
	settings := []storageSetting{
		{"AWS_ACCESS_KEY_ID", s.AWSAccessKeyID},
		{"AWS_SECRET_ACCESS_KEY", s.AWSSecretAccessKey},
		{"AWS_BUCKET", s.AWSBucket},
	}
	intentSettings := append(settings, storageSetting{"AWS_ENDPOINT", s.AWSEndpoint})
	if s.AWSS3ForcePathStyle {
		intentSettings = append(intentSettings, storageSetting{"AWS_S3_FORCE_PATH_STYLE", "true"})
	}
	if !anySettingPresent(intentSettings) {
		return nil
	}

	return missingRequiredSettingsError("AWS", settings)
}

func (s StorageOptions) validateGCP() error {
	inlineSettings := []storageSetting{
		{"GCP_PROJECT_ID", s.GCPProjectID},
		{"GCP_PRIVATE_KEY_ID", s.GCPPrivateKeyID},
		{"GCP_PRIVATE_KEY", s.GCPPrivateKey},
		{"GCP_CLIENT_EMAIL", s.GCPClientEmail},
		{"GCP_CLIENT_ID", s.GCPClientID},
	}
	gcpSettings := []storageSetting{
		{"GCP_ENDPOINT", s.GCPEndpoint},
		{"GCP_BUCKET", s.GCPBucket},
		{"GCP_CREDS_FILE", s.GCPCredsFile},
	}
	gcpSettings = append(gcpSettings, inlineSettings...)
	if !anySettingPresent(gcpSettings) {
		return nil
	}
	if s.GCPBucket == "" {
		return missingRequiredSettingsError("GCP", []storageSetting{{"GCP_BUCKET", s.GCPBucket}})
	}

	hasEmulator := s.GCPEndpoint != ""
	hasCredsFile := s.GCPCredsFile != ""
	hasInline := anySettingPresent(inlineSettings)
	modeCount := 0
	if hasEmulator {
		modeCount++
	}
	if hasCredsFile {
		modeCount++
	}
	if hasInline {
		modeCount++
	}
	if modeCount > 1 {
		return errors.New("GCP storage configuration must use exactly one credential mode: GCP_ENDPOINT, GCP_CREDS_FILE, inline service account settings, or application default credentials")
	}
	if hasInline {
		return missingRequiredSettingsError("GCP", inlineSettings)
	}

	return nil
}

func anySettingPresent(settings []storageSetting) bool {
	for _, setting := range settings {
		if setting.value != "" {
			return true
		}
	}
	return false
}

func missingRequiredSettingsError(provider string, settings []storageSetting) error {
	missing := make([]string, 0)
	for _, setting := range settings {
		if setting.value == "" {
			missing = append(missing, setting.identifier)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("%s storage configuration missing required settings: %s", provider, strings.Join(missing, ", "))
}

func (s StorageOptions) getAzBlobStorage(expiryDays int) (*storage.AzBlob, error) {
	az := new(storage.AzBlob)
	if err := az.Init(s.AZAccountName, s.AZAccountKey, s.AZContainerName, s.AZEndpoint, expiryDays, s.BackupPrefix); err != nil {
		return nil, err
	}
	return az, nil
}

func (s StorageOptions) getAwsS3Storage(expiryDays int) (*storage.AwsS3, error) {
	s3 := new(storage.AwsS3)
	if err := s3.Init(s.AWSEndpoint, s.AWSAccessKeyID, s.AWSSecretAccessKey, s.AWSRegion, s.AWSBucket, s.AWSS3ForcePathStyle, expiryDays, s.BackupPrefix); err != nil {
		return nil, err
	}
	return s3, nil
}

func (s StorageOptions) getGcpStorage(ctx context.Context, expiryDays int) (*storage.GcpStorage, error) {
	gcpStorage := new(storage.GcpStorage)
	if err := gcpStorage.Init(ctx, s.GCPEndpoint, s.GCPBucket, s.GCPCredsFile, s.GCPProjectID, s.GCPPrivateKeyID, s.GCPPrivateKey, s.GCPClientEmail, s.GCPClientID, expiryDays, s.BackupPrefix); err != nil {
		return nil, err
	}
	return gcpStorage, nil
}

func (s StorageOptions) getLocalStorage(expiryDays int) (*storage.LocalStorage, error) {
	localStorage := new(storage.LocalStorage)
	if err := localStorage.Init(s.LocalPath, expiryDays, s.BackupPrefix); err != nil {
		return nil, err
	}
	return localStorage, nil
}

func (s StorageOptions) useAzure() bool {
	return s.AZAccountName != "" && s.AZAccountKey != "" && s.AZContainerName != ""
}

func (s StorageOptions) useAWS() bool {
	return s.AWSAccessKeyID != "" && s.AWSSecretAccessKey != "" && s.AWSBucket != ""
}

func (s StorageOptions) useGCP() bool {
	return s.GCPBucket != ""
}

func (s StorageOptions) useLocal() bool {
	return s.LocalPath != ""
}
