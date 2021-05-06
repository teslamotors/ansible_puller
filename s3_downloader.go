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
	"io/ioutil"
	"path/filepath"
)

type s3Downloader struct {
	downloader
	manager *manager.Downloader
}

type s3BucketObject struct {
	Bucket string
	File   string
}

func parseS3ResourceFromARN(resource string) (*s3BucketObject, error) {
	// Support for object ARNs
	regexARN := regexp.MustCompile(`^arn:aws[\w-]*:s3:.*:.*:([^/]+)/(.+)$`)
	matches := regexARN.FindStringSubmatch(resource)
	if matches != nil {
		return &s3BucketObject{
			Bucket: matches[1],
			File:   matches[2],
		}, nil
	}

	return nil, errors.New(fmt.Sprintf("Not a valid S3 ARN: %s", resource))
}

func createS3Downloader(regionOverride string) (*s3Downloader, error) {
	ctx := context.TODO()
	// A default connection region should be selected based on the EC2
	// metadata by default. It ideally wouldn't matter because we're
	// only accessing S3 which is globally namespaced but we have to
	// consider connections orignating from China.
	// https://github.com/aws/aws-sdk-go-v2/pull/523
	var awsConfig aws.Config
	var err error
	if regionOverride != "" {
		awsConfig, err = config.LoadDefaultConfig(ctx, config.WithRegion(regionOverride))
	} else {
		awsConfig, err = config.LoadDefaultConfig(ctx)
	}
	if err != nil {
		logrus.Warn("Error loading AWS config")
		return nil, err
	}

	// Sessions aren't required in the v2 of this SDK and
	// EC2 role credentials are loaded automatically as per:
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#iam-roles-for-amazon-ec2-instances

	client := s3.NewFromConfig(awsConfig)

	manager := manager.NewDownloader(client)

	return &s3Downloader{
		manager: manager,
	}, nil
}

func (downloader s3Downloader) Download(remotePath, outputPath string) (err error) {
	ctx := context.TODO()
	bucketObject, err := parseS3ResourceFromARN(remotePath)
	if err != nil {
		return
	}

	file, err := os.Create(outputPath)
	if err != nil {
		logrus.Warnf("Could not create file '%s' for writing", outputPath)
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
	numBytes, err := downloader.manager.Download(ctx, file, parameters)
	if err != nil {
		logrus.Warnf("Could not download file '%s' from S3 bucket '%s': %v", bucketObject.File, bucketObject.Bucket, err)
		return
	}
	logrus.Debugf("Downloaded %d bytes from S3", numBytes)

	return
}

func (downloader s3Downloader) RemoteChecksum(remotePath string) (string, error) {
	hashRemotePath := fmt.Sprintf("%s.md5", remotePath)

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		logrus.Error("Cannot create temporary file to store remote checksum")
		return "", err
	}
	defer os.RemoveAll(dir)
	hashFile := filepath.Join(dir, "md5Hash")

	err = downloader.Download(hashFile, hashRemotePath)
	if err != nil {
		logrus.Infof("MD5 sum not reachable. %v", err)
		return "", nil
	}

	logrus.Infof("Found MD5 sum at: %s", hashRemotePath)

	content, err := ioutil.ReadFile(hashFile)
	if err != nil {
		logrus.Warn("Error reading remote checksum")
		return "", err
	}

	return string(content), nil
}
