package run_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// memoryJournal is a contract fake: atomic process-local visibility only.
// None of these tests establishes crash durability, fsync or controller fencing.
type memoryJournal struct {
	blobs                   map[run.BlobRef]run.Payload
	batches                 []run.AppendBatch
	head                    uint64
	appendCalls             int
	putFailure, readFailure bool
	appendMode              string
}

func TestCodecRejectsCanonicalBytesWithInvalidValueShapes(t *testing.T) {
	c, _ := run.NewCoordinator("repo")
	prepared, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
	if err != nil {
		t.Fatal(err)
	}
	runningCoordinator, _ := running(t)
	reserved, err := runningCoordinator.Propose(sessionPreparation(t, runningCoordinator, "s"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		batch  run.AppendBatch
		change func(*run.Command)
	}{
		"empty reservation identity": {reserved.Batch(), func(c *run.Command) { c.Reservation.ID = "" }},
		"unknown invocation":         {reserved.Batch(), func(c *run.Command) { c.Reservation.Kind = "unknown" }},
		"unknown purpose":            {reserved.Batch(), func(c *run.Command) { c.Reservation.Purpose = "unknown" }},
		"zero maximum":               {reserved.Batch(), func(c *run.Command) { c.Reservation.Maximum = 0 }},
		"negative maximum":           {reserved.Batch(), func(c *run.Command) { c.Reservation.Maximum = -1 }},
		"writing project check":      {reserved.Batch(), func(c *run.Command) { c.Reservation.Kind = run.ProjectCheck; c.Reservation.Purpose = run.Writing }},
		"whitespace media":           {prepared.Batch(), func(c *run.Command) { c.Basis.MediaType = " \t" }},
		"missing encoding":           {prepared.Batch(), func(c *run.Command) { c.Basis.Encoding = "" }},
		"whitespace encoding":        {prepared.Batch(), func(c *run.Command) { c.Basis.Encoding = " \t" }},
		"negative size":              {prepared.Batch(), func(c *run.Command) { c.Basis.Size = -1 }},
		"oversize":                   {prepared.Batch(), func(c *run.Command) { c.Basis.Size = (64 << 20) + 1 }},
		"absent content identity":    {prepared.Batch(), func(c *run.Command) { c.Basis.ContentID = run.BlobRef{}.ContentID }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// This has exactly batchWire's field order. Marshal keeps bytes canonical;
			// rejection must come from the value contract, not JSON formatting.
			var wire struct {
				Version          uint32
				ID               string
				ExpectedRevision uint64
				Events           []run.Event
			}
			if err := json.Unmarshal(tc.batch.Bytes(), &wire); err != nil {
				t.Fatal(err)
			}
			body := &wire.Events[len(wire.Events)-1].Body
			tc.change(body)
			data, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := run.DecodeBatch(data); !errors.Is(err, run.ErrInvalid) {
				t.Fatal("intrinsically invalid canonical record accepted", err)
			}
			if body.Kind == run.ReserveInvocation || body.Kind == run.PrepareSession {
				if _, err := runningCoordinator.Propose(*body); !errors.Is(err, run.ErrInvalid) {
					t.Fatal("normal admission disagrees with codec", err)
				}
			}
		})
	}
}

func journal() *memoryJournal { return &memoryJournal{blobs: map[run.BlobRef]run.Payload{}} }
func (j *memoryJournal) Put(ctx context.Context, p run.Payload) (run.BlobRef, error) {
	if err := ctx.Err(); err != nil {
		return run.BlobRef{}, err
	}
	if j.putFailure {
		return run.BlobRef{}, errors.New("put unavailable")
	}
	if !p.Valid() {
		return run.BlobRef{}, run.ErrInvalid
	}
	if existing, ok := j.blobs[p.Ref()]; ok && !bytes.Equal(existing.Bytes(), p.Bytes()) {
		return run.BlobRef{}, run.ErrConflict
	}
	j.blobs[p.Ref()] = p
	return p.Ref(), nil
}
func (j *memoryJournal) ReadBlob(ctx context.Context, ref run.BlobRef) (run.Payload, error) {
	if err := ctx.Err(); err != nil {
		return run.Payload{}, err
	}
	p, ok := j.blobs[ref]
	if j.readFailure || !ok {
		return run.Payload{}, errors.New("blob unavailable")
	}
	return p, nil
}
func (j *memoryJournal) Append(ctx context.Context, b run.AppendBatch) (run.AppendReceipt, error) {
	j.appendCalls++
	if err := ctx.Err(); err != nil {
		return run.AppendReceipt{}, err
	}
	stored, err := run.DecodeBatch(b.Bytes())
	if err != nil {
		return run.AppendReceipt{}, err
	}
	for _, previous := range j.batches {
		if previous.ID() == b.ID() {
			if !bytes.Equal(previous.Bytes(), b.Bytes()) {
				return run.AppendReceipt{}, run.ErrConflict
			}
			return run.ReceiptFor(previous), nil // Identity precedes stale-CAS check.
		}
	}
	if b.ExpectedRevision() != j.head {
		return run.AppendReceipt{}, run.ErrConflict
	}
	if j.appendMode == "absent" {
		return run.AppendReceipt{}, errors.New("unavailable before append")
	}
	for _, event := range b.Events() {
		if event.Body.Kind == run.PrepareRun {
			if _, ok := j.blobs[event.Body.Basis]; !ok {
				return run.AppendReceipt{}, run.ErrInvalid
			}
		}
		for _, previous := range j.batches {
			for _, old := range previous.Events() {
				if old.ID == event.ID {
					return run.AppendReceipt{}, run.ErrConflict
				}
			}
		}
	}
	refs, err := b.BindingRefs()
	if err != nil {
		return run.AppendReceipt{}, err
	}
	for _, ref := range refs {
		basis, ok := j.blobs[ref]
		if !ok || !basis.Valid() || basis.Ref() != ref {
			return run.AppendReceipt{}, run.ErrInvalid
		}
		owner, err := run.BindingOwnerRef(basis)
		if err != nil {
			return run.AppendReceipt{}, err
		}
		payload, ok := j.blobs[owner]
		if !ok || !payload.Valid() || payload.Ref() != owner {
			return run.AppendReceipt{}, run.ErrInvalid
		}
	}
	sessionRefs, err := b.SessionRefs()
	if err != nil {
		return run.AppendReceipt{}, err
	}
	for _, ref := range sessionRefs {
		p, ok := j.blobs[ref]
		if !ok || !p.Valid() || p.Ref() != ref {
			return run.AppendReceipt{}, run.ErrInvalid
		}
	}
	// Persist only the closed serialized form; no live proposal/Binding survives.
	j.batches = append(j.batches, stored)
	j.head = b.Revision()
	receipt := run.ReceiptFor(b)
	switch j.appendMode {
	case "absent-after-next":
		j.appendMode = "absent" // This append succeeds; the following one is never stored.
	case "lost-after-next":
		j.appendMode = "lost" // This append succeeds; the following one loses its acknowledgement.
		return receipt, nil
	}
	if j.appendMode == "lost" {
		return receipt, errors.New("ack lost after commit")
	}
	if j.appendMode == "wrong" {
		receipt.Revision++
	}
	return receipt, nil
}
func (j *memoryJournal) Read(ctx context.Context, q run.EventQuery) (run.EventPage, error) {
	if err := ctx.Err(); err != nil {
		return run.EventPage{}, err
	}
	if j.readFailure {
		return run.EventPage{}, errors.New("journal unavailable")
	}
	if !q.Valid() || q.After > j.head || q.Through > j.head {
		return run.EventPage{}, run.ErrInvalid
	}
	boundary := func(position uint64) bool {
		if position == 0 {
			return true
		}
		for _, b := range j.batches {
			if b.Revision() == position {
				return true
			}
		}
		return false
	}
	if !boundary(q.After) || !boundary(q.Through) {
		return run.EventPage{}, run.ErrInvalid
	}
	through := q.Through
	if through == 0 {
		through = j.head
	}
	page := run.EventPage{Revision: through}
	for _, b := range j.batches {
		if q.BatchID != "" {
			if q.BatchID == b.ID() {
				page.Batches = []run.AppendBatch{b}
				break
			}
			continue
		}
		if b.ExpectedRevision() < q.After || b.Revision() > through {
			continue
		}
		if len(page.Batches) == int(q.Limit) {
			page.Next = page.Batches[len(page.Batches)-1].Revision()
			break
		}
		page.Batches = append(page.Batches, b)
	}
	return page, nil
}

var _ run.Journal = (*memoryJournal)(nil)

func TestJournalCodecIdentityCASAndImmutablePayloads(t *testing.T) {
	ctx := context.Background()
	j := journal()
	data := []byte("original\r\n")
	p, err := run.NewPayload("text/plain", "utf-8", data)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	p.Bytes()[0] = 'Y'
	if string(p.Bytes()) != "original\r\n" {
		t.Fatal("payload aliases caller/view bytes")
	}
	ref, _ := j.Put(ctx, p)
	again, _ := j.Put(ctx, p)
	if ref != again {
		t.Fatal("put retry changed identity")
	}
	c, _ := run.NewCoordinator("repo")
	proposal, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
	if err != nil {
		t.Fatal(err)
	}
	b := proposal.Batch()
	decoded, err := run.DecodeBatch(b.Bytes())
	if err != nil || !bytes.Equal(decoded.Bytes(), b.Bytes()) {
		t.Fatal("codec roundtrip", err)
	}
	b.Bytes()[0] = '!'
	b.Events()[0].Body.Assignment = "mutated"
	if _, err := run.DecodeBatch(b.Bytes()); err != nil {
		t.Fatal("batch aliases view")
	}
	for _, malformed := range [][]byte{
		append(b.Bytes(), byte(' ')),
		[]byte(strings.Replace(string(b.Bytes()), `"Version":1`, `"Version":1,"Version":1`, 1)),
		[]byte(strings.Replace(string(b.Bytes()), `"Version":1`, `"Version":1,"Unknown":0`, 1)),
		[]byte(strings.Replace(string(b.Bytes()), `"Version":1`, `"Version":2`, 1)),
	} {
		if _, err := run.DecodeBatch(malformed); !errors.Is(err, run.ErrInvalid) {
			t.Fatal("accepted malformed record", err)
		}
	}
	if _, err := c.Commit(ctx, j, proposal); err != nil {
		t.Fatal(err)
	}
	receipt, err := j.Append(ctx, b)
	if err != nil || receipt != run.ReceiptFor(b) || len(j.batches) != 1 {
		t.Fatal("exact retry", err)
	}
	other, _ := run.NewCoordinator("repo")
	stale, _ := other.Propose(run.PrepareCommand("different", "a2", "r2", "", observed, binding(t)))
	if _, err := j.Append(ctx, stale.Batch()); !errors.Is(err, run.ErrConflict) {
		t.Fatal("stale CAS accepted")
	}
	// Keep the batch ID while changing a command's bytes.
	conflict := bytes.Replace(b.Bytes(), []byte(`"Assignment":"a"`), []byte(`"Assignment":"b"`), -1)
	changed, err := run.DecodeBatch(conflict)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, changed); !errors.Is(err, run.ErrConflict) {
		t.Fatal("conflicting batch ID accepted")
	}
	first, _ := j.Read(ctx, run.EventQuery{Limit: 1})
	if first.Revision != b.Revision() || len(first.Batches) != 1 || first.Next != 0 {
		t.Fatalf("atomic page: %+v", first)
	}
}

func TestProducerReferenceAPIRequiresBasisAndOriginalOwnerBeforeAppend(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := run.NewCoordinator("repo")
	p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, basis))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := p.Batch().BindingRefs()
	if err != nil || len(refs) != 1 || refs[0] != basis.Ref() {
		t.Fatal("direct basis references", refs, err)
	}
	owner, err := run.BindingOwnerRef(basis.Payload())
	if err != nil || owner != i.OwnerRequest.Ref() {
		t.Fatal("indirect owner reference", err)
	}
	j := journal()
	_, _ = j.Put(context.Background(), basis.Payload())
	if _, err := j.Append(context.Background(), p.Batch()); !errors.Is(err, run.ErrInvalid) || j.head != 0 || len(j.batches) != 0 {
		t.Fatal("basis without durable owner acknowledged", err)
	}
	_, _ = j.Put(context.Background(), i.OwnerRequest)
	if receipt, err := j.Append(context.Background(), p.Batch()); err != nil || !receipt.Matches(p.Batch()) {
		t.Fatal("complete references refused", err)
	}
}

func TestBindingReferenceCodecRejectsChangedRecordedFormats(t *testing.T) {
	i := fixture(t)
	basis, err := run.Bind(i)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func([]byte) []byte{
		"unknown binding version": func(data []byte) []byte { return bytes.Replace(data, []byte(`"Version":3`), []byte(`"Version":99`), 1) },
		"wrong compiled digest": func(data []byte) []byte {
			return bytes.Replace(data, []byte(i.Compilation.Models[0].Digest), []byte("sha256:"+strings.Repeat("0", 64)), 1)
		},
		"invalid policy": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"MaxStepTimeout":120000000000`), []byte(`"MaxStepTimeout":0`), 1)
		},
		"invalid topology": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"Parent":"lending"`), []byte(`"Parent":"empty"`), 1)
		},
		"invalid owner reference": func(data []byte) []byte {
			before, _ := json.Marshal(i.OwnerRequest.Ref())
			ref := i.OwnerRequest.Ref()
			ref.Size = -1
			after, _ := json.Marshal(ref)
			return bytes.Replace(data, before, after, 1)
		},
		"noncanonical bytes": func(data []byte) []byte { return append(data, '\n') },
	} {
		t.Run(name, func(t *testing.T) {
			data := change(basis.Payload().Bytes())
			if bytes.Equal(data, basis.Payload().Bytes()) {
				t.Fatal("fixture did not change record")
			}
			p, err := run.NewPayload(basis.Ref().MediaType, basis.Ref().Encoding, data)
			if err != nil {
				t.Fatal(err)
			}
			// Each replacement has a new correct content identity, so this must
			// distinguish the recorded codec from a simple outer hash check.
			if _, err := run.BindingOwnerRef(p); err == nil {
				t.Fatal("invalid stored format accepted")
			}
		})
	}
	for _, interpretation := range [][2]string{{"text/plain", "utf-8"}, {basis.Ref().MediaType, "binary"}} {
		p, _ := run.NewPayload(interpretation[0], interpretation[1], basis.Payload().Bytes())
		if _, err := run.BindingOwnerRef(p); err == nil {
			t.Fatal("wrong basis interpretation accepted")
		}
	}
}

func TestJournalReadUsesExactBatchBoundariesAndReportsLookupHead(t *testing.T) {
	_, j := running(t)
	for _, q := range []run.EventQuery{
		{After: 1, Limit: 1}, {Through: 1, Limit: 1},
		{BatchID: "prepare", Through: j.head, Limit: 1},
		{After: run.MaxRevision + 1, Limit: 1},
	} {
		if _, err := j.Read(context.Background(), q); !errors.Is(err, run.ErrInvalid) {
			t.Fatal("invalid interval accepted", q, err)
		}
	}
	p, err := j.Read(context.Background(), run.EventQuery{After: 2, Through: 3, Limit: 1})
	if err != nil || len(p.Batches) != 1 || p.Batches[0].ID() != "start" || p.Revision != 3 {
		t.Fatal("exact boundary interval", p, err)
	}
	p, err = j.Read(context.Background(), run.EventQuery{BatchID: "prepare", Limit: 1})
	if err != nil || len(p.Batches) != 1 || p.Batches[0].Revision() != 2 || p.Revision != 3 {
		t.Fatal("lookup confused receipt with later head", p, err)
	}
	p, err = j.Read(context.Background(), run.EventQuery{BatchID: "absent", Limit: 1})
	if err != nil || len(p.Batches) != 0 || p.Revision != 3 {
		t.Fatal("lookup absence lost observed head", p, err)
	}
}

func TestBatchCodecEnforcesSupportedRevisionRange(t *testing.T) {
	c, _ := run.NewCoordinator("repo")
	p, err := c.Propose(run.PrepareCommand("prepare", "a", "r", "", observed, binding(t)))
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Version          uint32
		ID               string
		ExpectedRevision uint64
		Events           []run.Event
	}
	if err := json.Unmarshal(p.Batch().Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	wire.ExpectedRevision = run.MaxRevision - uint64(len(wire.Events))
	data, _ := json.Marshal(wire)
	b, err := run.DecodeBatch(data)
	if err != nil || b.Revision() != run.MaxRevision {
		t.Fatal("supported boundary refused", err)
	}
	wire.ExpectedRevision++
	data, _ = json.Marshal(wire)
	if _, err := run.DecodeBatch(data); !errors.Is(err, run.ErrInvalid) {
		t.Fatal("out-of-range stream accepted", err)
	}
}
