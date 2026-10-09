package projectworkspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CandidateOverlayDigest canonicalizes parent-output bytes independently of the
// child write scope. PrepareCandidate additionally verifies inventory semantics.
func CandidateOverlayDigest(changes []Change) (string, error) {
	d, err := canonicalOverlay(changes)
	return d.Digest, err
}

// Public digest calls use a finite ceiling; PrepareCandidate applies the
// service's possibly tighter limits before any content cloning.
func canonicalOverlay(changes []Change) (Delta, error) {
	return canonicalOverlayWithLimits(changes, Limits{MaxFiles: 10000, MaxFileBytes: 64 << 20, MaxTotalBytes: 256 << 20})
}
func canonicalOverlayWithLimits(changes []Change, limits Limits) (Delta, error) {
	if len(changes) > limits.MaxFiles {
		return Delta{}, fmt.Errorf("%w: overlay change count exceeds limit", ErrInvalidDelta)
	}
	scopes := []string{}
	seen := map[string]bool{}
	for _, c := range changes {
		for _, p := range []string{c.Path, c.OldPath} {
			if p != "" && !seen[strings.ToLower(p)] {
				scopes = append(scopes, p)
				seen[strings.ToLower(p)] = true
			}
		}
	}
	// Canonical digest is deliberately independent of a particular repository or
	// task; the private prepared state also binds the full Request and inventory.
	r := Request{RepositoryRoot: filepath.FromSlash("/"), RepositoryIdentity: "candidate-overlay", BaseSHA: strings.Repeat("0", 40), OverlayDigest: "sha256:" + strings.Repeat("0", 64), TaskID: "candidate-overlay", AllowedPaths: scopes}
	if runtime.GOOS == "windows" {
		r.RepositoryRoot = filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	}
	h := Handle{ID: "overlay", CWD: r.RepositoryRoot, RepositoryRoot: r.RepositoryRoot, RepositoryIdentity: r.RepositoryIdentity, BaseSHA: r.BaseSHA, OverlayDigest: r.OverlayDigest, TaskID: r.TaskID, BaseDigest: r.OverlayDigest}
	return NormalizeDelta(r, h, changes, limits)
}
func materializeOverlay(inv inventory, changes []Change) error {
	for _, c := range changes {
		_, exists := inv[c.Path]
		switch c.Kind {
		case ChangeAdd:
			if exists {
				return fmt.Errorf("%w: overlay add already exists: %s", ErrInvalidDelta, c.Path)
			}
		case ChangeModify, ChangeDelete:
			if !exists {
				return fmt.Errorf("%w: overlay source absent: %s", ErrInvalidDelta, c.Path)
			}
		case ChangeRename:
			if _, ok := inv[c.OldPath]; !ok || exists {
				return fmt.Errorf("%w: invalid overlay rename inventory", ErrInvalidDelta)
			}
			delete(inv, c.OldPath)
		}
		if c.Kind == ChangeDelete {
			delete(inv, c.Path)
		} else {
			inv[c.Path] = inventoryFile{c.Mode, bytes.Clone(c.Content)}
		}
	}
	return nil
}
