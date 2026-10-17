package projectrun

import 
import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
)

// ExplorationBinding computes the exact executable selection without creating
// a run or starting an agent. Stable readiness excludes random plan IDs.
func ExplorationBinding(host Host, root, revision string, request PlanRequest) (projectvalue.Binding, error) {
	explorationID, scopeID := request.ExplorationID, request.ScopeID
	if explorationID != "" {
		fixed, err := resolveGitHead(root)
		if err != nil {
			return projectvalue.Binding{}, err
		}
		if revision != "" && revision != fixed {
			return projectvalue.Binding{}, fmt.Errorf("readiness requires the active committed HEAD")
		}
		history, err := captureAcceptedHistory(host, root, fixed, false)
		if err != nil {
			return projectvalue.Binding{}, err
		}
		request.acceptedBriefings = history
	}
	request.ExecuteAuthorized = false
	request.ExplorationID, request.ScopeID = "", ""
	plan, err := Plan(host, root, revision, request)
	if err != nil {
		return projectvalue.Binding{}, err
	}
	project, err := host.Load(root, plan.BaseRevision)
	if err != nil {
		return projectvalue.Binding{}, err
	}
	working, err := host.Load(root, "")
	if err != nil {
		return projectvalue.Binding{}, err
	}
	if working == nil || working.Snapshot == nil || working.Snapshot.Digest() != plan.WorkingSnapshot {
		return projectvalue.Binding{}, ErrStale
	}
	request.ExplorationID, request.ScopeID = explorationID, scopeID
	return explorationBindingFromPlan(host, root, project, working.Snapshot, request, plan)
}

func loadExplorationReadiness(host Host, root, explorationID, scopeID string) (DeliveryScope, projectvalue.Binding, ExplorationReadiness, error) {
	if host.LoadExplorationReadiness == nil {
		return DeliveryScope{}, projectvalue.Binding{}, ExplorationReadiness{}, fmt.Errorf("exploration readiness acquisition is not bound")
	}
	return host.LoadExplorationReadiness(root, explorationID, scopeID)
}

func explorationBindingFromPlan(host Host, root string, project *projectvalue.Project, working *source.Input, request PlanRequest, plan PlanRecord) (projectvalue.Binding, error) {
	var binding projectvalue.Binding
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
		if host.LoadDeliveryScope == nil {
			return binding, fmt.Errorf("delivery scope acquisition is not bound")
		}
		scope, err := host.LoadDeliveryScope(root, request.ExplorationID, request.ScopeID)
		if err != nil {
			return binding, err
		}
		scopeName = scope.Name
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
	binding = projectvalue.Binding{
		RepositoryRoot: abs, Branch: plan.TargetBranch, Head: plan.TargetHead,
		ModelRevision: plan.BaseRevision, ModelAccepted: !project.Provisional && plan.BaseRevision == plan.TargetHead && policy == "committed-model",
		AcceptancePolicy: policy, ProjectDigest: explorationDigest(plan.BaseProjectDigest), ModelDigest: explorationDigest(plan.ModelDigest), SnapshotDigest: explorationDigest(plan.BaseSnapshot),
		SelectionDigest: explorationDigest(selection), RuntimeDigest: explorationDigest(plan.RuntimeDigest), BriefingDigest: explorationDigest(briefings),
		ScopeID: request.ScopeID, ScopeName: scopeName, Goal: plan.Goal, Operation: plan.Operation,
		ManagerIDs: append([]string{}, request.Managers...), RequiredArtifacts: []string{}, FileStructure: []string{}, Checks: []string{}, BasisFiles: []projectvalue.BasisFile{},
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
	paths := append([]string{projectvalue.ManifestPath, projectvalue.RuntimePath}, project.Config.ModelFiles...)
	for _, path := range uniqueSorted(paths) {
		if data, ok := working.Files[path]; ok {
			binding.BasisFiles = append(binding.BasisFiles, projectvalue.BasisFile{Path: path, Digest: explorationBytesDigest(data)})
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
	_, err = projectvalue.BindingDigest(binding)
	return binding, err
}

func bindExplorationReadiness(host Host, root string, project *projectvalue.Project, working *source.Input, request PlanRequest, plan *PlanRecord) error {
	if !request.ExecuteAuthorized && request.ExplorationID == "" {
		return nil
	}
	if request.ExplorationID == "" && request.ScopeID == "" && project.Config.WorkflowMode != "guided" {
		return nil
	}
	if request.ExplorationID == "" || request.ScopeID == "" {
		return fmt.Errorf("guided implementation requires --exploration and --scope with current computed readiness")
	}
	binding, err := explorationBindingFromPlan(host, root, project, working, request, *plan)
	if err != nil {
		return err
	}
	if host.BindExploration == nil {
		return fmt.Errorf("exploration admission is not bound")
	}
	admission, err := host.BindExploration(root, binding, request, *plan)
	if err != nil {
		return err
	}
	plan.ExplorationID, plan.ScopeID = request.ExplorationID, request.ScopeID
	plan.ExplorationDigest, plan.ReadinessDigest, plan.ExplorationBinding = admission.ExplorationDigest, admission.ReadinessDigest, &binding
	return nil
}

func validateExplorationReadiness(host Host, root string, plan PlanRecord) error {
	if plan.ExplorationID == "" && plan.ScopeID == "" && plan.ExplorationBinding == nil {
		return nil
	}
	if plan.ExplorationID == "" || plan.ScopeID == "" || plan.ExplorationBinding == nil {
		return fmt.Errorf("incomplete exploration plan binding")
	}
	if host.ValidateExploration == nil {
		return fmt.Errorf("exploration readiness validation is not bound")
	}
	return host.ValidateExploration(root, plan)
}

func explorationBytesDigest(data []byte) string {
	value := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(value[:])
}

func explorationApplyCapturePaths(host Host, plan PlanRecord, paths []string) ([]string, error) {
	if plan.ExplorationID == "" {
		return paths, nil
	}
	if host.ExplorationCapturePaths == nil {
		return nil, fmt.Errorf("exploration capture paths are not bound")
	}
	extra, err := host.ExplorationCapturePaths(plan)
	if err != nil {
		return nil, err
	}
	return uniqueSorted(append(paths, extra...)), nil
}

func completeExploration(host Host, root string, plan PlanRecord, candidate candidateData, verification VerifyReport, applied ApplyReport) error {
	if err := recordDeliveryResolution(host, root, plan, candidate, verification, applied); err != nil {
		return err
	}
	if plan.ExplorationID == "" {
		return nil
	}
	if plan.ExplorationBinding == nil || applied.Status != StatusApplied || applied.RunID != plan.ID || applied.CandidateID != candidate.ID || verification.Status != "verified" || verification.RunID != plan.ID || verification.CandidateHash != candidate.Digest {
		return fmt.Errorf("exploration completion lacks matching successful Apply and Verify")
	}
	if host.CompleteExploration == nil {
		return fmt.Errorf("exploration completion is not bound")
	}
	return host.CompleteExploration(root, ExplorationCompletion{Plan: plan, CandidateID: candidate.ID, CandidateDigest: candidate.Digest, Verification: verification, Apply: applied})
}

func recordDeliveryResolution(host Host, root string, plan PlanRecord, candidate candidateData, verification VerifyReport, applied ApplyReport) error {
	full := verification.ManagerVerification
	if applied.Status != StatusApplied || full == nil || full.Status != "passed" || verification.Status != "verified" || verification.RunID != plan.ID || verification.CandidateID != candidate.ID || verification.CandidateHash != candidate.Digest || full.CandidateID != candidate.ID || full.ModelDigest != plan.ModelDigest {
		return nil
	}
	if host.ResolveVerifiedBriefing == nil {
		return fmt.Errorf("verified briefing resolution is not bound")
	}
	return host.ResolveVerifiedBriefing(root, DeliveredResolution{Plan: plan, CandidateID: candidate.ID, CandidateDigest: candidate.Digest, Files: candidate.Files, Verification: verification, Apply: applied})
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
	verification, err := latestVerification(dir, candidate.ID, store)
	if err != nil {
		return err
	}
	entries, err := listRecordDirectory(filepath.Join(dir, "apply"), store)
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
		if err := readJSON(filepath.Join(dir, "apply", entry.Name()), &value, store); err != nil {
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
	return completeExploration(host, root, plan, candidate, verification, applied)
}

func explorationDigest(value string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}
