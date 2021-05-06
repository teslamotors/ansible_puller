package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

var (
	testS3GoodARN = "arn:aws:s3:::test_bucket/some/file_name"
	testS3BadARN  = "arn:aws:lambda:us-east-1:0123:function:function:1"

	testS3GoodURI = "s3://test_bucket/some/file_name"
	testS3BadURI  = "s3://test_bucket"
)

// Register the below test suite
func TestDownloadS3TestSuite(t *testing.T) {
	suite.Run(t, new(S3DownloaderTestSuite))
}

type S3DownloaderTestSuite struct {
	suite.Suite
}

func (s *S3DownloaderTestSuite) TestParseS3ResourceGoodARN() {
	bucketObject, err := parseS3ResourceFromARN(testS3GoodARN)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), bucketObject)
	assert.Equal(s.T(), "test_bucket", bucketObject.Bucket)
	assert.Equal(s.T(), "some/file_name", bucketObject.File)
}

func (s *S3DownloaderTestSuite) TestParseS3ResourceBadARN() {
	bucketObject, err := parseS3ResourceFromARN(testS3BadARN)

	assert.Nil(s.T(), bucketObject)
	assert.NotNil(s.T(), err)
}
