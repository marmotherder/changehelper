package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

const operationsText = `Usage = changehelper [global options] <operation>

Operations:

new-version				Create a new in progress version interactively
print-current-version			Print the current version in the changelog file
print-unreleased-version		Print the unreleased version based on the changelog file, or conventional commit(s)
print-current-changes			Print the most recent changes recorded in changelog that have been released
print-unreleased-changes		Print the most recent unreleased changes recorded in changelog
print-changes				Print any changes matching the input version
update					Update the version in the changelog file
release					Commit and push changes to git, ie changes to the changelog, and branches
update-and-release			Run update, followed by release in order
enforce-unreleased			Validate that there is a pending unreleased change
enforce-conventional-commits		Enforce that all commits adhere to conventional commit standards
version					Print the tool version

Global Options:

LogLevel		-l, --log-level		Logging level verbosity, set at increasing level by calling the flag multiple times, eg. -lll will run at Info level. By default, runs at Fatal. The levels supported, in ascending verbosity are Fatal, Error, Warn, Info, and Debug.
ChangelogFile		-f, --changelog-file 	Location of the changelog file at a path. Defaults to ./CHANGELOG.md
Help			-h, --help		Print the help options for the selected operation`

func main() {
	var options GlobalOptions
	parser := flags.NewParser(&options, flags.IgnoreUnknown)
	args, err := parser.ParseArgs(os.Args)
	if err != nil {
		handleArgsErr(err)
	}

	if len(args) < 2 {
		fmt.Println(operationsText)
		os.Exit(execError)
	}

	if setupErr := setupLogger(len(options.LogLevel)); setupErr != nil {
		sLogger.Fatal(err.Error())
	}

	operation := args[1]

	switch operation {

	case "enforce-unreleased":
		enforceUnreleased(options.ChangelogFile)
	case "enforce-conventional-commits":
		enforceConventionalCommits()
	case "new-version":
		newVersion()
	case "print-current-version":
		printCurrentVersion(options.ChangelogFile)
	case "print-unreleased-version":
		printUnreleasedVersion(options.ChangelogFile)
	case "print-current-changes":
		printCurrentChanges(options.ChangelogFile)
	case "print-unreleased-changes":
		printUnreleasedChanges(options.ChangelogFile)
	case "print-changes":
		printChanges()
	case "update":
		update(false)
	case "update-and-release":
		update(true)
		fallthrough
	case "release":
		release()
	case "version":
		fmt.Println(version)
	default:
		fmt.Println(operationsText)
		os.Exit(execError)
	}
}
