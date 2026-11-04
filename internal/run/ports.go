package run

import (
	"context"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Git is the Command's own Git work outside the container (concept 3.0 §6): it
// reads the target branch, reads what the sandbox hands out, runs the gate's
// comparisons and lands the result. It works in a private repository of its
// own and never touches the user's working tree.
type Git interface {
	// Fetch reads the tip of a branch of the target repository.
	Fetch(ctx context.Context, branch string) (string, error)
	// Bundle writes a bundle that holds the commit.
	Bundle(ctx context.Context, commit string, w io.Writer) error
	// Import reads a bundle the sandbox wrote.
	Import(ctx context.Context, bundle io.Reader) error
	// Commit returns one commit.
	Commit(ctx context.Context, id string) (Commit, error)
	// Contains reports whether commit is tip or one of its ancestors.
	Contains(ctx context.Context, tip, commit string) (bool, error)
	// Changed lists the files that differ between two commits.
	Changed(ctx context.Context, from, to string) ([]analysis.Change, error)
	// Diff is the text of the changes between two commits.
	Diff(ctx context.Context, from, to string) (string, error)
	// Links returns the symbolic links of a commit, with where they point.
	Links(ctx context.Context, commit string) (map[string]string, error)
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
	// Landings returns the commits reachable from tip and not from since
	// (with an empty since, every one) that add a landing Result, newest
	// first.
	Landings(ctx context.Context, tip, since string) ([]string, error)
	// Branches lists the target repository's branches under a prefix with
	// their tips, and makes their commits readable.
	Branches(ctx context.Context, prefix string) (map[string]string, error)
}

// Commit is one commit of the Command's repository.
type Commit struct {
	ID      string
	Parents []string
	Tree    string
	Message string
}

// Sandboxes starts the one container of a Command (concept 3.0 §9).
type Sandboxes interface {
	// Start starts the Command's container, with the commit the bundle holds
	// inside it. Nothing of the user's is mounted.
	Start(ctx context.Context, run, commit string, bundle io.Reader) (Sandbox, error)
}

// Sandbox is one Command's started container. Code enters and leaves it only
// as a Git commit.
type Sandbox interface {
	agentport.Shell
	// Deliver makes a user fresh: it ends the user's processes, clears its home
	// and folder, and makes a new clean copy of a commit the sandbox holds. It
	// returns the copy's folder. Nothing of the user's earlier sessions or of
	// any other user is in it.
	Deliver(ctx context.Context, user, commit string) (string, error)
	// Collect ends the implementer's processes and commits its folder on top
	// of parent with the Command's message: what Git does not take is not
	// there. It returns the commit and a bundle that holds it for the host; a
	// folder with no change returns parent and no bundle.
	Collect(ctx context.Context, parent, message string) (string, io.ReadCloser, error)
	// Check runs the required checks on a commit in a clean copy, as a user
	// that is neither role and is given no token, and stops at the first
	// failure. A bundle, when given, brings a commit only the host holds. A
	// failing check is a result; an error means the checks could not run.
	Check(ctx context.Context, bundle io.Reader, commit string, checks []project.Check) ([]CheckResult, error)
	// Note ends a user's processes, so that nothing it left can change the note
	// any more, and reads a file it left outside its folder, and false when
	// there is none.
	Note(ctx context.Context, user, name string) ([]byte, bool, error)
	// NotePath is where a user writes a note of that name.
	NotePath(user, name string) string
	// Close ends the container and removes everything it created.
	Close(ctx context.Context) error
}

// CheckResult is one required check on one commit.
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
