package storage

import (
	gcs "cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

const backupListPageSize = 1000

func newS3ListObjectsInput(bucket string, prefix string) *s3.ListObjectsV2Input {
	return &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(backupListPageSize),
	}
}

func newAzureListBlobsFlatOptions(prefix string) *container.ListBlobsFlatOptions {
	maxResults := int32(backupListPageSize)
	return &container.ListBlobsFlatOptions{
		Prefix:     &prefix,
		MaxResults: &maxResults,
	}
}

type gcpListObjectsOptions struct {
	Query    *gcs.Query
	PageSize int
}

func newGCPListObjectsOptions(prefix string) gcpListObjectsOptions {
	return gcpListObjectsOptions{
		Query:    &gcs.Query{Prefix: prefix},
		PageSize: backupListPageSize,
	}
}
