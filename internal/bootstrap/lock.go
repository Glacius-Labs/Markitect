package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
)

var errStateLocked = errors.New("another Apply holds this state directory")

// stateLock is the exclusive operating-system lock of one state directory. The
// system releases it when the process ends, so a crash leaves no stale lock.
type stateLock struct{ file *os.File }

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
	return &stateLock{file}, nil
}

// Release gives the lock up. Closing the file releases the system lock.
func (l *stateLock) Release() error { return l.file.Close() }
