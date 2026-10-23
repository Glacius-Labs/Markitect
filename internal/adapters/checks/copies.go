package checks

import (
	"context"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"

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
	Subpath           string
	Tree              snapshot.TreeID
	Closed, Uncertain bool
	Reason            string
}

func (c CopyObservation) usable() bool { return c.Volume != "" }
func (c CopyObservation) sameLocation(other CopyObservation) bool {
	return c.Volume == other.Volume && c.Subpath == other.Subpath
}
func (c CopyObservation) mount(target string) container.Mount {
	return container.Mount{Volume: c.Volume, Subpath: c.Subpath, Target: target, NoCopy: true}
}
