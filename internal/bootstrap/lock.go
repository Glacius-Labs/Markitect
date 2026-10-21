package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var errStateLocked = errors.New("another Apply holds this state directory")

// stateLock is the exclusive operating-system lock of one state directory. The
// system releases it when the process ends, so a crash leaves no stale lock.
type stateLock struct {
	mu   sync.Mutex
	file *os.File
}

// lockState takes the state directory's lock without waiting and reports
// errStateLocked when another Apply holds it.
func lockState(stateDirectory string) (*stateLock, error) {
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(stateDirectory, lockFileName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(file); err != nil {
		file.Close()
		return nil, err
	}
	return &stateLock{file: file}, nil
}

// startLockWait is how long a starting Apply retries a held state lock. A
// concurrent cockpit probes the lock for an instant; that must not
// make a starting Apply fail.
var startLockWait = 500 * time.Millisecond

// lockStateWithin is lockState that retries a held lock for at most wait.
func lockStateWithin(ctx context.Context, stateDirectory string, wait time.Duration) (*stateLock, error) {
	deadline := time.Now().Add(wait)
	for {
		lock, err := lockState(stateDirectory)
		if !errors.Is(err, errStateLocked) || !time.Now().Before(deadline) {
			return lock, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// stateHeld reports whether an Apply holds the state directory. It creates
// nothing: a missing directory or lock file means no Apply runs. The probe takes
// the lock for an instant, which lockStateWithin absorbs.
func stateHeld(stateDirectory string) (bool, error) {
	file, err := os.OpenFile(filepath.Join(stateDirectory, lockFileName), os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	switch err := lockFile(file); {
	case errors.Is(err, errStateLocked):
		return true, nil
	case err != nil:
		return false, err
	}
	return false, nil // closing the file releases the probe's lock
}

// Release gives the lock up. Closing the file releases the system lock.
func (l *stateLock) Release() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Check fences a publication to the Apply that still holds the lock.
func (l *stateLock) Check(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return errors.New("the state directory lock was released")
	}
	return nil
}
