package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInvalidLayout = errors.New("invalid realization layout")

const markitectDirectory = ".markitect"

// LayoutException maps an exact path, a file or a whole directory subtree, to
// one owner regardless of the mirrored hierarchy. File declares that the path
// is a file; otherwise it is a directory. Owner is a local Package or Root; a
// file can only be owned by Root, because it cannot be mounted writable for a
// Package. Reason is the required persistent explanation.
type LayoutException struct {
	Path   snapshot.Path
	File   bool
	Owner  Owner
	Reason string
}

// Layout is the project's persistent artifact ownership: artifact roots that
// mirror the model hierarchy (root/lending/reminders/... belongs to the local
// Package lending/reminders) plus explicit exceptions. The deepest mapping wins.
// Root owns the files directly in an artifact root and every file outside the
// roots; a file in an unmapped subdirectory of a root has no owner (HasOwner).
// The zero Layout is "not declared": the
// registry shards alone say who owns what. A Layout fixes lexical facts only;
// it grants no write authority and observes no file.
type Layout struct {
	declared   bool
	roots      []string
	exceptions []LayoutException
	packages   map[string]bool
}

// NewLayout validates the declared roots and exceptions against the local
// Package inventory of the passed checked model. It refuses a root that is
// the repository root, repeated or nested; anything at or under .markitect;
// an exception owner that is not Root or a local Package, a file exception
// owned by a Package, a repeated exception path or a missing reason; and any roots, exception paths or mirrored Package
// directories whose directory spellings differ only by letter case, because
// they would name one location on a case-insensitive platform.
func NewLayout(checked model.Result, roots []snapshot.DirectoryBoundary, exceptions []LayoutException) (Layout, error) {
	inventory := inventoryFromResult(checked)
	if !inventory.ready {
		return Layout{}, fmt.Errorf("%w: a complete passed checked model with one local model is required", ErrInvalidLayout)
	}
	layout := Layout{declared: true, packages: inventory.owners}
	for _, root := range roots {
		name := root.String()
		if name == "" || root.IsRoot() {
			return Layout{}, fmt.Errorf("%w: an artifact root must be a directory below the repository root", ErrInvalidLayout)
		}
		if underMarkitect(name) {
			return Layout{}, fmt.Errorf("%w: artifact root %q lies in %s", ErrInvalidLayout, name, markitectDirectory)
		}
		for _, other := range layout.roots {
			if other == name {
				return Layout{}, fmt.Errorf("%w: artifact root %q is repeated", ErrInvalidLayout, name)
			}
			if within(other, name) || within(name, other) {
				return Layout{}, fmt.Errorf("%w: artifact roots %q and %q are nested", ErrInvalidLayout, other, name)
			}
		}
		layout.roots = append(layout.roots, name)
	}
	slices.Sort(layout.roots)
	seen := make(map[string]bool, len(exceptions))
	for _, exception := range exceptions {
		name := exception.Path.String()
		switch {
		case name == "":
			return Layout{}, fmt.Errorf("%w: an exception requires a path", ErrInvalidLayout)
		case underMarkitect(name):
			return Layout{}, fmt.Errorf("%w: exception path %q lies in %s", ErrInvalidLayout, name, markitectDirectory)
		case seen[name]:
			return Layout{}, fmt.Errorf("%w: exception path %q is repeated", ErrInvalidLayout, name)
		case !exception.Owner.valid():
			return Layout{}, fmt.Errorf("%w: exception %q requires an owner", ErrInvalidLayout, name)
		case strings.TrimSpace(exception.Reason) == "":
			return Layout{}, fmt.Errorf("%w: exception %q requires a reason", ErrInvalidLayout, name)
		}
		if packageName, isPackage := exception.Owner.Package(); isPackage {
			if !inventory.owners[packageName] {
				return Layout{}, fmt.Errorf("%w: exception %q names %q, which is not a Manager of the model", ErrInvalidLayout, name, packageName)
			}
			if exception.File {
				return Layout{}, fmt.Errorf("%w: file exception %q cannot be owned by Package %q: a file cannot be mounted and the Package's base is read-only, so only Root can own a file exception under an enforced layout; use a directory exception", ErrInvalidLayout, name, packageName)
			}
		}
		seen[name] = true
		layout.exceptions = append(layout.exceptions, exception)
	}
	slices.SortFunc(layout.exceptions, func(a, b LayoutException) int { return strings.Compare(a.Path.String(), b.Path.String()) })
	if err := layout.caseCollision(); err != nil {
		return Layout{}, err
	}
	return layout, nil
}

// caseCollision refuses distinct directory spellings that share one model name
// collision key, among every root, exception path and mirrored Package directory
// and each of their ancestor directories. A spelling without a key (invalid
// UTF-8) is refused.
func (l Layout) caseCollision() error {
	folded := make(map[string]string)
	var paths []string
	paths = append(paths, l.roots...)
	for _, exception := range l.exceptions {
		paths = append(paths, exception.Path.String())
	}
	for _, root := range l.roots {
		for name := range l.packages {
			paths = append(paths, root+"/"+name)
		}
	}
	slices.Sort(paths)
	for _, path := range paths {
		for end := 0; end <= len(path); end++ {
			if end != len(path) && path[end] != '/' {
				continue
			}
			prefix := path[:end]
			key, ok := model.NameCollisionKey(prefix)
			if !ok {
				return fmt.Errorf("%w: %q is not valid UTF-8 and has no collision key", ErrInvalidLayout, prefix)
			}
			if previous, found := folded[key]; found && previous != prefix {
				return fmt.Errorf("%w: %q and %q differ only by letter case and are ambiguous on case-insensitive platforms", ErrInvalidLayout, previous, prefix)
			}
			folded[key] = prefix
		}
	}
	return nil
}

// Declared reports whether the project declares a layout at all.
func (l Layout) Declared() bool { return l.declared }

func (l Layout) Roots() []snapshot.DirectoryBoundary {
	var roots []snapshot.DirectoryBoundary
	for _, root := range l.roots {
		path, _ := snapshot.NewPath(root)
		boundary, _ := snapshot.DirectoryAt(path)
		roots = append(roots, boundary)
	}
	return roots
}

func (l Layout) Exceptions() []LayoutException { return slices.Clone(l.exceptions) }

// OwnerOf returns the owner of an exact file path: the deepest exception whose
// path is the file or a directory containing it, unless a mirrored Package
// directory is deeper; otherwise the deepest local Package mirrored below the
// path's artifact root; otherwise Root. An exception on the very directory of a
// mirrored Package overrides it. Calling OwnerOf on an undeclared Layout returns
// Root, because no path mapping exists. A file in a subdirectory of an artifact
// root that no Package directory or exception covers has no owner (HasOwner is
// false); OwnerOf returns Root for it, which must resolve it.
func (l Layout) OwnerOf(path snapshot.Path) Owner {
	owner, _ := l.ownerOf(path)
	return owner
}

// HasOwner reports whether the layout assigns the path to an owner. Root owns the
// files directly in an artifact root and every file outside the roots; a file in
// a subdirectory of a root that is neither inside a mirrored Package directory
// nor inside an exception has no owner. An undeclared layout owns every path.
func (l Layout) HasOwner(path snapshot.Path) bool {
	_, owned := l.ownerOf(path)
	return owned || !l.declared
}

func (l Layout) ownerOf(path snapshot.Path) (Owner, bool) {
	name := path.String()
	var exception *LayoutException
	for i := range l.exceptions {
		candidate := &l.exceptions[i]
		if within(candidate.Path.String(), name) && (exception == nil || depth(candidate.Path.String()) > depth(exception.Path.String())) {
			exception = candidate
		}
	}
	mirrored, mirroredDepth := Owner{}, 0
	for _, root := range l.roots {
		if !strings.HasPrefix(name, root+"/") {
			continue
		}
		segments := strings.Split(name[len(root)+1:], "/")
		for k := len(segments) - 1; k >= 1; k-- {
			if candidate := strings.Join(segments[:k], "/"); l.packages[candidate] {
				mirrored, mirroredDepth = Owner{packageName: candidate}, depth(root)+k
				break
			}
		}
	}
	if exception != nil && depth(exception.Path.String()) >= mirroredDepth {
		return exception.Owner, true
	}
	if mirroredDepth > 0 {
		return mirrored, true
	}
	for _, root := range l.roots {
		if below, inside := strings.CutPrefix(name, root+"/"); inside && strings.Contains(below, "/") {
			return RootOwner(), false
		}
	}
	return RootOwner(), true
}

// Area is a directory subtree held by its owner except for the deeper areas
// inside it: a mirrored Package directory below an artifact root or an exception
// directory.
type Area struct {
	Path  snapshot.Path
	Owner Owner
}

// Areas returns the layout's directory areas, parents before descendants: every
// mirrored Package directory under each root and every directory exception.
// A mirrored directory that is a file of the tree and an exception that is
// declared a file or names a file of the tree are no areas. An exception on a
// mirrored directory replaces the mirrored area. Root's files directly in a root
// are not an area.
func (l Layout) Areas(tree snapshot.Tree) []Area {
	owners := make(map[string]Owner)
	for _, root := range l.roots {
		for name := range l.packages {
			if path, err := snapshot.NewPath(root + "/" + name); err == nil && !isFileIn(tree, path) {
				owners[root+"/"+name] = Owner{packageName: name}
			}
		}
	}
	for _, exception := range l.exceptions {
		if !exception.File && !isFileIn(tree, exception.Path) {
			owners[exception.Path.String()] = exception.Owner
		}
	}
	names := make([]string, 0, len(owners))
	for name := range owners {
		names = append(names, name)
	}
	slices.Sort(names)
	areas := make([]Area, len(names))
	for i, name := range names {
		path, _ := snapshot.NewPath(name)
		areas[i] = Area{Path: path, Owner: owners[name]}
	}
	return areas
}

// Describe returns short human-readable descriptions of the areas owner owns
// under this layout, for prompts. The tree only decides whether an exception
// names a file or a directory and supplies an example of a shared file.
func (l Layout) Describe(owner Owner, tree snapshot.Tree) []string {
	var areas []string
	packageName, isPackage := owner.Package()
	for _, root := range l.roots {
		if isPackage {
			base := root + "/" + packageName
			if exception, found := l.exceptionAt(base); found && exception.Owner != owner {
				continue
			}
			var excluded []string
			for name := range l.packages {
				if parent, _ := splitLast(name); parent == packageName {
					excluded = append(excluded, root+"/"+name+"/**")
				}
			}
			areas = append(areas, area(base, l.foreignInside(base, owner, tree, excluded)))
			continue
		}
		shared := ""
		for _, entry := range tree.Entries() {
			if below, inside := strings.CutPrefix(entry.Path().String(), root+"/"); inside && !strings.Contains(below, "/") && l.OwnerOf(entry.Path()).IsRoot() {
				shared = entry.Path().String()
				break
			}
		}
		if shared != "" {
			areas = append(areas, fmt.Sprintf("files directly in %s/ such as %s", root, shared))
		} else {
			areas = append(areas, fmt.Sprintf("files directly in %s/", root))
		}
	}
	for _, exception := range l.exceptions {
		name := exception.Path.String()
		if exception.Owner != owner || isPackage && l.mirrors(name+"/", packageName) {
			continue
		}
		if exception.File || isFileIn(tree, exception.Path) {
			areas = append(areas, name)
			continue
		}
		areas = append(areas, area(name, l.foreignInside(name, owner, tree, l.mirroredInside(name, owner))))
	}
	if owner.IsRoot() {
		areas = append(areas, "every file outside the artifact roots and exceptions")
	}
	return areas
}

// isFileIn reports whether the tree holds a file at path.
func isFileIn(tree snapshot.Tree, path snapshot.Path) bool {
	_, isFile := tree.Lookup(path)
	return isFile
}

func area(dir string, excluded []string) string {
	description := dir + "/**"
	if len(excluded) > 0 {
		slices.Sort(excluded)
		description += " except " + strings.Join(excluded, ", ")
	}
	return description
}

// foreignInside appends the exceptions of other owners strictly inside dir that
// the already excluded areas do not cover.
func (l Layout) foreignInside(dir string, owner Owner, tree snapshot.Tree, excluded []string) []string {
	for _, exception := range l.exceptions {
		name := exception.Path.String()
		if exception.Owner == owner || !strings.HasPrefix(name, dir+"/") {
			continue
		}
		covered := false
		for _, other := range excluded {
			covered = covered || strings.HasPrefix(name+"/", strings.TrimSuffix(other, "**"))
		}
		if covered {
			continue
		}
		if exception.File || isFileIn(tree, exception.Path) {
			excluded = append(excluded, name)
		} else {
			excluded = append(excluded, name+"/**")
		}
	}
	return excluded
}

// mirroredInside lists the mirrored Package directories strictly inside dir
// that another owner holds; an exception directory does not cover them.
func (l Layout) mirroredInside(dir string, owner Owner) []string {
	var excluded []string
	for _, root := range l.roots {
		for name := range l.packages {
			mirrored := root + "/" + name
			probe, err := snapshot.NewPath(mirrored + "/x")
			if strings.HasPrefix(mirrored, dir+"/") && err == nil && l.OwnerOf(probe) != owner {
				excluded = append(excluded, mirrored+"/**")
			}
		}
	}
	return excluded
}

// exceptionAt returns the exception on exactly this path.
func (l Layout) exceptionAt(path string) (LayoutException, bool) {
	for _, exception := range l.exceptions {
		if exception.Path.String() == path {
			return exception, true
		}
	}
	return LayoutException{}, false
}

// mirrors reports whether name lies inside the mirrored directory of the Package.
func (l Layout) mirrors(name, packageName string) bool {
	for _, root := range l.roots {
		if strings.HasPrefix(name, root+"/"+packageName+"/") {
			return true
		}
	}
	return false
}

func splitLast(name string) (string, string) {
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		return name[:slash], name[slash+1:]
	}
	return "", name
}

// within reports whether name is dir itself or lies below it.
func within(dir, name string) bool { return name == dir || strings.HasPrefix(name, dir+"/") }

func depth(path string) int { return strings.Count(path, "/") + 1 }

func underMarkitect(path string) bool { return within(markitectDirectory, path) }
