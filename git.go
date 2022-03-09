package main

import (
	"errors"
	"fmt"
	"path/filepath"
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

type gitCli struct {
	WorkingDirectory string
}

func (git gitCli) getRemote() (*string, error) {
	sLogger.Debug("looking up git remote")
	remote, _, err := runCommand(git.WorkingDirectory, gitCmd, "remote")
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

func (git gitCli) checkout(ref string) error {
	sLogger.Debug("looking up git remotes")
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "checkout", ref)
	sLogger.Info(*stdOut)
	return err
}

func (git gitCli) pull() error {
	sLogger.Debug("running git pull")
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "pull")
	sLogger.Info(*stdOut)
	return err
}

type gitCommit struct {
	Hash    string
	Message string
}

func (git gitCli) listCommits(commitRange ...string) ([]gitCommit, error) {
	sLogger.Debug("looking up git commits")
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, append([]string{"log", `--pretty=format:"%H %s"`}, commitRange...)...)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}

	gitCommits := []gitCommit{}
	commitLines := strings.Split(*stdOut, "\n")
	for _, commitLine := range commitLines {
		sLogger.Debugf("processing commit: %s", commitLine)
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

func (git gitCli) getRefChanges(ref string) (*gitDiff, error) {
	sLogger.Debugf("looking up changes for ref %s", ref)
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "show", "--name-status", ref, "--pretty=format:")
	if err != nil {
		sLogger.Errorf("git show for %s failed", ref)
		return nil, err
	}

	diff := parseChanges(*stdOut)
	return &diff, nil
}

func (git gitCli) getCurrentBranch() (*string, error) {
	sLogger.Debug("getting the current branch")
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		sLogger.Errorf("failed to get the current git branch")
		return nil, err
	}

	return stdOut, nil
}

func parseChanges(changes string, relativePath ...string) gitDiff {
	uniqueChanges := map[string]string{}
	for _, line := range strings.Split(changes, "\n") {
		sLogger.Debugf("attempting to parse commit to a diff: %s", line)
		fields := strings.Fields(line)

		if len(fields) < 2 || len(fields) > 3 {
			continue
		}

		changeType := fields[0]
		changedFile := fields[1]

		if len(relativePath) > 0 && relativePath[0] != "" {
			if strings.HasPrefix(changedFile, relativePath[0]) {
				changedFile = strings.ReplaceAll(changedFile, relativePath[0], "")[1:]
			}
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

func (git gitCli) getLastModifiedCommit(path string) (*string, error) {
	sLogger.Debug("looking up most recent commit for %s", path)
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "log", "-n", "1", "--pretty=format:%H", "--", path)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}

	return stdOut, nil
}

func (git gitCli) listRemoteBranches(prefix string, remotes ...string) ([]string, error) {
	var remote string
	if len(remotes) > 0 {
		remote = remotes[0]
	} else {
		foundRemote, err := git.getRemote()
		if err != nil {
			return nil, err
		}
		remote = *foundRemote
	}

	sLogger.Info("attempting to get a list of remote branches in git from %s", remote)
	foundRemoteBranches, _, err := runCommand(git.WorkingDirectory, gitCmd, "ls-remote", "--heads", remote)
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

func (git gitCli) checkoutAndPull(branch string) error {
	if branch != "" {
		if err := git.checkout(branch); err != nil {
			sLogger.Error(err.Error())
			return err
		}

		if err := git.pull(); err != nil {
			sLogger.Error(err.Error())
			return err
		}
	}

	return nil
}

func (git gitCli) diff(sourceRef, compareRef string) (*gitDiff, error) {
	sLogger.Debugf("running a git diff between %s and %s", sourceRef, compareRef)
	stdOut, _, err := runCommand(git.WorkingDirectory, gitCmd, "rev-parse", "--show-toplevel")
	if err != nil {
		sLogger.Error("failed to rev-parse")
		return nil, err
	}

	gitPath := strings.Trim(*stdOut, "\n")

	absPath, err := filepath.Abs(git.WorkingDirectory)
	if err != nil {
		sLogger.Error("failed to check if paths were absolute")
		return nil, err
	}

	relativePath := strings.ReplaceAll(absPath, gitPath, "")
	if relativePath != "" {
		relativePath = relativePath[1:]
	}
	sLogger.Debugf("determined the relative path as %s", relativePath)

	sLogger.Info("attempting to run git fetch")
	if _, _, err := runCommand(git.WorkingDirectory, gitCmd, "fetch"); err != nil {
		sLogger.Error("failed to run git fetch")
		return nil, err
	}

	sLogger.Info("attempting to run git diff between two refs")
	stdOutBranch, _, err := runCommand(git.WorkingDirectory, gitCmd, "diff", "--name-status", sourceRef, compareRef)
	if err != nil {
		sLogger.Errorf("failed to git diff between %s and %s", sourceRef, compareRef)
		return nil, err
	}

	sLogger.Info("attempting to run git diff on single ref")
	stdOutLocal, _, err := runCommand(git.WorkingDirectory, gitCmd, "diff", "--name-status", compareRef)
	if err != nil {
		sLogger.Errorf("failed to git diff %s", compareRef)
		return nil, err
	}

	allChanges := *stdOutBranch + "\n" + *stdOutLocal
	diff := parseChanges(allChanges, relativePath)

	sLogger.Debug(diff)
	return &diff, nil
}
