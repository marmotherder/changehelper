package main

const (
	gitCmd      = "git"
	gitAdded    = "A"
	GitRenamed  = "R100"
	GitModified = "M"
	GitDeleted  = "D"
)

func gitRemote(dir string) (*string, int, error) {
	sLogger.Debug("looking up git remotes")
	stdOut, exitCode, err := runCommand(dir, gitCmd, "remote")
	return stdOut, exitCode, err
}
