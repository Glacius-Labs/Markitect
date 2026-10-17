package journal

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var integrationAt = time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)

func integrationCommit(t *testing.T, c *apply.ControlledRun, command run.Command) run.AppendBatch {
	t.Helper()
	p, err := c.Coordinator().Propose(command)
	if err != nil {
		t.Fatal("propose", err)
	}
	receipt, err := c.Coordinator().Commit(context.Background(), c.Journal(), p)
	if err != nil || !receipt.Matches(p.Batch()) {
		t.Fatal("commit", receipt, err)
	}
	return p.Batch()
}

func integrationBindingRefs(t *testing.T, batch run.AppendBatch, basis run.Binding) (run.BlobRef, run.BlobRef) {
	t.Helper()
	refs, err := batch.BindingRefs()
	if err != nil || len(refs) != 1 || refs[0] != basis.Ref() {
		t.Fatal("producer basis references", refs, err)
	}
	owner, err := run.BindingOwnerRef(basis.Payload())
	if err != nil {
		t.Fatal("producer owner reference", err)
	}
	return refs[0], owner
}

func TestControlServiceRestoresDurableRunAndRecoveryBarrier(t *testing.T) {
	f := newIntegrationFixture(t)
	c := f.open(t)
	if c.Coordinator().Revision() != 0 || c.Coordinator().RecoveryRequired() {
		t.Fatal("opening created a run or recovery hold")
	}
	prepared := integrationCommit(t, c, run.PrepareCommand("prepare", "assignment", "run", "", integrationAt, c.Basis()))
	basis, owner := integrationBindingRefs(t, prepared, c.Basis())
	for _, ref := range []run.BlobRef{basis, owner} {
		payload, err := c.Journal().ReadBlob(context.Background(), ref)
		if err != nil || payload.Ref() != ref {
			t.Fatal("required preparation bytes not retained", err)
		}
	}
	integrationCommit(t, c, run.Command{ID: "start", Kind: run.StartRun, Assignment: "assignment", Run: "run", ObservedAt: integrationAt})
	reserve := integrationExploreStartup(t, c.Coordinator(), f.source.inputs.Raw.Census.Digest())
	integrationCommit(t, c, reserve)
	want, binding := c.Coordinator(), c.Binding()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	// An equivalent recapture may issue another guard; the prepared basis stays
	// bound to the original observation rather than this new comparison token.
	f.source.inputs.Guard = "equivalent-recapture-guard"
	reopened := f.open(t)
	got := reopened.Coordinator()
	if reopened.Binding().StoreID != binding.StoreID || reopened.Binding().Epoch <= binding.Epoch || reopened.Basis().Ref() != basis ||
		got.Revision() != want.Revision() || got.Current() != want.Current() || !reflect.DeepEqual(got.Budget(), want.Budget()) || !reflect.DeepEqual(got.Invocations(), want.Invocations()) || !got.RecoveryRequired() {
		t.Fatal("reopen lost registered history, binding, budget or unresolved invocation")
	}
	page, err := reopened.Journal().Read(context.Background(), run.EventQuery{Limit: 256})
	if err != nil || page.Revision != want.Revision() || len(page.Batches) != 3 || page.Next != 0 {
		t.Fatal("restore changed the committed stream", page, err)
	}
	reserve.ID, reserve.Reservation.ID = "new-reserve", "new-invocation"
	if _, err := got.Propose(reserve); !errors.Is(err, run.ErrRecovery) {
		t.Fatal("restored hold allowed new dispatch", err)
	}
	io, err := got.SessionChannel(reopened.Journal(), "session", run.RefuseTools)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ObserveStop(context.Background(), run.StopObservation{RecordMeta: run.RecordMeta{ID: "proven-not-started", ObservedAt: integrationAt}, Terminated: true, Reason: "Host proved no process was dispatched in this fixture"}); err != nil {
		t.Fatal(err)
	}
	integrationCommit(t, reopened, run.Command{ID: "release", Kind: run.ReleaseUndispatched, Assignment: "assignment", Run: "run", ObservedAt: integrationAt,
		Invocation: "invocation", Reason: "The test issued no external dispatch."})
	if got.RecoveryRequired() || got.Budget().Held.ExecutionTime != 0 || got.Current().State != run.Running {
		t.Fatal("confirmed explicit reconciliation did not clear only the hold")
	}
}

func TestPreparedAppendRequiresProducerBasisAndOwnerBytes(t *testing.T) {
	for _, missing := range []string{"basis", "owner"} {
		t.Run(missing, func(t *testing.T) {
			f := newIntegrationFixture(t)
			c := f.open(t)
			p, err := c.Coordinator().Propose(run.PrepareCommand("prepare", "assignment", "run", "", integrationAt, c.Basis()))
			if err != nil {
				t.Fatal(err)
			}
			basis, owner := integrationBindingRefs(t, p.Batch(), c.Basis())
			if owner != f.facts.OwnerRequest.Ref() {
				t.Fatal("producer owner reference differs from original input")
			}
			present, absent := c.Basis().Payload(), f.facts.OwnerRequest
			if missing == "basis" {
				present, absent = absent, present
			}
			if _, err := c.Journal().Put(context.Background(), present); err != nil {
				t.Fatal(err)
			}
			if _, err := c.Journal().Append(context.Background(), p.Batch()); !errors.Is(err, ErrCorrupt) {
				t.Fatal("append accepted missing required bytes", err)
			}
			page, err := c.Journal().Read(context.Background(), run.EventQuery{Limit: 1})
			if err != nil || page.Revision != 0 || len(page.Batches) != 0 {
				t.Fatal("failed append exposed partial preparation", page, err)
			}
			if _, err := c.Journal().Put(context.Background(), absent); err != nil {
				t.Fatal(err)
			}
			if receipt, err := c.Journal().Append(context.Background(), p.Batch()); err != nil || !receipt.Matches(p.Batch()) {
				t.Fatal("complete bytes refused", receipt, err)
			}
			if err := c.Close(); err != nil {
				t.Fatal(err)
			}
			restored := f.open(t).Coordinator()
			if restored.Current().Basis != basis || restored.Current().State != run.Prepared || restored.Revision() != p.Batch().Revision() {
				t.Fatal("complete producer preparation was not restored")
			}
		})
	}
}

func TestControlServiceRefusesMissingOrCorruptRecordedPreparationBytes(t *testing.T) {
	for _, target := range []string{"basis", "owner"} {
		for _, damage := range []string{"missing", "corrupt"} {
			t.Run(target+"/"+damage, func(t *testing.T) {
				f := newIntegrationFixture(t)
				c := f.open(t)
				prepared := integrationCommit(t, c, run.PrepareCommand("prepare", "assignment", "run", "", integrationAt, c.Basis()))
				basis, owner := integrationBindingRefs(t, prepared, c.Basis())
				ref := basis
				if target == "owner" {
					ref = owner
				}
				if err := c.Close(); err != nil {
					t.Fatal(err)
				}
				// Controlled fault injection changes only this owned real database.
				s := acquire(t, f.request)
				payload, err := s.ReadBlob(context.Background(), ref)
				if err != nil {
					t.Fatal(err)
				}
				if damage == "missing" {
					_, err = s.conn.ExecContext(context.Background(), "DELETE FROM blobs WHERE digest=?", ref.ContentID[:])
				} else {
					data := bytes.Clone(payload.Bytes())
					data[0] ^= 1 // Same size, different identity; SQLite constraints still hold.
					_, err = s.conn.ExecContext(context.Background(), "UPDATE blobs SET data=? WHERE digest=?", data, ref.ContentID[:])
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				result, err := apply.NewControlService(f.source, New()).Open(context.Background(), f.prepared, f.facts, f.options)
				if _, ok := result.Controlled(); !errors.Is(err, run.ErrUncertain) || ok {
					t.Fatal("damaged preparation became an executable or empty projection", err)
				}
				// Failed restore releases its actual lock. Journal reads still refuse
				// the damaged closure while retaining the known original head.
				check := acquire(t, f.request)
				page, err := check.Read(context.Background(), run.EventQuery{Limit: 1})
				if !errors.Is(err, ErrCorrupt) || page.Revision != prepared.Revision() || len(page.Batches) != 0 {
					t.Fatal("damaged closure became readable or lost the known head", page, err)
				}
				// Direct owned-database inspection proves physical retention, not a
				// usable projection or an alternate decoder for the damaged record.
				var recorded []byte
				if err := check.conn.QueryRowContext(context.Background(), "SELECT data FROM batches WHERE id=?", prepared.ID()).Scan(&recorded); err != nil || !bytes.Equal(recorded, prepared.Bytes()) {
					t.Fatal("failed restore replaced original committed batch bytes", err)
				}
			})
		}
	}
}

func TestControlServiceRecaptureDriftReleasesActualLock(t *testing.T) {
	f := newIntegrationFixture(t)
	original := f.source.inputs
	f.source.inputs.Head = "1111111111111111111111111111111111111111"
	result, err := apply.NewControlService(f.source, New()).Open(context.Background(), f.prepared, f.facts, f.options)
	if _, ok := result.Controlled(); !errors.Is(err, apply.ErrCaptureChanged) || ok {
		t.Fatal("changed recapture obtained control", err)
	}
	if observed, ok := result.Recapture(); !ok || observed.Head != f.source.inputs.Head {
		t.Fatal("failed recapture observation lost")
	}
	f.source.inputs = original
	c := f.open(t)
	if c.Binding().StoreID != result.Binding().StoreID || c.Binding().Epoch <= result.Binding().Epoch || c.Coordinator().Revision() != 0 {
		t.Fatal("failed recapture retained the lock or changed the store")
	}
}

func TestSessionPreparationRequiresActualSQLPayloadsAndRestoresRecovery(t *testing.T) {
	f := newIntegrationFixture(t)
	c := f.open(t)
	integrationCommit(t, c, run.PrepareCommand("prepare", "assignment", "run", "", integrationAt, c.Basis()))
	integrationCommit(t, c, run.Command{ID: "start", Kind: run.StartRun, Assignment: "assignment", Run: "run", ObservedAt: integrationAt})
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	s := acquire(t, f.request)
	coordinator, err := run.Restore(context.Background(), s, f.facts.RepositoryIdentity, f.options.RestoreLimits)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := run.NewPayload("application/json", "utf-8", []byte(`{"model":"controlled","credentials":"credential-ref"}`))
	if err != nil {
		t.Fatal(err)
	}
	tools, err := run.NewPayload("application/json", "utf-8", []byte(`{"host-tool":"controlled-schema"}`))
	if err != nil {
		t.Fatal(err)
	}
	plan := run.SessionPlan{ID: "session", Role: "manager", ReadOnly: true, Workspace: "controlled-linux-workspace", ProviderHome: "provider-home", Candidate: f.source.inputs.Raw.Census.Digest(),
		ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 4096, Profile: profile.Ref(), Tools: tools.Ref()}
	command, err := coordinator.PrepareExploreSessionCommand(run.RecordMeta{ID: "prepare-session", ObservedAt: integrationAt}, ".", plan,
		run.Reservation{ID: "session-start", Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := coordinator.Propose(command)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := proposal.Batch().SessionRefs()
	if err != nil || len(refs) != 2 {
		t.Fatal("producer session references", refs, err)
	}
	for _, payload := range []run.Payload{profile, tools} {
		if _, err := s.Put(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
	}
	for _, ref := range refs {
		payload, err := s.ReadBlob(context.Background(), ref)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.conn.ExecContext(context.Background(), "DELETE FROM blobs WHERE digest=?", ref.ContentID[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Append(context.Background(), proposal.Batch()); !errors.Is(err, ErrCorrupt) {
			t.Fatal("session acknowledged missing profile/tools bytes", err)
		}
		if _, err := s.Put(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		data := bytes.Clone(payload.Bytes())
		data[0] ^= 1
		if _, err := s.conn.ExecContext(context.Background(), "UPDATE blobs SET data=? WHERE digest=?", data, ref.ContentID[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Append(context.Background(), proposal.Batch()); !errors.Is(err, ErrCorrupt) {
			t.Fatal("session acknowledged corrupt profile/tools bytes", err)
		}
		if _, err := s.conn.ExecContext(context.Background(), "UPDATE blobs SET data=? WHERE digest=?", payload.Bytes(), ref.ContentID[:]); err != nil {
			t.Fatal(err)
		}
		page, err := s.Read(context.Background(), run.EventQuery{Limit: 256})
		if err != nil || page.Revision != coordinator.Revision() || len(page.Batches) != 2 {
			t.Fatal("refused session append changed history", page, err)
		}
	}
	if receipt, err := coordinator.Commit(context.Background(), s, proposal); err != nil || !receipt.Matches(proposal.Batch()) {
		t.Fatal("confirmed session preparation", receipt, err)
	}
	want, ok := coordinator.Session(plan.ID)
	if !ok {
		t.Fatal("confirmed session missing")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := f.open(t)
	got, ok := reopened.Coordinator().Session(plan.ID)
	if !ok || got != want || !reopened.Coordinator().RecoveryRequired() || reopened.Coordinator().Budget().Held.SessionStarts != 1 {
		t.Fatal("session restoration lost its admitted plan, payloads or recovery barrier")
	}
}

func integrationExploreStartup(t *testing.T, c *run.Coordinator, candidate snapshot.TreeID) run.Command {
	t.Helper()
	profile, err := run.NewPayload("application/json", "utf-8", []byte(`{"profile":"controlled"}`))
	if err != nil {
		t.Fatal(err)
	}
	tools, err := run.NewPayload("application/json", "utf-8", []byte(`{"tools":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	plan := run.SessionPlan{ID: "session", Role: "manager", Workspace: "controlled-linux-workspace", ProviderHome: "provider-home", Candidate: candidate, ReadOnly: true, ImageID: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", CredentialRef: "credential-ref", MaxWireBytes: 1 << 20, Profile: profile.Ref(), Tools: tools.Ref()}
	command, err := c.PrepareExploreSessionCommand(run.RecordMeta{ID: "reserve", ObservedAt: integrationAt}, ".", plan, run.Reservation{ID: "invocation", Kind: run.SessionStart, Purpose: run.Writing, Maximum: time.Minute}, profile, tools)
	if err != nil {
		t.Fatal(err)
	}
	return command
}
