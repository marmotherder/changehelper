package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	gitCmd      = "git"
	gitAdded    = "A"
	GitRenamed  = "R100"
	GitModified = "M"
	GitDeleted  = "D"
)

func gitRemote(dir string) (*string, error) {
	sLogger.Debug("looking up git remote")
	remote, _, err := runCommand(dir, gitCmd, "remote")
	if err != nil {
		sLogger.Error("failed to lookup git remote")
		return nil, err
	}
	if remote == nil {
		return nil, errors.New("failed to find a git remote")
	}

	remoteString := strings.TrimSpace(*remote)
	multipleRemotes := strings.Split(remoteString, "\n")

	if len(multipleRemotes) <= 1 {
		return &remoteString, nil
	}

	remoteString = multipleRemotes[len(multipleRemotes)-1]
	sLogger.Warn("multiple remotes were found, using the last one set '%s'", remoteString)

	return &remoteString, nil
}

func gitCheckout(ref, dir string) error {
	sLogger.Debug("looking up git remotes")
	stdOut, _, err := runCommand(dir, gitCmd, "checkout", ref)
	sLogger.Info(*stdOut)
	return err
}

func gitCommitMessages(dir string, commitRange ...string) ([]string, error) {
	sLogger.Debug("looking up git commit messages")
	stdOut, _, err := runCommand(dir, gitCmd, append([]string{"log", `--pretty=format:"%s"`}, commitRange...)...)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}

	tidyCommitMessages := make([]string, 0)
	commitMessages := strings.Split(*stdOut, "\n")
	for _, commitMessage := range commitMessages {
		if commitMessage != "" && commitMessage != "\"\"" {
			tidyCommitMessages = append(tidyCommitMessages, commitMessage[1:len(commitMessage)-1])
		}
	}

	return tidyCommitMessages, nil
}

func getLastModifiedCommit(dir, path string) (*string, error) {
	sLogger.Debug("looking up most recent commit for %s", path)
	stdOut, _, err := runCommand(dir, gitCmd, "log", "-n", "1", "--pretty=format:%H", "--", path)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}

	return stdOut, nil
}

func listRemoteGitBranches(dir, prefix string, remotes ...string) ([]string, error) {
	var remote string
	if len(remotes) > 0 {
		remote = remotes[0]
	} else {
		foundRemote, err := gitRemote(dir)
		if err != nil {
			return nil, err
		}
		remote = *foundRemote
	}

	sLogger.Info("attempting to get a list of remote branches in git from %s", remote)
	foundRemoteBranches, _, err := runCommand(dir, gitCmd, "ls-remote", "--heads", remote)
	if err != nil {
		sLogger.Error("failed to lookup branches from remote")
		return nil, err
	}
	if foundRemoteBranches == nil {
		return nil, fmt.Errorf("failed to find any branches against remote %s", remote)
	}

	remoteBranches := []string{}
	for _, remoteBranch := range strings.Split(*foundRemoteBranches, "\n") {
		sLogger.Debugf("parsing branch %s to see if it's a release", remoteBranch)
		prefixRegex := regexp.MustCompile(fmt.Sprintf("/%s/", prefix))
		if prefixRegex.MatchString(remoteBranch) {
			remoteBranchSplit := prefixRegex.Split(remoteBranch, 2)
			sLogger.Debugf("branch %s matched filter and trim, capturing %s", remoteBranch, remoteBranchSplit[1])
			remoteBranches = append(remoteBranches, remoteBranchSplit[1])
		}
	}

	return remoteBranches, nil
}
