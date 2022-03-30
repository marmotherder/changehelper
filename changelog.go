package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
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

type changelogParser struct {
	CurrentNode ast.Node
	Changelog   []byte
	Unreleased  *change
	Increment   *string
	Released    []*change

	currentChange            *change
	unreleasedRegex          *regexp.Regexp
	unreleasedIncrementRegex *regexp.Regexp
	releasedRegex            *regexp.Regexp
	versionRegex             *regexp.Regexp

	currentDeformattedText string
	currentText            string
	prefix                 string

	currentChangeType changeType
}

func buildChangelogParser(currentNode ast.Node, changelog []byte) (*changelogParser, error) {
	unreleasedRegex, unreleasedIncrementRegex, releasedRegex, versionRegex, err := parsingRegexes()
	if err != nil {
		return nil, err
	}

	parser := changelogParser{
		CurrentNode: currentNode,
		Changelog:   changelog,
		Released:    make([]*change, 0),

		unreleasedRegex:          unreleasedRegex,
		unreleasedIncrementRegex: unreleasedIncrementRegex,
		releasedRegex:            releasedRegex,
		versionRegex:             versionRegex,
	}

	return &parser, nil
}

func (p *changelogParser) resetLoop() {
	p.currentDeformattedText = ""
	p.currentText = ""
	p.prefix = ""
	p.CurrentNode = p.CurrentNode.NextSibling()
}

func (p *changelogParser) getDeformattedText() string {
	if p.currentDeformattedText == "" {
		p.currentDeformattedText = strings.ReplaceAll(strings.TrimSpace(strings.ToLower(string(p.CurrentNode.Text(p.Changelog)))), " ", "")
	}

	return p.currentDeformattedText
}

func (p *changelogParser) getCurrentText() *string {
	if p.currentText == "" {
		p.currentText = string(p.CurrentNode.Text(p.Changelog))
	}

	return &p.currentText
}

func (p *changelogParser) checkUnreleased() error {
	if p.unreleasedRegex.MatchString(p.getDeformattedText()) {
		if p.Unreleased != nil {
			return fmt.Errorf("duplicate pending unreleased changes found")
		}

		versionText := releasePrefix + *p.getCurrentText()
		p.Unreleased = &change{
			Version:     nil,
			VersionText: &versionText,
		}
		p.currentChange = p.Unreleased
	}

	return nil
}

func (p *changelogParser) checkUnreleasedIncrement() error {
	if p.unreleasedIncrementRegex.MatchString(p.getDeformattedText()) {
		if p.Unreleased != nil {
			return fmt.Errorf("duplicate pending unreleased changes found")
		}

		unreleasedIncrement := strings.SplitN(p.getDeformattedText(), "-", 2)
		upperIncrement := strings.ToUpper(unreleasedIncrement[1])
		p.Increment = &upperIncrement

		versionText := releasePrefix + *p.getCurrentText()
		p.Unreleased = &change{
			Version:     nil,
			VersionText: &versionText,
		}
		p.currentChange = p.Unreleased
	}

	return nil
}

func (p *changelogParser) checkReleased() bool {
	if p.releasedRegex.MatchString(p.getDeformattedText()) {
		p.currentChange = nil
		p.currentChangeType = ""
		extractedVersion := p.versionRegex.FindAllString(*p.getCurrentText(), -1)
		if len(extractedVersion) > 0 {
			cleanVersion := strings.ReplaceAll(strings.ReplaceAll(extractedVersion[0], "[", ""), "]", "")
			version, parseError := semver.Parse(cleanVersion)
			if parseError != nil {
				sLogger.Warn("failed to parse changelog node text as version")
				sLogger.Warn(p.getDeformattedText())
				sLogger.Warn(parseError.Error())
			} else {

				versionText := releasePrefix + *p.getCurrentText()
				releasedChange := &change{
					Version:     &version,
					VersionText: &versionText,
				}
				p.prefix = releasePrefix
				p.currentChange = releasedChange
				p.Released = append(p.Released, releasedChange)

				return true
			}
		}
	}

	return false
}

func (p *changelogParser) processChange() {
	childText := loopChildren(p.CurrentNode, p.Changelog)

	switch p.getDeformattedText() {
	case strings.ToLower(string(changeAdded)):
		p.currentChangeType = changeAdded
		p.prefix = changePrefix
	case strings.ToLower(string(changeChanged)):
		p.currentChangeType = changeChanged
		p.prefix = changePrefix
	case strings.ToLower(string(changeDeprecated)):
		p.currentChangeType = changeDeprecated
		p.prefix = changePrefix
	case strings.ToLower(string(changeRemoved)):
		p.currentChangeType = changeRemoved
		p.prefix = changePrefix
	case strings.ToLower(string(changeFixed)):
		p.currentChangeType = changeFixed
		p.prefix = changePrefix
	case strings.ToLower(string(changeSecurity)):
		p.currentChangeType = changeSecurity
		p.prefix = changePrefix
	default:
		if p.currentChangeType != "" && p.prefix != changePrefix {
			p.processChangeEntries(childText)
		}
	}

	if p.prefix != "" {
		p.buildChangeText(childText)
	}
}

func (p *changelogParser) processChangeEntries(childText []string) {
	switch p.currentChangeType {
	case changeAdded:
		p.currentChange.Added = append(p.currentChange.Added, childText...)
		p.prefix = linePrefix
	case changeChanged:
		p.currentChange.Changed = append(p.currentChange.Changed, childText...)
		p.prefix = linePrefix
	case changeDeprecated:
		p.currentChange.Deprecated = append(p.currentChange.Deprecated, childText...)
		p.prefix = linePrefix
	case changeRemoved:
		p.currentChange.Removed = append(p.currentChange.Removed, childText...)
		p.prefix = linePrefix
	case changeFixed:
		p.currentChange.Fixed = append(p.currentChange.Fixed, childText...)
		p.prefix = linePrefix
	case changeSecurity:
		p.currentChange.Security = append(p.currentChange.Security, childText...)
		p.prefix = linePrefix
	}
}

func (p *changelogParser) buildChangeText(childText []string) {
	fullText := p.prefix + *p.getCurrentText()
	if p.prefix == linePrefix {
		sb := strings.Builder{}
		for _, cText := range childText {
			sb.WriteString(p.prefix)
			sb.WriteString(cText)
			sb.WriteString("\n")
		}
		fullText = sb.String()
		fullText = fullText[:len(fullText)-1]
	}

	if p.currentChange.Text != nil {
		fullText = *p.currentChange.Text + "\n" + fullText
	}
	p.currentChange.Text = &fullText
}

func parseChangelog(changelogFile string) ([]byte, *change, *string, []*change, error) {
	sLogger.Infof("reading the changelog file %s for parsing", changelogFile)

	changelog, err := readChangelogFile(changelogFile)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	changelogNode := goldmark.DefaultParser().Parse(text.NewReader(changelog))

	clogParser, err := loopNodes(changelogNode.FirstChild(), changelog)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return clogParser.Changelog, clogParser.Unreleased, clogParser.Increment, clogParser.Released, nil
}

func loopNodes(currentNode ast.Node, changelog []byte) (*changelogParser, error) {
	clogParser, err := buildChangelogParser(currentNode, changelog)
	if err != nil {
		return nil, err
	}

	defer func() {
		if recover() != nil {
			clogParser.Unreleased = nil
			clogParser.Increment = nil
		}
	}()

	for {
		if clogParser.CurrentNode == nil {
			break
		}

		if err := clogParser.checkUnreleased(); err != nil {
			return nil, err
		}
		if err := clogParser.checkUnreleasedIncrement(); err != nil {
			return nil, err
		}
		if clogParser.checkReleased() {
			clogParser.resetLoop()
			continue
		}

		if clogParser.currentChange != nil {
			clogParser.processChange()
		}

		clogParser.resetLoop()
	}

	return clogParser, nil
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

func writeToChangelogFile(file string, unreleased *change, released []*change, update bool) error {
	sb := strings.Builder{}
	sb.WriteString(changelogHeader)
	if update {
		sb.WriteString(fmt.Sprintf("%s[%s] - %s\n", releasePrefix, unreleased.Version.String(), time.Now().Format("2006-01-02")))
	} else {
		sb.WriteString(*unreleased.VersionText)
	}

	if unreleased.Text == nil || *unreleased.Text == "" || *unreleased.Text == "\n" {
		return errors.New("no changes are recorded under the release")
	}

	sb.WriteString(*unreleased.Text)
	sb.WriteString("\n")

	sLogger.Debug(sb.String())

	for _, release := range released {
		sb.WriteString("\n")
		sb.WriteString(*release.VersionText)
		sb.WriteString("\n")
		sb.WriteString(*release.Text)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(file, []byte(sb.String()), 0644); err != nil {
		sLogger.Errorf("failed to write to changelog file %s", file)
		return err
	}

	return nil
}
