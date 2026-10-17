//go:build !linux && !windows

package journal

import "os"

func physicalFileID(*os.File) (string, error) { return "", ErrUnsafe }

// No best-effort lock or private-permission claim on an unimplemented host.
func mkdirPrivate(string) error                     { return ErrUnsafe }
func private(string) error                          { return ErrUnsafe }
func physicalID(string) (string, error)             { return "", ErrUnsafe }
func openLock(string, bool) (*os.File, bool, error) { return nil, false, ErrUnsafe }
func lockExclusive(*os.File) error                  { return ErrUnsafe }
func syncDirectory(string) error                    { return ErrUnsafe }
