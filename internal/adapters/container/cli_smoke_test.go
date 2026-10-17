package container_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// The CLI smoke repeats for every runtime image: both provider CLIs start in
// the hardened container and connect to the Host tools through
// `markitect-agentd mcp`. With a runtime credential the smoke also lets each
// CLI change a file and call a Host tool.
//
//	MARKITECT_TEST_ENGINE         absolute container engine path
//	MARKITECT_TEST_RUNTIME_IMAGE  runtime image reference, for example markitect-runtime:dev
//	OPENAI_API_KEY or MARKITECT_TEST_CODEX_AUTH_VOLUME  optional Codex credential: an API key,
//	                              or a dedicated runtime login volume with auth.json at its
//	                              root, written by uid 1000
//	ANTHROPIC_API_KEY or CLAUDE_CODE_OAUTH_TOKEN  optional Claude Code credential

const submitPrompt = `Create the file hello.txt containing the single line "hello from markitect". ` +
	`Then call the submit_work tool with a one-sentence summary. After the tool answers, reply with the delivery id it returned.`

var hostTool = frame.ToolSpec{
	Name:        "submit_work",
	Description: "Submit the finished delivery to the Markitect Host. Call exactly once when done.",
	InputSchema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`),
}

type runtimeSmoke struct {
	t      *testing.T
	ctx    context.Context
	engine *container.Engine
	image  string
	store  string
}

func newRuntimeSmoke(t *testing.T) *runtimeSmoke {
	enginePath, ref := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_RUNTIME_IMAGE")
	if enginePath == "" || ref == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_RUNTIME_IMAGE")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	image, err := engine.ImageID(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	return &runtimeSmoke{t: t, ctx: ctx, engine: engine, image: image, store: fmt.Sprintf("smoke-%d", time.Now().UnixNano())}
}

// volumes creates fresh provider-home and workspace volumes for one session.
func (r *runtimeSmoke) volumes(name string) []container.Mount {
	var mounts []container.Mount
	for _, m := range []struct{ suffix, target string }{{"home", "/home/agent"}, {"work", "/work"}} {
		volume := "mk-" + r.store + "-" + name + "-" + m.suffix
		if err := r.engine.CreateVolume(r.ctx, volume, r.labels()); err != nil {
			r.t.Fatal(err)
		}
		r.t.Cleanup(func() { _ = r.engine.RemoveVolume(context.Background(), volume) })
		mounts = append(mounts, container.Mount{Volume: volume, Target: m.target})
	}
	return mounts
}

func (r *runtimeSmoke) labels() map[string]string {
	return map[string]string{"io.markitect.store": r.store}
}

func (r *runtimeSmoke) spec(name string, mounts []container.Mount) container.ContainerSpec {
	return container.ContainerSpec{
		Name: "mk-" + r.store + "-" + name, Image: r.image, Labels: r.labels(), User: "1000:1000",
		Mounts: mounts, Workdir: "/work", CPUs: 2, MemoryBytes: 2 << 30, Pids: 256, TmpBytes: 256 << 20,
	}
}

// start runs a session container on the given volumes.
func (r *runtimeSmoke) start(name string, mounts []container.Mount) *container.Session {
	s, err := r.engine.Start(r.ctx, r.spec(name, mounts))
	if err != nil {
		r.t.Fatal(err)
	}
	r.t.Cleanup(func() { _ = s.Close() })
	return s
}

// session starts a container with fresh provider-home and workspace volumes.
func (r *runtimeSmoke) session(name string) *container.Session {
	return r.start(name, r.volumes(name))
}

// readFile reads a workspace file through a helper container.
func (r *runtimeSmoke) readFile(name string, mounts []container.Mount, file string) string {
	var out strings.Builder
	if err := r.engine.RunHelper(r.ctx, r.spec(name, mounts), []string{"/bin/cat", file}, nil, &out); err != nil {
		r.t.Fatalf("read %s: %v", file, err)
	}
	return out.String()
}

// writeFile replaces a workspace file through a helper container.
func (r *runtimeSmoke) writeFile(name string, mounts []container.Mount, file, content string) {
	if err := r.engine.RunHelper(r.ctx, r.spec(name, mounts), []string{"/usr/bin/tee", file}, strings.NewReader(content), io.Discard); err != nil {
		r.t.Fatalf("write %s: %v", file, err)
	}
}

// answerTools replies to every Host tool call and counts them.
func answerTools(s *container.Session, calls chan<- frame.ToolCall) {
	for call := range s.ToolCalls() {
		_ = s.Reply(frame.ToolReply{ID: call.ID, Result: json.RawMessage(`{"content":[{"type":"text","text":"Host accepted the delivery as D-1."}]}`)})
		calls <- call
	}
}

func credentialEnv(names ...string) []string {
	var env []string
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			env = append(env, n+"="+v)
		}
	}
	return env
}

// The owner-selected models for real probes: Codex Luna and Claude Haiku,
// both with high effort.
const (
	codexModel  = "gpt-6-luna"
	codexEffort = "high"
	claudeModel = "claude-haiku-5-5"
)

// codexCredential prepares a session's provider home for a real Codex turn.
// An API key travels in the start message. Otherwise a dedicated runtime
// login volume, never the desktop login, supplies a copy of its login without
// the refresh token: the session cannot refresh or rotate anything, and the
// access token outlives a run. A helper writes it into the provider home.
func (r *runtimeSmoke) codexCredential(name string, mounts []container.Mount) ([]string, bool) {
	if env := credentialEnv("OPENAI_API_KEY"); len(env) > 0 {
		return env, true
	}
	auth := os.Getenv("MARKITECT_TEST_CODEX_AUTH_VOLUME")
	if auth == "" {
		return nil, false
	}
	var raw strings.Builder
	read := r.spec(name+"-auth-read", []container.Mount{{Volume: auth, Target: "/auth", ReadOnly: true}})
	if err := r.engine.RunHelper(r.ctx, read, []string{"/bin/cat", "/auth/auth.json"}, nil, &raw); err != nil {
		r.t.Fatalf("read the runtime login: %v", err)
	}
	var login map[string]any
	if err := json.Unmarshal([]byte(raw.String()), &login); err != nil {
		r.t.Fatal("the runtime login is not valid JSON")
	}
	tokens, ok := login["tokens"].(map[string]any)
	if !ok || tokens["access_token"] == nil {
		r.t.Fatal("the runtime login has no access token")
	}
	tokens["refresh_token"] = ""
	session, _ := json.Marshal(login)
	issue := r.spec(name+"-auth-issue", mounts)
	write := []string{"/bin/sh", "-c", "mkdir -p /home/agent/.codex && umask 077 && cat > /home/agent/.codex/auth.json"}
	if err := r.engine.RunHelper(r.ctx, issue, write, strings.NewReader(string(session)), io.Discard); err != nil {
		r.t.Fatal("writing the session login into the provider home failed")
	}
	return nil, true
}

func codexArgs() []string {
	return []string{"codex", "app-server",
		"--disable", "apps", "--disable", "memories", "--disable", "multi_agent",
		"--disable", "image_generation", "--disable", "browser_use", "--disable", "computer_use",
		"-c", `mcp_servers.markitect.command="` + container.AgentdPath + `"`,
		"-c", `mcp_servers.markitect.args=["mcp"]`}
}

// startCodex starts the App Server, answers Host tools and initializes it.
func startCodex(t *testing.T, r *runtimeSmoke, s *container.Session, env []string, calls chan frame.ToolCall) *jsonRPC {
	t.Helper()
	if _, err := s.StartProvider(r.ctx, frame.Start{Argv: codexArgs(), Dir: "/work", Env: env, Tools: []frame.ToolSpec{hostTool}}); err != nil {
		t.Fatal(err)
	}
	go answerTools(s, calls)
	rpc := newJSONRPC(t, s)
	rpc.call("initialize", map[string]any{"clientInfo": map[string]any{"name": "markitect-smoke", "version": "0"}})
	rpc.notify("initialized")
	return rpc
}

// codexTurn runs one turn on the owner-selected model and returns the
// agent's text. A refused start or a turn that does not complete fails the
// test with the provider's error instead of a later, misleading assertion.
func codexTurn(rpc *jsonRPC, threadID any, prompt string) string {
	rpc.t.Helper()
	started := rpc.call("turn/start", map[string]any{"threadId": threadID, "model": codexModel, "effort": codexEffort,
		"input": []any{map[string]any{"type": "text", "text": prompt}}})
	if started["error"] != nil {
		rpc.t.Fatalf("turn/start: %v; diagnostics %q", started["error"], rpc.s.Diagnostics())
	}
	var text strings.Builder
	done := rpc.until(func(m map[string]any) bool {
		if m["method"] == "item/completed" {
			if p, _ := m["params"].(map[string]any); p != nil {
				if item, _ := p["item"].(map[string]any); item != nil && item["type"] == "agentMessage" {
					s, _ := item["text"].(string)
					text.WriteString(s)
				}
			}
		}
		return m["method"] == "turn/completed"
	})
	params, _ := done["params"].(map[string]any)
	turn, _ := params["turn"].(map[string]any)
	if turn["status"] != "completed" {
		rpc.t.Fatalf("turn ended %v: %v", turn["status"], turn["error"])
	}
	return text.String()
}

func TestRuntimeImageCodexReachesHostTools(t *testing.T) {
	r := newRuntimeSmoke(t)
	mounts := r.volumes("codex")
	env, credentialed := r.codexCredential("codex", mounts)
	s := r.start("codex", mounts)
	calls := make(chan frame.ToolCall, 4)
	rpc := startCodex(t, r, s, env, calls)
	thread := rpc.call("thread/start", map[string]any{"cwd": "/work", "approvalPolicy": "never", "sandbox": "danger-full-access", "ephemeral": true, "model": codexModel})
	if thread["error"] != nil {
		t.Fatalf("thread/start: %v; diagnostics %q", thread["error"], s.Diagnostics())
	}
	rpc.until(func(m map[string]any) bool {
		p, _ := m["params"].(map[string]any)
		if m["method"] != "mcpServer/startupStatus/updated" || p["name"] != "markitect" {
			return false
		}
		if p["status"] == "failed" {
			t.Fatalf("Host tool server failed: %v; diagnostics %q", p, s.Diagnostics())
		}
		return p["status"] == "ready"
	})
	if !credentialed {
		t.Log("Host tool server ready; no Codex credential, so no turn ran")
		return
	}
	threadID := thread["result"].(map[string]any)["thread"].(map[string]any)["id"]
	codexTurn(rpc, threadID, submitPrompt)
	expectToolCall(t, calls)
	if got := r.readFile("codex-check", mounts, "/work/hello.txt"); strings.TrimSpace(got) != "hello from markitect" {
		t.Fatalf("hello.txt = %q", got)
	}
}

func TestRuntimeImageClaudeCodeReachesHostTools(t *testing.T) {
	r := newRuntimeSmoke(t)
	mounts := r.volumes("claude")
	s := r.start("claude", mounts)
	credentials := credentialEnv("ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN")
	mcp := `{"mcpServers":{"markitect":{"type":"stdio","command":"` + container.AgentdPath + `","args":["mcp"]}}}`
	if _, err := s.StartProvider(r.ctx, frame.Start{
		Argv: []string{"claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
			"--strict-mcp-config", "--mcp-config", mcp, "--tools", "Read,Write",
			"--allowedTools", "mcp__markitect__submit_work", "Read", "Write",
			"--permission-mode", "dontAsk", "--no-session-persistence", "--model", claudeModel, "--effort", "high"},
		Dir: "/work", Env: credentials, Tools: []frame.ToolSpec{hostTool},
	}); err != nil {
		t.Fatal(err)
	}
	calls := make(chan frame.ToolCall, 4)
	go answerTools(s, calls)
	message, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": submitPrompt}})
	if err := s.WriteInput(append(message, '\n')); err != nil {
		t.Fatal(err)
	}
	events := bufio.NewScanner(s.Stdout())
	events.Buffer(make([]byte, 64<<10), frame.MaxPayload)
	var connected bool
	for events.Scan() {
		var e map[string]any
		if json.Unmarshal(events.Bytes(), &e) != nil {
			continue
		}
		if e["type"] == "system" && e["subtype"] == "init" {
			servers, _ := json.Marshal(e["mcp_servers"])
			var list []struct{ Name, Status string }
			_ = json.Unmarshal(servers, &list)
			connected = len(list) == 1 && list[0].Name == "markitect" && list[0].Status == "connected"
			if !connected {
				t.Fatalf("Host tool server not connected: %s; diagnostics %q", servers, s.Diagnostics())
			}
			if len(credentials) == 0 {
				t.Log("Host tool server connected; no Claude credential, so no turn ran")
				return
			}
		}
		if e["type"] == "result" {
			if e["is_error"] == true {
				t.Fatalf("Claude Code result: %v", e["result"])
			}
			break
		}
	}
	if !connected {
		t.Fatalf("no init event; diagnostics %q", s.Diagnostics())
	}
	expectToolCall(t, calls)
	if got := r.readFile("claude-check", mounts, "/work/hello.txt"); strings.TrimSpace(got) != "hello from markitect" {
		t.Fatalf("hello.txt = %q", got)
	}
}

func expectToolCall(t *testing.T, calls <-chan frame.ToolCall) {
	t.Helper()
	select {
	case call := <-calls:
		if call.Name != "submit_work" {
			t.Fatalf("tool call = %+v", call)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the provider never called the Host tool")
	}
}

// jsonRPC is the smallest App Server client: requests, notifications and a
// scan of incoming messages.
type jsonRPC struct {
	t    *testing.T
	s    *container.Session
	in   *bufio.Scanner
	next int
}

func newJSONRPC(t *testing.T, s *container.Session) *jsonRPC {
	in := bufio.NewScanner(s.Stdout())
	in.Buffer(make([]byte, 64<<10), frame.MaxPayload)
	return &jsonRPC{t: t, s: s, in: in}
}

func (c *jsonRPC) send(m map[string]any) {
	m["jsonrpc"] = "2.0"
	b, _ := json.Marshal(m)
	if err := c.s.WriteInput(append(b, '\n')); err != nil {
		c.t.Fatal(err)
	}
}

func (c *jsonRPC) notify(method string) { c.send(map[string]any{"method": method}) }

func (c *jsonRPC) call(method string, params any) map[string]any {
	c.next++
	id := float64(c.next)
	c.send(map[string]any{"id": c.next, "method": method, "params": params})
	return c.until(func(m map[string]any) bool { return m["id"] == id && m["method"] == nil })
}

func (c *jsonRPC) until(match func(map[string]any) bool) map[string]any {
	for c.in.Scan() {
		var m map[string]any
		if json.Unmarshal(c.in.Bytes(), &m) != nil {
			continue
		}
		if id, isRequest := m["id"]; isRequest && m["method"] != nil {
			// Refuse server requests such as approvals; the Host decides through its tools.
			c.send(map[string]any{"id": id, "error": map[string]any{"code": -32601, "message": "refused by the smoke Host"}})
			continue
		}
		if match(m) {
			return m
		}
	}
	c.t.Fatalf("App Server output ended; diagnostics %q", c.s.Diagnostics())
	return nil
}
