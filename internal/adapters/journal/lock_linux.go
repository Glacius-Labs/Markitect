package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func mkdirPrivate(path string) error { return os.Mkdir(path, 0700) }

func private(path string) error {
	if _, err := safeAbsolute(path); err != nil {
		return err
	}
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return errors.Join(ErrUnsafe, err)
	}
	if st.Uid != uint32(os.Geteuid()) || st.Mode&0077 != 0 || (st.Mode&unix.S_IFMT != unix.S_IFDIR && (st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1)) {
		return ErrUnsafe
	}
	return nil
}

func physicalID(path string) (string, error) {
	if _, err := safeAbsolute(path); err != nil {
		return "", err
	}
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return "", err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR && (st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1) {
		return "", ErrUnsafe
	}
	return fmt.Sprintf("%x:%x", st.Dev, st.Ino), nil
}

func physicalFileID(f *os.File) (string, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return "", err
	}
	if st.Nlink != 1 {
		return "", ErrUnsafe
	}
	return fmt.Sprintf("%x:%x", st.Dev, st.Ino), nil
}

func openLock(path string, create bool) (*os.File, bool, error) {
	flags := os.O_RDWR
	if create {
		f, err := os.OpenFile(path, flags|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			if err := errors.Join(f.Sync(), syncDirectory(filepath.Dir(path))); err != nil {
				f.Close()
				return nil, false, err
			}
			return f, true, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, false, err
		}
	}
	if err := private(path); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, flags|unix.O_NOFOLLOW, 0)
	return f, false, err
}

func lockExclusive(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) }
func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
