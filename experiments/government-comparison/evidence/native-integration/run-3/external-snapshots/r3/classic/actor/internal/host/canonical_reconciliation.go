package host

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const CanonicalReconciliationPlanAPIVersion = "markitect.canonical/reconcile-plan/v1alpha1"

type CanonicalWorkProposal struct {
	ProjectionID         string                      `json:"projectionId"`
	Module               canonical.Pin               `json:"module"`
	Reasons              []string                    `json:"reasons"`
	Request              canonical.ProjectionRequest `json:"request"`
	OwnedArtifacts       []string                    `json:"ownedArtifacts"`
	VerificationRequired bool                        `json:"verificationRequired"`
}

type CanonicalReconciliationPlan struct {
	APIVersion              string                          `json:"apiVersion"`
	Digest                  string                          `json:"digest"`
	Status                  string                          `json:"status"`
	InputSnapshotDigest     string                          `json:"inputSnapshotDigest"`
	ActiveRecordIDs         []string                        `json:"activeRecordIds"`
	Impact                  CanonicalImpact                 `json:"impact"`
	Work                    []CanonicalWorkProposal         `json:"work"`
	EvidenceRefreshRequired []string                        `json:"evidenceRefreshRequired"`
	NoApplicableWork        []string                        `json:"noApplicableWork"`
	Ownership               records.OwnershipIndex          `json:"ownership"`
	Escalations             []CanonicalProjectionEscalation `json:"escalations"`
}

// PlanCanonicalReconciliation derives representation work from intent delta and
// supplied active ownership, not from manually selected implementation files.
// It is read-only. Work requests are bounded Executor inputs, not source-code
// edits, automatic adoption, semantic PASS, or authorization to Apply.
func PlanCanonicalReconciliation(base, current *CanonicalSource, observed *snapshot.Snapshot, active []records.ProjectionRecord) (CanonicalReconciliationPlan, error) {
	plan := CanonicalReconciliationPlan{APIVersion: CanonicalReconciliationPlanAPIVersion, Status: "planned"}
	if err := validateCanonicalProjectionSnapshots(current, observed); err != nil {
		return plan, err
	}
	if err := validateCanonicalSourceUnchanged(current, observed); err != nil {
		return plan, err
	}
	impact, err := AnalyzeCanonicalImpact(base, current, active)
	if err != nil {
		return plan, err
	}
	plan.Impact = impact
	plan.InputSnapshotDigest = sha256Prefix(observed.Digest())
	prior := map[string]records.ProjectionRecord{}
	for _, record := range active {
		if _, exists := prior[record.ProjectionID]; exists {
			return plan, fmt.Errorf("multiple active records supplied for Projection %s", record.ProjectionID)
		}
		prior[record.ProjectionID] = record
		plan.ActiveRecordIDs = append(plan.ActiveRecordIDs, record.ID)
	}
	sort.Strings(plan.ActiveRecordIDs)
	requests, err := projectionRequestIndex(current)
	if err != nil {
		return plan, err
	}
	paths := map[string]bool{}
	for key, request := range requests {
		targetFiles, err := canonicalProjectionTargetFiles(request.Projection, observed.Files)
		if err != nil {
			return plan, err
		}
		bound, err := canonical.BindProjection(current.Model, current.Activation, current.Config.ProjectionBindings, request.Projection.Identity(), targetFiles)
		if err != nil {
			return plan, err
		}
		requests[key] = bound
		for name := range targetFiles {
			paths[name] = true
		}
	}
	for _, record := range active {
		for _, artifact := range record.Artifacts {
			paths[artifact.Path] = true
		}
	}
	canonicalPaths := map[string]bool{}
	for _, name := range canonicalSourcePaths(current) {
		canonicalPaths[name] = true
	}
	facts := []records.ArtifactFact{}
	for name := range paths {
		data, ok := observed.Files[name]
		if !ok {
			continue
		}
		mode := observed.Modes[name]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		role := records.RoleProjectionTarget
		if canonicalPaths[name] {
			role = records.RoleCanonicalSource
		}
		facts = append(facts, records.ArtifactFact{Path: name, Role: role, Digest: sha256Prefix(sha256Hex(data)), Mode: mode})
	}
	plan.Ownership, err = records.BuildOwnershipIndex(active, facts)
	if err != nil {
		return plan, err
	}
	orderedPaths := []string{}
	for name := range plan.Ownership.Artifacts {
		orderedPaths = append(orderedPaths, name)
	}
	sort.Strings(orderedPaths)
	for _, name := range orderedPaths {
		entry := plan.Ownership.Artifacts[name]
		if entry.Status == records.OwnershipUnknown {
			plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "artifact.unowned", Identity: name, Message: "configured target contains an artifact with no supplied active projection owner; owner review is required"})
		}
	}
	affected := map[string]bool{}
	for _, projection := range impact.ScopeAffectedProjections {
		affected[projection.ProjectionID] = true
	}
	keys := []string{}
	for key := range requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		request := requests[key]
		old, exists := prior[key]
		reasons := []string{}
		owned := []string{}
		if affected[key] {
			reasons = append(reasons, "canonical-or-binding-change")
		}
		if !exists {
			reasons = append(reasons, "not-materialized")
		}
		if exists {
			if old.State != records.StateMaterializedUnverified {
				reasons = append(reasons, "incomplete-materialization")
			}
			drift := false
			for _, artifact := range old.Artifacts {
				owned = append(owned, artifact.Path)
				entry := plan.Ownership.Artifacts[artifact.Path]
				if entry.Status == records.OwnershipDrift || entry.Status == records.OwnershipUnobserved {
					drift = true
				}
			}
			if drift {
				reasons = append(reasons, "projection-drift")
			}
			if old.Module.Name != request.ModulePin.Name || old.Module.Version != request.ModulePin.Version || old.Module.Digest != request.ModulePin.Digest || old.Projector.ID != request.Projector.ID || old.Projector.Version != request.Projector.Version {
				reasons = append(reasons, "runtime-capability-changed")
			}
		}
		if len(reasons) == 0 {
			if exists && (old.Revision != current.Model.Revision || old.ModelDigest != request.ModelDigest) {
				plan.EvidenceRefreshRequired = append(plan.EvidenceRefreshRequired, key)
			}
			plan.NoApplicableWork = append(plan.NoApplicableWork, key)
			continue
		}
		plan.Work = append(plan.Work, CanonicalWorkProposal{ProjectionID: key, Module: request.ModulePin, Reasons: sortedUniquePaths(reasons), Request: request, OwnedArtifacts: sortedUniquePaths(owned), VerificationRequired: true})
	}
	for _, proposal := range plan.Work {
		for name := range canonicalPaths {
			if name == proposal.Request.TargetPrefix || stringsHasPathPrefix(name, proposal.Request.TargetPrefix) {
				plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "projection.protected-target", Identity: proposal.ProjectionID, Message: "desired target overlaps canonical input " + name})
			}
		}
	}
	for key := range prior {
		if _, exists := requests[key]; !exists {
			plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "projection.retired-owner", Identity: key, Message: "an active record owns artifacts for a removed desired representation; deletion or transfer requires an explicit reviewed change"})
		}
	}
	// Target expertise currently proposes one Markdown page; .NET artifact
	// discovery remains an agent candidate or prior ownership. Reject known
	// location conflicts before handing either scope to an Executor.
	for i := 0; i < len(plan.Work); i++ {
		for j := i + 1; j < len(plan.Work); j++ {
			left, right := plan.Work[i].Request, plan.Work[j].Request
			if canonicalPrefixesOverlap(left.TargetPrefix, right.TargetPrefix) {
				plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "projection.target-overlap", Identity: plan.Work[i].ProjectionID, Message: "work target overlaps " + plan.Work[j].ProjectionID + "; exact artifact ownership must be resolved before execution"})
			}
		}
	}
	sort.Slice(plan.Escalations, func(i, j int) bool {
		a, b := plan.Escalations[i], plan.Escalations[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Identity != b.Identity {
			return a.Identity < b.Identity
		}
		return a.Message < b.Message
	})
	if len(plan.Escalations) > 0 {
		plan.Status = "escalated"
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return plan, err
	}
	plan.Digest = sha256Prefix(sha256Hex(encoded))
	return plan, nil
}
func canonicalPrefixesOverlap(a, b string) bool {
	return a == b || a == "" || b == "" || stringsHasPathPrefix(a, b) || stringsHasPathPrefix(b, a)
}
func stringsHasPathPrefix(file, prefix string) bool {
	return len(file) > len(prefix) && file[:len(prefix)] == prefix && file[len(prefix)] == '/'
}
