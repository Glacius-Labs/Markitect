package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func TestPendingNativeVerifierAndSavedChecksRequireOriginalEvidence(t *testing.T) {
	runtime := Runtime{Verifier: &Agent{Transport: TransportCodexAppServer}}
	run := RunReport{Status: StatusVerifying, Invocations: []InvocationLog{{Role: agentexec.RoleVerifier, Phase: "verify", Outcome: "started"}}}
	if !pendingNativeVerifier(run, runtime) {
		t.Fatal("started native verifier was not classified as pending")
	}
	run.Status = StatusFailed
	if !pendingNativeVerifier(run, runtime) {
		t.Fatal("failed run with pending native verifier was not recoverable")
	}
	run.Invocations[0].Outcome = agentexec.OutcomePassed
	if pendingNativeVerifier(run, runtime) {
		t.Fatal("completed verifier was treated as pending")
	}
	runtime.Verifier.Transport = TransportProcess
	if pendingNativeVerifier(run, runtime) {
		t.Fatal("process verifier was treated as recoverable native work")
	}

	plan := PlanRecord{Checks: []CheckPlan{{ID: "unit", Required: true}, {ID: "optional", Required: false}}}
	checks := []CheckResult{{ID: "unit", CandidateID: "candidate-1", Outcome: "passed"}, {ID: "optional", CandidateID: "candidate-1", Outcome: "failed"}}
	saved, err := savedVerificationChecks(RunReport{Checks: checks}, plan, "candidate-1")
	if err != nil || len(saved) != 2 || saved[1].Outcome != "failed" {
		t.Fatalf("original completed checks were not retained: %+v err=%v", saved, err)
	}
	for _, invalid := range [][]CheckResult{nil, {{ID: "unit", CandidateID: "candidate-1", Outcome: "started"}}, {{ID: "unit", CandidateID: "candidate-2", Outcome: "passed"}}} {
		if _, err := savedVerificationChecks(RunReport{Checks: invalid}, plan, "candidate-1"); err == nil {
			t.Fatalf("invalid original check evidence was accepted: %+v", invalid)
		}
	}
}

type verifierRecoveryOnly struct {
	result agentexec.RunResult
	runs   int
	gets   int
}

func (i *verifierRecoveryOnly) Run(context.Context, agentexec.Config, agentexec.Request, agentexec.RunOptions) (agentexec.RunResult, error) {
	i.runs++
	return agentexec.RunResult{}, os.ErrInvalid
}
func (i *verifierRecoveryOnly) Fingerprint(agentexec.Config) (string, error) {
	return "recovery-fingerprint", nil
}
func (i *verifierRecoveryOnly) Recover(_ context.Context, _ agentexec.Config, handle codexappserver.RecoveryHandle, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.gets++
	if i.result.Receipt.RunID != handle.Invocation.RunID {
		return agentexec.RunResult{}, os.ErrInvalid
	}
	return i.result, nil
}

func TestRunVerifierConsumesClosedOriginalCacheWithoutAnotherStart(t *testing.T) {
	root, project, _ := workspaceBridgeBase(t)
	identity, err := source.IdentifyGit(root)
	if err != nil {
		t.Fatal(err)
	}
	base, err := projectworkspace.InspectRepository(context.Background(), root, project.Revision)
	if err != nil {
		t.Fatal(err)
	}
	candidate := candidateData{APIVersion: APIVersion, ID: "candidate-verifier-cache", Files: map[string]File{}, Digest: "candidate-digest"}
	request := verifierRequest(project, candidate, PlanRecord{})
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	storage := t.TempDir()
	service, err := projectworkspace.NewGitService(storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	overlayDigest, err := projectworkspace.CandidateOverlayDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	taskID := "verifier-" + candidate.ID
	workspaceRequest := projectworkspace.Request{RepositoryRoot: root, RepositoryIdentity: identity.Digest, BaseSHA: project.Revision,
		OverlayDigest: base.OverlayDigest, TaskID: taskID}
	handle, err := service.PrepareCandidate(context.Background(), workspaceRequest, nil, overlayDigest)
	if err != nil {
		t.Fatal(err)
	}
	privateDir := filepath.Join(root, ".markitect", "runs", "private")
	journal, err := newNativeJournal(privateDir, handle.CWD, handle.ID)
	if err != nil {
		t.Fatal(err)
	}
	options := journal.wrapOptions(codexappserver.Options{})
	if err := options.BeforeStart(context.Background(), agentexec.RoleStartRequest{RequestID: invocation.RunID, Role: agentexec.RoleVerifier}); err != nil {
		t.Fatal(err)
	}
	recoveryHandle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/0.162.0", Fingerprint: "recovery-fingerprint", Invocation: invocation,
		Workspace: handle, ThreadID: "thread-verifier", SessionID: "session-verifier", TurnID: "turn-verifier", TurnDispatched: true}
	if err := options.OnHandle(context.Background(), recoveryHandle); err != nil {
		t.Fatal(err)
	}
	workspaceState := workspaceJournal{Request: workspaceRequest, Handle: handle, OverlayDigest: overlayDigest, State: "preserved",
		Receipt: agentexec.Receipt{RunID: invocation.RunID}}
	if err := persistWorkspaceJournal(filepath.Join(privateDir, "workspaces", handle.ID+".json"), workspaceState); err != nil {
		t.Fatal(err)
	}
	service, err = projectworkspace.NewGitService(storage, projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	zero := int64(0)
	var evidenceRefs []string
	var observations []agentexec.Observation
	for _, artifact := range project.Report.Artifacts {
		if artifact.Required {
			ref := "artifact:" + artifact.ID
			evidenceRefs = append(evidenceRefs, ref)
			observations = append(observations, agentexec.Observation{Subject: artifact.ID, Outcome: "passed", Detail: "fixture observation"})
		}
	}
	result := agentexec.RunResult{
		Response: agentexec.Response{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce, Role: agentexec.RoleVerifier,
			InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomePassed, EvidenceRefs: evidenceRefs, CandidateFiles: []agentexec.CandidateFile{},
			VerifierObservations: observations, Uncertainty: []string{}},
		Receipt: agentexec.Receipt{APIVersion: agentexec.APIVersion, RunID: invocation.RunID, InputDigest: invocation.InputDigest,
			ConfigDigest: "recovery-fingerprint", Outcome: agentexec.OutcomePassed, Usage: &agentexec.Usage{InputTokens: &zero, OutputTokens: &zero},
			Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, SessionID: "session-verifier", TurnID: "turn-verifier", State: "completed",
				StartRequests: []agentexec.RoleStartRequest{{RequestID: invocation.RunID, Role: agentexec.RoleVerifier, State: "completed"}}}},
	}
	invoker := &verifierRecoveryOnly{result: result}
	verifier := appServerAgent(t)
	runtime := Runtime{Verifier: &verifier, Limits: Limits{MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 4 << 20}}
	starts := 0
	report, log, err := runVerifier(requireNativeRecovery(context.Background()), Host{Load: projectwork.Load, Workspaces: service}, invoker, root,
		PlanRecord{}, runtime, project, candidate, []CheckResult{}, func(InvocationLog) error { starts++; return nil })
	if err != nil || report == nil || report.Outcome != agentexec.OutcomePassed {
		t.Fatalf("closed original verifier result was not consumed: report=%+v err=%v", report, err)
	}
	if invoker.gets != 1 || invoker.runs != 0 || starts != 0 || log.Receipt.RunID != invocation.RunID {
		t.Fatalf("cached recovery launched/reserved another verifier: gets=%d runs=%d starts=%d log=%+v", invoker.gets, invoker.runs, starts, log)
	}
	closed, err := readWorkspaceJournalFile(privateDir, handle.ID)
	if err != nil || closed.State != "closed" || closed.CachedResult == nil {
		t.Fatalf("recovered result was not durably closed and cached: %+v err=%v", closed, err)
	}
}

func verifierRequest(project *Project, candidate candidateData, plan PlanRecord) agentexec.Request {
	checks, artifacts, subjects, refs := []string{}, []string{}, []string{}, []string{}
	for _, check := range plan.Checks {
		if check.Required {
			checks = append(checks, check.ID)
			subjects = append(subjects, check.ID)
			refs = append(refs, "check:"+check.ID)
		}
	}
	for _, artifact := range project.Report.Artifacts {
		if artifact.Required {
			artifacts = append(artifacts, artifact.ID)
			subjects = append(subjects, artifact.ID)
			refs = append(refs, "artifact:"+artifact.ID)
		}
	}
	sort.Strings(subjects)
	sort.Strings(refs)
	contextJSON, _ := json.Marshal(struct {
		CandidateDigest      string        `json:"candidateDigest"`
		CandidateFiles       []string      `json:"candidateFiles"`
		RequiredSubjects     []string      `json:"requiredSubjects"`
		RequiredEvidenceRefs []string      `json:"requiredEvidenceRefs"`
		CheckResults         []CheckResult `json:"checkResults"`
	}{candidate.Digest, sortedFileKeys(candidate.Files), subjects, refs, []CheckResult{}})
	return agentexec.Request{Role: agentexec.RoleVerifier, SourceRevision: project.Revision, ModelDigest: project.Report.ModelDigest,
		ModulePin: project.Report.Digest, ProjectionID: project.Report.Digest, ScopeIDs: refs, PolicyIDs: checks, Context: contextJSON,
		Artifacts: []agentexec.Artifact{}}
}

func readWorkspaceJournalFile(privateDir, id string) (workspaceJournal, error) {
	wire, err := os.ReadFile(filepath.Join(privateDir, "workspaces", id+".json"))
	if err != nil {
		return workspaceJournal{}, err
	}
	var journal workspaceJournal
	err = json.Unmarshal(wire, &journal)
	return journal, err
}
