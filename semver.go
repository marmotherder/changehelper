package main

import (
	"github.com/blang/semver"
	"github.com/yuin/goldmark/ast"
)

const (
	PATCH = "PATCH"
	MINOR = "MINOR"
	MAJOR = "MAJOR"
)

func getLatestRelease(released []change) *change {
	nodeMap := map[string]ast.Node{}
	releasedVersions := make([]semver.Version, 0)

	for _, change := range released {
		nodeMap[change.Version.String()] = change.Node
		releasedVersions = append(releasedVersions, *change.Version)
	}

	if len(releasedVersions) <= 0 {
		return nil
	}

	semver.Sort(releasedVersions)

	if node, ok := nodeMap[releasedVersions[0].String()]; ok {
		return &change{
			Version: &releasedVersions[0],
			Node:    node,
		}
	}

	return &change{
		Version: &releasedVersions[0],
		Node:    nil,
	}
}

func listReleasedVersionFromGit(dir, prefix string, remotes ...string) ([]semver.Version, error) {
	releaseBranches, err := listRemoteGitBranches(dir, prefix, remotes...)
	if err != nil {
		return nil, err
	}

	releasedVersions := make([]semver.Version, 0)
	for _, releaseBranch := range releaseBranches {
		sLogger.Debugf("trying to parse trimmed branch name %s to a version", releaseBranch)
		version, err := semver.ParseTolerant(releaseBranch)
		if err != nil {
			sLogger.Debugf("failed to parse trimmed branch name %s to a version", releaseBranch)
			sLogger.Debug(err.Error())
			continue
		}

		releasedVersions = append(releasedVersions, version)
	}

	return releasedVersions, nil
}
