// Package mingo contains logic for scanning the packages in a Go module
// to determine the lowest-numbered version of Go that can build it.
package mingo

import (
	"runtime"
	"strconv"
	"strings"
)

func (s *Scanner) lookup(pkgpath, name, typ string) int {
	return s.h.lookup(pkgpath, name, typ)
}

func GoMinorVersion() int {
	vstr := runtime.Version()
	vstr = strings.TrimPrefix(vstr, "go")
	parts := strings.SplitN(vstr, ".", 3)
	if len(parts) < 2 {
		return 0
	}
	v, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return v
}
