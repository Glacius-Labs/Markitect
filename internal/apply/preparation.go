package apply

import (
	"context"
	"errors"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

var (
	ErrPreparationSelection = errors.New("invalid structural preparation selection")
	ErrPreparationClosure   = errors.New("structural preparation closure changed during recapture")
)

// PreparationService consumes the model acquisition ports through ModelService.
// It prepares structural file accounting and static context selections only:
// Passed establishes no settings, rendered context, execution or Apply readiness.
// No stage reads the current disk or augments a previous acquisition's bytes.
// An absent Execution selection explicitly means the required repository-root
// .markitect/execution.yaml. A nonempty caller selection is never replaced.
// Request retains the original selection; each Attempt.Model().Request() records
// the actual port selection, including this mandatory input and derived files.
type PreparationService struct{ models *ModelService }

func NewPreparationService(models *ModelService) *PreparationService {
	return &PreparationService{models: models}
}

// Each attempt retains its actual acquisition, error and bound stage reports,
// including incomplete Accounting. A failed acquisition is never checked.
type CapturedPreparationAttempt struct {
	stage     CapturedModel
	registry  CapturedRegistryPreparation
	files     CapturedFileCheck
	execution CapturedExecutionDecoding
	contexts  CapturedContextSelections
	err       error
}

func (a CapturedPreparationAttempt) Model() CapturedModel                  { return a.stage }
func (a CapturedPreparationAttempt) Registry() CapturedRegistryPreparation { return a.registry }
func (a CapturedPreparationAttempt) Files() CapturedFileCheck              { return a.files }
func (a CapturedPreparationAttempt) Execution() CapturedExecutionDecoding  { return a.execution }
func (a CapturedPreparationAttempt) Contexts() CapturedContextSelections   { return a.contexts }
func (a CapturedPreparationAttempt) Error() error                          { return a.err }

type PreviewPreparationAttempt struct {
	stage     PreviewedModel
	registry  PreviewRegistryPreparation
	files     PreviewFileCheck
	execution PreviewExecutionDecoding
	contexts  PreviewContextSelections
	err       error
}

func (a PreviewPreparationAttempt) Model() PreviewedModel                { return a.stage }
func (a PreviewPreparationAttempt) Registry() PreviewRegistryPreparation { return a.registry }
func (a PreviewPreparationAttempt) Files() PreviewFileCheck              { return a.files }
func (a PreviewPreparationAttempt) Execution() PreviewExecutionDecoding  { return a.execution }
func (a PreviewPreparationAttempt) Contexts() PreviewContextSelections   { return a.contexts }
func (a PreviewPreparationAttempt) Error() error                         { return a.err }

// Results retain the original request separately from each actual port request.
// Prepared returns only the final, complete acquisition and its own guard. Old
// guards remain historical observations in Attempts and are never extended.
// The zero value is NotCompleted. All accessors return caller-owned containers.
type CapturedPreparation struct {
	request  CaptureRequest
	attempts []CapturedPreparationAttempt
	status   model.Status
}

func (r CapturedPreparation) Request() CaptureRequest {
	return CaptureRequest{cloneSelection(r.request.Selection)}
}
func (r CapturedPreparation) Attempts() []CapturedPreparationAttempt { return slices.Clone(r.attempts) }
func (r CapturedPreparation) Status() model.Status                   { return preparationStatus(r.status) }
func (r CapturedPreparation) Prepared() (CapturedPreparationAttempt, bool) {
	if r.Status() != model.Passed || len(r.attempts) == 0 {
		return CapturedPreparationAttempt{}, false
	}
	return r.attempts[len(r.attempts)-1], true
}

// PreviewPreparation exposes only preview attempts, with no publication guard.
type PreviewPreparation struct {
	request  PreviewRequest
	attempts []PreviewPreparationAttempt
	status   model.Status
}

func (r PreviewPreparation) Request() PreviewRequest {
	request := r.request
	request.Selection = cloneSelection(request.Selection)
	return request
}
func (r PreviewPreparation) Attempts() []PreviewPreparationAttempt { return slices.Clone(r.attempts) }
func (r PreviewPreparation) Status() model.Status                  { return preparationStatus(r.status) }
func (r PreviewPreparation) Prepared() (PreviewPreparationAttempt, bool) {
	if r.Status() != model.Passed || len(r.attempts) == 0 {
		return PreviewPreparationAttempt{}, false
	}
	return r.attempts[len(r.attempts)-1], true
}

func preparationStatus(status model.Status) model.Status {
	if status == "" {
		return model.NotCompleted
	}
	return status
}

// Capture makes one acquisition when the selection is complete, otherwise one
// new full acquisition. The second acquisition must observe the same derived
// closure; a moving working tree never triggers an unbounded acquisition loop.
func (s *PreparationService) Capture(ctx context.Context, request CaptureRequest, limits model.Limits) (CapturedPreparation, error) {
	result, err := s.capturePreparation(ctx, request, limits, nil)
	r := CapturedPreparation{request: result.request, status: result.status}
	for _, attempt := range result.attempts {
		r.attempts = append(r.attempts, attempt.structural)
	}
	return r, err
}

// Preview preserves the original fixed Revision on both calls. Empty Revision
// remains a working preview; neither can acquire a publication guard.
func (s *PreparationService) Preview(ctx context.Context, request PreviewRequest, limits model.Limits) (PreviewPreparation, error) {
	result, err := s.previewPreparation(ctx, request, limits, nil)
	r := PreviewPreparation{request: result.request, status: result.status}
	for _, attempt := range result.attempts {
		r.attempts = append(r.attempts, attempt.structural)
	}
	return r, err
}

func structuralPreparationStatus(statuses ...model.Status) model.Status {
	result := model.Passed
	for _, status := range statuses {
		if status == model.Failed {
			return model.Failed
		}
		if status != model.Passed {
			result = model.NotCompleted
		}
	}
	return result
}

func acquisitionPreparationStatus(err error) model.Status {
	var manifest *ManifestPreflightError
	if errors.As(err, &manifest) && manifest.Inspection().Status == model.Failed {
		return model.Failed
	}
	return model.NotCompleted
}
