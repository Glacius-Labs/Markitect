//go:build !linux

package remoteserver

import "os/exec"

// The remote isolates checks only in its Linux container; elsewhere, as in
// tests on other systems, checks run as the current user.

func isolated() bool { return false }

func prepare(*exec.Cmd, int, bool) {}

func finished(*exec.Cmd) {}

func protect() error { return nil }

func victim() error { return nil }

func chownTree(string, int) error { return nil }

func harden(Layout) error { return nil }

func sweep(Layout, int, bool) error { return nil }
