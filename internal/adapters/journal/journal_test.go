package journal

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func request(t *testing.T) apply.ControlRequest {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Write(".markitect/model.yaml", "formatVersion: 1\n")
	repo.Write(".gitignore", ".markitect/state/\n")
	head := repo.Commit("fixture")
	return apply.ControlRequest{Repository: repo.Dir, RepositoryIdentity: "host-repository", Revision: head, Guard: "opaque-original-guard", StateDirectory: filepath.Join(testkit.TempDir(t), "store")}
}

func acquire(t *testing.T, r apply.ControlRequest) *session {
	t.Helper()
	s, err := New().Acquire(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s.(*session)
}

// These are syntactically valid producer records. The storage adapter does not
// decide whether a command is admitted by the reducer; integration tests do.
func batch(t *testing.T, id string, after uint64, eventIDs ...string) run.AppendBatch {
	t.Helper()
	w := struct {
		Version          uint32
		ID               string
		ExpectedRevision uint64
		Events           []run.Event
	}{Version: 1, ID: id, ExpectedRevision: after}
	for _, id := range eventIDs {
		w.Events = append(w.Events, run.Event{ID: id, Kind: "run.state-changed", Body: run.Command{ID: id, Kind: run.StartRun, Assignment: "a", Run: "r", ObservedAt: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)}})
	}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	b, err := run.DecodeBatch(data)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAtomicAppendExactRetryCASAndPinnedWholeBatchReads(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	ctx := context.Background()
	b := batch(t, "first", 0, "e1", "e2")
	receipt, err := s.Append(ctx, b)
	if err != nil || !receipt.Matches(b) {
		t.Fatal(receipt, err)
	}
	b2 := batch(t, "second", 2, "e3")
	if _, err := s.Append(ctx, b2); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Append(ctx, b); err != nil || got != receipt {
		t.Fatal("historical exact retry", got, err)
	}
	for _, bad := range []run.AppendBatch{batch(t, "first", 0, "changed"), batch(t, "stale", 0, "new"), batch(t, "duplicate-event", 3, "e1")} {
		known, err := s.Append(ctx, bad)
		if !errors.Is(err, run.ErrConflict) {
			t.Fatal("conflict accepted", err)
		}
		if bad.ID() == "first" && known != receipt {
			t.Fatal("known original receipt lost beside conflict", known)
		}
	}
	page, err := s.Read(ctx, run.EventQuery{Limit: 1})
	if err != nil || page.Revision != 3 || page.Next != 2 || len(page.Batches) != 1 {
		t.Fatal(page, err)
	}
	pinned := page.Revision
	if _, err := s.Append(ctx, batch(t, "third", 3, "e4")); err != nil {
		t.Fatal(err)
	}
	page, err = s.Read(ctx, run.EventQuery{After: 2, Through: pinned, Limit: 256})
	if err != nil || page.Revision != 3 || page.Next != 0 || len(page.Batches) != 1 || page.Batches[0].ID() != "second" {
		t.Fatal(page, err)
	}
	for _, q := range []run.EventQuery{{After: 1, Limit: 1}, {Through: 1, Limit: 1}, {Through: 5, Limit: 1}, {After: 5, Limit: 1}, {Through: run.MaxRevision + 1, Limit: 1}} {
		if _, err := s.Read(ctx, q); !errors.Is(err, run.ErrInvalid) {
			t.Fatal("bad boundary", q, err)
		}
	}
	page, err = s.Read(ctx, run.EventQuery{BatchID: "first", Limit: 1})
	if err != nil || page.Revision != 4 || len(page.Batches) != 1 {
		t.Fatal(page, err)
	}
	page, err = s.Read(ctx, run.EventQuery{BatchID: "absent", Limit: 1})
	if err != nil || page.Revision != 4 || len(page.Batches) != 0 {
		t.Fatal(page, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = acquire(t, r)
	page, err = s.Read(ctx, run.EventQuery{Limit: 256})
	if err != nil || page.Revision != 4 || len(page.Batches) != 3 {
		t.Fatal("reopen history", page, err)
	}
}

func TestImmutableBlobInterpretationsAndCorruption(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	ctx := context.Background()
	var refs []run.BlobRef
	for _, kind := range []string{"text/plain", "application/octet-stream"} {
		p, _ := run.NewPayload(kind, "identity", []byte("original"))
		ref, err := s.Put(ctx, p)
		if err != nil || ref != p.Ref() {
			t.Fatal(ref, err)
		}
		refs = append(refs, ref)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = acquire(t, r)
	for _, ref := range refs {
		p, err := s.ReadBlob(ctx, ref)
		if err != nil || p.Ref() != ref || !bytes.Equal(p.Bytes(), []byte("original")) {
			t.Fatal(err)
		}
	}
	bad := refs[0]
	bad.Size++
	if _, err := s.ReadBlob(ctx, bad); !errors.Is(err, ErrCorrupt) {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE blobs SET data=?", []byte("tampered")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadBlob(ctx, refs[0]); !errors.Is(err, ErrCorrupt) {
		t.Fatal("changed bytes accepted", err)
	}
}

func TestReadByteBudgetStopsOnlyAtWholeBatchBoundaries(t *testing.T) {
	s := acquire(t, request(t))
	ctx := context.Background()
	large := strings.Repeat("x", 5<<20)
	for n := range 2 {
		id := fmt.Sprintf("large-%d", n)
		w := struct {
			Version          uint32
			ID               string
			ExpectedRevision uint64
			Events           []run.Event
		}{1, id, uint64(n), []run.Event{{ID: id, Kind: "run.state-changed", Body: run.Command{ID: id, Kind: run.WaitRun, Assignment: "a", Run: "r", ObservedAt: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), Reason: large, NextActor: "owner"}}}}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatal(err)
		}
		b, err := run.DecodeBatch(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Append(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.Read(ctx, run.EventQuery{Limit: 256})
	if err != nil || page.Revision != 2 || page.Next != 1 || len(page.Batches) != 1 {
		t.Fatal("byte budget exposed partial or oversized page", page.Revision, page.Next, len(page.Batches), err)
	}
	page, err = s.Read(ctx, run.EventQuery{After: page.Next, Through: page.Revision, Limit: 256})
	if err != nil || page.Revision != 2 || page.Next != 0 || len(page.Batches) != 1 || page.Batches[0].ID() != "large-1" {
		t.Fatal("whole-batch continuation", page.Revision, page.Next, len(page.Batches), err)
	}
}

func TestControlSerializesDirectoriesAndFencesEveryOperation(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	original := s.Binding()
	other := r
	other.StateDirectory = filepath.Join(testkit.TempDir(t), "other")
	if unexpected, err := New().Acquire(context.Background(), other); err == nil {
		unexpected.Close()
		t.Fatal("alternate state bypassed lock")
	}
	if _, err := s.conn.ExecContext(context.Background(), "UPDATE store_meta SET epoch=epoch+1"); err != nil {
		t.Fatal(err)
	}
	assertFenced(t, s)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Acquire(context.Background(), other); !errors.Is(err, ErrReconcile) {
		t.Fatal("alternate store hid history", err)
	}
	s = acquire(t, r)
	if s.Binding().Epoch != original.Epoch+2 || s.Binding().StoreID != original.StoreID {
		t.Fatal("epoch or store reset", s.Binding())
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Check(context.Background()); !errors.Is(err, apply.ErrControlClosed) {
		t.Fatal(err)
	}
}

func assertFenced(t *testing.T, s *session) {
	t.Helper()
	ctx := context.Background()
	p, _ := run.NewPayload("text/plain", "utf-8", []byte("x"))
	for name, call := range map[string]func() error{
		"Check":    func() error { return s.Check(ctx) },
		"Put":      func() error { _, err := s.Put(ctx, p); return err },
		"ReadBlob": func() error { _, err := s.ReadBlob(ctx, p.Ref()); return err },
		"Append":   func() error { _, err := s.Append(ctx, batch(t, "fenced", 0, "fenced-event")); return err },
		"Read":     func() error { _, err := s.Read(ctx, run.EventQuery{Limit: 1}); return err },
	} {
		if err := call(); !errors.Is(err, apply.ErrControlBinding) {
			t.Fatalf("%s not fenced: %v", name, err)
		}
	}
}

func TestMissingAndCopiedPhysicalStoreNeverBecomeEmptyHistory(t *testing.T) {
	for _, change := range []string{"missing database", "copied database", "missing registration", "missing registration alternate store", "replaced registration", "copied checkout", "host assertion"} {
		t.Run(change, func(t *testing.T) {
			r := request(t)
			s := acquire(t, r)
			if _, err := s.Append(context.Background(), batch(t, "retained", 0, "e1")); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(r.StateDirectory, "journal.sqlite")
			switch change {
			case "missing database":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "copied database":
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			case "missing registration", "missing registration alternate store":
				if err := os.Remove(filepath.Join(r.Repository, ".markitect", "state", "journal.json")); err != nil {
					t.Fatal(err)
				}
				if change == "missing registration alternate store" {
					r.StateDirectory = filepath.Join(testkit.TempDir(t), "alternate")
				}
			case "replaced registration":
				reg := filepath.Join(r.Repository, ".markitect", "state", "journal.json")
				data, err := os.ReadFile(reg)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(reg, reg+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(reg, data, 0600); err != nil {
					t.Fatal(err)
				}
			case "copied checkout":
				other := request(t)
				data, err := os.ReadFile(filepath.Join(r.Repository, ".markitect", "state", "journal.json"))
				if err != nil {
					t.Fatal(err)
				}
				control := filepath.Join(other.Repository, ".markitect", "state")
				if err := mkdirPrivate(control); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(control, "journal.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
				// A lock file alone is no authority. Preserve it with copied metadata.
				if err := os.WriteFile(filepath.Join(control, "journal.lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
				r.Repository = other.Repository
			case "host assertion":
				r.RepositoryIdentity = "different-host-assertion"
			}
			if got, err := New().Acquire(context.Background(), r); err == nil {
				got.Close()
				t.Fatal("lost or copied history authorized")
			}
		})
	}
}

func TestBusyRollbackAndAdmissionCancellationAreBounded(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	ctx := context.Background()
	if err := s.settings(ctx); err != nil {
		t.Fatal(err)
	}
	// A real second SQLite connection holds the write lock. No dependent
	// receipt is acknowledged when append cannot enter its transaction.
	db, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(r.StateDirectory, "journal.sqlite"))+"?_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	receipt, err := s.Append(ctx, batch(t, "busy", 0, "e1"))
	if err == nil || receipt.BatchID != "" || time.Since(start) > 2*time.Second {
		t.Fatal("unbounded or acknowledged busy", receipt, err, time.Since(start))
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	// SQL failure after INSERTs rolls the entire batch back, including event IDs.
	if _, err := s.conn.ExecContext(ctx, "CREATE TRIGGER fail_head BEFORE UPDATE ON stream_head BEGIN SELECT RAISE(ABORT,'injected head failure'); END"); err != nil {
		t.Fatal(err)
	}
	b := batch(t, "atomic", 0, "e1", "e2")
	if _, err := s.Append(ctx, b); err == nil {
		t.Fatal("trigger failure ignored")
	}
	page, err := s.Read(ctx, run.EventQuery{BatchID: b.ID(), Limit: 1})
	if err != nil || page.Revision != 0 || len(page.Batches) != 0 {
		t.Fatal(page, err)
	}
	if _, err := s.conn.ExecContext(ctx, "DROP TRIGGER fail_head"); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Append(ctx, b); err != nil || !got.Matches(b) {
		t.Fatal(got, err)
	}
	s.gate <- struct{}{}
	cancelled, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	_, err = s.Read(cancelled, run.EventQuery{Limit: 1})
	<-s.gate
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("mutex admission ignored cancellation", err)
	}
}

func TestEmptyMaximumRevisionAndCorruptBatch(t *testing.T) {
	s := acquire(t, request(t))
	ctx := context.Background()
	page, err := s.Read(ctx, run.EventQuery{Limit: 1})
	if err != nil || page.Revision != 0 || page.Next != 0 || len(page.Batches) != 0 {
		t.Fatal(page, err)
	}
	b := batch(t, "last", run.MaxRevision-1, "last-event")
	// A synthetic near-limit store tests SQL integer conversion without an
	// impossible number of prior writes; it is deliberately not RUN history.
	digest := b.ContentID()
	if _, err := s.conn.ExecContext(ctx, "INSERT INTO batches VALUES(?,?,?,?,?)", b.ID(), digest[:], b.ExpectedRevision(), b.Revision(), b.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "INSERT INTO events VALUES(?,?,?,0)", "last-event", b.ID(), b.Revision()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE stream_head SET revision=?", run.MaxRevision); err != nil {
		t.Fatal(err)
	}
	page, err = s.Read(ctx, run.EventQuery{BatchID: "last", Limit: 1})
	if err != nil || page.Revision != run.MaxRevision || len(page.Batches) != 1 {
		t.Fatal(page, err)
	}
	if got, err := s.Append(ctx, b); err != nil || !got.Matches(b) {
		t.Fatal(got, err)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE batches SET digest=zeroblob(32)"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(ctx, run.EventQuery{BatchID: "last", Limit: 1}); !errors.Is(err, ErrCorrupt) {
		t.Fatal("corrupt batch accepted", err)
	}
}

func TestRegistrationInitializationAndInsecurePathsRefuse(t *testing.T) {
	r := request(t)
	if err := mkdirPrivate(filepath.Join(r.Repository, ".markitect", "state")); err != nil {
		t.Fatal(err)
	}
	reg := registration{Version: 1, Repository: r.RepositoryIdentity, StoreID: fmt.Sprintf("%064d", 1), Status: "initializing"}
	if err := createRegistration(filepath.Join(r.Repository, ".markitect", "state", "journal.json"), reg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Repository, ".markitect", "state", "journal.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Acquire(context.Background(), r); !errors.Is(err, ErrReconcile) {
		t.Fatal("interrupted initialization authorized")
	}
	r = request(t)
	r.StateDirectory = filepath.Join(r.Repository, "inside")
	if _, err := New().Acquire(context.Background(), r); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
}
