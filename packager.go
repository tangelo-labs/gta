package gta

import (
	"errors"
	"go/build"

	"golang.org/x/tools/refactor/importgraph"
)

type Packager interface {
	// Get a go package from directory
	PackageFromDir(string) (*build.Package, error)
	// Get a go package from import path
	PackageFromImport(string) (*build.Package, error)
	// DependentGraph returns the DependentGraph for the current
	// Golang workspace as defined by their import paths
	DependentGraph() (*Graph, error)
}

// verify DefaultPackager implements the the Packager interface
var _ Packager = DefaultPackager

var DefaultPackager = &PackageContext{
	ctx: &build.Default,
}

// PackageContext implements the Packager interface using build contexts
type PackageContext struct {
	ctx *build.Context
}

// PackageFromDir returns a build package from a directory as a string
func (p *PackageContext) PackageFromDir(dir string) (*build.Package, error) {
	return p.ctx.ImportDir(dir, build.ImportComment)
}

// PackageFromImport returns a build package from an import path as a string
func (p *PackageContext) PackageFromImport(importPath string) (*build.Package, error) {
	return p.ctx.Import(importPath, ".", build.ImportComment)
}

// DependentGraph returns a dependent graph based on the current golang workspace
func (p *PackageContext) DependentGraph() (*Graph, error) {
	_, graph, errs := importgraph.Build(p.ctx)
	if len(errs) != 0 {
		return nil, errors.New("error generating graph")
	}

	return &Graph{graph: graph}, nil
}
