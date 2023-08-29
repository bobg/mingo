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
}

func (p *pkgScanner) file(file *ast.File) error {
	for _, decl := range file.Decls {
		if err := p.decl(decl); err != nil {
			return errors.Wrapf(err, "scanning decl at %s", p.fset.Position(decl.Pos()))
		}
		if p.s.result.Version() == MaxGoMinorVersion {
			break
		}
	}
	return nil
}
