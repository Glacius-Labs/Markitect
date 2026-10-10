package projectrun

import (
	"encoding/json"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func TestRecoveryKeepsOriginalReceiptWhenInspectionFailsBeforeNewReceipt(t *testing.T) {
	prior := InvocationLog{InputDigest: "input", ReportID: "original", Outcome: "incomplete", CostMicros: 42, Receipt: agentexec.Receipt{RunID: "original", Lifecycle: &agentexec.Lifecycle{Accounting: "partial", StartRequests: []agentexec.RoleStartRequest{{RequestID: "root", State: "unknown"}, {RequestID: "child", State: "unknown"}}}}}
	updated := retainOriginalReceipt(prior, InvocationLog{InputDigest: "input"})
	if updated.Receipt.RunID != "original" || len(updated.Receipt.Lifecycle.StartRequests) != 2 || updated.CostMicros != 42 || updated.Outcome != "incomplete" {
		t.Fatalf("inspection failure erased observed evidence: %+v", updated)
	}
}

func TestSavedVerificationChecksRejectChangedExecutionPins(t *testing.T) {
	plan := PlanRecord{Checks: []CheckPlan{{ID: "check", Command: []string{"go", "test", "./..."}, ExecutablePath: "/selected/go", ExecutableDigest: "fixed", Required: true}}}
	original := CheckResult{ID: "check", Command: []string{"go", "test", "./..."}, ExecutablePath: "/selected/go", ExecutableDigest: "fixed", CandidateID: "candidate", Outcome: "passed"}
	if _, err := savedVerificationChecks(RunReport{Checks: []CheckResult{original}}, plan, "candidate"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"command", "path", "digest"} {
		changed := original
		switch kind {
		case "command":
			changed.Command = []string{"go", "version"}
		case "path":
			changed.ExecutablePath = "/other/go"
		case "digest":
			changed.ExecutableDigest = "other"
		}
		if _, err := savedVerificationChecks(RunReport{Checks: []CheckResult{changed}}, plan, "candidate"); err == nil {
			t.Fatalf("accepted changed %s pin", kind)
		}
	}
}

func TestOriginalNativeRecoveryBindingAcceptsLegacyRawLogDigestWithExactReceipt(t *testing.T) {
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: "revision",
		ModelDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ModulePin:   "module", ProjectionID: "project", Context: json.RawMessage(`{"z":1,"a":2}`)}
	rawDigest, err := digest(request)
	if err != nil {
		t.Fatal(err)
	}
	canonicalDigest, err := nativeRequestInputDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	if rawDigest == canonicalDigest {
		t.Fatal("fixture does not distinguish the legacy raw digest from the normalized receipt digest")
	}
	ledger := []InvocationLog{{TaskID: "task", Role: "reviewer", Phase: "review", InputDigest: rawDigest,
		Receipt: agentexec.Receipt{RunID: "original-run", InputDigest: canonicalDigest}}}
	binding, err := originalNativeRecoveryBinding("owner-run", ledger, "task", "reviewer", "review", request)
	if err != nil || binding.RunID != "original-run" || binding.InputDigest != canonicalDigest || binding.OwnerRunID != "owner-run" {
		t.Fatalf("legacy raw start log did not resolve through its exact normalized receipt: binding=%+v err=%v", binding, err)
	}
}

func TestOriginalInvocationIndexUsesExactReceiptAndRejectsAmbiguousStart(t *testing.T) {
	expected := InvocationLog{TaskID: "task", Role: "reviewer", Phase: "review", InputDigest: "raw-digest",
		Receipt: agentexec.Receipt{RunID: "run-2", InputDigest: "normalized-digest"}}
	rows := []InvocationLog{
		{TaskID: "task", Role: "reviewer", Phase: "review", InputDigest: "raw-digest", Receipt: agentexec.Receipt{RunID: "run-1"}},
		{TaskID: "task", Role: "reviewer", Phase: "review", InputDigest: "previous-digest", Receipt: agentexec.Receipt{RunID: "run-2"}},
	}
	index, err := originalInvocationIndex(rows, expected)
	if err != nil || index != 1 {
		t.Fatalf("exact receipt RunID did not select its durable row: index=%d err=%v", index, err)
	}
	rows = append(rows, rows[1])
	if _, err := originalInvocationIndex(rows, expected); err == nil {
		t.Fatal("duplicate exact receipt rows were accepted")
	}
	started := []InvocationLog{{TaskID: "task", Role: "reviewer", Phase: "review", InputDigest: "raw-digest"}}
	if index, err := originalInvocationIndex(started, expected); err != nil || index != 0 {
		t.Fatalf("receipt recovery did not reconcile its original raw-digest start row: index=%d err=%v", index, err)
	}
}
