package mingo

import (
	"io/fs"
	"testing"

	"github.com/bobg/errors"
)

func TestScanDepsErrs(t *testing.T) {
	s := Scanner{
		depScanner: errDepScanner{},
	}

	t.Run("NotExist", func(t *testing.T) {
		err := s.scanDeps("IDONOTEXIST")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("got error %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("BadGoMod", func(t *testing.T) {
		if err := s.scanDeps("_testdata/go.mod.bad"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("BadDepScan", func(t *testing.T) {
		err := s.scanDeps("_testdata/go.mod")
		if !errors.Is(err, depScannerErr) {
			t.Errorf("got error %v, want %v", err, depScannerErr)
		}
	})
}

type errDepScanner struct{}

var depScannerErr = errors.New("scan failed")

func (errDepScanner) scan(modpath, version string) (modDownload, error) {
	return modDownload{}, depScannerErr
}
