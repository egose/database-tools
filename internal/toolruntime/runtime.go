package toolruntime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type CleanupStack struct {
	entries []cleanupEntry
}

type cleanupEntry struct {
	path   string
	remove func(string) error
}

func (c *CleanupStack) AddFile(path string, remove func(string) error) {
	c.entries = append(c.entries, cleanupEntry{path: path, remove: remove})
}

func (c *CleanupStack) AddDirectory(path string, remove func(string) error) {
	c.entries = append(c.entries, cleanupEntry{path: path, remove: remove})
}

func (c *CleanupStack) Run(keep bool) error {
	if keep {
		return nil
	}

	var cleanupErrors []error
	for i := len(c.entries) - 1; i >= 0; i-- {
		entry := c.entries[i]
		if err := entry.remove(entry.path); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cleanup %q: %w", entry.path, err))
		}
	}

	return errors.Join(cleanupErrors...)
}

func JoinPrimaryAndCleanupErrors(primary error, cleanup error) error {
	if primary == nil {
		return fmt.Errorf("cleanup failed: %w", cleanup)
	}
	if cleanup == nil {
		return primary
	}
	return errors.Join(primary, fmt.Errorf("cleanup failed: %w", cleanup))
}

func CloseAll[T interface{ Close() error }](items []T) error {
	var closeErrors []error
	for _, item := range items {
		if err := item.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close %T: %w", item, err))
		}
	}

	return errors.Join(closeErrors...)
}

func OperationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout == 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, timeout)
}
