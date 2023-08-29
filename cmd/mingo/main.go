package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bobg/mingo"
)

func main() {
	var (
		dir     = "."
		api     string
		verbose bool
		deps    bool
		tests   bool
	)
	flag.StringVar(&dir, "dir", dir, "directory to scan")
	flag.StringVar(&api, "api", "", "path to api directory")
	flag.BoolVar(&verbose, "v", false, "be verbose")
	flag.BoolVar(&deps, "deps", false, "include dependencies")
	flag.BoolVar(&tests, "tests", false, "include tests")
	flag.Parse()

	s := mingo.Scanner{
		HistDir: api,
		Verbose: verbose,
		Deps:    deps,
		Tests:   tests,
	}

	result, err := s.ScanDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(result.String())
}
