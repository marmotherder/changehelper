package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

func main() {
	setupLogger(0)

	gOpts := GlobalOptions{}
	gParser := flags.NewParser(&gOpts, flags.IgnoreUnknown)
	args, err := gParser.Parse()
	if err != nil {
		sLogger.Fatal(err)
	}

	setupLogger(len(gOpts.LogLevel))

	switch args[0] {
	case "print-current-version":
		fmt.Print(mustGetLatestVersion(gOpts))
		os.Exit(0)
	case "print-next-version":
		fmt.Print(mustGetNextVersion(gOpts))
		os.Exit(0)
	case "update-changelog":

	case "release":
		mustUpdateReleaseVersions(gOpts)
	}
}
