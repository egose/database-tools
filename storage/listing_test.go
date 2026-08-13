package storage

import "testing"

func TestNewS3ListObjectsInputUsesScopedPrefixAndPagination(t *testing.T) {
	input := newS3ListObjectsInput("bucket-name", "custom/")
	if input.Bucket == nil || *input.Bucket != "bucket-name" {
		t.Fatalf("newS3ListObjectsInput() bucket = %#v", input.Bucket)
	}
	if input.Prefix == nil || *input.Prefix != "custom/" {
		t.Fatalf("newS3ListObjectsInput() prefix = %#v", input.Prefix)
	}
	if input.MaxKeys == nil || *input.MaxKeys != backupListPageSize {
		t.Fatalf("newS3ListObjectsInput() maxKeys = %#v, want %d", input.MaxKeys, backupListPageSize)
	}
}

func TestNewAzureListBlobsFlatOptionsUsesScopedPrefixAndPagination(t *testing.T) {
	options := newAzureListBlobsFlatOptions("custom/")
	if options.Prefix == nil || *options.Prefix != "custom/" {
		t.Fatalf("newAzureListBlobsFlatOptions() prefix = %#v", options.Prefix)
	}
	if options.MaxResults == nil || int(*options.MaxResults) != backupListPageSize {
		t.Fatalf("newAzureListBlobsFlatOptions() maxResults = %#v, want %d", options.MaxResults, backupListPageSize)
	}
}

func TestNewGCPListObjectsOptionsUsesScopedPrefixAndPagination(t *testing.T) {
	options := newGCPListObjectsOptions("custom/")
	if options.Query == nil || options.Query.Prefix != "custom/" {
		t.Fatalf("newGCPListObjectsOptions() prefix = %#v", options.Query)
	}
	if options.PageSize != backupListPageSize {
		t.Fatalf("newGCPListObjectsOptions() pageSize = %d, want %d", options.PageSize, backupListPageSize)
	}
}
