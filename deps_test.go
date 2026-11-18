package mingo

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestScanDeps(t *testing.T) {
	items := []listItem{{
		DepOnly: false,
		Module: &listModule{
			Path:      "x.y/z",
			Main:      true,
			GoVersion: "1.21.0",
		},
	}, {
		DepOnly: true,
		Module:  nil, // stdlib package
	}, {
		DepOnly: true,
		Module: &listModule{
			Path:      "pre.go111/mod",
			Version:   "v0.1.0",
			GoVersion: "",
		},
	}, {
		DepOnly: true,
		Module: &listModule{
			Path:      "foo.bar/baz",
			Version:   "v1.2.3",
			GoVersion: "1.16",
		},
	}, {
		DepOnly: true,
		Module: &listModule{
			Path:      "quux.plugh/xyzzy/v4",
			Version:   "v4.5.6",
			Indirect:  true,
			GoVersion: "1.20.0",
		},
	}}

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
		const mainDir = "testdata/deps/replace/main"

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

		t.Run("with tests", func(t *testing.T) {
			s := Scanner{Deps: true, Tests: true}
			res, err := s.ScanDir(mainDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 25 {
				t.Errorf("got %d, want 25 (%s)", got, res)
			}
		})

		t.Run("with build flags", func(t *testing.T) {
			s := Scanner{Deps: true, BuildFlags: []string{"-tags=customtag"}}
			res, err := s.ScanDir(mainDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 26 {
				t.Errorf("got %d, want 26 (%s)", got, res)
			}
		})
	})

	t.Run("gowork", func(t *testing.T) {
		s := Scanner{Deps: true}
		res, err := s.ScanDir("testdata/deps/gowork/main")
		if err != nil {
			t.Fatal(err)
		}
		if got := res.Version(); got != 24 {
			t.Errorf("got %d, want 24 (%s)", got, res)
		}
	})

	t.Run("sibling", func(t *testing.T) {
		const (
			mainDir = "testdata/deps/sibling/main"
			goodDir = mainDir + "/good"
		)

		t.Run("ScanDir", func(t *testing.T) {
			s := Scanner{Deps: true}
			res, err := s.ScanDir(goodDir)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 24 {
				t.Errorf("got %d, want 24 (%s)", got, res)
			}
		})

		t.Run("ScanPackages", func(t *testing.T) {
			conf := &packages.Config{
				Mode: Mode,
				Dir:  mainDir,
			}
			pkgs, err := packages.Load(conf, "./good")
			if err != nil {
				t.Fatal(err)
			}
			s := Scanner{Deps: true}
			res, err := s.ScanPackages(pkgs)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Version(); got != 24 {
				t.Errorf("got %d, want 24 (%s)", got, res)
			}
		})
	})
}

type mockDepScanner []listItem

func (m mockDepScanner) scan(_ string, _ ...string) ([]listItem, error) {
	return m, nil
}
