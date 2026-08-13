package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	BackendAzure = "azure"
	BackendAWS   = "aws"
	BackendGCP   = "gcp"
	BackendLocal = "local"
)

type objectTimestamp struct {
	Name       string
	ModifiedAt time.Time
}

func chooseLaterObject(current *objectTimestamp, candidate objectTimestamp) *objectTimestamp {
	if candidate.Name == "" || candidate.ModifiedAt.IsZero() {
		return current
	}

	if current == nil || candidate.ModifiedAt.After(current.ModifiedAt) || (candidate.ModifiedAt.Equal(current.ModifiedAt) && candidate.Name < current.Name) {
		candidateCopy := candidate
		return &candidateCopy
	}

	return current
}

func latestObject(candidates []objectTimestamp) (objectTimestamp, bool) {
	var latest *objectTimestamp
	for _, candidate := range candidates {
		latest = chooseLaterObject(latest, candidate)
	}

	if latest == nil {
		return objectTimestamp{}, false
	}

	return *latest, true
}

func latestS3Object(objects []*s3.Object) (objectTimestamp, bool) {
	candidates := make([]objectTimestamp, 0, len(objects))
	for _, obj := range objects {
		if obj == nil || obj.Key == nil || obj.LastModified == nil {
			continue
		}

		candidates = append(candidates, objectTimestamp{
			Name:       *obj.Key,
			ModifiedAt: *obj.LastModified,
		})
	}

	return latestObject(candidates)
}

func latestGCPObject(objects []*gcs.ObjectAttrs) (objectTimestamp, bool) {
	candidates := make([]objectTimestamp, 0, len(objects))
	for _, obj := range objects {
		if obj == nil || obj.Name == "" || obj.Updated.IsZero() {
			continue
		}

		candidates = append(candidates, objectTimestamp{
			Name:       obj.Name,
			ModifiedAt: obj.Updated,
		})
	}

	return latestObject(candidates)
}

func resolveExplicitObjectName(prefix string, objectName string, exists func(string) (bool, error)) (string, bool, error) {
	for _, candidate := range lookupObjectCandidates(prefix, objectName) {
		found, err := exists(candidate)
		if err != nil {
			return "", false, err
		}
		if found {
			return candidate, true, nil
		}
	}

	return "", false, nil
}

func BackendName(storageBackend Storage) (string, error) {
	switch storageBackend.(type) {
	case *AzBlob:
		return BackendAzure, nil
	case *AwsS3:
		return BackendAWS, nil
	case *GcpStorage:
		return BackendGCP, nil
	case *LocalStorage:
		return BackendLocal, nil
	default:
		return "", fmt.Errorf("unsupported storage backend type %T", storageBackend)
	}
}

func SelectRestoreStorage(storages []Storage, requestedBackend string) (Storage, error) {
	if len(storages) == 0 {
		return nil, fmt.Errorf("no storage backends configured")
	}

	requestedBackend = normalizeBackendName(requestedBackend)
	availableBackends, err := configuredBackendNames(storages)
	if err != nil {
		return nil, err
	}

	if requestedBackend == "" {
		if len(storages) == 1 {
			return storages[0], nil
		}
		return nil, fmt.Errorf("multiple storage backends configured (%s); specify --storage-backend", strings.Join(availableBackends, ", "))
	}

	for _, storageBackend := range storages {
		backendName, err := BackendName(storageBackend)
		if err != nil {
			return nil, err
		}
		if backendName == requestedBackend {
			return storageBackend, nil
		}
	}

	return nil, fmt.Errorf("storage backend %q is not configured; available backends: %s", requestedBackend, strings.Join(availableBackends, ", "))
}

func configuredBackendNames(storages []Storage) ([]string, error) {
	names := make([]string, 0, len(storages))
	seen := make(map[string]struct{}, len(storages))
	for _, storageBackend := range storages {
		name, err := BackendName(storageBackend)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func normalizeBackendName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
