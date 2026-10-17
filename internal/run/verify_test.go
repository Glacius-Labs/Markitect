package run_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type fakeChecks struct {
	exit int
	runs []run.CheckRequest
}

func (f *fakeChecks) Run(_ context.Context, r run.CheckRequest) (run.CheckReceipt, error) {
	f.runs = append(f.runs, r)
	return run.CheckReceipt{ID: r.ID, ImageID: r.ImageID, Tree: r.Tree.Digest(), Started: true, Exited: true, ExitCode: f.exit, Output: []byte("ok\n")}, nil
}

func requiredCheck(t *testing.T) run.RequiredCheck {
	t.Helper()
	dir := snapshot.RootDirectory()
	return run.RequiredCheck{Name: "go-tests", Request: run.CheckRequest{ID: "check-go-tests", ImageID: testImage, Executable: "/usr/local/go/bin/go", Args: []string{"test", "./..."},
		Directory: dir, Limits: run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 1 << 30, Processes: 256}, Timeout: time.Minute, MaxOutputBytes: 1 << 16}}
}

func verifiedHarness(t *testing.T, exit int) (*harness, *fakeChecks) {
	t.Helper()
	h := newHarness(t, true, happyAgent)
	checks := &fakeChecks{exit: exit}
	h.engine = h.newEngineWith(h.c, checks)
	runToEnd(t, h)
	if h.c.Current().State != run.Verifying {
		t.Fatal("run did not reach Verifying", h.c.Current())
	}
	passed, err := h.engine.Verify(context.Background(), []run.RequiredCheck{requiredCheck(t)})
	if err != nil {
		t.Fatal(err)
	}
	if h.c.Current().State == run.Verifying {
		commit(t, h.c, h.j, h.c.VerifyRunCommand(meta("verify"), passed))
	}
	return h, checks
}

// The required checks run on the exact aggregate before publication, and the
// final evidence check moves the run to Publishing (execution contract,
// Verifying).
func TestVerificationRunsRequiredChecksOnTheAggregate(t *testing.T) {
	h, checks := verifiedHarness(t, 0)
	if h.c.Current().State != run.Publishing || len(checks.runs) != 1 || checks.runs[0].Tree.Digest() != h.c.AggregateTree() {
		t.Fatal("verification did not check the aggregate", h.c.Current(), checks.runs)
	}
	if !h.engine.Aggregate().Tree().Equal(checks.runs[0].Tree) {
		t.Fatal("the checked tree is not the engine's aggregate")
	}
	views := h.c.Checks()
	if len(views) != 1 || !views[0].Passed || views[0].Record.Receipt == (run.BlobRef{}) {
		t.Fatal("check receipt not recorded", views)
	}
	if b := h.c.Budget(); b.Held != (run.Capacity{}) {
		t.Fatal("check reservation not settled", b.Held)
	}
}

func TestFailingRequiredCheckFinishesTheRunIncomplete(t *testing.T) {
	h, _ := verifiedHarness(t, 1)
	if r := h.c.Current(); r.State != run.Finished || r.Outcome != run.Incomplete {
		t.Fatal("failing check did not finish the run Incomplete", r)
	}
}

// Publication records its changes before the first write; its receipt
// decides the outcome (execution contract, Publishing and outcomes).
func TestPublicationOutcomes(t *testing.T) {
	for _, tc := range []struct {
		outcome string
		empty   bool
		state   run.State
		result  run.Outcome
		err     error
	}{
		{"applied", false, run.Finished, run.Applied, nil},
		{"applied", true, "", "", run.ErrInvalid},
		{"no_change", true, run.Finished, run.NoChange, nil},
		{"no_change", false, "", "", run.ErrInvalid},
		{"stale", false, run.Finished, run.Stale, nil},
		{"failed", false, run.Finished, run.Failed, nil},
		{"partial", false, run.Reconciling, "", nil},
		{"unknown", false, run.Reconciling, "", nil},
	} {
		t.Run(tc.outcome+map[bool]string{true: "-empty"}[tc.empty], func(t *testing.T) {
			h, _ := verifiedHarness(t, 0)
			source := h.inputs.Binding.Census
			changes := snapshot.Compare(source, h.engine.Aggregate().Tree())
			if tc.empty {
				changes = nil
			}
			if !tc.empty && len(changes) == 0 {
				t.Fatal("the aggregate has no changes")
			}
			prepare, err := h.c.PreparePublicationCommand(meta("prepare-publication"), "publication-1", changes)
			if err != nil {
				t.Fatal(err)
			}
			commit(t, h.c, h.j, prepare)
			publish, err := h.c.PublishRunCommand(meta("published"), run.PublicationResult{ID: "publication-1", Outcome: tc.outcome})
			if err != nil {
				t.Fatal(err)
			}
			if tc.err != nil {
				if _, err := h.c.Propose(publish); !errors.Is(err, tc.err) {
					t.Fatal("publication outcome accepted against its changes", err)
				}
				return
			}
			commit(t, h.c, h.j, publish)
			if r := h.c.Current(); r.State != tc.state || r.Outcome != tc.result {
				t.Fatal("wrong run outcome", r)
			}
			restored, err := run.Restore(context.Background(), &readJournal{store: h.j}, "repo", run.RestoreLimits{PageSize: 16, Batches: 2000, Events: 4000, Bytes: 64 << 20})
			if err != nil || restored.Current() != h.c.Current() || !reflect.DeepEqual(restored.Checks(), h.c.Checks()) {
				t.Fatal("restore lost verification or publication", err)
			}
		})
	}
}

func TestVerificationRefusesUnadmittedChecks(t *testing.T) {
	h := newHarness(t, true, happyAgent)
	check := requiredCheck(t)
	check.Request.Tree = h.inputs.Binding.Census
	if cmd, err := h.c.PrepareCheckCommand(meta("early-check"), check.Name, check.Request, run.Reservation{ID: "early", Kind: run.ProjectCheck, Purpose: run.Assessment, Maximum: time.Minute}); err != nil {
		t.Fatal(err)
	} else if _, err := h.c.Propose(cmd); err == nil {
		t.Fatal("a check ran while the run was not Verifying")
	}
	if _, err := h.c.Propose(h.c.VerifyRunCommand(meta("early-verify"), nil)); !errors.Is(err, run.ErrState) {
		t.Fatal("a running run was verified", err)
	}
}
