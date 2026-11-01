// Package repo reads a project's repository through Git: the working tree as
// Git sees it, a base commit, and the files changed between them.
package repo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Toplevel returns the root of the Git working tree that contains dir.
func Toplevel(ctx context.Context, dir string) (string, error) {
	out, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not inside a Git repository: %w", dir, err)
	}
	return filepath.FromSlash(strings.TrimSpace(string(out))), nil
}

// Init makes the existing folder dir a Git repository.
func Init(ctx context.Context, dir string) error {
	_, err := git(ctx, dir, "init", "--quiet")
	return err
}

// cloneLimit bounds a clone, so that a stalled one ends.
const cloneLimit = 30 * time.Minute

// Clone clones url into dir, which must not exist yet. Nobody sees a
// terminal, so neither Git nor SSH may ask on one; the ext transport, which
// runs a command, stays off.
func Clone(ctx context.Context, url, dir string) error {
	if strings.HasPrefix(url, "-") {
		return fmt.Errorf("%q is no repository URL", url)
	}
	if _, err := os.Lstat(dir); err == nil {
		return fmt.Errorf("%s already exists", dir)
	}
	ctx, cancel := context.WithTimeout(ctx, cloneLimit)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-c", "protocol.ext.allow=never", "clone", "--quiet", "--", url, dir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// SSH in batch mode, unless the user chose an SSH command, which wins.
	if os.Getenv("GIT_SSH_COMMAND") == "" && os.Getenv("GIT_SSH") == "" {
		if _, err := git(ctx, filepath.Dir(dir), "config", "--get", "core.sshCommand"); err != nil {
			cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
		}
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// A clone that was stopped leaves half a folder behind.
		os.RemoveAll(dir)
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git clone: %s", msg)
	}
	return nil
}

// WorkingTree is the working tree of a repository, as Git sees it.
type WorkingTree struct {
	Dir string // the repository root
	Ctx context.Context
}

// Root returns the repository root.
func (w WorkingTree) Root() string { return w.Dir }

var _ project.Repository = WorkingTree{}

// Files returns tracked files that still exist and untracked files Git does
// not ignore.
func (w WorkingTree) Files() ([]string, error) {
	out, err := git(w.Ctx, w.Dir, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []string
	for _, f := range split(out) {
		if seen[f] {
			continue
		}
		seen[f] = true
		info, err := os.Lstat(filepath.Join(w.Dir, filepath.FromSlash(f)))
		if err != nil || info.IsDir() {
			continue // deleted in the working tree, or a submodule
		}
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// Read returns a file of the working tree.
func (w WorkingTree) Read(file string) ([]byte, error) {
	return os.ReadFile(filepath.Join(w.Dir, filepath.FromSlash(file)))
}

// GitIgnored reports whether Git ignores the path.
func (w WorkingTree) GitIgnored(file string) bool {
	cmd := exec.CommandContext(w.Ctx, "git", "check-ignore", "-q", "--no-index", "--", file)
	cmd.Dir = w.Dir
	return cmd.Run() == nil
}

// Write writes a file of the working tree, creating its folders.
func (w WorkingTree) Write(file string, data []byte) error {
	full := filepath.Join(w.Dir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

// Exists reports whether a path exists in the working tree.
func (w WorkingTree) Exists(file string) bool {
	_, err := os.Lstat(filepath.Join(w.Dir, filepath.FromSlash(file)))
	return err == nil
}

// Remove deletes a file of the working tree.
func (w WorkingTree) Remove(file string) error {
	return os.Remove(filepath.Join(w.Dir, filepath.FromSlash(file)))
}

// List returns the files below a folder of the working tree, relative to the
// repository root, whether or not Git sees them.
func (w WorkingTree) List(dir string) ([]string, error) {
	var out []string
	root := filepath.Join(w.Dir, filepath.FromSlash(dir))
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(w.Dir, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

// Commit is one commit of a repository.
type Commit struct {
	Root string
	Rev  string // a full commit ID
	Ctx  context.Context
}

var _ project.Repository = Commit{}

// ResolveCommit resolves a revision to a commit of the repository.
func ResolveCommit(ctx context.Context, root, rev string) (Commit, error) {
	out, err := git(ctx, root, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return Commit{}, fmt.Errorf("%q is not a commit of this repository", rev)
	}
	return Commit{Root: root, Rev: strings.TrimSpace(string(out)), Ctx: ctx}, nil
}

// LastLanded returns the newest commit reachable from tip that adds a Run's
// Result page: the base apply and verify start from when none is named
// (internal/run). ok is false when no Run has landed.
func LastLanded(ctx context.Context, root, tip string) (id string, ok bool, err error) {
	out, err := git(ctx, root, "log", "-1", "--format=%H", "--diff-filter=A", "--end-of-options", tip, "--", project.ResultsDir+"/")
	if err != nil {
		return "", false, err
	}
	id = strings.TrimSpace(string(out))
	return id, id != "", nil
}

// Files returns every file of the commit.
func (c Commit) Files() ([]string, error) {
	out, err := git(c.Ctx, c.Root, "ls-tree", "-r", "-z", "--name-only", c.Rev)
	if err != nil {
		return nil, err
	}
	files := split(out)
	sort.Strings(files)
	return files, nil
}

// Read returns a file of the commit.
func (c Commit) Read(file string) ([]byte, error) {
	return git(c.Ctx, c.Root, "cat-file", "blob", c.Rev+":"+file)
}

// ID returns the full commit ID.
func (c Commit) ID() string { return c.Rev }

// GitIgnored is false: a commit holds only what Git tracks.
func (c Commit) GitIgnored(string) bool { return false }

// Changed returns the repository files that differ between the commit and
// the working tree, outside Markitect's own folder.
func (c Commit) Changed() ([]analysis.Change, error) {
	out, err := git(c.Ctx, c.Root, "diff", "--no-renames", "--name-status", "-z", c.Rev, "--")
	if err != nil {
		return nil, err
	}
	fields := split(out)
	changes := map[string]string{}
	for i := 0; i+1 < len(fields); i += 2 {
		status, file := fields[i], fields[i+1]
		switch status[0] {
		case 'A':
			changes[file] = analysis.Added
		case 'D':
			changes[file] = analysis.Removed
		default:
			changes[file] = analysis.Modified
		}
	}
	untracked, err := git(c.Ctx, c.Root, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, f := range split(untracked) {
		changes[f] = analysis.Added
	}
	var list []analysis.Change
	for f, change := range changes {
		if !strings.HasPrefix(f, project.Dir+"/") {
			list = append(list, analysis.Change{Path: f, Change: change})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	return list, nil
}

// Uncommitted lists the files of Markitect's folder that differ between the
// last commit and the working tree, every one of them when nothing is
// committed. It takes no lock, so a commit made at the same moment does not
// fail over it.
func Uncommitted(ctx context.Context, root string) ([]string, error) {
	out, err := git(ctx, root, "--no-optional-locks", "status", "--porcelain", "-z", "--no-renames", "--untracked-files=all", "--", project.Dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range split(out) {
		files = append(files, entry[3:]) // after the two status letters and a space
	}
	return files, nil
}

func split(out []byte) []string {
	var parts []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 {
			parts = append(parts, string(p))
		}
	}
	return parts
}

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.quotepath=off"}, args...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", args[0], msg)
	}
	return out, nil
}
