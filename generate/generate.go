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
		Mode: packages.NeedSyntax |
			packages.NeedName |
			packages.NeedDeps |
			packages.NeedTypes,
	}, patterns...)
}

func Generate(ctx context.Context, pkgs []*packages.Package) <-chan *ast.File {
	pkgCh := pkgChannel(ctx, pkgs)
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
				case pkg, ok := <-pkgCh:
					if !ok {
						return
					}

					file, found := NewFile(pkg)
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

func pkgChannel(ctx context.Context, pkgs []*packages.Package) <-chan *packages.Package {
	pkgCh := make(chan *packages.Package)
	go func() {
		defer close(pkgCh)
		for _, pkg := range pkgs {
			p := pkg
			select {
			case <-ctx.Done():
				return
			case pkgCh <- p:
			}
		}
	}()
	return pkgCh
}

func NewFile(pkg *packages.Package) (*ast.File, bool) {
	file := ast.File{
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
	var found bool
	for _, input := range pkg.Syntax {
		var decls []ast.Decl
		decls, ok := buildDecls(pkg.Name, input)
		if !ok {
			continue
		}
		found = true
		file.Decls = append(file.Decls, decls...)
	}
	return &file, found
}

func buildDecls(pkgName string, input ast.Node) (decls []ast.Decl, found bool) {
	ast.Inspect(input, func(n ast.Node) bool {
		var wrapped ast.Decl
		switch decl := n.(type) {
		case *ast.TypeSpec:
			if decl.Name.IsExported() {
				wrapped = newType(decl, pkgName)
			}
		case *ast.FuncDecl:
			if f, ok := newFunc(decl, pkgName); ok {
				wrapped = f
				found = true
			}
		default:
			return true
		}
		if wrapped != nil {
			decls = append(decls, wrapped)
		}
		return true
	})
	return decls, found
}

func newType(decl *ast.TypeSpec, pkgName string) *ast.GenDecl {
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

func newFunc(fdecl *ast.FuncDecl, pkgName string) (wrapped *ast.FuncDecl, ok bool) {
	if !fdecl.Name.IsExported() {
		return nil, false
	}

	var body ast.Stmt
	if fdecl.Recv != nil { // Method
		wrapped, ok = NewMethodDecl(fdecl)
		body = NewMethodBody(fdecl)
	} else { // Function
		wrapped, ok = NewFuncDecl(fdecl)
		body = NewFuncBody(fdecl, pkgName)
	}

	if !ok {
		return nil, false
	}
	wrapped.Body = &ast.BlockStmt{
		List: append(spanStmts(), body),
	}
	return wrapped, true
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
