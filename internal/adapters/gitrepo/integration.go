package gitrepo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// ErrIntegrationConflict means Git could not apply a child's change to a
// temporary copy of the parent's fixed stand. The live parent is untouched.
var ErrIntegrationConflict = errors.New("child Git result conflicts with parent stand")

// ErrIntegrationUncertain stops retries when Git's final transfer could not
// be read back as either the old parent head or the integrated result.
var ErrIntegrationUncertain = errors.New("parent Git integration outcome is uncertain")

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
	w.mu.Lock()
	volume, observed, err := w.ownedVolume(ctx, a.id)
	if err != nil || volume != a.volume || !observed.Exists {
		w.mu.Unlock()
		return "", errors.Join(err, fmt.Errorf("child workspace changed before review"))
	}
	spec := w.spec(a.volume, true)
	var output reviewOutput
	output.limit = 8 << 20
	err = w.runtime.RunHelper(ctx, spec, assignmentGitArgs("diff", "--binary", "--no-ext-diff", "--no-textconv", a.start, handoff.ResultCommit, "--"), nil, &output)
	w.mu.Unlock()
	if output.exceeded {
		return "", fmt.Errorf("child Git difference exceeds the parent review delivery limit")
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

type reviewOutput struct {
	// Keep Buffer named: embedding would promote ReaderFrom, letting io.Copy
	// bypass the bounded Write method used by the helper's stdout stream.
	data     bytes.Buffer
	limit    int
	exceeded bool
}

func (b *reviewOutput) Len() int       { return b.data.Len() }
func (b *reviewOutput) String() string { return b.data.String() }

func (b *reviewOutput) Write(p []byte) (int, error) {
	room := b.limit - b.Len()
	if room < len(p) {
		b.exceeded = true
		if room > 0 {
			_, _ = b.data.Write(p[:room])
		}
		return len(p), nil // drain the helper without retaining unbounded output
	}
	return b.data.Write(p)
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
	w.mu.Lock()
	blocked := w.uncertain[p.id]
	w.mu.Unlock()
	if blocked {
		return "", ErrIntegrationUncertain
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
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	temporary := "/work/.git/markitect-integration-" + hex.EncodeToString(nonce[:])
	if _, err := w.integrationGit(ctx, parentSpec, "worktree", "add", "--detach", temporary, parentBefore); err != nil {
		return "", fmt.Errorf("temporary parent Git stand could not start: %w", err)
	}
	removeTemporary := func() error {
		_, err := w.integrationGit(ctx, parentSpec, "worktree", "remove", "--force", temporary)
		return err
	}
	applyErr, sourceErr := w.patch(ctx, childSpec, parentSpec, temporary, base, handoff.ResultCommit)
	if applyErr != nil || sourceErr != nil {
		unmerged, inspectErr := w.integrationGitAt(ctx, parentSpec, temporary, "diff", "--name-only", "--diff-filter=U")
		cleanupErr := removeTemporary()
		if inspectErr == nil && cleanupErr == nil && sourceErr == nil && unmerged != "" && applyErr != nil {
			return "", fmt.Errorf("%w on %s: %v", ErrIntegrationConflict, unmerged, applyErr)
		}
		return "", fmt.Errorf("temporary Git integration failed: %w", errors.Join(applyErr, sourceErr, inspectErr, cleanupErr))
	}
	if _, err := w.integrationGitAt(ctx, parentSpec, temporary, "commit", "-m", "Integrate child result "+handoff.ResultCommit); err != nil {
		cleanupErr := removeTemporary()
		return "", fmt.Errorf("temporary parent commit failed: %w", errors.Join(err, cleanupErr))
	}
	result, err := w.integrationGitAt(ctx, parentSpec, temporary, "rev-parse", "HEAD")
	if err != nil {
		cleanupErr := removeTemporary()
		return "", errors.Join(err, cleanupErr)
	}
	if !commitIDPattern.MatchString(result) || result == parentBefore {
		cleanupErr := removeTemporary()
		return "", errors.Join(errors.New("parent integration commit was not observed"), cleanupErr)
	}
	current, err := w.integrationGit(ctx, parentSpec, "rev-parse", "HEAD")
	if err != nil || current != parentBefore {
		cleanupErr := removeTemporary()
		return "", errors.Join(err, cleanupErr, fmt.Errorf("parent Git head changed during child integration"))
	}
	beforeStatus, err := w.integrationGit(ctx, parentSpec, "status", "--porcelain")
	if err != nil {
		cleanupErr := removeTemporary()
		return "", errors.Join(err, cleanupErr)
	}
	if _, mergeErr := w.integrationGit(ctx, parentSpec, "merge", "--ff-only", result); mergeErr != nil {
		afterHead, headErr := w.integrationGit(ctx, parentSpec, "rev-parse", "HEAD")
		afterStatus, statusErr := w.integrationGit(ctx, parentSpec, "status", "--porcelain")
		cleanupErr := removeTemporary()
		if headErr == nil && afterHead == result {
			return result, fmt.Errorf("parent Git integrated the child but reported a helper error: %w", errors.Join(mergeErr, statusErr, cleanupErr))
		}
		if headErr == nil && statusErr == nil && afterHead == parentBefore && afterStatus == beforeStatus {
			return "", fmt.Errorf("parent Git refused the fast-forward without changing its stand: %w", errors.Join(mergeErr, cleanupErr))
		}
		if w.uncertain == nil {
			w.uncertain = make(map[run.WorkspaceID]bool)
		}
		w.uncertain[p.id] = true
		return "", fmt.Errorf("%w: %v", ErrIntegrationUncertain, errors.Join(mergeErr, headErr, statusErr, cleanupErr))
	}
	if err := removeTemporary(); err != nil {
		return result, fmt.Errorf("integrated parent commit %s but temporary Git worktree remains: %w", result, err)
	}
	return result, nil
}

func assignmentGitArgs(args ...string) []string {
	return assignmentGitArgsAt("/work", args...)
}

func assignmentGitArgsAt(directory string, args ...string) []string {
	return append([]string{"/usr/bin/git", "-c", "core.hooksPath=/dev/null", "-c", "user.name=Markitect", "-c", "user.email=assignment@markitect.invalid", "-C", directory}, args...)
}

// integrationGit runs while Workspaces.mu holds both volume ownerships stable.
func (w *Workspaces) integrationGit(ctx context.Context, spec container.ContainerSpec, args ...string) (string, error) {
	return w.integrationGitAt(ctx, spec, "/work", args...)
}

func (w *Workspaces) integrationGitAt(ctx context.Context, spec container.ContainerSpec, directory string, args ...string) (string, error) {
	var output reviewOutput
	output.limit = 16 << 10
	if err := w.runtime.RunHelper(ctx, spec, assignmentGitArgsAt(directory, args...), nil, &output); err != nil {
		return "", err
	}
	if output.exceeded {
		return "", ErrLimit
	}
	return strings.TrimSpace(output.String()), nil
}

// patch streams one fixed child diff into the temporary parent Git worktree.
// A conflict cannot change the live parent's index or working files.
func (w *Workspaces) patch(ctx context.Context, childSpec, parentSpec container.ContainerSpec, directory, base, result string) (error, error) {
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := w.runtime.RunHelper(ctx, childSpec, assignmentGitArgs("diff", "--binary", "--no-ext-diff", "--no-textconv", base, result, "--"), nil, writer)
		_ = writer.CloseWithError(err)
		done <- err
	}()
	var output reviewOutput
	output.limit = 16 << 10
	applyErr := w.runtime.RunHelper(ctx, parentSpec, assignmentGitArgsAt(directory, "apply", "--3way", "--index", "-"), reader, &output)
	_ = reader.CloseWithError(applyErr)
	childErr := <-done
	if output.exceeded {
		applyErr = errors.Join(applyErr, ErrLimit)
	}
	return applyErr, childErr
}
