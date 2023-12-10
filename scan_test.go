package mingo

import (
	"testing"

	"github.com/bobg/errors"
	"golang.org/x/tools/go/packages"
)

func TestScanErrors(t *testing.T) {
	var s Scanner

	pkgerr1 := packages.Error{
		Pos: "file1.go:1:1",
		Msg: "err1",
	}
	pkgerr2 := packages.Error{
		Pos: "file2.go:2:2",
		Msg: "err2",
	}
	pkgerr3 := packages.Error{
		Pos: "file3.go:3:3",
		Msg: "err3",
	}

	errpkgs := []*packages.Package{{
		PkgPath: "foo1.bar1/baz1",
		Errors:  []packages.Error{pkgerr1, pkgerr2},
	}, {
		PkgPath: "foo2.bar2/baz2",
		Errors:  []packages.Error{pkgerr2, pkgerr3},
	}}

	t.Run("pkgerrs", func(t *testing.T) {
		_, err := s.ScanPackages(errpkgs)
		if err == nil {
			t.Fatalf("got nil, want error")
		}
		if !errors.Is(err, pkgerr1) {
			t.Errorf("err is not pkgerr1")
		}
		if !errors.Is(err, pkgerr2) {
			t.Errorf("err is not pkgerr2")
		}
		if !errors.Is(err, pkgerr3) {
			t.Errorf("err is not pkgerr3")
		}
	})

	t.Run("nomod", func(t *testing.T) {
		_, err := s.ScanPackages([]*packages.Package{{}})
		if err == nil {
			t.Fatalf("got nil, want error")
		}
	})

	t.Run("multimod", func(t *testing.T) {
		_, err := s.ScanPackages([]*packages.Package{{
			Module: &packages.Module{Path: "foo1.bar1/baz1"},
		}, {
			Module: &packages.Module{Path: "foo2.bar2/baz2"},
		}})
		if err == nil {
			t.Fatalf("got nil, want error")
		}
	})
}
