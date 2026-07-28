package utils

import "testing"

func TestResolvePathWithinRootAllowsNestedPaths(t *testing.T) {
	got, err := ResolvePathWithinRoot("/tmp/root", "nested/archive.tar.gz")
	if err != nil {
		t.Fatalf("ResolvePathWithinRoot() returned error: %v", err)
	}

	want := "/tmp/root/nested/archive.tar.gz"
	if got != want {
		t.Fatalf("ResolvePathWithinRoot() = %q, want %q", got, want)
	}
}

func TestResolvePathWithinRootRejectsTraversal(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "../escape.tar.gz"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected traversal error")
	}
}

func TestResolvePathWithinRootRejectsAbsolutePath(t *testing.T) {
	if _, err := ResolvePathWithinRoot("/tmp/root", "/etc/passwd"); err == nil {
		t.Fatal("ResolvePathWithinRoot() expected absolute path error")
	}
}
