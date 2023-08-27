package mingo

import (
	"go/ast"

	"github.com/pkg/errors"
)

func (p *pkgScanner) decl(decl ast.Decl) error {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return p.funcDecl(decl)
	case *ast.GenDecl:
		return p.genDecl(decl)
	}
	return nil
}

func (p *pkgScanner) funcDecl(decl *ast.FuncDecl) error {
	if err := p.fieldList(decl.Recv); err != nil {
		return errors.Wrapf(err, "scanning receiver for func %s", decl.Name.Name)
	}
	if p.result.Version() == MaxGoMinorVersion {
		return nil
	}

	// Generics are supported in Go 1.18 and later.
	if decl.Type.TypeParams != nil && len(decl.Type.TypeParams.List) > 0 {
		declResult := posResult{
			version: 18,
			pos:     p.fset.Position(decl.Pos()),
			desc:    "generic func decl",
		}
		if p.greater(declResult) {
			return nil
		}
	}

	if err := p.fieldList(decl.Type.Params); err != nil {
		return errors.Wrapf(err, "scanning params for func %s", decl.Name.Name)
	}
	if p.result.Version() == MaxGoMinorVersion {
		return nil
	}

	if err := p.fieldList(decl.Type.Results); err != nil {
		return errors.Wrapf(err, "scanning results for func %s", decl.Name.Name)
	}
	if p.result.Version() == MaxGoMinorVersion {
		return nil
	}

	for _, stmt := range decl.Body.List {
		if err := p.stmt(stmt); err != nil {
			return err
		}
		if p.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}

	return nil
}

func (p *pkgScanner) fieldList(list *ast.FieldList) error {
	if list == nil {
		return nil
	}

	for _, field := range list.List {
		if err := p.field(field); err != nil {
			return err
		}
		if p.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}

	return nil
}

func (p *pkgScanner) field(field *ast.Field) error {
	return p.expr(field.Type)
}

func (p *pkgScanner) genDecl(decl *ast.GenDecl) error {
	for _, spec := range decl.Specs {
		if err := p.spec(spec); err != nil {
			return err
		}
		if p.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}
	return nil
}

func (p *pkgScanner) spec(spec ast.Spec) error {
	switch spec := spec.(type) {
	case *ast.ValueSpec:
		return p.valueSpec(spec)
	case *ast.TypeSpec:
		return p.typeSpec(spec)
	}
	return nil
}

func (p *pkgScanner) valueSpec(spec *ast.ValueSpec) error {
	for _, value := range spec.Values {
		if err := p.expr(value); err != nil {
			return err
		}
		if p.result.Version() == MaxGoMinorVersion {
			return nil
		}
	}
	return nil
}

// xxx check for interface definition with overlapping method sets,
// allowed as of Go 1.14.
func (p *pkgScanner) typeSpec(spec *ast.TypeSpec) error {
	// Generics are supported in Go 1.18 and later.
	if spec.TypeParams != nil && len(spec.TypeParams.List) > 0 {
		declResult := posResult{
			version: 18,
			pos:     p.fset.Position(spec.Pos()),
			desc:    "generic type decl",
		}
		if p.greater(declResult) {
			return nil
		}
	}
	return p.expr(spec.Type)
}
