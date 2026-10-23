package gitrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var _ run.AssignmentUpdates = (*Workspaces)(nil)

// UpdateAssignment keeps the live child volume and Git history. A clean child
// switches to the fixed parent stand. Own work is checkpointed before normal
// Git merge; conflicts remain in the child for its next task to resolve.
func (w *Workspaces) UpdateAssignment(ctx context.Context, target, source run.AssignmentWorkspace, commit, previous string) error {
	c, err := w.assignment(target)
	if err != nil {
		return err
	}
	p, err := w.assignment(source)
	if err != nil {
		return err
	}
	if c == p || !commitIDPattern.MatchString(commit) || !commitIDPattern.MatchString(previous) {
		return errors.New("child update needs distinct copies and fixed parent commits")
	}
	if present, err := p.ContainsCommit(ctx, commit); err != nil || !present {
		return errors.Join(err, errors.New("parent update commit is unavailable"))
	}
	if present, err := c.ContainsCommit(ctx, previous); err != nil || !present {
		return errors.Join(err, errors.New("previous child foundation is unavailable"))
	}
	if _, err := c.git(ctx, "merge-base", "--is-ancestor", previous, "HEAD"); err != nil {
		return fmt.Errorf("child HEAD no longer descends from its delivered parent stand: %w", err)
	}
	if _, err := p.git(ctx, "merge-base", "--is-ancestor", c.start, commit); err != nil {
		return fmt.Errorf("parent stand is outside this child copy's initial lineage: %w", err)
	}
	unmerged, err := c.git(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return err
	}
	if unmerged != "" {
		return fmt.Errorf("child has unresolved conflicts from earlier work: %s", unmerged)
	}
	if _, err := c.Commit(ctx, "Child work before parent update"); err != nil {
		return err
	}
	own, err := c.git(ctx, "diff", "--name-only", previous, "HEAD", "--")
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, a := range []*assignmentWorkspace{p, c} {
		volume, observed, err := w.ownedVolume(ctx, a.id)
		if err != nil || volume != a.volume || !observed.Exists {
			return errors.Join(err, fmt.Errorf("workspace %s changed before parent update", a.id))
		}
	}
	spec := w.spec(c.volume, false)
	spec.Mounts = append(spec.Mounts, container.Mount{Volume: p.volume, Target: "/source", ReadOnly: true})
	if _, err := w.integrationGit(ctx, spec, "fetch", "--no-tags", "--no-write-fetch-head", "/source", commit); err != nil {
		return fmt.Errorf("parent commit fetch failed: %w", err)
	}
	if own == "" {
		if _, err := w.integrationGit(ctx, spec, "checkout", "--detach", commit); err != nil {
			return err
		}
		head, err := w.integrationGit(ctx, spec, "rev-parse", "HEAD")
		if err != nil || head != commit {
			return errors.Join(err, errors.New("child switch to exact parent commit was not observed"))
		}
		return nil
	}
	_, mergeErr := w.integrationGit(ctx, spec, "merge", "--no-edit", "--no-ff", commit)
	if mergeErr != nil {
		conflicts, inspectErr := w.integrationGit(ctx, spec, "diff", "--name-only", "--diff-filter=U")
		if inspectErr == nil && conflicts != "" {
			return fmt.Errorf("%w on %s: %v", run.ErrAssignmentUpdateConflict, conflicts, mergeErr)
		}
		return fmt.Errorf("child Git merge did not complete: %w", errors.Join(mergeErr, inspectErr))
	}
	if _, err := w.integrationGit(ctx, spec, "merge-base", "--is-ancestor", commit, "HEAD"); err != nil {
		return fmt.Errorf("merged child does not contain delivered parent stand: %w", err)
	}
	return nil
}
