package container_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// TestRealEngineRunsAHardenedSession drives the actual container engine. It
// builds markitect-agentd and the agentd test binary for Linux, packs both
// into a scratch image, and runs them under the production hardening: the
// test binary plays a CLI that starts `agentd mcp`-style bridges and calls a
// Orchestrator tool. Set MARKITECT_TEST_ENGINE to the engine's absolute path to run it.
func TestRealEngineRunsAHardenedSession(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image := buildTestImage(ctx, t, enginePath)

	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	labels := map[string]string{"io.markitect.store": store}
	work := "mk-" + store + "-work"
	if err := engine.CreateVolume(ctx, work, labels); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.RemoveVolume(context.Background(), work) })

	spec := container.ContainerSpec{
		Name: "mk-" + store, Image: image, Labels: labels, User: "1000:1000",
		Mounts:  []container.Mount{{Volume: work, Target: "/work", ReadOnly: true}},
		Workdir: "/work", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20,
		Network: "none",
	}
	s, err := engine.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	tool := frame.ToolSpec{Name: "submit_work", Description: "Submit.", InputSchema: json.RawMessage(`{"type":"object"}`)}
	started, err := s.StartProvider(ctx, frame.Start{
		Argv:  []string{"/usr/local/bin/agentd.test"},
		Dir:   "/work",
		Env:   []string{"AGENTD_TEST_PROVIDER=client", "AGENTD_TEST_SOCKET=/tmp/markitect-agentd.sock"},
		Tools: []frame.ToolSpec{tool},
	})
	if err != nil {
		t.Fatalf("start provider: %v; diagnostics %q", err, s.Diagnostics())
	}
	if started.PID <= 0 {
		t.Fatalf("started = %+v", started)
	}
	stdout := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(s.Stdout())
		stdout <- string(b)
	}()
	select {
	case call := <-s.ToolCalls():
		if call.Name != "submit_work" || string(call.Arguments) != `{"summary":"done"}` {
			t.Fatalf("tool call = %+v", call)
		}
		if err := s.Reply(frame.ToolReply{ID: call.ID, Result: json.RawMessage(`{"content":[{"type":"text","text":"accepted as D-1"}]}`)}); err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatalf("no tool call; diagnostics %q", s.Diagnostics())
	}
	if out := <-stdout; !strings.Contains(out, "accepted as D-1") {
		t.Fatalf("provider output %q; diagnostics %q", out, s.Diagnostics())
	}
	if exit := <-s.Exited(); exit.Code != 0 {
		t.Fatalf("exit = %+v; diagnostics %q", exit, s.Diagnostics())
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if names, err := engine.Containers(ctx, "io.markitect.store", store); err != nil || len(names) != 0 {
		t.Fatalf("containers after close = %v, %v", names, err)
	}
	marker, err := engine.ReadImageFile(ctx, image, "/opt/markitect/runtime.env")
	if err != nil || string(marker) != "agentd=test\n" {
		t.Fatalf("image metadata = %q, %v", marker, err)
	}
}

// TestRealEngineStopsAWritingSessionWhenTheOrchestratorLeaves shows that closing the
// connection ends the provider and removes the container.
func TestRealEngineStopsAWritingSessionWhenTheOrchestratorLeaves(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image := buildTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	spec := container.ContainerSpec{
		Name: "mk-" + store, Image: image, Labels: map[string]string{"io.markitect.store": store}, User: "1000:1000",
		Workdir: "/tmp", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
	}
	s, err := engine.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartProvider(ctx, frame.Start{Argv: []string{"/usr/local/bin/agentd.test"}, Dir: "/tmp", Env: []string{"AGENTD_TEST_PROVIDER=stubborn-tree"}}); err != nil {
		t.Fatal(err)
	}
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("container kept running after the Orchestrator left")
	}
	if names, err := engine.Containers(ctx, "io.markitect.store", store); err != nil || len(names) != 0 {
		t.Fatalf("containers after close = %v, %v", names, err)
	}
}

func buildTestImage(ctx context.Context, t *testing.T, enginePath string) string {
	t.Helper()
	return buildTestImageFrom(ctx, t, enginePath, "scratch", "")
}

// buildShellTestImage adds agentd to a small Linux base with a shell, for
// tests whose provider is a shell script.
func buildShellTestImage(ctx context.Context, t *testing.T, enginePath string) string {
	t.Helper()
	return buildTestImageFrom(ctx, t, enginePath,
		"ubuntu:24.04@sha256:008173c23f95b170204355c12626cb5a965d779a7e1283b09e9cffbb1bf33ca3",
		"RUN mkdir -p /work && chown 1000:1000 /work\n")
}

func buildTestImageFrom(ctx context.Context, t *testing.T, enginePath, base, extra string) string {
	t.Helper()
	arch, err := exec.CommandContext(ctx, enginePath, "version", "--format", "{{.Server.Arch}}").Output()
	if err != nil {
		t.Fatalf("engine architecture: %v", err)
	}
	gomod, err := exec.CommandContext(ctx, "go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(strings.TrimSpace(string(gomod)))
	dir := t.TempDir()
	env := append(os.Environ(), "GOOS=linux", "GOARCH="+strings.TrimSpace(string(arch)), "CGO_ENABLED=0")
	for _, args := range [][]string{
		{"build", "-o", filepath.Join(dir, "markitect-agentd"), "./cmd/markitect-agentd"},
		{"test", "-c", "-o", filepath.Join(dir, "agentd.test"), "./internal/agentd"},
	} {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir, cmd.Env = root, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "runtime.env"), []byte("agentd=test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dockerfile := "FROM " + base + "\n" + extra +
		"COPY markitect-agentd /usr/local/bin/markitect-agentd\n" +
		"COPY agentd.test /usr/local/bin/agentd.test\n" +
		"COPY runtime.env /opt/markitect/runtime.env\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.CommandContext(ctx, enginePath, "build", "--quiet", dir).Output()
	if err != nil {
		t.Fatalf("build test image: %v", err)
	}
	id := strings.TrimSpace(string(out))
	t.Cleanup(func() { _ = exec.Command(enginePath, "image", "rm", "--force", id).Run() })
	return id
}

// TestRealEngineNameConflictLeavesTheOtherContainer shows that a failed start
// never removes another container that already uses the name.
func TestRealEngineNameConflictLeavesTheOtherContainer(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image := buildTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	spec := container.ContainerSpec{
		Name: "mk-" + store, Image: image, Labels: map[string]string{"io.markitect.store": store}, User: "1000:1000",
		Workdir: "/tmp", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
	}
	first, err := engine.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if second, err := engine.Start(ctx, spec); err == nil {
		_ = second.Close()
		t.Fatal("a second container started under the same name")
	}
	if names, err := engine.Containers(ctx, "io.markitect.store", store); err != nil || len(names) != 1 {
		t.Fatalf("containers after the conflict = %v, %v; the first must survive", names, err)
	}
}

// TestRealEngineCensusSeesOnlyTheSession shows that the census inside a
// session container lists the provider and its tool processes, and neither
// agentd nor the container's init.
func TestRealEngineCensusSeesOnlyTheSession(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image := buildTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	s, err := engine.Start(ctx, container.ContainerSpec{
		Name: "mk-" + store, Image: image, Labels: map[string]string{"io.markitect.store": store}, User: "1000:1000",
		Workdir: "/tmp", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	started, err := s.StartProvider(ctx, frame.Start{Argv: []string{"/usr/local/bin/agentd.test"}, Dir: "/tmp", Env: []string{"AGENTD_TEST_PROVIDER=background"}})
	if err != nil {
		t.Fatal(err)
	}
	// Wait until the provider reports its started child.
	stdout := bufio.NewScanner(s.Stdout())
	if !stdout.Scan() || !strings.HasPrefix(stdout.Text(), "child ") {
		t.Fatalf("no child report; diagnostics %q", s.Diagnostics())
	}
	go func() { _, _ = io.Copy(io.Discard, s.Stdout()) }()
	processes, err := s.Census(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var provider, child bool
	for _, p := range processes {
		switch {
		case p.PID == started.PID:
			provider = true
		case p.PPID == started.PID:
			child = true
		case p.PID == 1 || strings.Contains(p.Command, "markitect-agent"):
			t.Fatalf("census lists the container's init or agentd: %+v", p)
		}
	}
	if !provider || !child || len(processes) != 2 {
		t.Fatalf("census = %+v, want exactly the provider and its child", processes)
	}
}

// TestRealEnginePauseIsAWriteBarrier freezes a session whose provider keeps
// writing to the workspace and shows that nothing changes until it resumes.
func TestRealEnginePauseIsAWriteBarrier(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
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
	s, err := engine.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _, _ = io.Copy(io.Discard, s.Stdout()) }()
	ticker := `i=0; while true; do i=$((i+1)); echo $i > /work/tick; sleep 0.05; done`
	if _, err := s.StartProvider(ctx, frame.Start{Argv: []string{"/bin/sh", "-c", ticker}, Dir: "/work"}); err != nil {
		t.Fatal(err)
	}
	helper := spec
	helper.Name = spec.Name + "-read"
	read := func() string {
		var out strings.Builder
		if err := engine.RunHelper(ctx, helper, []string{"/bin/cat", "/work/tick"}, nil, &out); err != nil {
			t.Fatalf("read tick: %v", err)
		}
		return strings.TrimSpace(out.String())
	}
	time.Sleep(300 * time.Millisecond)
	if err := engine.Pause(ctx, spec.Name); err != nil {
		t.Fatal(err)
	}
	// A deferred unpause runs before the deferred Close, so a failure never
	// closes a frozen session.
	defer func() { _ = engine.Unpause(context.Background(), spec.Name) }()
	frozen := read()
	time.Sleep(time.Second)
	if again := read(); again != frozen {
		t.Fatalf("the paused provider kept writing: %s then %s", frozen, again)
	}
	if err := engine.Unpause(ctx, spec.Name); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if resumed := read(); resumed == frozen {
		t.Fatalf("the provider did not resume after Unpause: still %s", resumed)
	}
}

// TestRealEngineEnforcesFileAndDirectoryOverlays shows on the actual engine the
// mount shapes ownership needs: a writable directory of a volume, a read-only
// file of another owner inside it and a read-only deeper directory, and a
// writable exact file inside a read-only directory. Writes through the
// read-only overlays fail, the file cannot be removed or renamed, and a file
// mount cannot create a missing file.
func TestRealEngineEnforcesFileAndDirectoryOverlays(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := engine.RequireVolumeSubpaths(ctx); err != nil {
		t.Fatal(err)
	}
	image := buildShellTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	labels := map[string]string{"io.markitect.store": store}
	work := "mk-" + store + "-work"
	if err := engine.CreateVolume(ctx, work, labels); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.RemoveVolume(context.Background(), work) })
	base := container.ContainerSpec{
		Name: "mk-" + store + "-seed", Image: image, Labels: labels, User: "1000:1000",
		Mounts:  []container.Mount{{Volume: work, Target: "/work"}},
		Workdir: "/work", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
	}
	run := func(spec container.ContainerSpec, script string) string {
		var out strings.Builder
		if err := engine.RunHelper(ctx, spec, []string{"/bin/sh", "-c", script}, nil, &out); err != nil {
			t.Fatalf("helper: %v\n%s", err, out.String())
		}
		return out.String()
	}
	run(base, "mkdir -p /work/a/b && echo owned > /work/a/x.txt && echo dir > /work/a/b/f.txt")

	writer := base
	writer.Name = "mk-" + store + "-writer"
	writer.Mounts = []container.Mount{
		{Volume: work, Target: "/work", ReadOnly: true, NoCopy: true},
		{Volume: work, Target: "/work/a", Subpath: "a", NoCopy: true},
		{Volume: work, Target: "/work/a/b", Subpath: "a/b", ReadOnly: true, NoCopy: true},
		{Volume: work, Target: "/work/a/x.txt", Subpath: "a/x.txt", ReadOnly: true, NoCopy: true},
	}
	got := run(writer, `
try() { if sh -c "$2" 2>/dev/null; then echo "$1 allowed"; else echo "$1 refused"; fi; }
try create-in-own "echo n > /work/a/new.txt"
try write-foreign-file "echo x > /work/a/x.txt"
try remove-foreign-file "rm /work/a/x.txt"
try rename-foreign-file "mv /work/a/x.txt /work/a/y.txt"
try write-deeper "echo x > /work/a/b/f.txt"
try create-deeper "touch /work/a/b/g.txt"
try write-base "touch /work/top.txt"
`)
	for _, want := range []string{"create-in-own allowed", "write-foreign-file refused", "remove-foreign-file refused", "rename-foreign-file refused", "write-deeper refused", "create-deeper refused", "write-base refused"} {
		if !strings.Contains(got, want) {
			t.Errorf("writer: want %q in\n%s", want, got)
		}
	}

	owner := base
	owner.Name = "mk-" + store + "-owner"
	owner.Mounts = []container.Mount{
		{Volume: work, Target: "/work", ReadOnly: true, NoCopy: true},
		{Volume: work, Target: "/work/a/x.txt", Subpath: "a/x.txt", NoCopy: true},
	}
	got = run(owner, `
try() { if sh -c "$2" 2>/dev/null; then echo "$1 allowed"; else echo "$1 refused"; fi; }
try write-own-file "echo changed > /work/a/x.txt"
try create-beside "echo n > /work/a/z.txt"
`)
	for _, want := range []string{"write-own-file allowed", "create-beside refused"} {
		if !strings.Contains(got, want) {
			t.Errorf("exact-file owner: want %q in\n%s", want, got)
		}
	}
	final := run(base, "cat /work/a/x.txt; ls /work/a")
	if !strings.Contains(final, "changed") || strings.Contains(final, "y.txt") || strings.Contains(final, "z.txt") {
		t.Errorf("final state:\n%s", final)
	}
}

// Every turn runs in a container of its own. Consecutive turns of one
// conversation get different containers but mount the same provider home, so
// the permitted conversation state continues. A writer that escaped the first
// turn's session ends with that container, the ended container no longer
// mounts the shared tree, which a capture requires, and nothing is left.
func TestRealEngineGivesEveryTurnAFreshContainerWithItsHome(t *testing.T) {
	enginePath := os.Getenv("MARKITECT_TEST_ENGINE")
	if enginePath == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image := buildShellTestImage(ctx, t, enginePath)
	store := fmt.Sprintf("test-%d", time.Now().UnixNano())
	labels := map[string]string{"io.markitect.store": store}
	work, home := "mk-"+store+"-work", "mk-"+store+"-home"
	for _, v := range []string{work, home} {
		if err := engine.CreateVolume(ctx, v, labels); err != nil {
			t.Fatal(err)
		}
		v := v
		t.Cleanup(func() { _ = engine.RemoveVolume(context.Background(), v) })
	}
	turn := func(name, script string) string {
		t.Helper()
		s, err := engine.Start(ctx, container.ContainerSpec{
			Name: "mk-" + store + "-" + name, Image: image, Labels: labels, User: "1000:1000",
			Mounts:  []container.Mount{{Volume: work, Target: "/work"}, {Volume: home, Target: "/home/agent"}},
			Workdir: "/work", CPUs: 1, MemoryBytes: 256 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.StartProvider(ctx, frame.Start{Argv: []string{"/bin/sh", "-c", script}, Dir: "/work"}); err != nil {
			t.Fatalf("start %s: %v %q", name, err, s.Diagnostics())
		}
		out, _ := io.ReadAll(s.Stdout())
		select {
		case <-s.Exited():
		case <-time.After(30 * time.Second):
			t.Fatalf("turn %s did not exit", name)
		}
		// The container's own end is the observed end of every process of the turn.
		if err := s.Close(); err != nil {
			t.Fatalf("turn %s: the container did not end by itself: %v", name, err)
		}
		return strings.TrimSpace(string(out))
	}
	first := turn("t1", "echo conversation-1 > /home/agent/conversation && echo a > /work/a && "+
		"setsid sh -c 'while true; do echo x >> /work/bg; sleep 0.1; done' > /dev/null 2>&1 < /dev/null & "+
		"while [ ! -s /work/bg ]; do sleep 0.05; done; hostname")
	if mounts, err := engine.VolumeMounts(ctx, work); err != nil || len(mounts) != 0 {
		t.Fatalf("the ended turn still mounts the shared tree: %v %v", mounts, err)
	}
	got := strings.Split(turn("t2", `cat /home/agent/conversation; cat /work/a; a=$(wc -c < /work/bg); sleep 1; b=$(wc -c < /work/bg); if [ "$a" = "$b" ]; then echo stopped; else echo running; fi; hostname`), "\n")
	if len(got) != 4 || got[0] != "conversation-1" || got[1] != "a" || got[2] != "stopped" || got[3] == first || first == "" {
		t.Fatalf("second turn saw %q after %q", got, first)
	}
	if names, err := engine.Containers(ctx, "io.markitect.store", store); err != nil || len(names) != 0 {
		t.Fatalf("containers after the turns = %v, %v", names, err)
	}
}
