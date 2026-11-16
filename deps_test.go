package mingo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDeps(t *testing.T) {
	items := []listItem{
		{
			DepOnly: false,
			Module: &listModule{
				Path:      "x.y/z",
				Main:      true,
				GoVersion: "1.21.0",
			},
		},
		{
			DepOnly: true,
			Module:  nil, // stdlib package
		},
		{
			DepOnly: true,
			Module: &listModule{
				Path:      "pre.go111/mod",
				Version:   "v0.1.0",
				GoVersion: "",
			},
		},
		{
			DepOnly: true,
			Module: &listModule{
				Path:      "foo.bar/baz",
				Version:   "v1.2.3",
				GoVersion: "1.16",
			},
		},
		{
			DepOnly: true,
			Module: &listModule{
				Path:      "quux.plugh/xyzzy/v4",
				Version:   "v4.5.6",
				Indirect:  true,
				GoVersion: "1.20.0",
			},
		},
	}

	t.Run("direct only", func(t *testing.T) {
		s := Scanner{
			Deps:       true,
			depScanner: mockDepScanner(items),
			Result:     intResult(0),
		}
		if err := s.ensureHistory(); err != nil {
			t.Fatal(err)
		}
		if err := s.scanDeps("unused"); err != nil {
			t.Fatal(err)
		}
		if s.Result == nil {
			t.Fatal("nil result")
		}
		if s.Result.Version() != 16 {
			t.Errorf("got %d, want 16", s.Result.Version())
		}
	})

	t.Run("with indirect", func(t *testing.T) {
		s := Scanner{
			Deps:       true,
			Indirect:   true,
			depScanner: mockDepScanner(items),
			Result:     intResult(0),
		}
		if err := s.ensureHistory(); err != nil {
			t.Fatal(err)
		}
		if err := s.scanDeps("unused"); err != nil {
			t.Fatal(err)
		}
		if s.Result == nil {
			t.Fatal("nil result")
		}
		if s.Result.Version() != 20 {
			t.Errorf("got %d, want 20", s.Result.Version())
		}
	})
}

func TestScanDepsIntegration(t *testing.T) {
	t.Run("replace", func(t *testing.T) {
		dir := t.TempDir()
		mainDir := filepath.Join(dir, "main")
		directDir := filepath.Join(dir, "direct")
		indirectDir := filepath.Join(dir, "indirect")

		for _, d := range []string{mainDir, directDir, indirectDir} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}

		writeTestFile(t, filepath.Join(mainDir, "go.mod"), `module example.com/main

go 1.25

require (
	example.com/direct v1.0.0
	example.com/indirect v1.0.0 // indirect
)

replace (
	example.com/direct => ../direct
	example.com/indirect => ../indirect
)
`)
		writeTestFile(t, filepath.Join(mainDir, "main.go"), `package main

import _ "example.com/direct"

func main() {
	return
}
`)

		writeTestFile(t, filepath.Join(directDir, "go.mod"), `module example.com/direct

go 1.20

require example.com/indirect v1.0.0
`)
		writeTestFile(t, filepath.Join(directDir, "direct.go"), `package direct

import _ "example.com/indirect"
`)

		writeTestFile(t, filepath.Join(indirectDir, "go.mod"), `module example.com/indirect

go 1.24
`)
		writeTestFile(t, filepath.Join(indirectDir, "indirect.go"), `package indirect
`)

		t.Run("direct only", func(t *testing.T) {
			s := Scanner{Deps: true}
			res, err := s.ScanDir(mainDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 20 {
				t.Errorf("got %d, want 20 (%s)", got, res)
			}
		})

		t.Run("with indirect", func(t *testing.T) {
			s := Scanner{Deps: true, Indirect: true}
			res, err := s.ScanDir(mainDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 24 {
				t.Errorf("got %d, want 24 (%s)", got, res)
			}
		})
	})

	t.Run("gowork", func(t *testing.T) {
		dir := t.TempDir()
		mainDir := filepath.Join(dir, "main")
		depDir := filepath.Join(dir, "dep")

		for _, d := range []string{mainDir, depDir} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}

		writeTestFile(t, filepath.Join(dir, "go.work"), `go 1.25

use (
	./main
	./dep
)
`)
		writeTestFile(t, filepath.Join(mainDir, "go.mod"), `module example.com/main

go 1.25

require example.com/dep v1.0.0
`)
		writeTestFile(t, filepath.Join(mainDir, "main.go"), `package main

import _ "example.com/dep"

func main() {
	return
}
`)

		writeTestFile(t, filepath.Join(depDir, "go.mod"), `module example.com/dep

go 1.24
`)
		writeTestFile(t, filepath.Join(depDir, "dep.go"), `package dep
`)

		s := Scanner{Deps: true}
		res, err := s.ScanDir(mainDir)
		if err != nil {
			t.Fatal(err)
		}
		if got := res.Version(); got != 24 {
			t.Errorf("got %d, want 24 (%s)", got, res)
		}
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type mockDepScanner []listItem

func (m mockDepScanner) scan(_ string) ([]listItem, error) {
	return m, nil
}
