package projectcli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestBrownfieldStartPreviewAndCASWrite(t *testing.T) {
	repo := copyProjectWorld(t)
	selectedPath := filepath.Join(repo, "docs", "cancellation.md")
	selectedBytes, err := os.ReadFile(selectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selectedPath, append(selectedBytes, []byte("\nPRIVATE_SOURCE_SENTINEL: selected source body must stay in the session ledger.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "docs/cancellation.md")
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "add private evidence fixture")
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	request := projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion,
		ID:         "brownfield-loop",
		Purpose:    "Create a bounded reverse-modeling session",
		Review:     "owner-review-brownfield-loop",
		Commit:     commit,
		ScopeRoots: []string{"docs"},
		Selected: []projectadoption.SelectedPath{{
			ID: "cancellation-document", Path: "docs/cancellation.md", Reason: "Owner-selected current behavior", Basis: "documentation",
		}},
		Exclusions: []projectadoption.PathReason{},
		Unselected: []projectadoption.PathReason{},
	}
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(brownfieldStartInput{Discovery: discovery, ScopeStatuses: []projectadoption.ScopeStatus{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/brownfield-start.json", input); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(repo, ".markitect", "drafts", "brownfield", discovery.ID, "session.json")
	args := []string{"project", "brownfield", "--repo", repo, "--source-repo", repo, "--brownfield-action", "start", "--revision", commit, "--input", ".markitect/drafts/brownfield-start.json"}
	var previewOut, previewErr bytes.Buffer
	if code := Run(args, &previewOut, &previewErr); code != 0 {
		t.Fatalf("Brownfield start preview exit=%d stderr=%s", code, previewErr.String())
	}
	var preview brownfieldResult
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil {
		t.Fatalf("decode Brownfield preview: %v\n%s", err, previewOut.String())
	}
	if preview.Status != "preview" || preview.Action != "start" || preview.Session == nil || preview.Session.Digest == "" || preview.SessionDigest != preview.Session.Digest || preview.Session.Source.EvidenceCount != 1 || preview.Session.Source.Digest != discovery.Digest || preview.Session.Target.Revision != commit {
		t.Fatalf("unexpected Brownfield start preview: %+v", preview)
	}
	if strings.Contains(previewOut.String(), "PRIVATE_SOURCE_SENTINEL") || strings.Contains(previewOut.String(), "cancellation-document") {
		t.Fatalf("Brownfield start preview exposed raw source evidence: %s", previewOut.String())
	}
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("Brownfield preview wrote the session ledger: %v", err)
	}

	writeArgs := append(append([]string(nil), args...), "--expect", preview.SessionDigest, "--write")
	var writeOut, writeErr bytes.Buffer
	if code := Run(writeArgs, &writeOut, &writeErr); code != 0 {
		t.Fatalf("Brownfield start write exit=%d stderr=%s", code, writeErr.String())
	}
	var written brownfieldResult
	if err := json.Unmarshal(writeOut.Bytes(), &written); err != nil {
		t.Fatalf("decode Brownfield write result: %v\n%s", err, writeOut.String())
	}
	if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest {
		t.Fatalf("Brownfield start write differs from reviewed preview: %+v", written)
	}
	if strings.Contains(writeOut.String(), "PRIVATE_SOURCE_SENTINEL") || strings.Contains(writeOut.String(), "cancellation-document") {
		t.Fatalf("Brownfield start write exposed raw source evidence: %s", writeOut.String())
	}
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("Brownfield session was not durably written: %v", err)
	}

	var resumeOut, resumeErr bytes.Buffer
	resumeArgs := []string{"project", "brownfield", "--repo", repo, "--source-repo", repo, "--brownfield-action", "resume", "--session", discovery.ID}
	if code := Run(resumeArgs, &resumeOut, &resumeErr); code != 0 {
		t.Fatalf("Brownfield resume exit=%d stderr=%s", code, resumeErr.String())
	}
	var resumed brownfieldResult
	if err := json.Unmarshal(resumeOut.Bytes(), &resumed); err != nil {
		t.Fatalf("decode Brownfield resume: %v\n%s", err, resumeOut.String())
	}
	if resumed.Status != "resumed" || resumed.SessionDigest != preview.SessionDigest || resumed.Readiness == nil || !resumed.Readiness.SourceCurrent || !resumed.Readiness.TargetCurrent {
		t.Fatalf("resume did not validate both fixed bases: %+v", resumed)
	}
	if resumed.Session == nil || resumed.Session.Source.Digest != discovery.Digest || resumed.Session.Target.ProjectDigest == "" || strings.Contains(resumeOut.String(), "PRIVATE_SOURCE_SENTINEL") {
		t.Fatalf("resume did not return safe fixed-basis metadata or leaked source evidence: %s", resumeOut.String())
	}
}

func TestBrownfieldInputRejectsUnknownAndTrailingJSON(t *testing.T) {
	var request brownfieldPlanInput
	for _, input := range []string{`{"iterationId":"iteration-a","unexpected":true}`, `{"iterationId":"iteration-a"}{}`} {
		if err := decodeClosedProjectJSON([]byte(input), &request); err == nil {
			t.Fatalf("accepted non-closed Brownfield input: %s", input)
		}
	}
	if err := decodeClosedProjectJSON([]byte(strings.Repeat("x", (32<<20)+1)), &request); err == nil {
		t.Fatal("accepted oversized Brownfield input")
	}
}

func TestBrownfieldManagerContextIsReadOnlyAndBoundToIteration(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	discovery, err := projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "brownfield-context", Purpose: "Prepare manager-scoped reverse context",
		Review: "owner-review-context", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation-document", Path: "docs/cancellation.md", Reason: "Selected behavior evidence", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(repo, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	iterationID := "root-context"
	session, err = projectadoption.BeginReverseIteration(repo, target, session, projectadoption.ReverseIterationRequest{
		ID: iterationID, ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"cancellation-document"}, DelegationEvidenceIDs: []string{},
		Purpose: "Review selected cancellation behavior", Review: "manager-review-context",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(repo, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(brownfieldContextInput{IterationID: iterationID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/brownfield-context.json", input); err != nil {
		t.Fatal(err)
	}
	args := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "context", "--session", discovery.ID, "--input", ".markitect/drafts/brownfield-context.json"}
	var out, errOut bytes.Buffer
	if code := Run(args, &out, &errOut); code != 0 {
		t.Fatalf("Brownfield context exit=%d stderr=%s", code, errOut.String())
	}
	var result brownfieldResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("decode Brownfield context: %v\n%s", err, out.String())
	}
	if result.Action != "context" || result.ManagerContext == nil || result.ManagerContext.IterationID != iterationID || len(result.ManagerContext.Evidence) != 1 || result.ManagerContext.Evidence[0].EvidenceID != "cancellation-document" || result.ManagerContext.Evidence[0].Classification != "documented-intent" {
		t.Fatalf("context did not return the exact assigned evidence: %+v", result)
	}
	ledgerPath := filepath.Join(repo, ".markitect", "drafts", "brownfield", discovery.ID, "session.json")
	loaded, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Digest != session.Digest {
		t.Fatalf("read-only context changed session digest: got %s, want %s (ledger %s)", loaded.Digest, session.Digest, ledgerPath)
	}
}

func TestBrownfieldStagedManagerLoopBeginContextProposeAndIntegrate(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	discovery, err := projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "brownfield-staged-loop", Purpose: "Exercise staged parent and child modeling",
		Review: "owner-review-staged-loop", Commit: commit, ScopeRoots: []string{"."},
		Selected: []projectadoption.SelectedPath{
			{ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Selected intent for root Manager", Basis: "documentation"},
			{ID: "orders-code", Path: "src/shop/orders/order.py", Reason: "Selected implementation for child Manager", Basis: "code"},
		},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(repo, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(repo, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	rootID := session.TargetContext.RootManagerID
	schemaDigest, _, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}

	rootRequest := projectadoption.ReverseIterationRequest{ID: "root-pass", ManagerID: rootID, EvidenceIDs: []string{"cancellation-doc"}, DelegationEvidenceIDs: []string{"orders-code"}, Purpose: "Model cancellation ownership", Review: "root-pass-review"}
	result := runBrownfieldMutation(t, repo, discovery.ID, "begin", rootRequest, session.Digest)
	rootContext := runBrownfieldContext(t, repo, discovery.ID, "root-pass", "")
	if rootContext.ManagerContext.ManagerOrigin != "accepted-target" || len(rootContext.ManagerContext.Evidence) != 1 || rootContext.ManagerContext.Evidence[0].EvidenceID != "cancellation-doc" {
		t.Fatalf("root context was not assignment-bounded: %+v", rootContext.ManagerContext)
	}
	rootReport := makeStagedDistillation(discovery, target, result.Session.TargetContextDigest, schemaDigest, "root-scope", "root-claim", "cancellation-doc", "documented-intent", "documentation", "PRIVATE_MANAGER_REPORT_SENTINEL root cancellation understanding.")
	childID := "cancellation-owner"
	rootProposal := projectadoption.ManagerProposal{ManagerID: rootID, EvidenceIDs: []string{"cancellation-doc"},
		Hierarchy:       []projectadoption.ProposedManager{{ID: childID, Name: "Cancellation Owner", Purpose: "Own cancellation implementation", ParentID: rootID, EvidenceIDs: []string{"orders-code"}}},
		PublicContracts: []projectadoption.ManagerPublicContract{}, Report: rootReport}
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", brownfieldProposalInput{IterationID: "root-pass", Proposal: rootProposal}, result.SessionDigest)

	childRequest := projectadoption.ReverseIterationRequest{ID: "child-pass", ParentIterationID: "root-pass", ManagerID: childID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}, Purpose: "Model cancellation implementation", Review: "child-pass-review"}
	result = runBrownfieldMutation(t, repo, discovery.ID, "begin", childRequest, result.SessionDigest)
	var unassignedSource string
	for _, item := range discovery.Evidence {
		if item.ID == "cancellation-doc" {
			unassignedSource = item.Content
		}
	}
	childContext := runBrownfieldContext(t, repo, discovery.ID, "child-pass", "propose", unassignedSource, "Root cancellation understanding.")
	if childContext.ManagerContext.ManagerOrigin != "proposed-by-parent" || childContext.ManagerContext.Manager.ID != childID || len(childContext.ManagerContext.Evidence) != 1 || childContext.ManagerContext.Evidence[0].EvidenceID != "orders-code" {
		t.Fatalf("child context was not derived from the parent assignment: %+v", childContext.ManagerContext)
	}
	childReport := makeStagedDistillation(discovery, target, result.Session.TargetContextDigest, schemaDigest, "child-scope", "child-claim", "orders-code", "observation", "static-source", "PRIVATE_CHILD_REPORT_SENTINEL implementation owns cancellation handling.")
	contract := projectadoption.ManagerPublicContract{Contract: projectadoption.DistillationTargetContract{ID: "cancellation-api", Name: "Cancellation API", Namespace: "shop.cancellation", Owner: childID, Category: "capability", Description: "Public cancellation operation.", Uses: []string{}, Requires: []string{}}, ClaimIDs: []string{"child-claim"}}
	childProposal := projectadoption.ManagerProposal{ManagerID: childID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []projectadoption.ProposedManager{}, PublicContracts: []projectadoption.ManagerPublicContract{contract}, Report: childReport}
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", brownfieldProposalInput{IterationID: "child-pass", Proposal: childProposal}, result.SessionDigest)
	fullSession, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	childDigest := fullSession.Iterations[1].Proposal.Digest
	integrationContext := runBrownfieldContext(t, repo, discovery.ID, "root-pass", "integrate")
	if integrationContext.IntegrationContext == nil || integrationContext.IntegrationContext.ParentProposal.ManagerID != rootID || len(integrationContext.IntegrationContext.Children) != 1 || integrationContext.IntegrationContext.Children[0].ProposalDigest != childDigest || integrationContext.IntegrationContext.Children[0].PublicContracts[0].Contract.ID != "cancellation-api" {
		t.Fatalf("integration context omitted the assigned child proposal/report/contracts: %+v", integrationContext.IntegrationContext)
	}

	integrated := makeStagedIntegratedDistillation(discovery, target, result.Session.TargetContextDigest, schemaDigest)
	integration := projectadoption.ManagerIntegration{ManagerID: rootID, ChildProposalDigests: []string{childDigest},
		ChildContracts: []projectadoption.IntegratedChildContracts{{ManagerID: childID, ProposalDigest: childDigest, Contracts: []projectadoption.ManagerPublicContract{contract}}},
		Report:         integrated, Conflicts: []projectadoption.SessionConflict{}}
	result = runBrownfieldMutation(t, repo, discovery.ID, "integrate", brownfieldIntegrationInput{IterationID: "root-pass", Integration: integration}, result.SessionDigest)
	fullSession, err = projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fullSession.Iterations[0].Integration == nil || fullSession.Iterations[0].Integration.ChildProposalDigests[0] != childDigest || fullSession.Iterations[0].Integration.ChildContracts[0].Contracts[0].Contract.ID != "cancellation-api" {
		t.Fatalf("root integration did not retain immutable child contract evidence: %+v", fullSession.Iterations[0].Integration)
	}
}

func TestBrownfieldManagerRunPreviewAndStaleGuardDoNotInvokeProvider(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	discovery, err := projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "manager-run-preview", Purpose: "Exercise the provider-free manager-run boundary",
		Review: "owner-review-manager-run", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Selected behavior evidence", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(repo, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	managerID := session.TargetContext.RootManagerID
	session, err = projectadoption.BeginReverseIteration(repo, target, session, projectadoption.ReverseIterationRequest{
		ID: "manager-run-root", ManagerID: managerID, EvidenceIDs: []string{"cancellation-doc"}, DelegationEvidenceIDs: []string{},
		Purpose: "Model selected cancellation behavior", Review: "manager-run-review",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(repo, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	nativeManager := projectrun.Agent{
		Command: executable, Args: []string{"NATIVE_MANAGER_ARGUMENT_SENTINEL"}, Model: "native-manager-model", ModelOptions: map[string]any{"secret": "NATIVE_MANAGER_OPTION_SENTINEL"},
		ProviderVersion: "native-manager-provider-v1", WorkspaceMode: "scoped", InstructionPaths: []string{"AGENTS.md"},
		Timeout: projectrun.Duration(30 * time.Second), MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
		RuntimeFiles: []agentexec.RuntimeFile{}, Environment: []string{}, Pricing: projectrun.Pricing{InputMicrosPerMillion: 10, OutputMicrosPerMillion: 20},
	}
	readOnlyAgent := projectrun.Agent{
		Command: executable, Args: []string{"READONLY_ARGUMENT_SENTINEL"}, Model: "readonly-review-model", ModelOptions: map[string]any{"review": "READONLY_OPTION_SENTINEL"},
		ProviderVersion: "readonly-review-provider-v1", Timeout: projectrun.Duration(30 * time.Second), MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
		RuntimeFiles: []agentexec.RuntimeFile{}, Environment: []string{}, Pricing: projectrun.Pricing{InputMicrosPerMillion: 40, OutputMicrosPerMillion: 80},
	}
	runtime := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: map[string]projectrun.Agent{managerID: nativeManager},
		Review: &projectrun.ReviewConfig{Agents: map[string]projectrun.Agent{managerID: readOnlyAgent}, MaxRounds: 1, MaxManagerRounds: 1},
		Limits: projectrun.Limits{MaxDepth: 4, MaxStarts: 8, MaxRetries: 1, MaxParallel: 1, MaxDuration: projectrun.Duration(5 * time.Minute),
			MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 8 << 20},
	}
	runtimeBytes, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(projectrun.RuntimePath)), runtimeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", projectrun.RuntimePath)
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "add manager runtime fixture")
	commit = gitOutput(t, repo, "rev-parse", "HEAD")
	discovery, err = projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "manager-run-preview-final", Purpose: "Exercise the provider-free manager-run boundary",
		Review: "owner-review-manager-run-final", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Selected behavior evidence", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err = projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	session, err = projectadoption.StartBrownfieldSession(repo, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	session, err = projectadoption.BeginReverseIteration(repo, target, session, projectadoption.ReverseIterationRequest{
		ID: "manager-run-root", ManagerID: managerID, EvidenceIDs: []string{"cancellation-doc"}, DelegationEvidenceIDs: []string{},
		Purpose: "Model selected cancellation behavior", Review: "manager-run-review-final",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(repo, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	inputBytes, err := json.Marshal(brownfieldManagerRunInput{IterationID: "manager-run-root", Phase: "propose", AgentManagerID: managerID})
	if err != nil {
		t.Fatal(err)
	}
	inputPath := ".markitect/drafts/manager-run-preview.json"
	if _, err := writeRecord(repo, inputPath, inputBytes); err != nil {
		t.Fatal(err)
	}
	args := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "run", "--session", discovery.ID, "--input", inputPath}
	var out, errOut bytes.Buffer
	if code := Run(args, &out, &errOut); code != 0 {
		t.Fatalf("Brownfield manager-run preview exit=%d stderr=%s", code, errOut.String())
	}
	var preview brownfieldManagerRunOutput
	if err := json.Unmarshal(out.Bytes(), &preview); err != nil {
		t.Fatalf("decode manager-run preview: %v\n%s", err, out.String())
	}
	if preview.Status != "preview" || preview.SessionDigest != session.Digest || preview.Preview == nil || preview.PreviewDigest == "" || preview.Preview.PreviewDigest != preview.PreviewDigest || preview.Attempt != nil {
		t.Fatalf("unexpected Brownfield manager-run preview: %+v", preview)
	}
	if preview.Preview.Model != readOnlyAgent.Model || preview.Preview.ProviderVersion != readOnlyAgent.ProviderVersion ||
		preview.Preview.InputPriceMicrosPerMillion != readOnlyAgent.Pricing.InputMicrosPerMillion || preview.Preview.OutputPriceMicrosPerMillion != readOnlyAgent.Pricing.OutputMicrosPerMillion {
		t.Fatalf("Brownfield preview did not bind the explicit read-only assessment runtime: %+v", preview.Preview)
	}
	for _, secret := range []string{"NATIVE_MANAGER_ARGUMENT_SENTINEL", "NATIVE_MANAGER_OPTION_SENTINEL", "READONLY_ARGUMENT_SENTINEL", "READONLY_OPTION_SENTINEL", "cancellation-doc"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("manager-run preview exposed private runtime or evidence content %q: %s", secret, out.String())
		}
	}

	invoker := &countingManagerInvoker{}
	var staleOut bytes.Buffer
	err = runBrownfieldManagerStage(options{repo: repo, sourceRepo: repo, sessionID: discovery.ID, input: inputPath, write: true, expect: "sha256:stale-preview"}, &staleOut, invoker)
	if err == nil || !strings.Contains(err.Error(), "does not match") || invoker.runCalls != 0 || staleOut.Len() != 0 {
		t.Fatalf("stale manager-run preview was not rejected before invocation: err=%v calls=%d output=%s", err, invoker.runCalls, staleOut.String())
	}
	var freshPreviewOut bytes.Buffer
	if err := runBrownfieldManagerStage(options{repo: repo, sourceRepo: repo, sessionID: discovery.ID, input: inputPath}, &freshPreviewOut, &countingManagerInvoker{}); err != nil {
		t.Fatalf("refresh exact manager-run preview: %v", err)
	}
	if err := json.Unmarshal(freshPreviewOut.Bytes(), &preview); err != nil || preview.Preview == nil {
		t.Fatalf("decode refreshed manager-run preview: %v %s", err, freshPreviewOut.String())
	}
	invoker = &countingManagerInvoker{}
	err = runBrownfieldManagerStage(options{repo: repo, sourceRepo: repo, sessionID: discovery.ID, input: inputPath, write: true, expect: preview.PreviewDigest}, new(bytes.Buffer), invoker)
	if err == nil || invoker.runCalls != 1 {
		t.Fatalf("typed Brownfield invocation did not reach the mock assessment binding: err=%v calls=%d", err, invoker.runCalls)
	}
	if invoker.lastConfig.Model != readOnlyAgent.Model || len(invoker.lastConfig.Args) != 1 || invoker.lastConfig.Args[0] != "READONLY_ARGUMENT_SENTINEL" {
		t.Fatalf("Brownfield invocation used the native Manager binding: %#v", invoker.lastConfig)
	}
	var requestContext map[string]any
	if err := json.Unmarshal(invoker.lastRequest.Context, &requestContext); err != nil || requestContext["kind"] != "projectadoption-manager-proposal/v1" || requestContext["phase"] != "propose" {
		t.Fatalf("typed Brownfield context did not reach the read-only binding: context=%s err=%v", invoker.lastRequest.Context, err)
	}
}

type countingManagerInvoker struct {
	runCalls    int
	lastConfig  agentexec.Config
	lastRequest agentexec.Request
}

func (i *countingManagerInvoker) Run(_ context.Context, config agentexec.Config, request agentexec.Request, _ agentexec.RunOptions) (agentexec.RunResult, error) {
	i.runCalls++
	i.lastConfig = config
	i.lastRequest = request
	return agentexec.RunResult{}, nil
}

func (*countingManagerInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return agentexec.Fingerprint(config)
}

func TestBrownfieldApplyAdoptionAppliesModelAndRecordsTrustedReceipt(t *testing.T) {
	repo := copyProjectWorld(t)
	selectedSource := filepath.Join(repo, "docs", "cancellation.md")
	selectedBytes, err := os.ReadFile(selectedSource)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selectedSource, append(selectedBytes, []byte("\nPRIVATE_COORDINATOR_SOURCE_SENTINEL: selected source bytes stay ledger-only.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "docs/cancellation.md")
	runGitWithEnv(t, repo, []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid"}, "commit", "-m", "add coordinator privacy fixture")
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	discovery, err := projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "brownfield-real-adoption", Purpose: "Apply a resolved model-only adoption",
		Review: "owner-review-real-adoption", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "cancellation-doc", Path: "docs/cancellation.md", Reason: "Selected intent for order modeling", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, buildDigest, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	session, err := projectadoption.StartBrownfieldSession(repo, target, discovery, []projectadoption.ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectadoption.WriteBrownfieldSession(repo, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	rootID := session.TargetContext.RootManagerID
	result := runBrownfieldMutation(t, repo, discovery.ID, "begin", projectadoption.ReverseIterationRequest{
		ID: "root-pass", ManagerID: rootID, EvidenceIDs: []string{"cancellation-doc"}, DelegationEvidenceIDs: []string{}, Purpose: "Model cancellation intent", Review: "manager-review-adoption",
	}, session.Digest)
	report := makeApplyableStagedDistillation(discovery, target, result.Session.TargetContextDigest, schemaDigest)
	report.Claims[0].Statement = "PRIVATE_COORDINATOR_REPORT_SENTINEL selected claim stays inside the manager ledger."
	blocking := true
	report.Questions = []projectadoption.Question{{ID: "clarify-cancellation", ScopeID: "orders", Prompt: "Which owner-approved cancellation rule governs?",
		Alternatives: []string{"Documented behavior", "Implementation behavior"}, ClaimIDs: []string{"claim-orders"}, Blocking: &blocking}}
	projectadoption.SealDistillation(&report)
	proposal := projectadoption.ManagerProposal{ManagerID: rootID, EvidenceIDs: []string{"cancellation-doc"}, Hierarchy: []projectadoption.ProposedManager{}, PublicContracts: []projectadoption.ManagerPublicContract{}, Report: report}
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", brownfieldProposalInput{IterationID: "root-pass", Proposal: proposal}, result.SessionDigest)
	integration := projectadoption.ManagerIntegration{ManagerID: rootID, ChildProposalDigests: []string{}, ChildContracts: []projectadoption.IntegratedChildContracts{}, Report: report,
		Conflicts: []projectadoption.SessionConflict{{ID: "cancellation-intent-conflict", ScopeID: "orders", QuestionID: "clarify-cancellation",
			Description: "The selected documentation and implementation leave the desired cancellation rule unresolved.", EvidenceIDs: []string{"cancellation-doc"},
			Disposition: "unresolved", Reason: "Coordinator decision required."}}}
	result = runBrownfieldMutation(t, repo, discovery.ID, "integrate", brownfieldIntegrationInput{IterationID: "root-pass", Integration: integration}, result.SessionDigest)
	var resumeOut, resumeErr bytes.Buffer
	if code := Run([]string{"project", "brownfield", "--repo", repo, "--brownfield-action", "resume", "--session", discovery.ID}, &resumeOut, &resumeErr); code != 0 {
		t.Fatalf("resume with coordinator blockers exit=%d stderr=%s", code, resumeErr.String())
	}
	var resumed brownfieldResult
	if err := json.Unmarshal(resumeOut.Bytes(), &resumed); err != nil {
		t.Fatalf("decode coordinator readiness: %v\n%s", err, resumeOut.String())
	}
	if resumed.Readiness == nil || len(resumed.Readiness.BlockingQuestions) != 1 || resumed.Readiness.BlockingQuestions[0].Prompt != "Which owner-approved cancellation rule governs?" ||
		len(resumed.Readiness.UnresolvedConflicts) != 1 || resumed.Readiness.UnresolvedConflicts[0].Description != "The selected documentation and implementation leave the desired cancellation rule unresolved." {
		t.Fatalf("resume hid actionable coordinator diagnostics: %+v", resumed.Readiness)
	}
	for _, private := range []string{"PRIVATE_COORDINATOR_SOURCE_SENTINEL", "PRIVATE_COORDINATOR_REPORT_SENTINEL"} {
		if strings.Contains(resumeOut.String(), private) {
			t.Fatalf("resume exposed raw source or full manager report %q: %s", private, resumeOut.String())
		}
	}
	resolution := projectadoption.Resolution{APIVersion: projectadoption.ResolutionVersion, DiscoveryDigest: discovery.Digest,
		DistillationDigest: report.Digest, ProposalDigest: projectadoption.ProposalDigest(report.Proposal), TargetBasis: target.Digest,
		SchemaDigest: schemaDigest, BuildDigest: buildDigest, Actor: "user", AuthorityClaim: "Owner authorizes this model-only adoption",
		DecisionReference: "review-real-adoption", Authenticated: boolPointer(false), Questions: []projectadoption.QuestionResolution{{QuestionID: "clarify-cancellation", ScopeID: "orders", Disposition: "answer", Answer: "Documented behavior governs", Reason: "Owner answered the actionable question"}},
		Scopes: []projectadoption.ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Owner approved the grounded order scope"}}}
	projectadoption.SealResolution(&resolution)
	result = runBrownfieldMutation(t, repo, discovery.ID, "resolve", brownfieldResolveInput{IterationID: "root-pass", Resolution: resolution}, result.SessionDigest)
	plan, planOutput := runBrownfieldPlanWithOutput(t, repo, discovery.ID, "root-pass")
	if plan.Plan == nil || plan.Plan.PlanDigest == "" {
		t.Fatalf("plan preview returned no exact reviewed plan: %+v", plan)
	}
	if len(plan.Plan.Edit.Mutation.Files) == 0 || !strings.Contains(plan.Plan.Edit.Mutation.Files[0].Content, "A confirmed order can be cancelled before shipment.") {
		t.Fatalf("plan preview no longer exposes the exact reviewable candidate model edit: %+v", plan.Plan)
	}
	for _, private := range []string{"PRIVATE_COORDINATOR_SOURCE_SENTINEL", "PRIVATE_COORDINATOR_REPORT_SENTINEL"} {
		if strings.Contains(planOutput, private) {
			t.Fatalf("plan preview exposed raw source or full manager report %q: %s", private, planOutput)
		}
	}
	wrongPlanInput, err := json.Marshal(brownfieldApplyAdoptionInput{IterationID: "root-pass", ExpectedPlanDigest: strings.Repeat("0", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/apply-adoption-wrong-plan.json", wrongPlanInput); err != nil {
		t.Fatal(err)
	}
	var wrongPlanOut, wrongPlanErr bytes.Buffer
	wrongPlanArgs := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "apply-adoption", "--session", discovery.ID, "--input", ".markitect/drafts/apply-adoption-wrong-plan.json", "--expect", result.SessionDigest, "--write"}
	if code := Run(wrongPlanArgs, &wrongPlanOut, &wrongPlanErr); code == 0 || !strings.Contains(wrongPlanErr.String(), "exact reviewed adoption plan digest") {
		t.Fatalf("wrong reviewed plan digest was not rejected before mutation: exit=%d stderr=%s", code, wrongPlanErr.String())
	}
	implementationSource := filepath.Join(repo, "src", "shop", "orders", "order.py")
	docBefore, err := os.ReadFile(selectedSource)
	if err != nil {
		t.Fatal(err)
	}
	implementationBefore, err := os.ReadFile(implementationSource)
	if err != nil {
		t.Fatal(err)
	}
	forgedInput, err := json.Marshal(map[string]any{"iterationId": "root-pass", "plan": plan.Plan, "receipt": projectadoption.AdoptionReceipt{Status: "adopted", CandidateDigest: strings.Repeat("0", 64)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/forged-adoption-receipt.json", forgedInput); err != nil {
		t.Fatal(err)
	}
	var forgedOut, forgedErr bytes.Buffer
	if code := Run([]string{"project", "brownfield", "--repo", repo, "--brownfield-action", "record-adoption", "--session", discovery.ID, "--input", ".markitect/drafts/forged-adoption-receipt.json", "--expect", result.SessionDigest, "--write"}, &forgedOut, &forgedErr); code == 0 || !strings.Contains(forgedErr.String(), "caller-supplied adoption receipts are not accepted") {
		t.Fatalf("caller-supplied adoption receipt was not rejected: exit=%d stderr=%s", code, forgedErr.String())
	}
	modelPath := filepath.Join(repo, filepath.FromSlash(".markitect/model/commerce/sales/orders/brownfield-cancellation.yaml"))
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Fatalf("rejected receipt unexpectedly applied a model file: %v", err)
	}
	loadedBeforeApply, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil || loadedBeforeApply.Digest != result.SessionDigest {
		t.Fatalf("rejected receipt changed the session ledger: digest=%q err=%v", loadedBeforeApply.Digest, err)
	}

	applyInput, err := json.Marshal(brownfieldApplyAdoptionInput{IterationID: "root-pass", ExpectedPlanDigest: plan.Plan.PlanDigest})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecord(repo, ".markitect/drafts/apply-adoption.json", applyInput); err != nil {
		t.Fatal(err)
	}
	var applyOut, applyErr bytes.Buffer
	applyArgs := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "apply-adoption", "--session", discovery.ID, "--input", ".markitect/drafts/apply-adoption.json", "--expect", result.SessionDigest, "--write"}
	if code := Run(applyArgs, &applyOut, &applyErr); code != 0 {
		t.Fatalf("actual apply-adoption exit=%d stderr=%s", code, applyErr.String())
	}
	var applied brownfieldResult
	if err := json.Unmarshal(applyOut.Bytes(), &applied); err != nil {
		t.Fatalf("decode apply-adoption result: %v\n%s", err, applyOut.String())
	}
	if applied.Status != "recorded" || applied.Action != "apply-adoption" || applied.Readiness != nil || applied.Plan == nil || applied.Plan.PlanDigest != plan.Plan.PlanDigest || applied.Receipt == nil || applied.Receipt.Status != "adopted" || applied.Receipt.CandidateDigest != applied.Plan.Edit.CandidateDigest {
		t.Fatalf("apply result did not bind actual plan and trusted receipt: %+v", applied)
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Fatalf("actual model-only adoption did not create the planned model file: %v", err)
	}
	docAfter, err := os.ReadFile(selectedSource)
	if err != nil {
		t.Fatal(err)
	}
	implementationAfter, err := os.ReadFile(implementationSource)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(docBefore, docAfter) || !bytes.Equal(implementationBefore, implementationAfter) {
		t.Fatal("model adoption changed selected documentation or implementation source")
	}
	loadedAfterApply, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedAfterApply.Digest != applied.SessionDigest || len(loadedAfterApply.Adoptions) != 1 || loadedAfterApply.Adoptions[0].Receipt.CandidateDigest != applied.Receipt.CandidateDigest {
		t.Fatalf("trusted receipt was not durably recorded: %+v", loadedAfterApply.Adoptions)
	}
}

func runBrownfieldPlan(t *testing.T, repo, sessionID, iterationID string) brownfieldResult {
	result, _ := runBrownfieldPlanWithOutput(t, repo, sessionID, iterationID)
	return result
}

func runBrownfieldPlanWithOutput(t *testing.T, repo, sessionID, iterationID string) (brownfieldResult, string) {
	t.Helper()
	data, err := json.Marshal(brownfieldPlanInput{IterationID: iterationID})
	if err != nil {
		t.Fatal(err)
	}
	path := ".markitect/drafts/plan-" + iterationID + ".json"
	if _, err := writeRecord(repo, path, data); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	args := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "plan", "--session", sessionID, "--input", path}
	if code := Run(args, &out, &errOut); code != 0 {
		t.Fatalf("plan preview exit=%d stderr=%s", code, errOut.String())
	}
	var result brownfieldResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("decode plan preview: %v\n%s", err, out.String())
	}
	return result, out.String()
}

func makeApplyableStagedDistillation(discovery projectadoption.Discovery, target *projectwork.Project, contextDigest, schemaDigest string) projectadoption.Distillation {
	evidence := discovery.Evidence[0]
	line := strings.Split(strings.ReplaceAll(evidence.Content, "\r\n", "\n"), "\n")[0]
	content := "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: brownfield-cancellation\n  namespace: commerce.sales.orders\npurpose: Records the owner-reviewed cancellation rule.\nspec:\n  category: rule\n  description: A confirmed order can be cancelled before shipment.\n  public: true\n  uses: []\n  requires: []\n"
	report := projectadoption.Distillation{APIVersion: projectadoption.DistillationVersion, DiscoveryDigest: discovery.Digest,
		TargetBasis: target.Digest, TargetRevision: target.Revision, TargetContextDigest: contextDigest, Method: "human-review", SchemaDigest: schemaDigest,
		Claims: []projectadoption.Claim{{ID: "claim-orders", ScopeID: "orders", Kind: "documented-intent", Method: "documentation", Statement: "The selected documentation describes cancellation intent.",
			Evidence: []projectadoption.EvidenceRef{{EvidenceID: evidence.ID, StartLine: 1, EndLine: 1, Excerpt: line}}, Uncertainty: []string{}}},
		Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{}, Questions: []projectadoption.Question{},
		Scopes:   []projectadoption.ScopeProposal{{ID: "orders", Name: "Orders", ClaimIDs: []string{"claim-orders"}}},
		Proposal: projectadoption.ModelProposal{Goal: "Adopt cancellation documentation as a canonical rule", Files: []projectadoption.ProposedFile{{ScopeID: "orders", Path: ".markitect/model/commerce/sales/orders/brownfield-cancellation.yaml", Content: content}}}}
	projectadoption.SealDistillation(&report)
	return report
}

func runBrownfieldContext(t *testing.T, repo, sessionID, iterationID, phase string, forbidden ...string) brownfieldResult {
	t.Helper()
	data, err := json.Marshal(brownfieldContextInput{IterationID: iterationID, Phase: phase})
	if err != nil {
		t.Fatal(err)
	}
	path := ".markitect/drafts/context-" + iterationID + "-" + phase + ".json"
	if _, err := writeRecord(repo, path, data); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	args := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", "context", "--session", sessionID, "--input", path}
	if code := Run(args, &out, &errOut); code != 0 {
		t.Fatalf("context %s exit=%d stderr=%s", iterationID, code, errOut.String())
	}
	for _, sentinel := range forbidden {
		if sentinel != "" && bytes.Contains(out.Bytes(), []byte(sentinel)) {
			t.Fatalf("context %s leaked unassigned or private session content %q: %s", iterationID, sentinel, out.String())
		}
	}
	var result brownfieldResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("decode context %s: %v\n%s", iterationID, err, out.String())
	}
	if phase == "integrate" && result.IntegrationContext == nil || phase != "integrate" && result.ManagerContext == nil {
		t.Fatalf("context %s returned no Manager context: %s", iterationID, out.String())
	}
	if result.Session != nil || result.Readiness != nil {
		t.Fatalf("context %s returned broad session or readiness data: %s", iterationID, out.String())
	}
	return result
}

func runBrownfieldMutation(t *testing.T, repo, sessionID, action string, request any, priorDigest string) brownfieldResult {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	inputDigest := sha256.Sum256(data)
	path := ".markitect/drafts/" + action + "-" + sessionID + "-" + hex.EncodeToString(inputDigest[:6]) + ".json"
	if _, err := writeRecord(repo, path, data); err != nil {
		t.Fatal(err)
	}
	args := []string{"project", "brownfield", "--repo", repo, "--brownfield-action", action, "--session", sessionID, "--input", path}
	var previewOut, previewErr bytes.Buffer
	if code := Run(args, &previewOut, &previewErr); code != 0 {
		t.Fatalf("%s preview exit=%d stderr=%s input=%s", action, code, previewErr.String(), string(data))
	}
	var preview brownfieldResult
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil {
		t.Fatalf("decode %s preview: %v\n%s", action, err, previewOut.String())
	}
	if preview.Status != "preview" || preview.Session == nil || preview.Session.ID != sessionID || preview.Session.Digest != preview.SessionDigest || preview.Session.TargetContextDigest == "" || len(preview.Session.Iterations) == 0 || preview.PriorSessionDigest != priorDigest || preview.Readiness != nil {
		t.Fatalf("unexpected %s preview: %+v", action, preview)
	}
	for _, private := range []string{"PRIVATE_MANAGER_REPORT_SENTINEL", "PRIVATE_CHILD_REPORT_SENTINEL", "cancellation-doc", "orders-code"} {
		if strings.Contains(previewOut.String(), private) {
			t.Fatalf("%s preview exposed private evidence/report data %q: %s", action, private, previewOut.String())
		}
	}
	writeArgs := append(append([]string(nil), args...), "--expect", priorDigest, "--write")
	var writeOut, writeErr bytes.Buffer
	if code := Run(writeArgs, &writeOut, &writeErr); code != 0 {
		t.Fatalf("%s write exit=%d stderr=%s", action, code, writeErr.String())
	}
	var written brownfieldResult
	if err := json.Unmarshal(writeOut.Bytes(), &written); err != nil {
		t.Fatalf("decode %s write result: %v\n%s", action, err, writeOut.String())
	}
	if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest || written.Readiness != nil {
		t.Fatalf("%s write diverged from preview: preview=%s written=%s", action, preview.SessionDigest, written.SessionDigest)
	}
	for _, private := range []string{"PRIVATE_MANAGER_REPORT_SENTINEL", "PRIVATE_CHILD_REPORT_SENTINEL", "cancellation-doc", "orders-code"} {
		if strings.Contains(writeOut.String(), private) {
			t.Fatalf("%s write exposed private evidence/report data %q: %s", action, private, writeOut.String())
		}
	}
	return written
}

func makeStagedDistillation(discovery projectadoption.Discovery, target *projectwork.Project, contextDigest, schemaDigest, scopeID, claimID, evidenceID, kind, method, statement string) projectadoption.Distillation {
	var evidence projectadoption.Evidence
	for _, item := range discovery.Evidence {
		if item.ID == evidenceID {
			evidence = item
			break
		}
	}
	line := strings.Split(strings.ReplaceAll(evidence.Content, "\r\n", "\n"), "\n")[0]
	return makeStagedReport(discovery, target, contextDigest, schemaDigest,
		[]projectadoption.Claim{{ID: claimID, ScopeID: scopeID, Kind: kind, Method: method, Statement: statement,
			Evidence: []projectadoption.EvidenceRef{{EvidenceID: evidenceID, StartLine: 1, EndLine: 1, Excerpt: line}}, Uncertainty: []string{}}},
		[]projectadoption.ScopeProposal{{ID: scopeID, Name: scopeID, ClaimIDs: []string{claimID}}},
		[]projectadoption.ProposedFile{{ScopeID: scopeID, Path: ".markitect/model/staged/" + scopeID + ".yaml", Content: "apiVersion: test\nkind: Statement\n"}},
	)
}

func makeStagedIntegratedDistillation(discovery projectadoption.Discovery, target *projectwork.Project, contextDigest, schemaDigest string) projectadoption.Distillation {
	var doc, code projectadoption.Evidence
	for _, item := range discovery.Evidence {
		if item.ID == "cancellation-doc" {
			doc = item
		}
		if item.ID == "orders-code" {
			code = item
		}
	}
	return makeStagedReport(discovery, target, contextDigest, schemaDigest,
		[]projectadoption.Claim{
			{ID: "root-claim", ScopeID: "root-scope", Kind: "documented-intent", Method: "documentation", Statement: "Root behavior from docs.", Evidence: []projectadoption.EvidenceRef{{EvidenceID: doc.ID, StartLine: 1, EndLine: 1, Excerpt: strings.Split(strings.ReplaceAll(doc.Content, "\r\n", "\n"), "\n")[0]}}, Uncertainty: []string{}},
			{ID: "child-claim", ScopeID: "child-scope", Kind: "observation", Method: "static-source", Statement: "Child behavior from code.", Evidence: []projectadoption.EvidenceRef{{EvidenceID: code.ID, StartLine: 1, EndLine: 1, Excerpt: strings.Split(strings.ReplaceAll(code.Content, "\r\n", "\n"), "\n")[0]}}, Uncertainty: []string{}},
		},
		[]projectadoption.ScopeProposal{{ID: "root-scope", Name: "Root", ClaimIDs: []string{"root-claim"}}, {ID: "child-scope", Name: "Child", ClaimIDs: []string{"child-claim"}}},
		[]projectadoption.ProposedFile{{ScopeID: "root-scope", Path: ".markitect/model/staged/root.yaml", Content: "kind: Root\n"}, {ScopeID: "child-scope", Path: ".markitect/model/staged/child.yaml", Content: "kind: Child\n"}},
	)
}

func makeStagedReport(discovery projectadoption.Discovery, target *projectwork.Project, contextDigest, schemaDigest string, claims []projectadoption.Claim, scopes []projectadoption.ScopeProposal, files []projectadoption.ProposedFile) projectadoption.Distillation {
	report := projectadoption.Distillation{APIVersion: projectadoption.DistillationVersion, DiscoveryDigest: discovery.Digest,
		TargetBasis: target.Digest, TargetRevision: target.Revision, TargetContextDigest: contextDigest, Method: "human-review", SchemaDigest: schemaDigest,
		Claims: claims, Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{}, Questions: []projectadoption.Question{},
		Scopes: scopes, Proposal: projectadoption.ModelProposal{Goal: "Exercise staged Brownfield CLI", Files: files}}
	projectadoption.SealDistillation(&report)
	return report
}
