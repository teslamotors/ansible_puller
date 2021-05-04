package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"

	"github.com/hashicorp/go-getter"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	testFilename     = "testfile.txt"
	testText         = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer ac condimentum velit, a hendrerit felis.")
	testFilenameHash = "testfile.txt.md5"
	testMD5          = "7b20fda6af27c1b59ebdd8c09a93e770"

	testHashlessFilename = "nohash.txt"
	testHashlessText     = []byte("Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")
)

// Register the below test suite
func TestGoGetterDownloaderTestSuite(t *testing.T) {
	suite.Run(t, new(GoGetterDownloaderTestSuite))
}

type GoGetterDownloaderTestSuite struct {
	suite.Suite
	fromDir    string
	toDir      string
	downloader goGetterDownloader
}

func (s *GoGetterDownloaderTestSuite) SetupTest() {
	fromDir, err := ioutil.TempDir("", "*")
	assert.Nil(s.T(), err)
	s.fromDir = fromDir
	toDir, err := ioutil.TempDir("", "*")
	assert.Nil(s.T(), err)
	s.toDir = toDir
	err = ioutil.WriteFile(filepath.Join(s.fromDir, testFilename), []byte(testText), 0644)
	assert.Nil(s.T(), err)
	err = ioutil.WriteFile(filepath.Join(s.fromDir, testFilenameHash), []byte(testMD5), 0644)
	assert.Nil(s.T(), err)
	err = ioutil.WriteFile(filepath.Join(s.fromDir, testHashlessFilename), []byte(testHashlessText), 0644)
	assert.Nil(s.T(), err)
	s.downloader = goGetterDownloader{}
	getter.Getters["file"].(*getter.FileGetter).Copy = true
}

func (s *GoGetterDownloaderTestSuite) TearDownTest() {
	os.RemoveAll(filepath.Join(s.fromDir, testFilename))
	os.RemoveAll(filepath.Join(s.toDir, testFilename))
	os.RemoveAll(filepath.Join(s.fromDir, testFilenameHash))
	os.RemoveAll(filepath.Join(s.toDir, testFilenameHash))
	os.RemoveAll(filepath.Join(s.fromDir, testHashlessFilename))
	os.RemoveAll(filepath.Join(s.toDir, testHashlessFilename))
}

func (s *GoGetterDownloaderTestSuite) TestMD5Sum() {
	err := ioutil.WriteFile(filepath.Join(s.toDir, testFilename), testText, 0644)
	assert.Nil(s.T(), err)

	sum, err := md5sum(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), sum, testMD5)
}

func (s *GoGetterDownloaderTestSuite) TestIdempotentDownloadWhenNoFileExists() {
	err := idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testFilename), filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText)
}

func (s *GoGetterDownloaderTestSuite) TestIdempotentDownloadWhenCurrentFileExists() {
	err := idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testFilename), filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), text, testText, "file should download correctly")

	// Get original file info
	finfo, err := os.Stat(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// Idempotent Download
	err = idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testFilename), filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(filepath.Join(s.toDir, testFilename))
	newModtime := newFinfo.ModTime()

	// Make sure the file didn't change
	assert.Equal(s.T(), modtime, newModtime, "modification time should not change")
}

func (s *GoGetterDownloaderTestSuite) TestIdempotentDownloadWhenOldFileExists() {
	err := idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testHashlessFilename), filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	_, err = ioutil.ReadFile(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	// Get original file info
	finfo, err := os.Stat(filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// on linux time is in seconds, so we needs to wait at least one
	time.Sleep(1 * time.Second)

	// Idempotent Download
	err = idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testFilename), filepath.Join(s.toDir, testFilename))
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(filepath.Join(s.toDir, testFilename))
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "modification time should change")
}

func (s *GoGetterDownloaderTestSuite) TestIdempotentDownloadNoRemoteHash() {
	err := idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testHashlessFilename), filepath.Join(s.toDir, testHashlessFilename))
	assert.Nil(s.T(), err)

	text, err := ioutil.ReadFile(filepath.Join(s.toDir, testHashlessFilename))
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), testHashlessText, text)

	// Get original file info
	finfo, err := os.Stat(filepath.Join(s.toDir, testHashlessFilename))
	assert.Nil(s.T(), err)

	modtime := finfo.ModTime()

	// on linux time is in seconds, so we need to wait at least one
	time.Sleep(1 * time.Second)

	// Idempotent Download
	err = idempotentFileDownload(s.downloader, filepath.Join(s.fromDir, testHashlessFilename), filepath.Join(s.toDir, testHashlessFilename))
	assert.Nil(s.T(), err)

	newFinfo, err := os.Stat(filepath.Join(s.toDir, testHashlessFilename))
	newModtime := newFinfo.ModTime()

	// Make sure the file changed
	assert.NotEqual(s.T(), modtime, newModtime, "new file should be downloaded")
}
