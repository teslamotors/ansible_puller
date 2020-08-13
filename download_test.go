package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

var (
	testFilename = "testfile.txt"
	testText     = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer ac condimentum velit, a hendrerit felis.")
	testMD5      = "7b20fda6af27c1b59ebdd8c09a93e770"

	testHashlessFilename = "nohash.txt"
	testHashlessText     = []byte("Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")

	testBasicAuthFilename = "basicauth.txt"
	testBasicAuthText     = []byte("Morbi pulvinar dui sit amet metus imperdiet lobortis sed at urna.")
	testBasicAuthUser     = "foo-user"
	testBasicAuthPass     = "bar&r6j6ehw5:qae^^%#$as45"
)

var _ = Describe("Download", func() {
	var testServer *httptest.Server

	BeforeEach(func() {
		testServer = httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, req *http.Request) {
					user, pass, ok := req.BasicAuth()
					if ok {
						if user == testBasicAuthUser && pass == testBasicAuthPass {
							rw.Write(testBasicAuthText)
							return
						}
						rw.WriteHeader(http.StatusUnauthorized)
						return
					}

					switch req.URL.String() {
					case "/" + testFilename:
						rw.Write(testText)
					case "/" + testFilename + ".md5":
						rw.Write([]byte(testMD5))
					case "/" + testHashlessFilename:
						rw.Write(testHashlessText)
					default:
						rw.Write([]byte("this is not the text you are looking for"))
					}
				}))
	})

	AfterEach(func() {
		testServer.Close()
		os.RemoveAll(testFilename)
		os.RemoveAll(testHashlessFilename)
		os.RemoveAll(testBasicAuthFilename)
	})

	Describe("checking an md5sum", func() {
		Context("from a file on the filesystem", func() {
			It("should return the correct hash", func() {
				ioutil.WriteFile(testFilename, testText, 0644)

				sum, err := md5sum(testFilename)
				Expect(err).To(BeNil())
				Expect(sum).To(Equal(testMD5))
			})
		})
	})

	Describe("downloading a file", func() {
		Context("when nothing currently exists on disk", func() {
			It("should download a new file", func() {
				err := downloadFile(testServer.URL+"/"+testFilename, testFilename, "", "")
				Expect(err).To(BeNil())

				text, err := ioutil.ReadFile(testFilename)
				Expect(err).To(BeNil())
				Expect(text).To(Equal(testText))
			})
		})
	})

	Describe("idempotently downloading a file", func() {
		Context("when nothing currently exists on disk", func() {
			It("should download a new file", func() {
				err := idempotentFileDownload(testServer.URL+"/"+testFilename, testFilename, "", "")
				Expect(err).To(BeNil())

				text, err := ioutil.ReadFile(testFilename)
				Expect(err).To(BeNil())
				Expect(text).To(Equal(testText))
			})
		})

		Context("when the file exists on disk", func() {
			It("should not download a new file", func() {
				err := idempotentFileDownload(testServer.URL+"/"+testFilename, testFilename, "", "")
				Expect(err).To(BeNil())

				text, err := ioutil.ReadFile(testFilename)
				Expect(err).To(BeNil())
				Expect(text).To(Equal(testText))

				// Get original file info
				finfo, err := os.Stat(testFilename)
				Expect(err).To(BeNil())

				modtime := finfo.ModTime()

				// wait a few milliseconds
				time.Sleep(50 * time.Millisecond)

				// Idempotent Download
				err = idempotentFileDownload(testServer.URL+"/"+testFilename, testFilename, "", "")
				Expect(err).To(BeNil())

				newFinfo, err := os.Stat(testFilename)
				newModtime := newFinfo.ModTime()

				// Make sure the file didn't change
				Expect(modtime).To(Equal(newModtime))
			})
		})

		Context("when the file exists on disk and has a different hash", func() {
			It("should download a new file", func() {
				err := idempotentFileDownload(testServer.URL+"/something-else", testFilename, "", "")
				Expect(err).To(BeNil())

				_, err = ioutil.ReadFile(testFilename)
				Expect(err).To(BeNil())

				// Get original file info
				finfo, err := os.Stat(testFilename)
				Expect(err).To(BeNil())

				modtime := finfo.ModTime()

				// wait a few milliseconds
				time.Sleep(50 * time.Millisecond)

				// Idempotent Download
				err = idempotentFileDownload(testServer.URL+"/"+testFilename, testFilename, "", "")
				Expect(err).To(BeNil())

				newFinfo, err := os.Stat(testFilename)
				newModtime := newFinfo.ModTime()

				// Make sure the file changed
				Expect(modtime).NotTo(Equal(newModtime))
			})
		})

		Context("when the hashless file exists on disk", func() {
			It("should download a new file", func() {
				err := idempotentFileDownload(testServer.URL+"/"+testHashlessFilename, testHashlessFilename, "", "")
				Expect(err).To(BeNil())

				text, err := ioutil.ReadFile(testHashlessFilename)
				Expect(err).To(BeNil())
				Expect(text).To(Equal(testHashlessText))

				// Get original file info
				finfo, err := os.Stat(testHashlessFilename)
				Expect(err).To(BeNil())

				modtime := finfo.ModTime()

				// wait a few milliseconds
				time.Sleep(50 * time.Millisecond)

				// Idempotent Download
				err = idempotentFileDownload(testServer.URL+"/"+testHashlessFilename, testHashlessFilename, "", "")
				Expect(err).To(BeNil())

				newFinfo, err := os.Stat(testHashlessFilename)
				newModtime := newFinfo.ModTime()

				// Make sure the file changed
				Expect(modtime).NotTo(Equal(newModtime))
			})
		})

		Context("with correct basic auth", func() {
			It("should download a file", func() {
				err := idempotentFileDownload(testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename, testBasicAuthUser, testBasicAuthPass)
				Expect(err).To(BeNil())

				text, err := ioutil.ReadFile(testBasicAuthFilename)
				Expect(err).To(BeNil())
				Expect(text).To(Equal(testBasicAuthText))
			})
		})

		Context("with incorrect basic auth", func() {
			It("should return an error", func() {
				err := idempotentFileDownload(testServer.URL+"/"+testBasicAuthFilename, testBasicAuthFilename, "nottherightuser", "nottherightpass")
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
