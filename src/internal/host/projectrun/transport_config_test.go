package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"go.yaml.in/yaml/v3"
)

func appServerAgent(t *testing.T) Agent {
	t.Helper()
	command, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	agent := validRuntime().Agents["commerce"]
	agent.Command = command
	agent.Args = nil
	agent.Transport = TransportCodexAppServer
	agent.WorkspaceMode = "git"
	agent.InstructionPaths = []string{"AGENTS.md"}
	agent.AppServer = &AppServerSettings{
		ReasoningEffort: "high",
		Helpers:         AppServerHelpers{Enabled: false},
		MaxEventBytes:   1 << 20,
	}
	return agent
}

func TestAppServerRuntimeConfigProducesTransportFingerprint(t *testing.T) {
	config := validRuntime()
	agent := appServerAgent(t)
	config.Agents["commerce"] = agent
	if err := ValidateRuntime(config); err != nil {
		t.Fatalf("valid native runtime: %v", err)
	}
	shared, err := agent.AgentConfig()
	if err != nil {
		t.Fatalf("AgentConfig: %v", err)
	}
	if shared.Transport != TransportCodexAppServer || shared.WorkspaceMode != "" || len(shared.TransportConfig) == 0 {
		t.Fatalf("shared native config leaked or omitted transport binding: %#v", shared)
	}
	if shared.EnvironmentAllowlist == nil {
		t.Fatal("omitted environmentMode must preserve the existing explicit environment name list")
	}
	var encoded AppServerSettings
	if err := json.Unmarshal(shared.TransportConfig, &encoded); err != nil || encoded.ReasoningEffort != "high" || encoded.MaxEventBytes != 1<<20 {
		t.Fatalf("transport config = %#v err=%v", encoded, err)
	}
	first, err := agentexec.Fingerprint(shared)
	if err != nil {
		t.Fatalf("fingerprint native config: %v", err)
	}
	agent.AppServer.ReasoningEffort = "medium"
	changed, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	second, err := agentexec.Fingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("changing App Server settings did not change the shared config fingerprint")
	}
	if agent.WorkspaceMode != "git" {
		t.Fatal("AgentConfig erased the source runtime Git workspace contract")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Runtime
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("strict App Server YAML round trip: %v\n%s", err, data)
	}
	if decoded.Agents["commerce"].AppServer == nil || decoded.Agents["commerce"].AppServer.Helpers != agent.AppServer.Helpers {
		t.Fatalf("App Server settings did not round trip: %#v", decoded.Agents["commerce"])
	}
}

func TestNativeInheritedEnvironmentModeBindsEffectiveCallerEnvironment(t *testing.T) {
	const environmentName = "MARKITECT_NATIVE_ENV_MODE_FINGERPRINT_TEST"
	t.Setenv(environmentName, "first-sensitive-value")
	config := validRuntime()
	agent := appServerAgent(t)
	agent.Model = "gpt-6-luna"
	agent.ProviderVersion = codexappserver.SupportedProviderVersion
	agent.Environment = []string{"PATH"}
	agent.AppServer.EnvironmentMode = AppServerEnvironmentModeInherit
	config.Agents["commerce"] = agent
	if err := ValidateRuntime(config); err != nil {
		t.Fatalf("native inherited environment runtime: %v", err)
	}
	shared, err := agent.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	if shared.EnvironmentAllowlist != nil {
		t.Fatalf("native inherit mode must pass a nil environment allowlist: %#v", shared.EnvironmentAllowlist)
	}
	if !reflect.DeepEqual(agent.Environment, []string{"PATH"}) {
		t.Fatalf("native inheritance must not widen declared-check name policy: %#v", agent.Environment)
	}
	var encoded AppServerSettings
	if err := json.Unmarshal(shared.TransportConfig, &encoded); err != nil || encoded.EnvironmentMode != AppServerEnvironmentModeInherit {
		t.Fatalf("environment mode did not flow into transport config: %+v err=%v", encoded, err)
	}
	serialized, err := json.Marshal(shared)
	if err != nil || strings.Contains(string(serialized), "first-sensitive-value") {
		t.Fatalf("effective environment value leaked into invocation config: err=%v", err)
	}
	invoker := NewTransportInvokerWithoutHostHelpers(codexappserver.Options{})
	first, err := invoker.Fingerprint(shared)
	if err != nil {
		t.Fatalf("fingerprint inherited native environment: %v", err)
	}
	t.Setenv(environmentName, "second-sensitive-value")
	second, err := invoker.Fingerprint(shared)
	if err != nil {
		t.Fatalf("fingerprint changed inherited native environment: %v", err)
	}
	if first == second {
		t.Fatal("changing the effective caller environment did not invalidate the native transport fingerprint")
	}

	runtimeYAML, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Runtime
	decoder := yaml.NewDecoder(strings.NewReader(string(runtimeYAML)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil || decoded.Agents["commerce"].AppServer.EnvironmentMode != AppServerEnvironmentModeInherit {
		t.Fatalf("strict runtime YAML round trip lost environment mode: err=%v", err)
	}
}

func TestNativeInheritedEnvironmentModeRequiresNativeOwnedGitAndClosedValue(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Agent)
		want   string
	}{
		{"requires owned Git", func(agent *Agent) { agent.WorkspaceMode = ""; agent.InstructionPaths = nil }, "requires native codex-app-server with owned git workspace"},
		{"closed mode", func(agent *Agent) { agent.AppServer.EnvironmentMode = "all" }, "must be empty or inherit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := validRuntime()
			agent := appServerAgent(t)
			agent.AppServer.EnvironmentMode = AppServerEnvironmentModeInherit
			test.change(&agent)
			config.Agents["commerce"] = agent
			if err := ValidateRuntime(config); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q validation error, got %v", test.want, err)
			}
		})
	}
}

func TestWindowsSandboxBackendFlowsThroughRuntimeAndFingerprint(t *testing.T) {
	config := validRuntime()
	worker := appServerAgent(t)
	config.Agents["commerce"] = worker
	if err := ValidateRuntime(config); err != nil {
		t.Fatalf("omitted backend should remain valid: %v", err)
	}
	baseShared, err := worker.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	baseFingerprint, err := agentexec.Fingerprint(baseShared)
	if err != nil {
		t.Fatal(err)
	}
	worker.AppServer.WindowsSandboxBackend = codexappserver.WindowsSandboxBackendMXC
	config.Agents["commerce"] = worker
	if err := ValidateRuntime(config); err != nil {
		t.Fatalf("mxc runtime should validate: %v", err)
	}
	mxcShared, err := worker.AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	var encoded AppServerSettings
	if err := json.Unmarshal(mxcShared.TransportConfig, &encoded); err != nil || encoded.WindowsSandboxBackend != codexappserver.WindowsSandboxBackendMXC {
		t.Fatalf("mxc setting did not flow into explicit transport config: %+v err=%v", encoded, err)
	}
	mxcFingerprint, err := agentexec.Fingerprint(mxcShared)
	if err != nil || baseFingerprint == mxcFingerprint {
		t.Fatalf("mxc setting did not change runtime fingerprint: base=%s mxc=%s err=%v", baseFingerprint, mxcFingerprint, err)
	}
	runtimeYAML, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Runtime
	decoder := yaml.NewDecoder(strings.NewReader(string(runtimeYAML)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err != nil || decoded.Agents["commerce"].AppServer.WindowsSandboxBackend != codexappserver.WindowsSandboxBackendMXC {
		t.Fatalf("strict runtime YAML round trip lost the explicit backend: backend=%q err=%v", decoded.Agents["commerce"].AppServer.WindowsSandboxBackend, err)
	}
	worker.AppServer.WindowsSandboxBackend = "other"
	config.Agents["commerce"] = worker
	if err := ValidateRuntime(config); err == nil || !strings.Contains(err.Error(), "supported value is mxc") {
		t.Fatalf("unsupported backend was accepted: %v", err)
	}
}

func TestNativeGitWorkspaceAllowsSeparateReadOnlyReviewerAndVerifier(t *testing.T) {
	config := validRuntime()
	manager := appServerAgent(t)
	config.Agents["commerce"] = manager
	readonly := appServerAgent(t)
	readonly.WorkspaceMode = ""
	readonly.InstructionPaths = nil
	config.Review = &ReviewConfig{Agents: map[string]Agent{"commerce": readonly}, MaxRounds: 1, MaxManagerRounds: 1}
	config.Verifier = &readonly
	if err := ValidateRuntime(config); err != nil {
		t.Fatalf("separate native read-only roles were rejected: %v", err)
	}
	assessment, err := ReadOnlyAgent(config, "commerce")
	if err != nil || assessment.Transport != TransportCodexAppServer || assessment.WorkspaceMode != "" {
		t.Fatalf("read-only native assessment = %#v, err=%v", assessment, err)
	}
}

func TestValidateRuntimeRejectsNativeTransportMismatchAndUnboundedSettings(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Agent)
		want   string
	}{
		{"unknown transport", func(a *Agent) { a.Transport = "other" }, "unsupported transport"},
		{"missing app server settings", func(a *Agent) { a.AppServer = nil }, "requires explicit appServer settings"},
		{"settings with process transport", func(a *Agent) { a.AppServer = appServerAgent(t).AppServer }, "require transport"},
		{"git workspace with process transport", func(a *Agent) {
			a.Transport = TransportProcess
			a.AppServer = nil
			a.WorkspaceMode = "git"
			a.InstructionPaths = []string{"AGENTS.md"}
		}, "requires transport"},
		{"relative native executable", func(a *Agent) { a.Command = "codex" }, "absolute executable"},
		{"native command arguments", func(a *Agent) { a.Args = []string{"--version"} }, "does not accept command arguments"},
		{"native model options", func(a *Agent) { a.ModelOptions = map[string]any{"temperature": 0.1} }, "does not accept modelOptions"},
		{"native scoped workspace", func(a *Agent) { a.WorkspaceMode = "scoped" }, "requires workspaceMode git"},
		{"missing native instructions", func(a *Agent) { a.InstructionPaths = nil }, "instructionPaths"},
		{"invalid helper bounds", func(a *Agent) {
			a.AppServer.Helpers = AppServerHelpers{Enabled: true, MaxStartRequests: 0, MaxDepth: 1}
		}, "helper policy"},
		{"missing event bound", func(a *Agent) { a.AppServer.MaxEventBytes = 0 }, "positive time and event byte limits"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validRuntime()
			agent := appServerAgent(t)
			if strings.Contains(test.name, "settings with process") {
				agent.Transport = TransportProcess
			}
			test.change(&agent)
			config.Agents["commerce"] = agent
			if err := ValidateRuntime(config); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestAgentexecFingerprintValidatesTransportConfigAndRunDoesNotLaunchIt(t *testing.T) {
	shared, err := appServerAgent(t).AgentConfig()
	if err != nil {
		t.Fatal(err)
	}
	invalid := shared
	invalid.TransportConfig = nil
	if _, err := agentexec.Fingerprint(invalid); err == nil || !strings.Contains(err.Error(), "requires explicit transportConfig") {
		t.Fatalf("missing native transport config error = %v", err)
	}
	invalid = shared
	invalid.TransportConfig = json.RawMessage(`{"reasoningEffort":"high","helpers":{"enabled":false,"maxStartRequests":1,"maxDepth":0},"maxEventBytes":1024}`)
	if _, err := agentexec.Fingerprint(invalid); err == nil || !strings.Contains(err.Error(), "helper policy") {
		t.Fatalf("mismatched helper config error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := agentexec.Run(ctx, agentexec.Config{Transport: TransportCodexAppServer}, agentexec.Request{}, agentexec.RunOptions{}); err == nil || !strings.Contains(err.Error(), "route through its configured Invoker") {
		t.Fatalf("process runner native routing error = %v", err)
	}
}
