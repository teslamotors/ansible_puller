package main

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ansible", func() {
	Describe("ansible inventory and execution", func() {
		Context("when assembling a target list", func() {
			It("should include an ip and the hostame at least", func() {
				targets, err := CreateAnsibleTargetsList()
				hostname, _ = os.Hostname()
				Expect(err).To(BeNil())
				Expect(targets).ToNot(BeNil())
				Expect(targets).Should(ContainElement(ContainSubstring(hostname)))
				Expect(targets).Should(ContainElement(MatchRegexp(`\d+\.\d+\.\d+.\d+`)))
			})
		})
	})
})
