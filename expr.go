package mingo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

func (p *pkgScanner) expr(expr ast.Expr) error {
	return p.exprHelper(expr, false)
}

func (p *pkgScanner) exprHelper(expr ast.Expr, isCallFun bool) error {
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
		return fmt.Errorf("unknown expr type %T", expr)
	}
}

func (p *pkgScanner) ident(ident *ast.Ident) error {
	if tv, ok := p.info.Types[ident]; ok && tv.IsType() && ident.Name == "any" {
		// It's a type named "any," but is it the predefined "any" type?
		if obj, ok := p.info.Uses[ident]; ok && obj.Pkg() == nil {
			idResult := posResult{
				version: 18,
				pos:     p.fset.Position(ident.Pos()),
				desc:    `"any" builtin`,
			}
			p.greater(idResult)
			return nil
		}
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
		p.greater(idResult)
	}
	return nil
}

func (p *pkgScanner) basicLit(lit *ast.BasicLit) error {
	switch lit.Kind {
	case token.CHAR, token.STRING:
		return nil
	}

	// Maybe...
	numResult := posResult{
		version: 13,
		pos:     p.fset.Position(lit.Pos()),
		desc:    "expanded numeric literal",
	}

	// Does this numeric literal use expanded Go 1.13 syntax?
	if strings.Contains(lit.Value, "_") {
		p.greater(numResult)
		return nil
	}
	if strings.HasPrefix(lit.Value, "0b") || strings.HasPrefix(lit.Value, "0B") {
		p.greater(numResult)
		return nil
	}
	if strings.HasPrefix(lit.Value, "0o") || strings.HasPrefix(lit.Value, "0O") {
		p.greater(numResult)
		return nil
	}
	if strings.HasPrefix(lit.Value, "0x") || strings.HasPrefix(lit.Value, "0X") {
		p.greater(numResult)
		return nil
	}
	if strings.HasSuffix(lit.Value, "i") {
		p.greater(numResult)
		return nil
	}

	return nil
}

func (p *pkgScanner) funcLit(lit *ast.FuncLit) error {
	if lit.Type.TypeParams != nil && len(lit.Type.TypeParams.List) > 0 {
		result := posResult{
			version: 18,
			pos:     p.fset.Position(lit.Pos()),
			desc:    "generic function literal",
		}
		p.greater(result)
	}

	return p.funcBody(lit.Body)
}

func (p *pkgScanner) funcBody(body *ast.BlockStmt) error {
	if body == nil {
		return nil
	}

	if len(body.List) == 0 {
		return nil
	}

	for _, stmt := range body.List {
		if err := p.stmt(stmt); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}

	last := body.List[len(body.List)-1]
	if _, ok := last.(*ast.ReturnStmt); ok {
		return nil
	}
	p.greater(posResult{
		version: 1,
		pos:     p.fset.Position(last.End()),
		desc:    "function body with no final return statement",
	})

	return nil
}

func (p *pkgScanner) compositeLit(lit *ast.CompositeLit) error {
	for _, elt := range lit.Elts {
		if err := p.expr(elt); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}

		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if ck, ok := kv.Key.(*ast.CompositeLit); ok && ck.Type == nil {
				p.greater(posResult{
					version: 5,
					pos:     p.fset.Position(ck.Pos()),
					desc:    "composite literal with composite-type key and no explicit type",
				})
			}
		}
	}

	return nil
}

func (p *pkgScanner) parenExpr(expr *ast.ParenExpr) error {
	return p.expr(expr.X)
}

func (p *pkgScanner) selectorExpr(expr *ast.SelectorExpr, isCallFun bool) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}

	if obj, ok := p.info.Uses[expr.Sel]; ok && obj != nil {
		if _, ok := obj.Type().(*types.Signature); ok && !isCallFun {
			p.greater(posResult{
				version: 1,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "method used as value",
			})
		}

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
			p.greater(selResult)
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
	p.greater(selResult)
	return nil
}

func (p *pkgScanner) indexExpr(expr *ast.IndexExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if p.isTypeExpr(expr.Index) {
		p.greater(posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "generic instantiation",
		})
	}
	return p.expr(expr.Index)
}

func (p *pkgScanner) indexListExpr(expr *ast.IndexListExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	for _, index := range expr.Indices {
		if p.isTypeExpr(index) {
			p.greater(posResult{
				version: 18,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "generic instantiation",
			})
		}
		if err := p.expr(index); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) sliceExpr(expr *ast.SliceExpr) error {
	if expr.Slice3 {
		result := posResult{
			version: 5,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "slice expression with 3 indices",
		}
		if p.greater(result) {
			return nil
		}
	}

	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(expr.Low); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(expr.High); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.expr(expr.Max)
}

func (p *pkgScanner) typeAssertExpr(expr *ast.TypeAssertExpr) error {
	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
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

	if err := p.exprHelper(expr.Fun, true); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}

	for _, arg := range expr.Args {
		if err := p.expr(arg); err != nil {
			return err
		}
		if p.isMax() {
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

	var (
		funtyp = funtv.Type.Underlying()
		argtyp = argtv.Type.Underlying()
	)

	// Is this a conversion from struct A to struct B, which are the same except for differing struct tags?
	if argStruct, ok := argtyp.(*types.Struct); ok {
		if funStruct, ok := funtyp.(*types.Struct); ok {
			if differingTags(argStruct, funStruct) {
				p.greater(posResult{
					version: 8,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "conversion between structs with differing struct tags",
				})
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
			p.greater(convResult)
			return nil
		}
		if ptr, ok := funtyp.(*types.Pointer); ok {
			elemtype := ptr.Elem().Underlying()
			if _, ok := elemtype.(*types.Array); ok {
				convResult := posResult{
					version: 17,
					pos:     p.fset.Position(expr.Pos()),
					desc:    "conversion from slice to array pointer",
				}
				p.greater(convResult)
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
		p.greater(result)
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
	if expr.Op == token.TILDE {
		p.greater(posResult{
			version: 18,
			pos:     p.fset.Position(expr.Pos()),
			desc:    "tilde operator",
		})
	}

	return p.expr(expr.X)
}

func (p *pkgScanner) binaryExpr(expr *ast.BinaryExpr) error {
	switch expr.Op {
	case token.SHL, token.SHR:
		if p.isSigned(expr.Y) {
			p.greater(posResult{
				version: 13,
				pos:     p.fset.Position(expr.Pos()),
				desc:    "signed shift count",
			})
		}
	}

	if err := p.expr(expr.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.expr(expr.Y)
}

func (p *pkgScanner) keyValueExpr(expr *ast.KeyValueExpr) error {
	if err := p.expr(expr.Key); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) arrayType(expr *ast.ArrayType) error {
	if err := p.expr(expr.Len); err != nil {
		return err
	}
	if p.isMax() {
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
			desc:    "generic function type",
		}
		if p.greater(result) {
			return nil
		}
	}

	if err := p.fieldList(expr.Params); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.fieldList(expr.Results)
}

func (p *pkgScanner) interfaceType(expr *ast.InterfaceType) error {
	if tv, ok := p.info.Types[expr]; ok {
		if intf, ok := tv.Type.(*types.Interface); ok {
			p.checkInterfaceOverlaps(intf, expr.Pos())

			if !intf.IsMethodSet() {
				p.greater(posResult{
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
func (p *pkgScanner) checkInterfaceOverlaps(intf *types.Interface, pos token.Pos) {
	for i := 0; i < intf.NumEmbeddeds(); i++ {
		embed := intf.EmbeddedType(i)
		if embed1, ok := embed.Underlying().(*types.Interface); ok {
			for j := i + 1; j < intf.NumEmbeddeds(); j++ {
				embed = intf.EmbeddedType(j)
				if embed2, ok := embed.Underlying().(*types.Interface); ok {
					for ii := 0; ii < embed1.NumMethods(); ii++ {
						for jj := 0; jj < embed2.NumMethods(); jj++ {
							if embed1.Method(ii).Name() == embed2.Method(jj).Name() { // we don't care whether the signatures match
								p.greater(posResult{
									version: 14,
									pos:     p.fset.Position(pos),
									desc:    "interface defined in terms of overlapping method sets",
								})
								return
							}
						}
					}
				}
			}

			for j := 0; j < intf.NumExplicitMethods(); j++ {
				for ii := 0; ii < embed1.NumMethods(); ii++ {
					if intf.ExplicitMethod(j).Name() == embed1.Method(ii).Name() { // we don't care whether the signatures match
						p.greater(posResult{
							version: 14,
							pos:     p.fset.Position(pos),
							desc:    "interface defined in terms of overlapping method sets",
						})
						return
					}
				}
			}
		}
	}
}

func (p *pkgScanner) mapType(expr *ast.MapType) error {
	if err := p.expr(expr.Key); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.expr(expr.Value)
}

func (p *pkgScanner) chanType(expr *ast.ChanType) error {
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
