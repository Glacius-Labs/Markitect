package gitrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type realApplyGitRuntime struct {
	*container.Engine
	starts  int
	execs   atomic.Int64
	helpers atomic.Int64
}

type countedApplyGitExec struct {
	container.ExecSession
	count *atomic.Int64
}

func (s countedApplyGitExec) Exec(ctx context.Context, argv []string, input io.Reader, output io.Writer) error {
	s.count.Add(1)
	return s.ExecSession.Exec(ctx, argv, input, output)
}

func (r *realApplyGitRuntime) StartOwnedExec(ctx context.Context, spec container.ContainerSpec, argv []string) (container.ExecSession, error) {
	r.starts++
	session, err := r.Engine.StartOwnedExec(ctx, spec, argv)
	if err != nil {
		return nil, err
	}
	return countedApplyGitExec{ExecSession: session, count: &r.execs}, nil
}

func (r *realApplyGitRuntime) RunHelper(ctx context.Context, spec container.ContainerSpec, argv []string, input io.Reader, output io.Writer) error {
	if len(argv) > 0 && argv[0] == "/usr/bin/git" {
		return fmt.Errorf("short-lived Git helper is forbidden")
	}
	r.helpers.Add(1)
	return r.Engine.RunHelper(ctx, spec, argv, input, output)
}

// Opt-in engine-only smoke: no provider, credentials, image build or pull.
// MARKITECT_TEST_GIT_IMAGE names an already available immutable runtime image.
func TestRealApplyGitContainer(t *testing.T) {
	enginePath, image := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_GIT_IMAGE")
	if enginePath == "" || image == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_GIT_IMAGE for the engine-only Git smoke")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	observed, err := engine.ImageID(ctx, image)
	if err != nil || observed != image {
		t.Fatalf("smoke requires an exact local image ID: %s %v", observed, err)
	}
	store := fmt.Sprintf("git-smoke-%d", time.Now().UnixNano())
	runtime := &realApplyGitRuntime{Engine: engine}
	w, err := NewWorkspaces(runtime, container.ContainerSpec{Name: store, Image: image, User: "1000:1000", Labels: map[string]string{"markitect.store": store}, CPUs: 1, MemoryBytes: 256 << 20, Pids: 128, TmpBytes: 16 << 20, Network: "none"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []run.WorkspaceID
	for i := 0; i < 6; i++ {
		ids = append(ids, run.WorkspaceID(fmt.Sprintf("smoke-work-%d", i)))
	}
	ids = append(ids, "smoke-assess")
	for i := 1; i <= 40; i++ {
		ids = append(ids, run.WorkspaceID(fmt.Sprintf("smoke-verify-%d", i)))
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 2*time.Minute)
		defer done()
		if err := w.CloseGit(cleanup); err != nil {
			t.Errorf("Git service cleanup: %v", err)
		}
		for _, id := range ids {
			if w.owned[id] != "" {
				if _, err := w.Close(cleanup, id); err != nil {
					t.Errorf("volume cleanup %s: %v", id, err)
				}
			}
		}
	})
	if err := w.StartGit(ctx, ids); err != nil {
		t.Fatal(err)
	}
	if runtime.helpers.Load() != 0 {
		t.Fatalf("StartGit started %d per-volume helpers", runtime.helpers.Load())
	}
	capabilityCheck := []string{"/usr/local/bin/node", "-e", `const fs=require('node:fs'); const eff=fs.readFileSync('/proc/self/status','utf8').match(/^CapEff:\s*([0-9a-f]+)$/m); if(process.getuid()!==1000||process.getgid()!==1000||!eff||BigInt('0x'+eff[1])!==0n) throw Error('ordinary exec is privileged'); for(const p of process.argv.slice(1)){const s=fs.statSync(p); if(s.uid!==1000||s.gid!==1000||(s.mode&0o777)!==0o755) throw Error('unexpected assignment-root permissions');}`}
	for _, target := range w.gitPaths {
		capabilityCheck = append(capabilityCheck, target)
	}
	if err := w.git.Exec(ctx, capabilityCheck, nil, io.Discard); err != nil {
		t.Fatal(err)
	}
	serviceID := w.git.ID()
	root, err := w.PrepareAssignment(ctx, run.WorkspaceSpec{ID: ids[0], Tree: workspaceTree(t, workspaceEntry(t, "common.txt", snapshot.Regular, "baseline\n"))})
	if err != nil {
		t.Fatal(err)
	}
	child, err := w.ForkAssignment(ctx, root, ids[1])
	if err != nil {
		t.Fatal(err)
	}
	childPath := w.gitPaths[child.Volume()]
	if err := w.git.Exec(ctx, []string{"/usr/local/bin/node", "-e", "require('node:fs').writeFileSync(process.argv[1], 'changed\\n')", childPath + "/common.txt"}, nil, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "--all"}, {"commit", "-m", "ordinary Agent result"}} {
		if err := w.git.Exec(ctx, append([]string{"/usr/bin/git", "-C", childPath}, args...), nil, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	a := child.(*assignmentWorkspace)
	commit, err := a.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := a.git(ctx, "log", "-1", "--format=%an|%ae|%cn|%ce")
	if err != nil || identity != "Markitect|assignment@markitect.invalid|Markitect|assignment@markitect.invalid" {
		t.Fatalf("ordinary container commit identity: %q %v", identity, err)
	}
	handoff := run.AssignmentHandoff{StartingCommit: child.StartingCommit(), ResultCommit: commit, Summary: "changed common.txt"}
	if diff, err := w.Changes(ctx, child, handoff); err != nil || !strings.Contains(diff, "changed") {
		t.Fatalf("child diff: %s %v", diff, err)
	}
	merged, err := w.Integrate(ctx, root, child, handoff, "")
	if err != nil {
		t.Fatal(err)
	}
	review, err := w.ForkAssignmentCommit(ctx, root, merged, "smoke-assess")
	if err != nil {
		t.Fatal(err)
	}
	tree, err := w.ReadAssignmentCommit(ctx, run.AssignmentCommitRequest{Source: review, Commit: merged, MaxEntries: 10, MaxBytes: 1024})
	if err != nil || tree.EntryCount() != 1 {
		t.Fatalf("read reviewed stand: entries=%d %v", tree.EntryCount(), err)
	}
	if names, err := engine.Containers(ctx, "markitect.store", store); err != nil || len(names) != 1 || runtime.starts != 1 || w.git.ID() != serviceID {
		t.Fatalf("Git lifetime: names=%v starts=%d %v", names, runtime.starts, err)
	}
	if err := w.CloseGit(ctx); err != nil {
		t.Fatal(err)
	}
	if names, err := engine.Containers(ctx, "markitect.store", store); err != nil || len(names) != 0 {
		t.Fatalf("Git container remained: %v %v", names, err)
	}
	for _, id := range ids {
		if volume, err := engine.InspectVolume(ctx, w.volume(id)); err != nil || !volume.Exists {
			t.Fatalf("evidence volume disappeared: %s %v", id, err)
		}
	}
	if w.GitContainerID() != serviceID || w.GitExecCount() == 0 {
		t.Fatal("lost retained ID or started Git exec count")
	}
	t.Logf("one Git container %s, %d fixed mounts, %d started Git execs, zero StartGit helpers, normal UID/GID1000 CapEff=0; retained volumes and observed removal; no providers", serviceID, len(ids), w.GitExecCount())
}
