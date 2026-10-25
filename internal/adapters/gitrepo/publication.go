package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/tooling/guardedwrite"
)

// Publisher uses host Git and durable state after every Run has closed. The
// separate repository is never an Agent environment or a former live copy.
type Publisher struct {
	Repository  *Repository
	Store       *LocalGit
	Checks      run.CandidateChecks
	CheckCopies interface {
		Materialize(context.Context, snapshot.Tree, []snapshot.Path) (run.DisposableCopy, error)
	}
}

type publicationCheckCopy struct {
	run.Copy
	materialize func(context.Context, snapshot.Tree, []snapshot.Path) (run.DisposableCopy, error)
}

func (p *Publisher) Preflight(ctx context.Context, prepared project.Prepared) error {
	if p == nil || p.Repository == nil || !prepared.Valid() {
		return fmt.Errorf("checked publication target required")
	}
	current, err := p.Repository.Capture(ctx, project.CaptureRequest{Selection: prepared.Selection()})
	if err != nil {
		return err
	}
	if current.Guard != prepared.Attempt().Model().Inputs().Guard {
		return fmt.Errorf("%w: publication target/source changed before Run", ErrChanged)
	}
	entries := current.Raw.Census.Entries()
	if len(entries) == 0 {
		return fmt.Errorf("empty publication source")
	}
	_, err = guardedwrite.CaptureV1Files(ctx, prepared.Selection().Repository, []string{entries[0].Path().String()})
	return err
}

func (c publicationCheckCopy) Materialize(ctx context.Context, tree snapshot.Tree, dirs []snapshot.Path) (run.DisposableCopy, error) {
	return c.materialize(ctx, tree, dirs)
}

func (p *Publisher) Publish(ctx context.Context, request apply.PublicationRequest) (out apply.Publication, err error) {
	prepared, approval := request.Prepared, request.Approval
	if p == nil || p.Repository == nil || p.Store == nil {
		return out, fmt.Errorf("publication adapters unavailable")
	}
	if err = apply.ValidateApproval(prepared, approval); err != nil {
		return out, err
	}
	ref := *approval.Candidate
	out.Candidate = ref
	inputs, err := p.Store.retained(ctx, ref, prepared)
	if err != nil {
		return out, err
	}
	root, err := os.MkdirTemp(p.Store.TempRoot, "markitect-publication-")
	if err != nil {
		return out, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(root); cleanupErr != nil {
			out.Readback = false
			err = errors.Join(err, fmt.Errorf("publication cleanup unconfirmed: %w", cleanupErr))
		}
	}()
	dir := filepath.Join(root, "candidate")
	if err = os.Mkdir(dir, 0700); err != nil {
		return out, err
	}
	if err = p.Store.Reopen(ctx, ref, dir); err != nil {
		return out, err
	}
	copy := &localCopy{service: &localService{root: root, request: run.Request{Prepared: prepared}, inputs: inputs}, location: dir}
	fixed := run.Commit{ID: ref.Commit, Tree: ref.Tree}
	attempt, err := copy.Inspect(ctx, fixed)
	if err != nil {
		return out, err
	}
	// The complete candidate check keeps no stale registry link.
	if err = prepared.ValidateCandidate(attempt); err != nil {
		return out, err
	}
	if len(prepared.Checks()) > 0 {
		if p.Checks == nil {
			return out, fmt.Errorf("publication check runner unavailable")
		}
		var checkCopy run.Copy = copy
		if p.CheckCopies != nil {
			checkCopy = publicationCheckCopy{Copy: copy, materialize: p.CheckCopies.Materialize}
		}
		out.Checks, err = p.Checks.Check(ctx, prepared.Checks(), checkCopy, fixed)
		if err != nil {
			return out, err
		}
		dirs, err := project.CheckDirectories(attempt.Model().Inputs().Raw)
		if err != nil {
			return out, err
		}
		if err = run.ValidateCheckObservations(prepared.Checks(), out.Checks, fixed, attempt.Model().Inputs().Raw.Census, dirs); err != nil {
			return out, err
		}
	}
	baseline := prepared.Attempt().Model().Inputs()
	before, after := baseline.Raw.Census, attempt.Model().Inputs().Raw.Census
	delta := snapshot.Compare(before, after)
	out.NoChange = len(delta) == 0
	var changes []guardedwrite.Change
	var selected []string
	for _, d := range delta {
		change := guardedwrite.Change{Path: d.Path().String(), Delete: d.Kind() == snapshot.Delete}
		if e, exists := d.After(); exists {
			beforeEntry, existed := d.Before()
			if runtime.GOOS == "windows" && (existed && beforeEntry.Mode() != e.Mode() || !existed && e.Mode() == snapshot.Executable) {
				return out, fmt.Errorf("%w: Windows publication cannot establish a changed executable mode at %s", ErrUnrepresentable, e.Path())
			}
			change.Bytes = e.Bytes()
			change.Mode = 0644
			if e.Mode() == snapshot.Executable {
				change.Mode = 0755
			}
		}
		changes = append(changes, change)
		selected = append(selected, change.Path)
	}
	if len(selected) == 0 {
		// Acquire the cooperating lock and validate source/target even on no-change.
		selected = []string{before.Entries()[0].Path().String()}
	}
	capture, err := guardedwrite.CaptureV1Files(ctx, prepared.Selection().Repository, selected)
	if err != nil {
		return out, err
	}
	expected := before
	guard := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		selection := prepared.Selection()
		// Realization may add/delete registry shards. Their exact current bytes
		// are covered by expected Census; no original shard is reacquired by name.
		selection.Registry = nil
		a, err := p.Repository.acquire(ctx, selection, "", false)
		if err != nil {
			return err
		}
		if !a.raw.Census.Equal(expected) {
			return fmt.Errorf("%w: complete publication target Census drift", ErrChanged)
		}
		// Registry realization and completed transfers legitimately change those
		// two values. Every other captured source coordinate/byte remains fixed.
		a.raw.Census = before
		a.raw.Registry = baseline.Raw.Registry
		if a.guard() != baseline.Guard {
			return fmt.Errorf("%w: publication source binding drift", ErrChanged)
		}
		return nil
	}
	cursor := 0
	beforeTarget := func() error {
		if p.Repository.observe != nil {
			p.Repository.observe("publication-before-target", changes[cursor].Path)
		}
		if err := guard(); err != nil {
			return err
		}
		entries := map[string]snapshot.Entry{}
		for _, e := range expected.Entries() {
			entries[e.Path().String()] = e
		}
		d := delta[cursor]
		delete(entries, d.Path().String())
		if e, exists := d.After(); exists {
			entries[d.Path().String()] = e
		}
		var all []snapshot.Entry
		for _, e := range entries {
			all = append(all, e)
		}
		expected, err = snapshot.NewTree(all)
		cursor++
		return err
	}
	readback := func(guardedwrite.Result) error {
		if p.Repository.observe != nil {
			p.Repository.observe("publication-readback", "")
		}
		if err := guard(); err != nil {
			return err
		}
		if !expected.Equal(after) {
			return fmt.Errorf("publication did not transfer complete candidate")
		}
		return nil
	}
	result, err := guardedwrite.ApplyOrderedChecked(ctx, prepared.Selection().Repository, capture, changes, guard, beforeTarget, readback)
	out.CompletedPaths = result.CompletedPaths
	out.Readback = err == nil
	return out, err
}
