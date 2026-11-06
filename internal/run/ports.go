package run

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Git is the tool's Git work in the user's own repository. A Run works in the
// owner's folder, and its commits are readable from anywhere in the repository.
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
	// Merge merges theirs into ours as a new merge commit. A conflict returns the
	// conflicting paths and no commit.
	Merge(ctx context.Context, ours, theirs, message string) (string, []string, error)
	// Write commits files on top of parent.
	Write(ctx context.Context, parent string, files map[string][]byte, message string) (string, error)
	// Tip reads the tip of a branch, "" when it does not exist.
	Tip(ctx context.Context, branch string) (string, error)
	// Advance moves a branch from old to commit, only if it is still at old. Where
	// it is checked out, the files follow as in a fast-forward, a file that already
	// holds commit's version stays, and one with an edit of its own stops the move.
	// Where it is not, only the ref moves.
	Advance(ctx context.Context, branch, commit, old string) error
	// SetBranch points a branch at a commit.
	SetBranch(ctx context.Context, branch, commit string) error
	// DeleteBranch removes a branch, if it exists.
	DeleteBranch(ctx context.Context, branch string) error
	// Branches lists the branches under a prefix with their tips.
	Branches(ctx context.Context, prefix string) (map[string]string, error)
	// Landings returns the commits reachable from tip and not from since (with an
	// empty since, every one) that add a Result page, newest first.
	Landings(ctx context.Context, tip, since string) ([]string, error)
	// CountCommits says how many commits tip has that from does not.
	CountCommits(ctx context.Context, from, tip string) (int, error)
}

// Commit is one commit of the repository.
type Commit struct {
	ID      string
	Parents []string
	Tree    string
	Message string
}
