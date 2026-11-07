package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ErrBusy says another process holds a lock.
var ErrBusy = errors.New("busy")

// lockWait bounds how long mark waits for another to write the record.
const lockWait = time.Minute

// lockLifetime is the longest a lock can be real: a shell call is bounded by two
// hours, so an older one names a process ID that has been used again.
const lockLifetime = 2*time.Hour + 10*time.Minute

// lockSettling is how long a lock file may stay empty between being made and
// being written.
const lockSettling = 5 * time.Second

// lock takes the lock name in dir, a file that holds this process's ID, waiting at
// most wait. A lock whose process has gone is taken over, so that a command that
// was killed can be run again.
func lock(dir, name string, wait time.Duration) (func(), error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	file := filepath.Join(dir, name+".lock")
	until := time.Now().Add(wait)
	for {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(file) }, nil
		}
		if staleLock(file) {
			os.Remove(file)
			continue
		}
		if !time.Now().Before(until) {
			return nil, ErrBusy
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// staleLock says a lock can no longer be real: it is too old, empty for longer
// than it takes to write, or its process has gone.
func staleLock(file string) bool {
	info, err := os.Stat(file)
	if err != nil {
		return false
	}
	age := time.Since(info.ModTime())
	if age > lockLifetime {
		return true
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return age > lockSettling
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	defer p.Release()
	if runtime.GOOS == "windows" {
		// FindProcess opens the process, which fails once it has gone.
		return false
	}
	return p.Signal(syscall.Signal(0)) != nil
}

// writeAtomic writes a file of a folder in one step: a crash leaves the old one or
// the new one.
func writeAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, name+".*.tmp")
	if err != nil {
		return err
	}
	temp := tmp.Name()
	defer os.Remove(temp)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	for i := 0; ; i++ {
		err = os.Rename(temp, filepath.Join(dir, name))
		if err == nil || i == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}
