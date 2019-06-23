package main

import (
	"context"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/garsue/otwgen/generate"
)

var (
	version = "dev"
)

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:        "generate",
			ShortName:   "g",
			Description: "generate wrapper functions",
			Action: func(ctx *cli.Context) error {
				return start(ctx.Args())
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func start(patterns []string) error {
	pkgs, err := generate.LoadPackages(patterns)
	if err != nil {
		return err
	}

	for file := range generate.Generate(context.Background(), pkgs) {
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
