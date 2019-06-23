package generate

import (
	"go/ast"
)

func NewFuncDecl(decl *ast.FuncDecl) (wrapped *ast.FuncDecl, ok bool) {
	if !hasCtx(decl) {
		return nil, false
	}
	return &ast.FuncDecl{
		Doc:  decl.Doc,
		Name: decl.Name,
		Type: decl.Type,
	}, true
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

func hasCtx(decl *ast.FuncDecl) bool {
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
