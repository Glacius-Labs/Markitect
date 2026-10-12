package projectapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func TestApplicationApplyRejectsUnclassifiedPathAfterVerifiedPreflight(t *testing.T) {
	t.Parallel()
	root := deliveryFixture(t)
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatalf("remove generated document from the committed fixture: %v", err)
	}
	deliveryGit(t, root, "add", "-u", "README.md")
	deliveryGit(t, root, "commit", "-m", "remove generated document from fixture")
	setDeliveryProcessFixtureEnv(t, deliveryCandidateCheckEnv, "1")
	setDeliveryProcessFixtureEnv(t, deliveryFullVerifyCheckEnv, "1")
	operations := deliveryOperations()
	deliveryCreateAcknowledgedScope(t, operations, root)

	plan, err := operations.Plan(PlanOperation{Selection: Selection{Root: root}, Request: projectrun.PlanRequest{
		ExplorationID: "delivery", ScopeID: "orders-change", Operation: projectrun.OperationApply,
		Goal: "Update the orders implementation", Managers: deliveryManagers(), ExecuteAuthorized: true,
	}})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	run, err := operations.Run(context.Background(), RunOperation{Root: root, RunID: plan.ID})
	if err != nil || run.Status != projectrun.StatusIntegrated {
		t.Fatalf("Run: status=%s err=%v", run.Status, err)
	}
	document, err := operations.Document(DocumentOperation{Selection: Selection{Root: root}})
	if err != nil {
		t.Fatalf("render generated candidate document through Application: %v", err)
	}
	if got := run.Candidate.Files[document.Path]; got == "" || got != contentDigest(document.Content) {
		t.Fatalf("candidate document digest %q does not bind the exact generated document %q", got, contentDigest(document.Content))
	}
	candidatePath := fmt.Sprintf(".markitect/runs/%s/candidates/%s.json", run.ID, run.Candidate.ID)
	storedBytes, err := operations.ReadRecord(root, candidatePath)
	if err != nil {
		t.Fatalf("read persisted candidate through the public Application record path: %v", err)
	}
	var stored struct {
		Files map[string]struct {
			Path    string `json:"path"`
			Content []byte `json:"content"`
			Delete  bool   `json:"delete,omitempty"`
		} `json:"files"`
	}
	if err := json.Unmarshal(storedBytes, &stored); err != nil {
		t.Fatalf("decode persisted candidate record: %v", err)
	}
	storedDocument, ok := stored.Files[document.Path]
	if !ok || storedDocument.Delete || storedDocument.Path != document.Path || string(storedDocument.Content) != document.Content {
		t.Fatalf("persisted candidate omitted the exact Application-generated document: %+v", storedDocument)
	}
	if run.Candidate.Files[document.Path] != contentDigest(string(storedDocument.Content)) {
		t.Fatalf("persisted candidate document bytes do not match the public candidate digest: %q", run.Candidate.Files[document.Path])
	}
	verified, err := operations.Verify(context.Background(), RunOperation{Root: root, RunID: run.ID})
	if err != nil || verified.Status != projectrun.StatusVerified {
		t.Fatalf("Verify: status=%s err=%v", verified.Status, err)
	}
	preflight, err := operations.PreflightApply(PreflightOperation{Root: root, RunID: run.ID, CandidateID: run.Candidate.ID})
	if err != nil {
		t.Fatalf("PreflightApply: %v", err)
	}
	if preflight.VerificationDigest == "" || len(preflight.Paths) == 0 {
		t.Fatalf("verified preflight omitted its bindings or candidate paths: %+v", preflight)
	}

	beforeStatus, err := operations.Status(RunOperation{Root: root, RunID: run.ID})
	if err != nil || beforeStatus.Run.Status != projectrun.StatusVerified {
		t.Fatalf("pre-Apply status: report=%+v err=%v", beforeStatus, err)
	}
	planPath := ".markitect/runs/" + plan.ID + "/plan.json"
	statePath := fmt.Sprintf(".markitect/runs/%s/states/%08d.json", run.ID, beforeStatus.Run.Revision)
	planBytes, err := operations.ReadRecord(root, planPath)
	if err != nil {
		t.Fatalf("read persisted plan before rejected Apply: %v", err)
	}
	stateBytes, err := operations.ReadRecord(root, statePath)
	if err != nil {
		t.Fatalf("read persisted verified state before rejected Apply: %v", err)
	}
	sourceBefore := map[string][]byte{}
	for _, path := range []string{"src/orders/implementation.txt", "src/inventory/implementation.txt"} {
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sourceBefore[path] = content
	}

	const unclassifiedPath = "unclassified-after-preflight.txt"
	if err := os.WriteFile(filepath.Join(root, unclassifiedPath), []byte("new ordinary source after preflight\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := projectrun.ApplyRequest{RunID: run.ID, PlanID: plan.ID, CandidateID: run.Candidate.ID,
		ExpectedVerificationDigest: preflight.VerificationDigest, TargetBranch: preflight.TargetBranch,
		ExpectedHead: preflight.ExpectedHead, ExpectedWorktree: preflight.ExpectedWorktree}
	apply, err := operations.Apply(ApplyOperation{Root: root, Request: request})
	if err == nil || !errors.Is(err, projectrun.ErrStale) {
		t.Fatalf("Apply with the old preflight bindings did not reject the changed project: report=%+v err=%v", apply, err)
	}
	if apply.Status != "rejected" || len(apply.Written) != 0 || len(apply.Journal) != 0 {
		t.Fatalf("stale Apply wrote files or returned a receipt: %+v", apply)
	}
	for path, before := range sourceBefore {
		after, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if readErr != nil || !bytes.Equal(after, before) {
			t.Fatalf("rejected Apply changed %s: before=%q after=%q err=%v", path, before, after, readErr)
		}
	}
	if data, readErr := os.ReadFile(filepath.Join(root, unclassifiedPath)); readErr != nil || string(data) != "new ordinary source after preflight\n" {
		t.Fatalf("unclassified source was not left untouched: content=%q err=%v", data, readErr)
	}
	if _, readErr := os.Stat(filepath.Join(root, "README.md")); !os.IsNotExist(readErr) {
		t.Fatalf("rejected Apply wrote the generated README into the selected working tree: %v", readErr)
	}

	afterStatus, err := operations.Status(RunOperation{Root: root, RunID: run.ID})
	if err != nil || afterStatus.Run.Status != projectrun.StatusVerified || afterStatus.Run.Digest != beforeStatus.Run.Digest {
		t.Fatalf("rejected Apply changed the durable verified run: report=%+v err=%v", afterStatus, err)
	}
	if afterStatus.Plan.Digest != beforeStatus.Plan.Digest || afterStatus.Run.Candidate.ID != run.Candidate.ID {
		t.Fatalf("rejected Apply changed the persisted plan or candidate binding: %+v", afterStatus)
	}
	planAfter, err := operations.ReadRecord(root, planPath)
	if err != nil || !bytes.Equal(planAfter, planBytes) {
		t.Fatalf("rejected Apply changed the immutable plan record: err=%v", err)
	}
	stateAfter, err := operations.ReadRecord(root, statePath)
	if err != nil || !bytes.Equal(stateAfter, stateBytes) {
		t.Fatalf("rejected Apply changed the durable verified state: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "runs", run.ID, "apply")); !os.IsNotExist(err) {
		t.Fatalf("rejected Apply persisted an apply receipt directory: %v", err)
	}
}
