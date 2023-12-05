package mingo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bobg/gocheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
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

	var earlierCode string

	for _, min := range versions {
		minstr := strconv.Itoa(min)

		t.Run(minstr, func(t *testing.T) {
			var code string

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
					content, err := os.ReadFile("_testdata/" + minstr + "/" + entry.Name())
					if err != nil {
						t.Fatal(err)
					}
					code += string(content)

					tmpdir, err := os.MkdirTemp("", "mingo")
					if err != nil {
						t.Fatal(err)
					}
					defer os.RemoveAll(tmpdir)

					gomod := filepath.Join(tmpdir, "go.mod")
					if err := os.WriteFile(gomod, []byte("module foo\ngo 1.21.0\n"), 0644); err != nil {
						t.Fatal(err)
					}

					tmpfile, err := os.Create(filepath.Join(tmpdir, "foo.go"))
					if err != nil {
						t.Fatal(err)
					}

					t.Log(tmpfile.Name())

					fmt.Fprint(tmpfile, "package foo\n\n")
					fmt.Fprint(tmpfile, earlierCode)
					if _, err := tmpfile.Write(content); err != nil {
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
						}
					})

					t.Run("analyzer", func(t *testing.T) {
						s := Scanner{Verbose: testing.Verbose()}
						a, err := s.Analyzer()
						if err != nil {
							t.Fatal(err)
						}
						conf := &packages.Config{
							Mode:  Mode,
							Dir:   tmpdir,
							Tests: s.Tests,
						}
						pkgs, err := packages.Load(conf, "./...")
						if err != nil {
							t.Fatal(err)
						}

						c := gocheck.Controller{Verbose: testing.Verbose()}
						if _, err = c.Run(pkgs, []*analysis.Analyzer{a}); err != nil {
							t.Fatal(err)
						}

						if s.Result.Version() != min {
							t.Errorf("got %d, want %d", s.Result.Version(), min)
						}
					})
				})
			}

			earlierCode += code
		})
	}
}
