// Functions for expanding gzipped tarballs

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Ensures that a given file is gzip-encoded
func ensureGzip(file io.Reader) error {
	buff := make([]byte, 512) // content-type length

	_, err := file.Read(buff)
	if err != nil {
		return errors.Wrap(err, "unable to detect encoding")
	}

	if filetype := http.DetectContentType(buff); filetype != "application/x-gzip" {
		return fmt.Errorf("file not gzip encoded")
	}

	return nil
}

// Extract a tarball from the src into dest
func extractTgz(src, dest string) error {
	tgzFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "unable to open source file")
	}
	defer tgzFile.Close()

	err = ensureGzip(tgzFile)
	if err != nil {
		return errors.Wrap(err, "unable to make gzip reader")
	}
	tgzFile.Seek(0, 0)

	uncompressedStream, err := gzip.NewReader(tgzFile)
	if err != nil {
		errors.Wrap(err, "unable to make gzip reader")
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if err := os.Mkdir(dest, 0755); err != nil {
			return errors.Wrap(err, "unable to create target directory")
		}
	}

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return errors.Wrap(err, "unable to extract gzipped tarfile")
		case header == nil:
			continue // phantom file case
		}

		targetPath := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				if err := os.Mkdir(targetPath, 0755); err != nil {
					return errors.Wrap(err, "unable to create dir from tar")
				}
			}

		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, "unable to create file from tar")
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "unable to populate file from tar")
			}

			outFile.Close()
		}
	}
}
