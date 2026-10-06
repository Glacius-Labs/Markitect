package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/host/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// ExecuteCanonicalController makes no Host adopter-artifact or ledger writes.
// Configured runners execute with caller authority; this is not an OS sandbox
// or an audit of unselected repository bytes. Selected inputs are rechecked.
// Its result is an exact, reviewable candidate, never an acceptance result.
func ExecuteCanonicalController(ctx context.Context, root, base, revision, configPath string, cfg CanonicalControllerConfig, toolVersion, toolDigest string) (CanonicalReviewedRun, error) {
	run := CanonicalReviewedRun{APIVersion: CanonicalControllerAPIVersion, ConfigPath: configPath, ToolVersion: toolVersion, ToolDigest: toolDigest, Work: []CanonicalControllerWork{}}
	proposal, err := ProposeCanonicalController(root, base, revision, configPath, cfg)
	run.Proposal = proposal
	run.Status = proposal.Status
	if err != nil {
		return run, err
	}
	if proposal.Status != "planned" {
		return finalizeCanonicalReviewedRun(run)
	}
	run.ExecutorDigest, err = agentexec.Fingerprint(cfg.Executor.agentConfig())
	if err != nil {
		return run, err
	}
	run.VerifierDigest, err = agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if err != nil {
		return run, err
	}
	run.HostExecutableDigest, err = canonicalHostExecutableDigest()
	if err != nil {
		return run, err
	}
	ordered, err := canonicalControllerOrderedProposals(cfg, proposal)
	if err != nil {
		return run, err
	}
	stage := cloneCanonicalCandidateSnapshot(proposal.observed)
	staged := map[string]map[string][]byte{}
	scheduled := map[string]bool{}
	for _, p := range ordered {
		if p.Decision == "work" {
			scheduled[p.ProjectionID] = true
		}
	}
	for _, p := range ordered {
		if p.Decision != "work" {
			continue
		}
		item := CanonicalControllerWork{ProjectionID: p.ProjectionID}
		if p.Task != nil {
			contextModel, err := BuildCanonicalAgentContext(proposal.fixed.Model, p.Request, cfg.ReferenceDepth)
			if err != nil {
				return run, err
			}
			artifacts, dependencyPaths, err := canonicalControllerExecutorArtifacts(cfg, p, proposal.active, stage, staged, scheduled)
			if err != nil {
				item.Escalations = []CanonicalProjectionEscalation{{Code: "dependency.unavailable", Identity: p.ProjectionID, Message: err.Error()}}
				run.Work = append(run.Work, item)
				run.Status = "escalated"
				continue
			}
			contextBytes, err := json.Marshal(struct {
				Model              CanonicalAgentContext `json:"model"`
				Objective          string                `json:"objective"`
				AllowedRoots       []string              `json:"allowedRoots"`
				AllowedExtensions  []string              `json:"allowedExtensions"`
				Constraints        []string              `json:"constraints"`
				DependencyEvidence []string              `json:"readOnlyDependencyEvidence"`
				DependencyState    string                `json:"dependencyState"`
			}{contextModel, p.Task.Objective, p.Task.AllowedRoots, p.Task.AllowedExtensions, p.Task.Constraints, dependencyPaths, "Child candidates are unapplied and unverified. Retained child bytes are observed evidence, not new semantic acceptance. Do not edit dependency artifacts outside your own allowed roots."})
			if err != nil {
				return run, err
			}
			result, invokeErr := agentexec.Run(ctx, cfg.Executor.agentConfig(), agentexec.Request{
				Role: agentexec.RoleExecutor, SourceRevision: revision, ModelDigest: proposal.Plan.ModelDigest, ModulePin: p.Module.Digest, ProjectionID: p.ProjectionID,
				ScopeIDs: contextModel.ScopeIDs, PolicyIDs: canonicalRequestPolicyIDs(p.Request.Policies), Context: contextBytes, Artifacts: artifacts,
			}, agentexec.RunOptions{TempParent: filepath.Dir(cfg.PrivateLogs), PrivateLogDirectory: cfg.PrivateLogs})
			item.Executor = &result.Receipt
			if invokeErr != nil {
				item.Escalations = []CanonicalProjectionEscalation{{Code: "executor.invocation-failed", Identity: p.ProjectionID, Message: invokeErr.Error()}}
				run.Work = append(run.Work, item)
				run.Status = "incomplete"
				run, _ = finalizeCanonicalReviewedRun(run)
				return run, invokeErr
			}
			if result.Receipt.ConfigDigest != run.ExecutorDigest {
				return run, errors.New("Executor runtime fingerprint changed during preparation")
			}
			if result.Response.Outcome != agentexec.OutcomeProposed {
				item.Escalations = []CanonicalProjectionEscalation{{Code: "executor." + result.Response.Outcome, Identity: p.ProjectionID, Message: "Executor returned no applicable candidate; uncertainty is retained in private run evidence"}}
				run.Work = append(run.Work, item)
				run.Status = "escalated"
				continue
			}
			candidate := CanonicalCandidate{RequestDigest: p.Request.RequestDigest, Files: []CanonicalCandidateFile{}}
			for _, file := range result.Response.CandidateFiles {
				mode, err := canonicalCandidateMode(file.Mode)
				if err != nil {
					return run, fmt.Errorf("Executor candidate %s: %w", file.Path, err)
				}
				allowed := false
				for _, ext := range p.Task.AllowedExtensions {
					if filepath.Ext(file.Path) == ext {
						allowed = true
					}
				}
				if !allowed {
					return run, fmt.Errorf("Executor candidate extension outside Module task: %s", file.Path)
				}
				candidate.Files = append(candidate.Files, CanonicalCandidateFile{Path: file.Path, Content: file.Content, Mode: mode})
			}
			sort.Slice(candidate.Files, func(i, j int) bool { return candidate.Files[i].Path < candidate.Files[j].Path })
			item.Candidate, err = json.Marshal(candidate)
			if err != nil {
				return run, err
			}
		}
		prepared, err := PrepareCanonicalProjection(proposal.fixed, stage, p.Request.Projection.Identity(), toolVersion, toolDigest, item.Candidate, proposal.fixed.Config.Checks...)
		if err != nil {
			return run, err
		}
		if err := ValidateCanonicalControllerExclusionOutputs(prepared.Outputs, canonicalControllerExcludedPaths(cfg)); err != nil {
			return run, err
		}
		item.Outputs = prepared.Outputs
		item.CandidateDigest = prepared.CandidateDigest
		item.Escalations = prepared.Escalations
		if prepared.Plan != nil {
			item.PlanDigest = prepared.Plan.PlanDigest
		}
		// A task must return the complete owned representation. Omission does not
		// transfer or retire a file implicitly.
		for _, prior := range proposal.active {
			if prior.ProjectionID == p.ProjectionID {
				for _, artifact := range prior.Artifacts {
					if _, kept := item.Outputs[artifact.Path]; !kept {
						item.Escalations = append(item.Escalations, CanonicalProjectionEscalation{Code: "artifact.omitted-owner", Identity: p.ProjectionID, Message: "candidate omits active owned artifact: " + artifact.Path})
					}
				}
			}
		}
		if item.PlanDigest == "" || len(item.Escalations) > 0 {
			run.Status = "escalated"
		}
		run.Work = append(run.Work, item)
		if len(item.Escalations) == 0 && item.PlanDigest != "" {
			staged[item.ProjectionID] = item.Outputs
			stageCanonicalCandidate(stage, item.Outputs)
			stageCanonicalCandidateModes(stage, prepared.OutputModes)
		}
	}
	// Execution does not grant stale input a fresh approval.
	fresh, err := ProposeCanonicalController(root, base, revision, configPath, cfg)
	if err != nil {
		return run, err
	}
	if fresh.Digest != proposal.Digest {
		return run, errors.New("selected inputs or ownership changed during Executor invocation")
	}
	if err := validateCanonicalControllerRuntime(run, cfg); err != nil {
		return run, err
	}
	return finalizeCanonicalReviewedRun(run)
}
func finalizeCanonicalReviewedRun(run CanonicalReviewedRun) (CanonicalReviewedRun, error) {
	run.Digest = ""
	digest, err := digestCanonicalValue(run)
	run.Digest = digest
	return run, err
}

func stageCanonicalCandidateModes(input *snapshot.Snapshot, modes map[string]string) {
	if input == nil {
		return
	}
	for name, mode := range modes {
		input.Modes[name] = mode
	}
}

func canonicalCandidateMode(mode string) (string, error) {
	switch mode {
	case "0644":
		return snapshot.RegularMode, nil
	case "0755":
		return snapshot.ExecutableMode, nil
	default:
		return "", fmt.Errorf("unsupported file mode %q", mode)
	}
}

func canonicalHostExecutableDigest() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	data, err := readBoundedRegular(executable, 256<<20, false)
	if err != nil {
		return "", err
	}
	return sha256Prefix(sha256Hex(data)), nil
}
func validateCanonicalControllerRuntime(run CanonicalReviewedRun, cfg CanonicalControllerConfig) error {
	config, err := digestCanonicalValue(cfg)
	if err != nil {
		return err
	}
	if config != run.Proposal.ConfigDigest {
		return errors.New("controller runtime configuration changed since review")
	}
	executor, err := agentexec.Fingerprint(cfg.Executor.agentConfig())
	if err != nil {
		return err
	}
	verifier, err := agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if err != nil {
		return err
	}
	host, err := canonicalHostExecutableDigest()
	if err != nil {
		return err
	}
	if executor != run.ExecutorDigest || verifier != run.VerifierDigest || host != run.HostExecutableDigest {
		return errors.New("controller/agent executable or runtime-file bytes changed since review")
	}
	return nil
}

// Controller operations share this external lease. Crash remnants block;
// there is no automatic recovery or deletion of someone else's lease.
func acquireCanonicalControllerLease(cfg CanonicalControllerConfig) (func(), error) {
	path := cfg.RecordStore + ".controller.lock"
	if err := rejectReparseAncestors(filepath.Dir(path)); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("controller lease unavailable; inspect %s: %w", path, err)
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil
}

func ApplyCanonicalController(root, configPath string, cfg CanonicalControllerConfig, run CanonicalReviewedRun, expect string, write bool) (CanonicalControllerApply, error) {
	report := CanonicalControllerApply{Status: "refused", RunDigest: run.Digest, Written: []string{}, Records: []records.ProjectionRecord{}, EvidenceRefreshRequired: run.Proposal.Plan.EvidenceRefreshRequired}
	if !write || expect == "" || expect != run.Digest {
		return report, errors.New("controller Apply requires explicit write and exact reviewed run digest")
	}
	encoded, err := json.Marshal(run)
	if err != nil {
		return report, err
	}
	if _, err = DecodeCanonicalReviewedRun(encoded); err != nil {
		return report, err
	}
	if run.Status != "planned" || configPath != run.ConfigPath {
		return report, errors.New("controller run is escalated/incomplete or selects a different configuration")
	}
	if err := validateCanonicalControllerRuntime(run, cfg); err != nil {
		return report, err
	}
	if _, _, _, err = readCanonicalControllerLedger(root, cfg); err != nil {
		return report, err
	}
	unlock, err := acquireCanonicalControllerLease(cfg)
	if err != nil {
		return report, err
	}
	defer unlock()
	fresh, err := ProposeCanonicalController(root, run.Proposal.Plan.BaseRevision, run.Proposal.Plan.Revision, configPath, cfg)
	if err != nil {
		return report, err
	}
	if fresh.Digest != run.Proposal.Digest {
		return report, errors.New("controller candidate is stale for selected bytes, roots, source or ledger selection")
	}
	preparedByID := map[string]PreparedCanonicalProjection{}
	output := map[string][]byte{}
	outputModes := map[string]string{}
	proposals := map[string]CanonicalScopedProposal{}
	for _, p := range fresh.Plan.Proposals {
		if p.Decision == "work" {
			proposals[p.ProjectionID] = p
		}
	}
	if len(proposals) != len(run.Work) {
		return report, errors.New("reviewed work does not exactly cover current Module work")
	}
	ordered, err := canonicalControllerOrderedProposals(cfg, fresh)
	if err != nil {
		return report, err
	}
	expectedOrder := []string{}
	for _, p := range ordered {
		if p.Decision == "work" {
			expectedOrder = append(expectedOrder, p.ProjectionID)
		}
	}
	stage := cloneCanonicalCandidateSnapshot(fresh.observed)
	for index, work := range run.Work {
		if index >= len(expectedOrder) || work.ProjectionID != expectedOrder[index] {
			return report, errors.New("reviewed work differs from deterministic child-first execution order")
		}
		p, ok := proposals[work.ProjectionID]
		if !ok {
			return report, errors.New("reviewed work has an extra or duplicate Projection")
		}
		delete(proposals, work.ProjectionID)
		prepared, err := PrepareCanonicalProjection(fresh.fixed, stage, p.Request.Projection.Identity(), run.ToolVersion, run.ToolDigest, work.Candidate, fresh.fixed.Config.Checks...)
		if err != nil {
			return report, err
		}
		if prepared.Plan == nil || len(prepared.Escalations) != 0 || len(work.Escalations) != 0 || prepared.Plan.PlanDigest != work.PlanDigest || prepared.CandidateDigest != work.CandidateDigest || outputDigest(prepared.Outputs, prepared.OutputModes) != outputDigest(work.Outputs, prepared.OutputModes) {
			return report, errors.New("reviewed candidate or exact plan bytes differ from fresh preparation")
		}
		if err := ValidateCanonicalControllerExclusionOutputs(prepared.Outputs, canonicalControllerExcludedPaths(cfg)); err != nil {
			return report, err
		}
		if p.Task != nil && (work.Executor == nil || work.Executor.ConfigDigest != run.ExecutorDigest || work.Executor.Outcome != agentexec.OutcomeProposed) {
			return report, errors.New("AI candidate lacks its exact Executor receipt")
		}
		for name, data := range prepared.Outputs {
			if _, collision := output[name]; collision {
				return report, fmt.Errorf("multiple candidates target %s", name)
			}
			output[name] = data
			outputModes[name] = artifactMode(prepared.OutputModes, name)
		}
		for _, prior := range fresh.active {
			if prior.ProjectionID == work.ProjectionID {
				for _, a := range prior.Artifacts {
					if _, kept := prepared.Outputs[a.Path]; !kept {
						return report, fmt.Errorf("candidate omits active owned artifact %s", a.Path)
					}
				}
			}
		}
		preparedByID[work.ProjectionID] = prepared
		stageCanonicalCandidate(stage, prepared.Outputs)
		stageCanonicalCandidateModes(stage, prepared.OutputModes)
	}
	if len(output) == 0 {
		report.Status = "no-materialization-work"
		report.LedgerHead = fresh.LedgerHead
		return report, nil
	}
	// Known branch/platform refusals must not create an empty ledger and stale
	// an otherwise reviewed run. The scoped writer repeats these checks at the
	// mutation boundary; failures after initialization remain explicit partials.
	if _, err := writeBranchName(root); err != nil {
		return report, err
	}
	if err := validateProjectionWriteModes(output, outputModes); err != nil {
		return report, err
	}
	// Exclusive ledger initialization is validated before adopter mutations.
	store, state, _, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		return report, err
	}
	if state.Head != fresh.LedgerHead {
		return report, errors.New("ledger changed before Apply")
	}
	if store == nil {
		store, err = recordstore.Initialize(cfg.RecordStore, canonicalControllerForbiddenRoots(fresh.Plan.SourceScope.Repository))
		if err != nil {
			return report, err
		}
		state, err = store.Read()
		if err != nil {
			return report, err
		}
	}
	expectedLedgerHead := state.Head
	report.LedgerHead = state.Head
	// Check inputs are protected equally with canonical authored bytes.
	protected := sortedUniquePaths(append(append(canonicalSourcePaths(fresh.fixed), canonicalControllerCheckInputs(cfg)...), canonicalControllerExcludedPaths(cfg)...))
	written, writeErr := writeCanonicalScopedOutputs(root, canonicalScopedWriteCapture{Revision: fresh.Plan.Revision, CanonicalPaths: protected, Observed: fresh.observed, Inventory: fresh.Plan.Inventory}, output, outputModes)
	report.Written = written
	report.Status = records.StateMaterializedUnverified
	if writeErr != nil {
		report.Status = records.StatePartialFailure
	}
	// Preserve the exact applied/retained artifact set even when the writer skips
	// identical bytes. Partial failures never activate a new ownership set.
	writtenSet := map[string]bool{}
	for _, name := range written {
		writtenSet[name] = true
	}
	for _, work := range run.Work {
		names := []string{}
		for name, data := range work.Outputs {
			if writeErr == nil || writtenSet[name] || (bytes.Equal(fresh.observed.Files[name], data) && fresh.observed.Modes[name] == artifactMode(preparedByID[work.ProjectionID].OutputModes, name)) {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		if len(names) == 0 {
			continue
		} // No artifact means no honest materialization record.
		actual, observeErr := source.ObserveSelectedWorking(root, names)
		if observeErr != nil {
			return report, fmt.Errorf("artifacts remain as reported; selected post-write artifact observation failed: %w", observeErr)
		}
		if !equalCanonicalValue(actual.Identity, fresh.Plan.SourceScope.Repository) {
			return report, errors.New("artifacts remain as reported; repository identity changed before selected artifact observation")
		}
		prepared := preparedByID[work.ProjectionID]
		for _, name := range names {
			if actual.Snapshot.Modes[name] != artifactMode(prepared.OutputModes, name) {
				if writeErr == nil {
					writeErr = fmt.Errorf("post-write artifact mode differs from reviewed mode for %s", name)
				}
				report.Status = records.StatePartialFailure
			}
		}
		record, err := buildCanonicalProjectionRecord(prepared, fresh.observed, actual.Snapshot, names, report.Status)
		if err != nil {
			return report, fmt.Errorf("artifacts remain as reported; record construction failed: %w", err)
		}
		for _, prior := range fresh.active {
			if prior.ProjectionID == record.ProjectionID {
				record.PriorRecordID = prior.ID
			}
		}
		record, err = records.NewProjectionRecord(record)
		if err != nil {
			return report, err
		}
		report.Records = append(report.Records, record)
	}
	state, err = store.Read()
	if err != nil {
		return report, fmt.Errorf("artifacts remain as reported; ledger unavailable: %w", err)
	}
	if state.Head != expectedLedgerHead {
		return report, errors.New("artifacts remain as reported; ledger changed during Apply")
	}
	for _, record := range report.Records {
		state, err = store.AppendAttempt(state.Head, record)
		if err != nil {
			return report, fmt.Errorf("artifacts remain as reported; append failed: %w", err)
		}
		report.LedgerHead = state.Head
	}
	if writeErr != nil {
		return report, writeErr
	}
	replacements := map[string]string{}
	for _, r := range fresh.active {
		replacements[r.ProjectionID] = r.ID
	}
	for _, r := range report.Records {
		replacements[r.ProjectionID] = r.ID
	}
	ids := []string{}
	for _, id := range replacements {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state, err = store.SelectActive(state.Head, ids)
	if err != nil {
		return report, err
	}
	report.LedgerHead = state.Head
	// Evidence refresh is separate from materialization. The immutable evidence
	// commit contains only canonical/check inputs and explicitly active artifacts.
	report.EvidencePaths = sortedUniquePaths(append(canonicalSourcePaths(fresh.fixed), canonicalControllerCheckInputs(cfg)...))
	byRecordID := map[string]records.ProjectionRecord{}
	for _, r := range state.Records {
		byRecordID[r.ID] = r
	}
	for _, id := range state.ActiveSelection.RecordIDs {
		for _, a := range byRecordID[id].Artifacts {
			report.EvidencePaths = append(report.EvidencePaths, a.Path)
		}
	}
	report.EvidencePaths = sortedUniquePaths(report.EvidencePaths)
	evidence, err := source.ObserveSelectedWorking(root, report.EvidencePaths)
	if err != nil {
		return report, fmt.Errorf("artifacts and ledger remain; evidence observation failed: %w", err)
	}
	if !equalCanonicalValue(evidence.Identity, fresh.Plan.SourceScope.Repository) {
		return report, errors.New("artifacts and ledger remain; repository identity changed before evidence capture")
	}
	for _, name := range protected {
		if !bytes.Equal(evidence.Snapshot.Files[name], fresh.fixed.Snapshot.Files[name]) || evidence.Snapshot.Modes[name] != fresh.fixed.Snapshot.Modes[name] {
			return report, fmt.Errorf("artifacts and ledger remain; protected evidence changed: %s", name)
		}
	}
	for _, id := range state.ActiveSelection.RecordIDs {
		for _, a := range byRecordID[id].Artifacts {
			if sha256Prefix(sha256Hex(evidence.Snapshot.Files[a.Path])) != a.Digest || evidence.Snapshot.Modes[a.Path] != a.Mode {
				return report, fmt.Errorf("artifacts and ledger remain; active artifact evidence drifted: %s", a.Path)
			}
		}
	}
	report.EvidenceRevision, err = source.WriteSelectedEvidenceCommit(root, fresh.Plan.Revision, evidence.Snapshot.Files, evidence.Snapshot.Modes, true)
	if err != nil {
		return report, fmt.Errorf("artifacts and ledger remain; immutable evidence commit failed: %w", err)
	}
	return report, nil
}

func canonicalRequestPolicyIDs(policies []core.Definition) []string {
	ids := []string{}
	for _, p := range policies {
		ids = append(ids, p.Identity().Key())
	}
	sort.Strings(ids)
	return ids
}

func canonicalControllerExcludedPaths(cfg CanonicalControllerConfig) []string {
	paths := make([]string, 0, len(cfg.TargetExclusions))
	for _, exclusion := range cfg.TargetExclusions {
		paths = append(paths, exclusion.Path)
	}
	return sortedUniquePaths(paths)
}
