package gitrepo

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Explicit coordinator opt-in only. The frozen packet supplies the immutable
// existing image, literal engine and a new output directory; no build/pull/auth
// or model Session occurs. This exercises the actual LinuxGit consumer.
func TestLinuxGitNativeCopyProbe(t *testing.T) {
	if os.Getenv("MARKITECT_TEST_LINUX_COPY_PROBE") != "1" {
		t.Skip("requires a concrete bounded LinuxGit probe dispatch")
	}
	linuxNativeCopyProbe(t, false)
}

// Separate opt-in: no provider or Apply is started. Publication copies must work
// after the originating Run and its entire volume have already disappeared.
func TestLinuxPublicationCheckCopyProbe(t *testing.T) {
	if os.Getenv("MARKITECT_TEST_LINUX_PUBLICATION_COPY_PROBE") != "1" {
		t.Skip("requires a concrete bounded publication-copy probe dispatch")
	}
	linuxNativeCopyProbe(t, true)
}

func linuxNativeCopyProbe(t *testing.T, publication bool) {
	t.Helper()
	enginePath, image, out := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_RUNTIME_IMAGE"), os.Getenv("MARKITECT_TEST_LINUX_COPY_OUTPUT")
	if !filepath.IsAbs(enginePath) || !filepath.IsAbs(out) || !strings.HasPrefix(image, "sha256:") || len(image) != 71 {
		t.Fatal("explicit engine/image/new absolute evidence output required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	// The outer packet knows ownership before any native side effect, even if a
	// hard process stop prevents the deferred report from being written.
	nonce := os.Getenv("MARKITECT_TEST_LINUX_COPY_NONCE")
	decoded, err := hex.DecodeString(nonce)
	if err != nil || len(decoded) != 16 || nonce != strings.ToLower(nonce) {
		t.Fatal("explicit frozen packet ownership nonce required")
	}
	type observation struct {
		Action, Plain, Executable, GitMarker, Ignored       string
		PlainMode, ExecutableMode, ExecutableExit, UID, GID int
		EmptyPackage, GitPresent, HomeEmpty                 bool
	}
	report := struct {
		ProbeID, ImageID, Fixture                                                             string
		Locations                                                                             []run.CopyLocation
		Volumes                                                                               []string
		CheckHome                                                                             run.CopyLocation
		VolumeAbsent                                                                          map[string]bool
		Native                                                                                []observation
		Commands                                                                              []container.CommandResult
		Input, Retained                                                                       project.CandidateRef
		Commit                                                                                run.Commit
		BareCommit, BareTree, Bundle, BundleBytes, BundleExecutableMode, EmptyPackageObject   string
		RunClosed, ReopenedRunClosed, FirstVolumeAbsent, AllContainersAbsent, SourceUnchanged bool
		GitExecCount                                                                          int
		Failed                                                                                bool
		PublicationChecks                                                                     []run.CheckObservation
	}{ProbeID: nonce, ImageID: image}
	defer func() {
		report.Failed = t.Failed()
		data, e := json.MarshalIndent(report, "", "  ")
		if e == nil {
			e = os.WriteFile(filepath.Join(out, "copy.json"), append(data, '\n'), 0600)
		}
		if e != nil {
			t.Error(e)
		}
	}()
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	// The existing real-Git fixture includes Root, siblings, a grandchild and an
	// actual empty Package Git tree. Store proof bundles outside Go's temp cleanup.
	source, prepared, _, _ := runFixture(t)
	if publication {
		// The declaration is captured before opening any Linux resource. It is
		// part of the same fixed candidate inspected by the real check consumer.
		source.Write(".markitect/execution.yaml", fmt.Sprintf("apiVersion: markitect.execution/v2\nchecks:\n  - id: publication-bytes\n    profile: publication-node\n    directory: .\n    args: ['-e', %q]\n    timeout: 30s\n", strings.ReplaceAll(linuxPublicationObservation, "PACKET_NONCE", nonce)))
		source.Commit("Declare publication copy check")
		tree, e := withDirectories(ctx, source.Dir, source.Git("rev-parse", "HEAD^{tree}"), []string{".markitect/model/b/empty"})
		if e != nil {
			t.Fatal(e)
		}
		id, e := localCommand(ctx, source.Dir, []byte("Preserve empty Package\n"), "commit-tree", tree, "-p", source.Git("rev-parse", "HEAD"))
		if e != nil {
			t.Fatal(e)
		}
		source.Git("reset", "--mixed", id)
		repository := adapter(t)
		captured, e := project.NewPreparationService(project.NewModelService(repository, repository)).Capture(ctx, project.CaptureRequest{Selection: prepared.Selection()}, model.Limits{})
		a, ok := captured.Prepared()
		if e != nil || !ok {
			t.Fatal("declared fixture preparation", captured.Status(), e)
		}
		prepared, e = project.Prepare(a)
		if e != nil {
			t.Fatal(e)
		}
	}
	report.Fixture = source.Dir
	store := &LocalGit{StateRoot: filepath.Join(out, "durable"), TempRoot: filepath.Join(out, "temporary")}
	for _, dir := range []string{store.StateRoot, store.TempRoot} {
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	input, err := store.Capture(ctx, "fixture", prepared)
	if err != nil {
		t.Fatal(err)
	}
	report.Input = input
	initial := run.Commit{ID: input.Commit, Tree: input.Tree}
	labels := map[string]string{"markitect.store": "linux-copy-" + nonce, "markitect.native-volume-probe": nonce}
	spec := container.ContainerSpec{Name: "unused-git-name", Image: image, Labels: labels, User: "1000:1000", Workdir: "/assignments/run", Network: "none", CPUs: 1, MemoryBytes: 512 << 20, Pids: 64, TmpBytes: 32 << 20}
	adapter := &LinuxGit{Engine: engine, Spec: spec, Store: store, MaxTransferBytes: 32 << 20}
	request := run.Request{ID: "copy-probe-" + nonce, Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Implement}
	var service, reopened run.WorkspaceService
	var owned []run.Copy
	var scratch, home run.DisposableCopy
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 60*time.Second)
		defer done()
		if home != nil {
			if e := home.Close(cleanup); e != nil {
				t.Error(e)
			}
		}
		if scratch != nil {
			if e := scratch.Close(cleanup); e != nil {
				t.Error(e)
			}
		}
		for _, copy := range owned {
			if e := copy.Close(cleanup); e != nil {
				t.Error(e)
			}
		}
		if service != nil {
			if e := service.Close(cleanup); e != nil {
				t.Error(e)
			} else {
				report.RunClosed = true
			}
		}
		if reopened != nil {
			if e := reopened.Close(cleanup); e != nil {
				t.Error(e)
			} else {
				report.ReopenedRunClosed = true
			}
		}
		report.VolumeAbsent = map[string]bool{}
		for _, name := range report.Volumes {
			v, e := engine.InspectVolume(cleanup, name)
			report.VolumeAbsent[name] = e == nil && !v.Exists
			if !report.VolumeAbsent[name] {
				t.Errorf("Run volume absence unconfirmed: %s %+v %v", name, v, e)
			}
		}
		ids, e := engine.Containers(cleanup, "markitect.native-volume-probe", nonce)
		report.AllContainersAbsent = e == nil && len(ids) == 0
		if e != nil || len(ids) != 0 {
			t.Errorf("owned container absence unconfirmed: %v %v", ids, e)
		}
	}()
	service, err = adapter.Open(ctx, request)
	if actual, ok := service.(*linuxService); ok {
		report.Volumes = append(report.Volumes, actual.volume)
	}
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Copy(ctx, ".", initial)
	if first != nil {
		owned = append(owned, first)
	}
	if err != nil {
		t.Fatal(err)
	}
	sibling, err := service.Copy(ctx, "a", initial)
	if sibling != nil {
		owned = append(owned, sibling)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, copy := range []run.Copy{first, sibling} {
		report.Locations = append(report.Locations, copy.Location())
	}
	if first.Location().Volume != sibling.Location().Volume || first.Location().Subpath == sibling.Location().Subpath || first.Location().Directory != "" {
		t.Fatal("copies do not use distinct subpaths of the actual private Run volume")
	}
	observe := func(location run.CopyLocation, action string) observation {
		t.Helper()
		if !location.Valid() || location.Directory != "" {
			t.Fatal("not a Linux volume location")
		}
		cs := spec
		cs.Name = "copy-observe-" + nonce + "-" + fmt.Sprint(len(report.Native))
		cs.Workdir = "/work"
		cs.Mounts = []container.Mount{{Volume: location.Volume, Subpath: location.Subpath, Target: "/work", NoCopy: true}}
		if action == "check" {
			l := home.Location()
			cs.Mounts = append(cs.Mounts, container.Mount{Volume: l.Volume, Subpath: l.Subpath, Target: "/home/agent", NoCopy: true})
		}
		result, e := engine.RunCommand(ctx, cs, []string{"/usr/local/bin/node", "-e", linuxCopyObservation, action, nonce}, nil, 32<<10)
		report.Commands = append(report.Commands, result)
		if e != nil || !result.Started || !result.Exited || result.ExitCode != 0 || result.Uncertain || result.Truncated || result.ImageID != image {
			t.Fatalf("native copy command incomplete: %+v %v", result, e)
		}
		var value observation
		if e = json.Unmarshal(result.Output, &value); e != nil {
			t.Fatalf("native observation invalid: %v %s", e, result.Output)
		}
		if value.Ignored != "original ignored input\n" {
			t.Fatal("accounted Gitignored bytes lost", value.Ignored)
		}
		report.Native = append(report.Native, value)
		return value
	}
	written := observe(first.Location(), "write")
	if written.PlainMode != 0644 || written.ExecutableMode != 0755 || written.ExecutableExit != 0 || written.Plain != "plain:"+nonce+"\n" || written.Executable != "#!/bin/sh\nexit 0\n" || written.UID != 1000 || written.GID != 1000 || !written.EmptyPackage {
		t.Fatal("Linux manager bytes/modes/empty topology differ", written)
	}
	isolated := observe(sibling.Location(), "isolation")
	if isolated.Plain != "initial\n" || isolated.Executable != "initial\n" || isolated.GitMarker != "" || !isolated.EmptyPackage {
		t.Fatal("sibling working copy or Git administration leaked", isolated)
	}
	fixed, err := first.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	report.Commit = fixed
	retained, err := service.Retain(ctx, first, fixed)
	if err != nil {
		t.Fatal(err)
	}
	report.Retained = retained
	// The disposable check copy must preserve the same empty Package topology,
	// exact file bytes/modes and have no Git administration or Manager home.
	inspected, err := first.Inspect(ctx, fixed)
	if err != nil {
		t.Fatal(err)
	}
	directories, e := project.CheckDirectories(inspected.Model().Inputs().Raw)
	if e != nil {
		t.Fatal(e)
	}
	scratch, err = first.Materialize(ctx, inspected.Model().Inputs().Raw.Census, directories)
	if err != nil {
		t.Fatal(err)
	}
	if scratch.Tree() != inspected.Model().Inputs().Raw.Census.Digest() || !reflect.DeepEqual(scratch.Directories(), directories) {
		t.Fatal("materialized directory observation differs")
	}
	if scratch.Location().Volume != first.Location().Volume || scratch.Location().Subpath == first.Location().Subpath {
		t.Fatal("check copy is not a distinct same-service physical subpath")
	}
	empty, e := snapshot.NewTree(nil)
	if e != nil {
		t.Fatal(e)
	}
	home, err = first.Materialize(ctx, empty, nil)
	if err != nil {
		t.Fatal(err)
	}
	report.CheckHome = home.Location()
	if home.Tree() != empty.Digest() || home.Location().Volume != first.Location().Volume || home.Location().Subpath == first.Location().Subpath || home.Location().Subpath == scratch.Location().Subpath {
		t.Fatal("check home is not a separate empty same-service copy")
	}
	disposable := observe(scratch.Location(), "check")
	if !disposable.HomeEmpty || !disposable.EmptyPackage || disposable.GitPresent || disposable.Plain != written.Plain || disposable.Executable != written.Executable || disposable.PlainMode != 0644 || disposable.ExecutableMode != 0755 {
		t.Fatal("check copy lost bytes/modes/empty Package or includes Git", disposable)
	}
	afterScratch, e := first.Snapshot(ctx)
	if e != nil || afterScratch != fixed {
		t.Fatal("check copy aliases Manager copy", afterScratch, e)
	}
	if err = scratch.Close(ctx); err != nil {
		t.Fatal(err)
	}
	scratch = nil
	if err = home.Close(ctx); err != nil {
		t.Fatal(err)
	}
	home = nil
	volume := first.Location().Volume
	for _, copy := range owned {
		if err = copy.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}
	owned = nil
	report.GitExecCount = service.(*linuxService).exec.GitExecCount()
	if err = service.Close(ctx); err != nil {
		t.Fatal(err)
	}
	report.RunClosed = true
	service = nil
	v, err := engine.InspectVolume(ctx, volume)
	report.FirstVolumeAbsent = err == nil && !v.Exists
	if !report.FirstVolumeAbsent {
		t.Fatalf("old Run volume absence unconfirmed: %+v %v", v, err)
	}
	// Reopen independently in a brand-new bare repository only after old service
	// and all its copies/volume have gone. No host Manager mirror is used.
	stored, err := store.candidatePath(retained)
	if err != nil {
		t.Fatal(err)
	}
	report.Bundle = filepath.Join(stored, "candidate.bundle")
	bare := filepath.Join(out, "readback.git")
	if err = os.Mkdir(bare, 0700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		data, e := localCommand(ctx, bare, nil, args...)
		if e != nil {
			t.Fatal(e)
		}
		return data
	}
	git("init", "--quiet", "--bare", "--object-format="+retained.Format)
	git("fetch", "--quiet", "--no-tags", report.Bundle, "refs/markitect/retain/"+retained.Key+":refs/readback/result")
	git("fsck", "--full", "--no-reflogs")
	report.BareCommit = git("rev-parse", "refs/readback/result")
	report.BareTree = git("rev-parse", retained.Commit+"^{tree}")
	blob, e := exec.CommandContext(ctx, "git", "--no-replace-objects", "--literal-pathspecs", "-C", bare, "cat-file", "blob", retained.Commit+":out/a.txt").Output()
	if e != nil {
		t.Fatal(e)
	}
	report.BundleBytes = string(blob)
	report.BundleExecutableMode = git("ls-tree", retained.Commit, "--", "out/deep.txt")
	report.EmptyPackageObject = git("cat-file", "-t", retained.Commit+":.markitect/model/b/empty")
	if report.BareCommit != retained.Commit || report.BareTree != retained.Tree || report.BundleBytes != "plain:"+nonce+"\n" || !strings.HasPrefix(report.BundleExecutableMode, "100755 blob ") || report.EmptyPackageObject != "tree" {
		t.Fatal("independent bare bundle readback differs")
	}
	if publication {
		// This is Publisher's actual host-Git inspection and materializer wrapper,
		// without publishing into a user checkout or opening another Run service.
		dir := filepath.Join(out, "publication-host")
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err = store.Reopen(ctx, retained, dir); err != nil {
			t.Fatal(err)
		}
		inputs, e := store.retained(ctx, retained, prepared)
		if e != nil {
			t.Fatal(e)
		}
		copy := &localCopy{service: &localService{request: run.Request{Prepared: prepared}, inputs: inputs}, location: dir}
		a, e := copy.Inspect(ctx, fixed)
		if e != nil {
			t.Fatal(e)
		}
		if e = prepared.ValidateCandidate(a, "."); e != nil {
			t.Fatal(e)
		}
		tree := a.Model().Inputs().Raw.Census
		dirs, e := project.CheckDirectories(a.Model().Inputs().Raw)
		if e != nil {
			t.Fatal(e)
		}
		materializer := &LinuxCheckCopies{Engine: engine, Spec: spec, MaxTransferBytes: 32 << 20}
		var publicationCopies []run.DisposableCopy
		defer func() {
			cleanup, done := context.WithTimeout(context.Background(), 60*time.Second)
			defer done()
			for _, c := range publicationCopies {
				if e := c.Close(cleanup); e != nil {
					t.Error(e)
				}
			}
		}()
		wrapped := publicationCheckCopy{Copy: copy, materialize: func(ctx context.Context, tree snapshot.Tree, dirs []snapshot.Path) (run.DisposableCopy, error) {
			c, e := materializer.Materialize(ctx, tree, dirs)
			if c != nil {
				publicationCopies = append(publicationCopies, c)
				report.Volumes = append(report.Volumes, c.Location().Volume)
				report.Locations = append(report.Locations, c.Location())
			}
			return c, e
		}}
		document := fmt.Sprintf("apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles:\n  publication-node:\n    container:\n      profileVersion: 1\n      engineExecutable: %q\n      imageID: %s\n      executable: /usr/local/bin/node\n      argumentPrefix: []\n      env: []\n      limits: {milliCPUs: 1000, memoryBytes: 268435456, processes: 64}\n      maxOutputBytes: 4096\n", enginePath, image)
		filePath := fixturePath(t, "runtime.yaml")
		file, e := snapshot.NewEntry(filePath, snapshot.Regular, []byte(document))
		if e != nil {
			t.Fatal(e)
		}
		metadata, e := runtimeconfig.Prepare(ctx, project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: project.SourceLocation{Boundary: "publication-profile", Path: filePath}, File: file}}}, nil, checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader))
		if e != nil || metadata.Status != model.Passed {
			t.Fatal(metadata, e)
		}
		checker, e := checks.NewCandidateRunner(metadata.Profiles, engine, labels)
		if e != nil {
			t.Fatal(e)
		}
		report.PublicationChecks, e = checker.Check(ctx, prepared.Checks(), wrapped, fixed)
		if e != nil {
			t.Fatal(e)
		}
		if e = run.ValidateCheckObservations(prepared.Checks(), report.PublicationChecks, fixed, tree, dirs); e != nil {
			t.Fatal(e)
		}
		if len(report.PublicationChecks) != 1 || len(publicationCopies) != 2 {
			t.Fatal("missing real check receipt/input/home", report.PublicationChecks, len(publicationCopies))
		}
		for _, c := range publicationCopies {
			if c.Location().Volume == volume {
				t.Fatal("publication reused closed Run volume")
			}
			v, e := engine.InspectVolume(ctx, c.Location().Volume)
			if e != nil || v.Exists {
				t.Fatal("receipt closure did not remove publication volume", v, e)
			}
		}
		repository, e := New(Limits{})
		if e != nil {
			t.Fatal(e)
		}
		recaptured, e := repository.Capture(ctx, prepared.Attempt().Model().Request())
		if e != nil {
			t.Fatal(e)
		}
		report.SourceUnchanged = recaptured.Raw.Census.Equal(prepared.Attempt().Model().Inputs().Raw.Census) && recaptured.Head == prepared.Attempt().Model().Inputs().Head
		if !report.SourceUnchanged {
			t.Fatal("source repository changed")
		}
		return
	}
	request.ID = "copy-reopen-" + nonce
	request.Input = retained
	reopened, err = adapter.Open(ctx, request)
	if actual, ok := reopened.(*linuxService); ok {
		report.Volumes = append(report.Volumes, actual.volume)
	}
	if err != nil {
		t.Fatal(err)
	}
	restored, err := reopened.Copy(ctx, ".", fixed)
	if restored != nil {
		owned = append(owned, restored)
	}
	if err != nil {
		t.Fatal(err)
	}
	report.Locations = append(report.Locations, restored.Location())
	restoredValue := observe(restored.Location(), "restore")
	if restoredValue.GitMarker != "" || restoredValue.Plain != written.Plain || restoredValue.Executable != written.Executable || restoredValue.PlainMode != 0644 || restoredValue.ExecutableMode != 0755 || !restoredValue.EmptyPackage {
		t.Fatal("fresh Linux restore differs", restoredValue)
	}
	report.GitExecCount += reopened.(*linuxService).exec.GitExecCount()
	// GitExecCount is observed after work; the finite deadline and transfer bounds are enforced, not a hard command-count ceiling.
	repository, e := New(Limits{})
	if e != nil {
		t.Fatal(e)
	}
	recaptured, e := repository.Capture(ctx, prepared.Attempt().Model().Request())
	if e != nil {
		t.Fatal(e)
	}
	report.SourceUnchanged = recaptured.Raw.Census.Equal(prepared.Attempt().Model().Inputs().Raw.Census) && recaptured.Head == prepared.Attempt().Model().Inputs().Head
	if !report.SourceUnchanged {
		t.Fatal("source repository changed")
	}
}

const linuxCopyObservation = `const fs=require('fs');const action=process.argv[1],nonce=process.argv[2];if(action==='write'){fs.writeFileSync('/work/out/a.txt','plain:'+nonce+'\n');fs.chmodSync('/work/out/a.txt',0o644);fs.writeFileSync('/work/out/deep.txt','#!/bin/sh\nexit 0\n');fs.chmodSync('/work/out/deep.txt',0o755);fs.writeFileSync('/work/.git/markitect-probe',nonce)}const plain=fs.readFileSync('/work/out/a.txt','utf8'),executable=fs.readFileSync('/work/out/deep.txt','utf8');const marker='/work/.git/markitect-probe';const execution=action==='isolation'?null:require('child_process').spawnSync('/work/out/deep.txt',[],{stdio:'ignore'});if(execution&&(execution.error||execution.status!==0))throw Error('executable fixture failed');if(action==='check')fs.writeFileSync('/work/out/a.txt','scratch:'+nonce);process.stdout.write(JSON.stringify({Action:action,Plain:plain,Executable:executable,PlainMode:fs.statSync('/work/out/a.txt').mode&0o777,ExecutableMode:fs.statSync('/work/out/deep.txt').mode&0o777,UID:process.getuid(),GID:process.getgid(),EmptyPackage:fs.statSync('/work/.markitect/model/b/empty').isDirectory(),GitPresent:fs.existsSync('/work/.git'),Ignored:fs.readFileSync('/work/out/accounted.ignored','utf8'),HomeEmpty:action==='check'?fs.readdirSync('/home/agent').length===0:false,ExecutableExit:execution?execution.status:-1,GitMarker:fs.existsSync(marker)?fs.readFileSync(marker,'utf8'):''}));`

const linuxPublicationObservation = `const fs=require('fs'),assert=require('assert');assert.strictEqual(process.getuid(),1000);assert.strictEqual(process.getgid(),1000);assert.strictEqual(fs.readFileSync('out/a.txt','utf8'),'plain:PACKET_NONCE\n');assert.strictEqual(fs.readFileSync('out/deep.txt','utf8'),'#!/bin/sh\nexit 0\n');assert.strictEqual(fs.statSync('out/a.txt').mode&0o777,0o644);assert.strictEqual(fs.statSync('out/deep.txt').mode&0o777,0o755);assert.strictEqual(fs.readFileSync('out/accounted.ignored','utf8'),'original ignored input\n');assert(fs.statSync('.markitect/model/b/empty').isDirectory());assert(!fs.existsSync('.git'));assert.deepStrictEqual(fs.readdirSync('/home/agent'),[]);const p=require('child_process').spawnSync('./out/deep.txt',[],{stdio:'ignore'});assert(!p.error);assert.strictEqual(p.status,0);process.stdout.write('publication copy bytes/modes/directories/home passed\n');`
