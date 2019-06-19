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
					if !canTrace(decl) {
						return true
					}
					out.Decls = append(out.Decls, NewFunc(decl, pkg.Name))
				case *ast.TypeSpec:
					if !decl.Name.IsExported() {
						return true
					}
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

func canTrace(decl *ast.FuncDecl) bool {
	if !decl.Name.IsExported() {
		return false
	}
	for _, field := range decl.Type.Params.List {
		t, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		x, ok := t.X.(*ast.Ident)
		if !ok {
			continue
		}
		// Found context.Context in arguments
		if x.Name == "context" && t.Sel.Name == "Context" {
			return true
		}
	}
	return false
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
	startSpan := ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent("ctx"),
			ast.NewIdent("span"),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("trace"),
					Sel: ast.NewIdent("StartSpan"),
				},
				Args: []ast.Expr{
					ast.NewIdent("ctx"),
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote("auto generated span"),
					},
				},
			},
		},
	}
	end := ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("span"),
				Sel: ast.NewIdent("End"),
			},
		},
	}

	// Function
	if fdecl.Recv == nil {
		return &ast.FuncDecl{
			Doc:  fdecl.Doc,
			Name: fdecl.Name,
			Type: fdecl.Type,
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&startSpan,
					&end,
					NewFuncBody(fdecl, pkgName),
				},
			},
		}
	}
	// Method
	recv := *fdecl.Recv
	for i, field := range fdecl.Recv.List {
		if len(field.Names) == 0 {
			fdecl.Recv.List[i].Names = append(field.Names, ast.NewIdent("r"))
			continue
		}
		for j := range field.Names {
			fdecl.Recv.List[i].Names[j] = ast.NewIdent("r")
		}
	}
	return &ast.FuncDecl{
		Doc:  fdecl.Doc,
		Recv: &recv,
		Name: fdecl.Name,
		Type: fdecl.Type,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&startSpan,
				&end,
				NewMethodBody(fdecl),
			},
		},
	}
}

func NewMethodBody(fdecl *ast.FuncDecl) ast.Stmt {
	args := make([]ast.Expr, 0, fdecl.Type.Params.NumFields())
	for _, field := range fdecl.Type.Params.List {
		for _, name := range field.Names {
			args = append(args, name)
		}
	}
	expr := ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("r"),
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
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: strconv.Quote("go.opencensus.io/trace"),
						},
					},
				},
			},
		},
	}
}
