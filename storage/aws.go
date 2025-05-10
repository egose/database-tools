package storage

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/egose/database-tools/utils"
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
}

func (this *AwsS3) Init(endpoint string, accessKeyId string, secretAccessKey string, region string, bucket string, s3ForcePathStyle bool, expiryDays int) error {
	this.Endpoint = endpoint
	this.AccessKeyId = accessKeyId
	this.SecretAccessKey = secretAccessKey
	this.Region = region
	this.Bucket = bucket
	this.S3ForcePathStyle = s3ForcePathStyle
	this.ExpiryDays = expiryDays

	creds := credentials.NewStaticCredentials(accessKeyId, secretAccessKey, "")
	config := &aws.Config{
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(s3ForcePathStyle),
		Region:           aws.String(region),
		Credentials:      creds,
	}

	sess := session.Must(session.NewSession(config))
	this.Session = sess
	this.Service = s3.New(sess)

	return nil
}

func (this *AwsS3) GetTargetObjectName(objectKey string) (string, error) {
	if objectKey == "" {
		return this.getLastUpdatedObjectName()
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(this.Bucket),
		Key:    aws.String(objectKey),
	}

	_, err := this.Service.HeadObject(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			return this.getLastUpdatedObjectName()
		}
		return "", fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	return objectKey, nil
}

func (this *AwsS3) getLastUpdatedObjectName() (string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(this.Bucket),
	}

	result, err := this.Service.ListObjectsV2(input)
	if err != nil {
		return "", fmt.Errorf("failed to list objects: %v", err)
	}

	if len(result.Contents) == 0 {
		return "", errors.New("no objects found in the bucket")
	}

	lastUpdatedObject := result.Contents[0]
	return *lastUpdatedObject.Key, nil
}

func (this *AwsS3) Upload(blobName string, buffer []byte) (string, error) {
	uploader := s3manager.NewUploader(this.Session)
	input := &s3manager.UploadInput{
		Bucket: aws.String(this.Bucket),
		Key:    aws.String(blobName),
		Body:   bytes.NewReader(buffer),
	}

	output, err := uploader.Upload(input)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	return *output.ETag, nil
}

func (this *AwsS3) Download(objectName string, filePath string) error {
	dest, err := utils.CreateFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	downloader := s3manager.NewDownloader(this.Session)
	_, err = downloader.Download(dest, &s3.GetObjectInput{
		Bucket: aws.String(this.Bucket),
		Key:    aws.String(objectName),
	})

	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

func (this *AwsS3) DeleteOldObjects() error {
	// If expiry days is not set, do not delete backups
	if this.ExpiryDays == 0 {
		return nil
	}

	svc := this.Service
	bucket := aws.String(this.Bucket)
	expiryDays := float64(this.ExpiryDays)

	var err error

	err = svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: bucket,
		Prefix: aws.String(""),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			// Safety check (should never be nil)
			if obj.LastModified == nil || obj.Key == nil {
				continue
			}

			daysOld := time.Since(*obj.LastModified).Hours() / 24
			fmt.Printf("Checking object: %s (%.1f days old)\n", *obj.Key, daysOld)

			if daysOld > expiryDays {
				_, delErr := svc.DeleteObject(&s3.DeleteObjectInput{
					Bucket: bucket,
					Key:    obj.Key,
				})

				if delErr != nil {
					fmt.Printf("Failed to delete object %s: %v\n", *obj.Key, delErr)
					continue
				}
				fmt.Printf("Deleted object: %s\n", *obj.Key)
			}
		}

		return true // continue paging
	})

	if err != nil {
		return fmt.Errorf("error listing S3 objects: %w", err)
	}

	return nil
}
