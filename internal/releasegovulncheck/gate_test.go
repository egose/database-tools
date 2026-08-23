package releasegovulncheck

import (
	"os"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)

func TestEvaluateFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fixture       string
		scannerStatus int
		stderr        string
		wantErr       string
	}{
		{name: "clean", fixture: "testdata/clean.json"},
		{name: "allowed exception", fixture: "testdata/allowed.json", scannerStatus: 3},
		{name: "scanner error", fixture: "testdata/scanner-error.json", scannerStatus: 1, wantErr: "scanner error"},
		{name: "scanner stderr", fixture: "testdata/clean.json", scannerStatus: 1, stderr: "network failed", wantErr: "stderr output"},
		{name: "malformed output", fixture: "testdata/malformed.json", scannerStatus: 1, wantErr: "malformed govulncheck JSON output"},
		{name: "allowed finding in another scan mode", fixture: "testdata/allowed-other-scan-mode.json", scannerStatus: 3, wantErr: "unapproved reachable vulnerability"},
		{name: "allowed ID in another package", fixture: "testdata/allowed-id-other-package.json", scannerStatus: 3, wantErr: "unapproved reachable vulnerability"},
		{name: "new finding", fixture: "testdata/new-finding.json", scannerStatus: 3, wantErr: "GO-2099-0001"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := os.Open(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			_, err = Evaluate(f, tt.scannerStatus, tt.stderr, testNow, Exceptions)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Evaluate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Evaluate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestEvaluateRejectsExpiredException(t *testing.T) {
	t.Parallel()

	f, err := os.Open("testdata/allowed.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	exceptions := []Exception{{
		ID:        "GO-2024-2698",
		ScanMode:  "source",
		Module:    "github.com/mholt/archiver",
		Package:   "github.com/mholt/archiver",
		Symbol:    "Archive",
		Rationale: "temporary no-fix exception",
		Owner:     "release-security-maintainers",
		Created:   "2026-01-01",
		ReviewBy:  "2026-08-22",
		Tracking:  "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01",
	}}

	_, err = Evaluate(f, 3, "", testNow, exceptions)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("Evaluate() error = %v, want expired exception", err)
	}
}

func TestEvaluateRejectsMalformedExceptionMetadata(t *testing.T) {
	t.Parallel()

	_, err := Evaluate(strings.NewReader(`{"config":{"scan_mode":"source"}}`), 0, "", testNow, []Exception{{
		ID:       "GO-2024-2698",
		ScanMode: "source",
		Module:   "github.com/mholt/archiver",
		Package:  "github.com/mholt/archiver",
		Symbol:   "Archive",
		Created:  "2026-01-01",
		ReviewBy: "2026-11-30",
		Tracking: "ARCHIVE-01",
	}})
	if err == nil || !strings.Contains(err.Error(), "missing required metadata") {
		t.Fatalf("Evaluate() error = %v, want malformed exception metadata", err)
	}
}
