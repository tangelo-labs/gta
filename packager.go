/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/
package gta

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Package struct {
	ImportPath string
	Dir        string
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
	// Get a go package from directory. Should return a *build.NoGoError value
	// when there are no Go files in the directory.
	PackageFromDir(string) (*Package, error)
	// Get a go package from an empty directory.
	PackageFromEmptyDir(string) (*Package, error)
	// Get a go package from import path. Should return a *build.NoGoError value
	// when there are no Go files in the directory.
	PackageFromImport(string) (*Package, error)
	// DependentGraph returns the DependentGraph for the current
	// Golang workspace as defined by their import paths.
	DependentGraph() (*Graph, error)
}

func NewPackager(prefixes, tags []string) Packager {
	importPathsByDir, g, err := dependencyGraph(prefixes, tags)
	return &packageContext{
		ctx:              &build.Default,
		err:              err,
		packages:         make(map[string]struct{}),
		reverse:          g,
		importPathsByDir: importPathsByDir,
	}
}

// packageContext implements the Packager interface.
type packageContext struct {
	ctx *build.Context
	err error
	// packages is a set of import paths of packages that have been imported.
	packages map[string]struct{}
	// reverse is a reverse dependency graph (import path -> (dependent import path -> struct{}{}))
	reverse map[string]map[string]struct{}
	// importPathsByDir is a map of directories to import paths. absolute path directory -> import path
	importPathsByDir map[string]string
}

// PackageFromDir returns a build package from a directory.
func (p *packageContext) PackageFromDir(dir string) (*Package, error) {
	// try importing using ImportDir first so that the expected kinds of errors
	// (e.g. build.NoGoError) will be returned.
	pkg, err := p.ctx.ImportDir(dir, build.ImportComment)
	pkg2 := packageFrom(pkg)
	resolveLocal(pkg2, dir, p.importPathsByDir)
	p.packages[pkg2.ImportPath] = struct{}{}
	return pkg2, err
}

// PackageFromEmptyDir returns a build package from a directory.
func (p *packageContext) PackageFromEmptyDir(dir string) (*Package, error) {
	// TODO(bc): construct the Package from the information about the module or GOPATH
	pkg, err := p.ctx.ImportDir(dir, build.FindOnly)
	pkg2 := packageFrom(pkg)
	resolveLocal(pkg2, dir, p.importPathsByDir)
	p.packages[pkg2.ImportPath] = struct{}{}
	return pkg2, err
}

// PackageFromImport returns a build package from an import path.
func (p *packageContext) PackageFromImport(importPath string) (*Package, error) {
	pkg, err := p.ctx.Import(importPath, ".", build.ImportComment)
	pkg2 := packageFrom(pkg)
	p.packages[pkg2.ImportPath] = struct{}{}
	return pkg2, err
}

// DependentGraph returns a dependent graph based on the current imported packages.
func (p *packageContext) DependentGraph() (*Graph, error) {
	if p.err != nil {
		return nil, p.err
	}

	graph := make(map[string]map[string]bool)
	for k := range p.reverse {
		inner := make(map[string]bool)
		for k2 := range p.reverse[k] {
			inner[k2] = true
		}
		graph[k] = inner
	}

	return &Graph{graph: graph}, nil
}

func packageFrom(pkg *build.Package) *Package {
	return &Package{
		ImportPath: pkg.ImportPath,
		Dir:        pkg.SrcRoot,
	}
}

// resolveLocal resolves pkg.ImportPath and pkg.SrcRoot for dir against
// importPathsByDir when pkg.ImportPath is a relative path.
func resolveLocal(pkg *Package, dir string, importPathsByDir map[string]string) {
	if pkg.ImportPath != "." {
		return
	}

	importPath := pkg.ImportPath

	for k, v := range importPathsByDir {
		// check for an exact match
		if dir == k {
			importPath = v
			break
		}

		if strings.HasPrefix(dir, k) {
			vendorPathSegment := "/vendor/"
			candidateImportPath := strings.ReplaceAll(strings.TrimPrefix(dir, k), string(filepath.Separator), "/")
			if strings.HasPrefix(candidateImportPath, vendorPathSegment) {
				candidateImportPath = strings.TrimPrefix(candidateImportPath, vendorPathSegment)
			} else {
				candidateImportPath = path.Join(v, candidateImportPath)
			}

			if len(candidateImportPath) > len(importPath) {
				importPath = candidateImportPath
			}
		}
	}

	pkg.ImportPath = importPath
}

// dependencyGraph constructs a map of directories to import paths when in
// module aware mode and flattened reverse transitive dependency graph. When in
// GOPATH mode the map of directories to import paths will be empty.
func dependencyGraph(includePkgs, tags []string) (map[string]string, map[string]map[string]struct{}, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		BuildFlags: []string{
			fmt.Sprintf(`-tags=%s`, strings.Join(tags, ",")),
		},
		Tests: true,
	}

	pkgs := make([]string, 0, len(includePkgs))
	for _, pkg := range includePkgs {
		pkgs = append(pkgs, fmt.Sprintf("%s...", pkg))
	}

	loadedPackages, err := packages.Load(cfg, pkgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("loading packages: %w", err)
	}

	reverse := make(map[string]map[string]struct{})
	importPathsByDir := make(map[string]string)

	seen := make(map[string]struct{})
	var addPackage func(pkg *packages.Package)
	addPackage = func(pkg *packages.Package) {
		if _, ok := seen[pkg.ID]; ok {
			return
		}

		if pkg.Module != nil && pkg.Module.Main {
			importPathsByDir[pkg.Module.Dir] = pkg.Module.Path
		}

		// normalize the import path so that test packages will be flattened into
		// the package path of the primary package.
		pkgPath := normalizeImportPath(pkg.PkgPath)

		seen[pkg.ID] = struct{}{}

		// Ignore packages that do not have any Go files that satisfy the build
		// constraints.
		if len(pkg.GoFiles) == 0 {
			return
		}

		for _, importedPkg := range pkg.Imports {
			addPackage(importedPkg)

			importedPath := normalizeImportPath(importedPkg.PkgPath)

			// do not attempt to add the normalized import path to the dependent
			// graph when the normalized import path is the same as the package whose
			// dependents are being calculated.
			if importedPath == pkgPath {
				continue
			}

			if _, ok := reverse[importedPath]; !ok {
				reverse[importedPath] = make(map[string]struct{})
			}
			m := reverse[importedPath]
			m[pkgPath] = struct{}{}
		}

		return
	}

	for _, pkg := range loadedPackages {
		addPackage(pkg)
	}

	return importPathsByDir, reverse, nil
}

func normalizeImportPath(pkg string) string {
	switch {
	case strings.HasSuffix(pkg, "_test"):
		pkg = strings.TrimSuffix(pkg, "_test")
	case strings.HasSuffix(pkg, ".test"):
		pkg = strings.TrimSuffix(pkg, ".test")
	case strings.HasSuffix(pkg, ".test]"):
		pkg = strings.Fields(pkg)[0]
	}
	return pkg
}
