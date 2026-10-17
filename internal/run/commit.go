package run

import (
	"bytes"
	"context"
	"fmt"
)

// Commit first persists original basis payloads, then submits the exact atomic
// batch. It adopts no proposed state/budget on a failed, mismatched or uncertain
// acknowledgement, even if the adapter returned a useful partial receipt.
// A confirmed reservation records intent only; RUN-1 invokes no executor.
func (c *Coordinator) Commit(ctx context.Context, journal Journal, p Proposal) (AppendReceipt, error) {
	if ctx == nil || journal == nil {
		return AppendReceipt{}, ErrInvalid
	}
	if c.pending != nil {
		return AppendReceipt{}, ErrUncertain
	}
	// A proposal from another Coordinator/assignment is not a continuation of
	// this projection merely because its numeric revision happens to match.
	check, err := c.Propose(p.command)
	if err != nil {
		if err == ErrRecovery {
			return AppendReceipt{}, err
		}
		return AppendReceipt{}, ErrConflict
	}
	if check.replay != nil {
		if p.replay != nil && *p.replay == *check.replay || p.replay == nil && check.replay.Matches(p.batch) {
			return *check.replay, nil
		}
		return AppendReceipt{}, ErrConflict
	}
	if p.replay != nil || p.batch.ID() == "" || p.batch.ExpectedRevision() != c.state.revision || !bytes.Equal(check.batch.Bytes(), p.batch.Bytes()) {
		return AppendReceipt{}, ErrConflict
	}
	p = check
	c.pending = &p
	for _, payload := range p.payloads {
		ref, err := journal.Put(ctx, payload)
		if err != nil {
			return AppendReceipt{}, fmt.Errorf("%w: put: %v", ErrUncertain, err)
		}
		if ref != payload.Ref() {
			return AppendReceipt{}, ErrUncertain
		}
	}
	receipt, err := journal.Append(ctx, p.batch)
	if err != nil || !receipt.Matches(p.batch) {
		return receipt, fmt.Errorf("%w: append: %v", ErrUncertain, err)
	}
	c.state = p.next
	c.recovering = c.recovering && c.state.hasOpen()
	c.pending = nil
	return receipt, nil
}

// ReconcilePending inspects the exact known batch. Presence adopts only that
// matching complete commit. Proven absence at the unchanged stream head leaves
// the previous projection and allows the caller to propose/retry explicitly.
// Other committed work requires Restore; no committed work is silently skipped
// and no empty history is manufactured.
func (c *Coordinator) ReconcilePending(ctx context.Context, journal Journal) error {
	if ctx == nil || journal == nil {
		return ErrInvalid
	}
	if c.pending == nil {
		return nil
	}
	p := c.pending
	page, err := journal.Read(ctx, EventQuery{BatchID: p.batch.ID(), Limit: 1})
	if err != nil {
		return fmt.Errorf("%w: read: %v", ErrUncertain, err)
	}
	if page.Next != 0 || len(page.Batches) > 1 {
		return ErrUncertain
	}
	if len(page.Batches) == 0 {
		if page.Revision != p.batch.ExpectedRevision() {
			return ErrConflict
		}
		c.pending = nil
		return nil
	}
	observed := page.Batches[0]
	if page.Revision != p.batch.Revision() || !bytes.Equal(observed.Bytes(), p.batch.Bytes()) {
		return ErrConflict
	}
	for _, expected := range p.payloads {
		actual, err := journal.ReadBlob(ctx, expected.Ref())
		if err != nil || !actual.Valid() || actual.Ref() != expected.Ref() || !bytes.Equal(actual.Bytes(), expected.Bytes()) {
			return ErrUncertain
		}
	}
	c.state = p.next
	c.recovering = c.recovering && c.state.hasOpen()
	c.pending = nil
	return nil
}
