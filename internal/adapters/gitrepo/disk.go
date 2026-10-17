package gitrepo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type diskEntry struct {
	path string
	info os.FileInfo
}
type diskDirectory struct {
	root *os.Root
	info os.FileInfo
}
type diskBoundary struct {
	abs      string
	repoArea snapshot.DirectoryBoundary
	ctx      context.Context
	limits   Limits
	dirs     map[string]diskDirectory
	listings map[string][]string
	files    map[string]os.FileInfo
	contents map[string]snapshot.ContentID
	names    map[string]bool
	observe  func(string, string)
}

func realDirectory(abs string) (os.FileInfo, error) {
	var info os.FileInfo
	for current := abs; ; current = filepath.Dir(current) {
		checked, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if redirected(checked) || !checked.IsDir() {
			return nil, fmt.Errorf("%w: directory %q is redirected or not a directory", ErrUnsafe, current)
		}
		if current == abs {
			info = checked
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return info, nil
}

func openDisk(ctx context.Context, abs string, limits Limits, observe func(string, string)) (*diskBoundary, error) {
	info, err := realDirectory(abs)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = root.Close()
		return nil, ErrChanged
	}
	return &diskBoundary{abs: abs, ctx: ctx, limits: limits, dirs: map[string]diskDirectory{".": {root: root, info: opened}}, listings: make(map[string][]string), files: make(map[string]os.FileInfo), contents: make(map[string]snapshot.ContentID), names: make(map[string]bool), observe: observe}, nil
}

func (d *diskBoundary) close() {
	for _, dir := range d.dirs {
		_ = dir.root.Close()
	}
}

func directoryNames(root *os.Root, limit int) ([]string, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(names) > limit {
		return nil, fmt.Errorf("%w: directory entries", ErrLimit)
	}
	slices.Sort(names)
	return names, nil
}

func exactName(names []string, want string, collision bool) error {
	if !slices.Contains(names, want) {
		return fmt.Errorf("%w: exact stored name %q is absent", ErrUnsafe, want)
	}
	if !collision {
		return nil
	}
	key, ok := model.NameCollisionKey(want)
	if !ok {
		return fmt.Errorf("%w: invalid UTF8 name", ErrUnsafe)
	}
	for _, name := range names {
		other, valid := model.NameCollisionKey(name)
		if !valid {
			return fmt.Errorf("%w: invalid UTF8 sibling", ErrUnsafe)
		}
		if name != want && other == key {
			return fmt.Errorf("%w: colliding stored names %q and %q", ErrUnsafe, want, name)
		}
	}
	return nil
}

func (d *diskBoundary) directory(rel string, collision bool) (*os.Root, error) {
	current := d.dirs["."].root
	if rel == "." || rel == "" {
		return current, nil
	}
	if _, err := snapshot.NewPath(rel); err != nil {
		return nil, err
	}
	parts := strings.Split(rel, "/")
	for i, name := range parts {
		if err := d.ctx.Err(); err != nil {
			return nil, err
		}
		names, err := directoryNames(current, d.limits.Entries)
		if err != nil {
			return nil, err
		}
		if err := exactName(names, name, collision); err != nil {
			return nil, err
		}
		info, err := current.Lstat(name)
		if err != nil {
			return nil, err
		}
		if redirected(info) || !info.IsDir() {
			return nil, fmt.Errorf("%w: directory component %q", ErrUnsafe, name)
		}
		key := strings.Join(parts[:i+1], "/")
		if collision {
			d.names[key] = true
		}
		if cached, ok := d.dirs[key]; ok {
			if !os.SameFile(info, cached.info) {
				return nil, ErrChanged
			}
			current = cached.root
			continue
		}
		next, err := current.OpenRoot(name)
		if err != nil {
			return nil, err
		}
		opened, err := next.Stat(".")
		if err != nil || !opened.IsDir() || !os.SameFile(info, opened) {
			_ = next.Close()
			return nil, ErrChanged
		}
		d.dirs[key] = diskDirectory{root: next, info: opened}
		current = next
	}
	return current, nil
}

func (d *diskBoundary) read(rel string, collision bool) ([]byte, snapshot.Mode, bool, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, 0, false, err
	}
	if _, err := snapshot.NewPath(rel); err != nil {
		return nil, 0, false, err
	}
	parent, name := path.Split(rel)
	parent = strings.TrimSuffix(parent, "/")
	if parent == "" {
		parent = "."
	}
	root, err := d.directory(parent, collision)
	if err != nil {
		return nil, 0, false, err
	}
	names, err := directoryNames(root, d.limits.Entries)
	if err != nil {
		return nil, 0, false, err
	}
	if !slices.Contains(names, name) {
		if collision {
			want, _ := model.NameCollisionKey(name)
			for _, other := range names {
				key, ok := model.NameCollisionKey(other)
				if !ok || key == want {
					return nil, 0, false, fmt.Errorf("%w: aliased source file %q", ErrUnsafe, rel)
				}
			}
		}
		if existed := d.names[rel]; existed {
			return nil, 0, false, ErrChanged
		}
		if collision {
			d.names[rel] = false
		}
		return nil, 0, false, nil
	}
	if present, observed := d.names[rel]; observed && !present {
		return nil, 0, false, ErrChanged
	}
	if collision {
		d.names[rel] = true
	}
	if err := exactName(names, name, collision); err != nil {
		return nil, 0, false, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, 0, false, err
	}
	if redirected(info) || !info.Mode().IsRegular() {
		return nil, 0, false, fmt.Errorf("%w: nonregular file %q", ErrUnsafe, rel)
	}
	if info.Size() > d.limits.FileBytes {
		return nil, 0, false, fmt.Errorf("%w: file %q", ErrLimit, rel)
	}
	if d.observe != nil {
		d.observe("before-file-open", filepath.Join(d.abs, filepath.FromSlash(rel)))
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, 0, false, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, 0, false, ErrChanged
	}
	beforeRead, err := root.Lstat(name)
	if err != nil || redirected(beforeRead) || !os.SameFile(opened, beforeRead) {
		return nil, 0, false, ErrChanged
	}
	data, err := io.ReadAll(io.LimitReader(f, d.limits.FileBytes+1))
	if err != nil {
		return nil, 0, false, err
	}
	if int64(len(data)) > d.limits.FileBytes {
		return nil, 0, false, ErrLimit
	}
	after, err := f.Stat()
	atPath, pathErr := root.Lstat(name)
	if err != nil || pathErr != nil || redirected(atPath) || !os.SameFile(opened, after) || !os.SameFile(after, atPath) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) || after.Size() != int64(len(data)) {
		return nil, 0, false, ErrChanged
	}
	if prior, ok := d.files[rel]; ok && (!os.SameFile(prior, after) || prior.Size() != after.Size() || !prior.ModTime().Equal(after.ModTime()) || prior.Mode() != after.Mode()) {
		return nil, 0, false, ErrChanged
	}
	d.files[rel] = after
	content := snapshot.ContentID(sha256.Sum256(data))
	if previous, seen := d.contents[rel]; seen && previous != content {
		return nil, 0, false, ErrChanged
	}
	d.contents[rel] = content
	mode := snapshot.Regular
	if info.Mode().Perm()&0111 != 0 {
		mode = snapshot.Executable
	}
	return data, mode, true, d.ctx.Err()
}

func (d *diskBoundary) scan(prefix string, census bool) ([]diskEntry, error) {
	root, err := d.directory(prefix, false)
	if err != nil {
		return nil, err
	}
	var entries []diskEntry
	var walk func(*os.Root, string) error
	walk = func(dir *os.Root, rel string) error {
		if err := d.ctx.Err(); err != nil {
			return err
		}
		names, err := directoryNames(dir, d.limits.Entries)
		if err != nil {
			return err
		}
		var included []string
		for _, name := range names {
			p := name
			if rel != "." {
				p = rel + "/" + name
			}
			if d.excluded(p, false) {
				continue
			}
			info, err := dir.Lstat(name)
			if err != nil {
				return err
			}
			if d.excluded(p, info.IsDir()) {
				continue
			}
			included = append(included, name)
			if _, err := snapshot.NewPath(p); err != nil {
				return err
			}
			if redirected(info) {
				return fmt.Errorf("%w: redirected source %q", ErrUnsafe, p)
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("%w: special source %q", ErrUnsafe, p)
			}
			entries = append(entries, diskEntry{path: p, info: info})
			if len(entries) > d.limits.Entries {
				return ErrLimit
			}
			if info.IsDir() {
				next, err := d.directory(p, false)
				if err != nil {
					return err
				}
				if err := walk(next, p); err != nil {
					return err
				}
			}
		}
		if !census {
			if before, ok := d.listings[rel]; ok && !slices.Equal(before, included) {
				return ErrChanged
			}
			d.listings[rel] = included
		}
		return nil
	}
	if prefix != "." {
		info, err := root.Stat(".")
		if err != nil {
			return nil, err
		}
		entries = append(entries, diskEntry{path: prefix, info: info})
	}
	err = walk(root, prefix)
	return entries, err
}

func (d *diskBoundary) confirm() error {
	for rel, present := range d.names {
		parent, err := d.directory(path.Dir(rel), true)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrChanged, err)
		}
		names, err := directoryNames(parent, d.limits.Entries)
		if err != nil {
			return err
		}
		name := path.Base(rel)
		if slices.Contains(names, name) != present {
			return ErrChanged
		}
		if present {
			if err := exactName(names, name, true); err != nil {
				return fmt.Errorf("%w: %w", ErrChanged, err)
			}
		} else {
			key, _ := model.NameCollisionKey(name)
			for _, other := range names {
				candidate, ok := model.NameCollisionKey(other)
				if !ok || key == candidate {
					return ErrChanged
				}
			}
		}
	}
	// Metadata alone misses same-size rewrites with restored timestamps. Read
	// the original paths again and require their captured bytes and identities.
	for rel := range d.files {
		if _, _, found, err := d.read(rel, false); err != nil {
			return err
		} else if !found {
			return ErrChanged
		}
	}
	for rel, dir := range d.dirs {
		p := d.abs
		if rel != "." {
			p = filepath.Join(p, filepath.FromSlash(rel))
		}
		info, err := realDirectory(p)
		if err != nil || !os.SameFile(dir.info, info) {
			return ErrChanged
		}
		if names, watched := d.listings[rel]; watched {
			current, err := directoryNames(dir.root, d.limits.Entries)
			if err != nil {
				return err
			}
			current = slices.DeleteFunc(current, func(name string) bool {
				p := name
				if rel != "." {
					p = rel + "/" + name
				}
				info, err := dir.root.Lstat(name)
				return err == nil && d.excluded(p, info.IsDir())
			})
			if !slices.Equal(names, current) {
				return ErrChanged
			}
		}
	}
	for rel, before := range d.files {
		root, err := d.directory(path.Dir(rel), false)
		if err != nil {
			return err
		}
		after, err := root.Lstat(path.Base(rel))
		if err != nil || redirected(after) || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || before.Mode() != after.Mode() {
			return ErrChanged
		}
	}
	return d.ctx.Err()
}

func (d *diskBoundary) excluded(p string, directory bool) bool {
	for _, part := range strings.Split(p, "/") {
		if part == ".git" {
			return true
		}
	}
	if d.repoArea.String() == "" {
		return false
	}
	repoPath := appendRelative(d.repoArea.String(), p)
	return strings.HasPrefix(repoPath, ".markitect/state/") || directory && repoPath == ".markitect/state"
}
