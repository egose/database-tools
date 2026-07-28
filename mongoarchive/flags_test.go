package mongoarchive

import (
	"strings"
	"testing"
	"time"

	"github.com/egose/database-tools/internal/toolconfig"
	"github.com/egose/database-tools/storage"
)

func TestGetMongodumpOptionsPrunesURI(t *testing.T) {
	cfg := &Config{
		MongoOptions: toolconfig.MongoOptions{
			URI:      "mongodb://user:secret@host1:27017,host2:27018/testdb?replicaSet=rs0", // pragma: allowlist secret
			URIPrune: true,
		},
	}

	options := cfg.GetMongodumpOptions()
	joined := strings.Join(options, " ")

	if strings.Contains(joined, "secret") {
		t.Fatalf("GetMongodumpOptions() leaked credentials: %q", joined)
	}
	if !strings.Contains(joined, "--uri=mongodb://host1:27017,host2:27018/testdb?replicaSet=rs0") {
		t.Fatalf("GetMongodumpOptions() did not preserve pruned URI hosts/query: %q", joined)
	}
}

func TestConfigCronDefaults(t *testing.T) {
	cfg := &Config{}

	if got := cfg.GetCronExpression(); got != defaultCronExpr {
		t.Fatalf("GetCronExpression() = %q, want %q", got, defaultCronExpr)
	}
	if got := parseLocation(""); got != time.Local {
		t.Fatalf("parseLocation(\"\") = %v, want %v", got, time.Local)
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
