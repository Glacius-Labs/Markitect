package host

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const CanonicalEvidenceRefreshAPIVersion = "markitect.canonical/evidence-refresh/v1alpha1"

// CanonicalEvidenceRefreshFinding explains why one explicitly selected active
// record cannot be rebound to the current selected contract and retained bytes.
type CanonicalEvidenceRefreshFinding struct {
	ProjectionID string `json:"projectionId"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}

// CanonicalEvidenceRefreshItem is a proposed new record. The old record and
// verification results remain immutable history.
type CanonicalEvidenceRefreshItem struct {
	ProjectionID  string                   `json:"projectionId"`
	PriorRecordID string                   `json:"priorRecordId"`
	Record        records.ProjectionRecord `json:"record"`
}

// CanonicalEvidenceRefreshProposal binds a cohort of explicitly selected
// retained records to exact immutable source and evidence revisions.
type CanonicalEvidenceRefreshProposal struct {
	APIVersion                   string                            `json:"apiVersion"`
	Digest                       string                            `json:"digest,omitempty"`
	Status                       string                            `json:"status"`
	SourceRevision               string                            `json:"sourceRevision"`
	EvidenceRevision             string                            `json:"evidenceRevision"`
	ConfigDigest                 string                            `json:"configDigest,omitempty"`
	LedgerHead                   string                            `json:"ledgerHead,omitempty"`
	LedgerSelectionDigest        string                            `json:"ledgerSelectionDigest,omitempty"`
	ProjectionIDs                []string                          `json:"projectionIds"`
	UnselectedStaleProjectionIDs []string                          `json:"unselectedStaleProjectionIds"`
	InputPaths                   []string                          `json:"inputPaths"`
	InputDigest                  string                            `json:"inputDigest,omitempty"`
	Items                        []CanonicalEvidenceRefreshItem    `json:"items"`
	Findings                     []CanonicalEvidenceRefreshFinding `json:"findings"`
}

// CanonicalEvidenceRefreshApply reports append-only and active-selection
// progress. A failure after append leaves the appended records as history.
type CanonicalEvidenceRefreshApply struct {
	Status           string                     `json:"status"`
	ProposalDigest   string                     `json:"proposalDigest"`
	Records          []records.ProjectionRecord `json:"records"`
	ActiveRecordIDs  []string                   `json:"activeRecordIds"`
	LedgerHead       string                     `json:"ledgerHead"`
	EvidenceRevision string                     `json:"evidenceRevision"`
}

type canonicalEvidenceRefreshItemBinding struct {
	ProjectionID string                      `json:"projectionId"`
	Prior        records.ProjectionRecord    `json:"prior"`
	Request      canonical.ProjectionRequest `json:"request"`
	Artifacts    []records.ArtifactFact      `json:"artifacts"`
}

type canonicalEvidenceRefreshPayload struct {
	APIVersion            string                                `json:"apiVersion"`
	SourceRevision        string                                `json:"sourceRevision"`
	EvidenceRevision      string                                `json:"evidenceRevision"`
	ConfigDigest          string                                `json:"configDigest"`
	LedgerHead            string                                `json:"ledgerHead"`
	LedgerSelectionDigest string                                `json:"ledgerSelectionDigest"`
	ProjectionIDs         []string                              `json:"projectionIds"`
	UnselectedStale       []string                              `json:"unselectedStaleProjectionIds"`
	InputPaths            []string                              `json:"inputPaths"`
	InputDigest           string                                `json:"inputDigest"`
	Items                 []canonicalEvidenceRefreshItemBinding `json:"items"`
}

// ProposeCanonicalEvidenceRefresh prepares new current-bound records for an
// explicit cohort of active projections whose global source/model binding is
// stale but whose selected contract and owned target facts are unchanged. It
// reads no working-tree target bytes and invokes no Module, Executor, or Verifier.
func ProposeCanonicalEvidenceRefresh(root, sourceRevision, evidenceRevision, configPath string, cfg CanonicalControllerConfig, projectionIDs []string) (CanonicalEvidenceRefreshProposal, error) {
	proposal := CanonicalEvidenceRefreshProposal{
		APIVersion: CanonicalEvidenceRefreshAPIVersion, Status: "blocked",
		SourceRevision: sourceRevision, EvidenceRevision: evidenceRevision,
		ProjectionIDs:                append([]string(nil), projectionIDs...),
		UnselectedStaleProjectionIDs: []string{}, InputPaths: []string{},
		Items: []CanonicalEvidenceRefreshItem{}, Findings: []CanonicalEvidenceRefreshFinding{},
	}
	if err := validateControllerConfig(cfg); err != nil {
		return proposal, err
	}
	if !canonicalRevisionPattern.MatchString(sourceRevision) || !canonicalRevisionPattern.MatchString(evidenceRevision) {
		return proposal, errors.New("evidence refresh requires full immutable source and evidence commit IDs")
	}
	if len(projectionIDs) == 0 || len(projectionIDs) > 128 {
		return proposal, errors.New("evidence refresh requires 1 to 128 explicit Projection IDs")
	}
	selectedIDs := append([]string(nil), projectionIDs...)
	sort.Strings(selectedIDs)
	for i, id := range selectedIDs {
		if id == "" || (i > 0 && selectedIDs[i-1] == id) {
			return proposal, errors.New("evidence refresh Projection IDs must be nonempty and unique")
		}
	}
	proposal.ProjectionIDs = selectedIDs

	fixed, err := LoadSelectedCanonicalSource(root, sourceRevision, configPath, true)
	if err != nil {
		return proposal, err
	}
	if fixed.Snapshot == nil || fixed.Snapshot.Provisional || fixed.Snapshot.ID != sourceRevision || len(fixed.Diagnostics) != 0 {
		return proposal, errors.New("evidence refresh requires a structurally valid exact source revision")
	}
	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return proposal, err
	}
	if store == nil || len(active) == 0 {
		return proposal, errors.New("evidence refresh requires an initialized ledger with active records")
	}
	proposal.LedgerHead, proposal.LedgerSelectionDigest = state.Head, state.ActiveSelection.Digest
	proposal.ConfigDigest, err = digestCanonicalValue(cfg)
	if err != nil {
		return proposal, err
	}
	activeByProjection := make(map[string]records.ProjectionRecord, len(active))
	for _, prior := range active {
		if err := records.ValidateProjectionRecord(prior); err != nil {
			return proposal, err
		}
		activeByProjection[prior.ProjectionID] = prior
		if prior.Revision != sourceRevision || prior.ModelDigest != fixed.Model.Digest {
			if !containsString(selectedIDs, prior.ProjectionID) {
				proposal.UnselectedStaleProjectionIDs = append(proposal.UnselectedStaleProjectionIDs, prior.ProjectionID)
			}
		}
	}
	sort.Strings(proposal.UnselectedStaleProjectionIDs)

	priors := make([]records.ProjectionRecord, 0, len(selectedIDs))
	paths := append([]string(nil), canonicalSourcePaths(fixed)...)
	paths = append(paths, canonicalEvidenceRefreshCheckInputs(cfg, selectedIDs)...)
	for _, id := range selectedIDs {
		prior, ok := activeByProjection[id]
		if !ok {
			return proposal, fmt.Errorf("selected Projection %q is not an active record", id)
		}
		priors = append(priors, prior)
		for _, artifact := range prior.Artifacts {
			paths = append(paths, artifact.Path)
		}
	}
	paths = sortedUniquePaths(paths)
	proposal.InputPaths = paths
	if err := canonicalControllerFixedCheckInputs(root, sourceRevision, evidenceRevision, canonicalEvidenceRefreshCheckInputs(cfg, selectedIDs)); err != nil {
		return proposal, err
	}
	selected, err := source.LoadSelected(root, evidenceRevision, paths)
	if err != nil {
		return proposal, fmt.Errorf("load exact retained evidence: %w", err)
	}
	if selected.Snapshot == nil || selected.Snapshot.Provisional || selected.Snapshot.ID != evidenceRevision {
		return proposal, errors.New("evidence refresh requires the exact immutable evidence revision")
	}
	if err := validateCanonicalSourceUnchanged(fixed, selected.Snapshot); err != nil {
		return proposal, fmt.Errorf("canonical source differs at evidence revision: %w", err)
	}
	proposal.InputDigest = sha256Prefix(selected.Snapshot.Digest())

	bindings := make([]canonicalEvidenceRefreshItemBinding, 0, len(priors))
	requests := make([]canonical.ProjectionRequest, 0, len(priors))
	for _, prior := range priors {
		request, facts, finding := canonicalEvidenceRefreshBinding(root, sourceRevision, fixed, selected.Snapshot, prior)
		if finding != nil {
			proposal.Findings = append(proposal.Findings, *finding)
			continue
		}
		requests = append(requests, request)
		bindings = append(bindings, canonicalEvidenceRefreshItemBinding{ProjectionID: prior.ProjectionID, Prior: prior, Request: request, Artifacts: facts})
	}
	if len(proposal.Findings) > 0 {
		sort.Slice(proposal.Findings, func(i, j int) bool { return proposal.Findings[i].ProjectionID < proposal.Findings[j].ProjectionID })
		return proposal, nil
	}
	payload := canonicalEvidenceRefreshPayload{
		APIVersion: proposal.APIVersion, SourceRevision: sourceRevision, EvidenceRevision: evidenceRevision,
		ConfigDigest: proposal.ConfigDigest, LedgerHead: proposal.LedgerHead,
		LedgerSelectionDigest: proposal.LedgerSelectionDigest, ProjectionIDs: selectedIDs,
		UnselectedStale: proposal.UnselectedStaleProjectionIDs, InputPaths: paths, InputDigest: proposal.InputDigest,
		Items: bindings,
	}
	proposal.Digest, err = digestCanonicalValue(payload)
	if err != nil {
		return proposal, err
	}
	for i, prior := range priors {
		request := requests[i]
		artifacts := make([]records.Artifact, 0, len(prior.Artifacts))
		for _, old := range prior.Artifacts {
			artifacts = append(artifacts, records.Artifact{Path: old.Path, Digest: old.Digest, Mode: old.Mode, Change: records.ChangeRetained})
		}
		record, err := records.NewProjectionRecord(records.ProjectionRecord{
			Origin: prior.Origin, AdoptionRevision: prior.AdoptionRevision, ReviewReference: prior.ReviewReference,
			Revision: sourceRevision, ModelDigest: request.ModelDigest, PlanDigest: proposal.Digest,
			InputSnapshotDigest: proposal.InputDigest, RequestDigest: request.RequestDigest,
			Module:       records.ModuleIdentity{Name: request.ModulePin.Name, Version: request.ModulePin.Version, Digest: request.ModulePin.Digest},
			ProjectionID: prior.ProjectionID, Projector: records.ProjectorIdentity{ID: request.Projector.ID, Version: request.Projector.Version},
			ScopeIDs: requestScopeIDs(request.Definitions), PolicyIDs: canonicalRequestPolicyIDs(request.Policies),
			Artifacts: artifacts, PriorRecordID: prior.ID, State: records.StateMaterializedUnverified,
		})
		if err != nil {
			return proposal, fmt.Errorf("construct retained record for %s: %w", prior.ProjectionID, err)
		}
		proposal.Items = append(proposal.Items, CanonicalEvidenceRefreshItem{ProjectionID: prior.ProjectionID, PriorRecordID: prior.ID, Record: record})
	}
	proposal.Status = "planned"
	return proposal, nil
}

// ApplyCanonicalEvidenceRefresh appends reviewed retained records and replaces
// their active IDs. It never writes target files or runs a verifier.
func ApplyCanonicalEvidenceRefresh(root, configPath string, cfg CanonicalControllerConfig, reviewed CanonicalEvidenceRefreshProposal, expect string, write bool) (CanonicalEvidenceRefreshApply, error) {
	report := CanonicalEvidenceRefreshApply{Status: "refused", ProposalDigest: reviewed.Digest, Records: []records.ProjectionRecord{}, ActiveRecordIDs: []string{}, EvidenceRevision: reviewed.EvidenceRevision}
	if !write || expect == "" || expect != reviewed.Digest || reviewed.Status != "planned" || reviewed.APIVersion != CanonicalEvidenceRefreshAPIVersion {
		return report, errors.New("evidence refresh Apply requires explicit write and exact reviewed proposal digest")
	}
	if err := canonicalEvidenceRefreshHeadMatches(root, reviewed.SourceRevision); err != nil {
		return report, err
	}
	fresh, err := ProposeCanonicalEvidenceRefresh(root, reviewed.SourceRevision, reviewed.EvidenceRevision, configPath, cfg, reviewed.ProjectionIDs)
	if err != nil {
		return report, err
	}
	if !canonicalEvidenceRefreshReviewMatches(fresh, reviewed) {
		return report, errors.New("evidence refresh proposal is stale for source, evidence, selected records, checks or ledger")
	}
	unlock, err := acquireCanonicalControllerLease(cfg)
	if err != nil {
		return report, err
	}
	defer unlock()
	if err := canonicalEvidenceRefreshHeadMatches(root, reviewed.SourceRevision); err != nil {
		return report, err
	}
	fresh, err = ProposeCanonicalEvidenceRefresh(root, reviewed.SourceRevision, reviewed.EvidenceRevision, configPath, cfg, reviewed.ProjectionIDs)
	if err != nil {
		return report, err
	}
	if !canonicalEvidenceRefreshReviewMatches(fresh, reviewed) {
		return report, errors.New("evidence refresh changed while acquiring controller lease")
	}
	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	if store == nil || state.Head != fresh.LedgerHead || state.ActiveSelection.Digest != fresh.LedgerSelectionDigest {
		return report, errors.New("evidence refresh ledger head or active selection changed before append")
	}
	activeByProjection := make(map[string]records.ProjectionRecord, len(active))
	for _, record := range active {
		activeByProjection[record.ProjectionID] = record
	}
	for _, item := range fresh.Items {
		if prior, ok := activeByProjection[item.ProjectionID]; !ok || prior.ID != item.PriorRecordID {
			return report, fmt.Errorf("active record changed for refreshed Projection %s", item.ProjectionID)
		}
		state, err = store.AppendRefresh(state.Head, item.Record)
		if err != nil {
			report.Status, report.LedgerHead = "partial-evidence-refresh", state.Head
			report.Records = append([]records.ProjectionRecord(nil), report.Records...)
			return report, fmt.Errorf("retained evidence records remain as history; append failed: %w", err)
		}
		report.Records = append(report.Records, item.Record)
		report.LedgerHead = state.Head
	}
	replacements := make(map[string]string, len(active))
	for _, prior := range active {
		replacements[prior.ProjectionID] = prior.ID
	}
	for _, item := range fresh.Items {
		replacements[item.ProjectionID] = item.Record.ID
	}
	for _, id := range replacements {
		report.ActiveRecordIDs = append(report.ActiveRecordIDs, id)
	}
	sort.Strings(report.ActiveRecordIDs)
	state, err = store.SelectActive(state.Head, report.ActiveRecordIDs)
	if err != nil {
		report.Status, report.LedgerHead = "partial-evidence-refresh", state.Head
		return report, fmt.Errorf("retained records remain in history; active selection failed: %w", err)
	}
	report.Status, report.LedgerHead = "refreshed", state.Head
	return report, nil
}

func canonicalEvidenceRefreshReviewMatches(fresh, reviewed CanonicalEvidenceRefreshProposal) bool {
	return fresh.Status == "planned" && fresh.Digest == reviewed.Digest && equalCanonicalValue(fresh, reviewed)
}

func canonicalEvidenceRefreshHeadMatches(root, sourceRevision string) error {
	head, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("read Git HEAD for retained evidence refresh: %w", err)
	}
	if strings.TrimSpace(string(head)) != sourceRevision {
		return errors.New("source HEAD changed since reviewed evidence refresh; rerun preview")
	}
	return nil
}

func canonicalEvidenceRefreshBinding(root, sourceRevision string, fixed *CanonicalSource, evidence *snapshot.Snapshot, prior records.ProjectionRecord) (canonical.ProjectionRequest, []records.ArtifactFact, *CanonicalEvidenceRefreshFinding) {
	finding := func(code, message string) (canonical.ProjectionRequest, []records.ArtifactFact, *CanonicalEvidenceRefreshFinding) {
		return canonical.ProjectionRequest{}, nil, &CanonicalEvidenceRefreshFinding{ProjectionID: prior.ProjectionID, Code: code, Message: message}
	}
	if prior.State != records.StateMaterializedUnverified || prior.Revision == sourceRevision && prior.ModelDigest == fixed.Model.Digest {
		return finding("record.not-stale-complete", "selected active record is not a stale complete materialization")
	}
	old, err := LoadSelectedCanonicalSource(root, prior.Revision, fixed.ConfigPath, true)
	if err != nil {
		return finding("prior-source.unavailable", err.Error())
	}
	if old.Snapshot == nil || old.Snapshot.ID != prior.Revision || old.Model.Revision != prior.Revision || old.Model.Digest != prior.ModelDigest || len(old.Diagnostics) != 0 {
		return finding("prior-source.invalid", "prior record does not bind a valid exact canonical source revision")
	}
	identity, err := projectionIdentityFromKey(prior.ProjectionID)
	if err != nil {
		return finding("projection.identity-invalid", err.Error())
	}
	oldProjection, ok := old.Model.Definition(identity)
	if !ok {
		return finding("prior-projection.missing", "prior Projection is absent from its recorded canonical revision")
	}
	newProjection, ok := fixed.Model.Definition(identity)
	if !ok {
		return finding("projection.missing", "Projection is absent from the current canonical source")
	}
	if len(prior.Artifacts) == 0 {
		return finding("artifacts.empty", "complete prior record has no owned artifacts")
	}
	for _, artifact := range prior.Artifacts {
		data, exists := evidence.Files[artifact.Path]
		if !exists {
			return finding("artifact.missing", "owned artifact is absent at evidence revision: "+artifact.Path)
		}
		mode := evidence.Modes[artifact.Path]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		if sha256Prefix(sha256Hex(data)) != artifact.Digest || mode != artifact.Mode {
			return finding("artifact.changed", "owned artifact bytes or mode differ from prior record: "+artifact.Path)
		}
	}
	oldTargets, err := canonicalProjectionTargetFiles(oldProjection, evidence.Files)
	if err != nil {
		return finding("prior-target.invalid", err.Error())
	}
	newTargets, err := canonicalProjectionTargetFiles(newProjection, evidence.Files)
	if err != nil {
		return finding("target.invalid", err.Error())
	}
	oldRequest, err := canonical.BindProjection(old.Model, old.Activation, old.Config.ProjectionBindings, identity, oldTargets)
	if err != nil {
		return finding("prior-request.invalid", err.Error())
	}
	newRequest, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, identity, newTargets)
	if err != nil {
		return finding("request.invalid", err.Error())
	}
	if !canonicalEvidenceRefreshTargetsMatch(oldRequest.TargetFiles, prior.Artifacts) || !canonicalEvidenceRefreshTargetsMatch(newRequest.TargetFiles, prior.Artifacts) {
		return finding("artifact-set.changed", "selected target files do not exactly match the prior owned artifact set")
	}
	if !sameRefreshSelectedContract(oldRequest, newRequest) {
		return finding("selected-contract.changed", "selected Projection, Definitions, Policies, Schemas, edges, Module or Projector changed; rematerialization or escalation is required")
	}
	if !sameRefreshExternalDependencies(old.Model, fixed.Model, oldRequest.Definitions, oldRequest.ExternalEdges, newRequest.ExternalEdges) {
		return finding("external-dependency.changed", "a directly referenced external Definition or its schema changed; refresh requires unchanged dependency semantics")
	}
	if prior.Module.Name != oldRequest.ModulePin.Name || prior.Module.Version != oldRequest.ModulePin.Version || prior.Module.Digest != oldRequest.ModulePin.Digest ||
		prior.Projector.ID != oldRequest.Projector.ID || prior.Projector.Version != oldRequest.Projector.Version ||
		!equalStringSets(prior.ScopeIDs, requestScopeIDs(oldRequest.Definitions)) || !equalStringSets(prior.PolicyIDs, canonicalRequestPolicyIDs(oldRequest.Policies)) {
		return finding("prior-binding.mismatch", "prior record provenance differs from its selected canonical request")
	}
	if prior.Module.Name != newRequest.ModulePin.Name || prior.Module.Version != newRequest.ModulePin.Version || prior.Module.Digest != newRequest.ModulePin.Digest ||
		prior.Projector.ID != newRequest.Projector.ID || prior.Projector.Version != newRequest.Projector.Version {
		return finding("runtime-binding.changed", "current Module or Projector binding differs from the prior record")
	}
	facts := make([]records.ArtifactFact, 0, len(prior.Artifacts))
	for _, artifact := range prior.Artifacts {
		facts = append(facts, records.ArtifactFact{Path: artifact.Path, Role: records.RoleProjectionTarget, Digest: artifact.Digest, Mode: artifact.Mode})
	}
	if _, err := records.TargetSnapshotDigest(facts); err != nil {
		return finding("artifact-facts.invalid", err.Error())
	}
	return newRequest, facts, nil
}

func sameRefreshExternalDependencies(oldModel, currentModel core.Model, selected []core.Definition, oldEdges, currentEdges []core.Edge) bool {
	if !equalCanonicalValue(oldEdges, currentEdges) {
		return false
	}
	selectedIDs := map[string]bool{}
	for _, definition := range selected {
		selectedIDs[definition.Identity().Key()] = true
	}
	oldDefinitions, currentDefinitions := map[string]core.Definition{}, map[string]core.Definition{}
	for _, definition := range oldModel.Definitions {
		oldDefinitions[definition.Identity().Key()] = definition
	}
	for _, definition := range currentModel.Definitions {
		currentDefinitions[definition.Identity().Key()] = definition
	}
	oldSchemas, currentSchemas := map[string]core.Schema{}, map[string]core.Schema{}
	for _, schema := range oldModel.Schemas {
		oldSchemas[schema.APIVersion] = schema
	}
	for _, schema := range currentModel.Schemas {
		currentSchemas[schema.APIVersion] = schema
	}
	externalIDs := map[string]bool{}
	for _, edge := range oldEdges {
		var external string
		switch {
		case selectedIDs[edge.From] && !selectedIDs[edge.To]:
			external = edge.To
		case selectedIDs[edge.To] && !selectedIDs[edge.From]:
			external = edge.From
		default:
			return false
		}
		externalIDs[external] = true
	}
	for id := range externalIDs {
		oldDefinition, oldOK := oldDefinitions[id]
		currentDefinition, currentOK := currentDefinitions[id]
		if !oldOK || !currentOK || !equalCanonicalValue(oldDefinition, currentDefinition) {
			return false
		}
		oldSchema, oldOK := oldSchemas[oldDefinition.APIVersion]
		currentSchema, currentOK := currentSchemas[currentDefinition.APIVersion]
		if !oldOK || !currentOK || !equalCanonicalValue(oldSchema, currentSchema) {
			return false
		}
	}
	return true
}

func sameRefreshSelectedContract(old, current canonical.ProjectionRequest) bool {
	old.Revision, current.Revision = "", ""
	old.ModelDigest, current.ModelDigest = "", ""
	old.RequestDigest, current.RequestDigest = "", ""
	old.TargetFiles, current.TargetFiles = nil, nil
	old.TargetDigests, current.TargetDigests = nil, nil
	return equalCanonicalValue(old, current)
}

func canonicalEvidenceRefreshCheckInputs(cfg CanonicalControllerConfig, projectionIDs []string) []string {
	selected := map[string]bool{}
	for _, id := range projectionIDs {
		selected[id] = true
	}
	paths := append([]string(nil), cfg.CheckInputs...)
	for _, scope := range cfg.AssuranceScopes {
		if selected[scope.ProjectionID] {
			paths = append(paths, scope.CheckInputs...)
		}
	}
	return sortedUniquePaths(paths)
}

func canonicalEvidenceRefreshTargetsMatch(files map[string][]byte, artifacts []records.Artifact) bool {
	if len(files) != len(artifacts) {
		return false
	}
	for _, artifact := range artifacts {
		if _, ok := files[artifact.Path]; !ok {
			return false
		}
	}
	return true
}
