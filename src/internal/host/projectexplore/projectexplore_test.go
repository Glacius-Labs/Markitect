package projectexplore

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func testRepository(t *testing.T, committed bool) (string, string, string) {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("symbolic-ref", "HEAD", "refs/heads/codex/explore-test")
	basisPath := ".markitect/model/work.yaml"
	repo.Write(basisPath, "work: initial\n")
	head := ""
	if committed {
		head = repo.Commit("initial accepted model")
	}
	return repo.Dir, head, basisPath
}

func testBinding(root, head, basisPath string, accepted bool) Binding {
	content, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(basisPath)))
	return Binding{
		RepositoryRoot: root, Branch: "codex/explore-test", Head: head, ModelRevision: head,
		ModelAccepted: accepted, AcceptancePolicy: map[bool]string{true: "canonical-committed-head", false: ""}[accepted],
		ProjectDigest: testDigest("project"), ModelDigest: testDigest("model"), SnapshotDigest: testDigest("snapshot"), SelectionDigest: testDigest("selection"),
		RuntimeDigest: testDigest("runtime"), BriefingDigest: testDigest("briefings"), ScopeID: "work-item-1", ScopeName: "Add bounded work item",
		Goal: "Implement the requested behavior", Operation: "apply", ManagerIDs: []string{"orders"}, RequiredArtifacts: []string{"order-service"},
		ResponsibleManagerIDs: []string{"orders"},
		FileStructure:         []string{"internal/orders/service.go", "internal/orders/service_test.go"}, Checks: []string{"go test ./internal/orders"},
		BasisFiles: []BasisFile{{Path: basisPath, Digest: fileDigest(content)}},
	}
}

func testRecord(binding Binding, decisions []Decision) Record {
	return Record{
		APIVersion: APIVersion, ID: "work-item-1", Status: StatusActive, Request: "Add the requested order behavior",
		Scopes:    []Scope{{ID: binding.ScopeID, Name: binding.ScopeName, Goal: binding.Goal, Operation: binding.Operation, ManagerIDs: []string{"orders"}}},
		Decisions: decisions, Drafts: []DraftProposal{}, Acknowledgements: []StructureAcknowledgement{}, Completions: []ApplyReceipt{},
	}
}

func testDigest(value string) string {
	digest, _ := digest(map[string]string{"value": value})
	return digest
}

func createRecord(t *testing.T, root string, binding Binding, record Record) Record {
	t.Helper()
	plan, err := CreatePreview(root, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest == "" || plan.Target.Exists {
		t.Fatalf("create preview is incomplete: %+v", plan)
	}
	written, err := Write(root, plan, plan.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	return written
}

func TestClosedRecordCodecRejectsUnknownAndDuplicateFields(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := testRecord(binding, []Decision{})
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRecordInput(data)
	if err != nil || decoded.Digest == "" {
		t.Fatalf("DecodeRecordInput = %+v, %v", decoded, err)
	}
	if _, err := DecodeRecord(append(data, '\n')); err == nil {
		t.Fatal("persisted decoder accepted a digestless record")
	}
	if _, err := DecodeRecordInput([]byte(`{"apiVersion":"x","apiVersion":"y"}`)); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate key error = %v", err)
	}
	if _, err := DecodeRecordInput([]byte(`{"unexpected":true}`)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestCreateAndUpdateUseDigestCASAndSourceFreshness(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := testRecord(binding, []Decision{})
	plan, err := CreatePreview(root, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, plan, "sha256:wrong", binding); err == nil {
		t.Fatal("write accepted a wrong expected plan digest")
	}
	path := filepath.Join(root, filepath.FromSlash(basis))
	if err := os.WriteFile(path, []byte("work: changed after preview\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, plan, plan.Digest, binding); err == nil {
		t.Fatal("write accepted changed source basis bytes")
	}
	binding.BasisFiles[0].Digest = fileDigest([]byte("work: changed after preview\n"))
	record = createRecord(t, root, binding, record)
	loaded, err := Load(root, record.ID)
	if err != nil || loaded.Digest != record.Digest {
		t.Fatalf("Load = %+v, %v", loaded, err)
	}
	listed, err := List(root)
	if err != nil || len(listed) != 1 || listed[0].ID != record.ID {
		t.Fatalf("List = %+v, %v", listed, err)
	}
	updated := loaded
	updated.Drafts = []DraftProposal{{ID: "model-proposal", Goal: "Explore a possible model update", Files: []DraftFile{{Path: ".markitect/model/proposed.yaml", Content: "proposal: draft\n"}}}}
	updated.Digest = ""
	if err := SealRecord(&updated); err != nil {
		t.Fatal(err)
	}
	stale, err := UpdatePreview(root, updated, testDigest("wrong state"), binding)
	if err == nil || stale.Digest != "" {
		t.Fatalf("update accepted stale CAS: plan=%+v err=%v", stale, err)
	}
	updatePlan, err := UpdatePreview(root, updated, loaded.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	written, err := Write(root, updatePlan, updatePlan.Digest, binding)
	if err != nil || len(written.Drafts) != 1 {
		t.Fatalf("update write = %+v, %v", written, err)
	}
	if _, err := UpdatePreview(root, updated, loaded.Digest, binding); err == nil {
		t.Fatal("second update accepted an old record digest")
	}
}

func TestReadinessRequiresAcceptedModelDecisionResolutionAndExactAcknowledgement(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	open := Decision{ID: "public-contract", ScopeIDs: []string{binding.ScopeID}, Question: "May this outcome be externally visible?", Blocking: true, Status: "open"}
	deferred := Decision{ID: "later-reporting", ScopeIDs: []string{binding.ScopeID}, Question: "Should a later reporting view be added?", Blocking: false, Status: "deferred", Reason: "Outside this work item", Authority: "asserted project owner", Provenance: "conversation:defer-1"}
	record := createRecord(t, root, binding, testRecord(binding, []Decision{open, deferred}))
	stateDigest := record.Digest
	structureDigest, err := StructureDigest(binding)
	if err != nil {
		t.Fatal(err)
	}
	bindingDigest, err := BindingDigest(binding)
	if err != nil {
		t.Fatal(err)
	}
	ack := StructureAcknowledgement{ScopeID: binding.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Actor: "project-owner", Authority: "asserted project owner", Provenance: "conversation:structure-reviewed", RecordedAt: time.Now().UTC()}
	if err := AcknowledgeStructure(&record, binding.ScopeID, binding, ack); err != nil {
		t.Fatal(err)
	}
	plan, err := UpdatePreview(root, record, stateDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	record, err = Write(root, plan, plan.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := EvaluateReadiness(record, binding.ScopeID, binding)
	if err != nil || blocked.Ready || len(blocked.OpenDecisions) != 2 || len(blocked.Blockers) < 1 {
		t.Fatalf("open blocking decision readiness = %+v, %v", blocked, err)
	}
	updated := record
	updated.Decisions = []Decision{{ID: open.ID, ScopeIDs: open.ScopeIDs, Question: open.Question, Blocking: true, Status: "answered", Answer: "The outcome stays internal", Authority: "asserted project owner", Provenance: "conversation:decision-1"}, deferred}
	updated.Digest = ""
	if err := SealRecord(&updated); err != nil {
		t.Fatal(err)
	}
	plan, err = UpdatePreview(root, updated, record.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	record, err = Write(root, plan, plan.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := EvaluateReadiness(record, binding.ScopeID, binding)
	if err != nil || !ready.Ready || len(ready.OpenDecisions) != 1 || ready.OpenDecisions[0] != deferred.ID {
		t.Fatalf("resolved and acknowledged readiness = %+v, %v", ready, err)
	}
	changed := binding
	changed.FileStructure = append(append([]string(nil), binding.FileStructure...), "internal/orders/new.go")
	changed.FileStructure = sortedCopy(changed.FileStructure)
	changedReport, err := EvaluateReadiness(record, binding.ScopeID, changed)
	if err != nil || changedReport.Ready || !strings.Contains(strings.Join(changedReport.Blockers, " "), "acknowledged") {
		t.Fatalf("changed structure reused old acknowledgement: %+v, %v", changedReport, err)
	}
	changed = binding
	changed.ResponsibleManagerIDs = []string{"orders", "project-owner"}
	changedReport, err = EvaluateReadiness(record, binding.ScopeID, changed)
	if err != nil || changedReport.Ready || !strings.Contains(strings.Join(changedReport.Blockers, " "), "acknowledged") {
		t.Fatalf("changed responsible Manager set reused old acknowledgement: %+v, %v", changedReport, err)
	}
	unaccepted := binding
	unaccepted.ModelAccepted = false
	unaccepted.ModelRevision = ""
	unaccepted.AcceptancePolicy = ""
	unacceptedReport, err := EvaluateReadiness(record, binding.ScopeID, unaccepted)
	if err != nil || unacceptedReport.Ready || !strings.Contains(strings.Join(unacceptedReport.Blockers, " "), "accepted canonical model") {
		t.Fatalf("unaccepted model readiness = %+v, %v", unacceptedReport, err)
	}
}

func TestUnbornProjectCanPersistDraftButCannotBeReady(t *testing.T) {
	root, _, basis := testRepository(t, false)
	binding := testBinding(root, "", basis, false)
	record := createRecord(t, root, binding, testRecord(binding, []Decision{}))
	if record.CreatedAgainst == "" {
		t.Fatal("created record did not retain its initial binding")
	}
	report, err := EvaluateReadiness(record, binding.ScopeID, binding)
	if err != nil || report.Ready || !strings.Contains(strings.Join(report.Blockers, " "), "accepted canonical model") {
		t.Fatalf("unborn model readiness = %+v, %v", report, err)
	}
}

func TestWriteRejectsNewCommitSincePreview(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := testRecord(binding, []Decision{})
	plan, err := CreatePreview(root, record, binding)
	if err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("unrelated commit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-m", "advance HEAD"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if _, err := Write(root, plan, plan.Digest, binding); err == nil || !strings.Contains(err.Error(), "HEAD") {
		t.Fatalf("write accepted advanced HEAD: %v", err)
	}
}

func TestCompleteRequiresSuccessfulVerifiedApplyAndIsIdempotent(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := createRecord(t, root, binding, testRecord(binding, []Decision{}))
	stateDigest := record.Digest
	bindingDigest, _ := BindingDigest(binding)
	structureDigest, _ := StructureDigest(binding)
	ack := StructureAcknowledgement{ScopeID: binding.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Actor: "owner", Authority: "asserted owner", Provenance: "conversation:reviewed", RecordedAt: time.Now().UTC()}
	if err := AcknowledgeStructure(&record, binding.ScopeID, binding, ack); err != nil {
		t.Fatal(err)
	}
	plan, err := UpdatePreview(root, record, stateDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	record, err = Write(root, plan, plan.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := EvaluateReadiness(record, binding.ScopeID, binding)
	if err != nil || !ready.Ready {
		t.Fatalf("fixture readiness = %+v, %v", ready, err)
	}
	receipt := ApplyReceipt{
		ScopeID: binding.ScopeID, BindingDigest: ready.BindingDigest, StructureDigest: ready.StructureDigest, Status: "applied",
		RunID: "run-1", PlanID: "plan-1", PlanDigest: testDigest("plan"), CandidateID: "candidate-1", CandidateDigest: testDigest("candidate"),
		VerificationID: "verify-1", VerificationStatus: "passed", VerificationDigest: testDigest("verification"), ApplyID: "apply-1", ApplyDigest: testDigest("apply"), AppliedAt: time.Now().UTC(),
	}
	failed := receipt
	failed.Status = "failed"
	if _, err := Complete(root, record.ID, binding.ScopeID, binding, failed); err == nil {
		t.Fatal("completion accepted failed Apply receipt")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(recordsDirectory), record.ID+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(basis)), []byte("work: applied candidate\n"), 0644); err != nil {
		t.Fatal(err)
	}
	completed, err := Complete(root, record.ID, binding.ScopeID, binding, receipt)
	if err != nil || completed.Status != StatusCompleted || len(completed.Completions) != 1 {
		t.Fatalf("Complete = %+v, %v", completed, err)
	}
	repeated, err := Complete(root, record.ID, binding.ScopeID, binding, receipt)
	if err != nil || repeated.Digest != completed.Digest {
		t.Fatalf("repeated Complete = %+v, %v", repeated, err)
	}
	other := receipt
	other.ApplyID = "different-apply"
	if _, err := Complete(root, record.ID, binding.ScopeID, binding, other); err == nil {
		t.Fatal("completed scope accepted a different receipt")
	}
}

func TestGenericWriteCannotMintCompletionReceipt(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := createRecord(t, root, binding, testRecord(binding, []Decision{}))
	stateDigest := record.Digest
	bindingDigest, _ := BindingDigest(binding)
	structureDigest, _ := StructureDigest(binding)
	ack := StructureAcknowledgement{ScopeID: binding.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Actor: "owner", Authority: "asserted owner", Provenance: "conversation:reviewed", RecordedAt: time.Now().UTC()}
	if err := AcknowledgeStructure(&record, binding.ScopeID, binding, ack); err != nil {
		t.Fatal(err)
	}
	ackPlan, err := UpdatePreview(root, record, stateDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	record, err = Write(root, ackPlan, ackPlan.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	fake := ApplyReceipt{
		ScopeID: binding.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Status: "applied",
		RunID: "forged-run", PlanID: "forged-plan", PlanDigest: testDigest("plan"), CandidateID: "forged-candidate", CandidateDigest: testDigest("candidate"),
		VerificationID: "forged-verify", VerificationStatus: "passed", VerificationDigest: testDigest("verify"), ApplyID: "forged-apply", ApplyDigest: testDigest("apply"), AppliedAt: time.Now().UTC(),
	}
	plan, err := UpdatePreview(root, record, record.Digest, binding)
	if err != nil {
		t.Fatal(err)
	}
	plan.Next.Completions = []ApplyReceipt{fake}
	plan.Next.Status = StatusCompleted
	if err := SealRecord(&plan.Next); err != nil {
		t.Fatal(err)
	}
	plan.Digest, err = writePlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, plan, plan.Digest, binding); err == nil || !strings.Contains(err.Error(), "cannot mark") {
		t.Fatalf("generic write accepted forged completion: %v", err)
	}
	loaded, err := Load(root, record.ID)
	if err != nil || loaded.Status != StatusActive || len(loaded.Completions) != 0 {
		t.Fatalf("forged completion changed durable state: %+v, %v", loaded, err)
	}
}

func TestGenericWriteCannotDisableSourceBasisVerification(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	record := createRecord(t, root, binding, testRecord(binding, []Decision{}))
	stateDigest := record.Digest
	record.Decisions = append(record.Decisions, Decision{ID: "d2", ScopeIDs: []string{binding.ScopeID}, Question: "Which check should run?", Status: "open"})
	if err := SealRecord(&record); err != nil {
		t.Fatal(err)
	}
	plan, err := UpdatePreview(root, record, stateDigest, binding)
	if err != nil {
		t.Fatal(err)
	}
	plan.VerifyBasis = false
	plan.Digest, err = writePlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, plan, plan.Digest, binding); err == nil || !strings.Contains(err.Error(), "must verify") {
		t.Fatalf("generic write accepted a plan with source-basis verification disabled: %v", err)
	}
	loaded, err := Load(root, record.ID)
	if err != nil || loaded.Digest != stateDigest {
		t.Fatalf("rejected write changed durable state: %+v, %v", loaded, err)
	}
}

func TestBindingDigestCanonicalizesUnorderedSelectionInputs(t *testing.T) {
	root, head, basis := testRepository(t, true)
	binding := testBinding(root, head, basis, true)
	left, err := BindingDigest(binding)
	if err != nil {
		t.Fatal(err)
	}
	binding.ManagerIDs = []string{"inventory", "orders"}
	binding.RequiredArtifacts = []string{"z-artifact", "a-artifact"}
	binding.FileStructure = []string{"z/file.go", "a/file.go"}
	binding.Checks = []string{"z-check", "a-check"}
	binding.BasisFiles = append(binding.BasisFiles, BasisFile{Path: "README.md", Digest: testDigest("readme")})
	right, err := BindingDigest(binding)
	if err != nil {
		t.Fatal(err)
	}
	binding.ManagerIDs = []string{"orders", "inventory"}
	binding.RequiredArtifacts = []string{"a-artifact", "z-artifact"}
	binding.FileStructure = []string{"a/file.go", "z/file.go"}
	binding.Checks = []string{"a-check", "z-check"}
	binding.BasisFiles = []BasisFile{{Path: basis, Digest: binding.BasisFiles[0].Digest}, {Path: "README.md", Digest: testDigest("readme")}}
	canonical, err := BindingDigest(binding)
	if err != nil || right != canonical || left == right {
		t.Fatalf("binding canonicalization changed: original=%s unordered=%s canonical=%s err=%v", left, right, canonical, err)
	}
}

func TestBindingDigestNamesInvalidDigestFieldsInFixedOrder(t *testing.T) {
	binding := testBinding(t.TempDir(), "", ".markitect/model/work.yaml", false)
	binding.ProjectDigest, binding.ModelDigest, binding.SnapshotDigest, binding.SelectionDigest = "bad", "bad", "bad", "bad"
	for i := 0; i < 100; i++ {
		if _, err := BindingDigest(binding); err == nil || err.Error() != "binding projectDigest is not a sha256 digest" {
			t.Fatalf("BindingDigest error = %v, want the first invalid field in declaration order", err)
		}
	}
}

func TestBindingDigestPreservesExplicitEmptyScopeArrays(t *testing.T) {
	root, head, _ := testRepository(t, true)
	binding := testBinding(root, head, ".markitect/model/work.yaml", true)
	binding.ManagerIDs = []string{}
	binding.RequiredArtifacts = []string{}
	binding.Checks = []string{}
	binding.BasisFiles = []BasisFile{}
	if _, err := BindingDigest(binding); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalBinding(binding)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.ManagerIDs == nil || canonical.ResponsibleManagerIDs == nil || canonical.RequiredArtifacts == nil || canonical.Checks == nil || canonical.BasisFiles == nil {
		t.Fatalf("canonical binding lost explicit empty arrays: %+v", canonical)
	}
}
