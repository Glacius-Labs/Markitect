// Package rungit is a Command's own Git work outside its container (concept 3.0
// §6, §9): a private bare repository that reads the target branch, makes the start
// bundle, imports what the sandbox hands out, merges and writes the Result commit,
// and pushes to the target repository. It never touches a working
// tree: the user's repository is only a Git remote to it.
package rungit

import (
	"bufio"
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

// Repo is the host's private repository for one Run.
type Repo struct {
	// Dir is the bare repository.
	Dir string
	// Target is the target repository: a URL or a path Git can push to.
	Target string
}

var _ run.Git = (*Repo)(nil)

// waitDelay bounds how long a git command may hold on after its context ended.
const waitDelay = 5 * time.Second

const (
	targetRefs = "refs/markitect/target/"
	runRefs    = "refs/markitect/run/"
	keepRefs   = "refs/markitect/keep/"
	branchRefs = "refs/markitect/branches/"
)

// Open creates the private repository in dir, which must not exist or be
// empty, for the target repository.
func Open(ctx context.Context, dir, target string) (*Repo, error) {
	r := &Repo{Dir: dir, Target: target}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "git", "init", "-q", "--bare", dir)
	cmd.Env = cleanEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(out)))
	}
	for _, kv := range [][2]string{{"core.autocrlf", "false"}, {"gc.auto", "0"}, {"maintenance.auto", "false"}, {"core.longpaths", "true"}} {
		if _, err := r.git(ctx, nil, nil, "config", kv[0], kv[1]); err != nil {
			return nil, err
		}
	}
	// Merges use Git's own text merge, whatever drivers the repository's
	// attributes or the user's configuration name.
	if err := os.WriteFile(filepath.Join(dir, "info", "attributes"), []byte("* merge=text -filter -text\n"), 0o644); err != nil {
		return nil, err
	}
	return r, nil
}

// Target resolves where a Run starts from and lands: the remote's URL and
// the branch. An empty remote is the current branch's upstream remote, else
// origin, else the repository itself; an empty branch is the upstream's
// branch, else the current branch.
func Target(ctx context.Context, repoDir, remote, branch string) (url, name string, err error) {
	g := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoDir}, args...)...)
		cmd.Env = cleanEnv()
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}
	top, err := g("rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("%s is not inside a Git repository", repoDir)
	}
	current, _ := g("symbolic-ref", "--quiet", "--short", "HEAD")
	if remote == "" && current != "" {
		remote, _ = g("config", "branch."+current+".remote")
		if remote != "" && branch == "" {
			if merge, _ := g("config", "branch."+current+".merge"); merge != "" {
				branch = strings.TrimPrefix(merge, "refs/heads/")
			}
		}
	}
	if remote == "" {
		if _, err := g("remote", "get-url", "origin"); err == nil {
			remote = "origin"
		}
	}
	if branch == "" {
		branch = current
	}
	if branch == "" {
		return "", "", errors.New("HEAD is detached; name the target branch with --branch")
	}
	switch {
	case remote == "" || remote == ".":
		return top, branch, nil
	default:
		if u, err := g("remote", "get-url", remote); err == nil {
			remote = u
		}
		if !strings.Contains(remote, "://") && !strings.Contains(remote, "@") && !filepath.IsAbs(remote) {
			remote = filepath.Join(top, remote)
		}
		return remote, branch, nil
	}
}

// UncommittedSpec lists what the working tree of the repository at repoDir has
// not committed under Markitect's folder.
func UncommittedSpec(ctx context.Context, repoDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "status", "--porcelain", "--untracked-files=all", "--", ":/"+project.Dir)
	cmd.Env = cleanEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s is not inside a Git repository: %w", repoDir, err)
	}
	var dirty []string
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(l) > 3 {
			dirty = append(dirty, l[3:])
		}
	}
	return dirty, nil
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

func (r *Repo) git(ctx context.Context, stdin io.Reader, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--git-dir=" + r.Dir}, args...)...)
	cmd.Env = append(cleanEnv(), env...)
	cmd.Stdin = stdin
	// A deadline ends git even when a helper it started holds its output open.
	cmd.WaitDelay = waitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// A helper may hold the output open after a git that succeeded, as a shared
	// SSH connection does.
	if err := cmd.Run(); err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		return stdout.Bytes(), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (r *Repo) line(ctx context.Context, args ...string) (string, error) {
	out, err := r.git(ctx, nil, nil, args...)
	return strings.TrimSpace(string(out)), err
}

// Fetch reads the tip of a branch of the target repository.
func (r *Repo) Fetch(ctx context.Context, branch string) (string, error) {
	if _, err := r.git(ctx, nil, nil, "fetch", "-q", "--no-tags", r.Target, "+refs/heads/"+branch+":"+targetRefs+branch); err != nil {
		return "", err
	}
	return r.line(ctx, "rev-parse", "--verify", targetRefs+branch+"^{commit}")
}

// Bundle writes a bundle that holds the commit.
func (r *Repo) Bundle(ctx context.Context, commit string, w io.Writer) error {
	ref := keepRefs + commit
	if _, err := r.git(ctx, nil, nil, "update-ref", ref, commit); err != nil {
		return err
	}
	// Streamed: a repository can be larger than the host's memory.
	cmd := exec.CommandContext(ctx, "git", "--git-dir="+r.Dir, "bundle", "create", "-q", "-", ref)
	cmd.Env = cleanEnv()
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = w, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git bundle: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Import reads a bundle the sandbox wrote.
func (r *Repo) Import(ctx context.Context, bundle io.Reader) error {
	f, err := os.CreateTemp(r.Dir, "import-*.bundle")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	_, err = io.Copy(f, bundle)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	_, err = r.git(ctx, nil, nil, "fetch", "-q", "--no-tags", name, "+refs/heads/*:"+runRefs+"*")
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

// Diff is the text of the changes between two commits.
func (r *Repo) Diff(ctx context.Context, from, to string) (string, error) {
	out, err := r.git(ctx, nil, nil, "diff", "--no-color", "--no-ext-diff", from, to, "--")
	return string(out), err
}

// History returns every line any commit between two commits adds, with
// every commit message, and every path those commits touch.
func (r *Repo) History(ctx context.Context, from, to string) (string, []string, error) {
	span := from + ".." + to
	messages, err := r.git(ctx, nil, nil, "log", "--format=%B", span, "--")
	if err != nil {
		return "", nil, err
	}
	// A merge commit shows its changes against its first parent, so what a
	// merge resolution adds is read too.
	patches, err := r.git(ctx, nil, nil, "log", "-p", "--diff-merges=first-parent", "--format=", "--no-renames", "--no-color", "--unified=0", "--text", span, "--")
	if err != nil {
		return "", nil, err
	}
	names, err := r.git(ctx, nil, nil, "log", "--name-only", "--diff-merges=first-parent", "-z", "--format=", "--no-renames", span, "--")
	if err != nil {
		return "", nil, err
	}
	var b strings.Builder
	b.Write(messages)
	sc := bufio.NewScanner(bytes.NewReader(patches))
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	header := false
	for sc.Scan() {
		l := sc.Text()
		switch {
		case header && strings.HasPrefix(l, "+++ "):
			// The file header after "--- ", not an added line.
		case strings.HasPrefix(l, "+"):
			b.WriteString(l[1:])
			b.WriteByte('\n')
		}
		header = strings.HasPrefix(l, "--- ")
	}
	var paths []string
	for _, p := range split(names) {
		paths = append(paths, strings.TrimSpace(p))
	}
	return b.String(), paths, sc.Err()
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

// identity is the author and committer of the host's commits.
var identity = []string{"GIT_AUTHOR_NAME=Markitect", "GIT_AUTHOR_EMAIL=markitect@markitect.invalid", "GIT_COMMITTER_NAME=Markitect", "GIT_COMMITTER_EMAIL=markitect@markitect.invalid"}

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

// Write commits files on top of parent; nil content removes a file.
func (r *Repo) Write(ctx context.Context, parent string, files map[string][]byte, message string) (string, error) {
	index, err := os.CreateTemp(r.Dir, "index-*")
	if err != nil {
		return "", err
	}
	name := index.Name()
	index.Close()
	os.Remove(name)
	defer os.Remove(name)
	env := []string{"GIT_INDEX_FILE=" + name}
	if _, err := r.git(ctx, nil, env, "read-tree", parent); err != nil {
		return "", err
	}
	for _, file := range sortedFiles(files) {
		data := files[file]
		if data == nil {
			if _, err := r.git(ctx, nil, env, "update-index", "--force-remove", "--", file); err != nil {
				return "", err
			}
			continue
		}
		blob, err := r.git(ctx, bytes.NewReader(data), nil, "hash-object", "-w", "--stdin")
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

func sortedFiles(files map[string][]byte) []string {
	var names []string
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	return names
}

// Push sets a branch of the target repository to commit if it is still at
// old; an empty old means the branch must not exist yet, and an empty commit
// deletes the branch.
func (r *Repo) Push(ctx context.Context, branch, commit, old string) error {
	ref := "refs/heads/" + branch
	if out, err := r.checkedOut(ctx, ref); err != nil {
		return err
	} else if out != "" {
		return fmt.Errorf("%s is checked out in the working tree %s, which Markitect never changes; land through a remote you pull from", branch, out)
	}
	spec := commit + ":" + ref
	if commit == "" {
		spec = ":" + ref
	} else if _, err := r.git(ctx, nil, nil, "update-ref", keepRefs+commit, commit); err != nil {
		return err
	}
	_, err := r.git(ctx, nil, nil, "push", "-q", "--no-verify", "--force-with-lease="+ref+":"+old, r.Target, spec)
	return err
}

// checkedOut returns the working tree of a local target that has ref
// checked out, or "" when none does or the target is not a local folder.
// Git's own refusal of such a push depends on receive.denyCurrentBranch, so
// it is checked here.
func (r *Repo) checkedOut(ctx context.Context, ref string) (string, error) {
	if info, err := os.Stat(r.Target); err != nil || !info.IsDir() {
		return "", nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Target, "worktree", "list", "--porcelain")
	cmd.Env = cleanEnv()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list in %s: %w", r.Target, err)
	}
	tree := ""
	for _, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if p, ok := strings.CutPrefix(l, "worktree "); ok {
			tree = p
		}
		if l == "branch "+ref {
			return tree, nil
		}
	}
	return "", nil
}

// LastLanded returns the newest commit reachable from tip that adds a
// Result page under the results folder.
func (r *Repo) LastLanded(ctx context.Context, tip string) (string, bool, error) {
	id, err := r.line(ctx, "log", "-1", "--format=%H", "--diff-filter=A", tip, "--", project.ResultsDir+"/")
	if err != nil {
		return "", false, err
	}
	return id, id != "", nil
}

// Landings returns the commits reachable from tip and not from since that
// add a Result page under the results folder, newest first.
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

// Branches lists the target repository's branches under a prefix with their
// tips, and fetches them so their commits are readable.
func (r *Repo) Branches(ctx context.Context, prefix string) (map[string]string, error) {
	prefix = strings.TrimSuffix(prefix, "/") + "/"
	if _, err := r.git(ctx, nil, nil, "fetch", "-q", "--no-tags", "--prune", r.Target, "+refs/heads/"+prefix+"*:"+branchRefs+prefix+"*"); err != nil {
		return nil, err
	}
	out, err := r.git(ctx, nil, nil, "for-each-ref", "--format=%(objectname) %(refname)", branchRefs+prefix)
	if err != nil {
		return nil, err
	}
	tips := map[string]string{}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if id, ref, ok := strings.Cut(l, " "); ok {
			tips[strings.TrimPrefix(ref, branchRefs)] = id
		}
	}
	return tips, nil
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
