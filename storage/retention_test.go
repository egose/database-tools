package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		modifiedAt time.Time
		expiryDays int
		want       bool
	}{
		{name: "disabled retention", modifiedAt: now.Add(-48 * time.Hour), expiryDays: 0, want: false},
		{name: "within expiry", modifiedAt: now.Add(-23 * time.Hour), expiryDays: 1, want: false},
		{name: "older than expiry", modifiedAt: now.Add(-25 * time.Hour), expiryDays: 1, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpired(tt.modifiedAt, tt.expiryDays, now); got != tt.want {
				t.Fatalf("isExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeBackupPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "default", prefix: "", want: DefaultBackupPrefix},
		{name: "trim slashes", prefix: "/custom/prefix/", want: "custom/prefix/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeBackupPrefix(tt.prefix); got != tt.want {
				t.Fatalf("NormalizeBackupPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeBackupPrefixWithDatabaseDefault(t *testing.T) {
	tests := []struct {
		name          string
		prefix        string
		defaultPrefix string
		want          string
	}{
		{name: "MongoDB", defaultPrefix: MongoDefaultBackupPrefix, want: "mongo-archive/"},
		{name: "PostgreSQL", defaultPrefix: PostgreSQLDefaultBackupPrefix, want: "postgres-archive/"},
		{name: "normalized custom overrides MongoDB", prefix: " /tenant/mongo/ ", defaultPrefix: MongoDefaultBackupPrefix, want: "tenant/mongo/"},
		{name: "normalized custom overrides PostgreSQL", prefix: " /tenant/postgres/ ", defaultPrefix: PostgreSQLDefaultBackupPrefix, want: "tenant/postgres/"},
		{name: "empty custom default preserves MongoDB compatibility", want: DefaultBackupPrefix},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeBackupPrefixWithDefault(tt.prefix, tt.defaultPrefix); got != tt.want {
				t.Fatalf("NormalizeBackupPrefixWithDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMongoDefaultBackupObjectNameRemainsCompatible(t *testing.T) {
	if DefaultBackupPrefix != "mongo-archive/" {
		t.Fatalf("DefaultBackupPrefix = %q, want %q", DefaultBackupPrefix, "mongo-archive/")
	}
	got, err := BuildBackupObjectName("", "9987654321000-2026-08-24T010203.456Z.tar.gz")
	if err != nil {
		t.Fatalf("BuildBackupObjectName() error = %v", err)
	}
	if got != "mongo-archive/9987654321000-2026-08-24T010203.456Z.tar.gz" {
		t.Fatalf("BuildBackupObjectName() = %q", got)
	}
}

func TestBuildBackupObjectName(t *testing.T) {
	got, err := BuildBackupObjectName("custom", "9987654321000-2026-08-12T010203.456Z.tar.gz")
	if err != nil {
		t.Fatalf("BuildBackupObjectName() error = %v", err)
	}
	if got != "custom/9987654321000-2026-08-12T010203.456Z.tar.gz" {
		t.Fatalf("BuildBackupObjectName() = %q", got)
	}

	if _, err := BuildBackupObjectName("custom", "not-a-backup.tar.gz"); err == nil {
		t.Fatal("BuildBackupObjectName() expected contract error")
	}
}

func TestBuildBackupObjectNameUsesDatabaseDefault(t *testing.T) {
	filename := "9987654321000-2026-08-12T010203.456Z.tar.gz"
	got, err := BuildBackupObjectNameWithDefault("", filename, PostgreSQLDefaultBackupPrefix)
	if err != nil {
		t.Fatalf("BuildBackupObjectNameWithDefault() error = %v", err)
	}
	if got != "postgres-archive/"+filename {
		t.Fatalf("BuildBackupObjectNameWithDefault() = %q", got)
	}
}

func TestLookupObjectCandidatesSupportsPrefixedAndLegacyLookups(t *testing.T) {
	got := lookupObjectCandidates("custom", "9987654321000-2026-08-12T010203.456Z.tar.gz")
	want := []string{
		"custom/9987654321000-2026-08-12T010203.456Z.tar.gz",
		"9987654321000-2026-08-12T010203.456Z.tar.gz",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lookupObjectCandidates() = %#v, want %#v", got, want)
	}
}

func TestDeleteExpiredObjectsOnlyDeletesEligibleScopedBackups(t *testing.T) {
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	candidates := []objectTimestamp{
		{Name: "outside/9987654321000-2026-08-10T010203.456Z.tar.gz", ModifiedAt: now.Add(-72 * time.Hour)},
		{Name: "custom/not-a-backup.tar.gz", ModifiedAt: now.Add(-72 * time.Hour)},
		{Name: "custom/9987654321000-2026-08-10T010203.456Z.tar.gz", ModifiedAt: now.Add(-72 * time.Hour)},
		{Name: "custom/9987654320999-2026-08-12T010203.456Z.tar.gz", ModifiedAt: now.Add(-time.Hour)},
	}

	deleted := make([]string, 0, 1)
	err := deleteExpiredObjects(candidates, "custom", 1, now, "custom/9987654320999-2026-08-12T010203.456Z.tar.gz", func(name string) error {
		deleted = append(deleted, name)
		return nil
	})
	if err != nil {
		t.Fatalf("deleteExpiredObjects() error = %v", err)
	}

	want := []string{"custom/9987654321000-2026-08-10T010203.456Z.tar.gz"}
	if !reflect.DeepEqual(deleted, want) {
		t.Fatalf("deleteExpiredObjects() deleted = %#v, want %#v", deleted, want)
	}
}

func TestDeleteExpiredObjectsKeepsDatabasePrefixesIsolated(t *testing.T) {
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	objects := []objectTimestamp{
		{Name: "mongo-archive/9987654321000-2026-08-20T010203.456Z.tar.gz", ModifiedAt: now.Add(-96 * time.Hour)},
		{Name: "postgres-archive/9987654320999-2026-08-20T010203.456Z.tar.gz", ModifiedAt: now.Add(-96 * time.Hour)},
	}

	for _, prefix := range []string{"mongo-archive/", "postgres-archive/"} {
		t.Run(prefix, func(t *testing.T) {
			var deleted []string
			if err := deleteExpiredObjects(objects, prefix, 1, now, "", func(name string) error {
				deleted = append(deleted, name)
				return nil
			}); err != nil {
				t.Fatalf("deleteExpiredObjects() error = %v", err)
			}
			if len(deleted) != 1 || !strings.HasPrefix(deleted[0], prefix) {
				t.Fatalf("deleted = %v, want only prefix %q", deleted, prefix)
			}
		})
	}
}

func TestDeleteExpiredObjectsReturnsDeletionFailure(t *testing.T) {
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	deleteErr := errors.New("boom")
	err := deleteExpiredObjects([]objectTimestamp{{
		Name:       "custom/9987654321000-2026-08-10T010203.456Z.tar.gz",
		ModifiedAt: now.Add(-72 * time.Hour),
	}}, "custom", 1, now, "", func(string) error {
		return deleteErr
	})
	if !errors.Is(err, deleteErr) {
		t.Fatalf("deleteExpiredObjects() error = %v, want wrapped %v", err, deleteErr)
	}
}
