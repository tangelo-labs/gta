// Package gta provides a set of utilites to build a set of dirty packages and their dependents
// that can be used to target code changes.
package gta

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"go/scanner"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrNoDiffer is returned when there is no differ set on the GTA.
	ErrNoDiffer = errors.New("there is no differ set")
	// ErrNoPackager is returned when there is no packager set on the GTA.
	ErrNoPackager = errors.New("there is no packager set")
)

// Packages contains various detailed information about the structure of
// packages GTA has detected.
type Packages struct {
	// Dependencies contains a map of changed packages to their dependencies
	Dependencies map[string][]*build.Package

	// Changes represents the changed files
	Changes []*build.Package

	// AllChanges represents all packages that are dirty including the initial
	// changed packages
	AllChanges []*build.Package
}

// MarshalJSON implements the json.Marshaler interface.
func (p *Packages) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Dependencies map[string][]string `json:"dependencies,omitempty"`
		Changes      []string            `json:"changes,omitempty"`
		AllChanges   []string            `json:"all_changes,omitempty"`
	}{
		Dependencies: mapify(p.Dependencies),
		Changes:      stringify(p.Changes),
		AllChanges:   stringify(p.AllChanges),
	})
}

// A GTA provides a method of building dirty packages, and their dependent
// packages.
type GTA struct {
	differ   Differ
	packager Packager
	prefixes []string
}

// New returns a new GTA with various options passed to New.
func New(opts ...Option) (*GTA, error) {
	gta := &GTA{
		differ:   &Git{},
		packager: DefaultPackager,
	}

	for _, opt := range opts {
		err := opt(gta)
		if err != nil {
			return nil, err
		}
	}

	return gta, nil
}

// ChangedPackages uses the differ and packager to build a map of changed root
// packages to their dependent packages where dependent is defined as "changed"
// as well due their dependency to the changed packages. It returns the
// dependency graph, the changes differ detected and a set of all unique
// packages (including the changes).
//
// As an example: package "foo" is imported by packages "bar" and "qux". If
// "foo" has changed, it has two dependent packages, "bar" and "qux". The
// result would be then:
//
//   Dependencies = {"foo": ["bar", "qux"]}
//   Changes      = ["foo"]
//   AllChanges   = ["foo", "bar", "qux]
//
// Note that two different changed package might have the same dependent
// package. Below you see that both "foo" and "foo2" has changed. Each have
// "bar" because "bar" imports both "foo" and "foo2", i.e:
//
//   Dependencies = {"foo": ["bar", "qux"], "foo2" : ["afa", "bar", "qux"]}
//   Changes      = ["foo", "foo2"]
//   AllChanges   = ["foo", "foo2", "afa", "bar", "qux]
func (g *GTA) ChangedPackages() (*Packages, error) {
	paths, err := g.markedPackages()
	if err != nil {
		return nil, err
	}

	cp := &Packages{
		Dependencies: map[string][]*build.Package{},
	}

	// build our packages
	allChanges := map[string]*build.Package{}
	for changed, marked := range paths {
		var packages []*build.Package

		for path := range marked {
			pkg, err := g.packager.PackageFromImport(path)
			if err != nil {
				if _, ok := err.(*build.NoGoError); ok {
					// there are no buildable go files in this directory
					// so no dirty packges
					continue
				}
				return nil, fmt.Errorf("building packages for %q: %v", path, err)
			}

			addPackage := func() {
				allChanges[pkg.ImportPath] = pkg
				if changed == pkg.ImportPath {
					cp.Changes = append(cp.Changes, pkg)
				}
				packages = append(packages, pkg)
			}

			if len(g.prefixes) != 0 {
				for _, include := range g.prefixes {
					if strings.HasPrefix(pkg.ImportPath, include) {
						addPackage()
					}
				}
			} else {
				addPackage()
			}
		}

		if len(packages) != 0 {
			sort.Sort(byPackageImportPath(packages))
			cp.Dependencies[changed] = packages
		}
	}

	for _, pkg := range allChanges {
		cp.AllChanges = append(cp.AllChanges, pkg)
	}
	sort.Sort(byPackageImportPath(cp.AllChanges))
	sort.Sort(byPackageImportPath(cp.Changes))

	return cp, nil
}

func (g *GTA) markedPackages() (map[string]map[string]bool, error) {
	if g.differ == nil {
		return nil, ErrNoDiffer
	}
	if g.packager == nil {
		return nil, ErrNoPackager
	}

	// get our diff'd directories
	dirs, err := g.differ.Diff()
	if err != nil {
		return nil, fmt.Errorf("diffing directory for dirty packages, %v", err)
	}

	// we build our set of initial dirty packages from the git diff
	changed := make(map[string]bool)
	for dir := range dirs {
		// Avoid .foo, _foo, and testdata directory trees how the go tool does!
		// See https://github.com/golang/tools/blob/3a85b8d/go/buildutil/allpackages.go#L93
		// Above link is not guranteed to work.
		base := filepath.Base(dir)
		parent := filepath.Base(filepath.Dir(dir))
		if base == "" || base[0] == '.' || base[0] == '_' || base == "testdata" || parent == "testdata" {
			continue
		}

		pkg, err := g.packager.PackageFromDir(dir)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				// there are no buildable go files in this directory
				// so no dirty packages
				continue
			}
			if _, ok := err.(scanner.ErrorList); ok {
				// same, package is not buildable, so no dirty packages
				continue
			}

			return nil, fmt.Errorf("pulling package information for %q, %v", dir, err)
		}

		// we create a simple set of changed pkgs by import path
		changed[pkg.ImportPath] = false
	}

	// we build the dependent graph
	graph, err := g.packager.DependentGraph()
	if err != nil {
		return nil, fmt.Errorf("building dependency graph, %v", err)
	}

	paths := map[string]map[string]bool{}
	for change := range changed {
		marked := make(map[string]bool)

		// we traverse the graph and build our list of mark all dependents
		graph.Traverse(change, marked)
		paths[change] = marked
	}

	return paths, nil
}

type byPackageImportPath []*build.Package

func (b byPackageImportPath) Len() int               { return len(b) }
func (b byPackageImportPath) Less(i int, j int) bool { return b[i].ImportPath < b[j].ImportPath }
func (b byPackageImportPath) Swap(i int, j int)      { b[i], b[j] = b[j], b[i] }

func stringify(pkgs []*build.Package) []string {
	var out []string
	for _, pkg := range pkgs {
		out = append(out, pkg.ImportPath)
	}
	return out
}

func mapify(pkgs map[string][]*build.Package) map[string][]string {
	out := map[string][]string{}
	for key, pkgs := range pkgs {
		out[key] = stringify(pkgs)
	}
	return out
}
