package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os/exec"
)

// failedCommandLogger will print a bunch of context to the terminal when in debug mode
func failedCommandLogger(cmd *exec.Cmd) {
	if viper.GetBool("debug") {
		logrus.Debug("failed command: ", cmd.Args)
		logrus.Debug("stdout: ", cmd.Stdout)
		logrus.Debug("stderr: ", cmd.Stderr)
	}
}
