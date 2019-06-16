package otwrapper

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func Parse(pkgs []*packages.Package) (funcs []*ast.FuncDecl, methods []*ast.FuncDecl) {
	funcs = make([]*ast.FuncDecl, 0, len(pkgs))
	methods = make([]*ast.FuncDecl, 0, len(pkgs))
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				fdecl, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}
				if !fdecl.Name.IsExported() {
					return true
				}
				if fdecl.Recv == nil {
					funcs = append(funcs, fdecl)
				} else {
					methods = append(methods, fdecl)
				}
				return true
			})
		}
	}
	return funcs, methods
}
