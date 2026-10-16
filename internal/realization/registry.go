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

// Registry is a fixed set of exact-file registrations, not a storage location,
// mutable revision or approval. The zero Registry is the empty set.
type Registry struct{ entries []Registration }

// NewRegistry copies and sorts entries. Duplicates fail even for the same owner;
// file/directory conflicts also fail. No owner is chosen or transferred here.
// Errors are independent of input order, and failures return no partial registry.
func NewRegistry(entries []Registration) (Registry, error) {
	entries = slices.Clone(entries)
	slices.SortFunc(entries, func(a, b Registration) int {
		return strings.Compare(a.artifact.path.String(), b.artifact.path.String())
	})
	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entry.owner.valid() || entry.artifact.path.String() == "" {
			return Registry{}, fmt.Errorf("%w: zero registration", ErrInvalidRegistry)
		}
		path := entry.artifact.path.String()
		if files[path] {
			return Registry{}, fmt.Errorf("%w: duplicate ownership at %q", ErrInvalidRegistry, path)
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
		files[path] = true
	}
	return Registry{entries: entries}, nil
}

func (r Registry) Entries() []Registration { return slices.Clone(r.entries) }

func (r Registry) Lookup(path snapshot.Path) (Registration, bool) {
	i, found := slices.BinarySearchFunc(r.entries, path.String(), func(e Registration, p string) int {
		return strings.Compare(e.artifact.path.String(), p)
	})
	if !found {
		return Registration{}, false
	}
	return r.entries[i], true
}
