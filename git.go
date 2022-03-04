package main

const (
	gitCmd      = "git"
	gitAdded    = "A"
	GitRenamed  = "R100"
	GitModified = "M"
	GitDeleted  = "D"
)

func gitRemote(dir string) (*string, error) {
	sLogger.Debug("looking up git remotes")
	stdOut, _, err := runCommand(dir, gitCmd, "remote")
	return stdOut, err
}

func gitCheckout(ref, dir string) error {
	sLogger.Debug("looking up git remotes")
	stdOut, _, err := runCommand(dir, gitCmd, "checkout", ref)
	sLogger.Info(*stdOut)
	return err
}
