package main

import (
	"fmt"
	"go/build"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/refactor/importgraph"
)

func main() {
	// find our root directory
	cmd := []string{"rev-parse", "--show-toplevel"}
	out, err := exec.Command("git", cmd...).Output()
	if err != nil {
		log.Fatal(err)
	}
	root := strings.TrimSpace(string(out))

	// git diff all files from _master_
	cmd = []string{"diff", "origin/master", "--name-only"}
	out, err = exec.Command("git", cmd...).Output()
	if err != nil {
		log.Fatal(err)
	}

	// we build our dirty directories from out git diff
	dirtyDirs := make(map[string]bool)
	dirtyFiles := strings.Split(string(out), "\n")
	for _, dirtyFile := range dirtyFiles {
		if dirtyFile == "" {
			continue
		}
		dirtyDirs[filepath.Dir(filepath.Join(root, dirtyFile))] = false
	}

	// we build our set of initial dirty packages from the git diff
	ctx := &build.Default
	dirtyPkgs := make(map[string]bool)
	for dirtyDir := range dirtyDirs {
		pkg, err := ctx.ImportDir(dirtyDir, build.ImportComment)
		if err != nil {
			log.Fatalf("import dir failed: %v", err)
		}
		dirtyPkgs[pkg.ImportPath] = false
	}

	// we build all the dependent packages
	_, pkgs, _ := importgraph.Build(ctx)
	for pkg := range pkgs {
		if _, ok := dirtyPkgs[pkg]; ok {
			// we walk the graph and build our list of mark all dependents
			walk(pkg, &dirtyPkgs, pkgs)
		}
	}

	for pkg := range dirtyPkgs {
		fmt.Println(pkg)
	}
}

func walk(pkg string, ref *map[string]bool, graph map[string]map[string]bool) {
	pkgs := *ref
	// we've already visited this node
	if visited, ok := pkgs[pkg]; visited && ok {
		return
	}

	// we mark the node as visited
	pkgs[pkg] = true

	// we check its dependents
	if dependents, ok := graph[pkg]; ok {
		for dependent := range dependents {
			walk(dependent, ref, graph)
		}
	} else {
		// this package has no other pkgs dependent on it
		// so we can just return
	}

	return
}
