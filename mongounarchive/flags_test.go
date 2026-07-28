package mongounarchive

import (
	"strings"
	"testing"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
)

func TestGetMongounarchiveOptionsDoesNotDropByDefault(t *testing.T) {
	cfg := &Config{MongoOptions: toolconfig.MongoOptions{DB: "testdb"}, Drop: false}

	options := cfg.GetMongounarchiveOptions("/tmp/restore")
	for _, option := range options {
		if option == "--drop" {
			t.Fatalf("GetMongounarchiveOptions() unexpectedly included %q", option)
		}
	}
}

func TestGetMongoConnectionURIBuildsFromFlags(t *testing.T) {
	cfg := &Config{
		MongoOptions: toolconfig.MongoOptions{
			Host:                    "rs0/db1.example.com,db2.example.com",
			Port:                    "27018",
			Username:                "user",
			Password:                "secret",
			AuthenticationDatabase:  "admin",
			AuthenticationMechanism: "SCRAM-SHA-256",
			SSL:                     true,
			GSSAPIServiceName:       "mongodb",
		},
	}

	uri, err := cfg.MongoOptions.MongoConnectionURI()
	if err != nil {
		t.Fatalf("getMongoConnectionURI() returned error: %v", err)
	}

	wants := []string{
		"mongodb://user:secret@db1.example.com:27018,db2.example.com:27018/?", // pragma: allowlist secret
		"authMechanism=SCRAM-SHA-256",
		"authSource=admin",
		"replicaSet=rs0",
		"tls=true",
	}

	for _, want := range wants {
		if !strings.Contains(uri, want) {
			t.Fatalf("getMongoConnectionURI() = %q, missing %q", uri, want)
		}
	}
}

func TestGetStoragesUsesConfiguredLocalBackend(t *testing.T) {
	cfg := &Config{StorageOptions: toolconfig.StorageOptions{LocalPath: t.TempDir()}}

	storages := cfg.GetStorages()
	if len(storages) != 1 {
		t.Fatalf("GetStorages() len = %d, want 1", len(storages))
	}

	if _, ok := storages[0].(*storage.LocalStorage); !ok {
		t.Fatalf("GetStorages()[0] = %T, want *storage.LocalStorage", storages[0])
	}
}
