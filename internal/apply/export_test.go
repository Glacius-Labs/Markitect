package apply

import (
	"context"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// BuildReconcileConfig and PublishChanges expose the use case's steps to the
// external tests.
func BuildReconcileConfig(req ReconcileRequest, now time.Time) (run.Config, error) {
	return reconcileConfig(req, now)
}

func PublishChanges(ctx context.Context, req ReconcileRequest, changes []snapshot.Change) (*PublicationReceipt, error) {
	return publish(ctx, req, changes)
}

// OriginalText exposes the quoted original of a protected input; quoted is
// the candidate's running total of quoted bytes.
func OriginalText(before snapshot.Entry, code string, quoted *int) string {
	return originalText(before, code, quoted)
}

// Conclude exposes the publication and cleanup step that follows a run.
func Conclude(ctx context.Context, req ReconcileRequest, start snapshot.Tree, outcome run.Outcome, runErr error) (ReconcileResult, error) {
	return conclude(ctx, req, start, outcome, runErr)
}

// ExtractStructureConfig and ExtractModelConfig expose the engine configuration
// of the two Extract phases; the model phase starts from the given candidate of
// the structure phase.
func ExtractStructureConfig(req ExtractRequest, now time.Time) (run.Config, error) {
	start, err := newExtractStart(req.Prepared)
	if err != nil {
		return run.Config{}, err
	}
	accounting, err := newExtractAccounting(extractStructure, start)
	if err != nil {
		return run.Config{}, err
	}
	return req.phase("structure").config(accounting, start, now.Add(start.budget.ExecutionTime.Value), start.budget.AgentTurns.Value)
}

func ExtractModelConfig(req ExtractRequest, candidate snapshot.Tree, now time.Time) (run.Config, error) {
	start, err := newExtractStart(req.Prepared)
	if err != nil {
		return run.Config{}, err
	}
	next, err := start.next(candidate)
	if err != nil {
		return run.Config{}, err
	}
	accounting, err := newExtractAccounting(extractModel, next)
	if err != nil {
		return run.Config{}, err
	}
	return req.phase("model").config(accounting, next, now.Add(start.budget.ExecutionTime.Value), start.budget.AgentTurns.Value)
}

// RequireModelChangesOnly exposes the publication filter of an Extract.
func RequireModelChangesOnly(changes []snapshot.Change) error {
	return requireModelChangesOnly(changes)
}
