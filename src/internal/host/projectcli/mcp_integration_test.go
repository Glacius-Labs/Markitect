package projectcli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectonboarding"
)

func TestMCPSharedExplorePreviewAndCASParity(t *testing.T) {
	root := copyProjectWorld(t)
	operations := projectOperations()
	record := &projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "mcp-work", Status: projectexplore.StatusActive, Request: "Preview the same selected scope through both transports", Scopes: []projectexplore.Scope{{ID: "scope", Name: "Scope", Goal: "Inspect owned artifacts", Operation: "apply", ManagerIDs: []string{}}}, Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
	preview, err := operations.Explore(projectapp.ExploreOperation{Selection: projectapp.Selection{Root: root}, Record: record})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newMCPServer(env{root: root, ops: operations})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Call(context.Background(), "explore", mustRaw(t, map[string]any{"input": record}))
	if err != nil || result.IsError {
		t.Fatalf("MCP preview: %+v %v", result, err)
	}
	outcome := result.StructuredContent.(map[string]any)
	plan := outcome["data"].(map[string]any)["plan"].(map[string]any)
	if outcome["operation"] != "explore" || plan["digest"] != preview.Plan.Digest {
		t.Fatalf("transport changed guarded preview: operation=%v got=%v want=%s", outcome["operation"], plan["digest"], preview.Plan.Digest)
	}
	// The CLI decodes the same record through the same closed schema.
	input := writeDraft(t, root, ".markitect/drafts/mcp-work.json", record)
	if cli := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", root, "--input", input)); cli.Plan == nil || cli.Plan.Digest != preview.Plan.Digest {
		t.Fatalf("CLI preview differs from MCP preview: %+v", cli.Plan)
	}
	failed, err := server.Call(context.Background(), "explore", mustRaw(t, map[string]any{"input": record, "write": true, "expect": "sha256:stale"}))
	if err != nil || !failed.IsError {
		t.Fatalf("wrong digest accepted: %+v %v", failed, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".markitect", "state", "explorations", record.ID+".json")); !os.IsNotExist(err) {
		t.Fatalf("wrong digest wrote exploration: %v", err)
	}
	if failed, err := server.Call(context.Background(), "explore", mustRaw(t, map[string]any{"input": record, "write": true})); err != nil || !failed.IsError ||
		failed.StructuredContent.(map[string]any)["diagnostic"].(map[string]any)["code"] != "invalid_arguments" {
		t.Fatalf("MCP write without expect: %+v %v", failed, err)
	}
	written, err := server.Call(context.Background(), "explore", mustRaw(t, map[string]any{"input": record, "write": true, "expect": preview.Plan.Digest}))
	if err != nil || written.IsError {
		t.Fatalf("MCP write: %+v %v", written, err)
	}
	stored, err := projectexplore.Load(root, record.ID)
	if err != nil || stored.Digest != preview.Plan.Next.Digest {
		t.Fatalf("MCP stored different record: %+v %v", stored, err)
	}
}

// The minimal record that the onboarding guidance tells agents to pass to
// explore previews through both the CLI and the MCP tool.
func TestOnboardingExploreRecordPreviewsThroughCLIAndMCP(t *testing.T) {
	root := copyProjectWorld(t)
	operations := projectOperations()
	index, err := operations.Index(projectapp.Selection{Root: root})
	if err != nil || len(index.Managers) == 0 {
		t.Fatalf("index: %v", err)
	}
	plan, err := projectonboarding.Preview(root, index.ModelDigest, projectonboarding.Options{Providers: []projectonboarding.Provider{projectonboarding.Codex}})
	if err != nil {
		t.Fatal(err)
	}
	workflow := ""
	for _, file := range plan.Files {
		if file.Path == ".markitect/workflows/model-first.md" {
			workflow = strings.ReplaceAll(file.Content, "\r\n", "\n")
		}
	}
	fence := strings.Repeat("`", 3)
	start := strings.Index(workflow, "Minimal new exploration input record")
	if start < 0 {
		t.Fatal("rendered guidance has no minimal explore record")
	}
	open := strings.Index(workflow[start:], fence+"json\n")
	if open < 0 {
		t.Fatal("rendered guidance record has no opening fence")
	}
	start += open + len(fence+"json\n")
	end := strings.Index(workflow[start:], "\n"+fence)
	if end < 0 {
		t.Fatal("rendered guidance record has no closing fence")
	}
	manager, err := json.Marshal(index.Managers[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	record := strings.Replace(strings.TrimSpace(workflow[start:start+end]), `"<existing-manager-id>"`, string(manager), 1)
	if !json.Valid([]byte(record)) {
		t.Fatalf("guidance record is not JSON:\n%s", record)
	}

	input := writeDraft(t, root, ".markitect/drafts/guidance-record.json", []byte(record))
	cli := decodeOutput[projectapp.ExploreResult](t, mustCLI(t, "explore", "--repo", root, "--input", input))
	if cli.Plan == nil || cli.Plan.Digest == "" {
		t.Fatalf("CLI explore preview of the guidance record = %+v", cli)
	}
	server, err := newMCPServer(env{root: root, ops: operations})
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Call(context.Background(), "explore", json.RawMessage(`{"input":`+record+`}`))
	if err != nil || result.IsError {
		t.Fatalf("MCP explore rejected the record the onboarding guidance tells agents to pass: err=%v result=%+v", err, result.StructuredContent)
	}
	if digest := result.StructuredContent.(map[string]any)["data"].(map[string]any)["plan"].(map[string]any)["digest"]; digest != cli.Plan.Digest {
		t.Fatalf("MCP preview digest %v differs from CLI preview digest %s", digest, cli.Plan.Digest)
	}
}

func TestMCPCompositionFixesTheRootAndTakesNoSourceRepository(t *testing.T) {
	root := copyProjectWorld(t)
	server, err := newMCPServer(env{root: root, ops: projectOperations()})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"check", "explore", "config", "status"} {
		if _, err := server.Call(context.Background(), name, json.RawMessage(`{"root":"elsewhere"}`)); err == nil {
			t.Fatalf("%s accepted caller root redirection", name)
		}
		if _, err := server.Call(context.Background(), name, json.RawMessage(`{"repo":"elsewhere"}`)); err == nil {
			t.Fatalf("%s accepted a repo field", name)
		}
	}
	outside := t.TempDir()
	if _, err := server.Call(context.Background(), "adopt", mustRaw(t, map[string]any{"action": "status", "session": "s", "sourceRepo": outside})); err == nil {
		t.Fatal("adopt accepted a caller source repository")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("MCP wrote outside its root: %v %v", entries, err)
	}
	if _, err := newMCPServer(env{ops: projectOperations()}); err == nil {
		t.Fatal("MCP composition accepted an empty root")
	}
}

// mcpSession drives a server over the stdio protocol and returns the
// initialize instructions and the tools/list result.
func mcpSession(t *testing.T, server *mcp.Server) (string, []mcp.Tool) {
	t.Helper()
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + mcp.ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var instructions string
	var tools []mcp.Tool
	scanner := bufio.NewScanner(&out)
	scanner.Buffer(make([]byte, 1<<20), 64<<20)
	for scanner.Scan() {
		var response struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		switch response.ID {
		case 1:
			var result struct {
				Instructions string `json:"instructions"`
			}
			if err := json.Unmarshal(response.Result, &result); err != nil {
				t.Fatal(err)
			}
			instructions = result.Instructions
		case 2:
			var result struct {
				Tools []mcp.Tool `json:"tools"`
			}
			if err := json.Unmarshal(response.Result, &result); err != nil {
				t.Fatal(err)
			}
			tools = result.Tools
		}
	}
	if len(tools) == 0 {
		t.Fatalf("tools/list returned no tools: %s", out.String())
	}
	return instructions, tools
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	sort.Strings(out)
	return out
}

// TestVerbTableParity walks the verb table: CLI help covers every argument,
// and MCP tools/list exposes exactly the table's verbs with exactly their
// arguments, in normal and read-only mode.
func TestVerbTableParity(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range verbTable() {
		if seen[v.name] {
			t.Errorf("verb %s is listed twice", v.name)
		}
		seen[v.name] = true
		if v.summary == "" || v.synopsis == "" || v.group == "" || v.effect == "" {
			t.Errorf("verb %s lacks a summary, synopsis, group or effect", v.name)
		}
		if !strings.HasPrefix(v.synopsis, v.name) {
			t.Errorf("verb %s synopsis %q does not start with the verb", v.name, v.synopsis)
		}
		if !v.cliOnly && (v.invoke == nil || v.register == nil || v.input == nil) {
			t.Errorf("MCP verb %s has no handler", v.name)
		}
		if (v.effect == effectExecute || v.effect == effectExecuteWrite) && v.inspect == "" {
			t.Errorf("execute verb %s names no read-only verb to inspect first", v.name)
		}
		if v.defaultSub != "" && !v.hasSub(v.defaultSub) {
			t.Errorf("verb %s default sub-verb %q is not a sub-verb", v.name, v.defaultSub)
		}
		for _, sub := range v.subs {
			if sub.summary == "" || sub.effect == "" {
				t.Errorf("verb %s sub-verb %s lacks a summary or effect", v.name, sub.name)
			}
		}
		names := map[string]bool{}
		for _, a := range v.args {
			if names[a.name] {
				t.Errorf("verb %s lists argument %s twice", v.name, a.name)
			}
			names[a.name] = true
			if a.help == "" {
				t.Errorf("verb %s argument %s has no help", v.name, a.name)
			}
			switch {
			case a.operand:
				if a.value == "" || !strings.Contains(v.synopsis, a.value) {
					t.Errorf("verb %s synopsis %q omits operand %s", v.name, v.synopsis, a.value)
				}
			case a.name == "repo" || a.name == "source-repo":
			default:
				if !regexp.MustCompile(`--` + regexp.QuoteMeta(a.name) + `([^a-z0-9-]|$)`).MatchString(v.synopsis) {
					t.Errorf("verb %s synopsis %q omits --%s", v.name, v.synopsis, a.name)
				}
			}
		}
	}

	for _, readOnly := range []bool{false, true} {
		server, err := newMCPServer(env{root: t.TempDir(), ops: projectOperations(), readOnly: readOnly})
		if err != nil {
			t.Fatal(err)
		}
		instructions, tools := mcpSession(t, server)
		if readOnly != strings.Contains(instructions, "read-only") {
			t.Errorf("readOnly=%v initialize instructions = %q", readOnly, instructions)
		}
		want := []string{}
		for _, v := range verbTable() {
			if v.mcpVerb(readOnly) {
				want = append(want, v.name)
			}
		}
		sort.Strings(want)
		got := []string{}
		for _, tool := range tools {
			got = append(got, tool.Name)
		}
		if !slices.Equal(got, want) {
			t.Errorf("readOnly=%v tools/list = %v, want %v", readOnly, got, want)
		}
		for _, tool := range tools {
			v, ok := lookupVerb(tool.Name)
			if !ok {
				t.Errorf("tool %s is not in the verb table", tool.Name)
				continue
			}
			omitted := map[string]bool{}
			if readOnly {
				for _, field := range v.readOnlyOmits() {
					omitted[field] = true
				}
			}
			wantProperties, wantRequired := []string{}, []string{}
			for _, a := range v.args {
				field := camel(a.name)
				if a.cliOnly || omitted[field] {
					continue
				}
				wantProperties = append(wantProperties, field)
				if a.required {
					wantRequired = append(wantRequired, field)
				}
			}
			if len(v.subs) != 0 {
				wantProperties = append(wantProperties, "action")
				if v.defaultSub == "" {
					wantRequired = append(wantRequired, "action")
				}
			}
			sort.Strings(wantProperties)
			sort.Strings(wantRequired)
			schema, _ := tool.InputSchema.(map[string]any)
			properties, _ := schema["properties"].(map[string]any)
			required, _ := schema["required"].([]any)
			if gotProperties := sortedKeys(properties); !slices.Equal(gotProperties, wantProperties) {
				t.Errorf("readOnly=%v tool %s fields = %v, want %v", readOnly, tool.Name, gotProperties, wantProperties)
			}
			if gotRequired := sortedStrings(required); !slices.Equal(gotRequired, wantRequired) {
				t.Errorf("readOnly=%v tool %s required = %v, want %v", readOnly, tool.Name, gotRequired, wantRequired)
			}
			if schema["additionalProperties"] != false {
				t.Errorf("tool %s input schema is not closed", tool.Name)
			}
			if wantHint := readOnly || v.effect == effectRead; tool.Annotations["readOnlyHint"] != wantHint {
				t.Errorf("readOnly=%v tool %s readOnlyHint = %v, want %v", readOnly, tool.Name, tool.Annotations["readOnlyHint"], wantHint)
			}
			if len(v.subs) != 0 {
				wantActions := v.subNames()
				if readOnly {
					wantActions = v.readOnlySubs()
				}
				action, _ := properties["action"].(map[string]any)
				enum, _ := action["enum"].([]any)
				if gotActions := sortedStrings(enum); !slices.Equal(gotActions, sortedCopy(wantActions)) {
					t.Errorf("readOnly=%v tool %s actions = %v, want %v", readOnly, tool.Name, gotActions, wantActions)
				}
			}
		}
	}
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

// A read-only server writes nothing and starts nothing, whatever the client
// sends: write and execute fields, provider executables and dismissals are
// rejected, and execute verbs are not registered.
func TestReadOnlyMCPRejectsWritesExecutionsAndProbes(t *testing.T) {
	root := copyProjectWorld(t)
	server, err := newMCPServer(env{root: root, ops: projectOperations(), readOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	call := func(name, args string) (mcp.CallResult, error) {
		return server.Call(context.Background(), name, json.RawMessage(args))
	}
	explore := string(mustRaw(t, projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "read-only-work", Status: projectexplore.StatusActive, Request: "r",
		Scopes:    []projectexplore.Scope{{ID: "scope", Name: "Scope", Goal: "g", Operation: "apply", ManagerIDs: []string{}}},
		Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{}, Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}))
	executableOptions := `{"provider":"process","model":"m","effort":"","codexProfile":"","providerExecutable":"C:/tools/executor.exe","costMode":"unmetered","inputMicrosPerMillion":0,"outputMicrosPerMillion":0,"maxCostMicros":1}`
	for _, tc := range []struct {
		name, tool, args string
	}{
		{"init write", "init", `{"name":"x","expect":"d","write":true}`},
		{"init expect", "init", `{"name":"x","expect":"d"}`},
		{"docs write", "docs", `{"expect":"d","write":true}`},
		{"explore write", "explore", `{"input":` + explore + `,"expect":"d","write":true}`},
		{"ready write", "ready", `{"exploration":"e","scope":"s","write":true}`},
		{"plan write", "plan", `{"goal":"g","expect":"d","write":true}`},
		{"apply write", "apply", `{"plan":"p","run":"r","candidate":"c","branch":"b","head":"h","worktree":"w","expect":"d","write":true}`},
		{"onboard write", "onboard", `{"provider":"codex","expect":"d","write":true}`},
		{"brief create write", "brief", `{"since":"a","revision":"b","provenance":"p","expect":"d","write":true}`},
		{"brief dismiss", "brief", `{"action":"dismiss","event":"e","manager":"m"}`},
		{"adopt start write", "adopt", `{"action":"start","revision":"r","write":true}`},
		{"adopt run execute", "adopt", `{"action":"run","session":"s","execute":true}`},
		{"adopt apply", "adopt", `{"action":"apply","session":"s"}`},
		{"doctor probe executable", "doctor", `{"provider":"process","providerExecutable":"C:/tools/executor.exe"}`},
		{"config flag executable", "config", `{"provider":"process","model":"m","providerExecutable":"C:/tools/executor.exe","maxCostMicros":1}`},
		{"config input executable", "config", `{"input":` + executableOptions + `}`},
		{"run", "run", `{"plan":"p","execute":true}`},
		{"resume", "resume", `{"run":"r","execute":true}`},
		{"repair", "repair", `{"run":"r","execute":true}`},
		{"verify", "verify", `{"run":"r","execute":true}`},
		{"deliver", "deliver", `{"exploration":"e","scope":"s","execute":true,"write":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := call(tc.tool, tc.args)
			if err == nil && !result.IsError {
				t.Fatalf("read-only server accepted %s %s: %+v", tc.tool, tc.args, result.StructuredContent)
			}
		})
	}
	// The read-only server still serves reads and previews.
	for _, tc := range []struct{ tool, args string }{
		{"check", `{}`},
		{"docs", `{}`},
		{"explore", `{"input":` + explore + `}`},
		{"brief", `{"action":"list"}`},
	} {
		if result, err := call(tc.tool, tc.args); err != nil || result.IsError {
			t.Fatalf("read-only %s %s failed: %+v %v", tc.tool, tc.args, result.StructuredContent, err)
		}
	}
	if status := gitOutput(t, root, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("read-only MCP server changed the repository:\n%s", status)
	}
}

// KG-02: impact --explain and context --trace return the same JSON through the
// CLI and the MCP tool, and the new flags keep the plain outputs unchanged.
func TestExplainAndTraceAreEqualThroughCLIAndMCP(t *testing.T) {
	root := copyProjectWorld(t)
	since := gitOutput(t, root, "rev-parse", "HEAD")
	contract := filepath.Join(root, ".markitect", "model", "commerce", "sales", "inventory", "reservations", "release-reservation.yaml")
	data, err := os.ReadFile(contract)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "does not release quantity twice.", "never releases quantity twice.", 1)
	if edited == string(data) {
		t.Fatal("the Shop contract text changed; adjust the edit")
	}
	if err := os.WriteFile(contract, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGitWithEnv(t, root, testCommitEnv, "commit", "-m", "tighten release")
	revision := gitOutput(t, root, "rev-parse", "HEAD")
	server, err := newMCPServer(env{root: root, ops: projectOperations()})
	if err != nil {
		t.Fatal(err)
	}
	mcpJSON := func(tool string, args map[string]any) string {
		t.Helper()
		result, err := server.Call(context.Background(), tool, mustRaw(t, args))
		if err != nil || result.IsError {
			t.Fatalf("MCP %s %v: %+v %v", tool, args, result.StructuredContent, err)
		}
		data, err := json.Marshal(result.StructuredContent.(map[string]any)["data"])
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	cliJSON := func(args ...string) string {
		t.Helper()
		var value any
		if err := json.Unmarshal(mustCLI(t, append(args, "--repo", root)...), &value); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(value)
		return string(data)
	}
	inventory := `["project.markitect.example.org/v1alpha1","Manager","commerce.sales.inventory","inventory"]`
	orders := `["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]`
	cancel := `["project.markitect.example.org/v1alpha1","Statement","commerce.sales.orders","cancel-order"]`
	for name, pair := range map[string]struct {
		cli  []string
		tool string
		args map[string]any
	}{
		"impact":                {[]string{"impact", "--since", since, "--revision", revision}, "impact", map[string]any{"since": since, "revision": revision}},
		"impact --explain":      {[]string{"impact", "--since", since, "--revision", revision, "--explain"}, "impact", map[string]any{"since": since, "revision": revision, "explain": true}},
		"impact --explain as a": {[]string{"impact", "--since", since, "--revision", revision, "--explain", "--manager", inventory}, "impact", map[string]any{"since": since, "revision": revision, "explain": true, "manager": inventory}},
		"context":               {[]string{"context", orders}, "context", map[string]any{"manager": orders}},
		"context --trace":       {[]string{"context", orders, "--trace", cancel, "--direction", "both", "--depth", "2"}, "context", map[string]any{"manager": orders, "trace": cancel, "direction": "both", "depth": 2}},
	} {
		cli, viaMCP := cliJSON(pair.cli...), mcpJSON(pair.tool, pair.args)
		if cli != viaMCP {
			t.Fatalf("%s differs:\nCLI %s\nMCP %s", name, cli, viaMCP)
		}
		hasExplanation, hasTrace := strings.Contains(cli, `"explanation":`), strings.Contains(cli, `"trace":`)
		if hasExplanation != strings.Contains(name, "--explain") || hasTrace != strings.Contains(name, "--trace") {
			t.Fatalf("%s: explanation %v, trace %v", name, hasExplanation, hasTrace)
		}
	}
	// The explanation names why cancel-order is in the impact.
	explained := decodeOutput[impactResult](t, mustCLI(t, "impact", "--repo", root, "--since", since, "--revision", revision, "--explain"))
	if explained.Explanation == nil || explained.Explanation.ImpactDigest != explained.Digest {
		t.Fatalf("explanation is not bound to the impact: %+v", explained.Explanation)
	}
	found := false
	for _, e := range explained.Explanation.Elements {
		found = found || e.ID == cancel && e.Reason == "consumer" && e.Class == "change"
	}
	if !found {
		t.Fatalf("cancel-order is not explained as a changed consumer: %+v", explained.Explanation.Elements)
	}
	for _, args := range [][]string{
		{"impact", "--repo", root, "--since", since, "--revision", revision, "--manager", inventory},
		{"context", orders, "--repo", root, "--depth", "2"},
		{"context", orders, "--repo", root, "--trace", cancel, "--direction", "sideways"},
	} {
		if code, _, stderr := runCLI(t, args...); code != 2 {
			t.Fatalf("%v: exit %d, want 2: %s", args, code, stderr)
		}
	}
	if result, err := server.Call(context.Background(), "impact", mustRaw(t, map[string]any{"since": since, "revision": revision, "manager": inventory})); err != nil || !result.IsError ||
		result.StructuredContent.(map[string]any)["diagnostic"].(map[string]any)["code"] != "invalid_arguments" {
		t.Fatalf("MCP --manager without explain: %+v %v", result, err)
	}
}
