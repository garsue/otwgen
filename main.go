package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/garsue/otwgen/generate"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()
	if version == "dev" {
		app.Version = strings.Join([]string{version, date, commit}, "-")
	} else {
		app.Version = version
	}
	app.Commands = []cli.Command{
		{
			Name:        "generate",
			ShortName:   "g",
			Description: "generate wrapper functions",
			Action: func(ctx *cli.Context) error {
				return start(ctx.Args().First())
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func start(pattern string) error {
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedName | packages.NeedDeps | packages.NeedTypes}
	patterns := make([]string, 0, 1)
	if pattern != "" {
		patterns = append(patterns, pattern)
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		if _, err1 := fmt.Fprintf(os.Stderr, "load: %v\n", err); err1 != nil {
			return err1
		}
		return err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return errors.New("some errors found")
	}

	for file := range generate.Parse(context.Background(), pkgs) {
		name, err := Write(file)
		if err != nil {
			return err
		}
		log.Println(name)
	}
	return nil
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