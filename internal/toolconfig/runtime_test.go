package toolconfig

import (
	"testing"
	"time"
)

func TestRuntimeParsersUseInjectedEnvironmentAndRunInParallel(t *testing.T) {
	t.Parallel()

	t.Run("workspace", func(t *testing.T) {
		t.Parallel()
		got := ReadWorkspaceBase(testEnv{"DUMP_PATH": "/custom/work"}, "DUMP_PATH", "/default/work")
		if got != "/custom/work" {
			t.Fatalf("ReadWorkspaceBase() = %q, want injected value", got)
		}
	})

	t.Run("duration", func(t *testing.T) {
		t.Parallel()
		got, err := ReadOptionalDuration(testEnv{"STORAGE_OPERATION_TIMEOUT": "250ms"}, "STORAGE_OPERATION_TIMEOUT")
		if err != nil {
			t.Fatalf("ReadOptionalDuration() error = %v", err)
		}
		if got != 250*time.Millisecond {
			t.Fatalf("ReadOptionalDuration() = %v, want 250ms", got)
		}
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		got, err := ReadPositiveInt(testEnv{"ARCHIVE_MAX_ENTRIES": "12"}, "ARCHIVE_MAX_ENTRIES", 1)
		if err != nil {
			t.Fatalf("ReadPositiveInt() error = %v", err)
		}
		if got != 12 {
			t.Fatalf("ReadPositiveInt() = %d, want 12", got)
		}
	})

	t.Run("int64", func(t *testing.T) {
		t.Parallel()
		got, err := ReadPositiveInt64(testEnv{"UPDATE_MAX_BYTES": "4096"}, "UPDATE_MAX_BYTES", 1)
		if err != nil {
			t.Fatalf("ReadPositiveInt64() error = %v", err)
		}
		if got != 4096 {
			t.Fatalf("ReadPositiveInt64() = %d, want 4096", got)
		}
	})
}

func TestRuntimeParsersRejectInvalidInputs(t *testing.T) {
	t.Parallel()

	if _, err := ReadOptionalDuration(testEnv{"TIMEOUT": "nope"}, "TIMEOUT"); err == nil {
		t.Fatal("ReadOptionalDuration() expected invalid duration error")
	}
	if _, err := ReadOptionalDuration(testEnv{"TIMEOUT": "0s"}, "TIMEOUT"); err == nil {
		t.Fatal("ReadOptionalDuration() expected positive duration error")
	}
	if _, err := ReadPositiveInt(testEnv{"LIMIT": "0"}, "LIMIT", 1); err == nil {
		t.Fatal("ReadPositiveInt() expected positive integer error")
	}
	if _, err := ReadPositiveInt64(testEnv{"LIMIT": "abc"}, "LIMIT", 1); err == nil {
		t.Fatal("ReadPositiveInt64() expected positive integer error")
	}
}

type testEnv map[string]string

func (e testEnv) GetValue(key string, defaults ...string) string {
	if value, ok := e[key]; ok && value != "" {
		return value
	}
	for _, fallback := range defaults {
		if fallback != "" {
			return fallback
		}
	}
	return ""
}
