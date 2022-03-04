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
		if strings.Contains(message, string(supportedType)+":") || scopedRegex.MatchString(message) {
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
