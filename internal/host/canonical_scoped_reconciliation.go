package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/dotnet"
	"github.com/Glacius-Labs/Markitect/internal/modules/markdown"
)

type CanonicalScopedProposal struct {
	ProjectionID            string                          `json:"projectionId"`
	Module                  canonical.Pin                   `json:"module"`
	Decision                string                          `json:"decision"`
	Reasons                 []string                        `json:"reasons"`
	Request                 canonical.ProjectionRequest     `json:"request"`
	Task                    *dotnet.ExecutorTask            `json:"task,omitempty"`
	Outputs                 map[string][]byte               `json:"outputs,omitempty"`
	EvidenceRefreshRequired bool                            `json:"evidenceRefreshRequired"`
	Escalations             []CanonicalProjectionEscalation `json:"escalations,omitempty"`
}

// CanonicalScopedReconcilePlan records actual acquisition scope. Metadata-only
// unrelated targets are visibly unobserved, not implicitly verified or clean.
type CanonicalScopedReconcilePlan struct {
	APIVersion              string                          `json:"apiVersion"`
	Digest                  string                          `json:"digest"`
	Status                  string                          `json:"status"`
	BaseRevision            string                          `json:"baseRevision"`
	Revision                string                          `json:"revision"`
	ModelDigest             string                          `json:"modelDigest"`
	SourceScope             *SelectedInputScope             `json:"sourceScope"`
	ObservedPaths           []string                        `json:"observedPaths"`
	ObservedDigest          string                          `json:"observedDigest"`
	Inventory               *source.WorkingRootInventory    `json:"inventory"`
	Impact                  CanonicalImpact                 `json:"impact"`
	Proposals               []CanonicalScopedProposal       `json:"proposals"`
	UnobservedProjections   []string                        `json:"unobservedProjections"`
	EvidenceRefreshRequired []string                        `json:"evidenceRefreshRequired"`
	UnknownArtifacts        []string                        `json:"unknownArtifacts"`
	Escalations             []CanonicalProjectionEscalation `json:"escalations"`
	fixed                   *CanonicalSource
	observed                *snapshot.Snapshot
}

// ProposeScopedCanonicalReconciliation reads exact canonical inputs and byte
// content only for affected/new/runtime-changed scopes (or explicit auditAll).
// It never invokes agents, writes a ledger, mutates artifacts or grants Apply.
func ProposeScopedCanonicalReconciliation(root, baseRevision, revision, configPath string, active []records.ProjectionRecord, auditAll bool) (CanonicalScopedReconcilePlan, error) {
	plan := CanonicalScopedReconcilePlan{APIVersion: "markitect.canonical/scoped-reconcile/v1alpha1", Status: "planned"}
	base, err := LoadSelectedCanonicalSource(root, baseRevision, configPath, true)
	if err != nil {
		return plan, err
	}
	current, err := LoadSelectedCanonicalSource(root, revision, configPath, true)
	if err != nil {
		return plan, err
	}
	if !equalCanonicalValue(base.AcquisitionScope.Repository,current.AcquisitionScope.Repository) { return plan, errors.New("repository identity changed between base and candidate acquisition") }
	if len(base.Diagnostics) > 0 || len(current.Diagnostics) > 0 {
		return plan, errors.New("scoped reconcile requires structurally valid canonical models")
	}
	impact, err := AnalyzeCanonicalImpact(base, current, active)
	if err != nil {
		return plan, err
	}
	plan.Impact = impact
	plan.BaseRevision = baseRevision
	plan.Revision = revision
	plan.ModelDigest = current.Model.Digest
	plan.SourceScope = current.AcquisitionScope
	plan.fixed = current
	requests, err := projectionRequestIndex(current)
	if err != nil {
		return plan, err
	}
	if len(requests) > 128 {
		return plan, errors.New("scoped reconcile exceeds 128 Projection work scopes")
	}
	// This validates portable aliases and duplicate active owners without treating
	// the absent content facts as current verification.
	ownership, err := records.BuildOwnershipIndex(active, nil)
	if err != nil {
		return plan, err
	}
	prior := map[string]records.ProjectionRecord{}
	for _, record := range active {
		if _, found := prior[record.ProjectionID]; found {
			return plan, errors.New("multiple active records for one Projection")
		}
		prior[record.ProjectionID] = record
	}
	affected := map[string]bool{}
	for _, subject := range impact.ScopeAffectedProjections {
		affected[subject.ProjectionID] = true
	}
	keys := []string{}
	prefixes := []string{}
	for key := range requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		request := requests[key]
		if request.TargetPrefix == "" {
			return plan, errors.New("scoped reconcile requires bounded non-root target prefixes")
		}
		for _, existing := range prefixes {
			if canonicalPrefixesOverlap(strings.ToLower(existing), strings.ToLower(request.TargetPrefix)) {
				return plan, fmt.Errorf("overlapping Projection target roots require explicit ownership resolution: %s and %s", existing, request.TargetPrefix)
			}
		}
		prefixes = append(prefixes, request.TargetPrefix)
	}
	sort.Strings(keys)
	prefixes = sortedUniquePaths(prefixes)
	for i, left := range prefixes {
		for _, right := range prefixes[i+1:] {
			if canonicalPrefixesOverlap(strings.ToLower(left), strings.ToLower(right)) {
				return plan, fmt.Errorf("overlapping Projection target roots require explicit ownership resolution: %s and %s", left, right)
			}
		}
	}
	inventory, err := source.InventoryWorkingRoots(root, prefixes)
	if err != nil {
		return plan, err
	}
	if !equalCanonicalValue(inventory.Identity, current.AcquisitionScope.Repository) {
		return plan, errors.New("repository identity changed between canonical acquisition and inventory")
	}
	plan.Inventory = inventory
	inventoryByPath := map[string]source.WorkingFileMetadata{}
	for _, entry := range inventory.Entries {
		inventoryByPath[entry.Path] = entry
	}
	canonicalPaths := canonicalSourcePaths(current)
	protected := map[string]bool{}
	for _, name := range canonicalPaths {
		protected[name] = true
		for _, prefix := range prefixes {
			if strings.EqualFold(name, prefix) || stringsHasPathPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				return plan, fmt.Errorf("Projection target root overlaps canonical input: %s", name)
			}
		}
	}
	for _, entry := range inventory.Entries {
		if _, owned := ownership.Artifacts[entry.Path]; !owned && !protected[entry.Path] {
			plan.UnknownArtifacts = append(plan.UnknownArtifacts, entry.Path)
		}
	}
	if len(plan.UnknownArtifacts) > 0 {
		plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "artifact.unowned", Message: "declared target inventory contains unowned artifacts; review ownership before execution"})
	}
	selectedPrefixes := map[string]bool{}
	scheduled := map[string]bool{}
	canonicalWork := map[string]bool{}
	for _, key := range keys {
		request := requests[key]
		old, found := prior[key]
		metadataDrift := false
		if found {
			for _, artifact := range old.Artifacts {
				entry, present := inventoryByPath[artifact.Path]
				if !present || entry.Mode != artifact.Mode {
					metadataDrift = true
				}
				if !stringsHasPathPrefix(artifact.Path, request.TargetPrefix) {
					plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "artifact.owner-scope-changed", Identity: key, Message: "active artifact is outside the current Projection target prefix: " + artifact.Path})
				}
			}
		}
		bindingChanged := found && (old.Module.Name != request.ModulePin.Name || old.Module.Version != request.ModulePin.Version || old.Module.Digest != request.ModulePin.Digest || old.Projector.ID != request.Projector.ID || old.Projector.Version != request.Projector.Version)
		canonicalWork[key] = bindingChanged || (affected[key] && (!found || old.ModelDigest != request.ModelDigest || old.Revision != request.Revision))
		scheduled[key] = auditAll || metadataDrift || canonicalWork[key] || !found || old.State != records.StateMaterializedUnverified
		if scheduled[key] {
			selectedPrefixes[request.TargetPrefix] = true
		} else {
			plan.UnobservedProjections = append(plan.UnobservedProjections, key)
			if old.ModelDigest != request.ModelDigest || old.Revision != request.Revision {
				plan.EvidenceRefreshRequired = append(plan.EvidenceRefreshRequired, key)
			}
		}
	}
	priorKeys := []string{}
	for key := range prior {
		priorKeys = append(priorKeys, key)
	}
	sort.Strings(priorKeys)
	for _, key := range priorKeys {
		if _, found := requests[key]; !found {
			plan.Escalations = append(plan.Escalations, CanonicalProjectionEscalation{Code: "projection.retired-owner", Identity: key, Message: "active ownership remains for a removed Projection; no automatic deletion or transfer"})
		}
	}
	paths := append([]string(nil), canonicalPaths...)
	for _, entry := range inventory.Entries {
		for prefix := range selectedPrefixes {
			if stringsHasPathPrefix(entry.Path, prefix) {
				paths = append(paths, entry.Path)
				break
			}
		}
	}
	paths = sortedUniquePaths(paths)
	observed, err := source.ObserveSelectedWorking(root, paths)
	if err != nil {
		return plan, err
	}
	if !equalCanonicalValue(observed.Identity, current.AcquisitionScope.Repository) {
		return plan, errors.New("repository identity changed between canonical acquisition and target observation")
	}
	plan.observed = observed.Snapshot
	plan.ObservedPaths = paths
	plan.ObservedDigest = sha256Prefix(observed.Snapshot.Digest())
	if err := validateCanonicalSourceUnchanged(current, observed.Snapshot); err != nil {
		return plan, err
	}
	// Unknown paths are metadata only unless their scope was scheduled. They
	// still stop execution; skipped unrelated bytes never become clean facts.
	for _, key := range keys {
		if !scheduled[key] {
			continue
		}
		request := requests[key]
		targetFiles, err := canonicalProjectionTargetFiles(request.Projection, observed.Snapshot.Files)
		if err != nil {
			return plan, err
		}
		request, err = canonical.BindProjection(current.Model, current.Activation, current.Config.ProjectionBindings, request.Projection.Identity(), targetFiles)
		if err != nil {
			return plan, err
		}
		isCandidate, err := selectedHostProjector(request)
		if err != nil {
			return plan, err
		}
		p := CanonicalScopedProposal{ProjectionID: key, Module: request.ModulePin, Request: request}
		old, found := prior[key]
		if isCandidate {
			input := dotnet.Input{Definitions: request.Definitions, Schemas: request.Schemas, Policies: request.Policies, TargetPrefix: request.TargetPrefix, AllowedRoots: request.Projector.AllowedRoots, RequestDigest: request.RequestDigest, CanonicalAffected: canonicalWork[key], InventoryComplete: true}
			for _, entry := range inventory.Entries {
				if stringsHasPathPrefix(entry.Path, request.TargetPrefix) {
					input.ObservedArtifacts = append(input.ObservedArtifacts, dotnet.ArtifactObservation{Path: entry.Path, Bytes: observed.Snapshot.Files[entry.Path], Mode: entry.Mode})
				}
			}
			if found {
				input.Previous = &dotnet.PriorProjection{RequestDigest: old.RequestDigest, Complete: old.State == records.StateMaterializedUnverified}
				for _, a := range old.Artifacts {
					input.Previous.Artifacts = append(input.Previous.Artifacts, dotnet.ArtifactBinding{Path: a.Path, Digest: a.Digest, Mode: a.Mode})
				}
			}
			proposed := dotnet.Propose(input)
			p.Decision = string(proposed.Decision)
			p.Reasons = proposed.Reasons
			p.Task = proposed.Task
			p.EvidenceRefreshRequired = proposed.EvidenceRefreshRequired
			for _, e := range proposed.Escalations {
				p.Escalations = append(p.Escalations, CanonicalProjectionEscalation{Code: e.Code, Identity: e.Identity, Message: e.Message})
			}
		} else {
			input := markdown.Input{Definitions: request.Definitions, Schemas: request.Schemas, Policies: request.Policies, TargetPrefix: request.TargetPrefix, AllowedRoots: request.Projector.AllowedRoots, RequestDigest: request.RequestDigest, CanonicalAffected: canonicalWork[key], InventoryComplete: true}
			for _, entry := range inventory.Entries {
				if stringsHasPathPrefix(entry.Path, request.TargetPrefix) {
					input.ObservedArtifacts = append(input.ObservedArtifacts, markdown.ArtifactObservation{Path: entry.Path, Bytes: observed.Snapshot.Files[entry.Path], Mode: entry.Mode})
				}
			}
			if found {
				input.Previous = &markdown.PriorProjection{RequestDigest: old.RequestDigest, Complete: old.State == records.StateMaterializedUnverified}
				for _, a := range old.Artifacts {
					input.Previous.Artifacts = append(input.Previous.Artifacts, markdown.ArtifactBinding{Path: a.Path, Digest: a.Digest, Mode: a.Mode})
				}
			}
			proposed := markdown.Propose(input)
			p.Decision = string(proposed.Decision)
			p.Reasons = proposed.Reasons
			p.Outputs = proposed.Files
			p.EvidenceRefreshRequired = proposed.EvidenceRefreshRequired
			for _, e := range proposed.Escalations {
				p.Escalations = append(p.Escalations, CanonicalProjectionEscalation{Code: e.Code, Identity: key, Message: e.Message})
			}
		}
		if p.EvidenceRefreshRequired {
			plan.EvidenceRefreshRequired = append(plan.EvidenceRefreshRequired, key)
		}
		if len(p.Escalations) > 0 {
			plan.Escalations = append(plan.Escalations, p.Escalations...)
		}
		plan.Proposals = append(plan.Proposals, p)
	}
	plan.UnknownArtifacts = sortedUniquePaths(plan.UnknownArtifacts)
	plan.EvidenceRefreshRequired = sortedUniquePaths(plan.EvidenceRefreshRequired)
	if len(plan.Escalations) > 0 {
		plan.Status = "escalated"
	}
	plan.Digest, err = digestCanonicalValue(plan)
	return plan, err
}

func digestCanonicalValue(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return sha256Prefix(sha256Hex(data)), nil
}
