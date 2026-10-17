package gitrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func (a *acquisition) readWorking(d *diskBoundary, p string) (snapshot.Entry, error) {
	data, mode, found, err := d.read(p, false)
	if err != nil {
		return snapshot.Entry{}, err
	}
	if !found {
		return snapshot.Entry{}, ErrChanged
	}
	if indexed, ok := a.indexModes[p]; ok {
		mode = indexed
	}
	checked, err := snapshot.NewPath(p)
	if err != nil {
		return snapshot.Entry{}, err
	}
	entry, err := snapshot.NewEntry(checked, mode, data)
	if err != nil {
		return snapshot.Entry{}, err
	}
	if err = a.account("disk:"+filepath.Join(d.abs, filepath.FromSlash(p)), entry); err != nil {
		return snapshot.Entry{}, err
	}
	return entry, nil
}

func (a *acquisition) captureCensus() error {
	if a.revision != "" {
		var names []string
		for p, entry := range a.git.tree {
			if entry.kind != "tree" {
				names = append(names, p)
			}
		}
		slices.Sort(names)
		b := &sourceBoundary{abs: a.git.path, area: snapshot.RootDirectory()}
		for _, p := range names {
			file, found, err := a.read(b, p, false)
			if err != nil {
				return err
			}
			if !found {
				return ErrChanged
			}
			a.census = append(a.census, file)
		}
		return nil
	}
	d, err := a.repositoryDisk()
	if err != nil {
		return err
	}
	metadata, err := d.scan(".", true)
	if err != nil {
		return err
	}
	for _, item := range metadata {
		if item.info.IsDir() {
			continue
		}
		file, err := a.readWorking(d, item.path)
		if err != nil {
			return err
		}
		a.census = append(a.census, file)
		a.censusMetadata = append(a.censusMetadata, item)
	}
	return nil
}

func (a *acquisition) confirmCensus() error {
	if a.revision != "" {
		return nil
	}
	d, err := a.repositoryDisk()
	if err != nil {
		return err
	}
	current, err := d.scan(".", true)
	if err != nil {
		return err
	}
	files := make(map[string]os.FileInfo)
	for _, item := range current {
		if !item.info.IsDir() {
			files[item.path] = item.info
		}
	}
	if len(files) != len(a.censusMetadata) {
		return ErrChanged
	}
	for _, before := range a.censusMetadata {
		after, found := files[before.path]
		if !found || !os.SameFile(before.info, after) || before.info.Size() != after.Size() || !before.info.ModTime().Equal(after.ModTime()) || before.info.Mode() != after.Mode() {
			return ErrChanged
		}
	}
	return nil
}

func (a *acquisition) protectCommitted() error {
	census, err := snapshot.NewTree(a.census)
	if err != nil {
		return err
	}
	check := func(p string, expected snapshot.Entry) error {
		checked, err := snapshot.NewPath(p)
		if err != nil {
			return err
		}
		actual, found := census.Lookup(checked)
		if !found || actual.Mode() != expected.Mode() || actual.ContentID() != expected.ContentID() {
			return fmt.Errorf("%w: protected source %q", ErrDirty, p)
		}
		return nil
	}
	for _, m := range a.models {
		if m.boundary.area.String() == "" {
			continue
		}
		for _, file := range m.files {
			if err := check(within(m.boundary.area, file.Path().String()), file); err != nil {
				return err
			}
		}
		working, err := m.boundary.disk.scan(m.target.Root.String(), false)
		if err != nil {
			return err
		}
		for _, entry := range working {
			repoPath := within(m.boundary.area, entry.path)
			expected, found := a.git.tree[repoPath]
			if entry.info.IsDir() {
				if !found || expected.kind != "tree" {
					nonempty := false
					for _, other := range working {
						if !other.info.IsDir() && strings.HasPrefix(other.path, entry.path+"/") {
							nonempty = true
							break
						}
					}
					if !nonempty {
						return fmt.Errorf("%w: %q", ErrUnrepresentable, repoPath)
					}
					return fmt.Errorf("%w: extra model directory %q", ErrDirty, repoPath)
				}
			} else if !found {
				return fmt.Errorf("%w: extra model file %q", ErrDirty, repoPath)
			}
		}
	}
	for _, group := range [][]apply.SourceFile{a.raw.Execution, a.raw.Registry, a.raw.Ignore, a.raw.HostProfiles, a.raw.Instructions} {
		for _, file := range group {
			if file.RepositoryPath.String() != "" {
				if err := check(file.RepositoryPath.String(), file.File); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *acquisition) protectTracked() error {
	census, err := snapshot.NewTree(a.census)
	if err != nil {
		return err
	}
	b := &sourceBoundary{abs: a.git.path, area: snapshot.RootDirectory()}
	var paths []string
	for p, entry := range a.git.tree {
		if entry.kind != "tree" {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	for _, p := range paths {
		expected, found, err := a.read(b, p, false)
		if err != nil {
			return err
		}
		if !found {
			return ErrChanged
		}
		actual, present := census.Lookup(expected.Path())
		if !present || actual.Mode() != expected.Mode() || actual.ContentID() != expected.ContentID() {
			return fmt.Errorf("%w: raw tracked bytes or mode at %q", ErrDirty, p)
		}
	}
	return nil
}
