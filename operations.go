package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
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

	unreleasedVersionText := releasePrefix + "[Unreleased]"
	newChange := change{
		VersionText: &unreleasedVersionText,
	}
	increment := ""

	if mustGitResolveQuery(options.NonInteractive, options.Manual) {
		resolveVersionFromGit(options, git, &newChange, &increment)
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

func mustGitResolveQuery(nonInteractive, manual bool) bool {
	if !nonInteractive {
		if manual {
			prompt := promptui.Select{
				Label: "Should the new version attempt to be resolved from git?",
				Items: []string{"Yes", "No"},
			}

			_, confirm, err := prompt.Run()
			if err != nil {
				sLogger.Fatal(err.Error())
			}

			if confirm == "No" {
				return false
			}
		}
	}

	return true
}

func resolveVersionFromGit(options NewVersionOptions, git gitCli, newChange *change, increment *string) {
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
			options.Depth,
			newChange,
			git,
		)
		if err != nil {
			sLogger.Warn("failed to use conventional commits, falling back to diff only")
			sLogger.Debug(err.Error())
			return false
		}

		increment = ccIncrement

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
}

func update(ignoreUnknown bool) {
	var options UpdateOptions
	parseOptions(&options, ignoreUnknown)

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
		gitVersions, err := listReleasedVersionFromGit(options.UseTags, git, options.GitPrefix)
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
			Version: latestRelease.Version,
		}

		increment, err = loadConventionalCommitsToChange(
			options.GitWorkingDirectory,
			options.ChangelogFile,
			options.Depth,
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

	sLogger.Debug("Updating unrleased to:")
	sLogger.Debug(*unreleased.Text)

	if err := writeToChangelogFile(options.ChangelogFile, unreleased, released, true); err != nil {
		sLogger.Fatal(err.Error())
	}
}

func loadConventionalCommitsToChange(
	dir,
	changelogFile string,
	depth int,
	change *change,
	git gitCli,
) (*string, error) {
	increment, fixedUnique, addedUnique, changedUnique, removedUnique, err := resolveConventionalCommits(git, changelogFile, depth)
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

func printCurrentVersion(changelogFile string) {
	_, current, err := getCurrent(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(current.String())
	os.Exit(0)
}

func printUnreleasedVersion(changelogFile string) {
	_, unreleasedVersion, err := getUnreleased(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(unreleasedVersion.String())
	os.Exit(0)
}

func printCurrentChanges(changelogFile string) {
	currentText, _, err := getCurrent(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(*currentText)
	os.Exit(0)
}

func printUnreleasedChanges(changelogFile string) {
	unreleasedText, _, err := getUnreleased(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	fmt.Print(*unreleasedText)
	os.Exit(0)
}

func printChanges() {
	var options PrintChangesOptions
	parseOptions(&options)

	ver, err := semver.ParseTolerant(options.Version)
	if err != nil {
		sLogger.Errorf("could not parse the input version %s", options.Version)
		sLogger.Fatal(err.Error())
	}

	_, _, _, released, err := parseChangelog(options.ChangelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	filteredReleases := []*change{}

	switch strings.Count(options.Version, ".") {
	case 0:
		for _, release := range released {
			if release.Version.Major == ver.Major {
				filteredReleases = append(filteredReleases, release)
			}
		}
	case 1:
		for _, release := range released {
			if release.Version.Major == ver.Major && release.Version.Minor == ver.Minor {
				filteredReleases = append(filteredReleases, release)
			}
		}
	default:
		for _, release := range released {
			if release.Version.Major == ver.Major && release.Version.Minor == ver.Minor && release.Version.Patch == ver.Patch {
				filteredReleases = append(filteredReleases, release)
			}
		}
	}

	for _, release := range filteredReleases {
		fmt.Println(*release.VersionText)
		fmt.Println(*release.Text)
		fmt.Print("\n")
	}
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

	if err := git.fetch(); err != nil {
		sLogger.Error("failed to run a git fetch, trying to continue anyway")
	}

	releaseFiles := []string{options.ChangelogFile}
	if len(options.ReleaseFiles) > 0 {
		releaseFiles = append(releaseFiles, options.ReleaseFiles...)
	}

	if err := git.add(releaseFiles...); err != nil {
		sLogger.Fatal(err.Error())
	}

	_, version, err := getCurrent(options.ChangelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	if err := git.commit(fmt.Sprintf(options.GitCommitMessage, version.String())); err != nil {
		sLogger.Fatal(err.Error())
	}

	if err := git.push(false); err != nil {
		sLogger.Fatal(err.Error())
	}

	errs := createOrUpdateReleaseRefs(options.UseTags, branch, options.GitPrefix, options.VersionPrefix, version, git)

	if len(errs) > 0 {
		for _, err := range errs {
			sLogger.Error(err.Error())
		}
		sLogger.Fatal("one or more branches did not successfully update/create")
	}
}

func enforceUnreleased(changelogFile string) {
	_, unreleased, _, _, err := parseChangelog(changelogFile)
	if err != nil {
		sLogger.Fatal(err.Error())
	}

	if unreleased == nil {
		sLogger.Fatal("no unreleased change detected in changelogfile file")
	}
}

func createOrUpdateReleaseRefs(useTags bool, branch, gitPrefix, versionPrefix string, version *semver.Version, git gitCli) []error {
	remote := getRemote(git)

	majorRef := fmt.Sprintf("%s/%s%d", gitPrefix, versionPrefix, version.Major)
	minorRef := fmt.Sprintf("%s.%d", majorRef, version.Minor)
	patchRef := fmt.Sprintf("%s.%d", minorRef, version.Patch)

	errs := []error{}
	if useTags {
		if err := git.resetTag(remote, majorRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the major tag"))
		}
		if err := git.resetTag(remote, minorRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the minor tag"))
		}
		if err := git.resetTag(remote, patchRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the patch tag"))
		}
	} else {
		if err := git.resetBranch(remote, branch, majorRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the major branch"))
		}
		if err := git.resetBranch(remote, branch, minorRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the minor branch"))
		}
		if err := git.resetBranch(remote, branch, patchRef); err != nil {
			errs = append(errs, errors.New("failed to update/create the patch branch"))
		}
	}

	return errs
}

func enforceConventionalCommits() {
	var options EnforceConventionalCommitsOptions
	parseOptions(&options)

	git := gitCli{
		WorkingDirectory: options.GitWorkingDirectory,
	}

	branch := mustHaveBranch(options.GitBranch, "", true, git)

	if !options.SkipGitCheckout {
		if err := git.checkoutAndPull(branch); err != nil {
			sLogger.Fatal(err.Error())
		}
	}

	if err := git.fetch(); err != nil {
		sLogger.Error("failed to run a git fetch, trying to continue anyway")
	}

	var commits []gitCommit
	if options.Depth > 0 {
		var err error
		commits, err = git.listCommits(fmt.Sprintf("HEAD~%d..HEAD", options.Depth))
		if err != nil {
			sLogger.Errorf("could not list the commits between HEAD and HEAD~%d", options.Depth)
			sLogger.Fatal(err.Error())
		}
	} else {
		cLogCommit, err := git.getLastModifiedCommit(options.ChangelogFile)
		if err != nil {
			sLogger.Errorf("failed to get the last change for the changelog file %s", options.ChangelogFile)
			sLogger.Fatal(err.Error())
		}

		commits, err = git.listCommits(*cLogCommit + "..HEAD")
		if err != nil {
			sLogger.Errorf("could not list the commits between HEAD and %s", *cLogCommit)
			sLogger.Fatal(err.Error())
		}
	}

	machineOptions := []conventionalcommits.MachineOption{
		conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
		conventionalcommits.WithBestEffort(),
	}
	machine := parser.NewMachine(machineOptions...)

	failures := []int{}
	for idx, commit := range commits {
		ccMessage, err := machine.Parse([]byte(commit.Message))
		if err != nil {
			sLogger.Info(err.Error())
			failures = append(failures, idx)
			continue
		}
		if !ccMessage.Ok() {
			failures = append(failures, idx)
		}
	}

	if len(failures) > 0 {
		sb := strings.Builder{}
		sb.WriteString("not all commits were found to adhere to conventional commit principles\n\n")

		for _, idx := range failures {
			commit := commits[idx]
			sb.WriteString(fmt.Sprintf("Commit: %s was not conventional commit, instead found unparseable message: %s\n", commit.Hash, commit.Message))
		}

		if options.AllowNonConventionalcommits {
			sLogger.Warn(sb.String())
		} else {
			sLogger.Fatal(sb.String())
		}
	}
}
