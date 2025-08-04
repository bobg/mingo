package mingo

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v4/set"
	"github.com/bobg/go-generics/v4/slices"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Analyzer produces an [analysis.Analyzer] that can be used to scan packages.
// The result (which may depend on scanning multiple packages)
// is available in s.Result.
func (s *Scanner) Analyzer() (*analysis.Analyzer, error) {
	if err := s.ensureHistory(); err != nil {
		return nil, err
	}

	s.Result = intResult(0)

	return &analysis.Analyzer{
		Name:       "mingo",
		Doc:        "mingo finds the minimum version of Go that can build a module",
		Run:        s.runAnalyzer,
		ResultType: reflect.TypeFor[Result](),
	}, nil
}

func (s *Scanner) runAnalyzer(pass *analysis.Pass) (any, error) {
	var (
		pkgpath = pass.Pkg.Path()
		fset    = pass.Fset
		info    = pass.TypesInfo
		files   = pass.Files
	)

	result, err := s.scanPackageHelper(pkgpath, fset, info, files)
	return result, err
}

func (s *Scanner) AfterAnalyze(graph *checker.Graph, analyer *analysis.Analyzer) (Result, error) {
	var (
		result  Result = intResult(0)
		modules        = set.New[*packages.Module]()
		err     error
	)
	for _, action := range graph.Roots {
		actionResult, ok := action.Result.(Result)
		if !ok {
			continue
		}
		if actionResult.Version() > result.Version() {
			result = actionResult
		}
		err = errors.Join(err, action.Err)
		if mod := action.Package.Module; mod != nil {
			modules.Add(mod)
		}
	}

	var (
		modslice = modules.Slice()
		module   *packages.Module
	)
	switch len(modslice) {
	case 0:
	case 1:
		module = modslice[0]
	default:
		modpaths := slices.Map(modslice, func(m *packages.Module) string { return m.Path })
		err = errors.Join(err, fmt.Errorf("multiple modules: %s", strings.Join(modpaths, ", ")))
	}
	if err != nil {
		return nil, err
	}

	if module != nil {
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
