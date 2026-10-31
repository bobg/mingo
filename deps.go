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

		mod := r.Mod
		if len(f.Replace) > 0 {
			if rep := findReplace(f.Replace, mod); rep != nil {
				if modfile.IsDirectoryPath(rep.New.Path) {
					if err := s.scanLocalDep(mod, modDir, rep.New.Path); err != nil {
						return errors.Wrapf(err, "scanning dep %s", r.Mod.Path)
					}
					continue
				}

				mod = rep.New
				if mod.Version == "" {
					mod.Version = r.Mod.Version
				}
			}
		}

		if err := s.scanDep(mod); err != nil {
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

func (s *Scanner) scanDep(mv module.Version) error {
	scanner := s.depScanner
	if scanner == nil {
		scanner = realDepScanner{}
	}

	download, err := scanner.scan(mv.Path, mv.Version)
	if err != nil {
		return errors.Wrapf(err, "scanning %s@%s", mv.Path, mv.Version)
	}

	return s.recordDepGoMod(download.GoMod, mv.Path, mv.Version)
}

func (s *Scanner) scanLocalDep(orig module.Version, modDir, localPath string) error {
	dir := localPath
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(modDir, localPath)
	}

	return s.recordDepGoMod(filepath.Join(dir, "go.mod"), orig.Path, orig.Version)
}

func (s *Scanner) recordDepGoMod(gomodPath, modpath, modversion string) error {
	gomodBytes, err := os.ReadFile(gomodPath)
	if err != nil {
		return errors.Wrapf(err, "reading go.mod of %s", modpath)
	}
	parsed, err := modfile.ParseLax(gomodPath, gomodBytes, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing go.mod of %s", modpath)
	}
	if parsed.Go == nil {
		// Probably a pre-Go 1.11 module.
		return nil
	}
	parts := strings.SplitN(parsed.Go.Version, ".", 3)
	if len(parts) < 2 {
		return fmt.Errorf("go.mod of %s has invalid go version %s", modpath, parsed.Go.Version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("go.mod of %s has invalid go version %s", modpath, parsed.Go.Version)
	}

	s.result(depResult{
		version:    minor,
		modpath:    modpath,
		modversion: modversion,
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
