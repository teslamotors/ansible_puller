package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"os"
	"testing"
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
	s.tmpDir, err = ioutil.TempDir("/tmp", "ansible_puller")
	assert.Nil(s.T(), err)
}

func (s *UnarchiveTestSuite) TearDownTest() {
	os.RemoveAll(s.tmpDir)
}

func (s *UnarchiveTestSuite) TestTarballExists() {
	err := extractTgz("testdata/good.tgz", s.tmpDir)
	assert.Nil(s.T(), err)

	stats, err := os.Stat(s.tmpDir + "/foo.txt")
	assert.Nil(s.T(), err)
	assert.True(s.T(), stats.Mode().IsRegular(), "should create a regular foo file")

	stats, err = os.Stat(s.tmpDir + "/bar.txt")
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
