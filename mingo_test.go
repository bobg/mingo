package mingo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bobg/errors"
)

func TestLangChecks(t *testing.T) {
	entries, err := os.ReadDir("_testdata")
	if err != nil {
		t.Fatal(err)
	}

	var versions []int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		min, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // sic
		}
		versions = append(versions, min)
	}

	sort.Ints(versions)

	var (
		earlierCode    string
		earlierImports []string
	)

	for _, min := range versions {
		minstr := strconv.Itoa(min)

		var (
			thisVersionCode    string
			thisVersionImports []string
		)

		t.Run(minstr, func(t *testing.T) {
			entries, err := os.ReadDir("_testdata/" + minstr)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if !strings.HasSuffix(entry.Name(), ".go") {
					continue
				}
				t.Run(strings.TrimSuffix(entry.Name(), ".go"), func(t *testing.T) {
					filename := "_testdata/" + minstr + "/" + entry.Name()

					code, imports, err := readGoFile(filename)
					if err != nil {
						t.Fatal(err)
					}

					tmpdir, err := os.MkdirTemp("", "mingo")
					if err != nil {
						t.Fatal(err)
					}
					var rmdir bool
					defer func() {
						if rmdir {
							t.Logf("xxx removing %s", tmpdir)
							os.RemoveAll(tmpdir)
						}
					}()

					gomod := filepath.Join(tmpdir, "go.mod")
					if err := os.WriteFile(gomod, []byte("module foo\ngo 1.22.0\n"), 0644); err != nil {
						t.Fatal(err)
					}

					tmpfile, err := os.Create(filepath.Join(tmpdir, "foo.go"))
					if err != nil {
						t.Fatal(err)
					}

					t.Log(tmpfile.Name())

					fmt.Fprint(tmpfile, "package foo\n\n")

					combinedImports := append(earlierImports, imports...)
					sort.Strings(combinedImports)
					combinedImports = slices.Compact(combinedImports)

					if len(combinedImports) > 0 {
						fmt.Fprint(tmpfile, "import (\n")
						for _, imp := range combinedImports {
							fmt.Fprintf(tmpfile, "\t%q\n", imp)
						}
						fmt.Fprint(tmpfile, ")\n\n")
					}

					fmt.Fprint(tmpfile, earlierCode)
					if _, err := fmt.Fprint(tmpfile, code); err != nil {
						t.Fatal(err)
					}
					if err = tmpfile.Close(); err != nil {
						t.Fatal(err)
					}

					t.Run("ScanDir", func(t *testing.T) {
						s := Scanner{Verbose: testing.Verbose()}
						res, err := s.ScanDir(tmpdir)
						if err != nil {
							t.Fatal(err)
						}
						if res.Version() != min {
							t.Errorf("got %d, want %d", res.Version(), min)
						} else {
							rmdir = true
						}
					})

					// TODO: check the same thing using Scanner.Analyzer
					// (when the API in https://github.com/golang/go/issues/61324 lands)

					thisVersionCode += code
					thisVersionImports = append(thisVersionImports, imports...)
				})
			}
		})

		earlierCode += thisVersionCode

		earlierImports = append(earlierImports, thisVersionImports...)
		sort.Strings(earlierImports)
		earlierImports = slices.Compact(earlierImports)
	}
}

func readGoFile(filename string) (string, []string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", nil, errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	var (
		sc        = bufio.NewScanner(f)
		inImports bool
		code      strings.Builder
		imports   []string
	)
	for sc.Scan() {
		line := sc.Text()
		if inImports {
			if strings.HasPrefix(line, ")") {
				inImports = false
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			imp := fields[0] // No import alias allowed
			imports = append(imports, imp)
			continue
		}
		if strings.HasPrefix(line, "import (") {
			inImports = true
			continue
		}
		if strings.HasPrefix(line, "import ") {
			fields := strings.Fields(line)
			imp := fields[1] // No import alias allowed
			imports = append(imports, imp)
			continue
		}
		if strings.HasPrefix(line, "package ") {
			continue
		}
		fmt.Fprintln(&code, line)
	}
	return code.String(), imports, errors.Wrapf(sc.Err(), "scanning %s", filename)
}
