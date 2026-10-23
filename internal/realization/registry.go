package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInvalidRegistry = errors.New("invalid realization registry")

// Registration fixes the owner of one Artifact. The zero value is invalid.
type Registration struct {
	owner    Owner
	artifact Artifact
}

func NewRegistration(owner Owner, artifact Artifact) (Registration, error) {
	if !owner.valid() || artifact.path.String() == "" {
		return Registration{}, fmt.Errorf("%w: owner or artifact is zero", ErrInvalidRegistry)
	}
	return Registration{owner: owner, artifact: artifact}, nil
}

func (r Registration) Owner() Owner       { return r.owner }
func (r Registration) Artifact() Artifact { return r.artifact }

// Registry is a fixed set of registry entries, exact files and directories,
// not a storage location, mutable revision or approval. The zero Registry is
// the empty set.
type Registry struct{ entries []Registration }

// NewRegistry copies and sorts entries. Two entries on one path fail even for
// the same owner, and so does an exact file entry with any entry below it. A
// directory entry may contain more specific entries. No owner is chosen or
// transferred here. Errors are independent of input order, and failures return
// no partial registry.
func NewRegistry(entries []Registration) (Registry, error) {
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b Registration) int {
		return strings.Compare(a.artifact.path.String(), b.artifact.path.String())
	})
	paths := make(map[string]bool, len(entries))
	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entry.owner.valid() || entry.artifact.path.String() == "" {
			return Registry{}, fmt.Errorf("%w: zero registration", ErrInvalidRegistry)
		}
		path := entry.artifact.path.String()
		if paths[path] {
			return Registry{}, fmt.Errorf("%w: duplicate registration at %q", ErrInvalidRegistry, path)
		}
		for prefix := path; ; {
			separator := strings.LastIndexByte(prefix, '/')
			if separator < 0 {
				break
			}
			prefix = prefix[:separator]
			if files[prefix] {
				return Registry{}, fmt.Errorf("%w: file/directory conflict at %q", ErrInvalidRegistry, prefix)
			}
		}
		paths[path] = true
		if !entry.artifact.directory {
			files[path] = true
		}
	}
	return Registry{entries: entries}, nil
}

func (r Registry) Entries() []Registration { return slices.Clone(r.entries) }

// Lookup returns the exact file entry of a path.
func (r Registry) Lookup(path snapshot.Path) (Registration, bool) {
	i, found := slices.BinarySearchFunc(r.entries, path.String(), func(e Registration, p string) int {
		return strings.Compare(e.artifact.path.String(), p)
	})
	if !found || r.entries[i].artifact.directory {
		return Registration{}, false
	}
	return r.entries[i], true
}

// Covering returns every entry that covers an exact file path, most specific
// first: its exact entry, then its directory entries from the deepest.
func (r Registry) Covering(path snapshot.Path) []Registration {
	var out []Registration
	for _, e := range r.entries {
		if e.artifact.Covers(path) {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Registration) int {
		return entrySpecificity(b.artifact) - entrySpecificity(a.artifact)
	})
	return out
}

func entrySpecificity(a Artifact) int {
	if !a.directory {
		return 1 << 20
	}
	return depth(a.path.String())
}
