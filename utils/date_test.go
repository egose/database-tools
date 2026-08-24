package utils

import (
	"regexp"
	"testing"
)

func TestGetNewFilenamePreservesManagedArchiveShape(t *testing.T) {
	filename, name := GetNewFilename()
	if want := name + ".tar.gz"; filename != want {
		t.Fatalf("GetNewFilename() filename = %q, want %q", filename, want)
	}
	if !regexp.MustCompile(`^\d{13}-\d{4}-\d{2}-\d{2}T\d{6}\.\d{3}Z$`).MatchString(name) {
		t.Fatalf("GetNewFilename() name = %q, want managed archive shape", name)
	}
}
