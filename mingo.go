// Package mingo contains logic for scanning the packages in a Go module
// to determine the lowest-numbered version of Go that can build it.
package mingo

import (
	"go/ast"
	"go/types"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

const (
	MinGoMinorVersion = 13
	MaxGoMinorVersion = 21
)

func IDPkg(uses map[*ast.Ident]types.Object, id *ast.Ident) *types.Package {
	if id == nil {
		return nil
	}
	if obj := uses[id]; obj != nil {
		return obj.Pkg()
	}
	return nil
}

func ScanDir(dir string, h History) (int, error) {
	conf := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  dir,
	}
	pkgs, err := packages.Load(conf, "./...")
	if err != nil {
		return 0, errors.Wrap(err, "loading packages")
	}
	return ScanPackages(pkgs, h)
}

func ScanPackages(pkgs []*packages.Package, h History) (int, error) {
	result := MinGoMinorVersion
	for _, pkg := range pkgs {
		pkgResult, err := ScanPackage(pkg, h)
		if err != nil {
			return 0, errors.Wrapf(err, "scanning package %s", pkg.PkgPath)
		}
		result = max(result, pkgResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

type (
	scanner struct {
		pkg *packages.Package
		h   History
	}
	pkgapi struct {
		ids         map[string]int
		typemembers map[string]map[string]int
	}
)

func (s scanner) lookup(pkgpath, name, typ string) int {
	p, ok := s.h[pkgpath]
	if !ok {
		return 0
	}
	if typ == "" {
		return p.IDs[name]
	}
	m, ok := p.Types[typ]
	if !ok {
		return 0
	}
	return m[name]
}

func ScanPackage(pkg *packages.Package, h History) (int, error) {
	var (
		result = MinGoMinorVersion
		s      = scanner{pkg: pkg, h: h}
	)
	for _, file := range pkg.Syntax {
		fileResult, err := s.file(file)
		if err != nil {
			return 0, errors.Wrapf(err, "scanning file %s", file.Name)
		}
		result = max(result, fileResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func (s scanner) file(file *ast.File) (int, error) {
	result := MinGoMinorVersion
	for _, decl := range file.Decls {
		declResult, err := s.decl(decl)
		if err != nil {
			return 0, err
		}
		result = max(result, declResult)
		if result == MaxGoMinorVersion {
			return result, nil
		}
	}
	return result, nil
}

func GoMinorVersion() int {
	vstr := runtime.Version()
	vstr = strings.TrimPrefix(vstr, "go")
	parts := strings.SplitN(vstr, ".", 3)
	if len(parts) < 2 {
		return 0
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return v
}
