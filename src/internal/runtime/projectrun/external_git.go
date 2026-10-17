package projectrun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectworkspace"
)

var externalGitCommitID = regexp.MustCompile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
var externalGitSnapshotDigest = regexp.MustCompile("^[0-9a-f]{64}$")

// ImportExternalGitCandidate observes one immutable Git tree relative to the
// exact base snapshot supplied to an external task. The caller binds
// expectedBaseDigest to the packet's candidate digest; candidate changes are
// derived from Git objects rather than accepted from response JSON.
func ImportExternalGitCandidate(host Host, root, baseCommit, finalCommit, expectedBaseDigest string,
	request projectworkspace.Request, handle projectworkspace.Handle, limits projectworkspace.Limits,
) (projectworkspace.Delta, error) {
	if !externalGitCommitID.MatchString(baseCommit) || !externalGitCommitID.MatchString(finalCommit) {
		return projectworkspace.Delta{}, errors.New("external Git candidate requires full lowercase base and final commit IDs")
	}
	if !externalGitSnapshotDigest.MatchString(expectedBaseDigest) {
		return projectworkspace.Delta{}, errors.New("external Git candidate requires the packet's full candidate snapshot digest")
	}
	if err := externalGitRootIsReal(root); err != nil {
		return projectworkspace.Delta{}, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil || filepath.Clean(absRoot) != filepath.Clean(request.RepositoryRoot) {
		return projectworkspace.Delta{}, errors.New("external Git candidate root differs from the workspace request")
	}
	if filepath.Clean(absRoot) != filepath.Clean(handle.CWD) {
		return projectworkspace.Delta{}, errors.New("external Git candidate root differs from the workspace handle")
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("identify external Git candidate: %w", err)
	}
	if request.RepositoryIdentity != identity.Digest {
		return projectworkspace.Delta{}, errors.New("external Git candidate repository identity differs from the workspace request")
	}
	if err := handle.ValidateFor(request); err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("validate external Git candidate workspace binding: %w", err)
	}

	loadLimits := source.DefaultLimits()
	base, err := source.LoadWithLimits(root, baseCommit, loadLimits)
	if err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("load external Git base: %w", err)
	}
	if base.ID != baseCommit {
		return projectworkspace.Delta{}, errors.New("external Git base did not resolve to the supplied immutable commit")
	}
	if host.Load == nil {
		return projectworkspace.Delta{}, errors.New("project acquisition is not bound")
	}
	projectBase, err := host.Load(root, baseCommit)
	if err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("load selected external Git base: %w", err)
	}
	if projectBase == nil || projectBase.Snapshot == nil || projectBase.Snapshot.Digest() != expectedBaseDigest {
		return projectworkspace.Delta{}, errors.New("external Git base project snapshot differs from the packet candidate digest")
	}
	if handle.BaseDigest != "sha256:"+expectedBaseDigest {
		return projectworkspace.Delta{}, errors.New("external Git base snapshot differs from the workspace handle digest")
	}
	final, err := source.LoadWithLimits(root, finalCommit, loadLimits)
	if err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("load external Git candidate: %w", err)
	}
	if final.ID != finalCommit {
		return projectworkspace.Delta{}, errors.New("external Git candidate did not resolve to the supplied immutable commit")
	}

	freshIdentity, err := source.IdentifyGit(root)
	if err != nil || freshIdentity != identity {
		return projectworkspace.Delta{}, errors.New("external Git repository identity changed during import")
	}

	changes := externalGitChanges(base, final)
	delta, err := projectworkspace.NormalizeDelta(request, handle, changes, limits)
	if err != nil {
		return projectworkspace.Delta{}, fmt.Errorf("validate external Git candidate delta: %w", err)
	}
	return delta, nil
}

// externalGitChanges intentionally represents renames as a delete and an add.
// This is a byte-level snapshot comparison, not a heuristic Git rename guess.
func externalGitChanges(base, final *source.Input) []projectworkspace.Change {
	comparison := source.Compare(base, final)
	changes := make([]projectworkspace.Change, 0, len(comparison.Added)+len(comparison.Modified)+len(comparison.Removed))
	for _, path := range comparison.Added {
		changes = append(changes, projectworkspace.Change{Kind: projectworkspace.ChangeAdd, Path: path,
			Mode: final.Modes[path], Content: append([]byte{}, final.Files[path]...)})
	}
	for _, path := range comparison.Modified {
		changes = append(changes, projectworkspace.Change{Kind: projectworkspace.ChangeModify, Path: path,
			Mode: final.Modes[path], Content: append([]byte{}, final.Files[path]...)})
	}
	for _, path := range comparison.Removed {
		changes = append(changes, projectworkspace.Change{Kind: projectworkspace.ChangeDelete, Path: path})
	}
	return changes
}

func externalGitRootIsReal(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("external Git candidate root must be a real directory")
	}
	return nil
}
