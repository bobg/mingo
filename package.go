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
	res Result
}

// Bool result tells whether the max known Go version has been reached.
func (p *pkgScanner) file(file *ast.File) (bool, error) {
	for _, decl := range file.Decls {
		if isMax, err := p.decl(decl); err != nil || isMax {
			return isMax, errors.Wrapf(err, "scanning decl at %s", p.fset.Position(decl.Pos()))
		}
	}
	return false, nil
}

func (p *pkgScanner) result(r Result) bool {
	if r.Version() > p.res.Version() {
		p.res = r
	}
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
