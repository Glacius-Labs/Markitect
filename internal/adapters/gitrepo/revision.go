package gitrepo

import (
	"context"
	"fmt"
	"strings"
)

// ResolveRevision resolves an explicit ref once in the selected worktree root.
// It returns a full commit SHA without acquiring source inputs or a guard.
// The result can be used as a fixed PreviewRequest revision; an empty ref
// never selects HEAD implicitly.
func (r *Repository) ResolveRevision(ctx context.Context, repository string, explicitRef string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("%w: nil repository adapter", ErrUnsafe)
	}
	ctx, cancel := context.WithTimeout(ctx, r.limits.Timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if repository == "" || explicitRef == "" || strings.HasPrefix(explicitRef, "-") {
		return "", fmt.Errorf("%w: explicit repository and non-option revision required", ErrUnsafe)
	}
	git, err := identifyGit(ctx, repository)
	if err != nil {
		return "", err
	}
	return git.commit(explicitRef, false)
}
