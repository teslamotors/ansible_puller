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

func TestParsePlayRecap(t *testing.T) {
	target := `testhostname`
	var ansibleOutput AnsibleRunOutput
	ansibleOutput.CommandOutput.Stdout = `
		PLAY RECAP *********************************************************************
		testhostname                  : ok=120  changed=5    unreachable=1    failed=2    skipped=184  rescued=3    ignored=4
	`

	ansibleOutput, err := parsePlayRecap(ansibleOutput)
	assert.Nil(t, err)

	assert.Contains(t, ansibleOutput.Stats, target, "testhostname should be a key in the Stats map")
	assert.Equal(t, 120, ansibleOutput.Stats[target].Ok, "Stats[target].Ok should be 120")
	assert.Equal(t, 5, ansibleOutput.Stats[target].Changed, "Stats[target].Changed should be 5")
	assert.Equal(t, 1, ansibleOutput.Stats[target].Unreachable, "Stats[target].Unreachable should be 1")
	assert.Equal(t, 2, ansibleOutput.Stats[target].Failures, "Stats[target].Failed should be 2")
	assert.Equal(t, 184, ansibleOutput.Stats[target].Skipped, "Stats[target].Skipped should be 184")
}
