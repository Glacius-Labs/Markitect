package projectsetup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

func TestBuildRuntimeMapsAllManagersAndPinsTools(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: "root-manager", Namespace: ""},
		{ID: "orders-manager", Namespace: "commerce.orders", Parent: "root-manager"},
		{ID: "inventory-manager", Namespace: "commerce.inventory", Parent: "root-manager"},
	}}}
	options := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider})
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	if len(config.Agents) != 3 {
		t.Fatalf("agent mappings = %d, want every active manager", len(config.Agents))
	}
	if config.Review == nil || config.Review.MaxRounds != DefaultReviewMaxRounds || config.Review.MaxManagerRounds != DefaultReviewMaxManagerRounds {
		t.Fatalf("review defaults = %#v, want maxRounds=%d and maxManagerRounds=%d", config.Review, DefaultReviewMaxRounds, DefaultReviewMaxManagerRounds)
	}
	if len(config.Review.Agents) != len(config.Agents) {
		t.Fatalf("reviewer mappings = %d, want one for every Manager", len(config.Review.Agents))
	}
	wantBackend := codexappserver.WindowsSandboxBackend("")
	if runtime.GOOS == "windows" {
		wantBackend = codexappserver.WindowsSandboxBackendMXC
	}
	for _, id := range []string{"root-manager", "orders-manager", "inventory-manager"} {
		agent, ok := config.Agents[id]
		if !ok {
			t.Fatalf("missing runtime mapping for %s", id)
		}
		if agent.Command != provider.Path || len(agent.Args) != 0 || agent.Transport != projectrun.TransportCodexAppServer || agent.Model != "gpt-6-luna" || agent.ProviderVersion != provider.Version {
			t.Fatalf("unexpected mapping for %s: %#v", id, agent)
		}
		if agent.WorkspaceMode != "git" || agent.AppServer == nil || agent.AppServer.ReasoningEffort != "high" || agent.AppServer.PermissionProfile != ":workspace" || agent.AppServer.WindowsSandboxBackend != wantBackend || agent.AppServer.EnvironmentMode != projectrun.AppServerEnvironmentModeInherit || !agent.AppServer.Helpers.Enabled {
			t.Fatalf("default setup must select the typed native Codex App Server worker: %#v", agent)
		}
		if agent.ModelOptions != nil || agent.AppServer.Helpers.MaxStartRequests != DefaultMaxHelperStarts || agent.AppServer.Helpers.MaxDepth != 1 || agent.AppServer.MaxEventBytes != DefaultMaxEventBytes {
			t.Fatalf("native settings must be explicit, bounded, and leave account permissions inherited: %#v", agent.AppServer)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime file pins for %s = %d, want Codex and both instructions", id, len(agent.RuntimeFiles))
		}
		reviewer, ok := config.Review.Agents[id]
		if !ok {
			t.Fatalf("missing reviewer mapping for %s", id)
		}
		if reviewer.WorkspaceMode != "git" || reviewer.Transport != projectrun.TransportCodexAppServer || len(reviewer.InstructionPaths) != len(agent.InstructionPaths) || len(reviewer.RuntimeFiles) != len(agent.RuntimeFiles) || reviewer.Command != provider.Path {
			t.Fatalf("reviewer config for %s must use a fresh native owned workspace with pinned instructions: %#v", id, reviewer)
		}
		if reviewer.ModelOptions != nil || reviewer.AppServer == nil || reviewer.AppServer.PermissionProfile != ":workspace" || reviewer.Model != agent.Model || reviewer.ProviderVersion != agent.ProviderVersion {
			t.Fatalf("reviewer model profile for %s differs from worker profile", id)
		}
		if reviewer.AppServer.WindowsSandboxBackend != wantBackend || reviewer.AppServer.EnvironmentMode != projectrun.AppServerEnvironmentModeInherit {
			t.Fatalf("reviewer sandbox/environment settings for %s = %#v, want sandbox %q and inherited caller environment", id, reviewer.AppServer, wantBackend)
		}
		if containsName(reviewer.Environment, "CODEX_HOME") {
			t.Fatal("runtime must use the existing OS-default Codex profile, not override CODEX_HOME")
		}
		if runtime.GOOS == "windows" {
			for _, name := range []string{"USERPROFILE", "APPDATA", "LOCALAPPDATA"} {
				if !containsName(reviewer.Environment, name) {
					t.Fatalf("native reviewer allowlist omits OS-default profile variable %s: %#v", name, reviewer.Environment)
				}
			}
		} else if !containsName(reviewer.Environment, "HOME") {
			t.Fatalf("native reviewer allowlist omits OS-default HOME: %#v", reviewer.Environment)
		}
	}
	if config.Verifier == nil || config.Verifier.AppServer == nil || config.Verifier.AppServer.PermissionProfile != ":workspace" || config.Verifier.WorkspaceMode != "git" || config.Verifier.Command != provider.Path {
		t.Fatalf("default setup must configure an independent verifier in its own workspace with the shared workspace profile: %#v", config.Verifier)
	}
	if config.Verifier.AppServer.WindowsSandboxBackend != wantBackend || !reflect.DeepEqual(config.Verifier.RuntimeFiles, config.Agents["root-manager"].RuntimeFiles) || !reflect.DeepEqual(config.Verifier.InstructionPaths, config.Agents["root-manager"].InstructionPaths) {
		t.Fatalf("verifier does not use platform sandbox or share pinned executable/instructions: verifier=%+v root=%+v", config.Verifier, config.Agents["root-manager"])
	}
	if config.Mode != "controlled-local" || config.RequireIsolation || config.Limits.MaxRetries != 1 || config.Limits.MaxParallel != 2 || config.Limits.MaxStarts != DefaultMaxStarts || config.Limits.MaxDuration != projectrun.Duration(DefaultMaxRunTime) || config.Limits.MaxCostMicros != 5000 {
		t.Fatalf("unsafe or unexpected limits: %#v", config)
	}
	if config.Limits.MaxDuration <= 0 || config.Limits.MaxCostMicros <= 0 {
		t.Fatalf("runtime budget and deadline must remain finite and positive: %#v", config.Limits)
	}
	if err := projectrun.ValidateRuntime(config); err != nil {
		t.Fatalf("generated runtime is invalid: %v", err)
	}
}

func TestBuildRuntimeRoleOptionSurfacesConstructNativeAdapters(t *testing.T) {
	project := setupProjectFixture(t)
	provider := testTool(t, project.Root, providerName(), true)
	provider.Version = codexappserver.SupportedProviderVersion
	options := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high",
		InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 100}
	runtimeConfig, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider})
	if err != nil {
		t.Fatalf("normal setup runtime: %v", err)
	}
	managerID := project.Report.Managers[0].ID
	invoker := projectrun.NewTransportInvoker(codexappserver.Options{})

	for _, helpersEnabled := range []bool{true, false} {
		name := "helpers-enabled"
		if !helpersEnabled {
			name = "helpers-disabled"
		}
		t.Run(name, func(t *testing.T) {
			candidate := runtimeConfig
			candidate.Agents = make(map[string]projectrun.Agent, len(runtimeConfig.Agents))
			for id, agent := range runtimeConfig.Agents {
				settings := *agent.AppServer
				agent.AppServer = &settings
				agent.AppServer.Helpers.Enabled = helpersEnabled
				if !helpersEnabled {
					agent.AppServer.Helpers.MaxStartRequests = 0
					agent.AppServer.Helpers.MaxDepth = 0
				}
				candidate.Agents[id] = agent
			}
			candidate.Review = &projectrun.ReviewConfig{Agents: make(map[string]projectrun.Agent, len(runtimeConfig.Review.Agents)),
				MaxRounds: runtimeConfig.Review.MaxRounds, MaxManagerRounds: runtimeConfig.Review.MaxManagerRounds}
			for id, agent := range runtimeConfig.Review.Agents {
				settings := *agent.AppServer
				agent.AppServer = &settings
				agent.AppServer.Helpers.Enabled = helpersEnabled
				if !helpersEnabled {
					agent.AppServer.Helpers.MaxStartRequests = 0
					agent.AppServer.Helpers.MaxDepth = 0
				}
				candidate.Review.Agents[id] = agent
			}
			verifier := *runtimeConfig.Verifier
			verifierSettings := *verifier.AppServer
			verifier.AppServer = &verifierSettings
			verifier.AppServer.Helpers.Enabled = helpersEnabled
			if !helpersEnabled {
				verifier.AppServer.Helpers.MaxStartRequests = 0
				verifier.AppServer.Helpers.MaxDepth = 0
			}
			candidate.Verifier = &verifier

			roles := []struct {
				name    string
				agent   projectrun.Agent
				request agentexec.Request
				invoker *projectrun.TransportInvoker
			}{
				{name: "manager", agent: candidate.Agents[managerID], request: agentexec.Request{Role: agentexec.RoleExecutor,
					Context: json.RawMessage(`{"kind":"projectrun-task/v1","managerId":"root","phase":"work","allowedWritePaths":["src/"]}`)}},
				{name: "reviewer", agent: candidate.Review.Agents[managerID], request: agentexec.Request{Role: agentexec.RoleExecutor,
					Context: json.RawMessage(`{"kind":"projectrun-review/v1"}`)}},
				{name: "verifier", agent: *candidate.Verifier, request: agentexec.Request{Role: agentexec.RoleVerifier,
					Context: json.RawMessage(`{"kind":"projectrun-verify/v1"}`)}},
				{name: "helper", agent: candidate.Agents[managerID], invoker: projectrun.NewTransportInvokerWithoutHostHelpers(codexappserver.Options{}), request: agentexec.Request{Role: agentexec.RoleExecutor,
					Context: json.RawMessage(`{"kind":"projectrun-helper/v1","managerId":"root","helperDepth":1,"allowedWritePaths":["src/"]}`)}},
			}
			for _, role := range roles {
				config, err := role.agent.AgentConfig()
				if err != nil {
					t.Fatalf("%s AgentConfig: %v", role.name, err)
				}
				roleInvoker := role.invoker
				if roleInvoker == nil {
					roleInvoker = invoker
				}
				actual, err := roleInvoker.FingerprintForRequest(config, role.request)
				if err != nil {
					t.Fatalf("%s native adapter construction: %v", role.name, err)
				}
				planned, err := roleInvoker.Fingerprint(config)
				if err != nil {
					t.Fatalf("%s planned fingerprint: %v", role.name, err)
				}
				wantPlanMatch := role.name == "manager" || role.name == "helper" || !helpersEnabled
				if (actual == planned) != wantPlanMatch {
					t.Fatalf("%s helper surface mismatch: helpersEnabled=%t actual/planned match=%t", role.name, helpersEnabled, actual == planned)
				}
				if role.name == "helper" && helpersEnabled {
					parentAgentConfig, err := candidate.Agents[managerID].AgentConfig()
					if err != nil {
						t.Fatal(err)
					}
					parentFingerprint, err := invoker.Fingerprint(parentAgentConfig)
					if err != nil || actual == parentFingerprint {
						t.Fatalf("depth-one helper retained its parent's Host helper surface: child=%q parent=%q err=%v", actual, parentFingerprint, err)
					}
				}
				if role.name == "manager" && !helpersEnabled {
					enabledConfig, err := runtimeConfig.Agents[managerID].AgentConfig()
					if err != nil {
						t.Fatal(err)
					}
					enabledFingerprint, err := invoker.FingerprintForRequest(enabledConfig, role.request)
					if err != nil || enabledFingerprint == actual {
						t.Fatalf("disabling helpers did not remove the Manager helper spec: enabled=%q disabled=%q err=%v", enabledFingerprint, actual, err)
					}
				}
			}
		})
	}
}

func TestBuildRuntimeExplicitWindowsMXCUsesSharedWorkspaceProfile(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	options := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", CodexProfile: ":workspace",
		WindowsSandboxBackend: codexappserver.WindowsSandboxBackendMXC,
		InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 10}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider})
	if runtime.GOOS != "windows" {
		if err == nil || !strings.Contains(err.Error(), "supported only on Windows") {
			t.Fatalf("non-Windows setup accepted MXC: config=%#v err=%v", config, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("Windows setup rejected explicit MXC: %v", err)
	}
	worker := config.Agents["root"]
	reviewer := config.Review.Agents["root"]
	if worker.AppServer.WindowsSandboxBackend != codexappserver.WindowsSandboxBackendMXC || reviewer.AppServer.WindowsSandboxBackend != codexappserver.WindowsSandboxBackendMXC {
		t.Fatalf("explicit backend not applied consistently: worker=%+v reviewer=%+v", worker.AppServer, reviewer.AppServer)
	}
	if worker.AppServer.PermissionProfile != ":workspace" || reviewer.AppServer.PermissionProfile != ":workspace" || config.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("setup roles must share the workspace profile: worker=%+v reviewer=%+v verifier=%+v", worker.AppServer, reviewer.AppServer, config.Verifier.AppServer)
	}
}

func TestBuildRuntimeDefaultsWindowsMXCAndSharedWorkspaceProfile(t *testing.T) {
	project := setupProjectFixture(t)
	provider := testTool(t, t.TempDir(), providerName(), true)
	provider.Version = "codex-cli 0.162.0"
	options := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 10}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider})
	if err != nil {
		t.Fatalf("default setup: %v", err)
	}
	wantBackend := codexappserver.WindowsSandboxBackend("")
	if runtime.GOOS == "windows" {
		wantBackend = codexappserver.WindowsSandboxBackendMXC
	}
	managerID := project.Report.Managers[0].ID
	profiles := []projectrun.Agent{config.Agents[managerID], config.Review.Agents[managerID]}
	if config.Verifier != nil {
		profiles = append(profiles, *config.Verifier)
	}
	for i, agent := range profiles {
		if agent.AppServer == nil || agent.AppServer.PermissionProfile != ":workspace" || agent.AppServer.WindowsSandboxBackend != wantBackend {
			t.Fatalf("native role %d defaults must share workspace permission and platform sandbox: %+v want backend %q", i, agent.AppServer, wantBackend)
		}
	}
}

func TestNormalizeOptionsRejectsUnsupportedWindowsSandboxBackend(t *testing.T) {
	_, err := normalizeOptions(Options{Provider: "codex", WindowsSandboxBackend: "unsafe"})
	if err == nil || !strings.Contains(err.Error(), "supported value is mxc") {
		t.Fatalf("unsupported Windows sandbox backend was accepted: %v", err)
	}
}

func TestBuildRuntimeDefaultsNativeModelAndBindsInstructions(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{
		Root:   root,
		Config: projectwork.Config{},
		Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}, {ID: "orders", Parent: "root"}}},
	}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	options := Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high",
		InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
	}
	config, err := BuildRuntime(project, options, found)
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	if len(config.Agents) != 2 || config.Review == nil {
		t.Fatalf("native setup did not map both roles for each Manager: %#v", config)
	}
	for _, id := range []string{"root", "orders"} {
		agent := config.Agents[id]
		if agent.WorkspaceMode != "git" {
			t.Fatalf("workspace mode for %s = %q", id, agent.WorkspaceMode)
		}
		wantPaths := []string{".agents/skills/markitect-implement/SKILL.md", "AGENTS.md"}
		if !reflect.DeepEqual(agent.InstructionPaths, wantPaths) {
			t.Fatalf("instruction paths for %s = %#v, want %#v", id, agent.InstructionPaths, wantPaths)
		}
		if agent.Transport != projectrun.TransportCodexAppServer || len(agent.Args) != 0 || agent.Command != provider.Path || agent.ModelOptions != nil {
			t.Fatalf("manager must use the selected native executable directly with no wrapper arguments/options: %#v", agent)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime pins for %s = %d, want Codex and two instructions: %#v", id, len(agent.RuntimeFiles), agent.RuntimeFiles)
		}
		if !hasRuntimePin(agent.RuntimeFiles, found.ProviderBinary) {
			t.Fatalf("native runtime does not pin selected Codex executable %s: %#v", provider.Path, agent.RuntimeFiles)
		}
		for _, path := range agent.InstructionPaths {
			absolute := filepath.Join(root, filepath.FromSlash(path))
			foundPin := false
			for _, pin := range agent.RuntimeFiles {
				if pin.Path == absolute && pin.Mode == "0644" && strings.HasPrefix(pin.Digest, "sha256:") {
					foundPin = true
				}
			}
			if !foundPin {
				t.Fatalf("instruction %s lacks its absolute mode/digest runtime pin: %#v", path, agent.RuntimeFiles)
			}
		}
		if agent.AppServer == nil || agent.AppServer.ReasoningEffort != "high" || agent.AppServer.PermissionProfile != ":workspace" {
			t.Fatalf("native model effort/shared workspace permissions = %#v", agent.AppServer)
		}
		reviewer := config.Review.Agents[id]
		if reviewer.WorkspaceMode != "git" || len(reviewer.RuntimeFiles) != 3 || !reflect.DeepEqual(reviewer.InstructionPaths, agent.InstructionPaths) || reviewer.AppServer.PermissionProfile != ":workspace" {
			t.Fatalf("reviewer config should use its own Git workspace with the same instructions and shared permission profile: %#v", reviewer)
		}
	}
	if config.Verifier == nil || config.Verifier.AppServer.PermissionProfile != ":workspace" || config.Verifier.WorkspaceMode != "git" {
		t.Fatalf("verifier does not use the shared native workspace profile: %+v", config.Verifier)
	}
}

func TestBuildRuntimeDefaultNativeRequiresExistingCodexInstructions(t *testing.T) {
	root := t.TempDir()
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	_, err := BuildRuntime(project, Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high",
		InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2,
	}, found)
	if err == nil || !strings.Contains(err.Error(), "existing generated Codex project instructions") {
		t.Fatalf("default native setup without instructions error = %v", err)
	}
}

func writeNativeInstructions(t *testing.T, root string) {
	t.Helper()
	for path, content := range map[string]string{
		"AGENTS.md": "# Project agent instructions\n",
		".agents/skills/markitect-implement/SKILL.md": "# Generated implementation skill\n",
		"CLAUDE.md": "# Claude guidance is not selected for Codex\n",
		".claude/skills/markitect-implement/SKILL.md": "# Claude skill is not selected for Codex\n",
	} {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func hasRuntimePin(files []agentexec.RuntimeFile, tool Tool) bool {
	for _, file := range files {
		if file.Path == tool.Path && file.Mode == tool.Mode && file.Digest == tool.Digest {
			return true
		}
	}
	return false
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if strings.EqualFold(name, want) {
			return true
		}
	}
	return false
}

func TestDiscoverProviderUsesCurrentWindowsCodexVendorPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows npm vendor layout")
	}
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	want := filepath.Join(appData, "npm", "node_modules", "@openai", "codex", "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc", "bin", "codex.exe")
	if err := os.MkdirAll(filepath.Dir(want), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("codex fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := discoverProvider("codex"); got != want {
		t.Fatalf("discovered Codex path = %q, want %q", got, want)
	}
}

func TestBuildRuntimeRejectsUnsupportedNativeModelEffortAndProvider(t *testing.T) {
	root := t.TempDir()
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	base := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2}
	cases := []struct {
		name   string
		change func(*Options, *Discovery)
	}{
		{"different model", func(o *Options, _ *Discovery) { o.Model = "gpt-6-sol" }},
		{"different effort", func(o *Options, _ *Discovery) { o.Effort = "medium" }},
		{"different CLI version", func(_ *Options, d *Discovery) { d.ProviderBinary.Version = "0.161.0" }},
		{"default must not substitute model", func(o *Options, _ *Discovery) { o.CodexProfile = ""; o.Model = "gpt-6-sol" }},
		{"unsupported provider version", func(_ *Options, d *Discovery) { d.ProviderBinary.Version = "codex-cli 0.163.0" }},
		{"Claude native mode", func(o *Options, d *Discovery) { o.Provider = "claude"; d.Provider = "claude" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			options := base
			discovery := found
			test.change(&options, &discovery)
			if _, err := BuildRuntime(project, options, discovery); err == nil {
				t.Fatal("unsupported native setup was accepted")
			}
		})
	}
}

func TestBuildRuntimeExplicitCodexProfileIsSharedAndPreviewIsDigestGuarded(t *testing.T) {
	project := setupProjectFixture(t)
	provider := testTool(t, t.TempDir(), providerName(), true)
	provider.Version = "codex-cli 0.162.0"
	options := Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high", CodexProfile: ":workspace",
		InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
	}
	discoveryCalls := 0
	preview, err := previewEditWithDiscovery(project, options, func(got Options) (Discovery, error) {
		discoveryCalls++
		if got.CodexProfile != ":workspace" {
			t.Fatalf("explicit Codex profile was lost before discovery: %+v", got)
		}
		return Discovery{Provider: "codex", ProviderBinary: provider}, nil
	})
	if err != nil {
		t.Fatalf("preview runtime edit without starting an actor: %v", err)
	}
	if discoveryCalls != 1 || preview.EditPlan.Digest == "" || len(preview.Mutation.Files) != 1 || preview.Mutation.Files[0].Path != projectwork.RuntimePath {
		t.Fatalf("preview is not a single digest-bound runtime edit: calls=%d preview=%+v", discoveryCalls, preview)
	}
	var runtimeConfig projectrun.Runtime
	if err := yaml.Unmarshal([]byte(preview.Mutation.Files[0].Content), &runtimeConfig); err != nil {
		t.Fatalf("decode proposed runtime: %v", err)
	}
	if err := projectrun.ValidateRuntime(runtimeConfig); err != nil {
		t.Fatalf("proposed runtime is invalid: %v", err)
	}
	if runtimeConfig.Review == nil || len(runtimeConfig.Agents) != 1 || len(runtimeConfig.Review.Agents) != 1 {
		t.Fatalf("runtime lost Manager or separate Review binding: %+v", runtimeConfig)
	}
	managerID := project.Report.Managers[0].ID
	manager := runtimeConfig.Agents[managerID]
	reviewer := runtimeConfig.Review.Agents[managerID]
	if manager.AppServer == nil || reviewer.AppServer == nil {
		t.Fatalf("runtime YAML omitted App Server settings: %s", preview.Mutation.Files[0].Content)
	}
	if manager.AppServer.PermissionProfile != ":workspace" || reviewer.AppServer.PermissionProfile != ":workspace" || runtimeConfig.Verifier == nil || runtimeConfig.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("explicit setup profile must apply consistently to independent roles: manager=%+v reviewer=%+v verifier=%+v", manager.AppServer, reviewer.AppServer, runtimeConfig.Verifier)
	}
	assertSameNativePins(t, manager, reviewer, provider, options)

	if _, err := projectwork.ApplyEdit(project.Root, preview.EditPlan, "sha256:stale"); err == nil || !strings.Contains(err.Error(), "expected digest") {
		t.Fatalf("stale setup edit digest was not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project.Root, filepath.FromSlash(projectwork.RuntimePath))); !os.IsNotExist(err) {
		t.Fatalf("rejected preview changed runtime file: stat error=%v", err)
	}
	if _, err := projectwork.ApplyEdit(project.Root, preview.EditPlan, preview.EditPlan.BaseDigest); err != nil {
		t.Fatalf("apply exact reviewed setup digest: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(project.Root, filepath.FromSlash(projectwork.RuntimePath)))
	if err != nil {
		t.Fatal(err)
	}
	var applied projectrun.Runtime
	if err := yaml.Unmarshal(written, &applied); err != nil || applied.Agents[managerID].AppServer.PermissionProfile != ":workspace" || applied.Review.Agents[managerID].AppServer.PermissionProfile != ":workspace" || applied.Verifier == nil || applied.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("guarded write did not preserve the shared role profile: runtime=%+v err=%v", applied, err)
	}
}

func TestBuildRuntimeOmittedProfileDefaultsWorkspaceForEveryRoleAndMalformedProfileFailsConfigValidation(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, providerName(), true)
	provider.Version = "codex-cli 0.162.0"
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "codex", ProviderBinary: provider}
	base := Options{Provider: "codex", Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 2}
	config, err := BuildRuntime(project, base, found)
	if err != nil {
		t.Fatalf("default setup: %v", err)
	}
	managerID := project.Report.Managers[0].ID
	if config.Agents[managerID].AppServer.PermissionProfile != ":workspace" || config.Review.Agents[managerID].AppServer.PermissionProfile != ":workspace" || config.Verifier == nil || config.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("omitting --codex-profile must select the shared workspace profile for every role: manager=%+v reviewer=%+v verifier=%+v", config.Agents[managerID].AppServer, config.Review.Agents[managerID].AppServer, config.Verifier)
	}
	base.CodexProfile = ":read-only"
	custom, err := BuildRuntime(project, base, found)
	if err != nil {
		t.Fatalf("explicit shared profile: %v", err)
	}
	if custom.Agents[managerID].AppServer.PermissionProfile != ":read-only" || custom.Review.Agents[managerID].AppServer.PermissionProfile != ":read-only" || custom.Verifier == nil || custom.Verifier.AppServer.PermissionProfile != ":read-only" {
		t.Fatalf("explicit profile override was not shared across roles: manager=%+v reviewer=%+v verifier=%+v", custom.Agents[managerID].AppServer, custom.Review.Agents[managerID].AppServer, custom.Verifier)
	}
	base.CodexProfile = string([]byte{0xff})
	if _, err := BuildRuntime(project, base, found); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("malformed profile did not fail native App Server config validation: %v", err)
	}
}

func setupProjectFixture(t *testing.T) *projectwork.Project {
	t.Helper()
	root := t.TempDir()
	manifest := []byte("apiVersion: project.markitect.example.org/v1alpha1\nname: setup-test\nmodelFiles:\n  - .markitect/model/manager.yaml\ninventoryRoots: []\nexclusions: []\n")
	manager := []byte("apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: root\n  namespace: \"\"\npurpose: Own the setup test project.\nspec:\n  owns:\n    - .\n  instructions: Implement only the requested bounded changes.\n")
	s := &snapshot.Snapshot{Files: map[string][]byte{
		projectwork.ManifestPath:        manifest,
		".markitect/model/manager.yaml": manager,
	}, Modes: map[string]string{
		projectwork.ManifestPath:        snapshot.RegularMode,
		".markitect/model/manager.yaml": snapshot.RegularMode,
	}}
	for path, content := range s.Files {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeNativeInstructions(t, root)
	runSetupGit(t, root, "init", "-b", "setup-test")
	runSetupGit(t, root, "config", "user.email", "setup-test@example.test")
	runSetupGit(t, root, "config", "user.name", "Setup Test")
	runSetupGit(t, root, "add", ".")
	runSetupGit(t, root, "commit", "-m", "setup fixture")
	project, err := projectwork.Load(root, "")
	if err != nil {
		t.Fatalf("load setup fixture project: %v", err)
	}
	return project
}

func runSetupGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func providerName() string {
	if runtime.GOOS == "windows" {
		return "codex.exe"
	}
	return "codex"
}

func assertSameNativePins(t *testing.T, manager, reviewer projectrun.Agent, provider Tool, options Options) {
	t.Helper()
	if manager.Command != provider.Path || reviewer.Command != provider.Path || len(manager.Args) != 0 || len(reviewer.Args) != 0 || manager.Transport != projectrun.TransportCodexAppServer || reviewer.Transport != projectrun.TransportCodexAppServer {
		t.Fatalf("permission profile altered native process selection: manager=%+v reviewer=%+v", manager, reviewer)
	}
	if manager.Model != options.Model || reviewer.Model != options.Model || manager.ProviderVersion != provider.Version || reviewer.ProviderVersion != provider.Version || manager.AppServer.ReasoningEffort != options.Effort || reviewer.AppServer.ReasoningEffort != options.Effort {
		t.Fatalf("permission profile altered model/version/effort pins: manager=%+v reviewer=%+v", manager, reviewer)
	}
	if !reflect.DeepEqual(manager.InstructionPaths, reviewer.InstructionPaths) || !reflect.DeepEqual(manager.RuntimeFiles, reviewer.RuntimeFiles) || !reflect.DeepEqual(manager.Pricing, reviewer.Pricing) || manager.AppServer.Helpers != reviewer.AppServer.Helpers || manager.AppServer.MaxEventBytes != reviewer.AppServer.MaxEventBytes {
		t.Fatalf("role-specific permissions changed pinned instructions, provider, pricing or bounded helpers: manager=%+v reviewer=%+v", manager, reviewer)
	}
}

func TestBuildRuntimeRejectsNonCodexSetup(t *testing.T) {
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "claude", ProviderBinary: Tool{Path: filepath.Join(t.TempDir(), "provider"), Version: "claude 2.1.295", Digest: "sha256:" + strings.Repeat("a", 64), Mode: "0644"}}
	if _, err := BuildRuntime(project, Options{Provider: "custom", Model: "m", InputMicrosPerMillion: 1, MaxCostMicros: 1}, found); err == nil {
		t.Fatal("unsupported provider was accepted")
	}
	if _, err := BuildRuntime(project, Options{Provider: "claude"}, found); err == nil || !strings.Contains(err.Error(), "native Codex App Server only") {
		t.Fatalf("Claude setup error = %v", err)
	}
}

func TestDiscoverRejectsCommandShimAndParsesOnlyVersion(t *testing.T) {
	if !isCommandShim(`C:\tools\codex.cmd`) && runtime.GOOS == "windows" {
		t.Fatal("Windows command shim was accepted")
	}
	if match := versionPattern.FindStringSubmatch("codex-cli 0.130.0\n"); len(match) != 2 || match[1] != "0.130.0" {
		t.Fatalf("version parse = %#v", match)
	}
	if _, err := Discover(Options{Provider: "claude"}); err == nil || !strings.Contains(err.Error(), "Codex App Server only") {
		t.Fatalf("unsupported provider discovery error = %v", err)
	}
}

func TestDiscoverInstalledCodexDoesNotRequireMarkitectCheckoutOrLegacyHelpers(t *testing.T) {
	installedDir := t.TempDir()
	providerName := "codex"
	if runtime.GOOS == "windows" {
		providerName += ".exe"
	}
	provider := testTool(t, installedDir, providerName, true)
	versionCalls := 0
	var probedPath string
	discovery, err := discoverWithVersion(Options{
		Provider: "codex", ProviderExecutable: provider.Path,
	}, func(path string) (string, error) {
		versionCalls++
		probedPath = path
		return "0.162.0", nil
	})
	if err != nil {
		t.Fatalf("installed binary discovery without a source checkout: %v", err)
	}
	if versionCalls != 1 || discovery.ProviderBinary.Path != probedPath || discovery.ProviderBinary.Version != "codex-cli 0.162.0" {
		t.Fatalf("installed native binary was not pinned by path and version: calls=%d discovery=%+v", versionCalls, discovery)
	}
	if discovery.Authentication != "not-verified" || !strings.Contains(discovery.AuthenticationNote, "does not read credentials") {
		t.Fatalf("discovery must leave existing sign-in untouched and unverified: %+v", discovery)
	}
	entries, err := os.ReadDir(installedDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != providerName {
		t.Fatalf("installed runtime discovery depended on source helpers: entries=%v", entries)
	}
}

func TestInspectFileSupportsLargeProviderWithoutBufferingAsset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current-provider.exe")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	const size = (256 << 20) + 1
	if err := file.Truncate(size); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	want := sha256.New()
	zeros := make([]byte, 32<<10)
	for remaining := int64(size); remaining > 0; {
		chunk := int64(len(zeros))
		if remaining < chunk {
			chunk = remaining
		}
		want.Write(zeros[:chunk])
		remaining -= chunk
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	tool, err := inspectFile(path)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("inspect current-size provider: %v", err)
	}
	if tool.Digest != "sha256:"+hex.EncodeToString(want.Sum(nil)) {
		t.Fatalf("streamed provider digest = %s", tool.Digest)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 16<<20 {
		t.Fatalf("fingerprinting buffered a large asset: allocated %d bytes", allocated)
	}
}

func TestInspectFileRetainsFiniteAssetSizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized-provider.exe")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(agentexec.MaxRuntimeAssetBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	if _, err := inspectFile(path); err == nil || !strings.Contains(err.Error(), "512 MiB") {
		t.Fatalf("oversized provider was not rejected: %v", err)
	}
}

func testTool(t *testing.T, root, name string, executable bool) Tool {
	t.Helper()
	path := filepath.Join(root, name)
	content := []byte("fingerprint fixture " + name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0600)
	if executable && runtime.GOOS != "windows" {
		mode = 0700
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		t.Fatal(err)
	}
	actualMode := "0644"
	if runtime.GOOS != "windows" {
		actualMode = modeOctal(info.Mode().Perm())
	}
	sum := sha256.Sum256(content)
	version := ""
	if strings.Contains(name, "codex") {
		version = "0.130.0"
	}
	return Tool{Path: resolved, Version: version, Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: actualMode}
}

func modeOctal(mode os.FileMode) string { return fmt.Sprintf("%04o", mode) }
