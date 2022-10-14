package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/marmotherder/go-gitcliwrapper"
)

func parseArgs(obj interface{}) []string {
	parser := flags.NewParser(obj, flags.HelpFlag+flags.IgnoreUnknown)
	args, err := parser.Parse()
	if err != nil {
		sLogger.Fatal(err)
	}
	return args
}

func mustLoadGit(workingDirectory string) *gitcliwrapper.GitCLIWrapper {
	git, err := gitcliwrapper.NewGitCLIWrapper(workingDirectory, sLogger)
	if err != nil {
		sLogger.Error("failed to startup git cli interface")
		sLogger.Fatal(err)
	}

	return git
}
