package toolconfig

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/egose/database-tools/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
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

func BindMongoFlags(env interface {
	GetValue(string, ...string) string
}) MongoFlagBindings {
	return MongoFlagBindings{
		Verbose:                     flag.String("verbose", env.GetValue("VERBOSE"), "more detailed log output (include multiple times for more verbosity, e.g. -vvvvv, or specify a numeric value, e.g. --verbose=N)"),
		Quiet:                       flag.Bool("quiet", env.GetValue("QUIET") == "true", "hide all log output"),
		Host:                        flag.String("host", env.GetValue("HOST"), "MongoDB host to connect to (setname/host1,host2 for replica sets)"),
		Port:                        flag.String("port", env.GetValue("PORT"), "MongoDB port (can also use --host hostname:port)"),
		SSL:                         flag.Bool("ssl", env.GetValue("SSL") == "true", "connect to a mongod or mongos that has ssl enabled"),
		SSLCAFile:                   flag.String("ssl-ca-file", env.GetValue("SSL_CA_FILE"), "the .pem file containing the root certificate chain from the certificate authority"),
		SSLPEMKeyFile:               flag.String("ssl-pem-key-file", env.GetValue("SSL_PEM_KEY_FILE"), "the .pem file containing the certificate and key"),
		SSLPEMKeyPassword:           flag.String("ssl-pem-key-password", env.GetValue("SSL_PEM_KEY_PASSWORD"), "the password to decrypt the sslPEMKeyFile, if necessary"),
		SSLCRLFile:                  flag.String("ssl-crl-file", env.GetValue("SSL_CRL_FILE"), "the .pem file containing the certificate revocation list"),
		SSLAllowInvalidCertificates: flag.Bool("ssl-allow-invalid-certificates", env.GetValue("SSL_ALLOW_INVALID_CERTIFICATES") == "true", "bypass the validation for server certificates"),
		SSLAllowInvalidHostnames:    flag.Bool("ssl-allow-invalid-hostnames", env.GetValue("SSL_ALLOW_INVALID_HOSTNAMES") == "true", "bypass the validation for server name"),
		SSLFIPSMode:                 flag.Bool("ssl-fips-mode", env.GetValue("SSL_FIPS_MODE") == "true", "use FIPS mode of the installed openssl library"),
		Username:                    flag.String("username", env.GetValue("USERNAME"), "username for authentication"),
		Password:                    flag.String("password", env.GetValue("PASSWORD"), "password for authentication"),
		AuthenticationDatabase:      flag.String("authentication-database", env.GetValue("AUTHENTICATION_DATABASE"), "database that holds the user's credentials"),
		AuthenticationMechanism:     flag.String("authentication-mechanism", env.GetValue("AUTHENTICATION_MECHANISM"), "authentication mechanism to use"),
		GSSAPIServiceName:           flag.String("gssapi-service-name", env.GetValue("GSSAPI_SERVICE_NAME"), "service name to use when authenticating using GSSAPI/Kerberos (default: mongodb)"),
		GSSAPIHostName:              flag.String("gssapi-host-name", env.GetValue("GSSAPI_HOST_NAME"), "hostname to use when authenticating using GSSAPI/Kerberos (default: <remote server's address>)"),
		DB:                          flag.String("db", env.GetValue("DB"), "database to use"),
		Collection:                  flag.String("collection", env.GetValue("COLLECTION"), "collection to use"),
		URI:                         flag.String("uri", env.GetValue("URI"), "MongoDB uri connection string"),
		URIPrune:                    flag.Bool("uri-prune", env.GetValue("URI_PRUNE") == "true", "prune MongoDB uri connection string"),
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
}

func BindStorageFlags(env interface {
	GetValue(string, ...string) string
}) StorageFlagBindings {
	return StorageFlagBindings{
		AZEndpoint:          flag.String("az-endpoint", env.GetValue("AZ_ENDPOINT", ""), "specify the emulator hostname and Azure Blob Storage port"),
		AZAccountName:       flag.String("az-account-name", env.GetValue("AZ_ACCOUNT_NAME"), "Azure Blob Storage Account Name"),
		AZAccountKey:        flag.String("az-account-key", env.GetValue("AZ_ACCOUNT_KEY"), "Azure Blob Storage Account Key"),
		AZContainerName:     flag.String("az-container-name", env.GetValue("AZ_CONTAINER_NAME"), "Azure Blob Storage Container Name"),
		AWSEndpoint:         flag.String("aws-endpoint", env.GetValue("AWS_ENDPOINT", ""), "AWS endpoint URL (hostname only or fully qualified URI)"),
		AWSAccessKeyID:      flag.String("aws-access-key-id", env.GetValue("AWS_ACCESS_KEY_ID"), "AWS access key associated with an IAM account"),
		AWSSecretAccessKey:  flag.String("aws-secret-access-key", env.GetValue("AWS_SECRET_ACCESS_KEY"), "AWS secret key associated with the access key"),
		AWSRegion:           flag.String("aws-region", env.GetValue("AWS_REGION", "us-east-1"), "AWS Region whose servers you want to send your requests to"),
		AWSBucket:           flag.String("aws-bucket", env.GetValue("AWS_BUCKET"), "AWS S3 bucket name"),
		AWSS3ForcePathStyle: flag.Bool("aws-s3-force-path-style", env.GetValue("AWS_S3_FORCE_PATH_STYLE") == "true", "force the request to use path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`. By default, the S3 client will use virtual hosted bucket addressing when possible (`http://BUCKET.s3.amazonaws.com/KEY`)"),
		GCPEndpoint:         flag.String("gcp-endpoint", env.GetValue("GCP_ENDPOINT", ""), "GCP endpoint URL"),
		GCPBucket:           flag.String("gcp-bucket", env.GetValue("GCP_BUCKET"), "GCP storage bucket name"),
		GCPCredsFile:        flag.String("gcp-creds-file", env.GetValue("GCP_CREDS_FILE"), "GCP service account's credentials file"),
		GCPProjectID:        flag.String("gcp-project-id", env.GetValue("GCP_PROJECT_ID"), "GCP service account's project id"),
		GCPPrivateKeyID:     flag.String("gcp-private-key-id", env.GetValue("GCP_PRIVATE_KEY_ID"), "GCP service account's private key id"),
		GCPPrivateKey:       flag.String("gcp-private-key", env.GetValue("GCP_PRIVATE_KEY"), "GCP service account's private key"),
		GCPClientEmail:      flag.String("gcp-client-email", env.GetValue("GCP_CLIENT_EMAIL"), "GCP service account's client email"),
		GCPClientID:         flag.String("gcp-client-id", env.GetValue("GCP_CLIENT_ID"), "GCP service account's client id"),
		LocalPath:           flag.String("local-path", env.GetValue("LOCAL_PATH"), "Local directory path to store backups"),
	}
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
}

func (s StorageOptions) GetStorages(expiryDays int) []storage.Storage {
	type option struct {
		name    string
		enabled func() bool
		getter  func() (storage.Storage, error)
	}

	options := []option{
		{"Local", s.useLocal, func() (storage.Storage, error) { return s.getLocalStorage(expiryDays) }},
		{"Azure", s.useAzure, func() (storage.Storage, error) { return s.getAzBlobStorage(expiryDays) }},
		{"AWS", s.useAWS, func() (storage.Storage, error) { return s.getAwsS3Storage(expiryDays) }},
		{"GCP", s.useGCP, func() (storage.Storage, error) { return s.getGcpStorage(expiryDays) }},
	}

	storages := make([]storage.Storage, 0)
	for _, opt := range options {
		if opt.enabled() {
			storageBackend, err := opt.getter()
			if err != nil {
				mlog.Logvf(mlog.Always, "Failed to initialize %v storage: %v", opt.name, err)
				continue
			}
			if storageBackend != nil {
				mlog.Logvf(mlog.Always, "Found Storage Option: %v", opt.name)
				storages = append(storages, storageBackend)
			}
		}
	}

	return storages
}

func (s StorageOptions) getAzBlobStorage(expiryDays int) (storage.Storage, error) {
	az := new(storage.AzBlob)
	if err := az.Init(s.AZAccountName, s.AZAccountKey, s.AZContainerName, s.AZEndpoint, expiryDays); err != nil {
		return nil, err
	}
	return az, nil
}

func (s StorageOptions) getAwsS3Storage(expiryDays int) (storage.Storage, error) {
	s3 := new(storage.AwsS3)
	if err := s3.Init(s.AWSEndpoint, s.AWSAccessKeyID, s.AWSSecretAccessKey, s.AWSRegion, s.AWSBucket, s.AWSS3ForcePathStyle, expiryDays); err != nil {
		return nil, err
	}
	return s3, nil
}

func (s StorageOptions) getGcpStorage(expiryDays int) (storage.Storage, error) {
	gcpStorage := new(storage.GcpStorage)
	if err := gcpStorage.Init(s.GCPEndpoint, s.GCPBucket, s.GCPCredsFile, s.GCPProjectID, s.GCPPrivateKeyID, s.GCPPrivateKey, s.GCPClientEmail, s.GCPClientID, expiryDays); err != nil {
		return nil, err
	}
	return gcpStorage, nil
}

func (s StorageOptions) getLocalStorage(expiryDays int) (storage.Storage, error) {
	localStorage := new(storage.LocalStorage)
	if err := localStorage.Init(s.LocalPath, expiryDays); err != nil {
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
