package storage

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/egose/database-tools/utils"
)

type AwsS3 struct {
	AccessKeyId     string
	SecretAccessKey string
	Region          string
	Bucket          string
	Session         *session.Session
	Service         *s3.S3
}

func (this *AwsS3) Init(accessKeyId string, secretAccessKey string, region string, bucket string) error {
	this.AccessKeyId = accessKeyId
	this.SecretAccessKey = secretAccessKey
	this.Region = region
	this.Bucket = bucket

	creds := credentials.NewStaticCredentials(accessKeyId, secretAccessKey, "")
	config := &aws.Config{
		Region:      aws.String(region),
		Credentials: creds,
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
