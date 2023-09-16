package mingo

import (
	"os"
	"strconv"
	"testing"
)

func TestMingo(t *testing.T) {
	entries, err := os.ReadDir("_testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		min, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			s := Scanner{Verbose: testing.Verbose()}
			res, err := s.ScanDir("_testdata/" + entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			if res.Version() != min {
				t.Errorf("got %d, want %d", res.Version(), min)
			}
		})
	}
}
