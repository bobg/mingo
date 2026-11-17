package mingo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v4/set"
)

func (s *Scanner) scanDeps(dir string) error {
	scanner := s.depScanner
	if scanner == nil {
		scanner = realDepScanner{tests: s.Tests}
	}

	items, err := scanner.scan(dir)
	if err != nil {
		return err
	}

	seen := set.New[string]()
	for _, item := range items {
		if !item.DepOnly && item.Module != nil {
			seen.Add(item.Module.Path)
		}
	}

	for _, item := range items {
		if !item.DepOnly || item.Module == nil {
			continue
		}
		mod := item.Module
		if mod.Indirect && !s.Indirect {
			continue
		}
		if seen.Has(mod.Path) {
			continue
		}
		seen.Add(mod.Path)

		if mod.GoVersion == "" {
			// Probably a pre-Go 1.11 module.
			continue
		}
		parts := strings.SplitN(mod.GoVersion, ".", 3)
		if len(parts) < 2 {
			return fmt.Errorf("go.mod of %s has invalid go version %s", mod.Path, mod.GoVersion)
		}
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("go.mod of %s has invalid go version %s", mod.Path, mod.GoVersion)
		}

		s.result(depResult{
			version:    minor,
			modpath:    mod.Path,
			modversion: mod.Version,
		})
	}

	return nil
}

type listItem struct {
	DepOnly bool        `json:"DepOnly"`
	Module  *listModule `json:"Module"`
}

type listModule struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Main      bool   `json:"Main"`
	Indirect  bool   `json:"Indirect"`
	GoVersion string `json:"GoVersion"`
}

type depScanner interface {
	scan(dir string) ([]listItem, error)
}

type realDepScanner struct{
	tests bool
}

func (s realDepScanner) scan(dir string) (_ []listItem, err error) {
	args := []string{"list", "-deps", "-json"}
	if s.tests {
		args = append(args, "-test")
	}
	args = append(args, "./...")
	cmd := exec.Command("go", args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "creating stdout pipe for go list")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting go list")
	}
	defer func() {
		err2 := cmd.Wait()
		if err2 != nil {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				err2 = fmt.Errorf("%w: %s", err2, msg)
			}
			err = errors.Join(err, errors.Wrap(err2, "waiting for go list"))
		}
	}()

	var items []listItem
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var item listItem
		if err := dec.Decode(&item); err != nil {
			return nil, errors.Wrap(err, "decoding go list output")
		}
		items = append(items, item)
	}

	return items, nil
}

type depResult struct {
	version             int
	modpath, modversion string
}

func (r depResult) Version() int { return r.version }
func (r depResult) String() string {
	if r.modversion == "" {
		return fmt.Sprintf("%s declares Go version 1.%d", r.modpath, r.version)
	}
	return fmt.Sprintf("%s@%s declares Go version 1.%d", r.modpath, r.modversion, r.version)
}
