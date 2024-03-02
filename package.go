package mingo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/bobg/errors"
)

type pkgScanner struct {
	s       *Scanner
	pkgpath string
	fset    *token.FileSet
	info    *types.Info
}

func (p *pkgScanner) file(file *ast.File) error {
	for _, decl := range file.Decls {
		if err := p.decl(decl); err != nil {
			return errors.Wrapf(err, "scanning decl at %s", p.fset.Position(decl.Pos()))
		}
		if p.isMax() {
			break
		}
	}
	return nil
}

func (p *pkgScanner) result(r Result) bool {
	return p.s.result(r)
}

func (p *pkgScanner) isMax() bool {
	return p.s.isMax()
}

func (p *pkgScanner) isTypeExpr(expr ast.Expr) bool {
	tv, ok := p.info.Types[expr]
	if !ok {
		return false
	}
	return tv.IsType()
}

func (p *pkgScanner) isSigned(expr ast.Expr) bool {
	tv, ok := p.info.Types[expr]
	if !ok {
		return false
	}
	basic, ok := tv.Type.(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&types.IsInteger != 0 && basic.Info()&types.IsUnsigned == 0
}
