package storage

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestLatestObjectReturnsNewestCandidate(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	got, ok := latestObject([]objectTimestamp{
		{Name: "older", ModifiedAt: now.Add(-2 * time.Hour)},
		{Name: "newest", ModifiedAt: now},
		{Name: "middle", ModifiedAt: now.Add(-time.Hour)},
	})
	if !ok {
		t.Fatal("latestObject() ok = false, want true")
	}
	if got.Name != "newest" {
		t.Fatalf("latestObject() name = %q, want %q", got.Name, "newest")
	}
}

func TestLatestS3ObjectIgnoresNilEntries(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	got, ok := latestS3Object([]*s3.Object{
		nil,
		{Key: aws.String("missing-time")},
		{Key: aws.String("older"), LastModified: aws.Time(now.Add(-time.Hour))},
		{Key: aws.String("newest"), LastModified: aws.Time(now)},
	})
	if !ok {
		t.Fatal("latestS3Object() ok = false, want true")
	}
	if got.Name != "newest" {
		t.Fatalf("latestS3Object() name = %q, want %q", got.Name, "newest")
	}
}

func TestLatestGCPObjectUsesUpdatedTimestamp(t *testing.T) {
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	got, ok := latestGCPObject([]*gcs.ObjectAttrs{
		{Name: "older", Updated: now.Add(-2 * time.Hour)},
		{Name: "newest", Updated: now},
		{Name: "", Updated: now.Add(2 * time.Hour)},
	})
	if !ok {
		t.Fatal("latestGCPObject() ok = false, want true")
	}
	if got.Name != "newest" {
		t.Fatalf("latestGCPObject() name = %q, want %q", got.Name, "newest")
	}
}

func TestChooseLaterObjectUsesDeterministicNameTieBreak(t *testing.T) {
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	current := &objectTimestamp{Name: "mongo-archive/z.tar.gz", ModifiedAt: now}
	got := chooseLaterObject(current, objectTimestamp{Name: "mongo-archive/a.tar.gz", ModifiedAt: now})
	if got == nil || got.Name != "mongo-archive/a.tar.gz" {
		t.Fatalf("chooseLaterObject() = %#v, want candidate with lexicographically smaller name", got)
	}
}

func TestLatestEligibleObjectIgnoresMalformedAndOutOfScopeObjects(t *testing.T) {
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	got, ok := latestEligibleObject([]objectTimestamp{
		{Name: "other/9987654321000-2026-08-10T010203.456Z.tar.gz", ModifiedAt: now.Add(time.Hour)},
		{Name: "mongo-archive/not-a-backup.tar.gz", ModifiedAt: now.Add(2 * time.Hour)},
		{Name: "mongo-archive/9987654321000-2026-08-10T010203.456Z.tar.gz", ModifiedAt: now.Add(-time.Hour)},
		{Name: "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz", ModifiedAt: now},
	}, DefaultBackupPrefix)
	if !ok {
		t.Fatal("latestEligibleObject() ok = false, want true")
	}
	if got.Name != "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz" {
		t.Fatalf("latestEligibleObject() name = %q", got.Name)
	}
}

func TestLatestEligibleObjectKeepsDatabasePrefixesIsolated(t *testing.T) {
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	objects := []objectTimestamp{
		{Name: "mongo-archive/9987654321000-2026-08-24T100000.000Z.tar.gz", ModifiedAt: now.Add(-time.Hour)},
		{Name: "postgres-archive/9987654320999-2026-08-24T110000.000Z.tar.gz", ModifiedAt: now},
	}

	mongo, ok := latestEligibleObject(objects, DefaultBackupPrefix)
	if !ok || mongo.Name != objects[0].Name {
		t.Fatalf("MongoDB latest = %#v, %v; want %q", mongo, ok, objects[0].Name)
	}
	postgres, ok := latestEligibleObject(objects, "postgres-archive/")
	if !ok || postgres.Name != objects[1].Name {
		t.Fatalf("PostgreSQL latest = %#v, %v; want %q", postgres, ok, objects[1].Name)
	}
}

func TestResolveExplicitObjectNamePrefersManagedPrefixCandidate(t *testing.T) {
	var tried []string
	got, found, err := resolveExplicitObjectName(DefaultBackupPrefix, "9987654320999-2026-08-12T010203.456Z.tar.gz", func(candidate string) (bool, error) {
		tried = append(tried, candidate)
		return candidate == "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz", nil
	})
	if err != nil {
		t.Fatalf("resolveExplicitObjectName() error = %v", err)
	}
	if !found {
		t.Fatal("resolveExplicitObjectName() found = false, want true")
	}
	if got != "mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz" {
		t.Fatalf("resolveExplicitObjectName() = %q", got)
	}
	wantTried := []string{"mongo-archive/9987654320999-2026-08-12T010203.456Z.tar.gz"}
	if !reflect.DeepEqual(tried, wantTried) {
		t.Fatalf("resolveExplicitObjectName() tried = %#v, want %#v", tried, wantTried)
	}
}

func TestResolveExplicitObjectNameDoesNotFallBackToLatestSelection(t *testing.T) {
	var tried []string
	got, found, err := resolveExplicitObjectName(DefaultBackupPrefix, "missing.tar.gz", func(candidate string) (bool, error) {
		tried = append(tried, candidate)
		return false, nil
	})
	if err != nil {
		t.Fatalf("resolveExplicitObjectName() error = %v", err)
	}
	if found {
		t.Fatalf("resolveExplicitObjectName() found = true, want false with result %q", got)
	}
	wantTried := []string{"missing.tar.gz"}
	if !reflect.DeepEqual(tried, wantTried) {
		t.Fatalf("resolveExplicitObjectName() tried = %#v, want %#v", tried, wantTried)
	}
}

func TestResolveExplicitObjectNamePropagatesLookupErrors(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	_, _, err := resolveExplicitObjectName(DefaultBackupPrefix, "missing.tar.gz", func(string) (bool, error) {
		return false, lookupErr
	})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("resolveExplicitObjectName() error = %v, want wrapped %v", err, lookupErr)
	}
}

func TestSelectRestoreStorageRequiresExplicitBackendForMultipleStorages(t *testing.T) {
	_, err := SelectRestoreStorage([]RestoreBackend{&LocalStorage{}, &AwsS3{}}, "")
	if err == nil {
		t.Fatal("SelectRestoreStorage() expected error")
	}
	if err.Error() != "multiple storage backends configured (aws, local); specify --storage-backend" {
		t.Fatalf("SelectRestoreStorage() error = %q", err)
	}
}

func TestSelectRestoreStorageUsesRequestedBackend(t *testing.T) {
	storages := []RestoreBackend{&LocalStorage{}, &AwsS3{}}

	got, err := SelectRestoreStorage(storages, "aws")
	if err != nil {
		t.Fatalf("SelectRestoreStorage() error = %v", err)
	}
	if got != storages[1] {
		t.Fatalf("SelectRestoreStorage() = %T, want %T", got, storages[1])
	}
}

func TestSelectRestoreStorageRejectsUnknownBackend(t *testing.T) {
	_, err := SelectRestoreStorage([]RestoreBackend{&LocalStorage{}}, "gcp")
	if err == nil {
		t.Fatal("SelectRestoreStorage() expected error")
	}
	if err.Error() != "storage backend \"gcp\" is not configured; available backends: local" {
		t.Fatalf("SelectRestoreStorage() error = %q", err)
	}
}

type namedRestoreStorage struct {
	name string
}

func (s namedRestoreStorage) BackendName() string { return s.name }

func (s namedRestoreStorage) Download(context.Context, string, string) error { return nil }

func (s namedRestoreStorage) GetTargetObjectName(context.Context, string) (string, error) {
	return "", nil
}

func (s namedRestoreStorage) Close() error { return nil }

type restoreStorageWrapper struct {
	RestoreBackend
}

func TestSelectRestoreStorageUsesExplicitBackendIdentityFromWrapper(t *testing.T) {
	wrapped := restoreStorageWrapper{RestoreBackend: namedRestoreStorage{name: "custom"}}

	got, err := SelectRestoreStorage([]RestoreBackend{wrapped}, "custom")
	if err != nil {
		t.Fatalf("SelectRestoreStorage() error = %v", err)
	}
	if got != wrapped {
		t.Fatalf("SelectRestoreStorage() = %T, want wrapper", got)
	}
}

func TestSelectRestoreStorageRejectsDuplicateBackendNames(t *testing.T) {
	_, err := SelectRestoreStorage([]RestoreBackend{
		namedRestoreStorage{name: "local"},
		namedRestoreStorage{name: " LOCAL "},
	}, "local")
	if err == nil {
		t.Fatal("SelectRestoreStorage() expected error")
	}
	if err.Error() != "duplicate storage backend name \"local\"" {
		t.Fatalf("SelectRestoreStorage() error = %q", err)
	}
}
