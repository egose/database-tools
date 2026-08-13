package storage

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const DefaultBackupPrefix = "mongo-archive/"

var backupObjectPattern = regexp.MustCompile(`^\d{13}-\d{4}-\d{2}-\d{2}T\d{6}\.\d{3}Z\.tar\.gz$`)

func NormalizeBackupPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return DefaultBackupPrefix
	}

	return prefix + "/"
}

func BuildBackupObjectName(prefix string, filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if !backupObjectPattern.MatchString(filename) {
		return "", fmt.Errorf("invalid backup filename %q", filename)
	}

	return NormalizeBackupPrefix(prefix) + filename, nil
}

func lookupObjectCandidates(prefix string, objectName string) []string {
	objectName = strings.TrimSpace(objectName)
	if objectName == "" {
		return nil
	}

	candidates := make([]string, 0, 2)
	if !strings.Contains(objectName, "/") && backupObjectPattern.MatchString(objectName) {
		candidates = append(candidates, NormalizeBackupPrefix(prefix)+objectName)
	}
	candidates = append(candidates, objectName)

	unique := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}

	return unique
}

func isEligibleBackupObject(name string, prefix string) bool {
	prefix = NormalizeBackupPrefix(prefix)
	if !strings.HasPrefix(name, prefix) {
		return false
	}

	remainder := strings.TrimPrefix(name, prefix)
	if remainder == "" || strings.Contains(remainder, "/") {
		return false
	}

	return backupObjectPattern.MatchString(remainder)
}

func latestEligibleObject(candidates []objectTimestamp, prefix string) (objectTimestamp, bool) {
	filtered := make([]objectTimestamp, 0, len(candidates))
	for _, candidate := range candidates {
		if !isEligibleBackupObject(candidate.Name, prefix) {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return latestObject(filtered)
}

func deleteExpiredObjects(candidates []objectTimestamp, prefix string, expiryDays int, now time.Time, preserveName string, deleteFn func(string) error) error {
	if expiryDays == 0 {
		return nil
	}

	for _, candidate := range candidates {
		if !isEligibleBackupObject(candidate.Name, prefix) {
			continue
		}
		if candidate.Name == preserveName {
			continue
		}
		if !isExpired(candidate.ModifiedAt, expiryDays, now) {
			continue
		}
		if err := deleteFn(candidate.Name); err != nil {
			return fmt.Errorf("failed to delete object %q: %w", candidate.Name, err)
		}
	}

	return nil
}
