package journal

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type registration struct {
	Version                                                 int
	Repository, RepositoryPath, RepositoryFile              string
	RegistrationFile                                        string
	StoreID, Directory, DirectoryFile, DatabaseFile, Status string
}

func inside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Reject redirects in every existing component, including outside the checkout.
// Parent permissions are not changed. The controlled directories have their own
// private permissions; same-user administrative replacement is not a sandbox.
func safeAbsolute(path string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) {
		return "", ErrUnsafe
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", ErrUnsafe
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return absolute, nil
}

func directory(path string, create bool) error {
	if _, err := safeAbsolute(path); err != nil {
		return err
	}
	if create {
		if err := mkdirPrivate(path); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.Join(ErrUnsafe, err)
	}
	if create {
		if err := private(path); err != nil {
			return err
		}
		// Persist its name even after a prior acquisition failed while creating
		// this directory. A retry cannot silently skip an earlier failed fsync.
		return syncDirectory(filepath.Dir(path))
	}
	return nil
}

func readRegistration(path string) (registration, []byte, error) {
	var r registration
	f, err := os.Open(path)
	if err != nil {
		return r, nil, errors.Join(ErrReconcile, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(data) > 16384 {
		return r, nil, ErrReconcile
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, nil, ErrReconcile
	}
	canonical, _ := json.Marshal(r)
	if !bytes.Equal(canonical, data) || r.Version != 1 || !validID(r.Repository) || len(r.StoreID) != 64 {
		return r, nil, ErrReconcile
	}
	return r, data, nil
}

func createRegistration(path string, r registration) error {
	data, _ := json.Marshal(r)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.Join(ErrReconcile, err)
	}
	_, writeErr := f.Write(data)
	err = errors.Join(writeErr, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if err := private(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func finishRegistration(path string, r registration) error {
	// Preserve the checked-created physical registration identity. A crash
	// during this write leaves initialization visibly incomplete, never empty.
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	id, err := physicalFileID(f)
	if err != nil || id != r.RegistrationFile {
		return ErrReconcile
	}
	data, _ := json.Marshal(r)
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}
