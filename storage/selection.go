package storage

import (
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/service/s3"
)

type objectTimestamp struct {
	Name       string
	ModifiedAt time.Time
}

func chooseLaterObject(current *objectTimestamp, candidate objectTimestamp) *objectTimestamp {
	if candidate.Name == "" || candidate.ModifiedAt.IsZero() {
		return current
	}

	if current == nil || candidate.ModifiedAt.After(current.ModifiedAt) {
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
