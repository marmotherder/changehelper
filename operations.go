package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/manifoldco/promptui"
)

func newVersion() {
	var options NewVersionOptions
	parseOptions(&options)

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

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

	_, unreleased, _, released, err := parseChangelog(options.ChangelogFile)
	if err != nil {
		sLogger.Errorf("failed to read the changelog file %s", options.ChangelogFile)
		sLogger.Fatal(err.Error())
	}

	if unreleased != nil {
		if options.Force {
			sLogger.Warn("there is a pending release, going to replace with incoming")
		} else {
			sLogger.Fatal("a pending version already exists in the changelog")
		}
	}

	unreleasedText := "[Unreleased]"
	newChange := change{
		VersionText: &unreleasedText,
	}
	increment := ""

	gitResolve := true
	if !options.NonInteractive {
		if options.Manual {
			prompt := promptui.Select{
				Label: "Should the new version attempt to be resolved from git?",
				Items: []string{"Yes", "No"},
			}

			_, confirm, err := prompt.Run()
			if err != nil {
				sLogger.Fatal(err.Error())
			}

			if confirm == "No" {
				gitResolve = false
			}
		}
	}

	if gitResolve {
		branch := mustHaveBranch(options.GitBranch, "What git branch should the changes be loaded from?", options.NonInteractive, git)

		if !options.SkipGitCheckout {
			if err := git.checkoutAndPull(branch); err != nil {
				sLogger.Fatal(err.Error())
			}
		}

		hasConventionalCommits := func() bool {
			if options.IgnoreConventionalCommits {
				return false
			}
			ccIncrement, err := loadConventionalCommitsToChange(
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

			increment = *ccIncrement

			return true
		}()

		if !hasConventionalCommits {
			diff, err := git.diff(branch, getRemote(git)+"/HEAD")
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
	} else if !options.NonInteractive {
		mustCaptureMultiLineInput("Was anything added this release?", "Was anything more added this release?", "Describe what was added in this release", &newChange.Added)
		mustCaptureMultiLineInput("Was anything changed this release?", "Was anything more changed this release?", "Describe what was changed in this release", &newChange.Changed)
		mustCaptureMultiLineInput("Was anything deprecated this release?", "Was anything more deprecated this release?", "Describe what was deprecated in this release", &newChange.Deprecated)
		mustCaptureMultiLineInput("Was anything removed this release?", "Was anything more removed this release?", "Describe what was removed in this release", &newChange.Removed)
		mustCaptureMultiLineInput("Was anything fixed this release?", "Was anything more fixed this release?", "Describe what was fixed in this release", &newChange.Fixed)
		mustCaptureMultiLineInput("Was anything security related this release?", "Was anything more security related this release?", "Describe what was security related in this release", &newChange.Security)
	}

	if options.Increment == "" && increment == "" {
		if options.NonInteractive {
			sLogger.Fatal("no increment level set for the version")
		}

		incrementPrompt := promptui.Select{
			Label: "What is the incrementation level?",
			Items: []string{"MAJOR", "MINOR", "PATCH"},
		}

		_, pIncrement, err := incrementPrompt.Run()
		if err != nil {
			sLogger.Error("failed to get the incrementation level")
			sLogger.Fatal(err.Error())
		}

		increment = pIncrement
	}

	newChange.renderChangeText(increment)

	if err := writeToChangelogFile(options.ChangelogFile, &newChange, released, false); err != nil {
		sLogger.Fatal(err.Error())
	}
}

func update() {
	var options UpdateOptions
	parseOptions(&options)

	defaultVersion := semver.MustParse("0.0.0")

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

	if options.GitBranch != "" {
		git.checkoutAndPull(options.GitBranch)
	}

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

	updateUnreleasedVersion(unreleased, increment)

	if err := writeToChangelogFile(options.ChangelogFile, unreleased, released, true); err != nil {
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

func printCurrent(changelogFile string) {
	current, err := getCurrentVersion(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(current.String())
	os.Exit(0)
}

func printUnreleased(changelogFile string) {
	unreleasedVersion, err := getUnreleasedVersion(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(unreleasedVersion.String())
	os.Exit(0)
}

func release() {
	var options ReleaseOptions
	parseOptions(&options)

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

	branch := mustHaveBranch(options.GitBranch, "What git branch should be released from?", options.NonInteractive, git)

	if !options.SkipGitCheckout {
		if err := git.checkoutAndPull(branch); err != nil {
			sLogger.Fatal(err.Error())
		}
	}

	releaseFiles := []string{options.ChangelogFile}
	if len(options.ReleaseFiles) > 0 {
		releaseFiles = append(releaseFiles, options.ReleaseFiles...)
	}

	if err := git.add(releaseFiles...); err != nil {
		sLogger.Fatal(err.Error())
	}

	unreleasedVersion, err := getUnreleasedVersion(options.ChangelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	if err := git.commit(fmt.Sprintf(options.GitCommitMessage, *unreleasedVersion)); err != nil {
		sLogger.Fatal(err.Error())
	}

	if err := git.push(false); err != nil {
		sLogger.Fatal(err.Error())
	}

	majorBranch := fmt.Sprintf("%s/%d", options.GitPrefix, unreleasedVersion.Major)
	minorBranch := fmt.Sprintf("%s.%d", majorBranch, unreleasedVersion.Minor)
	patchBranch := fmt.Sprintf("%s.%d", minorBranch, unreleasedVersion.Patch)

	failed := false
	if err := git.resetBranch(getRemote(git), branch, majorBranch); err != nil {
		sLogger.Error("failed to update/create the major branch")
		failed = true
	}
	if err := git.resetBranch(getRemote(git), branch, minorBranch); err != nil {
		sLogger.Error("failed to update/create the minor branch")
		failed = true
	}
	if err := git.resetBranch(getRemote(git), branch, patchBranch); err != nil {
		sLogger.Error("failed to update/create the patch branch")
		failed = true
	}

	if failed {
		sLogger.Fatal("one or more branches did not successfully update/create")
	}
}
