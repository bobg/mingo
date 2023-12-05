package mingo

import (
	"fmt"
	"testing"
)

func TestScanDeps(t *testing.T) {
	s := Scanner{
		Deps: true,
		depScanner: mockDepScanner{
			"foo.bar/baz@v1.2.3": "_testdata/foobar.go.mod",
		},
		Result: intResult(0),
	}
	if err := s.ensureHistory(); err != nil {
		t.Fatal(err)
	}
	if err := s.scanDeps("_testdata/go.mod"); err != nil {
		t.Fatal(err)
	}
	if s.Result == nil {
		t.Fatal("nil result")
	}
	if s.Result.Version() != 16 {
		t.Errorf("got %d, want 16", s.Result.Version())
	}
}

type mockDepScanner map[string]string

func (m mockDepScanner) scan(modpath, version string) (modDownload, error) {
	str := modpath + "@" + version
	v, ok := m[str]
	if !ok {
		return modDownload{}, fmt.Errorf("no such module %s", str)
	}
	return modDownload{GoMod: v}, nil
}
