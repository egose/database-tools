package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/junminahn/mongo-tools-ext/utils"
)

type AzBlob struct {
	AccountName         string
	AccountKey          string
	ContainerName       string
	BlobServiceClient   *azblob.Client
	BlobContainerClient *container.Client
}

func (this *AzBlob) Init(accountName string, accountKey string, containerName string) error {
	this.AccountName = accountName
	this.AccountKey = accountKey
	this.ContainerName = containerName

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
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", this.AccountName)

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
	bname := ""

	pager := this.BlobContainerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Include: container.ListBlobsInclude{Snapshots: false, Versions: true},
	})

	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			return "", fmt.Errorf("failed to list objects: %v", err)
		}

		for _, blob := range resp.Segment.BlobItems {
			if blobName == "" || blobName == *blob.Name {
				bname = *blob.Name
				break
			}
		}

		if bname != "" {
			break
		}
	}

	if bname == "" {
		return "", errors.New("no target object name found")
	}

	return bname, nil
}

func (this *AzBlob) Upload(blobName string, buffer []byte) (string, error) {
	blockBlobClient := this.getBlockBlobClient(blobName)
	blockBlobUploadOptions := blockblob.UploadOptions{
		// Metadata: map[string]string{"meta": "value"},
		// Tags:     map[string]string{"tag": "value"},
	}
	uploadResp, err := blockBlobClient.Upload(context.Background(), streaming.NopCloser(bytes.NewReader(buffer)), &blockBlobUploadOptions)
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
			fmt.Printf("Downloaded %d.\n", bytesTransferred)
		},
	}

	_, err = blockBlobClient.DownloadFile(context.Background(), dest, downloadOptions)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}
