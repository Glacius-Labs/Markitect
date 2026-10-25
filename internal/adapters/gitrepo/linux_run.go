package gitrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// LinuxGit owns one hardened, held Git service and private Linux volume per Run.
// Store holds durable bundles; it is never a second writable Manager workspace.
// Spec and transfer limits are explicit outer runtime configuration.
type LinuxGit struct {
	Engine           *container.Engine
	Spec             container.ContainerSpec
	Store            *LocalGit
	MaxTransferBytes int64
}

type linuxService struct {
	adapter *LinuxGit
	request run.Request
	exec    container.ExecSession
	volume  string
	labels  map[string]string
	mu      sync.Mutex
	copies  []*linuxCopy
	inputs  retainedInputs
}
type linuxCopy struct {
	service            *linuxService
	directory, subpath string
}

const linuxRunRoot = "/assignments/run"

func (g *LinuxGit) Open(ctx context.Context, r run.Request) (run.WorkspaceService, error) {
	if g == nil || g.Engine == nil || g.Store == nil || g.MaxTransferBytes <= 0 || g.MaxTransferBytes > 1<<30 {
		return nil, fmt.Errorf("Linux Git needs an engine, durable store and finite transfer bound")
	}
	inputs, err := g.Store.retained(ctx, r.Input, r.Prepared)
	if err != nil {
		return nil, err
	}
	key, err := token()
	if err != nil {
		return nil, err
	}
	s := &linuxService{adapter: g, request: r, inputs: inputs, volume: "markitect-run-" + key, labels: maps.Clone(g.Spec.Labels)}
	if s.labels == nil {
		s.labels = map[string]string{}
	}
	s.labels["markitect.run-service"] = key
	s.labels["markitect.run"] = r.ID
	if err = g.Engine.RequireVolumeSubpaths(ctx); err != nil {
		return s, err
	}
	if _, err = g.Engine.CreateVolumeExclusive(ctx, s.volume, s.labels); err != nil {
		return s, err
	}
	spec := g.Spec
	spec.Name = "git-" + key
	spec.Labels = maps.Clone(s.labels)
	spec.Mounts = []container.Mount{{Volume: s.volume, Target: linuxRunRoot, NoCopy: true}}
	spec.Workdir = linuxRunRoot
	s.exec, err = g.Engine.StartOwnedExec(ctx, spec, []string{"/usr/local/bin/node", "-e", "setInterval(()=>{},2147483647)"})
	if err != nil {
		return s, err
	}
	if s.exec == nil {
		return s, fmt.Errorf("Git service startup returned no handle")
	}
	if err = s.mkdir(ctx, linuxRunRoot+"/seed"); err != nil {
		return s, err
	}
	stored, err := g.Store.candidatePath(r.Input)
	if err != nil {
		return s, err
	}
	manifest, err := os.ReadFile(filepath.Join(stored, "candidate.json"))
	if err != nil {
		return s, err
	}
	var bound project.CandidateRef
	if json.Unmarshal(manifest, &bound) != nil || bound != r.Input {
		return s, fmt.Errorf("input candidate manifest mismatch")
	}
	bundle, err := os.Open(filepath.Join(stored, "candidate.bundle"))
	if err != nil {
		return s, err
	}
	defer bundle.Close()
	info, err := bundle.Stat()
	if err != nil {
		return s, err
	}
	if info.Size() > g.MaxTransferBytes {
		return s, fmt.Errorf("input bundle exceeds transfer limit")
	}
	if err = s.write(ctx, linuxRunRoot+"/input.bundle", bundle); err != nil {
		return s, err
	}
	seed := linuxRunRoot + "/seed"
	if _, err = s.command(ctx, seed, nil, "init", "--quiet", "--object-format="+r.Input.Format); err != nil {
		return s, err
	}
	if _, err = s.command(ctx, seed, nil, "fetch", "--quiet", "--no-tags", linuxRunRoot+"/input.bundle", "refs/markitect/retain/"+r.Input.Key); err != nil {
		return s, err
	}
	if _, err = s.command(ctx, seed, nil, "checkout", "--quiet", "--detach", r.Input.Commit); err != nil {
		return s, err
	}
	actual, err := objectUsing(ctx, seed, r.Input.Commit, s.command)
	if err != nil || actual.ID != r.Input.Commit || actual.Tree != r.Input.Tree {
		return s, fmt.Errorf("Linux seed object identity mismatch: %w", err)
	}
	if _, err = s.command(ctx, seed, nil, "fsck", "--full", "--no-reflogs"); err != nil {
		return s, err
	}
	if err = s.restoreDirectories(ctx, seed, r.Input.Commit); err != nil {
		return s, err
	}
	return s, nil
}

func (s *linuxService) command(ctx context.Context, dir string, input []byte, args ...string) (string, error) {
	prefix := []string{"/usr/bin/git", "--no-replace-objects", "--literal-pathspecs", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "core.autocrlf=false", "-c", "core.filemode=true", "-c", "commit.gpgsign=false", "-c", "user.name=Markitect Run", "-c", "user.email=run@markitect.invalid", "-c", "protocol.allow=never", "-c", "protocol.file.allow=always", "-C", dir}
	out := &boundedOutput{limit: 64 << 20}
	err := s.exec.Exec(ctx, append(prefix, args...), bytes.NewReader(input), out)
	if err != nil {
		return "", err
	}
	if out.limitExceeded {
		return "", fmt.Errorf("Git output exceeds finite bound")
	}
	if args[0] == "ls-tree" || args[0] == "diff" || args[0] == "ls-files" {
		return out.String(), nil
	}
	return strings.TrimSpace(out.String()), nil
}

func (s *linuxService) node(ctx context.Context, script string, args []string, input io.Reader, out io.Writer) error {
	if s.exec == nil {
		return fmt.Errorf("Git service is not running")
	}
	return s.exec.Exec(ctx, append([]string{"/usr/local/bin/node", "-e", script}, args...), input, out)
}
func (s *linuxService) mkdir(ctx context.Context, directory string) error {
	return s.node(ctx, `require('fs').mkdirSync(process.argv[1],{recursive:true,mode:0o700})`, []string{directory}, nil, io.Discard)
}
func (s *linuxService) write(ctx context.Context, file string, input io.Reader) error {
	return s.node(ctx, `const fs=require('fs');fs.writeFileSync(process.argv[1],fs.readFileSync(0),{flag:'wx',mode:0o600})`, []string{file}, input, io.Discard)
}
func (s *linuxService) remove(ctx context.Context, directory string) error {
	if !strings.HasPrefix(directory, linuxRunRoot+"/") || path.Clean(directory) != directory {
		return fmt.Errorf("refuse cleanup outside private Run root")
	}
	return s.node(ctx, `const fs=require('fs');fs.rmSync(process.argv[1],{recursive:true,force:true});if(fs.existsSync(process.argv[1]))throw Error('private path remains')`, []string{directory}, nil, io.Discard)
}
func (s *linuxService) restoreDirectories(ctx context.Context, dir, commit string) error {
	data, err := s.command(ctx, dir, nil, "ls-tree", "-r", "-d", "--name-only", "-z", commit)
	if err != nil {
		return err
	}
	var dirs []string
	for _, d := range strings.Split(data, "\x00") {
		if d == "" {
			continue
		}
		if strings.HasPrefix(d, "/") || hasParentSegment(d) || strings.Contains(d, "\\") {
			return fmt.Errorf("unsafe stored directory")
		}
		dirs = append(dirs, d)
	}
	encoded, err := json.Marshal(dirs)
	if err != nil {
		return err
	}
	return s.node(ctx, `const fs=require('fs'),p=require('path');for(const d of JSON.parse(process.argv[2]))fs.mkdirSync(p.join(process.argv[1],d),{recursive:true,mode:0o700})`, []string{dir, string(encoded)}, nil, io.Discard)
}

func (s *linuxService) Copy(ctx context.Context, manager string, base run.Commit) (run.Copy, error) {
	key, err := token()
	if err != nil {
		return nil, err
	}
	c := &linuxCopy{service: s, subpath: "managers/" + key}
	c.directory = linuxRunRoot + "/" + c.subpath
	s.mu.Lock()
	s.copies = append(s.copies, c)
	sources := append([]*linuxCopy(nil), s.copies...)
	s.mu.Unlock()
	if err = s.mkdir(ctx, c.directory); err != nil {
		return c, err
	}
	source := linuxRunRoot + "/seed"
	if actual, e := objectUsing(ctx, source, base.ID, s.command); e != nil || actual != base {
		found := false
		for _, candidate := range sources {
			if candidate == c {
				continue
			}
			if actual, e = objectUsing(ctx, candidate.directory, base.ID, s.command); e == nil && actual == base {
				source = candidate.directory
				found = true
				break
			}
		}
		if !found {
			return c, fmt.Errorf("fixed copy base unavailable")
		}
	}
	if _, err = s.command(ctx, c.directory, nil, "init", "--quiet", "--object-format="+s.request.Input.Format); err != nil {
		return c, err
	}
	if _, err = s.command(ctx, c.directory, nil, "fetch", "--quiet", "--no-tags", source, base.ID); err != nil {
		return c, err
	}
	if _, err = s.command(ctx, c.directory, nil, "checkout", "--quiet", "--detach", base.ID); err != nil {
		return c, err
	}
	return c, s.restoreDirectories(ctx, c.directory, base.ID)
}
func (c *linuxCopy) Location() run.CopyLocation {
	return run.CopyLocation{Volume: c.service.volume, Subpath: c.subpath}
}

// Enumerate actual model directory entries and reject unsupported realization
// nodes before Git can silently omit them. No external symlink is followed.
func (c *linuxCopy) Snapshot(ctx context.Context) (run.Commit, error) {
	dirs, err := c.directories(ctx)
	if err != nil {
		return run.Commit{}, err
	}
	return snapshotGit(ctx, c.directory, c.service.command, dirs, c.executables)
}
func (c *linuxCopy) executables(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out := &boundedOutput{limit: 4 << 20}
	err := c.service.node(ctx, `const fs=require('fs'),p=require('path');const base=process.argv[1],found=[],head=Buffer.alloc(4);for(const rel of fs.readFileSync(0,'utf8').split('\0')){if(rel==='')continue;const file=p.join(base,rel);let s;try{s=fs.lstatSync(file)}catch(e){if(e.code==='ENOENT')continue;throw e}if(!s.isFile())continue;const fd=fs.openSync(file,fs.constants.O_RDONLY|fs.constants.O_NOFOLLOW|fs.constants.O_NONBLOCK);try{if(fs.readSync(fd,head,0,4,0)===4&&head.equals(Buffer.from([0x7f,0x45,0x4c,0x46])))found.push(rel)}finally{fs.closeSync(fd)}}process.stdout.write(JSON.stringify(found))`, []string{c.directory}, strings.NewReader(strings.Join(paths, "\x00")), out)
	if err != nil {
		return nil, err
	}
	if out.limitExceeded {
		return nil, fmt.Errorf("executable observation exceeds bound")
	}
	var found []string
	if err = json.Unmarshal([]byte(out.String()), &found); err != nil {
		return nil, err
	}
	listed := map[string]bool{}
	for _, p := range paths {
		listed[p] = true
	}
	for _, p := range found {
		if !listed[p] {
			return nil, fmt.Errorf("executable observation names an unlisted path")
		}
	}
	return found, nil
}
func (c *linuxCopy) directories(ctx context.Context) ([]string, error) {
	roots := modelDirectoryRoots(c.service.request.Prepared.Attempt().Model().Inputs().Raw)
	encoded, _ := json.Marshal(roots)
	out := &boundedOutput{limit: 4 << 20}
	err := c.service.node(ctx, `const fs=require('fs'),p=require('path');const base=process.argv[1],roots=JSON.parse(process.argv[2]),dirs=[];function walk(rel){const file=p.join(base,rel),s=fs.lstatSync(file);if(s.isSymbolicLink()||(!s.isDirectory()&&!s.isFile()))throw Error('unsupported working-copy node '+rel);if(s.isDirectory()){if(rel!==''&&roots.some(r=>r===''||rel===r||rel.startsWith(r+'/')))dirs.push(rel);for(const n of fs.readdirSync(file)){if(n==='.git'){if(rel===''&&!fs.lstatSync(p.join(file,n)).isDirectory())throw Error('invalid Git directory');continue}const next=rel?rel+'/'+n:n;if(next==='.markitect/state'&&fs.lstatSync(p.join(base,next)).isDirectory())continue;walk(next)}}}walk('');for(const r of roots)if(!fs.lstatSync(p.join(base,r)).isDirectory())throw Error('missing model root');process.stdout.write(JSON.stringify(dirs))`, []string{c.directory, string(encoded)}, nil, out)
	if err != nil {
		return nil, err
	}
	if out.limitExceeded {
		return nil, fmt.Errorf("model directory observation exceeds bound")
	}
	var dirs []string
	if err = json.Unmarshal([]byte(out.String()), &dirs); err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}
func (c *linuxCopy) Update(ctx context.Context, source run.Copy, base run.Commit) error {
	parent, ok := source.(*linuxCopy)
	if !ok || parent.service != c.service {
		return fmt.Errorf("foreign parent copy")
	}
	actual, err := objectUsing(ctx, parent.directory, base.ID, c.service.command)
	if err != nil || actual != base {
		return fmt.Errorf("fixed parent identity unavailable")
	}
	if _, err = c.service.command(ctx, c.directory, nil, "fetch", "--quiet", "--no-tags", parent.directory, base.ID); err != nil {
		return err
	}
	if err = clearOutputs(ctx, c.directory, base.ID, c.service.command, c.executables); err != nil {
		return err
	}
	if _, err = c.service.command(ctx, c.directory, nil, "merge", "--no-edit", base.ID); err != nil {
		conflicts, observed := c.service.command(ctx, c.directory, nil, "ls-files", "--unmerged", "-z")
		if observed == nil && conflicts != "" {
			mergeHead, observedHead := objectUsing(ctx, c.directory, "MERGE_HEAD", c.service.command)
			if observedHead == nil && mergeHead == base {
				return fmt.Errorf("%w: resolve and stage the child-side merge against parent %s", run.ErrMergeConflict, base.ID)
			}
			observed = errors.Join(observed, observedHead, fmt.Errorf("conflict parent identity mismatch"))
		}
		_, abort := c.service.command(ctx, c.directory, nil, "merge", "--abort")
		return errors.Join(err, observed, abort)
	}
	return c.service.restoreDirectories(ctx, c.directory, base.ID)
}
func (c *linuxCopy) Review(ctx context.Context, source run.Copy, base, result run.Commit) (string, error) {
	child, ok := source.(*linuxCopy)
	if !ok || child.service != c.service {
		return "", fmt.Errorf("foreign child copy")
	}
	actual, err := objectUsing(ctx, child.directory, result.ID, c.service.command)
	if err != nil || actual != result {
		return "", fmt.Errorf("fixed child identity unavailable")
	}
	if _, err = c.service.command(ctx, c.directory, nil, "fetch", "--quiet", "--no-tags", child.directory, result.ID); err != nil {
		return "", err
	}
	return c.service.command(ctx, c.directory, nil, "diff", "--no-ext-diff", "--no-textconv", "--full-index", base.ID, result.ID, "--")
}
func (c *linuxCopy) Integrate(ctx context.Context, source run.Copy, fixed run.Commit) (run.Commit, error) {
	child, ok := source.(*linuxCopy)
	if !ok || child.service != c.service {
		return run.Commit{}, fmt.Errorf("foreign child copy")
	}
	return integrateGit(ctx, c.directory, child.directory, fixed, c.service.command, c.directories, c.executables)
}

func (s *linuxService) export(ctx context.Context, source string, fixed run.Commit, key, destination string) error {
	actual, err := objectUsing(ctx, source, fixed.ID, s.command)
	if err != nil || actual != fixed {
		return fmt.Errorf("export fixed identity unavailable")
	}
	dir := linuxRunRoot + "/exports/" + key
	if err = s.mkdir(ctx, dir); err != nil {
		return err
	}
	if _, err = s.command(ctx, dir, nil, "init", "--quiet", "--object-format="+s.request.Input.Format); err != nil {
		return err
	}
	name := "refs/markitect/retain/" + key
	if _, err = s.command(ctx, dir, nil, "fetch", "--quiet", "--no-tags", source, fixed.ID+":"+name); err != nil {
		return err
	}
	if _, err = s.command(ctx, dir, nil, "bundle", "create", dir+"/candidate.bundle", name); err != nil {
		return err
	}
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	writer := &boundedTransfer{writer: f, remaining: s.adapter.MaxTransferBytes}
	err = s.exec.Exec(ctx, []string{"/bin/cat", dir + "/candidate.bundle"}, nil, writer)
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	return s.remove(ctx, dir)
}

type boundedTransfer struct {
	writer    io.Writer
	remaining int64
}

func (w *boundedTransfer) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, fmt.Errorf("bundle exceeds finite transfer bound")
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func (c *linuxCopy) Inspect(ctx context.Context, fixed run.Commit) (project.PreviewPreparationAttempt, error) {
	g := c.service.adapter.Store
	temporary, err := os.MkdirTemp(g.TempRoot, "markitect-fixed-inspect-")
	if err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	defer os.RemoveAll(temporary)
	key, err := token()
	if err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	ref := project.CandidateRef{Project: c.service.request.Project, Key: key, Binding: c.service.request.Prepared.Binding(), Format: c.service.request.Input.Format, Commit: fixed.ID, Tree: fixed.Tree}
	if err = c.service.export(ctx, c.directory, fixed, key, filepath.Join(temporary, "candidate.bundle")); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	data, _ := json.Marshal(ref)
	if err = os.WriteFile(filepath.Join(temporary, "candidate.json"), data, 0600); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	probe := filepath.Join(temporary, "bare")
	if err = os.Mkdir(probe, 0700); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	if err = verifyBundle(ctx, ref, temporary, probe); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	// This independent fixed-revision export is only structural observation.
	// No Session writes here, and no bytes synchronize back to the Linux copy.
	view := filepath.Join(temporary, "view")
	if err = os.Mkdir(view, 0700); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	if err = g.reopen(ctx, ref, temporary, view); err != nil {
		return project.PreviewPreparationAttempt{}, err
	}
	return inspectCandidate(ctx, c.service.request.Prepared, view, fixed, c.service.inputs)
}
func (s *linuxService) Retain(ctx context.Context, copy run.Copy, fixed run.Commit) (project.CandidateRef, error) {
	c, ok := copy.(*linuxCopy)
	if !ok || c.service != s {
		return project.CandidateRef{}, fmt.Errorf("foreign Run candidate")
	}
	return s.adapter.Store.storeBundle(ctx, s.request.Project, s.request.Prepared, s.request.Input.Format, fixed, func(ctx context.Context, key, bundle string) error {
		return s.export(ctx, c.directory, fixed, key, bundle)
	})
}
func (c *linuxCopy) Close(ctx context.Context) error { return c.service.remove(ctx, c.directory) }
func (s *linuxService) Close(ctx context.Context) error {
	var err error
	if s.exec != nil {
		err = s.exec.Close(ctx)
	}
	v, observe := s.adapter.Engine.InspectVolume(ctx, s.volume)
	err = errors.Join(err, observe)
	if observe != nil {
		return err
	}
	if !v.Exists {
		return err
	}
	for k, value := range s.labels {
		if v.Labels[k] != value {
			return errors.Join(err, fmt.Errorf("Run volume ownership not confirmed"))
		}
	}
	mounts, mountErr := s.adapter.Engine.VolumeMounts(ctx, s.volume)
	if mountErr != nil {
		return errors.Join(err, mountErr)
	}
	if len(mounts) > 0 {
		return errors.Join(err, fmt.Errorf("Run volume still mounted by %d containers", len(mounts)))
	}
	err = errors.Join(err, s.adapter.Engine.RemoveVolume(ctx, s.volume))
	v, observe = s.adapter.Engine.InspectVolume(ctx, s.volume)
	if observe != nil || v.Exists {
		return errors.Join(err, observe, fmt.Errorf("Run volume closure unconfirmed"))
	}
	return err
}
