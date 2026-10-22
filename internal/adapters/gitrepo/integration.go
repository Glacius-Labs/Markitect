package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// ErrIntegrationConflict means Git could not apply a child's change to its
// parent's current stand. The owning child receives a concrete correction;
// the parent is restored to its fixed pre-integration internal commit.
var ErrIntegrationConflict = errors.New("child Git result conflicts with parent stand")

var _ run.AssignmentIntegration = (*Workspaces)(nil)

func (w *Workspaces) assignment(copy run.AssignmentWorkspace) (*assignmentWorkspace, error) {
	a, ok := copy.(*assignmentWorkspace)
	if !ok || a == nil || a.owner != w || a.start == "" {
		return nil, errors.New("Git integration needs a complete workspace owned by this adapter")
	}
	return a, nil
}

func (w *Workspaces) checkedHandoff(ctx context.Context, copy run.AssignmentWorkspace, handoff run.AssignmentHandoff, previous string) (*assignmentWorkspace, string, error) {
	a, err := w.assignment(copy)
	if err != nil {
		return nil, "", err
	}
	if handoff.StartingCommit != a.start || !commitIDPattern.MatchString(handoff.ResultCommit) {
		return nil, "", errors.New("child handoff does not name this workspace's fixed starting and result commits")
	}
	base := a.start
	if previous != "" {
		if !commitIDPattern.MatchString(previous) {
			return nil, "", errors.New("previous child result is not a full commit identity")
		}
		if _, err := a.git(ctx, "merge-base", "--is-ancestor", a.start, previous); err != nil {
			return nil, "", fmt.Errorf("previous child result does not descend from its starting stand: %w", err)
		}
		base = previous
	}
	for _, id := range []string{base, handoff.ResultCommit} {
		present, err := a.ContainsCommit(ctx, id)
		if err != nil || !present {
			return nil, "", errors.Join(err, fmt.Errorf("child result commit %s is unavailable", id))
		}
	}
	if _, err := a.git(ctx, "merge-base", "--is-ancestor", base, handoff.ResultCommit); err != nil {
		return nil, "", fmt.Errorf("child result does not descend from its handed-off base: %w", err)
	}
	return a, base, nil
}

// Changes reads the fixed result's actual Git difference, independent of any
// subsequent child edits. A result too large to deliver for parent review is
// refused explicitly rather than silently shortened.
func (w *Workspaces) Changes(ctx context.Context, copy run.AssignmentWorkspace, handoff run.AssignmentHandoff) (string, error) {
	a, _, err := w.checkedHandoff(ctx, copy, handoff, "")
	if err != nil {
		return "", err
	}
	diff, err := a.git(ctx, "diff", "--binary", "--no-ext-diff", "--no-textconv", a.start, handoff.ResultCommit, "--")
	if err != nil {
		return "", err
	}
	if len(diff) > 8<<20 {
		return "", fmt.Errorf("child Git difference exceeds the parent review delivery limit")
	}
	return diff, nil
}

// Integrate transfers the child's fixed Git delta into the parent's own copy
// with Git's three-way patch handling. The two assignment repositories have
// independent root commits, so a direct merge is not available. A successful
// integration is secured as a named parent commit. previous is the last child
// result integrated by this parent, or empty on the first handoff.
func (w *Workspaces) Integrate(ctx context.Context, parent, child run.AssignmentWorkspace, handoff run.AssignmentHandoff, previous string) (string, error) {
	p, err := w.assignment(parent)
	if err != nil {
		return "", err
	}
	c, base, err := w.checkedHandoff(ctx, child, handoff, previous)
	if err != nil {
		return "", err
	}
	if p == c {
		return "", errors.New("a child result cannot integrate into its own workspace")
	}
	// Fix the parent's own work before applying the child's delta. This is an
	// internal commit; no owner repository receives a commit or push.
	parentBefore, err := p.Commit(ctx, "Parent stand before child integration")
	if err != nil {
		return "", err
	}
	paths, err := c.git(ctx, "diff", "--name-only", base, handoff.ResultCommit, "--")
	if err != nil {
		return "", err
	}
	if paths == "" {
		return parentBefore, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, a := range []*assignmentWorkspace{p, c} {
		volume, observed, err := w.ownedVolume(ctx, a.id)
		if err != nil || volume != a.volume || !observed.Exists {
			return "", errors.Join(err, fmt.Errorf("assignment workspace %s changed before Git integration", a.id))
		}
	}
	parentSpec, childSpec := w.spec(p.volume, false), w.spec(c.volume, true)
	applyErr, sourceErr := w.patch(ctx, childSpec, parentSpec, base, handoff.ResultCommit, true)
	if sourceErr != nil {
		return "", fmt.Errorf("child patch could not be read: %w", sourceErr)
	}
	if applyErr != nil {
		return "", fmt.Errorf("%w: %v", ErrIntegrationConflict, applyErr)
	}
	applyErr, sourceErr = w.patch(ctx, childSpec, parentSpec, base, handoff.ResultCommit, false)
	if applyErr != nil || sourceErr != nil {
		if applyErr != nil {
			unmerged, inspectErr := w.integrationGit(ctx, parentSpec, "diff", "--name-only", "--diff-filter=U")
			if inspectErr == nil && unmerged != "" {
				if _, resetErr := w.integrationGit(ctx, parentSpec, "reset", "--hard", parentBefore); resetErr != nil {
					return "", fmt.Errorf("Git conflict left an uncertain parent stand: %w", errors.Join(applyErr, resetErr))
				}
				return "", fmt.Errorf("%w on %s: %v", ErrIntegrationConflict, unmerged, applyErr)
			}
		}
		return "", fmt.Errorf("Git integration changed after its successful check: %w", errors.Join(applyErr, sourceErr))
	}
	if _, err := w.integrationGit(ctx, parentSpec, "commit", "-m", "Integrate child result "+handoff.ResultCommit); err != nil {
		return "", fmt.Errorf("applied child changes remain in parent workspace without a result commit: %w", err)
	}
	result, err := w.integrationGit(ctx, parentSpec, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if !commitIDPattern.MatchString(result) || result == parentBefore {
		return "", errors.New("parent integration commit was not observed")
	}
	return result, nil
}

func assignmentGitArgs(args ...string) []string {
	return append([]string{"/usr/bin/git", "-c", "core.hooksPath=/dev/null", "-c", "user.name=Markitect", "-c", "user.email=assignment@markitect.invalid", "-C", "/work"}, args...)
}

// integrationGit runs while Workspaces.mu holds both volume ownerships stable.
func (w *Workspaces) integrationGit(ctx context.Context, spec container.ContainerSpec, args ...string) (string, error) {
	var output boundedOutput
	output.limit = 16 << 10
	if err := w.runtime.RunHelper(ctx, spec, assignmentGitArgs(args...), nil, &output); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

// patch streams one fixed child diff to Git in the parent copy. The check
// phase does not mutate the parent. Git may still discover a three-way
// conflict during the applying phase; Integrate restores the parent then.
func (w *Workspaces) patch(ctx context.Context, childSpec, parentSpec container.ContainerSpec, base, result string, check bool) (error, error) {
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := w.runtime.RunHelper(ctx, childSpec, assignmentGitArgs("diff", "--binary", "--no-ext-diff", "--no-textconv", base, result, "--"), nil, writer)
		_ = writer.CloseWithError(err)
		done <- err
	}()
	args := []string{"apply", "--3way", "--index"}
	if check {
		args = append(args, "--check")
	}
	args = append(args, "-")
	var output boundedOutput
	output.limit = 16 << 10
	applyErr := w.runtime.RunHelper(ctx, parentSpec, assignmentGitArgs(args...), reader, &output)
	_ = reader.CloseWithError(applyErr)
	childErr := <-done
	return applyErr, childErr
}
