//go:build customtag

package main

import _ "example.com/tagdep"

type T[X any] = int
