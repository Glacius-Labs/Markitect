package assurance

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const testRevisionA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testRevisionB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func testDigest(ch string) string { return "sha256:" + strings.Repeat(ch, 64) }

func testChecks(ids ...string) []records.CheckIdentity {
	checks := make([]records.CheckIdentity, 0, len(ids))
	for i, id := range ids {
		checks = append(checks, records.CheckIdentity{ID: id, Version: "1", Digest: testDigest(string(rune('a' + i)))})
	}
	return checks
}

func testRecord(t *testing.T, scopes []string, state string) records.ProjectionRecord {
	t.Helper()
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: testRevisionA, ModelDigest: testDigest("a"), PlanDigest: testDigest("b"),
		InputSnapshotDigest: testDigest("c"), RequestDigest: testDigest("d"),
		Module:       records.ModuleIdentity{Name: "module", Version: "1", Digest: testDigest("e")},
		ProjectionID: "projection", Projector: records.ProjectorIdentity{ID: "projector", Version: "1"},
		ScopeIDs:  scopes,
		Artifacts: []records.Artifact{{Path: "out/result.txt", Digest: testDigest("f"), Mode: "100644", Change: records.ChangeCreated}},
		State:     state,
	})
	if err != nil {
		t.Fatalf("create projection record: %v", err)
	}
	return record
}

func testResult(t *testing.T, record records.ProjectionRecord, checks []records.CheckIdentity, outcome string) records.VerificationResult {
	t.Helper()
	checkResults := make([]records.CheckResult, 0, len(checks))
	for _, check := range checks {
		checkOutcome := records.CheckPassed
		switch outcome {
		case OutcomeFailed:
			checkOutcome = records.CheckFailed
		case OutcomeIncomplete, OutcomeEscalated:
			checkOutcome = records.CheckIncomplete
		}
		checkResults = append(checkResults, records.CheckResult{
			ID: check.ID, Version: check.Version, Digest: check.Digest, Outcome: checkOutcome,
		})
	}
	result, err := records.NewVerificationResult(records.VerificationResult{
		RecordID: record.ID, Revision: record.Revision, ModelDigest: record.ModelDigest,
		TargetSnapshotDigest: record.TargetSnapshotDigest,
		Verifier:             records.VerifierIdentity{ID: "verifier", Version: "1", Digest: testDigest("7")},
		Checks:               checkResults, Outcome: outcome, Reason: "supplied test result",
	})
	if err != nil {
		t.Fatalf("create verification result: %v", err)
	}
	return result
}

func testEvidence(t *testing.T, nodeID string, scopes []string, checks []records.CheckIdentity, outcome string) Evidence {
	t.Helper()
	record := testRecord(t, scopes, records.StateMaterializedUnverified)
	result := testResult(t, record, checks, outcome)
	return Evidence{
		NodeID: nodeID, Record: record, Result: result,
		Current: records.Freshness{
			Revision: record.Revision, ModelDigest: record.ModelDigest, RecordID: record.ID,
			TargetSnapshotDigest: record.TargetSnapshotDigest,
			Verifier:             result.Verifier, Checks: append([]records.CheckIdentity(nil), checks...),
		},
	}
}

func findNode(t *testing.T, report Report, nodeID string) NodeResult {
	t.Helper()
	for _, node := range report.Nodes {
		if node.NodeID == nodeID {
			return node
		}
	}
	t.Fatalf("node %q not found", nodeID)
	return NodeResult{}
}

func TestParentFailureRemainsFailureWhenAllChildrenPass(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parent := testEvidence(t, "parent", []string{"app"}, parentChecks, OutcomeFailed)
	child := testEvidence(t, "child", []string{"app"}, childChecks, OutcomePassed)

	report, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		},
		Evidence: []Evidence{child, parent},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := findNode(t, report, "parent")
	if got.Outcome != OutcomeFailed || got.ResultID != parent.Result.ID {
		t.Fatalf("parent outcome = %q, result = %q; want failed from own result %q", got.Outcome, got.ResultID, parent.Result.ID)
	}
	if findNode(t, report, "child").Outcome != OutcomePassed {
		t.Fatal("child should pass independently")
	}
}

func TestChildFailurePropagatesWithExactEvidenceIDs(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parent := testEvidence(t, "parent", []string{"app"}, parentChecks, OutcomePassed)
	child := testEvidence(t, "child", []string{"app"}, childChecks, OutcomeFailed)
	report, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		},
		Evidence: []Evidence{parent, child},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := findNode(t, report, "parent")
	if got.Outcome != OutcomeFailed {
		t.Fatalf("parent outcome = %q, want failed", got.Outcome)
	}
	found := false
	for _, cause := range got.Causes {
		if cause.ChildID == "child" && cause.ResultID == child.Result.ID && cause.RecordID == child.Record.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("parent causes do not identify exact child evidence: %#v", got.Causes)
	}
}

func TestMissingParentEvidenceIsIncompleteWhenChildrenPass(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	child := testEvidence(t, "child", []string{"app"}, childChecks, OutcomePassed)
	report, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		},
		Evidence: []Evidence{child},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := findNode(t, report, "parent")
	if got.Outcome != OutcomeIncomplete || len(got.Causes) == 0 || got.Causes[0].Code != "missing-evidence" {
		t.Fatalf("parent = %#v, want incomplete with missing-evidence cause", got)
	}
}

func TestMissingRequiredParentCheckResultIsIncomplete(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parent := testEvidence(t, "parent", []string{"app"}, parentChecks, OutcomeIncomplete)
	parent.Result.Checks = nil
	var err error
	parent.Result, err = records.NewVerificationResult(parent.Result)
	if err != nil {
		t.Fatal(err)
	}
	child := testEvidence(t, "child", []string{"app"}, childChecks, OutcomePassed)
	report, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		},
		Evidence: []Evidence{child, parent},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := findNode(t, report, "parent")
	if got.Outcome != OutcomeIncomplete {
		t.Fatalf("parent outcome = %q, want incomplete", got.Outcome)
	}
	if len(got.Causes) == 0 || got.Causes[0].ResultID != parent.Result.ID {
		t.Fatalf("missing exact result cause: %#v", got.Causes)
	}
}

func TestStaleChildResultCannotPassComposition(t *testing.T) {
	parentChecks, childChecks := testChecks("compose"), testChecks("unit")
	parent := testEvidence(t, "parent", []string{"app"}, parentChecks, OutcomePassed)
	child := testEvidence(t, "child", []string{"app"}, childChecks, OutcomePassed)
	child.Current.Revision = testRevisionB
	report, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: parentChecks},
			{ID: "child", ScopeIDs: []string{"app"}, RequiredChecks: childChecks},
		},
		Evidence: []Evidence{parent, child},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := findNode(t, report, "child"); got.Outcome != OutcomeIncomplete {
		t.Fatalf("stale child outcome = %q, want incomplete", got.Outcome)
	}
	if got := findNode(t, report, "parent"); got.Outcome != OutcomeIncomplete {
		t.Fatalf("parent outcome = %q, want incomplete", got.Outcome)
	}
}

func TestRejectsScopeEscape(t *testing.T) {
	checks := testChecks("check")
	_, err := Evaluate(Input{
		RootIDs: []string{"parent"},
		Nodes: []Node{
			{ID: "parent", ScopeIDs: []string{"app"}, Children: []string{"child"}, RequiredChecks: checks},
			{ID: "child", ScopeIDs: []string{"other"}, RequiredChecks: checks},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "escapes parent") {
		t.Fatalf("error = %v, want scope escape rejection", err)
	}
}

func TestRejectsCycles(t *testing.T) {
	checks := testChecks("check")
	_, err := Evaluate(Input{
		RootIDs: []string{"a"},
		Nodes: []Node{
			{ID: "a", ScopeIDs: []string{"app"}, Children: []string{"b"}, RequiredChecks: checks},
			{ID: "b", ScopeIDs: []string{"app"}, Children: []string{"a"}, RequiredChecks: checks},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %v, want cycle rejection", err)
	}
}

func TestRejectsTraversalDeeperThanLimit(t *testing.T) {
	checks := testChecks("check")
	nodes := make([]Node, MaxDepth+1)
	for i := range nodes {
		nodes[i] = Node{ID: string(rune('a' + i)), ScopeIDs: []string{"app"}, RequiredChecks: checks}
		if i+1 < len(nodes) {
			nodes[i].Children = []string{string(rune('a' + i + 1))}
		}
	}
	_, err := Evaluate(Input{RootIDs: []string{"a"}, Nodes: nodes})
	if err == nil || !strings.Contains(err.Error(), "maximum DAG depth") {
		t.Fatalf("error = %v, want depth limit rejection", err)
	}
}

func TestUnreachableNodesDoNotJoinRequestedRoot(t *testing.T) {
	checks := testChecks("check")
	root := testEvidence(t, "root", []string{"app"}, checks, OutcomePassed)
	unrelated := testEvidence(t, "unrelated", []string{"app"}, checks, OutcomeFailed)
	report, err := Evaluate(Input{
		RootIDs: []string{"root"},
		Nodes: []Node{
			{ID: "root", ScopeIDs: []string{"app"}, RequiredChecks: checks},
			{ID: "unrelated", ScopeIDs: []string{"app"}, RequiredChecks: checks},
		},
		Evidence: []Evidence{unrelated, root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Nodes) != 1 || report.Nodes[0].NodeID != "root" || report.Roots[0].Outcome != OutcomePassed {
		t.Fatalf("unreachable node affected requested root: %#v", report)
	}
}

func TestRejectsEmptyRequiredChecks(t *testing.T) {
	_, err := Evaluate(Input{
		RootIDs: []string{"root"},
		Nodes:   []Node{{ID: "root", ScopeIDs: []string{"app"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "required check count") {
		t.Fatalf("error = %v, want empty required checks rejection", err)
	}
}

func TestVerificationEscalationPropagatesAsEscalated(t *testing.T) {
	checks := testChecks("review")
	evidence := testEvidence(t, "scope", []string{"app"}, checks, OutcomeEscalated)
	report, err := Evaluate(Input{
		RootIDs:  []string{"scope"},
		Nodes:    []Node{{ID: "scope", ScopeIDs: []string{"app"}, RequiredChecks: checks}},
		Evidence: []Evidence{evidence},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := findNode(t, report, "scope")
	if got.Outcome != OutcomeEscalated || got.ResultID != evidence.Result.ID {
		t.Fatalf("node outcome = %q, result = %q; want escalated from supplied result %q", got.Outcome, got.ResultID, evidence.Result.ID)
	}
}
func TestReportOrderIsDeterministic(t *testing.T) {
	checkA, checkB := testChecks("a"), testChecks("b")
	evidenceA := testEvidence(t, "a", []string{"app"}, checkA, OutcomePassed)
	evidenceB := testEvidence(t, "b", []string{"app"}, checkB, OutcomePassed)
	nodes := []Node{
		{ID: "root", ScopeIDs: []string{"app"}, Children: []string{"b", "a"}, RequiredChecks: testChecks("root")},
		{ID: "b", ScopeIDs: []string{"app"}, RequiredChecks: checkB},
		{ID: "a", ScopeIDs: []string{"app"}, RequiredChecks: checkA},
	}
	root := testEvidence(t, "root", []string{"app"}, testChecks("root"), OutcomePassed)
	first, err := Evaluate(Input{RootIDs: []string{"root"}, Nodes: nodes, Evidence: []Evidence{root, evidenceB, evidenceA}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate(Input{RootIDs: []string{"root"}, Nodes: []Node{nodes[2], nodes[0], nodes[1]}, Evidence: []Evidence{evidenceA, root, evidenceB}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("reports differ by input order: first=%#v second=%#v", first, second)
	}
}
