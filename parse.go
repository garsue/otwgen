package otwrapper

import (
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strconv"

	"golang.org/x/tools/go/packages"
)

func Parse(pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			out := &ast.File{
				Name: ast.NewIdent(pkg.Name),
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: strconv.Quote(pkg.ID),
								},
							},
						},
					},
				},
			}

			ast.Inspect(file, func(n ast.Node) bool {
				fdecl, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}
				if !fdecl.Name.IsExported() {
					return true
				}

				// Function
				if fdecl.Recv == nil {
					args := make([]ast.Expr, 0, fdecl.Type.Params.NumFields())
					for _, field := range fdecl.Type.Params.List {
						for _, name := range field.Names {
							args = append(args, name)
						}
					}

					var stmt ast.Stmt
					if fdecl.Type.Results.NumFields() == 0 {
						stmt = &ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent(pkg.Name),
									Sel: fdecl.Name,
								},
								Args: args,
							},
						}
					} else {
						stmt = &ast.ReturnStmt{
							Results: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   ast.NewIdent(pkg.Name),
										Sel: fdecl.Name,
									},
									Args: args,
								},
							},
						}
					}

					out.Decls = append(out.Decls, &ast.FuncDecl{
						Doc:  fdecl.Doc,
						Name: fdecl.Name,
						Type: fdecl.Type,
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								stmt,
							},
						},
					})
				}

				return true
			})

			if err := format.Node(os.Stdout, token.NewFileSet(), out); err != nil {
				panic(err)
			}
		}
	}
}
