// Collection of methods and types for interacting with Ansible

package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

// AnsibleConfig is a collection of meta-information about an Ansible repository.
//
// The virtualenv specified by the VenvConfig needs to be initialized before running Ansible commands.
// All dir paths are relative to the root of the source tarball.
type AnsibleConfig struct {
	VenvConfig         VenvConfig // Virtualenv config that Ansible will be executed in
	Cwd                string     // Path to change to when running Ansible commands
	InventoryList      []string   // Paths to all desired inventories
}

// FindInventoryForHost takes a hostname and returns the inventory where the host was found.
// If the host was not found, it returns an error.
//
// This will check against all of the defined inventories in ansibleCfg.InventoryList,
// relative to the path defined in ansibleCfg.Cwd.
//
// The given hostname should be the full name that appears in the Ansible inventories.
func (a AnsibleConfig) FindInventoryForHost(host string) (string, error) {
	for _, item := range a.InventoryList {
		inv := filepath.Join(a.Cwd, item)
		_, err := os.Stat(inv)
		if err != nil {
			if os.IsNotExist(err) {
				return "", errors.Wrapf(err, "unable to find inventory: %s", item)
			}

			return "", err
		}

		vCmd := VenvCommand{
			Config: a.VenvConfig,
			Binary: "ansible-playbook",
			Args:   []string{viper.GetString("ansible-playbook"), "-i", inv, "--list-hosts"},
			Cwd:    a.Cwd,
		}

		output, _, err := vCmd.Run()
		if err != nil {
			logrus.Debugln("Ansible inventory output:", output)
			return "", errors.Wrap(err, "unable to list hosts for "+item)
		}

		if strings.Contains(output, host) {
			logrus.Debug("Found ", host, " in inventory ", inv)
			return inv, nil
		}

		logrus.Debug("Did not find ", host, " in inventory ", inv)
	}

	return "", errors.New("Unable to detect hostname in any inventory: " + host)
}

// AnsibleNodeStatus contains status information for a single node's Ansible run.
type AnsibleNodeStatus struct {
	Changed     int `json:"changed"`
	Failures    int `json:"failures"`
	Ok          int `json:"ok"`
	Skipped     int `json:"skipped"`
	Unreachable int `json:"unreachable"`
}

// AnsibleRunOutput is a collection of all of the information given by an Ansible run.
type AnsibleRunOutput struct {
	Stats map[string]AnsibleNodeStatus `json:"stats"`
	Stdout string
	Stderr string
}

// AnsiblePlaybookRunner defines an Ansible-Playbook command to run.
//
// All dirs are relative to the tarball root.
type AnsiblePlaybookRunner struct {
	AnsibleConfig   AnsibleConfig
	PlaybookPath    string   // Path to the playbook to run
	InventoryPath   string   // Path to the appropriate inventory
	LimitExpr       string   // "limit" expression to be passed to Ansible (default: none)
	LocalConnection bool     // Whether or not to use a local connection
	Env             []string // Envvars to pass into the Ansible run
}

// Run executes the ansible-playbook command defined in the associated AnsiblePlaybookRunner.
func (a AnsiblePlaybookRunner) Run() (AnsibleRunOutput, error) {
	args := []string{a.PlaybookPath, "-i", a.InventoryPath}

	if a.LimitExpr != "" {
		args = append(args, "-l", a.LimitExpr)
	}

	if a.LocalConnection {
		args = append(args, "-c", "local")
	}

	if len(a.Env) == 0 {
		a.Env = []string{
			"ANSIBLE_STDOUT_CALLBACK=json",
			"ANSIBLE_CALLBACK_WHITELIST=",
		}
	}

	vCmd := VenvCommand{
		Config: a.AnsibleConfig.VenvConfig,
		Binary: "ansible-playbook",
		Args:   args,
		Cwd:    a.AnsibleConfig.Cwd,
		Env:    a.Env,
	}

	stdout, stderr, err := vCmd.Run()
	if err != nil {
		logrus.Debug("Ansible stdout:\n", stdout, "Ansible stderr:\n", stderr)
		return AnsibleRunOutput{}, errors.Wrap(err, "ansible run failed")
	}

	var ansibleOutput AnsibleRunOutput
	err = json.Unmarshal([]byte(stdout), &ansibleOutput)
	if err != nil {
		logrus.Debug("Ansible stdout:\n", stdout, "Ansible stderr:\n", stderr)
		return AnsibleRunOutput{}, errors.Wrap(err, "unable to parse ansible JSON stdout")
	}

	ansibleOutput.Stdout = stdout
	ansibleOutput.Stderr = stderr

	return ansibleOutput, nil
}
