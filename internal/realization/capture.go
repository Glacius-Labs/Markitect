package realization

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CaptureCandidate compares the entire observed tree with the fixed base. The
// Host/adapter must supply a complete observation over the same capture boundary,
// including protected inputs; no flag or Agent file list can prove completeness.
// Operational state belongs outside that boundary, but supplied state differences
// are rejected, never filtered. The logical proposed Registry comes from Host
// tools, separately from protected registry storage bytes in the observed tree.
//
// The Host must establish actual admission before writing and select the current
// claims. Checking these fixed values cannot prove timing or live authority.
func CaptureCandidate(base CandidateBase, owner Owner, claims PathClaims, observed snapshot.Tree, proposed Registry) CandidateResult {
	if !base.Valid() {
		return CandidateResult{findings: []Finding{{Code: "candidate.base_unavailable", Message: "A complete fixed candidate base is required."}}}
	}
	changes := snapshot.Compare(base.Tree(), observed)
	registryChanges := registryDifference(base.Registry(), proposed)
	var touched []snapshot.Path
	for _, c := range changes {
		touched = append(touched, c.Path())
	}
	for _, c := range registryChanges {
		touched = append(touched, c.path)
	}
	slices.SortFunc(touched, comparePath)
	touched = slices.Compact(touched)
	facts := base.value.facts
	var used []PathClaim
	for _, path := range touched {
		guard := checkPath(facts, owner, path)
		if guard.status != PathAdmissible {
			return CandidateResult{status: CandidateRefused, findings: guard.Findings()}
		}
		storage, _ := snapshot.NewPath(".markitect/registry")
		if pathsOverlap(storage, path) {
			return candidateRefusal("candidate.registry_storage_changed", owner, path, "Registry storage is protected; only logical Host registry proposals may change.")
		}
		before, registered := base.Registry().Lookup(path)
		after, retained := proposed.Lookup(path)
		_, existed := base.Tree().Lookup(path)
		_, present := observed.Lookup(path)
		if registered {
			if before.owner != owner {
				return candidateRefusal("candidate.not_owner", owner, path, "The existing exact path belongs to another owner.")
			}
		} else if existed {
			return candidateRefusal("candidate.intake_required", owner, path, "An existing unregistered file requires explicit intake before it can change.")
		}
		if !present {
			if retained {
				return candidateRefusal("candidate.missing_file", owner, path, "A changed record must have a resulting file, or be explicitly removed for deletion/cleanup.")
			}
			// Removing one's own record is allowed even if its old file or basis was
			// already missing. This does not grant ownership of unexplained files.
			continue
		}
		if !retained {
			return candidateRefusal("candidate.registration_required", owner, path, "A changed resulting file needs a matching current registration.")
		}
		if after.owner != owner {
			return candidateRefusal("candidate.owner_changed", owner, path, "Candidate capture cannot assign or transfer ownership.")
		}
		for _, id := range after.artifact.basis {
			if !base.value.definitions[id] {
				return CandidateResult{status: CandidateRefused, findings: []Finding{{Code: "candidate.dangling_basis", Owner: owner, Path: path, Definition: id, Message: "Every changed/restored record or file needs a current resolvable Definition basis."}}}
			}
		}
		if !registered {
			var matched *PathClaim
			for i := range claims.entries {
				c := &claims.entries[i]
				if c.request.path == path && c.request.owner == owner {
					matched = c
					break
				}
			}
			if matched == nil {
				return candidateRefusal("candidate.claim_required", owner, path, "A new file needs a matching prior Host-admitted path claim.")
			}
			if matched.request.form != after.artifact.form {
				return candidateRefusal("candidate.claim_form_mismatch", owner, path, "The declared artifact form differs from the fixed path claim.")
			}
			_, replay := ReserveNewPath(facts, claims, matched.request)
			if replay.status != PathAdmissible || !replay.repeated {
				return CandidateResult{status: CandidateRefused, findings: replay.Findings()}
			}
			used = append(used, *matched)
		}
	}
	if !owner.valid() || !owner.root && !facts.basis.owners[owner.packageName] {
		return candidateRefusal("candidate.invalid_owner", owner, snapshot.Path{}, "An explicit current local owner is required, including for no-change observations.")
	}
	if path, collision := treeRegistryCollision(observed, proposed); collision {
		return candidateRefusal("candidate.file_directory_conflict", owner, path, "The resulting tree and registry have a file/directory conflict.")
	}
	c := Candidate{&candidateValue{base: base, owner: owner, tree: observed, registry: proposed, changes: changes, registryChanges: registryChanges, touched: touched, claims: used}}
	return CandidateResult{status: CandidateAdmissible, candidate: c}
}

// Tree and Registry constructors separately reject their own prefix collisions.
// Their union must also be unambiguous, including registered-but-missing files.
func treeRegistryCollision(tree snapshot.Tree, registry Registry) (snapshot.Path, bool) {
	files := tree.Entries()
	for _, r := range registry.entries {
		for _, file := range files {
			if r.artifact.path != file.Path() && pathsOverlap(r.artifact.path, file.Path()) {
				return r.artifact.path, true
			}
		}
	}
	return snapshot.Path{}, false
}
