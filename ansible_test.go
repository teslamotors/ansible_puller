package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		testhostname2                  : ok=125  changed=3    unreachable=2    failed=5    skipped=183  rescued=1    ignored=2
	`

	ansibleOutput, err := parsePlayRecap(ansibleOutput)
	assert.Nil(t, err)

	expectedStats := map[string]AnsibleNodeStatus{
		"testhostname": {
			Ok:          120,
			Changed:     5,
			Unreachable: 1,
			Failures:    2,
			Skipped:     184,
		},
		"testhostname2": {
			Ok:          125,
			Changed:     3,
			Unreachable: 2,
			Failures:    5,
			Skipped:     183,
		},
	}

	assert.Contains(t, ansibleOutput.Stats, target, "testhostname should be a key in the Stats map")
	assert.Empty(t, cmp.Diff(expectedStats, ansibleOutput.Stats))
}

func TestParsePlayRecapFailed(t *testing.T) {
	var ansibleOutput AnsibleRunOutput
	ansibleOutput.CommandOutput.Stdout = `
		PLAY RECAP *********************************************************************
		testhostname                  : ok:120  changed:5    unreachable:1    failed:2    skipped:184  rescued:3    ignored:4
	`

	ansibleOutput, err := parsePlayRecap(ansibleOutput)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "recap line doesn't match the expected format")
}
