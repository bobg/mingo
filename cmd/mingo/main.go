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
		api, deps                     string
		check, strict, tests, verbose bool
	)
	flag.StringVar(&api, "api", "", "path to api directory")
	flag.StringVar(&deps, "deps", "all", "which dependencies to scan (all, direct, none)")
	flag.BoolVar(&check, "check", false, "check that go.mod declares the right version of Go or higher")
	flag.BoolVar(&strict, "strict", false, "check that go.mod declares exactly the right version of Go")
	flag.BoolVar(&tests, "tests", false, "include tests")
	flag.BoolVar(&verbose, "v", false, "be verbose")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	if strict {
		check = true
	}

	switch deps {
	case "all", "direct", "none":
		// ok, do nothing
	default:
		fmt.Fprintf(os.Stderr, "Invalid value for -deps: %s (should be all, direct, or none)\n", deps)
		os.Exit(1)
	}

	s := mingo.Scanner{
		HistDir:  api,
		Verbose:  verbose,
		Deps:     deps != "none",
		Indirect: deps == "all",
		Tests:    tests,
		Check:    check,
		Strict:   strict,
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
