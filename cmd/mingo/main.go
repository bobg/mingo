// Command mingo scans the packages in a Go module to determine the lowest-numbered version of Go that can build it.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bobg/mingo"
)

func main() {
	var (
		api      string
		verbose  bool
		deps     bool
		indirect bool
		tests    bool
		check    bool
	)
	flag.StringVar(&api, "api", "", "path to api directory")
	flag.BoolVar(&verbose, "v", false, "be verbose")
	flag.BoolVar(&deps, "deps", false, "include dependencies")
	flag.BoolVar(&indirect, "indirect", false, "with -deps, include indirect dependencies")
	flag.BoolVar(&tests, "tests", false, "include tests")
	flag.BoolVar(&check, "check", false, "produce an error if module declares wrong version in go.mod")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	s := mingo.Scanner{
		HistDir:  api,
		Verbose:  verbose,
		Deps:     deps,
		Indirect: indirect,
		Tests:    tests,
		Check:    check,
	}

	result, err := s.ScanDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %s\n", err)
		os.Exit(1)
	}

	if !check {
		fmt.Println(result.Version())
	}
}
