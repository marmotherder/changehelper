package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/manifoldco/promptui"
)

func newVersion() {
	var options NewVersionOptions
	parseOptions(&options)

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

	git.checkoutAndPull(options.GitBranch)

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

	unreleasedText := "## [Unreleased]"
	newChange := change{
		VersionText: &unreleasedText,
	}

	if !options.Manual {
		prompt := promptui.Select{
			Label: "Try to resolve changes from git?",
			Items: []string{"Yes", "No"},
		}

		_, confirm, err := prompt.Run()
		if err != nil {
			sLogger.Fatal(err.Error())
		}

		if confirm == "Yes" {
			hasConventionalCommits := func() bool {
				increment, err := loadConventionalCommitsToChange(
					options.GitWorkingDirectory,
					options.ChangelogFile,
					&newChange,
					git,
				)
				if err != nil {
					sLogger.Warn("failed to use conventional commits, falling back to diff only")
					sLogger.Debug(err.Error())
					return false
				}

				unreleasedText = fmt.Sprintf("%s - %s", unreleasedText, *increment)
				newChange.VersionText = &unreleasedText

				return true
			}()

			if !hasConventionalCommits {
				diff, err := git.diff(options.GitBranch, "HEAD")
				if err != nil {
					sLogger.Error("failed to resolve diff from git")
					sLogger.Fatal(err.Error())
				}

				for _, added := range diff.Added {
					newChange.Added = append(newChange.Added, "- "+added)
				}
				for _, changed := range diff.Changed {
					newChange.Changed = append(newChange.Changed, "- "+changed)
				}
				for _, removed := range diff.Removed {
					newChange.Removed = append(newChange.Removed, "- "+removed)
				}
			}
		}
	}
}

func update() {
	var options UpdateOptions
	parseOptions(&options)

	defaultVersion := semver.MustParse("0.0.0")

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

	git.checkoutAndPull(options.GitBranch)

	_, unreleased, increment, released, err := parseChangelog(options.ChangelogFile)
	if err != nil {
		sLogger.Error("failed to parse the changelog file")
		sLogger.Fatal(err.Error())
	}

	if options.GitEvaluate {
		gitVersions, err := listReleasedVersionFromGit(git, options.GitPrefix)
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

		increment, err = loadConventionalCommitsToChange(
			options.GitWorkingDirectory,
			options.ChangelogFile,
			unreleased,
			git,
		)
		if err != nil {
			sLogger.Fatal(err.Error())
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

	if unreleased.Version == nil {
		unreleased.Version = latestRelease.Version
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
	sb.WriteString(fmt.Sprintf("## [%s] - %s", unreleased.Version.String(), time.Now().Format("2006-01-02")))
	sb.WriteString("\n")
	sb.WriteString(unreleasedTextLines[1])
	sb.WriteString("\n\n")

	for _, release := range released {
		sb.WriteString(*release.Text)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(options.ChangelogFile, []byte(sb.String()), 0644); err != nil {
		sLogger.Fatal(err.Error())
	}
}

func loadConventionalCommitsToChange(
	dir,
	changelogFile string,
	change *change,
	git gitCli,
) (*string, error) {
	increment, fixedUnique, addedUnique, changedUnique, removedUnique, err := resolveConventionalCommits(git, changelogFile)
	if err != nil {
		sLogger.Error("failed to lookup conventional commits when running update")
		return nil, err
	}

	for fixed, message := range fixedUnique {
		if dir+fixed != changelogFile {
			change.Fixed = append(change.Fixed, fmt.Sprintf("- %s; %s", fixed, message))
		}
	}

	for added, message := range addedUnique {
		if dir+added != changelogFile {
			change.Added = append(change.Added, fmt.Sprintf("- %s; %s", added, message))
		}
	}

	for changed, message := range changedUnique {
		if dir+changed != changelogFile {
			change.Changed = append(change.Changed, fmt.Sprintf("- %s; %s", changed, message))
		}
	}

	for removed, message := range removedUnique {
		if dir+removed != changelogFile {
			change.Removed = append(change.Removed, fmt.Sprintf("- %s; %s", removed, message))
		}
	}

	return increment, nil
}
