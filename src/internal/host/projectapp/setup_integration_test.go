package projectapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexecutor"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
	"go.yaml.in/yaml/v3"
)

func TestSetupRoutesRuntimeEditThroughApplicationAndGuardsReviewedDigest(t *testing.T) {
	root := makePlanFixture(t)
	runtimePath := filepath.Join(root, filepath.FromSlash(projectvalue.RuntimePath))
	if err := os.Remove(runtimePath); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "rm", projectvalue.RuntimePath)
	runGit(t, root, "commit", "-m", "start without project runtime")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_APP_SETUP_VERSION_HELPER", "1")
	options := projectsetup.Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high", CodexProfile: ":workspace",
		ProviderExecutable: executable, InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000,
	}
	operations := NewOperations()
	preview, err := operations.Setup(SetupOperation{Root: root, Options: options})
	if err != nil {
		t.Fatalf("Application setup preview: %v", err)
	}
	if preview.EditPlan.Digest == "" || len(preview.Mutation.Files) != 1 || preview.Mutation.Files[0].Path != projectvalue.RuntimePath {
		t.Fatalf("setup preview is not one digest-bound runtime edit: %+v", preview)
	}
	var runtimeConfig projectconfig.Runtime
	if err := yaml.Unmarshal([]byte(preview.Mutation.Files[0].Content), &runtimeConfig); err != nil {
		t.Fatalf("decode proposed runtime: %v", err)
	}
	if err := projectconfig.ValidateRuntime(runtimeConfig); err != nil {
		t.Fatalf("proposed runtime is invalid: %v", err)
	}
	report, err := operations.Index(Selection{Root: root})
	if err != nil || len(report.Managers) != 1 {
		t.Fatalf("load setup fixture model through Application: managers=%d err=%v", len(report.Managers), err)
	}
	managerID := report.Managers[0].ID
	manager := runtimeConfig.Agents[managerID]
	if runtimeConfig.Review == nil || len(runtimeConfig.Agents) != 1 || len(runtimeConfig.Review.Agents) != 1 {
		t.Fatalf("runtime lost Manager or separate Review binding: %+v", runtimeConfig)
	}
	reviewer := runtimeConfig.Review.Agents[managerID]
	if manager.AppServer == nil || reviewer.AppServer == nil || runtimeConfig.Verifier == nil || runtimeConfig.Verifier.AppServer == nil {
		t.Fatalf("runtime YAML omitted App Server settings: %s", preview.Mutation.Files[0].Content)
	}
	if manager.AppServer.PermissionProfile != ":workspace" || reviewer.AppServer.PermissionProfile != ":workspace" || runtimeConfig.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("explicit setup profile must apply consistently to independent roles: manager=%+v reviewer=%+v verifier=%+v", manager.AppServer, reviewer.AppServer, runtimeConfig.Verifier)
	}
	if preview.Discovery.ProviderBinary.Version != "codex-cli 0.162.0" {
		t.Fatalf("provider discovery version = %q", preview.Discovery.ProviderBinary.Version)
	}

	_, err = operations.Setup(SetupOperation{Root: root, Options: options, Write: true, ExpectedDigest: "sha256:stale"})
	if err == nil || !strings.Contains(err.Error(), "expectedDigest") {
		t.Fatalf("stale setup digest was not rejected: %v", err)
	}
	if _, err := os.Stat(runtimePath); !os.IsNotExist(err) {
		t.Fatalf("rejected preview changed runtime file: stat error=%v", err)
	}
	if _, err := operations.Setup(SetupOperation{Root: root, Options: options, Write: true, ExpectedDigest: preview.EditPlan.Digest}); err != nil {
		t.Fatalf("apply exact reviewed setup digest: %v", err)
	}
	written, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	var applied projectconfig.Runtime
	if err := yaml.Unmarshal(written, &applied); err != nil || applied.Agents[managerID].AppServer.PermissionProfile != ":workspace" || applied.Review.Agents[managerID].AppServer.PermissionProfile != ":workspace" || applied.Verifier == nil || applied.Verifier.AppServer.PermissionProfile != ":workspace" {
		t.Fatalf("guarded write did not preserve the shared role profile: runtime=%+v err=%v", applied, err)
	}
}

func TestSetupRoleOptionsPreserveNativeExecutorSurface(t *testing.T) {
	root := makePlanFixture(t)
	runtimePath := filepath.Join(root, filepath.FromSlash(projectvalue.RuntimePath))
	if err := os.Remove(runtimePath); err == nil {
		runGit(t, root, "rm", projectvalue.RuntimePath)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := os.Stat(runtimePath); !os.IsNotExist(err) {
		t.Fatalf("runtime file should be absent before setup: %v", err)
	}
	runGit(t, root, "commit", "-m", "start without project runtime")

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_APP_SETUP_VERSION_HELPER", "1")
	options := projectsetup.Options{
		Provider: "codex", Model: "gpt-6-luna", Effort: "high", ProviderExecutable: executable,
		InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 100,
	}
	preview, err := NewOperations().Setup(SetupOperation{Root: root, Options: options})
	if err != nil {
		t.Fatalf("Application setup preview: %v", err)
	}
	var runtimeConfig projectconfig.Runtime
	if err := yaml.Unmarshal([]byte(preview.Mutation.Files[0].Content), &runtimeConfig); err != nil {
		t.Fatalf("decode proposed runtime: %v", err)
	}
	if err := projectconfig.ValidateRuntime(runtimeConfig); err != nil {
		t.Fatalf("proposed runtime is invalid: %v", err)
	}
	if runtimeConfig.Review == nil || runtimeConfig.Verifier == nil {
		t.Fatalf("setup runtime omitted independent Review or Verifier roles: %+v", runtimeConfig)
	}
	report, err := NewOperations().Index(Selection{Root: root})
	if err != nil || len(report.Managers) != 1 {
		t.Fatalf("load setup fixture model through Application: managers=%d err=%v", len(report.Managers), err)
	}
	managerID := report.Managers[0].ID
	invoker := projectexecutor.New(codexappserver.Options{}, projectrun.NativeHelperPolicy{})

	for _, helpersEnabled := range []bool{true, false} {
		name := "helpers-enabled"
		if !helpersEnabled {
			name = "helpers-disabled"
		}
		t.Run(name, func(t *testing.T) {
			candidate := runtimeConfig
			candidate.Agents = make(map[string]projectconfig.Agent, len(runtimeConfig.Agents))
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
			candidate.Review = &projectconfig.ReviewConfig{
				Agents:    make(map[string]projectconfig.Agent, len(runtimeConfig.Review.Agents)),
				MaxRounds: runtimeConfig.Review.MaxRounds, MaxManagerRounds: runtimeConfig.Review.MaxManagerRounds,
			}
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
				agent   projectconfig.Agent
				request agentexec.Request
				invoker *projectexecutor.Executor
			}{
				{name: "manager", agent: candidate.Agents[managerID], request: agentexec.Request{Role: agentexec.RoleExecutor,
					Context: json.RawMessage(`{"kind":"projectrun-task/v1","managerId":"root","phase":"work","allowedWritePaths":["src/"]}`)}},
				{name: "reviewer", agent: candidate.Review.Agents[managerID], request: agentexec.Request{Role: agentexec.RoleExecutor,
					Context: json.RawMessage(`{"kind":"projectrun-review/v1"}`)}},
				{name: "verifier", agent: *candidate.Verifier, request: agentexec.Request{Role: agentexec.RoleVerifier,
					Context: json.RawMessage(`{"kind":"projectrun-verify/v1"}`)}},
				{name: "helper", agent: candidate.Agents[managerID], invoker: projectexecutor.NewChild(codexappserver.Options{}),
					request: agentexec.Request{Role: agentexec.RoleExecutor,
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

func init() {
	if os.Getenv("MARKITECT_APP_SETUP_VERSION_HELPER") == "1" {
		_, _ = os.Stdout.WriteString("codex-cli 0.162.0\n")
		os.Exit(0)
	}
}
