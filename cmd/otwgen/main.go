package main

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"

	"golang.org/x/tools/go/packages"

	"github.com/garsue/otwgen"
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

	for file := range otwgen.Parse(context.Background(), pkgs) {
		name, err := Write(file)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(name)
	}
}

func Write(file *ast.File) (name string, err error) {
	out, err := os.Create(file.Name.Name + ".go")
	if err != nil {
		return "", err
	}

	defer func() {
		if err1 := out.Close(); err1 != nil && err == nil {
			err = err1
		}
	}()
	return out.Name(), format.Node(out, token.NewFileSet(), file)
}
