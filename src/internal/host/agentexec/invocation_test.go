package agentexec

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

func TestSharedInvocationBindsFreshIdentityAndClosedResponse(t *testing.T) {
	req := testRequest(RoleExecutor)
	first, wire, err := PrepareInvocation(req)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := PrepareInvocation(req)
	if err != nil {
		t.Fatal(err)
	}
	if first.RunID == second.RunID || first.Nonce == second.Nonce || first.InputDigest != second.InputDigest {
		t.Fatal("identity freshness or deterministic input binding failed")
	}
	var decoded Invocation
	if err := json.Unmarshal(wire, &decoded); err != nil || decoded.InputDigest != first.InputDigest {
		t.Fatalf("invalid wire: %v", err)
	}
	response := Response{APIVersion: APIVersion, RunID: first.RunID, Nonce: first.Nonce, Role: RoleExecutor, InputDigest: first.InputDigest, Outcome: OutcomeIncomplete, CandidateFiles: []CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []Observation{}, Uncertainty: []string{}}
	valid, _ := json.Marshal(response)
	if _, err := DecodeResponse(valid, first, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeResponse(valid, second, ""); err == nil {
		t.Fatal("cross invocation response accepted")
	}
	for _, invalid := range []string{
		strings.TrimSuffix(string(valid), "}") + `,"unknown":true}`,
		strings.TrimSuffix(string(valid), "}") + `,"RunID":"alias"}`,
		strings.Replace(string(valid), `"evidenceRefs":[]`, `"evidenceRefs":["invented"]`, 1),
		strings.Replace(string(valid), `"uncertainty":[]`, `"uncertainty":null`, 1),
	} {
		if _, err := DecodeResponse([]byte(invalid), first, ""); err == nil {
			t.Fatalf("invalid response accepted: %s", invalid)
		}
	}
	first.Request.Context = json.RawMessage(`{"changed":true}`)
	if _, err := DecodeResponse(valid, first, ""); err == nil {
		t.Fatal("mutated invocation accepted")
	}
}

func TestProcessRejectsOwnedWorkspaceBeforeExecutableOrFilesystemWork(t *testing.T) {
	result, err := Run(context.Background(), Config{Command: "not-a-real-command"}, testRequest(RoleExecutor), RunOptions{Workspace: &projectworkspace.Handle{ID: "owned"}})
	if err == nil || !strings.Contains(err.Error(), "does not support") || result.Receipt.RunID != "" {
		t.Fatalf("unsupported workspace was ignored: %+v %v", result, err)
	}
}

func TestLifecycleUnknownIsNotZeroObservedRequests(t *testing.T) {
	receipt := Receipt{Lifecycle: &Lifecycle{Provider: "codex-app-server", State: "unknown", Accounting: "unavailable"}}
	wire, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"startRequests":null`) || strings.Contains(string(wire), `"effective"`) {
		t.Fatalf("unknown metadata turned into observed zero/effective settings: %s", wire)
	}
}
