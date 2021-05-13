package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

// Interface with logic to govern how to actually pull objects
type downloader interface {
	Download(remotePath, outputPath string) error
	RemoteChecksum(remotePath string) (string, error)
}

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

func validateMd5Sum(path, checksum string) error {
	newChecksum, err := md5sum(path)
	if err != nil {
		return err
	}

	if newChecksum != checksum {
		logrus.Debugf("Checksums for downloaded file do not match: '%s' != '%s'", newChecksum, checksum)
		return errors.New("checksum does not match expected value")
	}

	return nil
}

// Downloads a file from a given url to a local filepath
// Checks the md5sum of the file to see if the remote file should be downloaded
//
// The MD5 checking may be an Artifactory-specific setup because it will look for the hash at "${url}.md5"
// If the MD5 is not found, this will download the file
func idempotentFileDownload(downloader downloader, remotePath, localPath string) error {
	logrus.Debugf("Starting idempotent download of %s to %s", remotePath, localPath)

	currentChecksum, err := md5sum(localPath)
	if os.IsNotExist(err) {
		logrus.Infof("File '%s' does not exist yet so cannot validate for new checksum", localPath)
		currentChecksum = ""
	} else if err != nil {
		return err
	}

	remoteChecksum, err := downloader.RemoteChecksum(remotePath)
	if err != nil {
		return err
	}

	if currentChecksum != "" && remoteChecksum != "" {
		logrus.Debugf("Local checksum:  %s", currentChecksum)
		logrus.Debugf("Remote checksum: %s", remoteChecksum)
		if remoteChecksum == currentChecksum {
			logrus.Debug("Local and remote checksums match, skipping file download")
			return nil
		}
	}

	logrus.Infof("Downloading file: %s", remotePath)
	err = downloader.Download(remotePath, localPath)
	if err != nil {
		return err
	}

	if remoteChecksum != "" {
		logrus.Infof("Validating checksum: %s", remotePath)

		err = validateMd5Sum(localPath, remoteChecksum)
		if err != nil {
			return err
		}
	}

	return nil
}
