package projectworkspace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Binding describes an observed source checkout, including uncommitted and ignored
// regular files. It is input evidence, not authorization or a product proof.
type Binding struct {
	OverlayDigest   string
	InventoryDigest string
}
type inventoryFile struct {
	Mode    string
	Content []byte
}
type inventory map[string]inventoryFile

func git(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := gitCommand(ctx, root, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, out)
	}
	return out, nil
}
func gitCommand(ctx context.Context, root string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
	// A caller's ambient Git redirection must not redirect source observation
	// or candidate commands to an unrelated index/repository.
	for _, v := range os.Environ() {
		name, _, _ := strings.Cut(v, "=")
		if strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	return cmd
}
func sourceInventory(ctx context.Context, root, base string) (inventory, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	top, err := git(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	if !sameGitReportedPath(resolved, strings.TrimSpace(string(top))) {
		return nil, fmt.Errorf("%w: repository root must be top level", ErrInvalidRequest)
	}
	head, err := git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(head)) != base {
		return nil, fmt.Errorf("%w: source HEAD differs from selected base", ErrInvalidRequest)
	}
	shallow, err := git(ctx, root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(shallow)) == "true" {
		return nil, fmt.Errorf("%w: selected source history is shallow", ErrInvalidRequest)
	}
	return readInventoryCapture(ctx, root, nil, Limits{}, true)
}
func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// withinRepository reports whether path is the resolved repository root or lies
// below it. On one volume the lexical relation decides. filepath.Rel cannot
// relate two Windows volumes, and a subst drive or junction can spell the same
// directory on another one, so there an ancestor of path that is the root by
// file identity decides. projectadoption keeps the same helper.
func withinRepository(root, path string) (bool, error) {
	if strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(path)) {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return false, err
		}
		return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return false, err
	}
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if err == nil && os.SameFile(rootInfo, info) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if filepath.Dir(current) == current {
			return false, nil
		}
	}
}

// InspectRepository must be called on the explicit selected source checkout.
// Prepare reobserves these bytes and rejects a stale overlay binding.
func InspectRepository(ctx context.Context, root, base string) (Binding, error) {
	if !filepath.IsAbs(root) || !sha1OrSHA256.MatchString(base) {
		return Binding{}, ErrInvalidRequest
	}
	inv, err := sourceInventory(ctx, root, base)
	if err != nil {
		return Binding{}, err
	}
	digest := inventoryDigest(inv)
	encoded, _ := json.Marshal(struct {
		Base      string
		Inventory string
	}{base, digest})
	sum := sha256.Sum256(encoded)
	return Binding{OverlayDigest: "sha256:" + hex.EncodeToString(sum[:]), InventoryDigest: digest}, nil
}
func inventoryDigest(inv inventory) string {
	paths := make([]string, 0, len(inv))
	for p := range inv {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	type entry struct {
		Path    string
		Mode    string
		Content []byte
	}
	entries := make([]entry, 0, len(paths))
	for _, p := range paths {
		f := inv[p]
		entries = append(entries, entry{p, f.Mode, f.Content})
	}
	encoded, _ := json.Marshal(entries)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Full source context has independent finite ceilings rather than being reduced
// to the task's write scope. Oversize inventories fail explicitly.
const MaxInventoryFiles = 100000
const MaxInventoryFileBytes = 256 << 20
const MaxInventoryBytes = 1 << 30

func readInventory(ctx context.Context, root string) (inventory, error) {
	return readInventoryBounded(ctx, root, nil, Limits{})
}
func readInventoryBounded(ctx context.Context, root string, baseline inventory, limits Limits) (inventory, error) {
	return readInventoryCapture(ctx, root, baseline, limits, false)
}
func readInventoryCapture(ctx context.Context, root string, baseline inventory, limits Limits, source bool) (inventory, error) {
	index, err := git(ctx, root, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	modes := map[string]string{}
	for _, entry := range bytes.Split(index, []byte{0}) {
		if len(entry) == 0 {
			continue
		}
		parts := bytes.SplitN(entry, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, ErrInvalidDelta
		}
		fields := strings.Fields(string(parts[0]))
		if len(fields) != 3 || fields[2] != "0" {
			return nil, fmt.Errorf("%w: unresolved index", ErrInvalidDelta)
		}
		if fields[0] == "160000" || fields[0] == "120000" {
			return nil, fmt.Errorf("%w: unsupported link/submodule %s", ErrInvalidDelta, parts[1])
		}
		modes[string(parts[1])] = fields[0]
	}
	confined, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer confined.Close()
	inv := inventory{}
	aliases := map[string]string{}
	totalBytes, changedFiles, changedBytes := int64(0), 0, 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Host operational records are not task inputs. Only source capture skips
		// this exact tool subtree; candidate writes here remain observable/forbidden.
		if source && (strings.EqualFold(rel, ".markitect/runs") || strings.HasPrefix(strings.ToLower(rel), ".markitect/runs/")) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: link in inventory: %s", ErrInvalidDelta, rel)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: nonregular inventory: %s", ErrInvalidDelta, rel)
		}
		// Control files remain observable but are never accepted as writes.
		safe := rel
		if strings.HasPrefix(strings.ToLower(rel), ".markitect/") {
			safe = "control/" + rel[len(".markitect/"):]
		}
		if _, err := portablePath(safe); err != nil {
			return fmt.Errorf("%w: unsafe inventory path %s", ErrInvalidDelta, rel)
		}
		key := strings.ToLower(rel)
		if prior, ok := aliases[key]; ok {
			return fmt.Errorf("%w: alias %s and %s", ErrInvalidDelta, prior, rel)
		}
		aliases[key] = rel
		if len(inv) >= MaxInventoryFiles || info.Size() > MaxInventoryFileBytes || info.Size() > MaxInventoryBytes-totalBytes {
			return fmt.Errorf("%w: inventory size exceeds finite ceiling", ErrInvalidDelta)
		}
		if baseline != nil {
			prior, ok := baseline[rel]
			if (!ok || info.Size() != int64(len(prior.Content))) && info.Size() > int64(limits.MaxFileBytes) {
				return fmt.Errorf("%w: changed file exceeds limit", ErrInvalidDelta)
			}
		}
		mode := "100644"
		if runtime.GOOS == "windows" {
			if modes[rel] == "100755" {
				mode = "100755"
			}
		} else if info.Mode().Perm()&0111 != 0 {
			mode = "100755"
		}
		if modes[rel] == "120000" || modes[rel] == "160000" {
			return fmt.Errorf("%w: link or submodule %s", ErrInvalidDelta, rel)
		}
		file, err := confined.Open(filepath.FromSlash(rel))
		if err != nil {
			return err
		}
		opened, err := file.Stat()
		if err != nil {
			file.Close()
			return err
		}
		if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			file.Close()
			return fmt.Errorf("%w: inventory changed while reading %s", ErrInvalidDelta, rel)
		}
		content, err := io.ReadAll(io.LimitReader(file, info.Size()+1))
		after, statErr := file.Stat()
		file.Close()
		if statErr != nil {
			return statErr
		}
		current, statErr := confined.Lstat(filepath.FromSlash(rel))
		if statErr != nil {
			return statErr
		}
		if !os.SameFile(opened, current) || !os.SameFile(opened, after) || after.Size() != info.Size() || int64(len(content)) != info.Size() || !after.ModTime().Equal(info.ModTime()) || after.Mode() != info.Mode() {
			return fmt.Errorf("%w: file changed during inventory capture: %s", ErrInvalidDelta, rel)
		}
		if err != nil {
			return err
		}
		totalBytes += int64(len(content))
		if baseline != nil {
			prior, ok := baseline[rel]
			if !ok || prior.Mode != mode || !bytes.Equal(prior.Content, content) {
				changedFiles++
				if changedFiles > limits.MaxFiles || len(content) > limits.MaxFileBytes || len(content) > limits.MaxTotalBytes-changedBytes {
					return fmt.Errorf("%w: observed changes exceed limits", ErrInvalidDelta)
				}
				changedBytes += len(content)
			}
		}
		inv[rel] = inventoryFile{mode, content}
		return nil
	})
	return inv, err
}
