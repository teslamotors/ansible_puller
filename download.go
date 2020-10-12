// Helper methods for downloading a file over HTTP

package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Calculates the md5sum of a local file
func md5sum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	byteHash := hash.Sum(nil)[:16]
	return hex.EncodeToString(byteHash), nil
}

// Simply downloads a file
//
// If user and pass are both supplied, it will set basic auth
func downloadFile(url, path, user, pass string) error {
	outFile, err := os.Create(path)

	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 15 * time.Second,
	}

	req, _ := http.NewRequest("GET", url, nil)
	if user != "" && pass != "" {
		req.SetBasicAuth(user, pass)
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

// Downloads a file from a given url to a local filepath
// Checks the md5sum of the file to see if the remote file should be downloaded
//
// The MD5 checking may be an Artifactory-specific setup because it will look for the hash at "${url}.md5"
// If the MD5 is not found, this will download the file
//
// For Basic Auth to be used, the user and pass params must both not be ""
//
// url is the fully qualified web address for the host
// path is the local path to which the remote file is downloaded
// user is an http basic auth username
// pass is an http basic auth password
func idempotentFileDownload(url, path, user, pass string) error {
	logrus.Debugf("Starting idempotent download of %s", url)
	hashURL := url + ".md5"

	currentFileExists := true
	currentChecksum, err := md5sum(path)
	if os.IsNotExist(err) {
		currentFileExists = false
	} else if err != nil {
		return err
	}

	if currentFileExists {
		logrus.Debug("File exists locally")

		timeout := time.Duration(2 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		req, _ := http.NewRequest("GET", hashURL, nil)
		if user != "" && pass != "" {
			req.SetBasicAuth(user, pass)
		}
		md5Resp, err := client.Do(req)

		if err != nil {
			logrus.Debugf("MD5 sum not found at: %s", hashURL)
		} else {
			logrus.Debugf("Found MD5 sum at: %s", hashURL)
			defer md5Resp.Body.Close()
			remoteChecksum, err := ioutil.ReadAll(md5Resp.Body)
			if err != nil {
				logrus.Debug("Error reading remote checksum")
				return err
			}

			logrus.Debugf("Local checksum:  %s", currentChecksum)
			logrus.Debugf("Remote checksum: %s", remoteChecksum)

			// Checksums match, so there's nothing to do here
			if string(remoteChecksum) == currentChecksum {
				logrus.Debug("Local and remote checksums match, skipping file download")
				return nil
			}
		}
	}

	logrus.Debugf("Downloading file: %s", url)
	err = downloadFile(url, path, user, pass)
	if err != nil {
		return err
	}

	return nil
}
