package realization

import (
	"errors"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func (c *DefinitionCensus) add(status CensusStatus, code string, path snapshot.Path, id model.Identity, message string) {
	if status == CensusInvalid || c.status != CensusInvalid {
		c.status = status
	}
	c.findings = append(c.findings, Finding{Code: code, Path: path, Definition: id, Owner: c.owner, Message: message})
}

func (c DefinitionCensus) finish() DefinitionCensus {
	slices.SortFunc(c.findings, compareFinding)
	slices.SortFunc(c.expected, compareIdentity)
	slices.SortFunc(c.missing, compareIdentity)
	slices.SortFunc(c.unresolved, compareIdentity)
	c.unresolved = slices.Compact(c.unresolved)
	return c
}

func checkCensus(compiled model.Result, base CandidateBase, tree snapshot.Tree, registry Registry, result DefinitionCensus, classifications []Classification, rows []DefinitionAssessment) DefinitionCensus {
	return checkInventoryCensus(inventoryFromResult(compiled), base, tree, registry, result, classifications, rows)
}

func checkInventoryCensus(inventory modelInventory, base CandidateBase, tree snapshot.Tree, registry Registry, result DefinitionCensus, classifications []Classification, rows []DefinitionAssessment) DefinitionCensus {
	result.classifications = slices.Clone(classifications)
	result.assessments = copyAssessments(rows)
	if !base.Valid() || !result.candidate.Valid() && !result.composition.Valid() {
		result.add(CensusIncomplete, "census.subject_unavailable", snapshot.Path{}, model.Identity{}, "An exact structural candidate or composition is required.")
		return result.finish()
	}
	result.status = CensusStructurallyComplete
	result.accounting = accountFiles(inventory, tree, registry, result.classifications)
	// Invalid/duplicate/missing classifications contradict the supplied capture.
	// A stale registry record conflicting with a classification instead makes that
	// artifact uncitable; unrelated old registry problems are not a global gate.
	for _, f := range result.accounting.findings {
		if strings.HasPrefix(f.Code, "classification.") {
			result.add(CensusInvalid, f.Code, f.Path, f.Definition, f.Message)
		}
	}
	matching, err := newCandidateBase(inventory, base.value.facts)
	if err != nil {
		status := CensusInvalid
		if errors.Is(err, ErrIncompleteCandidateBase) {
			status = CensusIncomplete
		}
		result.add(status, "census.model_unavailable", snapshot.Path{}, model.Identity{}, "The complete compiled model must match the subject's fixed basis.")
		return result.finish()
	}
	if !base.Equal(matching) {
		result.add(CensusInvalid, "census.model_mismatch", snapshot.Path{}, model.Identity{}, "Compiled model and Definition/Package inventories differ from the exact subject basis.")
		return result.finish()
	}
	result.basis = matching
	if !result.project && (!result.owner.valid() || !result.owner.root && !matching.value.facts.basis.owners[result.owner.packageName]) {
		result.add(CensusInvalid, "census.invalid_owner", snapshot.Path{}, model.Identity{}, "Own scope requires an explicit root or current local Package owner.")
		return result.finish()
	}
	local := inventory.local
	expected := map[model.Identity]bool{}
	for id := range local {
		if result.project || !result.owner.root && id.Package == result.owner.packageName {
			expected[id] = true
			result.expected = append(result.expected, id)
		}
	}
	eligible := map[snapshot.Path]bool{}
	for _, paths := range result.accounting.links {
		for _, path := range paths {
			eligible[path] = true
		}
	}
	seen := map[model.Identity]bool{}
	for _, row := range result.assessments {
		id := row.Definition
		if seen[id] {
			result.add(CensusInvalid, "census.duplicate_definition", snapshot.Path{}, id, "Each expected Definition needs exactly one assessment.")
		}
		seen[id] = true
		if !expected[id] {
			result.add(CensusInvalid, "census.unexpected_definition", snapshot.Path{}, id, "The assessment is not for an own local Definition in this exact census scope.")
		}
		result.checkRow(row, local, eligible, registry, matching.value.facts)
	}
	for id := range expected {
		if !seen[id] {
			result.missing = append(result.missing, id)
			result.add(CensusIncomplete, "census.missing_definition", snapshot.Path{}, id, "An expected local Definition has no assessment.")
		}
	}
	return result.finish()
}

func (result *DefinitionCensus) checkRow(row DefinitionAssessment, local map[model.Identity]bool, eligible map[snapshot.Path]bool, registry Registry, facts PathFacts) {
	id := row.Definition
	positive := false
	switch row.Disposition {
	case Realized, Constrains, Vocabulary:
		positive = true
	case Unresolved:
		result.unresolved = append(result.unresolved, id)
		result.add(CensusIncomplete, "census.unresolved", snapshot.Path{}, id, "Unresolved work cannot make a structurally complete census.")
	default:
		result.add(CensusInvalid, "census.invalid_disposition", snapshot.Path{}, id, "Disposition must be realized, constrains, vocabulary or unresolved.")
	}
	if strings.TrimSpace(row.Reason) == "" {
		result.add(CensusIncomplete, "census.reason_required", snapshot.Path{}, id, "An explanation is required; its substance remains reviewable proposal data.")
	}
	if positive && len(row.Evidence) == 0 {
		result.add(CensusIncomplete, "census.evidence_required", snapshot.Path{}, id, "A positive proposal requires explained Host evidence references.")
	}
	evidence := map[string]bool{}
	for _, use := range row.Evidence {
		if strings.TrimSpace(use.Ref) == "" || strings.TrimSpace(use.Relevance) == "" {
			result.add(CensusIncomplete, "census.evidence_incomplete", snapshot.Path{}, id, "Every supplied evidence reference needs a nonblank handle and relevance explanation.")
		}
		if evidence[use.Ref] {
			result.add(CensusInvalid, "census.duplicate_evidence", snapshot.Path{}, id, "Repeated evidence handles cannot count as different support.")
		}
		evidence[use.Ref] = true
	}
	uses := map[model.Identity]bool{}
	for _, use := range row.Uses {
		if uses[use] {
			result.add(CensusInvalid, "census.duplicate_use", snapshot.Path{}, id, "A concrete-use reference is repeated.")
		}
		uses[use] = true
		if !local[use] {
			result.add(CensusInvalid, "census.invalid_use", snapshot.Path{}, id, "A concrete-use reference must resolve in the selected local model.")
		}
	}
	artifacts := map[snapshot.Path]bool{}
	direct := false
	storage, _ := snapshot.NewPath(".markitect/registry")
	for _, path := range row.Artifacts {
		if artifacts[path] {
			result.add(CensusInvalid, "census.duplicate_artifact", path, id, "An artifact reference is repeated.")
		}
		artifacts[path] = true
		entry, registered := registry.Lookup(path)
		if !eligible[path] || !registered || pathsOverlap(storage, path) || checkPath(facts, entry.owner, path).status != PathAdmissible {
			result.add(CensusInvalid, "census.invalid_artifact", path, id, "Cited artifacts must be present, valid, unconflicted registrations outside protected/input/ignored state.")
			continue
		}
		if slices.Contains(entry.artifact.basis, id) {
			direct = true
		}
	}
	if row.Disposition == Realized {
		if len(row.Artifacts) == 0 {
			result.add(CensusIncomplete, "census.realized_artifact_required", snapshot.Path{}, id, "Realized requires an actual artifact reference.")
		} else if !direct {
			result.add(CensusInvalid, "census.realized_basis_mismatch", snapshot.Path{}, id, "At least one cited artifact must directly register this Definition as its basis.")
		}
	}
	if row.Disposition == Constrains && len(row.Artifacts) == 0 && len(row.Uses) == 0 {
		result.add(CensusIncomplete, "census.constrained_realization_required", snapshot.Path{}, id, "Constrains must name affected artifacts or proposed concrete local uses with evidence.")
	}
}
