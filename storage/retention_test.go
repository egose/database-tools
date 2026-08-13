package storage

import (
	"errors"
	"reflect"
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
