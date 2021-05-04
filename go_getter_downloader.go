package main

import (
	"fmt"
	"github.com/hashicorp/go-getter"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
)

type goGetterDownloader struct {
	downloader
}

func (downloader goGetterDownloader) Download(remotePath, outputPath string) error {
	return getter.GetFile(outputPath, remotePath)
}

func (downloader goGetterDownloader) DownloadAndValidateChecksum(remotePath, outputPath, checksum string) error {
	return getter.GetFile(outputPath, fmt.Sprintf("%s?checksum=md5:%s", remotePath, checksum))
}

func (downloader goGetterDownloader) RemoteChecksum(remotePath string) (string, error) {
	hashRemotePath := fmt.Sprintf("%s.md5", remotePath)

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		logrus.Fatal(err)
	}
	defer os.RemoveAll(dir)
	hashFile := filepath.Join(dir, "md5Hash")

	err = getter.GetFile(hashFile, hashRemotePath)
	if err != nil {
		logrus.Infof("MD5 sum not reachable. %v", err)
		return "", nil
	}

	logrus.Debugf("Found MD5 sum at: %s", hashRemotePath)

	content, err := ioutil.ReadFile(hashFile)
	if err != nil {
		logrus.Debug("Error reading remote checksum")
		return "", err
	}

	return string(content), nil
}
