package recordstore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// inspectDirectoryPath performs the path-based part of an open. Callers must
// bind the returned identity to an os.Root before doing any store IO.
func inspectDirectoryPath(path string) (os.FileInfo, error) {
	path = filepath.Clean(path)
	if err := inspectPathComponents(path, false); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if err = rejectReparse(path, info); err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a real directory: %s", path)
	}
	return info, nil
}

// openDirectoryHandle binds subsequent operations to the directory inspected
// by the caller. It rechecks ancestors and the final path after opening so a
// substitution between inspection and OpenRoot is rejected.
func openDirectoryHandle(path string, expected os.FileInfo) (*os.Root, error) {
	path = filepath.Clean(path)
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("open directory root %s: %w", path, err)
	}
	opened, err := root.Stat(".")
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("stat opened directory root %s: %w", path, err)
	}
	if err = rejectReparse(path, opened); err != nil {
		_ = root.Close()
		return nil, err
	}
	if !opened.IsDir() || (expected != nil && !os.SameFile(expected, opened)) {
		_ = root.Close()
		return nil, fmt.Errorf("directory identity changed while opening: %s", path)
	}
	if err = inspectPathComponents(path, false); err != nil {
		_ = root.Close()
		return nil, err
	}
	current, err := os.Lstat(path)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("recheck opened directory %s: %w", path, err)
	}
	if err = rejectReparse(path, current); err != nil {
		_ = root.Close()
		return nil, err
	}
	if !current.IsDir() || !os.SameFile(opened, current) {
		_ = root.Close()
		return nil, fmt.Errorf("directory path changed while opening: %s", path)
	}
	return root, nil
}

func openInspectedDirectory(path string, expected os.FileInfo) (*os.Root, error) {
	current, err := inspectDirectoryPath(path)
	if err != nil {
		return nil, err
	}
	if expected != nil && !os.SameFile(expected, current) {
		return nil, fmt.Errorf("directory identity changed: %s", path)
	}
	return openDirectoryHandle(path, current)
}

func openDirectoryAt(parent *os.Root, name string, expected os.FileInfo) (*os.Root, os.FileInfo, error) {
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect directory entry %s: %w", name, err)
	}
	if err = rejectReparse(name, info); err != nil {
		return nil, nil, err
	}
	if !info.IsDir() || (expected != nil && !os.SameFile(expected, info)) {
		return nil, nil, fmt.Errorf("directory entry identity changed: %s", name)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, nil, fmt.Errorf("open directory entry %s: %w", name, err)
	}
	opened, err := child.Stat(".")
	if err != nil {
		_ = child.Close()
		return nil, nil, fmt.Errorf("stat opened directory entry %s: %w", name, err)
	}
	if err = rejectReparse(name, opened); err != nil {
		_ = child.Close()
		return nil, nil, err
	}
	if !opened.IsDir() || !os.SameFile(info, opened) || (expected != nil && !os.SameFile(expected, opened)) {
		_ = child.Close()
		return nil, nil, fmt.Errorf("directory entry changed while opening: %s", name)
	}
	current, err := parent.Lstat(name)
	if err != nil {
		_ = child.Close()
		return nil, nil, fmt.Errorf("recheck opened directory entry %s: %w", name, err)
	}
	if err = rejectReparse(name, current); err != nil {
		_ = child.Close()
		return nil, nil, err
	}
	if !current.IsDir() || !os.SameFile(opened, current) {
		_ = child.Close()
		return nil, nil, fmt.Errorf("directory entry path changed while opening: %s", name)
	}
	return child, opened, nil
}

func makeDirectoryAt(parent *os.Root, name string, mode os.FileMode) (os.FileInfo, error) {
	if err := parent.Mkdir(name, mode); err != nil {
		return nil, err
	}
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("inspect created directory %s: %w", name, err)
	}
	if err = rejectReparse(name, info); err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("created path is not a directory: %s", name)
	}
	return info, nil
}

func readDirAt(root *os.Root, name string, maxEntries int) (_ []os.DirEntry, retErr error) {
	if maxEntries < 0 {
		return nil, errors.New("directory entry limit cannot be negative")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); retErr == nil {
			retErr = closeErr
		}
	}()
	entries := make([]os.DirEntry, 0, min(maxEntries, 128))
	for len(entries) <= maxEntries {
		remaining := maxEntries + 1 - len(entries)
		batchSize := min(remaining, 128)
		batch, readErr := file.ReadDir(batchSize)
		entries = append(entries, batch...)
		if len(entries) > maxEntries {
			return nil, fmt.Errorf("directory %s exceeds %d entries", name, maxEntries)
		}
		if errors.Is(readErr, io.EOF) {
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			return entries, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return nil, fmt.Errorf("directory %s exceeds %d entries", name, maxEntries)
}

func readRegularAt(root *os.Root, name string, max int64) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if err = rejectReparse(name, info); err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > max {
		return nil, fmt.Errorf("expected bounded regular file: %s", name)
	}
	file, err := root.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, fmt.Errorf("file identity changed while opening: %s", name)
	}
	current, err := root.Lstat(name)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if err = rejectReparse(name, current); err != nil {
		_ = file.Close()
		return nil, err
	}
	if !os.SameFile(opened, current) {
		_ = file.Close()
		return nil, fmt.Errorf("file path changed while opening: %s", name)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, max+1))
	after, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if statErr != nil {
		return nil, statErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > max || int64(len(data)) != info.Size() || after.Size() != info.Size() {
		return nil, fmt.Errorf("file changed while reading: %s", name)
	}
	return data, nil
}

func writeExclusiveAt(root *os.Root, name string, data []byte) error {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		_ = file.Close()
		return fmt.Errorf("partial file retained at %s: %w", name, writeErr)
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("file retained at %s: %w", name, err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("file retained at %s: %w", name, err)
	}
	return nil
}

func createPendingAt(root *os.Root) (string, *os.File, error) {
	for attempt := 0; attempt < 16; attempt++ {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return "", nil, err
		}
		name := pending + hex.EncodeToString(nonce[:]) + ".json"
		file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		return name, file, nil
	}
	return "", nil, errors.New("could not allocate a unique pending event name")
}
