package apply

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// PublishAssignment publishes only the exact result accepted by independent
// Verify. The source guard and read-back are the same as the guarded Apply
// publication path; no Agent commit is written to the owner's repository.
func PublishAssignment(ctx context.Context, id string, selection InputSelection, prepared CapturedSettingsAttempt, result AssignmentRealizationResult, publisher Publisher, control ControlCheck) (*PublicationReceipt, error) {
	if id == "" || result.AcceptedCommit == "" || result.AcceptedCommit != result.TreeCommit || result.Work == nil || len(result.Reviews) == 0 || result.Reviews[len(result.Reviews)-1].AcceptedCommit != result.AcceptedCommit || publisher == nil || control == nil {
		return nil, errors.New("assignment result lacks exact independent Verify acceptance")
	}
	start := prepared.Structural().Model().Inputs().Raw.Census
	return publish(ctx, publisher, control, newPublicationRequest(id, selection, prepared, snapshot.Compare(start, result.Tree)))
}
