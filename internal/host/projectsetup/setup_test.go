package projectsetup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestBuildRuntimeMapsAllManagersAndPinsTools(t *testing.T) {
	root := t.TempDir()
	python := executableTool(t)
	adapter := testTool(t, root, "runner.py", false)
	provider := testTool(t, root, "codex.exe", true)
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{
		{ID: "root-manager", Namespace: ""},
		{ID: "orders-manager", Namespace: "commerce.orders", Parent: "root-manager"},
		{ID: "inventory-manager", Namespace: "commerce.inventory", Parent: "root-manager"},
	}}}
	options := Options{Provider: "codex", Model: "example-model", Effort: "high", InputMicrosPerMillion: 7, OutputMicrosPerMillion: 11, MaxCostMicros: 5000}
	config, err := BuildRuntime(project, options, Discovery{Provider: "codex", ProviderBinary: provider, Python: python, Adapter: adapter})
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	if len(config.Agents) != 3 {
		t.Fatalf("agent mappings = %d, want every active manager", len(config.Agents))
	}
	for _, id := range []string{"root-manager", "orders-manager", "inventory-manager"} {
		agent, ok := config.Agents[id]
		if !ok {
			t.Fatalf("missing runtime mapping for %s", id)
		}
		if agent.Command != python.Path || agent.Model != "example-model" || agent.ProviderVersion != provider.Version {
			t.Fatalf("unexpected mapping for %s: %#v", id, agent)
		}
		if agent.ModelOptions.(map[string]string)["model_reasoning_effort"] != "high" {
			t.Fatalf("Codex effort option missing for %s: %#v", id, agent.ModelOptions)
		}
		if len(agent.RuntimeFiles) != 3 {
			t.Fatalf("runtime file pins for %s = %d, want Python, adapter, provider", id, len(agent.RuntimeFiles))
		}
		for _, file := range agent.Environment {
			if strings.EqualFold(file, "HOME") || strings.EqualFold(file, "CODEX_HOME") || strings.EqualFold(file, "USERPROFILE") || strings.EqualFold(file, "APPDATA") {
				t.Fatalf("runtime allowlist includes private account variable %q", file)
			}
		}
	}
	if config.Mode != "controlled-local" || config.RequireIsolation || config.Limits.MaxRetries != 0 || config.Limits.MaxParallel != 1 || config.Limits.MaxCostMicros != 5000 {
		t.Fatalf("unsafe or unexpected limits: %#v", config)
	}
}

func TestBuildRuntimeRejectsUnsupportedProviderAndMissingPrices(t *testing.T) {
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	found := Discovery{Provider: "claude", ProviderBinary: Tool{Path: filepath.Join(t.TempDir(), "provider"), Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64), Mode: "0644"}, Python: Tool{Path: filepath.Join(t.TempDir(), "python"), Version: "3.13.0", Digest: "sha256:" + strings.Repeat("b", 64), Mode: "0644"}, Adapter: Tool{Path: filepath.Join(t.TempDir(), "runner.py"), Digest: "sha256:" + strings.Repeat("c", 64), Mode: "0644"}}
	if _, err := BuildRuntime(project, Options{Provider: "custom", Model: "m", InputMicrosPerMillion: 1, MaxCostMicros: 1}, found); err == nil {
		t.Fatal("unsupported provider was accepted")
	}
	if _, err := BuildRuntime(project, Options{Provider: "claude", Model: "m", MaxCostMicros: 1}, found); err == nil {
		t.Fatal("missing token price rates were accepted")
	}
}

func TestBuildRuntimeClaudeUsesOnlyItsDeclaredEffortOption(t *testing.T) {
	root := t.TempDir()
	found := Discovery{Provider: "claude", ProviderBinary: testTool(t, root, "claude.exe", true), Python: executableTool(t), Adapter: testTool(t, root, "claude-runner.py", false)}
	found.ProviderBinary.Version = "2.1.0"
	project := &projectwork.Project{Report: projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}}
	config, err := BuildRuntime(project, Options{Provider: "claude", Model: "model", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 2, MaxCostMicros: 10}, found)
	if err != nil {
		t.Fatalf("BuildRuntime: %v", err)
	}
	agent := config.Agents["root"]
	if got := agent.ModelOptions.(map[string]string); len(got) != 1 || got["effort"] != "high" {
		t.Fatalf("Claude options = %#v", got)
	}
}

func TestDiscoverRejectsCommandShimAndParsesOnlyVersion(t *testing.T) {
	if !isCommandShim(`C:\tools\codex.cmd`) && runtime.GOOS == "windows" {
		t.Fatal("Windows command shim was accepted")
	}
	if match := versionPattern.FindStringSubmatch("codex-cli 0.130.0\n"); len(match) != 2 || match[1] != "0.130.0" {
		t.Fatalf("version parse = %#v", match)
	}
	if _, err := Discover(Options{Provider: "codex"}); err == nil || !strings.Contains(err.Error(), "--tool-root") {
		t.Fatalf("missing source root error = %v", err)
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
	if strings.Contains(name, "python") {
		version = "3.13.0"
	} else if strings.Contains(name, "codex") {
		version = "0.130.0"
	}
	return Tool{Path: resolved, Version: version, Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: actualMode}
}

func modeOctal(mode os.FileMode) string { return fmt.Sprintf("%04o", mode) }

func executableTool(t *testing.T) Tool {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := "0644"
	if runtime.GOOS != "windows" {
		mode = modeOctal(info.Mode().Perm())
	}
	sum := sha256.Sum256(content)
	return Tool{Path: path, Version: "3.13.0", Digest: "sha256:" + hex.EncodeToString(sum[:]), Mode: mode}
}
