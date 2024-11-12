// Collection of methods and types for interacting with Ansible

package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// AnsibleConfig is a collection of meta-information about an Ansible repository.
//
// The virtualenv specified by the VenvConfig needs to be initialized before running Ansible commands.
// All dir paths are relative to the root of the source tarball.
type AnsibleConfig struct {
	VenvConfig    VenvConfig // Virtualenv config that Ansible will be executed in
	Cwd           string     // Path to change to when running Ansible commands
	InventoryList []string   // Paths to all desired inventories
}

// CreateAnsibleTargetsList generates and returns an array of possible targets
// The possible targets are ip addresses from the host interfaces or the hostname itself
func CreateAnsibleTargetsList() ([]string, error) {
	var inventoryTargets = []string{}

	// extract all ip addresses except the loopback one and add the hostname as a last fallback mechanism
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{}, errors.Wrap(err, "Unable to get the network interfaces")
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return []string{}, errors.Wrapf(err, "Unable to extract the ip addresses from %s", i.Name)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			inventoryTargets = append(inventoryTargets, ip.String())
		}
	}
	inventoryTargets = append(inventoryTargets, hostname)
	return inventoryTargets, nil
}

// FindInventoryForHost calls CreateAnsibleTargetsList to get a list of possible targets
// and returns the inventory and target that was found in that inventory.
// It stops on the first target match meaning that it assumes that hosts are
// defined by only 1 type of target like their eth0 ip address or hostname
// If the host was not found, it returns an error.
//
// This will check against all of the defined inventories in ansibleCfg.InventoryList,
// relative to the path defined in ansibleCfg.Cwd.
//
// The given hostname should be the full name that appears in the Ansible inventories.
func (a AnsibleConfig) FindInventoryForHost() (string, string, error) {
	targets, err := CreateAnsibleTargetsList()
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to perform the CreateAnsibleTargetsList command")
	}
	for _, item := range a.InventoryList {
		inv := filepath.Join(a.Cwd, item)
		_, err := os.Stat(inv)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", errors.Wrapf(err, "unable to find inventory: %s", item)
			}

			return "", "", err
		}
		playbook_path := filepath.Join(a.Cwd, viper.GetString("ansible-playbook"))

		vCmd := VenvCommand{
			Config: a.VenvConfig,
			Binary: "ansible-playbook",
			Args:   []string{playbook_path, "-i", inv, "--list-hosts"},
			Cwd:    a.Cwd,
		}

		venvCommandOutput := vCmd.Run()
		if err != nil {
			logrus.Debugln("Ansible inventory output:", venvCommandOutput.Stdout)
			return "", "", errors.Wrap(err, "unable to list hosts for "+item)
		}

		cleanOutput := trimMultilineWhiteSpace(venvCommandOutput.Stdout)

		for _, target := range targets {
			for _, line := range strings.Split(cleanOutput, "\n") {
				if target == line {
					logrus.Debug("Found ", target, " in inventory ", inv)
					return inv, target, nil
				}
			}

			logrus.Debug("Did not find ", target, " in inventory ", inv)
		}
	}
	return "", "", errors.New("Unable to find one of the target in any inventory")
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
	Stats         map[string]AnsibleNodeStatus `json:"stats"`
	CommandOutput VenvCommandRunOutput
}

// Ansible PlaybookRunner defines an Ansible-Playbook command to run.
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
		if viper.GetBool("debug") {
			a.Env = []string{
				"ANSIBLE_STDOUT_CALLBACK=default",
				"ANSIBLE_CALLBACK_WHITELIST=",
			}
			args = append(args, "-vvv")
		} else {
			a.Env = []string{
				"ANSIBLE_STDOUT_CALLBACK=json",
				"ANSIBLE_CALLBACK_WHITELIST=",
			}
		}
	}

	vCmd := VenvCommand{
		Config: a.AnsibleConfig.VenvConfig,
		Binary: "ansible-playbook",
		Args:   args,
		Cwd:    a.AnsibleConfig.Cwd,
		Env:    a.Env,
	}

	if viper.GetBool("debug") {
		vCmd.StreamOutput = true
	}

	var ansibleOutput AnsibleRunOutput
	ansibleOutput.CommandOutput = vCmd.Run()

	jsonErr := json.Unmarshal([]byte(ansibleOutput.CommandOutput.Stdout), &ansibleOutput)
	if ansibleOutput.CommandOutput.Error != nil && jsonErr != nil {
		logrus.Debug("Could not parse JSON from run. Ansible stdout:\n", ansibleOutput.CommandOutput.Stdout, "Ansible stderr:\n", ansibleOutput.CommandOutput.Stderr)
	}
	if ansibleOutput.CommandOutput.Error != nil {
		return ansibleOutput, errors.Wrap(ansibleOutput.CommandOutput.Error, "ansible run failed")
	}
	if jsonErr != nil {
		return ansibleOutput, errors.Wrap(jsonErr, "unable to parse ansible JSON stdout")
	}

	return ansibleOutput, nil
}
