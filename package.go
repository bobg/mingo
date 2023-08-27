package mingo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/pkg/errors"
)

type pkgScanner struct {
	s       *Scanner
	pkgpath string
	fset    *token.FileSet
	info    *types.Info
	result  Result
}

func (p *pkgScanner) file(file *ast.File) error {
	for _, decl := range file.Decls {
		if err := p.decl(decl); err != nil {
			return errors.Wrapf(err, "scanning decl at %s", p.fset.Position(decl.Pos()))
		}
		if p.result.Version() == MaxGoMinorVersion {
			break
		}
	}
	return nil
}

func (p *pkgScanner) greater(r Result) bool {
	var isMax bool
	p.result, isMax = p.s.greater(p.result, r)
	return isMax
}
