package realization

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	ErrInvalidClaimFacts    = errors.New("invalid path claim facts")
	ErrIncompleteClaimFacts = errors.New("incomplete path claim facts")
)

// CreationArea is an explicit owner-specific directory selection. It grants no
// authority by itself. Only new paths use it; existing ownership is separate.
type CreationArea struct {
	owner     Owner
	directory snapshot.DirectoryBoundary
}

func NewCreationArea(owner Owner, directory snapshot.DirectoryBoundary) (CreationArea, error) {
	if !owner.valid() || directory.String() == "" {
		return CreationArea{}, ErrInvalidClaimFacts
	}
	return CreationArea{owner, directory}, nil
}
func (a CreationArea) Owner() Owner                          { return a.owner }
func (a CreationArea) Directory() snapshot.DirectoryBoundary { return a.directory }

// Protection is a fixed exact-file or directory-subtree restriction. A protected
// file also forbids conflicting ancestor/descendant file claims. No globbing,
// physical existence or rights are inferred from these lexical facts.
type Protection struct {
	file      snapshot.Path
	directory snapshot.DirectoryBoundary
	why       string
}

func ProtectFile(path snapshot.Path, why string) (Protection, error) {
	if path.String() == "" || strings.TrimSpace(why) == "" {
		return Protection{}, ErrInvalidClaimFacts
	}
	return Protection{file: path, why: why}, nil
}
func ProtectDirectory(directory snapshot.DirectoryBoundary, why string) (Protection, error) {
	if directory.String() == "" || strings.TrimSpace(why) == "" {
		return Protection{}, ErrInvalidClaimFacts
	}
	return Protection{directory: directory, why: why}, nil
}
func (p Protection) Why() string                 { return p.why }
func (p Protection) File() (snapshot.Path, bool) { return p.file, p.file.String() != "" }
func (p Protection) Directory() (snapshot.DirectoryBoundary, bool) {
	return p.directory, p.directory.String() != ""
}

type claimModelBasis struct {
	id, digest string
	local      bool
	packages   []string
}

type pathBasis struct {
	models      []claimModelBasis
	owners      map[string]bool
	registry    Registry
	tree        snapshot.Tree
	areas       []CreationArea
	protections []Protection
}

// PathFacts fixes only the explicit facts used by path admissibility checks.
// The zero value is unavailable. Registry drift/dangling basis is deliberately
// not a constructor failure: an existing owner may need to repair it.
//
// The Host supplies the current captured tree, complete protected areas, effective
// creation selections and already recorded ownership/transfer outcomes. Source
// provenance is not a physical path map. These values neither establish live
// task authority nor replace source guards, serialization or durable admission.
type PathFacts struct{ basis *pathBasis }

func NewPathFacts(compiled model.Result, registry Registry, tree snapshot.Tree, areas []CreationArea, protections []Protection) (PathFacts, error) {
	owners, _, ready, findings := accountingModel(compiled)
	if !ready {
		if compiled.Status == model.Failed || hasModelInconsistency(findings) {
			return PathFacts{}, ErrInvalidClaimFacts
		}
		return PathFacts{}, ErrIncompleteClaimFacts
	}
	b := &pathBasis{owners: owners, registry: registry, tree: tree, areas: slices.Clone(areas), protections: slices.Clone(protections)}
	for _, scope := range compiled.Models {
		binding := claimModelBasis{id: scope.ID, digest: scope.Digest, local: scope.Local}
		for _, pkg := range scope.Packages {
			binding.packages = append(binding.packages, pkg.Name)
		}
		slices.Sort(binding.packages)
		b.models = append(b.models, binding)
	}
	slices.SortFunc(b.models, func(a, c claimModelBasis) int { return strings.Compare(a.id, c.id) })
	for _, a := range b.areas {
		if !a.owner.valid() || a.directory.String() == "" {
			return PathFacts{}, ErrInvalidClaimFacts
		}
	}
	slices.SortFunc(b.areas, func(a, c CreationArea) int {
		if order := compareOwner(a.owner, c.owner); order != 0 {
			return order
		}
		return strings.Compare(a.directory.String(), c.directory.String())
	})
	for i, a := range b.areas {
		if i > 0 && a == b.areas[i-1] {
			return PathFacts{}, fmt.Errorf("%w: duplicate creation area", ErrInvalidClaimFacts)
		}
	}
	for _, p := range b.protections {
		if p.why == "" || p.file.String() == "" && p.directory.String() == "" {
			return PathFacts{}, ErrInvalidClaimFacts
		}
	}
	slices.SortFunc(b.protections, func(a, c Protection) int {
		for _, pair := range [][2]string{{a.file.String(), c.file.String()}, {a.directory.String(), c.directory.String()}, {a.why, c.why}} {
			if order := strings.Compare(pair[0], pair[1]); order != 0 {
				return order
			}
		}
		return 0
	})
	return PathFacts{basis: b}, nil
}

func compareOwner(a, b Owner) int {
	if a.root != b.root {
		if a.root {
			return -1
		}
		return 1
	}
	return strings.Compare(a.packageName, b.packageName)
}

func (b *pathBasis) same(other *pathBasis) bool {
	if b == nil || other == nil {
		return false
	}
	if !b.tree.Equal(other.tree) || !slices.Equal(b.areas, other.areas) || !slices.Equal(b.protections, other.protections) {
		return false
	}
	if !slices.EqualFunc(b.models, other.models, func(a, c claimModelBasis) bool {
		return a.id == c.id && a.digest == c.digest && a.local == c.local && slices.Equal(a.packages, c.packages)
	}) {
		return false
	}
	return slices.EqualFunc(b.registry.entries, other.registry.entries, func(a, c Registration) bool {
		return a.owner == c.owner && a.artifact.path == c.artifact.path && a.artifact.form == c.artifact.form && a.artifact.why == c.artifact.why && slices.Equal(a.artifact.basis, c.artifact.basis)
	})
}

func pathsOverlap(a, b snapshot.Path) bool {
	d, _ := snapshot.DirectoryAt(a)
	other, _ := snapshot.DirectoryAt(b)
	return d.Contains(b) || other.Contains(a)
}

func (p Protection) covers(path snapshot.Path) bool {
	if p.file.String() != "" {
		return pathsOverlap(p.file, path)
	}
	d, _ := snapshot.DirectoryAt(path)
	return p.directory.Contains(path) || d.ContainsDirectory(p.directory)
}
