package main

import (
	"errors"
	"os"

	"github.com/blang/semver"
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
		sLogger.Fatal(err.Error())
	}

	machineOptions := []conventionalcommits.MachineOption{
		conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
		conventionalcommits.WithBestEffort(),
	}
	machine := parser.NewMachine(machineOptions...)
	// SETUP MACHINE
	sLogger.Info(machine)

	stdOut, err := gitRemote(options.GitWorkingDirectory)
	sLogger.Info(*stdOut)
	if err != nil {
		sLogger.Error(err.Error())
	}
}

func update() {
	var options UpdateOptions
	parseOptions(&options)

	if options.GitBranch != "" {
		if err := gitCheckout(options.GitBranch, options.GitWorkingDirectory); err != nil {
			sLogger.Fatal(err.Error())
		}
	}

	changelog, unreleased, increment, released, err := parseChangelog(options.ChangelogFile)
	if err != nil {
		sLogger.Error("failed to parse the changelog file")
		sLogger.Fatal(err.Error())
	}

	if options.GitEvaluate {
		gitVersions, err := listReleasedVersionFromGit(options.GitWorkingDirectory, options.GitPrefix)
		if err != nil {
			sLogger.Error("failed to lookup versions from git")
			sLogger.Fatal(err.Error())
		}

		for _, gitVersion := range gitVersions {
			released = append(released, change{
				Version: &gitVersion,
				Node:    nil,
			})
		}
	}

	var latestRelease change
	if len(released) > 0 {
		foundLatestRelease := getLatestRelease(released)
		latestRelease = *foundLatestRelease
	} else {
		defaultVersion := semver.MustParse("0.0.0")
		latestRelease = change{
			Version: &defaultVersion,
			Node:    nil,
		}
	}

	if unreleased == nil {
		commitMessages, err := gitCommitMessages(options.GitWorkingDirectory)
		if err != nil {
			sLogger.Fatal(err.Error())
		}

		machineOptions := []conventionalcommits.MachineOption{
			conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
			conventionalcommits.WithBestEffort(),
		}
		machine := parser.NewMachine(machineOptions...)

		for _, commitMessage := range commitMessages {
			ccMessage, err := machine.Parse([]byte(commitMessage))
			sLogger.Debug(ccMessage)
			sLogger.Debug(err)
		}
	}

	unreleased.Version.Major = latestRelease.Version.Major
	unreleased.Version.Minor = latestRelease.Version.Minor
	unreleased.Version.Patch = latestRelease.Version.Patch

	if increment != nil {
		switch *increment {
		case PATCH:
			unreleased.Version.Patch++
		case MINOR:
			unreleased.Version.Minor++
			unreleased.Version.Patch = 0
		case MAJOR:
			unreleased.Version.Major++
			unreleased.Version.Minor = 0
			unreleased.Version.Patch = 0
		}
	}

	sLogger.Debug(changelog)
	sLogger.Debug(unreleased)
	sLogger.Debug(increment)
	sLogger.Debug(released)
	sLogger.Debug(err)
}
