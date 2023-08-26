package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bobg/mingo"
)

func main() {
	var (
		dir = "."
		api string
	)
	flag.StringVar(&dir, "dir", dir, "directory to scan")
	flag.StringVar(&api, "api", "", "path to api directory")
	flag.Parse()

	h, err := mingo.ReadHist(api)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading API: %s\n", err)
		os.Exit(1)
	}

	result, err := mingo.ScanDir(dir, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
