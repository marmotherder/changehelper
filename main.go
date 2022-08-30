package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/jessevdk/go-flags"
	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
)

const (
	majorIncrement = 4
	minorIncrement = 3
	patchIncrement = 2
	buildIncrement = 1
)

func main() {
	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		if parseErr, ok := err.(*flags.Error); ok {
			if parseErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		fmt.Println(err)
	}

	setupLogger(len(opts.LogLevel))

	machineOptions := []conventionalcommits.MachineOption{
		conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
		conventionalcommits.WithBestEffort(),
	}

	git := &gitCli{
		WorkingDirectory: opts.WorkingDirectory,
	}

	if opts.Remote != nil {
		git.Remote = *opts.Remote
	} else {
		if err := git.getRemote(); err != nil {
			sLogger.Fatal("was unable to find a git remote")
		}
	}

	h := &handler{
		options:   opts,
		ccMachine: parser.NewMachine(machineOptions...),
		git:       git,
	}

	refs, err := h.getReleaseRefs()
	if err != nil {
		sLogger.Fatal(err)
	}

	scopedRefs := refsToOrderedScopedVersions(refs)

	for scope, refs := range scopedRefs {
		prefix := ""
		if scope != "" {
			prefix = scope + "/"
		}

		lastReleasedCommit, err := h.getLastCommitOnRef(prefix + refs[0].ver)
		if err != nil || lastReleasedCommit == nil {
			sLogger.Errorf("failed to get latest commit for scope %s", scope)
			continue
		}

		trimmedLastReleasedCommit := strings.TrimSpace(*lastReleasedCommit)
		commits, err := h.listCommits(trimmedLastReleasedCommit)
		if err != nil {
			sLogger.Errorf("failed to list commits after %s for scope %s", trimmedLastReleasedCommit, commits)
			continue
		}

		incomingVersion := semver.MustParse(refs[0].sver.String())
		switch h.determineIncrementFromCommits(commits) {
		case majorIncrement:
			incomingVersion.Major++
		case minorIncrement:
			incomingVersion.Minor++
		case patchIncrement:
			incomingVersion.Patch++
		case buildIncrement:
			incomingVersion.Build = append(incomingVersion.Build, opts.BuildID)
		}

		if opts.Prerelease {
			prereleaseVer := uint64(1)

			if len(incomingVersion.Pre) > 0 {
				if incomingVersion.Pre[0].IsNum {
					prereleaseVer = incomingVersion.Pre[0].VersionNum
				} else {
					splitPrerelease := strings.Split(incomingVersion.Pre[0].VersionStr, "-")
					splitPrereleaseVer := splitPrerelease[len(splitPrerelease)-1]
					splitPrereleaseVerParsed, err := strconv.ParseUint(splitPrereleaseVer, 10, 64)
					if err != nil {
						sLogger.Infof("could not parse number on existing prerelease version %s", splitPrereleaseVer)
						sLogger.Info(err)
					} else {
						prereleaseVer = splitPrereleaseVerParsed
					}
				}
			}

			if opts.PrereleasePrefix != nil {
				incomingVersion.Pre = append(incomingVersion.Pre, semver.PRVersion{
					VersionStr: fmt.Sprintf("%s-%d", *opts.PrereleasePrefix, prereleaseVer),
					IsNum:      false,
				})
			} else {
				incomingVersion.Pre = append(incomingVersion.Pre, semver.PRVersion{
					VersionNum: prereleaseVer,
					IsNum:      true,
				})
			}
		}

		if !incomingVersion.Equals(refs[0].sver) {
			sb := strings.Builder{}
			if opts.BranchPrefix != nil {
				sb.WriteString(fmt.Sprintf("%s/", *opts.BranchPrefix))
			}
			sb.WriteString(prefix)
			if opts.VersionPrefix != nil {
				sb.WriteString(*opts.VersionPrefix)
			}
			sb.WriteString(incomingVersion.String())

			releaseRef := sb.String()

			if !opts.DryRun {
				refType := "heads"
				if opts.Tags {
					refType = "tags"
				}

				h.git.forcePushHashToRef(trimmedLastReleasedCommit, releaseRef, refType)
			} else {
				sLogger.Infof("would have created a release for %s", releaseRef)
			}
		}
	}
}
