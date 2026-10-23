package apply

import (
	"context"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// BuildApplyConfig and PublishChanges expose the use case's steps to the
// external tests.
func BuildApplyConfig(req ApplyRequest, now time.Time) (run.Config, error) {
	cfg, _, err := applyConfig(req, now, entryApply)
	return cfg, err
}

// BuildApplyBinding also returns the binding run.start records.
func BuildApplyBinding(req ApplyRequest, now time.Time) (run.Config, RunBinding, error) {
	return applyConfig(req, now, entryApply)
}

func PublishChanges(ctx context.Context, req ApplyRequest, changes []snapshot.Change) (*PublicationReceipt, error) {
	return req.publishChanges(ctx, changes)
}

// OriginalText exposes the quoted original of a protected input; quoted is
// the candidate's running total of quoted bytes.
func OriginalText(before snapshot.Entry, code string, quoted *int) string {
	return originalText(before, code, quoted)
}

// Conclude exposes the publication and cleanup step that follows a run.
func Conclude(ctx context.Context, req ApplyRequest, start snapshot.Tree, outcome run.Outcome, runErr error) (ApplyResult, error) {
	return req.conclude(ctx, start, outcome, runErr)
}
