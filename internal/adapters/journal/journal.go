package journal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

const maxPayloadBytes = 64 << 20

func (s *session) Put(ctx context.Context, p run.Payload) (ref run.BlobRef, err error) {
	if !p.Valid() {
		return ref, run.ErrInvalid
	}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		r := p.Ref()
		var size int64
		err := tx.QueryRowContext(ctx, "SELECT size FROM blobs WHERE digest=?", r.ContentID[:]).Scan(&size)
		if err == nil {
			existing, err := readBlob(ctx, tx, r)
			if err != nil {
				return err
			}
			if !bytes.Equal(existing.Bytes(), p.Bytes()) {
				return ErrCorrupt
			}
		} else if errors.Is(err, sql.ErrNoRows) {
			if _, err = tx.ExecContext(ctx, "INSERT INTO blobs(digest,size,data) VALUES(?,?,?)", r.ContentID[:], r.Size, p.Bytes()); err != nil {
				return err
			}
		} else {
			return err
		}
		ref = r
		return nil
	})
	return
}

func (s *session) ReadBlob(ctx context.Context, ref run.BlobRef) (p run.Payload, err error) {
	if !ref.Valid() {
		return p, run.ErrInvalid
	}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		p, err = readBlob(ctx, tx, ref)
		return err
	})
	return
}

func readBlob(ctx context.Context, tx *sql.Tx, ref run.BlobRef) (run.Payload, error) {
	// Check length before loading attacker/corruption-controlled bytes.
	var size, length int64
	if err := tx.QueryRowContext(ctx, "SELECT size,length(data) FROM blobs WHERE digest=?", ref.ContentID[:]).Scan(&size, &length); err != nil {
		return run.Payload{}, errors.Join(ErrCorrupt, err)
	}
	if size != ref.Size || length != size || size < 0 || size > maxPayloadBytes {
		return run.Payload{}, ErrCorrupt
	}
	var data []byte
	if err := tx.QueryRowContext(ctx, "SELECT data FROM blobs WHERE digest=?", ref.ContentID[:]).Scan(&data); err != nil {
		return run.Payload{}, err
	}
	p, err := run.NewPayload(ref.MediaType, ref.Encoding, data)
	if err != nil || !p.Valid() || p.Ref() != ref {
		return run.Payload{}, ErrCorrupt
	}
	return p, nil
}

func requiredBlobs(ctx context.Context, tx *sql.Tx, b run.AppendBatch) error {
	refs, err := b.BindingRefs()
	if err != nil {
		return err
	}
	for _, ref := range refs {
		p, err := readBlob(ctx, tx, ref)
		if err != nil {
			return err
		}
		owner, err := run.BindingOwnerRef(p)
		if err != nil {
			return err
		}
		if _, err := readBlob(ctx, tx, owner); err != nil {
			return err
		}
	}
	sessionRefs, err := b.SessionRefs()
	if err != nil {
		return err
	}
	for _, ref := range sessionRefs {
		if _, err := readBlob(ctx, tx, ref); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) Append(ctx context.Context, b run.AppendBatch) (receipt run.AppendReceipt, err error) {
	validated, err := run.DecodeBatch(b.Bytes())
	if err != nil {
		return receipt, err
	}
	err = s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// Current control is checked by transaction BEFORE this exact repeat
		// lookup. Batch identity precedes revision CAS, including old retries.
		old, found, err := lookupBatch(ctx, tx, b.ID())
		if err != nil {
			return err
		}
		if found {
			receipt = run.ReceiptFor(old)
			if !bytes.Equal(old.Bytes(), b.Bytes()) {
				return run.ErrConflict
			}
			if err := requiredBlobs(ctx, tx, old); err != nil {
				return err
			}
			return nil
		}
		revision, err := head(ctx, tx)
		if err != nil {
			return err
		}
		if revision != validated.ExpectedRevision() {
			return run.ErrConflict
		}
		if err := requiredBlobs(ctx, tx, validated); err != nil {
			return err
		}
		for _, event := range validated.Events() {
			var exists int
			if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM events WHERE id=?", event.ID).Scan(&exists); err != nil {
				return err
			}
			if exists != 0 {
				return run.ErrConflict
			}
		}
		digest := validated.ContentID()
		if _, err := tx.ExecContext(ctx, "INSERT INTO batches(id,digest,previous,revision,data) VALUES(?,?,?,?,?)", validated.ID(), digest[:], validated.ExpectedRevision(), validated.Revision(), validated.Bytes()); err != nil {
			return err
		}
		for i, event := range validated.Events() {
			if _, err := tx.ExecContext(ctx, "INSERT INTO events(id,batch,revision,ordinal) VALUES(?,?,?,?)", event.ID, validated.ID(), validated.ExpectedRevision()+uint64(i)+1, i); err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(ctx, "UPDATE stream_head SET revision=? WHERE singleton=1 AND revision=?", validated.Revision(), validated.ExpectedRevision())
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return errors.Join(ErrCorrupt, err)
		}
		// This is a known proposed receipt until transaction confirms commit.
		// Callers must retain it beside an error and reconcile exact batch ID.
		receipt = run.ReceiptFor(validated)
		return nil
	})
	return
}

func lookupBatch(ctx context.Context, tx *sql.Tx, id string) (run.AppendBatch, bool, error) {
	var length int64
	err := tx.QueryRowContext(ctx, "SELECT length(data) FROM batches WHERE id=?", id).Scan(&length)
	if errors.Is(err, sql.ErrNoRows) {
		return run.AppendBatch{}, false, nil
	}
	if err != nil {
		return run.AppendBatch{}, false, err
	}
	if length <= 0 || length > maxPayloadBytes {
		return run.AppendBatch{}, false, ErrCorrupt
	}
	var data, digest []byte
	var previous, revision int64
	if err := tx.QueryRowContext(ctx, "SELECT data,digest,previous,revision FROM batches WHERE id=?", id).Scan(&data, &digest, &previous, &revision); err != nil {
		return run.AppendBatch{}, false, err
	}
	b, err := run.DecodeBatch(data)
	hash := sha256.Sum256(data)
	if err != nil || b.ID() != id || previous < 0 || revision < 0 || b.ExpectedRevision() != uint64(previous) || b.Revision() != uint64(revision) || !bytes.Equal(hash[:], digest) {
		return run.AppendBatch{}, false, ErrCorrupt
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,revision,ordinal FROM events WHERE batch=? ORDER BY ordinal", id)
	if err != nil {
		return run.AppendBatch{}, false, err
	}
	defer rows.Close()
	events := b.Events()
	index := 0
	for rows.Next() {
		var eventID string
		var rev int64
		var ordinal int
		if err := rows.Scan(&eventID, &rev, &ordinal); err != nil {
			return run.AppendBatch{}, false, err
		}
		if index >= len(events) || eventID != events[index].ID || rev != previous+int64(index)+1 || ordinal != index {
			return run.AppendBatch{}, false, ErrCorrupt
		}
		index++
	}
	if err := rows.Err(); err != nil {
		return run.AppendBatch{}, false, err
	}
	if index != len(events) {
		return run.AppendBatch{}, false, ErrCorrupt
	}
	return b, true, nil
}
