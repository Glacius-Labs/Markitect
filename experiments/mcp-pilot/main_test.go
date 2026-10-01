package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func init() {
	if os.Getenv("MARKITECT_MCP_TEST_HELPER") != "1" {
		return
	}
	invocation := struct {
		Args   []string `json:"args"`
		GitDir string   `json:"gitDir"`
	}{Args: os.Args[1:], GitDir: os.Getenv("GIT_DIR")}
	data, _ := json.Marshal(invocation)
	_ = os.WriteFile(os.Getenv("MARKITECT_MCP_TEST_RECORD"), data, 0600)
	fmt.Fprint(os.Stdout, "fixed-helper-result\n")
	os.Exit(0)
}

func TestStdioProtocolSmoke(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find","arguments":{"query":"sample","shell":"echo unsafe"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"execute","arguments":{"command":"echo unsafe"}}}`,
	}, "\n") + "\n"
	s := &server{cli: "must-not-be-run", repo: "/fixed/repo", revision: strings.Repeat("a", 40)}
	var output bytes.Buffer
	if err := s.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d protocol responses, want 4: %s", len(lines), output.String())
	}
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("stdout contained a non-JSON-RPC line %q: %v", line, err)
		}
		if response["jsonrpc"] != "2.0" {
			t.Fatalf("unexpected response: %#v", response)
		}
		responses = append(responses, response)
	}
	initialize := responses[0]["result"].(map[string]any)
	if initialize["protocolVersion"] != protocolVersion {
		t.Fatalf("unexpected negotiated version: %#v", initialize)
	}
	list := responses[1]["result"].(map[string]any)
	tools := list["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("listed %d tools, want find/explain/context", len(tools))
	}
	findSchema := tools[0].(map[string]any)["inputSchema"].(map[string]any)
	kinds := findSchema["properties"].(map[string]any)["kind"].(map[string]any)["enum"].([]any)
	if len(kinds) != 6 {
		t.Fatalf("find exposes an unsupported kind: %#v", kinds)
	}
	invalidCall := responses[2]["result"].(map[string]any)
	if invalidCall["isError"] != true {
		t.Fatalf("unexpected tool result for extra argument: %#v", invalidCall)
	}
	if _, ok := responses[3]["error"]; !ok {
		t.Fatalf("unknown tool did not return a JSON-RPC error: %#v", responses[3])
	}
}

func TestLifecycleOrdering(t *testing.T) {
	s := &server{}
	_, rpcErr, _ := s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"})
	if rpcErr == nil || rpcErr.Code != -32002 {
		t.Fatalf("tools/list before initialize returned %#v", rpcErr)
	}
	s.dispatch(request{JSONRPC: "2.0", Method: "notifications/initialized"})
	if s.ready {
		t.Fatal("initialized notification before initialize advanced lifecycle")
	}
	_, rpcErr, _ = s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2025-11-25"}`)})
	if rpcErr != nil || !s.initDone || s.ready {
		t.Fatalf("initialize state is wrong: initDone=%v ready=%v err=%#v", s.initDone, s.ready, rpcErr)
	}
	_, rpcErr, _ = s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: "tools/list"})
	if rpcErr == nil || rpcErr.Code != -32002 {
		t.Fatalf("tools/list before initialized notification returned %#v", rpcErr)
	}
	s.dispatch(request{JSONRPC: "2.0", Method: "notifications/initialized"})
	if !s.ready {
		t.Fatal("initialized notification did not advance lifecycle")
	}
	_, rpcErr, _ = s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2025-11-25"}`)})
	if rpcErr == nil || rpcErr.Code != -32600 {
		t.Fatalf("second initialize did not return invalid request: %#v", rpcErr)
	}
}

func TestInitializeProtocolNegotiation(t *testing.T) {
	for _, test := range []struct {
		name    string
		offered string
		want    string
	}{
		{name: "legacy Codex client", offered: legacyProtocolVersion, want: legacyProtocolVersion},
		{name: "current client", offered: protocolVersion, want: protocolVersion},
		{name: "unknown version falls back to current", offered: "2099-01-01", want: protocolVersion},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &server{}
			params, _ := json.Marshal(map[string]string{"protocolVersion": test.offered})
			result, rpcErr, reply := s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize", Params: params})
			if !reply || rpcErr != nil {
				t.Fatalf("initialize reply=%v error=%#v", reply, rpcErr)
			}
			got := result.(map[string]any)["protocolVersion"]
			if got != test.want {
				t.Fatalf("negotiated version = %#v, want %q", got, test.want)
			}
		})
	}

	for _, params := range []json.RawMessage{nil, json.RawMessage(`{}`), json.RawMessage(`{"protocolVersion":""}`), json.RawMessage(`[]`)} {
		s := &server{}
		_, rpcErr, reply := s.dispatch(request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "initialize", Params: params})
		if !reply || rpcErr == nil || rpcErr.Code != -32602 {
			t.Errorf("malformed initialize params %q returned reply=%v error=%#v", params, reply, rpcErr)
		}
		if s.initDone {
			t.Errorf("malformed initialize params %q advanced lifecycle", params)
		}
	}
}

func TestSuccessfulToolUsesPinnedArgumentsAndCleansGitEnvironment(t *testing.T) {
	recordPath := filepath.Join(t.TempDir(), "invocation.json")
	t.Setenv("MARKITECT_MCP_TEST_HELPER", "1")
	t.Setenv("MARKITECT_MCP_TEST_RECORD", recordPath)
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "attacker.git"))

	repo := filepath.Join(t.TempDir(), "canonical-repo")
	revision := strings.Repeat("b", 40)
	s := &server{cli: os.Args[0], repo: repo, revision: revision}
	result := s.callTool(json.RawMessage(`{"name":"find","arguments":{"query":"deploy policy","kind":"Skill","namespace":"sample"}}`))
	if result["isError"] != false {
		t.Fatalf("valid tool invocation failed: %#v", result)
	}
	content := result["content"].([]map[string]string)
	if content[0]["text"] != "fixed-helper-result\n" {
		t.Fatalf("unexpected fake CLI result: %#v", content)
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("fake CLI was not invoked: %v", err)
	}
	var invocation struct {
		Args   []string `json:"args"`
		GitDir string   `json:"gitDir"`
	}
	if err := json.Unmarshal(data, &invocation); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"find", "--query", "deploy policy", "--kind", "Skill", "--namespace", "sample", "--repo", repo, "--revision", revision}
	if !reflect.DeepEqual(invocation.Args, wantArgs) {
		t.Fatalf("CLI argv = %#v, want %#v", invocation.Args, wantArgs)
	}
	if invocation.GitDir != "" {
		t.Fatalf("child retained hostile GIT_DIR=%q", invocation.GitDir)
	}
	if err := os.Remove(recordPath); err != nil {
		t.Fatal(err)
	}
	invalid := s.callTool(json.RawMessage(`{"name":"find","arguments":{"query":"x","revision":"HEAD"}}`))
	if invalid["isError"] != true {
		t.Fatalf("invalid arguments were accepted: %#v", invalid)
	}
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("invalid call invoked the CLI; record stat error = %v", err)
	}
}

func TestCanonicalGitRootAndCleanGitEnvironment(t *testing.T) {
	repo := t.TempDir()
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "-c", "user.name=Pilot Test", "-c", "user.email=pilot@example.invalid", "commit", "--allow-empty", "-q", "-m", "fixture")
	revision := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "wrong.git"))
	root, err := canonicalGitRepoRoot(repo)
	if err != nil {
		t.Fatalf("canonical root check failed with hostile GIT_DIR: %v", err)
	}
	if !samePath(root, repo) {
		t.Fatalf("root = %q, want %q", root, repo)
	}
	if err := validateCommit(root, revision); err != nil {
		t.Fatalf("revision validation failed with hostile GIT_DIR: %v", err)
	}
	nested := filepath.Join(repo, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalGitRepoRoot(nested); err == nil {
		t.Fatal("accepted a nested directory instead of the canonical Git top-level")
	}
}

func gitTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	argv := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", argv...)
	cmd.Env = cleanGitEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", argv, err, output)
	}
	return string(output)
}

func TestToolCallRejectsUnknownArgumentBeforeInvocation(t *testing.T) {
	s := &server{cli: "unused"}
	for _, raw := range []string{
		`{"name":"execute","arguments":{"command":"echo unsafe"}}`,
		`{"name":"find","arguments":{"query":"x","revision":"HEAD"}}`,
	} {
		result := s.callTool(json.RawMessage(raw))
		if result["isError"] != true {
			t.Errorf("accepted unsafe or unknown request %s", raw)
		}
	}
}

func TestStdioLinesAreNewlineDelimited(t *testing.T) {
	var output bytes.Buffer
	if err := writeRPC(bufio.NewWriter(&output), json.RawMessage(`7`), map[string]any{"ok": true}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(output.String(), "\n") {
		t.Fatalf("response was not newline delimited: %q", output.String())
	}
}

func TestExposedToolsAreAnnotatedReadOnly(t *testing.T) {
	want := map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
	for _, tool := range toolDefinitions() {
		annotations, ok := tool["annotations"].(map[string]bool)
		if !ok || !reflect.DeepEqual(annotations, want) {
			t.Errorf("tool %v annotations = %#v, want %#v", tool["name"], tool["annotations"], want)
		}
	}
}

func TestInitializeCountersUnsupportedVersionWithSupportedVersion(t *testing.T) {
	s := &server{}
	result, rpcErr, reply := s.dispatch(request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2026-07-28","capabilities":{},"clientInfo":{"name":"Codex","version":"0.159.2"}}`),
	})
	if !reply || rpcErr != nil {
		t.Fatalf("initialize should counter with the supported legacy version: reply=%v err=%#v", reply, rpcErr)
	}
	initialized := result.(map[string]any)
	if initialized["protocolVersion"] != protocolVersion {
		t.Fatalf("server negotiated %v, want supported version %s", initialized["protocolVersion"], protocolVersion)
	}
	if !s.initDone || s.ready {
		t.Fatalf("initialize state is wrong: initDone=%v ready=%v", s.initDone, s.ready)
	}
}
