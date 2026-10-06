package recordstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "store")
	store, err := Initialize(root, []string{repo})
	if err != nil {
		t.Fatal(err)
	}
	return store, root
}

func testHash(value string) string { return hash([]byte(value)) }

func testProjection(t *testing.T, projection, artifact string, state string, prior string, content string) records.ProjectionRecord {
	t.Helper()
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: strings.Repeat("a", 40), ModelDigest: testHash("model"), PlanDigest: testHash("plan:" + content),
		InputSnapshotDigest: testHash("input"), RequestDigest: testHash("request:" + content),
		Module:       records.ModuleIdentity{Name: "mod", Version: "1.0.0", Digest: testHash("module")},
		ProjectionID: projection, Projector: records.ProjectorIdentity{ID: "projector", Version: "1"},
		ScopeIDs: []string{"scope:" + projection}, Artifacts: []records.Artifact{{Path: artifact, Digest: testHash(content), Mode: "100644", Change: records.ChangeModified}},
		PriorRecordID: prior, State: state,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func testVerification(t *testing.T, record records.ProjectionRecord, outcome string) records.VerificationResult {
	t.Helper()
	checkOutcome := records.CheckFailed
	if outcome == records.OutcomePassed {
		checkOutcome = records.CheckPassed
	}
	if outcome == records.OutcomeIncomplete {
		checkOutcome = records.CheckIncomplete
	}
	result, err := records.NewVerificationResult(records.VerificationResult{
		RecordID: record.ID, Revision: record.Revision, ModelDigest: record.ModelDigest, TargetSnapshotDigest: record.TargetSnapshotDigest,
		Verifier: records.VerifierIdentity{ID: "independent-checker", Version: "1", Digest: testHash("verifier")},
		Checks:   []records.CheckResult{{ID: "contract", Version: "1", Digest: testHash("contract"), Outcome: checkOutcome}}, Outcome: outcome,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestAppendCapacityRejectsLimitCrossingBeforeStaging(t *testing.T) {
	if err := checkAppendCapacity(maxEvents, 0, 1); err == nil {
		t.Fatal("event count limit was not enforced")
	}
	if err := checkAppendCapacity(0, maxBytes, 1); err == nil {
		t.Fatal("aggregate byte limit was not enforced")
	}
	if err := checkAppendCapacity(maxEvents-1, maxBytes-2, 1); err != nil {
		t.Fatalf("valid final bounded event rejected: %v", err)
	}
}

func TestReadDoesNotCreateOrChangeStoreFiles(t *testing.T) {
	store, root := openTestStore(t)
	before := inventoryStoreFiles(t, root)
	if _, err := store.Read(); err != nil {
		t.Fatal(err)
	}
	after := inventoryStoreFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Read changed store files: before=%v after=%v", before, after)
	}
	if _, err := os.Lstat(filepath.Join(root, lockName)); !os.IsNotExist(err) {
		t.Fatalf("Read left a writer lock behind: %v", err)
	}
}

func inventoryStoreFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	for _, dir := range []string{root, filepath.Join(root, eventsDir)} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				if path == filepath.Join(root, eventsDir) {
					continue
				}
				t.Fatalf("unexpected directory in store inventory: %s", path)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files[path] = data
		}
	}
	return files
}

func TestInitializeAppendReadAndKeepVerificationSeparateFromActiveOwnership(t *testing.T) {
	store, _ := openTestStore(t)
	initial, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if initial.Sequence != 0 || initial.Head == "" || initial.ActiveSelection.RecordIDs == nil || len(initial.ActiveSelection.RecordIDs) != 0 {
		t.Fatalf("initial state is not an explicit empty selection: %+v", initial)
	}
	record := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, "", "candidate-a")
	state, err := store.AppendAttempt(initial.Head, record)
	if err != nil {
		t.Fatal(err)
	}
	if state.Sequence != 1 || len(state.Records) != 1 || state.ActiveSelection.Generation != 0 {
		t.Fatalf("append changed unexpected state: %+v", state)
	}
	duplicate, err := store.AppendAttempt(initial.Head, record)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Sequence != state.Sequence {
		t.Fatalf("same content ID was not idempotent: %d != %d", duplicate.Sequence, state.Sequence)
	}
	failed := testVerification(t, record, records.OutcomeFailed)
	state, err = store.AppendVerification(state.Head, failed)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Verifications) != 1 || state.Verifications[0].Outcome != records.OutcomeFailed || len(state.ActiveSelection.RecordIDs) != 0 {
		t.Fatalf("failed result was lost or changed active selection: %+v", state)
	}
	state, err = store.SelectActive(state.Head, []string{record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ActiveSelection.RecordIDs) != 1 || state.ActiveSelection.RecordIDs[0] != record.ID || state.ActiveSelection.Generation != 1 {
		t.Fatalf("explicit active selection missing: %+v", state.ActiveSelection)
	}
	state, err = store.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Verifications) != 1 || state.Verifications[0].Outcome != records.OutcomeFailed || len(state.ActiveSelection.RecordIDs) != 1 {
		t.Fatalf("read lost history or active selection: %+v", state)
	}
}

func TestAppendAndActiveSelectionRejectStaleHead(t *testing.T) {
	store, _ := openTestStore(t)
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	a := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, "", "a")
	state, err = store.AppendAttempt(state.Head, a)
	if err != nil {
		t.Fatal(err)
	}
	oldHead := state.Head
	b := testProjection(t, "projection-b", "src/b.cs", records.StateMaterializedUnverified, "", "b")
	state, err = store.AppendAttempt(state.Head, b)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.SelectActive(oldHead, []string{a.ID})
	if !errors.Is(err, ErrStaleHead) {
		t.Fatalf("stale active CAS error=%v", err)
	}
	_, err = store.AppendAttempt(oldHead, testProjection(t, "projection-c", "src/c.cs", records.StateMaterializedUnverified, "", "c"))
	if !errors.Is(err, ErrStaleHead) {
		t.Fatalf("stale append error=%v", err)
	}
}

func TestActiveSelectionRefusesFailedMaterializationAndOwnershipCollisions(t *testing.T) {
	store, _ := openTestStore(t)
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	partial := testProjection(t, "partial", "src/partial.cs", records.StatePartialFailure, "", "partial")
	state, err = store.AppendAttempt(state.Head, partial)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.SelectActive(state.Head, []string{partial.ID}); err == nil {
		t.Fatal("partial failure became active")
	}
	if _, err = store.SelectActive(state.Head, nil); err == nil {
		t.Fatal("nil selection was accepted")
	}

	store2, _ := openTestStore(t)
	state, err = store2.Read()
	if err != nil {
		t.Fatal(err)
	}
	a := testProjection(t, "a", "out/Shared.cs", records.StateMaterializedUnverified, "", "a")
	state, err = store2.AppendAttempt(state.Head, a)
	if err != nil {
		t.Fatal(err)
	}
	b := testProjection(t, "b", "out/shared.cs", records.StateMaterializedUnverified, "", "b")
	state, err = store2.AppendAttempt(state.Head, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store2.SelectActive(state.Head, []string{a.ID, b.ID}); err == nil || !strings.Contains(err.Error(), "collide under portable case matching") {
		t.Fatalf("portable case alias was not rejected: %v", err)
	}
	if len(state.ActiveSelection.RecordIDs) != 0 {
		t.Fatal("failed selection mutated active state")
	}
}

func TestRecordHistoryRequiresEarlierPriorRecordForSameProjection(t *testing.T) {
	store, _ := openTestStore(t)
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	missing := testHash("missing")
	record := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, missing, "a")
	if _, err = store.AppendAttempt(state.Head, record); err == nil {
		t.Fatal("dangling prior record was accepted")
	}
	first := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, "", "first")
	state, err = store.AppendAttempt(state.Head, first)
	if err != nil {
		t.Fatal(err)
	}
	second := testProjection(t, "projection-b", "src/b.cs", records.StateMaterializedUnverified, first.ID, "second")
	if _, err = store.AppendAttempt(state.Head, second); err == nil {
		t.Fatal("prior record from a different Projection was accepted")
	}
}

func TestReadRejectsTamperingDuplicateKeysDanglingAndPendingFiles(t *testing.T) {
	t.Run("duplicate key", func(t *testing.T) {
		store, root := openTestStore(t)
		state, _ := store.Read()
		record := testProjection(t, "p", "out/a", records.StateMaterializedUnverified, "", "a")
		state, err := store.AppendAttempt(state.Head, record)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, eventsDir, eventName(state.Sequence, state.Head))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte(`"sequence":1`), []byte(`"sequence":1,"sequence":1`), 1)
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = store.Read(); err == nil || !strings.Contains(err.Error(), "duplicate key") {
			t.Fatalf("duplicate event field was accepted: %v", err)
		}
	})
	t.Run("tampered digest", func(t *testing.T) {
		store, root := openTestStore(t)
		state, _ := store.Read()
		record := testProjection(t, "p", "out/a", records.StateMaterializedUnverified, "", "a")
		state, err := store.AppendAttempt(state.Head, record)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, eventsDir, eventName(state.Sequence, state.Head))
		data, _ := os.ReadFile(path)
		original := append([]byte(nil), data...)
		data = bytes.Replace(data, []byte(`"state":"materialized-unverified"`), []byte(`"state":"partial-failure"`), 1)
		if bytes.Equal(data, original) {
			t.Fatal("tamper fixture did not modify serialized event")
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = store.Read(); err == nil {
			t.Fatal("tampered event accepted")
		}
	})
	t.Run("dangling result", func(t *testing.T) {
		store, _ := openTestStore(t)
		state, _ := store.Read()
		record := testProjection(t, "p", "out/a", records.StateMaterializedUnverified, "", "a")
		orphan := testVerification(t, record, records.OutcomeFailed)
		orphan.RecordID = testHash("unrecorded")
		orphan, err := records.NewVerificationResult(orphan)
		if err != nil {
			t.Fatal(err)
		}
		_, err = store.commit(eventBody{APIVersion: APIVersion, Sequence: 1, PreviousEventDigest: state.Head, Kind: "verification", Verification: &orphan})
		if err == nil {
			t.Fatal("malformed event readback unexpectedly passed")
		}
		if _, err = store.Read(); err == nil {
			t.Fatal("dangling event was accepted on next read")
		}
	})
	t.Run("pending file", func(t *testing.T) {
		store, root := openTestStore(t)
		path := filepath.Join(root, eventsDir, pending+"interrupted.json")
		if err := os.WriteFile(path, []byte("partial"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := store.Read()
		if !errors.Is(err, ErrIncomplete) || !strings.Contains(err.Error(), path) {
			t.Fatalf("pending staging result=%v", err)
		}
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("pending file was removed: %v", err)
		}
	})
}

func TestLockRemnantAndExternalRootOverlapBlockWithoutCleanup(t *testing.T) {
	store, root := openTestStore(t)
	lock := filepath.Join(root, lockName)
	if err := os.WriteFile(lock, []byte("owner lock"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := store.Read()
	if !errors.Is(err, ErrLocked) || !strings.Contains(err.Error(), lock) {
		t.Fatalf("lock path was not surfaced: %v", err)
	}
	if _, err = os.Stat(lock); err != nil {
		t.Fatalf("lock was removed: %v", err)
	}
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err = os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = Initialize(filepath.Join(repo, "records"), []string{repo}); err == nil || !strings.Contains(err.Error(), "overlaps forbidden root") {
		t.Fatalf("overlap accepted: %v", err)
	}
	if _, err = Open(root, nil); err == nil {
		t.Fatal("missing explicit forbidden-root argument accepted")
	}
}

func TestOpenRefusesUnknownOrMalformedEventNames(t *testing.T) {
	store, root := openTestStore(t)
	bad := filepath.Join(root, eventsDir, "00000000000000000001-extra.json")
	if err := os.WriteFile(bad, []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(); err == nil || !strings.Contains(err.Error(), "malformed event filename") {
		t.Fatalf("malformed event filename accepted: %v", err)
	}
	if _, err := Open(root, []string{}); err != nil {
		t.Fatal(err)
	}
}

func TestSelectionRetryUsesSelectionDigestAndNoOpDoesNotAppend(t *testing.T) {
	store, _ := openTestStore(t)
	state, _ := store.Read()
	record := testProjection(t, "p", "out/a", records.StateMaterializedUnverified, "", "a")
	state, err := store.AppendAttempt(state.Head, record)
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.SelectActive(state.Head, []string{record.ID})
	if err != nil {
		t.Fatal(err)
	}
	sequence := state.Sequence
	retry, err := store.SelectActive(state.Head, []string{record.ID})
	if err != nil {
		t.Fatal(err)
	}
	if retry.Sequence != sequence || retry.ActiveSelection.Digest != state.ActiveSelection.Digest {
		t.Fatal(fmt.Sprintf("selection retry changed state: %+v", retry))
	}
}

func TestAppendRefreshCreatesRetainedRecordAndPreservesOldVerification(t *testing.T) {
	store, root := openTestStore(t)
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	prior := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, "", "same-bytes")
	state, err = store.AppendAttempt(state.Head, prior)
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.SelectActive(state.Head, []string{prior.ID})
	if err != nil {
		t.Fatal(err)
	}
	oldResult := testVerification(t, prior, records.OutcomePassed)
	state, err = store.AppendVerification(state.Head, oldResult)
	if err != nil {
		t.Fatal(err)
	}

	retained := prior
	retained.Revision = strings.Repeat("b", 40)
	retained.ModelDigest = testHash("new-model")
	retained.PlanDigest = testHash("refresh-plan")
	retained.InputSnapshotDigest = testHash("new-input")
	retained.RequestDigest = testHash("new-request")
	retained.PriorRecordID = prior.ID
	retained.Artifacts = append([]records.Artifact(nil), prior.Artifacts...)
	retained.Artifacts[0].Change = records.ChangeRetained
	retained, err = records.NewProjectionRecord(retained)
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.AppendRefresh(state.Head, retained)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Records) != 2 || len(state.Verifications) != 1 || state.Verifications[0].ID != oldResult.ID {
		t.Fatalf("refresh changed historical records/results: %#v", state)
	}
	if len(state.ActiveSelection.RecordIDs) != 1 || state.ActiveSelection.RecordIDs[0] != prior.ID {
		t.Fatalf("refresh append changed active selection: %#v", state.ActiveSelection)
	}
	state, err = store.SelectActive(state.Head, []string{retained.ID})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root, store.forbidden)
	if err != nil {
		t.Fatal(err)
	}
	state, err = reopened.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Records) != 2 || len(state.Verifications) != 1 || state.ActiveSelection.RecordIDs[0] != retained.ID {
		t.Fatalf("refresh event did not survive validated read: %#v", state)
	}
}

func TestAppendRefreshRejectsChangedFactsInactivePriorAndStaleHead(t *testing.T) {
	store, _ := openTestStore(t)
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	prior := testProjection(t, "projection-a", "src/a.cs", records.StateMaterializedUnverified, "", "same-bytes")
	state, err = store.AppendAttempt(state.Head, prior)
	if err != nil {
		t.Fatal(err)
	}
	makeRefresh := func() records.ProjectionRecord {
		record := prior
		record.Revision = strings.Repeat("b", 40)
		record.ModelDigest = testHash("new-model")
		record.PlanDigest = testHash("refresh-plan")
		record.InputSnapshotDigest = testHash("new-input")
		record.RequestDigest = testHash("new-request")
		record.PriorRecordID = prior.ID
		record.Artifacts = append([]records.Artifact(nil), prior.Artifacts...)
		record.Artifacts[0].Change = records.ChangeRetained
		created, createErr := records.NewProjectionRecord(record)
		if createErr != nil {
			t.Fatal(createErr)
		}
		return created
	}
	valid := makeRefresh()
	if _, err = store.AppendRefresh(state.Head, valid); err == nil || !strings.Contains(err.Error(), "active") {
		t.Fatalf("inactive prior record was accepted: %v", err)
	}
	state, err = store.SelectActive(state.Head, []string{prior.ID})
	if err != nil {
		t.Fatal(err)
	}
	changed := valid
	changed.Artifacts = append([]records.Artifact(nil), valid.Artifacts...)
	changed.Artifacts[0].Digest = testHash("different-bytes")
	changed, err = records.NewProjectionRecord(changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.AppendRefresh(state.Head, changed); err == nil || !strings.Contains(err.Error(), "exact prior artifact set") {
		t.Fatalf("changed artifact facts were accepted: %v", err)
	}
	if _, err = store.AppendRefresh(testHash("stale-head"), valid); !errors.Is(err, ErrStaleHead) {
		t.Fatalf("stale head was not rejected: %v", err)
	}
	state, err = store.AppendRefresh(state.Head, valid)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.AppendRefresh(testHash("stale-head"), valid); err != nil || got.Head != state.Head {
		t.Fatalf("identical refresh retry was not idempotent: got=%s err=%v", got.Head, err)
	}
}
