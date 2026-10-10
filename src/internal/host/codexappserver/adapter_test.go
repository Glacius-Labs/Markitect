package codexappserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

// TestMain supplies a protocol-only child executable. It does not invoke Codex
// or a model. Using the real process boundary exercises version and cleanup too.
func TestMain(m *testing.M) {
	if os.Getenv("MARKITECT_P04_FIXTURE") == "1" {
		if len(os.Args) > 1 && os.Args[1] == "--version" {
			if os.Getenv("MARKITECT_P04_MODE") == "version" {
				fmt.Println("codex-cli 0.161.0")
			} else {
				fmt.Println(SupportedProviderVersion)
			}
			os.Exit(0)
		}
		serveFixture()
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func serveFixture() {
	mode := os.Getenv("MARKITECT_P04_MODE")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 65536), 16<<20)
	enc := json.NewEncoder(os.Stdout)
	note := func(method string, params any) { _ = enc.Encode(map[string]any{"method": method, "params": params}) }
	var response string
	var turnID = "turn-1"
	for scanner.Scan() {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
		}
		_ = json.Unmarshal(scanner.Bytes(), &msg)
		result := any(map[string]any{})
		switch msg.Method {
		case "initialize":
			result = map[string]string{"userAgent": "codex/0.162.0", "codexHome": os.TempDir(), "platformFamily": "windows", "platformOs": "windows"}
		case "initialized":
			continue
		case "thread/start", "thread/resume":
			if msg.Method == "thread/start" && strings.HasPrefix(mode, "recovery-") {
				return
			}
			var threadParams struct {
				Permissions string `json:"permissions"`
			}
			_ = json.Unmarshal(msg.Params, &threadParams)
			if mode == "request-failure" {
				_ = enc.Encode(map[string]any{"id": msg.ID, "error": map[string]any{"code": -32000, "message": "fixture failure"}})
				continue
			}
			cwd, _ := os.Getwd()
			sessionID := "thread-1"
			if mode == "different-session" {
				sessionID = "session-other"
			}
			model := "gpt-6-luna"
			if mode == "model-mismatch" {
				model = "wrong"
			}
			sandboxType := "workspaceWrite"
			if strings.Contains(mode, "readonly-sandbox") {
				sandboxType = "readOnly"
			}
			if strings.Contains(mode, "unknown-sandbox") {
				sandboxType = ""
			}
			sandbox := map[string]string{"type": sandboxType}
			if sandboxType == "" {
				sandbox = map[string]string{}
			}
			result = map[string]any{"thread": thread{ID: "thread-1", SessionID: sessionID, CLIVersion: "0.162.0", CWD: cwd}, "model": model, "reasoningEffort": "high", "cwd": cwd, "approvalPolicy": "on-request", "sandbox": sandbox, "instructionSources": []string{}}
			if threadParams.Permissions != "" && mode != "profile-unconfirmed" {
				result.(map[string]any)["activePermissionProfile"] = map[string]string{"id": threadParams.Permissions}
			}
		case "turn/start":
			if strings.HasPrefix(mode, "recovery-") {
				return
			}
			var p struct {
				Input []struct {
					Text string `json:"text"`
				} `json:"input"`
			}
			_ = json.Unmarshal(msg.Params, &p)
			var inv agentexec.Invocation
			if len(p.Input) > 0 {
				parts := strings.Split(p.Input[0].Text, "Invocation:\n")
				if len(parts) > 1 {
					_ = json.Unmarshal([]byte(parts[1]), &inv)
				}
			}
			r := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role, InputDigest: inv.InputDigest, Outcome: agentexec.OutcomeIncomplete, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
			b, _ := json.Marshal(r)
			response = string(b)
			if mode == "malformed-response" {
				response = `{"bad":true}`
			}
			if mode == "transport-loss" {
				return
			}
			if mode == "malformed-wire" {
				fmt.Println("{broken")
				return
			}
			note("turn/started", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: "inProgress"}})
			_ = enc.Encode(map[string]any{"id": msg.ID, "result": map[string]any{"turn": turn{ID: turnID, Status: "inProgress"}}})
			if mode == "filechange-childtelemetry-accept" {
				note("thread/started", map[string]any{"thread": thread{ID: "child-thread", SessionID: "child-session", ParentThreadID: "thread-1"}})
				note("item/fileChange/patchUpdated", map[string]any{"threadId": "child-thread", "turnId": "child-turn", "itemId": "child-patch", "changes": []map[string]any{{"path": "README.md", "kind": map[string]any{"type": "update"}}}})
			}
			if strings.HasPrefix(mode, "filechange-") {
				var changes []fileUpdateChange
				_ = json.Unmarshal([]byte(os.Getenv("MARKITECT_P04_CHANGES")), &changes)
				changeItem := item{ID: "patch-1", Type: "fileChange", Status: "inProgress", Changes: changes}
				note("item/started", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": changeItem})
				params := map[string]any{"threadId": "thread-1", "turnId": turnID, "itemId": changeItem.ID, "startedAtMs": time.Now().UnixMilli()}
				switch os.Getenv("MARKITECT_P04_APPROVAL_VARIANT") {
				case "wrong-thread":
					params["threadId"] = "thread-other"
				case "wrong-turn":
					params["turnId"] = "turn-other"
				case "wrong-item":
					params["itemId"] = "item-other"
				case "grant-root":
					cwd, _ := os.Getwd()
					params["grantRoot"] = cwd
				case "missing-start":
					delete(params, "startedAtMs")
				case "empty-grant-root":
					params["grantRoot"] = ""
				case "unknown-field":
					params["futurePermission"] = true
				case "nullable-optional-fields":
					params["reason"] = nil
					params["grantRoot"] = nil
				}
				_ = enc.Encode(map[string]any{"id": 99, "method": "item/fileChange/requestApproval", "params": params})
				continue
			}
			if mode == "timeout" || mode == "interrupt-confirmed" {
				continue
			}
			if mode == "approval" {
				_ = enc.Encode(map[string]any{"id": 99, "method": "item/commandExecution/requestApproval", "params": map[string]string{"threadId": "thread-1", "turnId": turnID}})
				continue
			}
			if mode == "dynamic" {
				_ = enc.Encode(map[string]any{"id": 98, "method": "item/tool/call", "params": ToolCall{ThreadID: "thread-1", TurnID: turnID, CallID: "call-1", Tool: "markitect_start_helper", Arguments: json.RawMessage(`{"task":"bounded"}`)}})
				continue
			}
			if mode == "event-limit" {
				note("item/agentMessage/delta", map[string]string{"delta": strings.Repeat("x", 100000)})
				continue
			}
			if mode == "helper" {
				v := item{ID: "spawn-1", Type: "collabAgentToolCall", Tool: "spawnAgent", SenderThreadID: "thread-1", Status: "failed", ReceiverThreadIDs: []string{}, AgentsStates: map[string]struct {
					Status string `json:"status"`
				}{}}
				note("item/started", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": v})
				note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": v})
			}
			status := "completed"
			if mode == "failed" {
				status = "failed"
			}
			if mode == "interrupted" {
				status = "interrupted"
			}
			note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": item{ID: "message-1", Type: "agentMessage", Phase: "final_answer", Text: response}})
			note("turn/completed", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: status}})
			continue
		case "turn/interrupt":
			if strings.HasPrefix(mode, "filechange-") {
				note("turn/completed", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: "interrupted"}})
			} else if mode == "interrupt-confirmed" {
				note("turn/completed", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: "interrupted"}})
			} else if mode == "timeout" {
				// The server closes without confirming interruption. The adapter
				// must retain unknown state instead of inventing a terminal result.
				return
			}
		case "":
			if strings.HasPrefix(mode, "filechange-") && string(msg.ID) == "99" {
				var decision struct {
					Decision string `json:"decision"`
				}
				_ = json.Unmarshal(msg.Result, &decision)
				if decision.Decision == "accept" {
					var changes []fileUpdateChange
					_ = json.Unmarshal([]byte(os.Getenv("MARKITECT_P04_CHANGES")), &changes)
					note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": item{ID: "patch-1", Type: "fileChange", Status: "completed", Changes: changes}})
					note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": item{ID: "message-1", Type: "agentMessage", Phase: "final_answer", Text: response}})
					note("turn/completed", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: "completed"}})
				}
			}
			if mode == "dynamic" {
				note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": item{ID: "message-1", Type: "agentMessage", Phase: "final_answer", Text: response}})
				note("turn/completed", map[string]any{"threadId": "thread-1", "turn": turn{ID: turnID, Status: "completed"}})
			}
			continue
		case "thread/read":
			var saved RecoveryHandle
			_ = json.Unmarshal([]byte(os.Getenv("MARKITECT_P04_RECOVERY")), &saved)
			inv := saved.Invocation
			r := agentexec.Response{APIVersion: agentexec.APIVersion, RunID: inv.RunID, Nonce: inv.Nonce, Role: inv.Request.Role, InputDigest: inv.InputDigest, Outcome: agentexec.OutcomeIncomplete, CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: []string{}, VerifierObservations: []agentexec.Observation{}, Uncertainty: []string{}}
			b, _ := json.Marshal(r)
			status := "completed"
			items := []item{{ID: "m", Type: "agentMessage", Text: string(b), Phase: "final_answer"}}
			if mode == "recovery-running" {
				status = "inProgress"
			}
			if mode == "recovery-children" || mode == "recovery-running" {
				state := "completed"
				if mode == "recovery-running" {
					state = "running"
				}
				child := item{ID: "spawn-child", Type: "collabAgentToolCall", Tool: "spawnAgent", Status: "completed", SenderThreadID: "thread-1", ReceiverThreadIDs: []string{"child-thread"}, AgentsStates: map[string]struct {
					Status string `json:"status"`
				}{"child-thread": {Status: state}}}
				note("item/completed", map[string]any{"threadId": "thread-1", "turnId": turnID, "item": child})
				note("thread/started", map[string]any{"thread": thread{ID: "child-thread", SessionID: "child-session", ParentThreadID: "thread-1"}})
				if mode == "recovery-children" {
					nested := item{ID: "spawn-nested", Type: "collabAgentToolCall", Tool: "spawnAgent", Status: "failed", SenderThreadID: "child-thread", ReceiverThreadIDs: []string{}}
					note("item/completed", map[string]any{"threadId": "child-thread", "turnId": "child-turn", "item": nested})
					items = append(items, child)
				} else {
					// An additional failed request in a running turn must survive
					// the uncertain return, even when present only in history.
					items = append(items, item{ID: "spawn-failed", Type: "collabAgentToolCall", Tool: "spawnAgent", Status: "failed", ReceiverThreadIDs: []string{}})
				}
			}
			recovered := thread{ID: "thread-1", Turns: []turn{{ID: turnID, Status: status, Items: items}}}
			if mode == "recovery-missing" {
				recovered.Turns = nil
			}
			result = map[string]any{"thread": recovered}
		default:
			return
		}
		if len(msg.ID) > 0 {
			_ = enc.Encode(map[string]any{"id": msg.ID, "result": result})
		}
	}
}

func fixture(t *testing.T, mode string, options Options) (*Adapter, agentexec.Config, agentexec.Request, agentexec.RunOptions) {
	t.Helper()
	t.Setenv("MARKITECT_P04_FIXTURE", "1")
	t.Setenv("MARKITECT_P04_MODE", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Command: exe, ProviderVersion: SupportedProviderVersion, Model: "gpt-6-luna", ReasoningEffort: "high", Timeout: 5 * time.Second, MaxEventBytes: 1 << 20, Helpers: HelperPolicy{Enabled: true, MaxStartRequests: 2, MaxDepth: 1}}
	a, err := NewAdapter(config, options)
	if err != nil {
		t.Fatal(err)
	}
	shared := agentexec.Config{Command: exe, ProviderVersion: config.ProviderVersion, Model: config.Model, Timeout: config.Timeout, MaxStdoutBytes: 65536, MaxStderrBytes: 65536}
	req := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: strings.Repeat("a", 40), ModelDigest: "sha256:" + strings.Repeat("b", 64), ModulePin: "test@1", ProjectionID: "test", ScopeIDs: []string{"source"}, PolicyIDs: []string{}, Context: json.RawMessage(`{}`), Artifacts: []agentexec.Artifact{}}
	opts := agentexec.RunOptions{Workspace: &projectworkspace.Handle{ID: "workspace-1", CWD: t.TempDir(), BaseSHA: req.SourceRevision}}
	return a, shared, req, opts
}

func TestNativeTurnPromptAndStrictTaskResponseContract(t *testing.T) {
	request := agentexec.Request{
		Role:           agentexec.RoleExecutor,
		SourceRevision: strings.Repeat("a", 40),
		ModelDigest:    "sha256:" + strings.Repeat("b", 64),
		ModulePin:      "test@1",
		ProjectionID:   "test",
		ScopeIDs:       []string{"manager"},
		PolicyIDs:      []string{},
		Context:        json.RawMessage(`{"kind":"projectrun-task/v1","phase":"work","responseSchema":{"type":"object","additionalProperties":false}}`),
		Artifacts:      []agentexec.Artifact{},
	}
	inv, wire, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	prompt := nativeTurnPrompt(inv, wire)
	for _, required := range []string{
		"candidateFiles, evidenceRefs, verifierObservations, and uncertainty as JSON arrays",
		"object with exactly subject, outcome, and detail string fields",
		"observation outcome must be passed, failed, incomplete, or escalated",
		"outer outcome must be one of proposed, failed, incomplete, or escalated",
		"reportJson as a JSON object matching request.context.responseSchema exactly",
		"reportJson.status is a task status (complete, partial, blocked, failed, or no-op)",
		"If reportJson.escalateTo is empty, outer outcome is proposed",
		"if escalateTo is nonempty, outer outcome is escalated",
		"Never copy reportJson.status into outer outcome",
	} {
		if !strings.Contains(prompt, required) {
			t.Errorf("native prompt omits contract clause %q", required)
		}
	}

	// This is the captured failure shape: the task's "blocked" status was used
	// as an outer executor outcome, observations were strings, and reportJson
	// was omitted. Keep rejecting it at the shared closed decoder.
	malformed := []byte(fmt.Sprintf(`{"apiVersion":%q,"runId":%q,"nonce":%q,"role":%q,"inputDigest":%q,"outcome":"blocked","candidateFiles":[],"evidenceRefs":[],"verifierObservations":["could not inspect"],"uncertainty":["workspace was not verified"]}`,
		inv.APIVersion, inv.RunID, inv.Nonce, request.Role, inv.InputDigest))
	if _, err := agentexec.DecodeResponse(malformed, inv, ""); err == nil {
		t.Fatal("captured malformed executor response passed the strict decoder")
	}

	// A blocked TaskResponse remains a typed report status. The outer executor
	// outcome is proposed because the report does not request escalation.
	corrected := []byte(fmt.Sprintf(`{"apiVersion":%q,"runId":%q,"nonce":%q,"role":%q,"inputDigest":%q,"outcome":"proposed","candidateFiles":[],"evidenceRefs":[],"verifierObservations":[],"reportJson":{"status":"blocked","summary":"workspace command setup failed before process start","delegations":[],"reworkRequests":[],"integrated":false,"questions":["Can workspace command setup be restored?"],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":""},"uncertainty":["Workspace contents were not independently verified."]}`,
		inv.APIVersion, inv.RunID, inv.Nonce, request.Role, inv.InputDigest))
	parsed, err := agentexec.DecodeResponse(corrected, inv, "")
	if err != nil {
		t.Fatalf("typed blocked task report with valid executor outcome was rejected: %v", err)
	}
	if parsed.Outcome != agentexec.OutcomeProposed || len(parsed.ReportJSON) == 0 || len(parsed.VerifierObservations) != 0 {
		t.Fatalf("corrected response contract was not preserved: outcome=%q report=%s observations=%#v", parsed.Outcome, parsed.ReportJSON, parsed.VerifierObservations)
	}
}

func TestFullVerifyNativeTurnPromptScopesTypedAudit(t *testing.T) {
	request := agentexec.Request{
		Role:           agentexec.RoleExecutor,
		SourceRevision: strings.Repeat("a", 40),
		ModelDigest:    "sha256:" + strings.Repeat("b", 64),
		ModulePin:      "test@1",
		ProjectionID:   "test",
		ScopeIDs:       []string{"manager"},
		PolicyIDs:      []string{},
		Context:        json.RawMessage(`{"kind":"projectrun-full-verify/v1","requiredSubjects":["statement:orders","evidence:negative stock case"],"responseSchema":{"type":"object","additionalProperties":false}}`),
		Artifacts:      []agentexec.Artifact{},
	}
	inv, wire, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	prompt := nativeTurnPrompt(inv, wire)
	for _, required := range []string{
		"read-only, bounded Manager audit",
		"fixed snapshot, model, files, briefing, child assessments, and check results",
		"copy every subject string verbatim into exactly one assessments[].subject",
		"no omissions, duplicates, paraphrases, or additional subjects",
		"Do not gate this scoped audit on unrelated Git inspection, Markitect CLI/MCP availability, or rerunning Host-supplied checks",
		"outer outcome is proposed; reportJson.status independently expresses pass, fail, or incomplete",
		"incomplete when relevant evidence for a required subject is unavailable",
		"put unresolved audit uncertainty outside the typed report",
	} {
		if !strings.Contains(prompt, required) {
			t.Errorf("full verification prompt omits contract clause %q", required)
		}
	}
}

func makeApprovalWorkspace(t *testing.T, taskID string, allowed, excluded []string) (*projectworkspace.GitService, projectworkspace.Request, projectworkspace.Handle) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("candidate\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	runGit("init", "--quiet")
	runGit("config", "user.name", "Codex Approval Test")
	runGit("config", "user.email", "approval-test@example.invalid")
	runGit("add", "README.md")
	runGit("commit", "--quiet", "-m", "candidate")
	base := runGit("rev-parse", "HEAD")
	binding, err := projectworkspace.InspectRepository(context.Background(), root, base)
	if err != nil {
		t.Fatal(err)
	}
	request := projectworkspace.Request{RepositoryRoot: root, RepositoryIdentity: "repo:approval-test", BaseSHA: base, OverlayDigest: binding.OverlayDigest,
		TaskID: taskID, AllowedPaths: append([]string(nil), allowed...), ExcludedPaths: append([]string(nil), excluded...)}
	service, err := projectworkspace.NewGitService(filepath.Join(t.TempDir(), "owned-workspaces"), projectworkspace.Limits{MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	handle, err := service.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close(context.Background(), handle) })
	return service, request, handle
}

type approvalTestCase struct {
	name          string
	mode          string
	paths         []string
	allowed       []string
	excluded      []string
	variant       string
	kind          string
	parentAllowed []string
	profile       string
	role          string
	moveKind      string
	wantDecision  string
	wantErr       bool
}

func runApprovalFixture(t *testing.T, tc approvalTestCase) (string, error) {
	t.Helper()
	if tc.role == "" {
		tc.role = agentexec.RoleExecutor
	}
	service, workspaceRequest, handle := makeApprovalWorkspace(t, "approval-"+tc.name, tc.allowed, tc.excluded)
	_ = service
	if tc.name == "symlink" {
		if err := os.Symlink(t.TempDir(), filepath.Join(handle.CWD, "link")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	if tc.kind == "helper" && tc.parentAllowed == nil {
		tc.parentAllowed = tc.allowed
	}
	contextValue := map[string]any{"kind": "projectrun-task/v1", "managerId": "docs-manager", "phase": "work", "allowedWritePaths": tc.allowed, "excludedWritePaths": tc.excluded}
	if tc.kind == "helper" {
		parent := map[string]any{"kind": "projectrun-task/v1", "managerId": "docs-manager", "phase": "work", "allowedWritePaths": tc.parentAllowed, "excludedWritePaths": tc.excluded}
		contextValue = map[string]any{"kind": "projectrun-helper/v1", "managerId": "docs-manager", "helperDepth": 1, "helperAccounting": "partial", "task": "edit the assigned documentation", "allowedWritePaths": tc.allowed, "excludedWritePaths": tc.excluded, "parentContext": parent}
	}
	contextJSON, err := json.Marshal(contextValue)
	if err != nil {
		t.Fatal(err)
	}
	events := []Event{}
	options := Options{OnEvent: func(_ context.Context, e Event) error { events = append(events, e); return nil }}
	if tc.kind == "helper" {
		options.BeforeStart = func(context.Context, agentexec.RoleStartRequest) error { return nil }
	}
	base, shared, request, runOptions := fixture(t, tc.mode, options)
	if tc.profile != "" {
		config := base.config
		config.PermissionProfile = tc.profile
		base, err = NewAdapter(config, options)
		if err != nil {
			t.Fatal(err)
		}
	}
	request.Role = tc.role
	request.SourceRevision = workspaceRequest.BaseSHA
	request.Context = contextJSON
	runOptions.Workspace = &handle
	changes := make([]fileUpdateChange, len(tc.paths))
	for index, path := range tc.paths {
		kind := "update"
		if strings.HasPrefix(path, "add:") {
			path, kind = strings.TrimPrefix(path, "add:"), "add"
		} else if strings.HasPrefix(path, "delete:") {
			path, kind = strings.TrimPrefix(path, "delete:"), "delete"
		}
		if !filepath.IsAbs(path) {
			if strings.HasPrefix(path, "foreign:") {
				path = filepath.Join(t.TempDir(), filepath.FromSlash(strings.TrimPrefix(path, "foreign:")))
			} else if strings.HasPrefix(path, "physical-alias:") {
				path = `\\?\` + strings.TrimSuffix(handle.CWD, `\`) + `\` + filepath.FromSlash(strings.TrimPrefix(path, "physical-alias:"))
			} else {
				path = handle.CWD + string(filepath.Separator) + filepath.FromSlash(path)
			}
		}
		changes[index] = fileUpdateChange{Path: path}
		changes[index].Kind.Type = kind
		if tc.moveKind == kind {
			movePath := "moved.md"
			changes[index].Kind.MovePath = &movePath
		}
	}
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		t.Fatal(err)
	}
	if tc.moveKind != "" {
		var raw []map[string]any
		if err := json.Unmarshal(changesJSON, &raw); err != nil {
			t.Fatal(err)
		}
		raw[0]["kind"].(map[string]any)["move_path"] = "moved.md"
		changesJSON, err = json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MARKITECT_P04_CHANGES", string(changesJSON))
	t.Setenv("MARKITECT_P04_APPROVAL_VARIANT", tc.variant)
	result, runErr := base.Run(context.Background(), shared, request, runOptions)
	if runErr == nil && result.Receipt.Lifecycle.State != "completed" {
		t.Fatalf("accepted fixture did not complete: %+v", result.Receipt.Lifecycle)
	}
	decision := ""
	for _, event := range events {
		if event.Method == "markitect/fileChangeApproval/decision" {
			var value struct {
				Decision string `json:"decision"`
				Scope    string `json:"scope"`
			}
			if json.Unmarshal(event.Params, &value) == nil {
				decision = value.Decision
				if value.Scope != "delegated-write-scope" {
					t.Errorf("approval decision misrepresented its authority: %#v", value)
				}
			}
		}
	}
	return decision, runErr
}

func TestFileChangeApprovalRequiresExactOwnedDelegatedScope(t *testing.T) {
	tests := []approvalTestCase{
		{name: "manager update", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, wantDecision: "accept"},
		{name: "protocol nullable optional fields", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "nullable-optional-fields", wantDecision: "accept"},
		{name: "child patch telemetry cannot disrupt parent approval", mode: "filechange-childtelemetry-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, wantDecision: "accept"},
		{name: "new owned file under new directories", mode: "filechange-accept", paths: []string{"add:new/nested/README.md"}, allowed: []string{"new/"}, wantDecision: "accept"},
		{name: "helper own scope", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, kind: "helper", parentAllowed: []string{"README.md"}, wantDecision: "accept"},
		{name: "helper cannot borrow parent scope", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"docs/"}, kind: "helper", parentAllowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "one out of scope change rejects whole patch", mode: "filechange-accept", paths: []string{"README.md", "add:src/other.go"}, allowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "excluded scope overrides allowed", mode: "filechange-accept", paths: []string{"add:src/other.go"}, allowed: []string{"src/"}, excluded: []string{"src/"}, wantDecision: "decline", wantErr: true},
		{name: "unverifiable physical alias", mode: "filechange-accept", paths: []string{"foreign:README.md"}, allowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "protected metadata", mode: "filechange-accept", paths: []string{"add:.markitect/policy.yaml"}, allowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "traversal", mode: "filechange-accept", paths: []string{`..\outside.md`}, allowed: []string{"outside.md"}, wantDecision: "decline", wantErr: true},
		{name: "wrong thread", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "wrong-thread", wantDecision: "decline", wantErr: true},
		{name: "wrong turn", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "wrong-turn", wantDecision: "decline", wantErr: true},
		{name: "wrong item", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "wrong-item", wantDecision: "decline", wantErr: true},
		{name: "no session grant", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "grant-root", wantDecision: "decline", wantErr: true},
		{name: "empty grant root still explicit", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "empty-grant-root", wantDecision: "decline", wantErr: true},
		{name: "unknown approval field", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "unknown-field", wantDecision: "decline", wantErr: true},
		{name: "move path on add rejected", mode: "filechange-accept", paths: []string{"add:README.md"}, allowed: []string{"README.md"}, moveKind: "add", wantDecision: "decline", wantErr: true},
		{name: "move path on delete rejected", mode: "filechange-accept", paths: []string{"delete:README.md"}, allowed: []string{"README.md"}, moveKind: "delete", wantDecision: "decline", wantErr: true},
		{name: "required timestamp", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, variant: "missing-start", wantDecision: "decline", wantErr: true},
		{name: "read only requested profile", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, profile: ":read-only", wantDecision: "decline", wantErr: true},
		{name: "effective read only sandbox", mode: "filechange-readonly-sandbox", paths: []string{"README.md"}, allowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "unknown effective sandbox", mode: "filechange-unknown-sandbox", paths: []string{"README.md"}, allowed: []string{"README.md"}, wantDecision: "decline", wantErr: true},
		{name: "non executor role", mode: "filechange-accept", paths: []string{"README.md"}, allowed: []string{"README.md"}, role: agentexec.RoleVerifier, wantDecision: "decline", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := runApprovalFixture(t, tc)
			if decision != tc.wantDecision {
				t.Fatalf("decision = %q, want %q (err=%v)", decision, tc.wantDecision, err)
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("run error = %v, wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr && !errors.Is(err, ErrApprovalRequired) {
				t.Fatalf("rejected approval did not fail closed with ErrApprovalRequired: %v", err)
			}
		})
	}
}

func TestFileChangeApprovalParamsRejectDuplicateAndUnknownFields(t *testing.T) {
	for _, raw := range []string{
		`{"threadId":"thread-1","threadId":"thread-1","turnId":"turn-1","itemId":"patch-1","startedAtMs":1}`,
		`{"threadId":"thread-1","turnId":"turn-1","itemId":"patch-1","startedAtMs":1,"futurePermission":true}`,
		`{"threadId":"thread-1","turnId":"turn-1","itemId":"patch-1","startedAtMs":1,"grantRoot":"C:\\\\outside","GrantRoot":null}`,
	} {
		var params fileChangeApprovalParams
		if err := decodeFileChangeApprovalParams([]byte(raw), &params); err == nil {
			t.Fatalf("accepted unpinned approval params: %s", raw)
		}
	}
	var nullable fileChangeApprovalParams
	if err := decodeFileChangeApprovalParams([]byte(`{"threadId":"thread-1","turnId":"turn-1","itemId":"patch-1","startedAtMs":1,"reason":null,"grantRoot":null}`), &nullable); err != nil {
		t.Fatalf("rejected pinned nullable optional fields: %v", err)
	}
}

func TestFileChangeApprovalRejectsSymlinkAndUnverifiablePaths(t *testing.T) {
	decision, err := runApprovalFixture(t, approvalTestCase{name: "symlink", mode: "filechange-accept", paths: []string{"add:link/new.go"}, allowed: []string{"link/"}})
	if decision != "decline" || !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("symlink write was not rejected: decision=%q err=%v", decision, err)
	}
}

func TestFileChangeApprovalAcceptsVerifiedWindowsExtendedPathAlias(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows extended-path alias is platform-specific")
	}
	decision, err := runApprovalFixture(t, approvalTestCase{name: "extended-path-alias", mode: "filechange-accept", paths: []string{"physical-alias:README.md"}, allowed: []string{"README.md"}, wantDecision: "accept"})
	if decision != "accept" || err != nil {
		t.Fatalf("verified same-directory Windows spelling alias was not accepted: decision=%q err=%v", decision, err)
	}
}

func TestWindowsApprovalAliasRejectsUNCWithoutResolvingIt(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path namespaces are platform-specific")
	}
	root := t.TempDir()
	if _, err := workspaceRelativeChangePath(root, `\\server\share\README.md`, "add"); err == nil {
		t.Fatal("foreign UNC proposal was treated as an owned local alias")
	}
}

func TestAppServerReceiptBindsActualProcessArguments(t *testing.T) {
	command, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, backend := range []WindowsSandboxBackend{"", WindowsSandboxBackendMXC} {
		config := Config{WindowsSandboxBackend: backend}
		args := appServerArgs(config)
		receipt := agentexec.Receipt{}
		if err := bindReceipt(&receipt, agentexec.Config{Command: command}, nil, args); err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(struct {
			Command string
			Args    []string
		}{command, args})
		if receipt.CommandDigest != digest(encoded) {
			t.Fatalf("command digest omitted the actual app-server args: backend=%q args=%v digest=%s", backend, args, receipt.CommandDigest)
		}
		if backend == WindowsSandboxBackendMXC && (len(args) < 2 || args[0] != "-c" || args[1] != "windows.sandbox=mxc") {
			t.Fatalf("receipt test did not exercise configured MXC process args: %v", args)
		}
	}
}

func TestRequestedWindowsSandboxBackendReceipt(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the mxc backend is Windows-only")
	}
	base, shared, request, options := fixture(t, "success", Options{})
	config := base.config
	config.WindowsSandboxBackend = "mxc"
	adapter, err := NewAdapter(config, base.options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Run(context.Background(), shared, request, options)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Receipt.Lifecycle.Requested.WindowsSandboxBackend; got != "mxc" {
		t.Fatalf("requested backend = %q, want mxc", got)
	}
	if result.Receipt.Lifecycle.Effective != nil && result.Receipt.Lifecycle.Effective.WindowsSandboxBackend != "" {
		t.Fatalf("adapter inferred an effective backend without authoritative readback: %#v", result.Receipt.Lifecycle.Effective)
	}
}

func TestAdapterLifecycle(t *testing.T) {
	for _, mode := range []string{"success", "different-session", "failed", "interrupted", "helper", "malformed-response", "malformed-wire", "transport-loss", "model-mismatch", "request-failure", "version", "event-limit", "approval"} {
		t.Run(mode, func(t *testing.T) {
			var saved RecoveryHandle
			reserved := 0
			a, cfg, req, opts := fixture(t, mode, Options{BeforeStart: func(_ context.Context, r agentexec.RoleStartRequest) error { reserved++; return nil }, OnHandle: func(_ context.Context, h RecoveryHandle) error { saved = h; return nil }})
			if mode == "event-limit" {
				a.config.MaxEventBytes = 4096
			}
			result, err := a.Run(context.Background(), cfg, req, opts)
			if mode == "success" || mode == "helper" || mode == "different-session" {
				if err != nil {
					t.Fatal(err)
				}
				if result.Receipt.Lifecycle.State != "completed" || result.Receipt.ConfigDigest == "" || saved.TurnID != "turn-1" || reserved != 1 {
					t.Fatalf("bad receipt/handle: %+v", result.Receipt)
				}
				if result.Receipt.Usage != nil {
					t.Fatal("unknown usage invented")
				}
				if mode == "different-session" && result.Receipt.Lifecycle.StartRequests[0].SessionID != "session-other" {
					t.Fatal("thread ID mislabeled as session ID")
				}
				if mode == "helper" {
					if len(result.Receipt.Lifecycle.StartRequests) != 2 || result.Receipt.Lifecycle.StartRequests[1].State != "failed" {
						t.Fatalf("helper failure not counted: %+v", result.Receipt.Lifecycle)
					}
				}
			} else if err == nil {
				t.Fatalf("%s accepted", mode)
			}
			if mode == "transport-loss" && (!errors.Is(err, ErrUncertain) || result.Receipt.Lifecycle.State != "unknown") {
				t.Fatalf("lost dispatch falsely certain: %+v %v", result.Receipt, err)
			}
			if mode == "version" && reserved != 0 {
				t.Fatal("version mismatch started thread")
			}
		})
	}
}

func TestTimeoutInterruptionAndReservation(t *testing.T) {
	for _, mode := range []string{"timeout", "interrupt-confirmed"} {
		t.Run(mode, func(t *testing.T) {
			ctx := newPostDispatchTimeoutContext(context.Background())
			dispatchObserved := false
			a, cfg, req, opts := fixture(t, mode, Options{OnHandle: func(_ context.Context, handle RecoveryHandle) error {
				if handle.TurnDispatched && handle.TurnID != "" {
					dispatchObserved = true
					ctx.arm(100 * time.Millisecond)
				}
				return nil
			}})
			r, err := a.Run(ctx, cfg, req, opts)
			if !dispatchObserved || !ctx.armed() {
				t.Fatal("negative timeout did not start after turn dispatch")
			}
			if err == nil {
				t.Fatal("timeout accepted")
			}
			if mode == "timeout" && (!errors.Is(err, ErrUncertain) || r.Receipt.Lifecycle.State != "unknown") {
				t.Fatalf("uncertain timeout mislabeled: %v %+v", err, r.Receipt)
			}
			if mode == "interrupt-confirmed" && r.Receipt.Lifecycle.State != "interrupted" {
				t.Fatalf("confirmed interruption lost: %v %+v", err, r.Receipt)
			}
		})
	}
	a, cfg, req, opts := fixture(t, "success", Options{BeforeStart: func(context.Context, agentexec.RoleStartRequest) error { return errors.New("budget refused") }})
	r, err := a.Run(context.Background(), cfg, req, opts)
	if err == nil || r.Receipt.Lifecycle.SessionID != "" {
		t.Fatal("refused budget launched thread")
	}
}

// postDispatchTimeoutContext gives Run an ordinary unbounded context while it
// probes the executable and completes handshake. The test arms a finite
// DeadlineExceeded only after OnHandle proves turn dispatch, so this tests the
// actual uncertain/interrupted turn path under slow test-suite load.
type postDispatchTimeoutContext struct {
	parent   context.Context
	done     chan struct{}
	mu       sync.Mutex
	deadline time.Time
	expired  bool
}

func newPostDispatchTimeoutContext(parent context.Context) *postDispatchTimeoutContext {
	return &postDispatchTimeoutContext{parent: parent, done: make(chan struct{})}
}

func (c *postDispatchTimeoutContext) arm(timeout time.Duration) {
	c.mu.Lock()
	if !c.deadline.IsZero() {
		c.mu.Unlock()
		return
	}
	c.deadline = time.Now().Add(timeout)
	c.mu.Unlock()
	time.AfterFunc(timeout, func() {
		c.mu.Lock()
		c.expired = true
		close(c.done)
		c.mu.Unlock()
	})
}

func (c *postDispatchTimeoutContext) armed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.deadline.IsZero()
}

func (c *postDispatchTimeoutContext) Deadline() (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deadline, !c.deadline.IsZero()
}

func (c *postDispatchTimeoutContext) Done() <-chan struct{} { return c.done }

func (c *postDispatchTimeoutContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.expired {
		return context.DeadlineExceeded
	}
	return nil
}

func (c *postDispatchTimeoutContext) Value(key any) any { return c.parent.Value(key) }

func TestDynamicToolAndRecovery(t *testing.T) {
	called := 0
	var handle RecoveryHandle
	options := Options{DynamicTools: []DynamicTool{{Type: "function", Name: "markitect_start_helper", Description: "Start bounded helper after reservation", InputSchema: json.RawMessage(`{"type":"object"}`)}}, MaxToolCalls: 1, ToolTimeout: time.Second, HandleToolCall: func(_ context.Context, call ToolCall) (ToolResult, error) {
		called++
		return ToolResult{Success: true, Text: "reserved helper result"}, nil
	}, OnHandle: func(_ context.Context, h RecoveryHandle) error { handle = h; return nil }}
	a, cfg, req, opts := fixture(t, "dynamic", options)
	r, err := a.Run(context.Background(), cfg, req, opts)
	if err != nil || called != 1 || r.Receipt.Lifecycle.State != "completed" {
		t.Fatalf("dynamic dispatch failed: %v %d %+v", err, called, r.Receipt)
	}
	t.Setenv("MARKITECT_P04_MODE", "success")
	b, _ := json.Marshal(handle)
	t.Setenv("MARKITECT_P04_RECOVERY", string(b))
	r, err = a.Recover(context.Background(), cfg, handle)
	if err != nil || r.Response.RunID != handle.Invocation.RunID {
		t.Fatalf("owned recovery failed: %v %+v", err, r)
	}
	handle.TurnID = ""
	_, err = a.Recover(context.Background(), cfg, handle)
	if !errors.Is(err, ErrUncertain) {
		t.Fatalf("lost turn ID guessed: %v", err)
	}
}

func TestRecoveryRetainsOriginalRootAndObservedChildren(t *testing.T) {
	for _, mode := range []string{"recovery-completed", "recovery-running", "recovery-missing", "recovery-children"} {
		t.Run(mode, func(t *testing.T) {
			var handle RecoveryHandle
			reservations := 0
			a, cfg, req, opts := fixture(t, "success", Options{BeforeStart: func(context.Context, agentexec.RoleStartRequest) error { reservations++; return nil }, OnHandle: func(_ context.Context, h RecoveryHandle) error { handle = h; return nil }})
			a.config.Helpers.MaxDepth = 2
			if _, err := a.Run(context.Background(), cfg, req, opts); err != nil {
				t.Fatal(err)
			}
			wire, _ := json.Marshal(handle)
			t.Setenv("MARKITECT_P04_RECOVERY", string(wire))
			t.Setenv("MARKITECT_P04_MODE", mode)
			result, err := a.Recover(context.Background(), cfg, handle)
			life := result.Receipt.Lifecycle
			if life == nil || len(life.StartRequests) == 0 {
				t.Fatalf("original root absent: %v %+v", err, result.Receipt)
			}
			root := life.StartRequests[0]
			if root.RequestID != handle.Invocation.RunID || root.Role != req.Role || root.SessionID != handle.SessionID || root.Model != cfg.Model || root.ReasoningEffort != "high" {
				t.Fatalf("original root binding changed: %+v", root)
			}
			if reservations != 1 || life.Accounting != "partial" || result.Receipt.Usage != nil || life.Effective.InstructionDigest != "" {
				t.Fatal("recovery invented new start or complete/usage/instruction evidence")
			}
			if mode == "recovery-completed" || mode == "recovery-children" {
				if err != nil || root.State != "completed" || life.State != "completed" {
					t.Fatalf("terminal original root not recovered: %v %+v", err, life)
				}
			} else if !errors.Is(err, ErrUncertain) || root.State != "unknown" || life.State != "unknown" {
				t.Fatalf("nonterminal original became certain: %v %+v", err, life)
			}
			if mode == "recovery-completed" && len(life.StartRequests) != 1 {
				t.Fatal("no-child recovery fabricated children")
			}
			if mode == "recovery-children" {
				if len(life.StartRequests) != 3 || life.StartRequests[1].SessionID != "child-session" || life.StartRequests[1].State != "completed" || life.StartRequests[2].State != "failed" || life.StartRequests[2].ParentSessionID != "child-session" || life.StartRequests[2].SessionID != "" {
					t.Fatalf("nested/failure evidence lost or invented: %+v", life.StartRequests)
				}
			}
			if mode == "recovery-running" && (len(life.StartRequests) != 3 || life.StartRequests[2].State != "failed") {
				t.Fatalf("running-turn child history dropped: %+v", life.StartRequests)
			}
		})
	}
}
