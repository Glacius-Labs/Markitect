package run

import (
	"bytes"
	"context"
	"fmt"
)

// RestoreLimits are explicit finite read budgets. Bytes include complete batches
// and each distinct required blob. Zero values are invalid, not unlimited.
type RestoreLimits struct {
	PageSize        uint32
	Batches, Events uint64
	Bytes           int64
}

// Restore rebuilds one confirmed projection from a pinned complete stream. It
// calls only Read and ReadBlob, never Commit or any external execution port.
// Any error returns nil: a partial prefix is not an executable Coordinator.
// Control/store identity and the current controller remain the Host's duties.
func Restore(ctx context.Context, journal Journal, repository string, limits RestoreLimits) (*Coordinator, error) {
	if ctx == nil || journal == nil || limits.PageSize < 1 || limits.PageSize > 256 || limits.Batches == 0 || limits.Events == 0 || limits.Bytes <= 0 {
		return nil, ErrInvalid
	}
	c, err := NewCoordinator(repository)
	if err != nil {
		return nil, err
	}
	reader := restoreReader{ctx: ctx, journal: journal, remaining: limits.Bytes, blobs: map[BlobRef]Payload{}}
	var after, head uint64
	pinned := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: read: %v", ErrUncertain, err)
		}
		page, err := journal.Read(ctx, EventQuery{After: after, Through: head, Limit: limits.PageSize})
		if err != nil {
			return nil, fmt.Errorf("%w: read: %v", ErrUncertain, err)
		}
		if !pinned {
			head = page.Revision
			pinned = true
		}
		if page.Revision != head || head > MaxRevision || head < after || len(page.Batches) > int(limits.PageSize) {
			return nil, ErrInvalid
		}
		for _, stored := range page.Batches {
			data := stored.Bytes()
			if limits.Batches == 0 {
				return nil, ErrRestoreLimit
			}
			if err := reader.consume(int64(len(data))); err != nil {
				return nil, err
			}
			limits.Batches--
			batch, err := DecodeBatch(data)
			if err != nil || batch.ExpectedRevision() != after || batch.Revision() > head {
				return nil, ErrInvalid
			}
			events := batch.Events()
			if uint64(len(events)) > limits.Events {
				return nil, ErrRestoreLimit
			}
			limits.Events -= uint64(len(events))
			command := events[len(events)-1].Body
			if command.Kind == PrepareRun {
				command.binding, err = reader.binding(command.Basis)
				if err != nil {
					return nil, err
				}
			}
			for _, ref := range command.Session.payloadRefs() {
				payload, readErr := reader.blob(ref)
				if readErr != nil {
					return nil, readErr
				}
				command.sessionPayloads = append(command.sessionPayloads, payload)
			}
			command.replaying = true
			command.managerProof = commandDigest(command)
			proposal, err := c.Propose(command)
			if err != nil || proposal.replay != nil || !bytes.Equal(proposal.batch.Bytes(), data) {
				return nil, ErrInvalid
			}
			// Adoption of an already committed fact; no Put/Append or effect.
			c.state = proposal.next
			after = batch.Revision()
		}
		if page.Next == 0 {
			if after != head {
				return nil, ErrInvalid
			}
			c.recovering = c.state.hasOpen() || c.state.hasLiveSessions()
			return c, nil
		}
		if len(page.Batches) == 0 || page.Next != after || after >= head {
			return nil, ErrInvalid
		}
	}
}

type restoreReader struct {
	ctx       context.Context
	journal   Journal
	remaining int64
	blobs     map[BlobRef]Payload
}

func (r *restoreReader) consume(size int64) error {
	if size < 0 {
		return ErrInvalid
	}
	if size > r.remaining {
		return ErrRestoreLimit
	}
	r.remaining -= size
	return nil
}

func (r *restoreReader) blob(ref BlobRef) (Payload, error) {
	if !ref.Valid() {
		return Payload{}, ErrInvalid
	}
	if p, ok := r.blobs[ref]; ok {
		return p, nil
	}
	if err := r.consume(ref.Size); err != nil {
		return Payload{}, err
	}
	if err := r.ctx.Err(); err != nil {
		return Payload{}, fmt.Errorf("%w: blob: %v", ErrUncertain, err)
	}
	p, err := r.journal.ReadBlob(r.ctx, ref)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: blob: %v", ErrUncertain, err)
	}
	if !p.Valid() || p.Ref() != ref {
		return Payload{}, ErrInvalid
	}
	r.blobs[ref] = p
	return p, nil
}
