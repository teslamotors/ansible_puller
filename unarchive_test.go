package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Register the below test suite
func TestUnarchiveTestSuite(t *testing.T) {
	suite.Run(t, new(UnarchiveTestSuite))
}

type UnarchiveTestSuite struct {
	suite.Suite
	tmpDir string
}

func (s *UnarchiveTestSuite) SetupTest() {
	var err error
	s.tmpDir, err = ioutil.TempDir("", "ansible_puller")
	assert.Nil(s.T(), err)
}

func (s *UnarchiveTestSuite) TearDownTest() {
	os.RemoveAll(s.tmpDir)
}

func (s *UnarchiveTestSuite) TestTarballExists() {
	err := extractTgz("testdata/good.tgz", s.tmpDir)
	assert.Nil(s.T(), err)

	stats, err := os.Stat(filepath.Join(s.tmpDir, "foo.txt"))
	assert.Nil(s.T(), err)
	assert.True(s.T(), stats.Mode().IsRegular(), "should create a regular foo file")

	stats, err = os.Stat(filepath.Join(s.tmpDir, "bar.txt"))
	assert.Nil(s.T(), err)
	assert.True(s.T(), stats.Mode().IsRegular(), "should create a regular bar file")
}

func (s *UnarchiveTestSuite) TestTarballDoesNotExist() {
	err := extractTgz("testdata/somethingthatdoesnotexist.tgz", s.tmpDir)
	assert.NotNil(s.T(), err)
}

func (s *UnarchiveTestSuite) TestTarballIsCorrupted() {
	err := extractTgz("testdata/corrupt.tgz", s.tmpDir)
	assert.NotNil(s.T(), err)
}

func (s *UnarchiveTestSuite) TestTarballHasInvalidBody() {
	err := extractTgz("testdata/half.tgz", s.tmpDir)
	assert.NotNil(s.T(), err)
}

func (s *UnarchiveTestSuite) TestTarballPathTraversalRejected() {
	// Generate an in-memory tar.gz with a path traversal filename
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("traversal test")
	hdr := &tar.Header{
		Name:     "../escaped.txt",
		Mode:     0600,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	err := tw.WriteHeader(hdr)
	assert.Nil(s.T(), err)
	_, err = tw.Write(content)
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), tw.Close())
	assert.Nil(s.T(), gw.Close())

	maliciousTgz := filepath.Join(s.tmpDir, "malicious.tgz")
	err = ioutil.WriteFile(maliciousTgz, buf.Bytes(), 0644)
	assert.Nil(s.T(), err)

	extractDir := filepath.Join(s.tmpDir, "extract")
	err = extractTgz(maliciousTgz, extractDir)
	assert.NotNil(s.T(), err, "should reject path traversal in tarball")
	assert.Contains(s.T(), err.Error(), "illegal file path in tar archive")
}
