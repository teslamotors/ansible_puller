package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os/exec"
	"strings"
)

// failedCommandLogger will print a bunch of context to the terminal when in debug mode
func failedCommandLogger(cmd *exec.Cmd) {
	if viper.GetBool("debug") {
		logrus.Debug("failed command: ", cmd.Args)
		logrus.Debug("stdout: ", cmd.Stdout)
		logrus.Debug("stderr: ", cmd.Stderr)
	}
}

// trimWhiteSpace trims all the leading and trailing whitespace of a multi-line string
func trimMultilineWhiteSpace(s string) string {
	ss := strings.Split(s, "\n")

	result := make([]string, len(ss))
	for i, item := range ss {
		result[i] = strings.TrimSpace(item)
	}

	return strings.Join(result, "\n")
}
