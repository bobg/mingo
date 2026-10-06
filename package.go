package mingo

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/bobg/errors"
)

type pkgScanner struct {
	s       *Scanner
	pkgpath string
	fset    *token.FileSet
	info    *types.Info
}

// Bool result tells whether the max known Go version has been reached.
func (p *pkgScanner) file(file *ast.File) (bool, error) {
	if v := parseFileGoVersion(file); v > 0 {
		if isMax := p.result(posResult{
			version: v,
			pos:     p.fset.Position(file.FileStart),
			desc:    "required by //go:build or // +build directives",
		}); isMax {
			return true, nil
		}
	}

	for _, decl := range file.Decls {
		if isMax, err := p.decl(decl); err != nil || isMax {
			return isMax, errors.Wrapf(err, "scanning decl at %s", p.fset.Position(decl.Pos()))
		}
	}
	return false, nil
}

// parseFileGoVersion parses an ast.File.GoVersion string of the form "", "go1", or "go1.N".
func parseFileGoVersion(file *ast.File) int {
	rest, ok := strings.CutPrefix(file.GoVersion, "go1.")
	if !ok {
		return 0
	}
	v, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return v
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
