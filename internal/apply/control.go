package apply

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

var (
	ErrControlUnavailable = errors.New("repository control unavailable")
	ErrControlBinding     = errors.New("repository control binding lost or mismatched")
	ErrControlClosed      = errors.New("repository control closed")
	ErrCaptureChanged     = errors.New("captured inputs changed under repository control")
)

// Control acquires the one repository execution lock, independently of the
// selected StateDirectory. Its adapter validates physical repository identity,
// the active store registration and the registered database before returning a
// new controller epoch. A missing/replaced registered store needs reconciliation;
// it must never silently become a new empty history. No SQL or lock is supplied
// by this consumer port. A partial session returned with an error must be closed.
type Control interface {
	Acquire(context.Context, ControlRequest) (ControlSession, error)
}

type ControlRequest struct {
	Repository         string // Final capture's explicit acquisition location.
	RepositoryIdentity string // Explicit Host assertion, checked by the adapter.
	Revision           string
	Guard              PublicationGuard // Original final capture's opaque guard.
	StateDirectory     string           // Explicit external store location.
}

// ControlBinding is an observed lifecycle capability, not owner decision or
// publication authority. StoreID is assigned and durably registered by the
// adapter; a path, source digest or caller assertion cannot create that identity.
type ControlBinding struct {
	Repository string
	StoreID    string
	Epoch      uint64
}

func (b ControlBinding) valid() bool {
	return controlID(b.Repository) && controlID(b.StoreID) && b.Epoch > 0 && b.Epoch <= 1<<63-1
}

type ControlSession interface {
	Binding() ControlBinding
	// Journal is bound to this repository, store and live epoch. The adapter
	// fences every operation at its actual boundary; Check is not atomic fencing.
	Journal() run.Journal
	Check(context.Context) error
	// Close releases controller resources only. It never proves that processes
	// terminated or that any unfinished run became quiescent.
	Close() error
}

// ControlledRun holds a restored projection and the original captured basis.
// Host callers serialize its use, like run.Coordinator. Opening creates no run.
// Journal checks control before each operation; its adapter remains responsible
// for atomic fencing. Do not retain/use an alternate unfenced journal.
type ControlledRun struct {
	prepared    CapturedSettingsResult
	facts       RunBindingFacts
	session     ControlSession
	binding     ControlBinding
	basis       run.Binding
	coordinator *run.Coordinator
	journal     controlledJournal
	closed      bool
	closeErr    error
	lost        error
}

func (c *ControlledRun) Binding() ControlBinding       { return c.binding }
func (c *ControlledRun) Basis() run.Binding            { return c.basis }
func (c *ControlledRun) Coordinator() *run.Coordinator { return c.coordinator }
func (c *ControlledRun) Journal() run.Journal          { return &c.journal }

func (c *ControlledRun) Check(ctx context.Context) error {
	if c == nil || c.closed || c.session == nil {
		return ErrControlClosed
	}
	if c.lost != nil {
		return c.lost
	}
	if ctx == nil {
		return run.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.session.Check(ctx); err != nil {
		c.lost = errors.Join(ErrControlBinding, err)
		return c.lost
	}
	if c.session.Binding() != c.binding {
		c.lost = ErrControlBinding
		return c.lost
	}
	return ctx.Err()
}

// Close is idempotent and retains the first cleanup error. Even a failed Close
// makes this handle unusable; a cleanup failure is not reported as release.
func (c *ControlledRun) Close() error {
	if c == nil {
		return nil
	}
	if !c.closed {
		c.closed = true
		if c.session != nil {
			c.closeErr = c.session.Close()
		}
	}
	return c.closeErr
}

type controlledJournal struct {
	control *ControlledRun
	journal run.Journal
}

func (j *controlledJournal) Put(ctx context.Context, p run.Payload) (run.BlobRef, error) {
	if err := j.control.Check(ctx); err != nil {
		return run.BlobRef{}, err
	}
	return j.journal.Put(ctx, p)
}
func (j *controlledJournal) ReadBlob(ctx context.Context, ref run.BlobRef) (run.Payload, error) {
	if err := j.control.Check(ctx); err != nil {
		return run.Payload{}, err
	}
	return j.journal.ReadBlob(ctx, ref)
}
func (j *controlledJournal) Append(ctx context.Context, b run.AppendBatch) (run.AppendReceipt, error) {
	if err := j.control.Check(ctx); err != nil {
		return run.AppendReceipt{}, err
	}
	return j.journal.Append(ctx, b)
}
func (j *controlledJournal) Read(ctx context.Context, q run.EventQuery) (run.EventPage, error) {
	if err := j.control.Check(ctx); err != nil {
		return run.EventPage{}, err
	}
	return j.journal.Read(ctx, q)
}

func controlText(s string) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && !strings.ContainsRune(s, 0)
}
func controlID(s string) bool { return controlText(s) && len(s) <= 256 }
