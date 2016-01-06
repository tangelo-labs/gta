package main

import (
	"fmt"
	"log"

	"github.com/digitalocean/gta"
)

func main() {
	gt, err := gta.New(
		gta.SetDiffer(&gta.Git{}),
		gta.SetPackager(gta.DefaultPackager),
	)
	if err != nil {
		log.Panic(err)
	}
	pkgs, err := gt.DirtyPackages()
	if err != nil {
		log.Panic(err)
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg.ImportPath)
	}
}
