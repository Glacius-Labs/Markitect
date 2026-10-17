package journal

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// A whole maximum-sized batch always fits, and a page never exceeds 64 MiB of
// canonical batch bytes even when the caller requests 256 batches.
const maxPageBytes = maxPayloadBytes

// A smaller target keeps ordinary multi-batch pages cheap. One individually
// valid larger batch still fits by itself, up to maxPageBytes.
const targetPageBytes = 1 << 20

func (s *session) Read(ctx context.Context, q run.EventQuery) (page run.EventPage, err error) {
	if !q.Valid() {
		return page, run.ErrInvalid
	}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		current, err := head(ctx, tx)
		if err != nil {
			return err
		}
		page.Revision = current
		if q.BatchID != "" {
			b, found, err := lookupBatch(ctx, tx, q.BatchID)
			if err != nil {
				return err
			}
			if found {
				if b.Revision() > current {
					return ErrCorrupt
				}
				if err := requiredBlobs(ctx, tx, b); err != nil {
					return err
				}
				page.Batches = []run.AppendBatch{b}
			}
			return nil
		}
		through := q.Through
		if through == 0 {
			through = current
		}
		if through > current || q.After > through {
			return run.ErrInvalid
		}
		for _, boundary := range []uint64{q.After, through} {
			if boundary == 0 {
				continue
			}
			var count int
			if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM batches WHERE revision=?", boundary).Scan(&count); err != nil {
				return err
			}
			if count != 1 {
				return run.ErrInvalid
			}
		}
		page.Revision = through
		next := q.After
		bytes := int64(0)
		for next < through && len(page.Batches) < int(q.Limit) {
			var id string
			var length, end int64
			err := tx.QueryRowContext(ctx, "SELECT id,length(data),revision FROM batches WHERE previous=?", next).Scan(&id, &length, &end)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrCorrupt
			}
			if err != nil {
				return err
			}
			if length <= 0 || length > maxPayloadBytes || end <= int64(next) || end > int64(through) {
				return ErrCorrupt
			}
			if len(page.Batches) > 0 && bytes+length > targetPageBytes {
				break
			}
			b, found, err := lookupBatch(ctx, tx, id)
			if err != nil {
				return err
			}
			if !found || b.ExpectedRevision() != next {
				return ErrCorrupt
			}
			if err := requiredBlobs(ctx, tx, b); err != nil {
				return err
			}
			page.Batches = append(page.Batches, b)
			next = b.Revision()
			bytes += length
		}
		if next < through {
			page.Next = next
		}
		return nil
	})
	return
}
