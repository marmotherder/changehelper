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

const changelogHeader = `# Changelog
All notable changes to this project will be documented in this file.
		
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

`

const (
	releasePrefix = "## "
	changePrefix  = "### "
	linePrefix    = "- "
)

func parseChangelog(changelogFile string) (changelog []byte, unreleased *change, increment *string, released []*change, err error) {
	sLogger.Infof("reading the changelog file %s for parsing", changelogFile)

	released = make([]*change, 0)

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
	var currentChange *change
	var currentChangeType changeType
	currentChangeType = ""
	for {
		if current == nil {
			break
		}

		currentText := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(string(current.Text(changelog)))), " ", "")
		prefix := ""

		if unreleasedRegex.MatchString(currentText) {
			if unreleased != nil {
				err = fmt.Errorf("duplicate pending unreleased changes found")
				break
			}

			unreleased = &change{
				Version:     nil,
				VersionText: nodeText(current, changelog),
			}
			currentChange = unreleased
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
				Version:     nil,
				VersionText: nodeText(current, changelog),
			}
			currentChange = unreleased
		}
		if releasedRegex.MatchString(currentText) {
			currentChange = nil
			currentChangeType = ""
			extractedVersion := versionRegex.FindAllString(string(current.Text(changelog)), -1)
			if len(extractedVersion) > 0 {
				cleanVersion := strings.ReplaceAll(strings.ReplaceAll(extractedVersion[0], "[", ""), "]", "")
				version, parseError := semver.Parse(cleanVersion)
				if parseError != nil {
					sLogger.Warn("failed to parse changelog node text as version")
					sLogger.Warn(currentText)
					sLogger.Warn(parseError.Error())
				} else {
					releasedChange := &change{
						Version:     &version,
						VersionText: nodeText(current, changelog),
					}
					prefix = releasePrefix
					currentChange = releasedChange
					released = append(released, releasedChange)
				}
			}
		}

		if currentChange != nil {
			text := *nodeText(current, changelog)

			switch currentText {
			case strings.ToLower(string(changeAdded)):
				currentChangeType = changeAdded
				prefix = changePrefix
			case strings.ToLower(string(changeChanged)):
				currentChangeType = changeChanged
				prefix = changePrefix
			case strings.ToLower(string(changeDeprecated)):
				currentChangeType = changeDeprecated
				prefix = changePrefix
			case strings.ToLower(string(changeRemoved)):
				currentChangeType = changeRemoved
				prefix = changePrefix
			case strings.ToLower(string(changeFixed)):
				currentChangeType = changeFixed
				prefix = changePrefix
			case strings.ToLower(string(changeSecurity)):
				currentChangeType = changeSecurity
				prefix = changePrefix
			default:
				if currentChangeType != "" {
					switch currentChangeType {
					case changeAdded:
						currentChange.Added = append(currentChange.Added, text)
						prefix = linePrefix
					case changeChanged:
						currentChange.Changed = append(currentChange.Changed, text)
						prefix = linePrefix
					case changeDeprecated:
						currentChange.Deprecated = append(currentChange.Deprecated, text)
						prefix = linePrefix
					case changeRemoved:
						currentChange.Removed = append(currentChange.Removed, text)
						prefix = linePrefix
					case changeFixed:
						currentChange.Fixed = append(currentChange.Fixed, text)
						prefix = linePrefix
					case changeSecurity:
						currentChange.Security = append(currentChange.Security, text)
						prefix = linePrefix
					}
				}
			}

			fullText := prefix + text
			if currentChange.Text != nil {
				fullText = *currentChange.Text + "\n" + prefix + text
			}
			currentChange.Text = &fullText
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
