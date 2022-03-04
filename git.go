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
	gitRenamed  = "R100"
	gitModified = "M"
	gitDeleted  = "D"
)

type gitDiff struct {
	Added   []string
	Changed []string
	Removed []string
}

func getGitRemote(dir string) (*string, error) {
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

type gitCommit struct {
	Hash    string
	Message string
}

func listGitCommits(dir string, commitRange ...string) ([]gitCommit, error) {
	sLogger.Debug("looking up git commits")
	stdOut, _, err := runCommand(dir, gitCmd, append([]string{"log", `--pretty=format:"%H %s"`}, commitRange...)...)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}

	gitCommits := []gitCommit{}
	commitLines := strings.Split(*stdOut, "\n")
	for _, commitLine := range commitLines {
		if commitLine != "" && commitLine != "\"\"" {
			tidyCommitLine := commitLine[1 : len(commitLine)-1]
			splitLine := strings.SplitN(tidyCommitLine, " ", 2)
			gitCommits = append(gitCommits, gitCommit{
				Hash:    splitLine[0],
				Message: splitLine[1],
			})
		}
	}

	return gitCommits, nil
}

func getGitRefChanges(dir, ref string) (*gitDiff, error) {
	sLogger.Debug("looking up changes for ref %s", ref)
	stdOut, _, err := runCommand(dir, gitCmd, "show", "--name-status", ref, "--pretty=format:")
	if err != nil {
		sLogger.Errorf("git show for %s failed", ref)
		return nil, err
	}

	diff := parseGitChanges(*stdOut)
	return &diff, nil
}

func parseGitChanges(changes string, relativePath ...string) gitDiff {
	uniqueChanges := map[string]string{}
	for _, line := range strings.Split(changes, "\n") {
		fields := strings.Fields(line)

		if len(fields) < 2 || len(fields) > 3 {
			continue
		}

		changeType := fields[0]
		changedFile := fields[1]

		if len(relativePath) > 0 && relativePath[0] != "" {
			if !strings.HasPrefix(changedFile, relativePath[0]) {
				continue
			}

			changedFile = strings.ReplaceAll(changedFile, relativePath[0], "")[1:]
		}

		if len(fields) == 3 {
			renameFile := fields[2]
			if len(relativePath) > 0 && relativePath[0] != "" {
				renameFile = strings.ReplaceAll(renameFile, relativePath[0], "")[1:]
			}
			changedFile = fmt.Sprintf("%s --> %s", changedFile, renameFile)
		}

		uniqueChanges[changedFile] = changeType
	}

	data := gitDiff{}
	for changedFile, changeType := range uniqueChanges {
		switch changeType {
		case gitAdded:
			data.Added = append(data.Added, changedFile)
		case gitRenamed, gitModified:
			data.Changed = append(data.Changed, changedFile)
		case gitDeleted:
			data.Removed = append(data.Removed, changedFile)
		}
	}

	return data
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
		foundRemote, err := getGitRemote(dir)
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
