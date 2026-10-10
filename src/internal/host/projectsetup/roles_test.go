package projectsetup

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

func int64Ref(value int64) *int64 { return &value }

// processExecutorTool creates a fixture executable and resolves its path the
// way discovery does, so short Windows temp names compare equal.
func processExecutorTool(t *testing.T, root string) Tool {
	t.Helper()
	tool := testTool(t, root, processExecutorName(), true)
	resolved, err := filepath.EvalSymlinks(tool.Path)
	if err != nil {
		t.Fatal(err)
	}
	tool.Path = resolved
	return tool
}

func processExecutorName() string {
	if runtime.GOOS == "windows" {
		return "exchange-executor.exe"
	}
	return "exchange-executor"
}

func TestBuildRuntimeMixesRoleProfilesAcrossProvidersModelsAndCostModes(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	executor := processExecutorTool(t, root)
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: "root-manager"}, {ID: "orders-manager", Parent: "root-manager"},
	}}}
	options := Options{Provider: ProviderCodex, Model: "gpt-6-luna", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
		Roles: &RoleOptions{
			Reviewer: &RoleProfile{Model: "gpt-6-sol", Effort: "medium", InputMicrosPerMillion: int64Ref(3)},
			Verifier: &RoleProfile{Provider: ProviderProcess, Model: "human-reviewer", Effort: "high", ProviderExecutable: executor.Path,
				ProviderArgs: []string{"--dir", "exchange"}, ProviderVersion: "exchange/1", CostMode: projectrun.CostModeUnmetered,
				Environment: []string{"EXCHANGE_TOKEN"}},
		}}
	config, err := BuildRuntime(project, options, Discovery{Provider: ProviderCodex, ProviderBinary: provider})
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	for _, id := range []string{"root-manager", "orders-manager"} {
		manager := config.Agents[id]
		if manager.Transport != projectrun.TransportCodexAppServer || manager.Model != "gpt-6-luna" || manager.AppServer.ReasoningEffort != DefaultCodexEffort ||
			manager.WorkspaceMode != "git" || manager.CostMode != "" || manager.Pricing != (projectrun.Pricing{InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11}) {
			t.Fatalf("Manager %s did not keep the default native profile: %+v", id, manager)
		}
		reviewer := config.Review.Agents[id]
		if reviewer.Transport != projectrun.TransportCodexAppServer || reviewer.Model != "gpt-6-sol" || reviewer.AppServer.ReasoningEffort != "medium" ||
			reviewer.WorkspaceMode != "git" || reviewer.Pricing != (projectrun.Pricing{InputMicrosPerMillion: 3, OutputMicrosPerMillion: 11}) ||
			!reflect.DeepEqual(reviewer.InstructionPaths, manager.InstructionPaths) {
			t.Fatalf("reviewer %s did not receive its own native model, effort and rate: %+v", id, reviewer)
		}
	}
	verifier := config.Verifier
	wantEnvironment := []string{"PATH", "TEMP", "TMP"}
	if runtime.GOOS == "windows" {
		wantEnvironment = append(wantEnvironment, "SystemRoot")
	}
	wantEnvironment = append(wantEnvironment, "EXCHANGE_TOKEN")
	if verifier == nil || verifier.Transport != projectrun.TransportProcess || verifier.AppServer != nil || verifier.WorkspaceMode != "" ||
		verifier.Command != executor.Path || !reflect.DeepEqual(verifier.Args, []string{"--dir", "exchange"}) || verifier.Model != "human-reviewer" ||
		verifier.ProviderVersion != "exchange/1" || verifier.CostMode != projectrun.CostModeUnmetered || verifier.Pricing != (projectrun.Pricing{}) ||
		!reflect.DeepEqual(verifier.Environment, wantEnvironment) || verifier.MaxStdoutBytes != DefaultProcessMaxStdout {
		t.Fatalf("verifier is not the selected unmetered process executor: %+v", verifier)
	}
	if !reflect.DeepEqual(verifier.ModelOptions, map[string]any{"reasoningEffort": "high"}) {
		t.Fatalf("process effort was not passed through modelOptions: %#v", verifier.ModelOptions)
	}
	if !reflect.DeepEqual(verifier.RuntimeFiles, []agentexec.RuntimeFile{{Path: executor.Path, Mode: executor.Mode, Digest: executor.Digest}}) {
		t.Fatalf("process executor bytes are not pinned: %+v", verifier.RuntimeFiles)
	}
	encoded, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(encoded), "costMode: unmetered") != 1 {
		t.Fatalf("runtime YAML must name the cost mode only for the unmetered agent:\n%s", encoded)
	}
}

func TestBuildRuntimeProcessOnlyProfileNeedsNoNativeInstructions(t *testing.T) {
	root := t.TempDir()
	executor := processExecutorTool(t, root)
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	options := Options{Provider: ProviderProcess, Model: "scripted", ProviderExecutable: executor.Path, CostMode: projectrun.CostModeUnmetered, MaxCostMicros: 1}
	found, err := Discover(options)
	if err != nil {
		t.Fatalf("Discover process executor: %v", err)
	}
	if found.ProviderBinary.Version != UndeclaredProviderVersion || found.ProviderBinary.Digest != executor.Digest {
		t.Fatalf("process discovery = %+v, want an undeclared version and the pinned digest", found)
	}
	config, err := BuildRuntime(project, options, found)
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	for name, agent := range map[string]projectrun.Agent{"manager": config.Agents["root"], "reviewer": config.Review.Agents["root"], "verifier": *config.Verifier} {
		if agent.Transport != projectrun.TransportProcess || len(agent.InstructionPaths) != 0 || !agent.Unmetered() || agent.ModelOptions != nil || agent.ProviderVersion != UndeclaredProviderVersion {
			t.Fatalf("%s is not a context-only unmetered process executor: %+v", name, agent)
		}
	}
	if config.Limits.MaxCostMicros != 1 || config.Limits.MaxStarts != DefaultMaxStarts || config.Limits.MaxDuration != projectrun.Duration(DefaultMaxRunTime) {
		t.Fatalf("unmetered profile changed the start or duration limits: %+v", config.Limits)
	}
}

func TestBuildRuntimeRejectsInvalidRoleProfiles(t *testing.T) {
	root := t.TempDir()
	writeNativeInstructions(t, root)
	provider := testTool(t, root, "codex.exe", true)
	provider.Version = "codex-cli 0.162.0"
	executor := processExecutorTool(t, root)
	project := &projectwork.Project{Root: root, Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	base := Options{Provider: ProviderCodex, Model: "gpt-6-luna", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaxCostMicros: 10}
	cases := []struct {
		name  string
		roles RoleOptions
		want  string
	}{
		{"unmetered with rates", RoleOptions{Reviewer: &RoleProfile{CostMode: projectrun.CostModeUnmetered, InputMicrosPerMillion: int64Ref(1)}}, "must not declare"},
		{"metered without rates", RoleOptions{Reviewer: &RoleProfile{InputMicrosPerMillion: int64Ref(0), OutputMicrosPerMillion: int64Ref(0)}}, "at least one positive rate"},
		{"unknown cost mode", RoleOptions{Reviewer: &RoleProfile{CostMode: "free"}}, "unsupported cost mode"},
		{"process without executable", RoleOptions{Manager: &RoleProfile{Provider: ProviderProcess, Model: "m"}}, "absolute providerExecutable"},
		{"process with relative executable", RoleOptions{Manager: &RoleProfile{Provider: ProviderProcess, Model: "m", ProviderExecutable: "executor"}}, "absolute providerExecutable"},
		{"provider switch inherits no model", RoleOptions{Manager: &RoleProfile{Provider: ProviderProcess, ProviderExecutable: executor.Path}}, "model must be nonempty"},
		{"codex with process arguments", RoleOptions{Reviewer: &RoleProfile{ProviderArgs: []string{"--x"}}}, "apply only to process executors"},
		{"unknown role provider", RoleOptions{Verifier: &RoleProfile{Provider: "claude", Model: "sonnet"}}, `unsupported provider "claude"`},
		{"unsupported native effort", RoleOptions{Verifier: &RoleProfile{Effort: "turbo"}}, "unsupported Codex reasoning effort"},
		{"unmetered native role", RoleOptions{Reviewer: &RoleProfile{CostMode: projectrun.CostModeUnmetered}}, "must be metered"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			options := base
			roles := test.roles
			options.Roles = &roles
			_, err := BuildRuntime(project, options, Discovery{Provider: ProviderCodex, ProviderBinary: provider})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("BuildRuntime error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPreviewDiscoversEachRoleExecutableOnce(t *testing.T) {
	project := setupProjectFixture(t)
	provider := testTool(t, t.TempDir(), providerName(), true)
	provider.Version = "codex-cli 0.162.0"
	executor := processExecutorTool(t, t.TempDir())
	process := &RoleProfile{Provider: ProviderProcess, Model: "scripted", ProviderExecutable: executor.Path, CostMode: projectrun.CostModeUnmetered}
	options := Options{Provider: ProviderCodex, Model: "gpt-6-luna", Effort: "high", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
		Roles: &RoleOptions{Reviewer: process, Verifier: process}}
	calls := map[string]int{}
	preview, err := previewEditWithDiscovery(project, options, func(got Options) (Discovery, error) {
		calls[got.Provider]++
		if got.Provider == ProviderProcess {
			return Discover(got)
		}
		return Discovery{Provider: ProviderCodex, ProviderBinary: provider}, nil
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if calls[ProviderCodex] != 1 || calls[ProviderProcess] != 1 {
		t.Fatalf("discovery calls = %v, want one per distinct executable", calls)
	}
	if preview.Discovery.Provider != ProviderCodex || len(preview.Roles) != 3 || preview.Roles[RoleReviewer].ProviderBinary.Path != executor.Path || preview.Roles[RoleManager].Provider != ProviderCodex {
		t.Fatalf("preview does not show each role's pinned executable: discovery=%+v roles=%+v", preview.Discovery, preview.Roles)
	}
	var config projectrun.Runtime
	if err := yaml.Unmarshal([]byte(preview.Mutation.Files[0].Content), &config); err != nil {
		t.Fatal(err)
	}
	if err := projectrun.ValidateRuntime(config); err != nil {
		t.Fatalf("previewed mixed runtime is invalid: %v", err)
	}
	for id, reviewer := range config.Review.Agents {
		if reviewer.Transport != projectrun.TransportProcess || !reviewer.Unmetered() || config.Agents[id].Transport != projectrun.TransportCodexAppServer {
			t.Fatalf("previewed roles lost their profiles: manager=%+v reviewer=%+v", config.Agents[id], reviewer)
		}
	}
}

func TestPreviewSkipsTheDefaultExecutableWhenNoRoleUsesIt(t *testing.T) {
	project := setupProjectFixture(t)
	executor := processExecutorTool(t, t.TempDir())
	process := &RoleProfile{Provider: ProviderProcess, Model: "scripted", ProviderExecutable: executor.Path, CostMode: projectrun.CostModeUnmetered}
	options := Options{Provider: ProviderCodex, Model: "gpt-6-luna", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
		Roles: &RoleOptions{Manager: process, Reviewer: process, Verifier: process}}
	calls := map[string]int{}
	preview, err := previewEditWithDiscovery(project, options, func(got Options) (Discovery, error) {
		calls[got.Provider]++
		if got.Provider != ProviderProcess {
			t.Fatalf("unused default executable was discovered: %+v", got)
		}
		return Discover(got)
	})
	if err != nil || calls[ProviderProcess] != 1 || preview.Discovery.Provider != ProviderProcess {
		t.Fatalf("process-only preview: discovery=%+v calls=%v err=%v", preview.Discovery, calls, err)
	}
}

func TestRoleThatSwitchesProviderDoesNotInheritTheCostMode(t *testing.T) {
	profiles, err := resolveRoleProfiles(Options{Provider: ProviderProcess, Model: "scripted", ProviderExecutable: "C:/tools/executor.exe",
		CostMode: projectrun.CostModeUnmetered, MaxCostMicros: 1, Roles: &RoleOptions{Manager: &RoleProfile{Provider: ProviderCodex, Model: "gpt-6-luna"}}})
	if err == nil || !strings.Contains(err.Error(), "metered agent requires explicit") {
		t.Fatalf("a Codex role under an unmetered default must ask for rates, got profiles=%+v err=%v", profiles, err)
	}
	profiles, err = resolveRoleProfiles(Options{Provider: ProviderProcess, Model: "scripted", ProviderExecutable: "C:/tools/executor.exe",
		CostMode: projectrun.CostModeUnmetered, MaxCostMicros: 1, Roles: &RoleOptions{Manager: &RoleProfile{Provider: ProviderCodex, Model: "gpt-6-luna",
			InputMicrosPerMillion: int64Ref(1), OutputMicrosPerMillion: int64Ref(2)}}})
	if err != nil || profiles[RoleManager].costMode != "" || profiles[RoleReviewer].costMode != projectrun.CostModeUnmetered {
		t.Fatalf("switched Codex role is not metered while process roles stay unmetered: %+v err=%v", profiles, err)
	}
}
