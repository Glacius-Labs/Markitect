package run

import (
	"context"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Git is the host's own Git work (concept 2.0 §9): it reads the target
// branch, puts the starting commit into the Run's remote, reads what the Run
// pushed and lands the result. It works in a private repository of its own
// and never touches the user's working tree.
type Git interface {
	// Fetch reads the tip of a branch of the target repository.
	Fetch(ctx context.Context, branch string) (string, error)
	// Bundle writes a bundle that holds the commit.
	Bundle(ctx context.Context, commit string, w io.Writer) error
	// Import reads a bundle of the Run's remote and returns its branches'
	// tips by branch name.
	Import(ctx context.Context, bundle io.Reader) (map[string]string, error)
	// Commit returns one commit.
	Commit(ctx context.Context, id string) (Commit, error)
	// Contains reports whether commit is tip or one of its ancestors.
	Contains(ctx context.Context, tip, commit string) (bool, error)
	// Changed lists the files that differ between two commits.
	Changed(ctx context.Context, from, to string) ([]analysis.Change, error)
	// History returns every line any commit between two commits adds, with
	// every commit message, and every path those commits touch, for the
	// credential scan at landing.
	History(ctx context.Context, from, to string) (string, []string, error)
	// State is one commit as a project repository state.
	State(ctx context.Context, commit string) project.Repository
	// Merge merges theirs into ours as a new merge commit. A conflict
	// returns the conflicting paths and no commit.
	Merge(ctx context.Context, ours, theirs, message string) (string, []string, error)
	// Write commits files on top of parent; nil content removes a file.
	Write(ctx context.Context, parent string, files map[string][]byte, message string) (string, error)
	// Push sets a branch of the target repository to commit if it is still
	// at old; an empty old means the branch must not exist yet, and an empty
	// commit deletes the branch.
	Push(ctx context.Context, branch, commit, old string) error
	// LastLanded returns the newest commit reachable from tip that adds a
	// landing Result, and false when there is none.
	LastLanded(ctx context.Context, tip string) (string, bool, error)
}

// Commit is one commit of the host's repository.
type Commit struct {
	ID      string
	Parents []string
	Tree    string
	Message string
}

// Remotes starts Run remotes (concept 2.0 §6.3, §9).
type Remotes interface {
	// Start starts the Run's remote with its plan, seeded from a bundle
	// holding the plan's base commit.
	Start(ctx context.Context, plan *runremote.Plan, bundle io.Reader) (Remote, error)
}

// Remote is one Run's started remote.
type Remote interface {
	// URL is the repository's address and Network the container network
	// the Manager containers reach it on.
	URL() string
	Network() string
	// Export writes a bundle of every branch.
	Export(ctx context.Context, w io.Writer) error
	// Log returns every push the hooks decided, oldest first.
	Log(ctx context.Context) ([]Push, error)
	// Check runs the required checks on a commit the bundle holds, without
	// changing any branch.
	Check(ctx context.Context, bundle io.Reader, commit string) ([]runremote.CheckResult, error)
	// Close removes the remote and everything it created.
	Close(ctx context.Context) error
}

// Push is one push the remote's hooks decided.
type Push struct {
	User     string
	Updates  []runremote.Update
	Decision runremote.Decision
}
