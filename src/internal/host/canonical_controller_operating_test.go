package host

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

const canonicalOperatingRepairActorEnv = "MARKITECT_CANONICAL_OPERATING_REPAIR_ACTOR"
const canonicalOperatingRepairInvocationEnv = "MARKITECT_CANONICAL_OPERATING_REPAIR_INVOCATION"

// Test helper process: the output is deterministic protocol input, not semantic evidence.
func TestCanonicalControllerOperatingRepairActor(t *testing.T) {
	if os.Getenv(canonicalOperatingRepairActorEnv) != "1" {
		return
	}
	var invocation agentexec.Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&invocation); err != nil {
		os.Exit(61)
	}
	if marker := os.Getenv(canonicalControllerMarkerEnv); marker != "" {
		if err := os.WriteFile(marker, []byte("invoked\n"), 0600); err != nil {
			os.Exit(63)
		}
	}
	if invocationPath := os.Getenv(canonicalOperatingRepairInvocationEnv); invocationPath != "" {
		data, err := json.Marshal(invocation)
		if err != nil {
			os.Exit(64)
		}
		if err := os.WriteFile(invocationPath, data, 0600); err != nil {
			os.Exit(65)
		}
	}
	response := agentexec.Response{
		APIVersion: agentexec.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: invocation.Request.Role, InputDigest: invocation.InputDigest,
		Outcome: agentexec.OutcomeProposed, EvidenceRefs: []string{},
		VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{},
		CandidateFiles: []agentexec.CandidateFile{
			{Path: "src/ControllerTest.cs", Mode: "0644", Content: "namespace ControllerLifecycle; // operating-model repair fixture\npublic sealed class ControllerTest {}\n"},
			{Path: "src/ControllerTest.csproj", Mode: "0644", Content: "<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup></Project>\n"},
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(62)
	}
	os.Exit(0)
}

func TestCanonicalControllerOperatingFailureRepairClosure(t *testing.T) {
	if !runCanonicalControllerScenarioInChild(t) {
		return
	}
	root, sourceRevision, evidenceRevision, cfg, marker := canonicalControllerVerificationFixture(t)
	cfg.Executor.Args = []string{"-test.run=^TestCanonicalControllerOperatingRepairActor$"}
	t.Setenv(canonicalOperatingRepairActorEnv, "1")
	cfg.AuditAll = true
	const configPath = "examples/canonical-projection/canonical.yaml"

	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	verified, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, configPath, cfg, true)
	if err != nil || verified.Outcome != "passed" {
		t.Fatalf("verify materialized projections: outcome=%q err=%v", verified.Outcome, err)
	}

	audit, err := AuditCanonicalController(root, sourceRevision, sourceRevision, configPath, cfg)
	if err != nil {
		t.Fatalf("audit verified projections: %v", err)
	}
	if audit.Status != "complete" {
		t.Fatalf("freshly verified declared scope did not close: status=%q findings=%+v", audit.Status, audit.Findings)
	}
	if len(audit.Projections) != 2 {
		t.Fatalf("audit projections=%d, want the two declared canonical projections: %+v", len(audit.Projections), audit.Projections)
	}
	for _, projection := range audit.Projections {
		if len(projection.ActiveRecordIDs) != 1 || len(projection.FreshVerificationResultIDs) != 1 || projection.Outcome != "passed" {
			t.Errorf("projection %s lacks exactly one active record and fresh passing verification: %+v", projection.ProjectionID, projection)
		}
	}

	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomeFailed)
	failed, err := VerifyCanonicalController(context.Background(), root, sourceRevision, evidenceRevision, configPath, cfg, true)
	if err != nil || failed.Outcome != agentexec.OutcomeFailed {
		t.Fatalf("persist semantic failure: outcome=%q err=%v", failed.Outcome, err)
	}
	failedAudit, err := AuditCanonicalController(root, sourceRevision, sourceRevision, configPath, cfg)
	if err != nil || failedAudit.Status != "incomplete" {
		t.Fatalf("failed verification did not keep audit open: status=%q err=%v", failedAudit.Status, err)
	}
	proposal, err := ProposeCanonicalController(root, sourceRevision, sourceRevision, configPath, cfg)
	if err != nil {
		t.Fatalf("propose repair from latest semantic failure: %v", err)
	}
	var leaf CanonicalScopedProposal
	for _, candidate := range proposal.Plan.Proposals {
		switch candidate.Module.Name {
		case "markitect-dotnet":
			leaf = candidate
		case "markitect-markdown":
			if candidate.Task != nil {
				t.Fatalf("parent projection unexpectedly received repair work: %+v", candidate.Task)
			}
		}
	}
	if leaf.Task == nil || leaf.Task.Repair == nil || leaf.Task.Repair.RecordID == "" || leaf.Task.Repair.ResultID == "" {
		t.Fatalf("latest leaf failure did not authorize a bounded repair task: %+v", leaf)
	}
	latestLeaf := resultByID(failed.Results, leaf.Task.Repair.ResultID)
	if latestLeaf.ID == "" || latestLeaf.RecordID != leaf.Task.Repair.RecordID || latestLeaf.Outcome != agentexec.OutcomeFailed {
		t.Fatalf("repair task is not bound to the persisted failed leaf result: task=%+v result=%+v", leaf.Task.Repair, latestLeaf)
	}

	store, _, active, err := readCanonicalControllerLedger(root, cfg)
	if err != nil || len(active) != 2 {
		t.Fatalf("read active projections before repair: records=%d err=%v", len(active), err)
	}
	beforeArtifacts := map[string][]byte{}
	parentArtifacts := map[string][]byte{}
	leafArtifactPaths := map[string]bool{}
	for _, record := range active {
		for _, artifact := range record.Artifacts {
			path := filepath.Join(root, filepath.FromSlash(artifact.Path))
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read pre-repair artifact %s: %v", artifact.Path, readErr)
			}
			beforeArtifacts[artifact.Path] = data
			if record.Module.Name == "markitect-markdown" {
				parentArtifacts[artifact.Path] = data
			}
			if record.Module.Name == "markitect-dotnet" {
				leafArtifactPaths[artifact.Path] = true
			}
		}
	}
	invocationPath := filepath.Join(filepath.Dir(cfg.RecordStore), "repair-executor-invocation.json")
	t.Setenv(canonicalOperatingRepairInvocationEnv, invocationPath)
	repairRun := canonicalControllerExecute(t, root, sourceRevision, cfg)
	if len(repairRun.Work) != 1 {
		t.Fatalf("repair execution work items=%d, want the single failed leaf: %+v", len(repairRun.Work), repairRun.Work)
	}
	invocationBytes, err := os.ReadFile(invocationPath)
	if err != nil {
		t.Fatalf("read actual repair Executor invocation: %v", err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(invocationBytes, &invocation); err != nil {
		t.Fatalf("decode actual repair Executor invocation: %v", err)
	}
	if invocation.Request.ProjectionID != leaf.ProjectionID {
		t.Fatalf("captured repair invocation Projection=%s, want %s", invocation.Request.ProjectionID, leaf.ProjectionID)
	}
	var executorContext map[string]json.RawMessage
	if err := json.Unmarshal(invocation.Request.Context, &executorContext); err != nil {
		t.Fatalf("decode actual repair Executor context: %v", err)
	}
	var ownedPaths []string
	if err := json.Unmarshal(executorContext["existingOwnedArtifactPaths"], &ownedPaths); err != nil {
		t.Fatalf("decode current-scope owned paths from Executor context: %v", err)
	}
	expectedOwnedPaths := make([]string, 0, len(leaf.Task.ExistingOwnedArtifacts))
	for _, artifact := range leaf.Task.ExistingOwnedArtifacts {
		expectedOwnedPaths = append(expectedOwnedPaths, artifact.Path)
	}
	sort.Strings(expectedOwnedPaths)
	if len(ownedPaths) != len(expectedOwnedPaths) || len(ownedPaths) == 0 {
		t.Fatalf("Executor context owned paths=%v, want nonempty task-owned paths %v", ownedPaths, expectedOwnedPaths)
	}
	for i, path := range expectedOwnedPaths {
		if ownedPaths[i] != path {
			t.Fatalf("Executor context owned paths=%v, want sorted exact paths %v", ownedPaths, expectedOwnedPaths)
		}
	}
	var constraints []string
	if err := json.Unmarshal(executorContext["constraints"], &constraints); err != nil {
		t.Fatalf("decode Executor task constraints: %v", err)
	}
	completeOwnedRepresentation := false
	for _, constraint := range constraints {
		if strings.Contains(constraint, "complete owned representation") && strings.Contains(constraint, "existing owned artifact path") && strings.Contains(constraint, "unchanged") {
			completeOwnedRepresentation = true
		}
	}
	if !completeOwnedRepresentation {
		t.Fatalf("Executor context omits the complete-owned-representation instruction: %v", constraints)
	}
	var dependencyPaths []string
	if err := json.Unmarshal(executorContext["readOnlyDependencyEvidence"], &dependencyPaths); err != nil {
		t.Fatalf("decode read-only dependency paths: %v", err)
	}
	for _, ownedPath := range ownedPaths {
		for _, dependencyPath := range dependencyPaths {
			if ownedPath == dependencyPath {
				t.Fatalf("Executor conflates current owned artifact %s with read-only dependency evidence", ownedPath)
			}
		}
	}
	repairJSON, ok := executorContext["repair"]
	if !ok {
		t.Fatalf("actual repair Executor context omitted repair evidence: %s", invocation.Request.Context)
	}
	var repairFields map[string]json.RawMessage
	if err := json.Unmarshal(repairJSON, &repairFields); err != nil {
		t.Fatalf("decode repair field names from Executor context: %v", err)
	}
	if len(repairFields) != 3 || repairFields["recordId"] == nil || repairFields["resultId"] == nil || repairFields["findings"] == nil {
		t.Fatalf("repair context does not use the stable camel-case wire keys: %s", repairJSON)
	}
	var findingFields []map[string]json.RawMessage
	if err := json.Unmarshal(repairFields["findings"], &findingFields); err != nil {
		t.Fatalf("decode finding field names from Executor context: %v", err)
	}
	if len(findingFields) != len(leaf.Task.Repair.Findings) {
		t.Fatalf("Executor context findings=%d, want %d", len(findingFields), len(leaf.Task.Repair.Findings))
	}
	for _, finding := range findingFields {
		if len(finding) != 2 || finding["subject"] == nil || finding["detail"] == nil {
			t.Fatalf("repair finding does not use stable camel-case wire keys: %+v", finding)
		}
	}
	var capturedRepair canonicalControllerExecutorRepair
	if err := json.Unmarshal(repairJSON, &capturedRepair); err != nil {
		t.Fatalf("decode typed repair evidence from Executor context: %v", err)
	}
	if capturedRepair.RecordID != leaf.Task.Repair.RecordID || capturedRepair.ResultID != leaf.Task.Repair.ResultID || len(capturedRepair.Findings) != len(leaf.Task.Repair.Findings) {
		t.Fatalf("Executor repair evidence differs from the exact current failure: got=%+v want=%+v", capturedRepair, leaf.Task.Repair)
	}
	for i, finding := range leaf.Task.Repair.Findings {
		if capturedRepair.Findings[i].Subject != finding.Subject || capturedRepair.Findings[i].Detail != finding.Detail {
			t.Fatalf("Executor repair finding %d differs from the exact current failure: got=%+v want=%+v", i, capturedRepair.Findings[i], finding)
		}
	}
	applied, err := canonicalControllerApply(t, root, cfg, repairRun)
	if err != nil || applied.EvidenceRevision == "" {
		t.Fatalf("apply normal repair execution: evidenceRevision=%q err=%v", applied.EvidenceRevision, err)
	}
	leafChanged := false
	for path, before := range beforeArtifacts {
		after, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			t.Fatalf("read post-repair artifact %s: %v", path, readErr)
		}
		if _, isParent := parentArtifacts[path]; isParent && string(after) != string(before) {
			t.Fatalf("unaffected parent artifact changed during leaf repair: %s", path)
		}
		if leafArtifactPaths[path] && string(after) != string(before) {
			leafChanged = true
		}
	}
	if len(parentArtifacts) == 0 || len(leafArtifactPaths) == 0 {
		t.Fatalf("fixture did not identify both owned output sets: parent=%v leaf=%v", parentArtifacts, leafArtifactPaths)
	}
	if !leafChanged {
		t.Fatal("normal repair Execute/Apply did not change any leaf-owned artifact bytes")
	}

	setCanonicalControllerVerifierActor(t, marker, agentexec.OutcomePassed)
	repaired, err := VerifyCanonicalController(context.Background(), root, sourceRevision, applied.EvidenceRevision, configPath, cfg, true)
	if err != nil || repaired.Outcome != agentexec.OutcomePassed || len(repaired.Results) != 2 {
		t.Fatalf("verify repaired declared scope: outcome=%q results=%d err=%v", repaired.Outcome, len(repaired.Results), err)
	}
	closed, err := AuditCanonicalController(root, sourceRevision, sourceRevision, configPath, cfg)
	if err != nil || closed.Status != "complete" {
		t.Fatalf("freshly reverified repaired scope did not close: status=%q findings=%+v err=%v", closed.Status, closed.Findings, err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	retainedFailure := false
	for _, result := range state.Verifications {
		if result.ID == latestLeaf.ID && result.Outcome == agentexec.OutcomeFailed {
			retainedFailure = true
		}
	}
	if !retainedFailure {
		t.Fatal("repair discarded the historical failed verification result")
	}
}

func TestCanonicalControllerExecutorContextOmitsAbsentRepair(t *testing.T) {
	data, err := json.Marshal(canonicalControllerExecutorContext{})
	if err != nil {
		t.Fatal(err)
	}
	var contextEnvelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &contextEnvelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := contextEnvelope["repair"]; ok {
		t.Fatalf("ordinary Executor context serialized absent repair evidence: %s", data)
	}
}
