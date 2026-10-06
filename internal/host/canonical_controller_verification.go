package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/assurance"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	canonicalControllerVerificationAPIVersion = "markitect.canonical/controller-verification/v1alpha1"
	canonicalControllerAgentCheckID           = "independent-verifier-receipt"
	canonicalControllerAgentCheckVersion      = "agent-execution-receipt/v1alpha1"
)

var canonicalControllerVerificationLimits = []string{
	"Repository commands are bounded to selected immutable Git blobs and their captured bytes; command execution does not authenticate the executable environment.",
	"The configured Verifier is a fresh, separately fingerprinted invocation. Its receipt binds invocation inputs and runtime fingerprint, but does not prove provider identity, semantic correctness, or human acceptance.",
	"Private runner logs remain in the explicitly configured external private log directory; this report includes receipt digests and structured observations, not raw logs.",
}

// CanonicalControllerVerification is a source- and evidence-revision-bound
// result. Receipts link each independent verifier invocation to one resulting
// VerificationResult; receipts do not constitute provider authentication.
type CanonicalControllerVerification struct {
	APIVersion            string                           `json:"apiVersion"`
	Digest                string                           `json:"digest"`
	Status                string                           `json:"status"`
	Outcome               string                           `json:"outcome"`
	SourceRevision        string                           `json:"sourceRevision"`
	EvidenceRevision      string                           `json:"evidenceRevision"`
	ConfigDigest          string                           `json:"configDigest"`
	LedgerHead            string                           `json:"ledgerHead"`
	LedgerSelectionDigest string                           `json:"ledgerSelectionDigest"`
	Results               []records.VerificationResult     `json:"results"`
	VerifierRuns          []CanonicalControllerVerifierRun `json:"verifierRuns"`
	Assurance             assurance.RunReport              `json:"assurance"`
	Limits                []string                         `json:"limits"`
}

// CanonicalControllerVerifierRun preserves the receipt and exact cited
// evidence references for one fresh Verifier process.
type CanonicalControllerVerifierRun struct {
	ScopeID                string                  `json:"scopeId"`
	ProjectionID           string                  `json:"projectionId"`
	RecordID               string                  `json:"recordId"`
	ResultID               string                  `json:"resultId"`
	EvidenceSnapshotDigest string                  `json:"evidenceSnapshotDigest"`
	ConfigFingerprint      string                  `json:"configFingerprint"`
	ReceiptDigest          string                  `json:"receiptDigest"`
	InputDigest            string                  `json:"inputDigest"`
	RunID                  string                  `json:"runId"`
	Outcome                string                  `json:"outcome"`
	EvidenceRefs           []string                `json:"evidenceRefs"`
	Observations           []agentexec.Observation `json:"observations"`
	Receipt                agentexec.Receipt       `json:"receipt"`
}

type canonicalControllerPreparedVerification struct {
	scope          CanonicalAssuranceScope
	record         records.ProjectionRecord
	request        canonical.ProjectionRequest
	context        CanonicalAgentContext
	target         *snapshot.Snapshot
	fixed          CanonicalVerification
	checks         []records.CheckResult
	required       []records.CheckIdentity
	artifacts      []agentexec.Artifact
	evidenceDigest string
	scopeIndex     map[string]CanonicalAssuranceScope
	recordByScope  map[string]records.ProjectionRecord
}

type canonicalControllerVerifierContext struct {
	Model                  CanonicalAgentContext    `json:"model"`
	Objective              string                   `json:"objective"`
	EvidenceRevision       string                   `json:"evidenceRevision"`
	EvidenceSnapshotDigest string                   `json:"evidenceSnapshotDigest"`
	RecordID               string                   `json:"recordId"`
	TargetSnapshotDigest   string                   `json:"targetSnapshotDigest"`
	FixedChecks            []records.CheckResult    `json:"fixedChecks"`
	OwnChecks              []records.CheckIdentity  `json:"ownChecks"`
	Children               []canonicalVerifierChild `json:"children"`
	Constraints            []string                 `json:"constraints"`
}

type canonicalVerifierChild struct {
	ScopeID       string                `json:"scopeId"`
	Outcome       string                `json:"outcome"`
	RecordID      string                `json:"recordId"`
	ResultID      string                `json:"resultId"`
	ProjectionID  string                `json:"projectionId"`
	ScopeIDs      []string              `json:"scopeIds"`
	ArtifactFacts []records.Artifact    `json:"artifacts"`
	Checks        []records.CheckResult `json:"checks"`
}

type canonicalVerifierBinding struct {
	Fingerprint        string
	SourceRevision     string
	EvidenceRevision   string
	ModelDigest        string
	RecordID           string
	ProjectionID       string
	ScopeIDs           []string
	PolicyIDs          []string
	EvidenceDigest     string
	RequiredChecks     []records.CheckIdentity
	DirectChildRecords []string
}

// VerifyCanonicalController verifies selected active materializations at
// immutable revisions. With assurance scopes, assurance.Execute invokes one
// fresh verifier per reachable node in postorder. write appends results under
// the controller lease and the record store's expected-head compare-and-swap.
func VerifyCanonicalController(ctx context.Context, root, sourceRevision, evidenceRevision, configPath string, cfg CanonicalControllerConfig, write bool) (CanonicalControllerVerification, error) {
	report := CanonicalControllerVerification{
		APIVersion: canonicalControllerVerificationAPIVersion,
		Status:     records.OutcomeIncomplete, Outcome: records.OutcomeIncomplete,
		SourceRevision: sourceRevision, EvidenceRevision: evidenceRevision,
		Results: []records.VerificationResult{}, VerifierRuns: []CanonicalControllerVerifierRun{},
		Limits: append([]string(nil), canonicalControllerVerificationLimits...),
	}
	if ctx == nil {
		return report, errors.New("canonical controller verification requires a context")
	}
	if err := validateControllerConfig(cfg); err != nil {
		return report, err
	}
	if !canonicalRevisionPattern.MatchString(sourceRevision) || !canonicalRevisionPattern.MatchString(evidenceRevision) {
		return report, errors.New("verification requires full immutable source and evidence commit IDs")
	}
	var err error
	report.ConfigDigest, err = digestCanonicalValue(cfg)
	if err != nil {
		return report, err
	}
	fixed, err := LoadSelectedCanonicalSource(root, sourceRevision, configPath, true)
	if err != nil {
		return report, err
	}
	if fixed.Snapshot == nil || fixed.Snapshot.ID != sourceRevision || fixed.Snapshot.Provisional {
		return report, errors.New("selected canonical source does not bind the exact immutable source revision")
	}
	if _, err := SelectedInputSnapshotDigest(fixed); err != nil {
		return report, err
	}
	if err := validateCanonicalControllerAssurance(cfg, fixed); err != nil {
		return report, err
	}
	store, state, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	report.LedgerHead, report.LedgerSelectionDigest = state.Head, state.ActiveSelection.Digest
	if len(active) == 0 {
		return report, errors.New("canonical verification requires at least one active materialization record")
	}
	if write {
		if store == nil {
			return report, errors.New("controller ledger is not initialized")
		}
		unlock, err := acquireCanonicalControllerLease(cfg)
		if err != nil {
			return report, err
		}
		defer unlock()
		store, state, active, err = readCanonicalControllerLedger(root, cfg)
		if err != nil {
			return report, err
		}
		if state.Head != report.LedgerHead || state.ActiveSelection.Digest != report.LedgerSelectionDigest {
			return report, errors.New("controller ledger changed before verification acquired its lease")
		}
	}
	if store == nil {
		return report, errors.New("controller ledger is not initialized")
	}

	verifierFingerprint, err := agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if err != nil {
		return report, fmt.Errorf("fingerprint configured Verifier: %w", err)
	}
	verifier := records.VerifierIdentity{ID: "markitect-agentexec/verifier", Version: cfg.Verifier.ProviderVersion, Digest: verifierFingerprint}
	scopes, roots, recordByProjection, err := canonicalControllerVerificationNodes(cfg, active)
	if err != nil {
		return report, err
	}
	checkInputs := canonicalControllerCheckInputs(cfg)
	if err := canonicalControllerFixedCheckInputs(root, sourceRevision, evidenceRevision, checkInputs); err != nil {
		return report, err
	}
	prepared, graph, current, err := prepareCanonicalControllerVerifications(
		root, sourceRevision, evidenceRevision, cfg, fixed, scopes, roots, recordByProjection, verifier, verifierFingerprint,
	)
	if err != nil {
		return report, err
	}
	// Validate graph structure and reachability before the first configured
	// process can run. Host also emits the missing canonical IDs for containment.
	structural, err := assurance.Evaluate(graph)
	if err != nil {
		return report, err
	}
	if len(structural.Nodes) != len(graph.Nodes) {
		return report, errors.New("every configured assurance scope must be reachable from a declared assurance root")
	}

	resultByScope := make(map[string]records.VerificationResult, len(graph.Nodes))
	runByScope := make(map[string]CanonicalControllerVerifierRun, len(graph.Nodes))
	runReport, runErr := assurance.Execute(ctx, assurance.RunInput{Graph: graph, Current: current, Retries: 0}, func(runCtx context.Context, node assurance.NodeRunInput) (assurance.NodeRunOutput, error) {
		item, ok := prepared[node.Node.ID]
		if !ok {
			return assurance.NodeRunOutput{}, fmt.Errorf("verification preparation missing for assurance scope %q", node.Node.ID)
		}
		result, verifierRun, err := invokeCanonicalControllerVerifier(runCtx, cfg, item, node, verifier, verifierFingerprint, evidenceRevision)
		if err != nil {
			return assurance.NodeRunOutput{}, err
		}
		resultByScope[node.Node.ID], runByScope[node.Node.ID] = result, verifierRun
		return assurance.NodeRunOutput{NodeID: node.Node.ID, Disposition: assurance.RunCompleted, Record: item.record, Result: result}, nil
	})
	report.Assurance = runReport
	for _, result := range resultByScope {
		report.Results = append(report.Results, result)
	}
	sort.Slice(report.Results, func(i, j int) bool { return report.Results[i].ID < report.Results[j].ID })
	for _, run := range runByScope {
		report.VerifierRuns = append(report.VerifierRuns, run)
	}
	sort.Slice(report.VerifierRuns, func(i, j int) bool { return report.VerifierRuns[i].ScopeID < report.VerifierRuns[j].ScopeID })
	report.Outcome = canonicalControllerComposedOutcome(runReport.Evaluation.Roots)
	if runErr != nil {
		report.Outcome = composeVerificationOutcomes(report.Outcome, records.OutcomeIncomplete)
	}
	report.Status = report.Outcome

	fingerprintAfter, fingerprintErr := agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if fingerprintErr != nil || fingerprintAfter != verifierFingerprint {
		if fingerprintErr != nil {
			return report, fmt.Errorf("Verifier runtime changed during scoped verification: %w", fingerprintErr)
		}
		return report, errors.New("Verifier runtime fingerprint changed during scoped verification")
	}
	_, freshState, freshActive, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, fmt.Errorf("recheck ledger freshness after verification: %w", err)
	}
	if freshState.Head != report.LedgerHead || freshState.ActiveSelection.Digest != report.LedgerSelectionDigest ||
		!sameCanonicalControllerActiveRecords(active, freshActive) {
		return report, errors.New("controller ledger head or active selection changed during scoped verification")
	}
	if write {
		for _, result := range report.Results {
			freshState, err = store.AppendVerification(freshState.Head, result)
			if err != nil {
				report.LedgerHead = freshState.Head
				return report, fmt.Errorf("append scoped verification result %s: %w", result.ID, err)
			}
		}
		report.LedgerHead = freshState.Head
	}
	report.Digest, err = digestCanonicalValue(report)
	if err != nil {
		return report, err
	}
	if runErr != nil {
		return report, runErr
	}
	return report, nil
}

func canonicalControllerVerificationNodes(cfg CanonicalControllerConfig, active []records.ProjectionRecord) ([]CanonicalAssuranceScope, []string, map[string]records.ProjectionRecord, error) {
	byProjection := make(map[string]records.ProjectionRecord, len(active))
	for _, record := range active {
		if err := records.ValidateProjectionRecord(record); err != nil {
			return nil, nil, nil, fmt.Errorf("active ProjectionRecord %s: %w", record.ID, err)
		}
		if record.State != records.StateMaterializedUnverified {
			return nil, nil, nil, fmt.Errorf("active ProjectionRecord %s is not a complete materialization", record.ID)
		}
		if _, duplicate := byProjection[record.ProjectionID]; duplicate {
			return nil, nil, nil, fmt.Errorf("active selection contains duplicate Projection %s", record.ProjectionID)
		}
		byProjection[record.ProjectionID] = record
	}
	if len(cfg.AssuranceScopes) == 0 {
		scopes := make([]CanonicalAssuranceScope, 0, len(active))
		for _, record := range active {
			scopes = append(scopes, CanonicalAssuranceScope{ID: record.ProjectionID, ProjectionID: record.ProjectionID})
		}
		sort.Slice(scopes, func(i, j int) bool { return scopes[i].ID < scopes[j].ID })
		roots := make([]string, len(scopes))
		for i := range scopes {
			roots[i] = scopes[i].ID
		}
		return scopes, roots, byProjection, nil
	}
	scopes := append([]CanonicalAssuranceScope(nil), cfg.AssuranceScopes...)
	sort.Slice(scopes, func(i, j int) bool { return scopes[i].ID < scopes[j].ID })
	roots := append([]string(nil), cfg.AssuranceRoots...)
	sort.Strings(roots)
	usedProjection := map[string]bool{}
	byScope := make(map[string]CanonicalAssuranceScope, len(scopes))
	for _, scope := range scopes {
		record, ok := byProjection[scope.ProjectionID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("assurance scope %q ProjectionID %q does not name an active record", scope.ID, scope.ProjectionID)
		}
		if usedProjection[scope.ProjectionID] {
			return nil, nil, nil, fmt.Errorf("assurance scopes do not map one-to-one to active Projection %q", scope.ProjectionID)
		}
		usedProjection[scope.ProjectionID] = true
		if len(scope.Checks) == 0 {
			return nil, nil, nil, fmt.Errorf("assurance scope %q must declare its own fixed checks", scope.ID)
		}
		byScope[scope.ID] = scope
		_ = record
	}
	if len(usedProjection) != len(byProjection) {
		uncovered := make([]string, 0, len(byProjection)-len(usedProjection))
		for projectionID := range byProjection {
			if !usedProjection[projectionID] {
				uncovered = append(uncovered, projectionID)
			}
		}
		sort.Strings(uncovered)
		return nil, nil, nil, fmt.Errorf("assurance scopes must cover every active ProjectionRecord; uncovered Projections: %s", strings.Join(uncovered, ", "))
	}
	for _, scope := range scopes {
		parent := byProjection[scope.ProjectionID]
		parentIDs := map[string]bool{}
		for _, id := range parent.ScopeIDs {
			parentIDs[id] = true
		}
		for _, childID := range scope.Children {
			childScope, ok := byScope[childID]
			if !ok {
				return nil, nil, nil, fmt.Errorf("assurance scope %q references undeclared child %q", scope.ID, childID)
			}
			child := byProjection[childScope.ProjectionID]
			missing := []string{}
			for _, sourceID := range child.ScopeIDs {
				if !parentIDs[sourceID] {
					missing = append(missing, sourceID)
				}
			}
			if len(missing) != 0 {
				sort.Strings(missing)
				return nil, nil, nil, fmt.Errorf("assurance parent %q Projection %q canonical source scope omits child %q identities: %s", scope.ID, parent.ProjectionID, childID, strings.Join(missing, ", "))
			}
		}
	}
	return scopes, roots, byProjection, nil
}

func prepareCanonicalControllerVerifications(root, sourceRevision, evidenceRevision string, cfg CanonicalControllerConfig, fixed *CanonicalSource, scopes []CanonicalAssuranceScope, roots []string, recordsByProjection map[string]records.ProjectionRecord, verifier records.VerifierIdentity, fingerprint string) (map[string]canonicalControllerPreparedVerification, assurance.Input, map[string]records.Freshness, error) {
	graph := assurance.Input{RootIDs: append([]string(nil), roots...), Nodes: make([]assurance.Node, 0, len(scopes))}
	scopeIndex := make(map[string]CanonicalAssuranceScope, len(scopes))
	recordByScope := make(map[string]records.ProjectionRecord, len(scopes))
	for _, scope := range scopes {
		scopeIndex[scope.ID] = scope
		recordByScope[scope.ID] = recordsByProjection[scope.ProjectionID]
	}
	prepared := make(map[string]canonicalControllerPreparedVerification, len(scopes))
	for _, scope := range scopes {
		record := recordsByProjection[scope.ProjectionID]
		request, target, err := loadCanonicalControllerScopeRequest(root, sourceRevision, evidenceRevision, cfg, fixed, record, scope, scopeIndex, recordsByProjection)
		if err != nil {
			return nil, assurance.Input{}, nil, fmt.Errorf("prepare assurance scope %q: %w", scope.ID, err)
		}
		contextModel, err := BuildCanonicalAgentContext(fixed.Model, request, cfg.ReferenceDepth)
		if err != nil {
			return nil, assurance.Input{}, nil, fmt.Errorf("build canonical context for scope %q: %w", scope.ID, err)
		}
		scopedSource := fixed
		if len(scope.Checks) != 0 {
			checks, err := canonicalControllerScopedChecks(fixed, request.Projector, scope.Checks)
			if err != nil {
				return nil, assurance.Input{}, nil, fmt.Errorf("fixed checks for assurance scope %q: %w", scope.ID, err)
			}
			copySource := *fixed
			copySource.Config.Checks = checks
			scopedSource = &copySource
		}
		fixedVerification, verifyErr := VerifyCanonicalProjection(scopedSource, target, record, verifier)
		if fixedVerification.Result.ID == "" {
			if verifyErr == nil {
				verifyErr = errors.New("fixed canonical projection verification returned no result")
			}
			return nil, assurance.Input{}, nil, fmt.Errorf("fixed verification for scope %q: %w", scope.ID, verifyErr)
		}
		checks := append([]records.CheckResult(nil), fixedVerification.Result.Checks...)
		if duplicate := duplicateCheckID(checks); duplicate != "" {
			return nil, assurance.Input{}, nil, fmt.Errorf("assurance scope %q reuses check identity %q", scope.ID, duplicate)
		}
		artifacts, err := canonicalControllerVerificationArtifacts(record, scope, scopeIndex, recordsByProjection, target, cfg.CheckInputs)
		if err != nil {
			return nil, assurance.Input{}, nil, err
		}
		evidenceDigest := sha256Prefix(target.Digest())
		agentCheckDigest, err := digestCanonicalValue(canonicalVerifierBinding{
			Fingerprint: fingerprint, SourceRevision: sourceRevision, EvidenceRevision: evidenceRevision,
			ModelDigest: record.ModelDigest, RecordID: record.ID, ProjectionID: record.ProjectionID,
			ScopeIDs: append([]string(nil), record.ScopeIDs...), PolicyIDs: append([]string(nil), record.PolicyIDs...),
			EvidenceDigest: evidenceDigest, RequiredChecks: checkIdentities(checks),
			DirectChildRecords: canonicalDirectChildRecordIDs(scope, scopeIndex, recordsByProjection),
		})
		if err != nil {
			return nil, assurance.Input{}, nil, err
		}
		checks = append(checks, records.CheckResult{ID: canonicalControllerAgentCheckID, Version: canonicalControllerAgentCheckVersion, Digest: agentCheckDigest, Outcome: records.CheckIncomplete})
		sort.Slice(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })
		required := checkIdentities(checks)
		graph.Nodes = append(graph.Nodes, assurance.Node{ID: scope.ID, ScopeIDs: append([]string(nil), record.ScopeIDs...), Children: append([]string(nil), scope.Children...), RequiredChecks: required})
		prepared[scope.ID] = canonicalControllerPreparedVerification{
			scope: scope, record: record, request: request, context: contextModel, target: target,
			fixed: fixedVerification, checks: checks, required: required, artifacts: artifacts, evidenceDigest: evidenceDigest,
			scopeIndex: scopeIndex, recordByScope: recordByScope,
		}
	}
	current := make(map[string]records.Freshness, len(prepared))
	for id, item := range prepared {
		current[id] = records.Freshness{
			Revision: item.record.Revision, RecordID: item.record.ID, ModelDigest: item.record.ModelDigest,
			TargetSnapshotDigest: item.record.TargetSnapshotDigest, Verifier: verifier, Checks: append([]records.CheckIdentity(nil), item.required...),
		}
	}
	return prepared, graph, current, nil
}

func canonicalControllerScopedChecks(fixed *CanonicalSource, projector core.Definition, requested []authoring.Check) ([]authoring.Check, error) {
	if fixed == nil {
		return nil, errors.New("fixed canonical source is required")
	}
	configured := make(map[string]authoring.Check, len(fixed.Config.Checks))
	for _, check := range fixed.Config.Checks {
		if _, duplicate := configured[check.Name]; duplicate {
			return nil, fmt.Errorf("source config repeats fixed check %q", check.Name)
		}
		configured[check.Name] = check
	}
	selected := make([]authoring.Check, 0, len(requested))
	seen := map[string]bool{}
	for _, check := range requested {
		if check.Name == "" || seen[check.Name] {
			return nil, fmt.Errorf("scope checks contain an empty or repeated check name %q", check.Name)
		}
		seen[check.Name] = true
		declared, ok := configured[check.Name]
		if !ok {
			return nil, fmt.Errorf("scope check %q is not declared by the selected source config", check.Name)
		}
		declaredBytes, declaredErr := json.Marshal(declared)
		requestedBytes, requestedErr := json.Marshal(check)
		if declaredErr != nil || requestedErr != nil || !bytes.Equal(declaredBytes, requestedBytes) {
			return nil, fmt.Errorf("scope check %q differs from its exact selected source-config definition", check.Name)
		}
		selected = append(selected, declared)
	}
	if len(selected) == 0 {
		return nil, errors.New("explicit assurance scope must select fixed checks")
	}
	names, escalations, err := selectedCanonicalChecks(projector, true, selected)
	if err != nil {
		return nil, err
	}
	if len(escalations) != 0 {
		return nil, fmt.Errorf("scope omits or cannot resolve required fixed check: %s", escalations[0].Message)
	}
	selectedNames := make([]string, len(selected))
	for i, check := range selected {
		selectedNames[i] = check.Name
	}
	sort.Strings(selectedNames)
	sort.Strings(names)
	if !equalStringSets(selectedNames, names) {
		return nil, fmt.Errorf("scope fixed checks %v do not exactly match Projection-required checks %v", selectedNames, names)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Name < selected[j].Name })
	return selected, nil
}

func loadCanonicalControllerScopeRequest(root, sourceRevision, evidenceRevision string, cfg CanonicalControllerConfig, fixed *CanonicalSource, record records.ProjectionRecord, scope CanonicalAssuranceScope, scopeIndex map[string]CanonicalAssuranceScope, recordsByProjection map[string]records.ProjectionRecord) (canonical.ProjectionRequest, *snapshot.Snapshot, error) {
	identity, err := projectionIdentityFromKey(record.ProjectionID)
	if err != nil {
		return canonical.ProjectionRequest{}, nil, err
	}
	projection, exists := fixed.Model.Definition(identity)
	if !exists {
		return canonical.ProjectionRequest{}, nil, fmt.Errorf("Projection %s is absent from the selected canonical Model", record.ProjectionID)
	}
	childRecords, err := canonicalControllerDirectChildRecords(scope, scopeIndex, recordsByProjection)
	if err != nil {
		return canonical.ProjectionRequest{}, nil, err
	}
	paths := canonicalSourcePaths(fixed)
	for _, artifact := range record.Artifacts {
		paths = append(paths, artifact.Path)
	}
	for _, child := range childRecords {
		for _, artifact := range child.Artifacts {
			paths = append(paths, artifact.Path)
		}
	}
	paths = append(paths, cfg.CheckInputs...)
	paths = append(paths, scope.CheckInputs...)
	paths = sortedUniquePaths(paths)
	selected, err := source.LoadSelected(root, evidenceRevision, paths)
	if err != nil {
		return canonical.ProjectionRequest{}, nil, fmt.Errorf("load selected evidence commit: %w", err)
	}
	if selected.Snapshot == nil || selected.Snapshot.ID != evidenceRevision || selected.Snapshot.Provisional {
		return canonical.ProjectionRequest{}, nil, errors.New("selected evidence does not bind the exact immutable evidence revision")
	}
	if err := validateCanonicalSourceUnchanged(fixed, selected.Snapshot); err != nil {
		return canonical.ProjectionRequest{}, nil, fmt.Errorf("canonical source changed between fixed and evidence revisions: %w", err)
	}
	targetFiles, err := canonicalProjectionTargetFiles(projection, selected.Snapshot.Files)
	if err != nil {
		return canonical.ProjectionRequest{}, nil, err
	}
	request, err := canonical.BindProjection(fixed.Model, fixed.Activation, fixed.Config.ProjectionBindings, identity, targetFiles)
	if err != nil {
		return canonical.ProjectionRequest{}, nil, err
	}
	if record.Revision != sourceRevision || record.ModelDigest != request.ModelDigest ||
		record.Module.Name != request.ModulePin.Name || record.Module.Version != request.ModulePin.Version ||
		record.Module.Digest != request.ModulePin.Digest || record.Projector.ID != request.Projector.ID || record.Projector.Version != request.Projector.Version {
		return canonical.ProjectionRequest{}, nil, errors.New("active record is stale for the selected canonical source or Module/Projector binding")
	}
	if !equalStringSets(requestScopeIDs(request.Definitions), record.ScopeIDs) || !equalStringSets(canonicalRequestPolicyIDs(request.Policies), record.PolicyIDs) {
		return canonical.ProjectionRequest{}, nil, errors.New("active record ScopeIDs or PolicyIDs differ from the selected canonical Projection")
	}
	return request, selected.Snapshot, nil
}

func canonicalControllerVerificationArtifacts(record records.ProjectionRecord, scope CanonicalAssuranceScope, scopeIndex map[string]CanonicalAssuranceScope, recordsByProjection map[string]records.ProjectionRecord, target *snapshot.Snapshot, fixedCheckInputs []string) ([]agentexec.Artifact, error) {
	selected := map[string]bool{}
	for _, artifact := range record.Artifacts {
		selected[artifact.Path] = true
	}
	children, err := canonicalControllerDirectChildRecords(scope, scopeIndex, recordsByProjection)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		for _, artifact := range child.Artifacts {
			selected[artifact.Path] = true
		}
	}
	for _, name := range fixedCheckInputs {
		selected[name] = true
	}
	for _, name := range scope.CheckInputs {
		selected[name] = true
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	artifacts := make([]agentexec.Artifact, 0, len(names))
	for _, name := range names {
		data, exists := target.Files[name]
		if !exists {
			return nil, fmt.Errorf("selected target artifact %q is absent at the evidence revision", name)
		}
		mode := target.Modes[name]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		agentMode := "0644"
		switch mode {
		case snapshot.RegularMode:
		case snapshot.ExecutableMode:
			agentMode = "0755"
		default:
			return nil, fmt.Errorf("selected target artifact %q has unsupported file mode %q", name, mode)
		}
		artifacts = append(artifacts, agentexec.Artifact{Path: name, Mode: agentMode, Digest: sha256Prefix(sha256Hex(data)), Content: append([]byte(nil), data...)})
	}
	return artifacts, nil
}

func canonicalControllerDirectChildRecords(scope CanonicalAssuranceScope, scopeIndex map[string]CanonicalAssuranceScope, recordsByProjection map[string]records.ProjectionRecord) ([]records.ProjectionRecord, error) {
	children := make([]records.ProjectionRecord, 0, len(scope.Children))
	for _, childID := range scope.Children {
		childScope, ok := scopeIndex[childID]
		if !ok {
			return nil, fmt.Errorf("assurance scope %q has undeclared child %q", scope.ID, childID)
		}
		record, ok := recordsByProjection[childScope.ProjectionID]
		if !ok {
			return nil, fmt.Errorf("assurance child %q has no active ProjectionRecord", childID)
		}
		children = append(children, record)
	}
	return children, nil
}

func canonicalDirectChildRecordIDs(scope CanonicalAssuranceScope, scopeIndex map[string]CanonicalAssuranceScope, recordsByProjection map[string]records.ProjectionRecord) []string {
	children, _ := canonicalControllerDirectChildRecords(scope, scopeIndex, recordsByProjection)
	values := make([]string, 0, len(children))
	for _, child := range children {
		values = append(values, child.ID)
	}
	sort.Strings(values)
	return values
}

func invokeCanonicalControllerVerifier(ctx context.Context, cfg CanonicalControllerConfig, item canonicalControllerPreparedVerification, node assurance.NodeRunInput, verifier records.VerifierIdentity, fingerprint, evidenceRevision string) (records.VerificationResult, CanonicalControllerVerifierRun, error) {
	run := CanonicalControllerVerifierRun{
		ScopeID: item.scope.ID, ProjectionID: item.record.ProjectionID, RecordID: item.record.ID,
		EvidenceSnapshotDigest: item.evidenceDigest, ConfigFingerprint: fingerprint,
		EvidenceRefs: []string{}, Observations: []agentexec.Observation{},
	}
	children := make([]canonicalVerifierChild, 0, len(node.Children))
	for _, child := range node.Children {
		childContext := canonicalVerifierChild{
			ScopeID: child.Result.NodeID, Outcome: child.Result.Outcome,
			RecordID: child.Result.RecordID, ResultID: child.Result.ResultID,
		}
		childScope, ok := item.scopeIndex[child.Result.NodeID]
		if !ok {
			return records.VerificationResult{}, run, errors.New("assurance scheduler supplied an undeclared direct child")
		}
		childRecord, ok := item.recordByScope[child.Result.NodeID]
		if !ok || childRecord.ID != child.Result.RecordID {
			return records.VerificationResult{}, run, errors.New("assurance scheduler child differs from the selected active record")
		}
		childContext.ProjectionID = childScope.ProjectionID
		childContext.ScopeIDs = append([]string(nil), childRecord.ScopeIDs...)
		childContext.ArtifactFacts = append([]records.Artifact(nil), childRecord.Artifacts...)
		if child.Evidence != nil {
			if child.Evidence.Record.ID != childRecord.ID || child.Evidence.Result.ID != child.Result.ResultID {
				return records.VerificationResult{}, run, errors.New("assurance scheduler supplied mismatched direct-child evidence")
			}
			childContext.Checks = append([]records.CheckResult(nil), child.Evidence.Result.Checks...)
		}
		children = append(children, childContext)
	}
	payload := canonicalControllerVerifierContext{
		Model:            item.context,
		Objective:        "Independently verify the selected canonical Projection and its exact evidence bytes. Assess only the declared scope, policies, fixed command results, direct child outcomes, and supplied artifact bytes.",
		EvidenceRevision: evidenceRevision, EvidenceSnapshotDigest: item.evidenceDigest,
		RecordID: item.record.ID, TargetSnapshotDigest: item.record.TargetSnapshotDigest,
		FixedChecks: item.checksWithoutAgent(), OwnChecks: item.scopeCheckIdentities(), Children: children,
		Constraints: []string{
			"Do not use Executor transcripts, hidden reasoning, workspace search, or unselected files.",
			"Return evidence references only for supplied canonical scopes, policies, and target artifact paths.",
			"Passing requires every fixed check and every required canonical obligation to pass.",
			"Escalate when evidence is missing, ambiguous, or outside the selected scope.",
		},
	}
	contextBytes, err := json.Marshal(payload)
	if err != nil {
		return records.VerificationResult{}, run, err
	}
	request := agentexec.Request{
		Role: agentexec.RoleVerifier, SourceRevision: evidenceRevision, ModelDigest: item.record.ModelDigest,
		ModulePin: item.record.Module.Digest, ProjectionID: item.record.ProjectionID,
		ScopeIDs:  append([]string(nil), item.context.ScopeIDs...),
		PolicyIDs: append([]string(nil), canonicalRequestPolicyIDs(item.request.Policies)...),
		Context:   contextBytes, Artifacts: cloneAgentArtifacts(item.artifacts),
	}
	execution, invokeErr := agentexec.Run(ctx, cfg.Verifier.agentConfig(), request, agentexec.RunOptions{
		TempParent: filepath.Dir(cfg.PrivateLogs), PrivateLogDirectory: cfg.PrivateLogs,
	})
	run.Receipt = execution.Receipt
	run.InputDigest, run.RunID = execution.Receipt.InputDigest, execution.Receipt.RunID
	run.ReceiptDigest, err = digestCanonicalValue(execution.Receipt)
	if err != nil {
		return records.VerificationResult{}, run, err
	}
	if invokeErr != nil {
		run.Outcome = agentexec.OutcomeIncomplete
		switch execution.Receipt.Outcome {
		case agentexec.OutcomeFailed, agentexec.OutcomeIncomplete, agentexec.OutcomeEscalated:
			run.Outcome = execution.Receipt.Outcome
		}
	} else {
		run.Outcome = execution.Response.Outcome
		run.EvidenceRefs = append([]string(nil), execution.Response.EvidenceRefs...)
		run.Observations = append([]agentexec.Observation(nil), execution.Response.VerifierObservations...)
		if !canonicalControllerExactEvidenceRefs(execution.Response.EvidenceRefs, execution.Response.VerifierObservations, request) {
			invokeErr = errors.New("Verifier did not cite the exact supplied scope, policy and artifact references")
			run.Outcome = agentexec.OutcomeEscalated
		}
	}
	if execution.Receipt.ConfigDigest != fingerprint {
		invokeErr = errors.New("Verifier receipt configuration fingerprint differs from the prepared identity")
		run.Outcome = agentexec.OutcomeIncomplete
	}
	checks := append([]records.CheckResult(nil), item.checks...)
	for i := range checks {
		if checks[i].ID == canonicalControllerAgentCheckID {
			switch {
			case run.Outcome == agentexec.OutcomeEscalated:
				checks[i].Outcome = records.CheckIncomplete
			case invokeErr != nil && run.Outcome == agentexec.OutcomeFailed:
				checks[i].Outcome = records.CheckFailed
			case invokeErr != nil:
				checks[i].Outcome = records.CheckIncomplete
			default:
				checks[i].Outcome = canonicalAgentCheckOutcome(execution.Response)
			}
		}
	}
	outcome := records.OutcomePassed
	for _, check := range checks {
		switch check.Outcome {
		case records.CheckFailed:
			outcome = composeVerificationOutcomes(outcome, records.OutcomeFailed)
		case records.CheckIncomplete:
			outcome = composeVerificationOutcomes(outcome, records.OutcomeIncomplete)
		}
	}
	switch run.Outcome {
	case agentexec.OutcomeFailed:
		outcome = composeVerificationOutcomes(outcome, records.OutcomeFailed)
	case agentexec.OutcomeEscalated:
		outcome = composeVerificationOutcomes(outcome, records.OutcomeEscalated)
	case agentexec.OutcomeIncomplete:
		outcome = composeVerificationOutcomes(outcome, records.OutcomeIncomplete)
	}
	reason := fmt.Sprintf("Verifier receipt %s; run %s; input %s. The separate invocation is bounded machine evidence; provider identity, semantic sufficiency, and human acceptance remain unproven.", run.ReceiptDigest, run.RunID, run.InputDigest)
	if invokeErr != nil {
		reason += " Invocation detail: " + boundedCanonicalVerificationReason(invokeErr.Error(), 1024)
	}
	if run.Outcome == agentexec.OutcomeEscalated && invokeErr == nil {
		reason += " Verifier explicitly requested escalation."
	}
	verification, err := records.NewVerificationResult(records.VerificationResult{
		RecordID: item.record.ID, Revision: item.record.Revision, ModelDigest: item.record.ModelDigest,
		TargetSnapshotDigest: item.record.TargetSnapshotDigest, Verifier: verifier, Checks: checks,
		Outcome: outcome, Reason: reason,
	})
	if err != nil {
		return records.VerificationResult{}, run, err
	}
	run.ResultID = verification.ID
	return verification, run, nil
}

func (item canonicalControllerPreparedVerification) checksWithoutAgent() []records.CheckResult {
	out := make([]records.CheckResult, 0, len(item.checks))
	for _, check := range item.checks {
		if check.ID != canonicalControllerAgentCheckID {
			out = append(out, check)
		}
	}
	return out
}

func (item canonicalControllerPreparedVerification) scopeCheckIdentities() []records.CheckIdentity {
	return checkIdentities(item.checksWithoutAgent())
}

func checkIdentities(checks []records.CheckResult) []records.CheckIdentity {
	out := make([]records.CheckIdentity, len(checks))
	for i, check := range checks {
		out[i] = records.CheckIdentity{ID: check.ID, Version: check.Version, Digest: check.Digest}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func requestScopeIDs(definitions []core.Definition) []string {
	values := make([]string, len(definitions))
	for i, definition := range definitions {
		values[i] = definition.Identity().Key()
	}
	sort.Strings(values)
	return values
}

func canonicalControllerFixedScopeChecks(checks []authoring.Check, target *snapshot.Snapshot) ([]records.CheckResult, []GateResult, error) {
	gates, err := VerifySnapshotChecks(target, checks)
	byName := map[string]GateResult{}
	for _, gate := range gates {
		byName[gate.Name] = gate
	}
	results := make([]records.CheckResult, 0, len(checks))
	for _, check := range checks {
		outcome := records.CheckIncomplete
		if gate, ok := byName[check.Name]; ok {
			outcome = records.CheckPassed
			if gate.ExitCode != 0 {
				outcome = records.CheckFailed
			}
		}
		results = append(results, records.CheckResult{
			ID: check.Name, Version: "fixed-command-input/v1", Digest: CanonicalCheckEvidenceDigest(check, target), Outcome: outcome,
		})
	}
	return results, gates, err
}

func duplicateCheckID(checks []records.CheckResult) string {
	seen := map[string]bool{}
	for _, check := range checks {
		if seen[check.ID] {
			return check.ID
		}
		seen[check.ID] = true
	}
	return ""
}

func canonicalControllerExactEvidenceRefs(refs []string, observations []agentexec.Observation, request agentexec.Request) bool {
	expected := map[string]bool{}
	for _, id := range request.ScopeIDs {
		expected[id] = true
	}
	for _, id := range request.PolicyIDs {
		expected[id] = true
	}
	for _, artifact := range request.Artifacts {
		expected[artifact.Path] = true
	}
	if len(refs) != len(expected) {
		return false
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if !expected[ref] || seen[ref] {
			return false
		}
		seen[ref] = true
	}
	for _, observation := range observations {
		if !expected[observation.Subject] {
			return false
		}
	}
	return true
}

func canonicalAgentCheckOutcome(response agentexec.Response) string {
	outcome := response.Outcome
	for _, observation := range response.VerifierObservations {
		if observation.Outcome == agentexec.OutcomeFailed {
			outcome = agentexec.OutcomeFailed
			continue
		}
		if observation.Outcome == agentexec.OutcomeEscalated && outcome != agentexec.OutcomeFailed {
			outcome = agentexec.OutcomeEscalated
			continue
		}
		if observation.Outcome == agentexec.OutcomeIncomplete && outcome == agentexec.OutcomePassed {
			outcome = agentexec.OutcomeIncomplete
		}
	}
	switch outcome {
	case agentexec.OutcomePassed:
		return records.CheckPassed
	case agentexec.OutcomeFailed:
		return records.CheckFailed
	default:
		return records.CheckIncomplete
	}
}

func composeVerificationOutcomes(left, right string) string {
	if left == records.OutcomeFailed || right == records.OutcomeFailed {
		return records.OutcomeFailed
	}
	if left == records.OutcomeEscalated || right == records.OutcomeEscalated {
		return records.OutcomeEscalated
	}
	if left == records.OutcomeIncomplete || right == records.OutcomeIncomplete {
		return records.OutcomeIncomplete
	}
	return records.OutcomePassed
}

func canonicalControllerComposedOutcome(roots []assurance.NodeResult) string {
	if len(roots) == 0 {
		return records.OutcomeIncomplete
	}
	outcome := records.OutcomePassed
	for _, root := range roots {
		outcome = composeVerificationOutcomes(outcome, root.Outcome)
	}
	return outcome
}

func canonicalControllerFixedCheckInputs(root, sourceRevision, evidenceRevision string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	fixed, err := source.LoadSelected(root, sourceRevision, paths)
	if err != nil {
		return fmt.Errorf("load fixed check inputs at source revision: %w", err)
	}
	evidence, err := source.LoadSelected(root, evidenceRevision, paths)
	if err != nil {
		return fmt.Errorf("load fixed check inputs at evidence revision: %w", err)
	}
	if fixed.Snapshot.ID != sourceRevision || evidence.Snapshot.ID != evidenceRevision {
		return errors.New("check input acquisition did not bind the requested full revisions")
	}
	for _, name := range paths {
		if !bytes.Equal(fixed.Snapshot.Files[name], evidence.Snapshot.Files[name]) || fixed.Snapshot.Modes[name] != evidence.Snapshot.Modes[name] {
			return fmt.Errorf("fixed verification input changed between source and evidence revisions: %s", name)
		}
	}
	return nil
}

func sameCanonicalControllerActiveRecords(left, right []records.ProjectionRecord) bool {
	if len(left) != len(right) {
		return false
	}
	byID := make(map[string]records.ProjectionRecord, len(left))
	for _, record := range left {
		byID[record.ID] = record
	}
	for _, record := range right {
		prior, ok := byID[record.ID]
		if !ok {
			return false
		}
		priorJSON, priorErr := json.Marshal(prior)
		recordJSON, recordErr := json.Marshal(record)
		if priorErr != nil || recordErr != nil || !bytes.Equal(priorJSON, recordJSON) {
			return false
		}
	}
	return true
}

func boundedCanonicalVerificationReason(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max] + "…"
	}
	return value
}
