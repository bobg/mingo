package mingo

import (
	"fmt"
	"go/ast"
)

func (s scanner) stmt(stmt ast.Stmt) (int, error) {
	if stmt == nil {
		return 0, nil
	}

	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		return s.declStmt(stmt)
	case *ast.EmptyStmt:
		return MinGoMinorVersion, nil
	case *ast.LabeledStmt:
		return s.labeledStmt(stmt)
	case *ast.ExprStmt:
		return s.exprStmt(stmt)
	case *ast.SendStmt:
		return s.sendStmt(stmt)
	case *ast.IncDecStmt:
		return s.incDecStmt(stmt)
	case *ast.AssignStmt:
		return s.assignStmt(stmt)
	case *ast.GoStmt:
		return s.goStmt(stmt)
	case *ast.DeferStmt:
		return s.deferStmt(stmt)
	case *ast.ReturnStmt:
		return s.returnStmt(stmt)
	case *ast.BranchStmt:
		return MinGoMinorVersion, nil
	case *ast.BlockStmt:
		return s.blockStmt(stmt)
	case *ast.IfStmt:
		return s.ifStmt(stmt)
	case *ast.CaseClause:
		return s.caseClause(stmt)
	case *ast.SwitchStmt:
		return s.switchStmt(stmt)
	case *ast.TypeSwitchStmt:
		return s.typeSwitchStmt(stmt)
	case *ast.CommClause:
		return s.commClause(stmt)
	case *ast.SelectStmt:
		return s.selectStmt(stmt)
	case *ast.ForStmt:
		return s.forStmt(stmt)
	case *ast.RangeStmt:
		return s.rangeStmt(stmt)
	default:
		return 0, fmt.Errorf("unknown statement type %T", stmt)
	}
}

func (s scanner) declStmt(stmt *ast.DeclStmt) (int, error) {
	return s.decl(stmt.Decl)
}

func (s scanner) labeledStmt(stmt *ast.LabeledStmt) (int, error) {
	return s.stmt(stmt.Stmt)
}

func (s scanner) exprStmt(stmt *ast.ExprStmt) (int, error) {
	return s.expr(stmt.X)
}

func (s scanner) sendStmt(stmt *ast.SendStmt) (int, error) {
	result1, err := s.expr(stmt.Chan)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(stmt.Value)
	return max(result1, result2), err
}

func (s scanner) incDecStmt(stmt *ast.IncDecStmt) (int, error) {
	return s.expr(stmt.X)
}

func (s scanner) assignStmt(stmt *ast.AssignStmt) (int, error) {
	result := MinGoMinorVersion
	for _, expr := range stmt.Lhs {
		exprResult, err := s.expr(expr)
		if err != nil {
			return 0, err
		}
		result = max(result, exprResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	for _, expr := range stmt.Rhs {
		exprResult, err := s.expr(expr)
		if err != nil {
			return 0, err
		}
		result = max(result, exprResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) goStmt(stmt *ast.GoStmt) (int, error) {
	return s.callExpr(stmt.Call)
}

func (s scanner) deferStmt(stmt *ast.DeferStmt) (int, error) {
	return s.callExpr(stmt.Call)
}

func (s scanner) returnStmt(stmt *ast.ReturnStmt) (int, error) {
	result := MinGoMinorVersion
	for _, expr := range stmt.Results {
		exprResult, err := s.expr(expr)
		if err != nil {
			return 0, err
		}
		result = max(result, exprResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) blockStmt(stmt *ast.BlockStmt) (int, error) {
	result := MinGoMinorVersion
	for _, stmt := range stmt.List {
		stmtResult, err := s.stmt(stmt)
		if err != nil {
			return 0, err
		}
		result = max(result, stmtResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) ifStmt(stmt *ast.IfStmt) (int, error) {
	result1, err := s.expr(stmt.Cond)
	if err != nil {
		return 0, err
	}
	result2, err := s.blockStmt(stmt.Body)
	if err != nil {
		return 0, err
	}
	result3, err := s.stmt(stmt.Else)
	return max(result1, result2, result3), err
}

func (s scanner) caseClause(stmt *ast.CaseClause) (int, error) {
	result := MinGoMinorVersion
	for _, expr := range stmt.List {
		exprResult, err := s.expr(expr)
		if err != nil {
			return 0, err
		}
		result = max(result, exprResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	for _, stmt := range stmt.Body {
		stmtResult, err := s.stmt(stmt)
		if err != nil {
			return 0, err
		}
		result = max(result, stmtResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) switchStmt(stmt *ast.SwitchStmt) (int, error) {
	result1, err := s.stmt(stmt.Init)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(stmt.Tag)
	if err != nil {
		return 0, err
	}
	result3, err := s.blockStmt(stmt.Body)
	return max(result1, result2, result3), err
}

func (s scanner) typeSwitchStmt(stmt *ast.TypeSwitchStmt) (int, error) {
	result1, err := s.stmt(stmt.Init)
	if err != nil {
		return 0, err
	}
	result2, err := s.stmt(stmt.Assign)
	if err != nil {
		return 0, err
	}
	result3, err := s.blockStmt(stmt.Body)
	return max(result1, result2, result3), err
}

func (s scanner) commClause(stmt *ast.CommClause) (int, error) {
	result, err := s.stmt(stmt.Comm)
	if err != nil {
		return 0, err
	}
	for _, stmt := range stmt.Body {
		stmtResult, err := s.stmt(stmt)
		if err != nil {
			return 0, err
		}
		result = max(result, stmtResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) selectStmt(stmt *ast.SelectStmt) (int, error) {
	return s.blockStmt(stmt.Body)
}

func (s scanner) forStmt(stmt *ast.ForStmt) (int, error) {
	result1, err := s.stmt(stmt.Init)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(stmt.Cond)
	if err != nil {
		return 0, err
	}
	result3, err := s.stmt(stmt.Post)
	if err != nil {
		return 0, err
	}
	result4, err := s.blockStmt(stmt.Body)
	return max(result1, result2, result3, result4), err
}

func (s scanner) rangeStmt(stmt *ast.RangeStmt) (int, error) {
	result1, err := s.expr(stmt.Key)
	if err != nil {
		return 0, err
	}
	result2, err := s.expr(stmt.Value)
	if err != nil {
		return 0, err
	}
	result3, err := s.expr(stmt.X)
	if err != nil {
		return 0, err
	}
	result4, err := s.blockStmt(stmt.Body)
	return max(result1, result2, result3, result4), err
}
