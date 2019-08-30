package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAnsiblePuller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ansible Puller Suite")
}
