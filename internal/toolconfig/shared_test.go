package toolconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/storage"
)

func TestGetStoragesReturnsConfiguredLocalBackend(t *testing.T) {
	options := StorageOptions{LocalPath: t.TempDir()}

	storages, err := options.GetStorages(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetStorages() error = %v", err)
	}
	if len(storages) != 1 {
		t.Fatalf("GetStorages() len = %d, want 1", len(storages))
	}
	if _, ok := storages[0].(*storage.LocalStorage); !ok {
		t.Fatalf("GetStorages()[0] = %T, want *storage.LocalStorage", storages[0])
	}
}

func TestGetStoragesFailsClosedOnMixedValidAndInvalidBackends(t *testing.T) {
	options := StorageOptions{
		LocalPath:    t.TempDir(),
		GCPBucket:    "test-bucket",
		GCPCredsFile: "/tmp/does-not-exist.json",
		BackupPrefix: storage.DefaultBackupPrefix,
	}

	storages, err := options.GetStorages(context.Background(), 0)
	if err == nil {
		t.Fatal("GetStorages() expected error")
	}
	if len(storages) != 0 {
		t.Fatalf("GetStorages() len = %d, want 0 on fail-closed init", len(storages))
	}
	if !strings.Contains(err.Error(), "GCP storage initialization failed") {
		t.Fatalf("GetStorages() error = %q, want GCP init failure", err)
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
