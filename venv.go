// Functions and types for interacting with virtual environments

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	venvCommandTimeout = 2 * time.Hour // 2 hours timeout
)

// VenvConfig defines a Python Virtual Environment.
type VenvConfig struct {
	Path   string // path to the virtualenv root
	Python string // path to the desired Python installation
}

// Takes a VenvConfig and will create a new virtual environment.
func makeVenv(cfg VenvConfig) error {
	venvExecutable, err := exec.LookPath("virtualenv")
	if err != nil {
		return errors.Wrap(err, "virtualenv not found in path")
	}

	cmd := exec.Command(venvExecutable, "--python", cfg.Python, cfg.Path)

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "unable to create virtual environment")
	}

	return nil
}

// Ensure ensures that a virtual environment exists, if not, it attempts to create it
func (c VenvConfig) Ensure() error {
	_, err := os.Stat(c.Path)
	if os.IsNotExist(err) {
		err := makeVenv(c)
		if err != nil {
			return err
		}
	}

	return nil
}

// Update updates the virtualenv for the given config with the specified requirements file
func (c VenvConfig) Update(requirementsFile string) error {
	vCmd := VenvCommand{
		Config: c,
		Binary: "pip",
		Args:   []string{"install", "-r", requirementsFile},
	}
	_, _, err := vCmd.Run()
	if err != nil {
		return errors.Wrap(err, "unable to update virtualenv")
	}

	return nil
}

// VenvCommand enables you to run a system command in a virtualenv.
type VenvCommand struct {
	Config VenvConfig
	Binary string   // path to the binary under $venv/bin
	Args   []string // args to pass to the command that is called
	Cwd    string   // Directory to change to, if needed
	Env    []string // Additions to the runtime environment
}

// Run will execute the command described in VenvCommand.
//
// The string returned is the combined Stdout/Stderr.
func (c VenvCommand) Run() (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), venvCommandTimeout)

	defer cancel() // The cancel should be deferred so resources are cleaned up

	path, ok := os.LookupEnv("PATH")
	if !ok {
		return "", "", errors.New("Unable to lookup the $PATH env variable")
	}

	// Updating $PATH variable to include the venv path
	venvPath := filepath.Join(c.Config.Path, "bin")
	if !strings.Contains(path, venvPath) {
		newVenvPath := fmt.Sprintf("%s:%s", filepath.Join(c.Config.Path, "bin"), path)
		logrus.Debugln("PATH: ", newVenvPath)
		os.Setenv("PATH", newVenvPath)
	}

	cmd := exec.CommandContext(
		ctx,
		filepath.Join(c.Config.Path, "bin", c.Binary),
		c.Args...,
	)

	if c.Cwd != "" {
		cmd.Dir = c.Cwd
	}

	cmd.Env = append(os.Environ(), c.Env...)

	logrus.Debugln("Running venv command: ", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	stdoutStr, stderrStr := string(stdout.Bytes()), string(stderr.Bytes())
	if ctx.Err() == context.DeadlineExceeded {
		return stdoutStr, stderrStr, errors.Wrap(err, "Execution timed out")
	} else if err != nil {
		return stdoutStr, stderrStr, errors.Wrap(err, "unable to complete command")
	}

	return stdoutStr, stderrStr, nil
}
