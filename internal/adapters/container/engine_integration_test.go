package container_test

import (
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
// Host tool. Set MARKITECT_TEST_ENGINE to the engine's absolute path to run it.
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

	spec := container.SessionSpec{
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

// TestRealEngineStopsAWritingSessionWhenTheHostLeaves shows that closing the
// connection ends the provider and removes the container.
func TestRealEngineStopsAWritingSessionWhenTheHostLeaves(t *testing.T) {
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
	spec := container.SessionSpec{
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
		t.Fatal("container kept running after the Host left")
	}
	if names, err := engine.Containers(ctx, "io.markitect.store", store); err != nil || len(names) != 0 {
		t.Fatalf("containers after close = %v, %v", names, err)
	}
}

func buildTestImage(ctx context.Context, t *testing.T, enginePath string) string {
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
	dockerfile := "FROM scratch\n" +
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
	spec := container.SessionSpec{
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
