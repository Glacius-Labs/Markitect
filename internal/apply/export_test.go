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
