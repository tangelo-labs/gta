package main

import (
	"fmt"
	"go/build"
	"log"
	"strings"
	"syscall"

	"github.com/digitalocean/gta"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	gt, err := gta.New()
	if err != nil {
		log.Panic(err)
	}
	pkgs, err := gt.DirtyPackages()
	if err != nil {
		log.Panic(err)
	}
	strung := stringify(pkgs)

	if terminal.IsTerminal(syscall.Stdin) {
		for _, pkg := range strung {
			fmt.Println(pkg)
		}
		return
	}

	fmt.Println(strings.Join(strung, " "))
}

func stringify(pkgs []*build.Package) []string {
	out := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		out[i] = pkg.ImportPath
	}
	return out
}
