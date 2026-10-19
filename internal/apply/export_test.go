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

// OriginalText exposes the quoted original of a protected input.
func OriginalText(before snapshot.Entry, code string) string { return originalText(before, code) }

// Conclude exposes the publication and cleanup step that follows a run.
func Conclude(ctx context.Context, req ReconcileRequest, start snapshot.Tree, outcome run.Outcome, runErr error) (ReconcileResult, error) {
	return conclude(ctx, req, start, outcome, runErr)
}
