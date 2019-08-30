package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("Unarchive", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "ansible_puller")
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("unarchiving a tarball", func() {
		Context("when the tarball exists", func() {
			It("unarchives into the expected directory", func() {
				err := extractTgz("testdata/good.tgz", tmpDir)
				Expect(err).To(BeNil())

				stats, err := os.Stat(tmpDir + "/foo.txt")
				Expect(err).To(BeNil())
				Expect(stats.Mode().IsRegular()).To(BeTrue())

				stats, err = os.Stat(tmpDir + "/bar.txt")
				Expect(err).To(BeNil())
				Expect(stats.Mode().IsRegular()).To(BeTrue())
			})
		})

		Context("when the tarball does not exist", func() {
			It("fails with an error", func() {
				err := extractTgz("testdata/somethingthatdoesnotexist.tgz", tmpDir)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when the tarball is corrupted", func() {
			It("fails with an error", func() {
				err := extractTgz("testdata/corrupt.tgz", tmpDir)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when the tarball has correct headers but invalid body", func() {
			It("fails with an error", func() {
				err := extractTgz("testdata/half.tgz", tmpDir)
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
