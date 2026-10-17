package run

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// WorkspaceID is the Host identity of one step workspace; the runtime labels
// the volume with it.
type WorkspaceID string

// Workspaces materializes step workspaces on runtime volumes and observes them.
// It knows no ownership, reviews or tasks: run decides what a workspace holds
// (core rule D3) and what its observed change means (core rule S4).
// Materialize writes exactly the supplied tree into a new volume and reads it
// back. Harvest observes the complete tree after the step's container stopped.
// Close removes the volume. Every result reports actual observations, also when
// an error is returned; uncertainty is reported, never guessed away.
type Workspaces interface {
	Materialize(context.Context, WorkspaceSpec) (WorkspaceObservation, error)
	Harvest(context.Context, HarvestRequest) (WorkspaceHarvest, error)
	Close(context.Context, WorkspaceID) (WorkspaceObservation, error)
}

// WorkspaceSpec names a new workspace and its exact content: the current
// accepted aggregate plus the Manager's own unaccepted changes.
type WorkspaceSpec struct {
	ID   WorkspaceID
	Tree snapshot.Tree
}

// Valid reports whether the spec names a workspace. An empty tree is valid.
func (s WorkspaceSpec) Valid() bool { return identifier(string(s.ID)) }

// Valid reports whether the request names a workspace and bounds the
// observation.
func (r HarvestRequest) Valid() bool {
	return identifier(string(r.ID)) && r.MaxEntries > 0 && r.MaxBytes > 0
}

// WorkspaceObservation is what the runtime observed. Volume is the runtime
// identity that a session plan mounts as its workspace. Tree is the identity
// of the content read back after materialization.
type WorkspaceObservation struct {
	ID        WorkspaceID
	Volume    string
	Tree      snapshot.TreeID
	Closed    bool
	Uncertain bool
	Reason    string
}

// HarvestRequest bounds one observation of a workspace.
type HarvestRequest struct {
	ID         WorkspaceID
	MaxEntries int
	MaxBytes   int64
}

// WorkspaceHarvest is the complete observed content of a workspace. Entries a
// snapshot tree cannot represent, such as symbolic links, special files or
// other modes, are listed in Unsupported by their observed name instead of
// being dropped. An observation that hit a limit is Uncertain.
type WorkspaceHarvest struct {
	ID          WorkspaceID
	Tree        snapshot.Tree
	Unsupported []string
	Uncertain   bool
	Reason      string
}

// Complete reports whether the harvest observed every entry as a tree entry.
func (h WorkspaceHarvest) Complete() bool { return !h.Uncertain && len(h.Unsupported) == 0 }
