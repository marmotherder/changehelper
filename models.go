package main

import (
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
	Version *semver.Version
	Node    ast.Node
}

func (c change) text(source []byte) string {
	return string(c.Node.Text(source))
}
