package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/egose/database-tools/utils"
	"golang.org/x/oauth2/google"
	iam "google.golang.org/api/iam/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GcpStorage struct {
	Bucket        string
	StorageClient *storage.Client
}

type GcpServiceAccountCreds struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthUri                 string `json:"auth_uri"`
	TokenUri                string `json:"token_uri"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

func (this *GcpStorage) Init(bucket, credsPath, projectID, privateKeyId, privateKey, clientEmail, clientID string) error {
	this.Bucket = bucket

	ctx := context.Background()

	var creds *google.Credentials
	var jsonData []byte

	if credsPath != "" {
		buf, err := ioutil.ReadFile(credsPath)
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}

		jsonData = buf
	} else if projectID != "" && privateKeyId != "" && privateKey != "" && clientEmail != "" && clientID != "" {
		decodedPrivateKey, err := strconv.Unquote(`"` + privateKey + `"`)
		if err != nil {
			return fmt.Errorf("failed to decode private key: %w", err)
		}

		credJson := GcpServiceAccountCreds{
			Type:                    "service_account",
			ProjectID:               projectID,
			PrivateKeyID:            privateKeyId,
			PrivateKey:              decodedPrivateKey,
			ClientEmail:             clientEmail,
			ClientID:                clientID,
			AuthUri:                 "https://accounts.google.com/o/oauth2/auth",
			TokenUri:                "https://oauth2.googleapis.com/token",
			AuthProviderX509CertUrl: "https://www.googleapis.com/oauth2/v1/certs",
			UniverseDomain:          "googleapis.com",
		}

		buf, err := json.Marshal(credJson)
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}

		jsonData = buf
	}

	if jsonData != nil {
		c, err := google.CredentialsFromJSON(context.Background(), jsonData, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}

		creds = c
	}

	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return err
	}

	this.StorageClient = client

	return nil
}

func (this *GcpStorage) Close() {
	this.StorageClient.Close()
}

func (this *GcpStorage) GetTargetObjectName(objectName string) (string, error) {
	if objectName == "" {
		return this.getLastUpdatedObjectName()
	}

	if _, err := this.getMetadata(objectName); err != nil {
		if err.Error() == "storage: object doesn't exist" {
			return this.getLastUpdatedObjectName()
		}
		return "", fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	return objectName, nil
}

func (this *GcpStorage) getLastUpdatedObjectName() (string, error) {
	bucket := this.StorageClient.Bucket(this.Bucket)

	ctx := context.Background()
	it := bucket.Objects(ctx, nil)

	var latestObject *storage.ObjectAttrs
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		if latestObject == nil || objAttrs.Updated.After(latestObject.Updated) {
			latestObject = objAttrs
		}
	}

	if latestObject == nil {
		return "", errors.New("no objects found in the bucket")
	}

	return latestObject.Name, nil
}

func generateServiceAccountKey(projectID, serviceAccountEmail string) ([]byte, error) {
	ctx := context.Background()

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("iam.NewService: %v", err)
	}

	keyReq := &iam.CreateServiceAccountKeyRequest{
		KeyAlgorithm: "KEY_ALG_RSA_2048",
	}

	key, err := iamService.Projects.ServiceAccounts.Keys.Create(
		fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, serviceAccountEmail), keyReq,
	).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("iamService.Projects.ServiceAccounts.Keys.Create: %v", err)
	}

	jsonKey, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %v", err)
	}

	return jsonKey, nil
}

// See https://cloud.google.com/storage/docs/uploading-objects-from-memory#storage-upload-object-from-memory-go
func (this *GcpStorage) Upload(objectName string, buffer []byte) (string, error) {
	bctx := context.Background()
	reader := bytes.NewReader(buffer)

	ctx, cancel := context.WithTimeout(bctx, time.Second*50)
	defer cancel()

	wc := this.StorageClient.Bucket(this.Bucket).Object(objectName).NewWriter(ctx)
	defer wc.Close()

	if _, err := io.Copy(wc, reader); err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	attrs, err := this.getMetadata(objectName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	return attrs.Etag, nil
}

// See https://cloud.google.com/storage/docs/viewing-editing-metadata#storage-view-object-metadata-go
func (this *GcpStorage) getMetadata(objectName string) (*storage.ObjectAttrs, error) {
	obj := this.StorageClient.Bucket(this.Bucket).Object(objectName)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		// storage: object doesn't exist
		return nil, err
	}

	return attrs, nil
}

// See https://cloud.google.com/storage/docs/downloading-objects#storage-download-object-go
func (this *GcpStorage) Download(objectName string, filePath string) error {
	obj := this.StorageClient.Bucket(this.Bucket).Object(objectName)

	dest, err := utils.CreateFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	ctx := context.Background()
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to create object reader: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(dest, reader)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}
