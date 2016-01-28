// Command gta uses git to find the subset of code changes from origin/master
// and then builds the list of go packages that have changed as a result,
// including all dependent go packages.
package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"strings"
	"syscall"

	"github.com/digitalocean/gta"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	include := flag.String("include", "doge/,services/,tools/,exp/", "include a set of comma separated prefixes on the output")
	flag.Parse()

	gt, err := gta.New()
	if err != nil {
		log.Fatal(err)
	}
	pkgs, err := gt.DirtyPackages()
	if err != nil {
		log.Fatal(err)
	}

	strung := stringify(pkgs, strings.Split(*include, ","))

	if terminal.IsTerminal(syscall.Stdin) {
		for _, pkg := range strung {
			fmt.Println(pkg)
		}
		return
	}

	fmt.Println(strings.Join(strung, " "))
}

func stringify(pkgs []*build.Package, included []string) []string {
	var out []string
	for _, pkg := range pkgs {
		for _, include := range included {
			if strings.HasPrefix(pkg.ImportPath, include) {
				out = append(out, pkg.ImportPath)
				break
			}
		}
	}
	return out
}
