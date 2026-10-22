package gitrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var _ run.AssignmentForks = (*Workspaces)(nil)
var _ run.AssignmentCommitForks = (*Workspaces)(nil)

// ForkAssignment secures the parent's current Git stand and clones it into a
// fresh volume. The child shares Git history, not a working directory or live
// files. A failed clone retains the new volume as a partial assignment copy.
func (w *Workspaces) ForkAssignment(ctx context.Context, parent run.AssignmentWorkspace, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	p, err := w.assignment(parent)
	if err != nil {
		return nil, err
	}
	if id == p.id || !(run.WorkspaceSpec{ID: id}).Valid() {
		return nil, errors.New("child assignment needs a distinct valid workspace ID")
	}
	parentCommit, err := p.Commit(ctx, "Parent foundation for child assignment")
	if err != nil {
		return nil, err
	}
	return w.forkAssignmentCommit(ctx, p, parentCommit, id)
}

// ForkAssignmentCommit copies the named Git stand for independent review. It
// never commits the source or reads its live files into the review result.
func (w *Workspaces) ForkAssignmentCommit(ctx context.Context, source run.AssignmentWorkspace, commit string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	p, err := w.assignment(source)
	if err != nil {
		return nil, err
	}
	if id == p.id || !(run.WorkspaceSpec{ID: id}).Valid() {
		return nil, errors.New("review assignment needs a distinct valid workspace ID")
	}
	present, err := p.ContainsCommit(ctx, commit)
	if err != nil || !present {
		return nil, errors.Join(err, errors.New("commissioned Git commit is unavailable"))
	}
	return w.forkAssignmentCommit(ctx, p, commit, id)
}

func (w *Workspaces) forkAssignmentCommit(ctx context.Context, p *assignmentWorkspace, parentCommit string, id run.WorkspaceID) (run.AssignmentWorkspace, error) {
	observed, err := w.Materialize(ctx, run.WorkspaceSpec{ID: id})
	var child *assignmentWorkspace
	if observed.Volume != "" {
		child = &assignmentWorkspace{owner: w, id: id, volume: observed.Volume}
	}
	if err != nil {
		return child, err
	}
	if child == nil {
		return nil, errors.New("child workspace volume was not established")
	}
	if err := w.cloneAssignment(ctx, p, child); err != nil {
		return child, err
	}
	if err := child.configureIdentity(ctx); err != nil {
		return child, err
	}
	if _, err := child.git(ctx, "checkout", "--detach", parentCommit); err != nil {
		return child, err
	}
	start, err := child.git(ctx, "rev-parse", "HEAD")
	if err != nil || start != parentCommit {
		return child, errors.Join(err, fmt.Errorf("child Git copy did not start at the parent's fixed foundation"))
	}
	child.start = start
	return child, nil
}

func (w *Workspaces) cloneAssignment(ctx context.Context, parent, child *assignmentWorkspace) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, a := range []*assignmentWorkspace{parent, child} {
		volume, observed, err := w.ownedVolume(ctx, a.id)
		if err != nil || volume != a.volume || !observed.Exists {
			return errors.Join(err, fmt.Errorf("assignment workspace %s changed before child clone", a.id))
		}
	}
	spec := w.spec(child.volume, false)
	spec.Mounts = append(spec.Mounts, container.Mount{Volume: parent.volume, Target: "/source", ReadOnly: true})
	var output reviewOutput
	output.limit = 16 << 10
	if err := w.runtime.RunHelper(ctx, spec, []string{"/usr/bin/git", "-c", "core.hooksPath=/dev/null", "clone", "--no-hardlinks", "/source", "/work"}, nil, &output); err != nil {
		return err
	}
	if output.exceeded {
		return ErrLimit
	}
	return nil
}
