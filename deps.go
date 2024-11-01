package mingo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bobg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func (s *Scanner) scanDeps(gomodPath string) error {
	gomodBytes, err := os.ReadFile(gomodPath)
	if err != nil {
		return errors.Wrapf(err, "reading go.mod at %s", gomodPath)
	}

	f, err := modfile.ParseLax(gomodPath, gomodBytes, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing go.mod at %s", gomodPath)
	}

	for _, r := range f.Require {
		if r.Indirect && !s.Indirect {
			continue
		}
		if err := s.scanDep(r.Mod); err != nil {
			return errors.Wrapf(err, "scanning dep %s", r.Mod.Path)
		}
	}

	return nil
}

type modDownload struct {
	GoMod string
}

type depScanner interface {
	scan(modpath, version string) (modDownload, error)
}

type realDepScanner struct{}

func (s realDepScanner) scan(modpath, version string) (modDownload, error) {
	var result modDownload

	cmd := exec.Command("go", "mod", "download", "-json", modpath+"@"+version)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, errors.Wrapf(err, "creating stdout pipe for download of %s", modpath)
	}

	if err := cmd.Start(); err != nil {
		return result, errors.Wrapf(err, "starting download of %s", modpath)
	}
	defer cmd.Wait()

	if err := json.NewDecoder(stdout).Decode(&result); err != nil {
		return result, errors.Wrapf(err, "decoding download of %s", modpath)
	}

	err = cmd.Wait()
	return result, errors.Wrapf(err, "waiting for download of %s", modpath)
}

func (s *Scanner) scanDep(mv module.Version) error {
	scanner := s.depScanner
	if scanner == nil {
		scanner = realDepScanner{}
	}

	download, err := scanner.scan(mv.Path, mv.Version)
	if err != nil {
		return errors.Wrapf(err, "scanning %s@%s", mv.Path, mv.Version)
	}

	gomodBytes, err := os.ReadFile(download.GoMod)
	if err != nil {
		return errors.Wrapf(err, "reading go.mod of %s", mv.Path)
	}
	parsed, err := modfile.ParseLax(download.GoMod, gomodBytes, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing go.mod of %s", mv.Path)
	}
	if parsed.Go == nil {
		// Probably a pre-Go 1.11 module.
		return nil
	}
	parts := strings.SplitN(parsed.Go.Version, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("go.mod of %s has invalid go version %s", mv.Path, parsed.Go.Version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("go.mod of %s has invalid go version %s", mv.Path, parsed.Go.Version)
	}

	s.result(depResult{
		version:    minor,
		modpath:    mv.Path,
		modversion: mv.Version,
	})
	return nil
}

type depResult struct {
	version             int
	modpath, modversion string
}

func (r depResult) Version() int { return r.version }
func (r depResult) String() string {
	return fmt.Sprintf("%s@%s declares Go version 1.%d", r.modpath, r.modversion, r.version)
}
