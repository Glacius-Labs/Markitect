package projectrun

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
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
