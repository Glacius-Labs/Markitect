package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
)

var _ Invoker = (*TransportInvoker)(nil)

func TestTransportInvokerRoutesFingerprintByTransportWithoutStartingAnything(t *testing.T) {
	invoker := NewTransportInvoker(codexappserver.Options{})
	processConfig := processFingerprintConfig(t)
	processFingerprint, err := invoker.Fingerprint(processConfig)
	if err != nil {
		t.Fatalf("process fingerprint: %v", err)
	}
	wantProcessFingerprint, err := (ProcessInvoker{}).Fingerprint(processConfig)
	if err != nil || processFingerprint != wantProcessFingerprint {
		t.Fatalf("process route fingerprint = %q, want %q, err=%v", processFingerprint, wantProcessFingerprint, err)
	}

	nativeConfig := nativeFingerprintConfig(t)
	nativeFingerprint, err := invoker.Fingerprint(nativeConfig)
	if err != nil || nativeFingerprint == "" || nativeFingerprint == processFingerprint {
		t.Fatalf("native route fingerprint = %q, err=%v", nativeFingerprint, err)
	}

	toolOptions := codexappserver.Options{
		BeforeStart: func(context.Context, agentexec.RoleStartRequest) error { return nil },
		OnHandle:    func(context.Context, codexappserver.RecoveryHandle) error { return nil },
		OnEvent:     func(context.Context, codexappserver.Event) error { return nil },
		DynamicTools: []codexappserver.DynamicTool{{
			Type: "function", Name: "markitect_inspect", InputSchema: json.RawMessage(`{"type":"object"}`),
		}},
		HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			return codexappserver.ToolResult{}, nil
		},
		MaxToolCalls: 1, ToolTimeout: time.Second,
	}
	withHostOptions := NewTransportInvoker(toolOptions)
	optionBoundFingerprint, err := withHostOptions.Fingerprint(nativeConfig)
	if err != nil || optionBoundFingerprint == nativeFingerprint {
		t.Fatalf("Host App Server options were not included in fingerprint: %q/%q err=%v", nativeFingerprint, optionBoundFingerprint, err)
	}
	if withHostOptions.appServerOptions.BeforeStart == nil || withHostOptions.appServerOptions.OnHandle == nil ||
		withHostOptions.appServerOptions.OnEvent == nil || len(withHostOptions.appServerOptions.DynamicTools) != 1 {
		t.Fatal("constructor did not retain Host-provided App Server options")
	}
}

func TestPlanRuntimeFingerprintsMatchDefaultNativeCLIInvokerForEveryRole(t *testing.T) {
	for _, helpersEnabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "helpers-disabled", true: "helpers-enabled"}[helpersEnabled], func(t *testing.T) {
			root := makeProjectRunFixture(t)
			writeE2E(t, root, "AGENTS.md", "Pinned instructions for native project-run fingerprint coverage.\n")
			gitE2E(t, root, "add", "AGENTS.md")
			gitE2E(t, root, "commit", "-m", "add pinned native instructions")
			instructionPath, err := filepath.Abs(filepath.Join(root, "AGENTS.md"))
			if err != nil {
				t.Fatal(err)
			}
			instructionBytes, err := os.ReadFile(instructionPath)
			if err != nil {
				t.Fatal(err)
			}
			updateE2ERuntime(t, root, func(runtime *Runtime) {
				for managerID, agent := range runtime.Agents {
					native := appServerAgent(t)
					native.ProviderVersion = codexappserver.SupportedProviderVersion
					native.Pricing = agent.Pricing
					native.Environment = append([]string(nil), agent.Environment...)
					if helpersEnabled {
						native.AppServer.Helpers = AppServerHelpers{Enabled: true, MaxStartRequests: 2, MaxDepth: 1}
					} else {
						native.AppServer.Helpers = AppServerHelpers{}
					}
					native.RuntimeFiles = []agentexec.RuntimeFile{{Path: instructionPath, Mode: "0644", Digest: "sha256:" + digestBytes(instructionBytes)}}
					runtime.Agents[managerID] = native
				}
				reviewers := map[string]Agent{}
				for managerID, manager := range runtime.Agents {
					reviewer := manager
					reviewer.WorkspaceMode, reviewer.InstructionPaths, reviewer.RuntimeFiles = "", nil, nil
					reviewers[managerID] = reviewer
				}
				runtime.Review = &ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
				verifier := runtime.Agents[e2eManagerID("", "project-owner")]
				verifier.WorkspaceMode, verifier.InstructionPaths, verifier.RuntimeFiles = "", nil, nil
				runtime.Verifier = &verifier
			})

			plan, err := Plan(projectworkHost(), root, identityHead(t, root), PlanRequest{Goal: "Check the default native transport binding."})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			runtime, err := LoadRuntime(root)
			if err != nil {
				t.Fatal(err)
			}
			invoker := NewTransportInvoker(codexappserver.Options{})
			gotDigest, err := runtimeDigestWithInvoker(invoker, runtime)
			if err != nil || gotDigest != plan.RuntimeDigest {
				t.Fatalf("planned runtime digest does not match default CLI invoker: got %q want %q err=%v", gotDigest, plan.RuntimeDigest, err)
			}
			for _, role := range []struct {
				name  string
				agent Agent
				key   string
			}{
				{name: "manager", agent: runtime.Agents[e2eManagerID("", "project-owner")], key: e2eManagerID("", "project-owner")},
				{name: "reviewer", agent: runtime.Review.Agents[e2eManagerID("", "project-owner")], key: "$reviewer:" + e2eManagerID("", "project-owner")},
				{name: "verifier", agent: *runtime.Verifier, key: "$verifier"},
			} {
				config, err := role.agent.AgentConfig()
				if err != nil {
					t.Fatalf("%s AgentConfig: %v", role.name, err)
				}
				want, err := invoker.Fingerprint(config)
				if err != nil || plan.RuntimeAgents[role.key] != want {
					t.Fatalf("%s fingerprint mismatch: got %q want %q err=%v", role.name, plan.RuntimeAgents[role.key], want, err)
				}
			}

			customOptions := codexappserver.Options{
				DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "custom_host_tool", InputSchema: json.RawMessage(`{"type":"object"}`)}},
				HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
					return codexappserver.ToolResult{}, nil
				},
				MaxToolCalls: 1, ToolTimeout: time.Second,
			}
			if drifted, err := runtimeDigestWithInvoker(NewTransportInvoker(customOptions), runtime); err != nil || drifted == plan.RuntimeDigest {
				t.Fatalf("custom App Server options drift did not stale the plan: got %q want != %q err=%v", drifted, plan.RuntimeDigest, err)
			}
			driftedRuntime := runtime
			driftedRuntime.Agents = make(map[string]Agent, len(runtime.Agents))
			for managerID, manager := range runtime.Agents {
				manager.Environment = append(append([]string(nil), manager.Environment...), "MARKITECT_FINGERPRINT_DRIFT")
				driftedRuntime.Agents[managerID] = manager
			}
			if drifted, err := runtimeDigestWithInvoker(invoker, driftedRuntime); err != nil || drifted == plan.RuntimeDigest {
				t.Fatalf("native environment drift did not stale the plan: got %q want != %q err=%v", drifted, plan.RuntimeDigest, err)
			}
		})
	}
}

func TestTransportInvokerRejectsUnsupportedAndMalformedTransportConfigs(t *testing.T) {
	invoker := NewTransportInvoker(codexappserver.Options{})
	unknown := processFingerprintConfig(t)
	unknown.Transport = "other"
	if _, err := invoker.Fingerprint(unknown); err == nil {
		t.Fatal("unsupported transport was accepted")
	}

	native := nativeFingerprintConfig(t)
	malformed := native
	malformed.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","maxEventBytes":1024,"helpers":{"enabled":false},"ignored":true}`)
	if _, err := invoker.Fingerprint(malformed); err == nil {
		t.Fatal("unknown transport config field was accepted")
	}
	malformed = native
	malformed.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","maxEventBytes":1024,"helpers":{"enabled":false}} trailing`)
	if _, err := invoker.Fingerprint(malformed); err == nil {
		t.Fatal("trailing transport config data was accepted")
	}
	malformed = native
	malformed.TransportConfig = nil
	if _, err := invoker.Fingerprint(malformed); err == nil {
		t.Fatal("missing transport config was accepted")
	}
	malformed = native
	malformed.WorkspaceMode = "git"
	if _, err := invoker.Fingerprint(malformed); err == nil {
		t.Fatal("Git workspace mode leaked into the shared adapter fingerprint config")
	}
}

func TestTransportInvokerHelperToolFingerprintAndNormalToolRouting(t *testing.T) {
	fixture := newHelperFixture(t)
	config := nativeFingerprintConfig(t)
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)
	normalCalls := 0
	baseOptions := codexappserver.Options{
		DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "native_read", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			normalCalls++
			return codexappserver.ToolResult{Success: true, Text: "native tool preserved"}, nil
		},
		MaxToolCalls: 10, ToolTimeout: config.Timeout,
	}
	transport := NewTransportInvokerWithHelpers(baseOptions, TransportHelperHost{Workspaces: fixture.options.Workspaces,
		Limits: fixture.options.Limits, MaxStartRequests: 256, Reserve: fixture.reserver.reserve})
	fingerprint, err := transport.Fingerprint(config)
	if err != nil {
		t.Fatalf("fingerprint with helper: %v", err)
	}
	if fingerprint == "" {
		t.Fatal("helper-enabled fingerprint is empty")
	}
	unboundFingerprint, err := NewTransportInvoker(baseOptions).Fingerprint(config)
	if err != nil || unboundFingerprint != fingerprint {
		t.Fatalf("helper fingerprint depends on reservation binding: %q != %q (%v)", unboundFingerprint, fingerprint, err)
	}
	clone := NewTransportInvoker(baseOptions).WithHelperHost(TransportHelperHost{Workspaces: fixture.options.Workspaces,
		Limits: fixture.options.Limits, MaxStartRequests: 256, Reserve: fixture.reserver.reserve})
	cloneFingerprint, err := clone.Fingerprint(config)
	if err != nil || cloneFingerprint != fingerprint {
		t.Fatalf("per-run WithHelperHost changed static fingerprint: %q != %q (%v)", cloneFingerprint, fingerprint, err)
	}
	disabled := config
	disabled.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":false,"maxStartRequests":0,"maxDepth":0},"maxEventBytes":1048576}`)
	disabledOptions, err := transport.optionsWithHelperSpec(disabled, baseOptions)
	if err != nil || helperToolPresent(disabledOptions.DynamicTools) {
		t.Fatalf("disabled native helpers advertised the Host tool: %#v err=%v", disabledOptions.DynamicTools, err)
	}

	request := fixture.options.ParentRequest
	request.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work",
		AllowedWritePaths: fixture.options.ParentScope.AllowedWritePaths, ExcludedWritePaths: fixture.options.ParentScope.ExcludedWritePaths,
		ActiveResponsibilities: fixture.options.ParentScope.ActiveResponsibilities})
	options, session, err := transport.optionsWithHelper(config, request,
		agentexec.RunOptions{Workspace: &fixture.workspace, PrivateLogDirectory: fixture.private})
	if err != nil {
		t.Fatalf("build runtime native options: %v", err)
	}
	if session == nil || len(options.DynamicTools) != 2 || options.DynamicTools[0].Name != "native_read" || options.DynamicTools[1].Name != HelperToolName {
		t.Fatalf("normal tool or Host helper spec missing: %#v", options.DynamicTools)
	}
	toolFingerprint, err := transport.appServerAdapterWithOptions(config, options)
	if err != nil {
		t.Fatal(err)
	}
	gotFingerprint, err := toolFingerprint.Fingerprint(config)
	if err != nil || gotFingerprint != fingerprint {
		t.Fatalf("pre-run fingerprint differs from exact runtime spec: %q != %q (%v)", gotFingerprint, fingerprint, err)
	}
	result, err := options.HandleToolCall(context.Background(), codexappserver.ToolCall{Tool: "native_read"})
	if err != nil || !result.Success || result.Text != "native tool preserved" || normalCalls != 1 {
		t.Fatalf("normal native tool route was replaced: %#v err=%v calls=%d", result, err, normalCalls)
	}
	if session == nil {
		t.Fatal("Host helper session was not constructed")
	}
}

func TestUnboundTransportRejectsHelperAndReviewContextSuppressesHostHelper(t *testing.T) {
	config := nativeFingerprintConfig(t)
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)
	unbound := NewTransportInvoker(codexappserver.Options{})
	options, err := unbound.optionsWithHelperSpec(config, codexappserver.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !helperToolPresent(options.DynamicTools) {
		t.Fatal("helper spec is absent from an unbound fingerprint surface")
	}
	options = rejectUnboundHelper(options)
	if _, err := options.HandleToolCall(context.Background(), helperCall("unbound", `{"task":"write","paths":["src/a.go"]}`)); err == nil || !strings.Contains(err.Error(), "reservation is unavailable") {
		t.Fatalf("unbound helper call was not rejected explicitly: %v", err)
	}

	fixture := newHelperFixture(t)
	baseOptions := codexappserver.Options{
		DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "native_read", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			return codexappserver.ToolResult{Success: true, Text: "read"}, nil
		},
	}
	transport := NewTransportInvokerWithHelpers(baseOptions, TransportHelperHost{Workspaces: fixture.options.Workspaces,
		Limits: fixture.options.Limits, MaxStartRequests: 256, Reserve: fixture.reserver.reserve})
	request := agentexec.Request{Context: json.RawMessage(`{"kind":"projectrun-review/v1"}`)}
	options, session, err := transport.optionsWithHelper(config, request, agentexec.RunOptions{})
	if err != nil {
		t.Fatalf("reviewer options: %v", err)
	}
	if session != nil || helperToolPresent(options.DynamicTools) || len(options.DynamicTools) != 1 || options.DynamicTools[0].Name != "native_read" {
		t.Fatalf("review context advertised Host delegation or lost its native tool: %#v session=%v", options.DynamicTools, session)
	}
	if fixture.reserver.reserveCount != 0 {
		t.Fatalf("review context reserved a Host helper despite suppressing it: %d", fixture.reserver.reserveCount)
	}
	managerWithoutScope := agentexec.Request{Role: agentexec.RoleExecutor}
	managerWithoutScope.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work"})
	options, session, err = transport.optionsWithHelper(config, managerWithoutScope, agentexec.RunOptions{})
	if err != nil || session != nil || helperToolPresent(options.DynamicTools) {
		t.Fatalf("Manager without a nonempty write scope advertised delegation: tools=%#v session=%v err=%v", options.DynamicTools, session, err)
	}
	if fixture.reserver.reserveCount != 0 {
		t.Fatalf("unscoped Manager context reserved a Host helper: %d", fixture.reserver.reserveCount)
	}
	verifierRequest := agentexec.Request{Role: agentexec.RoleVerifier}
	verifierRequest.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work", AllowedWritePaths: []string{"src/"}})
	verifierOptions, verifierSession, err := transport.optionsWithHelper(config, verifierRequest, agentexec.RunOptions{})
	if err != nil || verifierSession != nil || helperToolPresent(verifierOptions.DynamicTools) {
		t.Fatalf("verifier context advertised a Manager-only Host helper: tools=%#v session=%v err=%v", verifierOptions.DynamicTools, verifierSession, err)
	}
	managerRequest := fixture.options.ParentRequest
	managerRequest.Role = agentexec.RoleExecutor
	managerRequest.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work",
		AllowedWritePaths: fixture.options.ParentScope.AllowedWritePaths, ExcludedWritePaths: fixture.options.ParentScope.ExcludedWritePaths,
		ActiveResponsibilities: fixture.options.ParentScope.ActiveResponsibilities})
	staticFingerprint, err := transport.Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	managerFingerprint, err := transport.FingerprintForRequest(config, managerRequest)
	if err != nil || managerFingerprint != staticFingerprint {
		t.Fatalf("Manager request fingerprint drifted from planned helper schema: %q/%q err=%v", managerFingerprint, staticFingerprint, err)
	}
	managerOptions, managerSession, err := transport.optionsWithHelper(config, managerRequest,
		agentexec.RunOptions{Workspace: &fixture.workspace, PrivateLogDirectory: fixture.private})
	if err != nil || managerSession == nil {
		t.Fatalf("scoped Manager helper options: session=%v err=%v", managerSession, err)
	}
	managerAdapter, err := transport.appServerAdapterWithOptions(config, managerOptions)
	if err != nil {
		t.Fatal(err)
	}
	managerActualFingerprint, err := managerAdapter.Fingerprint(config)
	if err != nil || managerActualFingerprint != managerFingerprint {
		t.Fatalf("Manager recovery options drifted from request-aware fingerprint: %q/%q err=%v", managerActualFingerprint, managerFingerprint, err)
	}
	reviewFingerprint, err := transport.FingerprintForRequest(config, request)
	if err != nil || reviewFingerprint == staticFingerprint {
		t.Fatalf("review request fingerprint did not bind suppressed helper tool: %q/%q err=%v", reviewFingerprint, staticFingerprint, err)
	}
	reviewOptions, _, err := transport.optionsWithHelper(config, request, agentexec.RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	reviewAdapter, err := transport.appServerAdapterWithOptions(config, reviewOptions)
	if err != nil {
		t.Fatal(err)
	}
	reviewActualFingerprint, err := reviewAdapter.Fingerprint(config)
	if err != nil || reviewActualFingerprint != reviewFingerprint {
		t.Fatalf("review run/recovery options drifted from request-aware fingerprint: %q/%q err=%v", reviewActualFingerprint, reviewFingerprint, err)
	}
}

func TestChildTransportSuppressesOnlyHostHelperTool(t *testing.T) {
	config := nativeFingerprintConfig(t)
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)
	options := codexappserver.Options{DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "native_read", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
			return codexappserver.ToolResult{}, nil
		},
		MaxToolCalls: 4, ToolTimeout: config.Timeout}
	child := NewTransportInvokerWithoutHostHelpers(options)
	fingerprint, err := child.Fingerprint(config)
	if err != nil || fingerprint == "" {
		t.Fatalf("child App Server fingerprint: %q %v", fingerprint, err)
	}
	if helperToolPresent(child.appServerOptions.DynamicTools) {
		t.Fatal("child transport retained the recursive Host helper tool")
	}
	adapter, err := child.appServerAdapterWithOptions(config, child.appServerOptions)
	if err != nil {
		t.Fatalf("child native helper policy should remain valid: %v", err)
	}
	if adapter == nil {
		t.Fatal("child native App Server adapter is absent")
	}
}

func TestMergeHelperAccountingUsesHostRequestAndObservedNestedRequestsOnce(t *testing.T) {
	session := &HelperSession{
		requests: []agentexec.RoleStartRequest{{RequestID: "host-request", ParentSessionID: "parent-session", SessionID: "child-session", Role: "helper", State: "completed"}},
		receipts: []agentexec.Receipt{{Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "completed", Accounting: "partial",
			StartRequests: []agentexec.RoleStartRequest{
				{RequestID: "child-root", SessionID: "child-session", Role: "executor", State: "started"},
				{RequestID: "nested-1", ParentSessionID: "child-session", SessionID: "nested-session", Role: "reviewer", State: "completed"},
			}}}},
		protocolStarts: map[string]bool{"host-request": true}, receiptByRequest: map[string]bool{"host-request": true},
	}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "completed", Accounting: "complete",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: "parent-root", Role: "executor", State: "started"}}}}
	mergeHelperAccounting(&receipt, session)
	lifecycle := receipt.Lifecycle
	if lifecycle.Accounting != "partial" || len(lifecycle.StartRequests) != 3 {
		t.Fatalf("helper accounting not merged as lower bound: %#v", lifecycle)
	}
	if lifecycle.StartRequests[1].RequestID != "host-request" || lifecycle.StartRequests[2].RequestID != "nested-1" {
		t.Fatalf("child protocol root double-counted or nested request lost: %#v", lifecycle.StartRequests)
	}
}

func TestMergeHelperAccountingPreservesKnownPrelaunchFailureWithoutInventingChild(t *testing.T) {
	session := &HelperSession{requests: []agentexec.RoleStartRequest{{RequestID: "failed-before-child", ParentSessionID: "parent-session", Role: "helper", State: "failed"}}}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "completed", Accounting: "complete",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: "parent-root", Role: "executor", State: "started"}}}}
	mergeHelperAccounting(&receipt, session)
	if receipt.Lifecycle.Accounting != "complete" || len(receipt.Lifecycle.StartRequests) != 2 || receipt.Lifecycle.StartRequests[1].RequestID != "failed-before-child" {
		t.Fatalf("prelaunch failure was dropped or fabricated a child lifecycle: %#v", receipt.Lifecycle)
	}
}

func TestMergeHelperAccountingMarksDispatchedMissingReceiptPartial(t *testing.T) {
	session := &HelperSession{requests: []agentexec.RoleStartRequest{{RequestID: "unknown-child", ParentSessionID: "parent-session", Role: "helper", State: "unknown"}},
		protocolStarts: map[string]bool{"unknown-child": true}}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "completed", Accounting: "complete"}}
	mergeHelperAccounting(&receipt, session)
	if receipt.Lifecycle.Accounting != "partial" || len(receipt.Lifecycle.StartRequests) != 1 {
		t.Fatalf("dispatched child without receipt was claimed complete: %#v", receipt.Lifecycle)
	}
}

func TestMergeHelperAccountingDoesNotUpgradeUnavailableLifecycle(t *testing.T) {
	session := &HelperSession{requests: []agentexec.RoleStartRequest{{RequestID: "helper-request", ParentSessionID: "parent-session", Role: "helper", State: "failed"}}}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: TransportCodexAppServer, State: "unknown", Accounting: "unavailable"}}
	mergeHelperAccounting(&receipt, session)
	if receipt.Lifecycle.Accounting != "unavailable" || receipt.Lifecycle.State != "unknown" || len(receipt.Lifecycle.StartRequests) != 1 {
		t.Fatalf("unavailable lifecycle was upgraded or state fabricated: %#v", receipt.Lifecycle)
	}
}

func processFingerprintConfig(t *testing.T) agentexec.Config {
	t.Helper()
	command, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	emptyEnvironment := []string{}
	return agentexec.Config{
		Command: command, Model: "fixture-model", ProviderVersion: "fixture/1",
		Timeout: time.Minute, MaxStdoutBytes: 1 << 20, MaxStderrBytes: 1 << 20,
		EnvironmentAllowlist: &emptyEnvironment,
	}
}

func nativeFingerprintConfig(t *testing.T) agentexec.Config {
	t.Helper()
	agent := appServerAgent(t)
	agent.ProviderVersion = codexappserver.SupportedProviderVersion
	config, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	return config
}
