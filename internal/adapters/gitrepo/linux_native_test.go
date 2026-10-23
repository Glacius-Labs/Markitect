package gitrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Explicit coordinator opt-in only. The frozen packet supplies the immutable
// existing image, literal engine and a new output directory; no build/pull/auth
// or model Session occurs. This exercises the actual LinuxGit consumer.
func TestLinuxGitNativeCopyProbe(t *testing.T) {
	if os.Getenv("MARKITECT_TEST_LINUX_COPY_PROBE") != "1" {
		t.Skip("requires a concrete bounded LinuxGit probe dispatch")
	}
	enginePath, image, out := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_RUNTIME_IMAGE"), os.Getenv("MARKITECT_TEST_LINUX_COPY_OUTPUT")
	if !filepath.IsAbs(enginePath) || !filepath.IsAbs(out) || !strings.HasPrefix(image, "sha256:") || len(image) != 71 {
		t.Fatal("explicit engine/image/new absolute evidence output required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	nonce, err := token()
	if err != nil {
		t.Fatal(err)
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
