package run

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// WorkspaceID is the Orchestrator's identity of one step workspace; the runtime
// labels the volume with it.
type WorkspaceID string

// AssignmentWorkspace is one Agent's independent Git working copy. A submitted
// commit remains addressable after the Agent continues editing the same copy.
// Closing the Agent does not remove this workspace: incomplete work and earlier
// submissions remain available for inspection until the caller releases it.
type AssignmentWorkspace interface {
	ID() WorkspaceID
	Volume() string
	StartingCommit() string
	Commit(context.Context, string) (string, error)
	ContainsCommit(context.Context, string) (bool, error)
	Close(context.Context) error
}

// AssignmentWorkspaces prepares a fresh Git copy from a captured tree. It
// refuses an existing ID and returns its initial internal commit.
type AssignmentWorkspaces interface {
	PrepareAssignment(context.Context, WorkspaceSpec) (AssignmentWorkspace, error)
}

// AssignmentForks starts a direct child from its parent's fixed current Git
// stand in a fresh copy. It retains a partially created child copy on error.
type AssignmentForks interface {
	ForkAssignment(context.Context, AssignmentWorkspace, WorkspaceID) (AssignmentWorkspace, error)
}

// AssignmentCommitForks copies one named retained commit without changing the
// source HEAD, index or working files. Later source work does not change it.
type AssignmentCommitForks interface {
	ForkAssignmentCommit(context.Context, AssignmentWorkspace, string, WorkspaceID) (AssignmentWorkspace, error)
}

// Workspaces materializes step workspaces on runtime volumes and observes them.
// It knows no ownership, reviews or tasks: run decides what a workspace holds
// (core rule D3) and what its observed change means (core rule S4), and run
// stores harvested candidates as payloads in the Orchestrator's private journal store.
//
// Materialize writes exactly the supplied tree into a new volume and reads it
// back; it refuses an ID that already exists, so it never overwrites content
// that was not harvested. Harvest observes the complete tree after the step's
// container stopped; while any container still mounts the volume, it reports
// Uncertain. Peek observes the workspace while its step may still write; its
// tree is advisory, the Orchestrator only uses it before an outcome to refuse a call
// whose changes would not be captured or whose limited delivery would deliver
// nothing, and a candidate still comes only from Harvest after the step's
// container stopped. Close is idempotent and reports
// Closed for an absent volume. Every result reports actual observations, also
// when an error is returned.
//
// Observation follows Git's view of files: a regular file is Executable
// exactly when its owner execute bit is set, other permission bits are
// ignored, and empty directories are ignored. Only non-regular entries, such as
// symbolic links or devices, and names a snapshot Path rejects are Unsupported.
// Harvest excludes exactly what the repository census excludes: Git
// administration and .markitect/state.
type Workspaces interface {
	Materialize(context.Context, WorkspaceSpec) (WorkspaceObservation, error)
	Harvest(context.Context, HarvestRequest) (WorkspaceHarvest, error)
	Peek(context.Context, HarvestRequest) (WorkspaceHarvest, error)
	Close(context.Context, WorkspaceID) (WorkspaceObservation, error)
}

// WorkspaceSpec names a new workspace and its exact content. Directories are
// relative directory paths created empty in addition to the tree, such as the
// mount points of write plans; empty directories are not part of a harvest.
type WorkspaceSpec struct {
	ID          WorkspaceID
	Tree        snapshot.Tree
	Directories []string
}

// Valid reports whether the spec names a workspace. An empty tree is valid.
func (s WorkspaceSpec) Valid() bool { return identifier(string(s.ID)) }

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

const (
	maxHarvestEntries = 1 << 20
	maxHarvestBytes   = 1 << 32
)

// HarvestRequest bounds one observation of a workspace.
type HarvestRequest struct {
	ID         WorkspaceID
	MaxEntries int
	MaxBytes   int64
}

// Valid reports whether the request names a workspace and bounds the
// observation within the Orchestrator's ceilings.
func (r HarvestRequest) Valid() bool {
	return identifier(string(r.ID)) && r.MaxEntries > 0 && r.MaxEntries <= maxHarvestEntries && r.MaxBytes > 0 && r.MaxBytes <= maxHarvestBytes
}

// WorkspaceHarvest is the complete observed content of a workspace. Unsupported
// entries are listed by their observed name instead of being dropped. An
// observation that hit a limit, or found the volume still mounted, is
// Uncertain.
type WorkspaceHarvest struct {
	ID          WorkspaceID
	Tree        snapshot.Tree
	Unsupported []string
	Uncertain   bool
	Reason      string
}

// Complete reports whether the harvest observed every entry as a tree entry.
func (h WorkspaceHarvest) Complete() bool { return !h.Uncertain && len(h.Unsupported) == 0 }
