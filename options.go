package main

type Options struct {
	LogLevel         []bool  `short:"l" long:"log-level" description:"Level of logging verbosity"`
	DryRun           bool    `short:"d" long:"dry-run" description:"Is this a dry run?"`
	WorkingDirectory string  `short:"w" long:"workdir" description:"Working directory of the git repository" default:"./"`
	Branch           *string `short:"b" long:"branch" description:"(Optional) Git branch to run against, otherwise looks for default"`
	ReleasePrefix    *string `short:"r" long:"branch-prefix" description:"(Optional) Prefix to give to released branches" default:"release"`
	Remote           *string `short:"o" long:"remote" description:"(Optional) Git remote name, otherwise looks for default"`
	Prerelease       bool    `short:"p" long:"prerelease" description:"Is this a prerelease?"`
	PrereleasePrefix *string `short:"e" long:"pr-prefix" description:"(Optional) Prefix for prereleases" default:"prerelease"`
	SkipFetch        bool    `short:"s" long:"skip-fetch" description:"Should the tool try to skip running fetch?"`
	Tags             bool    `short:"t" long:"tags" description:"Use tags rather than branches?"`
	VersionPrefix    *string `short:"v" long:"ver-prefix" description:"(Optional) Prefix for the version" default:"v"`
	BuildID          string  `short:"i" long:"build-id" description:"(Optional) An ID for a build to append to the version"`
}
