package gitrepo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
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
type LocalGit struct {
	StateRoot, TempRoot string
	MaxInputBytes       int64
}
type localService struct {
	adapter    *LocalGit
	request    run.Request
	root, seed string
	mu         sync.Mutex
	copies     []*localCopy
	inputs     retainedInputs
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
	// Git for Windows also needs long paths for its state-owned pack files.
	prefix := []string{"--no-replace-objects", "--literal-pathspecs", "-c", "core.longpaths=true", "-c", "core.hooksPath=", "-c", "core.fsmonitor=false", "-c", "core.autocrlf=false", "-c", "commit.gpgsign=false", "-c", "user.name=Markitect Run", "-c", "user.email=run@markitect.invalid", "-C", dir}
	if runtime.GOOS == "windows" && len(args) > 0 && args[0] == "fetch" {
		// The local upload-pack server needs its own long-path setting. Client
		// -c does not cover its source object reads. Keep source paths separate
		// from this static command and leave user repository config untouched.
		args = append([]string{"fetch", "--upload-pack=git -c core.longpaths=true upload-pack"}, args[1:]...)
	}
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
	if args[0] == "ls-tree" || args[0] == "diff" || args[0] == "ls-files" || args[0] == "cat-file" {
		return out.String(), nil
	}
	return strings.TrimSpace(out.String()), nil
}

type gitCommand func(context.Context, string, []byte, ...string) (string, error)

func object(ctx context.Context, dir, rev string) (run.Commit, error) {
	return objectUsing(ctx, dir, rev, localCommand)
}
func objectUsing(ctx context.Context, dir, rev string, command gitCommand) (run.Commit, error) {
	id, err := command(ctx, dir, nil, "rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return run.Commit{}, err
	}
	tree, err := command(ctx, dir, nil, "rev-parse", "--verify", id+"^{tree}")
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
		if m.RepositoryArea.String() == "" {
			continue
		}
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
	service := &localService{adapter: g, request: run.Request{Prepared: p}, root: filepath.Dir(seed), inputs: retainedFrom(p)}
	copy := &localCopy{service: service, location: seed}
	fixed, err := copy.Snapshot(ctx)
	if err != nil {
		return project.CandidateRef{}, err
	}
	attempt, err := copy.Inspect(ctx, fixed)
	if err != nil {
		return project.CandidateRef{}, err
	}
	// The command's input may still owe realization: its stale registry links
	// are work for Implement, not a reason to refuse the input.
	if err = p.ValidateCandidateFor(attempt, project.CommandInput); err != nil {
		return project.CandidateRef{}, err
	}
	if attempt.Model().Inputs().Raw.Census.Digest() != raw.Census.Digest() {
		return project.CandidateRef{}, fmt.Errorf("initial candidate does not preserve complete checked Census")
	}
	return g.retain(ctx, id, p, seed, fixed)

}
func (g *LocalGit) retain(ctx context.Context, id string, p project.Prepared, repo string, c run.Commit) (project.CandidateRef, error) {
	format, err := localCommand(ctx, repo, nil, "rev-parse", "--show-object-format")
	if err != nil {
		return project.CandidateRef{}, err
	}
	return g.storeBundle(ctx, id, p, format, c, func(ctx context.Context, key, bundle string) error {
		export, err := os.MkdirTemp(g.TempRoot, "markitect-export-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(export)
		// This private export starts with no template refs and receives one ref.
		if _, err = localCommand(ctx, export, nil, "init", "--quiet", "--template=", "--object-format="+format); err != nil {
			return err
		}
		name := "refs/markitect/retain/" + key
		if _, err = localCommand(ctx, export, nil, "fetch", "--quiet", "--no-tags", repo, c.ID+":"+name); err != nil {
			return err
		}
		selection := name
		if runtime.GOOS == "windows" {
			// Git 2.56 cannot bundle long loose refs even with longpaths enabled.
			// Pack and enumerate this export's sole ref, preserving its full name.
			if _, err = localCommand(ctx, export, nil, "pack-refs", "--all"); err != nil {
				return err
			}
			selection = "--all"
		}
		_, err = localCommand(ctx, export, nil, "bundle", "create", bundle, selection)
		return err
	})
}
func (g *LocalGit) storeBundle(ctx context.Context, id string, p project.Prepared, format string, c run.Commit, write func(context.Context, string, string) error) (project.CandidateRef, error) {
	key, err := token()
	if err != nil {
		return project.CandidateRef{}, err
	}
	ref := project.CandidateRef{Project: id, Key: key, Format: format, Commit: c.ID, Tree: c.Tree, Binding: p.Binding()}
	destination, err := g.candidatePath(ref)
	if err != nil {
		return ref, err
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return ref, err
	}
	pending, err := os.MkdirTemp(filepath.Dir(destination), ".pending-")
	if err != nil {
		return ref, err
	}
	defer os.RemoveAll(pending)
	if err = write(ctx, key, filepath.Join(pending, "candidate.bundle")); err != nil {
		return ref, err
	}
	inputLimit, err := g.inputLimit()
	if err != nil {
		return ref, err
	}
	inputsData, err := json.Marshal(retainedFrom(p))
	if err != nil {
		return ref, err
	}
	if int64(len(inputsData)) > inputLimit {
		return ref, ErrLimit
	}
	if err = os.WriteFile(filepath.Join(pending, "inputs.json"), inputsData, 0600); err != nil {
		return ref, err
	}
	data, err := json.Marshal(ref)
	if err == nil {
		err = os.WriteFile(filepath.Join(pending, "candidate.json"), data, 0600)
	}
	if err != nil {
		return ref, err
	}
	for _, n := range []string{"candidate.bundle", "candidate.json", "inputs.json"} {
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
	if err = verifyBundle(ctx, ref, pending, probe); err != nil {
		return ref, err
	}
	if _, err = g.retainedAt(ctx, pending, p); err != nil {
		return ref, err
	}
	if err = ctx.Err(); err != nil {
		return ref, err
	}
	if err = os.Rename(pending, destination); err != nil {
		return ref, err
	}
	return ref, nil
}

// Verify Git objects in an independent bare repository. Host checkout modes
// are not an oracle for Linux working-copy permissions.
func verifyBundle(ctx context.Context, ref project.CandidateRef, stored, probe string) error {
	if ref.Format != "sha1" && ref.Format != "sha256" {
		return fmt.Errorf("unsupported Git object format")
	}
	if _, err := localCommand(ctx, probe, nil, "init", "--bare", "--quiet", "--object-format="+ref.Format); err != nil {
		return err
	}
	if _, err := localCommand(ctx, probe, nil, "fetch", "--quiet", "--no-tags", filepath.Join(stored, "candidate.bundle"), "refs/markitect/retain/"+ref.Key+":refs/markitect/verify"); err != nil {
		return err
	}
	actual, err := object(ctx, probe, ref.Commit)
	if err != nil || actual.ID != ref.Commit || actual.Tree != ref.Tree {
		return fmt.Errorf("independent bare candidate identity mismatch")
	}
	_, err = localCommand(ctx, probe, nil, "fsck", "--full", "--no-reflogs")
	return err
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
	inputs, err := g.retained(ctx, r.Input, r.Prepared)
	if err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp(g.TempRoot, "markitect-run-")
	if err != nil {
		return nil, err
	}
	s := &localService{adapter: g, request: r, root: root, seed: filepath.Join(root, "seed"), inputs: inputs}
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
	c := &localCopy{service: s, location: p}
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
	dirs, err := c.directories(ctx)
	if err != nil {
		return run.Commit{}, err
	}
	return snapshotGit(ctx, c.location, localCommand, dirs, localExecutables(c.location))
}

// Outputs lists the copy's build outputs, which no fixed state holds.
func (c *localCopy) Outputs(ctx context.Context) ([]string, error) {
	return untrackedOutputs(ctx, c.location, localCommand, localExecutables(c.location))
}

// elfMagic opens every ELF file: the executables, objects and libraries that
// builds in the Linux runtime produce.
var elfMagic = []byte{0x7f, 'E', 'L', 'F'}

// elfProbe selects the ELF files among untracked paths of a copy. It reads
// only their first bytes and follows no symlink; a vanished path is skipped.
type elfProbe func(context.Context, []string) ([]string, error)

// untrackedOutputs lists the build outputs of a copy: its untracked ELF files
// that no registry shard in its index names. Such files are operational state:
// they never enter a fixed commit.
func untrackedOutputs(ctx context.Context, dir string, command gitCommand, executables elfProbe) ([]string, error) {
	elf, err := untrackedELF(ctx, dir, command, executables)
	if err != nil {
		return nil, err
	}
	return unregistered(ctx, dir, command, elf)
}

func untrackedELF(ctx context.Context, dir string, command gitCommand, executables elfProbe) ([]string, error) {
	untracked, err := command(ctx, dir, nil, "ls-files", "--others", "-z")
	if err != nil {
		return nil, err
	}
	return executables(ctx, strings.FieldsFunc(untracked, func(r rune) bool { return r == 0 }))
}

// unregistered keeps the paths that no registry shard in the copy's index
// names exactly: registration, not the file format, makes a file a
// realization artifact. A directory entry claims no build output.
func unregistered(ctx context.Context, dir string, command gitCommand, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	listing, err := command(ctx, dir, nil, "ls-files", "-z", "--", ".markitect/registry")
	if err != nil {
		return nil, err
	}
	var shards []string
	var request strings.Builder
	for _, p := range strings.FieldsFunc(listing, func(r rune) bool { return r == 0 }) {
		if strings.HasSuffix(p, "/registry.yaml") && !strings.Contains(p, "\n") {
			shards = append(shards, p)
			request.WriteString(":" + p + "\n")
		}
	}
	registered := map[string]bool{}
	if len(shards) != 0 {
		out, err := command(ctx, dir, []byte(request.String()), "cat-file", "--batch")
		if err != nil {
			return nil, err
		}
		for _, shard := range shards {
			header, rest, _ := strings.Cut(out, "\n")
			fields := strings.Fields(header)
			if strings.HasSuffix(header, " missing") {
				out = rest
				continue
			}
			size := -1
			if len(fields) == 3 {
				if size, err = strconv.Atoi(fields[2]); err != nil {
					size = -1
				}
			}
			if size < 0 || len(rest) <= size || rest[size] != '\n' {
				return nil, fmt.Errorf("unreadable registry shard %s", shard)
			}
			for _, file := range project.RegisteredFiles(shard, []byte(rest[:size])) {
				registered[file] = true
			}
			out = rest[size+1:]
		}
	}
	var outputs []string
	for _, p := range paths {
		if !registered[p] {
			outputs = append(outputs, p)
		}
	}
	return outputs, nil
}

// clearOutputs removes the untracked build outputs that a merge of incoming
// would overwrite: at a path the commit holds, where it holds a directory, or
// below a file it holds. Git refuses such a merge outright instead of
// reporting a repairable conflict; the outputs are disposable.
func clearOutputs(ctx context.Context, dir, incoming string, command gitCommand, executables elfProbe) error {
	outputs, err := untrackedOutputs(ctx, dir, command, executables)
	if err != nil || len(outputs) == 0 {
		return err
	}
	listing, err := command(ctx, dir, nil, "ls-tree", "-r", "-t", "-z", incoming)
	if err != nil {
		return err
	}
	entries := strings.Split(listing, "\x00")
	var needed []string
	for _, p := range outputs {
		for _, entry := range entries {
			meta, name, _ := strings.Cut(entry, "\t")
			if name == p || strings.HasPrefix(name, p+"/") || strings.HasPrefix(p, name+"/") && !strings.Contains(meta, " tree ") {
				needed = append(needed, p)
				break
			}
		}
	}
	if len(needed) == 0 {
		return nil
	}
	_, err = command(ctx, dir, nil, append([]string{"clean", "--force", "-x", "--quiet", "--"}, needed...)...)
	return err
}

func localExecutables(location string) elfProbe {
	return func(_ context.Context, paths []string) ([]string, error) {
		var found []string
		head := make([]byte, len(elfMagic))
		for _, p := range paths {
			name := filepath.Join(location, filepath.FromSlash(p))
			info, err := os.Lstat(name)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			f, err := os.Open(name)
			if err != nil {
				return nil, err
			}
			_, err = io.ReadFull(f, head)
			f.Close()
			if err == nil && bytes.Equal(head, elfMagic) {
				found = append(found, p)
			} else if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, err
			}
		}
		return found, nil
	}
}
func modelDirectoryRoots(raw project.RawInputs) []string {
	var roots []string
	for _, m := range append([]project.ModelSource{raw.Local}, raw.Dependencies...) {
		if m.RepositoryArea.String() != "" {
			root := path.Join(m.RepositoryArea.String(), m.Root.String())
			if root == "." {
				root = ""
			}
			roots = append(roots, root)
		}
	}
	return roots
}
func (c *localCopy) directories(ctx context.Context) ([]string, error) {
	return localModelDirectories(ctx, c.location, modelDirectoryRoots(c.service.request.Prepared.Attempt().Model().Inputs().Raw))
}
func localModelDirectories(ctx context.Context, location string, roots []string) ([]string, error) {
	var err error
	var dirs []string
	for _, root := range roots {
		err = filepath.WalkDir(filepath.Join(location, filepath.FromSlash(root)), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			relative, e := filepath.Rel(location, p)
			if e != nil {
				return e
			}
			name := filepath.ToSlash(relative)
			// Match acquisition's administration exclusions before traversing them.
			if technical(name) || d.IsDir() && name == ".markitect/state" {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("redirected model directory")
			}
			if d.IsDir() && name != "." {
				dirs = append(dirs, name)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func snapshotGit(ctx context.Context, dir string, command gitCommand, dirs []string, executables elfProbe) (run.Commit, error) {
	// A Manager must resolve and stage its conflict before submission. Never let
	// the ordinary snapshot's add-all turn unresolved index entries into success.
	conflicts, err := command(ctx, dir, nil, "ls-files", "--unmerged", "-z")
	if err != nil {
		return run.Commit{}, err
	}
	if conflicts != "" {
		return run.Commit{}, fmt.Errorf("%w: unresolved Git index prevents fixed candidate; resolve and stage before resubmitting", run.ErrMergeConflict)
	}
	mergeParent, err := command(ctx, dir, nil, "rev-parse", "--revs-only", "--end-of-options", "MERGE_HEAD")
	if err != nil {
		return run.Commit{}, err
	}
	if mergeParent != "" {
		if _, err = objectUsing(ctx, dir, mergeParent, command); err != nil {
			return run.Commit{}, err
		}
	}
	// Untracked ELF files that no registry shard names, such as a Manager's own
	// build outputs, stay untracked in the copy. Everything else, Gitignored or
	// not, is captured; the shards are read once staged, as the copy holds them.
	elf, err := untrackedELF(ctx, dir, command, executables)
	if err != nil {
		return run.Commit{}, err
	}
	if _, err = command(ctx, dir, nil, "add", "--force", "--all", "--", "."); err != nil {
		return run.Commit{}, err
	}
	outputs, err := unregistered(ctx, dir, command, elf)
	if err != nil {
		return run.Commit{}, err
	}
	if len(outputs) != 0 {
		if _, err = command(ctx, dir, []byte(strings.Join(outputs, "\x00")+"\x00"), "update-index", "--force-remove", "-z", "--stdin"); err != nil {
			return run.Commit{}, err
		}
	}
	tree, err := command(ctx, dir, nil, "write-tree")
	if err != nil {
		return run.Commit{}, err
	}
	tree, err = withDirectoriesUsing(ctx, dir, tree, dirs, command)
	if err != nil {
		return run.Commit{}, err
	}
	before, err := objectUsing(ctx, dir, "HEAD", command)
	if err != nil {
		return run.Commit{}, err
	}
	if before.Tree == tree && mergeParent == "" {
		return before, nil
	}
	args := []string{"commit-tree", tree, "-p", before.ID}
	if mergeParent != "" && mergeParent != before.ID {
		args = append(args, "-p", mergeParent)
	}
	id, err := command(ctx, dir, []byte("Markitect fixed Manager candidate\n"), args...)
	if err != nil {
		return run.Commit{}, err
	}
	if _, err = command(ctx, dir, nil, "reset", "--mixed", id); err != nil {
		return run.Commit{}, err
	}
	return run.Commit{ID: id, Tree: tree}, nil
}

// Build only missing tree nodes; file entries remain exact Git index objects.
func withDirectories(ctx context.Context, repo, tree string, dirs []string) (string, error) {
	return withDirectoriesUsing(ctx, repo, tree, dirs, localCommand)
}
func withDirectoriesUsing(ctx context.Context, repo, tree string, dirs []string, command gitCommand) (string, error) {
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
	data, err := command(ctx, repo, nil, "ls-tree", "-z", tree)
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
			child, err = command(ctx, repo, []byte{}, "mktree")
			if err != nil {
				return "", err
			}
		}
		child, err = withDirectoriesUsing(ctx, repo, child, nested, command)
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
	return command(ctx, repo, []byte(b.String()), "mktree", "-z")
}
func (c *localCopy) Inspect(ctx context.Context, fixed run.Commit) (project.PreviewPreparationAttempt, error) {
	actual, err := object(ctx, c.location, fixed.ID)
	if err != nil || actual != fixed {
		return project.PreviewPreparationAttempt{}, fmt.Errorf("fixed candidate identity unavailable")
	}
	return inspectCandidate(ctx, c.service.request.Prepared, c.location, fixed, c.service.inputs)
}
func inspectCandidate(ctx context.Context, p project.Prepared, directory string, fixed run.Commit, retained retainedInputs) (project.PreviewPreparationAttempt, error) {
	selection := p.Selection()
	original := selection.Repository
	selection.Repository = directory
	remap := func(boundary string) string {
		if filepath.IsAbs(boundary) {
			relative, err := filepath.Rel(original, boundary)
			if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return filepath.Join(directory, relative)
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
	repo.retained = retained
	prepared, err := project.NewPreparationService(project.NewModelService(repo, repo)).Preview(ctx, project.PreviewRequest{Selection: selection, Revision: fixed.ID}, p.Attempt().Model().RequestedLimits())
	attempts := prepared.Attempts()
	if len(attempts) == 0 {
		return project.PreviewPreparationAttempt{}, err
	}
	if onlyInvalidManifest(err) {
		return attempts[len(attempts)-1], nil
	}
	return attempts[len(attempts)-1], err
}

// A known-invalid language manifest is retained in the attempt; independent
// acquisition/cleanup errors still travel as technical port errors.
func onlyInvalidManifest(err error) bool {
	if err == nil {
		return false
	}
	if direct, ok := err.(*project.ManifestPreflightError); ok {
		return direct.Inspection().Status == model.Failed
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		parts := joined.Unwrap()
		if len(parts) == 0 {
			return false
		}
		for _, part := range parts {
			if !onlyInvalidManifest(part) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return onlyInvalidManifest(wrapped.Unwrap())
	}
	return false
}
func (c *localCopy) Integrate(ctx context.Context, source run.Copy, commit run.Commit) (run.Commit, error) {
	child, ok := source.(*localCopy)
	if !ok || child.service != c.service {
		return run.Commit{}, fmt.Errorf("foreign Run copy")
	}
	return integrateGit(ctx, c.location, child.location, commit, localCommand, c.directories, localExecutables(c.location))
}

// Only a verified merge conflict with successful abort and exact restored parent
// state is repairable. Transport, object, abort and readback failures stay technical.
func integrateGit(ctx context.Context, directory, source string, fixed run.Commit, command gitCommand, directories func(context.Context) ([]string, error), executables elfProbe) (run.Commit, error) {
	actual, err := objectUsing(ctx, source, fixed.ID, command)
	if err != nil {
		return run.Commit{}, err
	}
	if actual != fixed {
		return run.Commit{}, fmt.Errorf("fixed child identity mismatch")
	}
	if _, err = command(ctx, directory, nil, "fetch", "--quiet", "--no-tags", source, fixed.ID); err != nil {
		return run.Commit{}, err
	}
	beforeDirs, err := directories(ctx)
	if err != nil {
		return run.Commit{}, err
	}
	before, err := snapshotGit(ctx, directory, command, beforeDirs, executables)
	if err != nil {
		return run.Commit{}, err
	}
	if err = clearOutputs(ctx, directory, fixed.ID, command, executables); err != nil {
		return run.Commit{}, err
	}
	// The snapshot leaves only the parent's own build outputs untracked.
	untracked, err := command(ctx, directory, nil, "ls-files", "--others", "-z")
	if err != nil {
		return run.Commit{}, err
	}
	if _, err = command(ctx, directory, nil, "merge", "--no-ff", "--no-edit", fixed.ID); err != nil {
		conflicts, indexErr := command(ctx, directory, nil, "ls-files", "--unmerged", "-z")
		mergeHead, headErr := objectUsing(ctx, directory, "MERGE_HEAD", command)
		_, abort := command(ctx, directory, nil, "merge", "--abort")
		readback := verifyRestoredGit(ctx, directory, before, beforeDirs, untracked, command, directories)
		if indexErr == nil && conflicts != "" && headErr == nil && mergeHead == fixed && abort == nil && readback == nil {
			return run.Commit{}, fmt.Errorf("%w: child %s conflicts with parent %s; parent was restored", run.ErrMergeConflict, fixed.ID, before.ID)
		}
		return run.Commit{}, errors.Join(err, indexErr, headErr, abort, readback)
	}
	dirs, err := directories(ctx)
	if err != nil {
		return run.Commit{}, err
	}
	return snapshotGit(ctx, directory, command, dirs, executables)
}

// Observe restoration without staging, committing, resetting or hiding residue.
// Untracked files must be exactly those present before the merge.
func verifyRestoredGit(ctx context.Context, directory string, before run.Commit, beforeDirs []string, beforeUntracked string, command gitCommand, directories func(context.Context) ([]string, error)) error {
	restored, err := objectUsing(ctx, directory, "HEAD", command)
	if err != nil {
		return err
	}
	if restored != before {
		return fmt.Errorf("failed integration changed fixed parent")
	}
	for _, args := range [][]string{{"diff", "--quiet", "--no-ext-diff", "--no-textconv", "--cached", "HEAD", "--"}, {"diff", "--quiet", "--no-ext-diff", "--no-textconv", "--"}} {
		if _, err = command(ctx, directory, nil, args...); err != nil {
			return fmt.Errorf("parent index/worktree restoration incomplete: %w", err)
		}
	}
	untracked, err := command(ctx, directory, nil, "ls-files", "--others", "-z")
	if err != nil {
		return err
	}
	if untracked != beforeUntracked {
		return fmt.Errorf("parent restoration changed untracked files")
	}
	pending, err := command(ctx, directory, nil, "rev-parse", "--revs-only", "--end-of-options", "MERGE_HEAD")
	if err != nil {
		return err
	}
	if pending != "" {
		return fmt.Errorf("parent merge remains open after abort")
	}
	dirs, err := directories(ctx)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(beforeDirs, dirs) {
		return fmt.Errorf("parent directory restoration incomplete")
	}
	return nil
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
	return s.adapter.retain(ctx, s.request.Project, s.request.Prepared, c.location, commit)
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
	if err = clearOutputs(ctx, c.location, base.ID, localCommand, localExecutables(c.location)); err != nil {
		return err
	}
	if _, err = localCommand(ctx, c.location, nil, "merge", "--no-edit", base.ID); err != nil {
		conflicts, observed := localCommand(ctx, c.location, nil, "ls-files", "--unmerged", "-z")
		if observed == nil && conflicts != "" {
			mergeHead, observedHead := object(ctx, c.location, "MERGE_HEAD")
			if observedHead == nil && mergeHead == base {
				return fmt.Errorf("%w: resolve and stage the child-side merge against parent %s", run.ErrMergeConflict, base.ID)
			}
			observed = errors.Join(observed, observedHead, fmt.Errorf("conflict parent identity mismatch"))
		}
		_, abort := localCommand(ctx, c.location, nil, "merge", "--abort")
		return errors.Join(err, observed, abort)
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
	return localCommand(ctx, c.location, nil, "diff", "--no-ext-diff", "--no-textconv", "--full-index", base.ID, result.ID, "--")
}
