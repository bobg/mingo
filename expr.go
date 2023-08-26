package mingo

import (
	"fmt"
	"go/ast"
	"go/types"
)

func (s scanner) expr(expr ast.Expr) (int, error) {
	if expr == nil {
		return 0, nil
	}

	switch expr := expr.(type) {
	case *ast.Ident:
		return s.ident(expr)
	case *ast.Ellipsis:
		return MinGoMinorVersion, nil
	case *ast.BasicLit:
		return s.basicLit(expr)
	case *ast.FuncLit:
		return s.funcLit(expr)
	case *ast.CompositeLit:
		return s.compositeLit(expr)
	case *ast.ParenExpr:
		return s.parenExpr(expr)
	case *ast.SelectorExpr:
		return s.selectorExpr(expr)
	case *ast.IndexExpr:
		return s.indexExpr(expr)
	case *ast.IndexListExpr:
		return s.indexListExpr(expr)
	case *ast.SliceExpr:
		return s.sliceExpr(expr)
	case *ast.TypeAssertExpr:
		return s.typeAssertExpr(expr)
	case *ast.CallExpr:
		return s.callExpr(expr)
	case *ast.StarExpr:
		return s.starExpr(expr)
	case *ast.UnaryExpr:
		return s.unaryExpr(expr)
	case *ast.BinaryExpr:
		return s.binaryExpr(expr)
	case *ast.KeyValueExpr:
		return s.keyValueExpr(expr)
	case *ast.ArrayType:
		return s.arrayType(expr)
	case *ast.StructType:
		return s.structType(expr)
	case *ast.FuncType:
		return s.funcType(expr)
	case *ast.InterfaceType:
		return s.interfaceType(expr)
	case *ast.MapType:
		return s.mapType(expr)
	case *ast.ChanType:
		return s.chanType(expr)
	default:
		return 0, fmt.Errorf("unknown expr type %T", expr)
	}
}

func (s scanner) ident(ident *ast.Ident) (int, error) {
	if tv, ok := s.pkg.TypesInfo.Types[ident]; ok && tv.IsBuiltin() && ident.Name == "any" {
		return 18, nil
	}
	obj, ok := s.pkg.TypesInfo.Uses[ident]
	if !ok || obj == nil {
		return 0, nil
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return 0, nil
	}
	pkgpath := obj.Pkg().Path()
	return s.lookup(pkgpath, ident.Name, ""), nil
}

func (s scanner) basicLit(lit *ast.BasicLit) (int, error) {
	return MinGoMinorVersion, nil
}

func (s scanner) funcLit(lit *ast.FuncLit) (int, error) {
	return MinGoMinorVersion, nil
}

func (s scanner) compositeLit(lit *ast.CompositeLit) (int, error) {
	return MinGoMinorVersion, nil
}

func (s scanner) parenExpr(expr *ast.ParenExpr) (int, error) {
	return s.expr(expr.X)
}

func (s scanner) selectorExpr(expr *ast.SelectorExpr) (int, error) {
	if sel, ok := s.pkg.TypesInfo.Selections[expr]; ok {
		pkgpath := sel.Obj().Pkg().Path()
		if v := s.lookup(pkgpath, expr.Sel.Name, types.TypeString(sel.Type(), nil)); v > 0 {
			return v, nil
		}
	}

	return s.expr(expr.X)
}

func (s scanner) indexExpr(expr *ast.IndexExpr) (int, error) {
	result1, err := s.expr(expr.X)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Index)
	return max(result1, result2), err
}

func (s scanner) indexListExpr(expr *ast.IndexListExpr) (int, error) {
	result, err := s.expr(expr.X)
	if err != nil {
		return 0, err
	}
	for _, index := range expr.Indices {
		indexResult, err := s.expr(index)
		if err != nil {
			return 0, err
		}
		result = max(result, indexResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) sliceExpr(expr *ast.SliceExpr) (int, error) {
	result1, err := s.expr(expr.X)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Low)
	if err != nil {
		return 0, err
	}
	result3, err := s.expr(expr.High)
	if err != nil {
		return 0, err
	}
	result4, err := s.expr(expr.Max)
	return max(result1, result2, result3, result4), err
}

func (s scanner) typeAssertExpr(expr *ast.TypeAssertExpr) (int, error) {
	result1, err := s.expr(expr.X)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Type)
	return max(result1, result2), err
}

func (s scanner) callExpr(expr *ast.CallExpr) (int, error) {
	tv, ok := s.pkg.TypesInfo.Types[expr.Fun]
	if !ok {
		return 0, fmt.Errorf("no type info for call expression")
	}

	switch {
	case tv.IsType() && len(expr.Args) == 1:
		return s.typeConversion(expr, tv)
	case tv.IsBuiltin():
		return s.builtinCall(expr)
	}

	result, err := s.expr(expr.Fun)
	if err != nil {
		return 0, err
	}
	for _, arg := range expr.Args {
		argResult, err := s.expr(arg)
		if err != nil {
			return 0, err
		}
		result = max(result, argResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

// expr.Fun is a type expression, and len(expr.Args) == 1.
func (s scanner) typeConversion(expr *ast.CallExpr, funtv types.TypeAndValue) (int, error) {
	argtv, ok := s.pkg.TypesInfo.Types[expr.Args[0]]
	if !ok {
		return 0, fmt.Errorf("no type info for type conversion argument")
	}

	// Is this a conversion from slice to array or array pointer?
	var (
		funtyp = funtv.Type.Underlying()
		argtyp = argtv.Type.Underlying()
	)
	if _, ok := argtyp.(*types.Slice); ok {
		if _, ok := funtyp.(*types.Array); ok {
			return 20, nil
		}
		if ptr, ok := funtyp.(*types.Pointer); ok {
			elemtype := ptr.Elem().Underlying()
			if _, ok := elemtype.(*types.Array); ok {
				return 17, nil
			}
		}
	}

	return MinGoMinorVersion, nil
}

func (s scanner) builtinCall(expr *ast.CallExpr) (int, error) {
	id := getID(expr.Fun)
	if id == nil {
		return 0, fmt.Errorf("builtin call expression has no identifier")
	}
	switch id.Name {
	case "min", "max", "clear":
		return 21, nil
	default:
		return MinGoMinorVersion, nil
	}
}

func getID(expr ast.Expr) *ast.Ident {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr
	case *ast.ParenExpr:
		return getID(expr.X)
	default:
		return nil
	}
}

func (s scanner) starExpr(expr *ast.StarExpr) (int, error) {
	return s.expr(expr.X)
}

func (s scanner) unaryExpr(expr *ast.UnaryExpr) (int, error) {
	return s.expr(expr.X)
}

func (s scanner) binaryExpr(expr *ast.BinaryExpr) (int, error) {
	result1, err := s.expr(expr.X)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Y)
	return max(result1, result2), err
}

func (s scanner) keyValueExpr(expr *ast.KeyValueExpr) (int, error) {
	result1, err := s.expr(expr.Key)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Value)
	return max(result1, result2), err
}

func (s scanner) arrayType(expr *ast.ArrayType) (int, error) {
	result1, err := s.expr(expr.Len)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Elt)
	return max(result1, result2), err
}

func (s scanner) structType(expr *ast.StructType) (int, error) {
	return s.fieldList(expr.Fields)
}

func (s scanner) funcType(expr *ast.FuncType) (int, error) {
	result := MinGoMinorVersion
	if expr.TypeParams != nil && len(expr.TypeParams.List) > 0 {
		result = 18
		typeParamsResult, err := s.fieldList(expr.TypeParams)
		if err != nil {
			return 0, err
		}
		result = max(result, typeParamsResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}

	exprResult, err := s.fieldList(expr.Params)
	if err != nil {
		return 0, err
	}
	result = max(result, exprResult)
	if result == MaxGoMinorVersion {
		return result, nil
	}
	resultsResult, err := s.fieldList(expr.Results)
	return max(result, resultsResult), err
}

func (s scanner) interfaceType(expr *ast.InterfaceType) (int, error) {
	return s.fieldList(expr.Methods)
}

func (s scanner) mapType(expr *ast.MapType) (int, error) {
	result1, err := s.expr(expr.Key)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(expr.Value)
	return max(result1, result2), err
}

func (s scanner) chanType(expr *ast.ChanType) (int, error) {
	return s.expr(expr.Value)
}
