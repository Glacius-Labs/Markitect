package apply

import (
	"context"
	"errors"
	"reflect"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type ControlService struct {
	source  Source
	control Control
}

func NewControlService(source Source, control Control) *ControlService {
	return &ControlService{source: source, control: control}
}

type ControlOptions struct {
	StateDirectory string
	RestoreLimits  run.RestoreLimits
}

// ControlOpenResult retains actual acquisition observations even on failure.
// Binding alone grants no capability. Only a successful Controlled result holds
// a live session; failed recapture never replaces the original captured basis.
type ControlOpenResult struct {
	binding    ControlBinding
	recaptured CapturedInputs
	observed   bool
	controlled *ControlledRun
}

func (r ControlOpenResult) Binding() ControlBinding { return r.binding }
func (r ControlOpenResult) Recapture() (CapturedInputs, bool) {
	input := r.recaptured
	input.Raw = cloneRaw(input.Raw)
	return input, r.observed
}
func (r ControlOpenResult) Controlled() (*ControlledRun, bool) {
	return r.controlled, r.controlled != nil
}

// Open acquires control, revalidates the original final capture under its lock,
// and restores the complete confirmed stream. It never substitutes empty history
// on a failed restore, appends a record, or starts execution. RecoveryRequired
// remains run.Coordinator's barrier. Physical identity/durability are the Control
// adapter's obligations; controlled tests cannot establish them.
func (s *ControlService) Open(ctx context.Context, prepared CapturedSettingsResult, facts RunBindingFacts, options ControlOptions) (result ControlOpenResult, err error) {
	if ctx == nil {
		return result, run.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	basis, err := prepared.BindRun(facts)
	if err != nil {
		return result, err
	}
	if !controlID(facts.RepositoryIdentity) || !controlText(options.StateDirectory) {
		return result, run.ErrInvalid
	}
	limits := options.RestoreLimits
	if limits.PageSize < 1 || limits.PageSize > 256 || limits.Batches == 0 || limits.Events == 0 || limits.Bytes <= 0 {
		return result, run.ErrInvalid
	}
	if s == nil || s.source == nil || s.control == nil {
		return result, ErrControlUnavailable
	}
	attempt, _ := prepared.Prepared()
	model := attempt.Structural().Model()
	original, request := model.Inputs(), model.Request()
	session, err := s.control.Acquire(ctx, ControlRequest{
		Repository: request.Selection.Repository, RepositoryIdentity: facts.RepositoryIdentity,
		Revision: original.Head, Guard: original.Guard, StateDirectory: options.StateDirectory,
	})
	if session == nil {
		if err == nil {
			err = ErrControlUnavailable
		}
		return result, err
	}
	handle := &ControlledRun{prepared: prepared, facts: facts, session: session, basis: basis}
	defer func() {
		if err != nil {
			err = errors.Join(err, handle.Close())
		}
	}()
	if err != nil {
		return result, err
	}
	handle.binding = session.Binding()
	result.binding = handle.binding
	if !handle.binding.valid() || handle.binding.Repository != facts.RepositoryIdentity {
		return result, ErrControlBinding
	}
	if err = handle.Check(ctx); err != nil {
		return result, err
	}
	journal := session.Journal()
	if journal == nil {
		return result, ErrControlUnavailable
	}
	handle.journal = controlledJournal{control: handle, journal: journal}
	observed, captureErr := s.source.Capture(ctx, CaptureRequest{Selection: cloneSelection(request.Selection)})
	observed.Raw = cloneRaw(observed.Raw)
	result.recaptured, result.observed = observed, true
	if captureErr != nil {
		return result, captureErr
	}
	if err = handle.Check(ctx); err != nil {
		return result, err
	}
	// Guard is opaque: an equivalent fresh capture need not mint equal bytes.
	// Keep the original guard in basis and require actual HEAD/Raw equality.
	if observed.Guard == "" || original.Head != observed.Head || !reflect.DeepEqual(original.Raw, observed.Raw) {
		return result, ErrCaptureChanged
	}
	handle.coordinator, err = run.Restore(ctx, handle.Journal(), handle.binding.Repository, options.RestoreLimits)
	if err != nil {
		return result, err
	}
	if err = handle.Check(ctx); err != nil {
		return result, err
	}
	result.controlled = handle
	return result, nil
}
