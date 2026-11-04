package run

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Git is the tool's Git work in the user's own repository. A Run works in a
// worktree of it, so every commit is readable from anywhere in the repository.
type Git interface {
	// Root is the top of the working tree that holds dir.
	Root(ctx context.Context, dir string) (string, error)
	// Branch is the branch checked out at dir, "" when HEAD is detached.
	Branch(ctx context.Context, dir string) (string, error)
	// Head is the commit checked out at dir.
	Head(ctx context.Context, dir string) (string, error)
	// Uncommitted lists what differs from the last commit at dir, outside
	// Markitect's folder and outside the provider's own files, as `git status`
	// sees it.
	Uncommitted(ctx context.Context, dir string) ([]string, error)
	// CommitSpec commits Markitect's folder at dir with the user's identity, and
	// returns the commit. With nothing to commit it returns HEAD.
	CommitSpec(ctx context.Context, dir, message string) (string, error)
	// Candidate makes a commit of every file Git would take in the worktree at
	// dir, on top of its HEAD, without changing the worktree, its index or HEAD.
	// With nothing changed it returns HEAD.
	Candidate(ctx context.Context, dir, message string) (string, error)
	// Adopt moves the worktree at dir to a commit made from its own files.
	Adopt(ctx context.Context, dir, commit string) error
	// Commit returns one commit.
	Commit(ctx context.Context, id string) (Commit, error)
	// Contains reports whether commit is tip or one of its ancestors.
	Contains(ctx context.Context, tip, commit string) (bool, error)
	// Changed lists the files that differ between two commits.
	Changed(ctx context.Context, from, to string) ([]analysis.Change, error)
	// Links returns the symbolic links of a commit, with where they point.
	Links(ctx context.Context, commit string) (map[string]string, error)
	// State is one commit as a project repository state.
	State(ctx context.Context, commit string) project.Repository
	// Export writes the files of a commit into an empty folder.
	Export(ctx context.Context, commit, dir string) error
	// Merge merges theirs into ours as a new merge commit. A conflict returns the
	// conflicting paths and no commit.
	Merge(ctx context.Context, ours, theirs, message string) (string, []string, error)
	// Write commits files on top of parent.
	Write(ctx context.Context, parent string, files map[string][]byte, message string) (string, error)
	// Tip reads the tip of a branch, "" when it does not exist.
	Tip(ctx context.Context, branch string) (string, error)
	// Advance moves a branch from old to commit, only if it is still at old: by a
	// fast-forward in the working tree that has it checked out, so that the
	// user's uncommitted edits stay, or by moving the ref where none does.
	Advance(ctx context.Context, branch, commit, old string) error
	// SetBranch points a branch at a commit.
	SetBranch(ctx context.Context, branch, commit string) error
	// DeleteBranch removes a branch, if it exists.
	DeleteBranch(ctx context.Context, branch string) error
	// Branches lists the branches under a prefix with their tips.
	Branches(ctx context.Context, prefix string) (map[string]string, error)
	// WorktreeOf returns the working tree that has a branch checked out, or "".
	WorktreeOf(ctx context.Context, branch string) (string, error)
	// RemoveWorktree removes the worktree that has a branch checked out, if there
	// is one.
	RemoveWorktree(ctx context.Context, branch string) error
	// LastLanded returns the newest commit reachable from tip that adds a landing
	// Result, and false when there is none.
	LastLanded(ctx context.Context, tip string) (string, bool, error)
	// Landings returns the commits reachable from tip and not from since (with an
	// empty since, every one) that add a landing Result, newest first.
	Landings(ctx context.Context, tip, since string) ([]string, error)
}

// Commit is one commit of the repository.
type Commit struct {
	ID      string
	Parents []string
	Tree    string
	Message string
}

// Checks runs the required checks.
type Checks interface {
	// Run runs the checks in a clean folder, stopping at the first failure. A
	// failing check is a result; only a check that could not start is NotRun.
	Run(ctx context.Context, dir string, checks []project.Check) []CheckResult
}

// CheckResult is one required check on one state.
type CheckResult struct {
	Name     string  `json:"name"`
	Passed   bool    `json:"passed"`
	Exit     int     `json:"exit"`
	TimedOut bool    `json:"timedOut,omitempty"`
	Seconds  float64 `json:"seconds"`
	// NotRun: the check's command could not start, which is the environment's
	// failure and not the work's.
	NotRun bool `json:"notRun,omitempty"`
	// Output is the tail of the check's combined output.
	Output string `json:"output,omitempty"`
}

// How a provider says a session stands.
type SessionState string

const (
	// SessionWorking is a session that is working or waiting for a person.
	SessionWorking SessionState = "working"
	// SessionEnded is a session that finished, failed or stopped.
	SessionEnded SessionState = "ended"
	// SessionUnknown is a session the provider no longer lists.
	SessionUnknown SessionState = "unknown"
)

// SessionSpec is one session a Command starts. Everything the provider needs to
// be told, it is told here.
type SessionSpec struct {
	Run string
	// Dir is the owner's folder, where the provider makes the worktree.
	Dir string
	// Agent is the agent the session runs as, "" for a plain session.
	Agent string
	// Model and Effort are empty for the provider adapter's default.
	Model, Effort string
	Prompt        string
	// Store is the folder of the run state, for the hooks.
	Store string
}

// Sessions starts and follows the provider's background sessions. A provider
// needs a way to start one in a worktree with settings, to resume it, hooks on
// stopping and on tool calls, and an agent type for the reviewer.
type Sessions interface {
	// Start starts a background session and returns its ID.
	Start(ctx context.Context, spec SessionSpec) (string, error)
	// Resume continues an ended session in the background and returns the ID it
	// now has, which may be a new one.
	Resume(ctx context.Context, id string, spec SessionSpec) (string, error)
	// State says how a session stands.
	State(ctx context.Context, id string) (SessionState, error)
	// Stop ends a session's process.
	Stop(ctx context.Context, id string) error
	// Remove forgets an ended session.
	Remove(ctx context.Context, id string) error
}
