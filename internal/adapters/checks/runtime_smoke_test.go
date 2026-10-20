package checks

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// This opt-in is deliberately separate from all generic runtime smoke settings.
// Preparing or checking this source does not authorize executing the smoke.
func TestCheckWorkspaceRuntimeSmoke(t *testing.T) {
	if os.Getenv("MARKITECT_CHECK_WORKSPACE_SMOKE") != "run-check-workspace-smoke" {
		t.Skip("requires the separately authorized Check+Workspace runtime smoke opt-in; see README.md")
	}
	engine, err := container.NewEngine(os.Getenv("MARKITECT_CHECK_WORKSPACE_ENGINE"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	helperImage := smokeImage(t, ctx, engine, "MARKITECT_CHECK_WORKSPACE_HELPER_IMAGE")
	checkImage := smokeImage(t, ctx, engine, "MARKITECT_CHECK_WORKSPACE_CHECK_IMAGE")
	if helperImage == checkImage {
		t.Fatal("trusted helper image and project check image must have distinct identities")
	}
	// Require a real copy-up challenge, observed without starting the image.
	for _, target := range []string{"/work/.markitect-smoke-image-seed", "/home/agent/.markitect-smoke-image-seed"} {
		seed, err := engine.ReadImageFile(ctx, checkImage, target)
		if err != nil || string(seed) != "image-copy-up-challenge\n" {
			t.Fatalf("check image must contain the documented seed at %s: bytes=%q err=%v", target, seed, err)
		}
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	name := "check-workspace-smoke-" + hex.EncodeToString(token[:])
	w, err := gitrepo.NewWorkspaces(engine, container.ContainerSpec{
		Name: name, Image: helperImage, Labels: map[string]string{"markitect.smoke": name},
		User: "1000:1000", Workdir: "/work", CPUs: 1, MemoryBytes: 256 << 20,
		Pids: 64, TmpBytes: 64 << 20, Network: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	observed := &smokeWorkspaces{Workspaces: w}
	runner, err := NewRunner(engine, observed)
	if err != nil {
		t.Fatal(err)
	}
	// Retry only adapter-owned cleanup after an assertion/operation failure.
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		for _, materialized := range observed.materialized {
			closed, err := w.Close(cleanup, materialized.ID)
			if err != nil || !closed.Closed || closed.Uncertain {
				t.Errorf("retained workspace %s (%s): %+v err=%v", materialized.ID, materialized.Volume, closed, err)
			}
		}
	})
	entry, err := snapshot.NewEntry(mustCheckPath(t, "pkg/input.bin"), snapshot.Executable, []byte{0, 1, 255, 10})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshot.NewTree([]snapshot.Entry{entry})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := snapshot.DirectoryAt(mustCheckPath(t, "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		tree snapshot.Tree
		dir  snapshot.DirectoryBoundary
		code string
		exit int
	}{
		{"exact-tree-pass", tree, directory, `
const fs = require('fs');
if (process.cwd() !== '/work/pkg' || process.env.SMOKE_LITERAL !== '$(literal); | >' || process.argv[1] !== 'space argument') throw Error('literal request changed');
if (JSON.stringify(fs.readdirSync('/work')) !== '["pkg"]' || JSON.stringify(fs.readdirSync('.')) !== '["input.bin"]') throw Error('tree paths changed');
if (!fs.readFileSync('input.bin').equals(Buffer.from([0,1,255,10])) || !(fs.statSync('input.bin').mode & 0o100)) throw Error('tree bytes or mode changed');
if (fs.readdirSync('/home/agent').length) throw Error('home not empty');
fs.writeFileSync('discarded', 'work mutation'); fs.writeFileSync('/home/agent/discarded', 'cache mutation');
process.stdout.write('exact-tree-pass\n');`, 0},
		{"empty-tree-known-failure", snapshot.Tree{}, snapshot.RootDirectory(), `
const fs = require('fs');
if (process.cwd() !== '/work' || fs.readdirSync('/work').length || fs.readdirSync('/home/agent').length) throw Error('verified empty volumes received image copy-up');
fs.writeFileSync('/work/discarded', 'work mutation'); fs.writeFileSync('/home/agent/discarded', 'cache mutation');
process.stdout.write('x'.repeat(8192)); process.exitCode = 23;`, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := len(observed.materialized)
			req := run.CheckRequest{
				ID: run.CheckID(name + "-" + tc.name), Tree: tc.tree, ImageID: checkImage,
				Executable: "/usr/local/bin/node", Args: []string{"-e", tc.code, "space argument"},
				Env: []string{"SMOKE_LITERAL=$(literal); | >"}, Directory: tc.dir,
				Limits:  run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 256 << 20, Processes: 64},
				Timeout: 2 * time.Minute, MaxOutputBytes: 512,
			}
			if !req.Valid() {
				t.Fatal("incomplete smoke request")
			}
			receipt, err := runner.Run(ctx, req)
			if err != nil || receipt.ID != req.ID || receipt.Tree != req.Tree.Digest() || receipt.ImageID != checkImage ||
				!receipt.Started || !receipt.Exited || receipt.ExitCode != tc.exit || receipt.TimedOut || receipt.Uncertain || receipt.Reason != "" ||
				receipt.Passed(req) != (tc.exit == 0) {
				t.Fatalf("receipt=%+v err=%v", receipt, err)
			}
			if tc.exit == 0 {
				if receipt.Truncated || string(receipt.Output) != "exact-tree-pass\n" {
					t.Fatalf("passing output=%q truncated=%v", receipt.Output, receipt.Truncated)
				}
			} else if !receipt.Truncated || !bytes.Equal(receipt.Output, bytes.Repeat([]byte("x"), int(req.MaxOutputBytes))) {
				t.Fatalf("failed output=%q truncated=%v", receipt.Output, receipt.Truncated)
			}
			if len(observed.materialized) != start+2 || len(observed.closed) != start+2 {
				t.Fatalf("expected two actual materializations and closures: materialized=%+v closed=%+v", observed.materialized, observed.closed)
			}
			for i, wantTree := range []snapshot.TreeID{req.Tree.Digest(), (snapshot.Tree{}).Digest()} {
				copy, closed := observed.materialized[start+i], observed.closed[start+i]
				if copy.ID == "" || copy.Volume == "" || copy.Tree != wantTree || copy.Closed || copy.Uncertain ||
					closed.ID != copy.ID || closed.Volume != copy.Volume || !closed.Closed || closed.Uncertain {
					t.Fatalf("materialized=%+v closed=%+v want tree=%s", copy, closed, wantTree)
				}
				v, err := engine.InspectVolume(ctx, copy.Volume)
				if err != nil || v.Name != copy.Volume || v.Exists {
					t.Fatalf("disposable volume absence unconfirmed: %+v err=%v", v, err)
				}
			}
			if observed.materialized[start].Volume == observed.materialized[start+1].Volume {
				t.Fatal("work and home shared a volume")
			}
			remaining, err := engine.Containers(ctx, "markitect.check", string(req.ID))
			if err != nil || len(remaining) != 0 {
				t.Fatalf("check container cleanup unconfirmed: %v err=%v", remaining, err)
			}
		})
	}
	remaining, err := engine.Containers(ctx, "markitect.smoke", name)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("workspace helper cleanup unconfirmed: %v err=%v", remaining, err)
	}
}

func smokeImage(t *testing.T, ctx context.Context, engine *container.Engine, variable string) string {
	t.Helper()
	id := os.Getenv(variable)
	if len(id) != 71 || id[:7] != "sha256:" {
		t.Fatalf("%s must be an exact preexisting sha256 image ID", variable)
	}
	if _, err := hex.DecodeString(id[7:]); err != nil {
		t.Fatalf("%s: %v", variable, err)
	}
	actual, err := engine.ImageID(ctx, id)
	if err != nil || actual != id {
		t.Fatalf("%s image identity: requested=%s observed=%s err=%v", variable, id, actual, err)
	}
	return actual
}

// Observe the concrete adapter without replacing its materialization or close.
type smokeWorkspaces struct {
	*gitrepo.Workspaces
	materialized []run.WorkspaceObservation
	closed       []run.WorkspaceObservation
}

func (w *smokeWorkspaces) Materialize(ctx context.Context, req run.WorkspaceSpec) (run.WorkspaceObservation, error) {
	observation, err := w.Workspaces.Materialize(ctx, req)
	w.materialized = append(w.materialized, observation)
	return observation, err
}

func (w *smokeWorkspaces) Close(ctx context.Context, id run.WorkspaceID) (run.WorkspaceObservation, error) {
	observation, err := w.Workspaces.Close(ctx, id)
	w.closed = append(w.closed, observation)
	return observation, err
}
