/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/
package gta

import (
	"fmt"
	"go/build"

	"golang.org/x/tools/refactor/importgraph"
)

type Package struct {
	ImportPath string
	SrcRoot    string
}

// graphError is a collection of errors from attempting to build the
// dependent graph.
type graphError struct {
	Errors map[string]error
}

// Error implements the error interface for GraphError.
func (g *graphError) Error() string {
	return fmt.Sprintf("errors while generating import graph: %v", g.Errors)
}

// Packager interface defines a set of means to access golang build Package information.
type Packager interface {
	// Get a go package from directory.
	PackageFromDir(string) (*Package, error)
	// Get a go package from an empty directory.
	PackageFromEmptyDir(string) (*Package, error)
	// Get a go package from import path.
	PackageFromImport(string) (*Package, error)
	// DependentGraph returns the DependentGraph for the current
	// Golang workspace as defined by their import paths.
	DependentGraph() (*Graph, error)
}

// verify DefaultPackager implements the the Packager interface
var _ Packager = defaultPackager

// defaultPackager is the default instance of PackageContext.
var defaultPackager = &packageContext{
	ctx: &build.Default,
}

// packageContext implements the Packager interface using build contexts.
type packageContext struct {
	ctx *build.Context
}

// PackageFromDir returns a build package from a directory.
func (p *packageContext) PackageFromDir(dir string) (*Package, error) {
	pkg, err := p.ctx.ImportDir(dir, build.ImportComment)
	return packageFrom(pkg), err
}

// PackageFromEmptyDir returns a build package from a directory.
func (p *packageContext) PackageFromEmptyDir(dir string) (*Package, error) {
	pkg, err := p.ctx.ImportDir(dir, build.FindOnly)
	return packageFrom(pkg), err
}

// PackageFromImport returns a build package from an import path.
func (p *packageContext) PackageFromImport(importPath string) (*Package, error) {
	pkg, err := p.ctx.Import(importPath, ".", build.ImportComment)
	return packageFrom(pkg), err
}

// DependentGraph returns a dependent graph based on the current Go workspace.
func (p *packageContext) DependentGraph() (*Graph, error) {
	_, graph, errs := importgraph.Build(p.ctx)
	if len(errs) != 0 {
		return nil, &graphError{Errors: errs}
	}

	return &Graph{graph: graph}, nil
}

func packageFrom(pkg *build.Package) *Package {
	return &Package{
		ImportPath: pkg.ImportPath,
		SrcRoot:    pkg.SrcRoot,
	}
}
