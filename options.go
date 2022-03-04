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
		sLogger.Fatal(err.Error())
	}
	sLogger.Debug("successfully loaded cli options")
}

type GlobalOptions struct {
	LogLevel      []bool `short:"l" long:"log-level" description:"Level of logging verbosity"`
	ChangelogFile string `short:"f" long:"changelog-file" description:"Location of the changelog file" default:"./CHANGELOG.md"`
}

type NewVersionOptions struct {
	GlobalOptions
	GitBranch           string   `short:"b" long:"git-branch" description:"Git branch to run against" default:"main"`
	GitWorkingDirectory string   `short:"w" long:"git-workdir" description:"Working directory of the git repository" default:"./"`
	Increment           string   `short:"i" long:"increment" description:"The incrementation level to use"`
	Force               bool     `short:"o" long:"force" description:"If there's a pending release in the changelog, should it be overwritten by this run?"`
	Manual              bool     `short:"m" long:"manual" description:"Don't attempt to evaluate any changes from git, and only load manually"`
	Added               []string `short:"a" long:"added" description:"What was added in this new release?"`
	Changed             []string `short:"c" long:"changed" description:"What was changed in this new release?"`
	Deprecated          []string `short:"d" long:"deprecated" description:"What was deprecated in this new release?"`
	Removed             []string `short:"r" long:"removed" description:"What was removed in this new release?"`
	Fixed               []string `short:"x" long:"fixed" description:"What was fixed in this new release?"`
	Security            []string `short:"s" long:"security" description:"What was security related in this new release?"`
}

type UpdateOptions struct {
	GlobalOptions
	GitBranch           string `short:"b" long:"git-branch" description:"Git branch to run against"`
	GitWorkingDirectory string `short:"w" long:"git-workdir" description:"Working directory of the git repository" default:"./"`
}
