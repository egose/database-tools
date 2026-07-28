package storage

import (
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
