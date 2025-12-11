package mingo

import "testing"

// https://github.com/bobg/mingo/issues/14
func TestBug14(t *testing.T) {
	var s Scanner
	res, err := s.ScanDir("_testdata/bug14")
	if err != nil {
		t.Fatal(err)
	}
	if v := res.Version(); v != 16 {
		t.Errorf("got %d, want 16", v)
	}
}

func TestBug26(t *testing.T) {
	var s Scanner
	res, err := s.ScanDir("_testdata/bug26")
	if err != nil {
		t.Fatal(err)
	}
	if v := res.Version(); v != 18 {
		t.Errorf("got %d, want 18", v)
	}
}
