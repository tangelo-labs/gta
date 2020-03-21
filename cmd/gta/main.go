/*
Copyright 2016 The gta AUTHORS. All rights reserved.

Use of this source code is governed by the Apache 2 license that can be found
in the LICENSE file.
*/

// Command gta uses git to find the subset of code changes from origin/master
// and then builds the list of go packages that have changed as a result,
// including all dependent go packages.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/digitalocean/gta"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/tools/go/buildutil"
)

// We define this so the tooling works with build tags
func init() {
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	include := flag.String("include",
		"do/doge/,do/services/,do/teams/,do/tools/,do/exp/",
		"define changes to be filtered with a set of comma separated prefixes")
	merge := flag.Bool("merge", false, "diff using the latest merge commit")
	flagJSON := flag.Bool("json", false, "output list of changes as json")
	flagBuildableOnly := flag.Bool("buildable-only", true, "keep buildable changed packages only")
	flag.Parse()

	if *flagJSON && *flagBuildableOnly {
		log.Fatal("-buildable-only must be set to false when using -json")
	}

	options := []gta.Option{
		gta.SetDiffer(gta.NewGitDiffer(*merge)),
		gta.SetPrefixes(strings.Split(*include, ",")...),
	}

	gt, err := gta.New(options...)
	if err != nil {
		log.Fatalf("can't prepare gta: %v", err)
	}

	packages, err := gt.ChangedPackages()
	if err != nil {
		log.Fatalf("can't list dirty packages: %v", err)
	}

	if *flagJSON {
		err = json.NewEncoder(os.Stdout).Encode(packages)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	strung := stringify(packages.AllChanges, *flagBuildableOnly)

	if terminal.IsTerminal(syscall.Stdin) {
		for _, pkg := range strung {
			fmt.Println(pkg)
		}
		return
	}

	fmt.Println(strings.Join(strung, " "))
}

func stringify(pkgs []*build.Package, validOnly bool) []string {
	var out []string
	for _, pkg := range pkgs {
		if !validOnly || (validOnly && pkg.SrcRoot != "") {
			out = append(out, pkg.ImportPath)
		}
	}
	return out
}
