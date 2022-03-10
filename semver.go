package main

import (
	"github.com/blang/semver"
)

const (
	PATCH = "PATCH"
	MINOR = "MINOR"
	MAJOR = "MAJOR"
)

func getLatestRelease(released []*change) *change {
	changeMap := map[string]*change{}
	releasedVersions := make([]semver.Version, 0)

	for _, change := range released {
		changeMap[change.Version.String()] = change
		releasedVersions = append(releasedVersions, *change.Version)
	}

	if len(releasedVersions) <= 0 {
		return nil
	}

	semver.Sort(releasedVersions)

	if change, ok := changeMap[releasedVersions[len(releasedVersions)-1].String()]; ok {
		return change
	}

	return nil
}

func listReleasedVersionFromGit(useTags bool, git gitCli, prefix string, remotes ...string) ([]semver.Version, error) {
	releaseRefs, err := git.listRemoteBranches(prefix, remotes...)
	if err != nil {
		return nil, err
	}

	if useTags {
		releaseRefs, err = git.listTags(prefix)
		if err != nil {
			return nil, err
		}
	}

	releasedVersions := make([]semver.Version, 0)
	for _, releaseRef := range releaseRefs {
		sLogger.Debugf("trying to parse trimmed release name %s to a version", releaseRef)
		version, err := semver.ParseTolerant(releaseRef)
		if err != nil {
			sLogger.Debugf("failed to parse trimmed release name %s to a version", releaseRef)
			sLogger.Debug(err.Error())
			continue
		}

		releasedVersions = append(releasedVersions, version)
	}

	return releasedVersions, nil
}

func updateUnreleasedVersion(unreleased *change, increment *string) {
	if increment != nil {
		switch *increment {
		case PATCH:
			unreleased.Version.Patch++
		case MINOR:
			unreleased.Version.Minor++
			unreleased.Version.Patch = 0
		case MAJOR:
			unreleased.Version.Major++
			unreleased.Version.Minor = 0
			unreleased.Version.Patch = 0
		}
	}
}
