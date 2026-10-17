package gitrepo

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func directoryBoundary(p string) (snapshot.DirectoryBoundary, error) {
	if p == "." {
		return snapshot.RootDirectory(), nil
	}
	checked, err := snapshot.NewPath(p)
	if err != nil {
		return snapshot.DirectoryBoundary{}, err
	}
	return snapshot.DirectoryAt(checked)
}

func appendRelative(parent, child string) string {
	if parent == "." {
		return child
	}
	if child == "." {
		return parent
	}
	return parent + "/" + child
}

func (a *acquisition) manifest(m *modelCapture) error {
	s := m.target
	s.Source.Origin = "gitrepo:" + s.Selection.Boundary + "::" + s.Selection.ManifestPath.String()
	s.Source.ManifestPath = s.Selection.ManifestPath.String()
	b, err := a.boundary(s.Selection.Boundary)
	m.boundary = b
	if b != nil {
		s.RepositoryArea = b.area
	}
	if err != nil {
		return err
	}
	if s == &a.raw.Local && b.area.String() == "" {
		return fmt.Errorf("%w: local model must be in repository", ErrUnsafe)
	}
	file, found, err := a.read(b, s.Source.ManifestPath, true)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("selected manifest absent: %q", s.Source.ManifestPath)
	}
	if err = m.addFile(file); err != nil {
		return err
	}
	inspection := model.InspectManifest(model.ManifestInput{Origin: s.Source.Origin, ManifestPath: s.Source.ManifestPath, Bytes: file.Bytes(), Limits: a.repository.limits.Manifest})
	if inspection.Status != model.Passed || inspection.Root == "" {
		return apply.NewManifestPreflightError(s.Selection, inspection)
	}
	s.Root, err = directoryBoundary(appendRelative(path.Dir(s.Source.ManifestPath), inspection.Root))
	return err
}

func (m *modelCapture) addFile(file snapshot.Entry) error {
	p := file.Path().String()
	if previous, ok := m.seen[p]; ok {
		if previous != file {
			return ErrChanged
		}
		return nil
	}
	m.seen[p] = file
	m.files = append(m.files, file)
	m.target.Source.Entries = append(m.target.Source.Entries, model.Entry{Path: p, Kind: model.Regular, Bytes: file.Bytes()})
	return nil
}

func (a *acquisition) virtualDirectory(p string, collision bool) error {
	if p == "." {
		return nil
	}
	parts := strings.Split(p, "/")
	for i, name := range parts {
		current := strings.Join(parts[:i+1], "/")
		parent := path.Dir(current)
		var siblings []string
		for candidate := range a.git.tree {
			if path.Dir(candidate) == parent {
				siblings = append(siblings, path.Base(candidate))
			}
		}
		if err := exactName(siblings, name, collision); err != nil {
			return err
		}
		entry, found := a.git.tree[current]
		if !found || entry.kind != "tree" || entry.mode != "040000" {
			return fmt.Errorf("%w: revision directory %q absent or redirected", ErrUnsafe, current)
		}
	}
	return nil
}

func (a *acquisition) virtualFile(p string) error {
	if err := a.virtualDirectory(path.Dir(p), true); err != nil {
		return err
	}
	var names []string
	for candidate := range a.git.tree {
		if path.Dir(candidate) == path.Dir(p) {
			names = append(names, path.Base(candidate))
		}
	}
	// Optional sidecars may be absent, but an alternate/colliding spelling may
	// not replace the requested name or receive the metadata exemption.
	if !slices.Contains(names, path.Base(p)) {
		want, _ := model.NameCollisionKey(path.Base(p))
		for _, name := range names {
			key, ok := model.NameCollisionKey(name)
			if !ok || key == want {
				return fmt.Errorf("%w: aliased revision file %q", ErrUnsafe, p)
			}
		}
		return nil
	}
	return exactName(names, path.Base(p), true)
}

func (a *acquisition) preflightRoots() error {
	for _, m := range a.models {
		if a.revision == "" && m.boundary.area.String() != "" {
			d, err := a.repositoryDisk()
			if err != nil {
				return err
			}
			if _, err = d.directory(m.boundary.area.String(), true); err != nil {
				return err
			}
		}
		if m.boundary.area.String() != "" {
			root := appendRelative(m.boundary.area.String(), m.target.Root.String())
			if root == ".markitect/state" || technical(root) {
				return fmt.Errorf("%w: model root enters excluded administration %q", ErrUnsafe, root)
			}
		}
		if a.commit != "" && m.boundary.area.String() != "" {
			if err := a.virtualDirectory(appendRelative(m.boundary.area.String(), m.target.Root.String()), true); err != nil {
				return err
			}
		}
		if a.revision == "" {
			if _, err := m.boundary.disk.directory(m.target.Root.String(), true); err != nil {
				return err
			}
		}
	}
	for i, left := range a.models {
		for _, right := range a.models[i+1:] {
			if a.revision != "" {
				l := appendRelative(left.boundary.area.String(), left.target.Root.String())
				r := appendRelative(right.boundary.area.String(), right.target.Root.String())
				if overlaps(l, r) {
					return fmt.Errorf("%w: overlapping revision model roots", ErrUnsafe)
				}
				continue
			}
			l := filepath.Join(left.boundary.abs, filepath.FromSlash(left.target.Root.String()))
			r := filepath.Join(right.boundary.abs, filepath.FromSlash(right.target.Root.String()))
			leftContains, err := physicalAncestor(l, r)
			if err != nil {
				return err
			}
			rightContains, err := physicalAncestor(r, l)
			if err != nil {
				return err
			}
			if leftContains || rightContains {
				return fmt.Errorf("%w: physically overlapping model roots", ErrUnsafe)
			}
		}
	}
	return nil
}

func overlaps(left, right string) bool {
	return left == "." || right == "." || left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func physicalAncestor(ancestor, descendant string) (bool, error) {
	a, err := os.Lstat(ancestor)
	if err != nil {
		return false, err
	}
	for current := descendant; ; current = filepath.Dir(current) {
		b, err := os.Lstat(current)
		if err != nil {
			return false, err
		}
		if redirected(b) {
			return false, ErrChanged
		}
		if os.SameFile(a, b) {
			return true, nil
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return false, nil
}

func (a *acquisition) modelTree(m *modelCapture) error {
	root := m.target.Root.String()
	if a.commit != "" && m.boundary.area.String() != "" {
		prefix := appendRelative(m.boundary.area.String(), root)
		var paths []string
		for p := range a.git.tree {
			if prefix == "." || p == prefix || strings.HasPrefix(p, prefix+"/") {
				paths = append(paths, p)
			}
		}
		slices.Sort(paths)
		for _, p := range paths {
			if m.target.Root.IsRoot() && p == m.boundary.area.String() {
				continue
			}
			rel := p
			if !m.boundary.area.IsRoot() {
				rel = strings.TrimPrefix(p, m.boundary.area.String()+"/")
			}
			entry := a.git.tree[p]
			if entry.kind == "tree" {
				m.target.Source.Entries = append(m.target.Source.Entries, model.Entry{Path: rel, Kind: model.Directory})
				continue
			}
			file, found, err := a.read(m.boundary, rel, false)
			if err != nil {
				return err
			}
			if !found {
				return ErrChanged
			}
			if err = m.addFile(file); err != nil {
				return err
			}
		}
	} else {
		metadata, err := m.boundary.disk.scan(root, false)
		if err != nil {
			return err
		}
		for _, entry := range metadata {
			if entry.info.IsDir() {
				m.target.Source.Entries = append(m.target.Source.Entries, model.Entry{Path: entry.path, Kind: model.Directory})
				continue
			}
			file, found, err := a.read(m.boundary, entry.path, false)
			if err != nil {
				return err
			}
			if !found {
				return ErrChanged
			}
			if err = m.addFile(file); err != nil {
				return err
			}
		}
	}
	lock := appendRelative(path.Dir(m.target.Selection.ManifestPath.String()), "model-lock.yaml")
	file, found, err := a.read(m.boundary, lock, true)
	if err != nil {
		return err
	}
	if found {
		if err = m.addFile(file); err != nil {
			return err
		}
	}
	return nil
}
