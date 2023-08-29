package mingo

import (
	"fmt"
	"go/ast"
	"go/types"
)

func (p *pkgScanner) expr(expr ast.Expr) error {
	if expr == nil {
		return nil
	}

	switch expr := expr.(type) {
	case *ast.Ident:
		return p.ident(expr)
	case *ast.Ellipsis:
		return nil
	case *ast.BasicLit:
		return p.basicLit(expr)
	case *ast.FuncLit:
		return p.funcLit(expr)
	case *ast.CompositeLit:
		return p.compositeLit(expr)
	case *ast.ParenExpr:
		return p.parenExpr(expr)
	case *ast.SelectorExpr:
		return p.selectorExpr(expr)
	case *ast.IndexExpr:
		return p.indexExpr(expr)
	case *ast.IndexListExpr:
		return p.indexListExpr(expr)
	case *ast.SliceExpr:
		return p.sliceExpr(expr)
	case *ast.TypeAssertExpr:
		return p.typeAssertExpr(expr)
	case *ast.CallExpr:
		return p.callExpr(expr)
	case *ast.StarExpr:
		return p.starExpr(expr)
	case *ast.UnaryExpr:
		return p.unaryExpr(expr)
	case *ast.BinaryExpr:
		return p.binaryExpr(expr)
	case *ast.KeyValueExpr:
		return p.keyValueExpr(expr)
	case *ast.ArrayType:
		return p.arrayType(expr)
	case *ast.StructType:
		return p.structType(expr)
	case *ast.FuncType:
		return p.funcType(expr)
	case *ast.InterfaceType:
		return p.interfaceType(expr)
	case *ast.MapType:
		return p.mapType(expr)
	case *ast.ChanType:
		return p.chanType(expr)
	default:
		return fmt.Errorf("unknown expr type %T", expr)
	}
}

func (p *pkgScanner) ident(ident *ast.Ident) error {
	if tv, ok := p.info.Types[ident]; ok && tv.IsBuiltin() && ident.Name == "any" {
		idResult := posResult{
			version: 18,
			pos:     p.fset.Position(ident.Pos()),
			desc:    `"any" builtin`,
		}
		p.s.greater(idResult)
		return nil
	}
	obj, ok := p.info.Uses[ident]
	if !ok || obj == nil {
		return nil
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return nil
	}
	pkgpath := obj.Pkg().Path()
	if v := p.s.lookup(pkgpath, ident.Name, ""); v > 0 {
		idResult := posResult{
			version: v,
			pos:     p.fset.Position(ident.Pos()),
			desc:    fmt.Sprintf(`"%s".%s`, pkgpath, ident.Name),
		}
		p.s.greater(idResult)
	}
	return nil
}

// xxx check for expanded numeric-literal syntax
func (p *pkgScanner) basicLit(lit *ast.BasicLit) error {
	return nil
}

// xxx check for non-empty type params
func (p *pkgScanner) funcLit(lit *ast.FuncLit) error {
	return nil
}

func (p *pkgScanner) compositeLit(lit *ast.CompositeLit) error {
	return nil
}

func (p *pkgScanner) parenExpr(expr *ast.ParenExpr) error {
	return p.expr(expr.X)
}

func (p *pkgScanner) selectorExpr(expr *ast.SelectorExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}

	if obj, ok := p.info.Uses[expr.Sel]; ok && obj != nil {
		pkg := obj.Pkg()
		if pkg == nil {
			return nil
		}
		pkgpath := obj.Pkg().Path()
		if v := p.s.lookup(pkgpath, expr.Sel.Name, ""); v > 0 {
			selResult := posResult{
				version: v,
				pos:     p.fset.Position(expr.Pos()),
				desc:    fmt.Sprintf(`"%s".%s`, pkgpath, expr.Sel.Name),
			}
			p.s.greater(selResult)
		}
		return nil
	}

	sel, ok := p.info.Selections[expr]
	if !ok {
		return nil
	}
	obj := sel.Obj()
	if obj == nil {
		return nil
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return nil
	}
	pkgpath := obj.Pkg().Path()

	typ := sel.Recv()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	typestr := typ.String()

	v := p.s.lookup(pkgpath, expr.Sel.Name, typestr)
	if v == 0 {
		return nil
	}

	selResult := posResult{
		version: v,
		pos:     p.fset.Position(expr.Pos()),
		desc:    fmt.Sprintf(`"%s".%s.%s`, pkgpath, typestr, expr.Sel.Name),
	}
	p.s.greater(selResult)
	return nil
}

func (p *pkgScanner) indexExpr(expr *ast.IndexExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Index)
}

func (p *pkgScanner) indexListExpr(expr *ast.IndexListExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	for _, index := range expr.Indices {
		if err := p.expr(index); err != nil {
			return err
		}
		if p.s.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) sliceExpr(expr *ast.SliceExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	if err := p.expr(expr.Low); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	if err := p.expr(expr.High); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Max)
}

func (p *pkgScanner) typeAssertExpr(expr *ast.TypeAssertExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Type)
}

func (p *pkgScanner) callExpr(expr *ast.CallExpr) error {
	tv, ok := p.info.Types[expr.Fun]
	if !ok {
		return fmt.Errorf("no type info for call expression at %s", p.fset.Position(expr.Pos()))
	}

	switch {
	case tv.IsType() && len(expr.Args) == 1:
		return p.typeConversion(expr, tv)
	case tv.IsBuiltin():
		return p.builtinCall(expr)
	}

	if err := p.expr(expr.Fun); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}

	for _, arg := range expr.Args {
		if err := p.expr(arg); err != nil {
			return err
		}
		if p.s.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}

	return nil
}

// expr.Fun is a type expression, and len(expr.Args) == 1.
func (p *pkgScanner) typeConversion(expr *ast.CallExpr, funtv types.TypeAndValue) error {
	argtv, ok := p.info.Types[expr.Args[0]]
	if !ok {
		return fmt.Errorf("no type info for type conversion argument at %s", p.fset.Position(expr.Args[0].Pos()))
	}

	// Is this a conversion from slice to array or array pointer?
	var (
		funtyp = funtv.Type.Underlying()
		argtyp = argtv.Type.Underlying()
	)
	if _, ok := argtyp.(*types.Slice); ok {
		if _, ok := funtyp.(*types.Array); ok {
			convResult := posResult{
				version: 20,
				pos:     p.fset.Position(expr.Pos()),
				desc:    fmt.Sprintf("conversion from slice to array"),
			}
			p.s.greater(convResult)
			return nil
		}
		if ptr, ok := funtyp.(*types.Pointer); ok {
			elemtype := ptr.Elem().Underlying()
			if _, ok := elemtype.(*types.Array); ok {
				convResult := posResult{
					version: 17,
					pos:     p.fset.Position(expr.Pos()),
					desc:    fmt.Sprintf("conversion from slice to array pointer"),
				}
				p.s.greater(convResult)
				return nil
			}
		}
	}

	return nil
}

func (p *pkgScanner) builtinCall(expr *ast.CallExpr) error {
	id := getID(expr.Fun)
	if id == nil {
		return fmt.Errorf("builtin call expression has no identifier")
	}
	switch id.Name {
	case "min", "max", "clear":
		result := posResult{
			version: 21,
			pos:     p.fset.Position(expr.Pos()),
			desc:    fmt.Sprintf("use of %s builtin", id.Name),
		}
		p.s.greater(result)
	}
	return nil
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

func (p *pkgScanner) starExpr(expr *ast.StarExpr) error {
	return p.expr(expr.X)
}

func (p *pkgScanner) unaryExpr(expr *ast.UnaryExpr) error {
	return p.expr(expr.X)
}

func (p *pkgScanner) binaryExpr(expr *ast.BinaryExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Y)
}

func (p *pkgScanner) keyValueExpr(expr *ast.KeyValueExpr) error {
	if err := p.expr(expr.Key); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) arrayType(expr *ast.ArrayType) error {
	if err := p.expr(expr.Len); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Elt)
}

func (p *pkgScanner) structType(expr *ast.StructType) error {
	return p.fieldList(expr.Fields)
}

func (p *pkgScanner) funcType(expr *ast.FuncType) error {
	if expr.TypeParams != nil && len(expr.TypeParams.List) > 0 {
		result := posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    fmt.Sprintf("generic function type"),
		}
		if p.s.greater(result) {
			return nil
		}
	}

	if err := p.fieldList(expr.Params); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.fieldList(expr.Results)
}

// xxx look for types in the field list; e.g. `interface { int8 | int16 | int32 | int64 }`
func (p *pkgScanner) interfaceType(expr *ast.InterfaceType) error {
	return p.fieldList(expr.Methods)
}

func (p *pkgScanner) mapType(expr *ast.MapType) error {
	if err := p.expr(expr.Key); err != nil {
		return err
	}
	if p.s.result.Version() == MaxGoMinorVersion {
		return nil
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) chanType(expr *ast.ChanType) error {
	return p.expr(expr.Value)
}
