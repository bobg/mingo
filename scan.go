package mingo

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

type Scanner struct {
	Deps     bool // include dependencies
	Indirect bool // with Deps, include indirect dependencies
	Verbose  bool // be verbose
	Tests    bool
	HistDir  string // find Go stdlib history in this directory (default: $GOROOT/api)

	h      *history
	result Result
}

func (s *Scanner) ScanDir(dir string) (Result, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	conf := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule,
		Dir:   dir,
		Tests: s.Tests,
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

	s.result = intResult(0)

	for _, pkg := range pkgs {
		if err := s.scanPackage(pkg); err != nil {
			return nil, errors.Wrapf(err, "scanning package %s", pkg.PkgPath)
		}
		if s.isMax() {
			break
		}
	}

	if s.Deps && len(pkgs) > 0 && pkgs[0].Module != nil {
		if err := s.scanDeps(pkgs[0].Module.GoMod); err != nil {
			return nil, errors.Wrap(err, "scanning dependencies")
		}
	}

	return s.result, nil
}

func (s *Scanner) scanPackage(pkg *packages.Package) error {
	p := pkgScanner{
		s:       s,
		pkgpath: pkg.PkgPath,
		fset:    pkg.Fset,
		info:    pkg.TypesInfo,
	}

	for _, file := range pkg.Syntax {
		filename := p.fset.Position(file.Pos()).Filename
		if err := p.file(file); err != nil {
			return errors.Wrapf(err, "scanning file %s", filename)
		}
		if p.isMax() {
			break
		}
	}

	return nil
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

func (s *Scanner) greater(result Result) bool {
	if result.Version() > s.result.Version() {
		s.result = result
		s.verbosef("%s", result)
	}
	return s.isMax()
}

var goverRegex = regexp.MustCompile(`^go(\d+)\.(\d+)`)

func (s *Scanner) ensureHistory() error {
	if s.h != nil {
		return nil
	}
	h, err := readHist(s.HistDir)
	if err != nil {
		return err
	}

	s.h = h

	gover := runtime.Version()
	m := goverRegex.FindStringSubmatch(gover)
	if len(m) == 0 {
		return nil
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return errors.Wrapf(err, "parsing major version from runtime version %s", gover)
	}
	if major != 1 {
		return fmt.Errorf("unexpected Go major version %d", major)
	}

	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return errors.Wrapf(err, "parsing minor version from runtime version %s", gover)
	}
	if minor != s.h.max {
		return fmt.Errorf("runtime Go version 1.%d does not match history max 1.%d (reading from %s)", minor, s.h.max, s.HistDir)
	}

	return nil
}

// Prereq: e.ensureHistory has been called.
func (s *Scanner) isMax() bool {
	return s.result.Version() >= s.h.max
}