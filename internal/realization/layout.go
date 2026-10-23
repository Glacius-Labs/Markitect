package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInvalidLayout = errors.New("invalid ownership")

const markitectDirectory = ".markitect"

// Area is one declared ownership assignment: an exact file, or with Directory
// a directory and its descendants. A Directory area with an empty Path is the
// whole repository. Owner is Root or a local Package; Reason is an optional
// persistent explanation.
type Area struct {
	Path      string
	Directory bool
	Owner     Owner
	Reason    string
}

// Selector writes an area as it is declared: an exact path, a directory path
// ending in "/", or "./" for the whole repository.
func (a Area) Selector() string {
	switch {
	case a.Directory && a.Path == "":
		return "./"
	case a.Directory:
		return a.Path + "/"
	}
	return a.Path
}

// covers reports whether the area contains path.
func (a Area) covers(path string) bool {
	if !a.Directory {
		return path == a.Path
	}
	return a.Path == "" || path == a.Path || strings.HasPrefix(path, a.Path+"/")
}

// Layout is the project's declared persistent ownership: explicit directory
// and exact-file areas, each owned by Root or a local Package. The most
// specific matching area owns a path: an exact file is more specific than any
// directory containing it, and a deeper directory more specific than one
// containing it. A path no area covers has no owner. The zero Layout is "not
// declared": the registry shards alone then say who registered what. A Layout
// fixes lexical facts only; it grants no write authority and observes no file.
type Layout struct {
	declared bool
	areas    []Area
	packages map[string]bool
}

// NewLayout validates declared areas against the local Package inventory of
// the passed checked model. It refuses an area at or under .markitect or .git
// (protected inputs and Git administration are never owned), an owner that is
// not Root or a local Package, two areas on one path (equal specificity), a path
// declared both as a file and as a directory or lying below an exact-file area,
// and spellings that differ only by letter case.
func NewLayout(checked model.Result, areas []Area) (Layout, error) {
	inventory := inventoryFromResult(checked)
	if !inventory.ready {
		return Layout{}, fmt.Errorf("%w: a complete passed checked model with one local model is required", ErrInvalidLayout)
	}
	layout := Layout{declared: true, packages: inventory.owners}
	seen := map[string]Area{}
	for _, area := range areas {
		name := area.Path
		switch {
		case name == "" && !area.Directory:
			return Layout{}, fmt.Errorf("%w: an exact-file area needs a path", ErrInvalidLayout)
		case name != "" && !cleanPath(name):
			return Layout{}, fmt.Errorf("%w: %q is not a clean repository-relative path", ErrInvalidLayout, area.Selector())
		case name != "" && (within(markitectDirectory, name) || within(".git", name)):
			return Layout{}, fmt.Errorf("%w: %q lies in %s or .git, which nobody owns", ErrInvalidLayout, area.Selector(), markitectDirectory)
		case !area.Owner.valid():
			return Layout{}, fmt.Errorf("%w: area %q requires an owner", ErrInvalidLayout, area.Selector())
		}
		if packageName, isPackage := area.Owner.Package(); isPackage && !inventory.owners[packageName] {
			return Layout{}, fmt.Errorf("%w: area %q names %q, which is not a Manager of the model", ErrInvalidLayout, area.Selector(), packageName)
		}
		if other, dup := seen[name]; dup {
			if other.Directory != area.Directory {
				return Layout{}, fmt.Errorf("%w: %q is declared both as a file and as a directory", ErrInvalidLayout, name)
			}
			return Layout{}, fmt.Errorf("%w: area %q is declared twice; equal specificity cannot decide its owner", ErrInvalidLayout, area.Selector())
		}
		seen[name] = area
		layout.areas = append(layout.areas, area)
	}
	for _, area := range layout.areas {
		if area.Directory {
			continue
		}
		for _, other := range layout.areas {
			if strings.HasPrefix(other.Path, area.Path+"/") {
				return Layout{}, fmt.Errorf("%w: %q lies below the exact-file area %q", ErrInvalidLayout, other.Selector(), area.Selector())
			}
		}
	}
	slices.SortFunc(layout.areas, func(a, b Area) int { return strings.Compare(a.Path, b.Path) })
	if err := layout.caseCollision(); err != nil {
		return Layout{}, err
	}
	return layout, nil
}

func cleanPath(name string) bool {
	_, err := snapshot.NewPath(name)
	return err == nil
}

// caseCollision refuses distinct area spellings, or ancestor directories of
// them, that share one model name collision key.
func (l Layout) caseCollision() error {
	folded := make(map[string]string)
	for _, area := range l.areas {
		path := area.Path
		for end := 1; end <= len(path); end++ {
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

// Declared reports whether the project declares its ownership at all.
func (l Layout) Declared() bool { return l.declared }

// Areas returns the declared areas in path order.
func (l Layout) Areas() []Area { return slices.Clone(l.areas) }

// best returns the most specific area that matches.
func (l Layout) best(match func(Area) bool) (Area, bool) {
	var best Area
	found := false
	for _, a := range l.areas {
		if match(a) && (!found || specificity(a) > specificity(best)) {
			best, found = a, true
		}
	}
	return best, found
}

// specificity orders areas covering one path: an exact file above every
// directory, a deeper directory above a shallower one.
func specificity(a Area) int {
	if !a.Directory {
		return 1 << 20
	}
	if a.Path == "" {
		return 0
	}
	return depth(a.Path)
}

// AreaOf returns the most specific area covering an exact file path.
func (l Layout) AreaOf(path snapshot.Path) (Area, bool) {
	return l.best(func(a Area) bool { return a.covers(path.String()) })
}

// OwnerOf returns the owner of an exact file path and whether one exists: the
// owner of the most specific area covering it. An undeclared Layout owns
// nothing.
func (l Layout) OwnerOf(path snapshot.Path) (Owner, bool) {
	a, ok := l.AreaOf(path)
	return a.Owner, ok
}

// DirectoryOwner returns the owner of a directory itself, as for a new file
// directly in it: the owner of the most specific directory area containing or
// naming it.
func (l Layout) DirectoryOwner(dir string) (Owner, bool) {
	a, ok := l.best(func(a Area) bool { return a.Directory && a.covers(dir) })
	return a.Owner, ok
}

// Describe returns short human-readable descriptions of the areas an owner
// holds, each with the more specific areas of other owners directly inside
// it, for prompts.
func (l Layout) Describe(owner Owner) []string {
	var out []string
	for _, a := range l.areas {
		if a.Owner != owner {
			continue
		}
		if !a.Directory {
			out = append(out, a.Path)
			continue
		}
		var excluded []string
		for _, other := range l.areas {
			if other.Path == a.Path || !a.covers(other.Path) {
				continue
			}
			// Only the next more specific area decides: name it when another
			// owner holds it inside this one.
			if above, _ := l.best(func(x Area) bool { return x.Directory && x.Path != other.Path && x.covers(other.Path) }); above.Path != a.Path || other.Owner == owner {
				continue
			}
			excluded = append(excluded, other.Selector()+" ("+other.Owner.label()+")")
		}
		text := a.Selector()
		if len(excluded) > 0 {
			text += " except " + strings.Join(excluded, ", ")
		}
		out = append(out, text)
	}
	return out
}

// within reports whether name is dir itself or lies below it.
func within(dir, name string) bool { return name == dir || strings.HasPrefix(name, dir+"/") }

func depth(path string) int { return strings.Count(path, "/") + 1 }
