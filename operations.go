package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
		contents := changelogHeader

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

	stdOut, err := getGitRemote(options.GitWorkingDirectory)
	sLogger.Info(*stdOut)
	if err != nil {
		sLogger.Error(err.Error())
	}
}

func update() {
	var options UpdateOptions
	parseOptions(&options)

	defaultVersion := semver.MustParse("0.0.0")

	if options.GitBranch != "" {
		if err := gitCheckout(options.GitBranch, options.GitWorkingDirectory); err != nil {
			sLogger.Fatal(err.Error())
		}
	}

	_, unreleased, increment, released, err := parseChangelog(options.ChangelogFile)
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
			released = append(released, &change{
				Version: &gitVersion,
				Text:    nil,
			})
		}
	}

	var latestRelease change
	if len(released) > 0 {
		foundLatestRelease := getLatestRelease(released)
		latestRelease = *foundLatestRelease
	} else {
		latestRelease = change{
			Version: &defaultVersion,
			Text:    nil,
		}
	}

	if unreleased == nil {
		unreleased = &change{
			Version: &defaultVersion,
		}

		lastCommit, err := getLastModifiedCommit(options.GitWorkingDirectory, options.ChangelogFile)
		if err != nil {
			sLogger.Fatal(err.Error())
		}

		commits, err := listGitCommits(options.GitWorkingDirectory, *lastCommit+"..")
		if err != nil {
			sLogger.Fatal(err.Error())
		}

		var commitMessages []string
		for _, commit := range commits {
			commitMessages = append(commitMessages, commit.Message)
		}

		var mappedTypes map[int]conventionalCommitType
		increment, mappedTypes = parseConventionalCommitMessages(commitMessages...)

		fixedUnique := map[string]string{}
		addedUnique := map[string]string{}
		changedUnique := map[string]string{}
		removedUnique := map[string]string{}
		for idx, ccType := range mappedTypes {
			commit := commits[idx]

			diff, err := getGitRefChanges(options.GitWorkingDirectory, commit.Hash)
			if err != nil {
				sLogger.Warnf("failed to read changes for commit %s, changes will not be recorded in changelog", commit.Hash)
			}

			switch ccType {
			case conventionalCommitFix:
				for _, changed := range diff.Changed {
					if existing, ok := fixedUnique[changed]; ok {
						fixedUnique[changed] = existing + ", " + commit.Message
					} else {
						fixedUnique[changed] = commit.Message
					}
				}
				fallthrough
			default:
				for _, added := range diff.Added {
					if existing, ok := addedUnique[added]; ok {
						addedUnique[added] = existing + ", " + commit.Message
					} else {
						addedUnique[added] = commit.Message
					}
				}
				if ccType != conventionalCommitFix {
					for _, changed := range diff.Changed {
						if existing, ok := changedUnique[changed]; ok {
							changedUnique[changed] = existing + ", " + commit.Message
						} else {
							changedUnique[changed] = commit.Message
						}
					}
				}
				for _, removed := range diff.Removed {
					if existing, ok := removedUnique[removed]; ok {
						removedUnique[removed] = existing + ", " + commit.Message
					} else {
						removedUnique[removed] = commit.Message
					}
				}
			}
		}

		for fixed, message := range fixedUnique {
			unreleased.Fixed = append(unreleased.Fixed, fmt.Sprintf("- %s; %s", fixed, message))
		}

		for added, message := range addedUnique {
			unreleased.Added = append(unreleased.Added, fmt.Sprintf("- %s; %s", added, message))
		}

		for changed, message := range changedUnique {
			unreleased.Changed = append(unreleased.Changed, fmt.Sprintf("- %s; %s", changed, message))
		}

		for removed, message := range removedUnique {
			unreleased.Removed = append(unreleased.Removed, fmt.Sprintf("- %s; %s", removed, message))
		}

		unreleased.renderChangeText(*increment)
	}

	if increment == nil {
		combinedMessages := []string{}
		combineCommits := func(messages []string) {
			for _, message := range messages {
				combinedMessages = append(combinedMessages, strings.TrimPrefix(message, "- "))
			}
		}
		combineCommits(unreleased.Added)
		combineCommits(unreleased.Changed)
		combineCommits(unreleased.Deprecated)
		combineCommits(unreleased.Fixed)
		combineCommits(unreleased.Removed)
		combineCommits(unreleased.Security)

		increment, _ = parseConventionalCommitMessages(combinedMessages...)

		if increment == nil {
			sLogger.Fatal("there is a pending release without a version in changelog, but was unable to determine it from messages")
		}

		sLogger.Debug(*unreleased.Text)
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

	sb := strings.Builder{}
	sb.WriteString(changelogHeader)
	unreleasedTextLines := strings.SplitN(*unreleased.Text, "\n", 2)
	sb.WriteString("## [Unreleased] - " + time.Now().Format("2006-01-02"))
	sb.WriteString("\n")
	sb.WriteString(unreleasedTextLines[1])
	sb.WriteString("\n")

	for _, release := range released {
		sb.WriteString(*release.Text)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(options.ChangelogFile, []byte(sb.String()), 0644); err != nil {
		sLogger.Fatal(err.Error())
	}
}
