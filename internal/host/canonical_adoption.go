package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const CanonicalAdoptionPlanAPIVersion = "markitect.canonical-adoption-plan/v1alpha1"

// Selection is an explicit owner-supplied artifact match and review claim.
// It grants no authority to infer source semantics or inspect additional paths.
type CanonicalAdoptionSelection struct {
	ActiveRecords   []records.ProjectionRecord `json:"activeRecords"`
	Artifacts       []string                   `json:"artifacts"`
	ReviewReference string                     `json:"reviewReference"`
}
type CanonicalAdoptionPlan struct {
	UnmatchedArtifacts []string                 `json:"unmatchedArtifacts"`
	APIVersion         string                   `json:"apiVersion"`
	PlanDigest         string                   `json:"planDigest"`
	EvidenceRevision   string                   `json:"evidenceRevision"`
	Record             records.ProjectionRecord `json:"record"`
}
type CanonicalAdoption struct {
	UnmatchedArtifacts []string                  `json:"unmatchedArtifacts"`
	Record             *records.ProjectionRecord `json:"record,omitempty"`
	Verification       CanonicalVerification     `json:"verification"`
}

// PrepareCanonicalAdoption matches existing exact bytes without executing a
// renderer, candidate generator or check. The prospective record is not active
// ownership, acceptance or verified evidence. Both snapshots must be fixed.
func PrepareCanonicalAdoption(fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection) (CanonicalAdoptionPlan, error) {
	plan := CanonicalAdoptionPlan{}
	if fixed == nil || fixed.Snapshot == nil || target == nil || fixed.Snapshot.Provisional || target.Provisional || !canonicalRevisionPattern.MatchString(fixed.Snapshot.ID) || !canonicalRevisionPattern.MatchString(target.ID) {
		return plan, errors.New("adoption requires full fixed canonical and evidence revisions")
	}
	if fixed.Model.Revision != fixed.Snapshot.ID || len(fixed.Diagnostics) != 0 {
		return plan, errors.New("adoption requires a structurally valid model bound to its fixed source")
	}
	if err := validateCanonicalSourceUnchanged(fixed, target); err != nil {
		return plan, err
	}
	if strings.TrimSpace(selection.ReviewReference) == "" || len(selection.ReviewReference) > 4096 || !utf8.ValidString(selection.ReviewReference) || strings.ContainsRune(selection.ReviewReference, 0) {
		return plan, errors.New("adoption requires a bounded owner-supplied review reference; it is not authentication")
	}
	if selection.ActiveRecords == nil {
		return plan, errors.New("adoption requires an explicit activeRecords array, even when empty")
	}
	if len(selection.ActiveRecords) > 4096 {
		return plan, errors.New("adoption activeRecords exceeds 4096 records")
	}
	if len(selection.Artifacts) == 0 || len(selection.Artifacts) > 4096 {
		return plan, errors.New("adoption requires 1..4096 exact selected artifact paths")
	}
	projection, ok := fixed.Model.Definition(identity)
	if !ok {
		return plan, errors.New("adoption Projection is absent from canonical intent")
	}
	targets, err := canonicalProjectionTargetFiles(projection, target.Files)
	if err != nil {
		return plan, err
	}
	request, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, identity, targets)
	if err != nil {
		return plan, err
	}
	names, escalations, err := selectedCanonicalChecks(request.Projector, true, fixed.Config.Checks)
	if err != nil {
		return plan, err
	}
	if len(escalations) != 0 {
		return plan, &VerifyError{Kind: "incomplete-evidence", Err: errors.New(escalations[0].Message)}
	}
	protected := map[string]bool{}
	for _, name := range canonicalSourcePaths(fixed) {
		protected[name] = true
	}
	aliases := map[string]string{}
	targetNames := make([]string, 0, len(targets))
	for name := range targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)
	for _, name := range targetNames {
		key := strings.ToLower(name)
		if prior, exists := aliases[key]; exists && prior != name {
			return plan, fmt.Errorf("target path alias collision %q and %q", prior, name)
		}
		aliases[key] = name
	}
	selected := append([]string(nil), selection.Artifacts...)
	sort.Strings(selected)
	artifacts := make([]records.Artifact, 0, len(selected))
	for i, name := range selected {
		if err := projectionengine.ValidateRelativePath(name); err != nil {
			return plan, err
		}
		if i > 0 && selected[i-1] == name {
			return plan, fmt.Errorf("duplicate adoption artifact %q", name)
		}
		if protected[name] {
			return plan, fmt.Errorf("adoption artifact %q overlaps canonical source", name)
		}
		data, exists := targets[name]
		if !exists {
			return plan, fmt.Errorf("adoption artifact %q is absent or outside the selected representation target", name)
		}
		if !utf8.Valid(data) || strings.IndexByte(string(data), 0) >= 0 {
			return plan, fmt.Errorf("adoption artifact %q must be regular UTF-8 text without NUL", name)
		}
		mode := target.Modes[name]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		artifacts = append(artifacts, records.Artifact{Path: name, Digest: sha256Prefix(sha256Hex(data)), Mode: mode, Change: records.ChangeRetained})
	}
	scope, policies := []string{}, []string{}
	for _, d := range request.Definitions {
		scope = append(scope, d.Identity().Key())
	}
	for _, d := range request.Policies {
		policies = append(policies, d.Identity().Key())
	}
	sort.Strings(scope)
	sort.Strings(policies)
	payload := struct {
		APIVersion, CanonicalRevision, EvidenceRevision, ModelDigest, RequestDigest, InputSnapshotDigest, ReviewReference string
		Artifacts                                                                                                         []records.Artifact
		CheckDigests                                                                                                      []string
	}{APIVersion: CanonicalAdoptionPlanAPIVersion, CanonicalRevision: fixed.Snapshot.ID, EvidenceRevision: target.ID, ModelDigest: request.ModelDigest, RequestDigest: request.RequestDigest, InputSnapshotDigest: sha256Prefix(target.Digest()), ReviewReference: selection.ReviewReference, Artifacts: artifacts, CheckDigests: []string{}}
	for _, check := range selectedAuthoringChecks(names, fixed.Config.Checks) {
		payload.CheckDigests = append(payload.CheckDigests, CanonicalCheckEvidenceDigest(check, target))
	}
	activeIDs := []string{}
	activeProjections := map[string]bool{}
	for _, active := range selection.ActiveRecords {
		if err := records.ValidateProjectionRecord(active); err != nil {
			return plan, fmt.Errorf("invalid active ownership record: %w", err)
		}
		if activeProjections[active.ProjectionID] || active.ProjectionID == identity.Key() {
			return plan, errors.New("adoption cannot replace an active representation owner implicitly")
		}
		activeProjections[active.ProjectionID] = true
		activeIDs = append(activeIDs, active.ID)
	}
	sort.Strings(activeIDs)
	envelope := struct {
		Plan            any
		ActiveRecordIDs []string
	}{payload, activeIDs}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return plan, err
	}
	digest := sha256Prefix(sha256Hex(encoded))
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Origin: records.OriginAdopted, AdoptionRevision: target.ID, ReviewReference: selection.ReviewReference,
		Revision: fixed.Snapshot.ID, ModelDigest: request.ModelDigest, PlanDigest: digest, InputSnapshotDigest: payload.InputSnapshotDigest, RequestDigest: request.RequestDigest,
		Module:    records.ModuleIdentity{Name: request.ModulePin.Name, Version: request.ModulePin.Version, Digest: request.ModulePin.Digest},
		Projector: records.ProjectorIdentity{ID: request.Projector.ID, Version: request.Projector.Version}, ProjectionID: identity.Key(), ScopeIDs: scope, PolicyIDs: policies,
		Artifacts: artifacts, State: records.StateMaterializedUnverified,
	})
	if err != nil {
		return plan, err
	}
	// Combining the proposed owner with supplied active owners must not collide.
	ownership := append([]records.ProjectionRecord(nil), selection.ActiveRecords...)
	ownership = append(ownership, record)
	inventory := make([]records.ArtifactFact, 0, len(artifacts))
	for _, artifact := range artifacts {
		inventory = append(inventory, records.ArtifactFact{Path: artifact.Path, Role: records.RoleProjectionTarget, Digest: artifact.Digest, Mode: artifact.Mode})
	}
	if _, err := records.BuildOwnershipIndex(ownership, inventory); err != nil {
		return plan, fmt.Errorf("adoption ownership conflict: %w", err)
	}
	unmatched := []string{}
	selectedSet := map[string]bool{}
	for _, name := range selected {
		selectedSet[name] = true
	}
	for _, name := range targetNames {
		if !selectedSet[name] {
			unmatched = append(unmatched, name)
		}
	}
	return CanonicalAdoptionPlan{APIVersion: CanonicalAdoptionPlanAPIVersion, PlanDigest: digest, EvidenceRevision: target.ID, Record: record, UnmatchedArtifacts: unmatched}, nil
}

// AdoptCanonicalProjection approves the freshly bound adoption identity, then
// verifies the existing representation using the normal immutable verifier.
// No record is returned on failure and no target or ledger is written. Owner
// review and semantic sufficiency remain supplied claims, not tool proof.
func AdoptCanonicalProjection(fixed *CanonicalSource, target *snapshot.Snapshot, identity core.DefinitionIdentity, selection CanonicalAdoptionSelection, acceptedDigest string, verifier records.VerifierIdentity) (CanonicalAdoption, error) {
	result := CanonicalAdoption{}
	plan, err := PrepareCanonicalAdoption(fixed, target, identity, selection)
	if err != nil {
		return result, err
	}
	if acceptedDigest == "" || acceptedDigest != plan.PlanDigest {
		return result, errors.New("adoption approval does not match the freshly bound plan digest")
	}
	result.UnmatchedArtifacts = append([]string{}, plan.UnmatchedArtifacts...)
	result.Verification, err = VerifyCanonicalProjection(fixed, target, plan.Record, verifier)
	if err != nil {
		return result, err
	}
	if result.Verification.Result.Outcome != records.OutcomePassed {
		return result, errors.New("adoption requires passed declared fixed checks")
	}
	result.Record = &plan.Record
	return result, nil
}
