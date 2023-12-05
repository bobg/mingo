package mingo

import "golang.org/x/tools/go/analysis"

// Analyzer produces an [analysis.Analyzer] that can be used to scan packages.
// The result (which may depend on scanning multiple packages)
// is available in s.Result.
func (s *Scanner) Analyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "mingo",
		Doc:  "mingo finds the minimum version of Go that can build a module",
		Run:  s.runAnalyzer,
	}
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
