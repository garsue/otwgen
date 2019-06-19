package otwrapper

import (
	"context"
	"go/ast"
	"go/token"
	"runtime"
	"strconv"
	"sync"

	"golang.org/x/tools/go/packages"
)

func Parse(ctx context.Context, pkgs []*packages.Package) <-chan *ast.File {
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

					file := NewFile(pkg)
					for _, input := range pkg.Syntax {
						decls, found := Generate(pkg.Name, input)
						if !found {
							continue
						}
						file.Decls = append(file.Decls, decls...)
					}
					if len(file.Decls) > 0 {
						select {
						case <-ctx.Done():
							return
						case files <- file:
						}
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

func canTrace(decl *ast.FuncDecl) bool {
	if !decl.Name.IsExported() {
		return false
	}
	if decl.Recv != nil {
		for _, field := range decl.Recv.List {
			// Ignore not exported receiver's method
			switch t := field.Type.(type) {
			case *ast.StarExpr:
				i, ok := t.X.(*ast.Ident)
				if !ok {
					continue
				}
				if !i.IsExported() {
					return false
				}
			case *ast.Ident:
				if !t.IsExported() {
					return false
				}
			}
		}
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
				NewMethodBody(&recv, fdecl),
			},
		},
	}
}

func NewMethodBody(recv *ast.FieldList, fdecl *ast.FuncDecl) ast.Stmt {
	args := make([]ast.Expr, 0, fdecl.Type.Params.NumFields())
	for _, field := range fdecl.Type.Params.List {
		for _, name := range field.Names {
			args = append(args, name)
		}
	}

	var recvTypeIdent *ast.Ident
	for _, field := range recv.List {
		switch t := field.Type.(type) {
		case *ast.StarExpr:
			i, ok := t.X.(*ast.Ident)
			if !ok {
				continue
			}
			recvTypeIdent = i
		case *ast.Ident:
			recvTypeIdent = t
		}
	}

	expr := ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent("r"),
				Sel: recvTypeIdent,
			},
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

func Generate(pkgName string, input *ast.File) (decls []ast.Decl, found bool) {
	ast.Inspect(input, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			if canTrace(decl) {
				decls = append(decls, NewFunc(decl, pkgName))
				found = true
			}
			return true
		case *ast.TypeSpec:
			if decl.Name.IsExported() {
				decls = append(decls, NewType(decl, pkgName))
			}
			return true
		default:
			return true
		}
	})
	return decls, found
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
