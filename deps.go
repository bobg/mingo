package mingo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	f, err := modfile.Parse(gomodPath, gomodBytes, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing go.mod at %s", gomodPath)
	}
	modDir := filepath.Dir(gomodPath)

	for _, r := range f.Require {
		if r.Indirect && !s.Indirect {
			continue
		}

		if err := s.scanDep(s.resolveDep(r.Mod, modDir, f.Replace)); err != nil {
			return errors.Wrapf(err, "scanning dep %s", r.Mod.Path)
		}
	}

	return nil
}

func findReplace(replaces []*modfile.Replace, mv module.Version) *modfile.Replace {
	var rep *modfile.Replace
	for _, r := range replaces {
		if r.Old.Path != mv.Path {
			continue
		}
		if r.Old.Version == "" {
			rep = r
			continue
		}
		if r.Old.Version == mv.Version {
			return r
		}
	}
	return rep
}

type modDownload struct {
	GoMod string
}

type depScanner interface {
	scan(modpath, version string) (modDownload, error)
}

type localDepScanner struct {
	gomod string
}

func (l localDepScanner) scan(_, _ string) (modDownload, error) {
	return modDownload{GoMod: l.gomod}, nil
}

type realDepScanner struct{}

func (s realDepScanner) scan(modpath, version string) (_ modDownload, err error) {
	var result modDownload

	cmd := exec.Command("go", "mod", "download", "-json", modpath+"@"+version)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, errors.Wrapf(err, "creating stdout pipe for download of %s", modpath)
	}

	if err := cmd.Start(); err != nil {
		return result, errors.Wrapf(err, "starting download of %s", modpath)
	}
	defer func() {
		err2 := cmd.Wait()
		err = errors.Join(err, errors.Wrapf(err2, "waiting for download of %s", modpath))
	}()

	err = json.NewDecoder(stdout).Decode(&result)
	return result, errors.Wrapf(err, "decoding download of %s", modpath)
}

func (s *Scanner) resolveDep(mv module.Version, modDir string, replaces []*modfile.Replace) (module.Version, depScanner) {
	if s.IgnoreReplace || len(replaces) == 0 {
		return mv, s.depScanner
	}

	rep := findReplace(replaces, mv)
	if rep == nil {
		return mv, s.depScanner
	}

	if modfile.IsDirectoryPath(rep.New.Path) {
		dir := rep.New.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(modDir, dir)
		}

		return mv, localDepScanner{gomod: filepath.Join(dir, "go.mod")}
	}

	eff := rep.New
	if eff.Version == "" {
		eff.Version = mv.Version
	}

	return eff, s.depScanner
}

func (s *Scanner) scanDep(mv module.Version, scanner depScanner) error {
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
