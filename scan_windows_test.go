//go:build windows

package mingo

import "testing"

func TestIsWithinDirWindowsDifferentVolumes(t *testing.T) {
	if isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`D:\a\project\project.go`,
	) {
		t.Fatal("path on another volume reported as contained")
	}
}

func TestIsWithinDirWindowsSameVolumeOutside(t *testing.T) {
	if isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`C:\git\project\project.go`,
	) {
		t.Fatal("outside path reported as contained")
	}
}

func TestIsWithinDirWindowsChild(t *testing.T) {
	if !isWithinDir(
		`C:\Users\runneradmin\AppData\Local`,
		`C:\Users\runneradmin\AppData\Local\go-build\x.go`,
	) {
		t.Fatal("child path not reported as contained")
	}
}
