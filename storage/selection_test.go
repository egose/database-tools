package storage

import (
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
