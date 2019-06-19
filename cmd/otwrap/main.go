package main

import (
	"context"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"os"

	"golang.org/x/tools/go/packages"

	"github.com/garsue/otwrapper"
)

func main() {
	var pattern string
	flag.StringVar(&pattern, "pattern", "", "package pattern")
	flag.Parse()

	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName | packages.NeedDeps | packages.NeedTypes}
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

	for file := range otwrapper.Parse(context.Background(), pkgs) {
		if err := format.Node(os.Stdout, token.NewFileSet(), file); err != nil {
			panic(err)
		}
		fmt.Println("==================")
	}
}
