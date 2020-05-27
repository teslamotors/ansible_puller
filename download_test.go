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
	testFilename = "testfile.txt"
	testText     = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer ac condimentum velit, a hendrerit felis.")
	testMD5      = "7b20fda6af27c1b59ebdd8c09a93e770"

	testHashlessFilename = "nohash.txt"
	testHashlessText     = []byte("Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")

	testBasicAuthFilename = "basicauth.txt"
	testBasicAuthText     = []byte("Morbi pulvinar dui sit amet metus imperdiet lobortis sed at urna.")
	testBasicAuthUser     = "foo-user"
	testBasicAuthPass     = "bar&r6j6ehw5:qae^^%#$as45"
)

// Register the below test suite
func TestDownloadTestSuite(t *testing.T) {
	suite.Run(t, new(DownloadTestSuite))
}

type DownloadTestSuite struct {
	suite.Suite
	testServer *httptest.Server
}

func (s *DownloadTestSuite) SetupTest() {
	s.testServer = httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, req *http.Request) {
				user, pass, ok := req.BasicAuth()
				if ok {
					if user == testBasicAuthUser && pass == testBasicAuthPass {
						rw.Write(testBasicAuthText)
						return
					}
					rw.WriteHeader(http.StatusUnauthorized)
					return
				}

				switch req.URL.String() {
				case "/" + testFilename:
					rw.Write(testText)
				case "/" + testFilename + ".md5":
					rw.Write([]byte(testMD5))
				case "/" + testHashlessFilename:
					rw.Write(testHashlessText)
				default:
					rw.Write([]byte("this is not the text you are looking for"))
				}
			}))
}

func (s *DownloadTestSuite) TearDownTest() {
	s.testServer.Close()
	os.RemoveAll(testFilename)
	os.RemoveAll(testHashlessFilename)
	os.RemoveAll(testBasicAuthFilename)
}

func (s *DownloadTestSuite) TestMD5Sum() {
	err := ioutil.WriteFile(testFilename, testText, 0644)
	assert.Nil(s.T(), err)

	sum, err := md5sum(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), sum, testMD5)
}

func (s *DownloadTestSuite) TestDownloadFile() {
	err := downloadFile(s.testServer.URL+"/"+testFilename, testFilename, "", "")
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText)
}

func (s *DownloadTestSuite) TestIdempotentDownloadWhenNoFileExists() {
	err := idempotentFileDownload(s.testServer.URL+"/"+testFilename, testFilename, "", "")
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText)
}

func (s *DownloadTestSuite) TestIdempotentDownloadWhenCurrentFileExists() {
	err := idempotentFileDownload(s.testServer.URL+"/"+testFilename, testFilename, "", "")
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText, "file should download correctly")

	// Get original file info
	finfo, err := os.Stat(testFilename)
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// Idempotent Download
	err = idempotentFileDownload(s.testServer.URL+"/"+testFilename, testFilename, "", "")
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file didn't change
	assert.Equal(s.T(), modtime, newModtime, "modification time should not change")
}

func (s *DownloadTestSuite) TestIdempotentDownloadWhenOldFileExists() {
	err := idempotentFileDownload(s.testServer.URL+"/something-else", testFilename, "", "")
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
	err = idempotentFileDownload(s.testServer.URL+"/"+testFilename, testFilename, "", "")
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "modification time should change")
}

func (s *DownloadTestSuite) TestIdempotentDownloadNoRemoteHash() {
	err := idempotentFileDownload(s.testServer.URL+"/"+testHashlessFilename, testHashlessFilename, "", "")
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
	err = idempotentFileDownload(s.testServer.URL+"/"+testHashlessFilename, testHashlessFilename, "", "")
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(testHashlessFilename)
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "new file should be downloaded")
}

func (s *DownloadTestSuite) TestIdempotentDownloadBasicAuth() {
	err := idempotentFileDownload(s.testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename, testBasicAuthUser, testBasicAuthPass)
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(testBasicAuthFilename)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), testBasicAuthText, text, "text should be equal")
}

func (s *DownloadTestSuite) TestIdempotentDownloadBasicAuthFailure() {
	err := idempotentFileDownload(s.testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename, "nottherightuser", "nottherightpass")
	assert.NotNil(s.T(), err)
}
