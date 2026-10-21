package container_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// heldFixture is a held instance of the shell test image on a fresh writable
// work volume, under the production hardening.
type heldFixture struct {
	t      *testing.T
	ctx    context.Context
	engine *container.Engine
	path   string
	spec   container.ContainerSpec
}

func newHeldFixture(t *testing.T) *heldFixture {
	t.Helper()
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)
	image := buildShellTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	labels := map[string]string{"io.markitect.store": store}
	work := "mk-" + store + "-work"
	if err := engine.CreateVolume(ctx, work, labels); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.RemoveVolume(context.Background(), work) })
	spec := container.ContainerSpec{
		Name: "mk-" + store, Image: image, Labels: labels, User: "1000:1000",
		Mounts:  []container.Mount{{Volume: work, Target: "/work"}},
		Workdir: "/work", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
	}
	return &heldFixture{t: t, ctx: ctx, engine: engine, path: enginePath, spec: spec}
}

func (f *heldFixture) hold(name string, readOnly bool, idle time.Duration) *container.Instance {
	f.t.Helper()
	spec := f.spec
	spec.Name = f.spec.Name + "-" + name
	spec.Mounts = []container.Mount{{Volume: f.spec.Mounts[0].Volume, Target: "/work", ReadOnly: readOnly}}
	inst, err := f.engine.Hold(f.ctx, spec, idle)
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = inst.Remove(context.Background()) })
	return inst
}

// sh runs one shell script as the provider of a fresh agentd session in the
// instance and returns its output.
func (f *heldFixture) sh(inst *container.Instance, script string) string {
	f.t.Helper()
	s, err := inst.Exec(f.ctx)
	if err != nil {
		f.t.Fatalf("exec: %v", err)
	}
	defer s.Close()
	if _, err := s.StartProvider(f.ctx, frame.Start{Argv: []string{"/bin/sh", "-c", script}, Dir: "/work"}); err != nil {
		f.t.Fatalf("start %q: %v %q", script, err, s.Diagnostics())
	}
	out, _ := io.ReadAll(s.Stdout())
	select {
	case <-s.Exited():
	case <-time.After(30 * time.Second):
		f.t.Fatalf("%q did not exit", script)
	}
	return strings.TrimSpace(string(out))
}

func (f *heldFixture) exists(inst *container.Instance) bool {
	return exec.Command(f.path, "container", "inspect", inst.Name()).Run() == nil
}

// One held instance carries several agentd sessions. A writer that leaves its
// session's process group survives the session's end, and only the reap stops
// it; the holder runs under its own user, which the Agent cannot signal.
func TestRealEngineReusesAHeldInstanceAndReapsWritersThatEscapedTheirSession(t *testing.T) {
	f := newHeldFixture(t)
	inst := f.hold("reuse", false, 2*time.Minute)
	if got := f.sh(inst, "echo first > /work/a && cat /work/a"); got != "first" {
		t.Fatalf("first session: %q", got)
	}
	if got := f.sh(inst, "cat /work/a; stat -c %u /proc/1; id -u"); got != "first\n65534\n1000" {
		t.Fatalf("second session in the same instance: %q", got)
	}
	// A background writer in its own session survives the provider's group,
	// which agentd ends with the session; the session waits until the writer
	// has escaped and written.
	f.sh(inst, "setsid sh -c 'while true; do echo x >> /work/bg; sleep 0.1; done' > /dev/null 2>&1 < /dev/null & while [ ! -s /work/bg ]; do sleep 0.05; done; echo started")
	size := func() int {
		n, _ := strconv.Atoi(f.sh(inst, "wc -c < /work/bg"))
		return n
	}
	before := size()
	time.Sleep(time.Second)
	if after := size(); after <= before {
		t.Fatalf("the escaped writer is not running: %d then %d bytes", before, after)
	}
	remaining, err := inst.Reap(f.ctx)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("reap: %+v %v", remaining, err)
	}
	stopped := size()
	time.Sleep(time.Second)
	if again := size(); again != stopped {
		t.Fatalf("a writer survived the reap: %d then %d bytes", stopped, again)
	}
	// The Agent cannot end the holder; the instance stays usable.
	f.sh(inst, "kill -TERM 1; kill -KILL 1; kill -HUP 1; true")
	if got := f.sh(inst, "echo alive"); got != "alive" {
		t.Fatalf("after signals to init: %q", got)
	}
	if err := inst.Remove(f.ctx); err != nil || f.exists(inst) {
		t.Fatalf("remove: %v, exists %v", err, f.exists(inst))
	}
}

// A read-only view of the same volume refuses every write, and a held
// instance without any session ends by itself after its idle bound.
func TestRealEngineHeldReadOnlyViewsAndIdleEnd(t *testing.T) {
	f := newHeldFixture(t)
	write := f.hold("write", false, 2*time.Minute)
	f.sh(write, "echo kept > /work/k")
	read := f.hold("read", true, 2*time.Minute)
	if got := f.sh(read, "cat /work/k; echo y > /work/k 2>/dev/null && echo wrote; touch /work/new 2>/dev/null && echo created; cat /work/k"); got != "kept\nkept" {
		t.Fatalf("read-only view: %q", got)
	}
	idle := f.hold("idle", true, 2*time.Second)
	deadline := time.Now().Add(45 * time.Second)
	for f.exists(idle) {
		if time.Now().After(deadline) {
			t.Fatal("an idle held instance did not end by itself")
		}
		time.Sleep(time.Second)
	}
}
