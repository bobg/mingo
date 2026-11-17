package mingo

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobg/errors"
)

func TestScanDepsErrs(t *testing.T) {
	t.Run("NotExist", func(t *testing.T) {
		var s Scanner
		err := s.scanDeps("IDONOTEXIST")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("got error %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("BadGoMod", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go jump in a lake\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var s Scanner
		if err := s.scanDeps(dir); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("BadDepScan", func(t *testing.T) {
		s := Scanner{
			depScanner: errDepScanner{},
		}
		err := s.scanDeps("unused")
		if !errors.Is(err, depScannerErr) {
			t.Errorf("got error %v, want %v", err, depScannerErr)
		}
	})

	t.Run("BadGoVersion", func(t *testing.T) {
		for _, ver := range []string{"1", "1.x"} {
			t.Run(ver, func(t *testing.T) {
				s := Scanner{
					depScanner: mockDepScanner{
						{
							DepOnly: true,
							Module: &listModule{
								Path:      "foo.bar/baz",
								Version:   "v1.2.3",
								GoVersion: ver,
							},
						},
					},
				}
				if err := s.scanDeps("unused"); err == nil {
					t.Errorf("expected error for GoVersion %q", ver)
				}
			})
		}
	})
}

type errDepScanner struct{}

var depScannerErr = errors.New("scan failed")

func (errDepScanner) scan(string, ...string) ([]listItem, error) {
	return nil, depScannerErr
}
