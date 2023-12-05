package mingo

import (
	"go/token"
	"testing"
)

func TestIntResult(t *testing.T) {
	r := intResult(4)
	const want = "4"
	if got := r.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPosResult(t *testing.T) {
	r := posResult{
		version: 4,
		pos:     token.Position{Filename: "foo.go", Line: 17},
		desc:    "foobar",
	}
	const want = "foo.go:17: 4 (foobar)"
	if got := r.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
