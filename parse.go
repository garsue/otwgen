package otwrapper

import (
	"fmt"
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
			out := NewFile(pkg)

			var found bool
			ast.Inspect(file, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.FuncDecl:
					if !decl.Name.IsExported() {
						return true
					}
					out.Decls = append(out.Decls, NewFunc(decl, pkg.Name))
				case *ast.TypeSpec:
					out.Decls = append(out.Decls, NewType(decl, pkg.Name))
					return true
				default:
					return true
				}
				found = true

				return true
			})

			if !found {
				continue
			}

			if err := format.Node(os.Stdout, token.NewFileSet(), out); err != nil {
				panic(err)
			}
			fmt.Println("==================")
		}
	}
}

func NewType(decl *ast.TypeSpec, pkgName string) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: decl.Name,
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Type: &ast.SelectorExpr{
									X:   ast.NewIdent(pkgName),
									Sel: decl.Name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewFunc(fdecl *ast.FuncDecl, pkgName string) *ast.FuncDecl {
	// Function
	if fdecl.Recv == nil {
		return &ast.FuncDecl{
			Doc:  fdecl.Doc,
			Name: fdecl.Name,
			Type: fdecl.Type,
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					NewFuncBody(fdecl, pkgName),
				},
			},
		}
	}
	// Method
	return &ast.FuncDecl{
		Doc:  fdecl.Doc,
		Recv: fdecl.Recv,
		Name: fdecl.Name,
		Type: fdecl.Type,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				NewFuncBody(fdecl, pkgName),
			},
		},
	}
}

func NewFuncBody(fdecl *ast.FuncDecl, pkgName string) ast.Stmt {
	args := make([]ast.Expr, 0, fdecl.Type.Params.NumFields())
	for _, field := range fdecl.Type.Params.List {
		for _, name := range field.Names {
			args = append(args, name)
		}
	}
	expr := ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(pkgName),
			Sel: fdecl.Name,
		},
		Args: args,
	}
	if fdecl.Type.Results.NumFields() == 0 {
		return &ast.ExprStmt{
			X: &expr,
		}
	}
	return &ast.ReturnStmt{
		Results: []ast.Expr{
			&expr,
		},
	}
}

func NewFile(pkg *packages.Package) *ast.File {
	return &ast.File{
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
}
