package toolconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/storage"
)

func TestBindStorageFlagsUsesDatabaseSpecificDefaultPrefix(t *testing.T) {
	tests := []struct {
		name          string
		defaultPrefix string
		compatibility bool
		want          string
	}{
		{name: "MongoDB compatibility wrapper", compatibility: true, want: storage.MongoDefaultBackupPrefix},
		{name: "MongoDB explicit default", defaultPrefix: storage.MongoDefaultBackupPrefix, want: storage.MongoDefaultBackupPrefix},
		{name: "PostgreSQL explicit default", defaultPrefix: storage.PostgreSQLDefaultBackupPrefix, want: storage.PostgreSQLDefaultBackupPrefix},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			var bindings StorageFlagBindings
			if tt.compatibility {
				bindings = BindStorageFlags(fs, testEnv{})
			} else {
				bindings = BindStorageFlagsWithDefaultPrefix(fs, testEnv{}, tt.defaultPrefix)
			}

			options := StorageOptions{}
			bindings.Apply(&options)
			if options.BackupPrefix != tt.want {
				t.Fatalf("BackupPrefix = %q, want %q", options.BackupPrefix, tt.want)
			}
		})
	}
}

func TestBindStorageFlagsKeepsCustomPrefixForBothDatabaseFamilies(t *testing.T) {
	for _, defaultPrefix := range []string{storage.MongoDefaultBackupPrefix, storage.PostgreSQLDefaultBackupPrefix} {
		t.Run(defaultPrefix, func(t *testing.T) {
			fs := flag.NewFlagSet(defaultPrefix, flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			bindings := BindStorageFlagsWithDefaultPrefix(fs, testEnv{
				"BACKUP_PREFIX": "/shared/custom/",
				"LOCAL_PATH":    t.TempDir(),
			}, defaultPrefix)
			options := StorageOptions{}
			bindings.Apply(&options)

			storages, err := options.GetArchiveStorages(context.Background(), 0)
			if err != nil {
				t.Fatalf("GetArchiveStorages() error = %v", err)
			}
			if len(storages) != 1 {
				t.Fatalf("GetArchiveStorages() len = %d, want 1", len(storages))
			}
			local, ok := storages[0].(*storage.LocalStorage)
			if !ok {
				t.Fatalf("GetArchiveStorages()[0] = %T, want *storage.LocalStorage", storages[0])
			}
			if local.BackupPrefix != "shared/custom/" {
				t.Fatalf("BackupPrefix = %q, want %q", local.BackupPrefix, "shared/custom/")
			}
		})
	}
}

func TestGetStoragesReturnsConfiguredLocalBackend(t *testing.T) {
	options := StorageOptions{LocalPath: t.TempDir()}

	storages, err := options.GetArchiveStorages(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetArchiveStorages() error = %v", err)
	}
	if len(storages) != 1 {
		t.Fatalf("GetArchiveStorages() len = %d, want 1", len(storages))
	}
	if _, ok := storages[0].(*storage.LocalStorage); !ok {
		t.Fatalf("GetArchiveStorages()[0] = %T, want *storage.LocalStorage", storages[0])
	}
}

func TestGetStoragesFailsClosedOnMixedValidAndInvalidBackends(t *testing.T) {
	options := StorageOptions{
		LocalPath:    t.TempDir(),
		GCPBucket:    "test-bucket",
		GCPCredsFile: "/tmp/does-not-exist.json",
		BackupPrefix: storage.DefaultBackupPrefix,
	}

	storages, err := options.GetArchiveStorages(context.Background(), 0)
	if err == nil {
		t.Fatal("GetArchiveStorages() expected error")
	}
	if len(storages) != 0 {
		t.Fatalf("GetArchiveStorages() len = %d, want 0 on fail-closed init", len(storages))
	}
	if !strings.Contains(err.Error(), "GCP storage initialization failed") {
		t.Fatalf("GetArchiveStorages() error = %q, want GCP init failure", err)
	}
}

func TestStorageOptionsValidateRejectsPartialAWSAndAzureRequiredSettings(t *testing.T) {
	tests := []struct {
		name   string
		fields []storageOptionField
	}{
		{
			name: "AWS",
			fields: []storageOptionField{
				{identifier: "AWS_ACCESS_KEY_ID", value: "aws-value-0", set: func(options *StorageOptions, value string) { options.AWSAccessKeyID = value }},
				{identifier: "AWS_SECRET_ACCESS_KEY", value: "aws-value-1", set: func(options *StorageOptions, value string) { options.AWSSecretAccessKey = value }},
				{identifier: "AWS_BUCKET", value: "aws-value-2", set: func(options *StorageOptions, value string) { options.AWSBucket = value }},
			},
		},
		{
			name: "Azure",
			fields: []storageOptionField{
				{identifier: "AZ_ACCOUNT_NAME", value: "azure-value-0", set: func(options *StorageOptions, value string) { options.AZAccountName = value }},
				{identifier: "AZ_ACCOUNT_KEY", value: "azure-value-1", set: func(options *StorageOptions, value string) { options.AZAccountKey = value }},
				{identifier: "AZ_CONTAINER_NAME", value: "azure-value-2", set: func(options *StorageOptions, value string) { options.AZContainerName = value }},
			},
		},
	}

	for _, tt := range tests {
		for mask := 1; mask < (1<<len(tt.fields))-1; mask++ {
			t.Run(tt.name, func(t *testing.T) {
				options := StorageOptions{}
				missing := make([]string, 0)
				suppliedValues := make([]string, 0)
				for i, field := range tt.fields {
					if mask&(1<<i) != 0 {
						field.set(&options, field.value)
						suppliedValues = append(suppliedValues, field.value)
					} else {
						missing = append(missing, field.identifier)
					}
				}

				err := options.Validate()
				if err == nil {
					t.Fatal("Validate() expected error")
				}
				assertErrorContainsAll(t, err, missing)
				assertErrorOmitsAll(t, err, suppliedValues)
			})
		}
	}
}

func TestStorageOptionsValidateKeepsAbsentAndCompleteProvidersEnabled(t *testing.T) {
	tests := []struct {
		name    string
		options StorageOptions
	}{
		{name: "absent"},
		{name: "local", options: StorageOptions{LocalPath: t.TempDir()}},
		{name: "AWS", options: StorageOptions{AWSAccessKeyID: "aws-id", AWSSecretAccessKey: "aws-key", AWSBucket: "aws-bucket"}},
		{name: "Azure", options: StorageOptions{AZAccountName: "az-account", AZAccountKey: "az-key", AZContainerName: "az-container"}},
		{name: "GCP emulator", options: StorageOptions{GCPEndpoint: "http://127.0.0.1:8080", GCPBucket: "gcp-bucket"}},
		{name: "GCP credentials file", options: StorageOptions{GCPBucket: "gcp-bucket", GCPCredsFile: "/tmp/credentials.json"}},
		{name: "GCP inline service account", options: StorageOptions{GCPBucket: "gcp-bucket", GCPProjectID: "project", GCPPrivateKeyID: "key-id", GCPPrivateKey: "key", GCPClientEmail: "client@example.com", GCPClientID: "client-id"}},
		{name: "GCP application default credentials", options: StorageOptions{GCPBucket: "gcp-bucket"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.options.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestStorageOptionsValidateRejectsPartialGCPCredentialModes(t *testing.T) {
	t.Run("missing bucket", func(t *testing.T) {
		options := StorageOptions{GCPCredsFile: "gcp-value-file"}

		err := options.Validate()
		if err == nil {
			t.Fatal("Validate() expected error")
		}
		assertErrorContainsAll(t, err, []string{"GCP_BUCKET"})
		assertErrorOmitsAll(t, err, []string{"gcp-value-file"})
	})

	inlineFields := []storageOptionField{
		{identifier: "GCP_PROJECT_ID", value: "gcp-value-0", set: func(options *StorageOptions, value string) { options.GCPProjectID = value }},
		{identifier: "GCP_PRIVATE_KEY_ID", value: "gcp-value-1", set: func(options *StorageOptions, value string) { options.GCPPrivateKeyID = value }},
		{identifier: "GCP_PRIVATE_KEY", value: "gcp-value-2", set: func(options *StorageOptions, value string) { options.GCPPrivateKey = value }},
		{identifier: "GCP_CLIENT_EMAIL", value: "gcp-value-3", set: func(options *StorageOptions, value string) { options.GCPClientEmail = value }},
		{identifier: "GCP_CLIENT_ID", value: "gcp-value-4", set: func(options *StorageOptions, value string) { options.GCPClientID = value }},
	}

	for mask := 1; mask < (1<<len(inlineFields))-1; mask++ {
		t.Run("partial inline service account", func(t *testing.T) {
			options := StorageOptions{GCPBucket: "gcp-bucket"}
			missing := make([]string, 0)
			suppliedValues := []string{options.GCPBucket}
			for i, field := range inlineFields {
				if mask&(1<<i) != 0 {
					field.set(&options, field.value)
					suppliedValues = append(suppliedValues, field.value)
				} else {
					missing = append(missing, field.identifier)
				}
			}

			err := options.Validate()
			if err == nil {
				t.Fatal("Validate() expected error")
			}
			assertErrorContainsAll(t, err, missing)
			assertErrorOmitsAll(t, err, suppliedValues)
		})
	}

	t.Run("mixed credential modes", func(t *testing.T) {
		options := StorageOptions{GCPBucket: "gcp-bucket", GCPEndpoint: "http://127.0.0.1:8080", GCPCredsFile: "gcp-value-file"}

		err := options.Validate()
		if err == nil {
			t.Fatal("Validate() expected error")
		}
		assertErrorContainsAll(t, err, []string{"GCP_ENDPOINT", "GCP_CREDS_FILE"})
		assertErrorOmitsAll(t, err, []string{options.GCPBucket, options.GCPEndpoint, options.GCPCredsFile})
	})
}

type storageOptionField struct {
	identifier string
	value      string
	set        func(*StorageOptions, string)
}

func assertErrorContainsAll(t *testing.T, err error, values []string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(err.Error(), value) {
			t.Fatalf("Validate() error = %q, missing %q", err, value)
		}
	}
}

func assertErrorOmitsAll(t *testing.T, err error, values []string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(err.Error(), value) {
			t.Fatalf("Validate() error = %q, leaked supplied value %q", err, value)
		}
	}
}

func TestMongoClientOptionsPreservesTLSAndAuthSettings(t *testing.T) {
	testCAFile, testClientPEMFile, password := writeMongoTLSFiles(t)

	options, err := (MongoOptions{
		Host:                    "localhost",
		Port:                    "27017",
		Username:                "user",
		Password:                "secret",
		AuthenticationDatabase:  "admin",
		AuthenticationMechanism: "SCRAM-SHA-256",
		SSL:                     true,
		SSLCAFile:               testCAFile,
		SSLPEMKeyFile:           testClientPEMFile,
		SSLPEMKeyPassword:       password,
	}).MongoClientOptions()
	if err != nil {
		t.Fatalf("MongoClientOptions() error = %v", err)
	}

	if options.Auth == nil {
		t.Fatal("MongoClientOptions() missing auth configuration")
	}
	if options.Auth.Username != "user" {
		t.Fatalf("MongoClientOptions() username = %q, want user", options.Auth.Username)
	}
	if options.Auth.Password != "secret" { // pragma: allowlist secret
		t.Fatalf("MongoClientOptions() password = %q, want secret", options.Auth.Password)
	}
	if options.Auth.AuthSource != "admin" {
		t.Fatalf("MongoClientOptions() auth source = %q, want admin", options.Auth.AuthSource)
	}
	if options.Auth.AuthMechanism != "SCRAM-SHA-256" {
		t.Fatalf("MongoClientOptions() auth mechanism = %q, want SCRAM-SHA-256", options.Auth.AuthMechanism)
	}
	if options.TLSConfig == nil {
		t.Fatal("MongoClientOptions() missing TLS config")
	}
	if options.TLSConfig.RootCAs == nil {
		t.Fatal("MongoClientOptions() missing CA pool")
	}
	if len(options.TLSConfig.Certificates) != 1 {
		t.Fatalf("MongoClientOptions() certificate count = %d, want 1", len(options.TLSConfig.Certificates))
	}
}

func TestMongoClientOptionsRejectsUnsupportedCRLAndFIPSOptions(t *testing.T) {
	t.Run("crl", func(t *testing.T) {
		_, err := (MongoOptions{SSL: true, SSLCRLFile: "/tmp/test.crl"}).MongoClientOptions()
		if err == nil || !strings.Contains(err.Error(), "ssl-crl-file") {
			t.Fatalf("MongoClientOptions() error = %v, want ssl-crl-file rejection", err)
		}
	})

	t.Run("fips", func(t *testing.T) {
		_, err := (MongoOptions{SSL: true, SSLFIPSMode: true}).MongoClientOptions()
		if err == nil || !strings.Contains(err.Error(), "ssl-fips-mode") {
			t.Fatalf("MongoClientOptions() error = %v, want ssl-fips-mode rejection", err)
		}
	})
}

func writeMongoTLSFiles(t *testing.T) (string, string, string) {
	t.Helper()

	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(ca) error = %v", err)
	}
	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(client) error = %v", err)
	}

	now := time.Now().UTC()
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		t.Fatalf("CreateCertificate(ca) error = %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate(ca) error = %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "client"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		t.Fatalf("CreateCertificate(client) error = %v", err)
	}

	password := "pem-password" // pragma: allowlist secret
	clientKeyBlock, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(clientPrivateKey), []byte(password), x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("EncryptPEMBlock(client key) error = %v", err)
	}

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	clientPEMFile := filepath.Join(t.TempDir(), "client.pem")
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0o600); err != nil {
		t.Fatalf("WriteFile(ca) error = %v", err)
	}
	clientPEM := append(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER}),
		pem.EncodeToMemory(clientKeyBlock)...,
	)
	if err := os.WriteFile(clientPEMFile, clientPEM, 0o600); err != nil {
		t.Fatalf("WriteFile(client) error = %v", err)
	}

	return caFile, clientPEMFile, password
}
