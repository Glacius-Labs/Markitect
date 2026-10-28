package agents_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/adapters/remotecontainer"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agents"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// scriptedTurn is a provider stand-in that runs in the real Manager image:
// a Python MCP client of markitect-agentd that edits files and calls the
// Manager tools as its steps say, and prints each tool result.
const scriptedTurn = `
import json, os, subprocess, sys
sys.stdin.read()
bridge = subprocess.Popen(["/usr/local/bin/markitect-agentd", "mcp"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
counter = 0
def rpc(method, params, notify=False):
    global counter
    message = {"jsonrpc": "2.0", "method": method, "params": params}
    if not notify:
        counter += 1
        message["id"] = counter
    bridge.stdin.write(json.dumps(message) + "\n")
    bridge.stdin.flush()
    while not notify:
        reply = json.loads(bridge.stdout.readline())
        if reply.get("id") == counter:
            return reply
rpc("initialize", {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "scripted", "version": "1"}})
rpc("notifications/initialized", {}, notify=True)
for step in json.loads(os.environ["SCRIPTED_STEPS"]):
    if "write" in step:
        path = os.path.join("/work", step["write"])
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w") as f:
            f.write(step["text"])
    else:
        reply = rpc("tools/call", {"name": step["call"], "arguments": step["args"]})
        print(json.dumps({"type": "tool", "reply": reply}), flush=True)
print(json.dumps({"type": "done"}), flush=True)
bridge.stdin.close()
bridge.wait()
`

type step map[string]any

// scripted is the provider stand-in: its login is a dummy, its login report
// a shell echo, and each turn runs the next steps.
type scripted struct {
	mu      sync.Mutex
	steps   [][]step
	results []string
}

func (*scripted) Name() string { return "scripted" }
func (*scripted) Login([]byte, time.Time) (agents.Login, error) {
	return agents.Login{Copy: []byte(`{"scripted":true}`)}, nil
}
func (*scripted) Home(l agents.Login, _ harness.Document) (frame.Initialize, error) {
	return frame.Initialize{CodexAuth: l.Copy}, nil
}
func (*scripted) AuthStatus() ([]string, []string) { return []string{"/bin/sh", "-c", "echo ok"}, nil }
func (*scripted) CheckAuth(out []byte) error {
	if strings.TrimSpace(string(out)) != "ok" {
		return errors.New("no login")
	}
	return nil
}
func (s *scripted) Turn(agents.Turn) ([]string, []string, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	steps, _ := json.Marshal(s.steps[0])
	s.steps = s.steps[1:]
	return []string{"/usr/bin/python3", "-c", scriptedTurn}, []string{"SCRIPTED_STEPS=" + string(steps)}, []byte("task"), nil
}
func (s *scripted) Reader(agents.Turn) agents.Reader { return &scriptedReader{p: s} }

type scriptedReader struct {
	p    *scripted
	done bool
}

func (r *scriptedReader) Line(raw []byte) error {
	var l struct {
		Type  string `json:"type"`
		Reply struct {
			Result struct {
				Content []struct{ Text string } `json:"content"`
				IsError bool                    `json:"isError"`
			} `json:"result"`
		} `json:"reply"`
	}
	if err := json.Unmarshal(raw, &l); err != nil {
		return err
	}
	switch l.Type {
	case "tool":
		text := "ok: "
		if l.Reply.Result.IsError {
			text = "refused: "
		}
		for _, c := range l.Reply.Result.Content {
			text += c.Text
		}
		r.p.mu.Lock()
		r.p.results = append(r.p.results, text)
		r.p.mu.Unlock()
	case "done":
		r.done = true
	}
	return nil
}
func (r *scriptedReader) Outcome() agents.Outcome {
	if r.done {
		return agents.Outcome{End: agents.Answered}
	}
	return agents.Outcome{}
}

// TestRealEngineManagersWorkThroughTheRunRemote starts a real Run remote
// with its hooks and real Manager containers of the Manager image, whose
// scripted turns call the Manager tools over agentd's MCP bridge: a maker's
// push outside its area is refused and its work kept, its submission passes
// the hook and the required check, and its checker approves it with an
// empty commit. Nothing of the Run remains.
//
//	MARKITECT_TEST_ENGINE         absolute path of the Docker CLI
//	MARKITECT_TEST_MANAGER_IMAGE  the Manager image (packaging/runtime)
func TestRealEngineManagersWorkThroughTheRunRemote(t *testing.T) {
	enginePath, image := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_MANAGER_IMAGE")
	if enginePath == "" || image == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_MANAGER_IMAGE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	remoteEngine, err := remotecontainer.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	imageID, err := engine.ImageID(ctx, image)
	if err != nil {
		t.Fatal(err)
	}
	arch, err := remoteEngine.Arch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "markitect-remote")
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", binary, "github.com/Glacius-Labs/Markitect/cmd/markitect-remote")
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build markitect-remote: %v\n%s", err, out)
	}
	run := fmt.Sprintf("t4b-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		if err := remotecontainer.Sweep(context.Background(), remoteEngine, run); err != nil {
			t.Errorf("sweep: %v", err)
		}
		for _, v := range must(engine.Volumes(context.Background(), remotecontainer.RunLabel, run)) {
			t.Errorf("volume %s remained", v)
			_ = engine.RemoveVolume(context.Background(), v)
		}
	})

	_, repo := library(t)
	source := filepath.Join(t.TempDir(), "source")
	for f, data := range repo {
		path := filepath.Join(source, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, source, "init", "-q")
	git(t, source, "add", "-A")
	git(t, source, "commit", "-q", "-m", "base")
	base := git(t, source, "rev-parse", "HEAD")

	token := func(user string) string { return "token-" + user + "-0123456789" }
	plan := &runremote.Plan{Run: run, Mode: runremote.Apply, Base: base,
		Checks: []project.Check{{Name: "tree", Command: "test -f src/members/member.go", Timeout: time.Minute, Environment: imageID}},
		Managers: []runremote.Manager{
			{ID: "", Working: true},
			{ID: "members", Patterns: []string{"src/members/**"}, Working: true, Checklist: []string{"members/member"}},
			{ID: "members/cards", Patterns: []string{"src/members/cards/**"}},
		},
		Pages: map[string]string{project.RootPage: "", project.RegistryDir + "/members.md": "members", project.RegistryDir + "/members/cards.md": "members/cards"}}
	for _, a := range []struct {
		user, manager string
		role          runremote.Role
	}{{"root-maker", "", runremote.Maker}, {"root-checker", "", runremote.Checker}, {"members-maker", "members", runremote.Maker}, {"members-checker", "members", runremote.Checker}} {
		sum := sha256.Sum256([]byte(token(a.user)))
		plan.Agents = append(plan.Agents, runremote.Agent{User: a.user, Manager: a.manager, Role: a.role, TokenSHA256: hex.EncodeToString(sum[:])})
	}
	var bundle bytes.Buffer
	if err := remotecontainer.StartBundle(ctx, "git", source, base, &bundle); err != nil {
		t.Fatal(err)
	}
	remote, err := remotecontainer.Start(ctx, remoteEngine, remotecontainer.Spec{Image: imageID, Binary: binary, Plan: plan, Limits: remotecontainer.Limits{CPUs: "1", Memory: "1g", Pids: 512}}, &bundle)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close(context.Background())

	loginFile := filepath.Join(t.TempDir(), "login.json")
	if err := os.WriteFile(loginFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcripts := t.TempDir()
	run1 := func(role agentport.Role, provider *scripted) agentport.End {
		t.Helper()
		g, err := agents.New(agents.Config{Runtime: agents.EngineRuntime(engine), Provider: provider, Image: imageID,
			Limits: agents.Limits{CPUs: 1, MemoryBytes: 1 << 30, Pids: 256, TmpBytes: 128 << 20}, LoginFile: loginFile,
			Model: "scripted", Effort: "high", Transcripts: transcripts})
		if err != nil {
			t.Fatal(err)
		}
		user := "members-" + string(role)
		return g.Run(ctx, agentport.Start{Run: run, Mode: agentport.Apply, Role: role, Round: 1,
			Manager:   agentport.Manager{ID: "members", Parent: "."},
			Checklist: []agentport.Item{{Key: "members/member", Part: agentport.Own, Definition: "members/member", Why: "changed"}},
			Git:       agentport.GitAccess{URL: remote.URL, Network: remote.Network, Branch: "managers/members", Base: base, User: user, Token: token(user)},
			Tools:     []agentport.Tool{agentport.ToolNote}, Deadline: time.Now().Add(5 * time.Minute)}, &host{})
	}
	verdict := []map[string]string{{"item": "members/member", "verdict": "changed", "evidence": "type Member in src/members/member.go"}}
	maker := &scripted{steps: [][]step{{
		{"write": "README.md", "text": "# Changed by members\n"},
		{"call": "submit", "args": map[string]any{"summary": "Outside the area", "verdicts": verdict}},
		{"write": "README.md", "text": "# Library\n"},
		{"write": "src/members/member.go", "text": "package members\n\ntype Member struct{}\n"},
		{"call": "submit", "args": map[string]any{"summary": "Realize the member", "verdicts": verdict}},
	}}}
	if end := run1(agentport.Maker, maker); end.Reason != agentport.Finished || end.Cleanup != "" || end.Agent != "scripted:scripted" {
		t.Fatalf("maker end %+v", end)
	}
	if len(maker.results) != 2 || !strings.HasPrefix(maker.results[0], "refused: the remote refused the push") || !strings.HasPrefix(maker.results[1], "ok: submitted ") {
		t.Fatalf("maker results %q", maker.results)
	}
	checker := &scripted{steps: [][]step{{
		{"call": "review", "args": map[string]any{"outcome": "approved", "checks": []map[string]string{{"item": "members/member", "answer": "confirmed", "evidence": "read src/members/member.go"}}}},
	}}}
	if end := run1(agentport.Checker, checker); end.Reason != agentport.Finished || end.Cleanup != "" {
		t.Fatalf("checker end %+v", end)
	}
	if len(checker.results) != 1 || !strings.HasPrefix(checker.results[0], "ok: approved ") {
		t.Fatalf("checker results %q", checker.results)
	}
	for _, name := range []string{"members-maker-1.jsonl", "members-checker-1.jsonl"} {
		data, err := os.ReadFile(filepath.Join(transcripts, run, name))
		if err != nil || !bytes.Contains(data, []byte(`"kind":"tool-call"`)) || bytes.Contains(data, []byte(token("members-maker"))) {
			t.Fatalf("transcript %s: %v", name, err)
		}
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
