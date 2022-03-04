package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func parseChangelog(changelogFile string) (changelog []byte, unreleased *change, increment *string, released []change, err error) {
	sLogger.Infof("reading the changelog file %s for parsing", changelogFile)

	released = make([]change, 0)

	changelog, err = readChangelogFile(changelogFile)
	if err != nil {
		return
	}

	defer func() {
		if recover() != nil {
			unreleased = nil
			increment = nil
		}
	}()

	unreleasedRegex, unreleasedIncrementRegex, releasedRegex, versionRegex, err := parsingRegexes()
	if err != nil {
		return
	}

	changelogNode := goldmark.DefaultParser().Parse(text.NewReader(changelog))

	current := changelogNode.FirstChild()
	for {
		if current == nil {
			break
		}

		currentText := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(string(current.Text(changelog)))), " ", "")

		if unreleasedRegex.MatchString(currentText) {
			if unreleased != nil {
				err = fmt.Errorf("duplicate pending unreleased changes found")
				break
			}

			unreleased = &change{
				Version: nil,
				Node:    current,
			}
		}
		if unreleasedIncrementRegex.MatchString(currentText) {
			if unreleased != nil {
				err = fmt.Errorf("duplicate pending unreleased changes found")
				break
			}

			unreleasedIncrement := strings.SplitN(currentText, "-", 2)
			upperIncrement := strings.ToUpper(unreleasedIncrement[1])
			increment = &upperIncrement

			unreleased = &change{
				Version: nil,
				Node:    current,
			}
		}
		if releasedRegex.MatchString(currentText) {
			extractedVersion := versionRegex.FindAllString(string(current.Text(changelog)), -1)
			if len(extractedVersion) > 0 {
				cleanVersion := strings.ReplaceAll(strings.ReplaceAll(extractedVersion[0], "[", ""), "]", "")
				version, parseError := semver.Parse(cleanVersion)
				if parseError != nil {
					sLogger.Warn("failed to parse changelog node text as version")
					sLogger.Warn(currentText)
					sLogger.Warn(parseError.Error())
				} else {
					released = append(released, change{
						Version: &version,
						Node:    current,
					})
				}
			}
		}

		current = current.NextSibling()
	}

	return
}

func readChangelogFile(changelogFile string) ([]byte, error) {
	sLogger.Debugf("starting read of changelog file %s", changelogFile)

	changelog, err := os.ReadFile(changelogFile)
	if err != nil {
		sLogger.Errorf("failed to open changelog file %s", changelogFile)
		return nil, err
	}

	sLogger.Infof("successfully read changelog file %s", changelogFile)
	sLogger.Debugf("changelog contents:")
	sLogger.Debug(string(changelog))

	return changelog, nil
}

func parsingRegexes() (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp, *regexp.Regexp, error) {
	unreleasedRegex, err := regexp.Compile(`\[unreleased]$`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	unreleasedIncrementRegex, err := regexp.Compile(`\[unreleased]-[a-zA-Z]+$`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	releasedRegex, err := regexp.Compile(`\[\d+\.\d+\.\d+]-[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	versionRegex, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return unreleasedRegex, unreleasedIncrementRegex, releasedRegex, versionRegex, nil
}
