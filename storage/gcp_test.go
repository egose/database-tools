package storage

import "testing"

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{name: "root", endpoint: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "storage api path", endpoint: "http://localhost:8080/storage/v1/", want: "http://localhost:8080"},
		{name: "upload api path", endpoint: "http://localhost:8080/upload/storage/v1", want: "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEndpoint(tt.endpoint); got != tt.want {
				t.Fatalf("normalizeEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}
