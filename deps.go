package mingo

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

func ScanDeps(ctx context.Context, f *modfile.File) (int, error) {
	result := MinGoMinorVersion
	for _, r := range f.Require {
		depResult, err := ScanDep(ctx, r.Mod.Path, r.Mod.Version)
		if err != nil {
			return 0, errors.Wrapf(err, "scanning dep %s", r.Mod.Path)
		}
		result = max(result, depResult)
	}
	return result, nil
}

type modDownload struct {
	Path, Version, GoMod string
}

func ScanDep(ctx context.Context, modpath, version string) (int, error) {
	cmd := exec.CommandContext(ctx, "go", "mod", "download", "-json", modpath+"@"+version)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, errors.Wrapf(err, "creating stdout pipe for download of %s", modpath)
	}
	if err := cmd.Start(); err != nil {
		return 0, errors.Wrapf(err, "starting download of %s", modpath)
	}
	defer cmd.Wait()

	var download modDownload
	if err := json.NewDecoder(stdout).Decode(&download); err != nil {
		return 0, errors.Wrapf(err, "decoding download of %s", modpath)
	}

	if err := cmd.Wait(); err != nil {
		return 0, errors.Wrapf(err, "waiting for download of %s", modpath)
	}

	gomodBytes, err := os.ReadFile(download.GoMod)
	if err != nil {
		return 0, errors.Wrapf(err, "reading go.mod of %s", modpath)
	}
	parsed, err := modfile.ParseLax(download.GoMod, gomodBytes, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "parsing go.mod of %s", modpath)
	}
	if parsed.Go == nil {
		return 0, errors.Errorf("go.mod of %s has no go version", modpath)
	}
	parts := strings.SplitN(parsed.Go.Version, ".", 3)
	if len(parts) < 2 {
		return 0, errors.Errorf("go.mod of %s has invalid go version %s", modpath, parsed.Go.Version)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, errors.Errorf("go.mod of %s has invalid go version %s", modpath, parsed.Go.Version)
	}
	return max(MinGoMinorVersion, minor), nil
}
