package main

import (
	"flag"
	"fmt"
	"go/ast"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	var pattern string
	flag.StringVar(&pattern, "pattern", "", "package pattern")
	flag.Parse()

	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedDeps | packages.NeedTypes}
	patterns := make([]string, 0, 1)
	if pattern != "" {
		patterns = append(patterns, pattern)
	}
	pkgs, err := packages.Load(cfg, patterns...)
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
		for _, s := range pkg.Syntax {
			//log.Println(s)
			if err := ast.Print(pkg.Fset, s); err != nil {
				panic(err)
			}
		}

		//for _, p := range pkg.GoFiles {
		//	f, err := parser.ParseFile(pkg.Fset, p, nil, 0)
		//	if err != nil {
		//		log.Panic(err)
		//	}
		//	if err := ast.Print(pkg.Fset, f); err != nil {
		//		log.Panic(err)
		//	}
		//}
	}
}
