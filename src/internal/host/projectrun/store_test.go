package projectrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanStoreRoundTripBindsRepositoryAndProjectDigests(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, RuntimePath), []byte("runtime"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	plan := PlanRecord{APIVersion: APIVersion, ID: "0123456789abcdef0123456789abcdef", Status: StatusPlanned,
		Goal: "bounded change", ExecuteAuthorized: true, BaseRevision: "0123456789012345678901234567890123456789",
		RepositoryDigest: "repo-sha", BaseSnapshot: "base-snapshot", WorkingSnapshot: "working-snapshot",
		BaseProjectDigest: "base-project", WorkingProjectDigest: "working-project", BaseModelDigest: "base-model",
		ModelDigest: "model", ReportDigest: "report", RuntimeDigest: "runtime", InitialCandidateID: "abcdef0123456789abcdef0123456789",
		RuntimeAgents: map[string]string{"root": "agent"}}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.createRun(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.writePlan(plan); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.readPlan(plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Digest != plan.Digest {
		t.Fatalf("plan digest changed: got %s want %s", loaded.Digest, plan.Digest)
	}

	loaded.RepositoryDigest = "different-repository"
	if _, err := planDigest(loaded); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.base, plan.ID, "plan.json"), []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.readPlan(plan.ID); err == nil {
		t.Fatal("corrupt plan unexpectedly loaded")
	}
}

func TestWriteImmutableDoesNotReplaceExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	if err := writeImmutable(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := writeImmutable(path, []byte("second")); err == nil {
		t.Fatal("immutable publish replaced existing state")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "first" {
		t.Fatalf("existing content changed to %q", content)
	}
}

func TestRunStateJournalStartsAtPositiveRevision(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, RuntimePath), []byte("runtime"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	id := "0123456789abcdef0123456789abcdef"
	if _, err := store.createRun(id); err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(store.base, id, "states")); err != nil {
		t.Fatal(err)
	}
	report := RunReport{APIVersion: APIVersion, ID: id, PlanID: id, Status: StatusRunning, Revision: 1}
	if err := store.appendState(report); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.readLatestState(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 1 {
		t.Fatalf("got revision %d, want 1", loaded.Revision)
	}
	report = loaded
	if err := persistState(store, &report); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.readLatestState(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 2 {
		t.Fatalf("got revision %d, want 2", loaded.Revision)
	}
}
