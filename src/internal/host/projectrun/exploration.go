package projectrun

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// ExplorationBinding computes the exact executable selection without creating
// a run or starting an agent. Stable readiness excludes random plan IDs.
func ExplorationBinding(host Host, root, revision string, request PlanRequest) (projectexplore.Binding, error) {
	explorationID, scopeID := request.ExplorationID, request.ScopeID
	if explorationID != "" {
		fixed, err := resolveGitHead(root)
		if err != nil {
			return projectexplore.Binding{}, err
		}
		if revision != "" && revision != fixed {
			return projectexplore.Binding{}, fmt.Errorf("readiness requires the active committed HEAD")
		}
		history, err := projectbriefing.ReadAcceptedHistory(root, fixed)
		if err != nil {
			return projectexplore.Binding{}, err
		}
		request.acceptedBriefings = &history.State
	}
	request.ExecuteAuthorized = false
	request.ExplorationID, request.ScopeID = "", ""
	plan, err := Plan(host, root, revision, request)
	if err != nil {
		return projectexplore.Binding{}, err
	}
	project, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return projectexplore.Binding{}, err
	}
	working, err := host.Load(root, "")
	if err != nil {
		return projectexplore.Binding{}, err
	}
	if working == nil || working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot {
		return projectexplore.Binding{}, ErrStale
	}
	request.ExplorationID, request.ScopeID = explorationID, scopeID
	return explorationBindingFromPlan(root, project, working.Snapshot, request, plan)
}

// LoadExplorationReadiness is the shared native contributor entry point.
func LoadExplorationReadiness(host Host, root, explorationID, scopeID string) (projectexplore.Record, projectexplore.Scope, projectexplore.Binding, projectexplore.ReadinessReport, error) {
	record, err := projectexplore.Load(root, explorationID)
	var scope projectexplore.Scope
	var binding projectexplore.Binding
	var ready projectexplore.ReadinessReport
	if err != nil {
		return record, scope, binding, ready, err
	}
	for _, value := range record.Scopes {
		if value.ID == scopeID {
			scope = value
			break
		}
	}
	if scope.ID == "" {
		return record, scope, binding, ready, fmt.Errorf("unknown exploration scope %q", scopeID)
	}
	binding, err = ExplorationBinding(host, root, "", PlanRequest{Goal: scope.Goal, Operation: scope.Operation, Managers: scope.ManagerIDs, ExplorationID: explorationID, ScopeID: scopeID})
	if err != nil {
		return record, scope, binding, ready, err
	}
	ready, err = projectexplore.EvaluateReadiness(record, scopeID, binding)
	return record, scope, binding, ready, err
}

func explorationBindingFromPlan(root string, project *Project, working *Snapshot, request PlanRequest, plan PlanRecord) (projectexplore.Binding, error) {
	var binding projectexplore.Binding
	if project == nil || project.Snapshot == nil || working == nil {
		return binding, fmt.Errorf("exploration requires fixed and working project snapshots")
	}
	if working.Digest() != plan.WorkingSnapshot {
		return binding, ErrStale
	}
	if request.ModelEdit != nil || plan.ModelEdit != nil {
		return binding, fmt.Errorf("draft model edits are not an accepted readiness basis")
	}
	if len(plan.Blockers) != 0 {
		return binding, fmt.Errorf("implementation selection is blocked: %s", strings.Join(plan.Blockers, "; "))
	}
	scopeName := request.ScopeID
	if request.ExplorationID != "" {
		record, err := projectexplore.Load(root, request.ExplorationID)
		if err != nil {
			return binding, err
		}
		for _, scope := range record.Scopes {
			if scope.ID == request.ScopeID {
				scopeName = scope.Name
				break
			}
		}
	}
	stable := plan
	stable.ID, stable.Status, stable.InitialCandidateID = "", "", ""
	stable.ExecuteAuthorized = false
	stable.ExplorationID, stable.ScopeID, stable.ExplorationDigest, stable.ReadinessDigest = "", "", "", ""
	stable.ExplorationBinding = nil
	selection, err := planDigest(stable)
	if err != nil {
		return binding, err
	}
	briefings, err := digest(plan.BriefingDigests)
	if err != nil {
		return binding, err
	}
	policy := project.Config.AcceptancePolicy
	if policy == "" {
		policy = "committed-model"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return binding, err
	}
	binding = projectexplore.Binding{
		RepositoryRoot: abs, Branch: plan.TargetBranch, Head: plan.TargetHead,
		ModelRevision: plan.BaseRevision, ModelAccepted: !project.Provisional && plan.BaseRevision == plan.TargetHead && policy == "committed-model",
		AcceptancePolicy: policy, ProjectDigest: explorationDigest(plan.BaseProjectDigest), ModelDigest: explorationDigest(plan.ModelDigest), SnapshotDigest: explorationDigest(plan.BaseSnapshot),
		SelectionDigest: explorationDigest(selection), RuntimeDigest: explorationDigest(plan.RuntimeDigest), BriefingDigest: explorationDigest(briefings),
		ScopeID: request.ScopeID, ScopeName: scopeName, Goal: plan.Goal, Operation: plan.Operation,
		ManagerIDs: append([]string{}, request.Managers...), RequiredArtifacts: []string{}, FileStructure: []string{}, Checks: []string{}, BasisFiles: []projectexplore.BasisFile{},
	}
	binding.ResponsibleManagerIDs = []string{}
	for _, task := range plan.Managers {
		binding.ResponsibleManagerIDs = append(binding.ResponsibleManagerIDs, task.ManagerID)
	}
	binding.ResponsibleManagerIDs = uniqueSorted(binding.ResponsibleManagerIDs)
	for _, artifact := range project.Report.Artifacts {
		if artifact.Required {
			binding.RequiredArtifacts = append(binding.RequiredArtifacts, artifact.ID)
		}
		binding.FileStructure = append(binding.FileStructure, artifact.Paths...)
	}
	for _, file := range project.Report.Files {
		binding.FileStructure = append(binding.FileStructure, file.Path)
	}
	if project.Config.DocumentPath != "" {
		binding.FileStructure = append(binding.FileStructure, project.Config.DocumentPath)
	}
	for _, check := range plan.Checks {
		binding.Checks = append(binding.Checks, check.ID)
	}
	paths := append([]string{projectwork.ManifestPath, projectwork.RuntimePath}, project.Config.ModelFiles...)
	for _, path := range uniqueSorted(paths) {
		if data, ok := working.Files[path]; ok {
			binding.BasisFiles = append(binding.BasisFiles, projectexplore.BasisFile{Path: path, Digest: explorationBytesDigest(data)})
		}
	}
	binding.ManagerIDs = uniqueSorted(binding.ManagerIDs)
	binding.RequiredArtifacts = uniqueSorted(binding.RequiredArtifacts)
	binding.FileStructure = uniqueSorted(binding.FileStructure)
	binding.Checks = uniqueSorted(binding.Checks)
	if binding.ManagerIDs == nil {
		binding.ManagerIDs = []string{}
	}
	if binding.RequiredArtifacts == nil {
		binding.RequiredArtifacts = []string{}
	}
	if binding.FileStructure == nil {
		binding.FileStructure = []string{}
	}
	if binding.Checks == nil {
		binding.Checks = []string{}
	}
	_, err = projectexplore.BindingDigest(binding)
	return binding, err
}

func bindExplorationReadiness(root string, project *Project, working *Snapshot, request PlanRequest, plan *PlanRecord) error {
	if !request.ExecuteAuthorized && request.ExplorationID == "" {
		return nil
	}
	if request.ExplorationID == "" && request.ScopeID == "" && project.Config.WorkflowMode != "guided" {
		return nil
	}
	if request.ExplorationID == "" || request.ScopeID == "" {
		return fmt.Errorf("guided implementation requires --exploration and --scope with current computed readiness")
	}
	binding, err := explorationBindingFromPlan(root, project, working, request, *plan)
	if err != nil {
		return err
	}
	record, err := projectexplore.Load(root, request.ExplorationID)
	if err != nil {
		return err
	}
	ready, err := projectexplore.EvaluateReadiness(record, request.ScopeID, binding)
	if err != nil {
		return err
	}
	if !ready.Ready {
		return fmt.Errorf("scope is not ready: %s", strings.Join(ready.Blockers, "; "))
	}
	plan.ExplorationID, plan.ScopeID = record.ID, request.ScopeID
	plan.ExplorationDigest, plan.ReadinessDigest, plan.ExplorationBinding = record.Digest, ready.Digest, &binding
	return nil
}

func validateExplorationReadiness(root string, plan PlanRecord) error {
	if plan.ExplorationID == "" && plan.ScopeID == "" && plan.ExplorationBinding == nil {
		return nil
	}
	if plan.ExplorationID == "" || plan.ScopeID == "" || plan.ExplorationBinding == nil {
		return fmt.Errorf("incomplete exploration plan binding")
	}
	record, err := projectexplore.Load(root, plan.ExplorationID)
	if err != nil {
		return err
	}
	if record.Digest != plan.ExplorationDigest {
		return fmt.Errorf("%w: exploration changed after planning", ErrStale)
	}
	ready, err := projectexplore.EvaluateReadiness(record, plan.ScopeID, *plan.ExplorationBinding)
	if err != nil {
		return err
	}
	if !ready.Ready || ready.Digest != plan.ReadinessDigest {
		return fmt.Errorf("%w: exploration readiness changed", ErrStale)
	}
	return nil
}

func explorationBytesDigest(data []byte) string {
	value := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(value[:])
}

func explorationApplyCapturePaths(plan PlanRecord, paths []string) ([]string, error) {
	if plan.ExplorationID == "" {
		return paths, nil
	}
	path, err := projectexplore.RecordPath(plan.ExplorationID)
	if err != nil {
		return nil, err
	}
	return uniqueSorted(append(paths, path)), nil
}

func completeExploration(root string, plan PlanRecord, candidate candidateData, verification VerifyReport, applied ApplyReport) error {
	if err := recordDeliveryResolution(root, plan, candidate, verification, applied); err != nil {
		return err
	}
	if plan.ExplorationID == "" {
		return nil
	}
	if plan.ExplorationBinding == nil || applied.Status != StatusApplied || applied.RunID != plan.ID || applied.CandidateID != candidate.ID || verification.Status != "verified" || verification.RunID != plan.ID || verification.CandidateHash != candidate.Digest {
		return fmt.Errorf("exploration completion lacks matching successful Apply and Verify")
	}
	bindingDigest, err := projectexplore.BindingDigest(*plan.ExplorationBinding)
	if err != nil {
		return err
	}
	structureDigest, err := projectexplore.StructureDigest(*plan.ExplorationBinding)
	if err != nil {
		return err
	}
	applyDigest, err := digest(applied)
	if err != nil {
		return err
	}
	_, err = projectexplore.Complete(root, plan.ExplorationID, plan.ScopeID, *plan.ExplorationBinding, projectexplore.ApplyReceipt{
		ScopeID: plan.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Status: StatusApplied,
		RunID: plan.ID, PlanID: plan.ID, PlanDigest: plan.Digest, CandidateID: candidate.ID, CandidateDigest: candidate.Digest,
		VerificationID: verification.Digest, VerificationDigest: verification.Digest, VerificationStatus: "passed", ApplyID: applyDigest, ApplyDigest: applyDigest, AppliedAt: applied.AppliedAt,
	})
	return err
}

func recordDeliveryResolution(root string, plan PlanRecord, candidate candidateData, verification VerifyReport, applied ApplyReport) error {
	full := verification.ManagerVerification
	if applied.Status != StatusApplied || full == nil || full.Status != "passed" || verification.Status != "verified" || verification.RunID != plan.ID || verification.CandidateID != candidate.ID || verification.CandidateHash != candidate.Digest || full.CandidateID != candidate.ID || full.ModelDigest != plan.ModelDigest {
		return nil
	}
	state, stateDigest, err := projectbriefing.Read(root)
	if err != nil {
		return err
	}
	if state.History == nil {
		return nil
	}
	eventIDs := []string{}
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if projectbriefing.EventResolutionStatus(state, event.ID).Status == "resolved" {
				continue
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", bundle.Revision, plan.BaseRevision); err == nil {
				eventIDs = append(eventIDs, event.ID)
			}
		}
	}
	if len(eventIDs) == 0 {
		return nil
	}
	managers := []string{}
	for _, result := range full.Managers {
		if result.Status != "passed" {
			return fmt.Errorf("incomplete full Manager evidence cannot resolve events")
		}
		managers = append(managers, result.ManagerID)
	}
	applyDigest, err := digest(applied)
	if err != nil {
		return err
	}
	// The resolution counts only where the written result is committed.
	delivered := make([]projectbriefing.DeliveredFile, 0, len(applied.Written))
	for _, path := range applied.Written {
		file, ok := candidate.Files[path]
		if !ok {
			return fmt.Errorf("applied path %s is not in the verified candidate", path)
		}
		item, err := projectbriefing.NewDeliveredFile(root, path, file.Mode, file.Content, file.Delete)
		if err != nil {
			return err
		}
		delivered = append(delivered, item)
	}
	_, err = projectbriefing.ResolveVerified(root, plan.BaseRevision, plan.ModelDigest, projectbriefing.VerifiedResolutionEvidence{
		EventIDs: uniqueSorted(eventIDs), RunID: plan.ID, PlanDigest: plan.Digest, CandidateID: candidate.ID, CandidateDigest: candidate.Digest,
		VerificationDigest: verification.Digest, ApplyDigest: applyDigest, FullVerifyPassed: true, CoveredManagerIDs: uniqueSorted(managers),
		EvidenceRefs: uniqueSorted([]string{"run:" + plan.ID, "full-verify:" + full.Digest, "verify:" + verification.Digest, "apply:" + applyDigest}),
		Delivered:    delivered,
	}, stateDigest)
	return err
}

// RecoverExplorationCompletion repairs only the exploration receipt after an
// already journalled Apply. It never executes work or reapplies source bytes.
func RecoverExplorationCompletion(host Host, root, runID string) error {
	store, err := newRunStore(root)
	if err != nil {
		return err
	}
	plan, err := store.readPlan(runID)
	if err != nil {
		return err
	}
	if plan.ExplorationID == "" {
		return nil
	}
	run, err := store.readLatestState(runID)
	if err != nil {
		return err
	}
	if run.Status != StatusApplied {
		return fmt.Errorf("recovery requires an already applied run")
	}
	if err := validateReportClosure(run); err != nil {
		return err
	}
	dir, err := store.runDir(runID)
	if err != nil {
		return err
	}
	candidate, err := store.readCandidate(dir, run.Candidate.ID)
	if err != nil {
		return err
	}
	verification, err := latestVerification(dir, candidate.ID)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(dir, "apply"))
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var applied ApplyReport
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var value ApplyReport
		if err := readJSON(filepath.Join(dir, "apply", entry.Name()), &value); err != nil {
			return err
		}
		if value.Status == StatusApplied && value.RunID == runID && value.CandidateID == candidate.ID {
			if applied.Status != "" {
				a, _ := json.Marshal(applied)
				b, _ := json.Marshal(value)
				if !bytes.Equal(a, b) {
					return fmt.Errorf("ambiguous successful Apply receipts")
				}
			}
			applied = value
		}
	}
	if applied.Status == "" {
		return fmt.Errorf("successful Apply receipt is missing; source bytes alone cannot establish completion")
	}
	paths := make([]string, 0, len(candidate.Files))
	for path := range candidate.Files {
		paths = append(paths, path)
	}
	if len(paths) > 0 {
		observed, err := source.ObserveSelectedWorking(root, paths)
		if err != nil {
			return err
		}
		for path, file := range candidate.Files {
			data, exists := observed.Snapshot.Files[path]
			if file.Delete && exists || !file.Delete && (!exists || !bytes.Equal(data, file.Content)) {
				return fmt.Errorf("applied candidate path %s changed before recovery", path)
			}
		}
	}
	return completeExploration(root, plan, candidate, verification, applied)
}

func explorationDigest(value string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}
