package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	gcs "cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	projectstorage "github.com/egose/database-tools/storage"
	"google.golang.org/api/iterator"
)

func main() {
	provider := flag.String("provider", "", "storage provider to inspect")
	localPath := flag.String("local-path", "", "local storage path")
	flag.Parse()

	count, err := getObjectCount(*provider, *localPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(count)
}

func getObjectCount(provider string, localPath string) (int, error) {
	switch provider {
	case "local":
		return countLocalObjects(localPath)
	case "s3":
		return countS3Objects()
	case "azure":
		return countAzureObjects()
	case "gcp":
		return countGCPObjects()
	default:
		return 0, fmt.Errorf("unsupported provider %q", provider)
	}
}

func countLocalObjects(localPath string) (int, error) {
	if localPath == "" {
		return 0, fmt.Errorf("--local-path is required for local storage")
	}

	scopeRoot := filepath.Join(localPath, filepath.FromSlash(getBackupPrefix()))
	count := 0
	err := filepath.Walk(scopeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countS3Objects() (int, error) {
	backend := new(projectstorage.AwsS3)
	if err := backend.Init(
		os.Getenv("MINIO_URL"),
		os.Getenv("MINIO_ACCESS_KEY"),
		os.Getenv("MINIO_SECRET_KEY"),
		"us-east-1",
		os.Getenv("MINIO_BUCKET"),
		true,
		0,
		getBackupPrefix(),
	); err != nil {
		return 0, err
	}

	count := 0
	err := backend.Service.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(backend.Bucket),
		Prefix: aws.String(backend.BackupPrefix),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		count += len(page.Contents)
		return true
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countAzureObjects() (int, error) {
	backend := new(projectstorage.AzBlob)
	if err := backend.Init(
		os.Getenv("AZURITE_ACCOUNT_NAME"),
		os.Getenv("AZURITE_ACCOUNT_KEY"),
		os.Getenv("AZURITE_CONTAINER"),
		os.Getenv("AZURITE_URL"),
		0,
		getBackupPrefix(),
	); err != nil {
		return 0, err
	}

	pager := backend.BlobContainerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{Prefix: aws.String(getBackupPrefix())})
	count := 0
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			return 0, err
		}

		for _, item := range resp.Segment.BlobItems {
			if item != nil && item.Name != nil {
				count++
			}
		}
	}

	return count, nil
}

func countGCPObjects() (int, error) {
	backend := new(projectstorage.GcpStorage)
	if err := backend.Init(
		context.Background(),
		fmt.Sprintf("http://localhost:%s/storage/v1/", os.Getenv("FAKE_GCP_PORT")),
		os.Getenv("FAKE_GCP_BUCKET"),
		"",
		"",
		"",
		"",
		"",
		"",
		0,
		getBackupPrefix(),
	); err != nil {
		return 0, err
	}
	defer backend.Close()

	count := 0
	it := backend.StorageClient.Bucket(backend.Bucket).Objects(context.Background(), &gcs.Query{Prefix: getBackupPrefix()})
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		count++
	}

	return count, nil
}

func getBackupPrefix() string {
	return projectstorage.NormalizeBackupPrefix(os.Getenv("MONGOARCHIVE__BACKUP_PREFIX"))
}
