package generate

import (
	"go/ast"
)

func NewMethodDecl(fdecl *ast.FuncDecl) (wrapped *ast.FuncDecl, ok bool) {
	if !isExportedRecv(fdecl) || !hasCtx(fdecl) {
		return nil, false
	}

	recv := *fdecl.Recv
	for i, field := range fdecl.Recv.List {
		if len(field.Names) == 0 {
			fdecl.Recv.List[i].Names = append(fdecl.Recv.List[i].Names, ast.NewIdent("r"))
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
	}, true
}

func isExportedRecv(decl *ast.FuncDecl) bool {
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
	return true
}

func NewMethodBody(fdecl *ast.FuncDecl) ast.Stmt {
	args := make([]ast.Expr, 0, fdecl.Type.Params.NumFields())
	for _, field := range fdecl.Type.Params.List {
		for _, name := range field.Names {
			args = append(args, name)
		}
	}

	var recvTypeIdent *ast.Ident
	for _, field := range fdecl.Recv.List {
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
