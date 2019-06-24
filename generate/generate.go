package generate

import (
	"context"
	"go/ast"
	"go/token"
	"runtime"
	"strconv"
	"sync"

	"golang.org/x/tools/go/packages"
)

func LoadPackages(patterns []string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedSyntax |
			packages.NeedTypes,
	}, patterns...)
}

func Generate(ctx context.Context, pkgs []*packages.Package) <-chan *ast.File {
	syntaxCh := syntaxChannel(ctx, pkgs)
	files := make(chan *ast.File)
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case syntax, ok := <-syntaxCh:
					if !ok {
						return
					}

					file, found := NewFile(syntax)
					if !found {
						continue
					}
					select {
					case <-ctx.Done():
						return
					case files <- file:
					}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(files)
	}()
	return files
}

type SyntaxTree struct {
	pkg  *packages.Package
	file *ast.File
}

func syntaxChannel(ctx context.Context, pkgs []*packages.Package) <-chan SyntaxTree {
	syntaxCh := make(chan SyntaxTree)
	go func() {
		defer close(syntaxCh)
		for _, pkg := range pkgs {
			for _, syntax := range pkg.Syntax {
				p, s := pkg, syntax
				select {
				case <-ctx.Done():
					return
				case syntaxCh <- SyntaxTree{
					pkg:  p,
					file: s,
				}:
				}
			}
		}
	}()
	return syntaxCh
}

func NewFile(syntax SyntaxTree) (*ast.File, bool) {
	var decls []ast.Decl
	var found bool
	ds, ok := buildDecls(syntax.pkg.Name, syntax.file)
	if !ok {
		return nil, false
	}
	found = true
	decls = append(decls, ds...)

	imports := []ast.Spec{
		&ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(syntax.pkg.ID),
			},
		},
		&ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote("go.opencensus.io/trace"),
			},
		},
	}

	return &ast.File{
		Name: ast.NewIdent(syntax.pkg.Name),
		Decls: append([]ast.Decl{
			&ast.GenDecl{
				Tok:   token.IMPORT,
				Specs: imports,
			},
		}, decls...),
	}, found
}

func buildDecls(pkgName string, input ast.Node) (decls []ast.Decl, found bool) {
	ast.Inspect(input, func(n ast.Node) bool {
		var wrapped []ast.Decl
		switch decl := n.(type) {
		case *ast.TypeSpec:
			if t, c, ok := newType(decl, pkgName); ok {
				wrapped = append(wrapped, t, c)
			}
		case *ast.FuncDecl:
			if f, ok := newFunc(decl, pkgName); ok {
				wrapped = append(wrapped, f)
				found = true
			}
		default:
			return true
		}
		if len(wrapped) > 0 {
			decls = append(decls, wrapped...)
		}
		return true
	})
	return decls, found
}

func newType(decl *ast.TypeSpec, pkgName string) (wrapped *ast.GenDecl, constructor *ast.FuncDecl, ok bool) {
	if !decl.Name.IsExported() {
		return nil, constructor, false
	}
	return &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: decl.Name,
					Type: &ast.StructType{
						Fields: &ast.FieldList{
							List: []*ast.Field{
								{
									Type: &ast.StarExpr{
										X: &ast.SelectorExpr{
											X:   ast.NewIdent(pkgName),
											Sel: decl.Name,
										},
									},
								},
							},
						},
					},
				},
			},
		}, &ast.FuncDecl{
			Name: ast.NewIdent("New" + decl.Name.Name),
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{
								ast.NewIdent("orig"),
							},
							Type: &ast.StarExpr{
								X: &ast.SelectorExpr{
									X:   ast.NewIdent(pkgName),
									Sel: decl.Name,
								},
							},
						},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{
							Type: &ast.StarExpr{
								X: decl.Name,
							},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.UnaryExpr{
								Op: token.AND,
								X: &ast.CompositeLit{
									Type: decl.Name,
									Elts: []ast.Expr{
										ast.NewIdent("orig"),
									},
								},
							},
						},
					},
				},
			},
		}, true
}

func newFunc(fdecl *ast.FuncDecl, pkgName string) (wrapped *ast.FuncDecl, ok bool) {
	if !fdecl.Name.IsExported() {
		return nil, false
	}

	var w *ast.FuncDecl
	var body ast.Stmt
	if fdecl.Recv != nil { // Method
		w, ok = NewMethodDecl(fdecl)
		body = NewMethodBody(fdecl)
	} else { // Function
		w, ok = NewFuncDecl(fdecl)
		body = NewFuncBody(fdecl, pkgName)
	}
	if !ok {
		return nil, false
	}

	w.Body = &ast.BlockStmt{
		List: append(spanStmts(), body),
	}
	return w, true
}

func spanStmts() []ast.Stmt {
	return []ast.Stmt{
		&ast.AssignStmt{
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
		},
		&ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("span"),
					Sel: ast.NewIdent("End"),
				},
			},
		},
	}
}
