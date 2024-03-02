package mingo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func (p *pkgScanner) stmt(stmt ast.Stmt) error {
	if stmt == nil {
		return nil
	}

	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		return p.declStmt(stmt)
	case *ast.EmptyStmt:
		return nil
	case *ast.LabeledStmt:
		return p.labeledStmt(stmt)
	case *ast.ExprStmt:
		return p.exprStmt(stmt)
	case *ast.SendStmt:
		return p.sendStmt(stmt)
	case *ast.IncDecStmt:
		return p.incDecStmt(stmt)
	case *ast.AssignStmt:
		return p.assignStmt(stmt)
	case *ast.GoStmt:
		return p.goStmt(stmt)
	case *ast.DeferStmt:
		return p.deferStmt(stmt)
	case *ast.ReturnStmt:
		return p.returnStmt(stmt)
	case *ast.BranchStmt:
		return nil
	case *ast.BlockStmt:
		return p.blockStmt(stmt)
	case *ast.IfStmt:
		return p.ifStmt(stmt)
	case *ast.CaseClause:
		return p.caseClause(stmt)
	case *ast.SwitchStmt:
		return p.switchStmt(stmt)
	case *ast.TypeSwitchStmt:
		return p.typeSwitchStmt(stmt)
	case *ast.CommClause:
		return p.commClause(stmt)
	case *ast.SelectStmt:
		return p.selectStmt(stmt)
	case *ast.ForStmt:
		return p.forStmt(stmt)
	case *ast.RangeStmt:
		return p.rangeStmt(stmt)
	default:
		return fmt.Errorf("unknown statement type %T", stmt)
	}
}

func (p *pkgScanner) declStmt(stmt *ast.DeclStmt) error {
	return p.decl(stmt.Decl)
}

func (p *pkgScanner) labeledStmt(stmt *ast.LabeledStmt) error {
	return p.stmt(stmt.Stmt)
}

func (p *pkgScanner) exprStmt(stmt *ast.ExprStmt) error {
	return p.expr(stmt.X)
}

func (p *pkgScanner) sendStmt(stmt *ast.SendStmt) error {
	if err := p.expr(stmt.Chan); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.expr(stmt.Value)
}

func (p *pkgScanner) incDecStmt(stmt *ast.IncDecStmt) error {
	return p.expr(stmt.X)
}

func (p *pkgScanner) assignStmt(stmt *ast.AssignStmt) error {
	switch stmt.Tok {
	case token.SHL_ASSIGN, token.SHR_ASSIGN:
		if len(stmt.Rhs) == 1 && p.isSigned(stmt.Rhs[0]) {
			p.result(posResult{
				version: 13,
				pos:     p.fset.Position(stmt.Pos()),
				desc:    "signed shift count",
			})
		}
	}

	for _, expr := range stmt.Lhs {
		if err := p.expr(expr); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	for _, expr := range stmt.Rhs {
		if err := p.expr(expr); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) goStmt(stmt *ast.GoStmt) error {
	return p.callExpr(stmt.Call)
}

func (p *pkgScanner) deferStmt(stmt *ast.DeferStmt) error {
	return p.callExpr(stmt.Call)
}

func (p *pkgScanner) returnStmt(stmt *ast.ReturnStmt) error {
	for _, expr := range stmt.Results {
		if err := p.expr(expr); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) blockStmt(stmt *ast.BlockStmt) error {
	for _, stmt := range stmt.List {
		if err := p.stmt(stmt); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}

	return nil
}

func (p *pkgScanner) ifStmt(stmt *ast.IfStmt) error {
	if err := p.expr(stmt.Cond); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.blockStmt(stmt.Body); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.stmt(stmt.Else)
}

func (p *pkgScanner) caseClause(stmt *ast.CaseClause) error {
	for _, expr := range stmt.List {
		if err := p.expr(expr); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	for _, stmt := range stmt.Body {
		if err := p.stmt(stmt); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) switchStmt(stmt *ast.SwitchStmt) error {
	if err := p.stmt(stmt.Init); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(stmt.Tag); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.blockStmt(stmt.Body)
}

func (p *pkgScanner) typeSwitchStmt(stmt *ast.TypeSwitchStmt) error {
	if err := p.stmt(stmt.Init); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.stmt(stmt.Assign); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.blockStmt(stmt.Body)
}

func (p *pkgScanner) commClause(stmt *ast.CommClause) error {
	if err := p.stmt(stmt.Comm); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	for _, stmt := range stmt.Body {
		if err := p.stmt(stmt); err != nil {
			return err
		}
		if p.isMax() {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) selectStmt(stmt *ast.SelectStmt) error {
	return p.blockStmt(stmt.Body)
}

func (p *pkgScanner) forStmt(stmt *ast.ForStmt) error {
	if err := p.stmt(stmt.Init); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(stmt.Cond); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.stmt(stmt.Post); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	return p.blockStmt(stmt.Body)
}

func (p *pkgScanner) rangeStmt(stmt *ast.RangeStmt) error {
	if stmt.Key == nil && stmt.Value == nil {
		p.result(posResult{
			version: 4,
			pos:     p.fset.Position(stmt.Pos()),
			desc:    `variable-free "for range" statement`,
		})
	}

	if err := p.expr(stmt.Key); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(stmt.Value); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}
	if err := p.expr(stmt.X); err != nil {
		return err
	}
	if p.isMax() {
		return nil
	}

	tv, ok := p.info.Types[stmt.X]
	if !ok {
		return fmt.Errorf("no type info for range expression at %s", p.fset.Position(stmt.X.Pos()))
	}
	switch typ := tv.Type.Underlying().(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64, types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			// TODO: all integer kinds, or just some?
			p.result(posResult{
				version: 22,
				pos:     p.fset.Position(stmt.Pos()),
				desc:    "range over integer",
			})
		}

	case *types.Signature:
		p.result(posResult{
			version: 23,
			pos:     p.fset.Position(stmt.Pos()),
			desc:    "range over function",
		})
	}
	if p.isMax() {
		return nil
	}

	return p.blockStmt(stmt.Body)
}
