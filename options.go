package main

type GlobalOptions struct {
	LogLevel         []bool `short:"l" long:"log-level" description:"Level of logging verbosity"`
	DryRun           bool   `short:"d" long:"dry-run" description:"Is this a dry run?"`
	WorkingDirectory string `short:"w" long:"workdir" description:"Working directory of the git repository" default:"./"`
	UpdateChangelog  bool   `short:"c" long:"update-changelog" description:"Should a changelog file be ept and updated?"`
}

type GitOptions struct {
	Branch           *string `short:"b" long:"branch" description:"(Optional) Git branch to run against, otherwise looks for default"`
	ReleasePrefix    *string `short:"r" long:"branch-prefix" description:"(Optional) Prefix to give to released branches" default:"release"`
	Remote           *string `short:"o" long:"remote" description:"(Optional) Git remote name, otherwise looks for default"`
	IsPrerelease     bool    `short:"p" long:"prerelease" description:"Is this a prerelease?"`
	PrereleasePrefix *string `short:"e" long:"pr-prefix" description:"(Optional) Prefix for prereleases" default:"prerelease"`
	SkipFetch        bool    `short:"f" long:"skip-fetch" description:"Should the tool try to skip running fetch?"`
	UseTags          bool    `short:"t" long:"use-tags" description:"Use tags rather than branches?"`
	VersionPrefix    *string `short:"v" long:"ver-prefix" description:"(Optional) Prefix for the version" default:"v"`
	BuildID          *string `short:"i" long:"build-id" description:"(Optional) An ID for a build to append to the version"`
}

type LookupVersionOptions struct {
	GitOptions
	Scope string `short:"s" long:"scope" description:"Specify the scope for a version" default:""`
}
