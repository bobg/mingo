package mingo

import (
	"bytes"
	"fmt"
	"go/token"
	"strconv"
)

type Result interface {
	Version() int
	String() string
}

type intResult int

func (r intResult) Version() int   { return int(r) }
func (r intResult) String() string { return strconv.Itoa(int(r)) }

type posResult struct {
	version int
	pos     token.Position
	desc    string
}

func (r posResult) Version() int { return r.version }

func (r posResult) String() string {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, "%s: %d", r.pos, r.version)
	if r.desc != "" {
		fmt.Fprintf(b, " (%s)", r.desc)
	}

	return b.String()
}
