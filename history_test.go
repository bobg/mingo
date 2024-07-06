package mingo

import (
	"fmt"
	"testing"
)

func TestHistory(t *testing.T) {
	h, err := readHist("")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		pkgpath, ident, typ string
		want                int
	}{{
		pkgpath: "os",
		ident:   "CreateTemp",
		want:    16,
	}, {
		pkgpath: "errors",
		ident:   "Join",
		want:    20,
	}, {
		pkgpath: "testing",
		typ:     "Cover",
		ident:   "Blocks",
		want:    2,
	}, {
		pkgpath: "crypto/elliptic",
		ident:   "Curve",
		want:    0,
	}, {
		// https://github.com/bobg/mingo/issues/14
		pkgpath: "io",
		ident:   "NopCloser",
		want:    16,
	}}

	for _, tc := range cases {
		var name string
		if tc.typ == "" {
			name = fmt.Sprintf("%s.%s", tc.pkgpath, tc.ident)
		} else {
			name = fmt.Sprintf("%s.%s.%s", tc.pkgpath, tc.typ, tc.ident)
		}
		t.Run(name, func(t *testing.T) {
			got := h.lookup(tc.pkgpath, tc.ident, tc.typ)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
