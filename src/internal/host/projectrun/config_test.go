package projectrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

func validRuntime() Runtime {
	return Runtime{
		APIVersion: APIVersion,
		Mode:       ModeControlledLocal,
		Agents: map[string]Agent{
			"commerce": {
				Command: "fake-agent", Args: []string{"--protocol"}, Model: "fixture",
				ProviderVersion: "fixture-v1", Timeout: Duration(30 * time.Second),
				MaxStdoutBytes: 64 << 10, MaxStderrBytes: 16 << 10,
				Environment: []string{"OPENAI_API_KEY"},
				Pricing:     Pricing{InputMicrosPerMillion: 100, OutputMicrosPerMillion: 200},
			},
		},
		Limits: Limits{
			MaxDepth: 4, MaxStarts: 12, MaxRetries: 1, MaxParallel: 2,
			MaxDuration: Duration(10 * time.Minute), MaxCostMicros: 5000,
			MaxCandidateFileBytes: 1 << 20, MaxCandidateBytes: 8 << 20,
		},
	}
}

func TestValidateRuntimeRequiresExplicitBoundedControlledLocalConfiguration(t *testing.T) {
	if err := ValidateRuntime(validRuntime()); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := []struct {
		name   string
		change func(*Runtime)
		want   string
	}{
		{"missing finite starts", func(c *Runtime) { c.Limits.MaxStarts = 0 }, "limits"},
		{"unbounded retries", func(c *Runtime) { c.Limits.MaxRetries = 4 }, "limits"},
		{"asserted isolation", func(c *Runtime) { c.Mode = ModeIsolated }, "unavailable"},
		{"required isolation", func(c *Runtime) { c.RequireIsolation = true }, "unavailable"},
		{"ambient git credential", func(c *Runtime) {
			agent := c.Agents["commerce"]
			agent.Environment = []string{"GIT_ASKPASS"}
			c.Agents["commerce"] = agent
		}, "may not inherit"},
		{"environment value persisted", func(c *Runtime) {
			agent := c.Agents["commerce"]
			agent.Environment = []string{"OPENAI_API_KEY=secret"}
			c.Agents["commerce"] = agent
		}, "invalid or duplicate"},
		{"no enforceable price", func(c *Runtime) {
			agent := c.Agents["commerce"]
			agent.Pricing = Pricing{}
			c.Agents["commerce"] = agent
		}, "pricing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := validRuntime()
			tc.change(&config)
			if err := ValidateRuntime(config); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q; got %v", tc.want, err)
			}
		})
	}
}

func TestNativeRuntimeRestrictsInstructionsRolesAndHomeInheritance(t *testing.T) {
	config := validRuntime()
	agent := config.Agents["commerce"]
	agent.WorkspaceMode = "scoped"
	agent.InstructionPaths = []string{"AGENTS.md"}
	agent.Environment = []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA"}
	config.Agents["commerce"] = agent
	if err := ValidateRuntime(config); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CODEX_HOME", "SSH_AUTH_SOCK", "GIT_ASKPASS"} {
		changed := agent
		changed.Environment = []string{name}
		config.Agents["commerce"] = changed
		if err := ValidateRuntime(config); err == nil {
			t.Fatalf("native runtime allowed forbidden environment %s", name)
		}
	}
	config.Agents["commerce"] = agent
	config.Verifier = &agent
	if err := ValidateRuntime(config); err == nil {
		t.Fatal("native verifier allowed executor workspace")
	}
	config.Verifier = nil
	config.Review = &ReviewConfig{Agents: map[string]Agent{"commerce": agent}, MaxRounds: 1, MaxManagerRounds: 1}
	if err := ValidateRuntime(config); err == nil {
		t.Fatal("native reviewer allowed executor workspace")
	}
	config.Review = nil
	agent.WorkspaceMode = ""
	config.Agents["commerce"] = agent
	if err := ValidateRuntime(config); err == nil {
		t.Fatal("proposal runtime allowed native-only instructions")
	}
}

func TestLoadRuntimeStrictlyDecodesOneDocumentAndDurations(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(validRuntime())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(RuntimePath))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRuntime(root)
	if err != nil {
		t.Fatalf("load config: %v\n%s", err, data)
	}
	if loaded.Limits.MaxDuration != Duration(10*time.Minute) || loaded.Agents["commerce"].Timeout != Duration(30*time.Second) {
		t.Fatalf("duration decode mismatch: %+v", loaded)
	}

	for name, content := range map[string]string{
		"unknown field":   strings.Replace(string(data), "maxStarts:", "mystery: true\n  maxStarts:", 1),
		"second document": string(data) + "---\napiVersion: ignored\n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadRuntime(root); err == nil {
				t.Fatal("invalid runtime config unexpectedly loaded")
			}
		})
	}
}

func TestLoadRuntimeRejectsSymlinkedConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, filepath.FromSlash(RuntimePath))); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := LoadRuntime(root); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestReadOnlyAgentKeepsNativeWorkAndAssessmentBindingsSeparate(t *testing.T) {
	config := validRuntime()
	custom, err := ReadOnlyAgent(config, "commerce")
	if err != nil || custom.Command != "fake-agent" || custom.WorkspaceMode != "" {
		t.Fatalf("explicit context-only custom adapter unavailable: %#v %v", custom, err)
	}
	native := config.Agents["commerce"]
	native.WorkspaceMode = "scoped"
	native.InstructionPaths = []string{"AGENTS.md"}
	native.Command = "native-manager"
	config.Agents["commerce"] = native
	if _, err := ReadOnlyAgent(config, "commerce"); err == nil {
		t.Fatal("native work silently used for a read-only assessment")
	}
	config.Review = &ReviewConfig{Agents: map[string]Agent{"commerce": custom}}
	assessment, err := ReadOnlyAgent(config, "commerce")
	if err != nil || assessment.Command != "fake-agent" || assessment.WorkspaceMode != "" {
		t.Fatalf("explicit read-only binding not selected: %#v %v", assessment, err)
	}
	if config.Agents["commerce"].Command != "native-manager" || config.Agents["commerce"].WorkspaceMode != "scoped" {
		t.Fatal("assessment selection mutated native work binding")
	}
	config.Review.Agents["commerce"] = native
	if _, err := ReadOnlyAgent(config, "commerce"); err == nil {
		t.Fatal("native assessment binding accepted")
	}
	if _, err := ReadOnlyAgent(config, "unknown"); err == nil {
		t.Fatal("unknown Manager gained an assessment binding")
	}
}
