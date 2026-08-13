package notification

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessageForSuccess(t *testing.T) {
	msg := BuildMessage(true, time.UTC, "[backup]", "archive.tar.gz")
	if msg.Status != "Success" {
		t.Fatalf("BuildMessage() status = %q, want %q", msg.Status, "Success")
	}
	if msg.FilenameOrErrorLabel != "Filename" {
		t.Fatalf("BuildMessage() label = %q, want %q", msg.FilenameOrErrorLabel, "Filename")
	}
}

func TestBuildMessageForFailure(t *testing.T) {
	msg := BuildMessage(false, time.UTC, "", "boom")
	if msg.Status != "Failure" {
		t.Fatalf("BuildMessage() status = %q, want %q", msg.Status, "Failure")
	}
	if msg.FilenameOrErrorLabel != "Error" {
		t.Fatalf("BuildMessage() label = %q, want %q", msg.FilenameOrErrorLabel, "Error")
	}
}

func TestBuildMessageRedactsCredentialedURIs(t *testing.T) {
	msg := BuildMessage(false, time.UTC, "", "failed to upload mongodb://user:secret@db1.example.com:27017/test?sig=secret")
	if strings.Contains(msg.FilenameOrError, "secret") {
		t.Fatalf("BuildMessage() filename/error = %q, leaked secret", msg.FilenameOrError)
	}
	if !strings.Contains(msg.FilenameOrError, "mongodb://db1.example.com:27017/test?sig=REDACTED") {
		t.Fatalf("BuildMessage() filename/error = %q, want redacted URI", msg.FilenameOrError)
	}
}
