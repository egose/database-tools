package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{name: "root", endpoint: "http://localhost:8080", want: "http://localhost:8080/storage/v1/"},
		{name: "storage api path", endpoint: "http://localhost:8080/storage/v1/", want: "http://localhost:8080/storage/v1/"},
		{name: "upload api path", endpoint: "http://localhost:8080/upload/storage/v1", want: "http://localhost:8080/storage/v1/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEndpoint(tt.endpoint); got != tt.want {
				t.Fatalf("normalizeEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewEmulatorStorageClientDoesNotMutateEnvironment(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	t.Setenv("STORAGE_EMULATOR_HOST", "sentinel")

	client, err := newEmulatorStorageClient(context.Background(), server.URL+"/storage/v1/")
	if err != nil {
		t.Fatalf("newEmulatorStorageClient() error = %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if got := os.Getenv("STORAGE_EMULATOR_HOST"); got != "sentinel" {
		t.Fatalf("STORAGE_EMULATOR_HOST = %q, want sentinel", got)
	}
}
