package mingo

import (
	"iter"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
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
		Name: "mingo",
		Doc:  "mingo finds the minimum version of Go that can build a module",
		Run:  s.runAnalyzer,
	}, nil
}

func (s *Scanner) runAnalyzer(pass *analysis.Pass) (any, error) {
	var (
		pkgpath = pass.Pkg.Path()
		fset    = pass.Fset
		info    = pass.TypesInfo
		files   = pass.Files
	)
	err := s.scanPackageHelper(pkgpath, fset, info, files)
	return nil, err
}

// GraphErrors returns a sequence of the non-nil errors found by walking a graph's tree of [checker.Action]s.
func GraphErrors(graph *checker.Graph) iter.Seq[error] {
	return func(yield func(error) bool) {
		for _, action := range graph.Roots {
			if !actionErrors(action, yield) {
				return
			}
		}
	}
}

func actionErrors(action *checker.Action, yield func(error) bool) bool {
	if action.Err != nil {
		if !yield(action.Err) {
			return false
		}
	}
	for _, dep := range action.Deps {
		if !actionErrors(dep, yield) {
			return false
		}
	}
	return true
}
