package mingo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// Bool result tells whether the max known Go version has been reached.
func (p *pkgScanner) expr(expr ast.Expr) (bool, error) {
	return p.exprHelper(expr, false)
}

func (p *pkgScanner) exprHelper(expr ast.Expr, isCallFun bool) (bool, error) {
	if expr == nil {
		return false, nil
	}

	switch expr := expr.(type) {
	case *ast.Ident:
		return p.ident(expr)
	case *ast.Ellipsis:
		return false, nil
	case *ast.BasicLit:
		return p.basicLit(expr)
	case *ast.FuncLit:
		return p.funcLit(expr)
	case *ast.CompositeLit:
		return p.compositeLit(expr)
	case *ast.ParenExpr:
		return p.parenExpr(expr)
	case *ast.SelectorExpr:
		return p.selectorExpr(expr, isCallFun)
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
		return false, fmt.Errorf("unknown expr type %T", expr)
	}
}

func (p *pkgScanner) ident(ident *ast.Ident) (bool, error) {
	if tv, ok := p.info.Types[ident]; ok && tv.IsType() && ident.Name == "any" {
		// It's a type named "any," but is it the predefined "any" type?
		if obj, ok := p.info.Uses[ident]; ok && obj.Pkg() == nil {
			idResult := posResult{
				version: 18,
				pos:     p.fset.Position(ident.Pos()),
				desc:    `"any" builtin`,
			}
			return p.result(idResult), nil
		}
	}

	obj, ok := p.info.Uses[ident]
	if !ok || obj == nil {
		return false, nil
	}
	if !obj.Exported() {
		return false, nil
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return false, nil
	}
	pkgpath := obj.Pkg().Path()

	if v := p.s.lookup(pkgpath, obj.Id(), ""); v > 0 {
		idResult := posResult{
			version: v,
			pos:     p.fset.Position(ident.Pos()),
			desc:    fmt.Sprintf(`"%s".%s`, pkgpath, obj.Id()),
		}
		return p.result(idResult), nil
	}
	return false, nil
}

func (p *pkgScanner) basicLit(lit *ast.BasicLit) (bool, error) {
	switch lit.Kind {
	case token.CHAR, token.STRING:
		return false, nil
	}

	// Maybe...
	numResult := posResult{
		version: 13,
		pos:     p.fset.Position(lit.Pos()),
		desc:    "expanded numeric literal",
	}

	// Does this numeric literal use expanded Go 1.13 syntax?
	if strings.Contains(lit.Value, "_") {
		return p.result(numResult), nil
	}
	if strings.HasPrefix(lit.Value, "0b") || strings.HasPrefix(lit.Value, "0B") {
		return p.result(numResult), nil
	}
	if strings.HasPrefix(lit.Value, "0o") || strings.HasPrefix(lit.Value, "0O") {
		return p.result(numResult), nil
	}
	if strings.HasPrefix(lit.Value, "0x") || strings.HasPrefix(lit.Value, "0X") {
		return p.result(numResult), nil
	}
	if strings.HasSuffix(lit.Value, "i") {
		return p.result(numResult), nil
	}

	return false, nil
}

func (p *pkgScanner) funcLit(lit *ast.FuncLit) (bool, error) {
	if lit.Type.TypeParams != nil && len(lit.Type.TypeParams.List) > 0 {
		// I think this case is impossible.
		p.result(posResult{
			version: 18,
			pos:     p.fset.Position(lit.Pos()),
			desc:    "generic function literal",
		})
	}

	return p.funcBody(lit.Body)
}

func (p *pkgScanner) funcBody(body *ast.BlockStmt) (bool, error) {
	if body == nil {
		return false, nil
	}

	if len(body.List) == 0 {
		return false, nil
	}

	for _, stmt := range body.List {
		if isMax, err := p.stmt(stmt); err != nil || isMax {
			return isMax, err
		}
	}

	last := body.List[len(body.List)-1]
	if _, ok := last.(*ast.ReturnStmt); ok {
		return false, nil
	}

	res := posResult{
		version: 1,
		pos:     p.fset.Position(last.End()),
		desc:    "function body with no final return statement",
	}
	return p.result(res), nil
}

func (p *pkgScanner) compositeLit(lit *ast.CompositeLit) (bool, error) {
	for _, elt := range lit.Elts {
		if isMax, err := p.expr(elt); err != nil || isMax {
			return isMax, err
		}

		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ck, ok := kv.Key.(*ast.CompositeLit); ok && ck.Type == nil {
				p.result(posResult{
					version: 5,
					pos:     p.fset.Position(ck.Pos()),
					desc:    "composite literal with composite-type key and no explicit type",
				})
			}
		}
	}

	return false, nil
}

func (p *pkgScanner) parenExpr(expr *ast.ParenExpr) (bool, error) {
	return p.expr(expr.X)
}

func (p *pkgScanner) selectorExpr(expr *ast.SelectorExpr, isCallFun bool) (bool, error) {
	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}

	if obj, ok := p.info.Uses[expr.Sel]; ok && obj != nil {
		typ := ""
		if sig, ok := obj.Type().(*types.Signature); ok {
			if !isCallFun {
				p.result(posResult{
					version: 1,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "method used as value",
				})
			}
			if sig.Recv() != nil {
				typ = sig.Recv().Type().String()
				parts := strings.Split(typ, ".")
				typ = parts[len(parts)-1]
			}
		}

		pkg := obj.Pkg()
		if pkg == nil {
			return false, nil
		}
		pkgpath := pkg.Path()
		if v := p.s.lookup(pkgpath, expr.Sel.Name, typ); v > 0 {
			selResult := posResult{
				version: v,
				pos:     p.fset.Position(expr.Pos()),
				desc:    fmt.Sprintf(`"%s".%s`, pkgpath, expr.Sel.Name),
			}
			if p.result(selResult) {
				return true, nil
			}
		}
		return false, nil
	}

	sel, ok := p.info.Selections[expr]
	if !ok {
		return false, nil
	}
	obj := sel.Obj()
	if obj == nil {
		return false, nil
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return false, nil
	}
	pkgpath := pkg.Path()

	typ := sel.Recv()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	typestr := typ.String()

	v := p.s.lookup(pkgpath, expr.Sel.Name, typestr)
	if v == 0 {
		return false, nil
	}

	selResult := posResult{
		version: v,
		pos:     p.fset.Position(expr.Pos()),
		desc:    fmt.Sprintf(`"%s".%s.%s`, pkgpath, typestr, expr.Sel.Name),
	}
	return p.result(selResult), nil
}

func (p *pkgScanner) indexExpr(expr *ast.IndexExpr) (bool, error) {
	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}
	if p.isTypeExpr(expr.Index) {
		p.result(posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "generic instantiation",
		})
	}
	return p.expr(expr.Index)
}

func (p *pkgScanner) indexListExpr(expr *ast.IndexListExpr) (bool, error) {
	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}
	for _, index := range expr.Indices {
		if p.isTypeExpr(index) {
			p.result(posResult{
				version: 18,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "generic instantiation",
			})
		}
		if isMax, err := p.expr(index); err != nil || isMax {
			return isMax, err
		}
	}
	return false, nil
}

func (p *pkgScanner) sliceExpr(expr *ast.SliceExpr) (bool, error) {
	if expr.Slice3 {
		p.result(posResult{
			version: 5,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "slice expression with 3 indices",
		})
	}

	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}
	if isMax, err := p.expr(expr.Low); err != nil || isMax {
		return isMax, err
	}
	if isMax, err := p.expr(expr.High); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Max)
}

func (p *pkgScanner) typeAssertExpr(expr *ast.TypeAssertExpr) (bool, error) {
	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Type)
}

func (p *pkgScanner) callExpr(expr *ast.CallExpr) (bool, error) {
	tv, ok := p.info.Types[expr.Fun]
	if !ok {
		return false, fmt.Errorf("no type info for call expression at %s", p.fset.Position(expr.Pos()))
	}

	switch {
	case tv.IsType() && len(expr.Args) == 1:
		return p.typeConversion(expr, tv)
	case tv.IsBuiltin():
		return p.builtinCall(expr)
	}

	if isMax, err := p.exprHelper(expr.Fun, true); err != nil || isMax {
		return isMax, err
	}

	return p.callArgs(expr)
}

func (p *pkgScanner) callArgs(expr *ast.CallExpr) (bool, error) {
	for _, arg := range expr.Args {
		if isMax, err := p.expr(arg); err != nil || isMax {
			return isMax, err
		}
	}

	return false, nil
}

// expr.Fun is a type expression, and len(expr.Args) == 1.
func (p *pkgScanner) typeConversion(expr *ast.CallExpr, funtv types.TypeAndValue) (bool, error) {
	argtv, ok := p.info.Types[expr.Args[0]]
	if !ok {
		return false, fmt.Errorf("no type info for type conversion argument at %s", p.fset.Position(expr.Args[0].Pos()))
	}

	var (
		funtyp = funtv.Type.Underlying()
		argtyp = argtv.Type.Underlying()
	)

	// Is this a conversion from struct A to struct B, which are the same except for differing struct tags?
	if argStruct, ok := argtyp.(*types.Struct); ok {
		if funStruct, ok := funtyp.(*types.Struct); ok {
			if differingTags(argStruct, funStruct) {
				res := posResult{
					version: 8,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "conversion between structs with differing struct tags",
				}
				return p.result(res), nil
			}
		}
	}

	// Is this a conversion from slice to array or array pointer?
	if _, ok := argtyp.(*types.Slice); ok {
		if _, ok := funtyp.(*types.Array); ok {
			convResult := posResult{
				version: 20,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "conversion from slice to array",
			}
			return p.result(convResult), nil
		}
		if ptr, ok := funtyp.(*types.Pointer); ok {
			elemtype := ptr.Elem().Underlying()
			if _, ok := elemtype.(*types.Array); ok {
				convResult := posResult{
					version: 17,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "conversion from slice to array pointer",
				}
				return p.result(convResult), nil
			}
		}
	}

	return false, nil
}

func (p *pkgScanner) builtinCall(expr *ast.CallExpr) (bool, error) {
	operator := ast.Unparen(expr.Fun)
	switch operator := operator.(type) {
	case *ast.Ident:
		switch operator.Name {
		case "min", "max", "clear":
			result := posResult{
				version: 21,
				pos:     p.fset.Position(expr.Pos()),
				desc:    fmt.Sprintf("use of %s builtin", operator.Name),
			}
			if p.result(result) {
				return true, nil
			}
		}

	case *ast.SelectorExpr:
		if operator.Sel.Name == "unsafe" {
			if id := getID(operator.X); id != nil {
				switch id.Name {
				case "Slice", "IntegerType", "Add":
					result := posResult{
						version: 17,
						pos:     p.fset.Position(expr.Pos()),
						desc:    fmt.Sprintf("use of unsafe.%s builtin", id.Name),
					}
					if p.result(result) {
						return true, nil
					}

				case "String", "StringData", "SliceData":
					result := posResult{
						version: 20,
						pos:     p.fset.Position(expr.Pos()),
						desc:    fmt.Sprintf("use of unsafe.%s builtin", id.Name),
					}
					if p.result(result) {
						return true, nil
					}
				}
			}
		}
	}

	return p.callArgs(expr)
}

func getID(expr ast.Expr) *ast.Ident {
	expr = ast.Unparen(expr)
	if id, ok := expr.(*ast.Ident); ok {
		return id
	}
	return nil
}

func (p *pkgScanner) starExpr(expr *ast.StarExpr) (bool, error) {
	return p.expr(expr.X)
}

func (p *pkgScanner) unaryExpr(expr *ast.UnaryExpr) (bool, error) {
	if expr.Op == token.TILDE {
		p.result(posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "tilde operator",
		})
	}
	return p.expr(expr.X)
}

func (p *pkgScanner) binaryExpr(expr *ast.BinaryExpr) (bool, error) {
	switch expr.Op {
	case token.SHL, token.SHR:
		if p.isSigned(expr.Y) {
			p.result(posResult{
				version: 13,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "signed shift count",
			})
		}
	}
	if isMax, err := p.expr(expr.X); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Y)
}

func (p *pkgScanner) keyValueExpr(expr *ast.KeyValueExpr) (bool, error) {
	if isMax, err := p.expr(expr.Key); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) arrayType(expr *ast.ArrayType) (bool, error) {
	if isMax, err := p.expr(expr.Len); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Elt)
}

func (p *pkgScanner) structType(expr *ast.StructType) (bool, error) {
	return p.fieldList(expr.Fields)
}

func (p *pkgScanner) funcType(expr *ast.FuncType) (bool, error) {
	if expr.TypeParams != nil && len(expr.TypeParams.List) > 0 {
		p.result(posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "generic function type",
		})
	}
	if isMax, err := p.fieldList(expr.Params); err != nil || isMax {
		return isMax, err
	}
	return p.fieldList(expr.Results)
}

func (p *pkgScanner) interfaceType(expr *ast.InterfaceType) (bool, error) {
	if tv, ok := p.info.Types[expr]; ok {
		if intf, ok := tv.Type.(*types.Interface); ok {
			if p.checkInterfaceOverlaps(intf, expr.Pos()) {
				return true, nil
			}
			if !intf.IsMethodSet() {
				p.result(posResult{
					version: 18,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "interface containing type terms",
				})
			}
		}
	}
	return p.fieldList(expr.Methods)
}

// Is intf defined in terms of overlapping method sets?
// If so, require Go 1.14 or later.
func (p *pkgScanner) checkInterfaceOverlaps(intf *types.Interface, pos token.Pos) bool {
	for i := 0; i < intf.NumEmbeddeds(); i++ {
		embed := intf.EmbeddedType(i)
		if embed1, ok := embed.Underlying().(*types.Interface); ok {
			for j := i + 1; j < intf.NumEmbeddeds(); j++ {
				embed = intf.EmbeddedType(j)
				if embed2, ok := embed.Underlying().(*types.Interface); ok {
					for ii := 0; ii < embed1.NumMethods(); ii++ {
						for jj := 0; jj < embed2.NumMethods(); jj++ {
							if embed1.Method(ii).Name() == embed2.Method(jj).Name() { // we don't care whether the signatures match
								return p.result(posResult{
									version: 14,
									pos:     p.fset.Position(pos),
									desc:    "interface defined in terms of overlapping method sets",
								})
							}
						}
					}
				}
			}

			for j := 0; j < intf.NumExplicitMethods(); j++ {
				for ii := 0; ii < embed1.NumMethods(); ii++ {
					if intf.ExplicitMethod(j).Name() == embed1.Method(ii).Name() { // we don't care whether the signatures match
						return p.result(posResult{
							version: 14,
							pos:     p.fset.Position(pos),
							desc:    "interface defined in terms of overlapping method sets",
						})
					}
				}
			}
		}
	}

	return false
}

func (p *pkgScanner) mapType(expr *ast.MapType) (bool, error) {
	if isMax, err := p.expr(expr.Key); err != nil || isMax {
		return isMax, err
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) chanType(expr *ast.ChanType) (bool, error) {
	return p.expr(expr.Value)
}

func differingTags(a, b *types.Struct) bool {
	n := a.NumFields()
	if n != b.NumFields() {
		return false // sic - we don't care if the fields differ, only whether the tags differ
	}
	for i := 0; i < n; i++ {
		if a.Tag(i) != b.Tag(i) {
			return true
		}
	}
	return false
}
