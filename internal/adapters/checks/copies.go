package checks

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// DisposableCopies materializes a complete immutable input for one command.
// A partial failure still owns its exact ID and must be closed by the caller.
// It has no Manager, delegation, Git integration or candidate responsibility.
type DisposableCopies interface {
	Materialize(context.Context, CopySpec) (CopyObservation, error)
	Close(context.Context, CopyID) (CopyObservation, error)
}

type CopyID string
type CopySpec struct {
	ID   CopyID
	Tree snapshot.Tree
}
type CopyObservation struct {
	ID                CopyID
	Volume            string
	Directory         string
	Tree              snapshot.TreeID
	Closed, Uncertain bool
	Reason            string
}
