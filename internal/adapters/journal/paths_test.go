package journal

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

func TestLiveRegistrationReplacementPoisonsHandle(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	path := filepath.Join(r.Repository, ".markitect", "state", "journal.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	assertFenced(t, s)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".retained", path); err != nil {
		t.Fatal(err)
	}
	// Even restoring the original object cannot revive the poisoned capability.
	if err := s.Check(context.Background()); !errors.Is(err, apply.ErrControlBinding) {
		t.Fatal(err)
	}
}

func TestControlDirectoryAndStateRedirectsRefuse(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// Hardlinking the database changes its alias count without changing content.
	path := filepath.Join(r.StateDirectory, "journal.sqlite")
	if err := os.Link(path, path+".alias"); err != nil {
		t.Fatal(err)
	}
	if got, err := New().Acquire(context.Background(), r); err == nil {
		got.Close()
		t.Fatal("hardlinked database authorized")
	}
}

func TestCommitWithoutConfirmationRetainsKnownObservation(t *testing.T) {
	r := request(t)
	s := acquire(t, r)
	ctx := context.Background()
	// Execute a real COMMIT in the action, then let transaction discover that
	// its own commit cannot be confirmed. This narrowly probes error handling
	// without a production failure-injection callback or a fake SQL driver.
	p, _ := run.NewPayload("text/plain", "utf-8", []byte("known committed bytes"))
	ref := p.Ref()
	var known run.BlobRef
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO blobs VALUES(?,?,?)", ref.ContentID[:], ref.Size, p.Bytes()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "COMMIT"); err != nil {
			return err
		}
		known = ref
		return nil
	})
	if !errors.Is(err, run.ErrUncertain) || known != ref {
		t.Fatal("commit without proof acknowledged or observation lost", known, err)
	}
	assertFenced(t, s)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = acquire(t, r)
	got, err := s.ReadBlob(ctx, known)
	if err != nil || got.Ref() != known {
		t.Fatal("committed observation lost on reopen", err)
	}
}
