// Package run owns fixed-basis admission and recorded lifecycle decisions.
// It starts no sessions or processes and performs no acquisition or publication.
package run

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const maxRecordBytes = 64 << 20

var (
	ErrInvalid   = errors.New("invalid run record")
	ErrConflict  = errors.New("conflicting run record")
	ErrUncertain = errors.New("unconfirmed journal commit; reconciliation required")
)

// Journal is bound by its adapter to one repository and live controller epoch.
// All returned partial observations remain useful beside an error. A nil error
// and exact AppendReceipt acknowledge one durable atomic commit, not dispatch.
// The adapter checks batch identity BEFORE CAS: an identical batch returns its
// original receipt, different bytes under that ID conflict, stale new work fails.
// Referenced blobs must be durable before Append acknowledges. No transaction
// stays open around an external effect. RUN-1 supplies no physical adapter.
type Journal interface {
	Put(context.Context, Payload) (BlobRef, error)
	ReadBlob(context.Context, BlobRef) (Payload, error)
	Append(context.Context, AppendBatch) (AppendReceipt, error)
	Read(context.Context, EventQuery) (EventPage, error)
}

// BlobRef binds original bytes AND their media/encoding interpretation.
type BlobRef struct {
	ContentID snapshot.ContentID
	Size      int64
	MediaType string
	Encoding  string
}

// Valid checks the reference shape; only reading the original bytes can verify
// their content identity. Empty payloads are allowed, absent references are not.
func (r BlobRef) Valid() bool {
	return r.ContentID != (snapshot.ContentID{}) && r.Size >= 0 && r.Size <= maxRecordBytes && text(r.MediaType) && text(r.Encoding)
}

type Payload struct {
	ref  BlobRef
	data []byte
}

func NewPayload(media, encoding string, data []byte) (Payload, error) {
	if !text(media) || !text(encoding) || len(data) > maxRecordBytes {
		return Payload{}, ErrInvalid
	}
	return Payload{BlobRef{snapshot.ContentID(sha256.Sum256(data)), int64(len(data)), media, encoding}, slices.Clone(data)}, nil
}
func (p Payload) Ref() BlobRef  { return p.ref }
func (p Payload) Bytes() []byte { return slices.Clone(p.data) }
func (p Payload) Valid() bool {
	return p.ref.Valid() && p.ref.ContentID == snapshot.ContentID(sha256.Sum256(p.data)) && p.ref.Size == int64(len(p.data))
}

// Event is a closed RUN-1 event. Body is a validated Command, never a free map.
// Additional session/review/delegation events belong to later consumer slices.
type Event struct {
	ID   string
	Kind string
	Body Command
}

type batchWire struct {
	Version          uint32
	ID               string
	ExpectedRevision uint64
	Events           []Event
}

// AppendBatch owns canonical fixed-order JSON. One revision unit is one event.
type AppendBatch struct {
	wire batchWire
	data []byte
}

func newBatch(id string, revision uint64, events []Event) (AppendBatch, error) {
	w := batchWire{1, id, revision, slices.Clone(events)}
	if !identifier(id) || len(events) == 0 || len(events) > 256 || revision > ^uint64(0)-uint64(len(events)) {
		return AppendBatch{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, event := range events {
		if !identifier(event.ID) || seen[event.ID] || !eventKind(event.Kind, event.Body.Kind) {
			return AppendBatch{}, ErrInvalid
		}
		if err := event.Body.validate(); err != nil {
			return AppendBatch{}, err
		}
		seen[event.ID] = true
	}
	data, err := json.Marshal(w)
	if err != nil || len(data) > maxRecordBytes {
		return AppendBatch{}, ErrInvalid
	}
	return AppendBatch{w, data}, nil
}

func DecodeBatch(data []byte) (AppendBatch, error) {
	var w batchWire
	if err := strictJSON(data, &w); err != nil || w.Version != 1 {
		return AppendBatch{}, ErrInvalid
	}
	b, err := newBatch(w.ID, w.ExpectedRevision, w.Events)
	if err != nil || !bytes.Equal(b.data, data) {
		return AppendBatch{}, ErrInvalid
	}
	return b, nil
}
func (b AppendBatch) ID() string                    { return b.wire.ID }
func (b AppendBatch) ExpectedRevision() uint64      { return b.wire.ExpectedRevision }
func (b AppendBatch) Revision() uint64              { return b.wire.ExpectedRevision + uint64(len(b.wire.Events)) }
func (b AppendBatch) ContentID() snapshot.ContentID { return snapshot.ContentID(sha256.Sum256(b.data)) }
func (b AppendBatch) Bytes() []byte                 { return slices.Clone(b.data) }
func (b AppendBatch) Events() []Event {
	var w batchWire
	_ = json.Unmarshal(b.data, &w)
	return w.Events
}

// A receipt is an adapter observation, not an authentication token. Constructors
// and comparisons cannot establish a test fake's physical durability.
type AppendReceipt struct {
	BatchID                    string
	ContentID                  snapshot.ContentID
	PreviousRevision, Revision uint64
}

func ReceiptFor(b AppendBatch) AppendReceipt {
	return AppendReceipt{b.ID(), b.ContentID(), b.ExpectedRevision(), b.Revision()}
}
func (r AppendReceipt) Matches(b AppendBatch) bool { return b.ID() != "" && r == ReceiptFor(b) }

// Read returns whole atomic batches, so a page never exposes a partial commit.
// Through=0 asks the adapter to pin its current head; later pages reuse that
// head. BatchID selects an exact lookup (at most one batch), used for uncertain
// commits. A complete empty lookup is observed absence at the returned head;
// an error/unreachable store is never absence. Next is the next After revision,
// zero means that this query is exhausted. Limit is in [1,256].
type EventQuery struct {
	BatchID        string
	After, Through uint64
	Limit          uint32
}
type EventPage struct {
	Revision uint64
	Batches  []AppendBatch
	Next     uint64
}

func eventKind(event, command string) bool {
	return event == eventTag(command) || command == PrepareRun && event == "assignment.created"
}

func eventTag(command string) string {
	switch command {
	case StartRun, WaitRun, ResumeRun, StopRun, ObserveUncertain:
		return "run.state-changed"
	default:
		return command
	}
}
func text(s string) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && !strings.ContainsRune(s, 0)
}
func identifier(s string) bool { return text(s) && len(s) <= 256 }

// Canonical round-trip rejects unknown/duplicate fields, alternate field order,
// trailing content and alternate numeric spellings, without repairing bytes.
func strictJSON(data []byte, out any) error {
	if len(data) == 0 || len(data) > maxRecordBytes {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return ErrInvalid
	}
	canonical, err := json.Marshal(out)
	if err != nil || !bytes.Equal(canonical, data) {
		return ErrInvalid
	}
	return nil
}
