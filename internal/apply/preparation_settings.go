package apply

import (
	"context"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Settings attempts retain each actual structural acquisition and its own
// product defaults and static Runtime preparation. Error includes metadata-port
// errors; Structural().Error() retains the original source error separately.
type CapturedSettingsAttempt struct {
	structural CapturedPreparationAttempt
	defaults   CapturedDefaultsPreparation
	settings   CapturedSettingsPreparation
	err        error
}

func (a CapturedSettingsAttempt) Structural() CapturedPreparationAttempt { return a.structural }
func (a CapturedSettingsAttempt) Defaults() CapturedDefaultsPreparation  { return a.defaults }
func (a CapturedSettingsAttempt) Settings() CapturedSettingsPreparation  { return a.settings }
func (a CapturedSettingsAttempt) Error() error                           { return a.err }

type PreviewSettingsAttempt struct {
	structural PreviewPreparationAttempt
	defaults   PreviewDefaultsPreparation
	settings   PreviewSettingsPreparation
	err        error
}

func (a PreviewSettingsAttempt) Structural() PreviewPreparationAttempt { return a.structural }
func (a PreviewSettingsAttempt) Defaults() PreviewDefaultsPreparation  { return a.defaults }
func (a PreviewSettingsAttempt) Settings() PreviewSettingsPreparation  { return a.settings }
func (a PreviewSettingsAttempt) Error() error                          { return a.err }

// CapturedSettingsResult describes complete structural and static settings
// preparation, never rendered context, runtime or Apply readiness. Prepared
// returns only the last complete acquisition; historical guards remain only in
// their original attempts. Zero is NotCompleted. Accessors copy containers.
type CapturedSettingsResult struct {
	request  CaptureRequest
	attempts []CapturedSettingsAttempt
	status   model.Status
}

func (r CapturedSettingsResult) Request() CaptureRequest {
	return CaptureRequest{cloneSelection(r.request.Selection)}
}
func (r CapturedSettingsResult) Attempts() []CapturedSettingsAttempt { return slices.Clone(r.attempts) }
func (r CapturedSettingsResult) Status() model.Status                { return preparationStatus(r.status) }
func (r CapturedSettingsResult) Prepared() (CapturedSettingsAttempt, bool) {
	if r.Status() != model.Passed || len(r.attempts) == 0 {
		return CapturedSettingsAttempt{}, false
	}
	return r.attempts[len(r.attempts)-1], true
}

// PreviewSettingsResult retains only working/fixed preview parents, without a
// guard. External Runtime locations keep the selected source port's explicit fixed
// revision refusal; this service never mixes today's Runtime bytes into history.
type PreviewSettingsResult struct {
	request  PreviewRequest
	attempts []PreviewSettingsAttempt
	status   model.Status
}

func (r PreviewSettingsResult) Request() PreviewRequest {
	r.request.Selection = cloneSelection(r.request.Selection)
	return r.request
}
func (r PreviewSettingsResult) Attempts() []PreviewSettingsAttempt { return slices.Clone(r.attempts) }
func (r PreviewSettingsResult) Status() model.Status               { return preparationStatus(r.status) }
func (r PreviewSettingsResult) Prepared() (PreviewSettingsAttempt, bool) {
	if r.Status() != model.Passed || len(r.attempts) == 0 {
		return PreviewSettingsAttempt{}, false
	}
	return r.attempts[len(r.attempts)-1], true
}

// CaptureSettings shares Capture's single two-acquisition bound. Static Runtime
// observations run before the first closure decision, including beside an
// incomplete FileCheck. Named instruction needs join the same full recapture;
// missing metadata alone never retries. Product defaults are actual build bytes.
// A recovered earlier metadata error remains in its original attempt; the
// returned error describes the final attempt, closure refusal or cancellation.
// It prepares the start of an Apply: the initial File check is strict, so every
// accounting finding (an unexplained file, a registration of a missing file, of
// an unknown Definition or outside the registrant's area) fails preparation and
// no run starts to repair it (DEC-064).
func (s *PreparationService) CaptureSettings(ctx context.Context, request CaptureRequest, limits model.Limits, agents AgentProfilePreparer, checks CheckProfilePreparer) (CapturedSettingsResult, error) {
	return s.capturePreparation(ctx, request, limits, &preparationSettingsPorts{agents: agents, checks: checks})
}

// CaptureExtract prepares the start of an Extract like CaptureSettings, except
// that the artifacts are not registered yet: the files that no registry shard
// explains and that the layout cannot yet assign to a Package are tolerated,
// because Extract is what accounts for them. Everything else is prepared and
// checked exactly as for Apply.
func (s *PreparationService) CaptureExtract(ctx context.Context, request CaptureRequest, limits model.Limits, agents AgentProfilePreparer, checks CheckProfilePreparer) (CapturedSettingsResult, error) {
	return s.capturePreparation(ctx, request, limits, &preparationSettingsPorts{agents: agents, checks: checks, extract: true})
}

func (s *PreparationService) PreviewSettings(ctx context.Context, request PreviewRequest, limits model.Limits, agents AgentProfilePreparer, checks CheckProfilePreparer) (PreviewSettingsResult, error) {
	return s.previewPreparation(ctx, request, limits, &preparationSettingsPorts{agents: agents, checks: checks})
}

// A nil set selects strictly structural preparation. A requested settings entry
// with unavailable metadata ports remains distinct and reports NotCompleted.
type preparationSettingsPorts struct {
	agents  AgentProfilePreparer
	checks  CheckProfilePreparer
	extract bool
}
