// Helper methods for downloading a file over HTTP

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

func s3Downloader(context context.Context) (*manager.Downloader, error) {
	// A default connection region should be selected based on the EC2
	// metadata by default. It ideally wouldn't matter because we're
	// only accessing S3 which is globally namespaced but we have to
	// consider connections orignating from China.
	// https://github.com/aws/aws-sdk-go-v2/pull/523
	config, err := config.LoadDefaultConfig(context)
	if err != nil {
		logrus.Debug("Error loading AWS config")
		return nil, err
	}

	// Sessions aren't required in the v2 of this SDK and
	// EC2 role credentials are loaded automatically as per:
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#iam-roles-for-amazon-ec2-instances

	client := s3.NewFromConfig(config)

	return manager.NewDownloader(client), nil
}

type s3BucketObject struct {
	Bucket string
	File string
}

func parseS3ResourceFromURI(resource string) (*s3BucketObject, error) {
	// The golang library seems to be missing the functionality to parse
	// bucket/object info from a URI
	regexURI := regexp.MustCompile(`^s3://([^/]+)/(.+)$`)
	matches := regexURI.FindStringSubmatch(resource)
	if matches != nil {
		return &s3BucketObject{
			Bucket: matches[1],
			File: matches[2],
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Not a valid S3 URI: %s", resource))
}

func parseS3ResourceFromARN(resource string) (*s3BucketObject, error) {
	// Support for object ARNs
	regexARN := regexp.MustCompile(`^arn:aws[\w-]*:s3:.*:.*:([^/]+)/(.+)$`)
	matches := regexARN.FindStringSubmatch(resource)
	if matches != nil {
		return &s3BucketObject{
			Bucket: matches[1],
			File: matches[2],
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Not a valid S3 ARN: %s", resource))
}

func downloadS3File(context context.Context, bucketObject *s3BucketObject, path string) (err error) {
	downloader, err := s3Downloader(context)
	if err != nil {
		return
	}

	file, err := os.Create(path)
	if err != nil {
		logrus.Errorf("Could not create file '%s' for writing", path)
		return
	}
	defer func() {
		closeError := file.Close()
		if err == nil {
			err = closeError
		}
	}()

	parameters := &s3.GetObjectInput{
		Bucket: aws.String(bucketObject.Bucket),
		Key:    aws.String(bucketObject.File),
	}
	numBytes, err := downloader.Download(context, file, parameters)
	if err != nil {
		logrus.Errorf("Could not download file '%s' from S3 bucket '%s'", bucketObject.Bucket, bucketObject.File)
		return
	}
	logrus.Debugf("Downloaded %d bytes from S3", numBytes)

	return
}
