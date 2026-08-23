package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
	"golang.org/x/oauth2/google"
	iam "google.golang.org/api/iam/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GcpStorage struct {
	Bucket          string
	StorageClient   *storage.Client
	ExpiryDays      int
	BackupPrefix    string
	closeOnce       sync.Once
	closeErr        error
	uploadObject    func(context.Context, string, string) (string, error)
	downloadObject  func(context.Context, string, *os.File) error
	objectExists    func(context.Context, string) (bool, error)
	listObjectPages func(context.Context, func([]objectTimestamp) bool) error
	deleteObject    func(context.Context, string) error
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

func (this *GcpStorage) Init(ctx context.Context, endpoint, bucket, credsPath, projectID, privateKeyId, privateKey, clientEmail, clientID string, expiryDays int, backupPrefix string) error {
	this.Bucket = bucket
	this.ExpiryDays = expiryDays
	this.BackupPrefix = NormalizeBackupPrefix(backupPrefix)

	ctx = contextOrBackground(ctx)

	if endpoint != "" {
		client, err := newEmulatorStorageClient(ctx, endpoint)
		if err != nil {
			return err
		}

		this.StorageClient = client

		return nil
	}

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
		c, err := google.CredentialsFromJSON(ctx, jsonData, "https://www.googleapis.com/auth/cloud-platform")
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

func newEmulatorStorageClient(ctx context.Context, endpoint string) (*storage.Client, error) {
	return storage.NewClient(ctx, option.WithoutAuthentication(), option.WithEndpoint(normalizeEndpoint(endpoint)))
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	endpoint = strings.TrimSuffix(endpoint, "/upload/storage/v1")
	if strings.HasSuffix(endpoint, "/storage/v1") {
		return endpoint + "/"
	}
	endpoint = strings.TrimSuffix(endpoint, "/storage/v1")
	return endpoint + "/storage/v1/"
}

func (this *GcpStorage) Close() error {
	if this.StorageClient == nil {
		return nil
	}

	this.closeOnce.Do(func() {
		this.closeErr = this.StorageClient.Close()
	})

	return this.closeErr
}

func (this *GcpStorage) GetTargetObjectName(ctx context.Context, objectName string) (string, error) {
	ctx = contextOrBackground(ctx)

	if objectName == "" {
		return this.getLastUpdatedObjectName(ctx)
	}

	resolved, found, err := resolveExplicitObjectName(this.BackupPrefix, objectName, func(candidate string) (bool, error) {
		return this.gcpObjectExists(ctx, candidate)
	})
	if err != nil {
		return "", err
	}
	if found {
		return resolved, nil
	}

	return "", fmt.Errorf("object %q not found in bucket %q", objectName, this.Bucket)
}

func (this *GcpStorage) getLastUpdatedObjectName(ctx context.Context) (string, error) {
	var latest *objectTimestamp
	if err := this.gcpListObjectPages(ctx, func(candidates []objectTimestamp) bool {
		pageLatest, ok := latestEligibleObject(candidates, this.BackupPrefix)
		if ok {
			latest = chooseLaterObject(latest, pageLatest)
		}
		return true
	}); err != nil {
		return "", err
	}

	if latest == nil {
		return "", errors.New("no objects found in the bucket")
	}

	return latest.Name, nil
}

func generateServiceAccountKey(ctx context.Context, projectID, serviceAccountEmail string) ([]byte, error) {
	ctx = contextOrBackground(ctx)

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
func (this *GcpStorage) Upload(ctx context.Context, objectName string, filePath string) (string, error) {
	ctx = contextOrBackground(ctx)
	if this.uploadObject != nil {
		return this.uploadObject(ctx, objectName, filePath)
	}

	reader, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer reader.Close()

	wc := this.StorageClient.Bucket(this.Bucket).Object(objectName).NewWriter(ctx)

	if _, err := io.Copy(wc, reader); err != nil {
		_ = wc.Close()
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	attrs, err := this.getMetadata(ctx, objectName)
	if err != nil {
		return "", fmt.Errorf("failed to verify uploaded object: %w", err)
	}

	return attrs.Etag, nil
}

// See https://cloud.google.com/storage/docs/viewing-editing-metadata#storage-view-object-metadata-go
func (this *GcpStorage) getMetadata(ctx context.Context, objectName string) (*storage.ObjectAttrs, error) {
	obj := this.StorageClient.Bucket(this.Bucket).Object(objectName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		// storage: object doesn't exist
		return nil, err
	}

	return attrs, nil
}

// See https://cloud.google.com/storage/docs/downloading-objects#storage-download-object-go
func (this *GcpStorage) Download(ctx context.Context, objectName string, filePath string) error {
	ctx = contextOrBackground(ctx)

	return utils.WriteFileAtomically(filePath, func(dest *os.File) error {
		if this.downloadObject != nil {
			return this.downloadObject(ctx, objectName, dest)
		}

		obj := this.StorageClient.Bucket(this.Bucket).Object(objectName)

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
	})
}

func (this *GcpStorage) DeleteOldObjects(ctx context.Context, currentObjectName string) error {
	ctx = contextOrBackground(ctx)

	if this.ExpiryDays == 0 {
		return nil
	}

	now := time.Now()
	var pageErr error

	if err := this.gcpListObjectPages(ctx, func(candidates []objectTimestamp) bool {
		for _, obj := range candidates {
			daysOld := now.Sub(obj.ModifiedAt).Hours() / 24
			mlog.Logvf(mlog.Info, "Checking object: %s (%.1f days old)", obj.Name, daysOld)
		}

		pageErr = deleteExpiredObjects(candidates, this.BackupPrefix, this.ExpiryDays, now, currentObjectName, func(name string) error {
			err := this.gcpDeleteObject(ctx, name)
			if err == nil {
				mlog.Logvf(mlog.Info, "Deleted object: %s", name)
			}
			return err
		})
		return pageErr == nil
	}); err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}
	if pageErr != nil {
		return pageErr
	}

	return nil
}

func (this *GcpStorage) gcpObjectExists(ctx context.Context, name string) (bool, error) {
	if this.objectExists != nil {
		return this.objectExists(ctx, name)
	}

	if _, err := this.getMetadata(ctx, name); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to retrieve metadata: %w", err)
	}
	return true, nil
}

func (this *GcpStorage) gcpListObjectPages(ctx context.Context, handle func([]objectTimestamp) bool) error {
	if this.listObjectPages != nil {
		return this.listObjectPages(ctx, handle)
	}

	bucket := this.StorageClient.Bucket(this.Bucket)
	listOptions := newGCPListObjectsOptions(this.BackupPrefix)
	it := bucket.Objects(ctx, listOptions.Query)
	it.PageInfo().MaxSize = listOptions.PageSize

	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		if !handle([]objectTimestamp{{Name: objAttrs.Name, ModifiedAt: objAttrs.Updated}}) {
			break
		}
	}
	return nil
}

func (this *GcpStorage) gcpDeleteObject(ctx context.Context, name string) error {
	if this.deleteObject != nil {
		return this.deleteObject(ctx, name)
	}

	return this.StorageClient.Bucket(this.Bucket).Object(name).Delete(ctx)
}
