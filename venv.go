// Functions and types for interacting with virtual environments

package main

import (
	"bufio"
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
		failedCommandLogger(cmd)
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
	Config       VenvConfig
	Binary       string   // path to the binary under $venv/bin
	Args         []string // args to pass to the command that is called
	Cwd          string   // Directory to change to, if needed
	Env          []string // Additions to the runtime environment
	StreamOutput bool     // Whether or not the application should stream output stdout/stderr
}

// Run will execute the command described in VenvCommand.
//
// The strings returned are Stdout/Stderr.
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

	if c.StreamOutput {
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			return "", "", errors.Wrap(err, "unable to start command")
		}

		for _, stream := range []io.ReadCloser{stdout, stderr} {
			go func(s io.ReadCloser) {
				scanner := bufio.NewScanner(s)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					m := scanner.Text()
					fmt.Println(m)
				}
			}(stream)
		}

		if err := cmd.Wait(); err != nil {
			return "", "", errors.Wrap(err, "unable to complete command")
		}

		return "", "", nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logrus.Debugln("Running venv command: ", cmd.Args)
	err := cmd.Run()

	stdoutStr, stderrStr := stdout.String(), stderr.String()

	if ctx.Err() == context.DeadlineExceeded {
		return stdoutStr, stderrStr, errors.Wrap(err, "Execution timed out")
	} else if err != nil {
		failedCommandLogger(cmd)
		return stdoutStr, stderrStr, errors.Wrap(err, "unable to complete command")
	}

	return stdoutStr, stderrStr, nil
}
