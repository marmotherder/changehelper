package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func runCommand(dir, name string, arg ...string) (*string, int, error) {
	sLogger.Infof("running command: %s %s in %s", name, arg, dir)

	cmd := exec.Command(name, arg...)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 0, err
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, 0, err
	}

	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		sLogger.Error("running command failed")
		sLogger.Error(err.Error())
		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr != nil {
				sLogger.Error(string(exitErr.Stderr))
			}
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
				sLogger.Errorf("exited with code %d", exitCode)
			}
		}
		return nil, exitCode, err
	}

	scannerStdOut := bufio.NewScanner(stdOut)
	sbStdOut := strings.Builder{}
	go func() {
		for scannerStdOut.Scan() {
			sbStdOut.WriteString(fmt.Sprintf("%s\n", scannerStdOut.Text()))
		}
	}()

	scannerStdErr := bufio.NewScanner(stdErr)
	sbStdErr := strings.Builder{}
	go func() {
		for scannerStdErr.Scan() {
			sbStdErr.WriteString(fmt.Sprintf("%s\n", scannerStdErr.Text()))
		}
	}()

	cmd.Wait()

	stdOutString := sbStdOut.String()

	if sbStdErr.String() != "" {
		sLogger.Error(sbStdErr.String())
	}

	sLogger.Infof("exited with code %d", cmd.ProcessState.ExitCode())

	return &stdOutString, cmd.ProcessState.ExitCode(), nil
}
