package mingo

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

type Scanner struct {
	Deps    bool   // include dependencies
	Verbose bool   // be verbose
	HistDir string // find Go stdlib history in this directory (default: $GOROOT/api)

	h History
}

func (s *Scanner) ScanDir(dir string) (Result, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	conf := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  dir,
	}
	pkgs, err := packages.Load(conf, "./...")
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	return s.ScanPackages(pkgs)
}

func (s *Scanner) ScanPackages(pkgs []*packages.Package) (Result, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	var result Result = intResult(0)

	for _, pkg := range pkgs {
		pkgResult, err := s.scanPackage(pkg, result)
		if err != nil {
			return nil, errors.Wrapf(err, "scanning package %s", pkg.PkgPath)
		}

		var isMax bool
		if result, isMax = s.greater(result, pkgResult); isMax {
			return result, nil
		}
	}

	return result, nil
}

func (s *Scanner) scanPackage(pkg *packages.Package, result Result) (Result, error) {
	p := pkgScanner{
		s:       s,
		pkgpath: pkg.PkgPath,
		fset:    pkg.Fset,
		info:    pkg.TypesInfo,
		result:  result,
	}

	for _, file := range pkg.Syntax {
		filename := p.fset.Position(file.Pos()).Filename
		if err := p.file(file); err != nil {
			return nil, errors.Wrapf(err, "scanning file %s", filename)
		}
		if p.result.Version() == MaxGoMinorVersion {
			break
		}
	}

	return p.result, nil
}

func (s *Scanner) verbosef(format string, args ...any) {
	if !s.Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

func (s *Scanner) greater(older, newer Result) (Result, bool) {
	result := older
	if newer.Version() > older.Version() {
		result = newer
		s.verbosef("%s", result)
	}
	return result, result.Version() == MaxGoMinorVersion
}

func (s *Scanner) ensureHistory() error {
	if s.h != nil {
		return nil
	}
	h, err := ReadHist(s.HistDir)
	if err != nil {
		return err
	}
	s.h = h
	return nil
}
