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
}

func (this *AzBlob) Init(accountName string, accountKey string, containerName string, endpoint string, expiryDays int) error {
	this.AccountName = accountName
	this.AccountKey = accountKey
	this.ContainerName = containerName
	this.Endpoint = endpoint
	this.ExpiryDays = expiryDays

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

func (this *AzBlob) GetTargetObjectName(blobName string) (string, error) {
	if blobName != "" {
		_, err := this.getBlockBlobClient(blobName).GetProperties(context.Background(), &blob.GetPropertiesOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to retrieve metadata: %w", err)
		}

		return blobName, nil
	}

	pager := this.BlobContainerClient.NewListBlobsFlatPager(nil)
	var latest *objectTimestamp

	for pager.More() {
		resp, err := pager.NextPage(context.Background())
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

		pageLatest, ok := latestObject(candidates)
		if ok {
			latest = chooseLaterObject(latest, pageLatest)
		}
	}

	if latest == nil {
		return "", errors.New("no target object name found")
	}

	return latest.Name, nil
}

func (this *AzBlob) Upload(blobName string, filePath string) (string, error) {
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
	uploadResp, err := blockBlobClient.Upload(context.Background(), streaming.NopCloser(file), &blockBlobUploadOptions)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
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

func (this *AzBlob) Download(blobName string, filePath string) error {
	dest, err := utils.CreateFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	blockBlobClient := this.getBlockBlobClient(blobName)
	downloadOptions := &azblob.DownloadFileOptions{
		Progress: func(bytesTransferred int64) {
			mlog.Logvf(mlog.Info, "Downloaded %d bytes", bytesTransferred)
		},
	}

	_, err = blockBlobClient.DownloadFile(context.Background(), dest, downloadOptions)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

func (this *AzBlob) DeleteOldObjects() error {
	if this.ExpiryDays == 0 {
		return nil
	}

	pager := this.BlobContainerClient.NewListBlobsFlatPager(nil)
	now := time.Now()

	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, item := range resp.Segment.BlobItems {
			if item == nil || item.Name == nil || item.Properties == nil || item.Properties.LastModified == nil {
				continue
			}

			daysOld := now.Sub(*item.Properties.LastModified).Hours() / 24
			mlog.Logvf(mlog.Info, "Checking object: %s (%.1f days old)", *item.Name, daysOld)

			if !isExpired(*item.Properties.LastModified, this.ExpiryDays, now) {
				continue
			}

			if _, err := this.getBlockBlobClient(*item.Name).Delete(context.Background(), nil); err != nil {
				return fmt.Errorf("failed to delete object %q: %w", *item.Name, err)
			}

			mlog.Logvf(mlog.Info, "Deleted object: %s", *item.Name)
		}
	}

	return nil
}
