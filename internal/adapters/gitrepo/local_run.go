package gitrepo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// LocalGit implements Run's Git responsibility with real local repositories.
// It is useful for controlled execution; it claims no container isolation.
type LocalGit struct{ StateRoot, TempRoot string }
type localService struct {
	adapter    *LocalGit
	request    run.Request
	root, seed string
	mu         sync.Mutex
	copies     []*localCopy
}
type localCopy struct {
	service  *localService
	location string
}

func token() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	return hex.EncodeToString(b[:]), err
}
func safeKey(s string) bool {
	if s == "" || s == "." || s == ".." || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.", r) {
			return false
		}
	}
	return true
}
func localCommand(ctx context.Context, dir string, input []byte, args ...string) (string, error) {
	prefix := []string{"--no-replace-objects", "--literal-pathspecs", "-c", "core.hooksPath=", "-c", "core.fsmonitor=false", "-c", "core.autocrlf=false", "-c", "commit.gpgsign=false", "-c", "user.name=Markitect Run", "-c", "user.email=run@markitect.invalid", "-C", dir}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	cmd.Env = append(gitEnvironment(), "GIT_ALLOW_PROTOCOL=file", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	cmd.WaitDelay = time.Second
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	out, stderr := &boundedOutput{limit: 64 << 20}, &boundedOutput{limit: 16 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if err := boundedCommandError(cmd.Run(), out, stderr); err != nil {
		return "", fmt.Errorf("local Git %s: %w: %s", args[0], err, stderr.String())
	}
	if args[0] == "ls-tree" || args[0] == "diff" {
		return out.String(), nil
	}
	return strings.TrimSpace(out.String()), nil
}
func object(ctx context.Context, dir, rev string) (run.Commit, error) {
	id, err := localCommand(ctx, dir, nil, "rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return run.Commit{}, err
	}
	tree, err := localCommand(ctx, dir, nil, "rev-parse", "--verify", id+"^{tree}")
	return run.Commit{ID: id, Tree: tree}, err
}
func (g *LocalGit) candidatePath(ref project.CandidateRef) (string, error) {
	if g == nil || !filepath.IsAbs(g.StateRoot) || !safeKey(ref.Project) || !safeKey(ref.Key) {
		return "", fmt.Errorf("invalid candidate state identity")
	}
	return filepath.Join(g.StateRoot, ref.Project, "candidates", ref.Key), nil
}

// Capture retains the source commit before any Run resource exists.
func (g *LocalGit) Capture(ctx context.Context, id string, p project.Prepared) (project.CandidateRef, error) {
	if !p.Valid() {
		return project.CandidateRef{}, fmt.Errorf("unchecked project")
	}
	c, err := object(ctx, p.Selection().Repository, p.Attempt().Model().Inputs().Head)
	if err != nil {
		return project.CandidateRef{}, err
	}
	raw := p.Attempt().Model().Inputs().Raw
	for _, m := range append([]project.ModelSource{raw.Local}, raw.Dependencies...) {
		if m.RepositoryArea.String() == "" {
			return project.CandidateRef{}, fmt.Errorf("local Run source must be repository-bound")
		}
	}
	seed, err := os.MkdirTemp(g.TempRoot, "markitect-input-")
	if err != nil {
		return project.CandidateRef{}, err
	}
	defer os.RemoveAll(seed)
	format, err := localCommand(ctx, p.Selection().Repository, nil, "rev-parse", "--show-object-format")
	if err != nil {
		return project.CandidateRef{}, err
	}
	if _, err = localCommand(ctx, seed, nil, "init", "--quiet", "--object-format="+format); err != nil {
		return project.CandidateRef{}, err
	}
	if _, err = localCommand(ctx, seed, nil, "fetch", "--quiet", "--no-tags", p.Selection().Repository, c.ID); err != nil {
		return project.CandidateRef{}, err
	}
	if _, err = localCommand(ctx, seed, nil, "checkout", "--quiet", "--detach", c.ID); err != nil {
		return project.CandidateRef{}, err
	}
	entries, err := os.ReadDir(seed)
	if err != nil {
		return project.CandidateRef{}, err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			if err = os.RemoveAll(filepath.Join(seed, e.Name())); err != nil {
				return project.CandidateRef{}, err
			}
		}
	}
	for _, f := range raw.Census.Entries() {
		path := filepath.Join(seed, filepath.FromSlash(f.Path().String()))
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return project.CandidateRef{}, err
		}
		mode := os.FileMode(0600)
		if f.Mode() == snapshot.Executable {
			mode = 0700
		}
		if err = os.WriteFile(path, f.Bytes(), mode); err != nil {
			return project.CandidateRef{}, err
		}
	}
	for _, m := range append([]project.ModelSource{raw.Local}, raw.Dependencies...) {
		for _, e := range m.Source.Entries {
			if e.Kind == model.Directory {
				base := m.RepositoryArea.String()
				if base == "." {
					base = ""
				}
				if err = os.MkdirAll(filepath.Join(seed, filepath.FromSlash(base), filepath.FromSlash(e.Path)), 0700); err != nil {
					return project.CandidateRef{}, err
				}
			}
		}
	}
	service := &localService{adapter: g, request: run.Request{Prepared: p}, root: filepath.Dir(seed)}
	copy := &localCopy{service: service, location: seed}
	fixed, err := copy.Snapshot(ctx)
	if err != nil {
		return project.CandidateRef{}, err
	}
	attempt, err := copy.Inspect(ctx, fixed)
	if err != nil {
		return project.CandidateRef{}, err
	}
	if err = p.ValidateCandidate(attempt); err != nil {
		return project.CandidateRef{}, err
	}
	if attempt.Model().Inputs().Raw.Census.Digest() != raw.Census.Digest() {
		return project.CandidateRef{}, fmt.Errorf("initial candidate does not preserve complete checked Census")
	}
	return g.retain(ctx, id, p.Binding(), seed, fixed)

}
func (g *LocalGit) retain(ctx context.Context, id, binding, repo string, c run.Commit) (project.CandidateRef, error) {
	key, err := token()
	if err != nil {
		return project.CandidateRef{}, err
	}
	format, err := localCommand(ctx, repo, nil, "rev-parse", "--show-object-format")
	if err != nil {
		return project.CandidateRef{}, err
	}
	ref := project.CandidateRef{Project: id, Key: key, Format: format, Commit: c.ID, Tree: c.Tree, Binding: binding}
	path, err := g.candidatePath(ref)
	if err != nil {
		return ref, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return ref, err
	}
	pending, err := os.MkdirTemp(filepath.Dir(path), ".pending-")
	if err != nil {
		return ref, err
	}
	defer os.RemoveAll(pending)
	export := filepath.Join(pending, "export")
	if err = os.Mkdir(export, 0700); err != nil {
		return ref, err
	}
	if _, err = localCommand(ctx, export, nil, "init", "--quiet", "--object-format="+format); err != nil {
		return ref, err
	}
	name := "refs/markitect/retain/" + key
	if _, err = localCommand(ctx, export, nil, "fetch", "--quiet", "--no-tags", repo, c.ID+":"+name); err != nil {
		return ref, err
	}
	if _, err = localCommand(ctx, export, nil, "bundle", "create", filepath.Join(pending, "candidate.bundle"), name); err != nil {
		return ref, err
	}
	if err = os.RemoveAll(export); err != nil {
		return ref, err
	}

	data, err := json.Marshal(ref)
	if err == nil {
		err = os.WriteFile(filepath.Join(pending, "candidate.json"), data, 0600)
	}
	if err != nil {
		return ref, err
	}
	for _, n := range []string{"candidate.bundle", "candidate.json"} {
		f, err := os.OpenFile(filepath.Join(pending, n), os.O_RDWR, 0)
		if err != nil {
			return ref, err
		}
		err = f.Sync()
		ce := f.Close()
		if err != nil {
			return ref, err
		}
		if ce != nil {
			return ref, ce
		}
	}
	probe, err := os.MkdirTemp(g.TempRoot, "markitect-retention-")
	if err != nil {
		return ref, err
	}
	defer os.RemoveAll(probe)
	if err = g.reopen(ctx, ref, pending, probe); err != nil {
		return ref, err
	}
	if err = ctx.Err(); err != nil {
		return ref, err
	}
	if err = os.Rename(pending, path); err != nil {
		return ref, err
	}
	return ref, nil
}

// Reopen uses only durable state. The destination must be empty and independent.
func (g *LocalGit) Reopen(ctx context.Context, ref project.CandidateRef, destination string) error {
	path, err := g.candidatePath(ref)
	if err != nil {
		return err
	}
	return g.reopen(ctx, ref, path, destination)
}
func (g *LocalGit) reopen(ctx context.Context, ref project.CandidateRef, path, destination string) error {
	data, err := os.ReadFile(filepath.Join(path, "candidate.json"))
	if err != nil {
		return err
	}
	var bound project.CandidateRef
	if err = json.Unmarshal(data, &bound); err != nil || !reflect.DeepEqual(bound, ref) {
		return fmt.Errorf("stored candidate metadata mismatch")
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 0 {
		return fmt.Errorf("candidate reopen requires empty existing destination")
	}
	if ref.Format != "sha1" && ref.Format != "sha256" {
		return fmt.Errorf("unsupported Git object format")
	}
	if _, err = localCommand(ctx, destination, nil, "init", "--quiet", "--object-format="+ref.Format); err != nil {
		return err
	}
	if _, err = localCommand(ctx, destination, nil, "fetch", "--quiet", "--no-tags", filepath.Join(path, "candidate.bundle"), "refs/markitect/retain/"+ref.Key); err != nil {
		return err
	}
	if _, err = localCommand(ctx, destination, nil, "checkout", "--quiet", "--detach", ref.Commit); err != nil {
		return err
	}
	actual, err := object(ctx, destination, "HEAD")
	if err != nil || actual.ID != ref.Commit || actual.Tree != ref.Tree {
		return fmt.Errorf("reopened candidate commit/tree mismatch")
	}
	if _, err = localCommand(ctx, destination, nil, "fsck", "--full", "--no-reflogs"); err != nil {
		return err
	}
	return restoreDirectories(ctx, destination, ref.Commit)
}
func restoreDirectories(ctx context.Context, dir, commit string) error {
	data, err := localCommand(ctx, dir, nil, "ls-tree", "-r", "-d", "--name-only", "-z", commit)
	if err != nil {
		return err
	}
	for _, p := range strings.Split(data, "\x00") {
		if p == "" {
			continue
		}
		if filepath.IsAbs(p) || hasParentSegment(p) || strings.Contains(p, "\\") {
			return fmt.Errorf("unsafe stored directory")
		}
		if err = os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p)), 0700); err != nil {
			return err
		}
	}
	return nil
}
func (g *LocalGit) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	root, err := os.MkdirTemp(g.TempRoot, "markitect-run-")
	if err != nil {
		return nil, err
	}
	s := &localService{adapter: g, request: r, root: root, seed: filepath.Join(root, "seed")}
	if err = os.Mkdir(s.seed, 0700); err == nil {
		err = g.Reopen(ctx, r.Input, s.seed)
	}
	return s, err
}
func (s *localService) Copy(ctx context.Context, manager string, base run.Commit) (run.Copy, error) {
	key, err := token()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(s.root, "manager-"+key)
	c := &localCopy{s, p}
	s.mu.Lock()
	s.copies = append(s.copies, c)
	sources := append([]*localCopy(nil), s.copies...)
	s.mu.Unlock()
	if err = os.Mkdir(p, 0700); err != nil {
		return c, err
	}
	if _, err = localCommand(ctx, p, nil, "init", "--quiet", "--object-format="+s.request.Input.Format); err != nil {
		return c, err
	}
	source := s.seed
	for _, candidate := range sources {
		if candidate == c {
			continue
		}
		if actual, e := object(ctx, candidate.location, base.ID); e == nil && actual == base {
			source = candidate.location
			break
		}
	}
	if _, err = localCommand(ctx, p, nil, "fetch", "--quiet", "--no-tags", source, base.ID); err != nil {
		return c, err
	}
	if _, err = localCommand(ctx, p, nil, "checkout", "--quiet", "--detach", base.ID); err != nil {
		return c, err
	}
	err = restoreDirectories(ctx, p, base.ID)
	return c, err
}
func (c *localCopy) Location() run.CopyLocation { return run.CopyLocation{Directory: c.location} }
func (c *localCopy) Snapshot(ctx context.Context) (run.Commit, error) {
	if _, err := localCommand(ctx, c.location, nil, "add", "--force", "--all", "--", "."); err != nil {
		return run.Commit{}, err
	}
	tree, err := localCommand(ctx, c.location, nil, "write-tree")
	if err != nil {
		return run.Commit{}, err
	}
	// Git's index omits empty directories. Canonical model Package directories
	// are intent, so encode the actual complete inventory as real Git trees.
	raw := c.service.request.Prepared.Attempt().Model().Inputs().Raw
	var dirs []string
	for _, m := range append([]project.ModelSource{raw.Local}, raw.Dependencies...) {
		if m.RepositoryArea.String() == "" {
			continue
		}
		root := m.Root.String()
		if root == "." {
			root = ""
		}
		if !m.RepositoryArea.IsRoot() {
			root = strings.Trim(m.RepositoryArea.String()+"/"+root, "/")
		}
		err = filepath.WalkDir(filepath.Join(c.location, filepath.FromSlash(root)), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("redirected model directory")
			}
			if d.IsDir() {
				relative, e := filepath.Rel(c.location, p)
				if e != nil {
					return e
				}
				if relative != "." {
					dirs = append(dirs, filepath.ToSlash(relative))
				}
			}
			return nil
		})
		if err != nil {
			return run.Commit{}, err
		}
	}
	tree, err = withDirectories(ctx, c.location, tree, dirs)
	if err != nil {
		return run.Commit{}, err
	}
	before, err := object(ctx, c.location, "HEAD")
	if err != nil {
		return run.Commit{}, err
	}
	if before.Tree == tree {
		return before, nil
	}
	id, err := localCommand(ctx, c.location, []byte("Markitect fixed Manager candidate\n"), "commit-tree", tree, "-p", before.ID)
	if err != nil {
		return run.Commit{}, err
	}
	if _, err = localCommand(ctx, c.location, nil, "reset", "--mixed", id); err != nil {
		return run.Commit{}, err
	}
	return run.Commit{ID: id, Tree: tree}, nil
}

// Build only missing tree nodes; file entries remain exact Git index objects.
func withDirectories(ctx context.Context, repo, tree string, dirs []string) (string, error) {
	children := map[string][]string{}
	for _, dir := range dirs {
		head, tail, has := strings.Cut(dir, "/")
		if has {
			children[head] = append(children[head], tail)
		} else if _, ok := children[head]; !ok {
			children[head] = nil
		}
	}
	if len(children) == 0 {
		return tree, nil
	}
	data, err := localCommand(ctx, repo, nil, "ls-tree", "-z", tree)
	if err != nil {
		return "", err
	}
	entries := map[string]string{}
	for _, entry := range strings.Split(data, "\x00") {
		if entry == "" {
			continue
		}
		_, name, ok := strings.Cut(entry, "\t")
		if !ok {
			return "", fmt.Errorf("invalid Git tree entry")
		}
		entries[name] = entry
	}
	for name, nested := range children {
		entry, ok := entries[name]
		var child string
		if ok {
			fields := strings.Fields(strings.SplitN(entry, "\t", 2)[0])
			if len(fields) != 3 || fields[1] != "tree" {
				return "", fmt.Errorf("model directory is not a tree")
			}
			child = fields[2]
		} else {
			child, err = localCommand(ctx, repo, []byte{}, "mktree")
			if err != nil {
				return "", err
			}
		}
		child, err = withDirectories(ctx, repo, child, nested)
		if err != nil {
			return "", err
		}
		entries[name] = "040000 tree " + child + "\t" + name
	}
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(entries[n])
		b.WriteByte(0)
	}
	return localCommand(ctx, repo, []byte(b.String()), "mktree", "-z")
}
func (c *localCopy) Inspect(ctx context.Context, fixed run.Commit) (project.PreviewPreparationAttempt, error) {
	actual, err := object(ctx, c.location, fixed.ID)
	if err != nil || actual != fixed {
		return project.PreviewPreparationAttempt{}, fmt.Errorf("fixed candidate identity unavailable")
	}
	selection := c.service.request.Prepared.Selection()
	original := selection.Repository
	selection.Repository = c.location
	remap := func(boundary string) string {
		if filepath.IsAbs(boundary) {
			relative, err := filepath.Rel(original, boundary)
			if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return filepath.Join(c.location, relative)
			}
		}
		return boundary
	}
	selection.Local.Boundary = remap(selection.Local.Boundary)
	for i := range selection.Dependencies {
		selection.Dependencies[i].Boundary = remap(selection.Dependencies[i].Boundary)
	}
	for _, group := range []*[]project.SourceLocation{&selection.Execution, &selection.Registry, &selection.Ignore, &selection.Instructions} {
		for i := range *group {
			(*group)[i].Boundary = remap((*group)[i].Boundary)
		}
	}
	// Registry closure must discover newly added shards rather than inherit the
	// old selected list. Runtime metadata is not needed for candidate checking.
	selection.Registry = nil
	selection.RuntimeProfiles = nil
	repo, err := New(Limits{})
	if err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	prepared, err := project.NewPreparationService(project.NewModelService(repo, repo)).Preview(ctx, project.PreviewRequest{Selection: selection, Revision: fixed.ID}, c.service.request.Prepared.Attempt().Model().RequestedLimits())
	attempts := prepared.Attempts()
	if len(attempts) == 0 {
		return project.PreviewPreparationAttempt{}, err
	}
	return attempts[len(attempts)-1], err
}
func (c *localCopy) Integrate(ctx context.Context, source run.Copy, commit run.Commit) (run.Commit, error) {
	child, ok := source.(*localCopy)
	if !ok || child.service != c.service {
		return run.Commit{}, fmt.Errorf("foreign Run copy")
	}
	actual, err := object(ctx, child.location, commit.ID)
	if err != nil || actual != commit {
		return run.Commit{}, fmt.Errorf("fixed child commit mismatch")
	}
	if _, err = localCommand(ctx, c.location, nil, "fetch", "--quiet", "--no-tags", child.location, commit.ID); err != nil {
		return run.Commit{}, err
	}
	if _, err = localCommand(ctx, c.location, nil, "merge", "--no-ff", "--no-edit", commit.ID); err != nil {
		localCommand(ctx, c.location, nil, "merge", "--abort")
		return run.Commit{}, err
	}
	return c.Snapshot(ctx)
}
func (s *localService) Retain(ctx context.Context, copy run.Copy, commit run.Commit) (project.CandidateRef, error) {
	c, ok := copy.(*localCopy)
	if !ok || c.service != s {
		return project.CandidateRef{}, fmt.Errorf("foreign Run candidate")
	}
	actual, err := object(ctx, c.location, commit.ID)
	if err != nil || actual != commit {
		return project.CandidateRef{}, fmt.Errorf("candidate identity unavailable")
	}
	return s.adapter.retain(ctx, s.request.Project, s.request.Prepared.Binding(), c.location, commit)
}
func removePrivate(ctx context.Context, parent, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(parent, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refuse cleanup outside private Run root")
	}
	return os.RemoveAll(target)
}
func (c *localCopy) Close(ctx context.Context) error {
	return removePrivate(ctx, c.service.root, c.location)
}
func (s *localService) Close(ctx context.Context) error {
	return removePrivate(ctx, filepath.Dir(s.root), s.root)
}

func hasParentSegment(p string) bool {
	for _, s := range strings.Split(p, "/") {
		if s == ".." || s == "." {
			return true
		}
	}
	return false
}
func (c *localCopy) Update(ctx context.Context, source run.Copy, base run.Commit) error {
	parent, ok := source.(*localCopy)
	if !ok || parent.service != c.service {
		return fmt.Errorf("foreign parent copy")
	}
	actual, err := object(ctx, parent.location, base.ID)
	if err != nil || actual != base {
		return fmt.Errorf("parent base unavailable")
	}
	if _, err = localCommand(ctx, c.location, nil, "fetch", "--quiet", "--no-tags", parent.location, base.ID); err != nil {
		return err
	}
	if _, err = localCommand(ctx, c.location, nil, "merge", "--no-edit", base.ID); err != nil {
		localCommand(ctx, c.location, nil, "merge", "--abort")
		return err
	}
	return restoreDirectories(ctx, c.location, base.ID)
}
func (c *localCopy) Review(ctx context.Context, source run.Copy, base, result run.Commit) (string, error) {
	child, ok := source.(*localCopy)
	if !ok || child.service != c.service {
		return "", fmt.Errorf("foreign child copy")
	}
	actual, err := object(ctx, child.location, result.ID)
	if err != nil || actual != result {
		return "", fmt.Errorf("child fixed result unavailable")
	}
	if _, err = localCommand(ctx, c.location, nil, "fetch", "--quiet", "--no-tags", child.location, result.ID); err != nil {
		return "", err
	}
	return localCommand(ctx, c.location, nil, "diff", "--no-ext-diff", "--no-textconv", "--binary", base.ID, result.ID, "--")
}
