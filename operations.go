package main

import (
	"errors"
	"os"

	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
)

func newVersion() {
	var options NewVersionOptions
	parseOptions(&options)

	sLogger.Infof("checking if changelog file %s exists", options.ChangelogFile)
	if _, err := os.Stat(options.ChangelogFile); err != nil && errors.Is(err, os.ErrNotExist) {
		sLogger.Info("changelog file does not exist, attempting to create a new one instead")
		contents := `# Changelog
All notable changes to this project will be documented in this file.
		
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
`

		if err := os.WriteFile(options.ChangelogFile, []byte(contents), 0644); err != nil {
			sLogger.Errorf("failed to create changelog file %s", options.ChangelogFile)
			sLogger.Fatal(err.Error())
		}
	} else if err != nil {
		sLogger.Errorf("failed to read changelog file %s", options.ChangelogFile)
		sLogger.Error(err.Error())
		os.Exit(ioError)
	}

	machineOptions := []conventionalcommits.MachineOption{
		conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
		conventionalcommits.WithBestEffort(),
	}
	machine := parser.NewMachine(machineOptions...)
	// SETUP MACHINE
	sLogger.Info(machine)

	stdOut, _, err := gitRemote(options.GitWorkingDirectory)
	sLogger.Info(*stdOut)
	if err != nil {
		sLogger.Error(err.Error())
	}
}
