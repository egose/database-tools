package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

type AzBlob struct {
	AccountName         string
	AccountKey          string
	ContainerName       string
	Endpoint            string
	BlobServiceClient   *azblob.Client
	BlobContainerClient *container.Client
	ExpiryDays          int
	BackupPrefix        string
}

func (this *AzBlob) Init(accountName string, accountKey string, containerName string, endpoint string, expiryDays int, backupPrefix string) error {
	this.AccountName = accountName
	this.AccountKey = accountKey
	this.ContainerName = containerName
	this.Endpoint = endpoint
	this.ExpiryDays = expiryDays
	this.BackupPrefix = NormalizeBackupPrefix(backupPrefix)

	serviceClient, err := this.getBlobServiceClient()
	if err != nil {
		return fmt.Errorf("failed to create blob service client: %v", err)
	}
	this.BlobServiceClient = serviceClient

	this.BlobContainerClient, err = this.getBlobContainerClient()
	if err != nil {
		return fmt.Errorf("failed to create blob container client: %v", err)
	}

	return nil
}

func (this *AzBlob) getBlobServiceClient() (*azblob.Client, error) {
	var serviceURL string

	// See https://learn.microsoft.com/en-us/rest/api/storageservices/list-containers2?tabs=microsoft-entra-id#Request
	if this.Endpoint == "" {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net/", this.AccountName)
	} else {
		serviceURL = fmt.Sprintf("%s/%s", this.Endpoint, this.AccountName)
	}

	cred, err := azblob.NewSharedKeyCredential(this.AccountName, this.AccountKey)
	if err != nil {
		return nil, err
	}

	return azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
}

func (this *AzBlob) getBlobContainerClient() (*container.Client, error) {
	containerClient := this.BlobServiceClient.ServiceClient().NewContainerClient(this.ContainerName)

	return containerClient, nil
}

func (this *AzBlob) getBlockBlobClient(blobName string) *blockblob.Client {
	return this.BlobContainerClient.NewBlockBlobClient(blobName)
}

func (this *AzBlob) GetTargetObjectName(ctx context.Context, blobName string) (string, error) {
	ctx = contextOrBackground(ctx)

	if blobName != "" {
		resolved, found, err := resolveExplicitObjectName(this.BackupPrefix, blobName, func(candidate string) (bool, error) {
			_, err := this.getBlockBlobClient(candidate).GetProperties(ctx, &blob.GetPropertiesOptions{})
			if err == nil {
				return true, nil
			}
			if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ResourceNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("failed to retrieve metadata: %w", err)
		})
		if err != nil {
			return "", err
		}
		if found {
			return resolved, nil
		}

		return "", fmt.Errorf("object %q not found in container %q", blobName, this.ContainerName)
	}

	pager := this.BlobContainerClient.NewListBlobsFlatPager(newAzureListBlobsFlatOptions(this.BackupPrefix))
	var latest *objectTimestamp

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list objects: %v", err)
		}

		candidates := make([]objectTimestamp, 0, len(resp.Segment.BlobItems))
		for _, item := range resp.Segment.BlobItems {
			if item == nil || item.Name == nil || item.Properties == nil || item.Properties.LastModified == nil {
				continue
			}

			candidates = append(candidates, objectTimestamp{
				Name:       *item.Name,
				ModifiedAt: *item.Properties.LastModified,
			})
		}

		pageLatest, ok := latestEligibleObject(candidates, this.BackupPrefix)
		if ok {
			latest = chooseLaterObject(latest, pageLatest)
		}
	}

	if latest == nil {
		return "", errors.New("no target object name found")
	}

	return latest.Name, nil
}

func (this *AzBlob) Upload(ctx context.Context, blobName string, filePath string) (string, error) {
	ctx = contextOrBackground(ctx)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	blockBlobClient := this.getBlockBlobClient(blobName)
	blockBlobUploadOptions := blockblob.UploadOptions{
		// Metadata: map[string]string{"meta": "value"},
		// Tags:     map[string]string{"tag": "value"},
	}
	uploadResp, err := blockBlobClient.Upload(ctx, streaming.NopCloser(file), &blockBlobUploadOptions)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}
	if _, err := blockBlobClient.GetProperties(ctx, &blob.GetPropertiesOptions{}); err != nil {
		return "", fmt.Errorf("failed to verify uploaded object: %w", err)
	}

	etag := toGeneratedETagString(uploadResp.ETag)
	return *etag, nil
}

func toGeneratedETagString(etag *azcore.ETag) *string {
	if etag == nil || *etag == azcore.ETagAny {
		return (*string)(etag)
	}

	str := "\"" + (string)(*etag) + "\""
	return &str
}

func (this *AzBlob) Download(ctx context.Context, blobName string, filePath string) error {
	ctx = contextOrBackground(ctx)

	blockBlobClient := this.getBlockBlobClient(blobName)
	downloadOptions := &azblob.DownloadFileOptions{
		Progress: func(bytesTransferred int64) {
			mlog.Logvf(mlog.Info, "Downloaded %d bytes", bytesTransferred)
		},
	}

	return utils.WriteFileAtomically(filePath, func(dest *os.File) error {
		_, err := blockBlobClient.DownloadFile(ctx, dest, downloadOptions)
		if err != nil {
			return fmt.Errorf("failed to download object: %w", err)
		}
		return nil
	})
}

func (this *AzBlob) DeleteOldObjects(ctx context.Context, currentObjectName string) error {
	ctx = contextOrBackground(ctx)

	if this.ExpiryDays == 0 {
		return nil
	}

	pager := this.BlobContainerClient.NewListBlobsFlatPager(newAzureListBlobsFlatOptions(this.BackupPrefix))
	now := time.Now()

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		candidates := make([]objectTimestamp, 0, len(resp.Segment.BlobItems))
		for _, item := range resp.Segment.BlobItems {
			if item == nil || item.Name == nil || item.Properties == nil || item.Properties.LastModified == nil {
				continue
			}

			daysOld := now.Sub(*item.Properties.LastModified).Hours() / 24
			mlog.Logvf(mlog.Info, "Checking object: %s (%.1f days old)", *item.Name, daysOld)
			candidates = append(candidates, objectTimestamp{Name: *item.Name, ModifiedAt: *item.Properties.LastModified})
		}

		if err := deleteExpiredObjects(candidates, this.BackupPrefix, this.ExpiryDays, now, currentObjectName, func(name string) error {
			_, err := this.getBlockBlobClient(name).Delete(ctx, nil)
			if err == nil {
				mlog.Logvf(mlog.Info, "Deleted object: %s", name)
			}
			return err
		}); err != nil {
			return err
		}
	}

	return nil
}

func (this *AzBlob) Close() error {
	return nil
}
