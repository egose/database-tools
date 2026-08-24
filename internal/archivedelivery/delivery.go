package archivedelivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egose/database-tools/internal/toolruntime"
	"github.com/egose/database-tools/storage"
)

type Logf func(format string, args ...any)

type deliveryError struct {
	message string
	cause   error
}

func (e *deliveryError) Error() string { return e.message }
func (e *deliveryError) Unwrap() error { return e.cause }

// Deliver uploads an archive to every backend before starting retention.
// Backend lifecycle remains owned by the caller that constructed the backends.
func Deliver(ctx context.Context, backends []storage.ArchiveBackend, objectName string, archivePath string, operationTimeout time.Duration, logf Logf) error {
	if len(backends) <= 1 {
		return deliverToSingleBackend(ctx, backends, objectName, archivePath, operationTimeout, logf)
	}

	uploadedBackends := make([]string, 0, len(backends))
	failedBackends := make([]string, 0, len(backends))
	uploadErrors := make([]error, 0, len(backends))
	for i, backend := range backends {
		backendName := describeBackend(i, backend)
		uploadCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		result, err := backend.Upload(uploadCtx, objectName, archivePath)
		cancel()
		if err != nil {
			failedBackends = append(failedBackends, fmt.Sprintf("%s: %v", backendName, err))
			uploadErrors = append(uploadErrors, err)
			continue
		}

		uploadedBackends = append(uploadedBackends, backendName)
		log(logf, "Successfully uploaded backup to %s: %v", backendName, result)
	}

	if len(uploadErrors) > 0 {
		partialState := "before any backend upload completed"
		if len(uploadedBackends) > 0 {
			partialState = "after successful uploads to " + formatBackends(uploadedBackends, "none")
		}
		return &deliveryError{
			message: fmt.Sprintf(
				"archive upload failed %s; retention was not run on any backend: failed uploads: %s",
				partialState,
				strings.Join(failedBackends, "; "),
			),
			cause: errors.Join(uploadErrors...),
		}
	}

	log(logf, "Verified archive upload across %d storage backends; starting retention.", len(uploadedBackends))

	retainedBackends := make([]string, 0, len(backends))
	for i, backend := range backends {
		backendName := describeBackend(i, backend)
		deleteCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		err := backend.DeleteOldObjects(deleteCtx, objectName)
		cancel()
		if err != nil {
			return &deliveryError{
				message: fmt.Sprintf(
					"archive retention failed after successful retention on %s; archive upload completed on all configured backends: failed to delete old objects in %s: %v",
					formatBackends(retainedBackends, "none"),
					backendName,
					err,
				),
				cause: err,
			}
		}
		retainedBackends = append(retainedBackends, backendName)
	}

	return nil
}

func deliverToSingleBackend(ctx context.Context, backends []storage.ArchiveBackend, objectName string, archivePath string, operationTimeout time.Duration, logf Logf) error {
	for _, backend := range backends {
		uploadCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		result, err := backend.Upload(uploadCtx, objectName, archivePath)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to upload to %T: %w", backend, err)
		}
		log(logf, "Successfully uploaded backup to %T: %v", backend, result)

		deleteCtx, cancel := toolruntime.OperationContext(ctx, operationTimeout)
		err = backend.DeleteOldObjects(deleteCtx, objectName)
		cancel()
		if err != nil {
			return fmt.Errorf("failed to delete old objects in %T: %w", backend, err)
		}
	}

	return nil
}

func describeBackend(index int, backend storage.ArchiveBackend) string {
	if named, ok := backend.(storage.BackendIdentifier); ok {
		if name, err := storage.BackendName(named); err == nil {
			return fmt.Sprintf("backend #%d (%s)", index+1, name)
		}
	}
	return fmt.Sprintf("backend #%d (%T)", index+1, backend)
}

func formatBackends(backends []string, empty string) string {
	if len(backends) == 0 {
		return empty
	}
	return strings.Join(backends, ", ")
}

func log(logf Logf, format string, args ...any) {
	if logf != nil {
		logf(format, args...)
	}
}
