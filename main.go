package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	flag.Parse()
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "load: %v\n", err); err != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	// Print the names of the source files
	// for each package listed on the command line.
	for _, pkg := range pkgs {
		fmt.Println(pkg.ID, pkg.GoFiles)
	}
}
