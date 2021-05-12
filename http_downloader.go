// Helper methods for downloading a file over HTTP

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type httpDownloader struct {
	downloader
	username string
	password string
}

func (downloader httpDownloader) Download(remotePath, outputPath string) error {
	outFile, err := os.Create(outputPath)

	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 15 * time.Second,
	}

	req, _ := http.NewRequest("GET", remotePath, nil)
	if downloader.username != "" && downloader.password != "" {
		req.SetBasicAuth(downloader.username, downloader.password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return errors.New(fmt.Sprintf("bad status code: %v", resp.StatusCode))
	}

	// Persist to file in 32K chunks, instead of slurping
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}
	if err := outFile.Close(); err != nil {
		logrus.Errorf("Failed to close file: %v", err)
	}

	return nil
}

func (downloader httpDownloader) RemoteChecksum(remotePath string) (string, error) {
	hashRemotePath := fmt.Sprintf("%s.md5", remotePath)

	timeout := time.Duration(2 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	req, _ := http.NewRequest("GET", hashRemotePath, nil)
	if downloader.username != "" && downloader.password != "" {
		req.SetBasicAuth(downloader.username, downloader.password)
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Debugf("MD5 sum not found at: %s", hashRemotePath)
	}

	logrus.Debugf("Found MD5 sum at: %s", hashRemotePath)
	defer resp.Body.Close()
	remoteChecksum, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Debug("Error reading remote checksum")
		return "", err
	}

	return string(remoteChecksum), nil
}
