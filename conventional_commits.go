package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
)

const (
	conventionalCommitFix      conventionalCommitType = "fix"
	conventionalCommitFeat     conventionalCommitType = "feat"
	conventionalCommitBuild    conventionalCommitType = "build"
	conventionalCommitChore    conventionalCommitType = "chore"
	conventionalCommitCI       conventionalCommitType = "ci"
	conventionalCommitDocs     conventionalCommitType = "docs"
	conventionalCommitStyle    conventionalCommitType = "style"
	conventionalCommitRefactor conventionalCommitType = "refactor"
	conventionalCommitPerf     conventionalCommitType = "perf"
	conventionalCommitTest     conventionalCommitType = "test"
)

type conventionalCommitType string

func getConventionalCommitType(message string) *conventionalCommitType {
	supportedTypes := []conventionalCommitType{
		conventionalCommitFix,
		conventionalCommitFeat,
		conventionalCommitBuild,
		conventionalCommitChore,
		conventionalCommitCI,
		conventionalCommitDocs,
		conventionalCommitStyle,
		conventionalCommitRefactor,
		conventionalCommitPerf,
		conventionalCommitTest,
	}

	for _, supportedType := range supportedTypes {
		scopedRegex := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\):`, supportedType))
		breakingScopedRegex := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\)!:`, supportedType))
		if strings.Contains(message, string(supportedType)+":") ||
			strings.Contains(message, string(supportedType)+"!:") ||
			scopedRegex.MatchString(message) ||
			breakingScopedRegex.MatchString(message) {
			return &supportedType
		}
	}

	return nil
}

func parseConventionalCommitMessages(commitMessages ...string) (*string, map[int]conventionalCommitType) {
	var increment string
	mappedTypes := map[int]conventionalCommitType{}
	machineOptions := []conventionalcommits.MachineOption{
		conventionalcommits.WithTypes(conventionalcommits.TypesConventional),
		conventionalcommits.WithBestEffort(),
	}
	machine := parser.NewMachine(machineOptions...)

	for idx, commitMessage := range commitMessages {
		ccMessage, err := machine.Parse([]byte(commitMessage))
		if err != nil {
			sLogger.Debugf("failed to parse commit '%s' as conventional commit: %s", commitMessage, err.Error())
			continue
		}
		if !ccMessage.Ok() {
			continue
		}

		ccType := getConventionalCommitType(commitMessage)
		if ccType == nil {
			sLogger.Warnf("failed to find appropriate conventional commit type in message: %s, skipping", commitMessage)
			continue
		}

		mappedTypes[idx] = *ccType

		if ccMessage.IsBreakingChange() {
			increment = MAJOR
		} else if increment != MAJOR {
			if *ccType == conventionalCommitFeat {
				increment = MINOR
			} else if increment != MINOR {
				increment = PATCH
			}
		}
	}

	if increment == "" {
		sLogger.Fatal("failed to find an increment for a change")
	}

	return &increment, mappedTypes
}

func resolveConventionalCommits(git gitCli, changelogFile string) (*string, map[string]string, map[string]string, map[string]string, map[string]string, error) {
	lastCommit, err := git.getLastModifiedCommit(changelogFile)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	commits, err := git.listCommits(*lastCommit + "..")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	selfCommit, err := git.listCommits("-n 1 " + *lastCommit)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	commits = append(commits, selfCommit...)
	uniqueCommits := []gitCommit{}

	uniqueHashes := []string{}
	for _, commit := range commits {
		hasHash := func() bool {
			for _, uniqueHash := range uniqueHashes {
				if uniqueHash == commit.Hash {
					return true
				}
			}

			uniqueHashes = append(uniqueHashes, commit.Hash)
			return false
		}()

		if hasHash {
			continue
		}

		uniqueCommits = append(uniqueCommits, commit)
	}

	var commitMessages []string
	for _, commit := range uniqueCommits {
		commitMessages = append(commitMessages, commit.Message)
	}

	increment, mappedTypes := parseConventionalCommitMessages(commitMessages...)

	fixedUnique := map[string]string{}
	addedUnique := map[string]string{}
	changedUnique := map[string]string{}
	removedUnique := map[string]string{}
	for idx, ccType := range mappedTypes {
		commit := uniqueCommits[idx]

		diff, err := git.getRefChanges(commit.Hash)
		if err != nil {
			sLogger.Warnf("failed to read changes for commit %s, changes will not be recorded in changelog", commit.Hash)
		}

		switch ccType {
		case conventionalCommitFix:
			for _, changed := range diff.Changed {
				if existing, ok := fixedUnique[changed]; ok {
					fixedUnique[changed] = existing + ", " + commit.Message
				} else {
					fixedUnique[changed] = commit.Message
				}
			}
			fallthrough
		default:
			for _, added := range diff.Added {
				if existing, ok := addedUnique[added]; ok {
					addedUnique[added] = existing + ", " + commit.Message
				} else {
					addedUnique[added] = commit.Message
				}
			}
			if ccType != conventionalCommitFix {
				for _, changed := range diff.Changed {
					if existing, ok := changedUnique[changed]; ok {
						changedUnique[changed] = existing + ", " + commit.Message
					} else {
						changedUnique[changed] = commit.Message
					}
				}
			}
			for _, removed := range diff.Removed {
				if existing, ok := removedUnique[removed]; ok {
					removedUnique[removed] = existing + ", " + commit.Message
				} else {
					removedUnique[removed] = commit.Message
				}
			}
		}
	}

	return increment, fixedUnique, addedUnique, changedUnique, removedUnique, nil
}
