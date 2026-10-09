package projectrun

import (
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"testing"
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
