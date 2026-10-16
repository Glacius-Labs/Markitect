package realization

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type compositionValue struct {
	base            CandidateBase
	parts           []Candidate
	tree            snapshot.Tree
	registry        Registry
	changes         []snapshot.Change
	registryChanges []RegistryChange
}

// CompositionPlan fixes an exact base, the supplied structural parts and their
// disjoint result. Zero is invalid. It neither selects accepted deliveries nor
// materializes a workspace, rebases changes, grants authority or passes review.
type CompositionPlan struct{ value *compositionValue }

func (p CompositionPlan) Valid() bool { return p.value != nil }
func (p CompositionPlan) Base() CandidateBase {
	if !p.Valid() {
		return CandidateBase{}
	}
	return p.value.base
}
func (p CompositionPlan) Parts() []Candidate {
	if !p.Valid() {
		return nil
	}
	return slices.Clone(p.value.parts)
}
func (p CompositionPlan) Tree() snapshot.Tree {
	if !p.Valid() {
		return snapshot.Tree{}
	}
	return p.value.tree
}
func (p CompositionPlan) Registry() Registry {
	if !p.Valid() {
		return Registry{}
	}
	return p.value.registry
}
func (p CompositionPlan) Changes() []snapshot.Change {
	if !p.Valid() {
		return nil
	}
	return slices.Clone(p.value.changes)
}
func (p CompositionPlan) RegistryChanges() []RegistryChange {
	if !p.Valid() {
		return nil
	}
	return slices.Clone(p.value.registryChanges)
}

type CompositionResult struct {
	status   CandidateStatus
	findings []Finding
	plan     CompositionPlan
}

func (r CompositionResult) Status() CandidateStatus { return r.status }
func (r CompositionResult) Findings() []Finding     { return slices.Clone(r.findings) }
func (r CompositionResult) Plan() (CompositionPlan, bool) {
	return r.plan, r.status == CandidateAdmissible && r.plan.Valid()
}

func compositionRefusal(code string, path snapshot.Path, message string) CompositionResult {
	return CompositionResult{status: CandidateRefused, findings: []Finding{{Code: code, Path: path, Message: message}}}
}

// ComposeCandidates forms a disjoint union over one exact complete base. The
// Host chooses already accepted work; these values contain no Delivery lifecycle.
// A consumer needing another base requires a separate explicit composition step.
// Any refusal returns no partial plan. No-change parts and an empty union are valid.
func ComposeCandidates(base CandidateBase, parts []Candidate) CompositionResult {
	if !base.Valid() {
		return CompositionResult{findings: []Finding{{Code: "composition.base_unavailable", Message: "A complete fixed base is required."}}}
	}
	parts = slices.Clone(parts)
	for _, part := range parts {
		if !part.Valid() {
			return CompositionResult{findings: []Finding{{Code: "composition.part_unavailable", Message: "Every supplied part must be a fixed structural candidate."}}}
		}
	}
	slices.SortFunc(parts, func(a, b Candidate) int {
		if order := compareOwner(a.Owner(), b.Owner()); order != 0 {
			return order
		}
		ap, bp := a.value.touched, b.value.touched
		if len(ap) == 0 || len(bp) == 0 {
			return len(ap) - len(bp)
		}
		return comparePath(ap[0], bp[0])
	})
	for _, part := range parts {
		if !base.Equal(part.Base()) {
			return compositionRefusal("composition.base_mismatch", snapshot.Path{}, "All parts must bind the exact same tree, registry, model inventories, protections and creation facts.")
		}
	}
	for i, a := range parts {
		for _, b := range parts[i+1:] {
			for _, ap := range a.value.touched {
				for _, bp := range b.value.touched {
					if pathsOverlap(ap, bp) {
						return compositionRefusal("composition.overlap", ap, "Different parts touch the same or conflicting file/directory path, including registry-only effects.")
					}
				}
			}
		}
	}
	files := map[snapshot.Path]snapshot.Entry{}
	for _, e := range base.Tree().Entries() {
		files[e.Path()] = e
	}
	records := map[snapshot.Path]Registration{}
	for _, r := range base.Registry().entries {
		records[r.artifact.path] = r
	}
	for _, part := range parts {
		for _, c := range part.value.changes {
			if next, found := c.After(); found {
				files[c.Path()] = next
			} else {
				delete(files, c.Path())
			}
		}
		for _, c := range part.value.registryChanges {
			if next, found := c.After(); found {
				records[c.path] = next
			} else {
				delete(records, c.path)
			}
		}
	}
	entries := make([]snapshot.Entry, 0, len(files))
	for _, e := range files {
		entries = append(entries, e)
	}
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		return compositionRefusal("composition.tree_collision", snapshot.Path{}, "The combined tree has a file/directory conflict.")
	}
	registrations := make([]Registration, 0, len(records))
	for _, r := range records {
		registrations = append(registrations, r)
	}
	registry, err := NewRegistry(registrations)
	if err != nil {
		return compositionRefusal("composition.registry_collision", snapshot.Path{}, "The combined registry has conflicting file ownership.")
	}
	if path, collision := treeRegistryCollision(tree, registry); collision {
		return compositionRefusal("composition.file_directory_conflict", path, "The combined tree and registry have a file/directory conflict.")
	}
	plan := CompositionPlan{&compositionValue{base: base, parts: parts, tree: tree, registry: registry, changes: snapshot.Compare(base.Tree(), tree), registryChanges: registryDifference(base.Registry(), registry)}}
	return CompositionResult{status: CandidateAdmissible, plan: plan}
}
