package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

var (
	testS3GoodARN = "arn:aws:s3:::test_bucket/some/file_name"
	testS3BadARN = "arn:aws:lambda:us-east-1:0123:function:function:1"

	testS3GoodURI = "s3://test_bucket/some/file_name"
	testS3BadURI ="s3://test_bucket"
)

// Register the below test suite
func TestDownloadS3TestSuite(t *testing.T) {
	suite.Run(t, new(DownloadS3TestSuite))
}

type DownloadS3TestSuite struct {
	suite.Suite
}

func (s *DownloadHTTPTestSuite) TestParseS3ResourceGoodARN() {
	bucketObject, err := parseS3ResourceFromARN(testS3GoodARN)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), bucketObject)
	assert.Equal(s.T(), "test_bucket", bucketObject.Bucket)
	assert.Equal(s.T(), "some/file_name", bucketObject.File)
}

func (s *DownloadHTTPTestSuite) TestParseS3ResourceBadARN() {
	bucketObject, err := parseS3ResourceFromARN(testS3BadARN)

	assert.Nil(s.T(), bucketObject)
	assert.NotNil(s.T(), err)
}

func (s *DownloadHTTPTestSuite) TestParseS3ResourceGoodURI() {
	bucketObject, err := parseS3ResourceFromURI(testS3GoodURI)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), bucketObject)
	assert.Equal(s.T(), "test_bucket", bucketObject.Bucket)
	assert.Equal(s.T(), "some/file_name", bucketObject.File)
}

func (s *DownloadHTTPTestSuite) TestParseS3ResourceBadURI() {
	bucketObject, err := parseS3ResourceFromURI(testS3BadURI)

	assert.Nil(s.T(), bucketObject)
	assert.NotNil(s.T(), err)
}
