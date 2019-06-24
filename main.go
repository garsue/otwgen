package main

// noinspection GoRedundantImportAlias
import (
	"context"
	"fmt"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "dir, d",
					Usage: "output directory",
					Value: ".",
				},
			},
			Action: func(ctx *cli.Context) error {
				return start(ctx.Args(), ctx.String("dir"))
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func start(patterns []string, dir string) error {
	pkgs, err := generate.LoadPackages(patterns)
	if err != nil {
		return err
	}
	if cnt := packages.PrintErrors(pkgs); cnt > 0 {
		return fmt.Errorf("%d package errors found", cnt)
	}

	if err := SetupDir(dir); err != nil {
		return err
	}

	for result := range generate.Generate(context.Background(), pkgs) {
		name, err := Write(dir, result)
		if err != nil {
			return err
		}
		log.Println(name)
	}
	return nil
}

func SetupDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}

func Write(dir string, wrapper generate.Wrapper) (name string, err error) {
	out, err := os.Create(filepath.Join(dir, wrapper.Name))
	if err != nil {
		return "", err
	}

	defer func() {
		if err1 := out.Close(); err1 != nil && err == nil {
			err = err1
		}
	}()
	return out.Name(), format.Node(out, token.NewFileSet(), wrapper.File)
}
