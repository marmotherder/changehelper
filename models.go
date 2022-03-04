package main

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/yuin/goldmark/ast"
)

const (
	noChangesError = 2
	ioError        = 3
	execError      = 4
	parseError     = 5
)

type change struct {
	Version     *semver.Version
	VersionText *string
	Text        *string
	Added       []string
	Changed     []string
	Deprecated  []string
	Removed     []string
	Fixed       []string
	Security    []string
}

const (
	changeAdded      changeType = "Added"
	changeChanged    changeType = "Changed"
	changeDeprecated changeType = "Deprecated"
	changeRemoved    changeType = "Removed"
	changeFixed      changeType = "Fixed"
	changeSecurity   changeType = "Security"
)

type changeType string

func (c *change) renderChangeText(increment ...string) {
	sb := strings.Builder{}
	versionText := "## [Unreleased]"
	if len(increment) > 0 {
		versionText = fmt.Sprintf("%s - %s", versionText, increment[0])
	}
	versionText = versionText + "\n"
	c.VersionText = &versionText
	sb.WriteString(versionText)
	appendSection := func(text changeType, section []string) {
		if len(section) > 0 {
			sb.WriteString(fmt.Sprintf("### %s\n", text))
			sb.WriteString(fmt.Sprintf("%s\n", strings.Join(section, "\n")))
		}
	}

	appendSection(changeAdded, c.Added)
	appendSection(changeChanged, c.Changed)
	appendSection(changeDeprecated, c.Deprecated)
	appendSection(changeRemoved, c.Removed)
	appendSection(changeFixed, c.Fixed)
	appendSection(changeSecurity, c.Security)

	text := sb.String()
	c.Text = &text

	sLogger.Debug("rendered new release text as:")
	sLogger.Debug(text)
}

func nodeText(node ast.Node, source []byte) *string {
	text := string(node.Text(source))
	return &text
}
