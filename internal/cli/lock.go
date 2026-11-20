package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// cacheDir is the user's cache folder, where the locks lie; tests replace it.
var cacheDir = os.UserCacheDir

// lockWait bounds how long mark waits for another mark of the same project;
// tests shorten it.
var lockWait = time.Minute

// lockLifetime is the longest a lock can be held. An older lock is stale,
// also when its process ID is alive, since the ID may have been reused.
const lockLifetime = time.Hour

// projectLock is a held lock: its file and the token it holds.
type projectLock struct {
	file, token string
}

// lockProject takes the lock of a project folder: a file in markitect/locks/
// of the user's cache folder, named after the folder's real path, so that a
// junction, a symbolic link, a short name or a substituted drive lead to the
// same lock. It holds this process's ID and a random token. A stale lock, of
// a process that is gone or older than lockLifetime, is moved aside and taken
// over; a lock moved aside by mistake is put back.
func lockProject(dir string) (*projectLock, error) {
	cache, err := cacheDir()
	if err != nil {
		return nil, fmt.Errorf("no cache folder for the lock of the state: %w", err)
	}
	locks := filepath.Join(cache, "markitect", "locks")
	if err := os.MkdirAll(locks, 0o755); err != nil {
		return nil, err
	}
	key, err := lockKey(dir)
	if err != nil {
		return nil, err
	}
	l := &projectLock{file: filepath.Join(locks, key+".lock"), token: fmt.Sprintf("%d %s\n", os.Getpid(), random())}
	until := time.Now().Add(lockWait)
	for {
		err := createExclusive(l.file, l.token)
		if err == nil {
			return l, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if seen, info, ok := readLock(l.file); ok && staleLock(seen, info) {
			takeOver(l.file, seen)
			continue
		}
		if time.Now().After(until) {
			return nil, fmt.Errorf("another mark of this project holds the lock %s", l.file)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// lockKey names the lock of a project folder after its real path.
func lockKey(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	real, err := canonicalPath(abs)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		real = strings.ToLower(real)
	}
	sum := sha256.Sum256([]byte(filepath.Clean(real)))
	return hex.EncodeToString(sum[:8]), nil
}

// held says whether the lock file still holds this lock's token.
func (l *projectLock) held() bool {
	data, err := os.ReadFile(l.file)
	return err == nil && string(data) == l.token
}

// release removes the lock file, only while it holds this lock's token.
func (l *projectLock) release() {
	if l.held() {
		os.Remove(l.file)
	}
}

func random() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// createExclusive creates file with text, and fails when it exists.
func createExclusive(file, text string) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(text)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func readLock(file string) (string, os.FileInfo, bool) {
	info, err := os.Stat(file)
	if err != nil {
		return "", nil, false
	}
	data, err := os.ReadFile(file)
	return string(data), info, err == nil
}

// staleLock says a lock can no longer be held: it is older than lockLifetime,
// its process is gone, or it has stayed without a process ID for longer than
// writing one takes.
func staleLock(text string, info os.FileInfo) bool {
	age := time.Since(info.ModTime())
	if age > lockLifetime {
		return true
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return age > 5*time.Second
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return age > 5*time.Second
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	defer p.Release()
	if runtime.GOOS == "windows" {
		// FindProcess opens the process, which fails once it is gone.
		return false
	}
	return p.Signal(syscall.Signal(0)) != nil
}

// takeOver moves a stale lock aside, in one rename that only one of several
// takers can make. When what it moved is not the lock it judged, another
// taker's fresh lock, it puts that back unless a newer lock stands there.
func takeOver(file, seen string) {
	aside := file + "." + random() + ".stale"
	if err := os.Rename(file, aside); err != nil {
		return
	}
	defer os.Remove(aside)
	data, info, ok := readLock(aside)
	if !ok || data == seen {
		return
	}
	if createExclusive(file, data) == nil {
		_ = os.Chtimes(file, info.ModTime(), info.ModTime())
	}
}
