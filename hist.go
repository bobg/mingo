package mingo

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v3/slices"
)

// Type history is the history of the Go stdlib,
// as parsed from the files in $GOROOT/api.
// It maps a Go stdlib package names to their individual histories.
type history struct {
	pkgs map[string]pkgHistory // maps package paths to package histories
	max  int                   // the highest minor version of Go seen
}

func (h history) lookup(pkgpath, id, typ string) int {
	if p, ok := h.pkgs[pkgpath]; ok {
		return p.lookup(id, typ)
	}
	return 0
}

// Type pkgHistory is the history of a single Go stdlib package.
type pkgHistory struct {
	// Maps top-level identifiers to the minor version of Go at which they were first introduced.
	ids map[string]int

	// Maps type "members" -
	// method names and struct fields -
	// to the minor version of Go at which they were first introduced.
	// First map key is the type name within its package;
	// second map key is the identifier in the type's scope.
	types map[string]map[string]int
}

func (p pkgHistory) lookup(id, typ string) int {
	if typ == "" {
		return p.ids[id]
	}
	if t, ok := p.types[typ]; ok {
		return t[id]
	}
	return 0
}

//go:embed api
var apiDir embed.FS

// Function readHist reads the history of the Go stdlib
// from the sequence of go1.*.txt files in the given directory.
// The default directory,
// which you get if dir is "",
// is $GOROOT/api,
// or a builtin snapshot of that directory made at build time
// if that can't be found.
func readHist(dir string) (*history, error) {
	var fsys fs.FS

	if dir == "" {
		dir = filepath.Join(goroot(), "api")
		_, err := os.Stat(dir)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			fsys = apiDir
			dir = "api"
		case err != nil:
			return nil, errors.Wrapf(err, "statting %s", dir)
		default:
			fsys = os.DirFS(dir)
			dir = "."
		}
	} else {
		fsys = os.DirFS(dir)
		dir = "."
	}

	return readHistFS(fsys, dir)
}

var apifilenameRegex = regexp.MustCompile(`^go1\.(\d+)\.txt$`)

// Function readHistFS reads the history of the Go stdlib
// from the sequence of go1.*.txt files
// in the given directory within the given filesystem.
func readHistFS(fsys fs.FS, dir string) (*history, error) {
	h := &history{
		pkgs: make(map[string]pkgHistory),
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		base := entry.Name()
		m := apifilenameRegex.FindStringSubmatch(base)
		if len(m) == 0 {
			continue
		}
		v, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, errors.Wrapf(err, "parsing version from filename %s", base)
		}
		if v > h.max {
			h.max = v
		}
		if err = readHistVersion(h, fsys, filepath.Join(dir, base), v); err != nil {
			return nil, errors.Wrapf(err, "reading version %d history", v)
		}
	}

	return h, nil
}

func goroot() string {
	if g := os.Getenv("GOROOT"); g != "" {
		return g
	}
	return runtime.GOROOT()
}

func readHistVersion(h *history, fsys fs.FS, filename string, v int) error {
	f, err := fsys.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "//deprecated") {
			continue
		}
		if m := constRegex.FindStringSubmatch(line); len(m) > 0 {
			match2(h, m[1], m[3], v)
		} else if m := fieldRegex.FindStringSubmatch(line); len(m) > 0 {
			match3(h, m[1], m[3], m[5], v)
		} else if m := funcRegex.FindStringSubmatch(line); len(m) > 0 {
			match2(h, m[1], m[3], v)
		} else if m := methodRegex.FindStringSubmatch(line); len(m) > 0 {
			match3(h, m[1], m[3], m[5], v)
		} else if m := iface1Regex.FindStringSubmatch(line); len(m) > 0 {
			pkgpath, id, methods := m[1], m[3], m[4]
			match2(h, pkgpath, id, v)
			methodIDs := slices.Map(strings.Split(methods, ","), strings.TrimSpace)
			for _, methodID := range methodIDs {
				match3(h, pkgpath, id, methodID, v)
			}
		} else if m := iface2Regex.FindStringSubmatch(line); len(m) > 0 {
			match2(h, m[1], m[3], v)
		} else if m := iface3Regex.FindStringSubmatch(line); len(m) > 0 {
			pkgpath, id, method := m[1], m[3], m[4]
			match2(h, pkgpath, id, v)
			match3(h, pkgpath, id, method, v)
		} else if m := varRegex.FindStringSubmatch(line); len(m) > 0 {
			match2(h, m[1], m[3], v)
		} else if m := typeRegex.FindStringSubmatch(line); len(m) > 0 { // This one must come last.
			match2(h, m[1], m[3], v)
		} else {
			return fmt.Errorf("unrecognized line %s", line)
		}
	}
	return errors.Wrapf(sc.Err(), "scanning %s", filename)
}

func match2(h *history, pkgpath, id string, v int) {
	p, ok := h.pkgs[pkgpath]
	if !ok {
		p = pkgHistory{
			ids:   make(map[string]int),
			types: make(map[string]map[string]int),
		}
		h.pkgs[pkgpath] = p
	}
	if _, ok := p.ids[id]; !ok {
		p.ids[id] = v
	}
}

func match3(h *history, pkgpath, typ, id string, v int) {
	p, ok := h.pkgs[pkgpath]
	if !ok {
		p = pkgHistory{ids: make(map[string]int), types: make(map[string]map[string]int)}
		h.pkgs[pkgpath] = p
	}
	t, ok := p.types[typ]
	if !ok {
		t = make(map[string]int)
		p.types[typ] = t
	}
	if _, ok := t[id]; !ok {
		t[id] = v
	}
}

var (
	constRegex  = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, const (\w+)`)
	fieldRegex  = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, type (\w+)(\[[^\[\]]+\])? struct, (\w+)`)
	funcRegex   = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, func (\w+)`)
	methodRegex = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, method \(\*?(\w+)(\[[^\[\]]+\])?\) (\w+)`)
	iface1Regex = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, type (\w+) interface { (.+) }`)
	iface2Regex = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, type (\w+) interface, unexported methods`)
	iface3Regex = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, type (\w+) interface, (\w+)`)
	typeRegex   = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, type (\w+)`)
	varRegex    = regexp.MustCompile(`^pkg (\S+)( \(\S+\))?, var (\w+)`)
)
