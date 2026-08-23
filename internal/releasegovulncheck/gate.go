package releasegovulncheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

type Exception struct {
	ID        string
	ScanMode  string
	Module    string
	Package   string
	Symbol    string
	Rationale string
	Owner     string
	Created   string
	ReviewBy  string
	Tracking  string
}

type Result struct {
	AllowedReachable int
	IgnoredImported  int
}

type message struct {
	Config  *config         `json:"config"`
	Finding *finding        `json:"finding"`
	Error   json.RawMessage `json:"error"`
	Errors  json.RawMessage `json:"errors"`
	Extra   json.RawMessage `json:"-"`
}

type config struct {
	ScanMode string `json:"scan_mode"`
}

type finding struct {
	OSV   string  `json:"osv"`
	Trace []frame `json:"trace"`
}

type frame struct {
	Module   string `json:"module"`
	Package  string `json:"package"`
	Function string `json:"function"`
	Receiver string `json:"receiver"`
}

type findingKey struct {
	ID       string
	ScanMode string
	Module   string
	Package  string
	Symbol   string
}

var Exceptions = []Exception{
	{ID: "GO-2024-2698", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "Archive", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2024-2698", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "init", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2024-2698", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "writeWalk$1", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-3605", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "Archive", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-3605", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "init", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-3605", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: "writeWalk$1", Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-4020", ScanMode: "source", Module: "github.com/nwaples/rardecode", Package: "github.com/nwaples/rardecode", Symbol: "init", Rationale: "No upstream fix is available through the legacy archiver dependency; RAR extraction is not used by the hardened restore path and dependency removal remains tracked.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-4020", ScanMode: "source", Module: "github.com/nwaples/rardecode", Package: "github.com/nwaples/rardecode", Symbol: "*cipherBlockReader.Read", Rationale: "No upstream fix is available through the legacy archiver dependency; RAR extraction is not used by the hardened restore path and dependency removal remains tracked.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
	{ID: "GO-2025-4020", ScanMode: "source", Module: "github.com/nwaples/rardecode", Package: "github.com/nwaples/rardecode", Symbol: "*limitedReader.Read", Rationale: "No upstream fix is available through the legacy archiver dependency; RAR extraction is not used by the hardened restore path and dependency removal remains tracked.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
}

func init() {
	for _, symbol := range []string{"*Bz2.String", "*Gz.String", "*Rar.String", "*Snappy.String", "*Tar.String", "*TarBz2.String", "*TarGz.String", "*TarLz4.String", "*TarSz.String", "*TarXz.String", "*Xz.String", "*Zip.String"} {
		Exceptions = append(Exceptions,
			Exception{ID: "GO-2024-2698", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: symbol, Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
			Exception{ID: "GO-2025-3605", ScanMode: "source", Module: "github.com/mholt/archiver", Package: "github.com/mholt/archiver", Symbol: symbol, Rationale: "No upstream fix is available for the legacy archiver dependency; restore extraction no longer uses archiver and remaining archive/create paths are tracked for dependency removal.", Owner: "release-security-maintainers", Created: "2026-08-23", ReviewBy: "2026-11-30", Tracking: "docs/tasks/20260823-112531-post-remediation-codebase-health.md#SUPPLY-01"},
		)
	}
}

func Evaluate(r io.Reader, scannerStatus int, stderr string, now time.Time, exceptions []Exception) (Result, error) {
	if err := validateExceptions(exceptions, now); err != nil {
		return Result{}, err
	}

	allowed := map[findingKey]struct{}{}
	for _, exception := range exceptions {
		allowed[findingKey{ID: exception.ID, ScanMode: exception.ScanMode, Module: exception.Module, Package: exception.Package, Symbol: exception.Symbol}] = struct{}{}
	}

	dec := json.NewDecoder(r)
	scanMode := ""
	seenMessage := false
	var disallowed []string
	result := Result{}

	for {
		var msg message
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return result, fmt.Errorf("malformed govulncheck JSON output: %w", err)
		}
		seenMessage = true

		if len(msg.Error) > 0 || len(msg.Errors) > 0 {
			return result, fmt.Errorf("govulncheck reported a scanner error")
		}

		if msg.Config != nil {
			if msg.Config.ScanMode == "" {
				return result, fmt.Errorf("govulncheck config is missing scan_mode")
			}
			scanMode = msg.Config.ScanMode
		}

		if msg.Finding == nil {
			continue
		}

		if scanMode == "" {
			return result, fmt.Errorf("govulncheck finding %q appeared before scan_mode config", msg.Finding.OSV)
		}

		key, ok := reachableFindingKey(scanMode, *msg.Finding)
		if !ok {
			result.IgnoredImported++
			continue
		}

		if _, ok := allowed[key]; ok {
			result.AllowedReachable++
			continue
		}

		disallowed = append(disallowed, key.String())
	}

	if !seenMessage {
		return result, fmt.Errorf("govulncheck produced no JSON output")
	}

	if scannerStatus != 0 && strings.TrimSpace(stderr) != "" {
		return result, fmt.Errorf("govulncheck exited with status %d and stderr output", scannerStatus)
	}

	if len(disallowed) > 0 {
		sort.Strings(disallowed)
		return result, fmt.Errorf("unapproved reachable vulnerability findings:\n%s", strings.Join(disallowed, "\n"))
	}

	if scannerStatus != 0 && result.AllowedReachable == 0 {
		return result, fmt.Errorf("govulncheck exited with status %d without an allowed reachable vulnerability finding", scannerStatus)
	}

	return result, nil
}

func validateExceptions(exceptions []Exception, now time.Time) error {
	seen := map[findingKey]struct{}{}
	for i, exception := range exceptions {
		if exception.ID == "" || exception.ScanMode == "" || exception.Module == "" || exception.Package == "" || exception.Symbol == "" || exception.Rationale == "" || exception.Owner == "" || exception.Created == "" || exception.ReviewBy == "" || exception.Tracking == "" {
			return fmt.Errorf("vulnerability exception %d is missing required metadata", i)
		}
		if !strings.HasPrefix(exception.ID, "GO-") {
			return fmt.Errorf("vulnerability exception %d has malformed vulnerability ID %q", i, exception.ID)
		}
		if !strings.HasPrefix(exception.Tracking, "docs/tasks/20260823-112531-post-remediation-codebase-health.md#") && !strings.HasPrefix(exception.Tracking, "https://github.com/egose/database-tools/issues/") {
			return fmt.Errorf("vulnerability exception %s must reference a live issue or task", exception.ID)
		}
		if _, err := time.Parse(dateLayout, exception.Created); err != nil {
			return fmt.Errorf("vulnerability exception %s has malformed creation date %q", exception.ID, exception.Created)
		}
		reviewBy, err := time.Parse(dateLayout, exception.ReviewBy)
		if err != nil {
			return fmt.Errorf("vulnerability exception %s has malformed review date %q", exception.ID, exception.ReviewBy)
		}
		if reviewBy.Before(midnightUTC(now)) {
			return fmt.Errorf("vulnerability exception %s expired on %s", exception.ID, exception.ReviewBy)
		}

		key := findingKey{ID: exception.ID, ScanMode: exception.ScanMode, Module: exception.Module, Package: exception.Package, Symbol: exception.Symbol}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate vulnerability exception for %s", key.String())
		}
		seen[key] = struct{}{}
	}
	return nil
}

func reachableFindingKey(scanMode string, f finding) (findingKey, bool) {
	if f.OSV == "" || len(f.Trace) == 0 {
		return findingKey{}, false
	}
	first := f.Trace[0]
	symbol := symbolName(first)
	if first.Module == "" || first.Package == "" || symbol == "" {
		return findingKey{}, false
	}
	return findingKey{ID: f.OSV, ScanMode: scanMode, Module: first.Module, Package: first.Package, Symbol: symbol}, true
}

func symbolName(f frame) string {
	if f.Function == "" {
		return ""
	}
	if f.Receiver == "" {
		return f.Function
	}
	return f.Receiver + "." + f.Function
}

func (k findingKey) String() string {
	return fmt.Sprintf("%s scan_mode=%s module=%s package=%s symbol=%s", k.ID, k.ScanMode, k.Module, k.Package, k.Symbol)
}

func midnightUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
