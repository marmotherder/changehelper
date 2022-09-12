package main

import (
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/marmotherder/go-semverhandler"
)

func mustGetHandlerAndReleases(workingDirectory string, opts GitOptions) (semverhandler.SemverHandler, map[string][]semverhandler.Version) {
	handler := semverhandler.SemverHandler{
		Logger:        sLogger,
		Git:           mustLoadGit(workingDirectory),
		ReleasePrefix: opts.ReleasePrefix,
		VersionPrefix: opts.VersionPrefix,
	}

	releases, err := handler.GetExistingScopedReleases(semverhandler.GetExistingScopedReleases{
		SkipFetch: opts.SkipFetch,
		UseTags:   opts.UseTags,
	})
	if err != nil {
		sLogger.Warn("failed to lookup any historic releases")
		sLogger.Fatal(err)
	}

	return handler, releases
}

func mustGetNextVersion(gOpts GlobalOptions) string {
	opts := LookupVersionOptions{}
	parseArgs(&opts)

	handler, releases := mustGetHandlerAndReleases(gOpts.WorkingDirectory, opts.GitOptions)

	newReleases, err := handler.LoadNewAndUpdatedReleases(releases, semverhandler.LoadNewAndUpdatedReleasesOptions{
		Branch:  opts.Branch,
		UseTags: opts.UseTags,
	})

	if err != nil {
		sLogger.Warn("failed to load any new or updated releases")
		sLogger.Fatal(err)
	}

	for scope, data := range newReleases {
		if scope == opts.Scope {
			version := semver.MustParse("0.0.0")
			if data.Version != nil {
				version = data.Version.Version
			}
			updatedVersion, err := handler.UpdateVersion(scope, data.Increment, version, opts.IsPrerelease, opts.PrereleasePrefix, opts.BuildID)
			if err != nil {
				if scope != "" {
					sLogger.Errorf("failed to get the updated version for scope %s", scope)
				} else {
					sLogger.Error("failed to get the updated version")
				}
				sLogger.Fatal(err)
			}
			if updatedVersion == nil {
				if scope != "" {
					sLogger.Fatalf("updated version for scope %s came back empty", scope)
				}
				sLogger.Fatal("updated version came back empty")
			}

			return updatedVersion.String()
		}
	}

	if opts.Scope != "" {
		fmt.Printf("no updates found for the selected scope %s\n", opts.Scope)
	} else {
		fmt.Println("no updates found")
	}

	os.Exit(1)
	return ""
}

func mustGetLatestVersion(gOpts GlobalOptions) string {
	opts := LookupVersionOptions{}
	parseArgs(&opts)

	_, releases := mustGetHandlerAndReleases(gOpts.WorkingDirectory, opts.GitOptions)

	for scope, versions := range releases {
		if scope == opts.Scope && len(versions) > 0 {
			return versions[0].Version.String()
		}
	}

	if opts.Scope != "" {
		fmt.Printf("no releases found for the selected scope %s\n", opts.Scope)
	} else {
		fmt.Println("no releases found")
	}

	os.Exit(1)
	return ""
}

func mustUpdateReleaseVersions(gOpts GlobalOptions) {
	opts := GitOptions{}
	parseArgs(&opts)

	handler, releases := mustGetHandlerAndReleases(gOpts.WorkingDirectory, opts)

	newReleases, err := handler.LoadNewAndUpdatedReleases(releases, semverhandler.LoadNewAndUpdatedReleasesOptions{
		Branch:  opts.Branch,
		UseTags: opts.UseTags,
	})
	if err != nil {
		sLogger.Warn("failed to load any new or updated releases")
		sLogger.Fatal(err)
	}

	if err := handler.CreateUpdateReleases(newReleases, semverhandler.CreateUpdateReleasesOptions{
		UseTags:          opts.UseTags,
		IsPrerelease:     opts.IsPrerelease,
		PrereleasePrefix: opts.PrereleasePrefix,
		BuildID:          opts.BuildID,
	}); err != nil {
		sLogger.Warn("failed to create or update releases")
		sLogger.Fatal(err)
	}
}
