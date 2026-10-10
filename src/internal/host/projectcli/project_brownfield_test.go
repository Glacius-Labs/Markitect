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

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// adoptInputKey is the --input key of each record-bearing adopt stage.
var adoptInputKey = map[string]string{
	"start": "start", "begin": "begin", "context": "context", "propose": "propose", "integrate": "integrate",
	"iterate": "iterate", "resolve": "resolve", "plan": "plan", "apply": "apply", "run": "run",
}

// writeAdoptInput writes one stage record under its stage key and returns the
// repository-relative input path.
func writeAdoptInput(t *testing.T, repo, stage string, record any) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{adoptInputKey[stage]: record})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return writeDraft(t, repo, ".markitect/drafts/"+stage+"-"+hex.EncodeToString(sum[:6])+".json", data)
}

func TestAdoptStartPreviewAndCASWrite(t *testing.T) {
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
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "add private evidence fixture")
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
	// The Host runs discovery itself; the test computes the same sealed record
	// only to compare digests.
	discovery, err := projectadoption.Discover(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	input := writeAdoptInput(t, repo, "start", projectapp.BrownfieldStartInput{Request: request, ScopeStatuses: []projectadoption.ScopeStatus{}})
	ledgerPath := filepath.Join(repo, ".markitect", "drafts", "brownfield", discovery.ID, "session.json")
	args := []string{"adopt", "start", "--repo", repo, "--revision", commit, "--input", input}
	code, previewOut, previewErr := runCLI(t, args...)
	if code != 0 {
		t.Fatalf("adopt start preview exit=%d stderr=%s", code, previewErr)
	}
	preview := decodeOutput[projectapp.BrownfieldResult](t, []byte(previewOut))
	if preview.Status != "preview" || preview.Action != "start" || preview.Session == nil || preview.Session.Digest == "" || preview.SessionDigest != preview.Session.Digest || preview.Session.Source.EvidenceCount != 1 || preview.Session.Source.Digest != discovery.Digest || preview.Session.Target.Revision != commit {
		t.Fatalf("unexpected adopt start preview: %+v", preview)
	}
	if strings.Contains(previewOut, "PRIVATE_SOURCE_SENTINEL") || strings.Contains(previewOut, "cancellation-document") {
		t.Fatalf("adopt start preview exposed raw source evidence: %s", previewOut)
	}
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("adopt start preview wrote the session ledger: %v", err)
	}
	if code, _, errout := runCLI(t, append(args, "--write")...); code != 2 || !strings.Contains(errout, "requires --expect") {
		t.Fatalf("adopt start write without --expect exit=%d stderr=%s", code, errout)
	}
	if code, _, errout := runCLI(t, append(args, "--expect", "sha256:stale", "--write")...); code != 2 || !strings.Contains(errout, "markitect adopt:") {
		t.Fatalf("adopt start write with a stale digest exit=%d stderr=%s", code, errout)
	}
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("rejected adopt start write created the session ledger: %v", err)
	}

	code, writeOut, writeErr := runCLI(t, append(args, "--expect", preview.SessionDigest, "--write")...)
	if code != 0 {
		t.Fatalf("adopt start write exit=%d stderr=%s", code, writeErr)
	}
	written := decodeOutput[projectapp.BrownfieldResult](t, []byte(writeOut))
	if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest {
		t.Fatalf("adopt start write differs from reviewed preview: %+v", written)
	}
	if strings.Contains(writeOut, "PRIVATE_SOURCE_SENTINEL") || strings.Contains(writeOut, "cancellation-document") {
		t.Fatalf("adopt start write exposed raw source evidence: %s", writeOut)
	}
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("adoption session was not durably written: %v", err)
	}

	statusOut := mustCLI(t, "adopt", "status", "--repo", repo, "--session", discovery.ID)
	status := decodeOutput[projectapp.BrownfieldResult](t, statusOut)
	if status.Status != "status" || status.SessionDigest != preview.SessionDigest || status.Readiness == nil || !status.Readiness.SourceCurrent || !status.Readiness.TargetCurrent {
		t.Fatalf("adopt status did not validate both fixed bases: %+v", status)
	}
	if status.Session == nil || status.Session.Source.Digest != discovery.Digest || status.Session.Target.ProjectDigest == "" || bytes.Contains(statusOut, []byte("PRIVATE_SOURCE_SENTINEL")) {
		t.Fatalf("adopt status did not return safe fixed-basis metadata or leaked source evidence: %s", statusOut)
	}
	if code, _, errout := runCLI(t, "adopt", "status", "--repo", repo, "--session", discovery.ID, "--expect", preview.SessionDigest); code != 2 || !strings.Contains(errout, "--expect is valid only together with --write or --execute") {
		t.Fatalf("adopt status accepted --expect: exit=%d stderr=%s", code, errout)
	}

	// The project overview lists the open adoption session without writing.
	summary := decodeOutput[overview](t, mustCLI(t, "status", "--repo", repo))
	if len(summary.Adoptions) != 1 || summary.Adoptions[0].Session != discovery.ID || summary.Adoptions[0].Stage != "started" {
		t.Fatalf("status overview adoptions = %+v", summary.Adoptions)
	}
}

func TestAdoptInputRejectsUnknownAndTrailingJSON(t *testing.T) {
	repo := copyProjectWorld(t)
	for name, input := range map[string]string{
		"unknown field":     `{"plan":{"iterationId":"iteration-a","unexpected":true}}`,
		"unknown stage key": `{"planning":{"iterationId":"iteration-a"}}`,
		"trailing JSON":     `{"plan":{"iterationId":"iteration-a"}}{}`,
		"duplicate key":     `{"plan":{"iterationId":"iteration-a","iterationId":"iteration-b"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := writeDraft(t, repo, ".markitect/drafts/closed-"+strings.ReplaceAll(name, " ", "-")+".json", []byte(input))
			code, out, errout := runCLI(t, "adopt", "plan", "--repo", repo, "--session", "missing-session", "--input", path)
			if code != 2 || out != "" || !strings.Contains(errout, "markitect adopt:") || strings.Contains(errout, "missing-session") {
				t.Fatalf("accepted non-closed adopt input: exit=%d stdout=%s stderr=%s", code, out, errout)
			}
		})
	}
}

func TestAdoptManagerContextIsReadOnlyAndBoundToIteration(t *testing.T) {
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
	input := writeAdoptInput(t, repo, "context", projectapp.BrownfieldContextInput{IterationID: iterationID})
	result := decodeOutput[projectapp.BrownfieldResult](t, mustCLI(t, "adopt", "context", "--repo", repo, "--session", discovery.ID, "--input", input))
	if result.Action != "context" || result.ManagerContext == nil || result.ManagerContext.IterationID != iterationID || len(result.ManagerContext.Evidence) != 1 || result.ManagerContext.Evidence[0].EvidenceID != "cancellation-document" || result.ManagerContext.Evidence[0].Classification != "documented-intent" {
		t.Fatalf("context did not return the exact assigned evidence: %+v", result)
	}
	if code, _, errout := runCLI(t, "adopt", "context", "--repo", repo, "--session", discovery.ID, "--input", input, "--expect", session.Digest, "--write"); code != 2 || !strings.Contains(errout, "read-only") {
		t.Fatalf("adopt context accepted a write: exit=%d stderr=%s", code, errout)
	}
	loaded, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Digest != session.Digest {
		t.Fatalf("read-only context changed session digest: got %s, want %s", loaded.Digest, session.Digest)
	}
}

func TestAdoptStagedManagerLoopBeginContextProposeAndIntegrate(t *testing.T) {
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
		Hierarchy:       []projectadoption.ProposedManager{{ID: childID, Name: "Cancellation Owner", Purpose: "Own cancellation implementation", ParentID: rootID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}}},
		PublicContracts: []projectadoption.ManagerPublicContract{}, Report: rootReport}
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", projectapp.BrownfieldProposalInput{IterationID: "root-pass", Proposal: rootProposal}, result.SessionDigest)

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
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", projectapp.BrownfieldProposalInput{IterationID: "child-pass", Proposal: childProposal}, result.SessionDigest)
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
	runBrownfieldMutation(t, repo, discovery.ID, "integrate", projectapp.BrownfieldIntegrationInput{IterationID: "root-pass", Integration: integration}, result.SessionDigest)
	fullSession, err = projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fullSession.Iterations[0].Integration == nil || fullSession.Iterations[0].Integration.ChildProposalDigests[0] != childDigest || fullSession.Iterations[0].Integration.ChildContracts[0].Contracts[0].Contract.ID != "cancellation-api" {
		t.Fatalf("root integration did not retain immutable child contract evidence: %+v", fullSession.Iterations[0].Integration)
	}
}

func TestAdoptRunPreviewAndStaleGuardDoNotInvokeProvider(t *testing.T) {
	repo := copyProjectWorld(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	initial, err := projectwork.Load(repo, gitOutput(t, repo, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	initialContext, err := projectadoption.TargetContextForProject(initial)
	if err != nil {
		t.Fatal(err)
	}
	managerID := initialContext.RootManagerID
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
	runtimeConfig := projectrun.Runtime{
		APIVersion: projectrun.APIVersion, Mode: projectrun.ModeControlledLocal,
		Agents: map[string]projectrun.Agent{managerID: nativeManager},
		Review: &projectrun.ReviewConfig{Agents: map[string]projectrun.Agent{managerID: readOnlyAgent}, MaxRounds: 1, MaxManagerRounds: 1},
		Limits: projectrun.Limits{MaxDepth: 4, MaxStarts: 8, MaxRetries: 1, MaxParallel: 1, MaxDuration: projectrun.Duration(5 * time.Minute),
			MaxCostMicros: 1000, MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 8 << 20},
	}
	runtimeBytes, err := json.Marshal(runtimeConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(projectrun.RuntimePath)), runtimeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", projectrun.RuntimePath)
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "add manager runtime fixture")
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
	if session.TargetContext.RootManagerID != managerID {
		t.Fatalf("fixture root Manager = %s, want %s", session.TargetContext.RootManagerID, managerID)
	}
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
	inputPath := writeAdoptInput(t, repo, "run", projectapp.BrownfieldManagerRunInput{IterationID: "manager-run-root", Phase: "propose", AgentManagerID: managerID})
	args := []string{"adopt", "run", "--repo", repo, "--session", discovery.ID, "--input", inputPath}
	code, out, errout := runCLI(t, args...)
	if code != 0 {
		t.Fatalf("adopt run preview exit=%d stderr=%s", code, errout)
	}
	preview := decodeOutput[projectapp.BrownfieldManagerRunOutput](t, []byte(out))
	if preview.Status != "preview" || preview.SessionDigest != session.Digest || preview.Preview == nil || preview.PreviewDigest == "" || preview.Preview.PreviewDigest != preview.PreviewDigest || preview.Attempt != nil {
		t.Fatalf("unexpected adopt run preview: %+v", preview)
	}
	if preview.Preview.Model != readOnlyAgent.Model || preview.Preview.ProviderVersion != readOnlyAgent.ProviderVersion ||
		preview.Preview.InputPriceMicrosPerMillion != readOnlyAgent.Pricing.InputMicrosPerMillion || preview.Preview.OutputPriceMicrosPerMillion != readOnlyAgent.Pricing.OutputMicrosPerMillion {
		t.Fatalf("adopt run preview did not bind the explicit read-only assessment runtime: %+v", preview.Preview)
	}
	for _, secret := range []string{"NATIVE_MANAGER_ARGUMENT_SENTINEL", "NATIVE_MANAGER_OPTION_SENTINEL", "READONLY_ARGUMENT_SENTINEL", "READONLY_OPTION_SENTINEL", "cancellation-doc"} {
		if strings.Contains(out, secret) {
			t.Fatalf("adopt run preview exposed private runtime or evidence content %q: %s", secret, out)
		}
	}

	operations := func(invoker projectrun.Invoker) projectapp.Operations {
		return projectapp.Operations{Host: projectRunHost(), Invoker: invoker}
	}
	invoker := &countingManagerInvoker{}
	if code, out, errout := runCLIWith(t, operations(invoker), append(args, "--expect", preview.PreviewDigest, "--write")...); code != 2 || out != "" || !strings.Contains(errout, "requires --execute") || invoker.runCalls != 0 {
		t.Fatalf("adopt run --write without --execute: exit=%d stdout=%s stderr=%s calls=%d", code, out, errout, invoker.runCalls)
	}
	code, staleOut, staleErr := runCLIWith(t, operations(invoker), append(args, "--expect", "sha256:stale-preview", "--write", "--execute")...)
	if code != 2 || !strings.Contains(staleErr, "does not match") || invoker.runCalls != 0 || staleOut != "" {
		t.Fatalf("stale adopt run preview was not rejected before invocation: exit=%d stderr=%s calls=%d output=%s", code, staleErr, invoker.runCalls, staleOut)
	}
	code, freshOut, freshErr := runCLIWith(t, operations(&countingManagerInvoker{}), args...)
	if code != 0 {
		t.Fatalf("refresh exact adopt run preview: exit=%d stderr=%s", code, freshErr)
	}
	preview = decodeOutput[projectapp.BrownfieldManagerRunOutput](t, []byte(freshOut))
	invoker = &countingManagerInvoker{}
	code, _, _ = runCLIWith(t, operations(invoker), append(args, "--expect", preview.PreviewDigest, "--write", "--execute")...)
	if code == 0 || invoker.runCalls != 1 {
		t.Fatalf("typed adopt run invocation did not reach the mock assessment binding: exit=%d calls=%d", code, invoker.runCalls)
	}
	if invoker.lastConfig.Model != readOnlyAgent.Model || len(invoker.lastConfig.Args) != 1 || invoker.lastConfig.Args[0] != "READONLY_ARGUMENT_SENTINEL" {
		t.Fatalf("adopt run used the native Manager binding: %#v", invoker.lastConfig)
	}
	var requestContext map[string]any
	if err := json.Unmarshal(invoker.lastRequest.Context, &requestContext); err != nil || requestContext["kind"] != "projectadoption-manager-proposal/v1" || requestContext["phase"] != "propose" {
		t.Fatalf("typed adopt run context did not reach the read-only binding: context=%s err=%v", invoker.lastRequest.Context, err)
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

func TestAdoptApplyAppliesModelAndRecordsTrustedReceipt(t *testing.T) {
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
	runGitWithEnv(t, repo, testCommitEnv, "commit", "-m", "add coordinator privacy fixture")
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
	result = runBrownfieldMutation(t, repo, discovery.ID, "propose", projectapp.BrownfieldProposalInput{IterationID: "root-pass", Proposal: proposal}, result.SessionDigest)
	integration := projectadoption.ManagerIntegration{ManagerID: rootID, ChildProposalDigests: []string{}, ChildContracts: []projectadoption.IntegratedChildContracts{}, Report: report,
		Conflicts: []projectadoption.SessionConflict{{ID: "cancellation-intent-conflict", ScopeID: "orders", QuestionID: "clarify-cancellation",
			Description: "The selected documentation and implementation leave the desired cancellation rule unresolved.", EvidenceIDs: []string{"cancellation-doc"},
			Disposition: "unresolved", Reason: "Coordinator decision required."}}}
	result = runBrownfieldMutation(t, repo, discovery.ID, "integrate", projectapp.BrownfieldIntegrationInput{IterationID: "root-pass", Integration: integration}, result.SessionDigest)
	statusOut := mustCLI(t, "adopt", "status", "--repo", repo, "--session", discovery.ID)
	status := decodeOutput[projectapp.BrownfieldResult](t, statusOut)
	if status.Readiness == nil || len(status.Readiness.BlockingQuestions) != 1 || status.Readiness.BlockingQuestions[0].Prompt != "Which owner-approved cancellation rule governs?" ||
		len(status.Readiness.UnresolvedConflicts) != 1 || status.Readiness.UnresolvedConflicts[0].Description != "The selected documentation and implementation leave the desired cancellation rule unresolved." {
		t.Fatalf("adopt status hid actionable coordinator diagnostics: %+v", status.Readiness)
	}
	for _, private := range []string{"PRIVATE_COORDINATOR_SOURCE_SENTINEL", "PRIVATE_COORDINATOR_REPORT_SENTINEL"} {
		if bytes.Contains(statusOut, []byte(private)) {
			t.Fatalf("adopt status exposed raw source or full manager report %q: %s", private, statusOut)
		}
	}

	// resolve takes only the human choices; the Host builds and seals the
	// Resolution against the session's own discovery, report and target.
	choices := projectapp.ResolutionChoices{Actor: "user", AuthorityClaim: "Owner authorizes this model-only adoption", DecisionReference: "review-real-adoption",
		Questions: []projectadoption.QuestionResolution{{QuestionID: "clarify-cancellation", ScopeID: "orders", Disposition: "answer", Answer: "Documented behavior governs", Reason: "Owner answered the actionable question"}},
		Scopes:    []projectadoption.ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Owner approved the grounded order scope"}}}
	incomplete := writeAdoptInput(t, repo, "resolve", map[string]any{"iterationId": "root-pass", "choices": map[string]any{
		"actor": "user", "authorityClaim": "x", "decisionReference": "d", "questions": nil, "scopes": choices.Scopes}})
	if code, _, errout := runCLI(t, "adopt", "resolve", "--repo", repo, "--session", discovery.ID, "--input", incomplete); code != 2 || !strings.Contains(errout, "choices must include") {
		t.Fatalf("adopt resolve accepted choices without questions: exit=%d stderr=%s", code, errout)
	}
	unknownActor := choices
	unknownActor.Actor = "someone-else"
	if code, _, errout := runCLI(t, "adopt", "resolve", "--repo", repo, "--session", discovery.ID, "--input", writeAdoptInput(t, repo, "resolve", projectapp.BrownfieldResolveInput{IterationID: "root-pass", Choices: unknownActor})); code != 2 || !strings.Contains(errout, "actor must be user or an active Manager ID") {
		t.Fatalf("adopt resolve accepted an unknown actor: exit=%d stderr=%s", code, errout)
	}
	modelPath := filepath.Join(repo, filepath.FromSlash(".markitect/model/commerce/sales/orders/brownfield-cancellation.yaml"))
	result = runBrownfieldMutation(t, repo, discovery.ID, "resolve", projectapp.BrownfieldResolveInput{IterationID: "root-pass", Choices: choices}, result.SessionDigest)
	resolved, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil {
		t.Fatal(err)
	}
	resolution := resolved.Iterations[0].Resolution
	if resolution == nil || resolution.Digest == "" || resolution.DiscoveryDigest != discovery.Digest || resolution.DistillationDigest != report.Digest ||
		resolution.ProposalDigest != projectadoption.ProposalDigest(report.Proposal) || resolution.TargetBasis != target.Digest ||
		resolution.SchemaDigest != schemaDigest || resolution.BuildDigest != buildDigest || resolution.Authenticated == nil || *resolution.Authenticated ||
		resolution.Actor != "user" || resolution.DecisionReference != "review-real-adoption" {
		t.Fatalf("resolution bindings do not match the session and active Host values: %+v", resolution)
	}
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Fatalf("resolve wrote a model proposal to disk: %v", err)
	}
	if after, err := projectwork.Load(repo, commit); err != nil || after.Digest != target.Digest {
		t.Fatalf("resolve changed the fixed project model: err=%v", err)
	}

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
	wrongPlanInput := writeAdoptInput(t, repo, "apply", projectapp.BrownfieldApplyAdoptionInput{IterationID: "root-pass", ExpectedPlanDigest: strings.Repeat("0", 64)})
	if code, _, errout := runCLI(t, "adopt", "apply", "--repo", repo, "--session", discovery.ID, "--input", wrongPlanInput, "--expect", result.SessionDigest, "--write"); code != 2 || !strings.Contains(errout, "exact reviewed adoption plan digest") {
		t.Fatalf("wrong reviewed plan digest was not rejected before mutation: exit=%d stderr=%s", code, errout)
	}
	applyInput := writeAdoptInput(t, repo, "apply", projectapp.BrownfieldApplyAdoptionInput{IterationID: "root-pass", ExpectedPlanDigest: plan.Plan.PlanDigest})
	if code, _, errout := runCLI(t, "adopt", "apply", "--repo", repo, "--session", discovery.ID, "--input", applyInput); code != 2 || !strings.Contains(errout, "requires --write") {
		t.Fatalf("adopt apply without --write was not rejected: exit=%d stderr=%s", code, errout)
	}
	if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
		t.Fatalf("rejected apply unexpectedly applied a model file: %v", err)
	}
	loadedBeforeApply, err := projectadoption.LoadBrownfieldSession(repo, discovery.ID)
	if err != nil || loadedBeforeApply.Digest != result.SessionDigest {
		t.Fatalf("rejected apply changed the session ledger: digest=%q err=%v", loadedBeforeApply.Digest, err)
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

	applied := decodeOutput[projectapp.BrownfieldResult](t, mustCLI(t, "adopt", "apply", "--repo", repo, "--session", discovery.ID, "--input", applyInput, "--expect", result.SessionDigest, "--write"))
	if applied.Status != "recorded" || applied.Action != "apply" || applied.Readiness != nil || applied.Plan == nil || applied.Plan.PlanDigest != plan.Plan.PlanDigest || applied.Receipt == nil || applied.Receipt.Status != "adopted" || applied.Receipt.CandidateDigest != applied.Plan.Edit.CandidateDigest {
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

func runBrownfieldPlan(t *testing.T, repo, sessionID, iterationID string) projectapp.BrownfieldResult {
	t.Helper()
	result, _ := runBrownfieldPlanWithOutput(t, repo, sessionID, iterationID)
	return result
}

func runBrownfieldPlanWithOutput(t *testing.T, repo, sessionID, iterationID string) (projectapp.BrownfieldResult, string) {
	t.Helper()
	path := writeAdoptInput(t, repo, "plan", projectapp.BrownfieldPlanInput{IterationID: iterationID})
	out := mustCLI(t, "adopt", "plan", "--repo", repo, "--session", sessionID, "--input", path)
	return decodeOutput[projectapp.BrownfieldResult](t, out), string(out)
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

func runBrownfieldContext(t *testing.T, repo, sessionID, iterationID, phase string, forbidden ...string) projectapp.BrownfieldResult {
	t.Helper()
	path := writeAdoptInput(t, repo, "context", projectapp.BrownfieldContextInput{IterationID: iterationID, Phase: phase})
	out := mustCLI(t, "adopt", "context", "--repo", repo, "--session", sessionID, "--input", path)
	for _, sentinel := range forbidden {
		if sentinel != "" && bytes.Contains(out, []byte(sentinel)) {
			t.Fatalf("context %s leaked unassigned or private session content %q: %s", iterationID, sentinel, out)
		}
	}
	result := decodeOutput[projectapp.BrownfieldResult](t, out)
	if phase == "integrate" && result.IntegrationContext == nil || phase != "integrate" && result.ManagerContext == nil {
		t.Fatalf("context %s returned no Manager context: %s", iterationID, out)
	}
	if result.Session != nil || result.Readiness != nil {
		t.Fatalf("context %s returned broad session or readiness data: %s", iterationID, out)
	}
	return result
}

// runBrownfieldMutation previews one adopt stage, then writes it with the
// prior session digest, and checks that the write equals the preview.
func runBrownfieldMutation(t *testing.T, repo, sessionID, stage string, request any, priorDigest string) projectapp.BrownfieldResult {
	t.Helper()
	path := writeAdoptInput(t, repo, stage, request)
	args := []string{"adopt", stage, "--repo", repo, "--session", sessionID, "--input", path}
	code, previewOut, previewErr := runCLI(t, args...)
	if code != 0 {
		t.Fatalf("%s preview exit=%d stderr=%s", stage, code, previewErr)
	}
	preview := decodeOutput[projectapp.BrownfieldResult](t, []byte(previewOut))
	if preview.Status != "preview" || preview.Session == nil || preview.Session.ID != sessionID || preview.Session.Digest != preview.SessionDigest || preview.Session.TargetContextDigest == "" || len(preview.Session.Iterations) == 0 || preview.PriorSessionDigest != priorDigest || preview.Readiness != nil {
		t.Fatalf("unexpected %s preview: %+v", stage, preview)
	}
	for _, private := range []string{"PRIVATE_MANAGER_REPORT_SENTINEL", "PRIVATE_CHILD_REPORT_SENTINEL", "cancellation-doc", "orders-code"} {
		if strings.Contains(previewOut, private) {
			t.Fatalf("%s preview exposed private evidence/report data %q: %s", stage, private, previewOut)
		}
	}
	if loaded, err := projectadoption.LoadBrownfieldSession(repo, sessionID); err != nil || loaded.Digest != priorDigest {
		t.Fatalf("%s preview changed the session ledger: err=%v", stage, err)
	}
	code, writeOut, writeErr := runCLI(t, append(args, "--expect", priorDigest, "--write")...)
	if code != 0 {
		t.Fatalf("%s write exit=%d stderr=%s", stage, code, writeErr)
	}
	written := decodeOutput[projectapp.BrownfieldResult](t, []byte(writeOut))
	if written.Status != "recorded" || written.SessionDigest != preview.SessionDigest || written.Readiness != nil {
		t.Fatalf("%s write diverged from preview: preview=%s written=%s", stage, preview.SessionDigest, written.SessionDigest)
	}
	for _, private := range []string{"PRIVATE_MANAGER_REPORT_SENTINEL", "PRIVATE_CHILD_REPORT_SENTINEL", "cancellation-doc", "orders-code"} {
		if strings.Contains(writeOut, private) {
			t.Fatalf("%s write exposed private evidence/report data %q: %s", stage, private, writeOut)
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
		Scopes: scopes, Proposal: projectadoption.ModelProposal{Goal: "Exercise staged adopt CLI", Files: files}}
	projectadoption.SealDistillation(&report)
	return report
}
