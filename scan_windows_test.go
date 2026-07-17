//go:build windows

package mingo

import "testing"

func TestIsWithinDirWindowsDifferentVolumes(t *testing.T) {
	got, err := isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`D:\a\project\project.go`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("path on another volume reported as contained")
	}
}

func TestIsWithinDirWindowsSameVolumeOutside(t *testing.T) {
	got, err := isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`C:\git\project\project.go`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("outside path reported as contained")
	}
}

func TestIsWithinDirWindowsChild(t *testing.T) {
	got, err := isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`C:\Users\runneradmin\AppData\Local\go-build\x.go`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("child path not reported as contained")
	}
}
