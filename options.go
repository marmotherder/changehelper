package main

import (
	"os"
	"reflect"

	"github.com/jessevdk/go-flags"
)

func parseOptions(options interface{}) {
	sLogger.Debug("loading cli options into interface")
	sLogger.Debug(reflect.TypeOf(options).String())
	if _, err := flags.ParseArgs(options, os.Args); err != nil {
		if parseErr, ok := err.(*flags.Error); ok {
			if parseErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		sLogger.Fatal(err.Error())
	}
	sLogger.Debug("successfully loaded cli options")
}

type GlobalOptions struct {
	LogLevel      []bool `short:"l" long:"log-level" description:"Level of logging verbosity"`
	ChangelogFile string `short:"f" long:"changelog-file" description:"Location of the changelog file" default:"./CHANGELOG.md"`
}

type GeneralGitOptions struct {
	GitBranch           string `short:"b" long:"git-branch" description:"Git branch to run against"`
	GitWorkingDirectory string `short:"w" long:"git-workdir" description:"Working directory of the git repository" default:"./"`
	SkipGitCheckout     bool   `short:"s" long:"skip-git-checkout" description:"Skip running git checkout?"`
}

type NewVersionOptions struct {
	GlobalOptions
	GeneralGitOptions
	Increment                 string   `short:"i" long:"increment" description:"The incrementation level to use"`
	Force                     bool     `short:"o" long:"force" description:"If there's a pending release in the changelog, should it be overwritten by this run?"`
	Manual                    bool     `short:"m" long:"manual" description:"Don't attempt to evaluate any changes from git, and only load manually"`
	NonInteractive            bool     `short:"n" long:"non-interactive" description:"Should the step be run non interactively?"`
	IgnoreConventionalCommits bool     `short:"g" long:"ignore-conventionalcommits" description:"Should conventional commits be ignored?"`
	Added                     []string `short:"a" long:"added" description:"What was added in this new release?"`
	Changed                   []string `short:"c" long:"changed" description:"What was changed in this new release?"`
	Deprecated                []string `short:"d" long:"deprecated" description:"What was deprecated in this new release?"`
	Removed                   []string `short:"r" long:"removed" description:"What was removed in this new release?"`
	Fixed                     []string `short:"x" long:"fixed" description:"What was fixed in this new release?"`
	Security                  []string `short:"e" long:"security" description:"What was security related in this new release?"`
}

type GitLookupOptions struct {
	GitEvaluate         bool   `short:"e" long:"git-evaluate" description:"Should git branches be evaluated when calcuating the most recent version?"`
	GitWorkingDirectory string `short:"w" long:"git-workdir" description:"Working directory of the git repository" default:"./"`
	GitPrefix           string `short:"p" long:"git-prefix" description:"The branch name prefix for releases" default:"release"`
	UseTags             bool   `short:"t" long:"use-tags" description:"Use tags for release, instead of branches"`
}

type UpdateOptions struct {
	GlobalOptions
	GitLookupOptions
	GitBranch string `short:"b" long:"git-branch" description:"Git branch to run against"`
}

type ReleaseOptions struct {
	UpdateOptions
	SkipGitCheckout  bool     `short:"s" long:"skip-git-checkout" description:"Skip running git checkout?"`
	NonInteractive   bool     `short:"n" long:"non-interactive" description:"Should the step be run non interactively?"`
	GitCommitMessage string   `short:"m" long:"git-commit-message" description:"The message to use for the git commit" default:"[skip ci] Release version %s"`
	ReleaseFiles     []string `short:"r" long:"release-file" description:"Additional files to add to the release"`
	VersionPrefix    string   `short:"v" long:"version-prefix" description:"Prefix for the version" default:"v"`
}

type EnforceConventionalCommitsOptions struct {
	GlobalOptions
	GeneralGitOptions
	Depth                       int  `short:"d" long:"depth" description:"How deep to go when checking that all commits are conventional" default:"1"`
	UseLastChangelogChange      bool `short:"u" long:"use-changelog" description:"Automatically determine the depth by finding the last time changelog changed in the tree, and comparing from there. If set, will override depth option"`
	AllowNonConventionalcommits bool `short:"a" long:"allow" description:"Allows non conventional commits to be present. Will pass if at least one conventional commits is found"`
}
