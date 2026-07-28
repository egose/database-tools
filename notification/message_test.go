package notification

import (
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
