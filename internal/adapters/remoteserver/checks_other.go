//go:build !linux

package remoteserver

import "os/exec"

// The remote isolates checks only in its Linux container; elsewhere, as in
// tests on other systems, checks run as the current user.

func isolated() bool { return false }

func prepare(*exec.Cmd, int, bool) {}

func started(*exec.Cmd) {}

func finished(*exec.Cmd) {}

func chownTree(string, int) error { return nil }

func sweep(int, bool) error { return nil }
