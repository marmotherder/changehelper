package main

import (
	"errors"
	"fmt"
	"strings"
)

const (
	gitCmd          = "git"
	nonZeroCodeText = "command returned a non zero code"
)

type gitCli struct {
	WorkingDirectory string
	Remote           string
}

func nonZeroCode(text string) error {
	return fmt.Errorf("%s %s %s", gitCmd, text, nonZeroCodeText)
}

func (git *gitCli) getRemote() error {
	sLogger.Debug("looking up git remote")
	remote, code, err := runCommand(git.WorkingDirectory, gitCmd, "remote")
	if err != nil {
		sLogger.Error("failed to lookup git remote")
		return err
	}
	if code != 0 {
		return nonZeroCode("remote")
	}
	if remote == nil {
		return errors.New("failed to find a git remote")
	}

	remoteString := strings.TrimSpace(*remote)
	multipleRemotes := strings.Split(remoteString, "\n")

	if len(multipleRemotes) <= 1 {
		git.Remote = remoteString
		return nil
	}

	remoteString = multipleRemotes[len(multipleRemotes)-1]
	sLogger.Warnf("multiple remotes were found, using the last one set '%s'", remoteString)

	git.Remote = remoteString
	return nil
}

func (git gitCli) getLastCommitOnRef(ref string) (*string, error) {
	sLogger.Debugf("get most recent commit for reference %s on remote %s", ref, git.Remote)
	stdOut, code, err := runCommand(git.WorkingDirectory, gitCmd, "rev-list", "-n", "1", ref)
	if code != 0 {
		return nil, nonZeroCode("rev-list")
	}
	if err != nil {
		sLogger.Infof("failed to get commit for reference %s on remote %s", ref, git.Remote)
		return nil, err
	}
	if stdOut != nil {
		return stdOut, nil
	}

	return nil, errors.New("failed to get commit on reference")
}

func (git gitCli) fetch() error {
	sLogger.Debugf("running git fetch against remote %s", git.Remote)
	_, code, err := runCommand(git.WorkingDirectory, gitCmd, "fetch", git.Remote)
	if code != 0 {
		return nonZeroCode("fetch")
	}
	return err
}

func (git gitCli) listRemoteRefs(refType string) ([]string, error) {
	sLogger.Infof("attempting to get a list of remote %s in git from %s", refType, git.Remote)
	remoteRefsResponse, code, err := runCommand(git.WorkingDirectory, gitCmd, "ls-remote", "--"+refType, git.Remote)
	if err != nil {
		sLogger.Error("failed to lookup from remote")
		return nil, err
	}
	if code != 0 {
		return nil, nonZeroCode("ls-remote")
	}
	if remoteRefsResponse == nil {
		return nil, fmt.Errorf("failed to find any branches against remote %s", git.Remote)
	}

	var remoteRefs []string
	for _, remoteRef := range strings.Split(*remoteRefsResponse, "\n") {
		splitRemoteRef := strings.Split(remoteRef, "refs/"+refType+"/")
		if len(splitRemoteRef) != 2 {
			sLogger.Warnf("attempted to parse a reference of unexpected format: %s", remoteRef)
			continue
		}
		remoteRefs = append(remoteRefs, splitRemoteRef[1])
	}

	return remoteRefs, nil
}

func (git gitCli) listCommits(commitRange ...string) ([]string, error) {
	sLogger.Debug("looking up git commits")
	commitRange = append(commitRange, git.Remote)
	stdOut, code, err := runCommand(git.WorkingDirectory, gitCmd, append([]string{"log", `--pretty=format:"%H"`}, commitRange...)...)
	if err != nil {
		sLogger.Error("failed to run git log")
		return nil, err
	}
	if code != 0 {
		return nil, nonZeroCode("log")
	}

	gitCommits := []string{}
	commitLines := strings.Split(*stdOut, "\n")
	for _, commitLine := range commitLines {
		sLogger.Debugf("processing commit: %s", commitLine)
		if commitLine != "" {
			gitCommits = append(gitCommits, strings.ReplaceAll(commitLine, "\"", ""))
		}
	}

	return gitCommits, nil
}

func (git gitCli) getCurrentBranch() (*string, error) {
	sLogger.Debug("getting the current branch")
	stdOut, code, err := runCommand(git.WorkingDirectory, gitCmd, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		sLogger.Error("failed to get the current git branch")
		return nil, err
	}
	if code != 0 {
		return nil, nonZeroCode("rev-parse")
	}

	return stdOut, nil
}

func (git gitCli) getCommitMessageBody(hash string) (*string, error) {
	sLogger.Debugf("getting the commit message for %s", hash)
	stdOut, code, err := runCommand(git.WorkingDirectory, gitCmd, "log", "--format=%B", "-n", "1", hash)
	if err != nil {
		sLogger.Errorf("failed to get the commit message for %s", hash)
		return nil, err
	}
	if code != 0 {
		return nil, nonZeroCode("log")
	}

	return stdOut, nil
}

func (git gitCli) forcePushHashToRef(hash, ref, refType string) error {
	sLogger.Debugf("going to try to push %s to %s on remote %s", hash, ref, git.Remote)
	_, code, err := runCommand(git.WorkingDirectory, gitCmd, "push", "-f", git.Remote, fmt.Sprintf("%s:refs/%s/%s", hash, refType, ref))
	if err != nil {
		sLogger.Errorf("failed to force push to git branch %s on remote %s", ref, git.Remote)
		return err
	}
	if code != 0 {
		return nonZeroCode("push")
	}

	return nil
}
