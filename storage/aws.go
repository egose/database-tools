package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/egose/database-tools/utils"
	mlog "github.com/mongodb/mongo-tools/common/log"
)

type AwsS3 struct {
	Endpoint         string
	AccessKeyId      string
	SecretAccessKey  string
	Region           string
	Bucket           string
	S3ForcePathStyle bool
	Session          *session.Session
	Service          *s3.S3
	ExpiryDays       int
	BackupPrefix     string
}

func (this *AwsS3) Init(endpoint string, accessKeyId string, secretAccessKey string, region string, bucket string, s3ForcePathStyle bool, expiryDays int, backupPrefix string) error {
	this.Endpoint = endpoint
	this.AccessKeyId = accessKeyId
	this.SecretAccessKey = secretAccessKey // pragma: allowlist secret
	this.Region = region
	this.Bucket = bucket
	this.S3ForcePathStyle = s3ForcePathStyle
	this.ExpiryDays = expiryDays
	this.BackupPrefix = NormalizeBackupPrefix(backupPrefix)

	creds := credentials.NewStaticCredentials(accessKeyId, secretAccessKey, "")
	config := &aws.Config{
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(s3ForcePathStyle),
		Region:           aws.String(region),
		Credentials:      creds,
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}
	this.Session = sess
	this.Service = s3.New(sess)

	return nil
}

func (this *AwsS3) GetTargetObjectName(ctx context.Context, objectKey string) (string, error) {
	ctx = contextOrBackground(ctx)

	if objectKey == "" {
		return this.getLastUpdatedObjectName(ctx)
	}

	resolved, found, err := resolveExplicitObjectName(this.BackupPrefix, objectKey, func(candidate string) (bool, error) {
		err := this.headObject(ctx, candidate)
		if err == nil {
			return true, nil
		}
		if isS3NotFound(err) {
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

	return "", fmt.Errorf("object %q not found in bucket %q", objectKey, this.Bucket)
}

func (this *AwsS3) getLastUpdatedObjectName(ctx context.Context) (string, error) {
	var latest objectTimestamp
	hasLatest := false

	err := this.Service.ListObjectsV2PagesWithContext(ctx, newS3ListObjectsInput(this.Bucket, this.BackupPrefix), func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		candidates := make([]objectTimestamp, 0, len(page.Contents))
		for _, obj := range page.Contents {
			if obj == nil || obj.Key == nil || obj.LastModified == nil {
				continue
			}
			candidates = append(candidates, objectTimestamp{Name: *obj.Key, ModifiedAt: *obj.LastModified})
		}

		pageLatest, ok := latestEligibleObject(candidates, this.BackupPrefix)
		if ok {
			latestPtr := chooseLaterObject(func() *objectTimestamp {
				if !hasLatest {
					return nil
				}
				latestCopy := latest
				return &latestCopy
			}(), pageLatest)
			if latestPtr != nil {
				latest = *latestPtr
				hasLatest = true
			}
		}

		return true
	})
	if err != nil {
		return "", fmt.Errorf("failed to list objects: %v", err)
	}

	if !hasLatest {
		return "", errors.New("no objects found in the bucket")
	}

	return latest.Name, nil
}

func (this *AwsS3) Upload(ctx context.Context, blobName string, filePath string) (string, error) {
	ctx = contextOrBackground(ctx)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	uploader := s3manager.NewUploader(this.Session)
	input := &s3manager.UploadInput{
		Bucket: aws.String(this.Bucket),
		Key:    aws.String(blobName),
		Body:   file,
	}

	output, err := uploader.UploadWithContext(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	if err := this.headObject(ctx, blobName); err != nil {
		return "", fmt.Errorf("failed to verify uploaded object: %w", err)
	}

	return *output.ETag, nil
}

func (this *AwsS3) Download(ctx context.Context, objectName string, filePath string) error {
	ctx = contextOrBackground(ctx)

	downloader := s3manager.NewDownloader(this.Session)
	return utils.WriteFileAtomically(filePath, func(dest *os.File) error {
		_, err := downloader.DownloadWithContext(ctx, dest, &s3.GetObjectInput{
			Bucket: aws.String(this.Bucket),
			Key:    aws.String(objectName),
		})
		if err != nil {
			return fmt.Errorf("failed to download object: %w", err)
		}
		return nil
	})
}

func (this *AwsS3) DeleteOldObjects(ctx context.Context, currentObjectName string) error {
	ctx = contextOrBackground(ctx)

	// If expiry days is not set, do not delete backups
	if this.ExpiryDays == 0 {
		return nil
	}

	svc := this.Service
	bucket := aws.String(this.Bucket)

	now := time.Now()
	var pageErr error

	err := svc.ListObjectsV2PagesWithContext(ctx, newS3ListObjectsInput(this.Bucket, this.BackupPrefix), func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		candidates := make([]objectTimestamp, 0, len(page.Contents))
		for _, obj := range page.Contents {
			if obj.LastModified == nil || obj.Key == nil {
				continue
			}

			daysOld := now.Sub(*obj.LastModified).Hours() / 24
			mlog.Logvf(mlog.Info, "Checking object: %s (%.1f days old)", *obj.Key, daysOld)
			candidates = append(candidates, objectTimestamp{Name: *obj.Key, ModifiedAt: *obj.LastModified})
		}

		pageErr = deleteExpiredObjects(candidates, this.BackupPrefix, this.ExpiryDays, now, currentObjectName, func(name string) error {
			_, delErr := svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
				Bucket: bucket,
				Key:    aws.String(name),
			})
			if delErr == nil {
				mlog.Logvf(mlog.Info, "Deleted object: %s", name)
			}
			return delErr
		})
		if pageErr != nil {
			return false
		}

		return true // continue paging
	})
	if pageErr != nil {
		return pageErr
	}

	if err != nil {
		return fmt.Errorf("error listing S3 objects: %w", err)
	}

	return nil
}

func (this *AwsS3) Close() error {
	return nil
}

func (this *AwsS3) headObject(ctx context.Context, objectKey string) error {
	_, err := this.Service.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(this.Bucket),
		Key:    aws.String(objectKey),
	})
	return err
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}

	var requestFailure awserr.RequestFailure
	if errors.As(err, &requestFailure) && requestFailure.StatusCode() == 404 {
		return true
	}

	var awsErr awserr.Error
	return errors.As(err, &awsErr) && awsErr.Code() == "NotFound"
}
