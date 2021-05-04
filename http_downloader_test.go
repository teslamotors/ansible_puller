package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
)

var (
	testBasicAuthChecksum     = []byte("bb651e9638be48e76bbbe936b9651083")
	testBasicAuthFilename     = "basicauth.txt"
	testBasicAuthFilenameHash = "basicauth.txt.md5"
	testBasicAuthText         = []byte("Morbi pulvinar dui sit amet metus imperdiet lobortis sed at urna.")
	testBasicAuthUser         = "foo-user"
	testBasicAuthPass         = "bar&r6j6ehw5:qae^^%#$as45"
)

// Register the below test suite
func TestHttpDownloaderTestSuite(t *testing.T) {
	suite.Run(t, new(HttpDownloaderTestSuite))
}

type HttpDownloaderTestSuite struct {
	suite.Suite
	testServer *httptest.Server
}

func (s *HttpDownloaderTestSuite) SetupTest() {
	s.testServer = httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, req *http.Request) {
				user, pass, ok := req.BasicAuth()
				if ok {
					if user != testBasicAuthUser || pass != testBasicAuthPass {
						rw.WriteHeader(http.StatusUnauthorized)
						return
					}
					switch req.URL.String() {
					case "/" + testBasicAuthFilename:
						rw.Write(testBasicAuthText)
					case "/" + testBasicAuthFilenameHash:
						rw.Write(testBasicAuthChecksum)
					default:
						rw.WriteHeader(404)
					}
					return
				}

				switch req.URL.String() {
				case "/" + testFilename:
					rw.Write(testText)
				case "/" + testFilenameHash:
					rw.Write([]byte(testMD5))
				case "/" + testHashlessFilename:
					rw.Write(testHashlessText)
				default:
					rw.WriteHeader(404)
				}
			}))
}

func (s *HttpDownloaderTestSuite) TearDownTest() {
	s.testServer.Close()
	os.RemoveAll(testFilename)
	os.RemoveAll(testHashlessFilename)
	os.RemoveAll(testBasicAuthFilename)
}

func (s *HttpDownloaderTestSuite) TestDownloadFile() {
	downloader := httpDownloader{
		username: "",
		password: "",
	}
	err := downloader.Download(s.testServer.URL+"/"+testFilename, testFilename)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText)
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadWhenNoFileExists() {
	downloader := httpDownloader{
		username: "",
		password: "",
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testFilename, testFilename)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText)
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadWhenCurrentFileExists() {
	downloader := httpDownloader{
		username: "",
		password: "",
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testFilename, testFilename)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText, "file should download correctly")

	// Get original file info
	finfo, err := os.Stat(testFilename)
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// Idempotent Download
	err = idempotentFileDownload(downloader, s.testServer.URL+"/"+testFilename, testFilename)
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file didn't change
	assert.Equal(s.T(), modtime, newModtime, "modification time should not change")
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadWhenOldFileExists() {
	downloader := httpDownloader{
		username: "",
		password: "",
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testHashlessFilename, testFilename)
	assert.Nil(s.T(), err)

	_, err = ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)

	// Get original file info
	finfo, err := os.Stat(testFilename)
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// on linux time is in seconds, so we needs to wait at least one
	time.Sleep(1 * time.Second)

	// Idempotent Download
	err = idempotentFileDownload(downloader, s.testServer.URL+"/"+testFilename, testFilename)
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "modification time should change")
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadNoRemoteHash() {
	downloader := httpDownloader{
		username: "",
		password: "",
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testHashlessFilename, testHashlessFilename)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testHashlessFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), testHashlessText, text)

	// Get original file info
	finfo, err := os.Stat(testHashlessFilename)
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// on linux time is in seconds, so we needs to wait at least one
	time.Sleep(1 * time.Second)

	// Idempotent Download
	err = idempotentFileDownload(downloader, s.testServer.URL+"/"+testHashlessFilename, testHashlessFilename)
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testHashlessFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "new file should be downloaded")
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadBasicAuth() {
	downloader := httpDownloader{
		username: testBasicAuthUser,
		password: testBasicAuthPass,
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testBasicAuthFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), testBasicAuthText, text, "text should be equal")
}

func (s *HttpDownloaderTestSuite) TestIdempotentDownloadBasicAuthFailure() {
	downloader := httpDownloader{
		username: "nottherightuser",
		password: "nottherightpass",
	}
	err := idempotentFileDownload(downloader, s.testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename)
	assert.NotNil(s.T(), err)
}
