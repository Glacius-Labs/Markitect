package projectworkspace

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type workspaceState struct {
	handle        Handle
	request       Request
	initial       inventory
	storage       string
	sourceDigest  string
	overlayDigest string
}

// GitService owns private full-history repositories, never adopting worktrees.
// Lifecycle operations are serialized; agents may use the returned CWD normally.
type GitService struct {
	mu     sync.Mutex
	root   string
	limits Limits
	states map[string]*workspaceState
}

func NewGitService(storageRoot string, limits Limits) (*GitService, error) {
	if !filepath.IsAbs(storageRoot) || limits.MaxFiles <= 0 || limits.MaxFileBytes <= 0 || limits.MaxTotalBytes <= 0 {
		return nil, fmt.Errorf("%w: absolute storage and positive limits required", ErrInvalidRequest)
	}
	if err := os.MkdirAll(storageRoot, 0700); err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(storageRoot)
	if err != nil {
		return nil, err
	}
	return &GitService{root: root, limits: limits, states: map[string]*workspaceState{}}, nil
}

func (s *GitService) Prepare(ctx context.Context, r Request) (Handle, error) {
	return s.PrepareCandidate(ctx, r, nil, "")
}

// PrepareCandidate materializes an explicitly bound parent candidate as read
// context. It never expands the task's write scopes. Source WIP and candidate
// baseline are independently bound and checked.
func (s *GitService) PrepareCandidate(ctx context.Context, r Request, overlay []Change, digest string) (Handle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := r.Validate(); err != nil {
		return Handle{}, err
	}
	// Storage must not become an input to the adopting checkout inventory.
	sourceRoot, err := filepath.EvalSymlinks(r.RepositoryRoot)
	if err != nil {
		return Handle{}, err
	}
	rel, err := filepath.Rel(sourceRoot, s.root)
	if err != nil {
		return Handle{}, err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return Handle{}, fmt.Errorf("%w: storage lies inside repository", ErrInvalidRequest)
	}
	r.AllowedPaths = append([]string(nil), r.AllowedPaths...)
	r.ExcludedPaths = append([]string(nil), r.ExcludedPaths...)
	binding, err := InspectRepository(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return Handle{}, err
	}
	if binding.OverlayDigest != r.OverlayDigest {
		return Handle{}, fmt.Errorf("%w: stale overlay", ErrInvalidRequest)
	}
	initial, err := sourceInventory(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return Handle{}, err
	}
	if inventoryDigest(initial) != binding.InventoryDigest {
		return Handle{}, fmt.Errorf("%w: source changed during capture", ErrInvalidRequest)
	}
	if len(overlay) > 0 || digest != "" {
		normalized, err := canonicalOverlayWithLimits(overlay, s.limits)
		if err != nil {
			return Handle{}, err
		}
		if normalized.Digest != digest {
			return Handle{}, fmt.Errorf("%w: candidate overlay digest differs", ErrInvalidRequest)
		}
		if err := materializeOverlay(initial, normalized.Changes); err != nil {
			return Handle{}, err
		}
	}
	baselineDigest := inventoryDigest(initial)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return Handle{}, err
	}
	id := hex.EncodeToString(idBytes)
	storage, err := os.MkdirTemp(s.root, "workspace-"+id+"-")
	if err != nil {
		return Handle{}, err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(storage)
		}
	}()
	cwd := filepath.Join(storage, "repo")
	if _, err := git(ctx, s.root, "clone", "--no-hardlinks", "--dissociate", "--no-checkout", "--", r.RepositoryRoot, cwd); err != nil {
		return Handle{}, err
	}
	if _, err := git(ctx, cwd, "config", "core.autocrlf", "false"); err != nil {
		return Handle{}, err
	}
	// Populate the index and detach HEAD without checkout filters or hooks;
	// filesystem bytes come only from the explicitly captured inventory.
	if _, err := git(ctx, cwd, "update-ref", "--no-deref", "HEAD", r.BaseSHA); err != nil {
		return Handle{}, err
	}
	if _, err := git(ctx, cwd, "read-tree", r.BaseSHA); err != nil {
		return Handle{}, err
	}
	if _, err := git(ctx, cwd, "config", "--remove-section", "remote.origin"); err != nil {
		return Handle{}, err
	}
	cloned, err := readInventory(ctx, cwd)
	if err != nil {
		return Handle{}, err
	}
	// This is an owned private clone. Remove only its checked-out entries,
	// preserving Git history; then recreate the exact source/candidate inventory.
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return Handle{}, err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(cwd, entry.Name())); err != nil {
			return Handle{}, err
		}
	}
	for p, f := range initial {
		dest := filepath.Join(cwd, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return Handle{}, err
		}
		perm := fs.FileMode(0644)
		if f.Mode == "100755" {
			perm = 0755
		}
		if err := os.WriteFile(dest, f.Content, perm); err != nil {
			return Handle{}, err
		}
		if err := os.Chmod(dest, perm); err != nil {
			return Handle{}, err
		}
		if runtime.GOOS == "windows" && (modesDiffer(cloned, p, f.Mode) || (f.Mode == "100755" && cloned[p].Mode == "")) {
			if _, err := git(ctx, cwd, "add", "--force", "--", p); err != nil {
				return Handle{}, err
			}
			if _, err := git(ctx, cwd, "update-index", "--chmod="+map[bool]string{true: "+x", false: "-x"}[f.Mode == "100755"], "--", p); err != nil {
				return Handle{}, err
			}
		}
	}
	actual, err := readInventory(ctx, cwd)
	if err != nil {
		return Handle{}, err
	}
	if inventoryDigest(actual) != baselineDigest {
		return Handle{}, fmt.Errorf("%w: copied inventory differs", ErrInvalidRequest)
	}
	fresh, err := InspectRepository(ctx, r.RepositoryRoot, r.BaseSHA)
	if err != nil {
		return Handle{}, err
	}
	if fresh != binding {
		return Handle{}, fmt.Errorf("%w: source changed during prepare", ErrInvalidRequest)
	}
	h := Handle{ID: id, CWD: cwd, RepositoryRoot: r.RepositoryRoot, RepositoryIdentity: r.RepositoryIdentity, BaseSHA: r.BaseSHA, OverlayDigest: r.OverlayDigest, TaskID: r.TaskID, BaseDigest: baselineDigest}
	prepared := &workspaceState{handle: h, request: r, initial: initial, storage: storage, sourceDigest: binding.InventoryDigest, overlayDigest: digest}
	if err := writeOwnershipRecord(prepared); err != nil {
		return Handle{}, err
	}
	s.states[id] = prepared
	ok = true
	return h, nil
}
func modesDiffer(inv inventory, p, mode string) bool { f, ok := inv[p]; return ok && f.Mode != mode }
func (s *GitService) state(h Handle) (*workspaceState, error) {
	st, ok := s.states[h.ID]
	if !ok || st.handle != h {
		return nil, ErrInvalidHandle
	}
	resolved, err := filepath.EvalSymlinks(h.CWD)
	if err != nil || !samePath(resolved, h.CWD) {
		return nil, ErrInvalidHandle
	}
	return st, nil
}
func (s *GitService) Harvest(ctx context.Context, h Handle) (Delta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.state(h)
	if err != nil {
		return Delta{}, err
	}
	binding, err := InspectRepository(ctx, h.RepositoryRoot, h.BaseSHA)
	if err != nil {
		return Delta{}, err
	}
	if binding.OverlayDigest != h.OverlayDigest || binding.InventoryDigest != st.sourceDigest {
		return Delta{}, fmt.Errorf("%w: adopting checkout changed", ErrInvalidDelta)
	}
	if err := validateCandidateGit(ctx, h); err != nil {
		return Delta{}, err
	}
	if _, err := git(ctx, h.CWD, "merge-base", "--is-ancestor", h.BaseSHA, "HEAD"); err != nil {
		return Delta{}, fmt.Errorf("%w: candidate lost base history", ErrInvalidDelta)
	}
	final, err := readInventoryBounded(ctx, h.CWD, st.initial, s.limits)
	if err != nil {
		return Delta{}, err
	}
	changes := []Change{}
	deleted := []string{}
	added := []string{}
	for p, f := range st.initial {
		next, ok := final[p]
		if !ok {
			deleted = append(deleted, p)
		} else if next.Mode != f.Mode || !bytes.Equal(next.Content, f.Content) {
			changes = append(changes, Change{Kind: ChangeModify, Path: p, Mode: next.Mode, Content: next.Content})
		}
	}
	for p := range final {
		if _, ok := st.initial[p]; !ok {
			added = append(added, p)
		}
	}
	// Conservatively count operations before rename scratch work; pairing can
	// reduce this count but never permits unlimited inventory work.
	if len(changes)+len(added) > s.limits.MaxFiles || len(deleted) > s.limits.MaxFiles {
		return Delta{}, fmt.Errorf("%w: observed change count exceeds limit", ErrInvalidDelta)
	}
	sort.Strings(added)
	sort.Strings(deleted)
	renames, err := detectRenames(ctx, st.storage, st.initial, final, deleted, added)
	if err != nil {
		return Delta{}, err
	}
	used := map[string]bool{}
	for _, old := range deleted {
		match := renames[old]
		if match != "" {
			used[match] = true
			f := final[match]
			changes = append(changes, Change{Kind: ChangeRename, OldPath: old, Path: match, Mode: f.Mode, Content: f.Content})
		} else {
			changes = append(changes, Change{Kind: ChangeDelete, Path: old})
		}
	}
	for _, p := range added {
		if !used[p] {
			f := final[p]
			changes = append(changes, Change{Kind: ChangeAdd, Path: p, Mode: f.Mode, Content: f.Content})
		}
	}
	delta, err := NormalizeDelta(st.request, h, changes, s.limits)
	if err != nil {
		return Delta{}, err
	}
	// Callers must stop task writers before harvesting. Re-observe both sides to
	// reject detectable races rather than asserting a mixed capture is stable.
	observed, err := readInventoryBounded(ctx, h.CWD, st.initial, s.limits)
	if err != nil {
		return Delta{}, err
	}
	if inventoryDigest(observed) != inventoryDigest(final) {
		return Delta{}, fmt.Errorf("%w: candidate changed during harvest", ErrInvalidDelta)
	}
	fresh, err := InspectRepository(ctx, h.RepositoryRoot, h.BaseSHA)
	if err != nil {
		return Delta{}, err
	}
	if fresh != binding {
		return Delta{}, fmt.Errorf("%w: source changed during harvest", ErrInvalidDelta)
	}
	return delta, nil
}
func (s *GitService) Close(ctx context.Context, h Handle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	st, err := s.state(h)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(st.storage)
	if err != nil || !samePath(parent, st.storage) || !samePath(filepath.Dir(parent), s.root) {
		return ErrInvalidHandle
	}
	if err := os.RemoveAll(st.storage); err != nil {
		return err
	}
	delete(s.states, h.ID)
	return nil
}
