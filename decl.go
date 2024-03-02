package mingo

import (
	"go/ast"

	"github.com/bobg/errors"
)

func (p *pkgScanner) decl(decl ast.Decl) (bool, error) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return p.funcDecl(decl)
	case *ast.GenDecl:
		return p.genDecl(decl)
	}
	return false, nil
}

func (p *pkgScanner) funcDecl(decl *ast.FuncDecl) (bool, error) {
	if isMax, err := p.fieldList(decl.Recv); err != nil || isMax {
		return isMax, errors.Wrapf(err, "scanning receiver for func %s", decl.Name.Name)
	}

	// Generics are supported in Go 1.18 and later.
	if decl.Type.TypeParams != nil && len(decl.Type.TypeParams.List) > 0 {
		declResult := posResult{
			version: 18,
			pos:     p.fset.Position(decl.Pos()),
			desc:    "generic func decl",
		}
		if p.result(declResult) {
			return true, nil
		}
	}

	if isMax, err := p.fieldList(decl.Type.Params); err != nil || isMax {
		return isMax, errors.Wrapf(err, "scanning params for func %s", decl.Name.Name)
	}
	if isMax, err := p.fieldList(decl.Type.Results); err != nil || isMax {
		return false, errors.Wrapf(err, "scanning results for func %s", decl.Name.Name)
	}

	return p.funcBody(decl.Body)
}

func (p *pkgScanner) fieldList(list *ast.FieldList) (bool, error) {
	if list == nil {
		return false, nil
	}

	for _, field := range list.List {
		if isMax, err := p.field(field); err != nil || isMax {
			return isMax, err
		}
	}

	return false, nil
}

func (p *pkgScanner) field(field *ast.Field) (bool, error) {
	return p.expr(field.Type)
}

func (p *pkgScanner) genDecl(decl *ast.GenDecl) (bool, error) {
	for _, spec := range decl.Specs {
		if isMax, err := p.spec(spec); err != nil || isMax {
			return isMax, err
		}
	}
	return false, nil
}

func (p *pkgScanner) spec(spec ast.Spec) (bool, error) {
	switch spec := spec.(type) {
	case *ast.ValueSpec:
		return p.valueSpec(spec)
	case *ast.TypeSpec:
		return p.typeSpec(spec)
	}
	return false, nil
}

func (p *pkgScanner) valueSpec(spec *ast.ValueSpec) (bool, error) {
	if isMax, err := p.expr(spec.Type); err != nil || isMax {
		return isMax, err
	}
	for _, value := range spec.Values {
		if isMax, err := p.expr(value); err != nil || isMax {
			return isMax, err
		}
	}
	return false, nil
}

func (p *pkgScanner) typeSpec(spec *ast.TypeSpec) (bool, error) {
	if spec.Assign.IsValid() {
		res := posResult{
			version: 9,
			pos:     p.fset.Position(spec.Pos()),
			desc:    "type alias",
		}
		if p.result(res) {
			return true, nil
		}
	}

	// Generics are supported in Go 1.18 and later.
	if spec.TypeParams != nil && len(spec.TypeParams.List) > 0 {
		declResult := posResult{
			version: 18,
			pos:     p.fset.Position(spec.Pos()),
			desc:    "generic type decl",
		}
		if p.result(declResult) {
			return true, nil
		}
	}
	return p.expr(spec.Type)
}
