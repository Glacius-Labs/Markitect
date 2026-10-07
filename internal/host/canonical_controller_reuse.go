package host

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/assurance"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

type canonicalControllerVerificationReuse struct {
	cfg                CanonicalControllerConfig
	results            map[string]records.VerificationResult
	active             []records.ProjectionRecord
	scopes             []CanonicalAssuranceScope
	roots              []string
	scopeIndex         map[string]CanonicalAssuranceScope
	recordByScope      map[string]records.ProjectionRecord
	recordByProjection map[string]records.ProjectionRecord
	verifier           records.VerifierIdentity
	fingerprint        string
	configDigest       string
}

type canonicalProjectionVerificationBinding struct {
	RequestDigest string
	Artifacts     []records.Artifact
	Repair        *canonicalControllerRepairEvidence
}

type canonicalControllerRepairEvidence struct {
	RecordID string
	ResultID string
	Findings []records.VerificationFinding
}

func newCanonicalControllerVerificationReuse(cfg CanonicalControllerConfig, active []records.ProjectionRecord, results []records.VerificationResult, configDigest string) (*canonicalControllerVerificationReuse, error) {
	if !cfg.AuditAll || len(results) == 0 || len(active) == 0 {
		return nil, nil
	}
	scopes, roots, recordsByProjection, err := canonicalControllerVerificationNodes(cfg, active)
	if err != nil {
		return nil, err
	}
	fingerprint, err := agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if err != nil {
		return nil, fmt.Errorf("fingerprint configured Verifier for cached evidence: %w", err)
	}
	latest := make(map[string]records.VerificationResult, len(active))
	for _, result := range results {
		latest[result.RecordID] = result
	}
	scopeIndex := make(map[string]CanonicalAssuranceScope, len(scopes))
	recordByScope := make(map[string]records.ProjectionRecord, len(scopes))
	for _, scope := range scopes {
		scopeIndex[scope.ID] = scope
		recordByScope[scope.ID] = recordsByProjection[scope.ProjectionID]
	}
	return &canonicalControllerVerificationReuse{
		cfg: cfg, results: latest, active: append([]records.ProjectionRecord(nil), active...),
		scopes: scopes, roots: roots, scopeIndex: scopeIndex, recordByScope: recordByScope, recordByProjection: recordsByProjection,
		verifier:    records.VerifierIdentity{ID: "markitect-agentexec/verifier", Version: cfg.Verifier.ProviderVersion, Digest: fingerprint},
		fingerprint: fingerprint, configDigest: configDigest,
	}, nil
}

func (reuse *canonicalControllerVerificationReuse) bindings(root string, fixed *CanonicalSource, active []records.ProjectionRecord, observed *snapshot.Snapshot) map[string]canonicalProjectionVerificationBinding {
	bindings := map[string]canonicalProjectionVerificationBinding{}
	if reuse == nil || !reuse.cfg.AuditAll || fixed == nil || fixed.Snapshot == nil || observed == nil {
		return bindings
	}
	items := map[string]canonicalControllerPreparedVerification{}
	current := map[string]records.Freshness{}
	currentMatchesEvidence := map[string]bool{}
	nodes := make([]assurance.Node, 0, len(reuse.scopes))
	for _, scope := range reuse.scopes {
		record, ok := reuse.recordByScope[scope.ID]
		if !ok {
			return bindings
		}
		result, hasResult := reuse.results[record.ID]
		var item canonicalControllerPreparedVerification
		var fresh records.Freshness
		var matched bool
		if hasResult {
			item, fresh, matched, ok = reuse.prepareScope(root, fixed, record, scope, result, observed)
		}
		if !hasResult || !ok {
			fresh = reuse.unavailableFreshness(record)
		} else {
			items[scope.ID] = item
			currentMatchesEvidence[scope.ID] = matched
		}
		current[scope.ID] = fresh
		nodes = append(nodes, assurance.Node{ID: scope.ID, ScopeIDs: append([]string(nil), record.ScopeIDs...), Children: append([]string(nil), scope.Children...), RequiredChecks: append([]records.CheckIdentity(nil), fresh.Checks...)})
	}
	graph := assurance.Input{RootIDs: append([]string(nil), reuse.roots...), Nodes: nodes}
	accepted := map[string]bool{}
	repairs := map[string]canonicalControllerRepairEvidence{}
	_, err := assurance.Execute(context.Background(), assurance.RunInput{Graph: graph, Current: current}, func(_ context.Context, in assurance.NodeRunInput) (assurance.NodeRunOutput, error) {
		item, ok := items[in.Node.ID]
		if !ok || !currentMatchesEvidence[in.Node.ID] {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verification provenance is unavailable or stale"}, nil
		}
		result := reuse.results[item.record.ID]
		if result.ControllerConfigDigest != reuse.configDigest ||
			result.EvidenceRevision != item.target.ID || result.EvidenceSnapshotDigest != item.evidenceDigest {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verification result does not bind the current evidence"}, nil
		}
		request, err := canonicalControllerVerifierRequest(reuse.cfg, item, in, result.EvidenceRevision)
		if err != nil {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verifier request could not be reconstructed"}, nil
		}
		inputDigest, err := canonicalControllerVerifierInputDigest(request)
		if err != nil || inputDigest != result.ControllerVerifierInputDigest {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verifier input differs from current direct evidence"}, nil
		}
		if err := records.ValidateVerificationFreshness(result, item.record, in.Current); err != nil {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verification freshness check failed"}, nil
		}
		if result.Outcome == records.OutcomePassed {
			accepted[in.Node.ID] = true
		} else if result.Outcome == records.OutcomeFailed && canonicalControllerResultSupportsRepair(result) && canonicalControllerChildrenPassed(in.Children) {
			repairs[in.Node.ID] = canonicalControllerRepairEvidence{
				RecordID: result.RecordID, ResultID: result.ID,
				Findings: append([]records.VerificationFinding(nil), result.SemanticFindings...),
			}
		} else {
			return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunSkipped, Reason: "cached verifier result is not eligible for scoped repair"}, nil
		}
		return assurance.NodeRunOutput{NodeID: in.Node.ID, Disposition: assurance.RunCompleted, Record: item.record, Result: result}, nil
	})
	if err != nil {
		return bindings
	}
	for scopeID, item := range items {
		if !accepted[scopeID] && repairs[scopeID].RecordID == "" {
			continue
		}
		binding := canonicalProjectionVerificationBinding{
			RequestDigest: item.request.RequestDigest,
			Artifacts:     append([]records.Artifact(nil), item.record.Artifacts...),
		}
		if repair, ok := repairs[scopeID]; ok {
			binding.Repair = &repair
		}
		bindings[item.record.ProjectionID] = binding
	}
	return bindings
}

func canonicalControllerResultSupportsRepair(result records.VerificationResult) bool {
	if result.Outcome != records.OutcomeFailed || len(result.SemanticFindings) == 0 {
		return false
	}
	agentCheckFailed := false
	for _, check := range result.Checks {
		if check.ID == canonicalControllerAgentCheckID {
			if check.Outcome != records.CheckFailed {
				return false
			}
			agentCheckFailed = true
			continue
		}
		if check.Outcome != records.CheckPassed {
			return false
		}
	}
	return agentCheckFailed
}

func canonicalControllerChildrenPassed(children []assurance.ChildRunInput) bool {
	for _, child := range children {
		if child.Result.Outcome != records.OutcomePassed {
			return false
		}
	}
	return true
}

func (reuse *canonicalControllerVerificationReuse) unavailableFreshness(record records.ProjectionRecord) records.Freshness {
	digest := sha256Prefix(sha256Hex([]byte("canonical cached verification unavailable:" + record.ID)))
	return records.Freshness{
		Revision: record.Revision, RecordID: record.ID, ModelDigest: record.ModelDigest,
		TargetSnapshotDigest: record.TargetSnapshotDigest, Verifier: reuse.verifier,
		Checks: []records.CheckIdentity{{ID: "cached-evidence-unavailable", Version: "v1", Digest: digest}},
	}
}

func (reuse *canonicalControllerVerificationReuse) prepareScope(root string, fixed *CanonicalSource, record records.ProjectionRecord, scope CanonicalAssuranceScope, result records.VerificationResult, observed *snapshot.Snapshot) (canonicalControllerPreparedVerification, records.Freshness, bool, bool) {
	failed := canonicalControllerPreparedVerification{}
	if result.EvidenceRevision == "" || result.EvidenceSnapshotDigest == "" || result.ControllerConfigDigest == "" || result.ControllerVerifierInputDigest == "" ||
		result.Revision != fixed.Snapshot.ID || result.RecordID != record.ID || record.Revision != fixed.Snapshot.ID || result.ModelDigest != fixed.Model.Digest {
		return failed, records.Freshness{}, false, false
	}
	request, evidence, err := loadCanonicalControllerScopeRequest(root, fixed.Snapshot.ID, result.EvidenceRevision, reuse.cfg, fixed, record, scope, reuse.scopeIndex, reuse.recordByProjection)
	if err != nil || evidence == nil || sha256Prefix(evidence.Digest()) != result.EvidenceSnapshotDigest {
		return failed, records.Freshness{}, false, false
	}
	currentTargetFiles, err := canonicalProjectionTargetFiles(request.Projection, observed.Files)
	if err != nil {
		return failed, records.Freshness{}, false, false
	}
	currentRequest, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, request.Projection.Identity(), currentTargetFiles)
	if err != nil || !sameCanonicalProjectionIntent(request, currentRequest) {
		return failed, records.Freshness{}, false, false
	}
	paths, err := canonicalControllerScopeEvidencePaths(fixed, reuse.cfg, record, scope, reuse.recordByProjection)
	if err != nil {
		return failed, records.Freshness{}, false, false
	}
	currentMatches := canonicalControllerObservedEvidenceMatches(paths, evidence, observed)
	checkInputs := canonicalControllerCheckInputs(reuse.cfg)
	fixedChecks, err := source.LoadSelected(root, fixed.Snapshot.ID, checkInputs)
	if err != nil || fixedChecks.Snapshot == nil || fixedChecks.Snapshot.ID != fixed.Snapshot.ID {
		return failed, records.Freshness{}, false, false
	}
	for _, name := range checkInputs {
		if !bytes.Equal(fixedChecks.Snapshot.Files[name], observed.Files[name]) || fixedChecks.Snapshot.Modes[name] != observed.Modes[name] {
			currentMatches = false
		}
	}
	for _, artifact := range record.Artifacts {
		data, present := observed.Files[artifact.Path]
		mode := observed.Modes[artifact.Path]
		if !present || mode != artifact.Mode || sha256Prefix(sha256Hex(data)) != artifact.Digest {
			currentMatches = false
		}
	}
	artifactFacts := make([]records.ArtifactFact, 0, len(record.Artifacts))
	for _, artifact := range record.Artifacts {
		artifactFacts = append(artifactFacts, records.ArtifactFact{Path: artifact.Path, Role: records.RoleProjectionTarget, Digest: artifact.Digest, Mode: artifact.Mode})
	}
	targetDigest, err := records.TargetSnapshotDigest(artifactFacts)
	if err != nil || targetDigest != record.TargetSnapshotDigest {
		return failed, records.Freshness{}, false, false
	}
	checks := reuse.fixedCheckResults(fixed, request, scope, evidence)
	if checks == nil {
		return failed, records.Freshness{}, false, false
	}
	artifacts, err := canonicalControllerVerificationArtifacts(reuse.cfg, record, scope, reuse.recordByProjection, evidence, reuse.cfg.CheckInputs)
	if err != nil {
		return failed, records.Freshness{}, false, false
	}
	contextModel, err := BuildCanonicalAgentContext(fixed.Model, request, reuse.cfg.ReferenceDepth)
	if err != nil {
		return failed, records.Freshness{}, false, false
	}
	controllerConfigDigest := reuse.configDigest
	agentCheckDigest, err := digestCanonicalValue(canonicalVerifierBinding{
		Fingerprint: reuse.fingerprint, ControllerConfigDigest: controllerConfigDigest,
		SourceRevision: fixed.Snapshot.ID, EvidenceRevision: result.EvidenceRevision,
		ModelDigest: record.ModelDigest, RecordID: record.ID, ProjectionID: record.ProjectionID,
		ScopeIDs: append([]string(nil), record.ScopeIDs...), PolicyIDs: append([]string(nil), record.PolicyIDs...),
		EvidenceDigest: result.EvidenceSnapshotDigest, RequiredChecks: checkIdentities(checks),
		DirectChildRecords: canonicalDirectChildRecordIDs(scope, reuse.scopeIndex, reuse.recordByProjection),
	})
	if err != nil {
		return failed, records.Freshness{}, false, false
	}
	checks = append(checks, records.CheckResult{ID: canonicalControllerAgentCheckID, Version: canonicalControllerAgentCheckVersion, Digest: agentCheckDigest, Outcome: records.CheckPassed})
	sort.Slice(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })
	item := canonicalControllerPreparedVerification{
		scope: scope, record: record, request: request, context: contextModel, target: evidence,
		checks: checks, required: checkIdentities(checks), artifacts: artifacts,
		evidenceDigest: result.EvidenceSnapshotDigest, scopeIndex: reuse.scopeIndex, recordByScope: reuse.recordByScope,
	}
	freshness := records.Freshness{
		Revision: record.Revision, RecordID: record.ID, ModelDigest: record.ModelDigest,
		TargetSnapshotDigest: record.TargetSnapshotDigest, Verifier: reuse.verifier, Checks: append([]records.CheckIdentity(nil), item.required...),
	}
	return item, freshness, currentMatches, true
}

func (reuse *canonicalControllerVerificationReuse) fixedCheckResults(fixed *CanonicalSource, request canonical.ProjectionRequest, scope CanonicalAssuranceScope, evidence *snapshot.Snapshot) []records.CheckResult {
	supplied := fixed.Config.Checks
	if len(scope.Checks) > 0 {
		var err error
		supplied, err = canonicalControllerScopedChecks(fixed, request.Projector, scope.Checks)
		if err != nil {
			return nil
		}
	}
	names, escalations, err := selectedCanonicalChecks(request.Projector, true, supplied)
	if err != nil || len(escalations) != 0 {
		return nil
	}
	checks := selectedAuthoringChecks(names, supplied)
	results := make([]records.CheckResult, 0, len(checks))
	for _, check := range checks {
		results = append(results, records.CheckResult{ID: check.Name, Version: "fixed-command-input/v1", Digest: CanonicalCheckEvidenceDigest(check, evidence), Outcome: records.CheckPassed})
	}
	return results
}

func sameCanonicalProjectionIntent(left, right canonical.ProjectionRequest) bool {
	left.RequestDigest, right.RequestDigest = "", ""
	left.TargetFiles, right.TargetFiles = nil, nil
	left.TargetDigests, right.TargetDigests = nil, nil
	return reflect.DeepEqual(left, right)
}

func canonicalControllerObservedEvidenceMatches(paths []string, evidence, observed *snapshot.Snapshot) bool {
	if evidence == nil || observed == nil {
		return false
	}
	for _, name := range paths {
		left, leftOK := evidence.Files[name]
		right, rightOK := observed.Files[name]
		if !leftOK || !rightOK || !bytes.Equal(left, right) || evidence.Modes[name] != observed.Modes[name] {
			return false
		}
	}
	return true
}
