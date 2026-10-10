package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/assurance"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
)

const (
	canonicalControllerVerifierTestActorEnv   = "MARKITECT_CANONICAL_CONTROLLER_VERIFIER_TEST_ACTOR"
	canonicalControllerVerifierTestOutcomeEnv = "MARKITECT_CANONICAL_CONTROLLER_VERIFIER_TEST_OUTCOME"
	canonicalControllerVerifierTestMarkerEnv  = "MARKITECT_CANONICAL_CONTROLLER_VERIFIER_TEST_MARKER"
	canonicalControllerVerifierTestPartialEnv = "MARKITECT_CANONICAL_CONTROLLER_VERIFIER_TEST_PARTIAL"
)

// This helper process speaks the runner protocol only. Its outcome is a test
// input; its references and observations do not establish semantic truth.
func TestCanonicalControllerVerifierProtocolProcess(t *testing.T) {
	if os.Getenv(canonicalControllerVerifierTestActorEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil || invocation.Request.Role != agentexec.RoleVerifier {
		os.Exit(41)
	}
	marker := os.Getenv(canonicalControllerVerifierTestMarkerEnv)
	if marker != "" {
		file, err := os.OpenFile(marker, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(42)
		}
		_, _ = fmt.Fprintln(file, invocation.Request.ProjectionID)
		_ = file.Close()
	}
	outcome := os.Getenv(canonicalControllerVerifierTestOutcomeEnv)
	if outcome == "" {
		outcome = agentexec.OutcomePassed
	}
	refs := append([]string(nil), invocation.Request.ScopeIDs...)
	refs = append(refs, invocation.Request.PolicyIDs...)
	observations := make([]agentexec.Observation, 0, len(refs)+len(invocation.Request.Artifacts))
	for _, artifact := range invocation.Request.Artifacts {
		refs = append(refs, artifact.Path)
	}
	sort.Strings(refs)
	var verifierContext canonicalControllerVerifierContext
	if err := json.Unmarshal(invocation.Request.Context, &verifierContext); err != nil {
		os.Exit(44)
	}
	for _, subject := range verifierContext.RequiredObservationSubjects {
		observations = append(observations, agentexec.Observation{Subject: subject.Subject, Outcome: outcome, Detail: "protocol fixture observation"})
	}
	if os.Getenv(canonicalControllerVerifierTestPartialEnv) == "1" && len(observations) > 1 {
		observations = observations[:1]
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleVerifier, InputDigest: invocation.InputDigest, Outcome: outcome,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: refs,
		VerifierObservations: observations, Uncertainty: []string{},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(43)
	}
	os.Exit(0)
}

func TestCanonicalControllerVerifierRequiresExactEvidenceReferences(t *testing.T) {
	request := agentexec.Request{
		ScopeIDs:  []string{"core/v1:Rule:policy", "core/v1:UseCase:orders"},
		PolicyIDs: []string{"core/v1:Rule:boundary"},
		Artifacts: []agentexec.Artifact{{Path: "docs/orders.md"}},
	}
	exact := []string{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:Rule:policy"}
	if !canonicalControllerExactEvidenceRefs(exact, request) {
		t.Fatal("exact selected scope, policy, and artifact references were rejected")
	}
	for _, refs := range [][]string{
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders"},
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:Rule:policy", "README.md"},
		{"docs/orders.md", "core/v1:Rule:boundary", "core/v1:UseCase:orders", "core/v1:UseCase:orders"},
	} {
		if canonicalControllerExactEvidenceRefs(refs, request) {
			t.Fatalf("inexact evidence references were accepted: %#v", refs)
		}
	}
}

func TestCanonicalControllerVerifierReceiptBindingKeepsMissingReceiptFailureDistinct(t *testing.T) {
	const expected = "sha256:" + "aabb"
	if err := canonicalControllerVerifierReceiptInputBinding(agentexec.Receipt{InputDigest: expected}, expected, nil); err != nil {
		t.Fatalf("matching receipt input digest was rejected: %v", err)
	}
	if err := canonicalControllerVerifierReceiptInputBinding(agentexec.Receipt{InputDigest: "sha256:ccdd"}, expected, nil); err == nil || err.Error() != "Verifier receipt input digest differs from the reconstructed request" {
		t.Fatalf("mismatched nonempty receipt digest was not rejected strictly: %v", err)
	}
	invocationErr := fmt.Errorf("private log directory could not be created")
	err := canonicalControllerVerifierReceiptInputBinding(agentexec.Receipt{}, expected, invocationErr)
	if err == nil || !strings.Contains(err.Error(), invocationErr.Error()) || strings.Contains(err.Error(), "input digest differs") {
		t.Fatalf("pre-receipt invocation failure was mislabeled as a digest mismatch: %v", err)
	}
	if err := canonicalControllerVerifierReceiptInputBinding(agentexec.Receipt{}, expected, nil); err == nil || !strings.Contains(err.Error(), "returned no receipt") {
		t.Fatalf("missing receipt without invocation error was accepted: %v", err)
	}
}

func TestCanonicalControllerVerifierRequestBindsAgentExecutionAPIAndCoverage(t *testing.T) {
	record := controllerVerificationTestRecord(t, "core/v1:Projection:protocol", "core/v1:UseCase:orders")
	check := records.CheckResult{ID: "fixed-check", Version: "command/v1", Digest: sha256Prefix(sha256Hex([]byte("fixed check"))), Outcome: records.CheckPassed}
	item := canonicalControllerPreparedVerification{
		scope: CanonicalAssuranceScope{ID: "orders", ProjectionID: record.ProjectionID}, record: record,
		context: CanonicalAgentContext{RequestDigest: "sha256:" + strings.Repeat("1", 64), ScopeIDs: record.ScopeIDs},
		request: canonical.ProjectionRequest{}, checks: []records.CheckResult{check},
		artifacts: []agentexec.Artifact{
			{Path: "docs/orders.md", Mode: "0644", Digest: sha256Prefix(sha256Hex([]byte("doc"))), Content: []byte("doc")},
			{Path: "checks/orders.go", Mode: "0644", Digest: sha256Prefix(sha256Hex([]byte("check source"))), Content: []byte("check source")},
		},
		recordByScope: map[string]records.ProjectionRecord{"orders": record}, scopeIndex: map[string]CanonicalAssuranceScope{"orders": {ID: "orders", ProjectionID: record.ProjectionID}},
	}
	request, err := canonicalControllerVerifierRequest(CanonicalControllerConfig{}, item, assurance.NodeRunInput{Node: assurance.Node{ID: "orders"}}, strings.Repeat("b", 40))
	if err != nil {
		t.Fatal(err)
	}
	var payload canonicalControllerVerifierContext
	if err := json.Unmarshal(request.Context, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AgentExecutionAPIVersion != agentexec.APIVersion || len(payload.RequiredObservationSubjects) != 4 {
		t.Fatalf("request context omitted API/required coverage binding: api=%q subjects=%#v", payload.AgentExecutionAPIVersion, payload.RequiredObservationSubjects)
	}
	var sawCheckInputArtifact bool
	for _, subject := range payload.RequiredObservationSubjects {
		if subject.Kind == "artifact" && subject.ID == "checks/orders.go" {
			sawCheckInputArtifact = true
		}
	}
	if !sawCheckInputArtifact {
		t.Fatal("exact supplied fixed-check input artifact was omitted from required observations")
	}
	before, err := canonicalControllerVerifierInputDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	payload.AgentExecutionAPIVersion += "/changed"
	request.Context, err = json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	after, err := canonicalControllerVerifierInputDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("changing the agent execution API context did not change the stable verifier request digest")
	}
}

func TestCanonicalControllerVerifierRequestRefusesOversizedRequiredCoverage(t *testing.T) {
	record := controllerVerificationTestRecord(t, "core/v1:Projection:protocol", "core/v1:UseCase:orders")
	item := canonicalControllerPreparedVerification{
		scope: CanonicalAssuranceScope{ID: "orders", ProjectionID: record.ProjectionID}, record: record,
		context:   CanonicalAgentContext{RequestDigest: "sha256:" + strings.Repeat("1", 64)},
		artifacts: []agentexec.Artifact{{Path: "docs/orders.md", Mode: "0644", Digest: sha256Prefix(sha256Hex([]byte("doc"))), Content: []byte("doc")}},
	}
	for i := 0; i < agentexec.MaxVerifierObservations; i++ {
		item.context.ScopeIDs = append(item.context.ScopeIDs, fmt.Sprintf("core/v1:UseCase:scope-%03d", i))
	}
	_, err := canonicalControllerVerifierRequest(CanonicalControllerConfig{}, item, assurance.NodeRunInput{Node: assurance.Node{ID: "orders"}}, strings.Repeat("b", 40))
	if err == nil || !strings.Contains(err.Error(), "128-entry protocol bound") {
		t.Fatalf("oversized required coverage was not refused before invocation: %v", err)
	}
}

func TestCanonicalControllerComposedOutcomeRetainsFailureAndEscalation(t *testing.T) {
	cases := []struct {
		left, right, want string
	}{
		{records.OutcomePassed, records.OutcomeIncomplete, records.OutcomeIncomplete},
		{records.OutcomeEscalated, records.OutcomeIncomplete, records.OutcomeEscalated},
		{records.OutcomeFailed, records.OutcomeEscalated, records.OutcomeFailed},
		{records.OutcomePassed, records.OutcomePassed, records.OutcomePassed},
	}
	for _, tc := range cases {
		if got := composeVerificationOutcomes(tc.left, tc.right); got != tc.want {
			t.Errorf("composeVerificationOutcomes(%q, %q) = %q, want %q", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCanonicalControllerAssuranceRejectsParentMissingChildSourceIDs(t *testing.T) {
	parent := controllerVerificationTestRecord(t, "core/v1:Projection:parent", "core/v1:UseCase:orders")
	child := controllerVerificationTestRecord(t, "core/v1:Projection:child", "core/v1:Handler:billing")
	cfg := CanonicalControllerConfig{
		AssuranceRoots: []string{"parent"},
		AssuranceScopes: []CanonicalAssuranceScope{
			{ID: "parent", ProjectionID: parent.ProjectionID, Children: []string{"child"}, Checks: []authoring.Check{{Name: "parent-check", Run: []string{"go", "version"}}}},
			{ID: "child", ProjectionID: child.ProjectionID, Checks: []authoring.Check{{Name: "child-check", Run: []string{"go", "version"}}}},
		},
	}
	_, _, _, err := canonicalControllerVerificationNodes(cfg, []records.ProjectionRecord{parent, child})
	if err == nil || !strings.Contains(err.Error(), "core/v1:Handler:billing") {
		t.Fatalf("missing child canonical source identity was not reported: %v", err)
	}
}

func controllerVerificationTestRecord(t *testing.T, projectionID, scopeID string) records.ProjectionRecord {
	t.Helper()
	digest := sha256Prefix(sha256Hex([]byte(projectionID + scopeID)))
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: strings.Repeat("a", 40), ModelDigest: digest, PlanDigest: digest,
		InputSnapshotDigest: digest, RequestDigest: digest,
		ProjectionID: projectionID,
		Module:       records.ModuleIdentity{Name: "module", Version: "1", Digest: digest},
		Projector:    records.ProjectorIdentity{ID: "markdown", Version: "1"},
		ScopeIDs:     []string{scopeID}, PolicyIDs: []string{},
		Artifacts: []records.Artifact{{Path: "docs/" + strings.TrimPrefix(projectionID, "core/v1:Projection:") + ".md", Digest: digest, Mode: "100644", Change: records.ChangeCreated}},
		State:     records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestCanonicalControllerVerifierLifecycleBindsEvidenceAndPersistsOnlyOnWrite(t *testing.T) {
	if !runCanonicalControllerScenarioInChild(t) {
		return
	}
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	beforeHead := scopedTestGit(t, root, "rev-parse", "HEAD")
	store, before, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil || len(active) != 2 {
		t.Fatalf("fixture ledger active records=%d: %v", len(active), err)
	}
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	readOnly, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, false)
	if err != nil {
		t.Fatalf("read-only scoped verification: %v", err)
	}
	if readOnly.SourceRevision != sourceRevision || readOnly.EvidenceRevision != evidenceRevision || readOnly.Outcome != records.OutcomePassed || len(readOnly.Results) != 2 || len(readOnly.VerifierRuns) != 2 {
		t.Fatalf("verification did not bind the exact two-record source/evidence cohort: %#v", readOnly)
	}
	if len(readOnly.Assurance.Evaluation.Roots) != 1 || readOnly.Assurance.Evaluation.Roots[0].Outcome != records.OutcomePassed {
		t.Fatalf("composed root did not pass with passing fixed checks and verifier outcomes: %#v", readOnly.Assurance)
	}
	for _, run := range readOnly.VerifierRuns {
		if run.ResultID == "" || run.Receipt.RunID != run.RunID || run.Receipt.InputDigest != run.InputDigest || run.Receipt.ConfigDigest != run.ConfigFingerprint || !strings.Contains(resultByID(readOnly.Results, run.ResultID).Reason, run.ReceiptDigest) {
			t.Fatalf("result does not bind its actual verifier receipt: %#v", run)
		}
	}
	state, err := store.Read()
	if err != nil || state.Head != before.Head || len(state.Verifications) != 0 {
		t.Fatalf("write=false changed verification ledger: head=%s verifications=%d err=%v", state.Head, len(state.Verifications), err)
	}
	if got := countCanonicalVerifierInvocations(t, marker); got != 2 {
		t.Fatalf("read-only verification invoked verifier %d times, want one per node", got)
	}
	if got := scopedTestGit(t, root, "rev-parse", "HEAD"); got != beforeHead {
		t.Fatalf("verification changed adopter repository HEAD from %s to %s", beforeHead, got)
	}

	writeReport, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, true)
	if err != nil || writeReport.Outcome != records.OutcomePassed {
		t.Fatalf("write verification: outcome=%q err=%v", writeReport.Outcome, err)
	}
	state, err = store.Read()
	if err != nil || len(state.Verifications) != 2 || state.ActiveSelection.Digest != before.ActiveSelection.Digest {
		t.Fatalf("write=true did not append exactly two results while retaining ownership: verifications=%d selection=%s err=%v", len(state.Verifications), state.ActiveSelection.Digest, err)
	}
	if got := scopedTestGit(t, root, "rev-parse", "HEAD"); got != beforeHead {
		t.Fatalf("ledger write changed adopter repository HEAD from %s to %s", beforeHead, got)
	}

}

func TestCanonicalControllerFreshSemanticFailureSuppliesOnlyLatestLeafRepairEvidence(t *testing.T) {
	if !runCanonicalControllerScenarioInChild(t) {
		return
	}
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.AuditAll = true
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomeFailed)
	failed, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, true)
	if err != nil || failed.Outcome != records.OutcomeFailed {
		t.Fatalf("persist completed semantic failure: outcome=%q err=%v", failed.Outcome, err)
	}
	failedByRecord := map[string]records.VerificationResult{}
	for _, result := range failed.Results {
		if !canonicalControllerResultSupportsRepair(result) {
			t.Fatalf("failed response lacked complete fixed-check and semantic-findings provenance: %#v", result)
		}
		failedByRecord[result.RecordID] = result
	}
	if len(failedByRecord) != 2 {
		t.Fatalf("verification did not produce both fixture results: %#v", failedByRecord)
	}

	proposal, err := ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("propose from fresh semantic failure: %v", err)
	}
	var dotnet CanonicalScopedProposal
	for _, candidate := range proposal.Plan.Proposals {
		if candidate.Module.Name == "markitect-dotnet" {
			dotnet = candidate
		}
		if candidate.Module.Name == "markitect-markdown" && candidate.Task != nil {
			t.Fatal("failed child was converted into a parent repair task")
		}
	}
	if dotnet.Task == nil || dotnet.Task.Repair == nil || dotnet.Decision != "work" {
		t.Fatalf("fresh failed leaf did not produce a bounded module-owned repair task: %#v", dotnet)
	}
	latestLeaf := failedByRecord[dotnet.Task.Repair.RecordID]
	if latestLeaf.ID != dotnet.Task.Repair.ResultID || len(dotnet.Task.Repair.Findings) != len(latestLeaf.SemanticFindings) {
		t.Fatalf("repair task was not bound to the exact latest failed leaf result: task=%#v result=%#v", dotnet.Task.Repair, latestLeaf)
	}
	for i, finding := range latestLeaf.SemanticFindings {
		if dotnet.Task.Repair.Findings[i].Subject != finding.Subject || dotnet.Task.Repair.Findings[i].Detail != finding.Detail {
			t.Fatalf("repair finding %d differs from persisted semantic observation", i)
		}
	}
	_, _, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var artifactPath string
	for _, record := range active {
		if record.ID == dotnet.Task.Repair.RecordID && len(record.Artifacts) > 0 {
			artifactPath = filepath.Join(root, filepath.FromSlash(record.Artifacts[0].Path))
			break
		}
	}
	if artifactPath == "" {
		t.Fatal("repair record had no exact owned artifact to use for the stale-evidence control")
	}
	artifactBefore, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, append(append([]byte(nil), artifactBefore...), []byte("// stale target evidence\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	staleProposal, staleErr := ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err := os.WriteFile(artifactPath, artifactBefore, 0644); err != nil {
		t.Fatalf("restore stale-evidence control artifact: %v", err)
	}
	if staleErr != nil {
		t.Fatalf("propose after selected target drift: %v", staleErr)
	}
	for _, candidate := range staleProposal.Plan.Proposals {
		if candidate.Task != nil && candidate.Task.Repair != nil {
			t.Fatalf("changed selected artifact retained stale semantic repair authority: %#v", candidate)
		}
	}

	// A newer incomplete result must supersede the failed result; the older
	// semantic finding cannot remain eligible through the append-only ledger.
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomeIncomplete)
	incomplete, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, true)
	if err != nil || incomplete.Outcome != records.OutcomeIncomplete {
		t.Fatalf("persist newer incomplete result: outcome=%q err=%v", incomplete.Outcome, err)
	}
	proposal, err = ProposeCanonicalController(root, sourceRevision, sourceRevision, "examples/canonical-projection/canonical.yaml", cfg)
	if err != nil {
		t.Fatalf("propose after newer incomplete result: %v", err)
	}
	for _, candidate := range proposal.Plan.Proposals {
		if candidate.Task != nil && candidate.Task.Repair != nil {
			t.Fatalf("older failed result masked the latest incomplete result: %#v", candidate)
		}
	}
}

func TestCanonicalControllerVerifierProtocolFailureAndIncompleteAreNotPassing(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	repo = canonicalControllerTestDirectory(t, repo, "repository")
	externalParent := canonicalControllerTestDirectory(t, filepath.Dir(repo), "external parent")
	external, err := os.MkdirTemp(externalParent, "canonical-verifier-outcome-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	privateLogs := filepath.Join(external, "private-logs")
	cfg := CanonicalControllerConfig{PrivateLogs: privateLogs, Verifier: CanonicalRunnerConfig{
		Command: os.Args[0], Args: []string{"-test.run=^TestCanonicalControllerVerifierProtocolProcess$"},
		Model: "protocol-test-double", ModelOptions: json.RawMessage(`{"protocolOnly":true}`), ProviderVersion: "test-protocol/1",
		TimeoutSeconds: 10, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
	}}
	record := controllerVerificationTestRecord(t, "core/v1:Projection:protocol", "core/v1:UseCase:orders")
	fingerprint, err := agentexec.Fingerprint(cfg.Verifier.agentConfig())
	if err != nil {
		t.Fatal(err)
	}
	verifier := records.VerifierIdentity{ID: "test-protocol", Version: "1", Digest: fingerprint}
	item := canonicalControllerPreparedVerification{
		scope: CanonicalAssuranceScope{ID: "scope", ProjectionID: record.ProjectionID}, record: record,
		context:        CanonicalAgentContext{RequestDigest: "sha256:" + strings.Repeat("1", 64), ScopeIDs: record.ScopeIDs},
		artifacts:      []agentexec.Artifact{{Path: "docs/protocol.md", Mode: "0644", Digest: sha256Prefix(sha256Hex([]byte("artifact"))), Content: []byte("artifact")}},
		evidenceDigest: sha256Prefix(sha256Hex([]byte("selected evidence"))),
		checks: []records.CheckResult{
			{ID: "fixed-check", Version: "fixed/v1", Digest: sha256Prefix(sha256Hex([]byte("fixed"))), Outcome: records.CheckPassed},
			{ID: canonicalControllerAgentCheckID, Version: canonicalControllerAgentCheckVersion, Digest: sha256Prefix(sha256Hex([]byte("receipt binding"))), Outcome: records.CheckIncomplete},
		},
	}
	item.request.Policies = nil
	node := assurance.NodeRunInput{Node: assurance.Node{ID: "scope"}}
	t.Run("preflight-error-does-not-fabricate-receipt-or-result", func(t *testing.T) {
		marker := filepath.Join(external, "preflight-runs.log")
		setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomeFailed)
		blockedLogs := filepath.Join(external, "private-logs-is-a-file")
		if err := os.WriteFile(blockedLogs, []byte("not a directory"), 0600); err != nil {
			t.Fatal(err)
		}
		blockedConfig := cfg
		blockedConfig.PrivateLogs = blockedLogs
		result, run, err := invokeCanonicalControllerVerifier(context.Background(), blockedConfig, item, node, verifier, fingerprint, strings.Repeat("a", 40))
		if err == nil || !strings.Contains(err.Error(), "private log directory") {
			t.Fatalf("preflight failure was not preserved: result=%#v run=%#v err=%v", result, run, err)
		}
		if run.Receipt.InputDigest != "" || run.RunID != "" || run.ResultID != "" || result.ID != "" {
			t.Fatalf("preflight failure fabricated verifier evidence: result=%#v run=%#v", result, run)
		}
		if got := countCanonicalVerifierInvocations(t, marker); got != 0 {
			t.Fatalf("preflight failure started verifier process %d times", got)
		}
	})
	for _, outcome := range []string{agentexec.OutcomeFailed, agentexec.OutcomeIncomplete} {
		t.Run(outcome, func(t *testing.T) {
			setCanonicalControllerVerifierActor(t, filepath.Join(external, "runs.log"), outcome)
			result, run, err := invokeCanonicalControllerVerifier(context.Background(), cfg, item, node, verifier, fingerprint, strings.Repeat("a", 40))
			if err != nil {
				t.Fatalf("valid %s protocol response returned invocation error: %v", outcome, err)
			}
			if result.Outcome != outcome || run.Outcome != outcome || canonicalControllerTestCheckOutcome(result, canonicalControllerAgentCheckID) != map[string]string{agentexec.OutcomeFailed: records.CheckFailed, agentexec.OutcomeIncomplete: records.CheckIncomplete}[outcome] {
				t.Fatalf("%s was upgraded by verifier integration: result=%#v run=%#v", outcome, result, run)
			}
			if outcome == agentexec.OutcomeFailed && len(result.SemanticFindings) == 0 {
				t.Fatalf("completed semantic failure did not persist its bounded negative observations: %#v", result.SemanticFindings)
			}
			if outcome != agentexec.OutcomeFailed && len(result.SemanticFindings) != 0 {
				t.Fatalf("non-failed response persisted repair findings: %#v", result.SemanticFindings)
			}
		})
	}
	t.Run("partial-passed-observations", func(t *testing.T) {
		setCanonicalControllerVerifierActor(t, filepath.Join(external, "partial-runs.log"), agentexec.OutcomePassed)
		t.Setenv(canonicalControllerVerifierTestPartialEnv, "1")
		result, run, err := invokeCanonicalControllerVerifier(context.Background(), cfg, item, node, verifier, fingerprint, strings.Repeat("a", 40))
		if err != nil {
			t.Fatalf("partial protocol response invocation error: %v", err)
		}
		if result.Outcome != records.OutcomeIncomplete || run.Outcome != agentexec.OutcomeIncomplete || canonicalControllerTestCheckOutcome(result, canonicalControllerAgentCheckID) != records.CheckIncomplete {
			t.Fatalf("partial observations composed to a passing verification: result=%#v run=%#v", result, run)
		}
	})
}

func canonicalControllerTestCheckOutcome(result records.VerificationResult, id string) string {
	for _, check := range result.Checks {
		if check.ID == id {
			return check.Outcome
		}
	}
	return ""
}

func TestCanonicalControllerSemanticRepairEligibilityRejectsNonsemanticAndIncompleteEvidence(t *testing.T) {
	response := agentexec.Response{Outcome: agentexec.OutcomeFailed, VerifierObservations: []agentexec.Observation{
		{Subject: "scope/behavior", Outcome: agentexec.OutcomeFailed, Detail: "The represented behavior violates the declared boundary."},
		{Subject: "artifact/docs.md", Outcome: agentexec.OutcomePassed, Detail: "The artifact exists and is readable."},
	}}
	checks := []records.CheckResult{
		{ID: "fixed-check", Outcome: records.CheckPassed},
		{ID: canonicalControllerAgentCheckID, Outcome: records.CheckFailed},
	}
	got, notice := canonicalControllerSemanticFindings(response, nil, agentexec.OutcomeFailed, checks)
	if notice != "" || len(got) != 1 || got[0].Subject != "scope/behavior" || got[0].Detail != response.VerifierObservations[0].Detail {
		t.Fatalf("repair findings differ from exact negative observation: %#v", got)
	}
	for _, tc := range []struct {
		name       string
		response   agentexec.Response
		err        error
		runOutcome string
		checks     []records.CheckResult
	}{
		{"invocation-error", response, fmt.Errorf("provider invocation failed"), agentexec.OutcomeFailed, checks},
		{"incomplete-response", agentexec.Response{Outcome: agentexec.OutcomeIncomplete, VerifierObservations: response.VerifierObservations}, nil, agentexec.OutcomeIncomplete, checks},
		{"fixed-check-failed", response, nil, agentexec.OutcomeFailed, []records.CheckResult{{ID: "fixed-check", Outcome: records.CheckFailed}, checks[1]}},
		{"no-negative-observation", agentexec.Response{Outcome: agentexec.OutcomeFailed, VerifierObservations: []agentexec.Observation{{Subject: "scope/behavior", Outcome: agentexec.OutcomeIncomplete, Detail: "not complete"}}}, nil, agentexec.OutcomeFailed, checks},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if findings, _ := canonicalControllerSemanticFindings(tc.response, tc.err, tc.runOutcome, tc.checks); len(findings) != 0 {
				t.Fatalf("ineligible response supplied repair findings: %#v", findings)
			}
		})
	}
	oversized := agentexec.Response{Outcome: agentexec.OutcomeFailed, VerifierObservations: []agentexec.Observation{{Subject: "scope/behavior", Outcome: agentexec.OutcomeFailed, Detail: strings.Repeat("x", 2*1024+1)}}}
	if findings, notice := canonicalControllerSemanticFindings(oversized, nil, agentexec.OutcomeFailed, checks); len(findings) != 0 || notice == "" {
		t.Fatalf("oversized finding was silently truncated or left eligible: findings=%#v notice=%q", findings, notice)
	}
	tooMany := agentexec.Response{Outcome: agentexec.OutcomeFailed, VerifierObservations: make([]agentexec.Observation, agentexec.MaxVerifierObservations+1)}
	for i := range tooMany.VerifierObservations {
		tooMany.VerifierObservations[i] = agentexec.Observation{Subject: fmt.Sprintf("scope/%03d", i), Outcome: agentexec.OutcomeFailed, Detail: "bounded detail"}
	}
	if findings, notice := canonicalControllerSemanticFindings(tooMany, nil, agentexec.OutcomeFailed, checks); len(findings) != 0 || notice == "" {
		t.Fatalf("over-count findings were not explicitly ineligible: findings=%#v notice=%q", findings, notice)
	}

	result := records.VerificationResult{
		Outcome: records.OutcomeFailed, SemanticFindings: []records.VerificationFinding{{Subject: "scope/behavior", Detail: "failed"}},
		Checks: checks,
	}
	if !canonicalControllerResultSupportsRepair(result) {
		t.Fatal("failed result with all fixed checks passed was not repair-eligible")
	}
	result.Checks[1].Outcome = records.CheckIncomplete
	if canonicalControllerResultSupportsRepair(result) {
		t.Fatal("placeholder agent-check outcome was mistaken for a final semantic failure")
	}
	result.Checks[1].Outcome = records.CheckFailed
	result.Checks[0].Outcome = records.CheckIncomplete
	if canonicalControllerResultSupportsRepair(result) {
		t.Fatal("incomplete fixed check did not suppress repair")
	}
	if !canonicalControllerChildrenPassed(nil) || canonicalControllerChildrenPassed([]assurance.ChildRunInput{{Result: assurance.NodeResult{Outcome: records.OutcomeFailed}}}) {
		t.Fatal("repair eligibility ignored an incomplete/failed direct child")
	}
}

func TestCanonicalControllerVerifierBlocksSourceAndCheckDriftBeforeInvocation(t *testing.T) {
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)

	checkInput := cfg.CheckInputs[0]
	checkBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checkInput)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(checkInput)), append(checkBytes, []byte("// evidence drift\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", checkInput)
	scopedTestGit(t, root, "commit", "-m", "change selected checker input")
	checkDriftRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	if _, err := VerifyCanonicalController(context.Background(), root, sourceRevision, checkDriftRevision, "examples/canonical-projection/canonical.yaml", cfg, false); err == nil || !strings.Contains(err.Error(), "fixed verification input changed") {
		t.Fatalf("selected check-input drift was not rejected before invocation: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Verifier ran before check-input drift rejection: %v", err)
	}

	fixed, err := LoadSelectedCanonicalSource(root, sourceRevision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"docs/represented/index.md", "src/ControllerTest.cs", "src/ControllerTest.csproj"} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(output))); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	scopedTestGit(t, root, "checkout", "--detach", evidenceRevision)
	definition := fixed.Config.Definitions[0]
	definitionPath := filepath.Join(root, filepath.FromSlash(definition))
	definitionBytes, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(definitionPath, append(definitionBytes, []byte("\n# source drift\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", definition)
	scopedTestGit(t, root, "commit", "-m", "change canonical source input")
	sourceDriftRevision := scopedTestGit(t, root, "rev-parse", "HEAD")
	if _, err := VerifyCanonicalController(context.Background(), root, sourceDriftRevision, evidenceRevision, "examples/canonical-projection/canonical.yaml", cfg, false); err == nil || !strings.Contains(err.Error(), "canonical source changed") {
		t.Fatalf("canonical source drift was not rejected before invocation: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Verifier ran before canonical source drift rejection: %v", err)
	}
	store, state, _, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil || len(state.Verifications) != 0 {
		t.Fatalf("drift rejection wrote to the ledger: %v", err)
	}
}

func TestCanonicalControllerVerifierIncludesGrandchildEvidenceBytes(t *testing.T) {
	cfg := CanonicalControllerConfig{AssuranceRoots: []string{"top"}, AssuranceScopes: []CanonicalAssuranceScope{
		{ID: "top", ProjectionID: "projection/top", Children: []string{"middle"}, Checks: []authoring.Check{{Name: "fixed", Run: []string{"go", "version"}}}},
		{ID: "middle", ProjectionID: "projection/middle", Children: []string{"leaf"}, Checks: []authoring.Check{{Name: "fixed", Run: []string{"go", "version"}}}},
		{ID: "leaf", ProjectionID: "projection/leaf", Checks: []authoring.Check{{Name: "fixed", Run: []string{"go", "version"}}}},
	}}
	recordsByProjection := map[string]records.ProjectionRecord{
		"projection/top":    {ProjectionID: "projection/top", Artifacts: []records.Artifact{{Path: "docs/top.md"}}},
		"projection/middle": {ProjectionID: "projection/middle", Artifacts: []records.Artifact{{Path: "docs/middle.md"}}},
		"projection/leaf":   {ProjectionID: "projection/leaf", Artifacts: []records.Artifact{{Path: "docs/leaf.md"}}},
	}
	target := &snapshot.Snapshot{ID: strings.Repeat("a", 40), Files: map[string][]byte{
		"docs/top.md": []byte("top"), "docs/middle.md": []byte("middle"), "docs/leaf.md": []byte("grandchild implementation evidence"),
	}, Modes: map[string]string{"docs/top.md": "100644", "docs/middle.md": "100644", "docs/leaf.md": "100644"}}
	artifacts, err := canonicalControllerVerificationArtifacts(cfg, recordsByProjection["projection/top"], cfg.AssuranceScopes[0], recordsByProjection, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string][]byte{}
	for _, artifact := range artifacts {
		byPath[artifact.Path] = artifact.Content
	}
	if !bytes.Equal(byPath["docs/leaf.md"], []byte("grandchild implementation evidence")) || len(byPath) != 3 {
		t.Fatalf("top verifier did not receive exact descendant artifact bytes: %#v", byPath)
	}
}

func TestCanonicalControllerThreeLevelCompositionRetainsTopFixedFailure(t *testing.T) {
	verifier := records.VerifierIdentity{ID: "test-protocol", Version: "1", Digest: sha256Prefix(sha256Hex([]byte("test-protocol")))}
	checks := map[string]records.CheckIdentity{}
	for _, id := range []string{"leaf-fixed", "middle-fixed", "top-fixed"} {
		checks[id] = records.CheckIdentity{ID: id, Version: "fixed-command-input/v1", Digest: sha256Prefix(sha256Hex([]byte(id)))}
	}
	projections := []records.ProjectionRecord{
		controllerVerificationTestRecord(t, "core/v1:Projection:leaf", "core/v1:UseCase:orders"),
		controllerVerificationTestRecord(t, "core/v1:Projection:middle", "core/v1:UseCase:orders"),
		controllerVerificationTestRecord(t, "core/v1:Projection:top", "core/v1:UseCase:orders"),
	}
	ids := []string{"leaf", "middle", "top"}
	byID := map[string]records.ProjectionRecord{}
	graph := assurance.Input{RootIDs: []string{"top"}, Nodes: []assurance.Node{}}
	current := map[string]records.Freshness{}
	for i, id := range ids {
		record := projections[i]
		byID[id] = record
		checkID := checks[id+"-fixed"]
		children := []string{}
		if id == "top" {
			children = []string{"middle"}
		} else if id == "middle" {
			children = []string{"leaf"}
		}
		graph.Nodes = append(graph.Nodes, assurance.Node{ID: id, ScopeIDs: append([]string(nil), record.ScopeIDs...), Children: children, RequiredChecks: []records.CheckIdentity{checkID}})
		current[id] = records.Freshness{RecordID: record.ID, Revision: record.Revision, ModelDigest: record.ModelDigest, TargetSnapshotDigest: record.TargetSnapshotDigest, Verifier: verifier, Checks: []records.CheckIdentity{checkID}}
	}
	results := map[string]records.VerificationResult{}
	report, err := assurance.Execute(context.Background(), assurance.RunInput{Graph: graph, Current: current}, func(_ context.Context, node assurance.NodeRunInput) (assurance.NodeRunOutput, error) {
		outcome, checkOutcome := records.OutcomePassed, records.CheckPassed
		if node.Node.ID == "top" {
			outcome, checkOutcome = records.OutcomeFailed, records.CheckFailed
		}
		check := checks[node.Node.ID+"-fixed"]
		result, err := records.NewVerificationResult(records.VerificationResult{
			RecordID: byID[node.Node.ID].ID, Revision: byID[node.Node.ID].Revision,
			ModelDigest: byID[node.Node.ID].ModelDigest, TargetSnapshotDigest: byID[node.Node.ID].TargetSnapshotDigest,
			Verifier: verifier, Outcome: outcome, Reason: "fixed-check fixture",
			Checks: []records.CheckResult{{ID: check.ID, Version: check.Version, Digest: check.Digest, Outcome: checkOutcome}},
		})
		if err != nil {
			return assurance.NodeRunOutput{}, err
		}
		results[node.Node.ID] = result
		return assurance.NodeRunOutput{NodeID: node.Node.ID, Disposition: assurance.RunCompleted, Record: byID[node.Node.ID], Result: result}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if results["leaf"].Outcome != records.OutcomePassed || results["middle"].Outcome != records.OutcomePassed || results["top"].Outcome != records.OutcomeFailed {
		t.Fatalf("fixed-check fixture outcomes were not leaf+middle PASS and top FAIL: %#v", results)
	}
	if got := canonicalControllerComposedOutcome(report.Evaluation.Roots); got != records.OutcomeFailed {
		t.Fatalf("passing leaf and middle hid top fixed-check failure: %q", got)
	}
}

func canonicalControllerVerificationFixture(t *testing.T) (root, sourceRevision, evidenceRevision string, cfg CanonicalControllerConfig, marker string) {
	t.Helper()
	root, _, cfg = canonicalControllerFixture(t)
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	repo = canonicalControllerTestDirectory(t, repo, "repository")
	externalParent := canonicalControllerTestDirectory(t, filepath.Dir(repo), "external parent")
	external, err := os.MkdirTemp(externalParent, "canonical-verifier-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(external) })
	cfg.RecordStore = filepath.Join(external, "ledger")
	cfg.PrivateLogs = filepath.Join(external, "private-logs")
	checkPath := "examples/canonical-projection/evidence/check.go"
	checkSource, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(checkPath)))
	if err != nil {
		t.Fatal(err)
	}
	checker := strings.NewReplacer(
		"src/Commerce/CreateOrderHandler.cs", "examples/canonical-projection/evidence/fixture/CreateOrderHandler.cs",
		"src/Commerce/EffectAxis.cs", "examples/canonical-projection/evidence/fixture/EffectAxis.cs",
		"src/Commerce/Commerce.csproj", "examples/canonical-projection/evidence/fixture/Commerce.csproj",
	).Replace(string(checkSource))
	scopedTestWrite(t, root, checkPath, checker)
	checkHandler := "examples/canonical-projection/evidence/fixture/CreateOrderHandler.cs"
	checkEffects := "examples/canonical-projection/evidence/fixture/EffectAxis.cs"
	checkProject := "examples/canonical-projection/evidence/fixture/Commerce.csproj"
	scopedTestWrite(t, root, checkHandler, "namespace Commerce; public sealed class CreateOrderHandler {}\n")
	scopedTestWrite(t, root, checkEffects, "namespace Commerce; public sealed record EffectAxis { public string Boundary { get; init; } = \"application\"; }\n")
	scopedTestWrite(t, root, checkProject, "<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>\n")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "add exact verifier check inputs")
	sourceRevision = scopedTestGit(t, root, "rev-parse", "HEAD")
	cfg.CheckInputs = []string{checkPath, checkHandler, checkEffects, checkProject}
	run := canonicalControllerExecute(t, root, sourceRevision, cfg)
	applied, err := canonicalControllerApply(t, root, cfg, run)
	if err != nil {
		t.Fatalf("prepare exact verifier evidence fixture: %v", err)
	}
	evidenceRevision = applied.EvidenceRevision
	if evidenceRevision == "" || evidenceRevision == sourceRevision {
		t.Fatalf("fixture did not create a separate evidence commit: %s", evidenceRevision)
	}
	store, _, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || store == nil || len(active) != 2 {
		t.Fatalf("fixture active selection: %d %v", len(active), err)
	}
	fixed, err := LoadSelectedCanonicalSource(root, sourceRevision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed.Config.Checks) != 1 || fixed.Config.Checks[0].Name != "canonical-projection-fixture" {
		t.Fatalf("unexpected canonical fixed checker definitions: %#v", fixed.Config.Checks)
	}
	var dotnetID, markdownID string
	for _, record := range active {
		switch record.Module.Name {
		case "markitect-dotnet":
			dotnetID = record.ProjectionID
		case "markitect-markdown":
			markdownID = record.ProjectionID
		}
	}
	if dotnetID == "" || markdownID == "" {
		t.Fatalf("fixture did not materialize the expected leaf and parent records: %#v", active)
	}
	checks := append([]authoring.Check(nil), fixed.Config.Checks...)
	cfg.AssuranceRoots = []string{"parent-markdown"}
	cfg.AssuranceScopes = []CanonicalAssuranceScope{
		{ID: "parent-markdown", ProjectionID: markdownID, Children: []string{"leaf-dotnet"}, Checks: checks},
		{ID: "leaf-dotnet", ProjectionID: dotnetID, Checks: checks},
	}
	cfg.Verifier.Args = []string{"-test.run=^TestCanonicalControllerVerifierProtocolProcess$"}
	marker = filepath.Join(filepath.Dir(cfg.RecordStore), "verifier-invocations.log")
	return root, sourceRevision, evidenceRevision, cfg, marker
}

func canonicalControllerTestDirectory(t *testing.T, path, description string) string {
	t.Helper()
	canonical, err := canonicalUserPath(path)
	if err != nil {
		t.Fatalf("canonicalize verifier test %s: %v", description, err)
	}
	canonical, err = realDirectory(canonical)
	if err != nil {
		t.Fatalf("validate verifier test %s: %v", description, err)
	}
	return canonical
}

func setCanonicalControllerVerifierActor(t *testing.T, marker, outcome string) {
	t.Helper()
	t.Setenv(canonicalControllerVerifierTestActorEnv, "1")
	t.Setenv(canonicalControllerVerifierTestOutcomeEnv, outcome)
	t.Setenv(canonicalControllerVerifierTestMarkerEnv, marker)
}

func countCanonicalVerifierInvocations(t *testing.T, path string) int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return count
}

func resultByID(results []records.VerificationResult, id string) records.VerificationResult {
	for _, result := range results {
		if result.ID == id {
			return result
		}
	}
	return records.VerificationResult{}
}
