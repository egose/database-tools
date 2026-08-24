package archivedelivery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egose/database-tools/storage"
)

type recordingBackend struct {
	mu        sync.Mutex
	name      string
	calls     *[]string
	uploadErr error
	deleteErr error
	closes    int
}

func (b *recordingBackend) Upload(_ context.Context, objectName string, _ string) (string, error) {
	b.record("upload:" + objectName)
	return "verified", b.uploadErr
}

func (b *recordingBackend) DeleteOldObjects(_ context.Context, objectName string) error {
	b.record("delete:" + objectName)
	return b.deleteErr
}

func (b *recordingBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closes++
	return nil
}

func (b *recordingBackend) BackendName() string { return b.name }

func (b *recordingBackend) record(call string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	*b.calls = append(*b.calls, b.name+":"+call)
}

func (b *recordingBackend) closeCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closes
}

func TestDeliverAttemptsAllUploadsBeforeRetention(t *testing.T) {
	firstErr := errors.New("first upload failed")
	lastErr := errors.New("last upload failed")
	tests := []struct {
		name            string
		uploadErrors    []error
		wantCalls       []string
		wantError       error
		wantErrorParts  []string
		wantLogMessages []string
	}{
		{
			name:         "all uploads and retention succeed",
			uploadErrors: []error{nil, nil},
			wantCalls: []string{
				"local:upload:archive.tar.gz", "aws:upload:archive.tar.gz",
				"local:delete:archive.tar.gz", "aws:delete:archive.tar.gz",
			},
			wantLogMessages: []string{
				"Successfully uploaded backup to backend #1 (local): verified",
				"Successfully uploaded backup to backend #2 (aws): verified",
				"Verified archive upload across 2 storage backends; starting retention.",
			},
		},
		{
			name:         "first upload fails but later uploads are attempted",
			uploadErrors: []error{firstErr, nil},
			wantCalls: []string{
				"local:upload:archive.tar.gz", "aws:upload:archive.tar.gz",
			},
			wantError:      firstErr,
			wantErrorParts: []string{"after successful uploads to backend #2 (aws)", "retention was not run on any backend", "backend #1 (local)"},
			wantLogMessages: []string{
				"Successfully uploaded backup to backend #2 (aws): verified",
			},
		},
		{
			name:         "later upload fails with explicit partial state",
			uploadErrors: []error{nil, lastErr},
			wantCalls: []string{
				"local:upload:archive.tar.gz", "aws:upload:archive.tar.gz",
			},
			wantError:      lastErr,
			wantErrorParts: []string{"after successful uploads to backend #1 (local)", "retention was not run on any backend", "backend #2 (aws)"},
			wantLogMessages: []string{
				"Successfully uploaded backup to backend #1 (local): verified",
			},
		},
		{
			name:         "all uploads fail and all causes are preserved",
			uploadErrors: []error{firstErr, lastErr},
			wantCalls: []string{
				"local:upload:archive.tar.gz", "aws:upload:archive.tar.gz",
			},
			wantError:      firstErr,
			wantErrorParts: []string{"before any backend upload completed", "retention was not run on any backend", "backend #1 (local)", "backend #2 (aws)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			var logs []string
			backends := []storage.ArchiveBackend{
				&recordingBackend{name: "local", calls: &calls, uploadErr: tt.uploadErrors[0]},
				&recordingBackend{name: "aws", calls: &calls, uploadErr: tt.uploadErrors[1]},
			}

			err := Deliver(context.Background(), backends, "archive.tar.gz", "/tmp/archive.tar.gz", 0, func(format string, args ...any) {
				logs = append(logs, fmt.Sprintf(format, args...))
			})
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("Deliver() error = %v, want wrapped %v", err, tt.wantError)
			}
			if tt.name == "all uploads fail and all causes are preserved" && !errors.Is(err, lastErr) {
				t.Fatalf("Deliver() error = %v, want wrapped %v", err, lastErr)
			}
			if tt.wantError == nil && err != nil {
				t.Fatalf("Deliver() unexpected error = %v", err)
			}
			for _, part := range tt.wantErrorParts {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("Deliver() error = %q, want substring %q", err, part)
				}
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Fatalf("Deliver() calls = %#v, want %#v", calls, tt.wantCalls)
			}
			if !reflect.DeepEqual(logs, tt.wantLogMessages) {
				t.Fatalf("Deliver() logs = %#v, want %#v", logs, tt.wantLogMessages)
			}
			for _, backend := range backends {
				if got := backend.(*recordingBackend).closeCount(); got != 0 {
					t.Fatalf("Deliver() closed backend %q %d times, want caller-owned lifecycle", backend.(*recordingBackend).name, got)
				}
			}
		})
	}
}

func TestDeliverReportsRetentionFailureAfterAllUploads(t *testing.T) {
	deleteErr := errors.New("delete failed")
	var calls []string
	first := &recordingBackend{name: "local", calls: &calls}
	second := &recordingBackend{name: "aws", calls: &calls, deleteErr: deleteErr}

	err := Deliver(context.Background(), []storage.ArchiveBackend{first, second}, "archive.tar.gz", "/tmp/archive.tar.gz", 0, nil)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("Deliver() error = %v, want wrapped %v", err, deleteErr)
	}
	for _, part := range []string{"successful retention on backend #1 (local)", "upload completed on all configured backends", "backend #2 (aws)"} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("Deliver() error = %q, want substring %q", err, part)
		}
	}
	want := []string{
		"local:upload:archive.tar.gz", "aws:upload:archive.tar.gz",
		"local:delete:archive.tar.gz", "aws:delete:archive.tar.gz",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("Deliver() calls = %#v, want %#v", calls, want)
	}
}

func TestDeliverPreservesSingleBackendBehavior(t *testing.T) {
	var calls []string
	backend := &recordingBackend{name: "local", calls: &calls}
	var logs []string

	err := Deliver(context.Background(), []storage.ArchiveBackend{backend}, "archive.tar.gz", "/tmp/archive.tar.gz", 0, func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	if err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	wantCalls := []string{"local:upload:archive.tar.gz", "local:delete:archive.tar.gz"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("Deliver() calls = %#v, want %#v", calls, wantCalls)
	}
	wantLog := fmt.Sprintf("Successfully uploaded backup to %T: verified", backend)
	if !reflect.DeepEqual(logs, []string{wantLog}) {
		t.Fatalf("Deliver() logs = %#v, want %#v", logs, []string{wantLog})
	}
}

type blockingBackend struct {
	calls *[]string
	name  string
}

func (b *blockingBackend) Upload(ctx context.Context, _ string, _ string) (string, error) {
	*b.calls = append(*b.calls, b.name+":upload")
	<-ctx.Done()
	return "", ctx.Err()
}

func (*blockingBackend) DeleteOldObjects(context.Context, string) error { return nil }
func (*blockingBackend) Close() error                                   { return nil }

func TestDeliverAppliesTimeoutToEveryUploadWithoutRetention(t *testing.T) {
	var calls []string
	backends := []storage.ArchiveBackend{
		&blockingBackend{name: "first", calls: &calls},
		&blockingBackend{name: "second", calls: &calls},
	}

	err := Deliver(context.Background(), backends, "archive.tar.gz", "/tmp/archive.tar.gz", time.Millisecond, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Deliver() error = %v, want deadline exceeded", err)
	}
	if want := []string{"first:upload", "second:upload"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("Deliver() calls = %#v, want %#v", calls, want)
	}
}
