package projectrun

import (
	"context"
	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func runtimeAbsenceFixture(t *testing.T, id string) (*runStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repository")
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.createRun(id)
	if err != nil {
		t.Fatal(err)
	}
	return store, dir
}

func TestRuntimeValidatedAbsenceRemainsDistinctFromStorageFailure(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	store, dir := runtimeAbsenceFixture(t, id)
	if _, err := store.readPlan(id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("genuinely missing plan: %v", err)
	}
	if _, err := store.readLatestState(id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("genuinely empty state list: %v", err)
	}
	if _, err := readRecordBytes(filepath.Join(dir, "plan.json"), store); !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrNotFound) {
		t.Fatalf("private validated absence lost its distinction: %v", err)
	}
	if packets, err := readExternalPackets(dir, store); err != nil || len(packets) != 0 {
		t.Fatalf("genuinely missing packet directory: %v %v", packets, err)
	}
	if attempted, err := persistedVerificationAttempt(dir, id, id, store); err != nil || attempted {
		t.Fatalf("genuinely missing verification: %v %v", attempted, err)
	}
	if _, err := writeStageEvidence(store, id, "attempt", ".json", []byte("evidence")); err != nil {
		t.Fatalf("validated missing evidence could not be created: %v", err)
	}
	marker := filepath.Join(dir, "reports", "retained.json")
	if err := os.WriteFile(marker, []byte("retained-original"), 0600); err != nil {
		t.Fatal(err)
	}
	// No physical scope remains open: this is an actual missing captured root,
	// with its records retained under the same test-owned parent.
	moved := store.root + "-original"
	if err := os.Rename(store.root, moved); err != nil {
		t.Fatal(err)
	}
	readers := map[string]func() error{
		"plan":         func() error { _, err := store.readPlan(id); return err },
		"state":        func() error { _, err := store.readLatestState(id); return err },
		"json":         func() error { var plan PlanRecord; return readJSON(filepath.Join(dir, "plan.json"), &plan, store) },
		"bytes":        func() error { _, err := readRecordBytes(filepath.Join(dir, "plan.json"), store); return err },
		"packets":      func() error { _, err := readExternalPackets(dir, store); return err },
		"submission":   func() error { _, err := readExternalSubmission(dir, id, store); return err },
		"verification": func() error { _, err := persistedVerificationAttempt(dir, id, id, store); return err },
		"evidence creation": func() error {
			_, err := writeStageEvidence(store, id, "new-attempt", ".json", []byte("new"))
			return err
		},
		"Manager stage": func() error {
			_, _, err := readManagerStageState(store, ManagerStageBinding{SessionID: "missing-root"})
			return err
		},
		"Manager initialization": func() error {
			_, err := initializeManagerStage(store, ManagerStageBinding{SessionID: "missing-root"})
			return err
		},
	}
	for name, read := range readers {
		err := read()
		if !errors.Is(err, fs.ErrNotExist) || errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrAwaitingExternal) {
			t.Errorf("%s classified missing captured storage as absence: %v", name, err)
		}
	}
	if _, err := os.Lstat(store.root); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("failed readers initialized replacement storage: %v", err)
	}
	relative, _ := filepath.Rel(store.root, marker)
	if wire, err := os.ReadFile(filepath.Join(moved, relative)); err != nil || string(wire) != "retained-original" {
		t.Fatalf("original bytes lost: %q %v", wire, err)
	}
}

func TestListedRuntimeStateDisappearanceDoesNotAuthorizeManagerInitialization(t *testing.T) {
	for _, stage := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest-state", true: "Manager-stage"}[stage], func(t *testing.T) {
			binding := ManagerStageBinding{SessionID: "listed-state-disappearance"}
			id := managerStageRunID(binding.SessionID)
			store, dir := runtimeAbsenceFixture(t, id)
			report := RunReport{APIVersion: APIVersion, ID: id, Revision: 1, ManagerStageBinding: &binding}
			if err := store.appendState(report); err != nil {
				t.Fatal(err)
			}
			statePath := filepath.Join(dir, "states", "00000001.json")
			if wire, err := os.ReadFile(statePath); err != nil || len(wire) == 0 {
				t.Fatalf("actual state setup: %v", err)
			}
			calls := 0
			store.beforeStateRead = func() error {
				calls++
				return os.Remove(statePath)
			}
			var err error
			if stage {
				var exists bool
				_, exists, err = readManagerStageState(store, binding)
				if exists {
					t.Fatal("disappeared state yielded stage authority")
				}
			} else {
				_, err = store.readLatestState(id)
			}
			if calls != 1 || !errors.Is(err, errRuntimeRecordMissing) || errors.Is(err, ErrNotFound) {
				t.Fatalf("listed disappearance became empty-state admission: calls=%d err=%v", calls, err)
			}
			for _, role := range []string{"states", "reports", "candidates"} {
				entries, err := os.ReadDir(filepath.Join(dir, role))
				if err != nil || len(entries) != 0 {
					t.Fatalf("failure added state/attempt/evidence: role=%s entries=%v err=%v", role, entries, err)
				}
			}
		})
	}
}

func TestPlanStoreRoundTripBindsRepositoryAndProjectDigests(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".markitect"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0o600); err != nil {
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
func TestAllRunIDsAndFullVerifyShareKindLease(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := first.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := first.createRun("0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	second, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if unlock, err := second.lock(); !errors.Is(err, ErrLocked) {
		if unlock != nil {
			unlock()
		}
		t.Fatalf("different-ID writer admitted: %v", err)
	}
	if _, err := finishFullVerify(root, true, FullVerifyReport{APIVersion: APIVersion, Status: "passed"}, nil); !errors.Is(err, ErrLocked) {
		t.Fatalf("FullVerify writer admitted: %v", err)
	}
	loads := 0
	if _, err := FullVerify(context.Background(), Host{Load: func(root, revision string) (*projectvalue.Project, error) {
		loads++
		return nil, errors.New("unexpected admission")
	}}, nil, root, FullVerifyRequest{Revision: "HEAD", Write: true}); !errors.Is(err, ErrLocked) || loads != 0 {
		t.Fatalf("FullVerify admission before lease: %v loads=%d", err, loads)
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(RunsPath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "full-") {
			t.Fatal("blocked FullVerify created a container")
		}
	}
}
func TestRuntimeReleaseFailureRetainsReplacementAndQualifiedError(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RunsPath), ".write.lock")
	if err := os.Rename(p, p+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := release(); !recordstore.WasPublished(err) {
		t.Fatalf("release failure hidden: %v", err)
	}
	wire, err := os.ReadFile(p)
	if err != nil || string(wire) != "replacement" {
		t.Fatal("release removed replacement")
	}
}
func TestPlanPersistenceReportsReleaseFailureWithoutLosingPublishedPlan(t *testing.T) {
	for _, invalidCandidate := range []bool{false, true} {
		t.Run(map[bool]string{false: "published", true: "primary-error"}[invalidCandidate], func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0600); err != nil {
				t.Fatal(err)
			}
			store, err := newRunStore(root)
			if err != nil {
				t.Fatal(err)
			}
			release, err := store.lock()
			if err != nil {
				t.Fatal(err)
			}
			id := "0123456789abcdef0123456789abcdef"
			plan := PlanRecord{APIVersion: APIVersion, ID: id, Status: StatusPlanned}
			plan.Digest, err = planDigest(plan)
			if err != nil {
				t.Fatal(err)
			}
			candidate := candidateData{ID: "abcdef0123456789abcdef0123456789", Files: map[string]File{}}
			if invalidCandidate {
				candidate.ID = "invalid"
			}
			lock := filepath.Join(root, filepath.FromSlash(RunsPath), ".write.lock")
			err = persistPlanWithLease(store, id, plan, candidate, func() error {
				if err := os.Rename(lock, lock+"-original"); err != nil {
					return err
				}
				if err := os.WriteFile(lock, []byte("replacement"), 0600); err != nil {
					return err
				}
				return release()
			})
			if !recordstore.WasPublishedAt(err, RunsPath+"/.write.lock") {
				t.Fatalf("release outcome hidden: %v", err)
			}
			loaded, readErr := store.readPlan(id)
			if invalidCandidate {
				if err == nil || !strings.Contains(err.Error(), "invalid candidate ID") || !errors.Is(readErr, ErrNotFound) {
					t.Fatalf("primary error lost: %v, read=%v", err, readErr)
				}
			} else if readErr != nil || loaded.Digest != plan.Digest {
				t.Fatalf("published plan lost: %+v %v", loaded, readErr)
			}
			wire, err := os.ReadFile(lock)
			if err != nil || string(wire) != "replacement" {
				t.Fatal("replacement removed")
			}
		})
	}
}

func TestWriteImmutableDoesNotReplaceExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), filepath.FromSlash(RunsPath), "0123456789abcdef0123456789abcdef", "plan.json")
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
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0o600); err != nil {
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

// canonicalRunFixture uses the same owning store and kind lease as production.
func canonicalRunFixture(t *testing.T) (*runStore, string) {
	t.Helper()
	store, dir, _ := canonicalRunFixtureWithRelease(t)
	return store, dir
}

func canonicalRunFixtureWithRelease(t *testing.T) (*runStore, string, func() error) {
	t.Helper()
	root := filepath.Join(testkit.TempDir(t), "repo")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, projectconfig.RuntimePath), []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := newRunStore(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := store.lock()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if store.storage == nil {
			return
		}
		if err := release(); err != nil {
			t.Errorf("release canonical run fixture: %v", err)
		}
	})
	dir, err := store.createRun("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	return store, dir, release
}

func TestKnownOwnerRecordReadersRefuseSamePathRootReplacement(t *testing.T) {
	owner, dir, release := canonicalRunFixtureWithRelease(t)
	runID := filepath.Base(dir)
	request := externalTestRequest(t)
	packet, err := createOrReadExternalPacket(owner, dir, PlanRecord{ID: runID, Digest: "sha256:plan"}, RunReport{ID: runID}, ManagerTask{ID: "task", ManagerID: "orders", WorkAttempts: 1}, "orders-work", request, externalRequestContext{Kind: "projectrun-task/v1", Phase: "work", ManagerID: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDirectory(filepath.Join(dir, "external", "responses"), owner); err != nil {
		t.Fatal(err)
	}
	response := []byte(`{"outcome":"proposed"}`)
	submission := externalSubmission{APIVersion: externalPacketVersion, PacketID: packet.ID, PacketDigest: packet.Digest, Response: response, ResponseHash: externalResponseDigest(response, nil, nil)}
	if err := writeImmutableJSON(externalSubmissionPath(dir, packet.ID), submission, owner); err != nil {
		t.Fatal(err)
	}
	verified := VerifyReport{APIVersion: APIVersion, RunID: runID, CandidateID: "11111111111111111111111111111111", Status: "verified", VerifiedAt: time.Now().UTC()}
	verified.Digest, err = verificationDigest(verified)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistVerify(owner, dir, verified); err != nil {
		t.Fatal(err)
	}

	// Release before substitution; the caller keeps its original identity.
	if err := release(); err != nil {
		t.Fatal(err)
	}
	moved := owner.root + "-original"
	if err := os.Rename(owner.root, moved); err != nil {
		t.Fatal(err)
	}
	if err := filepath.Walk(moved, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(moved, p)
		if err != nil {
			return err
		}
		destination := filepath.Join(owner.root, relative)
		if info.IsDir() {
			return os.Mkdir(destination, 0700)
		}
		wire, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, wire, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	fresh, err := newRunStore(owner.root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := readExternalPacket(dir, packet.ID, fresh); err != nil || got.Digest != packet.Digest {
		t.Fatalf("replacement packet control: %+v %v", got, err)
	}
	if got, err := readExternalSubmission(dir, packet.ID, fresh); err != nil || got.ResponseHash != submission.ResponseHash {
		t.Fatalf("replacement submission control: %+v %v", got, err)
	}
	if got, err := readExternalPackets(dir, fresh); err != nil || len(got) != 1 {
		t.Fatalf("replacement list control: %+v %v", got, err)
	}
	if got, err := latestVerification(dir, verified.CandidateID, fresh); err != nil || got.Digest != verified.Digest {
		t.Fatalf("replacement verification control: %+v %v", got, err)
	}
	if got, err := persistedVerificationAttempt(dir, runID, verified.CandidateID, fresh); err != nil || !got {
		t.Fatalf("replacement attempt control: %v %v", got, err)
	}
	readers := map[string]func() error{
		"packet": func() error { _, err := readExternalPacket(dir, packet.ID, owner); return err },
		"packet leaf": func() error {
			_, err := readExternalPacketByPath(filepath.Join(dir, "external", "packets", packet.ID+".json"), owner)
			return err
		},
		"submission":   func() error { _, err := readExternalSubmission(dir, packet.ID, owner); return err },
		"packets":      func() error { _, err := readExternalPackets(dir, owner); return err },
		"verification": func() error { _, err := latestVerification(dir, verified.CandidateID, owner); return err },
		"attempt": func() error {
			_, err := persistedVerificationAttempt(dir, runID, verified.CandidateID, owner)
			return err
		},
	}
	for name, read := range readers {
		if err := read(); err == nil || !strings.Contains(err.Error(), "root identity changed") {
			t.Errorf("%s switched to replacement root: %v", name, err)
		}
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("original root lost: %v", err)
	}
}
