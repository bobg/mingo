package main

import "testing"

func TestRun2(t *testing.T) {
	ctx := t.Context()
	if err := run2(ctx); err != nil {
		t.Fatal(err)
	}
}
