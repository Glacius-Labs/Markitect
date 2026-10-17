package realization

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	ErrInvalidCandidateBase    = errors.New("invalid or mismatched candidate base")
	ErrIncompleteCandidateBase = errors.New("incomplete candidate base")
)

type candidateBasis struct {
	facts       PathFacts
	definitions map[model.Identity]bool
}

// CandidateBase fixes the actual captured tree, logical Registry and the complete
// matching model inventory, protections and creation selections. Zero is invalid.
// It deliberately does not require Accounted: unchanged old problems may remain.
// Host input bindings, source guards and live authority are separate.
type CandidateBase struct{ value *candidateBasis }

func NewCandidateBase(compiled model.Result, facts PathFacts) (CandidateBase, error) {
	return newCandidateBase(inventoryFromResult(compiled), facts)
}

func newCandidateBase(inventory modelInventory, facts PathFacts) (CandidateBase, error) {
	if facts.basis == nil {
		return CandidateBase{}, ErrIncompleteCandidateBase
	}
	b := facts.basis
	matching, err := newPathFacts(inventory, b.registry, b.tree, b.areas, b.protections)
	if err != nil {
		if errors.Is(err, ErrIncompleteClaimFacts) {
			return CandidateBase{}, ErrIncompleteCandidateBase
		}
		return CandidateBase{}, ErrInvalidCandidateBase
	}
	if !b.same(matching.basis) {
		return CandidateBase{}, ErrInvalidCandidateBase
	}
	return CandidateBase{&candidateBasis{facts, inventory.definitions}}, nil
}

func (b CandidateBase) Valid() bool { return b.value != nil }
func (b CandidateBase) Equal(other CandidateBase) bool {
	return b.Valid() && other.Valid() && b.value.facts.basis.same(other.value.facts.basis) && maps.Equal(b.value.definitions, other.value.definitions)
}
func (b CandidateBase) Tree() snapshot.Tree {
	if !b.Valid() {
		return snapshot.Tree{}
	}
	return b.value.facts.basis.tree
}
func (b CandidateBase) Registry() Registry {
	if !b.Valid() {
		return Registry{}
	}
	return b.value.facts.basis.registry
}

// RegistryChange is an exact logical record difference, not a registry-file edit.
// A missing Before or After denotes addition or removal respectively.
type RegistryChange struct {
	path          snapshot.Path
	before, after Registration
}

func (c RegistryChange) Path() snapshot.Path          { return c.path }
func (c RegistryChange) Before() (Registration, bool) { return c.before, c.before.owner.valid() }
func (c RegistryChange) After() (Registration, bool)  { return c.after, c.after.owner.valid() }

func sameRegistration(a, b Registration) bool {
	return a.owner == b.owner && a.artifact.path == b.artifact.path && a.artifact.form == b.artifact.form && a.artifact.why == b.artifact.why && slices.Equal(a.artifact.basis, b.artifact.basis)
}

func registryDifference(before, after Registry) []RegistryChange {
	var changes []RegistryChange
	for _, old := range before.entries {
		next, found := after.Lookup(old.artifact.path)
		if !found || !sameRegistration(old, next) {
			changes = append(changes, RegistryChange{old.artifact.path, old, next})
		}
	}
	for _, next := range after.entries {
		if _, found := before.Lookup(next.artifact.path); !found {
			changes = append(changes, RegistryChange{path: next.artifact.path, after: next})
		}
	}
	slices.SortFunc(changes, func(a, b RegistryChange) int { return comparePath(a.path, b.path) })
	return changes
}

type candidateValue struct {
	base            CandidateBase
	owner           Owner
	tree            snapshot.Tree
	registry        Registry
	changes         []snapshot.Change
	registryChanges []RegistryChange
	touched         []snapshot.Path
	claims          []PathClaim
}

// Candidate is a fixed structurally admissible observation with matching logical
// registry effects and exact base provenance. Zero is invalid. It is neither an
// accepted delivery nor an Accounted, reviewed or completed realization.
type Candidate struct{ value *candidateValue }

func (c Candidate) Valid() bool { return c.value != nil }
func (c Candidate) Base() CandidateBase {
	if !c.Valid() {
		return CandidateBase{}
	}
	return c.value.base
}
func (c Candidate) Owner() Owner {
	if !c.Valid() {
		return Owner{}
	}
	return c.value.owner
}
func (c Candidate) Tree() snapshot.Tree {
	if !c.Valid() {
		return snapshot.Tree{}
	}
	return c.value.tree
}
func (c Candidate) Registry() Registry {
	if !c.Valid() {
		return Registry{}
	}
	return c.value.registry
}
func (c Candidate) Changes() []snapshot.Change {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.value.changes)
}
func (c Candidate) RegistryChanges() []RegistryChange {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.value.registryChanges)
}
func (c Candidate) TouchedPaths() []snapshot.Path {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.value.touched)
}
func (c Candidate) Claims() []PathClaim {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.value.claims)
}

// CandidateStatus describes structural admissibility only, not a run lifecycle.
type CandidateStatus uint8

const (
	CandidateIncomplete CandidateStatus = iota
	CandidateRefused
	CandidateAdmissible
)

type CandidateResult struct {
	status    CandidateStatus
	findings  []Finding
	candidate Candidate
}

func (r CandidateResult) Status() CandidateStatus { return r.status }
func (r CandidateResult) Findings() []Finding     { return slices.Clone(r.findings) }
func (r CandidateResult) Candidate() (Candidate, bool) {
	return r.candidate, r.status == CandidateAdmissible && r.candidate.Valid()
}

func candidateRefusal(code string, owner Owner, path snapshot.Path, message string) CandidateResult {
	return CandidateResult{status: CandidateRefused, findings: []Finding{{Code: code, Owner: owner, Path: path, Message: message}}}
}

func comparePath(a, b snapshot.Path) int {
	return strings.Compare(a.String(), b.String())
}
