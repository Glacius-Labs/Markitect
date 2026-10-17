package run_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

var restoreLimits = run.RestoreLimits{PageSize: 1, Batches: 100, Events: 200, Bytes: 16 << 20}

// readJournal exposes stored serialized history through bounded reads. Writes
// are counted and refused, so a restore cannot quietly create/release anything.
type readJournal struct {
	store    *memoryJournal
	queries  []run.EventQuery
	blobRefs []run.BlobRef
	writes   int
	page     func(int, run.EventQuery, run.EventPage) (run.EventPage, error)
	blob     func(run.BlobRef, run.Payload) (run.Payload, error)
}

func (r *readJournal) Put(context.Context, run.Payload) (run.BlobRef, error) {
	r.writes++
	return run.BlobRef{}, errors.New("restore must not Put")
}
func (r *readJournal) Append(context.Context, run.AppendBatch) (run.AppendReceipt, error) {
	r.writes++
	return run.AppendReceipt{}, errors.New("restore must not Append")
}
func (r *readJournal) Read(ctx context.Context, q run.EventQuery) (run.EventPage, error) {
	r.queries = append(r.queries, q)
	p, err := r.store.Read(ctx, q)
	if err != nil {
		return p, err
	}
	if r.page != nil {
		return r.page(len(r.queries), q, p)
	}
	return p, nil
}
func (r *readJournal) ReadBlob(ctx context.Context, ref run.BlobRef) (run.Payload, error) {
	r.blobRefs = append(r.blobRefs, ref)
	p, err := r.store.ReadBlob(ctx, ref)
	if err != nil {
		return p, err
	}
	if r.blob != nil {
		return r.blob(ref, p)
	}
	return p, nil
}

func restored(t *testing.T, j run.Journal) *run.Coordinator {
	t.Helper()
	c, err := run.Restore(context.Background(), j, "repo", restoreLimits)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func sameProjection(t *testing.T, want, got *run.Coordinator) {
	t.Helper()
	if want.Revision() != got.Revision() || want.Current() != got.Current() || !reflect.DeepEqual(want.Budget(), got.Budget()) || !reflect.DeepEqual(want.Invocations(), got.Invocations()) {
		t.Fatalf("projection differs: original=%+v/%+v restored=%+v/%+v", want.Current(), want.Budget(), got.Current(), got.Budget())
	}
}

func TestRestorePreparedRunningAndFinishedFromStoredBytes(t *testing.T) {
	for _, phase := range []string{"empty", "prepared", "running", "finished"} {
		t.Run(phase, func(t *testing.T) {
			original, _ := run.NewCoordinator("repo")
			j := journal()
			if phase != "empty" {
				commit(t, original, j, run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
			}
			if phase == "running" || phase == "finished" {
				commit(t, original, j, command("start", run.StartRun))
			}
			if phase == "finished" {
				finishIncomplete(t, original, j)
			}
			reads := &readJournal{store: j}
			c := restored(t, reads)
			sameProjection(t, original, c)
			if c.RecoveryRequired() || reads.writes != 0 {
				t.Fatal("restore invented recovery effects")
			}
			if phase == "finished" {
				if _, err := c.Propose(command("restart", run.StartRun)); !errors.Is(err, run.ErrState) {
					t.Fatal("restored Finished changed", err)
				}
			}
		})
	}
}

func TestRestorePinsPagesAndRebuildsBudgetAndRetryReceipts(t *testing.T) {
	c, j := running(t)
	start := reserve("session", run.SessionStart, run.Assessment, time.Minute)
	startProposal, err := c.Propose(start)
	if err != nil {
		t.Fatal(err)
	}
	startReceipt, err := c.Commit(context.Background(), j, startProposal)
	if err != nil {
		t.Fatal(err)
	}
	settlement := settle("session", 20*time.Second)
	settledReceipt := commit(t, c, j, settlement)
	commit(t, c, j, reserve("step", run.AgentStep, run.Writing, time.Minute))
	commit(t, c, j, settle("step", 10*time.Second))
	reads := &readJournal{store: j}
	c2 := restored(t, reads)
	sameProjection(t, c, c2)
	if len(reads.queries) < 3 || len(reads.blobRefs) != 2 || reads.writes != 0 {
		t.Fatal("not restored from paged stored batches/blobs", len(reads.queries), reads.blobRefs)
	}
	for n, q := range reads.queries {
		if q.Limit != 1 || n == 0 && q.Through != 0 || n > 0 && q.Through != c.Revision() {
			t.Fatal("unpinned query", q)
		}
	}
	before := c2.Budget()
	revision := c2.Revision()
	// Both the original exact batch proposal and a fresh identical logical
	// command return their old receipts, without touching the read-only port.
	r, err := c2.Commit(context.Background(), reads, startProposal)
	if err != nil || r != startReceipt {
		t.Fatal("batch retry after restart", err)
	}
	p, err := c2.Propose(settlement)
	if err != nil {
		t.Fatal(err)
	}
	r, err = c2.Commit(context.Background(), reads, p)
	if err != nil || r != settledReceipt {
		t.Fatal("command retry after restart", err)
	}
	if c2.Revision() != revision || !reflect.DeepEqual(before, c2.Budget()) || reads.writes != 0 {
		t.Fatal("retry appended or charged twice")
	}
}

func TestRestoreKeepsPinnedHeadWhenLaterWorkCommits(t *testing.T) {
	c, j := running(t)
	head := c.Revision()
	view, budget := c.Current(), c.Budget()
	reads := &readJournal{store: j, page: func(n int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
		if n == 1 {
			stop := command("later-stop", run.StopRun)
			stop.Reason = "Later explicit request"
			commit(t, c, j, stop)
		}
		return p, nil
	}}
	c2 := restored(t, reads)
	if c2.Revision() != head || c2.Current() != view || !reflect.DeepEqual(c2.Budget(), budget) || reads.writes != 0 {
		t.Fatal("later append leaked into pinned projection")
	}
	for _, q := range reads.queries[1:] {
		if q.Through != head {
			t.Fatal("pin changed")
		}
	}
}

func TestRestoreHoldsOpenInvocationsBehindSeparateRecoveryBarrier(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(map[bool]string{false: "reserved", true: "uncertain"}[uncertain], func(t *testing.T) {
			c, j := running(t)
			oldCommand := reserve("writer", run.AgentStep, run.Writing, time.Minute)
			oldReceipt := commit(t, c, j, oldCommand)
			commit(t, c, j, reserve("check", run.ProjectCheck, run.Assessment, time.Minute))
			if uncertain {
				u := command("uncertain", run.ObserveUncertain)
				u.Invocation = "writer"
				u.Reason = "Host could not establish termination"
				commit(t, c, j, u)
			}
			reads := &readJournal{store: j}
			c2 := restored(t, reads)
			sameProjection(t, c, c2)
			if !c2.RecoveryRequired() || c2.Budget().Held.ExecutionTime != 2*time.Minute || reads.writes != 0 {
				t.Fatal("open holds were lost")
			}
			if _, err := c2.Propose(reserve("new", run.AgentStep, run.Assessment, time.Second)); !errors.Is(err, run.ErrRecovery) {
				t.Fatal("dependent work admitted after restart", err)
			}
			p, err := c2.Propose(oldCommand)
			if err != nil {
				t.Fatal(err)
			}
			r, err := c2.Commit(context.Background(), reads, p)
			if err != nil || r != oldReceipt {
				t.Fatal("read-only exact retry blocked", err)
			}
			commit(t, c2, j, settle("writer", 5*time.Second))
			if !c2.RecoveryRequired() || c2.Budget().Held.ExecutionTime != time.Minute {
				t.Fatal("barrier cleared while another hold remained")
			}
			release := command("release-check", run.ReleaseUndispatched)
			release.Invocation = "check"
			release.Reason = "Explicit Host inspection established no dispatch"
			proposal, err := c2.Propose(release)
			if err != nil {
				t.Fatal(err)
			}
			if !c2.RecoveryRequired() {
				t.Fatal("proposal cleared recovery")
			}
			j.appendMode = "lost"
			if _, err := c2.Commit(context.Background(), j, proposal); !errors.Is(err, run.ErrUncertain) {
				t.Fatal(err)
			}
			if !c2.RecoveryRequired() || c2.Budget().Held.ExecutionTime != time.Minute {
				t.Fatal("unconfirmed release cleared recovery")
			}
			j.appendMode = ""
			if err := c2.ReconcilePending(context.Background(), j); err != nil {
				t.Fatal(err)
			}
			if c2.RecoveryRequired() || c2.Budget().Held.ExecutionTime != 0 {
				t.Fatal("confirmed explicit reconciliation not adopted")
			}
			if uncertain && c2.Current().State != run.Reconciling {
				t.Fatal("restore or release invented lifecycle resume")
			}
		})
	}
}

func TestRestoreLostAppendACKUsesOnlyCommittedHistory(t *testing.T) {
	for _, mode := range []string{"lost", "absent"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := run.NewCoordinator("repo")
			j := journal()
			original := run.PrepareCommand("prepare", "a", "r", "", observed, binding(t))
			p, err := c.Propose(original)
			if err != nil {
				t.Fatal(err)
			}
			j.appendMode = mode
			if _, err := c.Commit(context.Background(), j, p); !errors.Is(err, run.ErrUncertain) {
				t.Fatal(err)
			}
			j.appendMode = ""
			reads := &readJournal{store: j}
			c2 := restored(t, reads)
			if reads.writes != 0 {
				t.Fatal("restore retried commit")
			}
			if mode == "lost" {
				if c2.Current().State != run.Prepared || c2.Budget().Consumed.Runs != 1 {
					t.Fatal("committed preparation not recovered")
				}
				r, err := c2.Commit(context.Background(), reads, p)
				if err != nil || r != run.ReceiptFor(p.Batch()) {
					t.Fatal("exact batch acknowledgement not recovered", err)
				}
			} else {
				if c2.Current().ID != "" || c2.Budget().Consumed.Runs != 0 {
					t.Fatal("phantom preparation")
				}
				// The caller explicitly decides to commit; read alone did not do it.
				commit(t, c2, j, original)
			}
		})
	}
}

func TestRestoreAdvancedHistoryDoesNotRetryAbsentOldProposal(t *testing.T) {
	c, _ := run.NewCoordinator("repo")
	j := journal()
	old, err := c.Propose(run.PrepareCommand("old", "a", "r", "", observed, binding(t)))
	if err != nil {
		t.Fatal(err)
	}
	j.appendMode = "absent"
	if _, err := c.Commit(context.Background(), j, old); !errors.Is(err, run.ErrUncertain) {
		t.Fatal(err)
	}
	j.appendMode = ""
	other, _ := run.NewCoordinator("repo")
	commit(t, other, j, run.PrepareCommand("different", "other", "other-run", "", observed, binding(t)))
	reads := &readJournal{store: j}
	c2 := restored(t, reads)
	before := c2.Budget()
	if _, err := c2.Commit(context.Background(), reads, old); !errors.Is(err, run.ErrConflict) {
		t.Fatal("absent stale proposal retried", err)
	}
	if reads.writes != 0 || !reflect.DeepEqual(before, c2.Budget()) {
		t.Fatal("stale retry affected history")
	}
}

func TestRestoreStaleAbsentReservationNeedsExplicitNewProposal(t *testing.T) {
	c, j := running(t)
	intent := reserve("absent", run.AgentStep, run.Assessment, time.Minute)
	old, err := c.Propose(intent)
	if err != nil {
		t.Fatal(err)
	}
	j.appendMode = "absent"
	if _, err := c.Commit(context.Background(), j, old); !errors.Is(err, run.ErrUncertain) {
		t.Fatal(err)
	}
	j.appendMode = ""
	other := restored(t, &readJournal{store: j})
	commit(t, other, j, reserve("known", run.SessionStart, run.Assessment, time.Second))
	commit(t, other, j, settle("known", 0))
	reads := &readJournal{store: j}
	c2 := restored(t, reads)
	before, rev := c2.Budget(), c2.Revision()
	if c2.Current().Basis != c.Current().Basis {
		t.Fatal("fixture changed the basis")
	}
	if _, err := c2.Commit(context.Background(), reads, old); !errors.Is(err, run.ErrConflict) {
		t.Fatal("absent stale intent silently rebased", err)
	}
	if reads.writes != 0 || c2.Revision() != rev || !reflect.DeepEqual(before, c2.Budget()) {
		t.Fatal("stale intent affected projection")
	}
	fresh, err := c2.Propose(intent)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Batch().ExpectedRevision() != rev || bytes.Equal(fresh.Batch().Bytes(), old.Batch().Bytes()) {
		t.Fatal("new proposal not bound to actual revision")
	}
	if _, err := c2.Commit(context.Background(), j, fresh); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreLostReservationACKKeepsCommittedIntentHeld(t *testing.T) {
	for _, mode := range []string{"lost", "absent"} {
		t.Run(mode, func(t *testing.T) {
			c, j := running(t)
			intent := reserve("step", run.AgentStep, run.Assessment, time.Minute)
			proposal, err := c.Propose(intent)
			if err != nil {
				t.Fatal(err)
			}
			j.appendMode = mode
			if _, err := c.Commit(context.Background(), j, proposal); !errors.Is(err, run.ErrUncertain) {
				t.Fatal(err)
			}
			j.appendMode = ""
			reads := &readJournal{store: j}
			c2 := restored(t, reads)
			if c2.Budget().Consumed.Runs != 1 || c2.Budget().Consumed.Steps != 0 || reads.writes != 0 {
				t.Fatal("restart invented consumption or effects")
			}
			if mode == "lost" {
				if !c2.RecoveryRequired() || c2.Budget().Held.Steps != 1 || c2.Budget().Held.ExecutionTime != time.Minute {
					t.Fatal("committed intent not held")
				}
				if _, err := c2.Commit(context.Background(), reads, proposal); err != nil {
					t.Fatal("exact batch retry unavailable", err)
				}
				if !c2.RecoveryRequired() || c2.Budget().Held.Steps != 1 || reads.writes != 0 {
					t.Fatal("retry dispatched or cleared hold")
				}
			} else if c2.RecoveryRequired() || c2.Budget().Held.Steps != 0 {
				t.Fatal("phantom reservation")
			}
		})
	}
}

func alteredBatch(t *testing.T, b run.AppendBatch, change func(*struct {
	Version          uint32
	ID               string
	ExpectedRevision uint64
	Events           []run.Event
})) run.AppendBatch {
	t.Helper()
	var wire struct {
		Version          uint32
		ID               string
		ExpectedRevision uint64
		Events           []run.Event
	}
	if err := json.Unmarshal(b.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	change(&wire)
	data, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	out, err := run.DecodeBatch(data)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRestoreRejectsMalformedOrUnavailablePages(t *testing.T) {
	c, j := running(t)
	commit(t, c, j, reserve("step", run.AgentStep, run.Assessment, time.Minute))
	commit(t, c, j, settle("step", time.Second))
	for name, hook := range map[string]func(int, run.EventQuery, run.EventPage) (run.EventPage, error){
		"unreachable": func(_ int, _ run.EventQuery, _ run.EventPage) (run.EventPage, error) {
			return run.EventPage{}, errors.New("unreachable")
		},
		"head changes": func(n int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			if n > 1 {
				p.Revision++
			}
			return p, nil
		},
		"unsupported head": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			p.Revision = run.MaxRevision + 1
			return p, nil
		},
		"truncated page": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			p.Batches = nil
			p.Next = 0
			return p, nil
		},
		"empty unfinished page": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) { p.Batches = nil; return p, nil },
		"wrong cursor":          func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) { p.Next++; return p, nil },
		"overlapping batch": func(n int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			if n == 2 {
				p.Batches[0] = j.batches[0]
			}
			return p, nil
		},
		"revision gap": func(n int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			if n == 2 {
				p.Batches[0] = alteredBatch(t, p.Batches[0], func(w *struct {
					Version          uint32
					ID               string
					ExpectedRevision uint64
					Events           []run.Event
				}) {
					w.ExpectedRevision++
				})
			}
			return p, nil
		},
		"partial atomic batch": func(n int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			if n == 1 {
				p.Batches[0] = alteredBatch(t, p.Batches[0], func(w *struct {
					Version          uint32
					ID               string
					ExpectedRevision uint64
					Events           []run.Event
				}) {
					w.Events = w.Events[:1]
				})
			}
			return p, nil
		},
		"too many batches": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			p.Batches = append(p.Batches, p.Batches[0])
			return p, nil
		},
		"unknown version": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			_, err := run.DecodeBatch(bytes.Replace(p.Batches[0].Bytes(), []byte(`"Version":1`), []byte(`"Version":2`), 1))
			return run.EventPage{}, err
		},
		"unknown event": func(_ int, _ run.EventQuery, p run.EventPage) (run.EventPage, error) {
			_, err := run.DecodeBatch(bytes.Replace(p.Batches[0].Bytes(), []byte(`"Kind":"assignment.created"`), []byte(`"Kind":"unknown"`), 1))
			return run.EventPage{}, err
		},
	} {
		t.Run(name, func(t *testing.T) {
			reads := &readJournal{store: j, page: hook}
			out, err := run.Restore(context.Background(), reads, "repo", restoreLimits)
			if err == nil || out != nil || reads.writes != 0 {
				t.Fatal("bad history became executable", err)
			}
		})
	}
}

func TestKnownPendingBatchAtLaterHeadRequiresWholeRestore(t *testing.T) {
	c, _ := run.NewCoordinator("repo")
	j := journal()
	p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
	if err != nil {
		t.Fatal(err)
	}
	j.appendMode = "lost"
	if _, err := c.Commit(context.Background(), j, p); !errors.Is(err, run.ErrUncertain) {
		t.Fatal(err)
	}
	j.appendMode = ""
	other := restored(t, j)
	commit(t, other, j, command("start", run.StartRun))
	if err := c.ReconcilePending(context.Background(), j); !errors.Is(err, run.ErrConflict) {
		t.Fatal("advanced history skipped", err)
	}
	if c.Current().ID != "" || c.Budget().Consumed.Runs != 0 {
		t.Fatal("old projection promoted past unknown later work")
	}
	c2 := restored(t, j)
	sameProjection(t, other, c2)
}

func TestRestoreRejectsMissingOrChangedRequiredBlobs(t *testing.T) {
	for _, kind := range []string{"binding missing", "owner missing", "binding changed", "owner changed"} {
		t.Run(kind, func(t *testing.T) {
			c, j := running(t)
			basis := c.Current().Basis
			var wire struct{ OwnerRequest run.BlobRef }
			if err := json.Unmarshal(j.blobs[basis].Bytes(), &wire); err != nil {
				t.Fatal(err)
			}
			target := basis
			if kind == "owner missing" || kind == "owner changed" {
				target = wire.OwnerRequest
			}
			if kind == "binding missing" || kind == "owner missing" {
				delete(j.blobs, target)
			}
			reads := &readJournal{store: j, blob: func(ref run.BlobRef, p run.Payload) (run.Payload, error) {
				if ref == target {
					p, _ = run.NewPayload(p.Ref().MediaType, p.Ref().Encoding, []byte("changed original bytes"))
				}
				return p, nil
			}}
			out, err := run.Restore(context.Background(), reads, "repo", restoreLimits)
			if err == nil || out != nil || reads.writes != 0 {
				t.Fatal("bad blob became executable", err)
			}
		})
	}
}

func TestRestoreEnforcesFiniteBudgetsAndRepositoryIdentity(t *testing.T) {
	_, j := running(t)
	for _, limits := range []run.RestoreLimits{
		{}, {PageSize: 257, Batches: 100, Events: 200, Bytes: 16 << 20},
		{PageSize: 1, Batches: 1, Events: 200, Bytes: 16 << 20},
		{PageSize: 1, Batches: 100, Events: 1, Bytes: 16 << 20},
		{PageSize: 1, Batches: 100, Events: 200, Bytes: 1},
	} {
		reads := &readJournal{store: j}
		c, err := run.Restore(context.Background(), reads, "repo", limits)
		if err == nil || c != nil || reads.writes != 0 {
			t.Fatal("read budget ignored", limits, err)
		}
	}
	if c, err := run.Restore(context.Background(), j, "another-repository", restoreLimits); err == nil || c != nil {
		t.Fatal("foreign repository became executable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if c, err := run.Restore(ctx, j, "repo", restoreLimits); err == nil || c != nil {
		t.Fatal("canceled read became executable")
	}
}

func TestRestoreCountsRequiredBlobBytesInItsReadBudget(t *testing.T) {
	_, j := running(t)
	var total int64
	for _, b := range j.batches {
		total += int64(len(b.Bytes()))
	}
	for ref := range j.blobs {
		total += ref.Size
	}
	limits := restoreLimits
	limits.Bytes = total - 1
	reads := &readJournal{store: j}
	if c, err := run.Restore(context.Background(), reads, "repo", limits); !errors.Is(err, run.ErrRestoreLimit) || c != nil {
		t.Fatal("required blob bytes escaped read budget")
	}
	limits.Bytes = total
	if c, err := run.Restore(context.Background(), j, "repo", limits); err != nil || c == nil {
		t.Fatal("exact finite read budget refused", err)
	}
}

func TestRestoreContinuationKeepsAssignmentOwnerAndConsumedLedger(t *testing.T) {
	c, j := running(t)
	commit(t, c, j, reserve("step", run.AgentStep, run.Assessment, time.Minute))
	commit(t, c, j, settle("step", time.Second))
	finishIncomplete(t, c, j)
	i := fixture(t)
	i.Source.Guard = "changed-source"
	b, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	commit(t, c, j, run.PrepareCommand("continue", "a", "r2", "r", observed, b))
	reads := &readJournal{store: j}
	c2 := restored(t, reads)
	sameProjection(t, c, c2)
	if len(reads.blobRefs) != 3 {
		t.Fatal("continuation did not reuse immutable owner blob", len(reads.blobRefs))
	}
	f := command("finish-r2", run.FinishRun)
	f.Run = "r2"
	f.Outcome = run.Incomplete
	f.Reason = "Retain work"
	commit(t, c2, j, f)
	i.OwnerRequest, _ = run.NewPayload("text/plain", "utf-8", []byte("Changed assignment"))
	i.Source.Guard = "another-source"
	b, err = run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.Propose(run.PrepareCommand("different-owner", "a", "r3", "r2", observed, b)); !errors.Is(err, run.ErrState) {
		t.Fatal("restored assignment lost owner binding", err)
	}
}
