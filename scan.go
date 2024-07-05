package mingo

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/bobg/errors"
	"golang.org/x/tools/go/packages"
)

// Scanner scans a directory or set of packages to determine the lowest-numbered version of Go 1.x that can build them.
type Scanner struct {
	Deps     bool   // include dependencies
	Indirect bool   // with Deps, include indirect dependencies
	Verbose  bool   // be verbose
	Tests    bool   // scan *_test.go files
	Check    bool   // produce an error if the module declares the wrong version in go.mod
	HistDir  string // find Go stdlib history in this directory (default: $GOROOT/api)

	Result Result

	h          *history
	depScanner depScanner
}

// Mode is the minimum mode needed when using [packages.Load] to scan packages.
const Mode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule

// VersionError is the error returned by [Scanner.ScanDir] or [Scanner.ScanPackages] when [Scanner.Check] is enabled.
type VersionError struct {
	Computed Result
	Declared int
}

func (e VersionError) Error() string {
	return fmt.Sprintf("go.mod declares version 1.%d but computed minimum is 1.%d [%s]", e.Declared, e.Computed.Version(), e.Computed)
}

// ScanDir scans the module in a directory to determine the lowest-numbered version of Go 1.x that can build it.
func (s *Scanner) ScanDir(dir string) (Result, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	conf := &packages.Config{
		Mode:  Mode,
		Dir:   dir,
		Tests: s.Tests,
	}
	pkgs, err := packages.Load(conf, "./...")
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	return s.ScanPackages(pkgs)
}

// ScanPackages scans the given packages to determine the lowest-numbered version of Go 1.x that can build them.
// The packages must all be in the same module.
// When using [packages.Load] to load the packages,
// the value for [packages.Config.Mode] must be at least [Mode].
func (s *Scanner) ScanPackages(pkgs []*packages.Package) (Result, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	s.Result = intResult(0)

	// Check for loading errors.
	var err error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			err = errors.Join(err, errors.Wrapf(e, "loading package %s", pkg.PkgPath))
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "loading package(s)")
	}

	for i, pkg := range pkgs {
		if pkg.Module == nil {
			return nil, fmt.Errorf("package %s has no module", pkg.PkgPath)
		}

		if i > 0 && pkg.Module.Path != pkgs[0].Module.Path {
			return nil, fmt.Errorf("multiple modules: %s and %s", pkgs[0].Module.Path, pkg.Module.Path)
		}

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

	if s.Check && len(pkgs) > 0 {
		var declared int
		parts := strings.SplitN(pkgs[0].Module.GoVersion, ".", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("go.mod has invalid go version %s", pkgs[0].Module.GoVersion)
		}
		declared, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("go.mod has invalid go version %s", pkgs[0].Module.GoVersion)
		}
		if s.Result.Version() != declared {
			return nil, VersionError{
				Computed: s.Result,
				Declared: declared,
			}
		}
	}

	return s.Result, nil
}

func (s *Scanner) scanPackage(pkg *packages.Package) error {
	return s.scanPackageHelper(pkg.PkgPath, pkg.Fset, pkg.TypesInfo, pkg.Syntax)
}

func (s *Scanner) scanPackageHelper(pkgpath string, fset *token.FileSet, info *types.Info, files []*ast.File) error {
	p := pkgScanner{
		s:       s,
		pkgpath: pkgpath,
		fset:    fset,
		info:    info,
	}

	for _, file := range files {
		filename := p.fset.Position(file.Pos()).Filename
		isInCache, err := isCacheFile(filename)
		if err != nil {
			return errors.Wrapf(err, "checking whether %s is in GOCACHE", filename)
		}
		if isInCache {
			continue
		}
		if err := p.file(file); err != nil {
			return errors.Wrapf(err, "scanning file %s", filename)
		}
		if p.isMax() {
			break
		}
	}

	return nil
}

func (s *Scanner) lookup(pkgpath, name, typ string) int {
	return s.h.lookup(pkgpath, name, typ)
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
	if result.Version() > s.Result.Version() {
		s.Result = result
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
	return s.Result.Version() >= s.h.max
}

var goCache = os.Getenv("GOCACHE")

func isCacheFile(filename string) (bool, error) {
	if goCache == "" {
		return false, nil
	}
	rel, err := filepath.Rel(goCache, filename)
	if err != nil {
		return false, errors.Wrapf(err, "computing relative path from %s to %s", goCache, filename)
	}
	return !strings.HasPrefix(rel, "../"), nil
}
