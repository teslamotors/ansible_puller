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
				targets, targetErr := CreateAnsibleTargetsList()
				hostname, hostnameErr := os.Hostname()
				Expect(targetErr).To(BeNil())
				Expect(hostnameErr).To(BeNil())
				Expect(len(targets)).Should(BeNumerically(">=", 2))
				Expect(targets).Should(ContainElement(ContainSubstring(hostname)))
				Expect(targets).Should(ContainElement(MatchRegexp(`\d+\.\d+\.\d+.\d+`)))
			})
		})
	})
})
