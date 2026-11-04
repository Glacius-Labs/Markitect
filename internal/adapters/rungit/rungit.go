// Package rungit is a Run's Git work in the user's own repository (draft concept
// 4.0): it reads commits from anywhere in the repository, makes commits of a
// worktree's files without running the project's hooks, and brings the owner's
// branch up. It works in the repository's common Git directory, which every
// worktree shares.
package rungit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// Repo is the repository of one command.
type Repo struct {
	// Dir is the common Git directory.
	Dir string
}

var _ run.Git = (*Repo)(nil)

// waitDelay bounds how long a git command may hold on after its context ended.
const waitDelay = 5 * time.Second

// Open finds the repository that holds dir.
func Open(ctx context.Context, dir string) (*Repo, error) {
	out, err := command(ctx, dir, nil, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("%s is not inside a Git repository", dir)
	}
	return &Repo{Dir: strings.TrimSpace(string(out))}, nil
}

// cleanEnv is the environment without Git variables that would point git at
// another repository or index.
func cleanEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(name) {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR", "GIT_NAMESPACE", "GIT_PREFIX":
			continue
		}
		env = append(env, kv)
	}
	return append(env, "GIT_TERMINAL_PROMPT=0")
}

// command runs git in a folder.
func command(ctx context.Context, dir string, stdin io.Reader, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cleanEnv(), env...)
	cmd.Stdin = stdin
	// A deadline ends git even when a helper it started holds its output open.
	cmd.WaitDelay = waitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		return stdout.Bytes(), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (r *Repo) git(ctx context.Context, stdin io.Reader, env []string, args ...string) ([]byte, error) {
	return command(ctx, r.Dir, stdin, env, append([]string{"--git-dir=" + r.Dir}, args...)...)
}

func (r *Repo) line(ctx context.Context, args ...string) (string, error) {
	out, err := r.git(ctx, nil, nil, args...)
	return strings.TrimSpace(string(out)), err
}

func lineAt(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := command(ctx, dir, nil, nil, args...)
	return strings.TrimSpace(string(out)), err
}

// Root is the top of the working tree that holds dir.
func (r *Repo) Root(ctx context.Context, dir string) (string, error) {
	return lineAt(ctx, dir, "rev-parse", "--show-toplevel")
}

// Branch is the branch checked out at dir, "" when HEAD is detached.
func (r *Repo) Branch(ctx context.Context, dir string) (string, error) {
	out, err := lineAt(ctx, dir, "branch", "--show-current")
	return out, err
}

// Head is the commit checked out at dir.
func (r *Repo) Head(ctx context.Context, dir string) (string, error) {
	return lineAt(ctx, dir, "rev-parse", "--verify", "HEAD^{commit}")
}

// providerFiles are the provider's local files, which are never part of the work:
// a worktree of a Run would otherwise become a link in the tree of its own commit
// in a repository that does not ignore them.
var providerFiles = []string{":(exclude,top).claude/worktrees", ":(exclude,top).claude/settings.local.json"}

// notIgnored is the exclude patterns of the provider's files Git does not ignore
// anyway: git add refuses to be told to leave out a path it ignores.
func notIgnored(ctx context.Context, dir string) []string {
	var out []string
	for path, pattern := range map[string]string{".claude/worktrees/x": providerFiles[0], ".claude/settings.local.json": providerFiles[1]} {
		if _, err := command(ctx, dir, nil, nil, "check-ignore", "-q", "--", path); err != nil {
			out = append(out, pattern)
		}
	}
	return out
}

// Uncommitted lists what differs from the last commit at dir, outside
// Markitect's folder and the provider's local files.
func (r *Repo) Uncommitted(ctx context.Context, dir string) ([]string, error) {
	args := append([]string{"--no-optional-locks", "status", "--porcelain", "-z", "--no-renames", "--untracked-files=all", "--", ":/", ":(exclude,top).markitect"}, providerFiles...)
	out, err := command(ctx, dir, nil, nil, args...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range split(out) {
		if len(entry) > 3 {
			files = append(files, entry[3:])
		}
	}
	return files, nil
}

// CommitSpec commits Markitect's folder at dir with the user's identity. The
// project's hooks run, as they would for the user's own commit.
func (r *Repo) CommitSpec(ctx context.Context, dir, message string) (string, error) {
	out, err := command(ctx, dir, nil, nil, "--no-optional-locks", "status", "--porcelain", "--", ":/"+project.Dir)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(out)) != "" {
		if _, err := command(ctx, dir, nil, nil, "add", "-A", "--", ":/"+project.Dir); err != nil {
			return "", err
		}
		if _, err := command(ctx, dir, nil, nil, "commit", "-q", "-m", message, "--", ":/"+project.Dir); err != nil {
			return "", err
		}
	}
	return r.Head(ctx, dir)
}

// identity is the author and committer of the tool's commits.
var identity = []string{"GIT_AUTHOR_NAME=Markitect", "GIT_AUTHOR_EMAIL=markitect@markitect.invalid", "GIT_COMMITTER_NAME=Markitect", "GIT_COMMITTER_EMAIL=markitect@markitect.invalid"}

// Candidate makes a commit of every file Git would take in the worktree at dir,
// through a temporary index, so that the worktree, its index and HEAD stay as they
// are. With nothing changed it returns HEAD.
func (r *Repo) Candidate(ctx context.Context, dir, message string) (string, error) {
	head, err := r.Head(ctx, dir)
	if err != nil {
		return "", err
	}
	index, cleanup, err := r.tempIndex()
	if err != nil {
		return "", err
	}
	defer cleanup()
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err := command(ctx, dir, nil, env, "read-tree", head); err != nil {
		return "", err
	}
	if _, err := command(ctx, dir, nil, env, append([]string{"add", "-A", "--", ":/"}, notIgnored(ctx, dir)...)...); err != nil {
		return "", err
	}
	tree, err := command(ctx, dir, nil, env, "write-tree")
	if err != nil {
		return "", err
	}
	want := strings.TrimSpace(string(tree))
	c, err := r.Commit(ctx, head)
	if err != nil {
		return "", err
	}
	if c.Tree == want {
		return head, nil
	}
	id, err := r.git(ctx, strings.NewReader(message+"\n"), identity, "commit-tree", want, "-p", head)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(id)), nil
}

func (r *Repo) tempIndex() (string, func(), error) {
	f, err := os.CreateTemp(r.Dir, "markitect-index-*")
	if err != nil {
		return "", nil, err
	}
	name := f.Name()
	f.Close()
	// Git makes the index itself; an empty file is no index to it.
	os.Remove(name)
	return name, func() { os.Remove(name); os.Remove(name + ".lock") }, nil
}

// Adopt moves the worktree at dir to a commit made from its own files.
func (r *Repo) Adopt(ctx context.Context, dir, commit string) error {
	head, err := r.Head(ctx, dir)
	if err != nil || head == commit {
		return err
	}
	_, err = command(ctx, dir, nil, nil, "reset", "-q", "--mixed", commit)
	return err
}

// Commit returns one commit.
func (r *Repo) Commit(ctx context.Context, id string) (run.Commit, error) {
	out, err := r.git(ctx, nil, nil, "log", "-1", "--format=%H%x00%P%x00%T%x00%B", id+"^{commit}", "--")
	if err != nil {
		return run.Commit{}, err
	}
	f := strings.SplitN(string(out), "\x00", 4)
	if len(f) != 4 {
		return run.Commit{}, fmt.Errorf("commit %s: unreadable", id)
	}
	return run.Commit{ID: f[0], Parents: strings.Fields(f[1]), Tree: f[2], Message: strings.TrimRight(f[3], "\n") + "\n"}, nil
}

// Contains reports whether commit is tip or one of its ancestors.
func (r *Repo) Contains(ctx context.Context, tip, commit string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "--git-dir="+r.Dir, "merge-base", "--is-ancestor", commit, tip)
	cmd.Env = cleanEnv()
	cmd.WaitDelay = waitDelay
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return false, nil
	default:
		return false, fmt.Errorf("git merge-base: %w", err)
	}
}

// Changed lists the files that differ between two commits.
func (r *Repo) Changed(ctx context.Context, from, to string) ([]analysis.Change, error) {
	out, err := r.git(ctx, nil, nil, "diff", "--no-renames", "--name-status", "-z", from, to, "--")
	if err != nil {
		return nil, err
	}
	fields := split(out)
	var list []analysis.Change
	for i := 0; i+1 < len(fields); i += 2 {
		change := analysis.Modified
		switch fields[i][0] {
		case 'A':
			change = analysis.Added
		case 'D':
			change = analysis.Removed
		}
		list = append(list, analysis.Change{Path: fields[i+1], Change: change})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	return list, nil
}

// Links returns the symbolic links of a commit, with where they point.
func (r *Repo) Links(ctx context.Context, commit string) (map[string]string, error) {
	out, err := r.git(ctx, nil, nil, "ls-tree", "-r", "-z", commit)
	if err != nil {
		return nil, err
	}
	links := map[string]string{}
	for _, row := range split(out) {
		meta, file, ok := strings.Cut(row, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 3 || f[0] != "120000" {
			continue
		}
		target, err := r.git(ctx, nil, nil, "cat-file", "blob", f[2])
		if err != nil {
			return nil, err
		}
		links[file] = string(target)
	}
	return links, nil
}

// State is one commit as a project repository state.
func (r *Repo) State(ctx context.Context, commit string) project.Repository {
	return state{r: r, ctx: ctx, commit: commit}
}

type state struct {
	r      *Repo
	ctx    context.Context
	commit string
}

func (s state) Files() ([]string, error) {
	out, err := s.r.git(s.ctx, nil, nil, "ls-tree", "-r", "-z", "--name-only", s.commit)
	if err != nil {
		return nil, err
	}
	files := split(out)
	sort.Strings(files)
	return files, nil
}

func (s state) Read(file string) ([]byte, error) {
	return s.r.git(s.ctx, nil, nil, "cat-file", "blob", s.commit+":"+file)
}

func (s state) GitIgnored(string) bool { return false }

// Export writes the files of a commit into an empty folder, as a checkout would.
func (r *Repo) Export(ctx context.Context, commit, dir string) error {
	index, cleanup, err := r.tempIndex()
	if err != nil {
		return err
	}
	defer cleanup()
	env := []string{"GIT_INDEX_FILE=" + index, "GIT_WORK_TREE=" + dir}
	if _, err := r.git(ctx, nil, env, "read-tree", commit); err != nil {
		return err
	}
	_, err = r.git(ctx, nil, env, "checkout-index", "-a", "-f")
	return err
}

// Merge merges theirs into ours as a new merge commit.
func (r *Repo) Merge(ctx context.Context, ours, theirs, message string) (string, []string, error) {
	out, err := r.git(ctx, nil, nil, "merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", ours, theirs)
	fields := split(out)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(errors.Unwrap(err), &exit) && exit.ExitCode() == 1 && len(fields) > 1 {
			return "", fields[1:], nil
		}
		return "", nil, err
	}
	if len(fields) == 0 {
		return "", nil, errors.New("git merge-tree wrote no tree")
	}
	id, err := r.git(ctx, strings.NewReader(message), identity, "commit-tree", fields[0], "-p", ours, "-p", theirs, "-F", "-")
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(string(id)), nil, nil
}

// Write commits files on top of parent.
func (r *Repo) Write(ctx context.Context, parent string, files map[string][]byte, message string) (string, error) {
	index, cleanup, err := r.tempIndex()
	if err != nil {
		return "", err
	}
	defer cleanup()
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err := r.git(ctx, nil, env, "read-tree", parent); err != nil {
		return "", err
	}
	var names []string
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, file := range names {
		blob, err := r.git(ctx, bytes.NewReader(files[file]), nil, "hash-object", "-w", "--stdin")
		if err != nil {
			return "", err
		}
		if _, err := r.git(ctx, nil, env, "update-index", "--add", "--cacheinfo", "100644,"+strings.TrimSpace(string(blob))+","+file); err != nil {
			return "", err
		}
	}
	tree, err := r.git(ctx, nil, env, "write-tree")
	if err != nil {
		return "", err
	}
	id, err := r.git(ctx, strings.NewReader(message), identity, "commit-tree", strings.TrimSpace(string(tree)), "-p", parent, "-F", "-")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(id)), nil
}

// Tip reads the tip of a branch, "" when it does not exist.
func (r *Repo) Tip(ctx context.Context, branch string) (string, error) {
	out, err := r.line(ctx, "for-each-ref", "--format=%(objectname)", "refs/heads/"+branch)
	if err != nil {
		return "", err
	}
	return out, nil
}

// Advance moves a branch from old to commit, only if it is still at old.
func (r *Repo) Advance(ctx context.Context, branch, commit, old string) error {
	tree, err := r.checkedOutAt(ctx, branch)
	if err != nil {
		return err
	}
	if tree == "" {
		_, err := r.git(ctx, nil, nil, "update-ref", "refs/heads/"+branch, commit, old)
		return err
	}
	head, err := r.Head(ctx, tree)
	if err != nil {
		return err
	}
	if head != old {
		return fmt.Errorf("%s moved while landing", branch)
	}
	if _, err := command(ctx, tree, nil, nil, "merge", "-q", "--ff-only", commit); err != nil {
		return fmt.Errorf("%s could not be brought up in %s, probably because of uncommitted edits there: %w", branch, tree, err)
	}
	return nil
}

// SetBranch points a branch at a commit.
func (r *Repo) SetBranch(ctx context.Context, branch, commit string) error {
	_, err := r.git(ctx, nil, nil, "update-ref", "refs/heads/"+branch, commit)
	return err
}

// DeleteBranch removes a branch, if it exists.
func (r *Repo) DeleteBranch(ctx context.Context, branch string) error {
	tip, err := r.Tip(ctx, branch)
	if err != nil || tip == "" {
		return err
	}
	_, err = r.git(ctx, nil, nil, "update-ref", "-d", "refs/heads/"+branch)
	return err
}

// Branches lists the branches under a prefix with their tips.
func (r *Repo) Branches(ctx context.Context, prefix string) (map[string]string, error) {
	prefix = strings.TrimSuffix(prefix, "/") + "/"
	out, err := r.git(ctx, nil, nil, "for-each-ref", "--format=%(objectname) %(refname)", "refs/heads/"+prefix)
	if err != nil {
		return nil, err
	}
	tips := map[string]string{}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if id, ref, ok := strings.Cut(l, " "); ok {
			tips[strings.TrimPrefix(ref, "refs/heads/")] = id
		}
	}
	return tips, nil
}

// RemoveWorktree removes the worktree that has a branch checked out, if there is
// one. The provider holds a lock on it while its session runs, and a session that
// has ended leaves that lock until its next sweep.
func (r *Repo) RemoveWorktree(ctx context.Context, branch string) error {
	tree, err := r.checkedOutAt(ctx, branch)
	if err != nil || tree == "" {
		return err
	}
	_, _ = r.git(ctx, nil, nil, "worktree", "unlock", tree)
	if _, err := r.git(ctx, nil, nil, "worktree", "remove", "--force", tree); err != nil {
		return err
	}
	return nil
}

// WorktreeOf returns the working tree that has a branch checked out, or "".
func (r *Repo) WorktreeOf(ctx context.Context, branch string) (string, error) {
	return r.checkedOutAt(ctx, branch)
}

// checkedOutAt returns the working tree that has a branch checked out, or "".
func (r *Repo) checkedOutAt(ctx context.Context, branch string) (string, error) {
	out, err := r.git(ctx, nil, nil, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	tree := ""
	for _, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if p, ok := strings.CutPrefix(l, "worktree "); ok {
			tree = filepath.FromSlash(p)
		}
		if l == "branch refs/heads/"+branch {
			return tree, nil
		}
	}
	return "", nil
}

// LastLanded returns the newest commit reachable from tip that adds a Result page
// under the results folder.
func (r *Repo) LastLanded(ctx context.Context, tip string) (string, bool, error) {
	id, err := r.line(ctx, "log", "-1", "--format=%H", "--diff-filter=A", tip, "--", project.ResultsDir+"/")
	if err != nil {
		return "", false, err
	}
	return id, id != "", nil
}

// Landings returns the commits reachable from tip and not from since that add a
// Result page under the results folder, newest first.
func (r *Repo) Landings(ctx context.Context, tip, since string) ([]string, error) {
	args := []string{"log", "--format=%H", "--diff-filter=A", tip}
	if since != "" {
		args = append(args, "^"+since)
	}
	out, err := r.git(ctx, nil, nil, append(args, "--", project.ResultsDir+"/")...)
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
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
