package realization

import (
	"maps"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Aggregate is the accepted state of one run (core rule D2): the captured base
// as version zero, and the tree and registry after every accepted delivery.
// Zero is invalid. It is neither a workspace nor acceptance evidence.
type Aggregate struct{ value *aggregateValue }

type aggregateValue struct {
	base     CandidateBase
	version  uint64
	tree     snapshot.Tree
	registry Registry
}

// NewAggregate starts a run's aggregate at version zero from its captured base.
func NewAggregate(base CandidateBase) (Aggregate, error) {
	if !base.Valid() {
		return Aggregate{}, ErrInvalidCandidateBase
	}
	return Aggregate{&aggregateValue{base: base, tree: base.Tree(), registry: base.Registry()}}, nil
}

func (a Aggregate) Valid() bool { return a.value != nil }

// Base returns the captured base, version zero.
func (a Aggregate) Base() CandidateBase {
	if !a.Valid() {
		return CandidateBase{}
	}
	return a.value.base
}
func (a Aggregate) Version() uint64 {
	if !a.Valid() {
		return 0
	}
	return a.value.version
}
func (a Aggregate) Tree() snapshot.Tree {
	if !a.Valid() {
		return snapshot.Tree{}
	}
	return a.value.tree
}
func (a Aggregate) Registry() Registry {
	if !a.Valid() {
		return Registry{}
	}
	return a.value.registry
}

// PathFacts returns the facts a step's path claims are checked against at this
// aggregate version: the captured model inventory, creation areas and
// protections over the aggregate's current tree and registry. Claims are bound
// to one version, so the Host forms a fresh claim set for each step by
// reserving the task's recorded reservations again on these facts.
func (a Aggregate) PathFacts() PathFacts {
	if !a.Valid() {
		return PathFacts{}
	}
	basis := *a.value.base.value.facts.basis
	basis.tree = a.value.tree
	basis.registry = a.value.registry
	return PathFacts{&basis}
}

// CandidateBase returns the base a step starts from at this aggregate version
// (core rule D3). Candidates captured on it hold exactly the step owner's
// changes.
func (a Aggregate) CandidateBase() CandidateBase {
	if !a.Valid() {
		return CandidateBase{}
	}
	return CandidateBase{&candidateBasis{facts: a.PathFacts(), definitions: a.value.base.value.definitions}}
}

// ReplayResult is the outcome of composing one delivery onto an aggregate. A
// refusal returns no aggregate.
type ReplayResult struct {
	status    CandidateStatus
	findings  []Finding
	aggregate Aggregate
}

func (r ReplayResult) Status() CandidateStatus { return r.status }
func (r ReplayResult) Findings() []Finding     { return slices.Clone(r.findings) }
func (r ReplayResult) Aggregate() (Aggregate, bool) {
	return r.aggregate, r.status == CandidateAdmissible && r.aggregate.Valid()
}

func replayRefusal(code string, owner Owner, path snapshot.Path, message string) ReplayResult {
	return ReplayResult{status: CandidateRefused, findings: []Finding{{Code: code, Owner: owner, Path: path, Message: message}}}
}

// ReplayDelivery composes a candidate's owned changes onto the aggregate (core
// rule D2). The candidate must share the aggregate's model inventory, creation
// areas and protections. Every path it touches, by file or by record, must
// still be registered to the candidate's owner or be newly registered by the
// candidate, and must still hold in the aggregate both the file entry and the
// registry record the candidate was made on. No common base is required,
// because ownership is exclusive. Composition never merges content or chooses
// between owners.
func ReplayDelivery(a Aggregate, c Candidate) ReplayResult {
	if !a.Valid() || !c.Valid() {
		return replayRefusal("replay.unavailable", Owner{}, snapshot.Path{}, "A valid aggregate and candidate are required.")
	}
	owner := c.Owner()
	if !compatibleBasis(a.value.base.value, c.Base().value) {
		return replayRefusal("replay.base_mismatch", owner, snapshot.Path{}, "The candidate was made on a different model inventory, creation areas or protections.")
	}
	made := c.Base()
	current := map[snapshot.Path]Registration{}
	for _, r := range a.value.registry.entries {
		current[r.artifact.path] = r
	}
	madeRecords := map[snapshot.Path]Registration{}
	for _, r := range made.Registry().entries {
		madeRecords[r.artifact.path] = r
	}
	added := map[snapshot.Path]bool{}
	for _, change := range c.value.registryChanges {
		if next, ok := change.After(); ok && next.owner == owner {
			if _, had := madeRecords[change.path]; !had {
				added[change.path] = true
			}
		}
	}
	// File and record of one path are checked together, so two candidates of
	// one owner cannot combine into a pair that neither of them saw.
	for _, path := range c.value.touched {
		before, had := madeRecords[path]
		now, has := current[path]
		if had != has || had && !sameRegistration(before, now) {
			return replayRefusal("replay.registry_changed", owner, path, "The registry record changed after the candidate was made.")
		}
		if has && now.owner != owner || !has && !added[path] {
			return replayRefusal("replay.not_owned", owner, path, "The touched path is not registered to the delivering owner.")
		}
		entry, hadEntry := made.Tree().Lookup(path)
		current, hasEntry := a.value.tree.Lookup(path)
		if hadEntry != hasEntry || hadEntry && entry != current {
			return replayRefusal("replay.content_changed", owner, path, "The path no longer holds the content the candidate was made on.")
		}
	}
	files := map[snapshot.Path]snapshot.Entry{}
	for _, e := range a.value.tree.Entries() {
		files[e.Path()] = e
	}
	for _, change := range c.value.changes {
		if next, ok := change.After(); ok {
			files[change.Path()] = next
		} else {
			delete(files, change.Path())
		}
	}
	records := maps.Clone(current)
	for _, change := range c.value.registryChanges {
		if next, ok := change.After(); ok {
			records[change.path] = next
		} else {
			delete(records, change.path)
		}
	}
	tree, err := snapshot.NewTree(slices.Collect(maps.Values(files)))
	if err != nil {
		return replayRefusal("replay.tree_collision", owner, snapshot.Path{}, "The composed tree has a file/directory conflict.")
	}
	registry, err := NewRegistry(slices.Collect(maps.Values(records)))
	if err != nil {
		return replayRefusal("replay.registry_collision", owner, snapshot.Path{}, "The composed registry has conflicting file ownership.")
	}
	if path, collision := treeRegistryCollision(tree, registry); collision {
		return replayRefusal("replay.file_directory_conflict", owner, path, "The composed tree and registry have a file/directory conflict.")
	}
	next := &aggregateValue{base: a.value.base, version: a.value.version + 1, tree: tree, registry: registry}
	return ReplayResult{status: CandidateAdmissible, aggregate: Aggregate{next}}
}

// compatibleBasis compares everything but the tree and registry, which D2
// checks per changed path.
func compatibleBasis(a, b *candidateBasis) bool {
	if a == nil || b == nil {
		return false
	}
	x, y := *a.facts.basis, *b.facts.basis
	x.tree, y.tree = snapshot.Tree{}, snapshot.Tree{}
	x.registry, y.registry = Registry{}, Registry{}
	return x.same(&y) && maps.Equal(a.definitions, b.definitions)
}
