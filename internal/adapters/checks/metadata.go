package checks

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const maxExecutableBytes int64 = 512 << 20
const maxDirectoryEntries = 100000
const maxPathComponents = 256

type executableDirectory struct {
	root *os.Root
	info os.FileInfo
	name string // Exact child name in the preceding held directory.
}

func observeExecutable(ctx context.Context, literal string, observe func(string)) (observation apply.ProfileMetadataFile, err error) {
	boundary, name := filepath.Dir(literal), filepath.Base(literal)
	path, pathErr := snapshot.NewPath(name)
	if pathErr != nil {
		return observation, fmt.Errorf("%w: executable filename cannot be represented: %w", ErrUnsafeExecutable, pathErr)
	}
	location := apply.SourceLocation{Boundary: boundary, Path: path}
	rootPath := filepath.VolumeName(literal) + string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(literal, rootPath), string(filepath.Separator))
	if len(parts) > maxPathComponents {
		return observation, ErrMetadataLimit
	}
	info, err := os.Lstat(rootPath)
	if err != nil {
		return observation, err
	}
	if redirected(info) || !info.IsDir() {
		return observation, ErrUnsafeExecutable
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return observation, err
	}
	dirs := []executableDirectory{{root: root, info: info}}
	defer func() {
		for i := len(dirs) - 1; i >= 0; i-- {
			err = errors.Join(err, dirs[i].root.Close())
		}
	}()
	openedRoot, err := root.Stat(".")
	if err != nil {
		return observation, err
	}
	if !os.SameFile(info, openedRoot) {
		return observation, ErrExecutableChanged
	}
	for _, component := range parts[:len(parts)-1] {
		if err := exactEntry(ctx, root, component); err != nil {
			return observation, err
		}
		before, err := root.Lstat(component)
		if err != nil {
			return observation, err
		}
		if redirected(before) || !before.IsDir() {
			return observation, ErrUnsafeExecutable
		}
		next, err := root.OpenRoot(component)
		if err != nil {
			return observation, err
		}
		dirs = append(dirs, executableDirectory{root: next, info: before, name: component})
		after, err := next.Stat(".")
		if err != nil {
			return observation, err
		}
		atPath, err := root.Lstat(component)
		if err != nil {
			return observation, err
		}
		if redirected(atPath) || !os.SameFile(before, after) || !os.SameFile(after, atPath) {
			return observation, ErrExecutableChanged
		}
		root = next
	}
	if err := exactEntry(ctx, root, name); err != nil {
		return observation, err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return observation, err
	}
	if redirected(before) || !before.Mode().IsRegular() {
		return observation, ErrUnsafeExecutable
	}
	if before.Size() > maxExecutableBytes {
		return observation, ErrMetadataLimit
	}
	if observe != nil {
		observe("before-open")
	}
	f, err := openExecutable(root, name)
	if err != nil {
		return observation, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	opened, err := f.Stat()
	if err != nil {
		return observation, err
	}
	if !opened.Mode().IsRegular() || !sameMetadata(before, opened) {
		return observation, ErrExecutableChanged
	}
	confirm := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		after, err := f.Stat()
		if err != nil {
			return err
		}
		atPath, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if redirected(atPath) || !sameMetadata(opened, after) || !sameMetadata(after, atPath) {
			return ErrExecutableChanged
		}
		currentRoot, err := os.Lstat(rootPath)
		if err != nil {
			return err
		}
		if redirected(currentRoot) || !os.SameFile(dirs[0].info, currentRoot) {
			return ErrExecutableChanged
		}
		for i := 1; i < len(dirs); i++ {
			current, err := dirs[i-1].root.Lstat(dirs[i].name)
			if err != nil {
				return err
			}
			if redirected(current) || !current.IsDir() || !os.SameFile(dirs[i].info, current) {
				return ErrExecutableChanged
			}
		}
		return nil
	}
	if err := confirm(); err != nil {
		return observation, err
	}
	digest, size, err := hashExecutable(ctx, f)
	if err != nil {
		return observation, err
	}
	mode := snapshot.Regular
	if opened.Mode().Perm()&0111 != 0 {
		mode = snapshot.Executable
	}
	observation = apply.ProfileMetadataFile{Location: location, ContentID: digest, Size: size, Mode: mode}
	if observe != nil {
		observe("after-read")
	}
	if err := confirm(); err != nil {
		return observation, err
	}
	if size != opened.Size() {
		return observation, ErrExecutableChanged
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return observation, err
	}
	// A second bounded read catches same-size rewrites with restored timestamps.
	repeated, repeatedSize, err := hashExecutable(ctx, f)
	if err != nil {
		return observation, err
	}
	if repeated != digest || repeatedSize != size {
		return observation, ErrExecutableChanged
	}
	return observation, confirm()
}

func sameMetadata(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.Mode() == b.Mode() && a.ModTime().Equal(b.ModTime())
}

func exactEntry(ctx context.Context, root *os.Root, want string) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		names, err := f.Readdirnames(256)
		count += len(names)
		if count > maxDirectoryEntries {
			return ErrMetadataLimit
		}
		for _, name := range names {
			if name == want {
				return nil
			}
		}
		if errors.Is(err, io.EOF) {
			return &os.PathError{Op: "exact-name", Path: want, Err: os.ErrNotExist}
		}
		if err != nil {
			return err
		}
	}
}

func hashExecutable(ctx context.Context, f *os.File) (snapshot.ContentID, int64, error) {
	h := sha256.New()
	buffer := make([]byte, 64<<10)
	var size int64
	for {
		if err := ctx.Err(); err != nil {
			return snapshot.ContentID{}, size, err
		}
		n, err := f.Read(buffer)
		size += int64(n)
		if size > maxExecutableBytes {
			return snapshot.ContentID{}, size, ErrMetadataLimit
		}
		if n > 0 {
			_, _ = h.Write(buffer[:n])
		}
		if errors.Is(err, io.EOF) {
			var digest snapshot.ContentID
			copy(digest[:], h.Sum(nil))
			return digest, size, ctx.Err()
		}
		if err != nil {
			return snapshot.ContentID{}, size, err
		}
		if n == 0 {
			return snapshot.ContentID{}, size, io.ErrNoProgress
		}
	}
}
