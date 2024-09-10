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
	username    string
	password    string
	headerName  string
	headerValue string
}

func (downloader httpDownloader) Download(remotePath, outputPath string) error {
	outFile, err := os.Create(outputPath)

	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", remotePath, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	if downloader.username != "" && downloader.password != "" {
		req.SetBasicAuth(downloader.username, downloader.password)
	}
	if downloader.headerName != "" && downloader.headerValue != "" {
		req.Header.Set(downloader.headerName, downloader.headerValue)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("bad status code: %v", resp.StatusCode)
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

func (downloader httpDownloader) RemoteChecksum(checksumURL string) (string, error) {

	timeout := time.Duration(2 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest("GET", checksumURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	if downloader.username != "" && downloader.password != "" {
		req.SetBasicAuth(downloader.username, downloader.password)
	}
	if downloader.headerName != "" && downloader.headerValue != "" {
		req.Header.Set(downloader.headerName, downloader.headerValue)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to get remote md5sum")
	}
	// Ignore the checksum if it's not found, as assumed by the caller of this function.
	if resp.StatusCode == http.StatusNotFound {
		logrus.Debugf("MD5 sum not found at: %s", checksumURL)
		return "", nil
	}
	// A non-2xx status code does not cause an error, so we handle it here. https://pkg.go.dev/net/http#Client.Do
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status code: %v", resp.StatusCode)
	}

	logrus.Debugf("Found MD5 sum at: %s", checksumURL)
	defer resp.Body.Close()
	remoteChecksum, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Debug("Error reading remote checksum")
		return "", errors.Wrap(err, "failed to read remote md5sum")
	}

	return string(remoteChecksum), nil
}
