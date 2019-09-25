package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Http", func() {
	Describe("http server", func() {
		Context("when functioning normally", func() {
			It("should include an ip and the hostame at least", func() {
				req, err := http.NewRequest("GET", "/ansible/status", nil)
				Expect(err).ShouldNot(HaveOccurred())

				rr := httptest.NewRecorder()

				http.HandlerFunc(HandlerStatus).ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK), "Received bad status code %s", rr.Code)

				host, err := os.Hostname()
				Expect(err).ShouldNot(HaveOccurred())

				expected := string(fmt.Sprintf(`
				{
					"ansible_disabled": true,
					"ansible_last_run_success": true,
					"ansible_running": false,
					"app_name": "ansible-puller",
					"hostname": "%s",
					"version": ""
				}`, host))
				Expect(rr.Body).To(MatchJSON(expected))
			})
		})
	})
})
