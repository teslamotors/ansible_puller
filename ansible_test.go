package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostnameLookup(t *testing.T) {
	targets, err := CreateAnsibleTargetsList()
	assert.Nil(t, err)

	hostname, err := os.Hostname()
	assert.Nil(t, err)

	assert.GreaterOrEqual(t, len(targets), 2, "should have at least hostname and ip, so 2")
	assert.Contains(t, targets, hostname, "hostname should be in the list")

	var found bool
	for _, item := range targets {
		matched, err := regexp.MatchString(`\d+\.\d+\.\d+.\d+`, item)
		if err != nil {
			continue
		}

		found = found || matched
	}
	assert.True(t, found, "one of the targets should be an ip address")
}
