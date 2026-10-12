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
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectworkspace"
)

var _ Invoker = (*projectexecutor.Executor)(nil)

func TestProjectExecutorRoutesFingerprintByTransportWithoutStartingAnything(t *testing.T) {
	invoker := projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{})
	processConfig := processFingerprintConfig(t)
	processFingerprint, err := invoker.Fingerprint(processConfig)
	if err != nil {
		t.Fatalf("process fingerprint: %v", err)
	}
	wantProcessFingerprint, err := (projectexecutor.ProcessInvoker{}).Fingerprint(processConfig)
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
	withHostOptions := projectexecutor.New(toolOptions, NativeHelperPolicy{})
	optionBoundFingerprint, err := withHostOptions.Fingerprint(nativeConfig)
	if err != nil || optionBoundFingerprint == nativeFingerprint {
		t.Fatalf("Host App Server options were not included in fingerprint: %q/%q err=%v", nativeFingerprint, optionBoundFingerprint, err)
	}
	if toolOptions.BeforeStart == nil || toolOptions.OnHandle == nil ||
		toolOptions.OnEvent == nil || len(toolOptions.DynamicTools) != 1 {
		t.Fatal("executor composition input did not preserve Host-provided App Server options")
	}
}

func TestProjectExecutorRecoveryRequiresExactDispatchedBindingBeforeJournalOrProcess(t *testing.T) {
	invoker := projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{})
	config := nativeFingerprintConfig(t)
	request := agentexec.Request{Role: agentexec.RoleExecutor, SourceRevision: strings.Repeat("a", 40),
		ModelDigest: "sha256:" + strings.Repeat("b", 64), ModulePin: "module@1", ProjectionID: "projection",
		ScopeIDs: []string{"source"}, PolicyIDs: []string{}, Context: json.RawMessage(`{}`), Artifacts: []agentexec.Artifact{}}
	invocation, _, err := agentexec.PrepareInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := invoker.Fingerprint(config)
	if err != nil {
		t.Fatal(err)
	}
	workspace := projectworkspace.Handle{ID: "0123456789abcdef0123456789abcdef", CWD: filepath.Join(t.TempDir(), "repo"), BaseSHA: request.SourceRevision}
	handle := codexappserver.RecoveryHandle{Protocol: "codex-app-server/fixture", Fingerprint: fingerprint,
		Invocation: invocation, Workspace: workspace, ThreadID: "thread-1", SessionID: "session-1", TurnID: "turn-1", TurnDispatched: true}
	private := filepath.Join(t.TempDir(), "private")
	options := agentexec.RunOptions{Workspace: &workspace, PrivateLogDirectory: private}

	wrongWorkspace := handle
	wrongWorkspace.Workspace.BaseSHA = strings.Repeat("c", 40)
	if _, err := invoker.Recover(context.Background(), config, wrongWorkspace, options); err == nil {
		t.Fatal("recovery accepted a handle for a different owned workspace")
	}

	missingTurn := handle
	missingTurn.TurnID = ""
	if _, err := invoker.Recover(context.Background(), config, missingTurn, options); err == nil {
		t.Fatal("recovery accepted an ambiguous dispatched request without its exact turn ID")
	}

	wrongProtocol := handle
	wrongProtocol.Protocol = "codex-app-server/older-protocol"
	if _, err := invoker.Recover(context.Background(), config, wrongProtocol, options); err == nil {
		t.Fatal("recovery accepted a handle from a different protocol binding")
	}
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatalf("wrong protocol created a recovery journal before rejection: stat err=%v", err)
	}

	wrongConfig := handle
	wrongConfig.Fingerprint = "another-config"
	if _, err := invoker.Recover(context.Background(), config, wrongConfig, options); err == nil {
		t.Fatal("recovery accepted a handle from a different native configuration")
	}
	if _, err := os.Stat(private); !os.IsNotExist(err) {
		t.Fatalf("invalid recovery binding created a journal or reached process startup: stat err=%v", err)
	}
}

func TestPlanBindsNativeMethodPolicySeparatelyFromPerRoleExecutorFingerprints(t *testing.T) {
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
			updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
				for managerID, agent := range runtime.Agents {
					native := appServerAgent(t)
					native.ProviderVersion = codexappserver.SupportedProviderVersion
					native.Pricing = agent.Pricing
					native.Environment = append([]string(nil), agent.Environment...)
					if helpersEnabled {
						native.AppServer.Helpers = projectconfig.AppServerHelpers{Enabled: true, MaxStartRequests: 2, MaxDepth: 1}
					} else {
						native.AppServer.Helpers = projectconfig.AppServerHelpers{}
					}
					native.RuntimeFiles = []agentexec.RuntimeFile{{Path: instructionPath, Mode: "0644", Digest: "sha256:" + digestBytes(instructionBytes)}}
					runtime.Agents[managerID] = native
				}
				reviewers := map[string]projectconfig.Agent{}
				for managerID, manager := range runtime.Agents {
					reviewer := manager
					reviewer.WorkspaceMode, reviewer.InstructionPaths, reviewer.RuntimeFiles = "", nil, nil
					reviewers[managerID] = reviewer
				}
				runtime.Review = &projectconfig.ReviewConfig{Agents: reviewers, MaxRounds: 1, MaxManagerRounds: 1}
				verifier := runtime.Agents[e2eManagerID("", "project-owner")]
				verifier.WorkspaceMode, verifier.InstructionPaths, verifier.RuntimeFiles = "", nil, nil
				runtime.Verifier = &verifier
			})

			revision := identityHead(t, root)
			project := processRuntimeProject(t, root, revision, false)
			host := processRuntimeHost(root, project)
			plan, err := Plan(host, root, revision, PlanRequest{Goal: "Check the default native transport binding."})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if plan.RuntimeBindingVersion != methodRuntimeBindingVersion || len(plan.RuntimeAgents) != 0 {
				t.Fatalf("new plan must carry method binding v%d without plan-time executor fingerprints: version=%d fingerprints=%v", methodRuntimeBindingVersion, plan.RuntimeBindingVersion, plan.RuntimeAgents)
			}
			runtime, err := projectconfig.LoadRuntime(root)
			if err != nil {
				t.Fatal(err)
			}
			invoker := projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{})
			gotDigest, err := methodPolicyDigest(runtime)
			if err != nil || gotDigest != plan.RuntimeDigest {
				t.Fatalf("planned runtime digest does not match method policy: got %q want %q err=%v", gotDigest, plan.RuntimeDigest, err)
			}
			customOptions := codexappserver.Options{
				DynamicTools: []codexappserver.DynamicTool{{Type: "function", Name: "custom_host_tool", InputSchema: json.RawMessage(`{"type":"object"}`)}},
				HandleToolCall: func(context.Context, codexappserver.ToolCall) (codexappserver.ToolResult, error) {
					return codexappserver.ToolResult{}, nil
				},
				MaxToolCalls: 1, ToolTimeout: time.Second,
			}
			customInvoker := projectexecutor.New(customOptions, NativeHelperPolicy{})
			for _, role := range []struct {
				name  string
				agent projectconfig.Agent
			}{
				{name: "manager", agent: runtime.Agents[e2eManagerID("", "project-owner")]},
				{name: "reviewer", agent: runtime.Review.Agents[e2eManagerID("", "project-owner")]},
				{name: "verifier", agent: *runtime.Verifier},
			} {
				config, err := role.agent.AgentConfig()
				if err != nil {
					t.Fatalf("%s AgentConfig: %v", role.name, err)
				}
				defaultFingerprint, err := invoker.Fingerprint(config)
				if err != nil || defaultFingerprint == "" {
					t.Fatalf("%s default executor fingerprint is unavailable: %q err=%v", role.name, defaultFingerprint, err)
				}
				customFingerprint, err := customInvoker.Fingerprint(config)
				if err != nil || customFingerprint == "" || customFingerprint == defaultFingerprint {
					t.Fatalf("%s custom executor identity was not distinguished at invocation binding: default=%q custom=%q err=%v", role.name, defaultFingerprint, customFingerprint, err)
				}
			}
			if drifted, err := runtimeDigestWithInvoker(customInvoker, runtime); err != nil || drifted != plan.RuntimeDigest {
				t.Fatalf("executor implementation change altered method policy digest: got %q want %q err=%v", drifted, plan.RuntimeDigest, err)
			}
			policyDrift := runtime
			policyDrift.Agents = make(map[string]projectconfig.Agent, len(runtime.Agents))
			for managerID, manager := range runtime.Agents {
				if manager.AppServer == nil {
					t.Fatalf("manager %s has no native policy settings", managerID)
				}
				settings := *manager.AppServer
				settings.Helpers = projectconfig.AppServerHelpers{Enabled: !helpersEnabled}
				if !helpersEnabled {
					settings.Helpers.MaxStartRequests, settings.Helpers.MaxDepth = 2, 1
				}
				manager.AppServer = &settings
				policyDrift.Agents[managerID] = manager
			}
			if drifted, err := methodPolicyDigest(policyDrift); err != nil || drifted == plan.RuntimeDigest {
				t.Fatalf("App Server role capability drift did not stale method policy: got %q want != %q err=%v", drifted, plan.RuntimeDigest, err)
			}
			driftedRuntime := runtime
			driftedRuntime.Agents = make(map[string]projectconfig.Agent, len(runtime.Agents))
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

func TestProjectExecutorRejectsUnsupportedAndMalformedTransportConfigs(t *testing.T) {
	invoker := projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{})
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

func TestProjectExecutorHelperPolicyFingerprintAndNormalToolRouting(t *testing.T) {
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
	policy := NativeHelperPolicy{Workspaces: fixture.options.Workspaces, Limits: fixture.options.Limits,
		MaxStartRequests: 256, Reserve: fixture.reserver.reserve}
	transport := projectexecutor.New(baseOptions, policy)
	fingerprint, err := transport.Fingerprint(config)
	if err != nil {
		t.Fatalf("fingerprint with helper: %v", err)
	}
	if fingerprint == "" {
		t.Fatal("helper-enabled fingerprint is empty")
	}
	unboundFingerprint, err := projectexecutor.New(baseOptions, NativeHelperPolicy{}).Fingerprint(config)
	if err != nil || unboundFingerprint != fingerprint {
		t.Fatalf("helper fingerprint depends on reservation binding: %q != %q (%v)", unboundFingerprint, fingerprint, err)
	}
	clone := projectexecutor.New(baseOptions, NativeHelperPolicy{}).WithHelperPolicy(policy)
	cloneFingerprint, err := clone.Fingerprint(config)
	if err != nil || cloneFingerprint != fingerprint {
		t.Fatalf("per-run WithHelperPolicy changed static fingerprint: %q != %q (%v)", cloneFingerprint, fingerprint, err)
	}
	disabled := config
	disabled.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":false,"maxStartRequests":0,"maxDepth":0},"maxEventBytes":1048576}`)
	disabledOptions, err := policy.ToolOptions(disabled, baseOptions)
	if err != nil || helperToolPresent(disabledOptions.DynamicTools) {
		t.Fatalf("disabled native helpers advertised the Host tool: %#v err=%v", disabledOptions.DynamicTools, err)
	}

	request := fixture.options.ParentRequest
	request.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work",
		AllowedWritePaths: fixture.options.ParentScope.AllowedWritePaths, ExcludedWritePaths: fixture.options.ParentScope.ExcludedWritePaths,
		ActiveResponsibilities: fixture.options.ParentScope.ActiveResponsibilities})
	options, session, err := testOptionsForRequest(policy, config, request,
		baseOptions,
		agentexec.RunOptions{Workspace: &fixture.workspace, PrivateLogDirectory: fixture.private})
	if err != nil {
		t.Fatalf("build runtime native options: %v", err)
	}
	if session == nil || len(options.DynamicTools) != 2 || options.DynamicTools[0].Name != "native_read" || options.DynamicTools[1].Name != HelperToolName {
		t.Fatalf("normal tool or Host helper spec missing: %#v", options.DynamicTools)
	}
	gotFingerprint, err := projectexecutor.New(options, nil).Fingerprint(config)
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

func TestNativeHelperRunBudgetPreservesToolCallCapAndFingerprint(t *testing.T) {
	fixture := newHelperFixture(t)
	config := nativeFingerprintConfig(t)
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)
	request := fixture.options.ParentRequest
	request.Role = agentexec.RoleExecutor
	request.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work",
		AllowedWritePaths: fixture.options.ParentScope.AllowedWritePaths, ExcludedWritePaths: fixture.options.ParentScope.ExcludedWritePaths,
		ActiveResponsibilities: fixture.options.ParentScope.ActiveResponsibilities})
	policy := NativeHelperPolicy{Workspaces: fixture.options.Workspaces, Limits: fixture.options.Limits,
		MaxStartRequests: 2, Reserve: fixture.reserver.reserve}
	runOptions := agentexec.RunOptions{Workspace: &fixture.workspace, PrivateLogDirectory: fixture.private}

	for _, test := range []struct {
		name          string
		baseToolCalls int
		wantToolCalls int
	}{
		{name: "default cap", wantToolCalls: 256},
		{name: "explicit cap", baseToolCalls: 7, wantToolCalls: 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := codexappserver.Options{MaxToolCalls: test.baseToolCalls}
			plannedFingerprint, err := projectexecutor.New(base, policy).FingerprintForRequest(config, request)
			if err != nil {
				t.Fatalf("request-aware fingerprint: %v", err)
			}

			runtimeOptions, _, err := testOptionsForRequest(policy, config, request, base, runOptions)
			if err != nil {
				t.Fatalf("bind run helper session: %v", err)
			}
			if runtimeOptions.MaxToolCalls != test.wantToolCalls {
				t.Fatalf("run tool-call cap = %d, want %d", runtimeOptions.MaxToolCalls, test.wantToolCalls)
			}
			runtimeFingerprint, err := projectexecutor.New(runtimeOptions, nil).Fingerprint(config)
			if err != nil || runtimeFingerprint != plannedFingerprint {
				t.Fatalf("run fingerprint = %q, planned %q, err=%v", runtimeFingerprint, plannedFingerprint, err)
			}

			recoveryOptions, err := policy.BindRecovery(config, request, base)
			if err != nil {
				t.Fatalf("bind recovery options: %v", err)
			}
			if recoveryOptions.MaxToolCalls != test.wantToolCalls {
				t.Fatalf("recovery tool-call cap = %d, want %d", recoveryOptions.MaxToolCalls, test.wantToolCalls)
			}
			recoveryFingerprint, err := projectexecutor.New(recoveryOptions, nil).Fingerprint(config)
			if err != nil || recoveryFingerprint != plannedFingerprint {
				t.Fatalf("recovery fingerprint = %q, planned %q, err=%v", recoveryFingerprint, plannedFingerprint, err)
			}
		})
	}
}

func TestUnboundTransportRejectsHelperAndReviewContextSuppressesHostHelper(t *testing.T) {
	config := nativeFingerprintConfig(t)
	config.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":true,"maxStartRequests":2,"maxDepth":1},"maxEventBytes":1048576}`)
	options, err := (NativeHelperPolicy{}).ToolOptions(config, codexappserver.Options{})
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
	policy := NativeHelperPolicy{Workspaces: fixture.options.Workspaces, Limits: fixture.options.Limits,
		MaxStartRequests: 256, Reserve: fixture.reserver.reserve}
	transport := projectexecutor.New(baseOptions, policy)
	request := agentexec.Request{Context: json.RawMessage(`{"kind":"projectrun-review/v1"}`)}
	options, session, err := testOptionsForRequest(policy, config, request, baseOptions, agentexec.RunOptions{})
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
	options, session, err = testOptionsForRequest(policy, config, managerWithoutScope, baseOptions, agentexec.RunOptions{})
	if err != nil || session != nil || helperToolPresent(options.DynamicTools) {
		t.Fatalf("Manager without a nonempty write scope advertised delegation: tools=%#v session=%v err=%v", options.DynamicTools, session, err)
	}
	if fixture.reserver.reserveCount != 0 {
		t.Fatalf("unscoped Manager context reserved a Host helper: %d", fixture.reserver.reserveCount)
	}
	verifierRequest := agentexec.Request{Role: agentexec.RoleVerifier}
	verifierRequest.Context, _ = json.Marshal(nativeTaskContext{Kind: "projectrun-task/v1", ManagerID: "orders", Phase: "work", AllowedWritePaths: []string{"src/"}})
	verifierOptions, verifierSession, err := testOptionsForRequest(policy, config, verifierRequest, baseOptions, agentexec.RunOptions{})
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
	managerOptions, managerSession, err := testOptionsForRequest(policy, config, managerRequest, baseOptions,
		agentexec.RunOptions{Workspace: &fixture.workspace, PrivateLogDirectory: fixture.private})
	if err != nil || managerSession == nil {
		t.Fatalf("scoped Manager helper options: session=%v err=%v", managerSession, err)
	}
	managerActualFingerprint, err := projectexecutor.New(managerOptions, nil).Fingerprint(config)
	if err != nil || managerActualFingerprint != managerFingerprint {
		t.Fatalf("Manager recovery options drifted from request-aware fingerprint: %q/%q err=%v", managerActualFingerprint, managerFingerprint, err)
	}
	reviewFingerprint, err := transport.FingerprintForRequest(config, request)
	if err != nil || reviewFingerprint == staticFingerprint {
		t.Fatalf("review request fingerprint did not bind suppressed helper tool: %q/%q err=%v", reviewFingerprint, staticFingerprint, err)
	}
	reviewOptions, _, err := testOptionsForRequest(policy, config, request, baseOptions, agentexec.RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	reviewActualFingerprint, err := projectexecutor.New(reviewOptions, nil).Fingerprint(config)
	if err != nil || reviewActualFingerprint != reviewFingerprint {
		t.Fatalf("review run/recovery options drifted from request-aware fingerprint: %q/%q err=%v", reviewActualFingerprint, reviewFingerprint, err)
	}
	bareTransport := projectexecutor.New(codexappserver.Options{}, NativeHelperPolicy{Reserve: fixture.reserver.reserve})
	bareReviewOptions, bareSession, err := testOptionsForRequest(NativeHelperPolicy{Reserve: fixture.reserver.reserve}, config, request, codexappserver.Options{}, agentexec.RunOptions{})
	if err != nil || bareSession != nil || len(bareReviewOptions.DynamicTools) != 0 || bareReviewOptions.HandleToolCall != nil || bareReviewOptions.MaxToolCalls != 0 || bareReviewOptions.ToolTimeout != 0 {
		t.Fatalf("filtering the sole Host helper left invalid dynamic-tool options: %#v session=%v err=%v", bareReviewOptions, bareSession, err)
	}
	bareActualFingerprint, err := projectexecutor.New(bareReviewOptions, nil).Fingerprint(config)
	bareExpectedFingerprint, fingerprintErr := bareTransport.FingerprintForRequest(config, request)
	if err != nil || fingerprintErr != nil || bareActualFingerprint != bareExpectedFingerprint {
		t.Fatalf("adapter construction options differ from reviewer fingerprint: actual=%q expected=%q err=%v/%v", bareActualFingerprint, bareExpectedFingerprint, err, fingerprintErr)
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
	child := projectexecutor.NewChild(options)
	fingerprint, err := child.Fingerprint(config)
	if err != nil || fingerprint == "" {
		t.Fatalf("child App Server fingerprint: %q %v", fingerprint, err)
	}
	if helperToolPresent(options.DynamicTools) {
		t.Fatal("child transport retained the recursive Host helper tool")
	}
	if got, err := child.Fingerprint(config); err != nil || got != fingerprint {
		t.Fatalf("child native adapter fingerprint: %q %v", got, err)
	}
}

func TestMergeHelperAccountingUsesHostRequestAndObservedNestedRequestsOnce(t *testing.T) {
	session := &HelperSession{
		requests: []agentexec.RoleStartRequest{{RequestID: "host-request", ParentSessionID: "parent-session", SessionID: "child-session", Role: "helper", State: "completed"}},
		receipts: []agentexec.Receipt{{Lifecycle: &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "completed", Accounting: "partial",
			StartRequests: []agentexec.RoleStartRequest{
				{RequestID: "child-root", SessionID: "child-session", Role: "executor", State: "started"},
				{RequestID: "nested-1", ParentSessionID: "child-session", SessionID: "nested-session", Role: "reviewer", State: "completed"},
			}}}},
		protocolStarts: map[string]bool{"host-request": true}, receiptByRequest: map[string]bool{"host-request": true},
	}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "completed", Accounting: "complete",
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
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "completed", Accounting: "complete",
		StartRequests: []agentexec.RoleStartRequest{{RequestID: "parent-root", Role: "executor", State: "started"}}}}
	mergeHelperAccounting(&receipt, session)
	if receipt.Lifecycle.Accounting != "complete" || len(receipt.Lifecycle.StartRequests) != 2 || receipt.Lifecycle.StartRequests[1].RequestID != "failed-before-child" {
		t.Fatalf("prelaunch failure was dropped or fabricated a child lifecycle: %#v", receipt.Lifecycle)
	}
}

func TestMergeHelperAccountingMarksDispatchedMissingReceiptPartial(t *testing.T) {
	session := &HelperSession{requests: []agentexec.RoleStartRequest{{RequestID: "unknown-child", ParentSessionID: "parent-session", Role: "helper", State: "unknown"}},
		protocolStarts: map[string]bool{"unknown-child": true}}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "completed", Accounting: "complete"}}
	mergeHelperAccounting(&receipt, session)
	if receipt.Lifecycle.Accounting != "partial" || len(receipt.Lifecycle.StartRequests) != 1 {
		t.Fatalf("dispatched child without receipt was claimed complete: %#v", receipt.Lifecycle)
	}
}

func TestMergeHelperAccountingDoesNotUpgradeUnavailableLifecycle(t *testing.T) {
	session := &HelperSession{requests: []agentexec.RoleStartRequest{{RequestID: "helper-request", ParentSessionID: "parent-session", Role: "helper", State: "failed"}}}
	receipt := agentexec.Receipt{Lifecycle: &agentexec.Lifecycle{Provider: projectconfig.TransportCodexAppServer, State: "unknown", Accounting: "unavailable"}}
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

func testOptionsForRequest(policy NativeHelperPolicy, config agentexec.Config, request agentexec.Request, base codexappserver.Options, runOptions agentexec.RunOptions) (codexappserver.Options, *HelperSession, error) {
	options, err := policy.ToolOptions(config, base)
	if err != nil {
		return codexappserver.Options{}, nil, err
	}
	if !policy.Eligible(request) {
		filtered := make([]codexappserver.DynamicTool, 0, len(options.DynamicTools))
		for _, tool := range options.DynamicTools {
			if tool.Name != HelperToolName {
				filtered = append(filtered, tool)
			}
		}
		options.DynamicTools = filtered
		if len(filtered) == 0 {
			options.HandleToolCall, options.MaxToolCalls, options.ToolTimeout = nil, 0, 0
		}
		return options, nil, nil
	}
	bound, err := policy.BindRun(config, request, runOptions, options)
	if err != nil {
		return codexappserver.Options{}, nil, err
	}
	if bound.Merge == nil {
		return bound.Options, nil, nil
	}
	return bound.Options, &HelperSession{}, nil
}
