package common

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

func GetAzBlobServiceClient(accountName string, accountKey string) (*azblob.Client, error) {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	serviceClient, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, err
	}

	return serviceClient, nil
}

func GetAzBlobContainerClient(accountName string, accountKey string, containerName string) (*container.Client, error) {
	serviceClient, err := GetAzBlobServiceClient(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	containerClient := serviceClient.ServiceClient().NewContainerClient(containerName)

	return containerClient, nil
}
