//go:build linux

package codexappserver

import (
	"errors"
	"os"
	"runtime"
	"syscall"
)

func metadataOpenError(name string, err error) error {
	if errors.Is(err, syscall.ELOOP) {
		return invalid("refusing symlink at native open %q", name)
	}
	return &os.PathError{Op: "metadata-open", Path: name, Err: err}
}

func openMetadataVolume(name string) (*os.File, error) {
	fd, err := syscall.Open(name, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, metadataOpenError(name, err)
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openMetadataChild(parent *os.File, name string, directory bool) (*os.File, error) {
	flags := syscall.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_NONBLOCK | syscall.O_CLOEXEC
	if directory {
		flags |= syscall.O_DIRECTORY
	}
	// Single component, relative to an already held directory. In particular do
	// not use os.Root.OpenFile: its symlink resolver may retry after ELOOP.
	fd, err := syscall.Openat(int(parent.Fd()), name, flags, 0)
	runtime.KeepAlive(parent)
	if err != nil {
		return nil, metadataOpenError(name, err)
	}
	return os.NewFile(uintptr(fd), name), nil
}
