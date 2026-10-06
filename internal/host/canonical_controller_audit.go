package host

import (
	"errors"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const canonicalControllerAuditAPIVersion = "markitect.canonical/controller-audit/v1alpha1"

const canonicalControllerAuditCoverage = "declared canonical Projections and configured target roots; bounded current evidence"

// CanonicalControllerAudit is a read-only decision over the selected canonical
// source, declared target roots, active records, and their latest bound
// verification evidence. It does not claim whole-repository correctness.
type CanonicalControllerAudit struct {
	APIVersion            string                               `json:"apiVersion"`
	Digest                string                               `json:"digest"`
	Status                string                               `json:"status"`
	Coverage              string                               `json:"coverage"`
	BaseRevision          string                               `json:"baseRevision"`
	SourceRevision        string                               `json:"sourceRevision"`
	ConfigDigest          string                               `json:"configDigest"`
	LedgerHead            string                               `json:"ledgerHead"`
	LedgerSelectionDigest string                               `json:"ledgerSelectionDigest"`
	ProposalDigest        string                               `json:"proposalDigest"`
	PlanStatus            string                               `json:"planStatus"`
	Projections           []CanonicalControllerAuditProjection `json:"projections"`
	Exclusions            []CanonicalExcludedArtifact          `json:"exclusions"`
	Findings              []CanonicalControllerAuditFinding    `json:"findings"`
	NextSteps             []string                             `json:"nextSteps"`
}

type CanonicalControllerAuditProjection struct {
	ProjectionID               string   `json:"projectionId"`
	ScopeIDs                   []string `json:"scopeIds"`
	PolicyIDs                  []string `json:"policyIds"`
	AssuranceScopeIDs          []string `json:"assuranceScopeIds"`
	ActiveRecordIDs            []string `json:"activeRecordIds"`
	FreshVerificationResultIDs []string `json:"freshVerificationResultIds"`
	ArtifactPaths              []string `json:"artifactPaths"`
	CheckIDs                   []string `json:"checkIds"`
	Status                     string   `json:"status"`
	Outcome                    string   `json:"outcome"`
}

type CanonicalControllerAuditFinding struct {
	Code         string `json:"code"`
	ProjectionID string `json:"projectionId,omitempty"`
	ScopeID      string `json:"scopeId,omitempty"`
	RecordID     string `json:"recordId,omitempty"`
	ResultID     string `json:"resultId,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`
	Detail       string `json:"detail"`
	NextStep     string `json:"nextStep"`
}

// AuditCanonicalController audits every canonical Projection and the complete
// configured target-root inventory. It performs no Apply, command check,
// Executor, Verifier, or ledger write. An incomplete audit is a report with an
// incomplete status; invalid configuration and failed input acquisition are
// returned as errors.
func AuditCanonicalController(root, base, revision, configPath string, cfg CanonicalControllerConfig) (CanonicalControllerAudit, error) {
	report := CanonicalControllerAudit{
		APIVersion:   canonicalControllerAuditAPIVersion,
		Status:       records.OutcomeIncomplete,
		Coverage:     canonicalControllerAuditCoverage,
		BaseRevision: base, SourceRevision: revision,
		Projections: []CanonicalControllerAuditProjection{},
		Exclusions:  []CanonicalExcludedArtifact{},
		Findings:    []CanonicalControllerAuditFinding{},
		NextSteps:   []string{},
	}
	if !cfg.AuditAll {
		return report, errors.New("controller audit requires auditAll=true; runtime configuration is part of verification evidence and is not changed by audit")
	}
	if err := validateControllerConfig(cfg); err != nil {
		return report, err
	}
	if !canonicalRevisionPattern.MatchString(base) || !canonicalRevisionPattern.MatchString(revision) {
		return report, errors.New("controller audit requires full immutable base and source commit IDs")
	}

	fixed, err := LoadSelectedCanonicalSource(root, revision, configPath, true)
	if err != nil {
		return report, err
	}
	if fixed.Snapshot == nil || fixed.Snapshot.ID != revision || fixed.Snapshot.Provisional {
		return report, errors.New("selected canonical source does not bind the exact immutable source revision")
	}
	if err := validateCanonicalControllerAssurance(cfg, fixed); err != nil {
		return report, err
	}
	configDigest, err := digestCanonicalValue(cfg)
	if err != nil {
		return report, err
	}
	report.ConfigDigest = configDigest

	_, initialState, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	report.LedgerHead = initialState.Head
	report.LedgerSelectionDigest = initialState.ActiveSelection.Digest

	// Reconciliation proposal is the existing read-only full-content audit. Its
	// cached bindings are reused only when every active Projection is covered by
	// the configured assurance graph. Missing evidence remains a report finding.
	coverageOK := canonicalControllerAuditActiveScopeCoverage(cfg, active)
	var proposal CanonicalControllerProposal
	if coverageOK {
		proposal, err = ProposeCanonicalController(root, base, revision, configPath, cfg)
		if err != nil {
			return report, err
		}
	} else {
		plan, planErr := proposeScopedCanonicalReconciliation(root, base, revision, configPath, active, true, cfg.TargetExclusions, canonicalControllerCheckInputs(cfg))
		if planErr != nil {
			return report, planErr
		}
		proposal = CanonicalControllerProposal{
			APIVersion:            CanonicalControllerAPIVersion,
			Status:                plan.Status,
			ConfigDigest:          configDigest,
			LedgerHead:            initialState.Head,
			LedgerSelectionDigest: initialState.ActiveSelection.Digest,
			Plan:                  plan,
			fixed:                 plan.fixed,
			active:                append([]records.ProjectionRecord(nil), active...),
		}
		proposal.Digest, err = digestCanonicalValue(proposal)
		if err != nil {
			return report, err
		}
	}
	if proposal.LedgerHead != initialState.Head || proposal.LedgerSelectionDigest != initialState.ActiveSelection.Digest {
		return report, errors.New("controller ledger changed while the audit proposal was being acquired")
	}
	report.ProposalDigest = proposal.Digest
	report.PlanStatus = proposal.Plan.Status
	report.Exclusions = append(report.Exclusions, proposal.Plan.ExplicitExcludedArtifacts...)

	var bindings map[string]canonicalProjectionVerificationBinding
	latest := canonicalControllerAuditLatestResults(initialState.Verifications)
	if coverageOK {
		reuse, reuseErr := newCanonicalControllerVerificationReuse(cfg, active, initialState.Verifications, configDigest)
		if reuseErr != nil {
			return report, reuseErr
		}
		bindings = reuse.bindings(root, proposal.fixed, active, proposal.observed)
	} else {
		bindings = map[string]canonicalProjectionVerificationBinding{}
	}

	requests, err := projectionRequestIndex(proposal.fixed)
	if err != nil {
		return report, err
	}
	activeByProjection := make(map[string]records.ProjectionRecord, len(active))
	for _, record := range active {
		activeByProjection[record.ProjectionID] = record
	}
	planByProjection := make(map[string]CanonicalScopedProposal, len(proposal.Plan.Proposals))
	for _, item := range proposal.Plan.Proposals {
		planByProjection[item.ProjectionID] = item
	}
	scopesByProjection := canonicalControllerAuditScopesByProjection(cfg, active)
	projectionIDs := make([]string, 0, len(requests))
	for id := range requests {
		projectionIDs = append(projectionIDs, id)
	}
	sort.Strings(projectionIDs)

	complete := true
	addFinding := func(finding CanonicalControllerAuditFinding) {
		report.Findings = append(report.Findings, finding)
		report.NextSteps = append(report.NextSteps, finding.NextStep)
		complete = false
	}
	for _, id := range projectionIDs {
		request := requests[id]
		item := CanonicalControllerAuditProjection{
			ProjectionID:      id,
			ScopeIDs:          requestScopeIDs(request.Definitions),
			PolicyIDs:         requestPolicyIDs(request.Policies),
			AssuranceScopeIDs: append([]string(nil), scopesByProjection[id]...),
			Status:            records.OutcomeIncomplete,
			Outcome:           records.OutcomeIncomplete,
		}
		if len(item.AssuranceScopeIDs) == 0 {
			addFinding(CanonicalControllerAuditFinding{Code: "assurance.scope-missing", ProjectionID: id, Detail: "canonical Projection has no configured assurance scope", NextStep: "Add the Projection to a reachable assurance scope with its declared checks, then create fresh verification evidence."})
		}
		record, hasRecord := activeByProjection[id]
		if !hasRecord {
			addFinding(CanonicalControllerAuditFinding{Code: "record.missing-active", ProjectionID: id, Detail: "canonical Projection has no active materialization record", NextStep: "Materialize and apply the declared Projection, then run fresh verification."})
		} else {
			item.ActiveRecordIDs = []string{record.ID}
			for _, artifact := range record.Artifacts {
				item.ArtifactPaths = append(item.ArtifactPaths, artifact.Path)
			}
			sort.Strings(item.ArtifactPaths)
			result, hasResult := latest[record.ID]
			binding, fresh := bindings[id]
			if hasResult {
				item.Outcome = result.Outcome
				for _, check := range result.Checks {
					item.CheckIDs = append(item.CheckIDs, check.ID)
				}
				sort.Strings(item.CheckIDs)
			}
			if !hasResult {
				addFinding(CanonicalControllerAuditFinding{Code: "verification.missing", ProjectionID: id, RecordID: record.ID, Detail: "active materialization has no verification result", NextStep: "Run controller verification against the exact current source and target evidence."})
			} else if !fresh {
				code := "verification.stale"
				detail := "latest verification result does not bind the current model, runtime, checks, child dependencies, and actual artifact bytes"
				if result.Outcome != records.OutcomePassed {
					code = "verification.not-passing"
					detail = "latest verification result is not passing"
				}
				addFinding(CanonicalControllerAuditFinding{Code: code, ProjectionID: id, RecordID: record.ID, ResultID: result.ID, Detail: detail, NextStep: "Resolve the reported verification state and obtain fresh verification for the current inputs."})
			} else {
				item.FreshVerificationResultIDs = []string{result.ID}
				if binding.Repair != nil || result.Outcome != records.OutcomePassed {
					addFinding(CanonicalControllerAuditFinding{Code: "verification.not-passing", ProjectionID: id, RecordID: record.ID, ResultID: result.ID, Detail: "fresh verification evidence is not a passing result", NextStep: "Resolve the verification findings and obtain a fresh passing local and parent result."})
				}
			}
		}
		if planItem, ok := planByProjection[id]; !ok {
			addFinding(CanonicalControllerAuditFinding{Code: "projection.unplanned", ProjectionID: id, Detail: "canonical Projection is absent from the full-scope reconciliation plan", NextStep: "Resolve the planning gap before declaring the declared scope complete."})
		} else {
			switch planItem.Decision {
			case "no-op":
				if planItem.EvidenceRefreshRequired {
					addFinding(CanonicalControllerAuditFinding{Code: "projection.refresh-required", ProjectionID: id, Detail: "Projection requires evidence refresh", NextStep: "Refresh the retained evidence and run fresh verification before closure."})
				} else if item.Outcome == records.OutcomePassed && len(item.FreshVerificationResultIDs) > 0 && len(item.AssuranceScopeIDs) > 0 {
					item.Status = records.OutcomePassed
				}
			case "escalate":
				addFinding(CanonicalControllerAuditFinding{Code: "projection.escalated", ProjectionID: id, Detail: "Projection reconciliation has an unresolved escalation", NextStep: "Resolve the ownership, scope, or implementation escalation, then audit again."})
			default:
				addFinding(CanonicalControllerAuditFinding{Code: "projection.work-required", ProjectionID: id, Detail: "Projection reconciliation still requires work", NextStep: "Execute and review the bounded Projection work, apply it through the declared boundary, then verify again."})
			}
		}
		report.Projections = append(report.Projections, item)
	}
	for _, escalation := range proposal.Plan.Escalations {
		if escalation.Code == "artifact.unowned" {
			continue
		}
		addFinding(CanonicalControllerAuditFinding{Code: "reconcile.escalated", ProjectionID: escalation.Identity, Detail: escalation.Message, NextStep: "Resolve the reconciliation escalation before declaring the declared scope complete."})
	}
	for _, projectionID := range proposal.Plan.EvidenceRefreshRequired {
		addFinding(CanonicalControllerAuditFinding{Code: "projection.refresh-required", ProjectionID: projectionID, Detail: "Projection has stale canonical evidence requiring refresh", NextStep: "Refresh the retained evidence and obtain current verification before closure."})
	}
	for _, projectionID := range proposal.Plan.UnobservedProjections {
		addFinding(CanonicalControllerAuditFinding{Code: "projection.unobserved", ProjectionID: projectionID, Detail: "Projection target content was not included in the current full-scope observation", NextStep: "Run the audit with full declared target-root observation before closure."})
	}
	for _, path := range proposal.Plan.UnknownArtifacts {
		// Exact configured exclusions are absent from UnknownArtifacts. Keep this
		// guard in case the planner's classification changes in a future version.
		if canonicalControllerAuditExcludedPath(path, proposal.Plan.ExplicitExcludedArtifacts) {
			continue
		}
		addFinding(CanonicalControllerAuditFinding{Code: "artifact.unowned", ArtifactPath: path, Detail: "declared target inventory contains an unowned artifact", NextStep: "Resolve ownership for this target artifact, then audit again."})
	}
	if proposal.Plan.Status != "planned" {
		addFinding(CanonicalControllerAuditFinding{Code: "reconcile.plan-incomplete", Detail: "full-scope reconciliation plan is not in planned state", NextStep: "Resolve plan acquisition or impact uncertainty, then audit again."})
	}

	_, finalState, _, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	if finalState.Head != initialState.Head || finalState.ActiveSelection.Digest != initialState.ActiveSelection.Digest {
		addFinding(CanonicalControllerAuditFinding{Code: "ledger.changed-during-audit", Detail: "controller ledger head or active selection changed during the read-only audit", NextStep: "Rerun the audit against the stable current ledger."})
	}

	sort.Slice(report.Projections, func(i, j int) bool { return report.Projections[i].ProjectionID < report.Projections[j].ProjectionID })
	sort.Slice(report.Findings, func(i, j int) bool {
		left, right := report.Findings[i], report.Findings[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.ProjectionID != right.ProjectionID {
			return left.ProjectionID < right.ProjectionID
		}
		if left.ScopeID != right.ScopeID {
			return left.ScopeID < right.ScopeID
		}
		if left.ArtifactPath != right.ArtifactPath {
			return left.ArtifactPath < right.ArtifactPath
		}
		return left.Detail < right.Detail
	})
	report.NextSteps = sortedUniqueStrings(report.NextSteps)
	if complete && len(report.Findings) == 0 {
		report.Status = "complete"
	}
	report.Digest, err = digestCanonicalValue(report)
	return report, err
}

func canonicalControllerAuditActiveScopeCoverage(cfg CanonicalControllerConfig, active []records.ProjectionRecord) bool {
	if len(cfg.AssuranceScopes) == 0 || len(active) == 0 {
		return true
	}
	covered := make(map[string]bool, len(cfg.AssuranceScopes))
	for _, scope := range cfg.AssuranceScopes {
		if covered[scope.ProjectionID] {
			return false
		}
		covered[scope.ProjectionID] = true
	}
	for _, record := range active {
		if !covered[record.ProjectionID] {
			return false
		}
	}
	return true
}

func canonicalControllerAuditLatestResults(results []records.VerificationResult) map[string]records.VerificationResult {
	latest := make(map[string]records.VerificationResult, len(results))
	for _, result := range results {
		latest[result.RecordID] = result
	}
	return latest
}

func canonicalControllerAuditScopesByProjection(cfg CanonicalControllerConfig, active []records.ProjectionRecord) map[string][]string {
	result := map[string][]string{}
	if len(cfg.AssuranceScopes) == 0 {
		for _, record := range active {
			result[record.ProjectionID] = append(result[record.ProjectionID], record.ProjectionID)
		}
	} else {
		for _, scope := range cfg.AssuranceScopes {
			result[scope.ProjectionID] = append(result[scope.ProjectionID], scope.ID)
		}
	}
	for projectionID := range result {
		sort.Strings(result[projectionID])
	}
	return result
}

func requestPolicyIDs(policies []core.Definition) []string {
	ids := make([]string, len(policies))
	for i, policy := range policies {
		ids[i] = policy.Identity().Key()
	}
	sort.Strings(ids)
	return ids
}

func canonicalControllerAuditExcludedPath(path string, exclusions []CanonicalExcludedArtifact) bool {
	for _, exclusion := range exclusions {
		if exclusion.Path == path {
			return true
		}
	}
	return false
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if value == "" {
			continue
		}
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
