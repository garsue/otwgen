package generate

import (
	"context"
	"go/ast"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

func LoadPackages(patterns []string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes,
	}, patterns...)
}

type Wrapper struct {
	Name string
	File *ast.File
}

func Generate(ctx context.Context, pkgs []*packages.Package) <-chan Wrapper {
	syntaxCh := syntaxChannel(ctx, pkgs)
	files := make(chan Wrapper)
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
					case files <- Wrapper{
						Name: syntax.name,
						File: file,
					}:
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
	name string
	pkg  *packages.Package
	file *ast.File
}

func syntaxChannel(ctx context.Context, pkgs []*packages.Package) <-chan SyntaxTree {
	syntaxCh := make(chan SyntaxTree)
	go func() {
		defer close(syntaxCh)
		for _, pkg := range pkgs {
			for i, syntax := range pkg.Syntax {
				p, s := pkg, syntax
				select {
				case <-ctx.Done():
					return
				case syntaxCh <- SyntaxTree{
					name: filepath.Base(p.GoFiles[i]),
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
	specs, ds, ok := compose(syntax.pkg, syntax.file)
	if !ok {
		return nil, false
	}
	decls = append(decls, ds...)

	specs = append(
		specs,
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
	)
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].(*ast.ImportSpec).Path.Value < specs[j].(*ast.ImportSpec).Path.Value
	})

	return &ast.File{
		Name: ast.NewIdent(syntax.pkg.Name),
		Decls: append([]ast.Decl{
			&ast.GenDecl{
				Tok:   token.IMPORT,
				Specs: specs,
			},
		}, decls...),
	}, true
}

func compose(pkg *packages.Package, input *ast.File) (specs []ast.Spec, decls []ast.Decl, found bool) {
	origImports := importNameMap(input, pkg)
	requiredSelectors := make(map[string]struct{}, len(input.Imports))
	ast.Inspect(input, func(n ast.Node) bool {
		var wrapped []ast.Decl
		switch decl := n.(type) {
		case *ast.TypeSpec:
			if t, c, ok := newType(decl, pkg.Name); ok {
				wrapped = append(wrapped, t, c)
			}
		case *ast.FuncDecl:
			if f, ok := newFunc(decl, pkg.Name); ok {
				wrapped = append(wrapped, f)
				found = true
				for _, field := range f.Type.Params.List {
					name, ok := findSelectorName(field.Type)
					if !ok {
						continue
					}
					requiredSelectors[name] = struct{}{}
				}
			}
		default:
			return true
		}
		if len(wrapped) > 0 {
			decls = append(decls, wrapped...)
		}
		return true
	})

	for name := range requiredSelectors {
		specs = append(specs, origImports[name])
	}

	return specs, decls, found
}

func importNameMap(input *ast.File, pkg *packages.Package) map[string]*ast.ImportSpec {
	origImports := make(map[string]*ast.ImportSpec, len(input.Imports))
	for _, v := range input.Imports {
		if v.Name != nil {
			origImports[v.Name.Name] = v
		} else {
			ip := pkg.Imports[strings.Trim(v.Path.Value, `"`)]
			origImports[ip.Name] = v
		}
	}
	return origImports
}

func findSelectorName(fieldType ast.Expr) (name string, found bool) {
	switch t := fieldType.(type) {
	case *ast.StarExpr:
		se, ok := t.X.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		i, ok := se.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		return i.Name, true
	case *ast.SelectorExpr:
		i, ok := t.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		return i.Name, true
	default:
		return "", false
	}
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
