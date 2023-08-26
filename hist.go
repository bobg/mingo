package mingo

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/bobg/go-generics/v3/slices"
)

// History is the history of the Go stdlib,
// as parsed from the files in $GOROOT/api.
// It maps a Go stdlib package names to their individual histories.
type History map[string]PkgHistory

// PkgHistory is the history of a single Go stdlib package.
type PkgHistory struct {
	// IDs maps top-level identifiers to the minor version of Go at which they were first introduced.
	IDs map[string]int

	// Types maps type "members" -
	// method names and struct fields -
	// to the minor version of Go at which they were first introduced.
	// First map key is the type name within its package;
	// second map key is the identifier in the type's scope.
	Types map[string]map[string]int
}

// ReadHist reads the history of the Go stdlib
// from the sequence of go1.*.txt files in the given directory.
// The default directory,
// which you get if dir is "",
// is $GOROOT/api.
func ReadHist(dir string) (History, error) {
	if dir == "" {
		dir = filepath.Join(goroot(), "api")
	}

	return ReadHistFS(os.DirFS(dir), ".")
}

// ReadHistFS reads the history of the Go stdlib
// from the sequence of go1.*.txt files
// in the given directory within the given filesystem.
func ReadHistFS(fs fs.FS, dir string) (History, error) {
	h := make(History)

	for i := MinGoMinorVersion; i <= MaxGoMinorVersion; i++ {
		if err := readHistVersion(h, fs, dir, i); err != nil {
			return nil, errors.Wrapf(err, "reading history version %d", i)
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

func readHistVersion(h History, fs fs.FS, dir string, v int) error {
	filename := filepath.Join(dir, fmt.Sprintf("go1.%d.txt", v))
	f, err := fs.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
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
			return errors.Errorf("unrecognized line %s", line)
		}
	}
	return errors.Wrapf(sc.Err(), "scanning %s", filename)
}

func match2(h History, pkgpath, id string, v int) {
	p, ok := h[pkgpath]
	if !ok {
		p = PkgHistory{
			IDs:   make(map[string]int),
			Types: make(map[string]map[string]int),
		}
		h[pkgpath] = p
	}
	if _, ok := p.IDs[id]; !ok {
		p.IDs[id] = v
	}
}

func match3(h History, pkgpath, typ, id string, v int) {
	p, ok := h[pkgpath]
	if !ok {
		p = PkgHistory{IDs: make(map[string]int), Types: make(map[string]map[string]int)}
		h[pkgpath] = p
	}
	t, ok := p.Types[typ]
	if !ok {
		t = make(map[string]int)
		p.Types[typ] = t
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
