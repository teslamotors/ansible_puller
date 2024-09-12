// Functions and types for interacting with virtual environments

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	pythonVersion, err := getPythonVersion(cfg.Python)
	if err != nil {
		return errors.Wrap(err, "unable to determine Python version")
	}
	logrus.Debugln("Detected Python version:", pythonVersion)

        // venv was introduced in python version 3.3
	useVenv := pythonVersionAtLeast(pythonVersion, 3, 3)
	logrus.Debugln("Use venv:", useVenv)

	var cmd *exec.Cmd
	if useVenv {
		cmd = exec.Command(cfg.Python, "-m", "venv", cfg.Path)
	} else {
		venvExecutable, err := exec.LookPath("virtualenv")
		if err != nil {
			return errors.Wrap(err, "virtualenv not found in path")
		}
		cmd = exec.Command(venvExecutable, "--python", cfg.Python, cfg.Path)
	}

	err = cmd.Run()
	if err != nil {
		failedCommandLogger(cmd)
		return errors.Wrap(err, "unable to create virtual environment")
	}

	return nil
}

// pythonVersionAtLeast checks if the Python version is at least major.minor.
func pythonVersionAtLeast(version string, major int, minor int) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	majorVersion, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	minorVersion, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	if majorVersion > major {
		return true
	}

	if majorVersion == major && minorVersion >= minor {
		return true
	}

	return false
}

// getPythonVersion returns the Python version as a string.
func getPythonVersion(python string) (string, error) {
	cmd := exec.Command(python, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // Some versions output version info to stderr

	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "unable to execute Python version command")
	}

	versionOutput := strings.TrimSpace(out.String())
	parts := strings.Fields(versionOutput)
	if len(parts) != 2 {
		return "", errors.New("unexpected output from Python version command")
	}

	return parts[1], nil
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
	venvCommandOutput := vCmd.Run()
	if venvCommandOutput.Error != nil {
		return errors.Wrap(venvCommandOutput.Error, "unable to update virtualenv")
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

type VenvCommandRunOutput struct {
	Stdout   string
	Stderr   string
	Error    error
	Exitcode int
}

// Run will execute the command described in VenvCommand.
//
// The strings returned are Stdout/Stderr.
func (c VenvCommand) Run() VenvCommandRunOutput {
	ctx, cancel := context.WithTimeout(context.Background(), venvCommandTimeout)
	CommandOutput := VenvCommandRunOutput{
		Stdout:   "",
		Stderr:   "",
		Error:    nil,
		Exitcode: -1,
	}

	defer cancel() // The cancel should be deferred so resources are cleaned up

	path, ok := os.LookupEnv("PATH")
	if !ok {
		CommandOutput.Error = errors.New("Unable to lookup the $PATH env variable")
		return CommandOutput
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
			CommandOutput.Error = errors.Wrap(err, "unable to start command")
			return CommandOutput
		}
		
		output := map[io.ReadCloser]*string{
			stdout: &CommandOutput.Stdout,
			stderr: &CommandOutput.Stderr,
		}

		for stream, out := range output {
			go func(s io.ReadCloser, o *string) {
				var buff bytes.Buffer
				
				scanner := bufio.NewScanner(s)
				scanner.Split(bufio.ScanLines)

				for scanner.Scan() {
					m := scanner.Text()
					fmt.Println(m)
					buff.WriteString(m)
					buff.WriteString("\n")
				}
				*o = fmt.Sprint(buff.String())
				
			}(stream, out)
		}

		if err := cmd.Wait(); err != nil {
			exitError, _ := err.(*exec.ExitError)
			CommandOutput.Error = errors.Wrap(err, "unable to complete command")
			CommandOutput.Exitcode = exitError.ExitCode()
			return CommandOutput
		}

		CommandOutput.Exitcode = 0

		return CommandOutput
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logrus.Debugln("Running venv command: ", cmd.Args)
	err := cmd.Run()

	CommandOutput.Stderr = stderr.String()
	CommandOutput.Stdout = stdout.String()

	if ctx.Err() == context.DeadlineExceeded {
		CommandOutput.Error = errors.Wrap(err, "Execution timed out")
		return CommandOutput
	} else if err != nil {
		failedCommandLogger(cmd)
		if exitError, ok := err.(*exec.ExitError); ok {
			CommandOutput.Exitcode = exitError.ExitCode()
		}
		CommandOutput.Error = errors.Wrap(err, "unable to complete command")
		return CommandOutput
	}

	CommandOutput.Exitcode = 0

	return CommandOutput
}
