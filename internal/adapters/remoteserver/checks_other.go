//go:build !linux

package remoteserver

import (
	"os/exec"
	"sync"
)

// The remote isolates checks only in its Linux container; elsewhere, as in
// tests on other systems, checks run as the current user.

func isolated() bool { return false }

func prepare(*exec.Cmd, int, bool) {}

func finished(*exec.Cmd) {}

func lockChecks(Layout) (func(), error) { return func() {}, nil }

// pushLocks holds one lock per remote: tests on other systems serve agents'
// pushes and push the host's integration from one process.
var pushLocks sync.Map

// lockPushes makes the server's pushes and the host's integration take
// turns within the process.
func lockPushes(l Layout) (func(), error) {
	m, _ := pushLocks.LoadOrStore(l.Root, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock, nil
}

func protect() error { return nil }

func victim() error { return nil }

func chownTree(string, int) error { return nil }

func harden(Layout) error { return nil }

func sweep(Layout, int, bool) error { return nil }
