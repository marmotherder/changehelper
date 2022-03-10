package main

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/manifoldco/promptui"
)

func captureMultiLineInput(query, queryContinue, label string, obj *[]string) error {
	queryItems := []string{"No", "Yes"}

	querySelect := promptui.Select{
		Label: query,
		Items: queryItems,
	}

	_, queryResult, err := querySelect.Run()
	if err != nil {
		sLogger.Errorf("failed to capture query from label: '%s'", query)
		return err
	}

	if queryResult == "Yes" {
		for {
			prompt := promptui.Prompt{
				Label: label,
				Validate: func(input string) error {
					if input == "" {
						return errors.New("no text entered")
					}
					return nil
				},
			}

			result, err := prompt.Run()
			if err != nil {
				sLogger.Errorf("failed to capture query from label: '%s'", label)
				return err
			}

			*obj = append(*obj, result)

			queryContinueSelect := promptui.Select{
				Label: queryContinue,
				Items: queryItems,
			}

			_, queryContinueResults, err := queryContinueSelect.Run()
			if err != nil {
				sLogger.Errorf("failed to capture query from label: '%s'", queryContinueResults)
				return err
			}

			if queryContinueResults != "Yes" {
				break
			}
		}
	}

	return nil
}

func mustCaptureMultiLineInput(query, queryContinue, label string, obj *[]string) {
	if err := captureMultiLineInput(query, queryContinue, label, obj); err != nil {
		sLogger.Fatal(err.Error())
	}
}

func mustHaveBranch(branch, label string, nonInteractive bool, git gitCli) string {
	if branch != "" {
		return branch
	}

	currentBranch, err := git.getCurrentBranch()
	if err != nil {
		sLogger.Warn("failed to get the current git branch")
		sLogger.Error(err.Error())
	}
	if currentBranch != nil {
		return *currentBranch
	}

	if !nonInteractive {
		branchPrompt := promptui.Prompt{
			Label: label,
			Validate: func(input string) error {
				if err := git.checkoutAndPull(input); err != nil {
					return err
				}
				return nil
			},
		}

		branchResp, err := branchPrompt.Run()
		if err != nil {
			sLogger.Fatal(err.Error())
		}

		return branchResp
	}

	sLogger.Fatal("could not find a git branch to use")
	return ""
}

func resolvePathToRelativePath(path, relative string) (*string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		sLogger.Errorf("failed to resolve path %s to absolute", path)
		return nil, err
	}

	relativeAbsPath, err := filepath.Abs(relative)
	if err != nil {
		sLogger.Errorf("failed to resolve path %s to absolute", relative)
		return nil, err
	}

	relativePath := strings.ReplaceAll(absPath, relativeAbsPath, ".")

	return &relativePath, nil
}

func getCurrentVersion(changelogFile string) (*semver.Version, error) {
	_, _, _, released, err := parseChangelog(changelogFile)
	if err != nil {
		return nil, err
	}

	latest := getLatestRelease(released)

	if latest != nil && latest.Version != nil {
		return latest.Version, nil
	}

	return nil, errors.New("no releases found in changelog file")
}

func getUnreleasedVersion(changelogFile string) (*semver.Version, error) {
	_, unreleased, increment, released, err := parseChangelog(changelogFile)
	if err != nil {
		return nil, err
	}

	if unreleased == nil {
		return nil, errors.New("an unreleased change couldn't be found")
	}

	if increment == nil {
		return nil, errors.New("the unreleased change has no increment set, so version cannot be determined")
	}

	if len(released) > 0 {
		latest := getLatestRelease(released)
		unreleased.Version = latest.Version
	} else {
		defaultVersion := semver.MustParse("0.0.0")
		unreleased.Version = &defaultVersion
	}

	updateUnreleasedVersion(unreleased, increment)

	return unreleased.Version, nil
}

func getRemote(git gitCli) string {
	remote := "origin"
	origin, err := git.getRemote()
	if err != nil {
		sLogger.Error(err.Error())
	}
	if origin != nil && *origin != "" {
		remote = *origin
	}

	return remote
}
