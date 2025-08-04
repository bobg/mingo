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
	Check    bool   // produce an error if the module declares a version in go.mod lower than the computed minimum
	Strict   bool   // with Check, require the go.mod declaration to be equal to the computed minimum
	HistDir  string // find Go stdlib history in this directory (default: $GOROOT/api)

	Result Result

	h          *history
	depScanner depScanner
}

// Mode is the minimum mode needed when using [packages.Load] to scan packages.
const Mode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule | packages.LoadSyntax

// VersionError is the error returned by [Scanner.ScanDir] or [Scanner.ScanPackages] when [Scanner.Check] is enabled.
type VersionError struct {
	Computed Result
	Declared int
}

func (e VersionError) Error() string {
	return fmt.Sprintf("go.mod declares version 1.%d but computed minimum is 1.%d [%s]", e.Declared, e.Computed.Version(), e.Computed)
}

// LoadError is the error returned by [Scanner.ScanDir] or [Scanner.ScanPackages] when loading packages fails.
type LoadError struct {
	Err  error
	Path string
}

func (e LoadError) Error() string {
	return fmt.Sprintf("loading packages in %s: %s", e.Path, e.Err)
}

func (e LoadError) Unwrap() error {
	return e.Err
}

// ScanDir scans the module in a directory to determine the lowest-numbered version of Go 1.x that can build it.
func (s *Scanner) ScanDir(dir string) (Result, error) {
	pkgs, err := s.LoadPackages(dir)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	return s.ScanPackages(pkgs)
}

// LoadPackages loads the packages in a directory.
func (s *Scanner) LoadPackages(dir string) ([]*packages.Package, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	conf := &packages.Config{
		Mode:  Mode,
		Dir:   dir,
		Tests: s.Tests,
	}
	return packages.Load(conf, "./...")
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
			err = errors.Join(err, LoadError{Err: e, Path: pkg.PkgPath})
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

	if len(pkgs) > 0 && pkgs[0].Module != nil {
		module := pkgs[0].Module

		if s.Deps {
			if err := s.scanDeps(module.GoMod); err != nil {
				return nil, errors.Wrap(err, "scanning dependencies")
			}
		}

		if err := s.doCheck(module.GoVersion); err != nil {
			return nil, errors.Wrap(err, "checking go declaration")
		}
	}

	return s.Result, nil
}

func (s *Scanner) doCheck(goVersion string) error {
	if !s.Check {
		return nil
	}
	var declared int
	parts := strings.SplitN(goVersion, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("go.mod has invalid go version %s", goVersion)
	}
	declared, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("go.mod has invalid go version %s", goVersion)
	}
	if s.Strict {
		if s.Result.Version() != declared {
			return VersionError{
				Computed: s.Result,
				Declared: declared,
			}
		}
	} else {
		if s.Result.Version() > declared {
			return VersionError{
				Computed: s.Result,
				Declared: declared,
			}
		}
	}
	return nil
}

func (s *Scanner) reset() {
	s.Result = intResult(0)
}

func (s *Scanner) scanPackage(pkg *packages.Package) error {
	_, err := s.scanPackageHelper(pkg.PkgPath, pkg.Fset, pkg.TypesInfo, pkg.Syntax)
	return err
}

func (s *Scanner) scanPackageHelper(pkgpath string, fset *token.FileSet, info *types.Info, files []*ast.File) (Result, error) {
	p := pkgScanner{
		s:       s,
		pkgpath: pkgpath,
		fset:    fset,
		info:    info,
		res:     intResult(0),
	}

	for _, file := range files {
		filename := p.fset.Position(file.Pos()).Filename
		isInCache, err := isCacheFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "checking whether %s is in GOCACHE", filename)
		}
		if isInCache {
			continue
		}
		if isMax, err := p.file(file); err != nil || isMax {
			return nil, errors.Wrapf(err, "scanning file %s", filename)
		}
	}

	return p.res, nil
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

func (s *Scanner) result(r Result) bool {
	if r.Version() > s.Result.Version() {
		s.Result = r
		s.verbosef("%s", r)
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

func isCacheFile(filename string) (bool, error) {
	cacheDir := os.Getenv("GOCACHE")
	if cacheDir == "" {
		var err error
		cacheDir, err = os.UserCacheDir()
		if err != nil {
			return false, errors.Wrap(err, "getting user cache directory")
		}
		if cacheDir == "" {
			return false, nil
		}
	}
	rel, err := filepath.Rel(cacheDir, filename)
	if err != nil {
		return false, errors.Wrapf(err, "computing relative path from %s to %s", cacheDir, filename)
	}
	return !strings.HasPrefix(rel, "../"), nil
}
