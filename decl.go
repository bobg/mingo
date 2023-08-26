package mingo

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
)

func (s scanner) decl(decl ast.Decl) (int, error) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		return s.funcDecl(decl)
	case *ast.GenDecl:
		return s.genDecl(decl)
	default:
		return MinGoMinorVersion, nil
	}
}

func (s scanner) funcDecl(decl *ast.FuncDecl) (int, error) {
	result := MinGoMinorVersion
	fieldResult, err := s.fieldList(decl.Recv)
	if err != nil {
		return 0, errors.Wrapf(err, "scanning receiver for func %s", decl.Name.Name)
	}
	result = max(result, fieldResult)
	if result == MaxGoMinorVersion {
		return result, nil
	}

	// Generics are supported in Go 1.18 and later.
	if result < 18 && decl.Type.TypeParams != nil && len(decl.Type.TypeParams.List) > 0 {
		result = 18
		fieldResult, err = s.fieldList(decl.Type.TypeParams)
		if err != nil {
			return 0, errors.Wrapf(err, "scanning type params for func %s", decl.Name.Name)
		}
		result = max(result, fieldResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}

	fieldResult, err = s.fieldList(decl.Type.Params)
	if err != nil {
		return 0, errors.Wrapf(err, "scanning params for func %s", decl.Name.Name)
	}
	result = max(result, fieldResult)
	if result == MaxGoMinorVersion {
		return result, nil
	}
	fieldResult, err = s.fieldList(decl.Type.Results)
	if err != nil {
		return 0, errors.Wrapf(err, "scanning results for func %s", decl.Name.Name)
	}
	result = max(result, fieldResult)
	if result == MaxGoMinorVersion {
		return result, nil
	}
	for _, stmt := range decl.Body.List {
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

func (s scanner) fieldList(list *ast.FieldList) (int, error) {
	if list == nil {
		return 0, nil
	}

	result := MinGoMinorVersion
	for _, field := range list.List {
		fieldResult, err := s.field(field)
		if err != nil {
			return 0, err
		}
		result = max(result, fieldResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) field(field *ast.Field) (int, error) {
	return s.expr(field.Type)
}

func (s scanner) genDecl(decl *ast.GenDecl) (int, error) {
	result := MinGoMinorVersion
	for _, spec := range decl.Specs {
		specResult, err := s.spec(spec)
		if err != nil {
			return 0, err
		}
		result = max(result, specResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) spec(spec ast.Spec) (int, error) {
	switch spec := spec.(type) {
	case *ast.ImportSpec:
		return MinGoMinorVersion, nil
	case *ast.ValueSpec:
		return s.valueSpec(spec)
	case *ast.TypeSpec:
		return s.typeSpec(spec)
	default:
		return 0, fmt.Errorf("unknown spec type %T", spec)
	}
}

func (s scanner) valueSpec(spec *ast.ValueSpec) (int, error) {
	result := MinGoMinorVersion
	for _, value := range spec.Values {
		valueResult, err := s.expr(value)
		if err != nil {
			return 0, err
		}
		result = max(result, valueResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

// xxx check for interface definition with overlapping method sets,
// allowed as of Go 1.14.
func (s scanner) typeSpec(spec *ast.TypeSpec) (int, error) {
	result := MinGoMinorVersion
	if spec.TypeParams != nil && len(spec.TypeParams.List) > 0 {
		result = 18
		fieldResult, err := s.fieldList(spec.TypeParams)
		if err != nil {
			return 0, errors.Wrapf(err, "scanning type params for type %s", spec.Name.Name)
		}
		result = max(result, fieldResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	typeResult, err := s.expr(spec.Type)
	return max(result, typeResult), err
}
