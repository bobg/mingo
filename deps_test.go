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
		result: intResult(0),
	}
	if err := s.ensureHistory(); err != nil {
		t.Fatal(err)
	}
	if err := s.scanDeps("_testdata/go.mod"); err != nil {
		t.Fatal(err)
	}
	if s.result == nil {
		t.Fatal("nil result")
	}
	if s.result.Version() != 16 {
		t.Errorf("got %d, want 16", s.result.Version())
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
